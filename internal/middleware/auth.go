package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"anggota.pelajarnumagetan.or.id/internal/config"
	"anggota.pelajarnumagetan.or.id/internal/database"
	"anggota.pelajarnumagetan.or.id/internal/utils"
	"github.com/labstack/echo/v4"
)

type UserClaims struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type ssoUserResponse struct {
	Success bool `json:"success"`
	Data    struct {
		User UserClaims `json:"user"`
	} `json:"data"`
}

func Auth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cfg := config.Get()

			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return utils.Unauthorized(c, "Token tidak ditemukan")
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return utils.Unauthorized(c, "Format token tidak valid")
			}

			token := parts[1]

			// Validasi token via GET /v1/user/me di SSO
			ssoUserURL := fmt.Sprintf("%s/v1/user/me", strings.TrimRight(cfg.SSOValidateURL, "/"))
			req, err := http.NewRequest("GET", ssoUserURL, nil)
			if err != nil {
				return utils.InternalError(c, "Gagal memproses validasi token")
			}
			req.Header.Set("Authorization", "Bearer "+token)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return utils.InternalError(c, "Gagal menghubungi server SSO")
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return utils.Unauthorized(c, "Token tidak valid atau sudah expired")
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return utils.InternalError(c, "Gagal membaca respon SSO")
			}

			// SSO /v1/user/me returns { success, message, data: { id, name, email, role, ... } }
			var meResp struct {
				Success bool `json:"success"`
				Data    UserClaims `json:"data"`
			}
			if err := json.Unmarshal(body, &meResp); err != nil || !meResp.Success {
				return utils.Unauthorized(c, "Token tidak valid atau sudah expired")
			}

			c.Set("current_user", meResp.Data)
			return next(c)
		}
	}
}

// RequireRole checks if the SSO user is the default superadmin or mapped admin
func RequireAdmin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user, ok := c.Get("current_user").(UserClaims)
			if !ok {
				return utils.Forbidden(c, "Akses ditolak")
			}

			// 1. Bypass untuk SSO superadmin
			if user.Role == "superadmin" {
				return next(c)
			}

			// 2. Cek apakah user terdaftar sebagai admin unit secara lokal
			var count int64
			err := database.DB.Table("admin_users").Where("sso_user_id = ?", user.ID).Count(&count).Error
			if err != nil || count == 0 {
				return utils.Forbidden(c, "Akses ditolak: anda tidak memiliki wewenang admin")
			}

			return next(c)
		}
	}
}
