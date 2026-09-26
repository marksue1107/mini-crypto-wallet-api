//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorResponseShape_Consistency spot-checks several different error
// scenarios, across different handlers/middleware layers, and asserts
// every single one comes back with the same {error, code, message} shape
// with a non-empty, specific code - not a mix of ad hoc shapes.
func TestErrorResponseShape_Consistency(t *testing.T) {
	token, id, username := registerAndLogin(t, "errshape", "password123")
	_, otherID, _ := registerAndLogin(t, "errshape_other", "password123")

	cases := []struct {
		name       string
		method     string
		path       string
		body       any
		headers    map[string]string
		wantStatus int
		wantCode   string
	}{
		{"invalid currency id", http.MethodGet, "/currencies/not-a-number", nil, nil, http.StatusBadRequest, "INVALID_REQUEST"},
		{"currency not found", http.MethodGet, "/currencies/999999", nil, nil, http.StatusNotFound, "NOT_FOUND"},
		{"missing auth header", http.MethodGet, fmt.Sprintf("/wallet/%d", id), nil, nil, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"wrong-user wallet access", http.MethodGet, fmt.Sprintf("/wallet/%d", otherID), nil, authHeader(token), http.StatusForbidden, "FORBIDDEN"},
		{"duplicate username", http.MethodPost, "/users", map[string]string{"username": username, "email": "x-" + username + "@example.com", "password": "password123"}, nil, http.StatusConflict, "USER_ALREADY_EXISTS"},
		{"transaction not found", http.MethodGet, "/tx/0000000000000000000000000000000000000000000000000000000000000000", nil, nil, http.StatusNotFound, "TRANSACTION_NOT_FOUND"},
		{"missing required fields (validator middleware)", http.MethodPost, "/users", nil, nil, http.StatusBadRequest, "INVALID_REQUEST"},
	}

	checkShape := func(t *testing.T, resp httpResult, wantStatus int, wantCode string) {
		t.Helper()
		require.Equalf(t, wantStatus, resp.Status, "body=%s", resp.Body)
		eb := resp.errBody(t)
		assert.NotEmpty(t, eb.Error, "error field must be present")
		assert.NotEmpty(t, eb.Message, "message field must be present")
		assert.Equal(t, wantCode, eb.Code)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checkShape(t, do(t, tc.method, tc.path, tc.body, tc.headers), tc.wantStatus, tc.wantCode)
		})
	}

	// Login goes through the retrying login() helper (not the generic
	// table above) because /auth/login has a much stricter rate limiter
	// than the rest of the API - see loginRaw's doc comment.
	t.Run("invalid login credentials", func(t *testing.T) {
		checkShape(t, login(t, username, "wrong", nil), http.StatusUnauthorized, "INVALID_CREDENTIALS")
	})
}
