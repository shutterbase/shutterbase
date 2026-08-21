package repository

import "testing"

func TestNormalizeCopyrightTag(t *testing.T) {
	cases := map[string]string{
		"mm":             "mm",
		"MM":             "mm",
		"Max Mustermann": "max_mustermann",
		"max-mustermann": "max_mustermann",
		"max.muster - x": "max_muster_x",
		"Möller":         "moeller",
		"Äöü ß":          "aeoeue_ss",
		"":               "",
	}
	for in, want := range cases {
		if got := NormalizeCopyrightTag(in); got != want {
			t.Errorf("NormalizeCopyrightTag(%q) = %q, want %q", in, got, want)
		}
	}
}
