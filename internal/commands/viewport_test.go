package commands

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

// articleModel returns a model showing one long article, as the list's
// Open leaves it.
func articleModel(t *testing.T) (model, store.Store, int) {
	t.Helper()

	s, err := store.NewInMemorySQLiteStore()
	if err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertItem(&store.Item{Title: "article", FeedURL: "https://example.com/feed", Link: "https://example.com/a", GUID: "a"}); err != nil {
		t.Fatal(err)
	}
	items, err := s.GetAllItems("")
	if err != nil || len(items) != 1 {
		t.Fatalf("GetAllItems: %v, %d items", err, len(items))
	}
	id := items[0].ID

	cfg := &config.Config{}
	vp := viewport.New(78, 10)
	vp.KeyMap = viewportKeys()
	vp.SetContent(strings.Repeat("line\n", 100))

	m := model{
		selectedArticle: &id,
		cfg:             cfg,
		commands:        New(cfg, s),
		list:            list.New(nil, list.NewDefaultDelegate(), 20, 10),
		help:            help.New(),
		viewport:        vp,
	}
	return m, s, id
}

func TestFavouriteInArticleDoesNotPageDown(t *testing.T) {
	m, s, id := articleModel(t)

	next, _ := updateViewport(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}}, m)
	m = next.(model)

	if m.viewport.YOffset != 0 {
		t.Errorf("f scrolled the article to line %d, want it to stay at the top", m.viewport.YOffset)
	}
	item, err := s.GetItemByID(id)
	if err != nil {
		t.Fatal(err)
	}
	if !item.Favourite {
		t.Error("f did not favourite the article")
	}
}

func TestSpaceStillPagesDownInArticle(t *testing.T) {
	m, _, _ := articleModel(t)

	next, _ := updateViewport(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}, m)
	m = next.(model)

	if m.viewport.YOffset == 0 {
		t.Error("space did not page down")
	}
}
