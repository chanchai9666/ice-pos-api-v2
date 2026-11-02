package storage

import (
	"context"
)

type UploadInput struct {
	Filename    string
	ContentType string
	Data        []byte
	Folder      string
}

type UploadOutput struct {
	FileID       string `json:"file_id"`
	Filename     string `json:"file_name"`
	URL          string `json:"url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Size         int64  `json:"size"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
}

type StorageService interface {
	Upload(ctx context.Context, input UploadInput) (*UploadOutput, error)
}
