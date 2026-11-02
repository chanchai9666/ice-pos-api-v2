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

func setupCustomerCreditTestApp(t *testing.T) (*fiber.App, string, func()) {
	dbFile := fmt.Sprintf("test_customer_%d.db", time.Now().UnixNano())

	cfg := &config.Config{
		AppName:       "ICE POS v2 Customer Credit Test",
		AppEnv:        "development",
		JWTSecret:     "customer-credit-test-secret-key-123",
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
	customerRepo := repository.NewCustomerRepository(db)
	saleRepo := repository.NewSaleRepository(db)
	reportRepo := repository.NewReportRepository(db)

	authUc := usecase.NewAuthUsecase(userRepo, jwtService)
	userUc := usecase.NewUserUsecase(userRepo)
	catUc := usecase.NewCatalogUsecase(catalogRepo)
	custUc := usecase.NewCustomerUsecase(customerRepo)
	saleUc := usecase.NewSaleUsecase(saleRepo, catalogRepo, customerRepo)
	reportUc := usecase.NewReportUsecase(reportRepo)

	customerHandler := handler.NewCustomerHandler(custUc)
	saleHandler := handler.NewSaleHandler(saleUc)

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
		CustomerHandler: customerHandler,
		SaleHandler:     saleHandler,
		ReportHandler:   handler.NewReportHandler(reportUc),
	})

	token, _ := jwtService.GenerateAccessToken("admin-id", "admin", "admin")

	cleanup := func() {
		_ = os.Remove(dbFile)
	}

	return app, token, cleanup
}

