package usecase

import (
	"context"
	"strings"
	"time"

	"ice-pos-api-v2/internal/domain"

	"github.com/google/uuid"
)

type customerUsecase struct {
	customerRepo domain.CustomerRepository
}

func NewCustomerUsecase(customerRepo domain.CustomerRepository) domain.CustomerUsecase {
	return &customerUsecase{customerRepo: customerRepo}
}

func (u *customerUsecase) CreateCustomer(ctx context.Context, customer *domain.Customer, initialCreditLimit int64) error {
	customer.Name = strings.TrimSpace(customer.Name)
	if customer.Name == "" {
		return domain.NewAppError(400, "ชื่อลูกค้าจำเป็นต้องระบุ")
	}

	customer.ID = uuid.New().String()
	customer.Version = 1
	customer.CreatedAt = time.Now()
	customer.UpdatedAt = time.Now()

	account := domain.CreditAccount{
		ID:                 uuid.New().String(),
		CustomerID:         customer.ID,
		CreditLimit:        initialCreditLimit,
		OutstandingBalance: 0,
		Status:             domain.CreditStatusOK,
		Version:            1,
		UpdatedAt:          time.Now(),
	}
	return u.customerRepo.CreateWithCreditAccount(ctx, customer, &account)
}

func (u *customerUsecase) GetCustomerByID(ctx context.Context, id string) (*domain.Customer, error) {
	return u.customerRepo.GetByID(ctx, id)
}

func (u *customerUsecase) GetAllCustomers(ctx context.Context, filter domain.CustomerFilter) ([]domain.Customer, int64, error) {
	return u.customerRepo.GetAll(ctx, filter)
}

func (u *customerUsecase) UpdateCustomer(ctx context.Context, customer *domain.Customer) error {
	if strings.TrimSpace(customer.ID) == "" {
		return domain.NewAppError(400, "Customer ID is required")
	}
	customer.Name = strings.TrimSpace(customer.Name)
	if customer.Name == "" {
		return domain.NewAppError(400, "ชื่อลูกค้าจำเป็นต้องระบุ")
	}
	return u.customerRepo.Update(ctx, customer)
}

func (u *customerUsecase) DeleteCustomer(ctx context.Context, id string) error {
	return u.customerRepo.Delete(ctx, id)
}

func (u *customerUsecase) SetCreditLimit(ctx context.Context, customerID string, limit int64) error {
	if customerID == "" || limit < 0 {
		return domain.NewAppError(400, "ข้อมูลรหัสลูกค้าหรือวงเงินเครดิตไม่ถูกต้อง")
	}
	account, err := u.customerRepo.GetCreditAccount(ctx, customerID)
	if err == nil && account != nil {
		if limit < account.OutstandingBalance {
			return domain.NewAppError(400, "วงเงินเครดิตใหม่ต้องไม่ต่ำกว่ายอดหนี้คงค้างปัจจุบัน")
		}
	}
	return u.customerRepo.UpdateCreditLimit(ctx, customerID, limit)
}

func (u *customerUsecase) PayCredit(ctx context.Context, customerID string, amount int64, method string, note string, idempotencyKey *string, actorID string) (*domain.CreditTransaction, error) {
	if customerID == "" || amount <= 0 {
		return nil, domain.NewAppError(400, "กรุณาระบุรหัสลูกค้าและยอดเงินที่ชำระที่มากกว่า 0")
	}

	customer, err := u.customerRepo.GetByID(ctx, customerID)
	if err != nil || customer == nil {
		return nil, domain.NewAppError(404, "ไม่พบข้อมูลลูกค้า")
	}

	account, err := u.customerRepo.GetCreditAccount(ctx, customerID)
	if err != nil || account == nil {
		return nil, domain.NewAppError(404, "ไม่พบบัญชีเงินเชื่อของลูกค้า")
	}

	if account.OutstandingBalance <= 0 {
		return nil, domain.NewAppError(400, "ลูกค้าไม่มียอดหนี้คงค้างที่ต้องชำระ")
	}
	if amount > account.OutstandingBalance {
		return nil, domain.NewAppError(400, "ยอดเงินที่ชำระเกินยอดหนี้คงค้าง")
	}

	payMethod := method
	if payMethod == "" {
		payMethod = "CASH"
	}

	var notePtr *string
	if strings.TrimSpace(note) != "" {
		n := strings.TrimSpace(note)
		notePtr = &n
	}

	var actorPtr *string
	if actorID != "" {
		actorPtr = &actorID
	}

	txRecord := domain.CreditTransaction{
		ID:             uuid.New().String(),
		AccountID:      account.ID,
		CustomerID:     customerID,
		Type:           domain.CreditTxPayment,
		Amount:         -amount, // ลดหนี้
		RefType:        "PAYMENT",
		RefID:          uuid.New().String(),
		PaymentMethod:  &payMethod,
		Note:           notePtr,
		IdempotencyKey: idempotencyKey,
		OccurredAt:     time.Now(),
		CreatedBy:      actorPtr,
		CreatedAt:      time.Now(),
	}

	if err := u.customerRepo.RecordCreditTransaction(ctx, &txRecord); err != nil {
		return nil, err
	}

	return &txRecord, nil
}

func (u *customerUsecase) GetCreditHistory(ctx context.Context, customerID string, limit int) ([]domain.CreditTransaction, error) {
	return u.customerRepo.GetCreditHistory(ctx, customerID, limit)
}

func (u *customerUsecase) GetARAgingReport(ctx context.Context) ([]domain.ARAgingItem, error) {
	return u.customerRepo.GetARAgingReport(ctx)
}
