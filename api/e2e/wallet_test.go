//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWallet_HorizontalAccessControl verifies a user can read their own
// wallet but not another user's.
func TestWallet_HorizontalAccessControl(t *testing.T) {
	tokenA, idA, _ := registerAndLogin(t, "wallet_a", "password123")
	_, idB, _ := registerAndLogin(t, "wallet_b", "password123")

	own := do(t, http.MethodGet, fmt.Sprintf("/wallet/%d", idA), nil, authHeader(tokenA))
	require.Equal(t, http.StatusOK, own.Status, string(own.Body))

	other := do(t, http.MethodGet, fmt.Sprintf("/wallet/%d", idB), nil, authHeader(tokenA))
	require.Equal(t, http.StatusForbidden, other.Status, string(other.Body))
	eb := other.errBody(t)
	assert.Equal(t, "FORBIDDEN", eb.Code)

	// No token at all
	noAuth := do(t, http.MethodGet, fmt.Sprintf("/wallet/%d", idA), nil, nil)
	require.Equal(t, http.StatusUnauthorized, noAuth.Status)
	assert.Equal(t, "UNAUTHORIZED", noAuth.errBody(t).Code)
}
