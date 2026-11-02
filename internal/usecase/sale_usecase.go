package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/calc"

	"github.com/google/uuid"
)

type saleUsecase struct {
	saleRepo     domain.SaleRepository
	catalogRepo  domain.CatalogRepository
	customerRepo domain.CustomerRepository
}

func NewSaleUsecase(saleRepo domain.SaleRepository, catalogRepo domain.CatalogRepository, customerRepo domain.CustomerRepository) domain.SaleUsecase {
	return &saleUsecase{
		saleRepo:     saleRepo,
		catalogRepo:  catalogRepo,
		customerRepo: customerRepo,
	}
}

func (u *saleUsecase) CheckoutCash(ctx context.Context, cashierID string, lines []domain.SaleItem, billDiscount, tenderedAmount int64) (*domain.Sale, *domain.Receipt, error) {
	return u.Checkout(ctx, cashierID, lines, billDiscount, tenderedAmount, domain.PaymentMethodCash, "")
}

func (u *saleUsecase) Checkout(ctx context.Context, cashierID string, lines []domain.SaleItem, billDiscount, tenderedAmount int64, paymentMethod, customerID string) (*domain.Sale, *domain.Receipt, error) {
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

	paymentMethod = strings.ToUpper(strings.TrimSpace(paymentMethod))
	if paymentMethod == "" {
		paymentMethod = domain.PaymentMethodCash
	}

	var change int64
	var creditAmount int64
	var custName string
	var custIDPtr *string

	if paymentMethod == domain.PaymentMethodCredit {
		if strings.TrimSpace(customerID) == "" {
			return nil, nil, domain.NewAppError(400, "จำเป็นต้องระบุลูกค้าสำหรับการชำระด้วยเงินเชื่อ (Credit transaction)")
		}
		if u.customerRepo == nil {
			return nil, nil, domain.NewAppError(500, "Customer repository not configured")
		}

		cust, err := u.customerRepo.GetByID(ctx, customerID)
		if err != nil || cust == nil {
			return nil, nil, domain.NewAppError(404, "ไม่พบข้อมูลลูกค้าสำหรับลงบัญชีเงินเชื่อ")
		}
		custName = cust.Name
		custID := cust.ID
		custIDPtr = &custID

		account, err := u.customerRepo.GetCreditAccount(ctx, cust.ID)
		if err != nil || account == nil {
			return nil, nil, domain.NewAppError(404, "ไม่พบบัญชีเงินเชื่อของลูกค้า")
		}

		newBalance := calc.AddSatang(account.OutstandingBalance, grandTotal)
		if account.CreditLimit > 0 && newBalance > account.CreditLimit {
			return nil, nil, domain.NewAppError(400, fmt.Sprintf("ยอดเงินเชื่อเกินวงเงินที่กำหนด (วงเงิน: %d สตางค์, ยอดหนี้ใหม่: %d สตางค์)", account.CreditLimit, newBalance))
		}

		creditAmount = grandTotal
		tenderedAmount = 0
		change = 0

		txRecord := domain.CreditTransaction{
			ID:            uuid.New().String(),
			AccountID:     account.ID,
			CustomerID:    cust.ID,
			Type:          domain.CreditTxSaleCredit,
			Amount:        grandTotal, // เพิ่มหนี้
			RefType:       "SALE",
			RefID:         saleID,
			PaymentMethod: &paymentMethod,
			OccurredAt:    now,
			CreatedBy:     &cashierID,
			CreatedAt:     now,
		}
		if err := u.customerRepo.RecordCreditTransaction(ctx, &txRecord); err != nil {
			return nil, nil, fmt.Errorf("failed to record credit transaction: %w", err)
		}
	} else {
		// Cash payment validation
		calculatedChange, err := calc.CalculateChange(tenderedAmount, grandTotal)
		if err != nil {
			return nil, nil, domain.NewAppError(400, fmt.Sprintf("Insufficient cash: tendered %d satang is less than grand total %d satang", tenderedAmount, grandTotal))
		}
		change = calculatedChange

		if strings.TrimSpace(customerID) != "" && u.customerRepo != nil {
			cust, err := u.customerRepo.GetByID(ctx, customerID)
			if err == nil && cust != nil {
				custName = cust.Name
				custID := cust.ID
				custIDPtr = &custID
			}
		}
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
		PaymentMethod:  paymentMethod,
		CustomerID:     custIDPtr,
		CustomerName:   custName,
		CreditAmount:   creditAmount,
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
		ReceiptNo:     fmt.Sprintf("RCP-%s", saleNo),
		SaleNo:        saleNo,
		OccurredAt:    now,
		CashierName:   cashierID,
		CustomerName:  custName,
		PaymentMethod: paymentMethod,
		SubTotal:      subTotal,
		Discount:      billDiscount,
		GrandTotal:    grandTotal,
		Tendered:      tenderedAmount,
		Change:        change,
		Status:        domain.SaleStatusCompleted,
		Items:         receiptItems,
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
		ReceiptNo:     fmt.Sprintf("RCP-%s", sale.SaleNo),
		SaleNo:        sale.SaleNo,
		OccurredAt:    sale.OccurredAt,
		CashierName:   sale.CashierID,
		CustomerName:  sale.CustomerName,
		PaymentMethod: sale.PaymentMethod,
		SubTotal:      sale.SubTotal,
		Discount:      sale.BillDiscount,
		GrandTotal:    sale.GrandTotal,
		Tendered:      sale.TenderedAmount,
		Change:        sale.ChangeAmount,
		Status:        sale.Status,
		Items:         items,
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

	sale, _ := u.saleRepo.GetSaleByID(ctx, saleID)

	if err := u.saleRepo.VoidSale(ctx, saleID, reason); err != nil {
		return err
	}

	// Reverse credit debt if this was a credit sale
	if sale != nil && sale.PaymentMethod == domain.PaymentMethodCredit && sale.CustomerID != nil && *sale.CustomerID != "" && u.customerRepo != nil {
		account, err := u.customerRepo.GetCreditAccount(ctx, *sale.CustomerID)
		if err == nil && account != nil {
			txRecord := domain.CreditTransaction{
				ID:         uuid.New().String(),
				AccountID:  account.ID,
				CustomerID: *sale.CustomerID,
				Type:       domain.CreditTxAdjustment,
				Amount:     -sale.CreditAmount, // ลดหนี้คืน
				RefType:    "VOID_SALE",
				RefID:      sale.ID,
				Note:       &reason,
				OccurredAt: time.Now(),
				CreatedBy:  &sale.CashierID,
				CreatedAt:  time.Now(),
			}
			_ = u.customerRepo.RecordCreditTransaction(ctx, &txRecord)
		}
	}

	return nil
}
