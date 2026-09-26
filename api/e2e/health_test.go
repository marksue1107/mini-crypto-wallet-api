//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthAndReady(t *testing.T) {
	h := do(t, http.MethodGet, "/health", nil, nil)
	assert.Equal(t, http.StatusOK, h.Status, string(h.Body))

	r := do(t, http.MethodGet, "/ready", nil, nil)
	assert.Equal(t, http.StatusOK, r.Status, string(r.Body))
}
