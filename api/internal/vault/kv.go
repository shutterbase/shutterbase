package vault

import (
	"context"
	"fmt"
)

// GetKV reads a secret from a KV mount, unwrapping the KV v2 data/metadata
// envelope when present, so both v1 ("secret/foo") and v2 ("secret/data/foo")
// paths work.
func (c *Client) GetKV(ctx context.Context, path string) (map[string]interface{}, error) {
	secret, err := c.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault secret at %q: %w", path, err)
	}
	if secret == nil {
		return nil, fmt.Errorf("no secret found in vault at %q", path)
	}
	data := secret.Data
	if nested, ok := data["data"].(map[string]interface{}); ok {
		if _, hasMetadata := data["metadata"]; hasMetadata {
			data = nested
		}
	}
	return data, nil
}

func (c *Client) GetKVString(ctx context.Context, path string, key string) (string, error) {
	data, err := c.GetKV(ctx, path)
	if err != nil {
		return "", err
	}
	value, ok := data[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("no string value for key %q in vault secret at %q", key, path)
	}
	return value, nil
}
