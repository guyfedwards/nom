package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/thebanri/limoni"
	"github.com/thebanri/limoni/widgets"

	"github.com/guyfedwards/nom/v2/internal/constants"
	"github.com/guyfedwards/nom/v2/internal/store"
)

// setItems replaces the list's items, keeping the filter and, as far as it
// can, the selection.
func (m *model) setItems(items []TUIItem) {
	m.items = items
	m.applyFilter()
}

// applyFilter works out which items the filter lets through, in the order it
// ranks them, and builds their rows.
func (m *model) applyFilter() {
	term := m.filterTerm
	if m.filtering {
		term = m.filter.Value()
	}
	m.visible = m.visible[:0]
	if term == "" {
		for i := range m.items {
			m.visible = append(m.visible, i)
		}
	} else {
		targets := make([]string, len(m.items))
		for i, it := range m.items {
			targets[i] = it.FilterValue()
		}
		m.visible = append(m.visible, CustomFilter(*m.cfg)(term, targets)...)
	}
	m.rows, m.selectedRows = m.rows[:0], m.selectedRows[:0]
	for n, i := range m.visible {
		it := m.items[i]
		var row string
		if it.FeedName == "" {
			row = fmt.Sprintf("%3d. %s", n+1, it.Title)
		} else {
			row = fmt.Sprintf("%3d. %s: %s", n+1, it.FeedName, it.Title)
		}
		// A favourite's star takes the column the selection's arrow does.
		mark := "  "
		if it.Favourite {
			mark = "* "
		}
		m.rows = append(m.rows, mark+row)
		m.selectedRows = append(m.selectedRows, "> "+row)
	}
	// The first row is selected from the start, as it always was.
	m.list.Select(max(min(m.list.Selected, len(m.visible)-1), 0))
}

// index is the selected row; -1 when the list is empty.
func (m *model) index() int {
	if len(m.visible) == 0 {
		return -1
	}
	return m.list.Selected
}

// selectedItem is the item on the selected row.
func (m *model) selectedItem() (TUIItem, bool) {
	i := m.index()
	if i < 0 {
		return TUIItem{}, false
	}
	return m.items[m.visible[i]], true
}

// removeRow takes the item on row i out of the list.
func (m *model) removeRow(i int) TUIItem {
	at := m.visible[i]
	it := m.items[at]
	m.items = append(m.items[:at], m.items[at+1:]...)
	m.applyFilter()
	return it
}

// insertRow puts an item back in the list, on row i.
func (m *model) insertRow(i int, it TUIItem) {
	at := len(m.items)
	if i < len(m.visible) {
		at = m.visible[i]
	}
	m.items = append(m.items[:at], append([]TUIItem{it}, m.items[at:]...)...)
	m.applyFilter()
}

// listRows is the list's rows for the List widget: read items dimmed,
// favourites in bold.
type listRows struct{ m *model }

func (r listRows) Len() int { return len(r.m.rows) }
func (r listRows) ItemAt(i int) string {
	if i == r.m.list.Selected {
		return r.m.selectedRows[i]
	}
	return r.m.rows[i]
}
func (r listRows) StyleAt(i int) limoni.Style {
	it := r.m.items[r.m.visible[i]]
	var s limoni.Style
	if it.Read {
		s = r.m.readStyle
	}
	if it.Favourite {
		s.Modifier |= limoni.ModifierBold
	}
	return s
}

func (m *model) UpdateList() limoni.Cmd {
	fs, err := m.commands.GetAllFeeds()
	if err != nil {
		return m.setStatus(fmt.Sprintf("Error: %s", err))
	}
	m.setItems(convertItems(fs))
	return nil
}

func sortList(m *model) limoni.Cmd {
	return func(context.Context) limoni.Msg {
		// reverse sorting order
		switch m.commands.config.Ordering {
		case constants.AscendingOrdering:
			m.commands.config.Ordering = constants.DescendingOrdering
		case constants.DescendingOrdering:
			m.commands.config.Ordering = constants.AscendingOrdering
		default:
			// noop
		}

		items, err := m.commands.GetAllFeeds()
		if err != nil {
			return statusUpdate{status: err.Error()}
		}
		return listUpdate{items: convertItems(items)}
	}
}

type refreshDone struct {
	items  []TUIItem
	errors []string
}

func refreshList(m *model) limoni.Cmd {
	return func(context.Context) limoni.Msg {
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

		return refreshDone{
			items:  convertItems(items),
			errors: es,
		}
	}
}

type listUpdate struct {
	status string
	items  []TUIItem
}

type statusUpdate struct {
	status string
}

// configEdited is the editor E opened closing.
type configEdited struct{ err error }

