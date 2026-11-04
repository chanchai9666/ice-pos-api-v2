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
	"ice-pos-api-v2/internal/repository"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func setupMultiImageTestApp(t *testing.T) (*fiber.App, string, string, string, func()) {
	dbFile := fmt.Sprintf("test_multi_image_%d.db", time.Now().UnixNano())
	tempUploadDir, _ := os.MkdirTemp("", "pos_multi_img_test_*")

	cfg := &config.Config{
		AppName:       "ICE POS v2 Multi Image Test",
		AppEnv:        "development",
		JWTSecret:     "test-secret-multi-img-12345",
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
	syncRepo := repository.NewSyncRepository(db)
	storageService := storage.NewStorageService(cfg)

	authUc := usecase.NewAuthUsecase(userRepo, jwtService)
	userUc := usecase.NewUserUsecase(userRepo)
	catUc := usecase.NewCatalogUsecase(catalogRepo)
	custUc := usecase.NewCustomerUsecase(customerRepo)
	saleUc := usecase.NewSaleUsecase(saleRepo, catalogRepo, customerRepo)
	reportUc := usecase.NewReportUsecase(reportRepo)
	syncUc := usecase.NewSyncUsecase(syncRepo)

	uploadHandler := handler.NewUploadHandler(cfg, catUc, storageService)
	catalogHandler := handler.NewCatalogHandler(catUc)
	authHandler := handler.NewAuthHandler(authUc, cfg)
	userHandler := handler.NewUserHandler(userUc)
	saleHandler := handler.NewSaleHandler(saleUc)
	reportHandler := handler.NewReportHandler(reportUc)
	syncHandler := handler.NewSyncHandler(syncUc)
	customerHandler := handler.NewCustomerHandler(custUc)
	healthHandler := handler.NewHealthHandler()

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
	})

	// Generate admin token
	adminToken, err := jwtService.GenerateAccessToken("admin-id", "admin", string(domain.RoleAdmin))
	if err != nil {
		t.Fatalf("failed to generate admin token: %v", err)
	}

	cleanup := func() {
		os.Remove(dbFile)
		os.RemoveAll(tempUploadDir)
	}

	return app, adminToken, tempUploadDir, dbFile, cleanup
}

func performMultipartRequest(app *fiber.App, method, url, token string, fileBytes []byte, filename string, fields map[string]string) (*http.Response, map[string]interface{}, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if len(fileBytes) > 0 {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, nil, err
		}
		if _, err := part.Write(fileBytes); err != nil {
			return nil, nil, err
		}
	}

	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	_ = writer.Close()

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req)
	if err != nil {
		return nil, nil, err
	}

	respBody, _ := io.ReadAll(resp.Body)
	var jsonMap map[string]interface{}
	_ = json.Unmarshal(respBody, &jsonMap)

	return resp, jsonMap, nil
}

func performJSONRequest(app *fiber.App, method, url, token string, payload interface{}) (*http.Response, map[string]interface{}, error) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := app.Test(req)
	if err != nil {
		return nil, nil, err
	}

	respBody, _ := io.ReadAll(resp.Body)
	var jsonMap map[string]interface{}
	_ = json.Unmarshal(respBody, &jsonMap)

	return resp, jsonMap, nil
}

