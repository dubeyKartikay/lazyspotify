package v1

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/zmb3/spotify/v2"
)

type Entity struct {
	Name string
	Desc string
	ID   string
	Img  string
}

type MediaRequestKind int

const (
	GetUserPlaylists MediaRequestKind = iota
	GetSavedTracks
	GetSavedAlbums
	GetFollowedArtists
	GetPlaylistTracks
	GetArtistAlbums
	GetAlbumTracks
	PlayTrack
)

type MediaRequest struct {
	kind        MediaRequestKind
	offset      int
	entityURI   string
	append      bool
	showLoading bool
}

func NewEntity(name string, desc string, uri string, img string) Entity {
	return Entity{
		Name: name,
		Desc: desc,
		ID:   uri,
		Img:  img,
	}
}

type ListKind int

const (
	Playlists ListKind = iota
	Albums
	Artists
	Tracks
	Shows
	Episodes
	AudioBooks
	Loading
)

func requestKindForListKind(kind ListKind) MediaRequestKind {
	switch kind {
	case Tracks:
		return GetSavedTracks
	case Albums:
		return GetSavedAlbums
	case Artists:
		return GetFollowedArtists
	default:
		return GetUserPlaylists
	}
}

func nextLibraryListKind(current ListKind) ListKind {
	switch current {
	case Playlists:
		return Tracks
	case Tracks:
		return Albums
	case Albums:
		return Artists
	default:
		return Playlists
	}
}

func MediaRequestForListKind(kind ListKind, offset int) MediaRequest {
	return MediaRequest{kind: requestKindForListKind(kind), offset: offset, showLoading: true}
}

type MediaCenter struct {
	lists          utils.Stack[mediaList]
	cassettePlayer CassettePlayer
	displayScreen  displayScreen
}

func NewMediaCenter() MediaCenter {
	m := MediaCenter{
		cassettePlayer: NewCassettePlayer(),
		displayScreen:  newDisplayScreen(),
	}
	m.lists.Push(newMediaList())
	return m
}

func (m *MediaCenter) Update(msg tea.Msg) tea.Cmd {
	visibleList := m.lists.Peek()
	listCmd := visibleList.Update(msg)
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "enter":
			if item, ok := visibleList.list.SelectedItem().(mediaListItem); ok {
				return item.entity.Action(m)
			}
		case "backspace", "delete":
			if m.lists.Len() > 1 {
				m.lists.Pop()
				return m.SetStatus("Back")
			}
		}
	}

	return listCmd
}
func (m *MediaCenter) StartLoading() tea.Cmd {
	return m.lists.Peek().StartLoading()
}

func (m *MediaCenter) SetContent(entities []Entity, kind ListKind) tea.Cmd {
	visibleList := m.lists.Peek()
	cmd := visibleList.SetContent(entities, kind)
	kinds := make([]ListKind, 0, m.lists.Len())
for _, list := range m.lists.Items {
		kinds = append(kinds, list.kind)
	}
	visibleList.SetTitle(GenerateListTitle(kinds))
	logger.Log.Info().Any("entities", entities).Int("kind", int(kind)).Msg("set content")
	logger.Log.Info().Int("kind", int(visibleList.kind)).Msg("set content visibleList")
	return cmd
}

func (m *MediaCenter) StopSpinner() {
	m.lists.Peek().StopSpinner()
}

func (m *MediaCenter) SetStatus(message string) tea.Cmd {
	return m.lists.Peek().SetStatus(message)
}

func (m *MediaCenter) NextListKind() ListKind {
	return nextLibraryListKind(m.lists.Items[0].kind)
}

func (e *Entity) Action(m *MediaCenter) tea.Cmd {
	visibleList := m.lists.Peek()
	switch visibleList.kind {
	case Playlists:
		m.lists.Push(newMediaList())
		return func() tea.Msg {
			return MediaRequest{
				kind:        GetPlaylistTracks,
				offset:      0,
				entityURI:   e.ID,
				showLoading: true,
			}
		}
	case Artists:
		m.lists.Push(newMediaList())
		return func() tea.Msg {
			return MediaRequest{
				kind:        GetArtistAlbums,
				offset:      0,
				entityURI:   e.ID,
				showLoading: true,
			}
		}
	case Albums:
		m.lists.Push(newMediaList())
		return func() tea.Msg {
			return MediaRequest{
				kind:        GetAlbumTracks,
				offset:      0,
				entityURI:   e.ID,
				showLoading: true,
			}
		}
	case Tracks:
		return func() tea.Msg {
			return MediaRequest{
				kind:        PlayTrack,
				entityURI:   e.ID,
				showLoading: false,
			}
		}
	default:
		return nil
	}
}

