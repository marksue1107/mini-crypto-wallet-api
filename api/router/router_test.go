package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"mini-crypto-wallet-api/internal/auth"
	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/internal/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupRouter_LoginIsRateLimited is an integration-style regression test
// for docs/AUDIT.md S3: the rate limit middleware existed in this codebase
// but was never attached to any route. This drives requests through the
// actual router (not just the middleware in isolation) to prove
// /auth/login is really rate limited end-to-end.
func TestSetupRouter_LoginIsRateLimited(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	config.Config = &config.AppConfig{JWTSecret: "test-secret-at-least-32-characters-long"}

	r := SetupRouter(nil)

	lastCode := http.StatusOK
	for i := 0; i < 20; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "9.9.9.9:1111"
		r.ServeHTTP(w, req)
		lastCode = w.Code
		if lastCode == http.StatusTooManyRequests {
			break
		}
	}

	require.Equal(t, http.StatusTooManyRequests, lastCode, "expected /auth/login to eventually be rate limited")
}

// TestSetupRouter_FailsFastOnWeakJWTSecret is a regression test for
// docs/AUDIT.md S2: previously an empty/missing JWT secret silently fell
// back to a value hardcoded in this repo's source, instead of refusing to
// start. SetupRouter calls log.Fatal (os.Exit) in that case, which would
// kill this test binary if called in-process, so - the standard Go idiom -
// this re-execs the test binary in a subprocess and asserts it exits
// non-zero.
func TestSetupRouter_FailsFastOnWeakJWTSecret(t *testing.T) {
	if os.Getenv("MINI_WALLET_TEST_SUBPROCESS") == "1" {
		db := test.SetupTestDB()
		defer test.CleanupTestDB(db)
		config.Config = &config.AppConfig{JWTSecret: ""}
		SetupRouter(nil) // expected to log.Fatal before returning
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestSetupRouter_FailsFastOnWeakJWTSecret")
	cmd.Env = append(os.Environ(), "MINI_WALLET_TEST_SUBPROCESS=1")
	output, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	require.ErrorAsf(t, err, &exitErr, "expected SetupRouter to exit non-zero on a missing JWT secret; output:\n%s", output)
	require.NotEqual(t, 0, exitErr.ExitCode())
}

// TestErrorResponses_ConsistentShapeAcrossEndpoints is the acceptance test
// for docs/AUDIT.md S8: every error response, from every handler and
// middleware layer (JSON binding, auth, ownership checks, service-layer
// business errors, not-found lookups), must have the same
// {error, code, message} shape with a non-empty stable "code" - not the
// mix of ad hoc shapes ({"error":...} only, sometimes +"code", sometimes
// +"details") this API had before.
func TestErrorResponses_ConsistentShapeAcrossEndpoints(t *testing.T) {
	const secret = "test-secret-at-least-32-characters-long"

	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)
	test.CreateTestCurrency(db, "USDT")
	alice := test.CreateTestUser(db, "alice")

	config.Config = &config.AppConfig{JWTSecret: secret}
	r := SetupRouter(nil)

	jwtManager := auth.NewJWTManager(secret, time.Hour)
	aliceToken, err := jwtManager.GenerateToken(alice.ID, "alice")
	require.NoError(t, err)

	do := func(method, path, body, bearer string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	type errBody struct {
		Error   string `json:"error"`
		Code    string `json:"code"`
		Message string `json:"message"`
	}

	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		bearer     string
		wantStatus int
		wantCode   string
	}{
		{"invalid currency id (handler param parsing)", http.MethodGet, "/currencies/not-a-number", "", "", http.StatusBadRequest, "INVALID_REQUEST"},
		{"currency not found (handler + service lookup)", http.MethodGet, "/currencies/999999", "", "", http.StatusNotFound, "NOT_FOUND"},
		{"missing auth header (AuthMiddleware)", http.MethodGet, "/wallet/1", "", "", http.StatusUnauthorized, "UNAUTHORIZED"},
		{"wrong-user access (RequireUserID ownership check)", http.MethodGet, "/wallet/999999", "", aliceToken, http.StatusForbidden, "FORBIDDEN"},
		{"invalid login credentials (service-layer business error)", http.MethodPost, "/auth/login", `{"username":"alice","password":"wrong-password"}`, "", http.StatusUnauthorized, "INVALID_CREDENTIALS"},
		{"same-account transfer (service-layer sentinel error)", http.MethodPost, "/wallet/transfer", `{"from_user_id":` + strconv.Itoa(int(alice.ID)) + `,"to_user_id":` + strconv.Itoa(int(alice.ID)) + `,"currency_id":1,"amount":"1"}`, aliceToken, http.StatusBadRequest, "SAME_ACCOUNT_TRANSFER"},
		{"malformed JSON body (validator middleware)", http.MethodPost, "/users", `{"username":`, "", http.StatusBadRequest, "INVALID_REQUEST"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := do(tc.method, tc.path, tc.body, tc.bearer)
			require.Equalf(t, tc.wantStatus, w.Code, "status code for %s %s; body=%s", tc.method, tc.path, w.Body.String())

			var body errBody
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body), "response body must be valid JSON: %s", w.Body.String())

			assert.NotEmpty(t, body.Error, "error field must be present")
			assert.NotEmpty(t, body.Message, "message field must be present")
			assert.Equal(t, tc.wantCode, body.Code, "code field must be the stable error code")
		})
	}
}

// TestSetupRouter_CORS is a regression test for docs/AUDIT.md S5: without
// any CORS configuration, a browser-based frontend on a different origin
// can't call this API at all. It also confirms the safe-by-default
// behavior: CORS is off entirely until CORS_ALLOWED_ORIGINS is set, and
// only listed origins get the allow header - not a wildcard.
func TestSetupRouter_CORS(t *testing.T) {
	db := test.SetupTestDB()
	defer test.CleanupTestDB(db)

	preflight := func(r http.Handler, origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodOptions, "/health", nil)
		req.Header.Set("Origin", origin)
		req.Header.Set("Access-Control-Request-Method", "GET")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("disabled by default", func(t *testing.T) {
		config.Config = &config.AppConfig{JWTSecret: "test-secret-at-least-32-characters-long"}
		r := SetupRouter(nil)

		w := preflight(r, "https://app.example.com")
		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"), "CORS must be off until CORS_ALLOWED_ORIGINS is configured")
	})

	t.Run("allows a configured origin", func(t *testing.T) {
		config.Config = &config.AppConfig{
			JWTSecret:          "test-secret-at-least-32-characters-long",
			CORSAllowedOrigins: "https://app.example.com,https://admin.example.com",
		}
		r := SetupRouter(nil)

		w := preflight(r, "https://app.example.com")
		assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("rejects a non-whitelisted origin, never falls back to wildcard", func(t *testing.T) {
		config.Config = &config.AppConfig{
			JWTSecret:          "test-secret-at-least-32-characters-long",
			CORSAllowedOrigins: "https://app.example.com",
		}
		r := SetupRouter(nil)

		w := preflight(r, "https://evil.example.com")
		got := w.Header().Get("Access-Control-Allow-Origin")
		assert.NotEqual(t, "*", got)
		assert.NotEqual(t, "https://evil.example.com", got)
	})
}
