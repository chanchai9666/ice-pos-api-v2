package domain

import (
	"context"
	"time"
)

type Category struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	SortOrder int       `json:"sort_order" gorm:"default:0"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	ID         string    `json:"id" gorm:"primaryKey;size:36"`
	Code       string    `json:"code" gorm:"uniqueIndex;size:50;not null"`
	Barcode    string    `json:"barcode" gorm:"index;size:50"`
	Name       string    `json:"name" gorm:"size:200;not null"`
	Price      int64     `json:"price" gorm:"not null"`       // Satang
	CostPrice  int64     `json:"cost_price" gorm:"default:0"` // Satang (for reporting only)
	CategoryID string    `json:"category_id" gorm:"size:36;index"`
	Category   *Category `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	ImageURL   string    `json:"image_url" gorm:"size:255"`
	IsActive   bool      `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CatalogRepository interface {
	GetProductByID(ctx context.Context, id string) (*Product, error)
	GetProductByBarcode(ctx context.Context, barcode string) (*Product, error)
	GetProductByBarcodeExcludeID(ctx context.Context, barcode, excludeID string) (*Product, error)
	GetProductByCode(ctx context.Context, code string) (*Product, error)
	GetProductsByIDs(ctx context.Context, ids []string) ([]Product, error)
	ListProducts(ctx context.Context, search, categoryID, barcode string, limit, offset int) ([]Product, int64, error)
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error

	ListCategories(ctx context.Context) ([]Category, error)
	GetCategoryByID(ctx context.Context, id string) (*Category, error)
	CreateCategory(ctx context.Context, category *Category) error
}

type CatalogUsecase interface {
	GetProductByID(ctx context.Context, id string) (*Product, error)
	GetProductByBarcode(ctx context.Context, barcode string) (*Product, error)
	SearchProducts(ctx context.Context, search, categoryID, barcode string, limit, offset int) ([]Product, int64, error)
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error

	ListCategories(ctx context.Context) ([]Category, error)
	CreateCategory(ctx context.Context, name string, sortOrder int) (*Category, error)
}
