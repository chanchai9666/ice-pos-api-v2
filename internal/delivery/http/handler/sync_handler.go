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
