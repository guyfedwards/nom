package commands

import (
	"fmt"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"
)

// articleWidth is the widest the article's text runs, as glamour wrapped it.
const articleWidth = 80

// openArticle shows the selected article from its top, and starts fetching
// its pictures.
func (m *model) openArticle() []limoni.Cmd {
	content, err := m.commands.GetArticleMarkdown(*m.selectedArticle)
	if err != nil {
		m.selectedArticle = nil
		return []limoni.Cmd{m.setStatus(fmt.Sprintf("Error opening article: %s", err))}
	}
	m.articleOffset = 0
	if m.images != nil {
		m.images.forget()
	}
	m.article.Content = content
	return m.fetchImages()
}

// fetchImages starts downloading the pictures in the article.
func (m *model) fetchImages() []limoni.Cmd {
	if m.images == nil {
		return nil
	}
	return m.images.fetch(widgets.MarkdownImageSources(m.article.Content), "nom/"+m.cfg.Version)
}

// refreshArticle draws the article again with its state changed (read,
// favourite), where it was.
func (m *model) refreshArticle() limoni.Cmd {
	content, err := m.commands.GetArticleMarkdown(*m.selectedArticle)
	if err != nil {
		m.selectedArticle = nil
		return m.setStatus("Error rendering article")
	}
	m.article.Content = content
	return nil
}

func updateViewport(msg limoni.Msg, m *model) limoni.UpdateResult {
	var cmds []limoni.Cmd
	switch msg := msg.(type) {
	case limoni.KeyPressMsg:
		return articleKey(m, msg.Key)
	case limoni.MouseWheelMsg, limoni.MousePressMsg, limoni.ResizeMsg:
		// The article scrolls with the wheel by itself.
	case statusUpdate:
		cmds = append(cmds, m.setStatus(msg.status))
	default:
		return limoni.UpdateResult{}
	}
	return redraw(cmds...)
}

