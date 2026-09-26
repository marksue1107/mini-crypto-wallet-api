package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"mini-crypto-wallet-api/internal/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransfer_Fail_InvalidAmount_ReturnsDistinctCodes is the regression
// test for docs/BACKEND_PREP_PLAN.md 1.2: what used to be a single
// INVALID_AMOUNT code for three different validation failures must now be
// three distinct, stable codes.
func TestTransfer_Fail_InvalidAmount_ReturnsDistinctCodes(t *testing.T) {
	h, db, _ := newTestTransactionHandler(t)
	currency := test.CreateTestCurrency(db, "USDT") // Decimals: 8
	alice := test.CreateTestUser(db, "alice")
	bob := test.CreateTestUser(db, "bob")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)
	test.CreateTestWallet(db, bob.ID, currency.ID, 0)

	r := transferRouter(h, alice.ID)

	cases := []struct {
		name     string
		amount   string
		wantCode string
	}{
		{"zero amount", "0", "AMOUNT_NOT_POSITIVE"},
		{"negative amount", "-50", "AMOUNT_NOT_POSITIVE"},
		{"too many decimal places", "1.123456789", "INVALID_DECIMALS"},
		{"exceeds max transfer amount", "2000000", "AMOUNT_EXCEEDS_LIMIT"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doTransfer(r, alice.ID, bob.ID, currency.ID, tc.amount)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

			var errResp errorResponseBody
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
			assert.Equal(t, tc.wantCode, errResp.Code)
			assert.NotEqual(t, "INVALID_AMOUNT", errResp.Code)
		})
	}
}

// TestTransfer_Fail_WalletNotFound_ReturnsDistinctCodes is the regression
// test for docs/BACKEND_PREP_PLAN.md 1.3: "my own wallet doesn't exist for
// this currency" and "the recipient doesn't exist / has no wallet for this
// currency" used to share the same WALLET_NOT_FOUND code.
func TestTransfer_Fail_WalletNotFound_ReturnsDistinctCodes(t *testing.T) {
	h, db, _ := newTestTransactionHandler(t)
	currency := test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")
	test.CreateTestWallet(db, alice.ID, currency.ID, 1000)

	t.Run("sender has no wallet for this currency", func(t *testing.T) {
		bob := test.CreateTestUser(db, "bob_recipient_ok")
		test.CreateTestWallet(db, bob.ID, currency.ID, 0)

		noWalletUser := test.CreateTestUser(db, "no_wallet_sender")
		r := transferRouter(h, noWalletUser.ID)
		w := doTransfer(r, noWalletUser.ID, bob.ID, currency.ID, "10")
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

		var errResp errorResponseBody
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
		assert.Equal(t, "SENDER_WALLET_NOT_FOUND", errResp.Code)
	})

	t.Run("recipient does not exist", func(t *testing.T) {
		r := transferRouter(h, alice.ID)
		w := doTransfer(r, alice.ID, 999999, currency.ID, "10")
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())

		var errResp errorResponseBody
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &errResp))
		assert.Equal(t, "RECIPIENT_NOT_FOUND", errResp.Code)
	})
}
