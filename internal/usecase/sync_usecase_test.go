package usecase_test

import (
	"context"
	"testing"
	"time"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/usecase"
)

type mockSyncRepo struct {
	categories []domain.Category
	products   []domain.Product
	batchResp  *domain.SyncBatchResponse
}

func (m *mockSyncRepo) GetChanges(ctx context.Context, since *time.Time) ([]domain.Category, []domain.Product, error) {
	return m.categories, m.products, nil
}

func (m *mockSyncRepo) BatchUpsertSales(ctx context.Context, sales []domain.Sale) (*domain.SyncBatchResponse, error) {
	return m.batchResp, nil
}

func TestSyncUsecase(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockSyncRepo{
		categories: []domain.Category{{ID: "c1", Name: "เครื่องดื่ม"}},
		products:   []domain.Product{{ID: "p1", Name: "กาแฟโบราณ", Price: 2500}},
		batchResp:  &domain.SyncBatchResponse{SyncedSales: 2, SkippedSales: 1},
	}
	syncUc := usecase.NewSyncUsecase(mockRepo)

	t.Run("PullChanges without since returns full catalog", func(t *testing.T) {
		res, err := syncUc.PullChanges(ctx, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res.Categories) != 1 || len(res.Products) != 1 {
			t.Errorf("expected 1 cat and 1 prod, got %d, %d", len(res.Categories), len(res.Products))
		}
	})

	t.Run("PullChanges with invalid since returns error", func(t *testing.T) {
		_, err := syncUc.PullChanges(ctx, "invalid-date")
		if err == nil {
			t.Fatalf("expected error for invalid date, got nil")
		}
	})

	t.Run("PushBatch empty sales returns error", func(t *testing.T) {
		_, err := syncUc.PushBatch(ctx, domain.SyncBatchRequest{Sales: nil})
		if err == nil {
			t.Fatalf("expected error for empty sales batch, got nil")
		}
	})

	t.Run("PushBatch with sales returns sync summary", func(t *testing.T) {
		sales := []domain.Sale{
			{ID: "s1", SaleNo: "OFFLINE-1"},
		}
		res, err := syncUc.PushBatch(ctx, domain.SyncBatchRequest{Sales: sales})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.SyncedSales != 2 {
			t.Errorf("expected 2 synced sales, got %d", res.SyncedSales)
		}
	})
}
