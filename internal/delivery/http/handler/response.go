package handler

import (
	"ice-pos-api-v2/internal/domain"
	"ice-pos-api-v2/internal/usecase"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"ice-pos-api-v2"`
	Version string `json:"version" example:"2.0.0"`
}

// ErrorResponse represents an error payload
type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"คำขอไม่ถูกต้องหรือไม่พบข้อมูล"`
}

// MessageResponse represents a generic success response with message
type MessageResponse struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"ดำเนินการสำเร็จ"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool        `json:"success" example:"true"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// AuthDataResponse represents the login response
type AuthDataResponse struct {
	Success bool                  `json:"success" example:"true"`
	Data    *usecase.AuthResponse `json:"data"`
}

// MeData represents current user info
type MeData struct {
	UserID   string `json:"user_id" example:"usr_01hqxyz123"`
	Username string `json:"username" example:"admin"`
	Role     string `json:"role" example:"ADMIN"`
}

// MeResponse represents /auth/me response
type MeResponse struct {
	Success bool   `json:"success" example:"true"`
	Data    MeData `json:"data"`
}

// UserDataResponse represents a single user response
type UserDataResponse struct {
	Success bool         `json:"success" example:"true"`
	Data    *domain.User `json:"data"`
}

// UserListDataResponse represents a list of users response
type UserListDataResponse struct {
	Success bool          `json:"success" example:"true"`
	Data    []domain.User `json:"data"`
}

// ProductDataResponse represents a single product response
type ProductDataResponse struct {
	Success bool            `json:"success" example:"true"`
	Data    *domain.Product `json:"data"`
}

// ProductListDataResponse represents a paginated list of products
type ProductListDataResponse struct {
	Success bool             `json:"success" example:"true"`
	Data    []domain.Product `json:"data"`
	Total   int64            `json:"total" example:"42"`
}

// CategoryListDataResponse represents a list of categories
type CategoryListDataResponse struct {
	Success bool              `json:"success" example:"true"`
	Data    []domain.Category `json:"data"`
}

// CheckoutData represents sale and receipt generated at checkout
type CheckoutData struct {
	Sale    *domain.Sale    `json:"sale"`
	Receipt *domain.Receipt `json:"receipt"`
}

// CheckoutResponse represents checkout cash response
type CheckoutResponse struct {
	Success bool         `json:"success" example:"true"`
	Data    CheckoutData `json:"data"`
}

// SaleDataResponse represents a single sale response
type SaleDataResponse struct {
	Success bool         `json:"success" example:"true"`
	Data    *domain.Sale `json:"data"`
}

// SaleListDataResponse represents a paginated list of sales
type SaleListDataResponse struct {
	Success bool          `json:"success" example:"true"`
	Data    []domain.Sale `json:"data"`
	Total   int64         `json:"total" example:"100"`
}

// ReceiptDataResponse represents receipt response
type ReceiptDataResponse struct {
	Success bool            `json:"success" example:"true"`
	Data    *domain.Receipt `json:"data"`
}

// DailySummaryResponse represents daily sales summary response
type DailySummaryResponse struct {
	Success bool                      `json:"success" example:"true"`
	Data    *domain.DailySalesSummary `json:"data"`
}

// BestSellersResponse represents top best-selling products response
type BestSellersResponse struct {
	Success bool                    `json:"success" example:"true"`
	Data    []domain.BestSellerItem `json:"data"`
}

// SyncChangesResponseWrapper represents pull sync response
type SyncChangesResponseWrapper struct {
	Success bool                        `json:"success" example:"true"`
	Data    *domain.SyncChangesResponse `json:"data"`
}

// SyncBatchResponseWrapper represents push sync batch response
type SyncBatchResponseWrapper struct {
	Success bool                      `json:"success" example:"true"`
	Data    *domain.SyncBatchResponse `json:"data"`
}
