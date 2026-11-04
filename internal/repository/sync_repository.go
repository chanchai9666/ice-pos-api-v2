package repository

import (
	"context"
	"fmt"
	"time"

	"ice-pos-api-v2/internal/domain"

	"gorm.io/gorm"
)

type syncRepository struct {
	db *gorm.DB
}

func NewSyncRepository(db *gorm.DB) domain.SyncRepository {
	return &syncRepository{db: db}
}

func (r *syncRepository) GetChanges(ctx context.Context, since *time.Time) ([]domain.Category, []domain.Product, error) {
	var categories []domain.Category
	var products []domain.Product

	catQuery := r.db.WithContext(ctx).Order("sort_order ASC, name ASC")
	prodQuery := r.db.WithContext(ctx).Preload("Category").Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("product_images.is_primary DESC, product_images.sort_order ASC, product_images.created_at ASC")
	}).Order("code ASC")

	if since != nil && !since.IsZero() {
		catQuery = catQuery.Where("updated_at > ?", *since)
		prodQuery = prodQuery.Where("updated_at > ?", *since)
	}

	if err := catQuery.Find(&categories).Error; err != nil {
		return nil, nil, err
	}

	if err := prodQuery.Find(&products).Error; err != nil {
		return nil, nil, err
	}

	return categories, products, nil
}

func (r *syncRepository) BatchUpsertSales(ctx context.Context, sales []domain.Sale) (*domain.SyncBatchResponse, error) {
	resp := &domain.SyncBatchResponse{}

	for _, s := range sales {
		if s.ID == "" {
			resp.FailedSales++
			resp.Errors = append(resp.Errors, "sale id is required")
			continue
		}

		// Check if sale already exists (Idempotency)
		var existing domain.Sale
		err := r.db.WithContext(ctx).First(&existing, "id = ?", s.ID).Error
		if err == nil {
			// Already exists
			if existing.Status != domain.SaleStatusVoid && s.Status == domain.SaleStatusVoid {
				// Update to VOID if client voided it offline
				now := time.Now()
				if s.VoidedAt != nil {
					now = *s.VoidedAt
				}
				r.db.WithContext(ctx).Model(&domain.Sale{}).Where("id = ?", s.ID).Updates(map[string]interface{}{
					"status":      domain.SaleStatusVoid,
					"void_reason": s.VoidReason,
					"voided_at":   &now,
				})
				resp.SyncedSales++
			} else {
				resp.SkippedSales++
			}
			continue
		}

		if err != gorm.ErrRecordNotFound {
			resp.FailedSales++
			resp.Errors = append(resp.Errors, fmt.Sprintf("failed to check sale %s: %v", s.ID, err))
			continue
		}

		// Ensure lines have correct SaleID
		for i := range s.Lines {
			s.Lines[i].SaleID = s.ID
		}

		if s.OccurredAt.IsZero() {
			s.OccurredAt = time.Now()
		}
		if s.CreatedAt.IsZero() {
			s.CreatedAt = time.Now()
		}
		s.UpdatedAt = time.Now()

		insertErr := r.db.WithContext(ctx).Create(&s).Error
		if insertErr != nil {
			resp.FailedSales++
			resp.Errors = append(resp.Errors, fmt.Sprintf("failed to insert sale %s: %v", s.ID, insertErr))
		} else {
			resp.SyncedSales++
		}
	}

	return resp, nil
}
