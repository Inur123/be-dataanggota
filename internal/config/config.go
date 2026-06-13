package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort    string
	AppEnv     string
	AppName    string
	AppURL     string
	APIURL     string

	// Postgres DB
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSslMode  string

	// Redis Cache
	RedisHost     string
	RedisPort     string
	RedisPassword string

	// Security
	EncryptionKey string

	// SSO OAuth
	SSOClientID     string
	SSOClientSecret string
	SSOCallbackURL  string
	SSOValidateURL  string
}

var cfg *Config

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	cfg = &Config{
		AppPort:         getEnv("APP_PORT", "8081"),
		AppEnv:          getEnv("APP_ENV", "development"),
		AppName:         getEnv("APP_NAME", "Data Anggota Pelajar NU Magetan"),
		AppURL:          getEnv("APP_URL", "http://localhost:3000"),
		APIURL:          getEnv("API_URL", "http://localhost:8081"),
		DBHost:          getEnv("DB_HOST", "127.0.0.1"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", "anggota_pelajarnu"),
		DBSslMode:       getEnv("DB_SSLMODE", "disable"),
		RedisHost:       getEnv("REDIS_HOST", "127.0.0.1"),
		RedisPort:       getEnv("REDIS_PORT", "6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		EncryptionKey:   getEnv("ENCRYPTION_KEY", "1d15058c8ea6c89aa1c3ebc960aef1fc723f83bc2200c9357b047e12eea530d8"),
		SSOClientID:     getEnv("SSO_CLIENT_ID", "fcd16d470278ec22db4df94740198b10"),
		SSOClientSecret: getEnv("SSO_CLIENT_SECRET", "e4dc708be02a982f3a63be8b795f7afc39f7ba4cb7a770cae5e8d4d1380c1b00"),
		SSOCallbackURL:  getEnv("SSO_CALLBACK_URL", "http://localhost:3000/api/auth/callback/sso"),
		SSOValidateURL:  getEnv("SSO_VALIDATE_URL", "http://localhost:8080/v1/auth/validate"),
	}

	return cfg
}

func Get() *Config {
	if cfg == nil {
		return Load()
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

