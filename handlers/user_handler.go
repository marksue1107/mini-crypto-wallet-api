package handlers

import (
	"mini-crypto-wallet-api/internal/auth"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/services"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service    *services.UserService
	jwtManager *auth.JWTManager
}

func NewUserHandler(service *services.UserService, jwtManager *auth.JWTManager) *UserHandler {
	return &UserHandler{
		service:    service,
		jwtManager: jwtManager,
	}
}

// CreateUser 建立使用者，並初始化 Wallet（預設 1000 USDC）
//
// @Summary Create user
// @Description create a new wallet user
// @Tags Users
// @Accept json
// @Produce json
// @Param user body models.User true "User info"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.CreateUser(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user created", "user_id": user.ID})
}

// Login 用戶登入
//
// @Summary User login
// @Description authenticate user and return JWT token
// @Tags Auth
// @Accept json
// @Produce json
// @Param login body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	response := models.LoginResponse{
		Token:     token,
		UserID:    user.ID,
		Username:  user.Username,
		ExpiresIn: int(24 * time.Hour.Seconds()), // 24小時
	}

	c.JSON(http.StatusOK, response)
}
