package usecase_test

import (
	"context"
	"testing"
	"time"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/usecase"
)

type mockCatalogRepo struct {
	domain.CatalogRepository
	products map[string]domain.Product
}

func (m *mockCatalogRepo) GetProductsByIDs(ctx context.Context, ids []string) ([]domain.Product, error) {
	var result []domain.Product
	for _, id := range ids {
		if p, ok := m.products[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

type mockSaleRepo struct {
	domain.SaleRepository
	savedSale *domain.Sale
	voidedID  string
}

func (m *mockSaleRepo) GenerateNextSaleNo(ctx context.Context, t time.Time) (string, error) {
	return "SO-20261101-0001", nil
}

func (m *mockSaleRepo) CreateSale(ctx context.Context, sale *domain.Sale) error {
	m.savedSale = sale
	return nil
}

func (m *mockSaleRepo) VoidSale(ctx context.Context, id, reason string) error {
	m.voidedID = id
	return nil
}

func (m *mockSaleRepo) GetSaleByID(ctx context.Context, id string) (*domain.Sale, error) {
	if m.savedSale != nil && m.savedSale.ID == id {
		return m.savedSale, nil
	}
	return nil, domain.ErrNotFound
}

func TestSaleUsecase(t *testing.T) {
	catRepo := &mockCatalogRepo{
		products: map[string]domain.Product{
			"p1": {
				ID:        "p1",
				Code:      "P001",
				Name:      "น้ำดื่ม",
				Price:     1000, // 10.00 THB
				CostPrice: 600,  // 6.00 THB
				IsActive:  true,
			},
			"p2": {
				ID:        "p2",
				Code:      "P002",
				Name:      "ขนมปัง",
				Price:     2500, // 25.00 THB
				CostPrice: 1500, // 15.00 THB
				IsActive:  true,
			},
		},
	}
	saleRepo := &mockSaleRepo{}
	saleUc := usecase.NewSaleUsecase(saleRepo, catRepo)

	ctx := context.Background()

	t.Run("CheckoutCash success with exact change", func(t *testing.T) {
		lines := []domain.SaleItem{
			{ProductID: "p1", Qty: 2}, // 2 * 10 = 20 THB (2000 satang)
			{ProductID: "p2", Qty: 1}, // 1 * 25 = 25 THB (2500 satang)
		}
		// Grand total = 45.00 THB (4500 satang)
		// Tendered = 100.00 THB (10000 satang)
		// Expected change = 55.00 THB (5500 satang)
		tendered := int64(10000)
		sale, receipt, err := saleUc.CheckoutCash(ctx, "cashier-1", lines, 0, tendered)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if sale.GrandTotal != 4500 {
			t.Errorf("expected grand total 4500, got %d", sale.GrandTotal)
		}
		if sale.ChangeAmount != 5500 {
			t.Errorf("expected change 5500, got %d", sale.ChangeAmount)
		}
		if receipt == nil || receipt.Change != 5500 {
			t.Errorf("expected receipt change 5500, got %v", receipt)
		}
	})

	t.Run("CheckoutCash fails on insufficient cash", func(t *testing.T) {
		lines := []domain.SaleItem{
			{ProductID: "p1", Qty: 5}, // 50.00 THB (5000 satang)
		}
		tendered := int64(2000) // 20.00 THB (insufficient)
		_, _, err := saleUc.CheckoutCash(ctx, "cashier-1", lines, 0, tendered)
		if err == nil {
			t.Fatalf("expected error for insufficient payment, got nil")
		}
	})

	t.Run("VoidSale directly without admin restrictions", func(t *testing.T) {
		err := saleUc.VoidSale(ctx, "sale-123", "Customer returned items")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if saleRepo.voidedID != "sale-123" {
			t.Errorf("expected voided ID 'sale-123', got '%s'", saleRepo.voidedID)
		}
	})
}
