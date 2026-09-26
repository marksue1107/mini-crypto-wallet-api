//go:build e2e

package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUserLookup_Success_DoesNotLeakEmail is docs/BACKEND_PREP_PLAN.md batch
// 2's required e2e coverage for lookup not leaking email.
func TestUserLookup_Success_DoesNotLeakEmail(t *testing.T) {
	_, _, usernameTarget := registerAndLogin(t, "lookup_target", "password123")
	tokenCaller, _, _ := registerAndLogin(t, "lookup_caller", "password123")

	r := do(t, http.MethodGet, "/users/lookup?username="+usernameTarget, nil, authHeader(tokenCaller))
	require.Equal(t, http.StatusOK, r.Status, string(r.Body))

	assert.False(t, strings.Contains(string(r.Body), "email"), "lookup response must not contain an email field: %s", r.Body)

	var body struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}
	r.decode(t, &body)
	assert.Equal(t, usernameTarget, body.Username)
	assert.NotZero(t, body.ID)
}

func TestUserLookup_Fail_NotFound(t *testing.T) {
	token, _, _ := registerAndLogin(t, "lookup_caller_2", "password123")

	r := do(t, http.MethodGet, "/users/lookup?username=this_user_does_not_exist", nil, authHeader(token))
	require.Equal(t, http.StatusNotFound, r.Status, string(r.Body))
	assert.Equal(t, "USER_NOT_FOUND", r.errBody(t).Code)
}

func TestUserLookup_Fail_NoToken(t *testing.T) {
	r := do(t, http.MethodGet, "/users/lookup?username=anyone", nil, nil)
	require.Equal(t, http.StatusUnauthorized, r.Status, string(r.Body))
	assert.Equal(t, "UNAUTHORIZED", r.errBody(t).Code)
}
