package controller

import (
	"testing"
)

func TestGetDefaultCopyrightTagFromName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"HelloWorld", "helloworld"},
		{"Hello World", "hello_world"},
		{"Hello-World", "hello_world"},
		{"Hello@World", "hello_world"},
		{"HELLO", "hello"},
		{"Hello!World123", "hello_world123"},
		{"Hello.World", "hello_world"},
		{"", ""},   // empty string
		{"!", "_"}, // only special character
		{"Hello###World", "hello___world"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := getDefaultCopyrightTagFromName(tt.input)
			if got != tt.expected {
				t.Errorf("got %s, expected %s", got, tt.expected)
			}
		})
	}
}
