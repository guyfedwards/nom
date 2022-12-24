package commands

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guyfedwards/nom/internal/rss"
)

const listHeight = 14

var (
	titleStyle         = lipgloss.NewStyle().MarginLeft(2)
	itemStyle          = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle  = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle    = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle          = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle      = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	viewportTitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.Copy().BorderStyle(b)
	}()
)

type Item struct {
	Title string
	URL   string
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

	str := fmt.Sprintf("%d. %s", index+1, i.Title)

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
	quitting        bool
	viewport        viewport.Model
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.selectedArticle != "" {
		return updateViewport(msg, m)
	}

	return updateList(msg, m)
}

func updateList(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// case tea.WindowSizeMsg:
	// 	m.list.SetWidth(msg.Width)
	// 	return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {

		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "enter":
			i, ok := m.list.SelectedItem().(Item)
			if ok {
				m.selectedArticle = i.Title

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
	// case tea.WindowSizeMsg:
	// 	return m, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "esc", "q":
			m.selectedArticle = ""

		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		return quitTextStyle.Render("Bye!")
	}

	var s string

	if m.selectedArticle == "" {
		s = listView(m)
	} else {
		s = viewportView(m)
	}

	return s
}

func listView(m model) string {
	return "\n" + m.list.View()
}

func viewportView(m model) string {
	return m.viewport.View() + m.viewportHelp()
}

func (m model) viewportHelp() string {
	return helpStyle.Render("\n↑/k up • ↓/j down • q/esc: Back\n")
}

func RSSToItem(c rss.Item) Item {
	return Item{
		Title: c.Title,
		URL:   c.Link,
	}
}

func Render(items []list.Item, cmds Commands) error {
	const defaultWidth = 20

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	vp := viewport.New(78, 20)
	// vp.HighPerformanceRendering = true

	m := model{list: l, commands: cmds, viewport: vp}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		return fmt.Errorf("tui.Render: %w", err)
	}

	return nil
}
