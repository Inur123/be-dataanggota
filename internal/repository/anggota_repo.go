package repository

import (
	"anggota.pelajarnumagetan.or.id/internal/domain"
	"gorm.io/gorm"
)

type AnggotaRepository interface {
	Create(anggota *domain.Anggota) error
	GetByID(id string) (*domain.Anggota, error)
	GetBySSOUserID(ssoUserID string) (*domain.Anggota, error)
	Update(anggota *domain.Anggota) error
	List(filter map[string]interface{}, unitID string, role string) ([]domain.Anggota, error)
	Verify(anggotaID string, verifiedBy string) error
	ToggleActive(anggotaID string, isActive bool) error
}

type anggotaRepository struct {
	db *gorm.DB
}

func NewAnggotaRepository(db *gorm.DB) AnggotaRepository {
	return &anggotaRepository{db: db}
}

func (r *anggotaRepository) Create(anggota *domain.Anggota) error {
	return r.db.Create(anggota).Error
}

func (r *anggotaRepository) GetByID(id string) (*domain.Anggota, error) {
	var anggota domain.Anggota
	err := r.db.Preload("Pendidikan").Preload("Perkaderan").Preload("JabatanHistory").First(&anggota, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &anggota, nil
}

func (r *anggotaRepository) GetBySSOUserID(ssoUserID string) (*domain.Anggota, error) {
	var anggota domain.Anggota
	err := r.db.Preload("Pendidikan").Preload("Perkaderan").Preload("JabatanHistory").First(&anggota, "sso_user_id = ?", ssoUserID).Error
	if err != nil {
		return nil, err
	}
	return &anggota, nil
}

func (r *anggotaRepository) Update(anggota *domain.Anggota) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Hapus secara eksplisit dari tabel untuk menghindari error foreign key constraint/NULL anggota_id
		if err := tx.Where("anggota_id = ?", anggota.ID).Delete(&domain.RiwayatPendidikan{}).Error; err != nil {
			return err
		}
		if err := tx.Where("anggota_id = ?", anggota.ID).Delete(&domain.RiwayatPerkaderan{}).Error; err != nil {
			return err
		}

		// 2. Set AnggotaID di setiap element slice baru secara eksplisit & reset ID/primary key
		for i := range anggota.Pendidikan {
			anggota.Pendidikan[i].ID = ""
			anggota.Pendidikan[i].AnggotaID = anggota.ID
		}
		for i := range anggota.Perkaderan {
			anggota.Perkaderan[i].ID = ""
			anggota.Perkaderan[i].AnggotaID = anggota.ID
		}

		// 3. Simpan data anggota beserta asosiasi barunya
		return tx.Session(&gorm.Session{FullSaveAssociations: true}).Save(anggota).Error
	})
}

func (r *anggotaRepository) List(filter map[string]interface{}, unitID string, role string) ([]domain.Anggota, error) {
	var list []domain.Anggota
	query := r.db.Model(&domain.Anggota{}).Preload("Pendidikan").Preload("Perkaderan").Preload("JabatanHistory")

	// Superadmin melihat semua anggota tanpa filter unit
	if role != "superadmin" && unitID != "" {
		// Scope berdasarkan unit pimpinan admin
		query = query.Where("pimpinan_unit_id = ?", unitID)
	}

	// Admin bukan merupakan anggota, maka kecualikan user yang terdaftar sebagai admin
	query = query.Where("sso_user_id NOT IN (SELECT sso_user_id FROM admin_users)")

	// Terapkan filter tambahan
	if val, ok := filter["organisasi"]; ok && val != "" {
		query = query.Where("organisasi = ?", val)
	}
	if val, ok := filter["jenis_kelamin"]; ok && val != "" {
		query = query.Where("jenis_kelamin = ?", val)
	}
	if val, ok := filter["pimpinan_unit_id"]; ok && val != "" {
		query = query.Where("pimpinan_unit_id = ?", val)
	}
	if val, ok := filter["periode_masuk_id"]; ok && val != "" {
		query = query.Where("periode_masuk_id = ?", val)
	}
	if val, ok := filter["search"]; ok && val != "" {
		searchStr := "%" + val.(string) + "%"
		query = query.Where("nama_lengkap ILIKE ? OR email ILIKE ? OR nia ILIKE ?", searchStr, searchStr, searchStr)
	}

	err := query.Find(&list).Error
	return list, err
}

func (r *anggotaRepository) Verify(anggotaID string, verifiedBy string) error {
	return r.db.Model(&domain.Anggota{}).Where("id = ?", anggotaID).Updates(map[string]interface{}{
		"is_verified": true,
		"verified_by": verifiedBy,
	}).Error
}

func (r *anggotaRepository) ToggleActive(anggotaID string, isActive bool) error {
	return r.db.Model(&domain.Anggota{}).Where("id = ?", anggotaID).Update("is_active", isActive).Error
}
