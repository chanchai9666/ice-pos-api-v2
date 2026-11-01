package usecase

import (
	"context"
	"fmt"
	"time"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/calc"

	"github.com/google/uuid"
)

type saleUsecase struct {
	saleRepo    domain.SaleRepository
	catalogRepo domain.CatalogRepository
}

func NewSaleUsecase(saleRepo domain.SaleRepository, catalogRepo domain.CatalogRepository) domain.SaleUsecase {
	return &saleUsecase{
		saleRepo:    saleRepo,
		catalogRepo: catalogRepo,
	}
}

func (u *saleUsecase) CheckoutCash(ctx context.Context, cashierID string, lines []domain.SaleItem, billDiscount, tenderedAmount int64) (*domain.Sale, *domain.Receipt, error) {
	if len(lines) == 0 {
		return nil, nil, domain.NewAppError(400, "Sale must contain at least one item line")
	}

	productIDs := make([]string, 0, len(lines))
	for _, l := range lines {
		productIDs = append(productIDs, l.ProductID)
	}

	products, err := u.catalogRepo.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return nil, nil, err
	}

	prodMap := make(map[string]domain.Product, len(products))
	for _, p := range products {
		prodMap[p.ID] = p
	}

	saleID := uuid.New().String()
	now := time.Now()

	var subTotal int64
	saleLines := make([]domain.SaleItem, 0, len(lines))
	receiptItems := make([]domain.ReceiptItem, 0, len(lines))

	for _, l := range lines {
		prod, ok := prodMap[l.ProductID]
		if !ok || !prod.IsActive {
			return nil, nil, domain.NewAppError(400, fmt.Sprintf("Product not found or inactive: %s", l.ProductID))
		}
		if l.Qty <= 0 {
			return nil, nil, domain.NewAppError(400, "Quantity must be greater than 0")
		}

		gross := calc.LineGrossTotal(prod.Price, l.Qty)
		net := calc.LineNetTotal(gross, l.LineDiscount)

		subTotal = calc.AddSatang(subTotal, net)

		item := domain.SaleItem{
			ID:           uuid.New().String(),
			SaleID:       saleID,
			ProductID:    prod.ID,
			ProductName:  prod.Name,
			ProductCode:  prod.Code,
			UnitPrice:    prod.Price,
			CostPrice:    prod.CostPrice, // snapshot for reporting
			Qty:          l.Qty,
			LineDiscount: l.LineDiscount,
			GrossTotal:   gross,
			NetTotal:     net,
		}
		saleLines = append(saleLines, item)

		receiptItems = append(receiptItems, domain.ReceiptItem{
			ProductName: prod.Name,
			Qty:         l.Qty,
			UnitPrice:   prod.Price,
			NetTotal:    net,
		})
	}

	grandTotal := calc.GrandTotal(subTotal, billDiscount)

	// Validate cash payment and calculate change using shopspring/decimal
	change, err := calc.CalculateChange(tenderedAmount, grandTotal)
	if err != nil {
		return nil, nil, domain.NewAppError(400, fmt.Sprintf("Insufficient cash: tendered %d satang is less than grand total %d satang", tenderedAmount, grandTotal))
	}

	saleNo, err := u.saleRepo.GenerateNextSaleNo(ctx, now)
	if err != nil {
		return nil, nil, err
	}

	sale := &domain.Sale{
		ID:             saleID,
		SaleNo:         saleNo,
		SubTotal:       subTotal,
		BillDiscount:   billDiscount,
		GrandTotal:     grandTotal,
		TenderedAmount: tenderedAmount,
		ChangeAmount:   change,
		Status:         domain.SaleStatusCompleted,
		CashierID:      cashierID,
		OccurredAt:     now,
		CreatedAt:      now,
		UpdatedAt:      now,
		Lines:          saleLines,
	}

	if err := u.saleRepo.CreateSale(ctx, sale); err != nil {
		return nil, nil, err
	}

	receipt := &domain.Receipt{
		ReceiptNo:   fmt.Sprintf("RCP-%s", saleNo),
		SaleNo:      saleNo,
		OccurredAt:  now,
		CashierName: cashierID,
		SubTotal:    subTotal,
		Discount:    billDiscount,
		GrandTotal:  grandTotal,
		Tendered:    tenderedAmount,
		Change:      change,
		Status:      domain.SaleStatusCompleted,
		Items:       receiptItems,
	}

	return sale, receipt, nil
}

func (u *saleUsecase) GetSaleByID(ctx context.Context, id string) (*domain.Sale, error) {
	return u.saleRepo.GetSaleByID(ctx, id)
}

func (u *saleUsecase) GetReceipt(ctx context.Context, saleNoOrID string) (*domain.Receipt, error) {
	sale, err := u.saleRepo.GetSaleByNo(ctx, saleNoOrID)
	if err != nil {
		sale, err = u.saleRepo.GetSaleByID(ctx, saleNoOrID)
		if err != nil {
			return nil, err
		}
	}

	items := make([]domain.ReceiptItem, len(sale.Lines))
	for i, l := range sale.Lines {
		items[i] = domain.ReceiptItem{
			ProductName: l.ProductName,
			Qty:         l.Qty,
			UnitPrice:   l.UnitPrice,
			NetTotal:    l.NetTotal,
		}
	}

	return &domain.Receipt{
		ReceiptNo:   fmt.Sprintf("RCP-%s", sale.SaleNo),
		SaleNo:      sale.SaleNo,
		OccurredAt:  sale.OccurredAt,
		CashierName: sale.CashierID,
		SubTotal:    sale.SubTotal,
		Discount:    sale.BillDiscount,
		GrandTotal:  sale.GrandTotal,
		Tendered:    sale.TenderedAmount,
		Change:      sale.ChangeAmount,
		Status:      sale.Status,
		Items:       items,
	}, nil
}

func (u *saleUsecase) ListSales(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]domain.Sale, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	return u.saleRepo.ListSales(ctx, startDate, endDate, limit, offset)
}

func (u *saleUsecase) VoidSale(ctx context.Context, saleID, reason string) error {
	if reason == "" {
		reason = "พนักงานขอยกเลิกบิล"
	}
	return u.saleRepo.VoidSale(ctx, saleID, reason)
}
