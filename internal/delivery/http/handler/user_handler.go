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
	Username    string `json:"username"`
	Password    string `json:"password"`
	Pin         string `json:"pin"`
	DisplayName string `json:"display_name"`
}

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
	NewPin string `json:"new_pin"`
}

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

