package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/delivery/http/handler"
	"ice-pos-api-v2/internal/delivery/http/middleware"
	"ice-pos-api-v2/internal/delivery/http/router"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/repository"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func setupTestApp(t *testing.T) (*fiber.App, func()) {
	dbFile := fmt.Sprintf("test_%d.db", time.Now().UnixNano())

	cfg := &config.Config{
		AppName:       "ICE POS v2 Test",
		AppEnv:        "development",
		JWTSecret:     "test-secret",
		JWTAccessTTL:  time.Hour,
		JWTRefreshTTL: 24 * time.Hour,
		DBType:        "sqlite",
		DBPath:        dbFile,
	}

	db, err := repository.NewDatabase(cfg)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	jwtService := jwt.NewService(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	userRepo := repository.NewUserRepository(db)
	catalogRepo := repository.NewCatalogRepository(db)
	saleRepo := repository.NewSaleRepository(db)
	reportRepo := repository.NewReportRepository(db)

	customerRepo := repository.NewCustomerRepository(db)
	customerUc := usecase.NewCustomerUsecase(customerRepo)

	authUc := usecase.NewAuthUsecase(userRepo, jwtService)
	userUc := usecase.NewUserUsecase(userRepo)
	catUc := usecase.NewCatalogUsecase(catalogRepo)
	saleUc := usecase.NewSaleUsecase(saleRepo, catalogRepo, customerRepo)
	reportUc := usecase.NewReportUsecase(reportRepo)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
	})

	router.SetupRoutes(&router.RouterConfig{
		App:             app,
		Config:          cfg,
		JWTService:      jwtService,
		HealthHandler:   handler.NewHealthHandler(),
		AuthHandler:     handler.NewAuthHandler(authUc, cfg),
		UserHandler:     handler.NewUserHandler(userUc),
		CatalogHandler:  handler.NewCatalogHandler(catUc),
		CustomerHandler: handler.NewCustomerHandler(customerUc),
		SaleHandler:     handler.NewSaleHandler(saleUc),
		ReportHandler:   handler.NewReportHandler(reportUc),
	})

	cleanup := func() {
		_ = os.Remove(dbFile)
	}

	return app, cleanup
}

func TestE2EFullFlow(t *testing.T) {
	app, cleanup := setupTestApp(t)
	defer cleanup()

	// 1. Health check
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/health", nil)
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v, status: %d", err, resp.StatusCode)
	}

	// 2. PIN Login with default cashier PIN "0000"
	loginBody, _ := json.Marshal(map[string]string{"pin": "0000"})
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/auth/pin-login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("pin login failed: status %d", resp.StatusCode)
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&loginResp)
	token := loginResp.Data.AccessToken
	if token == "" {
		t.Fatalf("expected access token from pin login")
	}

	// 3. Lookup products
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/products", nil)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("list products failed: status %d", resp.StatusCode)
	}
	var productsResp struct {
		Data []domain.Product `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&productsResp)
	if len(productsResp.Data) == 0 {
		t.Fatalf("expected seeded products")
	}
	p1 := productsResp.Data[0] // 10.00 THB (1000 satang)

	// 4. Cash Checkout: Buy 3 of P1 (30.00 THB), tendered 100.00 THB (10000 satang)
	checkoutPayload := map[string]interface{}{
		"lines": []map[string]interface{}{
			{
				"product_id":    p1.ID,
				"qty":           3,
				"line_discount": 0,
			},
		},
		"bill_discount":   0,
		"tendered_amount": 10000, // 100 THB
	}
	body, _ := json.Marshal(checkoutPayload)
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("checkout failed: status %d", resp.StatusCode)
	}

	var saleResp struct {
		Data struct {
			Sale    domain.Sale    `json:"sale"`
			Receipt domain.Receipt `json:"receipt"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&saleResp)
	sale := saleResp.Data.Sale
	if sale.GrandTotal != 3000 {
		t.Errorf("expected grand total 3000 satang, got %d", sale.GrandTotal)
	}
	if sale.ChangeAmount != 7000 { // 100 - 30 = 70 THB (7000 satang)
		t.Errorf("expected change 7000 satang, got %d", sale.ChangeAmount)
	}

	// 5. Check Daily Sales Summary
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/reports/sales-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("daily summary failed: status %d", resp.StatusCode)
	}
	var reportResp struct {
		Data domain.DailySalesSummary `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&reportResp)
	if reportResp.Data.TotalSales != 3000 {
		t.Errorf("expected total sales 3000, got %d", reportResp.Data.TotalSales)
	}
	if reportResp.Data.BillCount != 1 {
		t.Errorf("expected 1 bill, got %d", reportResp.Data.BillCount)
	}

	// 6. Void Sale: Cashier voids the sale
	voidPayload := map[string]string{"reason": "ลูกค้าเปลี่ยนใจ"}
	voidBody, _ := json.Marshal(voidPayload)
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/sales/%s/void", sale.ID), bytes.NewReader(voidBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("void sale failed: status %d", resp.StatusCode)
	}

	// 7. Check Daily Sales Summary after void: Total sales should be 0, VoidCount should be 1
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/reports/sales-summary", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("daily summary after void failed: status %d", resp.StatusCode)
	}
	var reportAfterVoidResp struct {
		Data domain.DailySalesSummary `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&reportAfterVoidResp)
	if reportAfterVoidResp.Data.TotalSales != 0 {
		t.Errorf("expected 0 total sales after void, got %d", reportAfterVoidResp.Data.TotalSales)
	}
	if reportAfterVoidResp.Data.VoidCount != 1 {
		t.Errorf("expected 1 void count, got %d", reportAfterVoidResp.Data.VoidCount)
	}
}
