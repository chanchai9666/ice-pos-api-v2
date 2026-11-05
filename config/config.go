package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName               string
	AppEnv                string
	AppPort               string
	JWTSecret             string
	JWTAccessTTL          time.Duration
	JWTRefreshTTL         time.Duration
	DBType                string // sqlite | postgres
	DBPath                string // for sqlite
	DBHost                string
	DBPort                string
	DBUser                string
	DBPassword            string
	DBName                string
	DBSSLMode             string
	DBSchema              string
	AllowOrigin           string
	UploadDir             string
	UploadBaseURL         string
	StorageDriver         string
	ImageKitPublicKey     string
	ImageKitPrivateKey    string
	ImageKitURLEndpoint   string
	ImageKitFolder        string
	AIEmbeddingServiceURL string
	VisionModelPath       string
	VisionSharedLibPath   string
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	dbType := getEnv("DB_TYPE", "")
	if dbType == "" {
		dbType = getEnv("DB_DRIVER", "sqlite")
	}

	allowOrigin := getEnv("ALLOW_ORIGIN", "")
	if allowOrigin == "" {
		allowOrigin = getEnv("ALLOW_ORIGINS", "*")
	}

	cfg := &Config{
		AppName:               getEnv("APP_NAME", "ICE POS API v2"),
		AppEnv:                getEnv("APP_ENV", "development"),
		AppPort:               getEnv("APP_PORT", "8080"),
		JWTSecret:             getEnv("JWT_SECRET", "ice-pos-v2-super-secret-key-change-in-prod"),
		JWTAccessTTL:          24 * time.Hour,
		JWTRefreshTTL:         7 * 24 * time.Hour,
		DBType:                dbType,
		DBPath:                getEnv("DB_PATH", "ice_pos_v2.db"),
		DBHost:                getEnv("DB_HOST", "localhost"),
		DBPort:                getEnv("DB_PORT", "5432"),
		DBUser:                getEnv("DB_USER", "postgres"),
		DBPassword:            getEnv("DB_PASSWORD", "postgres"),
		DBName:                getEnv("DB_NAME", "ice_pos_v2"),
		DBSSLMode:             getEnv("DB_SSLMODE", "disable"),
		DBSchema:              getEnv("DB_SCHEMA", "public"),
		AllowOrigin:           allowOrigin,
		UploadDir:             getEnv("UPLOAD_DIR", "./uploads"),
		UploadBaseURL:         getEnv("UPLOAD_BASE_URL", "/uploads"),
		StorageDriver:         getEnv("STORAGE_DRIVER", "local"),
		ImageKitPublicKey:     getEnv("IMAGEKIT_PUBLIC_KEY", ""),
		ImageKitPrivateKey:    getEnv("IMAGEKIT_PRIVATE_KEY", ""),
		ImageKitURLEndpoint:   getEnv("IMAGEKIT_URL_ENDPOINT", ""),
		ImageKitFolder:        getEnv("IMAGEKIT_FOLDER", "pos"),
		AIEmbeddingServiceURL: getEnv("AI_EMBEDDING_SERVICE_URL", "http://localhost:5001"),
		VisionModelPath:       getEnv("VISION_MODEL_PATH", "models/clip_vision_model.onnx"),
		VisionSharedLibPath:   getEnv("ONNXRUNTIME_SHARED_LIBRARY_PATH", ""),
	}

	return cfg
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