func TestProductMultiImageLifecycle(t *testing.T) {
	app, token, uploadDir, _, cleanup := setupMultiImageTestApp(t)
	defer cleanup()

	// 1. Create a test product
	prodPayload := map[string]interface{}{
		"code":    "PROD-AI-01",
		"barcode": "8859990001",
		"name":    "เครื่องดื่มชูกำลังทดสอบ",
		"price":   1500,
	}
	createResp, createBody, err := performJSONRequest(app, "POST", "/api/v1/products", token, prodPayload)
	if err != nil || createResp.StatusCode != 201 {
		t.Fatalf("failed to create product: status=%d, body=%v, err=%v", createResp.StatusCode, createBody, err)
	}
	prodData := createBody["data"].(map[string]interface{})
	productID := prodData["id"].(string)

	pngBytes := createTestPNG(200, 200)

	// 2. Add Image 1 with Embedding
	img1Fields := map[string]string{
		"embedding":       "[0.123, -0.456, 0.789]",
		"embedding_model": "clip-vit-base-patch32",
	}
	resp1, body1, err := performMultipartRequest(app, "POST", fmt.Sprintf("/api/v1/products/%s/images", productID), token, pngBytes, "side1.png", img1Fields)
	if err != nil || resp1.StatusCode != 201 {
		t.Fatalf("failed to add image 1: status=%d, body=%v", resp1.StatusCode, body1)
	}
	img1Data := body1["data"].(map[string]interface{})
	imageID1 := img1Data["id"].(string)
	if imageID1 == "" {
		t.Fatalf("expected non-empty imageID1")
	}
	if img1Data["is_primary"] != true {
		t.Errorf("expected first image to be primary automatically, got %v", img1Data["is_primary"])
	}

	// Verify embedding returned in image 1
	emb1 := img1Data["embedding"].([]interface{})
	if len(emb1) != 3 {
		t.Fatalf("expected 3 embedding floats, got %d", len(emb1))
	}

	// 3. Add Image 2 (not primary)
	img2Fields := map[string]string{
		"embedding":       "[0.999, 0.888, 0.777]",
		"embedding_model": "clip-vit-base-patch32",
		"is_primary":      "false",
	}
	resp2, body2, err := performMultipartRequest(app, "POST", fmt.Sprintf("/api/v1/products/%s/images", productID), token, pngBytes, "side2.png", img2Fields)
	if err != nil || resp2.StatusCode != 201 {
		t.Fatalf("failed to add image 2: status=%d, body=%v", resp2.StatusCode, body2)
	}
	img2Data := body2["data"].(map[string]interface{})
	imageID2 := img2Data["id"].(string)
	if img2Data["is_primary"] == true {
		t.Errorf("expected image 2 to NOT be primary")
	}

	// 4. Verify Product GET includes images list and image_url is image 1
	getProdResp, getProdBody, err := performJSONRequest(app, "GET", fmt.Sprintf("/api/v1/products/%s", productID), "", nil)
	if err != nil || getProdResp.StatusCode != 200 {
		t.Fatalf("failed to get product: status=%d", getProdResp.StatusCode)
	}
	prodDetail := getProdBody["data"].(map[string]interface{})
	if prodDetail["image_url"] != img1Data["image_url"] {
		t.Errorf("expected product image_url to match image 1, got %v", prodDetail["image_url"])
	}
	imagesList := prodDetail["images"].([]interface{})
	if len(imagesList) != 2 {
		t.Fatalf("expected product to have 2 images, got %d", len(imagesList))
	}

	// 5. Test Switch Primary to Image 2
	primaryResp, primaryBody, err := performJSONRequest(app, "PATCH", fmt.Sprintf("/api/v1/products/%s/images/%s/primary", productID, imageID2), token, nil)
	if err != nil || primaryResp.StatusCode != 200 {
		t.Fatalf("failed to set primary: status=%d, body=%v", primaryResp.StatusCode, primaryBody)
	}

	// Verify image_url updated to image 2
	getProdResp2, getProdBody2, _ := performJSONRequest(app, "GET", fmt.Sprintf("/api/v1/products/%s", productID), "", nil)
	if getProdResp2.StatusCode != 200 {
		t.Fatalf("failed to get product after primary switch: %d", getProdResp2.StatusCode)
	}
	prodDetail2 := getProdBody2["data"].(map[string]interface{})
	if prodDetail2["image_url"] != img2Data["image_url"] {
		t.Errorf("expected product image_url to now match image 2, got %v", prodDetail2["image_url"])
	}

	// 6. Test Update Embedding Vector (Model Upgrade)
	newEmbedding := map[string]interface{}{
		"embedding":       []float32{0.5, 0.6, 0.7, 0.8},
		"embedding_model": "mobilenet-v3-large",
	}
	embResp, embBody, err := performJSONRequest(app, "PATCH", fmt.Sprintf("/api/v1/products/%s/images/%s/embedding", productID, imageID2), token, newEmbedding)
	if err != nil || embResp.StatusCode != 200 {
		t.Fatalf("failed to update embedding: status=%d, body=%v", embResp.StatusCode, embBody)
	}

	// Verify updated embedding via GET images
	listImgsResp, listImgsBody, _ := performJSONRequest(app, "GET", fmt.Sprintf("/api/v1/products/%s/images", productID), "", nil)
	if listImgsResp.StatusCode != 200 {
		t.Fatalf("failed to list product images: status=%d", listImgsResp.StatusCode)
	}
	imgs := listImgsBody["data"].([]interface{})
	primaryImg := imgs[0].(map[string]interface{}) // First is primary
	if primaryImg["id"] != imageID2 {
		t.Errorf("expected first listed image to be primary imageID2, got %v", primaryImg["id"])
	}
	updatedEmb := primaryImg["embedding"].([]interface{})
	if len(updatedEmb) != 4 {
		t.Fatalf("expected updated embedding to have 4 elements, got %d", len(updatedEmb))
	}
	if primaryImg["embedding_model"] != "mobilenet-v3-large" {
		t.Errorf("expected embedding_model to be mobilenet-v3-large, got %v", primaryImg["embedding_model"])
	}

	// 7. Verify Sync Changes endpoint preloads product images with embeddings
	syncResp, syncBody, _ := performJSONRequest(app, "GET", "/api/v1/sync/changes", token, nil)
	if syncResp.StatusCode != 200 {
		t.Fatalf("sync changes failed: status=%d", syncResp.StatusCode)
	}
	syncData := syncBody["data"].(map[string]interface{})
	syncProds := syncData["products"].([]interface{})
	var foundProd map[string]interface{}
	for _, p := range syncProds {
		pm := p.(map[string]interface{})
		if pm["id"] == productID {
			foundProd = pm
			break
		}
	}
	if foundProd == nil {
		t.Fatalf("synced products did not contain our test product")
	}
	syncImages := foundProd["images"].([]interface{})
	if len(syncImages) != 2 {
		t.Errorf("expected synced product to have 2 images, got %d", len(syncImages))
	}

	// 8. Delete primary image (image 2) and verify auto-promote of image 1
	img2File := filepath.Base(img2Data["image_url"].(string))
	delResp, delBody, err := performJSONRequest(app, "DELETE", fmt.Sprintf("/api/v1/products/%s/images/%s", productID, imageID2), token, nil)
	if err != nil || delResp.StatusCode != 200 {
		t.Fatalf("failed to delete image 2: status=%d, body=%v", delResp.StatusCode, delBody)
	}

	// Verify file was removed from storage
	diskFilePath := filepath.Join(uploadDir, "images", img2File)
	if _, err := os.Stat(diskFilePath); !os.IsNotExist(err) {
		t.Errorf("expected image file %s to be deleted from disk, but it still exists", diskFilePath)
	}

	// Verify image 1 was auto-promoted to primary
	getProdResp3, getProdBody3, _ := performJSONRequest(app, "GET", fmt.Sprintf("/api/v1/products/%s", productID), "", nil)
	if getProdResp3.StatusCode != 200 {
		t.Fatalf("failed to get product after deletion: %d", getProdResp3.StatusCode)
	}
	prodDetail3 := getProdBody3["data"].(map[string]interface{})
	if prodDetail3["image_url"] != img1Data["image_url"] {
		t.Errorf("expected product image_url to revert to auto-promoted image 1, got %v", prodDetail3["image_url"])
	}
	finalImages := prodDetail3["images"].([]interface{})
	if len(finalImages) != 1 {
		t.Errorf("expected exactly 1 remaining image, got %d", len(finalImages))
	}

	// 9. Verify Max 10 images quota limit
	for i := 2; i <= 10; i++ {
		_, _, _ = performMultipartRequest(app, "POST", fmt.Sprintf("/api/v1/products/%s/images", productID), token, pngBytes, fmt.Sprintf("side%d.png", i), nil)
	}
	// 11th image should fail
	resp11, body11, _ := performMultipartRequest(app, "POST", fmt.Sprintf("/api/v1/products/%s/images", productID), token, pngBytes, "side11.png", nil)
	if resp11.StatusCode != 400 {
		t.Errorf("expected status 400 for 11th image, got %d", resp11.StatusCode)
	}
	if errorMsg, ok := body11["message"].(string); !ok || errorMsg == "" {
		t.Errorf("expected error message for quota exceeded, got %v", body11)
	}
}

