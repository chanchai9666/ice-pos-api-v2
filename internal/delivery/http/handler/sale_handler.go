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

type CheckoutRequest struct {
	Lines          []domain.SaleItem `json:"lines"`
	BillDiscount   int64             `json:"bill_discount" example:"0"`
	TenderedAmount int64             `json:"tendered_amount" example:"50000"`
	PaymentMethod  string            `json:"payment_method" example:"CASH"` // CASH หรือ CREDIT
	CustomerID     string            `json:"customer_id,omitempty"`
}

// Checkout godoc
// @Summary      บันทึกการขายและชำระเงิน (Checkout - Cash or Credit)
// @Description  บันทึกการขาย รองรับทั้งการชำระเงินสด (คำนวณเงินทอน) และลงบัญชีเงินเชื่อ (ผูกกับข้อมูลลูกค้าและตัดยอดวงเงินเครดิต)
// @Tags         Sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CheckoutRequest true "ข้อมูลรายการสินค้า ส่วนลด ยอดเงินสด หรือระบุลูกค้าสำหรับเงินเชื่อ"
// @Success      201 {object} CheckoutResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /sales [post]
func (h *SaleHandler) Checkout(c fiber.Ctx) error {
	var req CheckoutRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	cashierID, _ := c.Locals("username").(string)
	if cashierID == "" {
		cashierID = "cashier"
	}

	sale, receipt, err := h.saleUsecase.Checkout(c.Context(), cashierID, req.Lines, req.BillDiscount, req.TenderedAmount, req.PaymentMethod, req.CustomerID)
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

// CheckoutCash alias for backward compatibility
func (h *SaleHandler) CheckoutCash(c fiber.Ctx) error {
	return h.Checkout(c)
}

// GetSaleByID godoc
// @Summary      ดูข้อมูลการขายตาม ID
// @Description  ดึงรายละเอียดรายการขายและสินค้าในบิลทั้งหมดด้วย Sale ID
// @Tags         Sales
// @Produce      json
// @Security     BearerAuth
// @Param        id  path string true "รหัสรายการขาย (Sale ID / UUID)"
// @Success      200 {object} SaleDataResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /sales/{id} [get]
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

// GetReceipt godoc
// @Summary      ดูข้อมูลใบเสร็จรับเงิน
// @Description  ดึงข้อมูลใบเสร็จรับเงินสำหรับพิมพ์ใบเสร็จเครื่องพิมพ์ความร้อน (Thermal Printer) ด้วยเลขที่ใบเสร็จหรือเลขที่บิลขาย
// @Tags         Sales
// @Produce      json
// @Security     BearerAuth
// @Param        receiptNo path string true "เลขที่ใบเสร็จรับเงิน (Receipt Number) หรือเลขที่บิล (Sale No)"
// @Success      200 {object} ReceiptDataResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /sales/receipt/{receiptNo} [get]
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

// ListSales godoc
// @Summary      ประวัติรายการขาย
// @Description  ค้นหาและดูรายการประวัติการขาย สามารถกรองตามช่วงวันที่และแบ่งหน้าได้
// @Tags         Sales
// @Produce      json
// @Security     BearerAuth
// @Param        start_date query string false "วันที่เริ่มต้น (รูปแบบ YYYY-MM-DD)"
// @Param        end_date   query string false "วันที่สิ้นสุด (รูปแบบ YYYY-MM-DD)"
// @Param        limit      query int    false "จำนวนรายการสูงสุดต่อหน้า (ค่าเริ่มต้น 50)"
// @Param        offset     query int    false "ตำแหน่งเริ่มต้น (ค่าเริ่มต้น 0)"
// @Success      200 {object} SaleListDataResponse
// @Failure      401 {object} ErrorResponse
// @Router       /sales [get]
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
	Reason string `json:"reason" example:"ลูกค้าเปลี่ยนใจ / คีย์รายการผิด"`
}

// VoidSale godoc
// @Summary      ยกเลิกบิลขาย (Void Sale)
// @Description  ยกเลิกรายการขายที่บันทึกแล้ว พร้อมระบุเหตุผลในการยกเลิกบิล
// @Tags         Sales
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id      path string          true "รหัสรายการขาย (Sale ID / UUID)"
// @Param        request body VoidSaleRequest true "เหตุผลในการขอยกเลิกบิล"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /sales/{id}/void [post]
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
