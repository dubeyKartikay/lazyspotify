package mediapanel

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/medialist"
)

type styles struct {
	panel          lipgloss.Style
	panelNav       lipgloss.Style
	panelNavActive lipgloss.Style
	panelNavMuted  lipgloss.Style
	searchLine     lipgloss.Style
	searchPrompt   lipgloss.Style
	searchValue    lipgloss.Style
}

type Model struct {
	panels        []panel
	active        int
	styles        styles
	width         int
	height        int
	searchInput   textinput.Model
	searchFocused bool
	searchQuery   string
}

type panel struct {
	kind   common.ListKind
	lists  utils.Stack[medialist.Model]
	width  int
	height int
}

func NewModel() Model {
	kinds := []common.ListKind{
		common.Playlists,
		common.Tracks,
		common.Albums,
		common.Artists,
	}
	panels := common.MapSlice(kinds, newPanel)
	return Model{
		panels:      panels,
		active:      0,
		styles:      defaultStyles(),
		searchInput: newSearchInput(),
	}
}

func defaultStyles() styles {
	return styles{
		panel:    lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()),
		panelNav: lipgloss.NewStyle(),
		panelNavActive: lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true),
		panelNavMuted: lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")),
		searchLine:   lipgloss.NewStyle(),
		searchPrompt: lipgloss.NewStyle().Foreground(lipgloss.Color("8")),
		searchValue:  lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	}
}

func newSearchInput() textinput.Model {
	input := textinput.New()
	input.Prompt = "Search: "
	input.Placeholder = "search"
	input.CharLimit = 120
	return input
}

func newPanel(kind common.ListKind) panel {
	p := panel{kind: kind}
	p.lists.Push(medialist.NewModel(kind))
	return p
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.searchInput.SetWidth(max(0, width-6))
}

func (m *Model) activePanel() *panel {
	return &m.panels[m.active]
}

func (m *Model) panelForKind(kind common.ListKind) *panel {
	for i := range m.panels {
		if m.panels[i].kind == kind {
			return &m.panels[i]
		}
	}
	return m.activePanel()
}

func (m *Model) focusSearch() tea.Cmd {
	m.searchFocused = true
	m.searchInput.SetValue(m.searchQuery)
	m.searchInput.CursorEnd()
	return m.searchInput.Focus()
}

func (m *Model) blurSearch() {
	m.searchFocused = false
	m.searchInput.Blur()
}

func (m *Model) submitSearch() tea.Cmd {
	query := strings.TrimSpace(m.searchInput.Value())
	m.searchInput.SetValue(query)
	m.blurSearch()
	if query == "" {
		return m.clearSearchAndReload()
	}
	return m.applySearch(query)
}

func (m *Model) applySearch(query string) tea.Cmd {
	m.searchQuery = query
	m.searchInput.SetValue(query)
	m.resetPanelsToRoot()
	return m.activePanel().Prepare(m.searchQuery)
}

func (m *Model) clearSearchAndReload() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.Reset()
	m.blurSearch()
	m.resetPanelsToRoot()
	return m.activePanel().Prepare(m.searchQuery)
}

func (m *Model) resetPanelsToRoot() {
	for i := range m.panels {
		m.panels[i].resetToRoot()
	}
}

func (p *panel) depth() int {
	return p.lists.Len()
}

func (p *panel) resetToRoot() {
	p.lists.Items = []medialist.Model{medialist.NewModel(p.kind)}
}
