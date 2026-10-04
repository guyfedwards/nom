package commands

import (
	"strings"
	"testing"
)

func TestHtmlToMdResolvesRelativeURLs(t *testing.T) {
	const base = "https://go.dev/blog/simd/"

	tests := []struct {
		name string
		html string
		want string
	}{
		{"root-relative link", `<a href="/doc/">docs</a>`, "[docs](https://go.dev/doc/)"},
		{"path-relative link", `<a href="next">next</a>`, "[next](https://go.dev/blog/simd/next)"},
		{"parent link", `<a href="../">blog</a>`, "[blog](https://go.dev/blog/)"},
		{"fragment", `<a href="#bench">bench</a>`, "[bench](https://go.dev/blog/simd/#bench)"},
		{"scheme-relative image", `<img src="//cdn.example.com/a.png" alt="a">`, "![a](https://cdn.example.com/a.png)"},
		{"relative image", `<img src="chart.svg" alt="chart">`, "![chart](https://go.dev/blog/simd/chart.svg)"},
		{"absolute link untouched", `<a href="http://example.com/x">x</a>`, "[x](http://example.com/x)"},
		{"mailto untouched", `<a href="mailto:a@example.com">mail</a>`, "[mail](mailto:a@example.com)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := htmlToMd(tt.html, base)
			if !strings.Contains(got, tt.want) {
				t.Errorf("htmlToMd(%q) = %q, want it to contain %q", tt.html, got, tt.want)
			}
		})
	}
}

func TestHtmlToMdWithoutBaseKeepsURLs(t *testing.T) {
	for _, base := range []string{"", "not a url", "/relative/only"} {
		got := htmlToMd(`<a href="/doc/">docs</a>`, base)
		if !strings.Contains(got, "[docs](/doc/)") {
			t.Errorf("base %q: got %q, want the link left as it was", base, got)
		}
	}
}
