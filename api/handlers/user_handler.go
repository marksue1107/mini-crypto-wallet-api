package handlers

import (
	"errors"
	"mini-crypto-wallet-api/internal/auth"
	apierrors "mini-crypto-wallet-api/internal/errors"
	"mini-crypto-wallet-api/middleware"
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
// @Param user body models.UserCreateRequest true "User info"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Failure 429 {object} models.ErrorResponse
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	// Bind to DTO instead of database model
	var req models.UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleValidationError(c, err)
		return
	}

	// Service creates user and returns the created model
	user, err := h.service.CreateUser(&req)
	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			apierrors.RespondError(c, http.StatusConflict, apierrors.ErrCodeUserAlreadyExists, err)
			return
		}
		apierrors.RespondError(c, http.StatusInternalServerError, apierrors.ErrCodeInternalError, errFailedToCreateUser)
		return
	}

	// Convert model to response DTO (excludes password)
	response := models.ToUserResponse(user)
	c.JSON(http.StatusOK, response)
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
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 429 {object} models.ErrorResponse
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.HandleValidationError(c, err)
		return
	}

	user, err := h.service.Login(req.Username, req.Password)
	if err != nil {
		apierrors.RespondError(c, http.StatusUnauthorized, apierrors.ErrCodeInvalidCredentials, err)
		return
	}

	token, err := h.jwtManager.GenerateToken(user.ID, user.Username)
	if err != nil {
		apierrors.RespondError(c, http.StatusInternalServerError, apierrors.ErrCodeInternalError, errFailedToGenerateToken)
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

// LookupUser 依使用者名稱查詢，僅回傳 id 與 username
//
// @Summary Look up a user by username
// @Description Look up a user's id by exact username match. Returns only id and username (never email) to avoid leaking account details via enumeration.
// @Tags Users
// @Security BearerAuth
// @Produce json
// @Param username query string true "Username to look up"
// @Success 200 {object} models.UserLookupResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 401 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 429 {object} models.ErrorResponse
// @Router /users/lookup [get]
func (h *UserHandler) LookupUser(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		apierrors.RespondError(c, http.StatusBadRequest, apierrors.ErrCodeInvalidRequest, errUsernameRequired)
		return
	}

	user, err := h.service.GetUserByUsername(username)
	if err != nil {
		apierrors.RespondError(c, http.StatusNotFound, apierrors.ErrCodeUserNotFound, errUserNotFound)
		return
	}

	c.JSON(http.StatusOK, models.ToUserLookupResponse(user))
}
