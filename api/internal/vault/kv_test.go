package vault

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	vaultapi "github.com/hashicorp/vault/api"
)

func TestGetKVUnwrapsEnvelopes(t *testing.T) {
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/secret/data/s3": // KV v2: data/metadata envelope
			w.Write([]byte(`{"data":{"data":{"access_key":"AK2","secret_key":"SK2"},"metadata":{"version":1}}}`))
		case "/v1/kv/s3": // KV v1: flat
			w.Write([]byte(`{"data":{"access_key":"AK1","data":"a literal field named data"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer fake.Close()

	apiClient, err := vaultapi.NewClient(&vaultapi.Config{Address: fake.URL})
	if err != nil {
		t.Fatal(err)
	}
	client := &Client{Client: apiClient, options: &Options{}}
	ctx := context.Background()

	for _, tc := range []struct{ path, key, want string }{
		{"secret/data/s3", "access_key", "AK2"},
		{"secret/data/s3", "secret_key", "SK2"},
		{"kv/s3", "access_key", "AK1"},
		{"kv/s3", "data", "a literal field named data"},
	} {
		got, err := client.GetKVString(ctx, tc.path, tc.key)
		if err != nil {
			t.Fatalf("GetKVString(%s, %s): %v", tc.path, tc.key, err)
		}
		if got != tc.want {
			t.Errorf("GetKVString(%s, %s) = %q, want %q", tc.path, tc.key, got, tc.want)
		}
	}

	if _, err := client.GetKVString(ctx, "kv/s3", "missing"); err == nil {
		t.Error("expected error for missing key")
	}
	if _, err := client.GetKVString(ctx, "kv/nope", "x"); err == nil {
		t.Error("expected error for missing secret")
	}
}
