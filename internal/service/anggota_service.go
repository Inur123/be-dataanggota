package service

import (
	"errors"
	"regexp"

	"anggota.pelajarnumagetan.or.id/internal/config"
	"anggota.pelajarnumagetan.or.id/internal/domain"
	"anggota.pelajarnumagetan.or.id/internal/repository"
	"anggota.pelajarnumagetan.or.id/internal/utils"
)

type AnggotaService interface {
	Onboard(ssoUserID string, nama string, email string, unitID string) error
	GetProfile(ssoUserID string) (*domain.Anggota, error)
	UpdateProfile(ssoUserID string, input *domain.Anggota) error
	List(filter map[string]interface{}, currentAdminSSOID string, isSuperAdmin bool) ([]domain.Anggota, error)
	Verify(anggotaID string, currentAdminSSOID string, isSuperAdmin bool) error
	ToggleActive(anggotaID string, currentAdminSSOID string, isSuperAdmin bool) (bool, error)
}

type anggotaService struct {
	repo      repository.AnggotaRepository
	adminRepo repository.AdminRepository
}

func NewAnggotaService(repo repository.AnggotaRepository, adminRepo repository.AdminRepository) AnggotaService {
	return &anggotaService{repo: repo, adminRepo: adminRepo}
}

func (s *anggotaService) Onboard(ssoUserID string, nama string, email string, unitID string) error {
	if ssoUserID == "" || nama == "" || email == "" || unitID == "" {
		return errors.New("sso user id, nama, email, dan pimpinan unit wajib diisi")
	}

	// 1. Dapatkan periode aktif dari unit tersebut
	activePeriod, err := s.adminRepo.GetActivePeriodByUnitID(unitID)
	if err != nil {
		return errors.New("unit pimpinan yang dipilih belum menentukan periode aktif saat ini")
	}

	// 2. Cek apakah sudah terdaftar
	existing, _ := s.repo.GetBySSOUserID(ssoUserID)
	if existing != nil {
		return errors.New("akun ini sudah terdaftar sebagai anggota")
	}

	isVerified := false
	verifiedBy := ""

	anggota := &domain.Anggota{
		SSOUserID:      ssoUserID,
		NamaLengkap:    nama,
		Email:          email,
		PimpinanUnitID: unitID,
		PeriodeMasukID: activePeriod.ID,
		IsVerified:     isVerified,
	}
	if verifiedBy != "" {
		anggota.VerifiedBy = &verifiedBy
	}

	return s.repo.Create(anggota)
}

func (s *anggotaService) GetProfile(ssoUserID string) (*domain.Anggota, error) {
	cfg := config.Get()
	anggota, err := s.repo.GetBySSOUserID(ssoUserID)
	if err != nil {
		return nil, err
	}

	// Dekripsi NIK untuk dibaca di profile
	if anggota.NIK != nil && *anggota.NIK != "" {
		decrypted, err := utils.DecryptField(*anggota.NIK, cfg.EncryptionKey)
		if err == nil {
			anggota.NIK = &decrypted
		}
	}

	return anggota, nil
}

func (s *anggotaService) UpdateProfile(ssoUserID string, input *domain.Anggota) error {
	cfg := config.Get()

	// 1. Ambil data asli
	anggota, err := s.repo.GetBySSOUserID(ssoUserID)
	if err != nil {
		return errors.New("anggota tidak ditemukan")
	}

	// 2. Update field yang diizinkan
	if input.NamaLengkap != "" {
		anggota.NamaLengkap = input.NamaLengkap
	}
	if input.FotoUrl != nil {
		anggota.FotoUrl = input.FotoUrl
	}
	if input.TempatLahir != nil {
		anggota.TempatLahir = input.TempatLahir
	}
	if input.TanggalLahir != nil {
		anggota.TanggalLahir = input.TanggalLahir
	}
	if input.AlamatLengkap != nil {
		anggota.AlamatLengkap = input.AlamatLengkap
	}
	if input.RFID != nil {
		anggota.RFID = input.RFID
	}
	if input.Pekerjaan != nil {
		anggota.Pekerjaan = input.Pekerjaan
	}
	if input.HobiMinat != nil {
		anggota.HobiMinat = input.HobiMinat
	}

	// 3. Validasi & Format Jenis Kelamin (Auto organisasi)
	if input.JenisKelamin != nil && *input.JenisKelamin != "" {
		anggota.JenisKelamin = input.JenisKelamin
		if *input.JenisKelamin == "Laki-laki" {
			ipnu := "IPNU"
			anggota.Organisasi = &ipnu
		} else if *input.JenisKelamin == "Perempuan" {
			ippnu := "IPPNU"
			anggota.Organisasi = &ippnu
		}
	}

	// 4. Validasi & Enkripsi NIK
	if input.NIK != nil && *input.NIK != "" {
		// Validasi regex NIK (16 digit angka)
		matched, _ := regexp.MatchString(`^[0-9]{16}$`, *input.NIK)
		if !matched {
			return errors.New("NIK harus berupa 16 digit angka")
		}

		encrypted, err := utils.EncryptField(*input.NIK, cfg.EncryptionKey)
		if err != nil {
			return errors.New("gagal memproses keamanan data NIK")
		}
		anggota.NIK = &encrypted
	}

	// 5. Format Handphone (WhatsApp)
	if input.Phone != nil && *input.Phone != "" {
		formattedPhone := utils.FormatWhatsApp(*input.Phone)
		anggota.Phone = &formattedPhone
	}

	// 6. Handle update detail (pendidikan & perkaderan jika diisi / dikirim)
	if input.Pendidikan != nil {
		anggota.Pendidikan = input.Pendidikan
	}
	if input.Perkaderan != nil {
		anggota.Perkaderan = input.Perkaderan
	}

	return s.repo.Update(anggota)
}

