package app

import (
	"context"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
	"github.com/zmb3/spotify/v2"
)

func (m *Model) handleGetUserPlaylists(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.GetUserPlaylists(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyPlaylists(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Playlists, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetSavedTracks(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.GetSavedTracks(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifySavedTracks(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetSavedAlbums(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.GetSavedAlbums(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifySavedAlbums(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetFollowedArtists(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		page, err := m.spotifyClient.GetFollowedArtists(context.Background(), request.Cursor)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtists(page)
		pagination := paginationFromCursor(request.Page, len(entities), int(page.Total), 10, page.Cursor.After)
		return mediaLoadedMsg{entities: entities, kind: common.Artists, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchPlaylists(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchPlaylists(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyPlaylists(page)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Playlists, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchTracks(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchTracks(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyTracks(page.Tracks)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchAlbums(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchAlbums(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtistAlbums(page.Albums)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleSearchArtists(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		page, err := m.spotifyClient.SearchArtists(context.Background(), request.Query, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtistsPage(page.Artists)
		pagination := paginationFromOffset(offset, len(entities), int(page.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Artists, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetPlaylistTracks(request common.MediaRequest) tea.Cmd {
	if m.player == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.Cursor)
		resp, err := m.player.GetPlaylistTracks(context.Background(), request.EntityURI, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptResolvedPlaylistTracks(resp.Tracks)
		nextCursor := ""
		if resp.HasNext {
			nextCursor = encodeOffsetCursor(offset + pageSize)
		}
		pagination := common.PaginationInfo{
			CurrentPage: request.Page,
			TotalPages:  totalPages(resp.Total, pageSize),
			TotalItems:  resp.Total,
			HasNext:     resp.HasNext,
			NextCursor:  nextCursor,
		}
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetArtistAlbums(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		offset := decodeOffsetCursor(request.Cursor)
		albums, err := m.spotifyClient.GetArtistAlbums(context.Background(), request.EntityURI, offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyArtistAlbums(albums.Albums)
		pagination := paginationFromOffset(offset, len(entities), int(albums.Total), 10)
		return mediaLoadedMsg{entities: entities, kind: common.Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetAlbumTracks(request common.MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}
	return func() tea.Msg {
		const pageSize = 50
		offset := decodeOffsetCursor(request.Cursor)
		tracks, err := m.spotifyClient.GetAlbumTracks(context.Background(), request.EntityURI, offset)
		if err != nil {
			return mediaLoadErrMsg{err: err, request: request}
		}
		entities := adaptSpotifyAlbumTracks(tracks.Tracks)
		pagination := paginationFromOffset(offset, len(entities), int(tracks.Total), pageSize)
		return mediaLoadedMsg{entities: entities, kind: common.Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handlePlayTrackRequest(request common.MediaRequest) tea.Cmd {
	if m.player == nil {
		return m.mediaCenter.SetStatus(request.PanelKind, "Player not ready")
	}
	m.mediaCenter.SetDisplay("Loading...")
	m.playerReady = false
	m.playing = false
	m.mediaCenter.CloseLibrary()
	return func() tea.Msg {
		err := m.player.PlayTrack(context.Background(), request.EntityURI, request.ContextURI)
		if err != nil {
			return playTrackErrMsg{err: err, panelKind: request.PanelKind}
		}
		return playTrackOkMsg{panelKind: request.PanelKind}
	}
}

func adaptSpotifyPlaylists(page *spotify.SimplePlaylistPage) []common.Entity {
	logger.Log.Info().Any("p", page).Msg("adapt spotify playlist page")
	return common.MapSlice(page.Playlists, func(pl spotify.SimplePlaylist) common.Entity {
		return common.NewEntity(pl.Name, pl.Description, string(pl.URI), imageURL(pl.Images))
	})
}

func adaptSpotifySavedTracks(page *spotify.SavedTrackPage) []common.Entity {
	return common.MapSlice(page.Tracks, func(savedTrack spotify.SavedTrack) common.Entity {
		track := savedTrack.FullTrack
		desc := strings.TrimSpace(joinArtists(track.Artists))
		if track.Album.Name != "" {
			if desc != "" {
				desc += " • " + track.Album.Name
			} else {
				desc = track.Album.Name
			}
		}
		return common.NewEntity(track.Name, desc, string(track.URI), imageURL(track.Album.Images))
	})
}

func adaptSpotifySavedAlbums(page *spotify.SavedAlbumPage) []common.Entity {
	return common.MapSlice(page.Albums, func(savedAlbum spotify.SavedAlbum) common.Entity {
		album := savedAlbum.FullAlbum
		return common.NewEntity(album.Name, joinArtists(album.Artists), string(album.URI), imageURL(album.Images))
	})
}

func adaptSpotifyArtists(page *spotify.FullArtistCursorPage) []common.Entity {
	return adaptSpotifyArtistsPage(page.Artists)
}

func adaptSpotifyArtistsPage(artists []spotify.FullArtist) []common.Entity {
	return common.MapSlice(artists, func(artist spotify.FullArtist) common.Entity {
		desc := ""
		if len(artist.Genres) > 0 {
			desc = strings.Join(artist.Genres, ", ")
		}
		return common.NewEntity(artist.Name, desc, string(artist.URI), imageURL(artist.Images))
	})
}

func adaptSpotifyTracks(tracks []spotify.FullTrack) []common.Entity {
	return common.MapSlice(tracks, func(track spotify.FullTrack) common.Entity {
		desc := strings.TrimSpace(joinArtists(track.Artists))
		if track.Album.Name != "" {
			if desc != "" {
				desc += " • " + track.Album.Name
			} else {
				desc = track.Album.Name
			}
		}
		return common.NewEntity(track.Name, desc, string(track.URI), imageURL(track.Album.Images))
	})
}

func adaptResolvedPlaylistTracks(tracks []models.ResolvedTrack) []common.Entity {
	return common.MapSlice(tracks, func(track models.ResolvedTrack) common.Entity {
		desc := strings.TrimSpace(strings.Join(track.Artists, ", "))
		if track.AlbumName != "" {
			if desc != "" {
				desc += " • " + track.AlbumName
			} else {
				desc = track.AlbumName
			}
		}
		return common.NewEntity(track.Name, desc, track.URI, track.Img)
	})
}

func adaptSpotifyArtistAlbums(albums []spotify.SimpleAlbum) []common.Entity {
	return common.MapSlice(albums, func(album spotify.SimpleAlbum) common.Entity {
		return common.NewEntity(album.Name, joinArtists(album.Artists), string(album.URI), imageURL(album.Images))
	})
}

func adaptSpotifyAlbumTracks(tracks []spotify.SimpleTrack) []common.Entity {
	return common.MapSlice(tracks, func(track spotify.SimpleTrack) common.Entity {
		return common.NewEntity(track.Name, strings.TrimSpace(joinArtists(track.Artists)), string(track.URI), "")
	})
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
	names := common.MapSlice(artists, func(artist spotify.SimpleArtist) string {
		return artist.Name
	})
	return strings.Join(names, ", ")
}
