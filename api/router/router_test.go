package router

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"mini-crypto-wallet-api/internal/config"
	"mini-crypto-wallet-api/internal/test"

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
