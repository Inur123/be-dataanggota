package database

import (
	"fmt"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"anggota.pelajarnumagetan.or.id/internal/config"
)

var DB *gorm.DB

func ConnectPostgres() *gorm.DB {
	cfg := config.Get()

	var dsn string
	if cfg.DBPassword != "" {
		dsn = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s&TimeZone=Asia/Jakarta",
			cfg.DBUser,
			cfg.DBPassword,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
			cfg.DBSslMode,
		)
	} else {
		dsn = fmt.Sprintf(
			"postgres://%s@%s:%s/%s?sslmode=%s&TimeZone=Asia/Jakarta",
			cfg.DBUser,
			cfg.DBHost,
			cfg.DBPort,
			cfg.DBName,
			cfg.DBSslMode,
		)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	DB = db
	return db
}
