package domain

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type FloatVector []float32

func (v FloatVector) Value() (driver.Value, error) {
	if len(v) == 0 {
		return nil, nil
	}
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return string(bytes), nil
}

func (v *FloatVector) Scan(value interface{}) error {
	if value == nil {
		*v = nil
		return nil
	}
	var data []byte
	switch val := value.(type) {
	case []byte:
		data = val
	case string:
		data = []byte(val)
	default:
		return fmt.Errorf("unsupported type for FloatVector: %T", value)
	}
	if len(data) == 0 {
		*v = nil
		return nil
	}
	return json.Unmarshal(data, v)
}

type Category struct {
	ID        string    `json:"id" gorm:"primaryKey;size:36"`
	Name      string    `json:"name" gorm:"size:100;not null;uniqueIndex"`
	SortOrder int       `json:"sort_order" gorm:"default:0;index"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProductImage struct {
	ID             string      `json:"id" gorm:"primaryKey;size:36"`
	ProductID      string      `json:"product_id" gorm:"size:36;index;not null"`
	ImageURL       string      `json:"image_url" gorm:"size:255;not null"`
	IsPrimary      bool        `json:"is_primary" gorm:"default:false;index"`
	SortOrder      int         `json:"sort_order" gorm:"default:0;index"`
	Embedding      FloatVector `json:"embedding,omitempty" gorm:"type:text"`
	EmbeddingModel string      `json:"embedding_model,omitempty" gorm:"size:100"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

type Product struct {
	ID         string         `json:"id" gorm:"primaryKey;size:36"`
	Code       string         `json:"code" gorm:"uniqueIndex;size:50;not null"`
	Barcode    string         `json:"barcode" gorm:"index;size:50"`
	Name       string         `json:"name" gorm:"size:200;not null"`
	Price      int64          `json:"price" gorm:"not null"`       // Satang
	CostPrice  int64          `json:"cost_price" gorm:"default:0"` // Satang (for reporting only)
	CategoryID string         `json:"category_id" gorm:"size:36;index"`
	Category   *Category      `json:"category,omitempty" gorm:"foreignKey:CategoryID"`
	ImageURL   string         `json:"image_url" gorm:"size:255"`
	Images     []ProductImage `json:"images" gorm:"foreignKey:ProductID;constraint:OnDelete:CASCADE"`
	IsActive   bool           `json:"is_active" gorm:"default:true"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type CreateProductImageInput struct {
	ProductID      string      `json:"product_id"`
	ImageURL       string      `json:"image_url"`
	IsPrimary      bool        `json:"is_primary"`
	SortOrder      int         `json:"sort_order"`
	Embedding      FloatVector `json:"embedding"`
	EmbeddingModel string      `json:"embedding_model"`
}

type UpdateProductImageEmbeddingInput struct {
	Embedding      FloatVector `json:"embedding" validate:"required"`
	EmbeddingModel string      `json:"embedding_model"`
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

	// Product Images
	CreateProductImage(ctx context.Context, img *ProductImage) error
	GetProductImageByID(ctx context.Context, id string) (*ProductImage, error)
	ListProductImages(ctx context.Context, productID string) ([]ProductImage, error)
	DeleteProductImage(ctx context.Context, id string) error
	SetPrimaryProductImage(ctx context.Context, productID, imageID string) error
	UpdateProductImageEmbedding(ctx context.Context, imageID string, embedding FloatVector, model string) error
	CountProductImages(ctx context.Context, productID string) (int64, error)
	ListAllProductImagesWithEmbeddings(ctx context.Context) ([]ProductImage, error)
	ListProductImagesWithoutEmbedding(ctx context.Context) ([]ProductImage, error)

	ListCategories(ctx context.Context) ([]Category, error)
	GetCategoryByID(ctx context.Context, id string) (*Category, error)
	GetCategoryByName(ctx context.Context, name string) (*Category, error)
	CreateCategory(ctx context.Context, category *Category) error
	UpdateCategory(ctx context.Context, category *Category) error
	DeleteCategory(ctx context.Context, id string) error
}

type CreateCategoryInput struct {
	Name      string `json:"name" validate:"required,min=1,max=100"`
	SortOrder int    `json:"sort_order"`
}

type UpdateCategoryInput struct {
	Name      string `json:"name" validate:"required,min=1,max=100"`
	SortOrder *int   `json:"sort_order"`
}

type CatalogUsecase interface {
	GetProductByID(ctx context.Context, id string) (*Product, error)
	GetProductByBarcode(ctx context.Context, barcode string) (*Product, error)
	SearchProducts(ctx context.Context, search, categoryID, barcode string, limit, offset int) ([]Product, int64, error)
	CreateProduct(ctx context.Context, product *Product) error
	UpdateProduct(ctx context.Context, product *Product) error

	// Product Images
	AddProductImage(ctx context.Context, input CreateProductImageInput) (*ProductImage, error)
	GetProductImageByID(ctx context.Context, id string) (*ProductImage, error)
	DeleteProductImage(ctx context.Context, productID, imageID string) error
	SetPrimaryProductImage(ctx context.Context, productID, imageID string) error
	UpdateProductImageEmbedding(ctx context.Context, productID, imageID string, input UpdateProductImageEmbeddingInput) error
	ListProductImages(ctx context.Context, productID string) ([]ProductImage, error)
	CountProductImages(ctx context.Context, productID string) (int64, error)

	ListCategories(ctx context.Context) ([]Category, error)
	GetCategoryByID(ctx context.Context, id string) (*Category, error)
	CreateCategory(ctx context.Context, input CreateCategoryInput) (*Category, error)
	UpdateCategory(ctx context.Context, id string, input UpdateCategoryInput) (*Category, error)
	DeleteCategory(ctx context.Context, id string) error
}
