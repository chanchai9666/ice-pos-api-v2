package handler

import (
	"encoding/json"
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
	visionService  domain.VisionEmbeddingService
}

func NewUploadHandler(cfg *config.Config, catalogUsecase domain.CatalogUsecase, storageService storage.StorageService, visionService ...domain.VisionEmbeddingService) *UploadHandler {
	var vs domain.VisionEmbeddingService
	if len(visionService) > 0 {
		vs = visionService[0]
	}
	return &UploadHandler{
		cfg:            cfg,
		catalogUsecase: catalogUsecase,
		storage:        storageService,
		visionService:  vs,
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

	resp, _, err := h.saveImageFile(c, fileHeader)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

// UploadProductImage godoc
// @Summary      อัปโหลดรูปภาพสินค้าโดยตรงพร้อมปรับขนาดและแปลงเป็น WebP ขนาดไม่เกิน 50KB (Legacy Endpoint)
// @Description  อัปโหลดไฟล์รูปภาพ ปรับขนาด/บีบอัดเป็น .webp ขนาดไม่เกิน 50KB ให้อัตโนมัติ และผูกเข้ากับสินค้าตาม ID ที่ระบุโดยตรงเป็นรูปภาพหลัก
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

	_, err := h.catalogUsecase.GetProductByID(c.Context(), id)
	if err != nil {
		return err
	}

	fileHeader, err := h.getUploadedFile(c)
	if err != nil {
		return domain.NewAppError(400, err.Error())
	}

	uploadOut, imgBytes, err := h.saveImageFile(c, fileHeader)
	if err != nil {
		return err
	}

	var embedding domain.FloatVector
	var embeddingModel string
	if h.visionService != nil {
		if emb, model, vErr := h.visionService.ExtractEmbedding(c.Context(), imgBytes); vErr == nil && len(emb) > 0 {
			embedding = emb
			embeddingModel = model
		}
	}

	_, err = h.catalogUsecase.AddProductImage(c.Context(), domain.CreateProductImageInput{
		ProductID:      id,
		ImageURL:       uploadOut.URL,
		IsPrimary:      true,
		Embedding:      embedding,
		EmbeddingModel: embeddingModel,
	})
	if err != nil {
		return err
	}

	product, err := h.catalogUsecase.GetProductByID(c.Context(), id)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    product,
	})
}

// AddProductImage godoc
// @Summary      เพิ่มรูปภาพสินค้าพร้อม Embedding สำหรับ Visual AI Matching
// @Description  อัปโหลดไฟล์รูปภาพสินค้า รองรับหลายมุมมอง พร้อมแนบ Embedding Vector (JSON Array of float32) และระบุชื่อโมเดล เพื่อใช้ในการสแกนหาสินค้าด้วยกล้อง
// @Tags         Products
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        id              path     string true  "Product ID (UUID)"
// @Param        file            formData file   true  "ไฟล์รูปภาพสินค้า"
// @Param        embedding       formData string false "JSON array ของตัวเลข float32 เช่น [0.12, -0.45, ...]"
// @Param        embedding_model formData string false "ชื่อหรือเวอร์ชันโมเดล Vision AI เช่น clip-vit-base-patch32"
// @Param        is_primary      formData bool   false "กำหนดให้เป็นรูปหลักหรือไม่ (true/false)"
// @Success      201 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id}/images [post]
func (h *UploadHandler) AddProductImage(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return domain.ErrNotFound
	}

	fileHeader, err := h.getUploadedFile(c)
	if err != nil {
		return domain.NewAppError(400, err.Error())
	}

	var embedding domain.FloatVector
	if embStr := strings.TrimSpace(c.FormValue("embedding")); embStr != "" {
		if err := json.Unmarshal([]byte(embStr), &embedding); err != nil {
			return domain.NewAppError(400, "รูปแบบ embedding ไม่ถูกต้อง ต้องเป็น JSON array ของตัวเลข (float32)")
		}
	}

	embeddingModel := strings.TrimSpace(c.FormValue("embedding_model"))
	isPrimaryStr := strings.ToLower(strings.TrimSpace(c.FormValue("is_primary")))
	isPrimary := isPrimaryStr == "true" || isPrimaryStr == "1"

	uploadOut, imgBytes, err := h.saveImageFile(c, fileHeader)
	if err != nil {
		return err
	}

	if len(embedding) == 0 && h.visionService != nil {
		if emb, model, vErr := h.visionService.ExtractEmbedding(c.Context(), imgBytes); vErr == nil && len(emb) > 0 {
			embedding = emb
			if embeddingModel == "" {
				embeddingModel = model
			}
		}
	}

	prodImage, err := h.catalogUsecase.AddProductImage(c.Context(), domain.CreateProductImageInput{
		ProductID:      id,
		ImageURL:       uploadOut.URL,
		IsPrimary:      isPrimary,
		Embedding:      embedding,
		EmbeddingModel: embeddingModel,
	})
	if err != nil {
		_ = h.storage.Delete(c.Context(), uploadOut.URL)
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    prodImage,
	})
}

