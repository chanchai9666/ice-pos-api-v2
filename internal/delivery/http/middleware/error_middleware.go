package middleware

import (
	"errors"

	"ice-pos-api-v2/internal/domain"

	"github.com/gofiber/fiber/v3"
)

func CustomErrorHandler(c fiber.Ctx, err error) error {
	var appErr *domain.AppError
	if errors.As(err, &appErr) {
		status := fiber.StatusBadRequest
		if appErr.Code >= 400 && appErr.Code < 600 {
			status = appErr.Code
		}
		return c.Status(status).JSON(fiber.Map{
			"success": false,
			"code":    appErr.Code,
			"message": appErr.Message,
		})
	}

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return c.Status(fiberErr.Code).JSON(fiber.Map{
			"success": false,
			"code":    fiberErr.Code,
			"message": fiberErr.Message,
		})
	}

	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
		"success": false,
		"code":    500,
		"message": err.Error(),
	})
}
