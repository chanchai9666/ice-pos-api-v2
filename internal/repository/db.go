package repository

import (
	"fmt"
	"log/slog"
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/password"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func NewDatabase(cfg *config.Config) (*gorm.DB, error) {
	logLevel := gormlogger.Warn
	if cfg.AppEnv == "development" {
		logLevel = gormlogger.Info
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(sqlite.Open(cfg.DBPath), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate simplified schema
	err = db.AutoMigrate(
		&domain.User{},
		&domain.Category{},
		&domain.Product{},
		&domain.Sale{},
		&domain.SaleItem{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	// Seed initial users if empty
	seedUsers(db)

	return db, nil
}

func seedUsers(db *gorm.DB) {
	var count int64
	db.Model(&domain.User{}).Count(&count)
	if count > 0 {
		return
	}

	slog.Info("Seeding initial users for ice-pos-api-v2...")

	adminPwd, _ := password.Hash("admin1234")
	adminPin, _ := password.Hash("1234")
	admin := domain.User{
		ID:           uuid.New().String(),
		Username:     "admin",
		PasswordHash: adminPwd,
		PinHash:      adminPin,
		DisplayName:  "ผู้ดูแลระบบ (Admin)",
		Role:         domain.RoleAdmin,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	cashierPwd, _ := password.Hash("cashier1234")
	cashierPin, _ := password.Hash("0000")
	cashier := domain.User{
		ID:           uuid.New().String(),
		Username:     "cashier",
		PasswordHash: cashierPwd,
		PinHash:      cashierPin,
		DisplayName:  "พนักงานหน้าร้าน (Cashier)",
		Role:         domain.RoleCashier,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	db.Create(&admin)
	db.Create(&cashier)

	// Seed sample category and products
	defaultCat := domain.Category{
		ID:        uuid.New().String(),
		Name:      "สินค้าทั่วไป",
		SortOrder: 1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.Create(&defaultCat)

	p1 := domain.Product{
		ID:         uuid.New().String(),
		Code:       "P001",
		Barcode:    "885000000001",
		Name:       "น้ำดื่ม 600ml",
		Price:      1000, // 10.00 THB in satang
		CostPrice:  600,  // 6.00 THB in satang
		CategoryID: defaultCat.ID,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	p2 := domain.Product{
		ID:         uuid.New().String(),
		Code:       "P002",
		Barcode:    "885000000002",
		Name:       "ขนมปังเนยสด",
		Price:      2500, // 25.00 THB
		CostPrice:  1500, // 15.00 THB
		CategoryID: defaultCat.ID,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	db.Create(&p1)
	db.Create(&p2)
}
