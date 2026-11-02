package usecase

import (
	"context"
	"strings"
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
	trimmed := strings.TrimSpace(barcode)
	if trimmed == "" {
		return nil, domain.ErrBadRequest
	}
	return u.catalogRepo.GetProductByBarcode(ctx, trimmed)
}

func (u *catalogUsecase) SearchProducts(ctx context.Context, search, categoryID, barcode string, limit, offset int) ([]domain.Product, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	return u.catalogRepo.ListProducts(ctx, search, categoryID, barcode, limit, offset)
}

func (u *catalogUsecase) CreateProduct(ctx context.Context, product *domain.Product) error {
	product.Barcode = strings.TrimSpace(product.Barcode)
	product.Code = strings.TrimSpace(product.Code)
	if product.Barcode != "" {
		existing, err := u.catalogRepo.GetProductByBarcode(ctx, product.Barcode)
		if err == nil && existing != nil {
			return domain.NewAppError(400, "รหัสบาร์โค้ดนี้ถูกใช้งานโดยสินค้าอื่นแล้ว")
		}
	}

	if product.ID == "" {
		product.ID = uuid.New().String()
	}
	product.CreatedAt = time.Now()
	product.UpdatedAt = time.Now()
	product.IsActive = true
	return u.catalogRepo.CreateProduct(ctx, product)
}

func (u *catalogUsecase) UpdateProduct(ctx context.Context, product *domain.Product) error {
	if strings.TrimSpace(product.ID) == "" {
		return domain.ErrBadRequest
	}

	existing, err := u.catalogRepo.GetProductByID(ctx, product.ID)
	if err != nil {
		return err
	}

	product.Barcode = strings.TrimSpace(product.Barcode)
	if product.Barcode != "" {
		dup, err := u.catalogRepo.GetProductByBarcodeExcludeID(ctx, product.Barcode, product.ID)
		if err == nil && dup != nil {
			return domain.NewAppError(400, "รหัสบาร์โค้ดนี้ถูกใช้งานโดยสินค้าอื่นแล้ว")
		}
	}

	if product.Code != "" {
		existing.Code = strings.TrimSpace(product.Code)
	}
	existing.Barcode = product.Barcode
	if product.Name != "" {
		existing.Name = product.Name
	}
	if product.Price > 0 {
		existing.Price = product.Price
	}
	if product.CostPrice > 0 {
		existing.CostPrice = product.CostPrice
	}
	if product.CategoryID != "" {
		existing.CategoryID = product.CategoryID
	}
	if product.ImageURL != "" {
		existing.ImageURL = product.ImageURL
	}
	existing.UpdatedAt = time.Now()
	if err := u.catalogRepo.UpdateProduct(ctx, existing); err != nil {
		return err
	}
	*product = *existing
	return nil
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
