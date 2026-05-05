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
	"github.com/dubeyKartikay/lazyspotify/core/lyricsexport"
	coreplayer "github.com/dubeyKartikay/lazyspotify/core/player"
	"github.com/dubeyKartikay/lazyspotify/core/ticker"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
	"github.com/dubeyKartikay/lazyspotify/spotify"
	spotifylyrics "github.com/dubeyKartikay/lazyspotify/spotify/lyrics"
)

type Model struct {
	authModel          *uiauth.Model
	playing            bool
	playerReady        bool
	songInfo           common.SongInfo
	volumeInfo         common.VolumeInfo
	volumeOverlayUntil time.Time
	fatalErr           error
	player             *coreplayer.Player
	spotifyClient      *spotify.SpotifyClient
	mediaCenter        mediacenter.Model
	width              int
	height             int
	help               help.Model
	keys               common.AppKeyMap
	requestHandlers    map[common.MediaRequestKind]func(common.MediaRequest) tea.Cmd

	lyricsLines        []spotifylyrics.Line
	lyricsErr          string
	lyricsSync         string
	lyricsFetchPending string
	lyricsBroadcaster  *lyricsexport.Broadcaster
	lastSockLineIdx    int
	lastSockTrackURI   string
	lastSockPlaying    bool
	lyricsViewOpen     bool

	posAnchorMs   int
	posAnchorWall time.Time
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

type daemonRestartErrMsg struct {
	err error
}

type playerEventMsg struct {
	event models.PlayerEvent
}

type playerEventsClosedMsg struct{}

type lyricsLoadMsg struct {
	trackURI string
	track    *spotifylyrics.Track
	err      error
}

type playPauseOkMsg struct {
	playing bool
}

type volumeChangedMsg struct {
	volumeInfo common.VolumeInfo
}

type transportErrMsg struct {
	err    error
	action string
}

type shuffleOkMsg struct{}

type fatalErrMsg struct {
	err error
}

type fatalQuitMsg struct{}

func NewModel() *Model {
	keys := common.NewAppKeyMap()
	model := &Model{
		authModel:   uiauth.NewModel(),
		mediaCenter: mediacenter.NewModel(keys),
		help:        newHelpModel(),
		keys:        keys,
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
			return fatalErrMsg{err: err}
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
	if m.lyricsBroadcaster != nil {
		m.lyricsBroadcaster.Close()
		m.lyricsBroadcaster = nil
	}
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

	m.player, err = coreplayer.NewPlayer(ctx, userID, token.AccessToken)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to create player")
		return err
	}
	if err := m.player.Start(ctx); err != nil {
		logger.Log.Error().Err(err).Msg("failed to start player")
		m.player = nil
		return fmt.Errorf("failed to start librespot daemon: %w", err)
	}

	sockPath := strings.TrimSpace(utils.GetConfig().Lyrics.SocketPath)
	if sockPath != "" {
		b, err := lyricsexport.Start(sockPath)
		if err != nil {
			logger.Log.Warn().Err(err).Str("path", sockPath).Msg("lyrics socket not started")
		} else {
			m.lyricsBroadcaster = b
		}
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

func (m *Model) waitForDaemonRestartFailure() tea.Cmd {
	if m.player == nil {
		return nil
	}
	return func() tea.Msg {
		err := m.player.WaitForDaemonFailure()
		if err != nil {
			return daemonRestartErrMsg{err: err}
		}
		return nil
	}
}

func (m *Model) quitAfterFatalError() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		return fatalQuitMsg{}
	}
}

func (m *Model) setFatalError(err error) tea.Cmd {
	if err == nil {
		return nil
	}
	m.fatalErr = err
	logger.Log.Error().Err(err).Msg("fatal application error")
	return m.quitAfterFatalError()
}

func (m *Model) showActionError(action string, err error) {
	if err == nil {
		return
	}
	if action == "" {
		action = "Error"
	}
	m.mediaCenter.SetDisplay(fmt.Sprintf("%s: %v", action, err))
}

func (m *Model) applyPlayerEvent(ev models.PlayerEvent) {
	switch ev.Type {
	case models.EventTypeMetadata:
		if ev.Metadata == nil {
			return
		}
		prevURI := m.songInfo.TrackURI
		artist := strings.Join(ev.Metadata.ArtistNames, ", ")
		m.songInfo = common.SongInfo{
			Title:    ev.Metadata.Name,
			Artist:   artist,
			Album:    ev.Metadata.AlbumName,
			TrackURI: ev.Metadata.URI,
			Position: ev.Metadata.Position,
			Duration: ev.Metadata.Duration,
		}
		m.setPositionAnchor(ev.Metadata.Position)
		m.mediaCenter.SetDisplayFromSong(m.songInfo)
		if ev.Metadata.URI != prevURI {
			if _, ok := spotifylyrics.TrackIDFromURI(ev.Metadata.URI); ok {
				m.lyricsLines = nil
				m.lyricsErr = ""
				m.lyricsSync = ""
				m.lyricsFetchPending = ev.Metadata.URI
			} else {
				m.lyricsLines = nil
				m.lyricsErr = ""
				m.lyricsSync = ""
				m.lyricsFetchPending = ""
			}
		}
	case models.EventTypePlaying:
		m.setPositionAnchor(m.currentPositionMs())
		m.playing = true
	case models.EventTypePaused, models.EventTypeStopped:
		frozen := m.currentPositionMs()
		m.playing = false
		if ev.Type == models.EventTypeStopped {
			m.songInfo.Position = 0
			m.songInfo.TrackURI = ""
			m.setPositionAnchor(0)
			m.lyricsLines = nil
			m.lyricsErr = ""
			m.lyricsSync = ""
			m.lyricsFetchPending = ""
		} else {
			m.songInfo.Position = frozen
			m.setPositionAnchor(frozen)
		}
	case models.EventTypeSeek:
		if ev.Seek != nil {
			m.songInfo.Position = ev.Seek.Position
			m.songInfo.Duration = ev.Seek.Duration
			m.setPositionAnchor(ev.Seek.Position)
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

func (m *Model) setPositionAnchor(posMs int) {
	if posMs < 0 {
		posMs = 0
	}
	m.posAnchorMs = posMs
	m.posAnchorWall = time.Now()
}

func (m *Model) currentPositionMs() int {
	if !m.playing {
		return m.posAnchorMs
	}
	if m.posAnchorWall.IsZero() {
		return m.posAnchorMs
	}
	elapsed := int(time.Since(m.posAnchorWall) / time.Millisecond)
	pos := m.posAnchorMs + elapsed
	if m.songInfo.Duration > 0 && pos > m.songInfo.Duration {
		pos = m.songInfo.Duration
	}
	return pos
}

func (m *Model) refreshPlaybackPosition() {
	if m.songInfo.Duration <= 0 {
		return
	}
	m.songInfo.Position = m.currentPositionMs()
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
		Shuffled:    m.player.Shuffled(),
	})
}

func (m *Model) currentLyricIdx() int {
	return spotifylyrics.CurrentLineIndex(m.lyricsLines, m.currentPositionMs())
}

func (m *Model) syncLyricsView() {
	// Lyrics state lives on app.Model and is read by app/view.go when the
	// lyrics view is open. Nothing else to do here — index is recomputed
	// from the wall-clock anchor on each access.
}

func (m *Model) maybePublishLyricsSocket() {
	if m.lyricsBroadcaster == nil {
		return
	}
	idx := m.currentLyricIdx()
	lineText := ""
	if idx >= 0 && idx < len(m.lyricsLines) {
		lineText = m.lyricsLines[idx].Words
	}
	if idx == m.lastSockLineIdx && m.songInfo.TrackURI == m.lastSockTrackURI && m.playing == m.lastSockPlaying {
		return
	}
	m.lastSockLineIdx = idx
	m.lastSockTrackURI = m.songInfo.TrackURI
	m.lastSockPlaying = m.playing
	errText := m.lyricsErr
	m.lyricsBroadcaster.Publish(lyricsexport.Snapshot{
		TrackURI:    m.songInfo.TrackURI,
		Title:       m.songInfo.Title,
		Artist:      m.songInfo.Artist,
		PositionMs:  m.currentPositionMs(),
		DurationMs:  m.songInfo.Duration,
		Playing:     m.playing,
		LineIndex:   idx,
		LineText:    lineText,
		SyncType:    m.lyricsSync,
		LyricsError: errText,
	})
}

func (m *Model) fetchLyricsCmd(trackURI, title, artist, album string, durationMs int) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		tkn, err := m.authModel.Authenticator().GetAuthToken(ctx)
		if err != nil {
			return lyricsLoadMsg{trackURI: trackURI, err: err}
		}
		if tkn == nil || tkn.AccessToken == "" {
			return lyricsLoadMsg{trackURI: trackURI, err: fmt.Errorf("no access token for lyrics")}
		}
		client := spotifylyrics.HTTPClientForLyrics(tkn.AccessToken)
		meta := spotifylyrics.LRCLIBTrackMeta{
			Title: title, Artist: artist, Album: album, DurationMs: durationMs,
		}
		tr, err := spotifylyrics.FetchWithLRCLIBFallback(ctx, client, trackURI, meta)
		return lyricsLoadMsg{trackURI: trackURI, track: tr, err: err}
	}
}

