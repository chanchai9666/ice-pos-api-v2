package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/delivery/http/handler"
	"ice-pos-api-v2/internal/delivery/http/middleware"
	"ice-pos-api-v2/internal/delivery/http/router"
	"ice-pos-api-v2/internal/pkg/imageutil"
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/pkg/storage"
	"ice-pos-api-v2/internal/repository"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func createComplexTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 7) ^ (y * 13)),
				G: uint8((x * 11) + (y * 5)),
				B: uint8((x + y) * 3),
				A: 255,
			})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func setupUploadTestApp(t *testing.T) (*fiber.App, string, string, func()) {
	dbFile := fmt.Sprintf("test_upload_%d.db", time.Now().UnixNano())
	tempUploadDir, _ := os.MkdirTemp("", "pos_upload_test_*")

	cfg := &config.Config{
		AppName:       "ICE POS v2 Upload Test",
		AppEnv:        "development",
		JWTSecret:     "upload-test-secret-key-123",
		JWTAccessTTL:  time.Hour,
		JWTRefreshTTL: 24 * time.Hour,
		DBType:        "sqlite",
		DBPath:        dbFile,
		UploadDir:     tempUploadDir,
		UploadBaseURL: "/uploads",
		StorageDriver: "local",
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
	storageService := storage.NewStorageService(cfg)

	authUc := usecase.NewAuthUsecase(userRepo, jwtService)
	userUc := usecase.NewUserUsecase(userRepo)
	catUc := usecase.NewCatalogUsecase(catalogRepo)
	custUc := usecase.NewCustomerUsecase(customerRepo)
	saleUc := usecase.NewSaleUsecase(saleRepo, catalogRepo, customerRepo)
	reportUc := usecase.NewReportUsecase(reportRepo)

	uploadHandler := handler.NewUploadHandler(cfg, catUc, storageService)

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
		CustomerHandler: handler.NewCustomerHandler(custUc),
		UploadHandler:   uploadHandler,
		SaleHandler:     handler.NewSaleHandler(saleUc),
		ReportHandler:   handler.NewReportHandler(reportUc),
	})

	token, _ := jwtService.GenerateAccessToken("admin-id", "admin", "admin")

	cleanup := func() {
		_ = os.Remove(dbFile)
		_ = os.RemoveAll(tempUploadDir)
	}

	return app, token, tempUploadDir, cleanup
}

func TestImageUtil_ResizeAndConvertToWebP(t *testing.T) {
	// Create large image (1600x1200)
	rawPNG := createTestPNG(1600, 1200)

	processed, err := imageutil.ProcessImage(bytes.NewReader(rawPNG), ".png")
	if err != nil {
		t.Fatalf("process image failed: %v", err)
	}

	if processed.Extension != ".webp" {
		t.Errorf("expected extension .webp, got %s", processed.Extension)
	}
	if processed.ContentType != "image/webp" {
		t.Errorf("expected contentType image/webp, got %s", processed.ContentType)
	}
	// MaxWidth is 1024, so width should be 1024 and height proportional (768)
	if processed.Width != 1024 {
		t.Errorf("expected resized width 1024, got %d", processed.Width)
	}
	if processed.Height != 768 {
		t.Errorf("expected resized height 768, got %d", processed.Height)
	}
	if len(processed.Data) == 0 {
		t.Fatalf("processed data is empty")
	}
}

func TestImageUtil_MaxFileSizeConstraint(t *testing.T) {
	// 1. Create a complex high-res PNG (1200x1200)
	complexPNG := createComplexTestPNG(1200, 1200)

	// Process image with default options (which sets MaxFileSize = 50KB)
	processed, err := imageutil.ProcessImage(bytes.NewReader(complexPNG), ".png")
	if err != nil {
		t.Fatalf("process complex image failed: %v", err)
	}

	maxAllowed := 50 * 1024 // 50KB
	if len(processed.Data) > maxAllowed {
		t.Errorf("expected processed image size <= %d bytes, got %d bytes", maxAllowed, len(processed.Data))
	}
	if processed.Extension != ".webp" {
		t.Errorf("expected extension .webp, got %s", processed.Extension)
	}
}

func TestE2EImageUploadAndProductImageBinding(t *testing.T) {
	app, token, uploadDir, cleanup := setupUploadTestApp(t)
	defer cleanup()

	pngBytes := createTestPNG(500, 500)

	// 1. Test POST /api/v1/uploads/images
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test_item.png")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write(pngBytes)
	_ = writer.Close()

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/uploads/images", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("image upload failed: status %d", resp.StatusCode)
	}

	var uploadResp struct {
		Success bool `json:"success"`
		Data    struct {
			URL         string `json:"url"`
			Filename    string `json:"file_name"`
			ContentType string `json:"content_type"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&uploadResp)

	if uploadResp.Data.URL == "" {
		t.Fatalf("expected non-empty image url")
	}
	if uploadResp.Data.ContentType != "image/webp" {
		t.Fatalf("expected contentType image/webp, got %s", uploadResp.Data.ContentType)
	}

	// 2. Test POST /api/v1/products/:id/image directly binding to product
	// Fetch products to get seeded product ID
	listReq, _ := http.NewRequest(http.MethodGet, "/api/v1/products", nil)
	listResp, err := app.Test(listReq)
	if err != nil || listResp.StatusCode != http.StatusOK {
		t.Fatalf("list products failed: status %d", listResp.StatusCode)
	}
	var prodList struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.NewDecoder(listResp.Body).Decode(&prodList)
	if len(prodList.Data) == 0 {
		t.Fatalf("no seeded products found")
	}
	prodID := prodList.Data[0].ID

	// Upload image for this product
	prodBody := &bytes.Buffer{}
	prodWriter := multipart.NewWriter(prodBody)
	prodPart, _ := prodWriter.CreateFormFile("file", "prod_photo.png")
	_, _ = prodPart.Write(pngBytes)
	_ = prodWriter.Close()

	prodImgReq, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/products/%s/image", prodID), prodBody)
	prodImgReq.Header.Set("Content-Type", prodWriter.FormDataContentType())
	prodImgReq.Header.Set("Authorization", "Bearer "+token)

	prodImgResp, err := app.Test(prodImgReq)
	if err != nil || prodImgResp.StatusCode != http.StatusOK {
		t.Fatalf("upload product image failed: status %d", prodImgResp.StatusCode)
	}

	var updatedProdResp struct {
		Success bool `json:"success"`
		Data    struct {
			ID       string `json:"id"`
			ImageURL string `json:"image_url"`
		} `json:"data"`
	}
	_ = json.NewDecoder(prodImgResp.Body).Decode(&updatedProdResp)

	if updatedProdResp.Data.ImageURL == "" {
		t.Fatalf("expected updated product to have image_url")
	}

	_ = uploadDir
}
