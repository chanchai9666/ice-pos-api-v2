package handler

import (
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/imageutil"
	"ice-pos-api-v2/internal/pkg/storage"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type UploadHandler struct {
	cfg            *config.Config
	catalogUsecase domain.CatalogUsecase
	storage        storage.StorageService
}

func NewUploadHandler(cfg *config.Config, catalogUsecase domain.CatalogUsecase, storageService storage.StorageService) *UploadHandler {
	return &UploadHandler{
		cfg:            cfg,
		catalogUsecase: catalogUsecase,
		storage:        storageService,
	}
}

const (
	maxImageSize = 10 * 1024 * 1024 // 10MB
)

var allowedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

// UploadImage godoc
// @Summary      อัปโหลดรูปภาพพร้อมปรับขนาดและแปลงเป็น WebP ขนาดไม่เกิน 50KB
// @Description  อัปโหลดไฟล์รูปภาพ (JPG, PNG, WEBP, GIF) รองรับขนาดสูงสุด 10MB ระบบจะทำการปรับขนาดและบีบอัดเป็น .webp อัตโนมัติให้มีขนาดไฟล์ไม่เกิน 50KB เพื่อความรวดเร็วในการโหลดและการซิงค์ข้อมูลออฟไลน์
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "ไฟล์รูปภาพที่ต้องการอัปโหลด (field 'file' หรือ 'image')"
// @Success      201 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /uploads/images [post]
func (h *UploadHandler) UploadImage(c fiber.Ctx) error {
	fileHeader, err := h.getUploadedFile(c)
	if err != nil {
		return domain.NewAppError(400, err.Error())
	}

	resp, err := h.saveImageFile(c, fileHeader)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

// UploadProductImage godoc
// @Summary      อัปโหลดรูปภาพสินค้าโดยตรงพร้อมปรับขนาดและแปลงเป็น WebP ขนาดไม่เกิน 50KB
// @Description  อัปโหลดไฟล์รูปภาพ ปรับขนาด/บีบอัดเป็น .webp ขนาดไม่เกิน 50KB ให้อัตโนมัติ และผูกเข้ากับสินค้าตาม ID ที่ระบุโดยตรง
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        id   path     string true "Product ID (UUID)"
// @Param        file formData file   true "ไฟล์รูปภาพที่ต้องการอัปโหลด"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id}/image [post]
func (h *UploadHandler) UploadProductImage(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrNotFound
	}

	product, err := h.catalogUsecase.GetProductByID(c.Context(), id)
	if err != nil {
		return err
	}

	fileHeader, err := h.getUploadedFile(c)
	if err != nil {
		return domain.NewAppError(400, err.Error())
	}

	uploadOut, err := h.saveImageFile(c, fileHeader)
	if err != nil {
		return err
	}

	product.ImageURL = uploadOut.URL
	if err := h.catalogUsecase.UpdateProduct(c.Context(), product); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    product,
	})
}

func (h *UploadHandler) getUploadedFile(c fiber.Ctx) (*multipart.FileHeader, error) {
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader == nil {
		fileHeader, err = c.FormFile("image")
	}
	if err != nil || fileHeader == nil {
		return nil, fmt.Errorf("กรุณาเลือกไฟล์รูปภาพที่ต้องการอัปโหลด (field 'file' หรือ 'image')")
	}
	return fileHeader, nil
}

func (h *UploadHandler) saveImageFile(c fiber.Ctx, fileHeader *multipart.FileHeader) (*storage.UploadOutput, error) {
	if fileHeader.Size > maxImageSize {
		return nil, domain.NewAppError(400, fmt.Sprintf("ขนาดไฟล์รูปภาพเกินกำหนด (ขนาดสูงสุด %dMB)", maxImageSize/(1024*1024)))
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedImageExtensions[ext] {
		return nil, domain.NewAppError(400, fmt.Sprintf("ประเภทไฟล์ไม่รองรับ (%s) รองรับเฉพาะ .jpg, .jpeg, .png, .webp, .gif", ext))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, domain.NewAppError(500, "ไม่สามารถเปิดไฟล์ที่อัปโหลดได้")
	}
	defer file.Close()

	processed, err := imageutil.ProcessImage(file, ext, imageutil.DefaultOptions)
	if err != nil {
		return nil, domain.NewAppError(500, fmt.Sprintf("เกิดข้อผิดพลาดในการประมวลผลรูปภาพ: %s", err.Error()))
	}

	uniqueFilename := fmt.Sprintf("%s%s", uuid.New().String(), processed.Extension)

	uploadOut, err := h.storage.Upload(c.Context(), storage.UploadInput{
		Filename:    uniqueFilename,
		ContentType: processed.ContentType,
		Data:        processed.Data,
		Folder:      "products",
	})
	if err != nil {
		return nil, domain.NewAppError(500, fmt.Sprintf("การบันทึกรูปภาพล้มเหลว: %s", err.Error()))
	}

	uploadOut.Width = processed.Width
	uploadOut.Height = processed.Height

	return uploadOut, nil
}
