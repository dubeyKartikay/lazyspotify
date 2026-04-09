package v1

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	"github.com/dubeyKartikay/lazyspotify/core/auth"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/player"
	"github.com/dubeyKartikay/lazyspotify/core/ticker"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
	"github.com/dubeyKartikay/lazyspotify/spotify"
)

type Model struct {
	authModel          *AuthModel
	playing            bool
	playerReady        bool
	songInfo           SongInfo
	volumeInfo         VolumeInfo
	volumeOverlayUntil time.Time
	player             *player.Player
	spotifyClient      *spotify.SpotifyClient
	mediaCenter        MediaCenter
	width              int
	height             int
	help               help.Model
	keys               appKeyMap
}

type SongInfo struct {
	title    string
	artist   string
	album    string
	position int
	duration int
}

type VolumeInfo struct {
	volume int
	max    int
}

type mediaLoadedMsg struct {
	entities   []Entity
	kind       ListKind
	pagination PaginationInfo
	request    MediaRequest
}

type mediaLoadErrMsg struct {
	err error
}

type playTrackErrMsg struct {
	err error
}

type playTrackOkMsg struct{}

type startupCompleteMsg struct{}

type playerReadyMsg struct{}

type playerReadyErrMsg struct {
	err error
}

type playerEventMsg struct {
	event models.PlayerEvent
}

type playerEventsClosedMsg struct{}

func (m *Model) playPause() {
	if m.player == nil {
		logger.Log.Error().Msg("cannot play/pause without player")
		return
	}

	if err := m.player.PlayPause(context.Background()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to play/pause track")
	}
}

func (m *Model) seekForward() {
	if m.player == nil {
		logger.Log.Error().Msg("cannot seek forward without player")
		return
	}

	step := utils.GetConfig().Librespot.SeekStepMs
	if err := m.player.Seek(context.Background(), step, true); err != nil {
		logger.Log.Error().Err(err).Int("seek_step_ms", step).Msg("failed to seek forward")
	}
}

func (m *Model) seekBackward() {
	if m.player == nil {
		logger.Log.Error().Msg("cannot seek backward without player")
		return
	}

	step := utils.GetConfig().Librespot.SeekStepMs
	if err := m.player.Seek(context.Background(), -step, true); err != nil {
		logger.Log.Error().Err(err).Int("seek_step_ms", step).Msg("failed to seek backward")
	}
}
func (m *Model) next() {
	if m.player == nil {
		logger.Log.Error().Msg("cannot skip to next track without player")
		return
	}

	if err := m.player.Next(context.Background()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to skip to next track")
	}
}

func (m *Model) previous() {
	if m.player == nil {
		logger.Log.Error().Msg("cannot skip to previous track without player")
		return
	}

	if err := m.player.Previous(context.Background()); err != nil {
		logger.Log.Error().Err(err).Msg("failed to skip to previous track")
	}
}

func (m *Model) decrementVolume() {
	if m.player == nil {
		logger.Log.Error().Msg("cannot decrement volume without player")
		return
	}

	step := utils.GetConfig().Librespot.VolumeStep
	volume, err := m.player.GetVolume(context.Background())
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to read volume before decrement")
		return
	}

	target := max(volume.Value-step, 0)

	if err := m.player.SetVolume(context.Background(), target, false); err != nil {
		logger.Log.Error().Err(err).Int("target_volume", target).Msg("failed to decrement volume")
		return
	}
}

func (m *Model) incrementVolume() {
	if m.player == nil {
		logger.Log.Error().Msg("cannot increment volume without player")
		return
	}

	step := utils.GetConfig().Librespot.VolumeStep
	volume, err := m.player.GetVolume(context.Background())
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to read volume before increment")
		return
	}

	target := min(volume.Value+step, volume.Max)

	if err := m.player.SetVolume(context.Background(), target, false); err != nil {
		logger.Log.Error().Err(err).Int("target_volume", target).Msg("failed to increment volume")
		return
	}
}

func (m *Model) playPauseCmd() tea.Cmd {
	return func() tea.Msg {
		m.playPause()
		return nil
	}
}

func (m *Model) seekForwardCmd() tea.Cmd {
	return func() tea.Msg {
		m.seekForward()
		return nil
	}
}

func (m *Model) seekBackwardCmd() tea.Cmd {
	return func() tea.Msg {
		m.seekBackward()
		return nil
	}
}

