package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ice-pos-api-v2/internal/domain"

	"gorm.io/gorm"
)

type saleRepository struct {
	db *gorm.DB
}

func NewSaleRepository(db *gorm.DB) domain.SaleRepository {
	return &saleRepository{db: db}
}

func (r *saleRepository) GenerateNextSaleNo(ctx context.Context, t time.Time) (string, error) {
	datePrefix := t.Format("20060102")
	pattern := fmt.Sprintf("SO-%s-%%", datePrefix)

	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Sale{}).Where("sale_no LIKE ?", pattern).Count(&count).Error
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("SO-%s-%04d", datePrefix, count+1), nil
}

func (r *saleRepository) CreateSale(ctx context.Context, sale *domain.Sale) error {
	return r.db.WithContext(ctx).Create(sale).Error
}

func (r *saleRepository) GetSaleByID(ctx context.Context, id string) (*domain.Sale, error) {
	var sale domain.Sale
	err := r.db.WithContext(ctx).Preload("Lines").First(&sale, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &sale, nil
}

func (r *saleRepository) GetSaleByNo(ctx context.Context, saleNo string) (*domain.Sale, error) {
	var sale domain.Sale
	err := r.db.WithContext(ctx).Preload("Lines").First(&sale, "sale_no = ?", saleNo).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &sale, nil
}

func (r *saleRepository) ListSales(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]domain.Sale, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Sale{}).Where("occurred_at BETWEEN ? AND ?", startDate, endDate)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var sales []domain.Sale
	err := query.Preload("Lines").Order("occurred_at DESC").Limit(limit).Offset(offset).Find(&sales).Error
	return sales, total, err
}

func (r *saleRepository) VoidSale(ctx context.Context, id, reason string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&domain.Sale{}).
		Where("id = ? AND status = ?", id, domain.SaleStatusCompleted).
		Updates(map[string]interface{}{
			"status":      domain.SaleStatusVoid,
			"void_reason": reason,
			"voided_at":   &now,
		})

	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return domain.NewAppError(400, "Sale not found or already voided")
	}
	return nil
}