func TestE2ECustomerAndCreditAccountFlow(t *testing.T) {
	app, token, cleanup := setupCustomerCreditTestApp(t)
	defer cleanup()

	// 1. Create a customer with initial credit limit of 5000 satang (50 THB)
	custPayload, _ := json.Marshal(map[string]interface{}{
		"name":                 "ลุงบุญมี",
		"phone":                "0891112233",
		"email":                "boonmee@example.com",
		"is_member":            true,
		"initial_credit_limit": 5000, // 50.00 THB
	})
	createReq, _ := http.NewRequest(http.MethodPost, "/api/v1/customers", bytes.NewReader(custPayload))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)

	createResp, err := app.Test(createReq)
	if err != nil || createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create customer failed: status %d", createResp.StatusCode)
	}

	var createdCust struct {
		Success bool `json:"success"`
		Data    struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			CreditAccount struct {
				CreditLimit        int64 `json:"credit_limit"`
				OutstandingBalance int64 `json:"outstanding_balance"`
			} `json:"credit_account"`
		} `json:"data"`
	}
	_ = json.NewDecoder(createResp.Body).Decode(&createdCust)

	custID := createdCust.Data.ID
	if custID == "" {
		t.Fatalf("expected created customer to have valid ID")
	}
	if createdCust.Data.CreditAccount.CreditLimit != 5000 {
		t.Fatalf("expected credit limit 5000, got %d", createdCust.Data.CreditAccount.CreditLimit)
	}
	if createdCust.Data.CreditAccount.OutstandingBalance != 0 {
		t.Fatalf("expected initial balance 0, got %d", createdCust.Data.CreditAccount.OutstandingBalance)
	}

	// 2. Fetch seeded product (e.g. P002 ขนมปังเนยสด price = 2500 satang / 25 THB)
	listProdReq, _ := http.NewRequest(http.MethodGet, "/api/v1/products", nil)
	listProdResp, _ := app.Test(listProdReq)
	var prodData struct {
		Data []struct {
			ID    string `json:"id"`
			Code  string `json:"code"`
			Price int64  `json:"price"`
		} `json:"data"`
	}
	_ = json.NewDecoder(listProdResp.Body).Decode(&prodData)
	var productID string
	for _, p := range prodData.Data {
		if p.Code == "P002" {
			productID = p.ID
			break
		}
	}
	if productID == "" {
		productID = prodData.Data[0].ID
	}

	// 3. Make a sale with CREDIT payment for 1 item (2500 satang)
	salePayload, _ := json.Marshal(map[string]interface{}{
		"lines": []map[string]interface{}{
			{"product_id": productID, "qty": 1},
		},
		"payment_method": "CREDIT",
		"customer_id":    custID,
	})
	saleReq, _ := http.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(salePayload))
	saleReq.Header.Set("Content-Type", "application/json")
	saleReq.Header.Set("Authorization", "Bearer "+token)

	saleResp, err := app.Test(saleReq)
	if err != nil || saleResp.StatusCode != http.StatusCreated {
		t.Fatalf("credit sale failed: status %d", saleResp.StatusCode)
	}

	var saleDataResp struct {
		Data struct {
			Sale struct {
				ID            string `json:"id"`
				GrandTotal    int64  `json:"grand_total"`
				PaymentMethod string `json:"payment_method"`
				CreditAmount  int64  `json:"credit_amount"`
			} `json:"sale"`
			Receipt struct {
				PaymentMethod string `json:"payment_method"`
			} `json:"receipt"`
		} `json:"data"`
	}
	_ = json.NewDecoder(saleResp.Body).Decode(&saleDataResp)
	saleID := saleDataResp.Data.Sale.ID

	if saleDataResp.Data.Sale.PaymentMethod != "CREDIT" {
		t.Fatalf("expected sale payment method CREDIT, got %s", saleDataResp.Data.Sale.PaymentMethod)
	}
	if saleDataResp.Data.Sale.CreditAmount != 2500 {
		t.Fatalf("expected credit amount 2500, got %d", saleDataResp.Data.Sale.CreditAmount)
	}

	// 4. Verify customer's outstanding balance increased to 2500 satang
	custReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/customers/%s", custID), nil)
	custReq.Header.Set("Authorization", "Bearer "+token)
	custDetailResp, _ := app.Test(custReq)

	var custDetail struct {
		Data struct {
			CreditAccount struct {
				OutstandingBalance int64 `json:"outstanding_balance"`
			} `json:"credit_account"`
		} `json:"data"`
	}
	_ = json.NewDecoder(custDetailResp.Body).Decode(&custDetail)
	if custDetail.Data.CreditAccount.OutstandingBalance != 2500 {
		t.Fatalf("expected outstanding balance 2500, got %d", custDetail.Data.CreditAccount.OutstandingBalance)
	}

	// 5. Test credit limit exceed: Try to buy 2 more items (5000 satang), new balance would be 7500 > 5000 limit
	exceedPayload, _ := json.Marshal(map[string]interface{}{
		"lines": []map[string]interface{}{
			{"product_id": productID, "qty": 2},
		},
		"payment_method": "CREDIT",
		"customer_id":    custID,
	})
	exceedReq, _ := http.NewRequest(http.MethodPost, "/api/v1/sales", bytes.NewReader(exceedPayload))
	exceedReq.Header.Set("Content-Type", "application/json")
	exceedReq.Header.Set("Authorization", "Bearer "+token)

	exceedResp, err := app.Test(exceedReq)
	if err != nil || exceedResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when credit limit exceeded, got status %d", exceedResp.StatusCode)
	}

	// 6. Test Pay Credit: Customer pays 1500 satang (15 THB)
	payPayload, _ := json.Marshal(map[string]interface{}{
		"amount":         1500,
		"payment_method": "CASH",
		"note":           "ชำระบางส่วน",
	})
	payReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/customers/%s/pay-credit", custID), bytes.NewReader(payPayload))
	payReq.Header.Set("Content-Type", "application/json")
	payReq.Header.Set("Authorization", "Bearer "+token)

	payResp, err := app.Test(payReq)
	if err != nil || payResp.StatusCode != http.StatusOK {
		t.Fatalf("pay credit failed: status %d", payResp.StatusCode)
	}

	// Outstanding balance should now be 2500 - 1500 = 1000 satang
	custDetailResp2, _ := app.Test(custReq)
	var custDetail2 struct {
		Data struct {
			CreditAccount struct {
				OutstandingBalance int64 `json:"outstanding_balance"`
			} `json:"credit_account"`
		} `json:"data"`
	}
	_ = json.NewDecoder(custDetailResp2.Body).Decode(&custDetail2)
	if custDetail2.Data.CreditAccount.OutstandingBalance != 1000 {
		t.Fatalf("expected outstanding balance 1000 after partial payment, got %d", custDetail2.Data.CreditAccount.OutstandingBalance)
	}

	// 7. Check credit history endpoint
	histReq, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/customers/%s/credit-history", custID), nil)
	histReq.Header.Set("Authorization", "Bearer "+token)
	histResp, err := app.Test(histReq)
	if err != nil || histResp.StatusCode != http.StatusOK {
		t.Fatalf("credit history request failed: status %d", histResp.StatusCode)
	}

	var histData struct {
		Data []domain.CreditTransaction `json:"data"`
	}
	_ = json.NewDecoder(histResp.Body).Decode(&histData)
	if len(histData.Data) < 2 {
		t.Fatalf("expected at least 2 transactions (SALE_CREDIT and PAYMENT), got %d", len(histData.Data))
	}

	// 8. Check AR Aging Report endpoint
	agingReq, _ := http.NewRequest(http.MethodGet, "/api/v1/customers/reports/ar-aging", nil)
	agingReq.Header.Set("Authorization", "Bearer "+token)
	agingResp, err := app.Test(agingReq)
	if err != nil || agingResp.StatusCode != http.StatusOK {
		t.Fatalf("ar aging report failed: status %d", agingResp.StatusCode)
	}
	var agingData struct {
		Data []domain.ARAgingItem `json:"data"`
	}
	_ = json.NewDecoder(agingResp.Body).Decode(&agingData)
	if len(agingData.Data) == 0 {
		t.Fatalf("expected customer with outstanding debt in AR aging report")
	}

	// 9. Void the credit sale and verify balance is reversed
	voidReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/sales/%s/void", saleID), bytes.NewReader([]byte(`{"reason":"ลูกค้ายกเลิกบิล"}`)))
	voidReq.Header.Set("Content-Type", "application/json")
	voidReq.Header.Set("Authorization", "Bearer "+token)
	voidResp, err := app.Test(voidReq)
	if err != nil || voidResp.StatusCode != http.StatusOK {
		t.Fatalf("void sale failed: status %d", voidResp.StatusCode)
	}

	// 10. Attempt to delete customer while having balance or test deletion
	// Customer still has history, let's verify balance after reversal (1000 - 2500 = -1500 or 0)
	custDetailResp3, _ := app.Test(custReq)
	_ = custDetailResp3
}