func (m *Model) nextCmd() tea.Cmd {
	return func() tea.Msg {
		m.next()
		return nil
	}
}

func (m *Model) previousCmd() tea.Cmd {
	return func() tea.Msg {
		m.previous()
		return nil
	}
}

func (m *Model) incrementVolumeCmd() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		m.incrementVolume()
		return nil
	}, m.mediaCenter.cassettePlayer.ShowVolume())
}

func (m *Model) decrementVolumeCmd() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		m.decrementVolume()
		m.mediaCenter.cassettePlayer.ShowVolume()
		return nil
	}, m.mediaCenter.cassettePlayer.ShowVolume())
}

func (m *Model) setSize(width, height int) {
	m.width = width
	m.height = height
	m.help.SetWidth(width)
	if m.authModel != nil {
		m.authModel.SetSize(width, height)
	}
}

func (m *Model) playDailyMix() {
	uri, err := m.spotifyClient.GetFirstSavedTrack(context.Background())
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to get first saved track")
		return
	}
	err = m.player.PlayTrack(context.Background(), uri, "")
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to play daily mix")
	}
	m.playing = true
}

func (m *Model) shutdown() {
	if m.player == nil {
		return
	}

	m.player.Destroy(context.Background())
}

func (m *Model) start() error {
	ctx := context.Background()
	var err error
	m.authModel = newAuthModel()
	if m.width != 0 || m.height != 0 {
		m.authModel.SetSize(m.width, m.height)
	}
	m.spotifyClient, err = spotify.NewSpotifyClient(ctx, m.authModel.auth)
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.authState = NeedsAuth
		}
		logger.Log.Error().Err(err).Msg("failed to create spotify client")
		return err
	}
	userId, err := m.spotifyClient.GetUserID(ctx)
	logger.Log.Info().Str("user id", userId).Msg("got user id")
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.authState = NeedsAuth
		}
		return err
	}

	tkn, err := auth.New().GetAuthToken(ctx)

	if err != nil || tkn == nil {
		m.authModel.authState = NeedsAuth
		return err
	}

	m.player = player.NewPlayer(ctx, userId, tkn.AccessToken)

	m.player.Start(ctx)

	if m.player == nil {
		logger.Log.Error().Msg("failed to create player")
		return fmt.Errorf("failed to create player")
	}

	return nil
}

func (m *Model) waitForPlayerReady() tea.Cmd {
	if m.player == nil {
		return nil
	}

	return func() tea.Msg {
		err := m.player.WaitTillReady()
		if err != nil {
			return playerReadyErrMsg{err: err}
		}
		return playerReadyMsg{}
	}
}

func (m *Model) waitForPlayerEvent() tea.Cmd {
	if m.player == nil {
		return nil
	}

	events := m.player.Events()
	if events == nil {
		return nil
	}

	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return playerEventsClosedMsg{}
		}
		return playerEventMsg{event: ev}
	}
}

func (m *Model) applyPlayerEvent(ev models.PlayerEvent) {
	switch ev.Type {
	case models.EventTypeMetadata:
		if ev.Metadata == nil {
			return
		}
		artist := strings.Join(ev.Metadata.ArtistNames, ", ")
		m.songInfo = SongInfo{
			title:    ev.Metadata.Name,
			artist:   artist,
			album:    ev.Metadata.AlbumName,
			position: ev.Metadata.Position,
			duration: ev.Metadata.Duration,
		}
		m.mediaCenter.displayScreen.SetDisplayFromSong(m.songInfo)
	case models.EventTypePlaying:
		m.playing = true
	case models.EventTypePaused, models.EventTypeStopped:
		m.playing = false
		if ev.Type == models.EventTypeStopped {
			m.songInfo.position = 0
		}
	case models.EventTypeSeek:
		if ev.Seek != nil {
			m.songInfo.position = ev.Seek.Position
			m.songInfo.duration = ev.Seek.Duration
		}
	case models.EventTypeVolume:
		if ev.Volume != nil {
			m.volumeInfo.volume = ev.Volume.Value
			if ev.Volume.Max > 0 {
				m.volumeInfo.max = ev.Volume.Max
			}
		}
	}
}

func (m *Model) NextFrame() tea.Cmd {
	m.advancePlayback(180)
	m.mediaCenter.cassettePlayer.NextFrame(m.playing)
	return ticker.DoTickFast()
}

