package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mini-crypto-wallet-api/internal/auth"
	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/middleware"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLookupHandler(t *testing.T) *UserHandler {
	t.Helper()
	db := test.SetupTestDB()
	t.Cleanup(func() { test.CleanupTestDB(db) })
	test.CreateTestCurrency(db, "USDT")

	userService := services.NewUserService(repositories.NewUserRepository(), repositories.NewWalletRepository(), repositories.NewCurrencyRepository())
	jwtManager := auth.NewJWTManager("test-secret-at-least-32-characters-long", time.Hour)
	return NewUserHandler(userService, jwtManager)
}

func lookupRouter(h *UserHandler, authedAsUserID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/users/lookup", fakeAuth(authedAsUserID), h.LookupUser)
	return r
}

// TestLookupUser_Success_ReturnsOnlyIDAndUsername is the regression test
// for docs/BACKEND_PREP_PLAN.md 2.3: the response must never leak email or
// any other field beyond id/username.
func TestLookupUser_Success_ReturnsOnlyIDAndUsername(t *testing.T) {
	h := newTestLookupHandler(t)
	regR := gin.New()
	gin.SetMode(gin.TestMode)
	regR.POST("/users", h.CreateUser)
	regBody := doCreateUser(regR, map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "password123",
	})
	require.Equal(t, http.StatusOK, regBody.Code, regBody.Body.String())

	r := lookupRouter(h, 1)
	req := httptest.NewRequest(http.MethodGet, "/users/lookup?username=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.NotContains(t, w.Body.String(), "email")
	assert.NotContains(t, w.Body.String(), "alice@example.com")

	var resp struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "alice", resp.Username)
	assert.NotZero(t, resp.ID)
}

func TestLookupUser_Fail_NotFound_ReturnsUserNotFound(t *testing.T) {
	h := newTestLookupHandler(t)
	r := lookupRouter(h, 1)

	req := httptest.NewRequest(http.MethodGet, "/users/lookup?username=nobody", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

	var errResp errorResponseBody
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
	assert.Equal(t, "USER_NOT_FOUND", errResp.Code)
}

func TestLookupUser_Fail_NoToken_Returns401(t *testing.T) {
	h := newTestLookupHandler(t)

	gin.SetMode(gin.TestMode)
	jwtManager := auth.NewJWTManager("test-secret-at-least-32-characters-long", time.Hour)
	r := gin.New()
	r.GET("/users/lookup", middleware.AuthMiddleware(jwtManager), h.LookupUser)

	req := httptest.NewRequest(http.MethodGet, "/users/lookup?username=alice", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
}
