package commands

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

// navModel returns a model over three articles, set up the way Render does.
func navModel(t *testing.T, titles ...string) model {
	t.Helper()

	s, err := store.NewInMemorySQLiteStore()
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range titles {
		if err := s.UpsertItem(&store.Item{Title: title, FeedURL: "https://example.com/feed", Link: "https://example.com/" + title, GUID: title}); err != nil {
			t.Fatal(err)
		}
	}
	items, err := s.GetAllItems("")
	if err != nil {
		t.Fatal(err)
	}

	// Opening an article reloads the list, which drops items of feeds
	// the config does not name.
	cfg := &config.Config{Feeds: []config.Feed{{URL: "https://example.com/feed"}}}
	l := list.New(convertItems(items), itemDelegate{theme: cfg.Theme}, 40, 20)
	l.Filter = CustomFilter(*cfg)
	ListKeyMap.SetOverrides(&l)
	// Keep the commands the pump runs from sleeping: a blinking cursor
	// waits half a second, a status message a second.
	l.FilterInput.Cursor.SetMode(cursor.CursorStatic)
	l.StatusMessageLifetime = 0

	return model{
		cfg:      cfg,
		commands: New(cfg, s),
		list:     l,
		help:     help.New(),
		viewport: viewport.New(78, 10),
	}
}

// press sends keys to the model as the program would, and runs the commands
// they return, feeding their messages back in, until none are left.
func press(t *testing.T, m model, keys ...tea.KeyMsg) model {
	t.Helper()
	for _, k := range keys {
		next, cmd := m.Update(k)
		m = pump(t, next.(model), cmd, 0)
	}
	return m
}

func pump(t *testing.T, m model, cmd tea.Cmd, depth int) model {
	t.Helper()
	if cmd == nil || depth > 20 {
		return m
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	var msg tea.Msg
	select {
	case msg = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("a command did not return")
	}
	switch msg := msg.(type) {
	case nil:
		return m
	case tea.BatchMsg:
		for _, c := range msg {
			m = pump(t, m, c, depth+1)
		}
		return m
	}
	next, cmd := m.Update(msg)
	return pump(t, next.(model), cmd, depth+1)
}

func keys(s string) []tea.KeyMsg {
	var ks []tea.KeyMsg
	for _, r := range s {
		ks = append(ks, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return ks
}

var enter = tea.KeyMsg{Type: tea.KeyEnter}

func openTitle(t *testing.T, m model) string {
	t.Helper()
	if m.selectedArticle == nil {
		t.Fatal("no article open")
	}
	item, err := m.commands.store.GetItemByID(*m.selectedArticle)
	if err != nil {
		t.Fatal(err)
	}
	return item.Title
}

func TestNextAndPrevFollowTheFilteredList(t *testing.T) {
	m := navModel(t, "rust news", "golang one", "golang two")

	// Filter the way a user does, then open the first row.
	m = press(t, m, keys("/golang")...)
	m = press(t, m, enter)
	if len(m.list.VisibleItems()) != 2 {
		t.Fatalf("filter shows %d rows, want 2", len(m.list.VisibleItems()))
	}
	m = press(t, m, enter)
	if got := openTitle(t, m); got != "golang one" {
		t.Fatalf("opened %q, want the first filtered row, %q", got, "golang one")
	}

	m = press(t, m, keys("l")...)
	if got := openTitle(t, m); got != "golang two" {
		t.Errorf("next opened %q, want %q", got, "golang two")
	}

	m = press(t, m, keys("h")...)
	if got := openTitle(t, m); got != "golang one" {
		t.Errorf("prev opened %q, want %q", got, "golang one")
	}
}
