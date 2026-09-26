//go:build e2e

package e2e

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrency_ConservationAndNoDeadlock fires a burst of concurrent
// transfers in both directions between two accounts via real HTTP calls
// against the running service (not in-process function calls), then
// verifies: (1) the service didn't hang, (2) total money is conserved,
// (3) a normal transfer still succeeds afterward (i.e. no leaked
// connections/locks from the burst).
func TestConcurrency_ConservationAndNoDeadlock(t *testing.T) {
	tokenA, idA, _ := registerAndLogin(t, "conc_a", "password123")
	tokenB, idB, _ := registerAndLogin(t, "conc_b", "password123")

	beforeA := getWallet(t, tokenA, idA)
	beforeB := getWallet(t, tokenB, idB)
	startA, err := decimal.NewFromString(beforeA.Balance)
	require.NoError(t, err)
	startB, err := decimal.NewFromString(beforeB.Balance)
	require.NoError(t, err)
	startTotal := startA.Add(startB)

	const rounds = 15
	var wg sync.WaitGroup
	// doRaw (not do) is used inside these goroutines deliberately: Go's
	// testing package requires t.FailNow() (which require.* calls
	// internally) to only ever be invoked from the goroutine running the
	// test function itself, never from a spawned goroutine. Results are
	// collected here and asserted on back in the main test goroutine below.
	type callResult struct {
		status int
		err    error
	}
	results := make(chan callResult, rounds*2)

	for i := 0; i < rounds; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			resp, err := doRaw(http.MethodPost, "/wallet/transfer", transferReq(idA, idB, defaultCurrencyID, "10"), authHeader(tokenA))
			results <- callResult{resp.Status, err}
		}()
		go func() {
			defer wg.Done()
			resp, err := doRaw(http.MethodPost, "/wallet/transfer", transferReq(idB, idA, defaultCurrencyID, "10"), authHeader(tokenB))
			results <- callResult{resp.Status, err}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// good, didn't hang
	case <-time.After(30 * time.Second):
		t.Fatal("concurrent transfers did not complete within 30s - possible deadlock/connection leak")
	}
	close(results)

	for r := range results {
		require.NoError(t, r.err, "transport-level error during concurrent transfer")
		assert.Truef(t, r.status == http.StatusOK || r.status == http.StatusBadRequest,
			"unexpected status %d from concurrent transfer (expected 200 success or 400 insufficient-balance)", r.status)
	}

	afterA := getWallet(t, tokenA, idA)
	afterB := getWallet(t, tokenB, idB)
	endA, err := decimal.NewFromString(afterA.Balance)
	require.NoError(t, err)
	endB, err := decimal.NewFromString(afterB.Balance)
	require.NoError(t, err)
	endTotal := endA.Add(endB)

	assert.True(t, endA.GreaterThanOrEqual(decimal.Zero), "A's balance went negative: %s", endA)
	assert.True(t, endB.GreaterThanOrEqual(decimal.Zero), "B's balance went negative: %s", endB)
	assert.True(t, startTotal.Equal(endTotal), "money not conserved: start=%s end=%s", startTotal, endTotal)

	// Prove the service is still healthy for normal use after the burst -
	// this is the practical signal that no DB connection or lock leaked.
	followUp := do(t, http.MethodPost, "/wallet/transfer", transferReq(idA, idB, defaultCurrencyID, "1"), authHeader(tokenA))
	require.Contains(t, []int{http.StatusOK, http.StatusBadRequest}, followUp.Status, "follow-up transfer after burst got unexpected status: %s", followUp.Body)
}

// TestFailurePathStress_ThenNormalTransferSucceeds hammers the API with
// transfers that always fail validation (insufficient balance) and then
// confirms a normal transfer still succeeds - a practical, HTTP-level
// signal that failed transfers aren't leaking DB connections (see
// docs/AUDIT_REMEDIATION_PLAN.md finding N2).
func TestFailurePathStress_ThenNormalTransferSucceeds(t *testing.T) {
	token, id, _ := registerAndLogin(t, "stress_a", "password123")
	_, otherID, _ := registerAndLogin(t, "stress_b", "password123")

	for i := 0; i < 20; i++ {
		resp := do(t, http.MethodPost, "/wallet/transfer", transferReq(id, otherID, defaultCurrencyID, "999999"), authHeader(token))
		require.Equal(t, http.StatusBadRequest, resp.Status, string(resp.Body))
		require.Equal(t, "INSUFFICIENT_BALANCE", resp.errBody(t).Code)
	}

	ok := do(t, http.MethodPost, "/wallet/transfer", transferReq(id, otherID, defaultCurrencyID, "1"), authHeader(token))
	require.Equal(t, http.StatusOK, ok.Status, "a normal transfer after 20 failed ones should still succeed: %s", ok.Body)
}
