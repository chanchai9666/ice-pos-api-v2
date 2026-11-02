package handler

import (
	"net/url"
	"strconv"
	"strings"

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
// @Description  ค้นหาและแสดงรายการสินค้าในแคตตาล็อก พร้อมตัวกรองตามหมวดหมู่และการแบ่งหน้า (Pagination) หรือค้นหาตรงตามบาร์โค้ด
// @Tags         Products
// @Produce      json
// @Param        search      query string false "ค้นหาตามชื่อสินค้าหรือรหัสสินค้า/บาร์โค้ด"
// @Param        barcode     query string false "ค้นหาตรงตามรหัสบาร์โค้ด (Exact Match)"
// @Param        category_id query string false "กรองตามรหัสหมวดหมู่ (Category ID)"
// @Param        limit       query int    false "จำนวนรายการสูงสุดต่อหน้า (ค่าเริ่มต้น 50)"
// @Param        offset      query int    false "ตำแหน่งเริ่มต้น (ค่าเริ่มต้น 0)"
// @Success      200 {object} ProductListDataResponse
// @Failure      500 {object} ErrorResponse
// @Router       /products [get]
func (h *CatalogHandler) ListProducts(c fiber.Ctx) error {
	search := c.Query("search")
	barcode := c.Query("barcode")
	categoryID := c.Query("category_id")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	products, total, err := h.catalogUsecase.SearchProducts(c.Context(), search, categoryID, barcode, limit, offset)
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
// @Description  ค้นหาข้อมูลสินค้าอย่างรวดเร็วด้วยรหัสบาร์โค้ด สำหรับการสแกนด้วยกล้องหรือบาร์โค้ดสแกนเนอร์
// @Tags         Products
// @Produce      json
// @Param        barcode path string true "รหัสบาร์โค้ดสินค้า (Barcode)"
// @Success      200 {object} ProductDataResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/barcode/{barcode} [get]
func (h *CatalogHandler) GetProductByBarcode(c fiber.Ctx) error {
	rawBarcode := c.Params("barcode")
	decoded, err := url.PathUnescape(rawBarcode)
	if err != nil {
		decoded = rawBarcode
	}
	barcode := strings.TrimSpace(decoded)
	if barcode == "" {
		return domain.ErrBadRequest
	}
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

// UpdateProduct godoc
// @Summary      แก้ไขข้อมูลสินค้า
// @Description  อัปเดตข้อมูลสินค้า เช่น ชื่อ ราคา หรือรหัสบาร์โค้ด (เฉพาะ Admin)
// @Tags         Products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path string true "รหัสสินค้า (Product ID / UUID)"
// @Param        request body domain.Product true "ข้อมูลสินค้าที่ต้องการแก้ไข"
// @Success      200 {object} ProductDataResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id} [put]
func (h *CatalogHandler) UpdateProduct(c fiber.Ctx) error {
	id := c.Params("id")
	var prod domain.Product
	if err := c.Bind().JSON(&prod); err != nil {
		return domain.ErrBadRequest
	}
	prod.ID = id

	if err := h.catalogUsecase.UpdateProduct(c.Context(), &prod); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
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

// GetCategoryByID godoc
// @Summary      ดึงข้อมูลหมวดหมู่สินค้าตาม ID
// @Description  ดึงรายละเอียดข้อมูลหมวดหมู่สินค้าตามรหัส ID
// @Tags         Categories
// @Produce      json
// @Param        id   path      string true "รหัสหมวดหมู่ (Category ID / UUID)"
// @Success      200  {object}  CategoryDataResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /categories/{id} [get]
func (h *CatalogHandler) GetCategoryByID(c fiber.Ctx) error {
	id := c.Params("id")
	cat, err := h.catalogUsecase.GetCategoryByID(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    cat,
	})
}

// CreateCategoryRequest represents request payload for creating category
type CreateCategoryRequest struct {
	Name      string `json:"name" example:"เครื่องดื่ม" validate:"required"`
	SortOrder int    `json:"sort_order" example:"1"`
}

// CreateCategory godoc
// @Summary      สร้างหมวดหมู่สินค้าใหม่
// @Description  สร้างหมวดหมู่สินค้าใหม่ในระบบ (เฉพาะ Admin)
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateCategoryRequest true "ข้อมูลหมวดหมู่ที่ต้องการสร้าง"
// @Success      201 {object} CategoryActionResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /categories [post]
func (h *CatalogHandler) CreateCategory(c fiber.Ctx) error {
	var req CreateCategoryRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	cat, err := h.catalogUsecase.CreateCategory(c.Context(), domain.CreateCategoryInput{
		Name:      req.Name,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"message": "สร้างหมวดหมู่สินค้าเรียบร้อยแล้ว",
		"data":    cat,
	})
}

// UpdateCategoryRequest represents request payload for updating category
type UpdateCategoryRequest struct {
	Name      string `json:"name" example:"เครื่องดื่มเย็น" validate:"required"`
	SortOrder *int   `json:"sort_order" example:"2"`
}

// UpdateCategory godoc
// @Summary      แก้ไขข้อมูลหมวดหมู่สินค้า
// @Description  อัปเดตข้อมูลหมวดหมู่ เช่น ชื่อ หรือลำดับการแสดงผล (เฉพาะ Admin)
// @Tags         Categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path string true "รหัสหมวดหมู่ (Category ID / UUID)"
// @Param        request body UpdateCategoryRequest true "ข้อมูลหมวดหมู่ที่ต้องการแก้ไข"
// @Success      200 {object} CategoryActionResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      409 {object} ErrorResponse
// @Router       /categories/{id} [put]
func (h *CatalogHandler) UpdateCategory(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateCategoryRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	cat, err := h.catalogUsecase.UpdateCategory(c.Context(), id, domain.UpdateCategoryInput{
		Name:      req.Name,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "อัปเดตข้อมูลหมวดหมู่สินค้าเรียบร้อยแล้ว",
		"data":    cat,
	})
}

// DeleteCategory godoc
// @Summary      ลบหมวดหมู่สินค้า
// @Description  ลบหมวดหมู่สินค้าออกจากระบบ และปลดหมวดหมู่ออกจากสินค้าที่เคยผูกอยู่โดยอัตโนมัติ (เฉพาะ Admin)
// @Tags         Categories
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "รหัสหมวดหมู่ (Category ID / UUID)"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /categories/{id} [delete]
func (h *CatalogHandler) DeleteCategory(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.catalogUsecase.DeleteCategory(c.Context(), id); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ลบหมวดหมู่สินค้าเรียบร้อยแล้ว",
	})
}
