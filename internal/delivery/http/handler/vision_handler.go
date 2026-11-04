package handler

import (
	"fmt"
	"io"
	"strconv"

	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type VisionHandler struct {
	visionUsecase domain.VisionUsecase
}

func NewVisionHandler(visionUsecase domain.VisionUsecase) *VisionHandler {
	return &VisionHandler{
		visionUsecase: visionUsecase,
	}
}

// MatchFrame godoc
// @Summary      ค้นหาสินค้าจากเฟรมภาพกล้อง (Server-side Visual AI Recognition)
// @Description  อัปโหลดเฟรมภาพจากกล้อง (JPG, PNG, WebP) ระบบหลังบ้านจะสกัด Embedding Vector และคำนวณหาความคล้ายคลึงของสินค้า (Cosine Similarity) กับสินค้าในฐานข้อมูล
// @Tags         Vision AI
// @Security     BearerAuth
// @Accept       multipart/form-data
// @Produce      json
// @Param        file      formData file    true  "ไฟล์รูปภาพหรือเฟรมกล้อง"
// @Param        threshold formData number  false "เกณฑ์ความแม่นยำขั้นต่ำ (Default: 0.70)"
// @Param        limit     formData integer false "จำนวนผลลัพธ์สูงสุด (Default: 5)"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      503 {object} ErrorResponse
// @Router       /vision/match [post]
func (h *VisionHandler) MatchFrame(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader == nil {
		fileHeader, err = c.FormFile("image")
	}
	if err != nil || fileHeader == nil {
		return domain.NewAppError(400, "กรุณาแนบไฟล์รูปภาพหรือเฟรมภาพ (field 'file' หรือ 'image')")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return domain.NewAppError(500, "ไม่สามารถเปิดไฟล์รูปภาพได้")
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		return domain.NewAppError(500, "เกิดข้อผิดพลาดในการอ่านข้อมูลรูปภาพ")
	}

	threshold := 0.70
	if thStr := c.FormValue("threshold"); thStr != "" {
		if val, err := strconv.ParseFloat(thStr, 64); err == nil && val > 0 {
			threshold = val
		}
	}

	limit := 5
	if limStr := c.FormValue("limit"); limStr != "" {
		if val, err := strconv.Atoi(limStr); err == nil && val > 0 {
			limit = val
		}
	}

	resp, err := h.visionUsecase.MatchFrame(c.Context(), imgBytes, threshold, limit)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

// ReindexImages godoc
// @Summary      สร้าง Vector Embedding ให้กับรูปภาพสินค้าที่ยังไม่มี
// @Description  สแกนหารูปภาพสินค้าในระบบที่ยังไม่มี Embedding Vector แล้วส่งไปประมวลผลผ่านโมเดล Vision AI หลังบ้าน เพื่อให้พร้อมสำหรับการค้นหาด้วยภาพ
// @Tags         Vision AI
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} SuccessResponse
// @Failure      503 {object} ErrorResponse
// @Router       /vision/reindex [post]
func (h *VisionHandler) ReindexImages(c fiber.Ctx) error {
	count, err := h.visionUsecase.ReindexImages(c.Context())
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"reindexed_count": count,
			"message":         fmt.Sprintf("สร้าง Vector Embedding สำเร็จ %d รูปภาพ", count),
		},
	})
}
