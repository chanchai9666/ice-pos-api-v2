package validator

import "github.com/go-playground/validator/v10"

type CustomValidator struct {
	validate *validator.Validate
}

func InitValidator() *CustomValidator {
	return &CustomValidator{
		validate: validator.New(),
	}
}

func (cv *CustomValidator) Validate(i interface{}) error {
	return cv.validate.Struct(i)
}
