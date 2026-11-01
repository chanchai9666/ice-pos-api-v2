package handler

import (
	"strconv"

	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type CatalogHandler struct {
	catalogUsecase domain.CatalogUsecase
}

func NewCatalogHandler(catalogUsecase domain.CatalogUsecase) *CatalogHandler {
	return &CatalogHandler{catalogUsecase: catalogUsecase}
}

func (h *CatalogHandler) ListProducts(c fiber.Ctx) error {
	search := c.Query("search")
	categoryID := c.Query("category_id")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	products, total, err := h.catalogUsecase.SearchProducts(c.Context(), search, categoryID, limit, offset)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    products,
		"total":   total,
	})
}

func (h *CatalogHandler) GetProductByID(c fiber.Ctx) error {
	id := c.Params("id")
	prod, err := h.catalogUsecase.GetProductByID(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    prod,
	})
}

func (h *CatalogHandler) GetProductByBarcode(c fiber.Ctx) error {
	barcode := c.Params("barcode")
	prod, err := h.catalogUsecase.GetProductByBarcode(c.Context(), barcode)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    prod,
	})
}

func (h *CatalogHandler) CreateProduct(c fiber.Ctx) error {
	var prod domain.Product
	if err := c.Bind().JSON(&prod); err != nil {
		return domain.ErrBadRequest
	}

	if err := h.catalogUsecase.CreateProduct(c.Context(), &prod); err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    prod,
	})
}

func (h *CatalogHandler) ListCategories(c fiber.Ctx) error {
	cats, err := h.catalogUsecase.ListCategories(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    cats,
	})
}
