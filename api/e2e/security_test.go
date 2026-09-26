//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTxByHash_PublicAccess verifies /tx/{hash} works without any
// Authorization header (intentional design - see docs/AUDIT.md S6), by
// making a real transfer and then looking its hash up with no token.
func TestTxByHash_PublicAccess(t *testing.T) {
	token, idA, _ := registerAndLogin(t, "pubtx_a", "password123")
	_, idB, _ := registerAndLogin(t, "pubtx_b", "password123")

	xfer := do(t, http.MethodPost, "/wallet/transfer", transferReq(idA, idB, defaultCurrencyID, "1"), authHeader(token))
	require.Equal(t, http.StatusOK, xfer.Status, string(xfer.Body))

	txList := do(t, http.MethodGet, fmt.Sprintf("/transactions/%d?page=1&page_size=1", idA), nil, authHeader(token))
	require.Equal(t, http.StatusOK, txList.Status)
	var listBody struct {
		Data []struct {
			Hash string `json:"hash"`
		} `json:"data"`
	}
	txList.decode(t, &listBody)
	require.NotEmpty(t, listBody.Data)
	hash := listBody.Data[0].Hash
	require.NotEmpty(t, hash)

	// No Authorization header at all.
	pub := do(t, http.MethodGet, "/tx/"+hash, nil, nil)
	require.Equal(t, http.StatusOK, pub.Status, string(pub.Body))
}

// TestRateLimit_Login verifies /auth/login gets rate limited past its
// configured burst, and that a spoofed X-Forwarded-For header cannot be
// used to bypass it (SetTrustedProxies defaults to trusting no proxy).
func TestRateLimit_Login(t *testing.T) {
	username := uniqueUsername("ratelimited")
	do(t, http.MethodPost, "/users", map[string]string{
		"username": username,
		"email":    username + "@example.com",
		"password": "password123",
	}, nil)

	loginBody := map[string]string{"username": username, "password": "wrong-password-on-purpose"}

	sawTooManyRequests := false
	for i := 0; i < 30; i++ {
		resp := do(t, http.MethodPost, "/auth/login", loginBody, nil)
		if resp.Status == http.StatusTooManyRequests {
			sawTooManyRequests = true
			break
		}
		require.Equal(t, http.StatusUnauthorized, resp.Status, "expected either 401 (wrong password) or 429 (rate limited), got body=%s", resp.Body)
	}
	require.True(t, sawTooManyRequests, "expected /auth/login to eventually return 429 after enough rapid requests")

	// Same client, but now claiming a different origin IP via a spoofed
	// X-Forwarded-For. Since TRUSTED_PROXIES is not set, this must be
	// ignored and the caller must still be rate limited.
	spoofed := do(t, http.MethodPost, "/auth/login", loginBody, map[string]string{"X-Forwarded-For": "1.2.3.4"})
	assert.Equal(t, http.StatusTooManyRequests, spoofed.Status, "spoofed X-Forwarded-For must not bypass rate limiting")

	// Give the token bucket a moment to partially refill so later tests
	// hitting /auth/login (e.g. TestLogin_WrongPassword...) aren't
	// permanently starved by this test's burst.
	time.Sleep(2 * time.Second)
}

// TestCORS verifies preflight requests: a whitelisted origin gets an
// Access-Control-Allow-Origin echoing that exact origin, and a
// non-whitelisted origin never gets a wildcard or its own origin reflected
// back. Requires CORS_ALLOWED_ORIGINS to include https://app.example.com
// (see .env.example / e2e setup instructions).
func TestCORS(t *testing.T) {
	preflight := func(origin string) http.Header {
		t.Helper()
		req, err := http.NewRequest(http.MethodOptions, baseURL()+"/health", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		return resp.Header
	}

	allowed := preflight("https://app.example.com")
	assert.Equal(t, "https://app.example.com", allowed.Get("Access-Control-Allow-Origin"),
		"expected the whitelisted origin to be echoed back - if this fails, confirm CORS_ALLOWED_ORIGINS=https://app.example.com is set for the running api container")

	blocked := preflight("https://evil.example.com")
	got := blocked.Get("Access-Control-Allow-Origin")
	assert.NotEqual(t, "*", got)
	assert.NotEqual(t, "https://evil.example.com", got)
}
