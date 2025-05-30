package commands

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
)

// ListKeyMapT shows either (o)verrides or new keybinds
type ListKeyMapT struct {
	Open                  key.Binding
	Read                  key.Binding
	Favourite             key.Binding
	ToggleReads           key.Binding
	MarkAllRead           key.Binding
	ToggleFavourites      key.Binding
	Refresh               key.Binding
	OpenInEditor          key.Binding
	OpenInBrowser         key.Binding
	Sort                  key.Binding
	oQuit                 key.Binding
	oForceQuit            key.Binding
	oClearFilter          key.Binding
	oCancelWhileFiltering key.Binding
	oNextPage             key.Binding
	oPrevPage             key.Binding
	EditConfig            key.Binding
	Suspend               key.Binding
}

// ViewportKeyMapT shows *all* keybinds, pulling from viewport.DefaultKeyMap()
type ViewportKeyMapT struct {
	Quit          key.Binding
	Escape        key.Binding
	OpenInEditor  key.Binding
	OpenInBrowser key.Binding
	Favourite     key.Binding
	Read          key.Binding
	GotoStart     key.Binding
	GotoEnd       key.Binding
	Next          key.Binding
	Prev          key.Binding
	ShowFullHelp  key.Binding
	CloseFullHelp key.Binding
	Suspend       key.Binding
}

// ListKeyMap shows either (o)verrides or new keybinds
var ListKeyMap = ListKeyMapT{
	Open: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open"),
	),
	Favourite: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "favourite"),
	),
	ToggleFavourites: key.NewBinding(
		key.WithKeys("F"),
		key.WithHelp("F", "toggle show favourite"),
	),
	Read: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "mark read"),
	),
	ToggleReads: key.NewBinding(
		key.WithKeys("M"),
		key.WithHelp("M", "toggle show read"),
	),
	MarkAllRead: key.NewBinding(
		key.WithKeys("alt+m"),
		key.WithHelp("alt+m", "mark all read"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	OpenInEditor: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "open in $EDITOR"),
	),
	OpenInBrowser: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
	),
	EditConfig: key.NewBinding(
		key.WithKeys("E"),
		key.WithHelp("E", "edit config in $EDITOR"),
	),
	Suspend: key.NewBinding(
		key.WithKeys("ctrl+z"),
		key.WithHelp("ctrl+z", "suspend"),
	),
	// o for override
	oQuit: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "quit"),
	),
	oForceQuit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	oClearFilter: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("esc/q", "clear filter"),
	),
	oCancelWhileFiltering: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
	oPrevPage: key.NewBinding(
		key.WithKeys("left", "h", "pgup"),
		key.WithHelp("←/h/pgup", "prev page"),
	),
	oNextPage: key.NewBinding(
		key.WithKeys("right", "l", "pgdown"),
		key.WithHelp("→/l/pgdn", "next page"),
	),
}

// ViewportKeyMapT shows *all* keybinds, pulling from viewport.DefaultKeyMap()
var ViewportKeyMap = ViewportKeyMapT{
	Next: key.NewBinding(
		key.WithKeys("l", "right"),
		key.WithHelp("l/→", "next"),
	),
	Prev: key.NewBinding(
		key.WithKeys("h", "left"),
		key.WithHelp("h/←", "prev"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Escape: key.NewBinding(
		key.WithKeys("esc", "q"),
		key.WithHelp("q/esc", "escape"),
	),
	Suspend: key.NewBinding(
		key.WithKeys("ctrl+z"),
		key.WithHelp("ctrl+z", "suspend"),
	),
	OpenInEditor: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "open in $EDITOR"),
	),
	OpenInBrowser: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	Favourite: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "favourite"),
	),
	Read: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "mark read"),
	),
	GotoStart: key.NewBinding(
		key.WithKeys("g", "home"),
		key.WithHelp("g", "top"),
	),
	GotoEnd: key.NewBinding(
		key.WithKeys("G", "end"),
		key.WithHelp("G", "bottom"),
	),
	ShowFullHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "more"),
	),
	CloseFullHelp: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "close help"),
	),
}

// This show *all* keybinds, as bubbles/viewport doesn't provide a help function
func (k ViewportKeyMapT) FullHelp() [][]key.Binding {
	v := viewport.DefaultKeyMap()
	return [][]key.Binding{
		{v.Up, v.Down, v.HalfPageUp, v.HalfPageDown},
		{k.GotoStart, k.GotoEnd, v.PageUp, v.PageDown},
		{k.Next, k.Prev, k.OpenInEditor, k.OpenInBrowser, k.Favourite, k.Read},
		{k.Escape, k.Quit, k.CloseFullHelp},
	}
}

// This show *all* keybinds, as bubbles/viewport doesn't provide a help function
func (k ViewportKeyMapT) ShortHelp() []key.Binding {
	v := viewport.DefaultKeyMap()
	return []key.Binding{
		k.Next, k.Prev, v.Down, v.Up, k.Escape, k.ShowFullHelp,
	}
}

// This shows *additional* (or overridden) keybinds alongside built-ins, which *must* take []key.Binding unfortunately.
func (k ListKeyMapT) FullHelp() []key.Binding {
	return []key.Binding{
		k.Open, k.Read, k.Favourite, k.Refresh,
		k.OpenInEditor, k.OpenInBrowser, k.Sort, k.ToggleFavourites, k.ToggleReads,
		k.MarkAllRead, k.EditConfig,
	}
}

// This shows *additional* (or overridden) keybinds alongside built-ins
func (k ListKeyMapT) ShortHelp() []key.Binding {
	return []key.Binding{k.Open}
}

func (k ListKeyMapT) SetOverrides(l *list.Model) {
	l.AdditionalFullHelpKeys = ListKeyMap.FullHelp
	l.AdditionalShortHelpKeys = ListKeyMap.ShortHelp
	l.KeyMap.Quit.SetKeys(k.oQuit.Keys()...)
	l.KeyMap.Quit.SetHelp(k.oQuit.Help().Key, k.oQuit.Help().Desc)
	l.KeyMap.ForceQuit.SetKeys(k.oForceQuit.Keys()...)
	l.KeyMap.ForceQuit.SetHelp(k.oForceQuit.Help().Key, k.oForceQuit.Help().Desc)
	l.KeyMap.ClearFilter.SetKeys(k.oClearFilter.Keys()...)
	l.KeyMap.ClearFilter.SetHelp(k.oClearFilter.Help().Key, k.oClearFilter.Help().Desc)
	l.KeyMap.CancelWhileFiltering.SetKeys(k.oCancelWhileFiltering.Keys()...)
	l.KeyMap.CancelWhileFiltering.SetHelp(k.oCancelWhileFiltering.Help().Key, k.oCancelWhileFiltering.Help().Desc)
	l.KeyMap.NextPage.SetKeys(k.oNextPage.Keys()...)
	l.KeyMap.NextPage.SetHelp(k.oNextPage.Help().Key, k.oNextPage.Help().Desc)
	l.KeyMap.PrevPage.SetKeys(k.oPrevPage.Keys()...)
	l.KeyMap.PrevPage.SetHelp(k.oPrevPage.Help().Key, k.oPrevPage.Help().Desc)
}
