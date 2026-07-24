package server

import (
	"strings"
	"testing"

	"github.com/shutterbase/shutterbase/ent/schema"
)

func TestValidateHotkeys(t *testing.T) {
	valid := &schema.UserHotkeys{
		Bindings:    map[string][]string{"images.next-image": {"ArrowRight", "l"}, "images.open-tagging": {}},
		TagBindings: map[string]string{"p": "review", "shift+1": "podium"},
	}
	if msg := validateHotkeys(valid); msg != "" {
		t.Fatalf("expected valid config, got %q", msg)
	}
	if msg := validateHotkeys(&schema.UserHotkeys{}); msg != "" {
		t.Fatalf("expected empty config to be valid, got %q", msg)
	}

	invalid := []*schema.UserHotkeys{
		{Bindings: map[string][]string{"a": {""}}},
		{Bindings: map[string][]string{"a": {strings.Repeat("x", 65)}}},
		{Bindings: map[string][]string{strings.Repeat("x", 65): {"a"}}},
		{Bindings: map[string][]string{"a": {"1", "2", "3", "4", "5", "6", "7", "8", "9"}}},
		{TagBindings: map[string]string{"": "review"}},
		{TagBindings: map[string]string{"p": ""}},
		{TagBindings: map[string]string{"p": strings.Repeat("x", 129)}},
	}
	for i, h := range invalid {
		if msg := validateHotkeys(h); msg == "" {
			t.Errorf("case %d: expected validation error, got none", i)
		}
	}

	tooMany := &schema.UserHotkeys{Bindings: map[string][]string{}}
	for i := 0; i < 129; i++ {
		tooMany.Bindings[strings.Repeat("a", 10)+string(rune('0'+i%10))+string(rune('a'+i/10))] = []string{"x"}
	}
	if msg := validateHotkeys(tooMany); msg == "" {
		t.Error("expected error for too many bindings")
	}
}
