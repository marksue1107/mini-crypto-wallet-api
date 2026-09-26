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

// defaultCurrencyID is the auto-seeded default currency (USDT) - see
// db_conn.seedDefaultCurrency(). Every fresh database has exactly this one
// currency as the first row.
const defaultCurrencyID = 1

func transferReq(from, to, currencyID uint, amount string) map[string]any {
	return map[string]any{
		"from_user_id": from,
		"to_user_id":   to,
		"currency_id":  currencyID,
		"amount":       amount,
	}
}

func getWallet(t *testing.T, token string, userID uint) walletResponse {
	t.Helper()
	r := do(t, http.MethodGet, fmt.Sprintf("/wallet/%d", userID), nil, authHeader(token))
	require.Equal(t, http.StatusOK, r.Status, string(r.Body))
	var w walletResponse
	r.decode(t, &w)
	return w
}

// TestTransfer_Success_BalanceAndHistory covers the golden path: transfer
// succeeds, both balances move by the right amount, and the sender's
// transaction history shows it with valid pagination metadata.
func TestTransfer_Success_BalanceAndHistory(t *testing.T) {
	tokenA, idA, _ := registerAndLogin(t, "xfer_a", "password123")
	_, idB, _ := registerAndLogin(t, "xfer_b", "password123")

	beforeA := getWallet(t, tokenA, idA)

	resp := do(t, http.MethodPost, "/wallet/transfer", transferReq(idA, idB, defaultCurrencyID, "100"), authHeader(tokenA))
	require.Equal(t, http.StatusOK, resp.Status, string(resp.Body))

	var transferBody struct {
		FromUserID uint   `json:"from_user_id"`
		ToUserID   uint   `json:"to_user_id"`
		CurrencyID uint   `json:"currency_id"`
		Amount     string `json:"amount"`
		Hash       string `json:"hash"`
		Status     string `json:"status"`
	}
	resp.decode(t, &transferBody)
	assert.NotEmpty(t, transferBody.Hash, "transfer response must include the created transaction's hash")
	assert.Equal(t, defaultCurrencyID, transferBody.CurrencyID)
	assert.Equal(t, "completed", transferBody.Status)

	afterA := getWallet(t, tokenA, idA)
	before, err := decimal.NewFromString(beforeA.Balance)
	require.NoError(t, err)
	after, err := decimal.NewFromString(afterA.Balance)
	require.NoError(t, err)
	assert.True(t, before.Sub(after).Equal(decimal.NewFromInt(100)), "sender balance should drop by exactly 100 (before=%s after=%s)", before, after)

	txList := do(t, http.MethodGet, fmt.Sprintf("/transactions/%d?page=1&page_size=10", idA), nil, authHeader(tokenA))
	require.Equal(t, http.StatusOK, txList.Status, string(txList.Body))

	var listBody struct {
		Data []struct {
			FromUserID uint   `json:"from_user_id"`
			ToUserID   uint   `json:"to_user_id"`
			Amount     string `json:"amount"`
			Hash       string `json:"hash"`
			Status     string `json:"status"`
		} `json:"data"`
		Pagination struct {
			Page       int   `json:"page"`
			PageSize   int   `json:"page_size"`
			Total      int64 `json:"total"`
			TotalPages int   `json:"total_pages"`
		} `json:"pagination"`
	}
	txList.decode(t, &listBody)

	require.NotEmpty(t, listBody.Data)
	found := false
	for _, tx := range listBody.Data {
		if tx.FromUserID == idA && tx.ToUserID == idB && tx.Amount == "100" {
			found = true
			assert.Equal(t, "completed", tx.Status)
			assert.Equal(t, transferBody.Hash, tx.Hash, "history hash must match the hash returned by the transfer response")
		}
	}
	assert.True(t, found, "expected to find the just-made transfer in alice's transaction history")
	assert.Equal(t, 1, listBody.Pagination.Page)
	assert.Equal(t, 10, listBody.Pagination.PageSize)
	assert.GreaterOrEqual(t, listBody.Pagination.Total, int64(1))
}

// TestTransfer_SameAccount verifies transferring to yourself is rejected.
func TestTransfer_SameAccount(t *testing.T) {
	token, id, _ := registerAndLogin(t, "self_xfer", "password123")

	resp := do(t, http.MethodPost, "/wallet/transfer", transferReq(id, id, defaultCurrencyID, "1"), authHeader(token))
	require.Equal(t, http.StatusBadRequest, resp.Status, string(resp.Body))
	assert.Equal(t, "SAME_ACCOUNT_TRANSFER", resp.errBody(t).Code)
}

// TestTransfer_ValidationFailures covers insufficient balance, zero/negative
// amount, too many decimal places, and exceeding the max transfer amount -
// each must fail with a distinct, stable error code.
func TestTransfer_ValidationFailures(t *testing.T) {
	token, id, _ := registerAndLogin(t, "validation", "password123")
	_, otherID, _ := registerAndLogin(t, "validation_other", "password123")

	cases := []struct {
		name       string
		amount     string
		wantStatus int
		wantCode   string
	}{
		{"insufficient balance", "999999", http.StatusBadRequest, "INSUFFICIENT_BALANCE"},
		{"zero amount", "0", http.StatusBadRequest, "AMOUNT_NOT_POSITIVE"},
		{"negative amount", "-50", http.StatusBadRequest, "AMOUNT_NOT_POSITIVE"},
		{"too many decimal places", "1.123456789", http.StatusBadRequest, "INVALID_DECIMALS"},
		{"exceeds max transfer amount", "2000000", http.StatusBadRequest, "AMOUNT_EXCEEDS_LIMIT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := do(t, http.MethodPost, "/wallet/transfer", transferReq(id, otherID, defaultCurrencyID, tc.amount), authHeader(token))
			require.Equalf(t, tc.wantStatus, resp.Status, "body=%s", resp.Body)
			assert.Equal(t, tc.wantCode, resp.errBody(t).Code)
		})
	}
}

// TestTransfer_RequiresOwnAccountAsSender verifies a user cannot initiate a
// transfer "from" someone else's account.
func TestTransfer_RequiresOwnAccountAsSender(t *testing.T) {
	tokenA, _, _ := registerAndLogin(t, "notmine_a", "password123")
	_, idB, _ := registerAndLogin(t, "notmine_b", "password123")
	_, idC, _ := registerAndLogin(t, "notmine_c", "password123")

	// alice's token, but claiming to send FROM bob's account
	resp := do(t, http.MethodPost, "/wallet/transfer", transferReq(idB, idC, defaultCurrencyID, "1"), authHeader(tokenA))
	require.Equal(t, http.StatusForbidden, resp.Status, string(resp.Body))
	assert.Equal(t, "FORBIDDEN", resp.errBody(t).Code)
}
