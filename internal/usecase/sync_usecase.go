package usecase

import (
	"context"
	"time"

	"ice-pos-api-v2/internal/domain"
)

type syncUsecase struct {
	syncRepo domain.SyncRepository
}

func NewSyncUsecase(syncRepo domain.SyncRepository) domain.SyncUsecase {
	return &syncUsecase{syncRepo: syncRepo}
}

func (u *syncUsecase) PullChanges(ctx context.Context, sinceStr string) (*domain.SyncChangesResponse, error) {
	var sinceTime *time.Time
	if sinceStr != "" {
		t, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			t, err = time.Parse("2006-01-02 15:04:05", sinceStr)
			if err != nil {
				return nil, domain.NewAppError(400, "Invalid since format, expected ISO 8601 / RFC3339 (e.g. 2026-11-01T21:00:00Z)")
			}
		}
		sinceTime = &t
	}

	categories, products, err := u.syncRepo.GetChanges(ctx, sinceTime)
	if err != nil {
		return nil, err
	}

	return &domain.SyncChangesResponse{
		Since:      sinceTime,
		ServerTime: time.Now(),
		Categories: categories,
		Products:   products,
	}, nil
}

func (u *syncUsecase) PushBatch(ctx context.Context, req domain.SyncBatchRequest) (*domain.SyncBatchResponse, error) {
	if len(req.Sales) == 0 {
		return nil, domain.NewAppError(400, "Sync batch request must contain at least one sale")
	}

	return u.syncRepo.BatchUpsertSales(ctx, req.Sales)
}
