package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorageConfig struct {
	UploadDir string
	BaseURL   string
}

type LocalStorageService struct {
	cfg LocalStorageConfig
}

func NewLocalStorageService(cfg LocalStorageConfig) *LocalStorageService {
	return &LocalStorageService{
		cfg: cfg,
	}
}

func (s *LocalStorageService) Upload(ctx context.Context, input UploadInput) (*UploadOutput, error) {
	targetDir := filepath.Join(s.cfg.UploadDir, "images")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory: %w", err)
	}

	destPath := filepath.Join(targetDir, input.Filename)
	if err := os.WriteFile(destPath, input.Data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write local file: %w", err)
	}

	baseURL := strings.TrimRight(s.cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "/uploads"
	}
	filePath := fmt.Sprintf("%s/images/%s", baseURL, input.Filename)

	return &UploadOutput{
		FileID:      input.Filename,
		Filename:    input.Filename,
		URL:         filePath,
		Size:        int64(len(input.Data)),
		ContentType: input.ContentType,
	}, nil
}
