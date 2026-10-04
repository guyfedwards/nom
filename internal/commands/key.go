package commands

import (
	"slices"
	"strings"

	"github.com/thebanri/limoni"
)

// binding is a set of keys for one action, and how the help shows it.
type binding struct {
	keys []string
	key  string // the keys as the help writes them
	desc string
}

func bind(help, desc string, keys ...string) binding {
	return binding{keys: keys, key: help, desc: desc}
}

func (b binding) matches(name string) bool { return slices.Contains(b.keys, name) }

// keyName names a key press the way the bindings do: "j", "enter",
// "ctrl+c", "alt+m", "pgdown".
func keyName(k limoni.KeyEvent) string {
	var name string
	switch k.Type {
	case limoni.KeyRune:
		name = string(k.Ch)
		if k.Ctrl {
			return "ctrl+" + strings.ToLower(name)
		}
	case limoni.KeySpace:
		name = " "
	case limoni.KeyEnter:
		name = "enter"
	case limoni.KeyEsc:
		name = "esc"
	case limoni.KeyBackspace:
		name = "backspace"
	case limoni.KeyTab:
		name = "tab"
	case limoni.KeyUp:
		name = "up"
	case limoni.KeyDown:
		name = "down"
	case limoni.KeyLeft:
		name = "left"
	case limoni.KeyRight:
		name = "right"
	case limoni.KeyHome:
		name = "home"
	case limoni.KeyEnd:
		name = "end"
	case limoni.KeyPageUp:
		name = "pgup"
	case limoni.KeyPageDown:
		name = "pgdown"
	case limoni.KeyDelete:
		name = "delete"
	}
	if k.Ctrl && k.Type != limoni.KeyRune {
		name = "ctrl+" + name
	}
	if k.Alt {
		name = "alt+" + name
	}
	return name
}

// listKeys are the list's keys: the ones it always had, and nom's own.
var listKeys = struct {
	Up, Down, PrevPage, NextPage, Top, Bottom    binding
	Filter, ClearFilter, CancelFilter, Accept    binding
	Open, Read, Favourite, ToggleFavourites      binding
	ToggleReads, MarkAllRead, Refresh            binding
	OpenInBrowser, Sort, EditConfig, Suspend     binding
	Quit, ForceQuit, ShowFullHelp, CloseFullHelp binding
}{
	Up:               bind("↑/k", "up", "up", "k"),
	Down:             bind("↓/j", "down", "down", "j"),
	PrevPage:         bind("←/h/pgup", "prev page", "left", "h", "pgup"),
	NextPage:         bind("→/l/pgdn", "next page", "right", "l", "pgdown"),
	Top:              bind("g/home", "go to start", "home", "g"),
	Bottom:           bind("G/end", "go to end", "end", "G"),
	Filter:           bind("/", "filter", "/"),
	ClearFilter:      bind("esc/q", "clear filter", "esc", "q"),
	CancelFilter:     bind("esc", "cancel", "esc"),
	Accept:           bind("enter", "apply filter", "enter"),
	Open:             bind("enter", "open", "enter"),
	Read:             bind("m", "mark read", "m"),
	Favourite:        bind("f", "favourite", "f"),
	ToggleFavourites: bind("F", "toggle show favourite", "F"),
	ToggleReads:      bind("M", "toggle show read", "M"),
	MarkAllRead:      bind("alt+m", "mark all read", "alt+m"),
	Refresh:          bind("r", "refresh", "r"),
	OpenInBrowser:    bind("o", "open in browser", "o"),
	Sort:             bind("s", "sort", "s"),
	EditConfig:       bind("E", "edit config in $EDITOR", "E"),
	Suspend:          bind("ctrl+z", "suspend", "ctrl+z"),
	Quit:             bind("q/esc", "quit", "q", "esc"),
	ForceQuit:        bind("ctrl+c", "quit", "ctrl+c"),
	ShowFullHelp:     bind("?", "more", "?"),
	CloseFullHelp:    bind("?", "close help", "?"),
}

