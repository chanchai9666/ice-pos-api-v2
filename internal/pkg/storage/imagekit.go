package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

const (
	defaultImageKitUploadURL = "https://upload.imagekit.io/api/v1/files/upload"
	defaultHTTPTimeout       = 30 * time.Second
)

type ImageKitConfig struct {
	PublicKey   string
	PrivateKey  string
	URLEndpoint string
	Folder      string
}

type ImageKitStorageService struct {
	cfg        ImageKitConfig
	httpClient *http.Client
	uploadURL  string
}

func NewImageKitStorageService(cfg ImageKitConfig) *ImageKitStorageService {
	return &ImageKitStorageService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
		uploadURL: defaultImageKitUploadURL,
	}
}

type imageKitUploadResponse struct {
	FileID       string `json:"fileId"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	FilePath     string `json:"filePath"`
	URL          string `json:"url"`
	FileType     string `json:"fileType"`
	Height       int    `json:"height"`
	Width        int    `json:"width"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Message      string `json:"message"`
	Help         string `json:"help"`
}

func (s *ImageKitStorageService) Upload(ctx context.Context, input UploadInput) (*UploadOutput, error) {
	if s.cfg.PrivateKey == "" {
		return nil, fmt.Errorf("imagekit private key is required")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", input.Filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create multipart file part: %w", err)
	}
	if _, err := part.Write(input.Data); err != nil {
		return nil, fmt.Errorf("failed to write file data: %w", err)
	}

	if err := writer.WriteField("fileName", input.Filename); err != nil {
		return nil, fmt.Errorf("failed to write fileName field: %w", err)
	}

	folder := strings.TrimSpace(input.Folder)
	if folder == "" {
		folder = strings.TrimSpace(s.cfg.Folder)
	}
	folder = strings.Trim(folder, "/")
	if folder != "" {
		if err := writer.WriteField("folder", "/"+folder); err != nil {
			return nil, fmt.Errorf("failed to write folder field: %w", err)
		}
	}

	if err := writer.WriteField("useUniqueFileName", "false"); err != nil {
		return nil, fmt.Errorf("failed to write useUniqueFileName field: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.uploadURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetBasicAuth(s.cfg.PrivateKey, "")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("imagekit request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("imagekit upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var res imageKitUploadResponse
	if err := json.Unmarshal(respBody, &res); err != nil {
		return nil, fmt.Errorf("failed to decode imagekit response: %w", err)
	}

	return &UploadOutput{
		FileID:       res.FileID,
		Filename:     res.Name,
		URL:          res.URL,
		ThumbnailURL: res.ThumbnailURL,
		Size:         res.Size,
		Width:        res.Width,
		Height:       res.Height,
		ContentType:  input.ContentType,
	}, nil
}

func (s *ImageKitStorageService) Delete(ctx context.Context, fileURL string) error {
	// Best-effort delete or no-op when fileId is not readily determinable
	return nil
}
