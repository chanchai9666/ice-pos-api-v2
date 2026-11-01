package handler

import (
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
}

func NewAuthHandler(authUsecase usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	res, err := h.authUsecase.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

type PinLoginRequest struct {
	Pin string `json:"pin"`
}

func (h *AuthHandler) PinLogin(c fiber.Ctx) error {
	var req PinLoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	res, err := h.authUsecase.PinLogin(c.Context(), req.Pin)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

func (h *AuthHandler) GetMe(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user_id":  c.Locals("userID"),
			"username": c.Locals("username"),
			"role":     c.Locals("role"),
		},
	})
}
