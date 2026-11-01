package usecase

import (
	"context"
	"time"

	"ice-pos-api-v2/internal/domain"

	"github.com/google/uuid"
)

type catalogUsecase struct {
	catalogRepo domain.CatalogRepository
}

func NewCatalogUsecase(catalogRepo domain.CatalogRepository) domain.CatalogUsecase {
	return &catalogUsecase{catalogRepo: catalogRepo}
}

func (u *catalogUsecase) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	return u.catalogRepo.GetProductByID(ctx, id)
}

func (u *catalogUsecase) GetProductByBarcode(ctx context.Context, barcode string) (*domain.Product, error) {
	return u.catalogRepo.GetProductByBarcode(ctx, barcode)
}

func (u *catalogUsecase) SearchProducts(ctx context.Context, search, categoryID string, limit, offset int) ([]domain.Product, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	return u.catalogRepo.ListProducts(ctx, search, categoryID, limit, offset)
}

func (u *catalogUsecase) CreateProduct(ctx context.Context, product *domain.Product) error {
	if product.ID == "" {
		product.ID = uuid.New().String()
	}
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	product.IsActive = true
	return u.catalogRepo.CreateProduct(ctx, product)
}

func (u *catalogUsecase) UpdateProduct(ctx context.Context, product *domain.Product) error {
	product.UpdatedAt = time.Now()
	return u.catalogRepo.UpdateProduct(ctx, product)
}

func (u *catalogUsecase) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return u.catalogRepo.ListCategories(ctx)
}

func (u *catalogUsecase) CreateCategory(ctx context.Context, name string, sortOrder int) (*domain.Category, error) {
	cat := &domain.Category{
		ID:        uuid.New().String(),
		Name:      name,
		SortOrder: sortOrder,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := u.catalogRepo.CreateCategory(ctx, cat); err != nil {
		return nil, err
	}
	return cat, nil
}
