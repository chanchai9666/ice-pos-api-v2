package handler

import (
	"strconv"
	"time"

	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type SaleHandler struct {
	saleUsecase domain.SaleUsecase
}

func NewSaleHandler(saleUsecase domain.SaleUsecase) *SaleHandler {
	return &SaleHandler{saleUsecase: saleUsecase}
}

type CheckoutCashRequest struct {
	Lines          []domain.SaleItem `json:"lines"`
	BillDiscount   int64             `json:"bill_discount"`
	TenderedAmount int64             `json:"tendered_amount"`
}

func (h *SaleHandler) CheckoutCash(c fiber.Ctx) error {
	var req CheckoutCashRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	cashierID, _ := c.Locals("username").(string)
	if cashierID == "" {
		cashierID = "cashier"
	}

	sale, receipt, err := h.saleUsecase.CheckoutCash(c.Context(), cashierID, req.Lines, req.BillDiscount, req.TenderedAmount)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"sale":    sale,
			"receipt": receipt,
		},
	})
}

func (h *SaleHandler) GetSaleByID(c fiber.Ctx) error {
	id := c.Params("id")
	sale, err := h.saleUsecase.GetSaleByID(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    sale,
	})
}

func (h *SaleHandler) GetReceipt(c fiber.Ctx) error {
	receiptNo := c.Params("receiptNo")
	receipt, err := h.saleUsecase.GetReceipt(c.Context(), receiptNo)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    receipt,
	})
}

func (h *SaleHandler) ListSales(c fiber.Ctx) error {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var startDate, endDate time.Time
	if startStr != "" {
		startDate, _ = time.Parse("2006-01-02", startStr)
	} else {
		startDate = time.Now().AddDate(0, 0, -30)
	}
	if endStr != "" {
		endDate, _ = time.Parse("2006-01-02", endStr)
	} else {
		endDate = time.Now()
	}

	sales, total, err := h.saleUsecase.ListSales(c.Context(), startDate, endDate, limit, offset)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    sales,
		"total":   total,
	})
}

type VoidSaleRequest struct {
	Reason string `json:"reason"`
}

func (h *SaleHandler) VoidSale(c fiber.Ctx) error {
	id := c.Params("id")
	var req VoidSaleRequest
	_ = c.Bind().JSON(&req)

	if err := h.saleUsecase.VoidSale(c.Context(), id, req.Reason); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "บิลถูกยกเลิกเรียบร้อยแล้ว",
	})
}
