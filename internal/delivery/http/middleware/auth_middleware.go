package middleware

import (
	"strings"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/jwt"

	"github.com/gofiber/fiber/v3"
)

func AuthMiddleware(jwtService *jwt.Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		tokenStr := ""
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
			}
		}

		if tokenStr == "" {
			tokenStr = c.Cookies("access_token")
		}

		if tokenStr == "" {
			return domain.ErrUnauthorized
		}

		claims, err := jwtService.ValidateToken(tokenStr)
		if err != nil || claims.TokenType == "refresh" {
			return domain.ErrUnauthorized
		}

		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

func RequireRoles(roles ...string) fiber.Handler {
	return func(c fiber.Ctx) error {
		userRole, ok := c.Locals("role").(string)
		if !ok {
			return domain.ErrForbidden
		}

		for _, r := range roles {
			if userRole == r {
				return c.Next()
			}
		}

		return domain.ErrForbidden
	}
}
