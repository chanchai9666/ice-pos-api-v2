package usecase

import (
	"context"
	"time"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/password"

	"github.com/google/uuid"
)

type userUsecase struct {
	userRepo domain.UserRepository
}

func NewUserUsecase(userRepo domain.UserRepository) domain.UserUsecase {
	return &userUsecase{userRepo: userRepo}
}

func (u *userUsecase) GetAllUsers(ctx context.Context) ([]domain.User, error) {
	return u.userRepo.ListAll(ctx)
}

func (u *userUsecase) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return u.userRepo.GetByID(ctx, id)
}

func (u *userUsecase) CreateCashier(ctx context.Context, username, plainPassword, pin, displayName string) (*domain.User, error) {
	existing, err := u.userRepo.GetByUsername(ctx, username)
	if err == nil && existing != nil {
		return nil, domain.NewAppError(400, "Username already exists")
	}

	pwdHash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, domain.NewAppError(500, "Failed to hash password")
	}

	var pinHash string
	if pin != "" {
		pinHash, err = password.Hash(pin)
		if err != nil {
			return nil, domain.NewAppError(500, "Failed to hash PIN")
		}
	}

	user := &domain.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: pwdHash,
		PinHash:      pinHash,
		DisplayName:  displayName,
		Role:         domain.RoleCashier,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := u.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (u *userUsecase) ResetPin(ctx context.Context, userID, newPin string) error {
	if len(newPin) < 4 {
		return domain.NewAppError(400, "PIN must be at least 4 digits")
	}

	pinHash, err := password.Hash(newPin)
	if err != nil {
		return domain.NewAppError(500, "Failed to hash PIN")
	}

	return u.userRepo.UpdatePin(ctx, userID, pinHash)
}
