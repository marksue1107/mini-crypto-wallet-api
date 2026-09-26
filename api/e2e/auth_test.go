//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegister_Success_DuplicateRejected_ShortPasswordRejected covers
// registration happy path, duplicate-username handling (409 +
// USER_ALREADY_EXISTS), and the min=8 password rule.
func TestRegister_Success_DuplicateRejected_ShortPasswordRejected(t *testing.T) {
	username := uniqueUsername("alice")
	email := username + "@example.com"

	// Success
	reg := do(t, http.MethodPost, "/users", map[string]string{
		"username": username,
		"email":    email,
		"password": "password123",
	}, nil)
	require.Equal(t, http.StatusOK, reg.Status, string(reg.Body))
	var user userResponse
	reg.decode(t, &user)
	assert.Equal(t, username, user.Username)
	assert.NotContains(t, string(reg.Body), "password", "response must never include the password/hash")

	// Duplicate username (different email)
	dup := do(t, http.MethodPost, "/users", map[string]string{
		"username": username,
		"email":    "different-" + email,
		"password": "password123",
	}, nil)
	require.Equal(t, http.StatusConflict, dup.Status, string(dup.Body))
	eb := dup.errBody(t)
	assert.Equal(t, "USER_ALREADY_EXISTS", eb.Code)

	// Password too short (7 chars, min is 8)
	shortPw := do(t, http.MethodPost, "/users", map[string]string{
		"username": uniqueUsername("shortpw"),
		"email":    uniqueUsername("shortpw") + "@example.com",
		"password": "short7c",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, shortPw.Status, string(shortPw.Body))
}

// TestRegister_WalletExistsImmediately is a regression check for the
// user-creation-atomicity fix: a newly registered user must have a wallet
// immediately, not intermittently/never (which would indicate an orphaned
// user from a non-atomic create).
func TestRegister_WalletExistsImmediately(t *testing.T) {
	token, userID, _ := registerAndLogin(t, "walletcheck", "password123")

	w := do(t, http.MethodGet, fmt.Sprintf("/wallet/%d", userID), nil, authHeader(token))
	require.Equal(t, http.StatusOK, w.Status, string(w.Body))

	var wallet walletResponse
	w.decode(t, &wallet)
	assert.Equal(t, userID, wallet.UserID)
	assert.NotEmpty(t, wallet.Balance)
}

// TestLogin_WrongPassword_And_UnknownUser_SameGenericError verifies login
// failures don't leak whether a username exists.
func TestLogin_WrongPassword_And_UnknownUser_SameGenericError(t *testing.T) {
	username := uniqueUsername("bob")
	do(t, http.MethodPost, "/users", map[string]string{
		"username": username,
		"email":    username + "@example.com",
		"password": "password123",
	}, nil)

	wrongPw := login(t, username, "totally-wrong-password", nil)
	require.Equal(t, http.StatusUnauthorized, wrongPw.Status, string(wrongPw.Body))
	wrongPwBody := wrongPw.errBody(t)

	unknownUser := login(t, uniqueUsername("does-not-exist"), "whatever12345", nil)
	require.Equal(t, http.StatusUnauthorized, unknownUser.Status, string(unknownUser.Body))
	unknownUserBody := unknownUser.errBody(t)

	assert.Equal(t, wrongPwBody.Code, unknownUserBody.Code, "wrong password vs unknown user must return the same error code")
	assert.Equal(t, wrongPwBody.Message, unknownUserBody.Message, "wrong password vs unknown user must return the same message (no username enumeration)")
	assert.Equal(t, "INVALID_CREDENTIALS", wrongPwBody.Code)
}
