package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	tea "charm.land/bubbletea/v2"
	uiauth "github.com/dubeyKartikay/lazyspotify/ui/v1/auth"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/common"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/mediacenter"
	"github.com/dubeyKartikay/lazyspotify/ui/v1/player"

	"github.com/dubeyKartikay/lazyspotify/core/auth"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	coreplayer "github.com/dubeyKartikay/lazyspotify/core/player"
	"github.com/dubeyKartikay/lazyspotify/core/ticker"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
	"github.com/dubeyKartikay/lazyspotify/spotify"
)

type Model struct {
	authModel          *uiauth.Model
	playing            bool
	playerReady        bool
	songInfo           common.SongInfo
	volumeInfo         common.VolumeInfo
	volumeOverlayUntil time.Time
	player             *coreplayer.Player
	spotifyClient      *spotify.SpotifyClient
	mediaCenter        mediacenter.Model
	width              int
	height             int
	help               help.Model
	keys               common.AppKeyMap
	requestHandlers    map[common.MediaRequestKind]func(common.MediaRequest) tea.Cmd
}

type mediaLoadedMsg struct {
	entities   []common.Entity
	kind       common.ListKind
	pagination common.PaginationInfo
	request    common.MediaRequest
}

type mediaLoadErrMsg struct {
	err     error
	request common.MediaRequest
}

type playTrackErrMsg struct {
	err       error
	panelKind common.ListKind
}

type playTrackOkMsg struct {
	panelKind common.ListKind
}
type startupCompleteMsg struct{}
type playerReadyMsg struct{}

type playerReadyErrMsg struct {
	err error
}

type playerEventMsg struct {
	event models.PlayerEvent
}

type playerEventsClosedMsg struct{}

func NewModel() *Model {
	model := &Model{
		authModel:   uiauth.NewModel(),
		mediaCenter: mediacenter.NewModel(),
		help:        newHelpModel(),
		keys:        common.NewAppKeyMap(),
	}
	model.requestHandlers = map[common.MediaRequestKind]func(common.MediaRequest) tea.Cmd{
		common.GetUserPlaylists:   model.handleGetUserPlaylists,
		common.GetSavedTracks:     model.handleGetSavedTracks,
		common.GetSavedAlbums:     model.handleGetSavedAlbums,
		common.GetFollowedArtists: model.handleGetFollowedArtists,
		common.SearchPlaylists:    model.handleSearchPlaylists,
		common.SearchTracks:       model.handleSearchTracks,
		common.SearchAlbums:       model.handleSearchAlbums,
		common.SearchArtists:      model.handleSearchArtists,
		common.GetPlaylistTracks:  model.handleGetPlaylistTracks,
		common.GetArtistAlbums:    model.handleGetArtistAlbums,
		common.GetAlbumTracks:     model.handleGetAlbumTracks,
		common.PlayTrack:          model.handlePlayTrackRequest,
	}
	return model
}

func newHelpModel() help.Model {
	h := help.New()
	h.ShowAll = false
	return h
}

func Run() error {
	model := NewModel()
	_, err := tea.NewProgram(model).Run()
	model.shutdown()
	return err
}

func (m *Model) Init() tea.Cmd {
	cmd := func() tea.Msg {
		err := m.start()
		if err != nil && m.authModel.State() == uiauth.Authenticated {
			return tea.Msg(err)
		}
		if m.authModel.State() == uiauth.NeedsAuth {
			return tea.Msg(m.authModel.State())
		}
		return startupCompleteMsg{}
	}
	return tea.Batch(cmd, ticker.StartTicker())
}

