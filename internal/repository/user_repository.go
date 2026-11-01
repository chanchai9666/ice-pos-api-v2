package repository

import (
	"context"
	"errors"

	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/pkg/password"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, "username = ? AND is_active = ?", username, true).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByPin(ctx context.Context, pin string) (*domain.User, error) {
	var users []domain.User
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&users).Error
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.PinHash != "" && password.Check(pin, u.PinHash) {
			return &u, nil
		}
	}

	return nil, domain.ErrNotFound
}

func (r *userRepository) ListAll(ctx context.Context) ([]domain.User, error) {
	var users []domain.User
	err := r.db.WithContext(ctx).Order("created_at ASC").Find(&users).Error
	return users, err
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *userRepository) UpdatePin(ctx context.Context, userID, newPinHash string) error {
	return r.db.WithContext(ctx).Model(&domain.User{}).Where("id = ?", userID).Update("pin_hash", newPinHash).Error
}

