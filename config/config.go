package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName       string
	AppEnv        string
	AppPort       string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	DBType        string // sqlite | postgres
	DBPath        string // for sqlite
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	AllowOrigin   string
	UploadDir     string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppName:       getEnv("APP_NAME", "ICE POS API v2"),
		AppEnv:        getEnv("APP_ENV", "development"),
		AppPort:       getEnv("APP_PORT", "8080"),
		JWTSecret:     getEnv("JWT_SECRET", "ice-pos-v2-super-secret-key-change-in-prod"),
		JWTAccessTTL:  24 * time.Hour,
		JWTRefreshTTL: 7 * 24 * time.Hour,
		DBType:        getEnv("DB_TYPE", "sqlite"),
		DBPath:        getEnv("DB_PATH", "ice_pos_v2.db"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "postgres"),
		DBName:        getEnv("DB_NAME", "ice_pos_v2"),
		DBSSLMode:     getEnv("DB_SSLMODE", "disable"),
		AllowOrigin:   getEnv("ALLOW_ORIGIN", "*"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