// DeleteProductImage godoc
// @Summary      ลบรูปภาพสินค้า
// @Description  ลบรูปภาพสินค้าและลบไฟล์จริงออกจาก Storage หากรูปที่ลบเป็นรูปหลัก ระบบจะสลับรูปถัดไปขึ้นมาเป็นรูปหลักแทนให้อัตโนมัติ
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        id      path string true "Product ID (UUID)"
// @Param        imageId path string true "Image ID (UUID)"
// @Success      200 {object} SuccessResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id}/images/{imageId} [delete]
func (h *UploadHandler) DeleteProductImage(c fiber.Ctx) error {
	id := c.Params("id")
	imageID := c.Params("imageId")

	img, err := h.catalogUsecase.GetProductImageByID(c.Context(), imageID)
	if err != nil {
		return err
	}
	if img.ProductID != id {
		return domain.ErrNotFound
	}

	if err := h.catalogUsecase.DeleteProductImage(c.Context(), id, imageID); err != nil {
		return err
	}

	_ = h.storage.Delete(c.Context(), img.ImageURL)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ลบรูปภาพสินค้าเรียบร้อยแล้ว",
	})
}

// SetPrimaryProductImage godoc
// @Summary      ตั้งรูปภาพเป็นรูปหลักของสินค้า
// @Description  กำหนดรูปภาพเป้าหมายให้เป็น Primary Image สำหรับแสดงบน POS Card พร้อมซิงค์ image_url ของสินค้า
// @Tags         Products
// @Security     BearerAuth
// @Produce      json
// @Param        id      path string true "Product ID (UUID)"
// @Param        imageId path string true "Image ID (UUID)"
// @Success      200 {object} SuccessResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id}/images/{imageId}/primary [patch]
func (h *UploadHandler) SetPrimaryProductImage(c fiber.Ctx) error {
	id := c.Params("id")
	imageID := c.Params("imageId")

	if err := h.catalogUsecase.SetPrimaryProductImage(c.Context(), id, imageID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "ตั้งเป็นรูปภาพหลักเรียบร้อยแล้ว",
	})
}

// UpdateProductImageEmbedding godoc
// @Summary      อัปเดตเฉพาะ Embedding Vector ของรูปภาพสินค้า
// @Description  ส่ง Embedding Vector ชุดใหม่มาอัปเดตสำหรับรูปเดิม โดยไม่ต้องอัปโหลดไฟล์รูปภาพใหม่ (เหมาะสำหรับการอัปเกรดโมเดล Vision AI)
// @Tags         Products
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id      path string true "Product ID (UUID)"
// @Param        imageId path string true "Image ID (UUID)"
// @Param        body    body domain.UpdateProductImageEmbeddingInput true "Embedding vector และ Model"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id}/images/{imageId}/embedding [patch]
func (h *UploadHandler) UpdateProductImageEmbedding(c fiber.Ctx) error {
	id := c.Params("id")
	imageID := c.Params("imageId")

	var input domain.UpdateProductImageEmbeddingInput
	if err := c.Bind().JSON(&input); err != nil {
		return domain.ErrBadRequest
	}

	if len(input.Embedding) == 0 {
		return domain.NewAppError(400, "กรุณาระบุ embedding vector")
	}

	if err := h.catalogUsecase.UpdateProductImageEmbedding(c.Context(), id, imageID, input); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "อัปเดต embedding เรียบร้อยแล้ว",
	})
}

// ListProductImages godoc
// @Summary      ดูรายการรูปภาพทั้งหมดของสินค้า
// @Description  ดึงรายการรูปภาพทั้งหมดพร้อม Embedding Vector ของสินค้าตาม ID
// @Tags         Products
// @Produce      json
// @Param        id path string true "Product ID (UUID)"
// @Success      200 {object} SuccessResponse
// @Failure      404 {object} ErrorResponse
// @Router       /products/{id}/images [get]
func (h *UploadHandler) ListProductImages(c fiber.Ctx) error {
	id := c.Params("id")
	images, err := h.catalogUsecase.ListProductImages(c.Context(), id)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    images,
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

func (h *UploadHandler) saveImageFile(c fiber.Ctx, fileHeader *multipart.FileHeader) (*storage.UploadOutput, []byte, error) {
	if fileHeader.Size > maxImageSize {
		return nil, nil, domain.NewAppError(400, fmt.Sprintf("ขนาดไฟล์รูปภาพเกินกำหนด (ขนาดสูงสุด %dMB)", maxImageSize/(1024*1024)))
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !allowedImageExtensions[ext] {
		return nil, nil, domain.NewAppError(400, fmt.Sprintf("ประเภทไฟล์ไม่รองรับ (%s) รองรับเฉพาะ .jpg, .jpeg, .png, .webp, .gif", ext))
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, nil, domain.NewAppError(500, "ไม่สามารถเปิดไฟล์ที่อัปโหลดได้")
	}
	defer file.Close()

	processed, err := imageutil.ProcessImage(file, ext, imageutil.DefaultOptions)
	if err != nil {
		return nil, nil, domain.NewAppError(500, fmt.Sprintf("เกิดข้อผิดพลาดในการประมวลผลรูปภาพ: %s", err.Error()))
	}

	uniqueFilename := fmt.Sprintf("%s%s", uuid.New().String(), processed.Extension)

	uploadOut, err := h.storage.Upload(c.Context(), storage.UploadInput{
		Filename:    uniqueFilename,
		ContentType: processed.ContentType,
		Data:        processed.Data,
		Folder:      "products",
	})
	if err != nil {
		return nil, nil, domain.NewAppError(500, fmt.Sprintf("การบันทึกรูปภาพล้มเหลว: %s", err.Error()))
	}

	uploadOut.Width = processed.Width
	uploadOut.Height = processed.Height

	return uploadOut, processed.Data, nil
}
