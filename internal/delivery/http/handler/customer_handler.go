package handler

import (
	"strconv"
	"strings"

	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type CustomerHandler struct {
	customerUsecase domain.CustomerUsecase
}

func NewCustomerHandler(customerUsecase domain.CustomerUsecase) *CustomerHandler {
	return &CustomerHandler{customerUsecase: customerUsecase}
}

type CreateCustomerRequest struct {
	Name               string  `json:"name" example:"สมชาย ใจดี"`
	Phone              *string `json:"phone,omitempty" example:"0812345678"`
	Email              *string `json:"email,omitempty" example:"somchai@example.com"`
	IsMember           bool    `json:"is_member" example:"true"`
	InitialCreditLimit int64   `json:"initial_credit_limit" example:"100000"` // สตางค์
	CreditLimit        int64   `json:"credit_limit,omitempty"`
}

// CreateCustomer godoc
// @Summary      ลงทะเบียนลูกค้าใหม่และเปิดบัญชีเงินเชื่อ
// @Description  เพิ่มข้อมูลลูกค้าใหม่ สมาชิก และกำหนดวงเงินเครดิตเริ่มต้น (หน่วยสตางค์)
// @Tags         Customers
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body CreateCustomerRequest true "ข้อมูลสำหรับสร้างลูกค้า"
// @Success      201 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers [post]
func (h *CustomerHandler) CreateCustomer(c fiber.Ctx) error {
	var req CreateCustomerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	limit := req.InitialCreditLimit
	if limit == 0 && req.CreditLimit > 0 {
		limit = req.CreditLimit
	}

	customer := domain.Customer{
		Name:     strings.TrimSpace(req.Name),
		Phone:    req.Phone,
		Email:    req.Email,
		IsMember: req.IsMember,
	}

	if err := h.customerUsecase.CreateCustomer(c.Context(), &customer, limit); err != nil {
		return err
	}

	created, err := h.customerUsecase.GetCustomerByID(c.Context(), customer.ID)
	if err != nil {
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{
			"success": true,
			"data":    customer,
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    created,
	})
}

// GetAllCustomers godoc
// @Summary      ดูรายการลูกค้าและยอดค้างชำระ
// @Description  ดึงรายการลูกค้าทั้งหมด พร้อมแสดงวงเงินเครดิต ยอดหนี้คงค้าง และคะแนนสะสม
// @Tags         Customers
// @Security     BearerAuth
// @Produce      json
// @Param        search query string false "ค้นหาจากชื่อ เบอร์โทร หรืออีเมล"
// @Param        is_member query boolean false "กรองเฉพาะสมาชิก (true/false)"
// @Param        page query int false "หน้าที่ต้องการ (เริ่มต้น 1)"
// @Param        limit query int false "จำนวนรายการต่อหน้า (เริ่มต้น 20, สูงสุด 100)"
// @Success      200 {object} SuccessResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers [get]
func (h *CustomerHandler) GetAllCustomers(c fiber.Ctx) error {
	search := c.Query("search")
	var isMemberPtr *bool
	if isMemberStr := c.Query("is_member"); isMemberStr != "" {
		val := isMemberStr == "true" || isMemberStr == "1"
		isMemberPtr = &val
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	customers, total, err := h.customerUsecase.GetAllCustomers(c.Context(), domain.CustomerFilter{
		Search:   search,
		IsMember: isMemberPtr,
		Page:     page,
		Limit:    limit,
	})
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    customers,
		"meta": fiber.Map{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

// GetCustomerByID godoc
// @Summary      ดูรายละเอียดข้อมูลลูกค้า
// @Description  ดึงข้อมูลโปรไฟล์ลูกค้า สถานะวงเงินเครดิต และยอดหนี้คงค้าง
// @Tags         Customers
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Customer UUID"
// @Success      200 {object} SuccessResponse
// @Failure      401 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /customers/{id} [get]
func (h *CustomerHandler) GetCustomerByID(c fiber.Ctx) error {
	id := c.Params("id")
	customer, err := h.customerUsecase.GetCustomerByID(c.Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    customer,
	})
}

type UpdateCustomerRequest struct {
	Name     string  `json:"name" example:"สมชาย ใจดีมาก"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
	IsMember bool    `json:"is_member"`
}

// UpdateCustomer godoc
// @Summary      แก้ไขข้อมูลลูกค้า
// @Description  อัปเดตข้อมูลลูกค้า เช่น ชื่อ เบอร์โทรศัพท์ อีเมล หรือสถานะสมาชิก
// @Tags         Customers
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Customer UUID"
// @Param        request body UpdateCustomerRequest true "ข้อมูลสำหรับอัปเดตลูกค้า"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers/{id} [put]
func (h *CustomerHandler) UpdateCustomer(c fiber.Ctx) error {
	id := c.Params("id")
	var req UpdateCustomerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	customer, err := h.customerUsecase.GetCustomerByID(c.Context(), id)
	if err != nil {
		return err
	}

	customer.Name = strings.TrimSpace(req.Name)
	customer.Phone = req.Phone
	customer.Email = req.Email
	customer.IsMember = req.IsMember

	if err := h.customerUsecase.UpdateCustomer(c.Context(), customer); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    customer,
	})
}

// DeleteCustomer godoc
// @Summary      ลบข้อมูลลูกค้า (Soft Delete)
// @Description  ลบข้อมูลลูกค้า (ไม่อนุญาตให้ลบหากยังมียอดหนี้คงค้างในบัญชีเงินเชื่อ)
// @Tags         Customers
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Customer UUID"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers/{id} [delete]
func (h *CustomerHandler) DeleteCustomer(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.customerUsecase.DeleteCustomer(c.Context(), id); err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"message": "ลบข้อมูลลูกค้าเรียบร้อยแล้ว",
	})
}

type SetCreditLimitRequest struct {
	CreditLimit int64 `json:"credit_limit" example:"200000"`
}

// SetCreditLimit godoc
// @Summary      กำหนดหรือปรับเปลี่ยนวงเงินเครดิตลูกค้า
// @Description  ปรับเปลี่ยนวงเงินเครดิตสูงสุดที่ลูกค้าสามารถติดค้างชำระได้ (หน่วยสตางค์)
// @Tags         Customers
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Customer UUID"
// @Param        request body SetCreditLimitRequest true "ข้อมูลวงเงินเครดิตใหม่"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers/{id}/credit-limit [put]
func (h *CustomerHandler) SetCreditLimit(c fiber.Ctx) error {
	id := c.Params("id")
	var req SetCreditLimitRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	if err := h.customerUsecase.SetCreditLimit(c.Context(), id, req.CreditLimit); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "อัปเดตวงเงินเครดิตสำเร็จ",
	})
}

type PayCreditRequest struct {
	Amount         int64   `json:"amount" example:"50000"`
	PaymentMethod  string  `json:"payment_method" example:"CASH"`
	Note           string  `json:"note,omitempty" example:"ชำระหนี้ค้างรอบสัปดาห์"`
	IdempotencyKey *string `json:"idempotency_key,omitempty"`
}

// PayCredit godoc
// @Summary      บันทึกการรับชำระหนี้จากลูกค้า
// @Description  รับชำระหนี้คงค้างจากลูกค้า (เงินสด, โอนเงิน) พร้อมลดยอดหนี้คงเหลือและบันทึกประวัติธุรกรรม
// @Tags         Customers
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        id path string true "Customer UUID"
// @Param        request body PayCreditRequest true "ข้อมูลการรับชำระหนี้"
// @Success      200 {object} SuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers/{id}/pay-credit [post]
func (h *CustomerHandler) PayCredit(c fiber.Ctx) error {
	id := c.Params("id")
	var req PayCreditRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	actorID, _ := c.Locals("username").(string)
	txRecord, err := h.customerUsecase.PayCredit(c.Context(), id, req.Amount, req.PaymentMethod, req.Note, req.IdempotencyKey, actorID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    txRecord,
	})
}

// GetCreditHistory godoc
// @Summary      ดูประวัติการเคลื่อนไหวของหนี้และยอดชำระ
// @Description  ดึงรายการประวัติการซื้อเชื่อ ชำระหนี้ และปรับยอดหนี้ย้อนหลังของลูกค้า
// @Tags         Customers
// @Security     BearerAuth
// @Produce      json
// @Param        id path string true "Customer UUID"
// @Param        limit query int false "จำนวนรายการที่ต้องการ (เริ่มต้น 50)"
// @Success      200 {object} SuccessResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers/{id}/credit-history [get]
func (h *CustomerHandler) GetCreditHistory(c fiber.Ctx) error {
	id := c.Params("id")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	history, err := h.customerUsecase.GetCreditHistory(c.Context(), id, limit)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    history,
	})
}

// GetARAgingReport godoc
// @Summary      รายงานลูกหนี้ค้างชำระตามอายุหนี้ (AR Aging Report)
// @Description  รายงานวิเคราะห์อายุหนี้คงค้างของลูกค้า แบ่งเป็นช่วง 0-30 วัน, 31-60 วัน, 61-90 วัน และเกิน 90 วัน
// @Tags         Customers
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} SuccessResponse
// @Failure      401 {object} ErrorResponse
// @Router       /customers/reports/ar-aging [get]
func (h *CustomerHandler) GetARAgingReport(c fiber.Ctx) error {
	report, err := h.customerUsecase.GetARAgingReport(c.Context())
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    report,
	})
}
