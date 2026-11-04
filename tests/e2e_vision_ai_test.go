package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/delivery/http/handler"
	"ice-pos-api-v2/internal/delivery/http/middleware"
	"ice-pos-api-v2/internal/delivery/http/router"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/pkg/storage"
	"ice-pos-api-v2/internal/pkg/vision"
	"ice-pos-api-v2/internal/repository"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func setupVisionAITestApp(t *testing.T) (*fiber.App, string, string, *httptest.Server, func()) {
	// Mock Vision AI HTTP service
	mockVector := domain.FloatVector{0.5, 0.5, 0.5}
	mockModel := "Xenova/clip-vit-base-patch32"

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"embedding": mockVector,
			"model":     mockModel,
		})
	}))

	dbFile := fmt.Sprintf("test_vision_%d.db", time.Now().UnixNano())
	tempUploadDir, _ := os.MkdirTemp("", "pos_vision_test_*")

	cfg := &config.Config{
		AppName:               "ICE POS v2 Vision Test",
		AppEnv:                "development",
		JWTSecret:             "test-secret-vision-12345",
		JWTAccessTTL:          time.Hour,
		JWTRefreshTTL:         24 * time.Hour,
		DBType:                "sqlite",
		DBPath:                dbFile,
		UploadDir:             tempUploadDir,
		UploadBaseURL:         "/uploads",
		StorageDriver:         "local",
		AIEmbeddingServiceURL: mockServer.URL,
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
	syncRepo := repository.NewSyncRepository(db)
	storageService := storage.NewStorageService(cfg)
	visionService := vision.NewClient(cfg.AIEmbeddingServiceURL, 5*time.Second)

	authUc := usecase.NewAuthUsecase(userRepo, jwtService)
	userUc := usecase.NewUserUsecase(userRepo)
	catUc := usecase.NewCatalogUsecase(catalogRepo)
	custUc := usecase.NewCustomerUsecase(customerRepo)
	saleUc := usecase.NewSaleUsecase(saleRepo, catalogRepo, customerRepo)
	reportUc := usecase.NewReportUsecase(reportRepo)
	syncUc := usecase.NewSyncUsecase(syncRepo)
	visionUc := usecase.NewVisionUsecase(catalogRepo, visionService)

	uploadHandler := handler.NewUploadHandler(cfg, catUc, storageService, visionService)
	catalogHandler := handler.NewCatalogHandler(catUc)
	authHandler := handler.NewAuthHandler(authUc, cfg)
	userHandler := handler.NewUserHandler(userUc)
	saleHandler := handler.NewSaleHandler(saleUc)
	reportHandler := handler.NewReportHandler(reportUc)
	syncHandler := handler.NewSyncHandler(syncUc)
	customerHandler := handler.NewCustomerHandler(custUc)
	healthHandler := handler.NewHealthHandler()
	visionHandler := handler.NewVisionHandler(visionUc)

	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.CustomErrorHandler,
	})

	router.SetupRoutes(&router.RouterConfig{
		App:             app,
		Config:          cfg,
		JWTService:      jwtService,
		HealthHandler:   healthHandler,
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		CatalogHandler:  catalogHandler,
		SaleHandler:     saleHandler,
		ReportHandler:   reportHandler,
		SyncHandler:     syncHandler,
		CustomerHandler: customerHandler,
		UploadHandler:   uploadHandler,
		VisionHandler:   visionHandler,
	})

	adminToken, err := jwtService.GenerateAccessToken("admin-id", "admin", string(domain.RoleAdmin))
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	prod := &domain.Product{
		ID:      "test-prod-vision-uuid",
		Code:    "PROD-VISION-001",
		Barcode: "8859999001",
		Name:    "Ice Vision Cup",
		Price:   3500,
	}
	err = catUc.CreateProduct(t.Context(), prod)
	if err != nil {
		t.Fatalf("failed to create sample product: %v", err)
	}

	cleanup := func() {
		mockServer.Close()
		_ = os.Remove(dbFile)
		_ = os.RemoveAll(tempUploadDir)
	}

	return app, adminToken, prod.ID, mockServer, cleanup
}

func createTestPNGImage() []byte {
	// Minimal valid 1x1 PNG image
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
		0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}

func TestE2E_BackendAutoExtractEmbeddingOnUpload(t *testing.T) {
	app, token, productID, _, cleanup := setupVisionAITestApp(t)
	defer cleanup()

	// Upload image WITHOUT providing embedding in form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "product_photo.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(createTestPNGImage())
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/products/%s/images", productID), body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status 201 Created, got %d: %s", resp.StatusCode, string(respBody))
	}

	var res struct {
		Success bool                `json:"success"`
		Data    domain.ProductImage `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)

	// Verify that the backend automatically generated and stored the embedding
	if len(res.Data.Embedding) != 3 {
		t.Fatalf("expected auto-extracted embedding of len 3, got %d", len(res.Data.Embedding))
	}
	if res.Data.EmbeddingModel != "Xenova/clip-vit-base-patch32" {
		t.Errorf("expected model Xenova/clip-vit-base-patch32, got %s", res.Data.EmbeddingModel)
	}
}

func TestE2E_ServerSideVisionMatch(t *testing.T) {
	app, token, productID, _, cleanup := setupVisionAITestApp(t)
	defer cleanup()

	// 1. First upload an image to product
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("file", "photo1.png")
	_, _ = part.Write(createTestPNGImage())
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/products/%s/images", productID), body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, _ := app.Test(req)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("upload failed with status %d", resp.StatusCode)
	}

	// 2. Query /api/v1/vision/match with a frame
	matchBody := &bytes.Buffer{}
	matchWriter := multipart.NewWriter(matchBody)
	mPart, _ := matchWriter.CreateFormFile("file", "frame.png")
	_, _ = mPart.Write(createTestPNGImage())
	_ = matchWriter.WriteField("threshold", "0.7")
	_ = matchWriter.Close()

	mReq, _ := http.NewRequest(http.MethodPost, "/api/v1/vision/match", matchBody)
	mReq.Header.Set("Authorization", "Bearer "+token)
	mReq.Header.Set("Content-Type", matchWriter.FormDataContentType())

	mResp, err := app.Test(mReq)
	if err != nil {
		t.Fatalf("match request failed: %v", err)
	}

	if mResp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(mResp.Body)
		t.Fatalf("expected status 200 OK, got %d: %s", mResp.StatusCode, string(respBody))
	}

	var matchRes struct {
		Success bool                       `json:"success"`
		Data    domain.VisionMatchResponse `json:"data"`
	}
	_ = json.NewDecoder(mResp.Body).Decode(&matchRes)

	if len(matchRes.Data.Matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matchRes.Data.Matches))
	}

	matchedItem := matchRes.Data.Matches[0]
	if matchedItem.ProductID != productID {
		t.Errorf("expected product ID %s, got %s", productID, matchedItem.ProductID)
	}
	if matchedItem.Confidence <= 0.8 {
		t.Errorf("expected high confidence, got %f", matchedItem.Confidence)
	}
}

func TestE2E_ServerSideVisionReindex(t *testing.T) {
	app, token, productID, _, cleanup := setupVisionAITestApp(t)
	defer cleanup()

	// Upload image manually without embedding by mocking local upload
	// Then call /api/v1/vision/reindex
	reindexReq, _ := http.NewRequest(http.MethodPost, "/api/v1/vision/reindex", nil)
	reindexReq.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(reindexReq)
	if err != nil {
		t.Fatalf("reindex request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200 OK, got %d: %s", resp.StatusCode, string(respBody))
	}
	_ = productID
	_ = filepath.Join
}
