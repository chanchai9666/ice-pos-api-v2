package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ice-pos-api-v2/config"
	_ "ice-pos-api-v2/docs/swagger"
	"ice-pos-api-v2/internal/delivery/http/handler"
	"ice-pos-api-v2/internal/delivery/http/middleware"
	"ice-pos-api-v2/internal/delivery/http/router"
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/pkg/logger"
	"ice-pos-api-v2/internal/pkg/storage"
	"ice-pos-api-v2/internal/pkg/vision"
	"ice-pos-api-v2/internal/repository"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

// @title                      ICE POS API v2
// @version                    2.0
// @description                ระบบ REST API สำหรับระบบขายหน้าร้าน (POS) ประสิทธิภาพสูง พร้อมรองรับการทำงานออฟไลน์ (Offline-first Synchronization)
// @termsOfService             http://swagger.io/terms/

// @contact.name               ทีมพัฒนา ICE POS
// @contact.email              support@ice-pos.local

// @license.name               MIT
// @license.url                https://opensource.org/licenses/MIT

// @host                       localhost:8080
// @BasePath                   /api/v1
// @schemes                    http https

// @securityDefinitions.apikey BearerAuth
// @in                         header
// @name                       Authorization
// @description                ใส่ "Bearer " ตามด้วย JWT Token (ตัวอย่าง: 'Bearer eyJhbGci...')

// @tag.name                   Auth
// @tag.description            ระบบจัดการสิทธิ์และการเข้าสู่ระบบ

// @tag.name                   Products
// @tag.description            ระบบจัดการข้อมูลสินค้าและแคตตาล็อก

// @tag.name                   Categories
// @tag.description            ระบบจัดการหมวดหมู่สินค้า

// @tag.name                   Sales
// @tag.description            ระบบบันทึกการขาย ชำระเงิน และออกใบเสร็จรับเงิน

// @tag.name                   Reports
// @tag.description            ระบบรายงานยอดขายและสถิติสินค้าขายดี

// @tag.name                   Sync
// @tag.description            ระบบซิงค์ข้อมูลสำหรับรองรับการทำงานออฟไลน์ (Offline-first Synchronization)

// @tag.name                   Users
// @tag.description            ระบบจัดการบัญชีผู้ใช้งานและพนักงานแคชเชียร์

// @tag.name                   Health
// @tag.description            ตรวจสอบสถานะความพร้อมและการทำงานของระบบ API
func main() {
	// 1. Config
	cfg := config.LoadConfig()

	// 2. Logger
	logger.InitLogger(cfg.AppEnv)
	slog.Info("Starting ICE POS API v2 Server...", "port", cfg.AppPort, "env", cfg.AppEnv)

	// 3. Database
	db, err := repository.NewDatabase(cfg)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}

	// 4. JWT Service
	jwtService := jwt.NewService(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)

	// 5. Repositories
	userRepo := repository.NewUserRepository(db)
	catalogRepo := repository.NewCatalogRepository(db)
	saleRepo := repository.NewSaleRepository(db)
	reportRepo := repository.NewReportRepository(db)
	syncRepo := repository.NewSyncRepository(db)
	customerRepo := repository.NewCustomerRepository(db)

	// 6. Services & Usecases
	storageService := storage.NewStorageService(cfg)
	visionService := vision.NewClient(cfg.AIEmbeddingServiceURL, 10*time.Second)
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtService)
	userUsecase := usecase.NewUserUsecase(userRepo)
	catalogUsecase := usecase.NewCatalogUsecase(catalogRepo)
	customerUsecase := usecase.NewCustomerUsecase(customerRepo)
	saleUsecase := usecase.NewSaleUsecase(saleRepo, catalogRepo, customerRepo)
	reportUsecase := usecase.NewReportUsecase(reportRepo)
	syncUsecase := usecase.NewSyncUsecase(syncRepo)
	visionUsecase := usecase.NewVisionUsecase(catalogRepo, visionService)

	// 7. Handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authUsecase, cfg)
	userHandler := handler.NewUserHandler(userUsecase)
	catalogHandler := handler.NewCatalogHandler(catalogUsecase)
	customerHandler := handler.NewCustomerHandler(customerUsecase)
	uploadHandler := handler.NewUploadHandler(cfg, catalogUsecase, storageService, visionService)
	saleHandler := handler.NewSaleHandler(saleUsecase)
	reportHandler := handler.NewReportHandler(reportUsecase)
	syncHandler := handler.NewSyncHandler(syncUsecase)
	visionHandler := handler.NewVisionHandler(visionUsecase)

	// 8. Fiber v3 App
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: middleware.CustomErrorHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	})

	// 9. Routes
	router.SetupRoutes(&router.RouterConfig{
		App:             app,
		Config:          cfg,
		JWTService:      jwtService,
		HealthHandler:   healthHandler,
		AuthHandler:     authHandler,
		UserHandler:     userHandler,
		CatalogHandler:  catalogHandler,
		CustomerHandler: customerHandler,
		UploadHandler:   uploadHandler,
		SaleHandler:     saleHandler,
		ReportHandler:   reportHandler,
		SyncHandler:     syncHandler,
		VisionHandler:   visionHandler,
	})

	// 10. Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Gracefully shutting down server...")
		if err := app.Shutdown(); err != nil {
			slog.Error("Error during shutdown", "error", err)
		}
	}()

	// 11. Listen
	addr := fmt.Sprintf(":%s", cfg.AppPort)
	slog.Info("Server is listening", "address", addr, "swagger", fmt.Sprintf("http://localhost:%s/swagger/index.html", cfg.AppPort))
	if err := app.Listen(addr, fiber.ListenConfig{DisableStartupMessage: false}); err != nil {
		slog.Error("Server error", "error", err)
	}
}
