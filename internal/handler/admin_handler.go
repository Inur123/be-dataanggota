package handler

import (
	"log"

	"anggota.pelajarnumagetan.or.id/internal/domain"
	"anggota.pelajarnumagetan.or.id/internal/middleware"
	"anggota.pelajarnumagetan.or.id/internal/service"
	"anggota.pelajarnumagetan.or.id/internal/utils"
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	srv service.AdminService
}

func NewAdminHandler(srv service.AdminService) *AdminHandler {
	return &AdminHandler{srv: srv}
}

func (h *AdminHandler) CreateUnit(c echo.Context) error {
	var unit domain.AdminUnit
	if err := c.Bind(&unit); err != nil {
		return utils.BadRequest(c, "Data unit tidak valid")
	}

	err := h.srv.CreateUnit(&unit)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.Created(c, "Unit kepemimpinan berhasil dibuat", unit)
}

func (h *AdminHandler) ListUnits(c echo.Context) error {
	role := c.QueryParam("role")
	kecamatan := c.QueryParam("kecamatan")
	hasActivePeriod := c.QueryParam("has_active_period") == "true"

	list, err := h.srv.ListUnits(role, kecamatan, hasActivePeriod)
	if err != nil {
		return utils.InternalError(c, "Gagal memuat unit")
	}

	return utils.OK(c, "Berhasil memuat unit", list)
}

func (h *AdminHandler) CreateAdminUser(c echo.Context) error {
	var req struct {
		Role        string `json:"role"` // Cabang / PAC / PR / PK
		AdminUnitID string `json:"admin_unit_id"`
		Identifier  string `json:"identifier"` // Email / UUID
	}

	if err := c.Bind(&req); err != nil {
		log.Printf("[CreateAdminUser] Bind error: %v", err)
		return utils.BadRequest(c, "Data admin tidak valid")
	}

	err := h.srv.CreateAdminUser(req.Role, req.AdminUnitID, req.Identifier)
	if err != nil {
		log.Printf("[CreateAdminUser] Service error: %v", err)
		return utils.BadRequest(c, err.Error())
	}

	return utils.Created(c, "Akses admin berhasil ditambahkan", nil)
}

func (h *AdminHandler) ListAdminUsers(c echo.Context) error {
	list, err := h.srv.ListAdminUsers()
	if err != nil {
		return utils.InternalError(c, "Gagal memuat admin users")
	}
	return utils.OK(c, "Berhasil memuat admin users", list)
}

func (h *AdminHandler) DeleteAdminUser(c echo.Context) error {
	ssoUserID := c.Param("sso_user_id")

	err := h.srv.DeleteAdminUser(ssoUserID)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Admin berhasil dihapus", nil)
}

func (h *AdminHandler) CreatePeriod(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)
	isSuperAdmin := user.Role == "superadmin"
	var req struct {
		Nama        string `json:"nama"`
		AdminUnitID string `json:"admin_unit_id"`
	}
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Data tidak valid")
	}

	err := h.srv.CreatePeriod(req.Nama, req.AdminUnitID, user.ID, isSuperAdmin)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.Created(c, "Periode masa bakti berhasil dibuat", nil)
}

func (h *AdminHandler) ListPeriods(c echo.Context) error {
	unitID := c.Param("unit_id")

	list, err := h.srv.ListPeriods(unitID)
	if err != nil {
		return utils.InternalError(c, "Gagal memuat periode")
	}

	return utils.OK(c, "Berhasil memuat periode", list)
}

func (h *AdminHandler) SetActivePeriod(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)
	isSuperAdmin := user.Role == "superadmin"
	unitID := c.Param("unit_id")
	var req struct {
		PeriodID string `json:"period_id"`
	}
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Data tidak valid")
	}

	err := h.srv.SetActivePeriod(unitID, req.PeriodID, user.ID, isSuperAdmin)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.OK(c, "Periode aktif berhasil diperbarui", nil)
}

func (h *AdminHandler) GetMyUnit(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)

	admin, err := h.srv.GetAdminUserBySSOUserID(user.ID)
	if err != nil {
		if user.Role == "superadmin" {
			// Superadmin fallback
			return utils.OK(c, "Superadmin", map[string]interface{}{
				"sso_user_id":   user.ID,
				"admin_unit_id": "",
				"role":          "Superadmin",
			})
		}
		return utils.NotFound(c, "Anda tidak terdaftar sebagai admin unit manapun")
	}

	return utils.OK(c, "Berhasil memuat unit admin", admin)
}

func (h *AdminHandler) SearchSSOUser(c echo.Context) error {
	query := c.QueryParam("query")
	if query == "" {
		return utils.BadRequest(c, "Query pencarian wajib diisi")
	}

	list, err := h.srv.SearchAnggota(query)
	if err != nil {
		return utils.InternalError(c, "Gagal memproses pencarian: "+err.Error())
	}

	results := make([]map[string]interface{}, 0)
	for _, anggota := range list {
		results = append(results, map[string]interface{}{
			"id":           anggota.ID,
			"sso_user_id":  anggota.SSOUserID,
			"nama_lengkap": anggota.NamaLengkap,
			"email":        anggota.Email,
		})
	}

	return utils.OK(c, "Hasil pencarian", results)
}
