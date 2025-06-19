package handlers

import (
	"github.com/gin-gonic/gin"
	"mini-crypto-wallet-api/models"
	"mini-crypto-wallet-api/services"
	"net/http"
)

type UserHandler struct {
	service *services.UserService
}

func NewUserHandler(service *services.UserService) *UserHandler {
	return &UserHandler{service}
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
