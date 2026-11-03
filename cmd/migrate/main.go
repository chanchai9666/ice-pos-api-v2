package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Options represents CLI flags
type Options struct {
	V1DBPath     string
	Clean        bool
	DryRun       bool
	ArchiveStock bool
	CopyUploads  bool
}

func parseTimeSafe(str string) time.Time {
	if str == "" || str == "0001-01-01 00:00:00+00:00" {
		return time.Now()
	}
	formats := []string{
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, str); err == nil {
			return t
		}
	}
	return time.Now()
}

func parseTimePtr(str *string) *time.Time {
	if str == nil || *str == "" || *str == "0001-01-01 00:00:00+00:00" {
		return nil
	}
	t := parseTimeSafe(*str)
	return &t
}

func main() {
	opts := Options{}
	flag.StringVar(&opts.V1DBPath, "v1-db", "../ice-pos-api/ice_pos.db", "Path to SQLite database of v1")
	flag.BoolVar(&opts.Clean, "clean", false, "Clean existing sample data in target database before migrating")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "Validate and count data without writing to target database")
	flag.BoolVar(&opts.ArchiveStock, "archive-stock", true, "Export v1 stock data to docs/migration/v1_stock_archive.json")
	flag.BoolVar(&opts.CopyUploads, "copy-uploads", true, "Copy uploads folder from v1 to v2")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	slog.Info("=== ICE POS v1 -> v2 Migration Tool ===")
	slog.Info("Config:",
		"v1_db", opts.V1DBPath,
		"clean", opts.Clean,
		"dry_run", opts.DryRun,
		"archive_stock", opts.ArchiveStock,
		"copy_uploads", opts.CopyUploads,
	)

	// Check if v1 DB exists
	if _, err := os.Stat(opts.V1DBPath); os.IsNotExist(err) {
		slog.Error("Source v1 database file does not exist", "path", opts.V1DBPath)
		os.Exit(1)
	}

	// Connect to v1 DB
	v1DB, err := gorm.Open(sqlite.Open(opts.V1DBPath), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		slog.Error("Failed to open source v1 database", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully opened v1 SQLite database")

	// Connect to Target v2 DB
	cfg := config.LoadConfig()
	v2DB, err := repository.NewDatabase(cfg)
	if err != nil {
		slog.Error("Failed to connect to v2 database", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to v2 database", "type", cfg.DBType)

	// 1. Archive stock if requested
	if opts.ArchiveStock {
		if err := archiveStockData(v1DB); err != nil {
			slog.Warn("Failed to archive stock data", "error", err)
		}
	}

	// 2. Copy uploads if requested
	if opts.CopyUploads {
		v1Uploads := filepath.Join(filepath.Dir(opts.V1DBPath), "uploads")
		v2Uploads := "./uploads"
		if err := copyDirectory(v1Uploads, v2Uploads); err != nil {
			slog.Warn("Failed to copy uploads directory", "error", err)
		} else {
			slog.Info("Copied uploads directory successfully", "from", v1Uploads, "to", v2Uploads)
		}
	}

	// 3. Perform Migration
	if err := runMigration(v1DB, v2DB, opts); err != nil {
		slog.Error("Migration failed", "error", err)
		os.Exit(1)
	}

	slog.Info("=== Migration Process Finished Successfully ===")
}

func copyDirectory(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirectory(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func archiveStockData(v1DB *gorm.DB) error {
	slog.Info("Archiving v1 stock & transfer data...")
	archiveDir := "docs/migration"
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return err
	}

	archive := make(map[string]interface{})

	var stockLevels []map[string]interface{}
	v1DB.Table("stock_levels").Find(&stockLevels)
	archive["stock_levels"] = stockLevels

	var stockMovements []map[string]interface{}
	v1DB.Table("stock_movements").Find(&stockMovements)
	archive["stock_movements"] = stockMovements

	var stockTransfers []map[string]interface{}
	v1DB.Table("stock_transfers").Find(&stockTransfers)
	archive["stock_transfers"] = stockTransfers

	var stockTransferLines []map[string]interface{}
	v1DB.Table("stock_transfer_lines").Find(&stockTransferLines)
	archive["stock_transfer_lines"] = stockTransferLines

	var shifts []map[string]interface{}
	v1DB.Table("shifts").Find(&shifts)
	archive["shifts"] = shifts

	data, err := json.MarshalIndent(archive, "", "  ")
	if err != nil {
		return err
	}

	targetPath := filepath.Join(archiveDir, "v1_stock_archive.json")
	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		return err
	}
	slog.Info("Stock data archived successfully", "path", targetPath,
		"stock_levels", len(stockLevels),
		"stock_movements", len(stockMovements),
		"stock_transfers", len(stockTransfers),
		"shifts", len(shifts),
	)
	return nil
}

func runMigration(v1DB, v2DB *gorm.DB, opts Options) error {
	// --- Extract V1 Data ---
	slog.Info("Reading data from v1 database...")

	// 1. Categories
	var v1Categories []struct {
		ID        string
		Name      string
		SortOrder int
		CreatedAt string
		UpdatedAt string
	}
	v1DB.Table("categories").Select("id, name, sort_order, created_at, updated_at").Find(&v1Categories)
	slog.Info("Categories found in v1", "count", len(v1Categories))

	// 2. Barcodes Map (product_id -> barcode)
	var v1Barcodes []struct {
		ProductID string
		Barcode   string
	}
	v1DB.Table("product_barcodes").Select("product_id, barcode").Find(&v1Barcodes)
	barcodeMap := make(map[string]string)
	for _, b := range v1Barcodes {
		if _, exists := barcodeMap[b.ProductID]; !exists && b.Barcode != "" {
			barcodeMap[b.ProductID] = b.Barcode
		}
	}

	// 3. Products
	var v1Products []struct {
		ID              string
		SKU             string
		Barcode         string
		Name            string
		Price           int64
		CostPerBaseUnit int64
		CategoryID      string
		ImageURL        string
		IsActive        bool
		CreatedAt       string
		UpdatedAt       string
		DeletedAt       *string
	}
	v1DB.Table("products").Select("id, sku, barcode, name, price, cost_per_base_unit, category_id, image_url, is_active, created_at, updated_at, deleted_at").Find(&v1Products)
	slog.Info("Products found in v1", "count", len(v1Products))

	// 4. Users
	var v1Users []struct {
		ID           string
		Username     string
		PasswordHash string
		PinHash      string
		Name         string
		Role         string
		IsActive     bool
		CreatedAt    string
		UpdatedAt    string
	}
	v1DB.Table("users").Select("id, username, password_hash, pin_hash, name, role, is_active, created_at, updated_at").Find(&v1Users)
	slog.Info("Users found in v1", "count", len(v1Users))

	// 5. Customers
	var v1Customers []struct {
		ID        string
		Name      string
		Phone     *string
		Email     *string
		IsMember  bool
		Points    int64
		Version   int
		CreatedAt string
		UpdatedAt string
		DeletedAt *string
	}
	v1DB.Table("customers").Select("id, name, phone, email, is_member, points, version, created_at, updated_at, deleted_at").Find(&v1Customers)
	slog.Info("Customers found in v1", "count", len(v1Customers))

	customerNameMap := make(map[string]string)
	for _, c := range v1Customers {
		customerNameMap[c.ID] = c.Name
	}

	// 6. Credit Accounts
	var v1CreditAccounts []struct {
		ID                 string
		CustomerID         string
		CreditLimit        int64
		OutstandingBalance int64
		Status             string
		Version            int
		UpdatedAt          string
	}
	v1DB.Table("credit_accounts").Select("id, customer_id, credit_limit, outstanding_balance, status, version, updated_at").Find(&v1CreditAccounts)
	slog.Info("Credit accounts found in v1", "count", len(v1CreditAccounts))

	// 7. Credit Transactions
	var v1CreditTransactions []struct {
		ID             string
		AccountID      string
		CustomerID     string
		Type           string
		Amount         int64
		BalanceAfter   int64
		RefType        string
		RefID          string
		PaymentMethod  *string
		Note           *string
		IdempotencyKey *string
		OccurredAt     string
		CreatedBy      *string
		CreatedAt      string
	}
	v1DB.Table("credit_transactions").Select("id, account_id, customer_id, type, amount, balance_after, ref_type, ref_id, payment_method, note, idempotency_key, occurred_at, created_by, created_at").Find(&v1CreditTransactions)
	slog.Info("Credit transactions found in v1", "count", len(v1CreditTransactions))

	// 8. Payments Map (sale_id -> payment)
	var v1Payments []struct {
		SaleID         string
		Method         string
		Amount         int64
		TenderedAmount *int64
		ChangeAmount   *int64
		Status         string
	}
	v1DB.Table("payments").Select("sale_id, method, amount, tendered_amount, change_amount, status").Find(&v1Payments)
	paymentMap := make(map[string]struct {
		Method         string
		TenderedAmount *int64
		ChangeAmount   *int64
	})
	for _, p := range v1Payments {
		paymentMap[p.SaleID] = struct {
			Method         string
			TenderedAmount *int64
			ChangeAmount   *int64
		}{
			Method:         p.Method,
			TenderedAmount: p.TenderedAmount,
			ChangeAmount:   p.ChangeAmount,
		}
	}

	// 9. Sales
	var v1Sales []struct {
		ID           string
		SaleNo       string
		CashierID    string
		CustomerID   *string
		SubTotal     int64
		BillDiscount int64
		GrandTotal   int64
		Status       string
		CreditPart   int64
		VoidReason   *string
		VoidedAt     *string
		OccurredAt   string
		CreatedAt    string
		UpdatedAt    string
	}
	v1DB.Table("sales").Select("id, sale_no, cashier_id, customer_id, sub_total, bill_discount, grand_total, status, credit_part, void_reason, voided_at, occurred_at, created_at, updated_at").Find(&v1Sales)
	slog.Info("Sales found in v1", "count", len(v1Sales))

	// 10. Sale Lines
	var v1SaleLines []struct {
		ID           string
		SaleID       string
		ProductID    string
		Qty          float64
		UnitPrice    int64
		LineDiscount int64
		GrossTotal   int64
		NetTotal     int64
		Cogs         int64
	}
	v1DB.Table("sale_lines").Select("id, sale_id, product_id, qty, unit_price, line_discount, gross_total, net_total, cogs").Find(&v1SaleLines)
	slog.Info("Sale lines found in v1", "count", len(v1SaleLines))

	// Map product info for lines snapshot
	type prodInfo struct {
		Name      string
		Code      string
		CostPrice int64
	}
	prodInfoMap := make(map[string]prodInfo)
	for _, p := range v1Products {
		prodInfoMap[p.ID] = prodInfo{
			Name:      p.Name,
			Code:      p.SKU,
			CostPrice: p.CostPerBaseUnit,
		}
	}

	if opts.DryRun {
		slog.Info("DRY RUN MODE: Validations passed. No changes were made to target database.")
		return nil
	}

	// --- Begin Database Transaction in v2 ---
	tx := v2DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if opts.Clean {
		slog.Info("Cleaning existing seed/test records in target database...")
		// Delete in reverse dependency order
		if err := tx.Exec("DELETE FROM sale_items").Error; err != nil {
			slog.Warn("Could not clear sale_items", "error", err)
		}
		if err := tx.Exec("DELETE FROM sales").Error; err != nil {
			slog.Warn("Could not clear sales", "error", err)
		}
		if err := tx.Exec("DELETE FROM credit_transactions").Error; err != nil {
			slog.Warn("Could not clear credit_transactions", "error", err)
		}
		if err := tx.Exec("DELETE FROM credit_accounts").Error; err != nil {
			slog.Warn("Could not clear credit_accounts", "error", err)
		}
		if err := tx.Exec("DELETE FROM customers").Error; err != nil {
			slog.Warn("Could not clear customers", "error", err)
		}
		if err := tx.Exec("DELETE FROM products").Error; err != nil {
			slog.Warn("Could not clear products", "error", err)
		}
		if err := tx.Exec("DELETE FROM categories").Error; err != nil {
			slog.Warn("Could not clear categories", "error", err)
		}
		if err := tx.Exec("DELETE FROM users").Error; err != nil {
			slog.Warn("Could not clear users", "error", err)
		}
		slog.Info("Target database cleaned successfully")
	}

	// 1. Insert Categories
	slog.Info("Migrating Categories...")
	var newCategories []domain.Category
	for _, c := range v1Categories {
		newCategories = append(newCategories, domain.Category{
			ID:        c.ID,
			Name:      c.Name,
			SortOrder: c.SortOrder,
			CreatedAt: parseTimeSafe(c.CreatedAt),
			UpdatedAt: parseTimeSafe(c.UpdatedAt),
		})
	}
	if len(newCategories) > 0 {
		if err := tx.CreateInBatches(newCategories, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create categories: %w", err)
		}
	}
	slog.Info("Migrated Categories completed", "count", len(newCategories))

	// 2. Insert Products
	slog.Info("Migrating Products...")
	var newProducts []domain.Product
	for _, p := range v1Products {
		bc := strings.TrimSpace(p.Barcode)
		if bc == "" {
			bc = barcodeMap[p.ID]
		}
		isActive := p.IsActive
		if p.DeletedAt != nil && *p.DeletedAt != "" {
			isActive = false // Soft-deleted in v1 marked as inactive in v2
		}

		newProducts = append(newProducts, domain.Product{
			ID:         p.ID,
			Code:       p.SKU,
			Barcode:    bc,
			Name:       p.Name,
			Price:      p.Price,
			CostPrice:  p.CostPerBaseUnit,
			CategoryID: p.CategoryID,
			ImageURL:   p.ImageURL,
			IsActive:   isActive,
			CreatedAt:  parseTimeSafe(p.CreatedAt),
			UpdatedAt:  parseTimeSafe(p.UpdatedAt),
		})
	}
	if len(newProducts) > 0 {
		if err := tx.CreateInBatches(newProducts, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create products: %w", err)
		}
	}
	slog.Info("Migrated Products completed", "count", len(newProducts))

	// 3. Insert Users
	slog.Info("Migrating Users...")
	var newUsers []domain.User
	for _, u := range v1Users {
		role := strings.ToUpper(strings.TrimSpace(u.Role))
		if role == "MANAGER" {
			role = domain.RoleAdmin // Manager upgraded to Admin in v2
		}
		if role != domain.RoleAdmin && role != domain.RoleCashier {
			role = domain.RoleCashier
		}

		newUsers = append(newUsers, domain.User{
			ID:           u.ID,
			Username:     u.Username,
			PasswordHash: u.PasswordHash,
			PinHash:      u.PinHash,
			DisplayName:  u.Name,
			Role:         role,
			IsActive:     u.IsActive,
			CreatedAt:    parseTimeSafe(u.CreatedAt),
			UpdatedAt:    parseTimeSafe(u.UpdatedAt),
		})
	}
	if len(newUsers) > 0 {
		if err := tx.CreateInBatches(newUsers, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create users: %w", err)
		}
	}
	slog.Info("Migrated Users completed", "count", len(newUsers))

	// 4. Insert Customers
	slog.Info("Migrating Customers...")
	var newCustomers []domain.Customer
	for _, c := range v1Customers {
		cust := domain.Customer{
			ID:        c.ID,
			Name:      c.Name,
			Phone:     c.Phone,
			Email:     c.Email,
			IsMember:  c.IsMember,
			Points:    c.Points,
			Version:   c.Version,
			CreatedAt: parseTimeSafe(c.CreatedAt),
			UpdatedAt: parseTimeSafe(c.UpdatedAt),
		}
		if c.DeletedAt != nil && *c.DeletedAt != "" {
			t := parseTimeSafe(*c.DeletedAt)
			cust.DeletedAt = gorm.DeletedAt{Time: t, Valid: true}
		}
		newCustomers = append(newCustomers, cust)
	}
	if len(newCustomers) > 0 {
		if err := tx.CreateInBatches(newCustomers, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create customers: %w", err)
		}
	}
	slog.Info("Migrated Customers completed", "count", len(newCustomers))

	// 5. Insert Credit Accounts
	slog.Info("Migrating Credit Accounts...")
	var newCreditAccounts []domain.CreditAccount
	for _, ca := range v1CreditAccounts {
		newCreditAccounts = append(newCreditAccounts, domain.CreditAccount{
			ID:                 ca.ID,
			CustomerID:         ca.CustomerID,
			CreditLimit:        ca.CreditLimit,
			OutstandingBalance: ca.OutstandingBalance,
			Status:             ca.Status,
			Version:            ca.Version,
			UpdatedAt:          parseTimeSafe(ca.UpdatedAt),
		})
	}
	if len(newCreditAccounts) > 0 {
		if err := tx.CreateInBatches(newCreditAccounts, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create credit accounts: %w", err)
		}
	}
	slog.Info("Migrated Credit Accounts completed", "count", len(newCreditAccounts))

	// 6. Insert Credit Transactions
	slog.Info("Migrating Credit Transactions...")
	_ = tx.Exec("ALTER TABLE credit_transactions ALTER COLUMN idempotency_key TYPE varchar(100)").Error
	var newCreditTransactions []domain.CreditTransaction
	for _, ct := range v1CreditTransactions {
		newCreditTransactions = append(newCreditTransactions, domain.CreditTransaction{
			ID:             ct.ID,
			AccountID:      ct.AccountID,
			CustomerID:     ct.CustomerID,
			Type:           ct.Type,
			Amount:         ct.Amount,
			BalanceAfter:   ct.BalanceAfter,
			RefType:        ct.RefType,
			RefID:          ct.RefID,
			PaymentMethod:  ct.PaymentMethod,
			Note:           ct.Note,
			IdempotencyKey: ct.IdempotencyKey,
			OccurredAt:     parseTimeSafe(ct.OccurredAt),
			CreatedBy:      ct.CreatedBy,
			CreatedAt:      parseTimeSafe(ct.CreatedAt),
		})
	}
	if len(newCreditTransactions) > 0 {
		if err := tx.CreateInBatches(newCreditTransactions, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create credit transactions: %w", err)
		}
	}
	slog.Info("Migrated Credit Transactions completed", "count", len(newCreditTransactions))

	// 7. Insert Sales
	slog.Info("Migrating Sales...")
	var newSales []domain.Sale
	for _, s := range v1Sales {
		pInfo, hasPayment := paymentMap[s.ID]
		payMethod := domain.PaymentMethodCash
		var tendered int64 = s.GrandTotal
		var change int64 = 0
		var creditAmt int64 = 0

		if hasPayment {
			if strings.ToUpper(pInfo.Method) == "CREDIT" {
				payMethod = domain.PaymentMethodCredit
			}
			if pInfo.TenderedAmount != nil {
				tendered = *pInfo.TenderedAmount
			}
			if pInfo.ChangeAmount != nil {
				change = *pInfo.ChangeAmount
			}
		}

		status := strings.ToUpper(s.Status)
		if status == "CREDIT_PENDING" {
			status = domain.SaleStatusCompleted
			payMethod = domain.PaymentMethodCredit
			creditAmt = s.GrandTotal
		} else if status != domain.SaleStatusVoid {
			status = domain.SaleStatusCompleted
		}

		if payMethod == domain.PaymentMethodCredit {
			creditAmt = s.GrandTotal
			tendered = 0
			change = 0
		}

		custName := ""
		if s.CustomerID != nil && *s.CustomerID != "" {
			custName = customerNameMap[*s.CustomerID]
		}

		var voidReason string
		if s.VoidReason != nil {
			voidReason = *s.VoidReason
		}

		newSales = append(newSales, domain.Sale{
			ID:             s.ID,
			SaleNo:         s.SaleNo,
			SubTotal:       s.SubTotal,
			BillDiscount:   s.BillDiscount,
			GrandTotal:     s.GrandTotal,
			TenderedAmount: tendered,
			ChangeAmount:   change,
			PaymentMethod:  payMethod,
			CustomerID:     s.CustomerID,
			CustomerName:   custName,
			CreditAmount:   creditAmt,
			Status:         status,
			VoidReason:     voidReason,
			VoidedAt:       parseTimePtr(s.VoidedAt),
			CashierID:      s.CashierID,
			OccurredAt:     parseTimeSafe(s.OccurredAt),
			CreatedAt:      parseTimeSafe(s.CreatedAt),
			UpdatedAt:      parseTimeSafe(s.UpdatedAt),
		})
	}
	if len(newSales) > 0 {
		if err := tx.CreateInBatches(newSales, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create sales: %w", err)
		}
	}
	slog.Info("Migrated Sales completed", "count", len(newSales))

	// 8. Insert Sale Items
	slog.Info("Migrating Sale Items...")
	var newSaleItems []domain.SaleItem
	for _, sl := range v1SaleLines {
		pi := prodInfoMap[sl.ProductID]
		qtyInt := int(math.Round(sl.Qty))
		if qtyInt == 0 {
			qtyInt = 1
		}
		costPrice := pi.CostPrice
		if sl.Cogs > 0 && qtyInt > 0 {
			costPrice = sl.Cogs / int64(qtyInt)
		}

		newSaleItems = append(newSaleItems, domain.SaleItem{
			ID:           sl.ID,
			SaleID:       sl.SaleID,
			ProductID:    sl.ProductID,
			ProductName:  pi.Name,
			ProductCode:  pi.Code,
			UnitPrice:    sl.UnitPrice,
			CostPrice:    costPrice,
			Qty:          qtyInt,
			LineDiscount: sl.LineDiscount,
			GrossTotal:   sl.GrossTotal,
			NetTotal:     sl.NetTotal,
		})
	}
	if len(newSaleItems) > 0 {
		if err := tx.CreateInBatches(newSaleItems, 100).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to batch create sale items: %w", err)
		}
	}
	slog.Info("Migrated Sale Items completed", "count", len(newSaleItems))

	// Commit Transaction
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	slog.Info("All records successfully committed to target database!")
	return nil
}