func (m *Model) playPause() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.PlayPause(context.Background())
}

func (m *Model) seekForward() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	step := utils.GetConfig().Librespot.SeekStepMs
	return m.player.Seek(context.Background(), step, true)
}

func (m *Model) seekBackward() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	step := utils.GetConfig().Librespot.SeekStepMs
	return m.player.Seek(context.Background(), -step, true)
}

func (m *Model) next() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Next(context.Background())
}

func (m *Model) previous() error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Previous(context.Background())
}

func (m *Model) shuffle(shuffle bool) error {
	if m.player == nil {
		return fmt.Errorf("player not ready")
	}
	return m.player.Shuffle(context.Background(), shuffle)
}

func (m *Model) changeVolume(deltaPercent int) (common.VolumeInfo, error) {
	if m.player == nil {
		return common.VolumeInfo{}, fmt.Errorf("player not ready")
	}

	volume, err := m.player.GetVolume(context.Background())
	if err != nil {
		return common.VolumeInfo{}, err
	}

	maxVolume := volume.Max
	if maxVolume <= 0 {
		maxVolume = m.volumeInfo.Max
	}
	if maxVolume <= 0 {
		maxVolume = 65535
	}

	delta := calcVolumeDelta(maxVolume, deltaPercent)
	target := max(0, min(maxVolume, volume.Value+delta))
	if err := m.player.SetVolume(context.Background(), target, false); err != nil {
		return common.VolumeInfo{}, err
	}
	return common.VolumeInfo{Volume: target, Max: maxVolume}, nil
}

