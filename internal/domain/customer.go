package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// Credit Statuses
const (
	CreditStatusOK        = "OK"
	CreditStatusOverLimit = "OVER_LIMIT"
	CreditStatusOverdue   = "OVERDUE"
)

// Credit Transaction Types
const (
	CreditTxSaleCredit = "SALE_CREDIT"
	CreditTxPayment    = "PAYMENT"
	CreditTxAdjustment = "ADJUSTMENT"
	CreditTxWriteOff   = "WRITE_OFF"
)

type Customer struct {
	ID            string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	Name          string         `json:"name" gorm:"type:varchar(150);not null"`
	Phone         *string        `json:"phone,omitempty" gorm:"type:varchar(20);index"`
	Email         *string        `json:"email,omitempty" gorm:"type:varchar(150)"`
	IsMember      bool           `json:"is_member" gorm:"default:false"`
	Points        int64          `json:"points" gorm:"default:0"`
	CreditAccount *CreditAccount `json:"credit_account,omitempty" gorm:"foreignKey:CustomerID"`
	Version       int            `json:"version" gorm:"default:1"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
}

type CreditAccount struct {
	ID                 string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	CustomerID         string    `json:"customer_id" gorm:"type:varchar(36);not null;uniqueIndex"`
	Customer           *Customer `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	CreditLimit        int64     `json:"credit_limit" gorm:"not null;default:0"`               // satang
	OutstandingBalance int64     `json:"outstanding_balance" gorm:"not null;default:0"`        // satang
	Status             string    `json:"status" gorm:"type:varchar(20);not null;default:'OK'"` // OK, OVER_LIMIT, OVERDUE
	Version            int       `json:"version" gorm:"default:1"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreditTransaction struct {
	ID             string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	AccountID      string    `json:"account_id" gorm:"type:varchar(36);not null;index"`
	CustomerID     string    `json:"customer_id" gorm:"type:varchar(36);not null;index:idx_credit_cust_date"`
	Type           string    `json:"type" gorm:"type:varchar(20);not null"` // SALE_CREDIT, PAYMENT, ADJUSTMENT, WRITE_OFF
	Amount         int64     `json:"amount" gorm:"not null"`                // + increase debt, - pay debt
	BalanceAfter   int64     `json:"balance_after" gorm:"not null"`
	RefType        string    `json:"ref_type" gorm:"type:varchar(30)"` // SALE, PAYMENT
	RefID          string    `json:"ref_id" gorm:"type:varchar(36)"`
	PaymentMethod  *string   `json:"payment_method,omitempty" gorm:"type:varchar(20)"`
	Note           *string   `json:"note,omitempty" gorm:"type:text"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty" gorm:"type:varchar(100);uniqueIndex"`
	OccurredAt     time.Time `json:"occurred_at" gorm:"index:idx_credit_cust_date"`
	CreatedBy      *string   `json:"created_by,omitempty" gorm:"type:varchar(36)"`
	CreatedAt      time.Time `json:"created_at"`
}

type CustomerFilter struct {
	Search   string
	IsMember *bool
	Page     int
	Limit    int
}

type CustomerRepository interface {
	CreateWithCreditAccount(ctx context.Context, customer *Customer, account *CreditAccount) error
	GetByID(ctx context.Context, id string) (*Customer, error)
	GetAll(ctx context.Context, filter CustomerFilter) ([]Customer, int64, error)
	Update(ctx context.Context, customer *Customer) error
	Delete(ctx context.Context, id string) error

	// Credit & Debt
	GetCreditAccount(ctx context.Context, customerID string) (*CreditAccount, error)
	UpdateCreditLimit(ctx context.Context, customerID string, limit int64) error
	RecordCreditTransaction(ctx context.Context, tx *CreditTransaction) error
	GetCreditHistory(ctx context.Context, customerID string, limit int) ([]CreditTransaction, error)
	GetARAgingReport(ctx context.Context) ([]ARAgingItem, error)
}

type ARAgingItem struct {
	CustomerID         string `json:"customer_id"`
	CustomerName       string `json:"customer_name"`
	Phone              string `json:"phone"`
	CreditLimit        int64  `json:"credit_limit"`
	OutstandingBalance int64  `json:"outstanding_balance"`
	CurrentDue         int64  `json:"current_due"`  // 0-30 days
	Due30To60          int64  `json:"due_30_to_60"` // 31-60 days
	Due60To90          int64  `json:"due_60_to_90"` // 61-90 days
	Over90             int64  `json:"over_90"`      // 90+ days
	Status             string `json:"status"`
}

type ARAgingBuckets struct {
	CurrentDue int64 // 0-30 days
	Due30To60  int64 // 31-60 days
	Due60To90  int64 // 61-90 days
	Over90     int64 // >90 days
}

// CalculateARAgingBuckets calculates aging buckets for an outstanding balance against historical credit transactions.
func CalculateARAgingBuckets(outstandingBalance int64, txs []CreditTransaction, asOf time.Time) ARAgingBuckets {
	var buckets ARAgingBuckets
	remainingDec := decimal.NewFromInt(outstandingBalance)

	for _, tx := range txs {
		if remainingDec.LessThanOrEqual(decimal.Zero) {
			break
		}
		portionDec := decimal.NewFromInt(tx.Amount)
		if portionDec.GreaterThan(remainingDec) {
			portionDec = remainingDec
		}
		remainingDec = remainingDec.Sub(portionDec)

		days := int(asOf.Sub(tx.OccurredAt).Hours() / 24)
		switch {
		case days <= 30:
			buckets.CurrentDue += portionDec.IntPart()
		case days <= 60:
			buckets.Due30To60 += portionDec.IntPart()
		case days <= 90:
			buckets.Due60To90 += portionDec.IntPart()
		default:
			buckets.Over90 += portionDec.IntPart()
		}
	}
	return buckets
}

type CustomerUsecase interface {
	CreateCustomer(ctx context.Context, customer *Customer, initialCreditLimit int64) error
	GetCustomerByID(ctx context.Context, id string) (*Customer, error)
	GetAllCustomers(ctx context.Context, filter CustomerFilter) ([]Customer, int64, error)
	UpdateCustomer(ctx context.Context, customer *Customer) error
	DeleteCustomer(ctx context.Context, id string) error
	SetCreditLimit(ctx context.Context, customerID string, limit int64) error
	PayCredit(ctx context.Context, customerID string, amount int64, method string, note string, idempotencyKey *string, actorID string) (*CreditTransaction, error)
	GetCreditHistory(ctx context.Context, customerID string, limit int) ([]CreditTransaction, error)
	GetARAgingReport(ctx context.Context) ([]ARAgingItem, error)
}
