package domain

import (
	"context"
	"time"
)

const (
	RoleAdmin   = "ADMIN"
	RoleCashier = "CASHIER"
)

type User struct {
	ID           string    `json:"id" gorm:"primaryKey;size:36"`
	Username     string    `json:"username" gorm:"uniqueIndex;size:50;not null"`
	PasswordHash string    `json:"-" gorm:"size:255;not null"`
	PinHash      string    `json:"-" gorm:"size:255"` // for 4-6 digit cashier PIN
	DisplayName  string    `json:"display_name" gorm:"size:100;not null"`
	Role         string    `json:"role" gorm:"size:20;not null;default:'CASHIER'"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UserRepository interface {
	GetByID(ctx context.Context, id string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByPin(ctx context.Context, pin string) (*User, error)
	ListAll(ctx context.Context) ([]User, error)
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	UpdatePin(ctx context.Context, userID, newPinHash string) error
}

type UserUsecase interface {
	GetAllUsers(ctx context.Context) ([]User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateCashier(ctx context.Context, username, password, pin, displayName string) (*User, error)
	ResetPin(ctx context.Context, userID, newPin string) error
}
