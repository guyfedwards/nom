package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	"github.com/guyfedwards/nom/v2/internal/store"
)

var (
	appStyle               = lipgloss.NewStyle().Padding(0).Margin(0)
	titleStyle             = list.DefaultStyles().Title.Margin(1, 0, 0, 0).Width(5)
	itemStyle              = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle      = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	readStyle              = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("240"))
	selectedReadStyle      = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	favouriteStyle         = itemStyle.Copy().PaddingLeft(2).Bold(true)
	selectedFavouriteStyle = selectedItemStyle.Copy().Bold(true)
	paginationStyle        = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle              = list.DefaultStyles().
				HelpStyle.
				PaddingLeft(4).
				PaddingBottom(1).
				Foreground(lipgloss.Color("#4A4A4A"))
)

type TUIItem struct {
	ID        int
	Title     string
	FeedName  string
	URL       string
	Read      bool
	Favourite bool
}

func (i TUIItem) FilterValue() string { return i.Title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(TUIItem)
	if !ok {
		return
	}

	var str string
	if i.FeedName == "" {
		str = fmt.Sprintf("%3d. %s", index+1, i.Title)
	} else {
		str = fmt.Sprintf("%3d. %s: %s", index+1, i.FeedName, i.Title)
	}

	fn := itemStyle.Render

	if i.Read {
		fn = readStyle.Render
	}

	if i.Favourite {
		fn = func(s string) string {
			return favouriteStyle.Render("* " + s)
		}
	}

	if index == m.Index() {
		fn = func(s string) string {
			if i.Read {
				return selectedReadStyle.Render("> " + s)
			}
			if i.Favourite {
				return selectedFavouriteStyle.Render("* " + s)
			}
			return selectedItemStyle.Render("> " + s)
		}
	}

	fmt.Fprint(w, fn(str))
}

type model struct {
	list            list.Model
	commands        Commands
	selectedArticle *int
	viewport        viewport.Model
	prevKeyWasG     bool
	errors          []string
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

func (m *model) UpdateList() tea.Cmd {
	fs, err := m.commands.GetAllFeeds()
	if err != nil {
		return tea.Quit
	}

	cmd := m.list.SetItems(convertItems(fs))

	return cmd
}

func updateList(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {

		case "ctrl+c":
			return m, tea.Quit

		case "r":
			if m.list.SettingFilter() {
				break
			}

			_, errorItems, err := m.commands.fetchAllFeeds()
			if err != nil {
				return m, tea.Quit
			}

			// refetch for consistent data across calls
			items, err := m.commands.GetAllFeeds()
			if err != nil {
				return m, tea.Quit
			}

			es := []string{}
			for _, e := range errorItems {
				es = append(es, fmt.Sprintf("Error fetching %s: %s", e.FeedURL, e.Err))
			}

			m.list.SetItems(convertItems(items))
			m.errors = es

			return m, m.list.NewStatusMessage("Refreshed.")

		case "m":
			if m.list.SettingFilter() {
				break
			}

			if len(m.list.Items()) == 0 {
				return m, m.list.NewStatusMessage("No items to mark.")
			}

			current := m.list.SelectedItem().(TUIItem)
			err := m.commands.store.ToggleRead(current.ID)
			if err != nil {
				return m, tea.Quit
			}
			m.UpdateList()

		case "M":
			if m.list.SettingFilter() {
				break
			}

			m.commands.config.ToggleShowRead()
			m.UpdateList()

		case "f":
			if m.list.SettingFilter() {
				break
			}

			if len(m.list.Items()) == 0 {
				return m, m.list.NewStatusMessage("No items to favourite.")
			}

			current := m.list.SelectedItem().(TUIItem)
			err := m.commands.store.ToggleFavourite(current.ID)
			if err != nil {
				return m, tea.Quit
			}
			m.UpdateList()

		case "F":
			if m.list.SettingFilter() {
				break
			}

			if m.commands.config.ShowFavourites {
				m.list.NewStatusMessage("")
			} else {
				m.list.NewStatusMessage("favourites")
			}

			m.commands.config.ToggleShowFavourites()
			m.UpdateList()

		case "o":
			if m.list.SettingFilter() {
				break
			}
			current := m.list.SelectedItem().(TUIItem)
			err := m.commands.OpenLink(current.URL)
			if err != nil {
				return m, tea.Quit
			}

		case "enter":
			i, ok := m.list.SelectedItem().(TUIItem)
			if ok {
				m.selectedArticle = &i.ID

				m.viewport.GotoTop()

				content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
				if err != nil {
					return m, tea.Quit
				}

				m.viewport.SetContent(content)

				cmd = m.UpdateList()
				cmds = append(cmds, cmd)
			}
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func updateViewport(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

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
		case "esc", "q":
			m.selectedArticle = nil

		case "o":
			current := m.list.SelectedItem().(TUIItem)
			err := m.commands.OpenLink(current.URL)
			if err != nil {
				return m, tea.Quit
			}

		case "h":
			current := m.list.Index()
			if current-1 < 0 {
				return m, nil
			}

			m.list.Select(current - 1)
			items := m.list.Items()
			item := items[current-1]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				return m, tea.Quit
			}

			m.viewport.SetContent(content)
			cmd = m.UpdateList()
			cmds = append(cmds, cmd)

		case "l":
			current := m.list.Index()
			items := m.list.Items()
			if current+1 >= len(items) {
				return m, nil
			}

			m.list.Select(current + 1)
			item := items[current+1]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				return m, tea.Quit
			}

			m.viewport.SetContent(content)
			cmd = m.UpdateList()
			cmds = append(cmds, cmd)
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
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

func listView(m model) string {
	if len(m.errors) > 0 {
		m.list.NewStatusMessage(m.errors[0])
	}

	return "\n" + m.list.View()
}

func viewportView(m model) string {
	return m.viewport.View() + "\n" + m.viewportHelp()
}

func (m model) viewportHelp() string {
	return helpStyle.Render("\nk/j up/down • h/l prev/next • gg/G top/bot • o open • q/esc back")
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

const defaultTitle = "nom"

func Render(items []list.Item, cmds Commands, errors []string) error {
	const defaultWidth = 20
	_, ts, _ := term.GetSize(int(os.Stdout.Fd()))
	_, y := appStyle.GetFrameSize()
	height := ts - y

	appStyle.Height(height)

	l := list.New(items, itemDelegate{}, defaultWidth, height)
	l.SetShowStatusBar(false)
	l.Title = defaultTitle
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle
	// remove some extra keys from next/prev as used for other things
	l.KeyMap.NextPage.SetKeys("right", "l", "pgdown")
	l.KeyMap.PrevPage.SetKeys("left", "h", "pgup")

	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("m"),
				key.WithHelp("m", "toggle read"),
			),
			key.NewBinding(
				key.WithKeys("M"),
				key.WithHelp("M", "show/hide read"),
			),
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "toggle favourite"),
			),
			key.NewBinding(
				key.WithKeys("F"),
				key.WithHelp("F", "show/hide all favourites"),
			),
			key.NewBinding(
				key.WithKeys("r"),
				key.WithHelp("r", "refresh feed"),
			),
		}
	}
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("o"),
				key.WithHelp("o", "open link"),
			),
		}
	}

	vp := viewport.New(78, height)

	m := model{list: l, commands: cmds, viewport: vp, errors: errors}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return fmt.Errorf("tui.Render: %w", err)
	}

	return nil
}
