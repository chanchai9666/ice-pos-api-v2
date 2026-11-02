package usecase

import (
	"context"
	"sync"
	"time"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/pkg/password"
)

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at,omitempty"`
	User         *domain.User `json:"user"`
}

type AuthUsecase interface {
	Login(ctx context.Context, username, plainPassword string) (*AuthResponse, error)
	PinLogin(ctx context.Context, pin string) (*AuthResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error)
}

type authUsecase struct {
	userRepo          domain.UserRepository
	jwtService        *jwt.Service
	usedRefreshTokens sync.Map
}

func NewAuthUsecase(userRepo domain.UserRepository, jwtService *jwt.Service) AuthUsecase {
	return &authUsecase{
		userRepo:   userRepo,
		jwtService: jwtService,
	}
}

func (u *authUsecase) Login(ctx context.Context, username, plainPassword string) (*AuthResponse, error) {
	user, err := u.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, domain.NewAppError(401, "Invalid username or password")
	}

	if !password.Check(plainPassword, user.PasswordHash) {
		return nil, domain.NewAppError(401, "Invalid username or password")
	}

	tokens, err := u.jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, domain.NewAppError(500, "Failed to generate token")
	}

	return &AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
		User:         user,
	}, nil
}

func (u *authUsecase) PinLogin(ctx context.Context, pin string) (*AuthResponse, error) {
	if len(pin) < 4 {
		return nil, domain.NewAppError(400, "PIN must be at least 4 digits")
	}

	user, err := u.userRepo.GetByPin(ctx, pin)
	if err != nil {
		return nil, domain.NewAppError(401, "Invalid PIN")
	}

	tokens, err := u.jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, domain.NewAppError(500, "Failed to generate token")
	}

	return &AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt,
		User:         user,
	}, nil
}

func (u *authUsecase) RefreshToken(ctx context.Context, refreshToken string) (*jwt.TokenPair, error) {
	claims, err := u.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return nil, domain.NewAppError(401, "Invalid or expired refresh token")
	}

	if claims.TokenType != "refresh" {
		return nil, domain.NewAppError(401, "Invalid token type: expected refresh token")
	}

	tokenID := claims.ID
	if tokenID == "" {
		tokenID = refreshToken
	}

	if _, used := u.usedRefreshTokens.Load(tokenID); used {
		return nil, domain.NewAppError(401, "Refresh token has already been used")
	}
	u.usedRefreshTokens.Store(tokenID, true)

	user, err := u.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || !user.IsActive {
		return nil, domain.NewAppError(401, "User account no longer active")
	}

	return u.jwtService.GenerateTokenPair(user.ID, user.Username, user.Role)
}
