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

func setupSyncTestApp(t *testing.T) (*fiber.App, func()) {
	dbFile := fmt.Sprintf("test_sync_%d.db", time.Now().UnixNano())

	cfg := &config.Config{
		AppName:       "ICE POS v2 Sync Test",
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
	syncRepo := repository.NewSyncRepository(db)

	authUc := usecase.NewAuthUsecase(userRepo, jwtService)
	userUc := usecase.NewUserUsecase(userRepo)
	catUc := usecase.NewCatalogUsecase(catalogRepo)
	saleUc := usecase.NewSaleUsecase(saleRepo, catalogRepo)
	reportUc := usecase.NewReportUsecase(reportRepo)
	syncUc := usecase.NewSyncUsecase(syncRepo)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
	})

	router.SetupRoutes(&router.RouterConfig{
		App:            app,
		Config:         cfg,
		JWTService:     jwtService,
		HealthHandler:  handler.NewHealthHandler(),
		AuthHandler:    handler.NewAuthHandler(authUc),
		UserHandler:    handler.NewUserHandler(userUc),
		CatalogHandler: handler.NewCatalogHandler(catUc),
		SaleHandler:    handler.NewSaleHandler(saleUc),
		ReportHandler:  handler.NewReportHandler(reportUc),
		SyncHandler:    handler.NewSyncHandler(syncUc),
	})

	cleanup := func() {
		_ = os.Remove(dbFile)
	}

	return app, cleanup
}

func TestE2EOfflineSync(t *testing.T) {
	app, cleanup := setupSyncTestApp(t)
	defer cleanup()

	// 1. PIN Login
	loginBody, _ := json.Marshal(map[string]string{"pin": "0000"})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/pin-login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("pin login failed: status %d", resp.StatusCode)
	}

	var loginResp struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&loginResp)
	token := loginResp.Data.AccessToken

	// 2. Pull Changes (Initial sync before going offline)
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/sync/changes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("pull changes failed: status %d", resp.StatusCode)
	}

	var pullResp struct {
		Data domain.SyncChangesResponse `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&pullResp)
	if len(pullResp.Data.Products) == 0 {
		t.Fatalf("expected products in pull changes")
	}
	p1 := pullResp.Data.Products[0]

	// 3. Simulate Tablet Offline Sales: 1 Completed, 1 Voided
	offlineSales := []domain.Sale{
		{
			ID:             "client-offline-uuid-1",
			SaleNo:         "SO-OFFLINE-0001",
			SubTotal:       2000, // 20.00 THB
			BillDiscount:   0,
			GrandTotal:     2000,
			TenderedAmount: 5000, // 50.00 THB
			ChangeAmount:   3000, // 30.00 THB
			Status:         domain.SaleStatusCompleted,
			CashierID:      "cashier",
			OccurredAt:     time.Now(),
			Lines: []domain.SaleItem{
				{
					ID:          "line-offline-1",
					ProductID:   p1.ID,
					ProductName: p1.Name,
					ProductCode: p1.Code,
					UnitPrice:   1000,
					Qty:         2,
					GrossTotal:  2000,
					NetTotal:    2000,
				},
			},
		},
		{
			ID:             "client-offline-uuid-2",
			SaleNo:         "SO-OFFLINE-0002",
			SubTotal:       1000,
			GrandTotal:     1000,
			TenderedAmount: 1000,
			ChangeAmount:   0,
			Status:         domain.SaleStatusVoid,
			VoidReason:     "ลูกค้าเปลี่ยนใจตอนออฟไลน์",
			CashierID:      "cashier",
			OccurredAt:     time.Now(),
			Lines: []domain.SaleItem{
				{
					ID:          "line-offline-2",
					ProductID:   p1.ID,
					ProductName: p1.Name,
					ProductCode: p1.Code,
					UnitPrice:   1000,
					Qty:         1,
					GrossTotal:  1000,
					NetTotal:    1000,
				},
			},
		},
	}

	syncPayload := domain.SyncBatchRequest{Sales: offlineSales}
	syncBytes, _ := json.Marshal(syncPayload)

	// 4. Push Batch to Server
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/sync/batch", bytes.NewReader(syncBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("push batch failed: status %d", resp.StatusCode)
	}

	var batchResp struct {
		Data domain.SyncBatchResponse `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&batchResp)
	if batchResp.Data.SyncedSales != 2 {
		t.Errorf("expected 2 synced sales, got %d", batchResp.Data.SyncedSales)
	}

	// 5. Test Idempotency: Push identical batch again
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/sync/batch", bytes.NewReader(syncBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("push batch idempotency retry failed: status %d", resp.StatusCode)
	}

	var batchRetryResp struct {
		Data domain.SyncBatchResponse `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&batchRetryResp)
	if batchRetryResp.Data.SkippedSales != 2 {
		t.Errorf("expected 2 skipped sales on retry, got %d", batchRetryResp.Data.SkippedSales)
	}

	// 6. Verify Daily Summary Report includes the synced sales correctly
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

	if reportResp.Data.TotalSales != 2000 {
		t.Errorf("expected 2000 satang total sales, got %d", reportResp.Data.TotalSales)
	}
	if reportResp.Data.BillCount != 1 {
		t.Errorf("expected 1 completed bill, got %d", reportResp.Data.BillCount)
	}
	if reportResp.Data.VoidCount != 1 {
		t.Errorf("expected 1 void count, got %d", reportResp.Data.VoidCount)
	}
}