func TestProductImageValidationErrors(t *testing.T) {
	app, token, _, _, cleanup := setupMultiImageTestApp(t)
	defer cleanup()

	prodPayload := map[string]interface{}{
		"code":    "PROD-VAL-01",
		"barcode": "8859990002",
		"name":    "สินค้าทดสอบ Validation",
		"price":   1000,
	}
	_, createBody, _ := performJSONRequest(app, "POST", "/api/v1/products", token, prodPayload)
	productID := createBody["data"].(map[string]interface{})["id"].(string)

	pngBytes := createTestPNG(100, 100)

	// Invalid embedding format (string instead of JSON float array)
	badFields := map[string]string{
		"embedding": "not-a-json-array",
	}
	resp, body, _ := performMultipartRequest(app, "POST", fmt.Sprintf("/api/v1/products/%s/images", productID), token, pngBytes, "test.png", badFields)
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 Bad Request for invalid embedding, got %d: %v", resp.StatusCode, body)
	}

	// Empty embedding in PATCH embedding
	patchResp, _, _ := performJSONRequest(app, "PATCH", fmt.Sprintf("/api/v1/products/%s/images/non-existent-img/embedding", productID), token, map[string]interface{}{
		"embedding": []float32{},
	})
	if patchResp.StatusCode != 400 {
		t.Errorf("expected 400 Bad Request for empty embedding, got %d", patchResp.StatusCode)
	}
}
