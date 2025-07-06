package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

const defaultTitle = "nom"

var (
	appStyle        = lipgloss.NewStyle().Padding(1, 0, 0, 0).Margin(0)
	titleStyle      = list.DefaultStyles().Title.Margin(0).Width(5)
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
)

type TUIItem struct {
	Title     string
	FeedName  string
	URL       string
	ID        int
	Read      bool
	Favourite bool
}

func (i TUIItem) FilterValue() string { return fmt.Sprintf("%s||%s", i.Title, i.FeedName) }

type model struct {
	selectedArticle *int
	cfg             *config.Config
	commands        *Commands
	errors          []string
	list            list.Model
	help            help.Model
	viewport        viewport.Model
	lastRead        *list.Item
	lastReadIndex   int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// resize all views regardless of which is showing to keep consistent
	// when switching
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		x, y := appStyle.GetFrameSize()

		m.list.SetSize(msg.Width-x, msg.Height-y)

		m.viewport.Width = msg.Width - x
		footerHeight := lipgloss.Height(m.viewportHelp())
		m.viewport.Height = msg.Height - footerHeight

		return m, nil
	}

	if m.selectedArticle != nil {
		return updateViewport(msg, m)
	}

	return updateList(msg, m)
}

func (m model) View() string {
	var s string

	if m.selectedArticle == nil {
		s = listView(m)
	} else {
		s = viewportView(m)
	}

	return appStyle.Render(s)
}

func (m model) OpenLink(url string) tea.Cmd {
	hasOpener := false
	for _, o := range m.cfg.Openers {
		match, err := regexp.MatchString(o.Regex, url)
		if err != nil {
			return tea.Quit
		}

		if match {
			hasOpener = true
			cmdStr := fmt.Sprintf(o.Cmd, url)
			parts := strings.Fields(cmdStr)
			cmd := exec.Command(parts[0], parts[1:]...)

			if o.Takeover {
				return tea.ExecProcess(cmd, func(err error) tea.Msg {
					log.Println("OpenLink: takeover exec:", err)
					return nil
				})
			} else {
				return func() tea.Msg {
					if err := cmd.Run(); err != nil {
						log.Println("OpenLink: exec: ", err)
						return statusUpdate{
							status: err.Error(),
						}
					}
					return nil
				}
			}
		}
	}

	// if no opener, default to browser
	if !hasOpener {
		err := m.OpenInBrowser(url)
		if err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (m model) OpenInBrowser(url string) error {
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

func ItemToTUIItem(i store.Item) TUIItem {
	return TUIItem{
		ID:        i.ID,
		FeedName:  i.FeedName,
		Title:     i.Title,
		URL:       i.Link,
		Read:      i.Read(),
		Favourite: i.Favourite,
	}
}

func (c *Commands) TUI() error {
	debug := os.Getenv("DEBUGNOM")
	if debug != "" {
		f, err := tea.LogToFile(debug, "debug")
		if err != nil {
			fmt.Println("fatal:", err)
			os.Exit(1)
		}
		defer f.Close()
	}

	its, err := c.GetAllFeeds()
	if err != nil {
		return fmt.Errorf("commands List: %w", err)
	}

	var errorItems []ErrorItem
	// if no feeds in store, fetchAllFeeds, which will return previews
	if len(c.config.PreviewFeeds) > 0 {
		its, errorItems, err = c.fetchAllFeeds()
		if err != nil {
			return fmt.Errorf("[commands.go] TUI: %w", err)
		}
		// if no items, fetchAllFeeds and GetAllFeeds
	} else if len(its) == 0 {
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

	items := convertItems(its)

	es := []string{}
	for _, e := range errorItems {
		es = append(es, fmt.Sprintf("Error fetching %s: %s", e.FeedURL, e.Err))
	}

	if err := Render(items, c, es, c.config); err != nil {
		return fmt.Errorf("commands.TUI: %w", err)
	}

	return nil
}

func Render(items []list.Item, cmds *Commands, errors []string, cfg *config.Config) error {
	const defaultWidth = 20
	_, ts, _ := term.GetSize(int(os.Stdout.Fd()))
	_, y := appStyle.GetFrameSize()
	height := ts - y

	appStyle.Height(height)

	l := list.New(items, itemDelegate{theme: cfg.Theme}, defaultWidth, height)
	l.SetShowStatusBar(false)
	l.Title = defaultTitle
	l.Styles.Title = titleStyle.
		Background(lipgloss.Color(cfg.Theme.TitleColor)).
		Foreground(lipgloss.Color(cfg.Theme.TitleColorFg))

	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	l.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(cfg.Theme.FilterColor))

	l.Filter = CustomFilter(cfg.Filtering)

	ListKeyMap.SetOverrides(&l)

	vp := viewport.New(78, height)

	m := model{
		cfg:      cfg,
		commands: cmds,
		errors:   errors,
		help:     help.New(),
		list:     l,
		viewport: vp,
	}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("tui.Render: %w", err)
	}

	return nil
}
