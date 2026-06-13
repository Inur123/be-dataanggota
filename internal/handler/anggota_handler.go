package handler

import (
	"fmt"

	"anggota.pelajarnumagetan.or.id/internal/domain"
	"anggota.pelajarnumagetan.or.id/internal/middleware"
	"anggota.pelajarnumagetan.or.id/internal/service"
	"anggota.pelajarnumagetan.or.id/internal/utils"
	"github.com/labstack/echo/v4"
)

type AnggotaHandler struct {
	srv service.AnggotaService
}

func NewAnggotaHandler(srv service.AnggotaService) *AnggotaHandler {
	return &AnggotaHandler{srv: srv}
}

func (h *AnggotaHandler) Onboard(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)

	var req struct {
		PimpinanUnitID string `json:"pimpinan_unit_id"`
	}
	if err := c.Bind(&req); err != nil {
		return utils.BadRequest(c, "Request body tidak valid")
	}

	err := h.srv.Onboard(user.ID, user.Name, user.Email, req.PimpinanUnitID)
	if err != nil {
		return utils.BadRequest(c, err.Error())
	}

	return utils.Created(c, "Onboarding berhasil, pimpinan unit terpilih", nil)
}

func (h *AnggotaHandler) GetProfile(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)

	profile, err := h.srv.GetProfile(user.ID)
	if err != nil {
		return utils.NotFound(c, "Profil anggota belum diisi atau tidak ditemukan")
	}

	return utils.OK(c, "Berhasil memuat profil", profile)
}

func (h *AnggotaHandler) UpdateProfile(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)

	var input domain.Anggota
	if err := c.Bind(&input); err != nil {
		return utils.BadRequest(c, "Data tidak valid")
	}

	// Logging received request payload
	println(">>> UPDATE PROFILE REQUEST RECEIVED for SSO User ID:", user.ID)
	if input.Pendidikan != nil {
		fmt.Printf(">>> Received Pendidikan: %d items\n", len(input.Pendidikan))
		for idx, edu := range input.Pendidikan {
			fmt.Printf("    [%d] ID: %s | Jenjang: %s | NamaSekolah: %s\n", idx, edu.ID, edu.Jenjang, edu.NamaSekolah)
		}
	} else {
		println(">>> Received Pendidikan: NIL")
	}

	err := h.srv.UpdateProfile(user.ID, &input)
	if err != nil {
		println(">>> UPDATE PROFILE FAILED:", err.Error())
		return utils.BadRequest(c, err.Error())
	}

	println(">>> UPDATE PROFILE SUCCESSFUL")
	return utils.OK(c, "Profil berhasil diperbarui", nil)
}

func (h *AnggotaHandler) List(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)
	isSuperAdmin := user.Role == "superadmin"

	pimpinanUnitID := c.QueryParam("pimpinan_unit_id")
	periodeMasukID := c.QueryParam("periode_masuk_id")

	filter := map[string]interface{}{
		"organisasi":       c.QueryParam("organisasi"),
		"jenis_kelamin":    c.QueryParam("jenis_kelamin"),
		"pimpinan_unit_id": pimpinanUnitID,
		"periode_masuk_id": periodeMasukID,
		"search":           c.QueryParam("search"),
	}

	list, err := h.srv.List(filter, user.ID, isSuperAdmin)
	if err != nil {
		return utils.Forbidden(c, err.Error())
	}

	return utils.OK(c, "Berhasil memuat daftar anggota", list)
}

func (h *AnggotaHandler) Verify(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)
	isSuperAdmin := user.Role == "superadmin"
	anggotaID := c.Param("id")

	err := h.srv.Verify(anggotaID, user.ID, isSuperAdmin)
	if err != nil {
		return utils.Forbidden(c, err.Error())
	}

	return utils.OK(c, "Anggota berhasil diverifikasi", nil)
}

func (h *AnggotaHandler) ToggleActive(c echo.Context) error {
	user := c.Get("current_user").(middleware.UserClaims)
	isSuperAdmin := user.Role == "superadmin"
	anggotaID := c.Param("id")

	newActiveStatus, err := h.srv.ToggleActive(anggotaID, user.ID, isSuperAdmin)
	if err != nil {
		return utils.Forbidden(c, err.Error())
	}

	statusStr := "dinonaktifkan"
	if newActiveStatus {
		statusStr = "diaktifkan kembali"
	}
	return utils.OK(c, fmt.Sprintf("Anggota berhasil %s", statusStr), map[string]interface{}{
		"is_active": newActiveStatus,
	})
}