func updateList(msg limoni.Msg, m *model) limoni.UpdateResult {
	var cmds []limoni.Cmd

	switch msg := msg.(type) {
	case statusUpdate:
		cmds = append(cmds, m.setStatus(msg.status))
	case refreshDone:
		m.isRefreshing = false
		if !m.filtering {
			m.setItems(msg.items)
		}
		m.errors = msg.errors
		cmds = append(cmds, m.setStatus("Refreshed."))
	case listUpdate:
		m.isRefreshing = false
		if m.filtering {
			break
		}
		m.setItems(msg.items)
		cmds = append(cmds, m.setStatus(msg.status))
	case configEdited:
		if msg.err != nil {
			cmds = append(cmds, m.setStatus(msg.err.Error()))
			break
		}
		if err := m.cfg.Load(); err != nil {
			cmds = append(cmds, m.setStatus(err.Error()))
			break
		}
		m.applyTheme()
	case tickLoadMsg:
		if m.isRefreshing {
			loadchar := []rune{'⏐', '/', 'ー', '\\'}
			frame := int(msg)
			nextFrame := (frame + 1) % len(loadchar)
			m.status = fmt.Sprintf("Refreshing... %c", loadchar[frame])
			cmds = append(cmds, m.TickLoad(nextFrame))
		}
	case limoni.KeyPressMsg:
		return listKey(m, msg.Key)
	case limoni.PasteMsg:
		if m.filtering {
			m.filter.SetValue(m.filter.Value() + msg.Text)
			m.applyFilter()
		}
	case limoni.MouseWheelMsg, limoni.MousePressMsg:
		// The list scrolls and selects with the mouse by itself.
	default:
		return limoni.UpdateResult{}
	}
	return redraw(cmds...)
}