func (m *Model) markVolumeOverlay() {
	m.volumeOverlayUntil = time.Now().Add(1500 * time.Millisecond)
}

func (m *Model) previewVolume(delta int) {
	maxVolume := m.volumeInfo.max
	if maxVolume <= 0 {
		maxVolume = 100
	}
	m.volumeInfo.volume = max(0, min(maxVolume, m.volumeInfo.volume+delta))
}

func (m *Model) advancePlayback(elapsedMs int) {
	if !m.playing || elapsedMs <= 0 {
		return
	}
	if m.songInfo.duration <= 0 {
		return
	}
	m.songInfo.position = min(m.songInfo.position+elapsedMs, m.songInfo.duration)
	m.UpdateStatus()
}

func (m *Model) UpdateStatus()  {
	maxVolume := m.volumeInfo.max
	if maxVolume <= 0 {
		maxVolume = 100
	}
	m.mediaCenter.cassettePlayer.UpdateStatus(m.playerReady,m.playing, m.songInfo.position, m.songInfo.duration, m.volumeInfo.volume, maxVolume)
}

func (m *Model) NextButtonFrame() tea.Cmd {
	m.mediaCenter.cassettePlayer.NextButtonFrame()
	return nil
}

func (m *Model) HandleButtonPress(buttonKind ButtonKind) tea.Cmd {
	m.mediaCenter.cassettePlayer.HandleButtonPress(buttonKind)
	return ticker.DoTickClick()
}

func (m *Model) HandleMediaRequest(mediaRequest MediaRequest) tea.Cmd {
	if mediaRequest.page <= 0 {
		mediaRequest.page = 1
	}
	switch mediaRequest.kind {
	case GetUserPlaylists:
		return m.handleGetUserPlaylists(mediaRequest)
	case GetSavedTracks:
		return m.handleGetSavedTracks(mediaRequest)
	case GetSavedAlbums:
		return m.handleGetSavedAlbums(mediaRequest)
	case GetFollowedArtists:
		return m.handleGetFollowedArtists(mediaRequest)
	case GetPlaylistTracks:
		return m.handleGetPlaylistTracks(mediaRequest)
	case GetArtistAlbums:
		return m.handleGetArtistAlbums(mediaRequest)
	case GetAlbumTracks:
		return m.handleGetAlbumTracks(mediaRequest)
	case PlayTrack:
		return m.handlePlayTrack(mediaRequest.entityURI, mediaRequest.contextURI)
	}
	return nil
}

