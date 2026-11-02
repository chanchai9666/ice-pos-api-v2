package domain

import (
	"context"
	"time"
)

const (
	SaleStatusCompleted = "COMPLETED"
	SaleStatusVoid      = "VOID"

	PaymentMethodCash   = "CASH"
	PaymentMethodCredit = "CREDIT"
)

type Sale struct {
	ID             string     `json:"id" gorm:"primaryKey;size:36"`
	SaleNo         string     `json:"sale_no" gorm:"uniqueIndex;size:50;not null"`
	SubTotal       int64      `json:"subtotal" gorm:"not null"`        // Satang
	BillDiscount   int64      `json:"bill_discount" gorm:"default:0"`  // Satang
	GrandTotal     int64      `json:"grand_total" gorm:"not null"`     // Satang
	TenderedAmount int64      `json:"tendered_amount" gorm:"not null"` // Satang (cash received)
	ChangeAmount   int64      `json:"change_amount" gorm:"not null"`   // Satang (change calculated)
	PaymentMethod  string     `json:"payment_method" gorm:"size:20;not null;default:'CASH'"`
	CustomerID     *string    `json:"customer_id,omitempty" gorm:"size:36;index"`
	CustomerName   string     `json:"customer_name,omitempty" gorm:"size:150"`
	CreditAmount   int64      `json:"credit_amount" gorm:"default:0"`
	Status         string     `json:"status" gorm:"size:20;not null;default:'COMPLETED'"`
	VoidReason     string     `json:"void_reason,omitempty" gorm:"size:255"`
	VoidedAt       *time.Time `json:"voided_at,omitempty"`
	CashierID      string     `json:"cashier_id" gorm:"size:36;not null"`
	CashierName    string     `json:"cashier_name,omitempty" gorm:"-"`
	OccurredAt     time.Time  `json:"occurred_at" gorm:"index;not null"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Lines []SaleItem `json:"lines" gorm:"foreignKey:SaleID"`
}

type SaleItem struct {
	ID           string `json:"id" gorm:"primaryKey;size:36"`
	SaleID       string `json:"sale_id" gorm:"size:36;index;not null"`
	ProductID    string `json:"product_id" gorm:"size:36;index;not null"`
	ProductName  string `json:"product_name" gorm:"size:200;not null"`
	ProductCode  string `json:"product_code" gorm:"size:50;not null"`
	UnitPrice    int64  `json:"unit_price" gorm:"not null"`  // Satang
	CostPrice    int64  `json:"cost_price" gorm:"default:0"` // Satang (snapshot for profit calculation)
	Qty          int    `json:"qty" gorm:"not null"`
	LineDiscount int64  `json:"line_discount" gorm:"default:0"` // Satang
	GrossTotal   int64  `json:"gross_total" gorm:"not null"`    // Satang
	NetTotal     int64  `json:"net_total" gorm:"not null"`      // Satang
}

type Receipt struct {
	ReceiptNo     string        `json:"receipt_no"`
	SaleNo        string        `json:"sale_no"`
	OccurredAt    time.Time     `json:"occurred_at"`
	CashierName   string        `json:"cashier_name"`
	CustomerName  string        `json:"customer_name,omitempty"`
	PaymentMethod string        `json:"payment_method"`
	SubTotal      int64         `json:"subtotal"`
	Discount      int64         `json:"discount"`
	GrandTotal    int64         `json:"grand_total"`
	Tendered      int64         `json:"tendered"`
	Change        int64         `json:"change"`
	Status        string        `json:"status"`
	Items         []ReceiptItem `json:"items"`
}

type ReceiptItem struct {
	ProductName string `json:"product_name"`
	Qty         int    `json:"qty"`
	UnitPrice   int64  `json:"unit_price"`
	NetTotal    int64  `json:"net_total"`
}

type SaleRepository interface {
	CreateSale(ctx context.Context, sale *Sale) error
	GetSaleByID(ctx context.Context, id string) (*Sale, error)
	GetSaleByNo(ctx context.Context, saleNo string) (*Sale, error)
	ListSales(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]Sale, int64, error)
	VoidSale(ctx context.Context, id, reason string) error
	GenerateNextSaleNo(ctx context.Context, t time.Time) (string, error)
}

type SaleUsecase interface {
	Checkout(ctx context.Context, cashierID string, lines []SaleItem, billDiscount, tenderedAmount int64, paymentMethod, customerID string) (*Sale, *Receipt, error)
	CheckoutCash(ctx context.Context, cashierID string, lines []SaleItem, billDiscount, tenderedAmount int64) (*Sale, *Receipt, error)
	GetSaleByID(ctx context.Context, id string) (*Sale, error)
	GetReceipt(ctx context.Context, saleNoOrID string) (*Receipt, error)
	ListSales(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]Sale, int64, error)
	VoidSale(ctx context.Context, saleID, reason string) error
}
