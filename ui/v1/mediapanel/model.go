package mediapanel

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
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
	infoStrip      lipgloss.Style
	infoLabel      lipgloss.Style
	infoValue      lipgloss.Style
}

type Model struct {
	panels        []panel
	active        int
	keys          common.AppKeyMap
	styles        styles
	width         int
	height        int
	searchInput   textinput.Model
	searchFocused bool
	searchQuery   string
	infoOpen      bool
	infoViewport  viewport.Model
	infoSelection string
}

type panel struct {
	kind   common.ListKind
	lists  utils.Stack[medialist.Model]
	width  int
	height int
}

func NewModel(keys common.AppKeyMap) Model {
	kinds := []common.ListKind{
		common.Playlists,
		common.Tracks,
		common.Albums,
		common.Artists,
	}
	panels := common.MapSlice(kinds, newPanel)
	infoViewport := viewport.New()
	infoViewport.SoftWrap = true
	return Model{
		panels:       panels,
		active:       0,
		keys:         keys,
		styles:       defaultStyles(),
		searchInput:  newSearchInput(),
		infoViewport: infoViewport,
	}
}

func defaultStyles() styles {
	return styles{
		panel:    lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()),
		panelNav: lipgloss.NewStyle(),
		panelNavActive: lipgloss.NewStyle().
			Foreground(lipgloss.BrightCyan).
			Bold(true),
		panelNavMuted: lipgloss.NewStyle().
			Foreground(lipgloss.BrightBlack),
		searchLine:   lipgloss.NewStyle(),
		searchPrompt: lipgloss.NewStyle().Foreground(lipgloss.BrightBlack),
		searchValue:  lipgloss.NewStyle().Foreground(lipgloss.White),
		infoStrip: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			BorderLeft(false).
			BorderRight(false).
			BorderBottom(false).
			BorderForeground(lipgloss.BrightBlack).
			Padding(0, 1),
		infoLabel: lipgloss.NewStyle().
			Foreground(lipgloss.BrightBlack).
			Bold(true),
		infoValue: lipgloss.NewStyle().
			Foreground(lipgloss.White),
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
	m.syncInfoViewportLayout()
}

func (m *Model) InfoOpen() bool {
	return m.infoOpen
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
	m.syncInfoViewportLayout()
	return m.searchInput.Focus()
}

func (m *Model) blurSearch() {
	m.searchFocused = false
	m.searchInput.Blur()
	m.syncInfoViewportLayout()
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
	m.syncInfoContent(true)
	return m.activePanel().Prepare(m.searchQuery)
}

func (m *Model) clearSearchAndReload() tea.Cmd {
	m.searchQuery = ""
	m.searchInput.Reset()
	m.blurSearch()
	m.resetPanelsToRoot()
	m.syncInfoContent(true)
	return m.activePanel().Prepare(m.searchQuery)
}

func (m *Model) resetPanelsToRoot() {
	for i := range m.panels {
		m.panels[i].resetToRoot()
	}
}

func (m *Model) toggleInfo() tea.Cmd {
	if m.infoOpen {
		m.closeInfo()
		return nil
	}
	if _, ok := m.selectedEntity(); !ok {
		return m.activePanel().SetStatus("No item selected")
	}
	m.infoOpen = true
	m.refreshInfoForSelection(true)
	m.syncInfoViewportLayout()
	return nil
}

func (m *Model) closeInfo() {
	m.infoOpen = false
	m.infoSelection = ""
	m.infoViewport.SetContent("")
	m.infoViewport.GotoTop()
}

func (m *Model) syncInfoContent(resetScroll bool) {
	if !m.infoOpen {
		return
	}
	m.refreshInfoForSelection(resetScroll)
}

func (m *Model) refreshInfoForSelection(resetScroll bool) {
	entity, ok := m.selectedEntity()
	if !ok {
		m.infoSelection = ""
		m.infoViewport.SetContent("")
		if resetScroll {
			m.infoViewport.GotoTop()
		}
		return
	}

	signature := m.selectionSignature(entity)
	if !resetScroll && signature == m.infoSelection {
		return
	}

	m.infoSelection = signature
	m.infoViewport.SetContent(m.infoContentFor(entity))
	if resetScroll {
		m.infoViewport.GotoTop()
	}
}

func (m *Model) infoContentFor(entity common.Entity) string {
	sections := []string{
		m.styles.infoLabel.Render("Name"),
		m.styles.infoValue.Render(entity.Name),
	}
	if entity.Desc != "" {
		sections = append(sections,
			"",
			m.styles.infoLabel.Render("Description"),
			m.styles.infoValue.Render(entity.Desc),
		)
	}
	return strings.Join(sections, "\n")
}

func (m *Model) selectedEntity() (common.Entity, bool) {
	return m.activePanel().activeList().SelectedEntity()
}

func (m *Model) selectionSignature(entity common.Entity) string {
	return fmt.Sprintf("%s\x00%s\x00%s", entity.ID, entity.Name, entity.Desc)
}

func (m *Model) CloseInfo() {
	m.closeInfo()
}

func (p *panel) depth() int {
	return p.lists.Len()
}

func (p *panel) resetToRoot() {
	p.lists.Items = []medialist.Model{medialist.NewModel(p.kind)}
}
