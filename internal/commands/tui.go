package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/pkg/browser"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/guyfedwards/nom/internal/rss"
)

const listHeight = 14

var (
	appStyle          = lipgloss.NewStyle().Padding(0).Margin(0)
	titleStyle        = list.DefaultStyles().Title.Margin(1, 0, 0, 0)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().
				HelpStyle.
				PaddingLeft(4).
				PaddingBottom(1).
				Foreground(lipgloss.Color("#4A4A4A"))
)

type Item struct {
	Title    string
	FeedName string
	URL      string
}

func (i Item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(Item)
	if !ok {
		return
	}

	var str string
	if i.FeedName == "" {
		str = fmt.Sprintf("%d. %s", index+1, i.Title)
	} else {
		str = fmt.Sprintf("%d. %s: %s", index+1, i.FeedName, i.Title)
	}

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s string) string {
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list            list.Model
	commands        Commands
	selectedArticle string
	viewport        viewport.Model
	prevKeyWasG     bool
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

	if m.selectedArticle != "" {
		return updateViewport(msg, m)
	}

	return updateList(msg, m)
}

func updateList(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {

		case "ctrl+c":
			return m, tea.Quit

		case "r":
			rss, err := m.commands.fetchAllFeeds(true)
			if err != nil {
				return m, tea.Quit
			}

			m.list.SetItems(getItemsFromRSS(rss))

			return m, nil

		case "enter":
			i, ok := m.list.SelectedItem().(Item)
			if ok {
				m.selectedArticle = i.Title

				m.viewport.GotoTop()

				content, err := m.commands.FindGlamourisedArticle(m.selectedArticle)
				if err != nil {
					return m, tea.Quit
				}

				m.viewport.SetContent(content)
			}

			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func updateViewport(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "g":
			if m.prevKeyWasG {
				m.viewport.GotoTop()
				m.prevKeyWasG = false
			} else {
				m.prevKeyWasG = true
			}
		case "G":
			m.viewport.GotoBottom()
		case "O":
			browser.OpenURL(m.list.SelectedItem().(Item).URL)
		case "esc", "q":
			m.selectedArticle = ""

		case "ctrl+c":
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	var s string

	if m.selectedArticle == "" {
		s = listView(m)
	} else {
		s = viewportView(m)
	}

	return appStyle.Render(s)
}

func listView(m model) string {
	return "\n" + m.list.View()
}

func viewportView(m model) string {
	return m.viewport.View() + "\n" + m.viewportHelp()
}

func (m model) viewportHelp() string {
	return helpStyle.Render("\n↑/k up • ↓/j down • gg top • G bottom • O open browser • q/esc back")
}

func RSSToItem(c rss.Item) Item {
	return Item{
		FeedName: c.FeedName,
		Title:    c.Title,
		URL:      c.Link,
	}
}

func Render(items []list.Item, cmds Commands) error {
	const defaultWidth = 20
	_, ts, _ := term.GetSize(int(os.Stdout.Fd()))
	_, y := appStyle.GetFrameSize()
	height := ts - y

	appStyle.Height(height)

	l := list.New(items, itemDelegate{}, defaultWidth, height)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Title = "nom 🍜"
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "refresh cache"),
			),
		}
	}

	vp := viewport.New(78, height)

	m := model{list: l, commands: cmds, viewport: vp}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("tui.Render: %w", err)
	}

	return nil
}
