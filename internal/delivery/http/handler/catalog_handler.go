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

// ListProducts godoc
// @Summary      รายการสินค้าทั้งหมด
// @Description  ค้นหาและแสดงรายการสินค้าในแคตตาล็อก พร้อมตัวกรองตามหมวดหมู่และการแบ่งหน้า (Pagination)
// @Tags         Products
// @Produce      json
// @Param        search      query string false "ค้นหาตามชื่อสินค้าหรือรหัสสินค้า/บาร์โค้ด"
// @Param        category_id query string false "กรองตามรหัสหมวดหมู่ (Category ID)"
// @Param        limit       query int    false "จำนวนรายการสูงสุดต่อหน้า (ค่าเริ่มต้น 50)"
// @Param        offset      query int    false "ตำแหน่งเริ่มต้น (ค่าเริ่มต้น 0)"
// @Success      200 {object} ProductListDataResponse
// @Failure      500 {object} ErrorResponse
// @Router       /products [get]
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

// GetProductByID godoc
// @Summary      ดูข้อมูลสินค้าตาม ID
// @Description  ดึงรายละเอียดข้อมูลสินค้าด้วยรหัส UUID ของสินค้า
// @Tags         Products
// @Produce      json
// @Param        id  path string true "รหัสสินค้า (Product ID / UUID)"
// @Success      200 {object} ProductDataResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id} [get]
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

// GetProductByBarcode godoc
// @Summary      ค้นหาสินค้าด้วยบาร์โค้ด
// @Description  ค้นหาข้อมูลสินค้าอย่างรวดเร็วด้วยรหัสบาร์โค้ด สำหรับการยิงสแกนขายหน้าร้าน
// @Tags         Products
// @Produce      json
// @Param        barcode path string true "รหัสบาร์โค้ดสินค้า (Barcode)"
// @Success      200 {object} ProductDataResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/barcode/{barcode} [get]
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

// CreateProduct godoc
// @Summary      สร้างสินค้าใหม่
// @Description  เพิ่มสินค้าใหม่เข้าสู่ระบบแคตตาล็อก (เฉพาะสิทธิ์ Admin)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.Product true "ข้อมูลสินค้าที่ต้องการสร้าง"
// @Success      201 {object} ProductDataResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /products [post]
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

// ListCategories godoc
// @Summary      รายการหมวดหมู่สินค้าทั้งหมด
// @Description  ดึงรายการหมวดหมู่สินค้าทั้งหมดที่มีในระบบ
// @Tags         Categories
// @Produce      json
// @Success      200 {object} CategoryListDataResponse
// @Failure      500 {object} ErrorResponse
// @Router       /categories [get]
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
