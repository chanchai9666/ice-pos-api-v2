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
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/repository"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

func setupAuthTestApp(t *testing.T) (*fiber.App, func()) {
	dbFile := fmt.Sprintf("test_auth_%d.db", time.Now().UnixNano())

	cfg := &config.Config{
		AppName:       "ICE POS v2 Auth Test",
		AppEnv:        "development",
		JWTSecret:     "auth-test-secret-key-12345",
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
		SaleHandler:     handler.NewSaleHandler(saleUc),
		ReportHandler:   handler.NewReportHandler(reportUc),
	})

	cleanup := func() {
		_ = os.Remove(dbFile)
	}

	return app, cleanup
}

func TestE2ERefreshTokenFlow(t *testing.T) {
	app, cleanup := setupAuthTestApp(t)
	defer cleanup()

	// 1. Login with username & password
	loginPayload, _ := json.Marshal(map[string]string{
		"username": "admin",
		"password": "admin1234",
	})
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("login failed: status %d", resp.StatusCode)
	}

	var authResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&authResp)

	accessToken := authResp.Data.AccessToken
	refreshToken := authResp.Data.RefreshToken

	if accessToken == "" {
		t.Fatalf("expected non-empty access token")
	}
	if refreshToken == "" {
		t.Fatalf("expected non-empty refresh token")
	}

	// 2. Access protected endpoint /api/v1/auth/me with access token
	meReq, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+accessToken)
	meResp, err := app.Test(meReq)
	if err != nil || meResp.StatusCode != http.StatusOK {
		t.Fatalf("me request with access token failed: status %d", meResp.StatusCode)
	}

	// 3. Attempt to access protected endpoint using refresh token directly (must be rejected)
	badReq, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	badReq.Header.Set("Authorization", "Bearer "+refreshToken)
	badResp, err := app.Test(badReq)
	if err != nil || badResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when using refresh token on access endpoint, got %d", badResp.StatusCode)
	}

	// 4. Refresh token via JSON body
	refreshPayload, _ := json.Marshal(map[string]string{
		"refreshToken": refreshToken,
	})
	refreshReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshPayload))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshResp, err := app.Test(refreshReq)
	if err != nil || refreshResp.StatusCode != http.StatusOK {
		t.Fatalf("refresh token request failed: status %d", refreshResp.StatusCode)
	}

	var newTokensResp struct {
		Success bool `json:"success"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	_ = json.NewDecoder(refreshResp.Body).Decode(&newTokensResp)

	newAccessToken := newTokensResp.Data.AccessToken
	newRefreshToken := newTokensResp.Data.RefreshToken

	if newAccessToken == "" || newRefreshToken == "" {
		t.Fatalf("expected new access and refresh tokens, got empty")
	}

	// 5. Test replay attack: Re-using the already consumed refreshToken should fail
	reuseReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshPayload))
	reuseReq.Header.Set("Content-Type", "application/json")
	reuseResp, err := app.Test(reuseReq)
	if err != nil || reuseResp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 when reusing consumed refresh token, got %d", reuseResp.StatusCode)
	}

	// 6. Access protected endpoint with the new access token
	newMeReq, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	newMeReq.Header.Set("Authorization", "Bearer "+newAccessToken)
	newMeResp, err := app.Test(newMeReq)
	if err != nil || newMeResp.StatusCode != http.StatusOK {
		t.Fatalf("me request with new access token failed: status %d", newMeResp.StatusCode)
	}

	// 7. Logout test
	logoutReq, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutResp, err := app.Test(logoutReq)
	if err != nil || logoutResp.StatusCode != http.StatusOK {
		t.Fatalf("logout request failed: status %d", logoutResp.StatusCode)
	}
}