func (m *Model) setSize(width, height int) {
	m.width = width
	m.height = height
	m.help.SetWidth(width)
	if m.authModel != nil {
		m.authModel.SetSize(width, height)
	}
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
	m.authModel = uiauth.NewModel()
	if m.width != 0 || m.height != 0 {
		m.authModel.SetSize(m.width, m.height)
	}

	m.spotifyClient, err = spotify.NewSpotifyClient(ctx, m.authModel.Authenticator())
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.SetState(uiauth.NeedsAuth)
		}
		logger.Log.Error().Err(err).Msg("failed to create spotify client")
		return err
	}

	userID, err := m.spotifyClient.GetUserID(ctx)
	logger.Log.Info().Str("user id", userID).Msg("got user id")
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.SetState(uiauth.NeedsAuth)
		}
		return err
	}

	token, err := auth.New().GetAuthToken(ctx)
	if err != nil || token == nil {
		m.authModel.SetState(uiauth.NeedsAuth)
		return err
	}

	m.player = coreplayer.NewPlayer(ctx, userID, token.AccessToken)
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
		m.songInfo = common.SongInfo{
			Title:    ev.Metadata.Name,
			Artist:   artist,
			Album:    ev.Metadata.AlbumName,
			Position: ev.Metadata.Position,
			Duration: ev.Metadata.Duration,
		}
		m.mediaCenter.SetDisplayFromSong(m.songInfo)
	case models.EventTypePlaying:
		m.playing = true
	case models.EventTypePaused, models.EventTypeStopped:
		m.playing = false
		if ev.Type == models.EventTypeStopped {
			m.songInfo.Position = 0
		}
	case models.EventTypeSeek:
		if ev.Seek != nil {
			m.songInfo.Position = ev.Seek.Position
			m.songInfo.Duration = ev.Seek.Duration
		}
	case models.EventTypeVolume:
		if ev.Volume != nil {
			m.volumeInfo.Volume = ev.Volume.Value
			if ev.Volume.Max > 0 {
				m.volumeInfo.Max = ev.Volume.Max
			}
		}
	}
}

func (m *Model) markVolumeOverlay() {
	m.volumeOverlayUntil = time.Now().Add(1500 * time.Millisecond)
}

func (m *Model) previewVolume(delta int) {
	maxVolume := m.volumeInfo.Max
	if maxVolume <= 0 {
		maxVolume = 100
	}
	m.volumeInfo.Volume = max(0, min(maxVolume, m.volumeInfo.Volume+delta))
}

func (m *Model) advancePlayback(elapsedMs int) {
	if !m.playing || elapsedMs <= 0 || m.songInfo.Duration <= 0 {
		return
	}
	m.songInfo.Position = min(m.songInfo.Position+elapsedMs, m.songInfo.Duration)
	m.updatePlayerStatus()
}

func (m *Model) updatePlayerStatus() {
	maxVolume := m.volumeInfo.Max
	if maxVolume <= 0 {
		maxVolume = 100
	}
	m.mediaCenter.UpdatePlayerStatus(player.Status{
		PlayerReady: m.playerReady,
		Playing:     m.playing,
		Position:    m.songInfo.Position,
		Duration:    m.songInfo.Duration,
		Volume:      m.volumeInfo.Volume,
		MaxVolume:   maxVolume,
	})
}

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
	}, m.mediaCenter.ShowVolume())
}

func (m *Model) decrementVolumeCmd() tea.Cmd {
	return tea.Batch(func() tea.Msg {
		m.decrementVolume()
		return nil
	}, m.mediaCenter.ShowVolume())
}

func decodeOffsetCursor(cursor string) int {
	if cursor == "" {
		return 0
	}
	value, err := strconv.Atoi(cursor)
	if err != nil || value < 0 {
		return 0
	}
	return value
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

func paginationFromOffset(offset, count, total, pageSize int) common.PaginationInfo {
	currentPage := 1
	if pageSize > 0 {
		currentPage = (offset / pageSize) + 1
	}
	hasNext := offset+count < total
	nextCursor := ""
	if hasNext {
		nextCursor = encodeOffsetCursor(offset + pageSize)
	}
	return common.PaginationInfo{
		CurrentPage: currentPage,
		TotalPages:  totalPages(total, pageSize),
		TotalItems:  total,
		HasNext:     hasNext,
		NextCursor:  nextCursor,
	}
}

func paginationFromCursor(page, count, total, pageSize int, nextCursor string) common.PaginationInfo {
	hasNext := nextCursor != "" && count > 0
	if page <= 0 {
		page = 1
	}
	return common.PaginationInfo{
		CurrentPage: page,
		TotalPages:  totalPages(total, pageSize),
		TotalItems:  total,
		HasNext:     hasNext,
		NextCursor:  nextCursor,
	}
}

func ExitIfRunFails(err error) {
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to run program")
		os.Exit(1)
	}
}
