package usecase

import (
	"context"
	"testing"

	"ice-pos-api-v2/internal/domain"
)

type mockVisionService struct {
	vector domain.FloatVector
	model  string
	err    error
}

func (m *mockVisionService) ExtractEmbedding(ctx context.Context, imageBytes []byte) (domain.FloatVector, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.vector, m.model, nil
}

type mockCatalogRepoForVision struct {
	images   []domain.ProductImage
	products map[string]*domain.Product
}

func (m *mockCatalogRepoForVision) ListAllProductImagesWithEmbeddings(ctx context.Context) ([]domain.ProductImage, error) {
	return m.images, nil
}

func (m *mockCatalogRepoForVision) ListProductImagesWithoutEmbedding(ctx context.Context) ([]domain.ProductImage, error) {
	return nil, nil
}

func (m *mockCatalogRepoForVision) GetProductByID(ctx context.Context, id string) (*domain.Product, error) {
	if p, ok := m.products[id]; ok {
		return p, nil
	}
	return nil, domain.ErrNotFound
}

func (m *mockCatalogRepoForVision) UpdateProductImageEmbedding(ctx context.Context, imageID string, embedding domain.FloatVector, model string) error {
	return nil
}

// Stubs for interface
func (m *mockCatalogRepoForVision) GetProductByBarcode(ctx context.Context, barcode string) (*domain.Product, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) GetProductByBarcodeExcludeID(ctx context.Context, barcode, excludeID string) (*domain.Product, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) GetProductByCode(ctx context.Context, code string) (*domain.Product, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) GetProductsByIDs(ctx context.Context, ids []string) ([]domain.Product, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) ListProducts(ctx context.Context, search, categoryID, barcode string, limit, offset int) ([]domain.Product, int64, error) {
	return nil, 0, nil
}
func (m *mockCatalogRepoForVision) CreateProduct(ctx context.Context, product *domain.Product) error {
	return nil
}
func (m *mockCatalogRepoForVision) UpdateProduct(ctx context.Context, product *domain.Product) error {
	return nil
}
func (m *mockCatalogRepoForVision) DeleteProduct(ctx context.Context, id string) error {
	return nil
}
func (m *mockCatalogRepoForVision) CreateProductImage(ctx context.Context, img *domain.ProductImage) error {
	return nil
}
func (m *mockCatalogRepoForVision) GetProductImageByID(ctx context.Context, id string) (*domain.ProductImage, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) ListProductImages(ctx context.Context, productID string) ([]domain.ProductImage, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) DeleteProductImage(ctx context.Context, id string) error {
	return nil
}
func (m *mockCatalogRepoForVision) SetPrimaryProductImage(ctx context.Context, productID, imageID string) error {
	return nil
}
func (m *mockCatalogRepoForVision) CountProductImages(ctx context.Context, productID string) (int64, error) {
	return 0, nil
}
func (m *mockCatalogRepoForVision) ListCategories(ctx context.Context) ([]domain.Category, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) GetCategoryByID(ctx context.Context, id string) (*domain.Category, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) GetCategoryByName(ctx context.Context, name string) (*domain.Category, error) {
	return nil, nil
}
func (m *mockCatalogRepoForVision) CreateCategory(ctx context.Context, category *domain.Category) error {
	return nil
}
func (m *mockCatalogRepoForVision) UpdateCategory(ctx context.Context, category *domain.Category) error {
	return nil
}
func (m *mockCatalogRepoForVision) DeleteCategory(ctx context.Context, id string) error {
	return nil
}

func TestMatchFrame_Success(t *testing.T) {
	mockService := &mockVisionService{
		vector: domain.FloatVector{1.0, 0.0, 0.0},
		model:  "Xenova/clip-vit-base-patch32",
	}

	repo := &mockCatalogRepoForVision{
		images: []domain.ProductImage{
			{
				ID:        "img-1",
				ProductID: "p-1",
				ImageURL:  "/uploads/p1.webp",
				Embedding: domain.FloatVector{1.0, 0.0, 0.0}, // identical -> 1.0
			},
			{
				ID:        "img-2",
				ProductID: "p-2",
				ImageURL:  "/uploads/p2.webp",
				Embedding: domain.FloatVector{0.0, 1.0, 0.0}, // orthogonal -> 0.0
			},
		},
		products: map[string]*domain.Product{
			"p-1": {
				ID:       "p-1",
				Code:     "P001",
				Barcode:  "885001",
				Name:     "Ice Cup A",
				Price:    2500,
				ImageURL: "/uploads/p1.webp",
			},
			"p-2": {
				ID:       "p-2",
				Code:     "P002",
				Barcode:  "885002",
				Name:     "Ice Cup B",
				Price:    3000,
				ImageURL: "/uploads/p2.webp",
			},
		},
	}

	uc := NewVisionUsecase(repo, mockService)
	resp, err := uc.MatchFrame(context.Background(), []byte("frame-bytes"), 0.8, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(resp.Matches))
	}

	match := resp.Matches[0]
	if match.ProductID != "p-1" {
		t.Errorf("expected product p-1, got %s", match.ProductID)
	}
	if match.Confidence < 0.99 {
		t.Errorf("expected confidence ~1.0, got %f", match.Confidence)
	}
}
