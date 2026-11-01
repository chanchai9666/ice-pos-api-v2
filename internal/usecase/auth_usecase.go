package usecase

import (
	"context"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/jwt"
	"ice-pos-api-v2/internal/pkg/password"
)

type AuthResponse struct {
	AccessToken string       `json:"access_token"`
	User        *domain.User `json:"user"`
}

type AuthUsecase interface {
	Login(ctx context.Context, username, plainPassword string) (*AuthResponse, error)
	PinLogin(ctx context.Context, pin string) (*AuthResponse, error)
}

type authUsecase struct {
	userRepo   domain.UserRepository
	jwtService *jwt.Service
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

	token, err := u.jwtService.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, domain.NewAppError(500, "Failed to generate token")
	}

	return &AuthResponse{
		AccessToken: token,
		User:        user,
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

	token, err := u.jwtService.GenerateAccessToken(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, domain.NewAppError(500, "Failed to generate token")
	}

	return &AuthResponse{
		AccessToken: token,
		User:        user,
	}, nil
}
