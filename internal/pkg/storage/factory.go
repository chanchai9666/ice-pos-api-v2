package storage

import (
	"log/slog"
	"strings"

	"ice-pos-api-v2/config"
)

func NewStorageService(cfg *config.Config) StorageService {
	driver := strings.ToLower(strings.TrimSpace(cfg.StorageDriver))

	if driver == "imagekit" && cfg.ImageKitPrivateKey != "" {
		slog.Info("Initializing Storage: ImageKit Provider")
		return NewImageKitStorageService(ImageKitConfig{
			PublicKey:   cfg.ImageKitPublicKey,
			PrivateKey:  cfg.ImageKitPrivateKey,
			URLEndpoint: cfg.ImageKitURLEndpoint,
			Folder:      cfg.ImageKitFolder,
		})
	}

	if driver == "imagekit" && cfg.ImageKitPrivateKey == "" {
		slog.Warn("STORAGE_DRIVER is 'imagekit' but IMAGEKIT_PRIVATE_KEY is not set. Falling back to Local Storage.")
	} else {
		slog.Info("Initializing Storage: Local Disk Provider", "upload_dir", cfg.UploadDir)
	}

	return NewLocalStorageService(LocalStorageConfig{
		UploadDir: cfg.UploadDir,
		BaseURL:   cfg.UploadBaseURL,
	})
}