func (m *Model) handleGetUserPlaylists(request MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}

	return func() tea.Msg {
		offset := decodeOffsetCursor(request.cursor)
		p, err := m.spotifyClient.GetUserPlaylists(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err}
		}
		e := AdaptSpotifyPlaylistPage(p)
		pagination := paginationFromOffset(offset, len(e), int(p.Total), 10)
		return mediaLoadedMsg{entities: e, kind: Playlists, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetSavedTracks(request MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}

	return func() tea.Msg {
		offset := decodeOffsetCursor(request.cursor)
		p, err := m.spotifyClient.GetSavedTracks(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err}
		}
		e := AdaptSpotifySavedTrackPage(p)
		pagination := paginationFromOffset(offset, len(e), int(p.Total), 10)
		return mediaLoadedMsg{entities: e, kind: Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetSavedAlbums(request MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}

	return func() tea.Msg {
		offset := decodeOffsetCursor(request.cursor)
		p, err := m.spotifyClient.GetSavedAlbums(context.Background(), offset)
		if err != nil {
			return mediaLoadErrMsg{err: err}
		}
		e := AdaptSpotifySavedAlbumPage(p)
		pagination := paginationFromOffset(offset, len(e), int(p.Total), 10)
		return mediaLoadedMsg{entities: e, kind: Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetFollowedArtists(request MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}

	return func() tea.Msg {
		p, err := m.spotifyClient.GetFollowedArtists(context.Background(), request.cursor)
		if err != nil {
			return mediaLoadErrMsg{err: err}
		}
		e := AdaptSpotifyFollowedArtistsPage(p)
		pagination := paginationFromCursor(request.page, len(e), int(p.Total), 10, p.Cursor.After)
		return mediaLoadedMsg{entities: e, kind: Artists, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetPlaylistTracks(request MediaRequest) tea.Cmd {
	if m.player == nil {
		return nil
	}

	return func() tea.Msg {
		const pageSize = 10
		offset := decodeOffsetCursor(request.cursor)
		resp, err := m.player.GetPlaylistTracks(context.Background(), request.entityURI, offset, pageSize)
		if err != nil {
			return mediaLoadErrMsg{err: err}
		}
		e := AdaptResolvedPlaylistTracks(resp.Tracks)
		nextCursor := ""
		if resp.HasNext {
			nextCursor = encodeOffsetCursor(offset + pageSize)
		}
		pagination := PaginationInfo{
			CurrentPage: request.page,
			TotalPages:  totalPages(resp.Total, pageSize),
			TotalItems:  resp.Total,
			HasNext:     resp.HasNext,
			NextCursor:  nextCursor,
		}
		return mediaLoadedMsg{entities: e, kind: Tracks, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetArtistAlbums(request MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}

	return func() tea.Msg {
		offset := decodeOffsetCursor(request.cursor)
		albums, err := m.spotifyClient.GetArtistAlbums(context.Background(), request.entityURI, offset)
		if err != nil {
			return mediaLoadErrMsg{err: err}
		}
		e := AdaptSpotifyArtistAlbums(albums.Albums)
		pagination := paginationFromOffset(offset, len(e), int(albums.Total), 10)
		return mediaLoadedMsg{entities: e, kind: Albums, pagination: pagination, request: request}
	}
}

func (m *Model) handleGetAlbumTracks(request MediaRequest) tea.Cmd {
	if m.spotifyClient == nil {
		return nil
	}

	return func() tea.Msg {
		const pageSize = 50
		offset := decodeOffsetCursor(request.cursor)
		tracks, err := m.spotifyClient.GetAlbumTracks(context.Background(), request.entityURI, offset)
		if err != nil {
			return mediaLoadErrMsg{err: err}
		}
		e := AdaptSpotifyAlbumTracks(tracks.Tracks)
		pagination := paginationFromOffset(offset, len(e), int(tracks.Total), pageSize)
		return mediaLoadedMsg{entities: e, kind: Tracks, pagination: pagination, request: request}
	}
}

func decodeOffsetCursor(cursor string) int {
	if cursor == "" {
		return 0
	}
	v, err := strconv.Atoi(cursor)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func encodeOffsetCursor(offset int) string {
	if offset < 0 {
		offset = 0
	}
	return strconv.Itoa(offset)
}

func totalPages(totalItems int, pageSize int) int {
	if totalItems <= 0 || pageSize <= 0 {
		return 1
	}
	pages := totalItems / pageSize
	if totalItems%pageSize != 0 {
		pages++
	}
	if pages <= 0 {
		return 1
	}
	return pages
}

func paginationFromOffset(offset int, count int, total int, pageSize int) PaginationInfo {
	currentPage := 1
	if pageSize > 0 {
		currentPage = (offset / pageSize) + 1
	}
	hasNext := offset+count < total
	nextCursor := ""
	if hasNext {
		nextCursor = encodeOffsetCursor(offset + pageSize)
	}
	return PaginationInfo{
		CurrentPage: currentPage,
		TotalPages:  totalPages(total, pageSize),
		TotalItems:  total,
		HasNext:     hasNext,
		NextCursor:  nextCursor,
	}
}

func paginationFromCursor(page int, count int, total int, pageSize int, nextCursor string) PaginationInfo {
	hasNext := nextCursor != "" && count > 0
	if page <= 0 {
		page = 1
	}
	return PaginationInfo{
		CurrentPage: page,
		TotalPages:  totalPages(total, pageSize),
		TotalItems:  total,
		HasNext:     hasNext,
		NextCursor:  nextCursor,
	}
}

func (m *Model) handlePlayTrack(uri string, contextURI string) tea.Cmd {
	if m.player == nil {
		return m.mediaCenter.SetStatus("Player not ready")
	}
	m.mediaCenter.displayScreen.SetDisplay("Loading...")
	m.playerReady = false
	m.playing = false
	m.mediaCenter.mediaListOpen = false
	return func() tea.Msg {
		err := m.player.PlayTrack(context.Background(), uri, contextURI)
		if err != nil {
			return playTrackErrMsg{err: err}
		}
		return playTrackOkMsg{}
	}
}