func articleKey(m *model, key limoni.KeyEvent) limoni.UpdateResult {
	k := articleKeys
	name := keyName(key)
	page := max(m.articleH, 1)
	var cmds []limoni.Cmd

	switch {
	case k.Quit.matches(name):
		return limoni.UpdateResult{Quit: true}
	case k.Suspend.matches(name):
		return redraw(limoni.SuspendCmd())

	case k.Up.matches(name):
		m.articleOffset = max(m.articleOffset-1, 0)
	case k.Down.matches(name):
		m.articleOffset++ // Markdown stops it at the end
	case k.PageUp.matches(name):
		m.articleOffset = max(m.articleOffset-page, 0)
	case k.PageDown.matches(name):
		m.articleOffset += page
	case k.HalfPageUp.matches(name):
		m.articleOffset = max(m.articleOffset-page/2, 0)
	case k.HalfPageDown.matches(name):
		m.articleOffset += page / 2
	case k.GotoStart.matches(name):
		m.articleOffset = 0
	case k.GotoEnd.matches(name):
		m.articleOffset = 1 << 30

	case k.Escape.matches(name):
		// reset cursor if last post is read and quit
		index := m.list.Selected
		length := len(m.visible)
		if index >= length && length >= 1 {
			m.list.Select(index - 1)
		}
		m.selectedArticle = nil
		m.fullHelp = false
		if m.images != nil {
			m.images.forget()
		}
		cmds = append(cmds, m.UpdateList())

	case k.OpenInBrowser.matches(name):
		current, err := m.commands.store.GetItemByID(*m.selectedArticle)
		if err != nil {
			m.selectedArticle = nil
			return redraw(m.setStatus("Error: failed to get article"))
		}
		cmds = append(cmds, m.OpenLink(ItemToTUIItem(current).URL))
		if !current.Read() && m.commands.config.AutoRead {
			if cmd := markRead(m); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case k.Favourite.matches(name):
		current, err := m.commands.store.GetItemByID(*m.selectedArticle)
		if err != nil {
			m.selectedArticle = nil
			return redraw(m.setStatus("Error: failed to get article"))
		}
		if err := m.commands.store.ToggleFavourite(current.ID); err != nil {
			m.selectedArticle = nil
			return redraw(m.setStatus("Error toggling favourite"))
		}

	case k.Read.matches(name):
		if cmd := markRead(m); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case k.Prev.matches(name):
		navIndex := m.getPrevIndex()
		if m.isPrevOutOfBounds(navIndex) {
			return limoni.UpdateResult{}
		}
		cmds = append(cmds, m.showRow(navIndex)...)

	case k.Next.matches(name):
		navIndex := m.getNextIndex()
		if m.isNextOutOfBounds(navIndex, len(m.visible)) {
			return limoni.UpdateResult{}
		}
		cmds = append(cmds, m.showRow(navIndex)...)

	case k.ShowFullHelp.matches(name):
		m.fullHelp = !m.fullHelp

	default:
		return limoni.UpdateResult{}
	}
	return redraw(cmds...)
}

// showRow opens the article on row i of the list, as next and prev do.
func (m *model) showRow(i int) []limoni.Cmd {
	m.list.Select(i)
	id := m.items[m.visible[i]].ID
	m.selectedArticle = &id
	cmds := m.openArticle()
	if m.selectedArticle == nil {
		return cmds
	}
	if m.commands.config.AutoRead && !m.commands.config.ShowRead {
		m.removeRow(m.list.Selected)
	}
	return cmds
}

func (m *model) isPrevOutOfBounds(i int) bool {
	if len(m.visible) == 0 {
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
		return m.list.Selected
	}

	// check for favorite within post
	current, err := m.commands.store.GetItemByID(*m.selectedArticle)
	if err != nil {
		return m.list.Selected
	}
	if !m.commands.config.AutoRead && current.Read() && !m.commands.config.ShowRead {
		return m.list.Selected
	}

	return m.list.Selected + 1
}

func (m *model) getPrevIndex() int {
	current := m.list.Selected
	if m.commands.config.AutoRead && !m.commands.config.ShowRead && current < len(m.visible) {
		return m.list.Selected
	}

	if current == 0 {
		return 0
	}

	return m.list.Selected - 1
}

func viewportView(m *model, f *limoni.Frame) {
	area := f.Area()
	if area.Width < 8 || area.Height < 3 {
		return
	}
	help := []string{articleShortHelp()}
	if m.fullHelp {
		help = articleFullHelp()
	}
	help = append(help, "")
	body := drawHelp(f, area, help)
	if body.Height < 2 {
		return
	}
	// One row of space above, and the text in a column no wider than
	// glamour wrapped it, two columns in from the edge.
	width := min(int(body.Width)-4, articleWidth)
	text := limoni.NewRect(body.X+2, body.Y+1, uint16(max(width, 1)), body.Height-2)
	m.articleH = int(text.Height)
	f.FocusManager.SetFocused("article")
	f.RenderWidget(m.article, text)
	if m.status != "" {
		f.RenderWidget(&limoni.Paragraph{ID: "status", Text: m.status, Style: limoni.Style{Fg: limoni.Hex("#04B575")}},
			limoni.NewRect(body.X+2, body.Y+body.Height-1, body.Width-min(2, body.Width), 1))
	}
}

func markRead(m *model) limoni.Cmd {
	if m.commands.config.AutoRead {
		return nil
	}
	current, err := m.commands.store.GetItemByID(*m.selectedArticle)
	if err != nil {
		m.selectedArticle = nil
		return m.setStatus("Error: failed to get article")
	}
	err = m.commands.store.ToggleRead(current.ID)
	if err != nil {
		m.selectedArticle = nil
		return m.setStatus("Error marking read")
	}

	if !m.commands.config.ShowRead {
		index := m.list.Selected

		if m.lastRead != nil && current.ID == m.lastRead.ID {
			// un-read re-add post back to list
			m.insertRow(index, *m.lastRead)
			m.lastReadIndex = index
			m.lastRead = nil
		} else if index < len(m.visible) {
			// remove post and store backup for un-read
			item := m.removeRow(index)
			m.lastReadIndex = index
			m.lastRead = &item
		}
	}

	// trigger refresh to update read indication
	return m.refreshArticle()
}
