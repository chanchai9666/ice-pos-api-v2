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

