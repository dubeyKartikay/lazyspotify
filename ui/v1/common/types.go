package common

import "strings"
import "unicode"

type Entity struct {
	Name string
	Desc string
	ID   string
	Img  string
}

func NewEntity(name, desc, uri, img string) Entity {
	return Entity{
		Name: name,
		Desc: desc,
		ID:   uri,
		Img:  img,
	}
}

type MediaRequestKind int

const (
	GetUserPlaylists MediaRequestKind = iota
	GetSavedTracks
	GetSavedAlbums
	GetFollowedArtists
	SearchPlaylists
	SearchTracks
	SearchAlbums
	SearchArtists
	GetPlaylistTracks
	GetArtistAlbums
	GetAlbumTracks
	PlayTrack
)

type MediaRequest struct {
	Kind        MediaRequestKind
	PanelKind   ListKind
	Cursor      string
	Page        int
	EntityURI   string
	ContextURI  string
	Query       string
	ShowLoading bool
}

type PaginationInfo struct {
	CurrentPage int
	TotalPages  int
	TotalItems  int
	HasNext     bool
	NextCursor  string
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
)

type SongInfo struct {
	Title    string
	Artist   string
	Album    string
	Position int
	Duration int
}

type VolumeInfo struct {
	Volume int
	Max    int
}

func RequestKindForListKind(kind ListKind) MediaRequestKind {
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

func SearchRequestKindForListKind(kind ListKind) MediaRequestKind {
	switch kind {
	case Tracks:
		return SearchTracks
	case Albums:
		return SearchAlbums
	case Artists:
		return SearchArtists
	default:
		return SearchPlaylists
	}
}

func MediaRequestForListKind(kind ListKind) MediaRequest {
	return MediaRequest{Kind: RequestKindForListKind(kind), PanelKind: kind, Page: 1, ShowLoading: true}
}

func SearchMediaRequestForListKind(kind ListKind, query string) MediaRequest {
	query = strings.TrimSpace(query)
	return MediaRequest{Kind: SearchRequestKindForListKind(kind), PanelKind: kind, Page: 1, Query: query, ShowLoading: true}
}

func RootMediaRequestForListKind(kind ListKind, query string) MediaRequest {
	query = strings.TrimSpace(query)
	if query == "" {
		return MediaRequestForListKind(kind)
	}
	return SearchMediaRequestForListKind(kind, query)
}

func KindForRequestKind(kind MediaRequestKind) ListKind {
	switch kind {
	case GetSavedTracks, SearchTracks, GetPlaylistTracks, GetAlbumTracks:
		return Tracks
	case GetSavedAlbums, SearchAlbums, GetArtistAlbums:
		return Albums
	case GetFollowedArtists, SearchArtists:
		return Artists
	case SearchPlaylists:
		return Playlists
	default:
		return Playlists
	}
}

func ListTitle(kind ListKind) string {
	switch kind {
	case Albums:
		return "Albums"
	case Artists:
		return "Artists"
	case Playlists:
		return "Playlists"
	case Tracks:
		return "Tracks"
	case Shows:
		return "Shows"
	case Episodes:
		return "Episodes"
	case AudioBooks:
		return "Audiobooks"
	default:
		return "Media"
	}
}

func ListTitleAbbr(kind ListKind) string {
	switch kind {
	case Albums:
		return "AL"
	case Artists:
		return "AR"
	case Playlists:
		return "PL"
	case Tracks:
		return "TR"
	case Shows:
		return "SH"
	case Episodes:
		return "EP"
	case AudioBooks:
		return "AB"
	default:
		return "Media"
	}
}

func MapSlice[T any, U any](items []T, mapFn func(T) U) []U {
	mapped := make([]U, 0, len(items))
	for _, item := range items {
		mapped = append(mapped, mapFn(item))
	}
	return mapped
}

func StripEmojis(s string) string {
	return strings.Map(func(r rune) rune {
		if IsEmoji(r) {
			return -1
		}
		return r
	}, s)
}

func IsEmoji(r rune) bool {
	return unicode.Is(unicode.So, r) ||
		unicode.Is(unicode.Sk, r) ||
		(r >= 0x1F000 && r <= 0x1FAFF) || // the big emoji block (😀🎉🐶 etc.)
		(r >= 0x2600 && r <= 0x27BF) || // misc symbols & dingbats (☀️⚡ etc.)
		r == 0x200D || // zero-width joiner (used in family emoji etc.)
		(r >= 0xFE00 && r <= 0xFE0F)
}
