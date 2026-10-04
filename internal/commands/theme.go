package commands

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"

	"github.com/guyfedwards/nom/v2/internal/config"
)

// parseColor reads a colour the way the config always has: an ANSI 256
// number ("62") or a hex value ("#5A56E0"). Anything else is the terminal's
// default colour.
func parseColor(s string) limoni.Color {
	s = strings.TrimSpace(s)
	if s == "" {
		return limoni.ColorDefault
	}
	if strings.HasPrefix(s, "#") {
		return limoni.Hex(s)
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 255 {
		return limoni.ColorDefault
	}
	return limoni.ANSI(uint8(n))
}

// glamourBlock is the part of a glamour style entry the article view reads.
type glamourBlock struct {
	Color           *string `json:"color"`
	BackgroundColor *string `json:"background_color"`
	Bold            *bool   `json:"bold"`
	Italic          *bool   `json:"italic"`
	Underline       *bool   `json:"underline"`
}

// glamourStyle is the part of a glamour style file the article view reads:
// the colours. Margins, prefixes and the chroma theme have no equivalent.
type glamourStyle struct {
	Document   glamourBlock `json:"document"`
	Heading    glamourBlock `json:"heading"`
	H1         glamourBlock `json:"h1"`
	H2         glamourBlock `json:"h2"`
	H3         glamourBlock `json:"h3"`
	H4         glamourBlock `json:"h4"`
	H5         glamourBlock `json:"h5"`
	H6         glamourBlock `json:"h6"`
	Link       glamourBlock `json:"link"`
	LinkText   glamourBlock `json:"link_text"`
	Code       glamourBlock `json:"code"`
	CodeBlock  glamourBlock `json:"code_block"`
	BlockQuote glamourBlock `json:"block_quote"`
	HR         glamourBlock `json:"hr"`
	ImageText  glamourBlock `json:"image_text"`
}

func str(s string) *string { return &s }
func yes() *bool           { b := true; return &b }

// The glamour styles nom has always offered, by the colours glamour v0.10
// gives them.
var glamourStyles = map[string]glamourStyle{
	"dark": {
		Document:  glamourBlock{Color: str("252")},
		Heading:   glamourBlock{Color: str("39"), Bold: yes()},
		H1:        glamourBlock{Color: str("228"), BackgroundColor: str("63"), Bold: yes()},
		Link:      glamourBlock{Color: str("30"), Underline: yes()},
		LinkText:  glamourBlock{Color: str("35"), Bold: yes()},
		Code:      glamourBlock{Color: str("203"), BackgroundColor: str("236")},
		HR:        glamourBlock{Color: str("240")},
		ImageText: glamourBlock{Color: str("243")},
	},
	"light": {
		Document:  glamourBlock{Color: str("234")},
		Heading:   glamourBlock{Color: str("27"), Bold: yes()},
		H1:        glamourBlock{Color: str("228"), BackgroundColor: str("63"), Bold: yes()},
		Link:      glamourBlock{Color: str("36"), Underline: yes()},
		LinkText:  glamourBlock{Color: str("29"), Bold: yes()},
		Code:      glamourBlock{Color: str("203"), BackgroundColor: str("254")},
		HR:        glamourBlock{Color: str("249")},
		ImageText: glamourBlock{Color: str("243")},
	},
	"dracula": {
		Document:   glamourBlock{Color: str("#f8f8f2")},
		Heading:    glamourBlock{Color: str("#bd93f9"), Bold: yes()},
		Link:       glamourBlock{Color: str("#8be9fd"), Underline: yes()},
		LinkText:   glamourBlock{Color: str("#ff79c6")},
		Code:       glamourBlock{Color: str("#50fa7b")},
		BlockQuote: glamourBlock{Color: str("#f1fa8c"), Italic: yes()},
		HR:         glamourBlock{Color: str("#6272A4")},
		ImageText:  glamourBlock{Color: str("#ff79c6")},
	},
	"pink": {
		Heading:   glamourBlock{Color: str("212"), Bold: yes()},
		Link:      glamourBlock{Color: str("99"), Underline: yes()},
		LinkText:  glamourBlock{Bold: yes()},
		Code:      glamourBlock{Color: str("212"), BackgroundColor: str("236")},
		HR:        glamourBlock{Color: str("212")},
		ImageText: glamourBlock{Underline: yes()},
	},
	// ascii and notty are glamour's styles without colour.
	"ascii": {},
	"notty": {},
}

// over is s with every block o sets laid over it.
func (s glamourStyle) over(o glamourStyle) glamourStyle {
	s.Document = s.Document.over(o.Document)
	s.Heading = s.Heading.over(o.Heading)
	s.H1 = s.H1.over(o.H1)
	s.H2 = s.H2.over(o.H2)
	s.H3 = s.H3.over(o.H3)
	s.H4 = s.H4.over(o.H4)
	s.H5 = s.H5.over(o.H5)
	s.H6 = s.H6.over(o.H6)
	s.Link = s.Link.over(o.Link)
	s.LinkText = s.LinkText.over(o.LinkText)
	s.Code = s.Code.over(o.Code)
	s.CodeBlock = s.CodeBlock.over(o.CodeBlock)
	s.BlockQuote = s.BlockQuote.over(o.BlockQuote)
	s.HR = s.HR.over(o.HR)
	s.ImageText = s.ImageText.over(o.ImageText)
	return s
}

func (b glamourBlock) style() limoni.Style {
	var s limoni.Style
	if b.Color != nil {
		s.Fg = parseColor(*b.Color)
	}
	if b.BackgroundColor != nil {
		s.Bg = parseColor(*b.BackgroundColor)
	}
	if b.Bold != nil && *b.Bold {
		s.Modifier |= limoni.ModifierBold
	}
	if b.Italic != nil && *b.Italic {
		s.Modifier |= limoni.ModifierItalic
	}
	if b.Underline != nil && *b.Underline {
		s.Modifier |= limoni.ModifierUnderline
	}
	return s
}

// over is b with the fields o sets taken from o: a glamour level style
// (h2) refines the shared heading style.
func (b glamourBlock) over(o glamourBlock) glamourBlock {
	if o.Color != nil {
		b.Color = o.Color
	}
	if o.BackgroundColor != nil {
		b.BackgroundColor = o.BackgroundColor
	}
	if o.Bold != nil {
		b.Bold = o.Bold
	}
	if o.Italic != nil {
		b.Italic = o.Italic
	}
	if o.Underline != nil {
		b.Underline = o.Underline
	}
	return b
}

// articleTheme is the article view's colours for the configured theme:
// theme.glamour names one of glamour's styles, or "custom" a glamour style
// file at theme.customPath, whose colours are read. The first heading is the
// title, in titleColor and titleColorFg as before.
func articleTheme(theme config.Theme) (widgets.MarkdownTheme, limoni.Style) {
	gs, ok := glamourStyles[theme.Glamour]
	if theme.Glamour == "custom" {
		// Read into a style of its own and lay it over dark: decoding into a
		// copy of dark would write through the pointers it shares with it.
		gs = glamourStyles["dark"]
		var custom glamourStyle
		data, err := os.ReadFile(theme.CustomPath)
		if err != nil {
			log.Println(err)
		} else if err := json.Unmarshal(data, &custom); err != nil {
			log.Println(err)
		} else {
			gs = gs.over(custom)
		}
	} else if !ok {
		gs = glamourStyles["dark"]
	}

	t := widgets.MarkdownTheme{
		Link:      gs.Link.over(gs.LinkText).style(),
		Code:      gs.Code.style(),
		CodeBlock: limoni.Style{Bg: gs.CodeBlock.style().Bg},
		Quote:     gs.BlockQuote.style(),
		Rule:      gs.HR.style(),
		Image:     gs.ImageText.style(),
		Bullet:    gs.Document.style(),
	}
	for i, h := range []glamourBlock{gs.H1, gs.H2, gs.H3, gs.H4, gs.H5, gs.H6} {
		t.Headings[i] = gs.Heading.over(h).style()
	}
	if theme.Glamour == "ascii" || theme.Glamour == "notty" {
		for i := range t.Headings {
			t.Headings[i] = limoni.Style{Modifier: limoni.ModifierBold}
		}
		t.Link = limoni.Style{Modifier: limoni.ModifierUnderline}
		return t, limoni.Style{}
	}
	t.Headings[0] = limoni.Style{
		Fg:       parseColor(theme.TitleColorFg),
		Bg:       parseColor(theme.TitleColor),
		Modifier: limoni.ModifierBold,
	}
	return t, gs.Document.style()
}
