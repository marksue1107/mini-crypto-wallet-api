package middleware

import (
	"errors"
	"net/http"
	"strings"

	"mini-crypto-wallet-api/internal/auth"
	apierrors "mini-crypto-wallet-api/internal/errors"

	"github.com/gin-gonic/gin"
)

var (
	errAuthHeaderRequired = errors.New("authorization header required")
	errAuthHeaderFormat   = errors.New("invalid authorization header format")
	errUnauthorized       = errors.New("unauthorized")
	errForbidden          = errors.New("forbidden: cannot access other user's resources")
)

func AuthMiddleware(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			apierrors.RespondError(c, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, errAuthHeaderRequired)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			apierrors.RespondError(c, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, errAuthHeaderFormat)
			return
		}

		token := parts[1]
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			apierrors.RespondError(c, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, err)
			return
		}

		// 將用戶信息存儲到 context 中
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)

		c.Next()
	}
}

func RequireUserID(c *gin.Context, targetUserID uint) bool {
	userID, exists := c.Get("user_id")
	if !exists {
		apierrors.RespondError(c, http.StatusUnauthorized, apierrors.ErrCodeUnauthorized, errUnauthorized)
		return false
	}

	uid, ok := userID.(uint)
	if !ok || uid != targetUserID {
		apierrors.RespondError(c, http.StatusForbidden, apierrors.ErrCodeForbidden, errForbidden)
		return false
	}

	return true
}