// articleKeys are the article view's keys.
var articleKeys = struct {
	Up, Down, PageUp, PageDown, HalfPageUp, HalfPageDown binding
	GotoStart, GotoEnd, Next, Prev                       binding
	OpenInBrowser, Favourite, Read                       binding
	Escape, Quit, Suspend, ShowFullHelp, CloseFullHelp   binding
}{
	Up:            bind("↑/k", "up", "up", "k"),
	Down:          bind("↓/j", "down", "down", "j"),
	PageUp:        bind("b/pgup", "page up", "pgup", "b"),
	PageDown:      bind("pgdn/space", "page down", "pgdown", " "),
	HalfPageUp:    bind("u", "½ page up", "u", "ctrl+u"),
	HalfPageDown:  bind("d", "½ page down", "d", "ctrl+d"),
	GotoStart:     bind("g", "top", "g", "home"),
	GotoEnd:       bind("G", "bottom", "G", "end"),
	Next:          bind("l/→", "next", "l", "right"),
	Prev:          bind("h/←", "prev", "h", "left"),
	OpenInBrowser: bind("o", "open in browser", "o"),
	Favourite:     bind("f", "favourite", "f"),
	Read:          bind("m", "mark read", "m"),
	Escape:        bind("q/esc", "escape", "esc", "q"),
	Quit:          bind("ctrl+c", "quit", "ctrl+c"),
	Suspend:       bind("ctrl+z", "suspend", "ctrl+z"),
	ShowFullHelp:  bind("?", "more", "?"),
	CloseFullHelp: bind("?", "close help", "?"),
}

// helpLine is bindings written as one line, "key desc • key desc".
func helpLine(bs ...binding) string {
	parts := make([]string, len(bs))
	for i, b := range bs {
		parts[i] = b.key + " " + b.desc
	}
	return strings.Join(parts, " • ")
}

// helpColumns is bindings written in columns, one binding per row of each.
func helpColumns(columns ...[]binding) []string {
	var rows []string
	widths := make([]int, len(columns))
	for c, col := range columns {
		for _, b := range col {
			widths[c] = max(widths[c], limoni.StringWidth(b.key+" "+b.desc))
		}
	}
	for r := 0; ; r++ {
		var line strings.Builder
		any := false
		for c, col := range columns {
			cellText := ""
			if r < len(col) {
				cellText = col[r].key + " " + col[r].desc
				any = true
			}
			line.WriteString(cellText)
			if c < len(columns)-1 {
				line.WriteString(strings.Repeat(" ", widths[c]-limoni.StringWidth(cellText)+4))
			}
		}
		if !any {
			return rows
		}
		rows = append(rows, strings.TrimRight(line.String(), " "))
	}
}

func (m *model) listShortHelp() string {
	k := listKeys
	if m.filtering {
		return helpLine(k.CancelFilter, k.Accept)
	}
	quit := k.Quit
	if m.filterTerm != "" {
		quit = k.ClearFilter
	}
	return helpLine(k.Up, k.Down, k.Filter, k.Open, quit, k.ShowFullHelp)
}

func (m *model) listFullHelp() []string {
	k := listKeys
	return helpColumns(
		[]binding{k.Up, k.Down, k.PrevPage, k.NextPage, k.Top, k.Bottom},
		[]binding{k.Filter, k.ClearFilter, k.Quit, k.ForceQuit, k.CloseFullHelp},
		[]binding{k.Open, k.Read, k.Favourite, k.Refresh, k.OpenInBrowser},
		[]binding{k.Sort, k.ToggleFavourites, k.ToggleReads, k.MarkAllRead, k.EditConfig},
	)
}

func articleShortHelp() string {
	k := articleKeys
	return helpLine(k.Next, k.Prev, k.Down, k.Up, k.Escape, k.ShowFullHelp)
}

func articleFullHelp() []string {
	k := articleKeys
	return helpColumns(
		[]binding{k.Up, k.Down, k.HalfPageUp, k.HalfPageDown},
		[]binding{k.GotoStart, k.GotoEnd, k.PageUp, k.PageDown},
		[]binding{k.Next, k.Prev, k.OpenInBrowser, k.Favourite, k.Read},
		[]binding{k.Escape, k.Quit, k.CloseFullHelp},
	)
}
