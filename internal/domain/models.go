package domain

import (
	"time"
)

// AdminUnit - PC / PAC / PR / PK
type AdminUnit struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Role      string    `gorm:"type:varchar(20);not null" json:"role"` // Cabang / PAC / Ranting / PK
	NamaUnit  string    `gorm:"type:varchar(150);not null;uniqueIndex" json:"nama_unit"` // Contoh: "PAC Ngariboyo", "PR Balegondo", "PK MAN 1 Magetan"
	Kecamatan string    `gorm:"type:varchar(100);not null" json:"kecamatan"` // Untuk pengelompokan wilayah
	Periods   []Period  `gorm:"foreignKey:AdminUnitID;constraint:OnDelete:CASCADE" json:"periods,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// AdminUser - Memetakan SSO User ID ke Unit Kepengurusan lokal
type AdminUser struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SSOUserID   string    `gorm:"type:uuid;uniqueIndex;not null" json:"sso_user_id"`
	AdminUnitID string    `gorm:"type:uuid;not null" json:"admin_unit_id"`
	Role        string    `gorm:"type:varchar(20);not null" json:"role"` // Cabang / PAC / Ranting / PK
	CreatedAt   time.Time `json:"created_at"`
}

// Period - Masa bakti kepengurusan masing-masing unit
type Period struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Nama        string    `gorm:"type:varchar(50);not null" json:"nama"` // Contoh: "2025-2027", "2026-2027"
	AdminUnitID string    `gorm:"type:uuid;not null" json:"admin_unit_id"`
	IsActive    bool      `gorm:"default:false" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// Anggota - Profil personal anggota ternormalisasi (Boleh null saat onboarding awal)
type Anggota struct {
	ID             string              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SSOUserID      string              `gorm:"type:uuid;index" json:"sso_user_id"`
	FotoUrl        *string             `gorm:"type:varchar(255);default:null" json:"foto_url"`
	NamaLengkap    string              `gorm:"type:varchar(255);not null" json:"nama_lengkap"`
	Email          string              `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	NIK            *string             `gorm:"type:varchar(255);uniqueIndex;default:null" json:"nik"` // Terenkripsi AES
	NIA            *string             `gorm:"type:varchar(50);uniqueIndex;default:null" json:"nia"`
	JenisKelamin   *string             `gorm:"type:varchar(10);default:null" json:"jenis_kelamin"` // Laki-laki / Perempuan
	Organisasi     *string             `gorm:"type:varchar(50);default:null" json:"organisasi"`    // IPNU / IPPNU / Admin Cabang / PAC / Ranting / PK
	Phone          *string             `gorm:"type:varchar(20);default:null" json:"phone"`          // Format 62...
	TempatLahir    *string             `gorm:"type:varchar(100);default:null" json:"tempat_lahir"`
	TanggalLahir   *time.Time          `gorm:"type:date;default:null" json:"tanggal_lahir"`
	AlamatLengkap  *string             `gorm:"type:text;default:null" json:"alamat_lengkap"`
	
	PimpinanUnitID string              `gorm:"type:uuid;not null" json:"pimpinan_unit_id"`
	PeriodeMasukID string              `gorm:"type:uuid;not null" json:"periode_masuk_id"`
	
	RFID           *string             `gorm:"type:varchar(100);default:null" json:"rfid"`
	Pekerjaan      *string             `gorm:"type:varchar(100);default:null" json:"pekerjaan"`
	HobiMinat      *string             `gorm:"type:varchar(255);default:null" json:"hobi_minat"`
	
	IsVerified     bool                `gorm:"default:false" json:"is_verified"`
	VerifiedBy     *string             `gorm:"type:varchar(100);default:null" json:"verified_by"`
	IsActive       bool                `gorm:"default:true" json:"is_active"`
	
	Pendidikan     []RiwayatPendidikan `gorm:"foreignKey:AnggotaID;constraint:OnDelete:CASCADE" json:"pendidikan,omitempty"`
	Perkaderan     []RiwayatPerkaderan `gorm:"foreignKey:AnggotaID;constraint:OnDelete:CASCADE" json:"perkaderan,omitempty"`
	JabatanHistory []RiwayatJabatan    `gorm:"foreignKey:AnggotaID;constraint:OnDelete:CASCADE" json:"jabatan_history,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
}

// RiwayatJabatan - Jabatan struktural anggota di periode tertentu
type RiwayatJabatan struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AnggotaID   string    `gorm:"type:uuid;not null" json:"anggota_id"`
	PeriodID    string    `gorm:"type:uuid;not null" json:"period_id"`
	AdminUnitID string    `gorm:"type:uuid;not null" json:"admin_unit_id"`
	Jabatan     string    `gorm:"type:varchar(100);not null" json:"jabatan"` // Contoh: Ketua, Sekretaris
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// RiwayatPendidikan - Riwayat pendidikan sekolah/kampus
type RiwayatPendidikan struct {
	ID          string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AnggotaID   string    `gorm:"type:uuid;not null" json:"anggota_id"`
	Jenjang     string    `gorm:"type:varchar(10);not null" json:"jenjang"` // SD, MI, SMP, MTs, SMA, SMK, MA, PT
	NamaSekolah string    `gorm:"type:varchar(255);not null" json:"nama_sekolah"`
	CreatedAt   time.Time `json:"created_at"`
}

// RiwayatPerkaderan - Riwayat perkaderan formal IPNU-IPPNU & CBP-KPP
type RiwayatPerkaderan struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AnggotaID string    `gorm:"type:uuid;not null" json:"anggota_id"`
	Nama      string    `gorm:"type:varchar(50);not null" json:"nama"` // Makesta, Lakmud, Lakut, Diklatama, Diklatmad, latin, latpel
	Tanggal   time.Time `gorm:"type:date" json:"tanggal"`
	Tempat    string    `gorm:"type:varchar(255)" json:"tempat"`
	CreatedAt time.Time `json:"created_at"`
}
