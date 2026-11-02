package handler

import (
	"strconv"

	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type ReportHandler struct {
	reportUsecase domain.ReportUsecase
}

func NewReportHandler(reportUsecase domain.ReportUsecase) *ReportHandler {
	return &ReportHandler{reportUsecase: reportUsecase}
}

// GetDailySummary godoc
// @Summary      สรุปยอดขายประจำวัน (Daily Sales Summary)
// @Description  ดึงข้อมูลสรุปยอดขายรวม กำไรเบื้องต้น จำนวนบิล และยอดเงินสดที่ได้รับ ตามวันที่ที่ระบุ
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Param        date query string false "วันที่ต้องการดูสรุปยอดขาย (รูปแบบ YYYY-MM-DD, ค่าเริ่มต้นคือวันนี้)"
// @Success      200 {object} DailySummaryResponse
// @Failure      401 {object} ErrorResponse
// @Router       /reports/sales-summary [get]
func (h *ReportHandler) GetDailySummary(c fiber.Ctx) error {
	dateStr := c.Query("date")
	summary, err := h.reportUsecase.GetDailySummary(c.Context(), dateStr)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    summary,
	})
}

// GetBestSellers godoc
// @Summary      รายงานสินค้าขายดี (Best Sellers)
// @Description  ดึงอันดับสินค้าขายดีตามจำนวนชิ้นและยอดขายรวม ในช่วงวันที่ที่กำหนด
// @Tags         Reports
// @Produce      json
// @Security     BearerAuth
// @Param        start_date query string false "วันที่เริ่มต้น (รูปแบบ YYYY-MM-DD)"
// @Param        end_date   query string false "วันที่สิ้นสุด (รูปแบบ YYYY-MM-DD)"
// @Param        limit      query int    false "จำนวนอันดับสินค้าขายดี (ค่าเริ่มต้น 10)"
// @Success      200 {object} BestSellersResponse
// @Failure      401 {object} ErrorResponse
// @Router       /reports/best-sellers [get]
func (h *ReportHandler) GetBestSellers(c fiber.Ctx) error {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	items, err := h.reportUsecase.GetBestSellers(c.Context(), startDate, endDate, limit)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    items,
	})
}
