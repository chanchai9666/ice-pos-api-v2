package domain

import "fmt"

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return fmt.Sprintf("Error %d: %s", e.Code, e.Message)
}

func NewAppError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Common error codes
var (
	ErrNotFound       = NewAppError(404, "Record not found")
	ErrUnauthorized   = NewAppError(401, "Unauthorized")
	ErrForbidden      = NewAppError(403, "Forbidden")
	ErrBadRequest     = NewAppError(400, "Bad request")
	ErrInternalServer = NewAppError(500, "Internal server error")
)
