package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"ice-pos-api-v2/internal/domain"

	"gorm.io/gorm"
)

type catalogRepository struct {
	db *gorm.DB
}

func NewCatalogRepository(db *gorm.DB) domain.CatalogRepository {
	return &catalogRepository{db: db}
}

func preloadImages(db *gorm.DB) *gorm.DB {
	return db.Order("product_images.is_primary DESC, product_images.sort_order ASC, product_images.created_at ASC")
}

func (r *catalogRepository) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	var prod domain.Product
	err := r.db.WithContext(ctx).Preload("Category").Preload("Images", preloadImages).First(&prod, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &prod, nil
}

func (r *catalogRepository) GetProductByBarcode(ctx context.Context, barcode string) (*domain.Product, error) {
	trimmed := strings.TrimSpace(barcode)
	if trimmed == "" {
		return nil, domain.ErrNotFound
	}
	var prod domain.Product
	err := r.db.WithContext(ctx).Preload("Category").Preload("Images", preloadImages).First(&prod, "barcode = ? AND is_active = ?", trimmed, true).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &prod, nil
}

func (r *catalogRepository) GetProductByBarcodeExcludeID(ctx context.Context, barcode, excludeID string) (*domain.Product, error) {
	trimmed := strings.TrimSpace(barcode)
	if trimmed == "" {
		return nil, domain.ErrNotFound
	}
	var prod domain.Product
	query := r.db.WithContext(ctx).Preload("Images", preloadImages).Where("barcode = ? AND is_active = ?", trimmed, true)
	if excludeID != "" {
		query = query.Where("id != ?", excludeID)
	}
	err := query.First(&prod).Error
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
	err := r.db.WithContext(ctx).Preload("Category").Preload("Images", preloadImages).First(&prod, "code = ? AND is_active = ?", code, true).Error
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
	err := r.db.WithContext(ctx).Preload("Category").Preload("Images", preloadImages).Where("id IN ?", ids).Find(&products).Error
	return products, err
}

func (r *catalogRepository) ListProducts(ctx context.Context, search, categoryID, barcode string, limit, offset int) ([]domain.Product, int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Product{}).Where("is_active = ?", true)

	if barcode != "" {
		query = query.Where("barcode = ?", strings.TrimSpace(barcode))
	}
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
	err := query.Preload("Category").Preload("Images", preloadImages).Order("code ASC").Limit(limit).Offset(offset).Find(&products).Error
	return products, total, err
}

func (r *catalogRepository) CreateProduct(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Create(product).Error
}

func (r *catalogRepository) UpdateProduct(ctx context.Context, product *domain.Product) error {
	return r.db.WithContext(ctx).Save(product).Error
}

func (r *catalogRepository) CreateProductImage(ctx context.Context, img *domain.ProductImage) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&domain.ProductImage{}).Where("product_id = ?", img.ProductID).Count(&count).Error; err != nil {
			return err
		}

		if count == 0 || img.IsPrimary {
			img.IsPrimary = true
			if err := tx.Model(&domain.ProductImage{}).Where("product_id = ?", img.ProductID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}

		if err := tx.Create(img).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"updated_at": time.Now(),
		}
		if img.IsPrimary {
			updates["image_url"] = img.ImageURL
		}
		return tx.Model(&domain.Product{}).Where("id = ?", img.ProductID).Updates(updates).Error
	})
}

func (r *catalogRepository) GetProductImageByID(ctx context.Context, id string) (*domain.ProductImage, error) {
	var img domain.ProductImage
	err := r.db.WithContext(ctx).First(&img, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &img, nil
}

func (r *catalogRepository) ListProductImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	var images []domain.ProductImage
	err := r.db.WithContext(ctx).Where("product_id = ?", productID).Order("is_primary DESC, sort_order ASC, created_at ASC").Find(&images).Error
	return images, err
}

func (r *catalogRepository) DeleteProductImage(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var img domain.ProductImage
		if err := tx.First(&img, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}

		if err := tx.Delete(&img).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"updated_at": time.Now(),
		}
		if img.IsPrimary {
			var nextImg domain.ProductImage
			err := tx.Where("product_id = ?", img.ProductID).Order("sort_order ASC, created_at ASC").First(&nextImg).Error
			if err == nil {
				if err := tx.Model(&nextImg).Update("is_primary", true).Error; err != nil {
					return err
				}
				updates["image_url"] = nextImg.ImageURL
			} else {
				updates["image_url"] = ""
			}
		}

		return tx.Model(&domain.Product{}).Where("id = ?", img.ProductID).Updates(updates).Error
	})
}

func (r *catalogRepository) SetPrimaryProductImage(ctx context.Context, productID, imageID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var targetImg domain.ProductImage
		if err := tx.First(&targetImg, "id = ? AND product_id = ?", imageID, productID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}

		if err := tx.Model(&domain.ProductImage{}).Where("product_id = ?", productID).Update("is_primary", false).Error; err != nil {
			return err
		}

		if err := tx.Model(&targetImg).Update("is_primary", true).Error; err != nil {
			return err
		}

		return tx.Model(&domain.Product{}).Where("id = ?", productID).Updates(map[string]interface{}{
			"image_url":  targetImg.ImageURL,
			"updated_at": time.Now(),
		}).Error
	})
}

func (r *catalogRepository) UpdateProductImageEmbedding(ctx context.Context, imageID string, embedding domain.FloatVector, model string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var img domain.ProductImage
		if err := tx.First(&img, "id = ?", imageID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return domain.ErrNotFound
			}
			return err
		}

		updates := map[string]interface{}{
			"embedding":  embedding,
			"updated_at": time.Now(),
		}
		if model != "" {
			updates["embedding_model"] = model
		}

		if err := tx.Model(&img).Updates(updates).Error; err != nil {
			return err
		}

		return tx.Model(&domain.Product{}).Where("id = ?", img.ProductID).Update("updated_at", time.Now()).Error
	})
}

func (r *catalogRepository) CountProductImages(ctx context.Context, productID string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.ProductImage{}).Where("product_id = ?", productID).Count(&count).Error
	return count, err
}

func (r *catalogRepository) ListAllProductImagesWithEmbeddings(ctx context.Context) ([]domain.ProductImage, error) {
	var images []domain.ProductImage
	err := r.db.WithContext(ctx).
		Where("embedding IS NOT NULL AND embedding != '' AND embedding != 'null' AND embedding != '[]'").
		Find(&images).Error
	return images, err
}

func (r *catalogRepository) ListProductImagesWithoutEmbedding(ctx context.Context) ([]domain.ProductImage, error) {
	var images []domain.ProductImage
	err := r.db.WithContext(ctx).
		Where("embedding IS NULL OR embedding = '' OR embedding = 'null' OR embedding = '[]'").
		Find(&images).Error
	return images, err
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

func (r *catalogRepository) GetCategoryByName(ctx context.Context, name string) (*domain.Category, error) {
	var cat domain.Category
	trimmedLower := strings.ToLower(strings.TrimSpace(name))
	err := r.db.WithContext(ctx).Where("LOWER(TRIM(name)) = ?", trimmedLower).First(&cat).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &cat, nil
}

func (r *catalogRepository) UpdateCategory(ctx context.Context, category *domain.Category) error {
	return r.db.WithContext(ctx).Save(category).Error
}

func (r *catalogRepository) DeleteCategory(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Unlink any products currently referencing this category
		if err := tx.Model(&domain.Product{}).Where("category_id = ?", id).Update("category_id", "").Error; err != nil {
			return err
		}
		// 2. Delete the category record
		res := tx.Delete(&domain.Category{}, "id = ?", id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
