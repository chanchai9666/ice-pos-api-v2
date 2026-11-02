package handler

import (
	"time"

	"ice-pos-api-v2/config"
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/cookie"
	"ice-pos-api-v2/internal/usecase"

	"github.com/gofiber/fiber/v3"
)

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
	cfg         *config.Config
}

func NewAuthHandler(authUsecase usecase.AuthUsecase, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		authUsecase: authUsecase,
		cfg:         cfg,
	}
}

func (h *AuthHandler) getTokensTTL() (time.Duration, time.Duration) {
	accessTTL := 24 * time.Hour
	refreshTTL := 7 * 24 * time.Hour
	if h.cfg != nil {
		if h.cfg.JWTAccessTTL > 0 {
			accessTTL = h.cfg.JWTAccessTTL
		}
		if h.cfg.JWTRefreshTTL > 0 {
			refreshTTL = h.cfg.JWTRefreshTTL
		}
	}
	return accessTTL, refreshTTL
}

type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"admin123"`
}

// Login godoc
// @Summary      เข้าสู่ระบบด้วย Username & Password
// @Description  ยืนยันตัวตนผู้ใช้งานด้วยชื่อผู้ใช้และรหัสผ่าน เพื่อรับ JWT access token และ refresh token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "ข้อมูลเข้าสู่ระบบ (Username และ Password)"
// @Success      200 {object} AuthDataResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	res, err := h.authUsecase.Login(c.Context(), req.Username, req.Password)
	if err != nil {
		return err
	}

	accessTTL, refreshTTL := h.getTokensTTL()
	cookie.SetAuthCookies(c, h.cfg, res.AccessToken, res.RefreshToken, accessTTL, refreshTTL)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

type PinLoginRequest struct {
	Pin string `json:"pin" example:"1234"`
}

// PinLogin godoc
// @Summary      เข้าสู่ระบบด่วนด้วย PIN (สำหรับพนักงานแคชเชียร์)
// @Description  เข้าสู่ระบบด่วนสำหรับพนักงานขายหน้าร้านด้วยรหัสตัวเลข PIN 4-6 หลัก
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body PinLoginRequest true "รหัส PIN พนักงานแคชเชียร์"
// @Success      200 {object} AuthDataResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /auth/pin-login [post]
func (h *AuthHandler) PinLogin(c fiber.Ctx) error {
	var req PinLoginRequest
	if err := c.Bind().JSON(&req); err != nil {
		return domain.ErrBadRequest
	}

	res, err := h.authUsecase.PinLogin(c.Context(), req.Pin)
	if err != nil {
		return err
	}

	accessTTL, refreshTTL := h.getTokensTTL()
	cookie.SetAuthCookies(c, h.cfg, res.AccessToken, res.RefreshToken, accessTTL, refreshTTL)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    res,
	})
}

type RefreshTokenRequest struct {
	RefreshToken      string `json:"refreshToken"`
	RefreshTokenSnake string `json:"refresh_token"`
}

// RefreshToken godoc
// @Summary      ขอต่ออายุ Access Token (Refresh Token)
// @Description  รับ Access Token และ Refresh Token ชุดใหม่โดยใช้ Refresh Token ที่ยังไม่หมดอายุ (รองรับทั้ง Cookie refresh_token และ JSON Body)
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request body RefreshTokenRequest false "Payload สำหรับต่ออายุ Token (ไม่จำเป็นหากส่งผ่าน Cookie)"
// @Success      200 {object} AuthDataResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Router       /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c fiber.Ctx) error {
	var refreshToken string
	cookieToken := c.Cookies(cookie.RefreshTokenCookieName)

	var req RefreshTokenRequest
	if len(c.Body()) > 0 {
		_ = c.Bind().JSON(&req)
	}

	if req.RefreshToken != "" {
		refreshToken = req.RefreshToken
	} else if req.RefreshTokenSnake != "" {
		refreshToken = req.RefreshTokenSnake
	} else if cookieToken != "" {
		refreshToken = cookieToken
	}

	if refreshToken == "" {
		return domain.NewAppError(400, "Refresh token is required via cookie or JSON body")
	}

	tokens, err := h.authUsecase.RefreshToken(c.Context(), refreshToken)
	if err != nil {
		return err
	}

	accessTTL, refreshTTL := h.getTokensTTL()
	cookie.SetAuthCookies(c, h.cfg, tokens.AccessToken, tokens.RefreshToken, accessTTL, refreshTTL)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
			"expires_at":    tokens.ExpiresAt,
		},
	})
}

// Logout godoc
// @Summary      ออกจากระบบและล้างคุกกี้ (Logout)
// @Description  ออกจากระบบและสั่งให้เบราว์เซอร์ทำลายคุกกี้ access_token และ refresh_token
// @Tags         Auth
// @Produce      json
// @Success      200 {object} SuccessResponse
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(c fiber.Ctx) error {
	cookie.ClearAuthCookies(c, h.cfg)
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Logged out successfully",
	})
}

// GetMe godoc
// @Summary      ดูข้อมูลโปรไฟล์ผู้ใช้งานปัจจุบัน
// @Description  ดึงข้อมูลโปรไฟล์และบทบาทของผู้ใช้งานที่กำลังเข้าสู่ระบบผ่าน Token
// @Tags         Auth
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} MeResponse
// @Failure      401 {object} ErrorResponse
// @Router       /auth/me [get]
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
