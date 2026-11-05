package usecase

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/calc"
)

type visionUsecase struct {
	catalogRepo   domain.CatalogRepository
	visionService domain.VisionEmbeddingService
}

func NewVisionUsecase(catalogRepo domain.CatalogRepository, visionService domain.VisionEmbeddingService) domain.VisionUsecase {
	return &visionUsecase{
		catalogRepo:   catalogRepo,
		visionService: visionService,
	}
}

func (u *visionUsecase) MatchFrame(ctx context.Context, imageBytes []byte, threshold float64, limit int) (*domain.VisionMatchResponse, error) {
	if u.visionService == nil {
		return nil, domain.NewAppError(http.StatusServiceUnavailable, "vision AI embedding service is not configured")
	}

	if len(imageBytes) == 0 {
		return nil, domain.NewAppError(http.StatusBadRequest, "empty image frame")
	}

	if threshold <= 0 {
		threshold = 0.70
	}
	if limit <= 0 {
		limit = 5
	}

	queryEmbedding, queryModel, err := u.visionService.ExtractEmbedding(ctx, imageBytes)
	if err != nil {
		return nil, domain.NewAppError(http.StatusServiceUnavailable, fmt.Sprintf("ไม่สามารถสกัด Vector จากภาพได้ (กรุณาตรวจสอบว่า Vision AI Service พอร์ต 5001 เปิดใช้งานแล้วหรือไม่): %v", err))
	}

	productImages, err := u.catalogRepo.ListAllProductImagesWithEmbeddings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list product images: %w", err)
	}

	// Calculate similarity and deduplicate by product ID (keep highest confidence per product)
	bestScoreByProduct := make(map[string]struct {
		score float64
		image domain.ProductImage
	})

	for _, img := range productImages {
		if len(img.Embedding) == 0 {
			continue
		}
		score := calc.CosineSimilarity(queryEmbedding, img.Embedding)
		if score < threshold {
			continue
		}

		if existing, ok := bestScoreByProduct[img.ProductID]; !ok || score > existing.score {
			bestScoreByProduct[img.ProductID] = struct {
				score float64
				image domain.ProductImage
			}{score: score, image: img}
		}
	}

	if len(bestScoreByProduct) == 0 {
		return &domain.VisionMatchResponse{
			QueryModel: queryModel,
			Matches:    []domain.VisionMatchItem{},
		}, nil
	}

	// Retrieve product details
	var matches []domain.VisionMatchItem
	for prodID, best := range bestScoreByProduct {
		prod, err := u.catalogRepo.GetProductByID(ctx, prodID)
		if err != nil {
			continue
		}
		imgURL := best.image.ImageURL
		if imgURL == "" {
			imgURL = prod.ImageURL
		}

		matches = append(matches, domain.VisionMatchItem{
			ProductID:      prod.ID,
			ProductCode:    prod.Code,
			Barcode:        prod.Barcode,
			ProductName:    prod.Name,
			Price:          prod.Price,
			ImageURL:       imgURL,
			Confidence:     best.score,
			EmbeddingModel: best.image.EmbeddingModel,
		})
	}

	// Sort descending by confidence
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Confidence > matches[j].Confidence
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}

	return &domain.VisionMatchResponse{
		QueryModel: queryModel,
		Matches:    matches,
	}, nil
}

func (u *visionUsecase) ReindexImages(ctx context.Context) (int, error) {
	if u.visionService == nil {
		return 0, domain.NewAppError(http.StatusServiceUnavailable, "vision AI embedding service is not configured")
	}

	images, err := u.catalogRepo.ListProductImagesWithoutEmbedding(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to list product images missing embeddings: %w", err)
	}

	reindexedCount := 0
	for _, img := range images {
		imgBytes, err := fetchImageBytes(ctx, img.ImageURL)
		if err != nil {
			slog.Warn("reindex: failed to fetch image", "id", img.ID, "url", img.ImageURL, "err", err)
			continue
		}

		emb, model, err := u.visionService.ExtractEmbedding(ctx, imgBytes)
		if err != nil {
			slog.Warn("reindex: failed to extract embedding", "id", img.ID, "err", err)
			continue
		}

		if err := u.catalogRepo.UpdateProductImageEmbedding(ctx, img.ID, emb, model); err != nil {
			slog.Warn("reindex: failed to save embedding", "id", img.ID, "err", err)
			continue
		}

		reindexedCount++
	}

	return reindexedCount, nil
}

func fetchImageBytes(ctx context.Context, imgURL string) ([]byte, error) {
	if strings.HasPrefix(imgURL, "http://") || strings.HasPrefix(imgURL, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		return io.ReadAll(resp.Body)
	}

	// Local file path (strip prefix if any)
	cleanPath := strings.TrimPrefix(imgURL, "/")
	return os.ReadFile(cleanPath)
}
