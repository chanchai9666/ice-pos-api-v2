package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/calc"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) domain.CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) CreateWithCreditAccount(ctx context.Context, customer *domain.Customer, account *domain.CreditAccount) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(customer).Error; err != nil {
			return err
		}
		if account != nil {
			account.CustomerID = customer.ID
			if err := tx.Create(account).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *customerRepository) GetByID(ctx context.Context, id string) (*domain.Customer, error) {
	var customer domain.Customer
	err := r.db.WithContext(ctx).
		Preload("CreditAccount").
		First(&customer, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &customer, nil
}

func (r *customerRepository) GetAll(ctx context.Context, filter domain.CustomerFilter) ([]domain.Customer, int64, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := r.db.WithContext(ctx).Model(&domain.Customer{}).Preload("CreditAccount")

	if filter.Search != "" {
		searchTerm := "%" + filter.Search + "%"
		query = query.Where("name LIKE ? OR phone LIKE ? OR email LIKE ?", searchTerm, searchTerm, searchTerm)
	}
	if filter.IsMember != nil {
		query = query.Where("is_member = ?", *filter.IsMember)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var customers []domain.Customer
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&customers).Error
	if err != nil {
		return nil, 0, err
	}

	return customers, total, nil
}

func (r *customerRepository) Update(ctx context.Context, customer *domain.Customer) error {
	customer.Version++
	customer.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(customer).Error
}

func (r *customerRepository) Delete(ctx context.Context, id string) error {
	var account domain.CreditAccount
	if err := r.db.WithContext(ctx).Where("customer_id = ?", id).First(&account).Error; err == nil {
		if account.OutstandingBalance > 0 {
			return domain.NewAppError(400, fmt.Sprintf("ไม่สามารถลบข้อมูลลูกค้าที่มียอดหนี้ค้างชำระ (%d สตางค์) ได้", account.OutstandingBalance))
		}
	}
	return r.db.WithContext(ctx).Delete(&domain.Customer{}, "id = ?", id).Error
}

func (r *customerRepository) GetCreditAccount(ctx context.Context, customerID string) (*domain.CreditAccount, error) {
	var account domain.CreditAccount
	err := r.db.WithContext(ctx).
		Preload("Customer").
		First(&account, "customer_id = ?", customerID).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &account, nil
}

func (r *customerRepository) UpdateCreditLimit(ctx context.Context, customerID string, limit int64) error {
	return r.db.WithContext(ctx).Model(&domain.CreditAccount{}).
		Where("customer_id = ?", customerID).
		Updates(map[string]interface{}{
			"credit_limit": limit,
			"updated_at":   time.Now(),
		}).Error
}

func (r *customerRepository) RecordCreditTransaction(ctx context.Context, txRecord *domain.CreditTransaction) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var account domain.CreditAccount
		err := tx.Where("customer_id = ?", txRecord.CustomerID).First(&account).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				account = domain.CreditAccount{
					ID:                 uuid.New().String(),
					CustomerID:         txRecord.CustomerID,
					CreditLimit:        0,
					OutstandingBalance: 0,
					Status:             domain.CreditStatusOK,
					Version:            1,
					UpdatedAt:          time.Now(),
				}
				if err := tx.Create(&account).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}

		newBalance := calc.AddSatang(account.OutstandingBalance, txRecord.Amount) // + increase debt, - pay debt

		if txRecord.Amount > 0 && account.CreditLimit > 0 && newBalance > account.CreditLimit {
			account.Status = domain.CreditStatusOverLimit
		} else {
			account.Status = domain.CreditStatusOK
		}

		account.OutstandingBalance = newBalance
		account.Version++
		account.UpdatedAt = time.Now()
		if err := tx.Save(&account).Error; err != nil {
			return err
		}

		txRecord.AccountID = account.ID
		txRecord.BalanceAfter = newBalance
		return tx.Create(txRecord).Error
	})
}

func (r *customerRepository) GetCreditHistory(ctx context.Context, customerID string, limit int) ([]domain.CreditTransaction, error) {
	var list []domain.CreditTransaction
	if limit <= 0 {
		limit = 50
	}
	err := r.db.WithContext(ctx).
		Where("customer_id = ?", customerID).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&list).Error
	return list, err
}

func (r *customerRepository) GetARAgingReport(ctx context.Context) ([]domain.ARAgingItem, error) {
	var accounts []domain.CreditAccount
	query := r.db.WithContext(ctx).
		Joins("JOIN customers ON customers.id = credit_accounts.customer_id AND customers.deleted_at IS NULL").
		Preload("Customer").
		Where("credit_accounts.outstanding_balance > 0")

	if err := query.Find(&accounts).Error; err != nil {
		return nil, err
	}

	now := time.Now()
	report := make([]domain.ARAgingItem, 0, len(accounts))

	txsByCustomer := make(map[string][]domain.CreditTransaction, len(accounts))
	if len(accounts) > 0 {
		customerIDs := make([]string, 0, len(accounts))
		for _, acc := range accounts {
			customerIDs = append(customerIDs, acc.CustomerID)
		}
		var allTxs []domain.CreditTransaction
		_ = r.db.WithContext(ctx).
			Where("customer_id IN ? AND type = ?", customerIDs, domain.CreditTxSaleCredit).
			Order("customer_id ASC, occurred_at DESC").
			Find(&allTxs).Error
		for _, tx := range allTxs {
			txsByCustomer[tx.CustomerID] = append(txsByCustomer[tx.CustomerID], tx)
		}
	}

	for _, acc := range accounts {
		txs := txsByCustomer[acc.CustomerID]
		buckets := domain.CalculateARAgingBuckets(acc.OutstandingBalance, txs, now)

		custName := ""
		custPhone := ""
		if acc.Customer != nil {
			custName = acc.Customer.Name
			if acc.Customer.Phone != nil {
				custPhone = *acc.Customer.Phone
			}
		}

		report = append(report, domain.ARAgingItem{
			CustomerID:         acc.CustomerID,
			CustomerName:       custName,
			Phone:              custPhone,
			CreditLimit:        acc.CreditLimit,
			OutstandingBalance: acc.OutstandingBalance,
			CurrentDue:         buckets.CurrentDue,
			Due30To60:          buckets.Due30To60,
			Due60To90:          buckets.Due60To90,
			Over90:             buckets.Over90,
			Status:             acc.Status,
		})
	}

	return report, nil
}
