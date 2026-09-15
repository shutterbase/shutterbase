package web

import (
	"fmt"
	"html/template"
	"net/url"
	"strconv"
	"strings"

	"github.com/shutterbase/shutterbase/ent/schema"
)

// themeCSS turns the gallery theme into the CSS custom properties the
// stylesheet consumes (RGB triplets so Tailwind opacity modifiers work) and
// the Google Fonts stylesheet href. Colors were validated as hex by the API;
// anything that still fails to parse falls back to the stylesheet default.
func themeCSS(t schema.GalleryTheme) (template.CSS, string) {
	var b strings.Builder
	b.WriteString(":root{")
	set := func(name, hex string) {
		if r, g, bl, ok := parseHex(hex); ok {
			fmt.Fprintf(&b, "--c-%s:%d %d %d;", name, r, g, bl)
		}
	}
	set("primary", t.Primary)
	set("accent", t.Accent)
	set("surface", t.Surface)
	if t.Radius != "" && validRadius(t.Radius) {
		fmt.Fprintf(&b, "--radius:%s;", t.Radius)
	}
	if f := fontName(t.FontHeading); f != "" {
		fmt.Fprintf(&b, "--font-heading:%q;", f)
	}
	if f := fontName(t.FontBody); f != "" {
		fmt.Fprintf(&b, "--font-body:%q;", f)
	}
	b.WriteString("}")
	if r, g, bl, ok := parseHex(t.SurfaceDark); ok {
		fmt.Fprintf(&b, `[data-theme="dark"]{--c-surface:%d %d %d;}`, r, g, bl)
	}
	return template.CSS(b.String()), fontHref(t.FontHeading, t.FontBody)
}

func parseHex(h string) (int, int, int, bool) {
	h = strings.TrimPrefix(strings.TrimSpace(h), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v >> 16), int(v >> 8 & 0xff), int(v & 0xff), true
}

func validRadius(r string) bool {
	for _, c := range r {
		if !strings.ContainsRune("0123456789.remp%x", c) {
			return false
		}
	}
	return len(r) <= 10
}

// fontName keeps a Google Fonts family name to letters, digits and spaces.
func fontName(f string) string {
	f = strings.TrimSpace(f)
	for _, c := range f {
		if !(c == ' ' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return ""
		}
	}
	if len(f) > 40 {
		return ""
	}
	return f
}

func fontHref(fonts ...string) string {
	var families []string
	seen := map[string]bool{}
	for _, f := range fonts {
		f = fontName(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		families = append(families, "family="+url.PathEscape(f)+":wght@400;500;600;700")
	}
	if len(families) == 0 {
		return ""
	}
	return "https://fonts.googleapis.com/css2?" + strings.Join(families, "&") + "&display=swap"
}
