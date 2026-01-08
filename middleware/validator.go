package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ErrorResponse 驗證錯誤響應
type ValidationErrorResponse struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ValidationMiddleware 驗證中間件（可選，因為 Gin 已經內建驗證）
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}

// FormatValidationError 格式化驗證錯誤
func FormatValidationError(err error) []ValidationErrorResponse {
	var errors []ValidationErrorResponse

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errors = append(errors, ValidationErrorResponse{
				Field:   e.Field(),
				Tag:     e.Tag(),
				Value:   e.Param(),
				Message: getErrorMessage(e),
			})
		}
	} else {
		errors = append(errors, ValidationErrorResponse{
			Message: err.Error(),
		})
	}

	return errors
}

// getErrorMessage 獲取錯誤訊息
func getErrorMessage(e validator.FieldError) string {
	switch e.Tag() {
	case "required":
		return e.Field() + " is required"
	case "email":
		return "Invalid email format"
	case "min":
		return e.Field() + " must be at least " + e.Param() + " characters"
	case "max":
		return e.Field() + " must be at most " + e.Param() + " characters"
	default:
		return e.Error()
	}
}

// HandleValidationError 處理驗證錯誤
func HandleValidationError(c *gin.Context, err error) {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		errors := FormatValidationError(validationErrors)
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "validation failed",
			"code":    "VALIDATION_ERROR",
			"details": errors,
		})
		c.Abort()
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
			"code":  "VALIDATION_ERROR",
		})
		c.Abort()
	}
}
