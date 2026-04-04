package v1

import (
	"context"
	"fmt"

	"github.com/dubeyKartikay/lazyspotify/core/auth"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/player"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/spotify"
)

type Model struct {
	authModel     *AuthModel
	playing       bool
	songInfo      SongInfo
	volumeInfo    VolumeInfo
	player        *player.Player
	spotifyClient *spotify.SpotifyClient
	spoke         Spoke
	width         int
	height        int
}

type SongInfo struct {
	title    string
	artist   string
	album    string
	duration int
}

type VolumeInfo struct {
	volume int
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

	target := volume.Value - step
	if target < 0 {
		target = 0
	}

	if err := m.player.SetVolume(context.Background(), target, false); err != nil {
		logger.Log.Error().Err(err).Int("target_volume", target).Msg("failed to decrement volume")
		return
	}

	m.volumeInfo.volume = target
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

	target := volume.Value + step
	if target > volume.Max {
		target = volume.Max
	}

	if err := m.player.SetVolume(context.Background(), target, false); err != nil {
		logger.Log.Error().Err(err).Int("target_volume", target).Msg("failed to increment volume")
		return
	}

	m.volumeInfo.volume = target
}

func (m *Model) setSize(width, height int) {
	m.width = width
	m.height = height
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
	err = m.player.PlayTrack(context.Background(), uri)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to play daily mix")
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
	m.spoke = NewSpoke(m.width, m.height)
	var err error
	m.authModel = newAuthModel()
	if m.width != 0 || m.height != 0 {
		m.authModel.SetSize(m.width, m.height)
	}
	m.spotifyClient, err = spotify.NewSpotifyClient(ctx, m.authModel.auth)
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.needsAuth = true
		}
		logger.Log.Error().Err(err).Msg("failed to create spotify client")
		return err
	}
	userId, err := m.spotifyClient.GetUserID(ctx)
	logger.Log.Info().Str("user id", userId).Msg("got user id")
	if err != nil {
		if spotify.IsAuthError(err) {
			m.authModel.needsAuth = true
		}
		return err
	}

	tkn, err := auth.New().GetAuthToken(ctx)

	if err != nil || tkn == nil {
		m.authModel.needsAuth = true
		return err
	}

	m.player = player.NewPlayer(ctx, userId, tkn.AccessToken)

	m.player.Start(ctx)

	if m.player == nil {
		logger.Log.Error().Msg("failed to create player")
		return fmt.Errorf("failed to create player")
	}

	err = m.player.WaitTillReady()

	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to wait for player to be ready")
		return err
	}
	m.playDailyMix()

	return nil

}
