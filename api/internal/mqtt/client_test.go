package mqtt

import (
	"testing"
)

func TestNoOpClient(t *testing.T) {
	// Empty broker => no-op client
	c := New(&Options{
		Broker:      "",
		ClientID:    "test",
		TopicPrefix: "test",
	})
	if c.IsConnected() {
		t.Fatal("no-op client should not be connected")
	}
	// Should not panic
	c.Publish("foo/bar", map[string]string{"hello": "world"})
	c.Close()
}

func TestNilClient(t *testing.T) {
	var c *Client
	// All methods should be safe on nil
	c.Publish("foo", "bar")
	if c.IsConnected() {
		t.Fatal("nil client should not be connected")
	}
	c.Close()
}
