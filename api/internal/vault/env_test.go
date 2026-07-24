package vault

import (
	"os"
	"testing"
)

func TestApplyEnvOverlay(t *testing.T) {
	t.Setenv("OVERLAY_EXISTING", "from-env")
	os.Unsetenv("OVERLAY_NEW")
	t.Cleanup(func() { os.Unsetenv("OVERLAY_NEW") })

	applied := ApplyEnvOverlay(map[string]interface{}{
		"OVERLAY_EXISTING": "from-vault", // must not win over process env
		"OVERLAY_NEW":      "from-vault",
		"OVERLAY_NUMBER":   42, // non-string: ignored
	})

	if applied != 1 {
		t.Errorf("applied = %d, want 1", applied)
	}
	if got := os.Getenv("OVERLAY_EXISTING"); got != "from-env" {
		t.Errorf("OVERLAY_EXISTING = %q, want env value to win", got)
	}
	if got := os.Getenv("OVERLAY_NEW"); got != "from-vault" {
		t.Errorf("OVERLAY_NEW = %q, want from-vault", got)
	}
	if _, exists := os.LookupEnv("OVERLAY_NUMBER"); exists {
		t.Error("OVERLAY_NUMBER should not be set from a non-string field")
	}
}
