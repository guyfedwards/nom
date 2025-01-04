package commands

import (
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func updateViewport(msg tea.Msg, m model) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.help.Width = msg.Width

	case tea.ResumeMsg:
		return m, nil
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, ViewportKeyMap.Suspend):
			return m, tea.Suspend
		case key.Matches(msg, ViewportKeyMap.GotoStart):
			m.viewport.GotoTop()

		case key.Matches(msg, ViewportKeyMap.GotoEnd):
			m.viewport.GotoBottom()

		case key.Matches(msg, ViewportKeyMap.Escape):
			m.selectedArticle = nil

		case key.Matches(msg, ViewportKeyMap.OpenInBrowser):
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				return m, nil
			}

			it := ItemToTUIItem(current)
			cmd = m.commands.OpenLink(it.URL)
			cmds = append(cmds, cmd)

		case key.Matches(msg, ViewportKeyMap.Favourite):
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				return m, nil
			}
			err = m.commands.store.ToggleFavourite(current.ID)
			if err != nil {
				return m, tea.Quit
			}
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.Read):
			if m.commands.config.AutoRead {
				return m, nil
			}
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				return m, nil
			}
			err = m.commands.store.ToggleRead(current.ID)
			if err != nil {
				return m, tea.Quit
			}

			m.list.RemoveItem(m.list.Index())
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.Prev):
			debugIndex(&m)
			debugList(&m)
			navIndex := getPrevIndex(&m)
			debugNav(navIndex)
			items := m.list.Items()
			if isOutOfBounds(navIndex, len(items), &m) {
				return m, nil
			}

			m.list.Select(navIndex)
			item := items[navIndex]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				return m, tea.Quit
			}

			m.viewport.SetContent(content)
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.Next):
			debugIndex(&m)
			debugList(&m)
			navIndex := getNextIndex(&m)
			debugNav(navIndex)
			items := m.list.Items()
			if isOutOfBounds(navIndex, len(items), &m) {
				return m, nil
			}

			m.list.Select(navIndex)
			item := items[navIndex]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				return m, tea.Quit
			}

			m.viewport.SetContent(content)
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.Quit):
			return m, tea.Quit

		case key.Matches(msg, ViewportKeyMap.ShowFullHelp):
			m.help.ShowAll = !m.help.ShowAll
			if m.help.ShowAll {
				m.viewport.Height = m.viewport.Height + lipgloss.Height(m.help.ShortHelpView(ViewportKeyMap.ShortHelp())) - lipgloss.Height(m.help.FullHelpView(ViewportKeyMap.FullHelp()))
			} else {
				m.viewport.Height = m.viewport.Height + lipgloss.Height(m.help.FullHelpView(ViewportKeyMap.FullHelp())) - lipgloss.Height(m.help.ShortHelpView(ViewportKeyMap.ShortHelp()))
			}
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func isOutOfBounds(i int, l int, m *model) bool {
	length := l - 1

	// when autoread and don't show read the first opened item doesn't exist in list
	if m.commands.config.AutoRead && !m.commands.config.ShowRead {
		length = l
	}

	if i < 0 || i > length || length <= 0 || (i == 0 && length <= 0) {
		return true
	}
	return false
}

func getNextIndex(m *model) int {
	if m.commands.config.AutoRead && !m.commands.config.ShowRead {
		return m.list.Index()
	}

	// check for favorite within post
	current, err := m.commands.store.GetItemByID(*m.selectedArticle)
	if err != nil {
		return m.list.Index()
	}
	if !m.commands.config.AutoRead && current.Read() && !m.commands.config.ShowRead {
		return m.list.Index()
	}

	return m.list.Index() + 1
}

func getPrevIndex(m *model) int {
	if m.commands.config.AutoRead && !m.commands.config.ShowRead {
		return m.list.Index()
	}

	return m.list.Index() - 1
}

func debugIndex(m *model) {
	log.Printf("Index: %d", m.list.Index())
}

func debugNav(i int) {
	log.Printf("Nav: %d", i)
}

func debugList(m *model) {
	arr := []string{}

	for _, item := range m.list.Items() {
		arr = append(arr, item.(TUIItem).Title)
	}
	log.Println(strings.Join(arr, ","))
}

func viewportView(m model) string {
	return m.viewport.View() + "\n" + m.viewportHelp()
}

func (m model) viewportHelp() string {
	return helpStyle.Render(m.help.View(ViewportKeyMap))
}
