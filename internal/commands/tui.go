package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

const defaultTitle = "nom"

// statusLifetime is how long a status message stays.
const statusLifetime = time.Second

type TUIItem struct {
	Title     string
	FeedName  string
	URL       string
	ID        int
	Read      bool
	Favourite bool
	Tags      []string
}

func (i TUIItem) FilterValue() string {
	return fmt.Sprintf("%s||%s||%s", i.Title, i.FeedName, strings.Join(i.Tags, "||"))
}

type model struct {
	cfg      *config.Config
	commands *Commands
	errors   []string

	// The list: every item, the ones the filter lets through (indices
	// into items, in the filter's order), and their rows as drawn.
	items   []TUIItem
	visible []int
	rows    []string // as drawn: a favourite's star, or room for the arrow
	// selectedRows are the rows as drawn when selected, behind the arrow.
	selectedRows []string
	list         *widgets.ListState
	listH        int // rows the list had when last drawn: a page

	filter     *widgets.TextInputState
	filtering  bool   // the filter is being typed
	filterTerm string // the filter applied, "" for none

	status       string
	statusExpiry int // which status message an expiry belongs to
	isRefreshing bool
	fullHelp     bool

	// The article: which, its text, and how far it is scrolled.
	selectedArticle *int
	article         *widgets.Markdown
	articleOffset   int
	articleH        int
	images          *images

	lastRead      *TUIItem
	lastReadIndex int

	titleStyle, selectedStyle, readStyle, filterStyle limoni.Style
}

func newModel(items []TUIItem, cmds *Commands, errors []string, cfg *config.Config) *model {
	m := &model{
		cfg:      cfg,
		commands: cmds,
		errors:   errors,
		list:     widgets.NewListState(),
		filter:   widgets.NewTextInputState(),
	}
	m.article = widgets.NewMarkdown("").WithID("article").WithScrollOffset(&m.articleOffset)
	m.applyTheme()
	if cfg.ShowImages() {
		m.images = newImages(cfg.HTTPClient())
		m.article.Images = m.images.get
	}
	m.setItems(items)
	return m
}

// applyTheme reads the colours from the config, which E can change.
func (m *model) applyTheme() {
	theme := m.cfg.Theme
	m.titleStyle = limoni.Style{Fg: parseColor(theme.TitleColorFg), Bg: parseColor(theme.TitleColor)}
	m.selectedStyle = limoni.Style{Fg: parseColor(theme.SelectedItemColor)}
	m.readStyle = limoni.Style{Fg: limoni.ANSI(240)}
	m.filterStyle = limoni.Style{Fg: parseColor(theme.FilterColor)}
	mt, doc := articleTheme(theme)
	m.article.Theme = &mt
	m.article.Style = doc
}

func (m *model) Init() []limoni.Cmd { return nil }

// statusExpired clears a status message once it has been shown long enough.
type statusExpired int

// setStatus shows a message for a second, as the list always did.
func (m *model) setStatus(s string) limoni.Cmd {
	m.status = s
	m.statusExpiry++
	n := m.statusExpiry
	return func(ctx context.Context) limoni.Msg {
		select {
		case <-time.After(statusLifetime):
		case <-ctx.Done():
		}
		return statusExpired(n)
	}
}

func (m *model) Update(msg limoni.Msg) limoni.UpdateResult {
	switch msg := msg.(type) {
	case statusExpired:
		if int(msg) == m.statusExpiry && !m.isRefreshing {
			m.status = ""
		}
		return redraw()
	case imageLoaded:
		if m.images != nil {
			m.images.arrived(msg)
		}
		return redraw()
	}
	if m.selectedArticle != nil {
		return updateViewport(msg, m)
	}
	return updateList(msg, m)
}

func redraw(cmds ...limoni.Cmd) limoni.UpdateResult {
	return limoni.UpdateResult{Redraw: true, Commands: cmds}
}

func (m *model) View(f *limoni.Frame) {
	if m.selectedArticle == nil {
		listView(m, f)
	} else {
		viewportView(m, f)
	}
}

// The help's colours, as the help always had them on a dark background.
var (
	helpDescStyle = limoni.Style{Fg: limoni.Hex("#4A4A4A")}
	helpKeyStyle  = limoni.Style{Fg: limoni.Hex("#626262")}
	helpSepStyle  = limoni.Style{Fg: limoni.Hex("#3C3C3C")}
)

