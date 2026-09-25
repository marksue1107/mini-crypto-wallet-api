package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runInEmptyDir chdirs into a fresh temp directory (guaranteed to have no
// config.yaml) for the duration of fn, then restores the working directory.
func runInEmptyDir(t *testing.T, fn func()) {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	defer func() {
		require.NoError(t, os.Chdir(wd))
	}()
	fn()
}

// TestLoadConfig_FromEnvOnly_NoConfigFile is a regression test for
// docs/AUDIT.md S1: config.yaml is no longer committed/deployed, so
// production must be configurable purely from environment variables, with
// no config.yaml present at all.
func TestLoadConfig_FromEnvOnly_NoConfigFile(t *testing.T) {
	viper.Reset()
	t.Setenv("JWT_SECRET", "env-only-secret-min-32-characters-long")
	t.Setenv("DB_DRIVER", "postgres")

	runInEmptyDir(t, LoadConfig)

	assert.Equal(t, "env-only-secret-min-32-characters-long", Config.JWTSecret)
	assert.Equal(t, "postgres", Config.DBDriver)
}

// TestLoadConfig_DefaultsWhenNothingSet verifies sane defaults are used when
// there's no config.yaml and no relevant environment variables.
func TestLoadConfig_DefaultsWhenNothingSet(t *testing.T) {
	viper.Reset()

	runInEmptyDir(t, LoadConfig)

	assert.Equal(t, "development", Config.AppEnv)
	assert.Equal(t, "sqlite", Config.DBDriver)
	assert.Equal(t, "", Config.JWTSecret)
}
