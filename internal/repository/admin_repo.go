package repository

import (
	"anggota.pelajarnumagetan.or.id/internal/domain"
	"gorm.io/gorm"
)

type AdminRepository interface {
	// Admin Unit
	CreateUnit(unit *domain.AdminUnit) error
	GetUnitByID(id string) (*domain.AdminUnit, error)
	ListUnits(role string, kecamatan string, hasActivePeriod bool) ([]domain.AdminUnit, error)

	// Admin User Mapping
	CreateAdminUser(admin *domain.AdminUser) error
	GetAdminUserBySSOUserID(ssoUserID string) (*domain.AdminUser, error)
	ListAdminUsers() ([]domain.AdminUser, error)
	DeleteAdminUser(ssoUserID string) error

	FindSSOUserIDByIdentifier(identifier string) (string, error)
	SearchAnggota(query string) ([]domain.Anggota, error)

	// Period
	CreatePeriod(period *domain.Period) error
	GetPeriodByID(id string) (*domain.Period, error)
	ListPeriods(unitID string) ([]domain.Period, error)
	SetActivePeriod(unitID string, periodID string) error
	GetActivePeriodByUnitID(unitID string) (*domain.Period, error)
}

type adminRepository struct {
	db *gorm.DB
}

func NewAdminRepository(db *gorm.DB) AdminRepository {
	return &adminRepository{db: db}
}

func (r *adminRepository) CreateUnit(unit *domain.AdminUnit) error {
	return r.db.Create(unit).Error
}

func (r *adminRepository) GetUnitByID(id string) (*domain.AdminUnit, error) {
	var unit domain.AdminUnit
	err := r.db.First(&unit, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &unit, nil
}

func (r *adminRepository) ListUnits(role string, kecamatan string, hasActivePeriod bool) ([]domain.AdminUnit, error) {
	var list []domain.AdminUnit
	query := r.db.Model(&domain.AdminUnit{})
	if role != "" {
		query = query.Where("role = ?", role)
	}
	if kecamatan != "" {
		query = query.Where("kecamatan = ?", kecamatan)
	}
	if hasActivePeriod {
		query = query.Joins("JOIN periods ON periods.admin_unit_id = admin_units.id").Where("periods.is_active = ?", true)
	}
	err := query.Find(&list).Error
	return list, err
}

func (r *adminRepository) CreateAdminUser(admin *domain.AdminUser) error {
	// Check if already exists to update, otherwise create
	var existing domain.AdminUser
	err := r.db.Where("sso_user_id = ?", admin.SSOUserID).First(&existing).Error
	if err == nil {
		existing.AdminUnitID = admin.AdminUnitID
		existing.Role = admin.Role
		err = r.db.Save(&existing).Error
	} else {
		err = r.db.Create(admin).Error
	}

	if err != nil {
		return err
	}

	// Update tabel anggota agar organisasi, pimpinan_unit_id sesuai, dan otomatis terverifikasi
	orgName := "Admin " + admin.Role
	return r.db.Table("anggota").Where("sso_user_id = ?", admin.SSOUserID).Updates(map[string]interface{}{
		"organisasi":       orgName,
		"pimpinan_unit_id": admin.AdminUnitID,
		"is_verified":      true,
		"verified_by":      "System Auto",
	}).Error
}

func (r *adminRepository) GetAdminUserBySSOUserID(ssoUserID string) (*domain.AdminUser, error) {
	var admin domain.AdminUser
	err := r.db.First(&admin, "sso_user_id = ?", ssoUserID).Error
	if err != nil {
		return nil, err
	}
	return &admin, nil
}

func (r *adminRepository) ListAdminUsers() ([]domain.AdminUser, error) {
	var list []domain.AdminUser
	err := r.db.Find(&list).Error
	return list, err
}

func (r *adminRepository) DeleteAdminUser(ssoUserID string) error {
	err := r.db.Where("sso_user_id = ?", ssoUserID).Delete(&domain.AdminUser{}).Error
	if err != nil {
		return err
	}

	// Reset organisasi anggota kembali ke default "Anggota"
	return r.db.Table("anggota").Where("sso_user_id = ?", ssoUserID).Updates(map[string]interface{}{
		"organisasi": "Anggota",
	}).Error
}

// FindSSOUserIDByIdentifier mencari sso_user_id dari tabel anggota berdasarkan email atau sso_user_id
func (r *adminRepository) FindSSOUserIDByIdentifier(identifier string) (string, error) {
	var result struct {
		SSOUserID string
	}

	// Cek apakah format UUID
	isUUID := len(identifier) == 36 && identifier[8] == '-'

	var err error
	if isUUID {
		// Cari berdasarkan sso_user_id
		err = r.db.Table("anggota").Select("sso_user_id").Where("sso_user_id = ?", identifier).Scan(&result).Error
	} else {
		// Cari berdasarkan email
		err = r.db.Table("anggota").Select("sso_user_id").Where("email = ?", identifier).Scan(&result).Error
	}

	if err != nil {
		return "", err
	}
	if result.SSOUserID == "" {
		return "", nil // tidak ditemukan, tapi bukan error DB
	}
	return result.SSOUserID, nil
}

// SearchAnggota mencari profile anggota secara lengkap berdasarkan nama, email, atau sso_user_id
func (r *adminRepository) SearchAnggota(queryStr string) ([]domain.Anggota, error) {
	var list []domain.Anggota
	likeQuery := "%" + queryStr + "%"

	err := r.db.Where("nama_lengkap ILIKE ? OR email ILIKE ? OR CAST(sso_user_id AS TEXT) ILIKE ?", likeQuery, likeQuery, likeQuery).
		Limit(10).
		Find(&list).Error

	return list, err
}

func (r *adminRepository) CreatePeriod(period *domain.Period) error {
	return r.db.Create(period).Error
}

func (r *adminRepository) GetPeriodByID(id string) (*domain.Period, error) {
	var period domain.Period
	err := r.db.First(&period, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func (r *adminRepository) ListPeriods(unitID string) ([]domain.Period, error) {
	var list []domain.Period
	err := r.db.Where("admin_unit_id = ?", unitID).Find(&list).Error
	return list, err
}

func (r *adminRepository) SetActivePeriod(unitID string, periodID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Set all active to false for this unit
		err := tx.Model(&domain.Period{}).Where("admin_unit_id = ?", unitID).Update("is_active", false).Error
		if err != nil {
			return err
		}
		// 2. Set the target period to active
		return tx.Model(&domain.Period{}).Where("id = ? AND admin_unit_id = ?", periodID, unitID).Update("is_active", true).Error
	})
}

func (r *adminRepository) GetActivePeriodByUnitID(unitID string) (*domain.Period, error) {
	var period domain.Period
	err := r.db.Where("admin_unit_id = ? AND is_active = ?", unitID, true).First(&period).Error
	if err != nil {
		return nil, err
	}
	return &period, nil
}
