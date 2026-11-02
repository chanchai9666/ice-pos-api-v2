package cookie

import (
	"time"

	"ice-pos-api-v2/config"

	"github.com/gofiber/fiber/v3"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"
	RefreshTokenPath       = "/api/v1/auth/refresh"
)

// SetAuthCookies issues access_token and refresh_token HttpOnly cookies
func SetAuthCookies(c fiber.Ctx, cfg *config.Config, accessToken, refreshToken string, accessTTL, refreshTTL time.Duration) {
	secure := false
	if cfg != nil && cfg.AppEnv == "production" {
		secure = true
	}

	if accessToken != "" {
		maxAge := int(accessTTL.Seconds())
		c.Cookie(&fiber.Cookie{
			Name:     AccessTokenCookieName,
			Value:    accessToken,
			Path:     "/",
			MaxAge:   maxAge,
			Expires:  time.Now().Add(accessTTL),
			Secure:   secure,
			HTTPOnly: true,
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}

	if refreshToken != "" {
		maxAge := int(refreshTTL.Seconds())
		c.Cookie(&fiber.Cookie{
			Name:     RefreshTokenCookieName,
			Value:    refreshToken,
			Path:     RefreshTokenPath,
			MaxAge:   maxAge,
			Expires:  time.Now().Add(refreshTTL),
			Secure:   secure,
			HTTPOnly: true,
			SameSite: fiber.CookieSameSiteLaxMode,
		})
	}
}

// ClearAuthCookies clears both access_token and refresh_token cookies
func ClearAuthCookies(c fiber.Ctx, cfg *config.Config) {
	secure := false
	if cfg != nil && cfg.AppEnv == "production" {
		secure = true
	}

	c.Cookie(&fiber.Cookie{
		Name:     AccessTokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Now().Add(-1 * time.Hour),
		Secure:   secure,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})

	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     RefreshTokenPath,
		MaxAge:   -1,
		Expires:  time.Now().Add(-1 * time.Hour),
		Secure:   secure,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}
