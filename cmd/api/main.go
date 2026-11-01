package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/delivery/http/handler"
	"ice-pos-api-v2/internal/delivery/http/middleware"
	"ice-pos-api-v2/internal/delivery/http/router"
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/pkg/logger"
	"ice-pos-api-v2/internal/repository"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

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

	// 6. Usecases
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtService)
	userUsecase := usecase.NewUserUsecase(userRepo)
	catalogUsecase := usecase.NewCatalogUsecase(catalogRepo)
	saleUsecase := usecase.NewSaleUsecase(saleRepo, catalogRepo)
	reportUsecase := usecase.NewReportUsecase(reportRepo)

	// 7. Handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authUsecase)
	userHandler := handler.NewUserHandler(userUsecase)
	catalogHandler := handler.NewCatalogHandler(catalogUsecase)
	saleHandler := handler.NewSaleHandler(saleUsecase)
	reportHandler := handler.NewReportHandler(reportUsecase)

	// 8. Fiber v3 App
	app := fiber.New(fiber.Config{
		AppName:      cfg.AppName,
		ErrorHandler: middleware.CustomErrorHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	})

	// 9. Routes
	router.SetupRoutes(&router.RouterConfig{
		App:            app,
		Config:         cfg,
		JWTService:     jwtService,
		HealthHandler:  healthHandler,
		AuthHandler:    authHandler,
		UserHandler:    userHandler,
		CatalogHandler: catalogHandler,
		SaleHandler:    saleHandler,
		ReportHandler:  reportHandler,
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
	slog.Info("Server is listening", "address", addr)
	if err := app.Listen(addr); err != nil {
		slog.Error("Server terminated", "error", err)
	}
}