// listKey handles a key press in the list.
func listKey(m *model, key limoni.KeyEvent) limoni.UpdateResult {
	k := listKeys
	name := keyName(key)

	if k.ForceQuit.matches(name) {
		return limoni.UpdateResult{Quit: true}
	}
	if m.filtering {
		switch {
		case k.CancelFilter.matches(name):
			m.filtering, m.filterTerm = false, ""
			m.filter.SetValue("")
		case k.Accept.matches(name):
			m.filtering = false
			m.filterTerm = m.filter.Value()
		default:
			m.filter.HandleKey(key)
		}
		m.applyFilter()
		return redraw()
	}

	var cmds []limoni.Cmd
	switch {
	case m.fullHelp && (k.CloseFullHelp.matches(name) || k.Quit.matches(name)):
		// if help is showing, close help don't quit
		m.fullHelp = false

	case m.filterTerm != "" && k.ClearFilter.matches(name):
		m.filterTerm = ""
		m.filter.SetValue("")
		m.applyFilter()

	case k.Quit.matches(name):
		return limoni.UpdateResult{Quit: true}

	case k.ShowFullHelp.matches(name):
		m.fullHelp = true

	case k.Filter.matches(name):
		// Filtering starts from the top, where the best match will be.
		m.filtering = true
		m.filter.SetValue(m.filterTerm)
		m.list.Select(0)
		m.list.Offset = 0

	case k.Up.matches(name):
		m.list.Select(max(m.list.Selected-1, 0))
	case k.Down.matches(name):
		m.list.Select(min(m.list.Selected+1, max(len(m.visible)-1, 0)))
	case k.PrevPage.matches(name):
		m.list.Select(max(m.list.Selected-max(m.listH, 1), 0))
	case k.NextPage.matches(name):
		m.list.Select(min(m.list.Selected+max(m.listH, 1), max(len(m.visible)-1, 0)))
	case k.Top.matches(name):
		m.list.Select(0)
	case k.Bottom.matches(name):
		m.list.Select(max(len(m.visible)-1, 0))

	case k.Suspend.matches(name):
		return redraw(limoni.SuspendCmd())

	case k.Refresh.matches(name):
		if m.isRefreshing || m.filterTerm != "" {
			break
		}
		m.isRefreshing = true
		cmds = append(cmds, refreshList(m), m.TickLoad(0))

	case k.Read.matches(name):
		if cmd := markReadList(m); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case k.ToggleReads.matches(name):
		m.commands.config.ToggleShowRead()
		cmds = append(cmds, m.UpdateList())

	case k.MarkAllRead.matches(name):
		m.commands.store.MarkAllRead()
		cmds = append(cmds, m.UpdateList())

	case k.Favourite.matches(name):
		current, ok := m.selectedItem()
		if !ok {
			return redraw(m.setStatus("No items to favourite."))
		}
		if err := m.commands.store.ToggleFavourite(current.ID); err != nil {
			return redraw(m.setStatus(fmt.Sprintf("Error toggling favourite: %s", err)))
		}
		cmds = append(cmds, m.UpdateList())

	case k.ToggleFavourites.matches(name):
		if m.commands.config.ShowFavourites {
			cmds = append(cmds, m.setStatus(""))
		} else {
			cmds = append(cmds, m.setStatus("favourites"))
		}
		m.commands.config.ToggleShowFavourites()
		cmds = append(cmds, m.UpdateList())

	case k.OpenInBrowser.matches(name):
		current, ok := m.selectedItem()
		if !ok {
			return redraw(m.setStatus("No link selected."))
		}
		cmds = append(cmds, m.setStatus("Opening..."), m.OpenLink(current.URL))
		if !current.Read && m.commands.config.AutoRead {
			if cmd := markReadList(m); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case k.Sort.matches(name):
		if m.filterTerm != "" {
			break
		}
		if len(m.items) == 0 {
			return redraw(m.setStatus("No items to sort."))
		}
		cmds = append(cmds, sortList(m))

	case k.Open.matches(name):
		current, ok := m.selectedItem()
		if !ok {
			break
		}
		id := current.ID
		m.selectedArticle = &id
		cmds = append(cmds, m.openArticle()...)
		if m.selectedArticle != nil {
			cmds = append(cmds, m.UpdateList())
		}

	case k.EditConfig.matches(name):
		cmd := strings.Split(getEditor("NOMEDITOR", "VISUAL", "EDITOR"), " ")
		cmd = append(cmd, m.cfg.ConfigPath)
		execCmd := exec.Command(cmd[0], cmd[1:]...)
		return redraw(limoni.ExecCmd(execCmd, func(err error) limoni.Msg {
			return configEdited{err}
		}))

	default:
		return limoni.UpdateResult{}
	}

	return redraw(cmds...)
}

// statusText is what the status line says: the first error, the applied
// filter, or the last message.
func (m *model) statusText() string {
	if len(m.errors) > 0 {
		return m.errors[0]
	}
	if m.filterTerm != "" && !m.filtering && m.status == "" {
		return "filtering: " + m.filterTerm
	}
	return m.status
}

func listView(m *model, f *limoni.Frame) {
	area := f.Area()
	if area.Width < 8 || area.Height < 5 {
		return
	}
	help := []string{m.listShortHelp()}
	if m.fullHelp {
		help = m.listFullHelp()
	}
	help = append(help, "") // a blank row below, as the help always had
	body := drawHelp(f, area, help)
	if body.Height > 0 {
		body.Height-- // and one above
	}

	// The title, or the filter being typed, with the status beside it.
	titleRow := limoni.NewRect(area.X+2, area.Y+1, area.Width-2, 1)
	if m.filtering {
		f.RenderWidget(&limoni.Paragraph{Text: "Filter: ", Style: m.filterStyle}, limoni.NewRect(titleRow.X, titleRow.Y, 8, 1))
		f.RenderWidget(&widgets.TextInput{ID: "filter", State: m.filter, Focused: true},
			limoni.NewRect(titleRow.X+8, titleRow.Y, titleRow.Width-8, 1))
	} else {
		title := " " + defaultTitle + " "
		f.RenderWidget(&limoni.Paragraph{ID: "title", Text: title, Style: m.titleStyle}, limoni.NewRect(titleRow.X, titleRow.Y, uint16(len(title)), 1))
		status := m.statusText()
		x := titleRow.X + uint16(len(title)) + 2
		f.RenderWidget(&limoni.Paragraph{ID: "status", Text: status, Style: limoni.Style{Fg: limoni.Hex("#04B575")}},
			limoni.NewRect(x, titleRow.Y, area.Width-min(x, area.Width), 1))
	}

	// The items, and where the selection is among them.
	top := area.Y + 3
	// The list, a blank row, and the position under it.
	if body.Y+body.Height <= top+2 {
		return
	}
	listArea := limoni.NewRect(area.X+2, top, area.Width-2, body.Y+body.Height-top-2)
	focus := "items"
	if m.filtering {
		focus = "filter"
	}
	f.FocusManager.SetFocused(focus)
	m.listH = int(listArea.Height)
	if len(m.visible) == 0 {
		empty := "No items."
		if m.filterTerm != "" || m.filtering {
			empty = "Nothing matched."
		}
		f.RenderWidget(&limoni.Paragraph{Text: empty, Style: m.readStyle}, limoni.NewRect(listArea.X+2, listArea.Y, listArea.Width-2, 1))
		return
	}
	f.RenderWidget(&widgets.List{
		ID:            "items",
		Label:         "Items",
		Provider:      listRows{m},
		State:         m.list,
		SelectedStyle: m.selectedStyle,
	}, listArea)
	position := fmt.Sprintf("%d/%d", m.list.Selected+1, len(m.visible))
	f.RenderWidget(&limoni.Paragraph{ID: "position", Text: position, Style: m.readStyle},
		limoni.NewRect(area.X+4, listArea.Y+listArea.Height+1, area.Width-min(4, area.Width), 1))
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

func markReadList(m *model) limoni.Cmd {
	current, ok := m.selectedItem()
	if !ok {
		return m.setStatus("No items to mark.")
	}
	if err := m.commands.store.ToggleRead(current.ID); err != nil {
		return m.setStatus(fmt.Sprintf("Error marking read: %s", err))
	}
	return m.UpdateList()
}
