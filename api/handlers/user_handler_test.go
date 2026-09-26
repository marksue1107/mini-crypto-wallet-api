package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"mini-crypto-wallet-api/internal/auth"
	"mini-crypto-wallet-api/internal/test"
	"mini-crypto-wallet-api/repositories"
	"mini-crypto-wallet-api/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUserHandler(t *testing.T) (*gin.Engine, *UserHandler) {
	t.Helper()
	db := test.SetupTestDB()
	t.Cleanup(func() { test.CleanupTestDB(db) })
	test.CreateTestCurrency(db, "USDT")

	userRepo := repositories.NewUserRepository()
	walletRepo := repositories.NewWalletRepository()
	currencyRepo := repositories.NewCurrencyRepository()
	userService := services.NewUserService(userRepo, walletRepo, currencyRepo)
	jwtManager := auth.NewJWTManager("test-secret-at-least-32-characters-long", 0)

	h := NewUserHandler(userService, jwtManager)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/users", h.CreateUser)
	return r, h
}

func doCreateUser(r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestCreateUser_Success_ReturnsUserWithoutPassword checks the happy path
// returns 200 and never leaks the password/hash in the response.
func TestCreateUser_Success_ReturnsUserWithoutPassword(t *testing.T) {
	r, _ := newTestUserHandler(t)

	w := doCreateUser(r, map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "password123",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "password")
	assert.NotContains(t, w.Body.String(), "password123")
}

// TestCreateUser_Fail_PasswordTooShort is a regression test for
// docs/AUDIT.md S11: the minimum password length was raised from 6 to 8.
func TestCreateUser_Fail_PasswordTooShort(t *testing.T) {
	r, _ := newTestUserHandler(t)

	w := doCreateUser(r, map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "short7c", // 7 characters: rejected under the new min=8 rule
	})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestCreateUser_Fail_DuplicateUsername_Returns409 is a regression test for
// docs/AUDIT.md S7: duplicate signups used to come back as a generic 500.
func TestCreateUser_Fail_DuplicateUsername_Returns409(t *testing.T) {
	r, _ := newTestUserHandler(t)

	w1 := doCreateUser(r, map[string]any{
		"username": "alice",
		"email":    "alice@example.com",
		"password": "password123",
	})
	require.Equal(t, http.StatusOK, w1.Code)

	w2 := doCreateUser(r, map[string]any{
		"username": "alice",
		"email":    "someone-else@example.com",
		"password": "password123",
	})

	require.Equal(t, http.StatusConflict, w2.Code)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &body))
	assert.Equal(t, "USER_ALREADY_EXISTS", body["code"])
}
