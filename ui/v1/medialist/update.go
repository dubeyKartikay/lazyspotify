package medialist

import (
	"fmt"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

func (m *Model) StartLoading() tea.Cmd {
	m.state = Loading
	return tea.Batch(m.list.StartSpinner())
}

func (m *Model) StopLoading() {
	m.state = Ready
	m.list.StopSpinner()
}

func (m *Model) SetContent(entities []common.Entity, kind common.ListKind) tea.Cmd {
	items := make([]listItem, 0, len(entities))
	for _, entity := range entities {
		if entity.Name == "" {
			continue
		}
		items = append(items, listItem{entity: entity})
	}

	m.kind = kind
	m.items = entities
	m.StopLoading()

	bubbleItems := make([]list.Item, 0, len(items))
	for _, entry := range items {
		bubbleItems = append(bubbleItems, entry)
	}
	setItemsCmd := m.list.SetItems(bubbleItems)
	logger.Log.Info().Any("items", entities).Int("kind", int(kind)).Msg("set content")
	return setItemsCmd
}

func (m *Model) ApplyPagination(info common.PaginationInfo, request common.MediaRequest) {
	if request.Page <= 0 {
		request.Page = 1
	}
	m.request.Kind = request.Kind
	m.request.PanelKind = request.PanelKind
	m.kind = common.KindForRequestKind(request.Kind)
	m.request.EntityURI = request.EntityURI
	m.request.ContextURI = request.ContextURI
	m.request.Query = request.Query
	m.request.ShowLoading = true

	m.pagination.CurrentPage = request.Page
	m.pagination.TotalItems = info.TotalItems
	m.pagination.TotalPages = info.TotalPages
	m.pagination.HasNext = info.HasNext
	m.pagination.HasPrev = request.Page > 1
	m.pagination.NextCursor = info.NextCursor

	if len(m.pagination.history) < request.Page {
		missing := request.Page - len(m.pagination.history)
		for i := 0; i < missing; i++ {
			m.pagination.history = append(m.pagination.history, "")
		}
	} else {
		m.pagination.history = m.pagination.history[:request.Page]
	}
	m.pagination.history[request.Page-1] = request.Cursor

	m.pagination.PrevCursor = ""
	if request.Page > 1 {
		m.pagination.PrevCursor = m.pagination.history[request.Page-2]
	}

	totalPages := m.pagination.TotalPages
	if totalPages <= 0 {
		totalPages = 1
	}
	m.pager.TotalPages = totalPages
	m.pager.Page = max(0, m.pagination.CurrentPage-1)
	m.list.Title = common.ListTitle(m.kind)
}

func (m *Model) NextPageRequest() (common.MediaRequest, bool) {
	if !m.pagination.HasNext || m.pagination.NextCursor == "" {
		return common.MediaRequest{}, false
	}
	return common.MediaRequest{
		Kind:        m.request.Kind,
		PanelKind:   m.request.PanelKind,
		Cursor:      m.pagination.NextCursor,
		Page:        m.pagination.CurrentPage + 1,
		EntityURI:   m.request.EntityURI,
		ContextURI:  m.request.ContextURI,
		Query:       m.request.Query,
		ShowLoading: true,
	}, true
}

func (m *Model) PrevPageRequest() (common.MediaRequest, bool) {
	if !m.pagination.HasPrev {
		return common.MediaRequest{}, false
	}
	return common.MediaRequest{
		Kind:        m.request.Kind,
		PanelKind:   m.request.PanelKind,
		Cursor:      m.pagination.PrevCursor,
		Page:        m.pagination.CurrentPage - 1,
		EntityURI:   m.request.EntityURI,
		ContextURI:  m.request.ContextURI,
		Query:       m.request.Query,
		ShowLoading: true,
	}, true
}

func (m *Model) SetStatus(message string) tea.Cmd {
	return m.list.NewStatusMessage(message)
}

type listItem struct {
	entity common.Entity
}

func (i listItem) Title() string {
	return i.entity.Name
}

func (i listItem) Description() string {
	return i.entity.Desc
}

func (i listItem) FilterValue() string {
	return fmt.Sprintf("%s %s", i.entity.Name, i.entity.Desc)
}
