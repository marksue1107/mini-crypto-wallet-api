package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func newTestRouter(t *testing.T, limiter *RateLimiter) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Trust no proxies, matching router.SetupRouter's default. See
	// docs/AUDIT.md S4.
	require.NoError(t, r.SetTrustedProxies(nil))
	r.Use(RateLimitMiddleware(limiter))
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

// TestRateLimitMiddleware_BlocksAfterLimit is a regression test for
// docs/AUDIT.md S3: the rate limiter existed but was never wired into any
// route. This tests the middleware itself actually enforces a limit.
func TestRateLimitMiddleware_BlocksAfterLimit(t *testing.T) {
	limiter := NewRateLimiter(rate.Every(time.Minute), 2) // burst of 2, then blocked
	r := newTestRouter(t, limiter)

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "9.9.9.9:1111"
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "9.9.9.9:1111"
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// TestRateLimitMiddleware_SeparateClientsHaveSeparateLimits verifies the
// limiter is keyed per client, not global.
func TestRateLimitMiddleware_SeparateClientsHaveSeparateLimits(t *testing.T) {
	limiter := NewRateLimiter(rate.Every(time.Minute), 1)
	r := newTestRouter(t, limiter)

	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "2.2.2.2:1234"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// TestRateLimitMiddleware_ClientIPNotSpoofableViaXFF_WhenNoTrustedProxies is
// a regression test for docs/AUDIT.md S4: without SetTrustedProxies(nil),
// Gin trusts every proxy's X-Forwarded-For header by default, letting a
// client bypass IP-based rate limiting just by sending its own
// X-Forwarded-For with a different value each request. With no trusted
// proxies configured, Gin's ClientIP() must ignore that header and use the
// real connection's remote address, so the limit is still enforced.
func TestRateLimitMiddleware_ClientIPNotSpoofableViaXFF_WhenNoTrustedProxies(t *testing.T) {
	limiter := NewRateLimiter(rate.Every(time.Minute), 1) // burst of 1
	r := newTestRouter(t, limiter)

	// First request from real IP 9.9.9.9 uses up the burst of 1.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	req1.RemoteAddr = "9.9.9.9:1111"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request from the SAME real IP, claiming via a spoofed
	// X-Forwarded-For to be a completely different client. If that header
	// were trusted, this would get its own fresh limit and succeed;
	// instead it must still be rate-limited.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "9.9.9.9:2222"
	req2.Header.Set("X-Forwarded-For", "1.2.3.4")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}
