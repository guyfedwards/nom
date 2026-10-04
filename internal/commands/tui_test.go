package commands

import (
	"bytes"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/thebanri/limoni/uitest"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

const testFeed = "https://blog.example/feed.xml"

// newTestTUI is the TUI over an in-memory store holding items, run through
// Limoni's real message loop on an 80×24 terminal. Tests drive it as a user
// does, and read it through its semantic tree: the list's rows, the article's
// text, the status line.
func newTestTUI(t *testing.T, cfg *config.Config, items ...store.Item) (*uitest.Page, *Commands) {
	t.Helper()
	s, err := store.NewInMemorySQLiteStore()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	for i := range items {
		items[i].FeedURL = testFeed
		if items[i].Link == "" {
			items[i].Link = "https://blog.example/" + strings.ReplaceAll(strings.ToLower(items[i].Title), " ", "-")
		}
		items[i].PublishedAt = now.Add(-time.Duration(i) * time.Hour)
		items[i].CreatedAt = items[i].PublishedAt
		if err := s.UpsertItem(&items[i]); err != nil {
			t.Fatal(err)
		}
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	cfg.Feeds = []config.Feed{{URL: testFeed, Name: "Blog"}}
	cfg.Theme = config.DefaultTheme
	if cfg.Ordering == "" {
		cfg.Ordering = "desc"
	}
	cmds := New(cfg, s)
	all, err := cmds.GetAllFeeds()
	if err != nil {
		t.Fatal(err)
	}
	return uitest.Program(t, 80, 24, newModel(convertItems(all), cmds, nil, cfg)), cmds
}

func rows(page *uitest.Page) uitest.Locator {
	return page.GetByRole("list-item", "").Within(page.GetByID("items"))
}

func threeItems() []store.Item {
	return []store.Item{
		{Title: "First post", Content: "<p>Hello from the <strong>first</strong> post.</p>"},
		{Title: "Second post", Content: "<p>The second one.</p>"},
		{Title: "Third post", Content: "<p>And a third.</p>"},
	}
}

// The list shows the items newest first, as "n. Feed: Title"; Enter opens
// the selected one, and Esc goes back to the list.
func TestTUIOpensAnArticleFromTheList(t *testing.T) {
	page, _ := newTestTUI(t, nil, threeItems()...)

	page.Expect(rows(page)).ToHaveCount(3)
	page.Expect(rows(page).Nth(0)).ToHaveLabel(">   1. Blog: First post")
	page.Expect(rows(page).Nth(1)).ToHaveLabel("    2. Blog: Second post")
	page.Expect(rows(page).Nth(0)).ToBeSelected()

	page.Press("j")
	page.Expect(rows(page).Nth(1)).ToBeSelected()
	page.Press("enter")
	page.Expect(page.GetByID("article")).ToContainValue("Second post")
	page.Expect(page.GetByID("article")).ToContainValue("The second one.")

	page.Press("l")
	page.Expect(page.GetByID("article")).ToContainValue("And a third.")
	page.Press("h")
	page.Expect(page.GetByID("article")).ToContainValue("The second one.")

	page.Press("esc")
	page.Expect(page.GetByID("article")).Not().ToBeVisible()
	page.Expect(rows(page).Nth(1)).ToBeSelected()
}

// "/" filters as it is typed, Enter keeps the filter and says so, and Esc
// clears it.
func TestTUIFiltersTheList(t *testing.T) {
	page, _ := newTestTUI(t, nil, threeItems()...)

	page.Press("G") // the selection is on the last row, as filtering starts
	page.Press("/")
	page.Expect(page.GetByID("filter")).ToBeFocused()
	page.Type("post")
	page.Expect(rows(page).Nth(0)).ToBeSelected()
	page.Press("ctrl+u")
	page.Type("third")
	page.Expect(rows(page)).ToHaveCount(1)
	page.Expect(rows(page).Nth(0)).ToContainLabel("Third post")

	page.Press("enter")
	page.Expect(page.GetByID("status")).ToContainValue("filtering: third")
	page.Press("enter") // opens the one row left
	page.Expect(page.GetByID("article")).ToContainValue("And a third.")
	page.Press("esc")

	page.Press("esc")
	page.Expect(rows(page)).ToHaveCount(3)
}

// m marks the selected item read, which takes it out of the default view;
// M shows read items again.
func TestTUIMarksItemsRead(t *testing.T) {
	page, cmds := newTestTUI(t, nil, threeItems()...)

	page.Press("m")
	page.Expect(rows(page)).ToHaveCount(2)
	page.Expect(rows(page).Nth(0)).ToContainLabel("Second post")
	if n := cmds.CountUnread(); n != 2 {
		t.Fatalf("the store has %d unread, want 2", n)
	}

	page.Press("M")
	page.Expect(rows(page)).ToHaveCount(3)
}

// f favourites the selected item: it is marked with a star.
func TestTUIFavourites(t *testing.T) {
	page, _ := newTestTUI(t, nil, threeItems()...)

	page.Press("f")
	page.Press("j")
	page.Expect(rows(page).Nth(0)).ToHaveLabel("*   1. Blog: First post")
	page.Press("F") // favourites only
	page.Expect(rows(page)).ToHaveCount(1)
}

// The help line lists the keys; ? shows them all and closes again.
func TestTUIHelp(t *testing.T) {
	page, _ := newTestTUI(t, nil, threeItems()...)

	page.Expect(page.GetByID("help")).ToContainValue("/ filter")
	page.Press("?")
	page.Expect(page.GetByID("help")).ToContainValue("edit config in $EDITOR")
	page.Press("?")
	page.Expect(page.GetByID("help")).Not().ToContainValue("edit config in $EDITOR")
	page.Press("q")
	page.ExpectExit()
}

// E hands the terminal to $EDITOR. A test's terminal has nothing to hand
// over, and says so on the status line instead of hanging.
func TestTUIEditConfigWithoutATerminal(t *testing.T) {
	t.Setenv("NOMEDITOR", "true")
	page, _ := newTestTUI(t, &config.Config{ConfigPath: t.TempDir() + "/config.yml"}, threeItems()...)

	page.Press("E")
	page.Expect(page.GetByID("status")).ToContainValue("cannot be handed to another program")
}

// An article's pictures are downloaded — a relative address resolved
// against the article's own — and drawn: here in half blocks, the test
// terminal having no image protocol.
func TestTUIShowsAnArticlesPictures(t *testing.T) {
	t.Setenv("LIMONI_GRAPHICS", "halfblock")
	pic := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+3] = 255, 255 // red
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, pic); err != nil {
		t.Fatal(err)
	}
	fetched := make(chan string, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetched <- r.URL.Path
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()

	page, _ := newTestTUI(t, nil, store.Item{
		Title:   "With a picture",
		Link:    srv.URL + "/posts/one",
		Content: `<p>Before.</p><p><img src="/img/red.png" alt="a red box"></p><p>After.</p>`,
	})
	page.Press("enter")
	page.Expect(page.GetByID("article")).ToContainValue("[image: a red box]")
	select {
	case path := <-fetched:
		if path != "/img/red.png" {
			t.Fatalf("fetched %q, want the address resolved against the article", path)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the picture was never fetched")
	}

	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(page.Screen(), "▄") {
		if time.Now().After(deadline) {
			t.Fatalf("the picture was not drawn:\n%s", page.Screen())
		}
		time.Sleep(20 * time.Millisecond)
	}
	if strings.Contains(page.Screen(), "▣ a red box") {
		t.Error("the alt text is still drawn in place of the picture")
	}
}

// With images: false nothing is downloaded, and the alt text stands in.
func TestTUIImagesOff(t *testing.T) {
	off := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("fetched %s with images off", r.URL)
	}))
	defer srv.Close()
	page, _ := newTestTUI(t, &config.Config{Images: &off}, store.Item{
		Title:   "With a picture",
		Content: `<p><img src="` + srv.URL + `/a.png" alt="a box"></p>`,
	})
	page.Press("enter")
	page.Expect(page.GetByID("article")).ToContainValue("[image: a box]")
	time.Sleep(200 * time.Millisecond)
}