func AdaptSpotifyPlaylistPage(p *spotify.SimplePlaylistPage) []Entity {
	entities := make([]Entity, 0)
	logger.Log.Info().Any("p", p).Msg("AdaptSpotifyPlaylistPage")
	for _, pl := range p.Playlists {
		img := imageURL(pl.Images)
		entities = append(entities,
			NewEntity(pl.Name,
				pl.Description,
				string(pl.URI),
				img,
			))
	}
	return entities
}

func AdaptSpotifySavedTrackPage(p *spotify.SavedTrackPage) []Entity {
	entities := make([]Entity, 0, len(p.Tracks))
	for _, savedTrack := range p.Tracks {
		track := savedTrack.FullTrack
		desc := strings.TrimSpace(joinArtists(track.Artists))
		if track.Album.Name != "" {
			if desc != "" {
				desc += " • " + track.Album.Name
			} else {
				desc = track.Album.Name
			}
		}
		entities = append(entities,
			NewEntity(track.Name,
				desc,
				string(track.URI),
				imageURL(track.Album.Images),
			))
	}
	return entities
}

func AdaptSpotifySavedAlbumPage(p *spotify.SavedAlbumPage) []Entity {
	entities := make([]Entity, 0, len(p.Albums))
	for _, savedAlbum := range p.Albums {
		album := savedAlbum.FullAlbum
		entities = append(entities,
			NewEntity(album.Name,
				joinArtists(album.Artists),
				string(album.URI),
				imageURL(album.Images),
			))
	}
	return entities
}

func AdaptSpotifyFollowedArtistsPage(p *spotify.FullArtistCursorPage) []Entity {
	entities := make([]Entity, 0, len(p.Artists))
	for _, artist := range p.Artists {
		desc := ""
		if len(artist.Genres) > 0 {
			desc = strings.Join(artist.Genres, ", ")
		}
		entities = append(entities,
			NewEntity(artist.Name,
				desc,
				string(artist.URI),
				imageURL(artist.Images),
			))
	}
	return entities
}

func AdaptSpotifyPlaylistTracks(tracks []spotify.FullTrack) []Entity {
	entities := make([]Entity, 0, len(tracks))
	for _, track := range tracks {
		desc := strings.TrimSpace(joinArtists(track.Artists))
		if track.Album.Name != "" {
			if desc != "" {
				desc += " • " + track.Album.Name
			} else {
				desc = track.Album.Name
			}
		}
		entities = append(entities, NewEntity(track.Name, desc, string(track.URI), imageURL(track.Album.Images)))
	}
	return entities
}

func AdaptSpotifyArtistAlbums(albums []spotify.SimpleAlbum) []Entity {
	entities := make([]Entity, 0, len(albums))
	for _, album := range albums {
		entities = append(entities,
			NewEntity(album.Name,
				joinArtists(album.Artists),
				string(album.URI),
				imageURL(album.Images),
			))
	}
	return entities
}

func AdaptSpotifyAlbumTracks(tracks []spotify.SimpleTrack) []Entity {
	entities := make([]Entity, 0, len(tracks))
	for _, track := range tracks {
		desc := strings.TrimSpace(joinArtists(track.Artists))
		entities = append(entities,
			NewEntity(track.Name,
				desc,
				string(track.URI),
				"",
			))
	}
	return entities
}

func imageURL(images []spotify.Image) string {
	if len(images) == 0 {
		return ""
	}
	return images[0].URL
}

func joinArtists(artists []spotify.SimpleArtist) string {
	if len(artists) == 0 {
		return ""
	}
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		names = append(names, artist.Name)
	}
	return strings.Join(names, ", ")
}