func (m *Model) playPauseCmd() tea.Cmd {
	targetPlaying := !m.playing
	return func() tea.Msg {
		if err := m.playPause(); err != nil {
			return transportErrMsg{err: err, action: "Failed to play/pause track"}
		}
		return playPauseOkMsg{playing: targetPlaying}
	}
}

func (m *Model) seekForwardCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.seekForward(); err != nil {
			return transportErrMsg{err: err, action: "Failed to seek forward"}
		}
		return nil
	}
}

func (m *Model) seekBackwardCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.seekBackward(); err != nil {
			return transportErrMsg{err: err, action: "Failed to seek backward"}
		}
		return nil
	}
}

func (m *Model) nextCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.next(); err != nil {
			return transportErrMsg{err: err, action: "Failed to skip to next track"}
		}
		return nil
	}
}

func (m *Model) previousCmd() tea.Cmd {
	return func() tea.Msg {
		if err := m.previous(); err != nil {
			return transportErrMsg{err: err, action: "Failed to skip to previous track"}
		}
		return nil
	}
}

func (m *Model) shuffleCmd() tea.Cmd {
	targetShuffle := !m.player.Shuffled()
	return func() tea.Msg {
		if err := m.shuffle(targetShuffle); err != nil {
			return transportErrMsg{err: err, action: "Failed to toggle shuffle"}
		}
		return shuffleOkMsg{}
	}
}

func (m *Model) incrementVolumeCmd() tea.Cmd {
	return func() tea.Msg {
		volumeInfo, err := m.changeVolume(utils.GetConfig().Librespot.VolumeStep)
		if err != nil {
			return transportErrMsg{err: err, action: "Failed to increase volume"}
		}
		return volumeChangedMsg{volumeInfo: volumeInfo}
	}
}

func (m *Model) decrementVolumeCmd() tea.Cmd {
	return func() tea.Msg {
		volumeInfo, err := m.changeVolume(-utils.GetConfig().Librespot.VolumeStep)
		if err != nil {
			return transportErrMsg{err: err, action: "Failed to decrease volume"}
		}
		return volumeChangedMsg{volumeInfo: volumeInfo}
	}
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

func calcVolumeDelta(maxVolume, stepPercent int) int {
	return maxVolume * stepPercent / 100
}

func ExitIfRunFails(err error) {
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to run program")
		os.Exit(1)
	}
}

func IsZenMode() bool {
	return os.Getenv("ZEN_MODE") != ""
}
