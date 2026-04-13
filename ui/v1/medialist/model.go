package medialist

import (
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/paginator"
	"charm.land/lipgloss/v2"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
)

type State int

const (
	Initialized State = iota
	Loading
	Ready
)

type PaginationState struct {
	CurrentPage int
	TotalPages  int
	TotalItems  int
	HasNext     bool
	HasPrev     bool
	NextCursor  string
	PrevCursor  string
	history     []string
}

type Model struct {
	kind       common.ListKind
	items      []common.Entity
	list       list.Model
	pager      paginator.Model
	width      int
	height     int
	state      State
	pagination PaginationState
	request    common.MediaRequest
}

func NewModel(kind common.ListKind) Model {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(lipgloss.White).PaddingLeft(1)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(lipgloss.BrightBlack).PaddingLeft(1)
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(lipgloss.BrightCyan).Bold(true).BorderLeft(false).PaddingLeft(1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Cyan).BorderLeft(false).PaddingLeft(1)
	delegate.SetSpacing(1)

	listModel := list.New(nil, delegate, 0, 0)
	styles := listModel.Styles
	styles.Title = styles.Title.MarginLeft(1)
	styles.TitleBar = lipgloss.NewStyle().MarginBottom(1)
	styles.NoItems = styles.NoItems.Foreground(lipgloss.BrightBlack)
	listModel.Styles = styles
	listModel.SetShowHelp(false)
	listModel.SetShowStatusBar(false)
	listModel.SetShowFilter(false)
	listModel.SetShowPagination(false)
	listModel.DisableQuitKeybindings()
	listModel.InfiniteScrolling = true
	listModel.Title = common.ListTitle(kind)

	pager := paginator.New()
	pager.Type = paginator.Arabic
	pager.TotalPages = 1
	pager.Page = 0

	return Model{
		kind:       kind,
		list:       listModel,
		pager:      pager,
		state:      Initialized,
		request:    common.RootMediaRequestForListKind(kind, ""),
		pagination: PaginationState{CurrentPage: 1, history: []string{""}},
	}
}

func (m *Model) Kind() common.ListKind {
	return m.kind
}

func (m *Model) State() State {
	return m.state
}

func (m *Model) SelectedEntity() (common.Entity, bool) {
	item, ok := m.list.SelectedItem().(listItem)
	if !ok {
		return common.Entity{}, false
	}
	return item.entity, true
}

func (m *Model) Request() common.MediaRequest {
	return m.request
}

func (m *Model) SetTitle(title string) {
	m.list.Title = title
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}
