package vision

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

	"ice-pos-api-v2/internal/domain"
)

type Client struct {
	serviceURL string
	httpClient *http.Client
}

type embedResponse struct {
	Embedding domain.FloatVector `json:"embedding"`
	Model     string             `json:"model"`
	Error     string             `json:"error,omitempty"`
}

// NewClient creates a new HTTP client for the Vision AI embedding microservice.
func NewClient(serviceURL string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		serviceURL: strings.TrimRight(strings.TrimSpace(serviceURL), "/"),
		httpClient: &http.Client{Timeout: timeout},
	}
}

// ExtractEmbedding sends image bytes to the Vision AI service and returns the embedding vector and model name.
func (c *Client) ExtractEmbedding(ctx context.Context, imageBytes []byte) (domain.FloatVector, string, error) {
	if c.serviceURL == "" {
		return nil, "", domain.NewAppError(http.StatusServiceUnavailable, "vision AI embedding service is not configured")
	}

	if len(imageBytes) == 0 {
		return nil, "", domain.NewAppError(http.StatusBadRequest, "empty image bytes provided for embedding extraction")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create multipart form file: %w", err)
	}

	if _, err := part.Write(imageBytes); err != nil {
		return nil, "", fmt.Errorf("failed to write image bytes to multipart form: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	endpoint := c.serviceURL + "/embed"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to connect to vision AI service (%s): %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response from vision AI service: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("vision AI service returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var res embedResponse
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return nil, "", fmt.Errorf("failed to decode vision AI response: %w", err)
	}

	if len(res.Embedding) == 0 {
		return nil, "", fmt.Errorf("vision AI service returned empty embedding vector")
	}

	return res.Embedding, res.Model, nil
}
