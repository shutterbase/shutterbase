package util_test

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/mxcd/go-config/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shutterbase/shutterbase/internal/util"
)

// S1 unit: config defaults are applied when only required keys are set.
func TestConfigDefaults(t *testing.T) {
	t.Setenv("SESSION_SECRET_KEY", "test-secret")
	require.NoError(t, util.InitConfig())

	assert.Equal(t, 8080, config.Get().Int("PORT"))
	assert.Equal(t, "/api/v1", config.Get().String("API_BASE_URL"))
	assert.Equal(t, "psql", config.Get().String("DATABASE_TYPE"))
	assert.Equal(t, "shutterbase", config.Get().String("S3_BUCKET"))
	assert.False(t, config.Get().Bool("DEV"))
}

// S1 unit: a required key with no default fails when missing.
func TestConfigRequiredMissingFails(t *testing.T) {
	orig, had := os.LookupEnv("SESSION_SECRET_KEY")
	os.Unsetenv("SESSION_SECRET_KEY")
	t.Cleanup(func() {
		if had {
			os.Setenv("SESSION_SECRET_KEY", orig)
		}
	})

	// go-config panics (log.Panic) on a missing required key with no default.
	assert.Panics(t, func() { _ = util.InitConfig() }, "SESSION_SECRET_KEY is required (NotEmpty, no default)")
}

// S1 unit: a .Sensitive() value is masked in config.Print().
func TestConfigSensitiveMaskedInPrint(t *testing.T) {
	const sentinel = "supersecretsentinel123"
	t.Setenv("SESSION_SECRET_KEY", sentinel)
	require.NoError(t, util.InitConfig())

	out := captureStdout(config.Print)

	assert.Contains(t, out, "SESSION_SECRET_KEY", "the var name should be listed")
	assert.NotContains(t, out, sentinel, "the sensitive value must be masked")
}

func captureStdout(fn func()) string {
	orig := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = orig
	b, _ := io.ReadAll(r)
	return strings.TrimSpace(string(b))
}
