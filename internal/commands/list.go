package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

var (
	itemStyle              = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle      = lipgloss.NewStyle().PaddingLeft(2)
	readStyle              = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("240"))
	selectedReadStyle      = lipgloss.NewStyle().PaddingLeft(2)
	favouriteStyle         = itemStyle.PaddingLeft(2).Bold(true)
	selectedFavouriteStyle = selectedItemStyle.Bold(true)
	helpStyle              = list.DefaultStyles().
				HelpStyle.
				PaddingLeft(4).
				PaddingBottom(1).
				Foreground(lipgloss.Color("#4A4A4A"))
)

type itemDelegate struct {
	theme config.Theme
}

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
		fn = func(s ...string) string {
			if i.Read {
				return favouriteStyle.Foreground(lipgloss.Color("240")).Render("* " + strings.Join(s, " "))
			}
			return favouriteStyle.Render("* " + strings.Join(s, " "))
		}
	}

	if index == m.Index() {
		fn = func(s ...string) string {
			if i.Favourite {
				return selectedFavouriteStyle.Foreground(lipgloss.Color(d.theme.SelectedItemColor)).Render("> " + strings.Join(s, " "))
			}
			if i.Read {
				return selectedReadStyle.Foreground(lipgloss.Color(d.theme.SelectedItemColor)).Render("> " + strings.Join(s, " "))
			}
			return selectedItemStyle.Foreground(lipgloss.Color(d.theme.SelectedItemColor)).Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func (m *model) UpdateList() tea.Cmd {
	fs, err := m.commands.GetAllFeeds()
	if err != nil {
		return tea.Quit
	}

	cmd := m.list.SetItems(convertItems(fs))

	return cmd
}

func refreshList(m model) func() tea.Msg {
	return func() tea.Msg {
		var errorItems []ErrorItem
		es := []string{}
		var err error
		var items []store.Item
		// if no feeds in store, fetchAllFeeds, which will return previews
		if len(m.commands.config.PreviewFeeds) > 0 {
			items, errorItems, err = m.commands.fetchAllFeeds()
			if err != nil {
				es = append(es, fmt.Errorf("[tui.go] updateList: %w", err).Error())
			}
			// if no items, fetchAllFeeds and GetAllFeeds
		} else if len(items) == 0 {
			_, errorItems, err = m.commands.fetchAllFeeds()
			if err != nil {
				es = append(es, fmt.Errorf("[tui.go] updateList: %w", err).Error())
			}

			// refetch for consistent data across calls
			items, err = m.commands.GetAllFeeds()
			if err != nil {
				es = append(es, fmt.Errorf("[tui.go] updateList: %w", err).Error())
			}
		}

		for _, e := range errorItems {
			es = append(es, fmt.Sprintf("Error fetching %s: %s", e.FeedURL, e.Err))
		}

		m.errors = es
		// return tea.Batch(m.list.SetItems(convertItems(items)), m.list.NewStatusMessage("Refreshed."))
		// get date in YYYY-MM-DD hh:mm:ss
		now := time.Now().Format("2006-01-02 15:04:05")
		return listUpdate{
			items:  convertItems(items),
			status: fmt.Sprintf("Refreshed at %s.", now),
		}
	}
}

type listUpdate struct {
	status string
	items  []list.Item
}

func updateList(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case listUpdate:
		m.list.SetItems(msg.items)
		m.list.NewStatusMessage(msg.status)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ListKeyMap.Refresh):
			if m.list.SettingFilter() || m.list.IsFiltered() {
				break
			}

			cmds = append(cmds, refreshList(m))

		case key.Matches(msg, ListKeyMap.Read):
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

		case key.Matches(msg, ListKeyMap.ToggleReads):
			if m.list.SettingFilter() {
				break
			}

			m.commands.config.ToggleShowRead()
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ListKeyMap.MarkAllRead):
			if m.list.SettingFilter() {
				break
			}

			m.commands.store.MarkAllRead()
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ListKeyMap.Favourite):
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

			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ListKeyMap.ToggleFavourites):
			if m.list.SettingFilter() {
				break
			}

			if m.commands.config.ShowFavourites {
				m.list.NewStatusMessage("")
			} else {
				m.list.NewStatusMessage("favourites")
			}

			m.commands.config.ToggleShowFavourites()
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.OpenInBrowser):
			if m.list.SettingFilter() {
				break
			}

			item := m.list.SelectedItem()
			if item == nil {
				return m, m.list.NewStatusMessage("No link selected.")
			}

			current := item.(TUIItem)
			cmd = m.commands.OpenLink(current.URL)
			cmds = append(cmds, cmd)

		case key.Matches(msg, ListKeyMap.Open):
			if m.list.SettingFilter() {
				m.list.FilterInput.Blur()
				break
			}
			i, ok := m.list.SelectedItem().(TUIItem)
			if ok {
				m.selectedArticle = &i.ID

				m.viewport.GotoTop()

				content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
				if err != nil {
					return m, tea.Quit
				}

				m.viewport.SetContent(content)

				cmds = append(cmds, m.UpdateList())
			}

		case key.Matches(msg, ListKeyMap.EditConfig):
			filePath := m.cfg.ConfigPath

			cmd := strings.Split(getEditor("NOMEDITOR", "EDITOR"), " ")
			cmd = append(cmd, filePath)

			execCmd := exec.Command(cmd[0], cmd[1:]...)
			return m, tea.ExecProcess(execCmd, func(err error) tea.Msg {
				if err != nil {
					m.list.NewStatusMessage(err.Error())
					return nil
				}

				err = m.cfg.Load()
				if err != nil {
					m.list.NewStatusMessage(err.Error())
					return nil
				}
				return nil
			})
		}
	}

	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func listView(m model) string {
	if len(m.errors) > 0 {
		m.list.NewStatusMessage(m.errors[0])
	} else if m.list.IsFiltered() {
		m.list.NewStatusMessage("filtering: " + m.list.FilterInput.Value())
	}

	return "\n" + m.list.View()
}

func getEditor(vars ...string) string {
	for _, e := range vars {
		val := os.Getenv(e)
		if val != "" {
			return val
		}
	}

	return "nano"
}
