package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/thebanri/limoni"

	"github.com/guyfedwards/nom/v2/internal/config"
)

// Colours are read as they always were: ANSI 256 numbers and hex values.
func TestParseColor(t *testing.T) {
	for in, want := range map[string]limoni.Color{
		"62":      limoni.ANSI(62),
		"#5A56E0": limoni.Hex("#5A56E0"),
		"":        limoni.ColorDefault,
		"blue":    limoni.ColorDefault,
		"300":     limoni.ColorDefault,
	} {
		if got := parseColor(in); got != want {
			t.Errorf("parseColor(%q) = %v, want %v", in, got, want)
		}
	}
}

// The first heading is the article's title, in titleColor and titleColorFg;
// the other headings and the links take the glamour style's colours.
func TestArticleThemeFromGlamourStyles(t *testing.T) {
	theme := config.DefaultTheme
	mt, doc := articleTheme(theme)
	if mt.Headings[0].Bg != limoni.ANSI(62) || mt.Headings[0].Fg != limoni.ANSI(231) {
		t.Errorf("the title is %+v, want titleColorFg on titleColor", mt.Headings[0])
	}
	if mt.Headings[1].Fg != limoni.ANSI(39) || mt.Link.Fg != limoni.ANSI(35) || doc.Fg != limoni.ANSI(252) {
		t.Errorf("dark: heading %v, link %v, text %v", mt.Headings[1].Fg, mt.Link.Fg, doc.Fg)
	}

	theme.Glamour = "dracula"
	if mt, _ := articleTheme(theme); mt.Headings[2].Fg != limoni.Hex("#bd93f9") {
		t.Errorf("dracula heading %v", mt.Headings[2].Fg)
	}
	theme.Glamour = "notty"
	if mt, doc := articleTheme(theme); mt.Headings[0].Bg != limoni.ColorDefault || doc != (limoni.Style{}) {
		t.Errorf("notty has colour: title %+v, text %+v", mt.Headings[0], doc)
	}
}

// "custom" reads the colours from a glamour style file; one that cannot be
// read falls back to dark.
func TestArticleThemeCustom(t *testing.T) {
	path := filepath.Join(t.TempDir(), "style.json")
	if err := os.WriteFile(path, []byte(`{"heading": {"color": "#ff0000", "bold": true}, "link_text": {"color": "#00ff00"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	theme := config.DefaultTheme
	theme.Glamour, theme.CustomPath = "custom", path
	mt, _ := articleTheme(theme)
	if mt.Headings[1].Fg != limoni.Hex("#ff0000") || mt.Link.Fg != limoni.Hex("#00ff00") {
		t.Errorf("custom: heading %v, link %v", mt.Headings[1].Fg, mt.Link.Fg)
	}

	theme.CustomPath = filepath.Join(t.TempDir(), "missing.json")
	if mt, _ := articleTheme(theme); mt.Headings[1].Fg != limoni.ANSI(39) {
		t.Errorf("a missing custom file did not fall back to dark: %v", mt.Headings[1].Fg)
	}
}
