//go:build e2e

// Package e2e drives a real, already-running instance of this API over
// HTTP (see docker-compose.yml) - it does not import any of the API's own
// internal packages for logic, only for convenient JSON decoding of
// well-known response shapes. Run with:
//
//	cd api && go test -tags=e2e ./e2e/... -v
//
// Requires the stack to already be up (`docker compose up -d --build`) and
// E2E_BASE_URL (default http://localhost:8080) to point at it.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func baseURL() string {
	if v := os.Getenv("E2E_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

type errBody struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type httpResult struct {
	Status int
	Body   []byte
	Header http.Header
}

func (r httpResult) decode(t *testing.T, v any) {
	t.Helper()
	require.NoErrorf(t, json.Unmarshal(r.Body, v), "response body must be valid JSON: %s", r.Body)
}

func (r httpResult) errBody(t *testing.T) errBody {
	t.Helper()
	var e errBody
	r.decode(t, &e)
	return e
}

// doRaw issues a raw HTTP request against the running API and returns any
// error instead of failing a test - safe to call from a goroutine other
// than the one running the test function, unlike do() below. body, if
// non-nil, is JSON-marshaled. Extra headers are applied after
// Content-Type.
func doRaw(method, path string, body any, headers map[string]string) (httpResult, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return httpResult{}, err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL()+path, reader)
	if err != nil {
		return httpResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return httpResult{}, err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return httpResult{}, err
	}

	return httpResult{Status: resp.StatusCode, Body: b, Header: resp.Header}, nil
}

// do is doRaw, but fails the test immediately on any transport-level error.
// Per Go's testing rules, t.FailNow() (which require.* calls under the
// hood) must only be called from the goroutine running the test function -
// do NOT call this from a spawned goroutine; use doRaw there instead and
// assert on the result back in the main test goroutine.
func do(t *testing.T, method, path string, body any, headers map[string]string) httpResult {
	t.Helper()
	res, err := doRaw(method, path, body, headers)
	require.NoError(t, err)
	return res
}

func authHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// loginRaw performs POST /auth/login, transparently retrying on 429. The
// login endpoint has a deliberately strict limiter (10 req/min, burst 5 -
// see router.go), and this suite legitimately registers/logs in many fresh
// users across many independent tests in quick succession, which exceeds
// that burst well before real wall-clock time lets it refill (1 token per
// 6s). Retrying with a backoff matching that refill rate is what a
// well-behaved real client would do, and avoids the whole suite going
// flaky purely from cross-test rate-limit exhaustion - this is a test
// suite concern, not a product bug (TestRateLimit_Login below verifies
// the limiter itself, and deliberately does NOT use this helper so it can
// observe a raw, unretried 429).
func loginRaw(username, password string, headers map[string]string) (httpResult, error) {
	const maxAttempts = 12
	const backoff = 6500 * time.Millisecond

	body := map[string]string{"username": username, "password": password}
	var res httpResult
	var err error
	for i := 0; i < maxAttempts; i++ {
		res, err = doRaw(http.MethodPost, "/auth/login", body, headers)
		if err != nil || res.Status != http.StatusTooManyRequests {
			return res, err
		}
		time.Sleep(backoff)
	}
	return res, err
}

// login is loginRaw, but fails the test on a transport-level error. Must
// only be called from the main test goroutine (see do() above).
func login(t *testing.T, username, password string, headers map[string]string) httpResult {
	t.Helper()
	res, err := loginRaw(username, password, headers)
	require.NoError(t, err)
	return res
}

// uniqueUsername returns a username that won't collide with a previous e2e
// run against the same (persistent) database.
func uniqueUsername(prefix string) string {
	return fmt.Sprintf("%s_%d_%d", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
}

type userResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type loginResponse struct {
	Token    string `json:"token"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}

type walletResponse struct {
	ID         uint   `json:"id"`
	UserID     uint   `json:"user_id"`
	CurrencyID uint   `json:"currency_id"`
	Balance    string `json:"balance"`
}

// registerAndLogin creates a fresh user (unique username derived from
// prefix) with the given password and logs in, returning the token and
// user id. Fails the test on any non-200.
func registerAndLogin(t *testing.T, prefix, password string) (token string, userID uint, username string) {
	t.Helper()
	username = uniqueUsername(prefix)
	email := username + "@example.com"

	regResp := do(t, http.MethodPost, "/users", map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}, nil)
	require.Equalf(t, http.StatusOK, regResp.Status, "register %s failed: %s", username, regResp.Body)

	var user userResponse
	regResp.decode(t, &user)

	loginResp := login(t, username, password, nil)
	require.Equalf(t, http.StatusOK, loginResp.Status, "login %s failed: %s", username, loginResp.Body)

	var login loginResponse
	loginResp.decode(t, &login)

	return login.Token, login.UserID, username
}
