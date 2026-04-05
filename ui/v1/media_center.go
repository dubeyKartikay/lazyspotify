package v1

import (
	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/spotify"
)

type Entity struct {
	Name string
	Desc string
	uri  string
	img  string
}


type ListKind int

const (
	Albums ListKind = iota
	Artists
	Playlists
	Tracks
	Shows
	Episodes
	AudioBooks
	Loading
)

type mediaList struct {
	kind  ListKind
	items []Entity
}

type MediaCenter struct {
	visibleList    mediaList
	cassettePlayer CassettePlayer
}

func NewMediaCenter() MediaCenter {
	return MediaCenter{
		cassettePlayer: NewCassettePlayer(),
	}
}

func (m *MediaCenter) View() string {
	return m.cassettePlayer.View()
}

func (m *MediaCenter) Init(spClient *spotify.SpotifyClient) tea.Cmd {
	
}
func (m *MediaCenter) SetContent(entities []Entity, kind ListKind) {
	if !isContentListKind(kind) {
		m.visibleList = mediaList{
			kind:  kind,
			items: make([]Entity, 0),
		}
		return
	}

	m.visibleList = newMediaList(entities, kind)
}

func isContentListKind(kind ListKind) bool {
	switch kind {
	case Albums, Artists, Playlists, Tracks, Shows, Episodes, AudioBooks:
		return true
	default:
		return false
	}
}

func newMediaList(entities []Entity, kind ListKind) mediaList {
	return mediaList{
		kind:  kind,
		items: entities,
	}
}

func (e *Entity) Action(m *MediaCenter) tea.Cmd {
	return nil
}
