//go:build e2e

package e2e

import "os"

func writeFile(path string, data []byte) error { return os.WriteFile(path, data, 0o600) }
func readFile(path string) ([]byte, error)     { return os.ReadFile(path) }
