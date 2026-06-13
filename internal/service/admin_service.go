package service

import (
	"errors"
	"fmt"
	"log"

	"anggota.pelajarnumagetan.or.id/internal/config"
	"anggota.pelajarnumagetan.or.id/internal/domain"
	"anggota.pelajarnumagetan.or.id/internal/repository"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type AdminService interface {
	CreateUnit(unit *domain.AdminUnit) error
	ListUnits(role string, kecamatan string, hasActivePeriod bool) ([]domain.AdminUnit, error)
	CreateAdminUser(adminRole string, unitID string, identifier string) error
	ListAdminUsers() ([]domain.AdminUser, error)
	DeleteAdminUser(ssoUserID string) error
	CreatePeriod(name string, unitID string, currentAdminSSOID string, isSuperAdmin bool) error
	ListPeriods(unitID string) ([]domain.Period, error)
	SetActivePeriod(unitID string, periodID string, currentAdminSSOID string, isSuperAdmin bool) error
	GetAdminUserBySSOUserID(ssoUserID string) (*domain.AdminUser, error)
	SearchAnggota(query string) ([]domain.Anggota, error)
}

type adminService struct {
	repo repository.AdminRepository
}

func NewAdminService(repo repository.AdminRepository) AdminService {
	return &adminService{repo: repo}
}

func (s *adminService) CreateUnit(unit *domain.AdminUnit) error {
	if unit.NamaUnit == "" || unit.Role == "" {
		return errors.New("nama unit dan role wajib diisi")
	}

	validRoles := map[string]bool{"Cabang": true, "PAC": true, "PR": true, "PK": true}
	if !validRoles[unit.Role] {
		return errors.New("role tidak valid. Pilih: Cabang, PAC, PR, atau PK")
	}

	return s.repo.CreateUnit(unit)
}

func (s *adminService) ListUnits(role string, kecamatan string, hasActivePeriod bool) ([]domain.AdminUnit, error) {
	return s.repo.ListUnits(role, kecamatan, hasActivePeriod)
}

// Helper to auto verify registered admin in SSO database
func (s *adminService) autoVerifySSOUser(email string) {
	cfg := config.Get()
	var dsn string
	ssoDBName := "sso_pelajarnu"
	if cfg.DBPassword != "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s&TimeZone=Asia/Jakarta",
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBHost,
			cfg.DBPort,
			ssoDBName,
			cfg.DBSslMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgres://%s@%s:%s/%s?sslmode=%s&TimeZone=Asia/Jakarta",
			cfg.DBUser,
			cfg.DBHost,
			cfg.DBPort,
			ssoDBName,
			cfg.DBSslMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to SSO database for auto-verify: %v", err)
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		return
	}
	defer sqlDB.Close()

	err = db.Exec("UPDATE users SET is_verified = ? WHERE email = ?", true, email).Error
	if err != nil {
		log.Printf("Failed to auto-verify user in SSO database: %v", err)
	} else {
		log.Printf("Successfully auto-verified SSO user: %s", email)
	}
}

func (s *adminService) getSSOUserByIdentifier(identifier string) (string, error) {
	ssoUserID, err := s.repo.FindSSOUserIDByIdentifier(identifier)
	if err != nil {
		return "", fmt.Errorf("gagal mencari data anggota: %v", err)
	}

	if ssoUserID == "" {
		return "", errors.New("akun tidak ditemukan di database anggota. Silakan minta user tersebut untuk mendaftar dan melengkapi data anggota terlebih dahulu")
	}

	return ssoUserID, nil
}

// CreateAdminUser assigns local admin privilege to an existing SSO user by their email or UUID.
func (s *adminService) CreateAdminUser(adminRole string, unitID string, identifier string) error {
	if adminRole == "" || unitID == "" || identifier == "" {
		return errors.New("role, unit id, dan email/UUID SSO wajib diisi")
	}

	validRoles := map[string]bool{"Cabang": true, "PAC": true, "PR": true, "PK": true}
	if !validRoles[adminRole] {
		return errors.New("role admin tidak valid. Pilih: Cabang, PAC, PR, atau PK")
	}

	// 1. Cari user di database SSO
	ssoUserID, err := s.getSSOUserByIdentifier(identifier)
	if err != nil {
		return err
	}

	// 2. Cek apakah admin dengan SSO User ID ini sudah terdaftar secara lokal
	existing, _ := s.repo.GetAdminUserBySSOUserID(ssoUserID)
	if existing != nil {
		return errors.New("user ini sudah terdaftar sebagai admin di sistem lokal")
	}

	// 3. Simpan relasi admin lokal
	adminUser := &domain.AdminUser{
		SSOUserID:   ssoUserID,
		AdminUnitID: unitID,
		Role:        adminRole,
	}

	return s.repo.CreateAdminUser(adminUser)
}

func (s *adminService) ListAdminUsers() ([]domain.AdminUser, error) {
	return s.repo.ListAdminUsers()
}

func (s *adminService) DeleteAdminUser(ssoUserID string) error {
	return s.repo.DeleteAdminUser(ssoUserID)
}

func (s *adminService) CreatePeriod(name string, unitID string, currentAdminSSOID string, isSuperAdmin bool) error {
	if name == "" || unitID == "" {
		return errors.New("nama periode dan unit id wajib diisi")
	}

	// Bypass check for superadmin
	if !isSuperAdmin {
		admin, err := s.repo.GetAdminUserBySSOUserID(currentAdminSSOID)
		if err != nil {
			return errors.New("akses ditolak: anda tidak terdaftar sebagai admin")
		}

		if admin.AdminUnitID != unitID {
			return errors.New("akses ditolak: anda hanya boleh mengelola periode untuk unit pimpinan anda sendiri")
		}
	}

	// Jika belum ada periode sama sekali → langsung aktif
	// Jika sudah ada → non-aktif, admin bisa aktifkan manual saat ganti periode
	existing, _ := s.repo.ListPeriods(unitID)
	isFirstPeriod := len(existing) == 0

	period := &domain.Period{
		Nama:        name,
		AdminUnitID: unitID,
		IsActive:    isFirstPeriod,
	}
	return s.repo.CreatePeriod(period)
}

func (s *adminService) ListPeriods(unitID string) ([]domain.Period, error) {
	return s.repo.ListPeriods(unitID)
}

func (s *adminService) SetActivePeriod(unitID string, periodID string, currentAdminSSOID string, isSuperAdmin bool) error {
	// Bypass check for superadmin
	if !isSuperAdmin {
		admin, err := s.repo.GetAdminUserBySSOUserID(currentAdminSSOID)
		if err != nil {
			return errors.New("akses ditolak: anda tidak terdaftar sebagai admin")
		}
		if admin.AdminUnitID != unitID {
			return errors.New("akses ditolak: anda hanya boleh mengelola periode untuk unit pimpinan anda sendiri")
		}
	}

	// Harus ada minimal 2 periode agar bisa ganti aktif
	// (mencegah kondisi semua periode jadi non-aktif)
	existing, _ := s.repo.ListPeriods(unitID)
	if len(existing) < 2 {
		return errors.New("harus ada minimal 2 periode untuk dapat mengganti periode aktif")
	}

	return s.repo.SetActivePeriod(unitID, periodID)
}

func (s *adminService) GetAdminUserBySSOUserID(ssoUserID string) (*domain.AdminUser, error) {
	return s.repo.GetAdminUserBySSOUserID(ssoUserID)
}

func (s *adminService) SearchAnggota(query string) ([]domain.Anggota, error) {
	return s.repo.SearchAnggota(query)
}
