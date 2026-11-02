package handler

import (
	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type SyncHandler struct {
	syncUsecase domain.SyncUsecase
}

func NewSyncHandler(syncUsecase domain.SyncUsecase) *SyncHandler {
	return &SyncHandler{syncUsecase: syncUsecase}
}

// PullChanges godoc
// @Summary      ดึงข้อมูลอัพเดตแคตตาล็อก (Pull Catalog Changes)
// @Description  ดึงข้อมูลสินค้าและหมวดหมู่ที่มีการเปลี่ยนแปลงตั้งแต่เวลาที่ระบุ สำหรับการซิงค์ข้อมูลกับเครื่อง POS ออฟไลน์
// @Tags         Sync
// @Produce      json
// @Security     BearerAuth
// @Param        since query string false "เวลาที่ซิงค์ล่าสุด (รูปแบบ RFC3339 เช่น 2026-01-01T00:00:00Z)"
// @Success      200 {object} SyncChangesResponseWrapper
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /sync/changes [get]
func (h *SyncHandler) PullChanges(c fiber.Ctx) error {
	since := c.Query("since")
	res, err := h.syncUsecase.PullChanges(c.Context(), since)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

// PushBatch godoc
// @Summary      ส่งข้อมูลบิลขายออฟไลน์ขึ้นเซิร์ฟเวอร์ (Push Offline Sales Batch)
// @Description  อัปโหลดรายการขายที่บันทึกไว้ในเครื่อง POS ช่วงออฟไลน์ขึ้นมาบันทึกที่เซิร์ฟเวอร์พร้อมกันเป็นชุด (Batch)
// @Tags         Sync
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body domain.SyncBatchRequest true "ชุดข้อมูลรายการขายช่วงออฟไลน์"
// @Success      200 {object} SyncBatchResponseWrapper
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /sync/batch [post]
func (h *SyncHandler) PushBatch(c fiber.Ctx) error {
	var req domain.SyncBatchRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	res, err := h.syncUsecase.PushBatch(c.Context(), req)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}
