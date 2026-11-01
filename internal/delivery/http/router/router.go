package router

import (
	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/delivery/http/handler"
	"ice-pos-api-v2/internal/delivery/http/middleware"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/jwt"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

type RouterConfig struct {
	App            *fiber.App
	Config         *config.Config
	JWTService     *jwt.Service
	HealthHandler  *handler.HealthHandler
	AuthHandler    *handler.AuthHandler
	UserHandler    *handler.UserHandler
	CatalogHandler *handler.CatalogHandler
	SaleHandler    *handler.SaleHandler
	ReportHandler  *handler.ReportHandler
}

func SetupRoutes(rc *RouterConfig) {
	rc.App.Use(recover.New())
	rc.App.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	rc.App.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	}))

	v1 := rc.App.Group("/api/v1")

	// 1. Health
	v1.Get("/health", rc.HealthHandler.Check)

	// 2. Auth
	auth := v1.Group("/auth")
	{
		auth.Post("/login", rc.AuthHandler.Login)
		auth.Post("/pin-login", rc.AuthHandler.PinLogin)
		auth.Get("/me", middleware.AuthMiddleware(rc.JWTService), rc.AuthHandler.GetMe)
	}

	// 3. User Management (Admin Only)
	users := v1.Group("/users", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin))
	{
		users.Get("", rc.UserHandler.GetAllUsers)
		users.Post("", rc.UserHandler.CreateCashier)
		users.Post("/:id/reset-pin", rc.UserHandler.ResetPin)
	}

	// 4. Products & Categories (Public list for fast POS, protected write)
	products := v1.Group("/products")
	{
		products.Get("", rc.CatalogHandler.ListProducts)
		products.Get("/barcode/:barcode", rc.CatalogHandler.GetProductByBarcode)
		products.Get("/:id", rc.CatalogHandler.GetProductByID)
		products.Post("", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.CreateProduct)
	}

	categories := v1.Group("/categories")
	{
		categories.Get("", rc.CatalogHandler.ListCategories)
	}

	// 5. Sales & Checkout (Protected for authenticated cashiers / admins)
	sales := v1.Group("/sales", middleware.AuthMiddleware(rc.JWTService))
	{
		sales.Post("", rc.SaleHandler.CheckoutCash)
		sales.Get("", rc.SaleHandler.ListSales)
		sales.Get("/receipt/:receiptNo", rc.SaleHandler.GetReceipt)
		sales.Get("/:id", rc.SaleHandler.GetSaleByID)
		sales.Post("/:id/void", rc.SaleHandler.VoidSale)
	}

	// 6. Reports & Analytics
	reports := v1.Group("/reports", middleware.AuthMiddleware(rc.JWTService))
	{
		reports.Get("/sales-summary", rc.ReportHandler.GetDailySummary)
		reports.Get("/best-sellers", rc.ReportHandler.GetBestSellers)
	}
}
