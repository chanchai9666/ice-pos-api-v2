package repository

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/password"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
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
		Logger:      gormlogger.Default.LogMode(logLevel),
		PrepareStmt: false,
	}

	var dialector gorm.Dialector
	dbType := strings.ToLower(cfg.DBType)

	switch dbType {
	case "postgres", "postgresql":
		dsn := fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
			cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode,
		)
		if cfg.DBSchema != "" {
			dsn = fmt.Sprintf("%s search_path=%s", dsn, cfg.DBSchema)
		}
		slog.Info("Connecting to PostgreSQL/Supabase database...", "host", cfg.DBHost, "port", cfg.DBPort, "dbname", cfg.DBName)
		dialector = postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true, // Disables prepared statements for Supabase PgBouncer (port 6543)
		})
	default:
		slog.Info("Connecting to SQLite database...", "path", cfg.DBPath)
		dialector = sqlite.Open(cfg.DBPath)
	}

	db, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database (%s): %w", dbType, err)
	}

	// Auto-migrate simplified schema
	err = db.AutoMigrate(
		&domain.User{},
		&domain.Category{},
		&domain.Product{},
		&domain.ProductImage{},
		&domain.Customer{},
		&domain.CreditAccount{},
		&domain.CreditTransaction{},
		&domain.Sale{},
		&domain.SaleItem{},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	// Backfill existing product images if needed
	backfillProductImages(db)

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

func backfillProductImages(db *gorm.DB) {
	var products []domain.Product
	err := db.Where("image_url != '' AND image_url IS NOT NULL AND id NOT IN (SELECT DISTINCT product_id FROM product_images)").Find(&products).Error
	if err != nil {
		slog.Warn("Failed to query products for image backfill:", "error", err)
		return
	}

	for _, p := range products {
		img := domain.ProductImage{
			ID:        uuid.New().String(),
			ProductID: p.ID,
			ImageURL:  p.ImageURL,
			IsPrimary: true,
			SortOrder: 0,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		}
		if err := db.Create(&img).Error; err != nil {
			slog.Warn("Failed to backfill product image:", "product_id", p.ID, "error", err)
		}
	}
}
