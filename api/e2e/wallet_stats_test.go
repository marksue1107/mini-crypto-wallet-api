//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type walletStatsResponse struct {
	Window           string `json:"window"`
	TransactionCount int64  `json:"transaction_count"`
	TotalSent        string `json:"total_sent"`
	TotalReceived    string `json:"total_received"`
	BalanceChange    string `json:"balance_change"`
}

func getStats(t *testing.T, token string, userID uint) (walletStatsResponse, int) {
	t.Helper()
	r := do(t, http.MethodGet, fmt.Sprintf("/wallet/%d/stats", userID), nil, authHeader(token))
	var s walletStatsResponse
	if r.Status == http.StatusOK {
		r.decode(t, &s)
	}
	return s, r.Status
}

// TestWalletStats_MixedSentReceived_CorrectValues is docs/BACKEND_PREP_PLAN.md
// batch 2's required e2e coverage for stats correctness.
func TestWalletStats_MixedSentReceived_CorrectValues(t *testing.T) {
	tokenA, idA, _ := registerAndLogin(t, "stats_a", "password123")
	tokenB, idB, _ := registerAndLogin(t, "stats_b", "password123")

	resp := do(t, http.MethodPost, "/wallet/transfer", transferReq(idA, idB, defaultCurrencyID, "100"), authHeader(tokenA))
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	resp = do(t, http.MethodPost, "/wallet/transfer", transferReq(idB, idA, defaultCurrencyID, "30"), authHeader(tokenB))
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	stats, status := getStats(t, tokenA, idA)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "24h", stats.Window)
	assert.Equal(t, int64(2), stats.TransactionCount)

	sent, err := decimal.NewFromString(stats.TotalSent)
	require.NoError(t, err)
	received, err := decimal.NewFromString(stats.TotalReceived)
	require.NoError(t, err)
	change, err := decimal.NewFromString(stats.BalanceChange)
	require.NoError(t, err)
	assert.True(t, sent.Equal(decimal.NewFromInt(100)), "total_sent=%s", stats.TotalSent)
	assert.True(t, received.Equal(decimal.NewFromInt(30)), "total_received=%s", stats.TotalReceived)
	assert.True(t, change.Equal(decimal.NewFromInt(-70)), "balance_change=%s", stats.BalanceChange)
}

// TestWalletStats_HorizontalAccessControl mirrors TestWallet_HorizontalAccessControl.
func TestWalletStats_HorizontalAccessControl(t *testing.T) {
	tokenA, idA, _ := registerAndLogin(t, "stats_access_a", "password123")
	_, idB, _ := registerAndLogin(t, "stats_access_b", "password123")

	_, status := getStats(t, tokenA, idB)
	assert.Equal(t, http.StatusForbidden, status)

	_, status = getStats(t, tokenA, idA)
	assert.Equal(t, http.StatusOK, status)
}

// TestWalletStats_UnsupportedWindow verifies anything other than the only
// supported window value ("24h") is rejected.
func TestWalletStats_UnsupportedWindow(t *testing.T) {
	token, id, _ := registerAndLogin(t, "stats_window", "password123")

	r := do(t, http.MethodGet, fmt.Sprintf("/wallet/%d/stats?window=7d", id), nil, authHeader(token))
	require.Equal(t, http.StatusBadRequest, r.Status, string(r.Body))
	assert.Equal(t, "INVALID_REQUEST", r.errBody(t).Code)
}
