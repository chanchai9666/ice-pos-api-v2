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
		if appErr.Code == 401 {
			status = fiber.StatusUnauthorized
		} else if appErr.Code == 403 {
			status = fiber.StatusForbidden
		} else if appErr.Code == 404 {
			status = fiber.StatusNotFound
		} else if appErr.Code >= 500 {
			status = fiber.StatusInternalServerError
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