// drawHelp draws the help at the bottom of area and returns the rows above
// it. The text is one Paragraph, which is what a screen reader or an agent
// reads; the keys are then drawn over it a shade lighter, as before.
func drawHelp(f *limoni.Frame, area limoni.Rect, lines []string) limoni.Rect {
	h := uint16(min(len(lines), int(area.Height)))
	if h == 0 {
		return area
	}
	box := limoni.NewRect(area.X+4, area.Y+area.Height-h, area.Width-min(4, area.Width), h)
	f.RenderWidget(&limoni.Paragraph{ID: "help", Text: strings.Join(lines, "\n"), Style: helpDescStyle}, box)
	for row, line := range lines[:h] {
		if limoni.StringWidth(line) > int(box.Width) {
			continue // wrapped: the columns are not where they were written
		}
		y := box.Y + uint16(row)
		col := 0
		for _, field := range helpFields(line) {
			x := box.X + uint16(col+field.at)
			if field.sep {
				f.Buffer.SetString(x, y, field.text, helpSepStyle)
			} else {
				f.Buffer.SetString(x, y, field.text, helpKeyStyle)
			}
		}
	}
	return limoni.NewRect(area.X, area.Y, area.Width, area.Height-h)
}

// helpField is a key, or a separator, in a line of help, and its column.
type helpField struct {
	text string
	at   int
	sep  bool
}

// helpFields finds the keys in a line of help — the first word of each
// entry, entries being separated by " • " or by a gap between columns — and
// the separators.
func helpFields(line string) []helpField {
	var fields []helpField
	col, start := 0, true
	for i, r := range line {
		switch {
		case r == '•':
			fields = append(fields, helpField{text: "•", at: col, sep: true})
			start = true
		case r == ' ':
			if i+1 < len(line) && line[i+1] == ' ' {
				start = true
			}
		default:
			if start {
				end := strings.IndexByte(line[i:], ' ')
				if end < 0 {
					end = len(line) - i
				}
				fields = append(fields, helpField{text: line[i : i+end], at: col})
				start = false
			}
		}
		col += limoni.RuneWidth(r)
	}
	return fields
}

func (m *model) OpenLink(url string) limoni.Cmd {
	for _, o := range m.cfg.Openers {
		match, err := regexp.MatchString(o.Regex, url)
		if err != nil {
			log.Printf("[tui.go] OpenLink: invalid regex pattern: %v", err)
			continue
		}
		if !match {
			continue
		}
		cmdStr := fmt.Sprintf(o.Cmd, url)
		parts := strings.Fields(cmdStr)
		cmd := exec.Command(parts[0], parts[1:]...)

		if o.Takeover {
			return limoni.ExecCmd(cmd, func(err error) limoni.Msg {
				if err != nil {
					log.Println("OpenLink: takeover exec:", err)
					return statusUpdate{status: err.Error()}
				}
				return nil
			})
		}
		return func(context.Context) limoni.Msg {
			if err := cmd.Run(); err != nil {
				log.Println("OpenLink: exec: ", err)
				return statusUpdate{status: err.Error()}
			}
			return nil
		}
	}

	// if no opener, default to browser
	if err := m.OpenInBrowser(url); err != nil {
		log.Println(err)
	}
	return nil
}

func (m *model) OpenInBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		if IsWSL() {
			cmd = "cmd.exe"
			args = []string{"/c", "start"}
		} else {
			cmd = "xdg-open"
		}
	}

	args = append(args, url)
	err := exec.Command(cmd, args...).Start()
	if err != nil {
		return fmt.Errorf("OpenInBrowser: %w", err)
	}

	return nil
}

type tickLoadMsg int

func (m *model) TickLoad(frame int) limoni.Cmd {
	return func(ctx context.Context) limoni.Msg {
		select {
		case <-time.After(150 * time.Millisecond):
		case <-ctx.Done():
		}
		return tickLoadMsg(frame)
	}
}

func ItemToTUIItem(i store.Item) TUIItem {
	return TUIItem{
		ID:        i.ID,
		FeedName:  i.FeedName,
		Title:     i.Title,
		URL:       i.Link,
		Read:      i.Read(),
		Favourite: i.Favourite,
		Tags:      i.Tags,
	}
}

func (c *Commands) TUI() error {
	if debug := os.Getenv("DEBUGNOM"); debug != "" {
		f, err := os.OpenFile(debug, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0o600)
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
		log.SetOutput(f)
		log.SetPrefix("debug ")
	}

	its, err := c.GetAllFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	var errorItems []ErrorItem
	// if no feeds in store, fetchAllFeeds, which will return previews
	if len(c.config.PreviewFeeds) > 0 || len(its) == 0 {
		_, errorItems, err = c.fetchAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}
		// refetch for consistent data across calls
		its, err = c.GetAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}
	}

	es := []string{}
	for _, e := range errorItems {
		es = append(es, fmt.Sprintf("Error fetching %s: %s", e.FeedURL, e.Err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	term, err := limoni.New()
	if err != nil {
		return fmt.Errorf("commands.TUI: %w", err)
	}
	defer term.Close()
	prog := limoni.NewProgram(newModel(convertItems(its), c, es, c.config))
	c.Monitor(ctx, func(msg limoni.Msg) { _ = prog.Send(ctx, msg) })

	if err := prog.RunTerminal(ctx, term, term.Backend()); err != nil {
		return fmt.Errorf("tui.Render: %w", err)
	}
	return nil
}
