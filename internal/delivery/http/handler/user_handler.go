package handler

import (
	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

type UserHandler struct {
	userUsecase domain.UserUsecase
}

func NewUserHandler(userUsecase domain.UserUsecase) *UserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

// GetAllUsers godoc
// @Summary      รายการผู้ใช้งานทั้งหมด
// @Description  ดึงข้อมูลรายชื่อผู้ใช้งานระบบทั้งหมด (เฉพาะสิทธิ์ Admin)
// @Tags         Users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} UserListDataResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /users [get]
func (h *UserHandler) GetAllUsers(c fiber.Ctx) error {
	users, err := h.userUsecase.GetAllUsers(c.Context())
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{
		"success": true,
		"data":    users,
	})
}

type CreateCashierRequest struct {
	Username    string `json:"username" example:"cashier1"`
	Password    string `json:"password" example:"pass1234"`
	Pin         string `json:"pin" example:"1234"`
	DisplayName string `json:"display_name" example:"สมชาย แคชเชียร์"`
}

// CreateCashier godoc
// @Summary      สร้างบัญชีพนักงานแคชเชียร์ใหม่
// @Description  ลงทะเบียนบัญชีพนักงานขายหน้าร้าน พร้อมชื่อผู้ใช้ รหัสผ่าน และรหัส PIN ด่วน (เฉพาะสิทธิ์ Admin)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body CreateCashierRequest true "ข้อมูลลงทะเบียนพนักงานแคชเชียร์"
// @Success      201 {object} UserDataResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Router       /users [post]
func (h *UserHandler) CreateCashier(c fiber.Ctx) error {
	var req CreateCashierRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	user, err := h.userUsecase.CreateCashier(c.Context(), req.Username, req.Password, req.Pin, req.DisplayName)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

type ResetPinRequest struct {
	NewPin string `json:"new_pin" example:"5678"`
}

// ResetPin godoc
// @Summary      รีเซ็ตรหัส PIN พนักงานแคชเชียร์
// @Description  เปลี่ยนรหัส PIN ด่วนสำหรับพนักงานแคชเชียร์ (เฉพาะสิทธิ์ Admin)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id   path string true "รหัสผู้ใช้งาน (User ID / UUID)"
// @Param        request body ResetPinRequest true "รหัส PIN ใหม่ (ตัวเลข 4-6 หลัก)"
// @Success      200 {object} MessageResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      403 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Router       /users/{id}/reset-pin [post]
func (h *UserHandler) ResetPin(c fiber.Ctx) error {
	id := c.Params("id")
	var req ResetPinRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	if err := h.userUsecase.ResetPin(c.Context(), id, req.NewPin); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "PIN updated successfully",
	})
}
