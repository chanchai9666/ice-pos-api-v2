package router

import (
	"ice-pos-api-v2/config"
	_ "ice-pos-api-v2/docs/swagger"
	"ice-pos-api-v2/internal/delivery/http/handler"
	"ice-pos-api-v2/internal/delivery/http/middleware"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/jwt"

	swaggo "github.com/gofiber/contrib/v3/swaggo"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/static"
)

type RouterConfig struct {
	App             *fiber.App
	Config          *config.Config
	JWTService      *jwt.Service
	HealthHandler   *handler.HealthHandler
	AuthHandler     *handler.AuthHandler
	UserHandler     *handler.UserHandler
	CatalogHandler  *handler.CatalogHandler
	SaleHandler     *handler.SaleHandler
	ReportHandler   *handler.ReportHandler
	SyncHandler     *handler.SyncHandler
	CustomerHandler *handler.CustomerHandler
	UploadHandler   *handler.UploadHandler
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

	// Static files serving for uploaded assets (e.g. /uploads/images/...)
	if rc.Config != nil && rc.Config.UploadDir != "" {
		rc.App.Use("/uploads", static.New(rc.Config.UploadDir, static.Config{
			Browse: false,
		}))
	}

	// Swagger API Docs
	rc.App.Get("/swagger", func(c fiber.Ctx) error {
		return c.Redirect().To("/swagger/index.html")
	})
	rc.App.Get("/swagger/*", swaggo.HandlerDefault)

	v1 := rc.App.Group("/api/v1")

	// 1. Health
	v1.Get("/health", rc.HealthHandler.Check)

	// 2. Auth
	auth := v1.Group("/auth")
	{
		auth.Post("/login", rc.AuthHandler.Login)
		auth.Post("/pin-login", rc.AuthHandler.PinLogin)
		auth.Post("/refresh", rc.AuthHandler.RefreshToken)
		auth.Post("/logout", rc.AuthHandler.Logout)
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
		products.Put("/:id", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.UpdateProduct)
		if rc.UploadHandler != nil {
			products.Post("/:id/image", middleware.AuthMiddleware(rc.JWTService), rc.UploadHandler.UploadProductImage)
		}
	}

	categories := v1.Group("/categories")
	{
		categories.Get("", rc.CatalogHandler.ListCategories)
		categories.Get("/:id", rc.CatalogHandler.GetCategoryByID)
		categories.Post("", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.CreateCategory)
		categories.Put("/:id", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.UpdateCategory)
		categories.Delete("/:id", middleware.AuthMiddleware(rc.JWTService), middleware.RequireRoles(domain.RoleAdmin), rc.CatalogHandler.DeleteCategory)
	}

	// 5. Image Uploads
	if rc.UploadHandler != nil {
		uploads := v1.Group("/uploads", middleware.AuthMiddleware(rc.JWTService))
		{
			uploads.Post("/images", rc.UploadHandler.UploadImage)
		}
	}

	// 6. Customers & Credit Accounts
	if rc.CustomerHandler != nil {
		customers := v1.Group("/customers", middleware.AuthMiddleware(rc.JWTService))
		{
			customers.Post("", rc.CustomerHandler.CreateCustomer)
			customers.Get("", rc.CustomerHandler.GetAllCustomers)
			customers.Get("/reports/ar-aging", rc.CustomerHandler.GetARAgingReport)
			customers.Get("/:id", rc.CustomerHandler.GetCustomerByID)
			customers.Put("/:id", rc.CustomerHandler.UpdateCustomer)
			customers.Delete("/:id", rc.CustomerHandler.DeleteCustomer)
			customers.Put("/:id/credit-limit", rc.CustomerHandler.SetCreditLimit)
			customers.Post("/:id/pay-credit", rc.CustomerHandler.PayCredit)
			customers.Get("/:id/credit-history", rc.CustomerHandler.GetCreditHistory)
		}
	}

	// 7. Sales & Checkout (Protected for authenticated cashiers / admins)
	sales := v1.Group("/sales", middleware.AuthMiddleware(rc.JWTService))
	{
		sales.Post("", rc.SaleHandler.Checkout)
		sales.Get("", rc.SaleHandler.ListSales)
		sales.Get("/receipt/:receiptNo", rc.SaleHandler.GetReceipt)
		sales.Get("/:id", rc.SaleHandler.GetSaleByID)
		sales.Post("/:id/void", rc.SaleHandler.VoidSale)
	}

	// 8. Reports & Analytics
	reports := v1.Group("/reports", middleware.AuthMiddleware(rc.JWTService))
	{
		reports.Get("/sales-summary", rc.ReportHandler.GetDailySummary)
		reports.Get("/best-sellers", rc.ReportHandler.GetBestSellers)
	}

	// 9. Offline Synchronization
	if rc.SyncHandler != nil {
		sync := v1.Group("/sync", middleware.AuthMiddleware(rc.JWTService))
		{
			sync.Get("/changes", rc.SyncHandler.PullChanges)
			sync.Post("/batch", rc.SyncHandler.PushBatch)
		}
	}
}
