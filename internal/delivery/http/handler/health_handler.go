package handler

import "github.com/gofiber/fiber/v3"

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Check(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "ice-pos-api-v2",
		"version": "2.0.0",
	})
}
