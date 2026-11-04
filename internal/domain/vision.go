package domain

import (
	"context"
)

// VisionMatchItem represents a matched product candidate from a vision query.
type VisionMatchItem struct {
	ProductID      string  `json:"product_id"`
	ProductCode    string  `json:"product_code"`
	Barcode        string  `json:"barcode"`
	ProductName    string  `json:"product_name"`
	Price          int64   `json:"price"` // Satang
	ImageURL       string  `json:"image_url"`
	Confidence     float64 `json:"confidence"` // Cosine similarity (0.0 - 1.0)
	EmbeddingModel string  `json:"embedding_model,omitempty"`
}

// VisionMatchResponse contains the match results for a vision query.
type VisionMatchResponse struct {
	QueryModel string            `json:"query_model"`
	Matches    []VisionMatchItem `json:"matches"`
}

// VisionEmbeddingService defines the interface for converting image bytes to a vector embedding.
type VisionEmbeddingService interface {
	ExtractEmbedding(ctx context.Context, imageBytes []byte) (FloatVector, string, error)
}

// VisionUsecase defines operations for server-side visual AI recognition and vector management.
type VisionUsecase interface {
	MatchFrame(ctx context.Context, imageBytes []byte, threshold float64, limit int) (*VisionMatchResponse, error)
	ReindexImages(ctx context.Context) (int, error)
}
