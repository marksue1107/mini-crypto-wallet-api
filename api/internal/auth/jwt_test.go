package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateSecretStrength is a regression test for docs/AUDIT.md S2:
// router.go used to silently fall back to a secret hardcoded in this
// repo's source code whenever JWTSecret was empty, instead of refusing to
// start.
func TestValidateSecretStrength(t *testing.T) {
	cases := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "short-secret", true},
		{"exactly 31 chars", strings.Repeat("a", 31), true},
		{"exactly 32 chars", strings.Repeat("a", 32), false},
		{"long random secret", "this-is-a-sufficiently-long-random-secret-value", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSecretStrength(tc.secret)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
