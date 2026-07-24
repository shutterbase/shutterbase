package vault

import "os"

// ApplyEnvOverlay sets every string field of a KV secret as a process env var,
// skipping variables that are already set — explicit process env always wins
// over vault. Non-string fields are ignored. Returns how many were applied.
func ApplyEnvOverlay(data map[string]interface{}) int {
	applied := 0
	for key, value := range data {
		s, ok := value.(string)
		if !ok {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		os.Setenv(key, s)
		applied++
	}
	return applied
}