func (s *anggotaService) List(filter map[string]interface{}, currentAdminSSOID string, isSuperAdmin bool) ([]domain.Anggota, error) {
	cfg := config.Get()

	var list []domain.Anggota
	var err error

	if isSuperAdmin {
		// Superadmin bisa lihat semua data tanpa filter unit
		list, err = s.repo.List(filter, "", "superadmin")
	} else {
		// 1. Dapatkan detail kepengurusan admin
		admin, adminErr := s.adminRepo.GetAdminUserBySSOUserID(currentAdminSSOID)
		if adminErr != nil {
			return nil, errors.New("anda tidak memiliki wewenang admin untuk mengakses data ini")
		}
		list, err = s.repo.List(filter, admin.AdminUnitID, admin.Role)
	}
	if err != nil {
		return nil, err
	}

	// Dekripsi NIK untuk list admin
	for i := range list {
		if list[i].NIK != nil && *list[i].NIK != "" {
			decrypted, err := utils.DecryptField(*list[i].NIK, cfg.EncryptionKey)
			if err == nil {
				list[i].NIK = &decrypted
			}
		}
	}

	return list, nil
}

func (s *anggotaService) Verify(anggotaID string, currentAdminSSOID string, isSuperAdmin bool) error {
	var admin *domain.AdminUser

	if isSuperAdmin {
		// Superadmin bisa verifikasi siapapun tanpa perlu terdaftar sebagai admin unit
		// verified_by akan ditampilkan sebagai "Super Admin"
		return s.repo.Verify(anggotaID, "Super Admin")
	} else {
		// 1. Dapatkan info admin
		var err error
		admin, err = s.adminRepo.GetAdminUserBySSOUserID(currentAdminSSOID)
		if err != nil {
			return errors.New("anda tidak memiliki wewenang untuk memverifikasi anggota")
		}
	}

	// 2. Ambil data anggota
	anggota, err := s.repo.GetByID(anggotaID)
	if err != nil {
		return errors.New("anggota tidak ditemukan")
	}

	// 3. Hak akses verifikasi
	if admin.Role != "Cabang" {
		if admin.Role == "PAC" {
			// Pastikan anggota berada dalam kecamatan yang sama
			var adminUnit, memberUnit *domain.AdminUnit
			adminUnit, _ = s.adminRepo.GetUnitByID(admin.AdminUnitID)
			memberUnit, _ = s.adminRepo.GetUnitByID(anggota.PimpinanUnitID)
			if adminUnit.Kecamatan != memberUnit.Kecamatan {
				return errors.New("anda hanya boleh memverifikasi anggota di dalam kecamatan anda")
			}
		} else {
			// Admin PR/PK hanya boleh memverifikasi anggotanya sendiri
			if anggota.PimpinanUnitID != admin.AdminUnitID {
				return errors.New("anda hanya boleh memverifikasi anggota unit pimpinan anda")
			}
		}
	}

	// 4. Verifikasi
	var verifiedBy string
	switch admin.Role {
	case "Cabang":
		verifiedBy = "Admin Cabang"
	case "PAC":
		verifiedBy = "Admin PAC"
	case "PR":
		verifiedBy = "Admin PR"
	case "PK":
		verifiedBy = "Admin PK"
	default:
		verifiedBy = "Admin " + admin.Role
	}
	return s.repo.Verify(anggotaID, verifiedBy)
}

func (s *anggotaService) ToggleActive(anggotaID string, currentAdminSSOID string, isSuperAdmin bool) (bool, error) {
	var admin *domain.AdminUser

	// 1. Ambil data anggota
	anggota, err := s.repo.GetByID(anggotaID)
	if err != nil {
		return false, errors.New("anggota tidak ditemukan")
	}

	// 2. Hak akses check
	if !isSuperAdmin {
		var err error
		admin, err = s.adminRepo.GetAdminUserBySSOUserID(currentAdminSSOID)
		if err != nil {
			return false, errors.New("anda tidak memiliki wewenang untuk menonaktifkan/mengaktifkan anggota")
		}

		if admin.Role != "Cabang" {
			if admin.Role == "PAC" {
				// Pastikan anggota berada dalam kecamatan yang sama
				var adminUnit, memberUnit *domain.AdminUnit
				adminUnit, _ = s.adminRepo.GetUnitByID(admin.AdminUnitID)
				memberUnit, _ = s.adminRepo.GetUnitByID(anggota.PimpinanUnitID)
				if adminUnit.Kecamatan != memberUnit.Kecamatan {
					return false, errors.New("anda hanya boleh mengelola anggota di dalam kecamatan anda")
				}
			} else {
				// Admin PR/PK hanya boleh mengelola anggotanya sendiri
				if anggota.PimpinanUnitID != admin.AdminUnitID {
					return false, errors.New("anda hanya boleh mengelola anggota unit pimpinan anda")
				}
			}
		}
	}

	// 3. Toggle IsActive
	newActiveStatus := !anggota.IsActive
	err = s.repo.ToggleActive(anggotaID, newActiveStatus)
	if err != nil {
		return false, err
	}

	return newActiveStatus, nil
}
