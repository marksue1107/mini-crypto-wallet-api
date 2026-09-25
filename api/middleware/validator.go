package middleware

import (
	"net/http"

	apierrors "mini-crypto-wallet-api/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// FieldValidationError describes one failed validation rule on one field.
type FieldValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// FormatValidationError turns go-playground/validator's error into a
// frontend-friendly, per-field list.
func FormatValidationError(err error) []FieldValidationError {
	var out []FieldValidationError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			out = append(out, FieldValidationError{
				Field:   e.Field(),
				Tag:     e.Tag(),
				Value:   e.Param(),
				Message: getErrorMessage(e),
			})
		}
	} else {
		out = append(out, FieldValidationError{Message: err.Error()})
	}

	return out
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

// HandleValidationError responds to a c.ShouldBindJSON/ShouldBindQuery
// error. If it's a go-playground/validator error, the response includes a
// per-field "details" array on top of the same base {error, code, message}
// shape every other error response uses (see docs/AUDIT.md S8) - anything
// else (malformed JSON, wrong type, etc.) falls back to the plain shape via
// apierrors.RespondError.
func HandleValidationError(c *gin.Context, err error) {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		apierrors.RespondError(c, http.StatusBadRequest, apierrors.ErrCodeInvalidRequest, err)
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "validation failed",
		"code":    apierrors.ErrCodeInvalidRequest,
		"message": "validation failed",
		"details": FormatValidationError(validationErrors),
	})
	c.Abort()
}
