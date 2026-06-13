package main

import (
	"log"
	"net/http"
	"time"

	"anggota.pelajarnumagetan.or.id/internal/config"
	"anggota.pelajarnumagetan.or.id/internal/database"
	"anggota.pelajarnumagetan.or.id/internal/domain"
	"anggota.pelajarnumagetan.or.id/internal/handler"
	"anggota.pelajarnumagetan.or.id/internal/middleware"
	"anggota.pelajarnumagetan.or.id/internal/repository"
	"anggota.pelajarnumagetan.or.id/internal/service"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Load Config
	cfg := config.Load()

	// 2. Connect Database
	db := database.ConnectPostgres()
	database.ConnectRedis()

	// 3. Auto Migration GORM
	log.Println("Running GORM auto-migrations...")
	err := db.AutoMigrate(
		&domain.AdminUnit{},
		&domain.AdminUser{},
		&domain.Period{},
		&domain.Anggota{},
		&domain.RiwayatPendidikan{},
		&domain.RiwayatPerkaderan{},
		&domain.RiwayatJabatan{},
	)
	if err != nil {
		log.Fatalf("Auto-migration failed: %v", err)
	}
	log.Println("Auto-migrations completed successfully.")



	// 4. Inisialisasi Repositories, Services, Handlers
	adminRepo := repository.NewAdminRepository(db)
	anggotaRepo := repository.NewAnggotaRepository(db)

	adminSrv := service.NewAdminService(adminRepo)
	anggotaSrv := service.NewAnggotaService(anggotaRepo, adminRepo)

	adminHandler := handler.NewAdminHandler(adminSrv)
	anggotaHandler := handler.NewAnggotaHandler(anggotaSrv)

	// 5. Inisialisasi Echo Web Server
	e := echo.New()

	// Middlewares
	e.Use(echoMiddleware.LoggerWithConfig(echoMiddleware.LoggerConfig{
		Format: "⇨ [DataAnggota] ${time_custom} | ${status} | ${latency_human} | ${remote_ip} | ${method} ${uri}\n",
		CustomTimeFormat: "2006-01-02 15:04:05",
	}))
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
	}))



	// 6. Routing
	// Public Routes
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status": "up",
			"time":   time.Now().Format(time.RFC3339),
			"system": "DataAnggota BE",
		})
	})
	e.GET("/v1/public/units", adminHandler.ListUnits)
	e.GET("/v1/public/periods/:unit_id", adminHandler.ListPeriods)

	// Protected Routes (Wajib Login SSO)
	api := e.Group("/v1")
	api.Use(middleware.Auth())

	// Member/Anggota Profiling
	api.POST("/anggota/onboard", anggotaHandler.Onboard)
	api.GET("/anggota/me", anggotaHandler.GetProfile)
	api.PUT("/anggota/me", anggotaHandler.UpdateProfile)

	// Admin Management (Wajib memiliki role Admin di local DB atau Superadmin)
	adm := api.Group("/admin")
	adm.Use(middleware.RequireAdmin())

	adm.GET("/my-unit", adminHandler.GetMyUnit)

	// Manage Units & Periods
	adm.POST("/units", adminHandler.CreateUnit)
	adm.GET("/units", adminHandler.ListUnits)
	adm.POST("/periods", adminHandler.CreatePeriod)
	adm.GET("/periods/:unit_id", adminHandler.ListPeriods)
	adm.PUT("/periods/:unit_id/active", adminHandler.SetActivePeriod)

	// Manage Admin Users
	adm.GET("/users/search", adminHandler.SearchSSOUser)
	adm.POST("/users", adminHandler.CreateAdminUser)
	adm.GET("/users", adminHandler.ListAdminUsers)
	adm.DELETE("/users/:sso_user_id", adminHandler.DeleteAdminUser)

	// Manage Anggota Data
	adm.GET("/anggota", anggotaHandler.List)
	adm.POST("/anggota/:id/verify", anggotaHandler.Verify)
	adm.PUT("/anggota/:id/toggle-active", anggotaHandler.ToggleActive)

	// 7. Start Server
	port := cfg.AppPort
	log.Printf("🚀 DataAnggota Backend running on port :%s", port)
	e.Logger.Fatal(e.Start(":" + port))
}
