package repository

import (
	"context"
	"errors"

	"ice-pos-api-v2/internal/domain"

	"gorm.io/gorm"
)

type catalogRepository struct {
	db *gorm.DB
}

func NewCatalogRepository(db *gorm.DB) domain.CatalogRepository {
	return &catalogRepository{db: db}
}

func (r *catalogRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	var prod domain.Product
	err := r.db.WithContext(ctx).Preload("Category").First(&prod, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &prod, nil
}

func (r *catalogRepository) GetProductByBarcode(ctx context.Context, barcode string) (*domain.Product, error) {
	var prod domain.Product
	err := r.db.WithContext(ctx).Preload("Category").First(&prod, "barcode = ? AND is_active = ?", barcode, true).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &prod, nil
}

func (r *catalogRepository) GetProductByCode(ctx context.Context, code string) (*domain.Product, error) {
	var prod domain.Product
	err := r.db.WithContext(ctx).Preload("Category").First(&prod, "code = ? AND is_active = ?", code, true).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &prod, nil
}

func (r *catalogRepository) GetProductsByIDs(ctx context.Context, ids []string) ([]domain.Product, error) {
	var products []domain.Product
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&products).Error
	return products, err
}

func (r *catalogRepository) ListProducts(ctx context.Context, search, categoryID string, limit, offset int) ([]domain.Product, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Product{}).Where("is_active = ?", true)

	if search != "" {
		likeTerm := "%" + search + "%"
		query = query.Where("name LIKE ? OR code LIKE ? OR barcode LIKE ?", likeTerm, likeTerm, likeTerm)
	}
	if categoryID != "" {
		query = query.Where("category_id = ?", categoryID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var products []domain.Product
	err := query.Preload("Category").Order("code ASC").Limit(limit).Offset(offset).Find(&products).Error
	return products, total, err
}

func (r *catalogRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *catalogRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *catalogRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category
	err := r.db.WithContext(ctx).Order("sort_order ASC, name ASC").Find(&categories).Error
	return categories, err
}

func (r *catalogRepository) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	var cat domain.Category
	err := r.db.WithContext(ctx).First(&cat, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &cat, nil
}

func (r *catalogRepository) CreateCategory(ctx context.Context, category *domain.Category) error {
	return r.db.WithContext(ctx).Create(category).Error
}
