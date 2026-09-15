package worker

import "testing"

func TestSafeFilename(t *testing.T) {
	cases := map[string]string{
		"20260804_12-30-00_1234_by_mm.jpg": "20260804_12-30-00_1234_by_mm.jpg",
		"../../etc/passwd":                 "passwd.jpg",
		`C:\evil\..\x.JPG`:                 "x.jpg",
		"weird name (1).jpeg":              "weird name _1.jpg",
		"":                                 "img000000000001.jpg",
		"..":                               "img000000000001.jpg",
		"\x00\x01ctrl":                     "ctrl.jpg",
	}
	for in, want := range cases {
		if got := SafeFilename(in, "img000000000001"); got != want {
			t.Errorf("SafeFilename(%q) = %q, want %q", in, got, want)
		}
	}
	long := SafeFilename(string(make([]byte, 300)), "id")
	if len(long) > 124 {
		t.Errorf("long name not capped: %d", len(long))
	}
}
