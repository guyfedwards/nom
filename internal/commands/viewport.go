package commands

import (
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
			// reset cursor if last post is read and quit
			index := m.list.Index()
			length := len(m.list.Items())
			if index >= length && length >= 1 {
				m.list.Select(index - 1)
			}

			m.selectedArticle = nil
			cmds = append(cmds, m.UpdateList())

		case key.Matches(msg, ViewportKeyMap.OpenInBrowser):
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				m.selectedArticle = nil
				return m, m.list.NewStatusMessage("Error: failed to get article")
			}

			it := ItemToTUIItem(current)
			cmd = m.OpenLink(it.URL)

			cmds = append(cmds, cmd)
			if !current.Read() && m.commands.config.AutoRead {
				markRead(m)
			}

		case key.Matches(msg, ViewportKeyMap.Favourite):
			current, err := m.commands.store.GetItemByID(*m.selectedArticle)
			if err != nil {
				m.selectedArticle = nil
				return m, m.list.NewStatusMessage("Error: failed to get article")
			}
			err = m.commands.store.ToggleFavourite(current.ID)
			if err != nil {
				m.selectedArticle = nil
				return m, m.list.NewStatusMessage("Error toggling favourite")
			}

		case key.Matches(msg, ViewportKeyMap.Read):
			markRead(m)

		case key.Matches(msg, ViewportKeyMap.Prev):
			navIndex := m.getPrevIndex()
			items := m.list.Items()
			if m.isPrevOutOfBounds(navIndex) {
				return m, nil
			}

			m.list.Select(navIndex)
			item := items[navIndex]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				m.selectedArticle = nil
				return m, m.list.NewStatusMessage("Error rendering article")
			}

			m.viewport.SetContent(content)
			m.viewport.GotoTop()
			if m.commands.config.AutoRead && !m.commands.config.ShowRead {
				m.list.RemoveItem(m.list.Index())
			}

		case key.Matches(msg, ViewportKeyMap.Next):
			navIndex := m.getNextIndex()
			items := m.list.Items()
			if m.isNextOutOfBounds(navIndex, len(items)) {
				return m, nil
			}

			m.list.Select(navIndex)
			item := items[navIndex]
			id := item.(TUIItem).ID
			m.selectedArticle = &id

			content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
			if err != nil {
				m.selectedArticle = nil
				return m, m.list.NewStatusMessage("Error rendering article")
			}

			m.viewport.SetContent(content)
			m.viewport.GotoTop()
			if m.commands.config.AutoRead && !m.commands.config.ShowRead {
				m.list.RemoveItem(m.list.Index())
			}

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

func (m *model) isPrevOutOfBounds(i int) bool {
	if len(m.list.Items()) == 0 {
		return true
	}
	return i < 0
}

func (m *model) isNextOutOfBounds(i int, l int) bool {
	maxIndex := l - 1

	// when autoread and don't show read the first opened item doesn't exist in list
	if m.commands.config.AutoRead && !m.commands.config.ShowRead && i == 0 {
		maxIndex = l
	}

	if i < 0 || i > maxIndex || maxIndex < 0 || l == 0 {
		return true
	}
	return false
}

func (m *model) getNextIndex() int {
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

func (m *model) getPrevIndex() int {
	current := m.list.Index()
	if m.commands.config.AutoRead && !m.commands.config.ShowRead && current < len(m.list.Items()) {
		return m.list.Index()
	}

	if current == 0 {
		return 0
	}

	return m.list.Index() - 1
}

func viewportView(m model) string {
	return m.viewport.View() + "\n" + m.viewportHelp()
}

func (m model) viewportHelp() string {
	return helpStyle.Render(m.help.View(ViewportKeyMap))
}

func markRead(m model) (tea.Model, tea.Cmd) {
	if m.commands.config.AutoRead {
		return m, nil
	}
	current, err := m.commands.store.GetItemByID(*m.selectedArticle)
	if err != nil {
		m.selectedArticle = nil
		return m, m.list.NewStatusMessage("Error: failed to get article")
	}
	err = m.commands.store.ToggleRead(current.ID)
	if err != nil {
		m.selectedArticle = nil
		return m, m.list.NewStatusMessage("Error marking read")
	}

	if !m.commands.config.ShowRead {
		index := m.list.Index()

		if m.lastRead != nil && current.ID == (*m.lastRead).(TUIItem).ID {
			// un-read re-add post back to list
			m.list.InsertItem(index, *m.lastRead)
			m.lastReadIndex = index
			m.lastRead = nil
		} else {
			// remove post and store backup for un-read
			items := m.list.Items()
			item := items[index]
			m.list.RemoveItem(index)
			m.lastReadIndex = index
			m.lastRead = &item
		}
	}

	// trigger refresh to update read indication
	content, err := m.commands.GetGlamourisedArticle(*m.selectedArticle)
	if err != nil {
		m.selectedArticle = nil
		return m, m.list.NewStatusMessage("Error rendering article")
	}
	m.viewport.SetContent(content)
	return m, nil
}
