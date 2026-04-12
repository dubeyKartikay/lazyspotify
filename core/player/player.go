package player

import (
	"context"
	"fmt"
	"strings"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/librespot"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

type Player struct {
	librespot *librespot.Librespot
}

func NewPlayer(ctx context.Context, userId string, accessToken string) (*Player, error) {
	l, err := librespot.InitLibrespot(ctx, userId, accessToken, true)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize librespot: %w", err)
	}
	return &Player{
		librespot: l,
	}, nil
}

func (p *Player) requireLibrespot() (*librespot.Librespot, error) {
	if p == nil {
		return nil, fmt.Errorf("player is not initialized")
	}
	if p.librespot == nil {
		return nil, fmt.Errorf("librespot is not initialized")
	}
	return p.librespot, nil
}

func (p *Player) PlayTrack(ctx context.Context, uri string, contextURI string) error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	playURI := uri
	skipToURI := ""
	if strings.HasPrefix(contextURI, "spotify:playlist:") || strings.HasPrefix(contextURI, "spotify:album:") {
		playURI = contextURI
		skipToURI = uri
	}

	logger.Log.Info().Str("uri", uri).Str("context_uri", contextURI).Str("play_uri", playURI).Str("skip_to_uri", skipToURI).Msg("playing track")
	res := l.Client.Play(ctx, playURI, skipToURI, false)
	if res >= 400 {
		return fmt.Errorf("failed to play track: daemon returned status %d", res)
	}
	logger.Log.Info().Int("status", res).Msg("play response")
	return nil
}

func (p *Player) PlayPause(ctx context.Context) error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	logger.Log.Info().Msg("pausing track")
	res := l.Client.PlayPause(ctx)
	if res >= 400 {
		return fmt.Errorf("failed to pause track: daemon returned status %d", res)
	}
	logger.Log.Info().Int("status", res).Msg("pause response")
	return nil
}

func (p *Player) Seek(ctx context.Context, position int, relative bool) error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	logger.Log.Info().Int("position", position).Bool("relative", relative).Msg("seeking track")
	res := l.Client.Seek(ctx, position, relative)
	if res >= 400 {
		return fmt.Errorf("failed to seek track: daemon returned status %d", res)
	}
	logger.Log.Info().Int("status", res).Msg("seek response")
	return nil
}

func (p *Player) Next(ctx context.Context) error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	logger.Log.Info().Msg("skipping to next track")
	res := l.Client.Next(ctx)
	if res >= 400 {
		return fmt.Errorf("failed to skip to next track: daemon returned status %d", res)
	}
	logger.Log.Info().Int("status", res).Msg("next response")
	return nil
}

func (p *Player) Previous(ctx context.Context) error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	logger.Log.Info().Msg("skipping to previous track")
	res := l.Client.Previous(ctx)
	if res >= 400 {
		return fmt.Errorf("failed to skip to previous track: daemon returned status %d", res)
	}
	logger.Log.Info().Int("status", res).Msg("previous response")
	return nil
}

func (p *Player) SetVolume(ctx context.Context, volume int, relative bool) error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	logger.Log.Info().Int("volume", volume).Bool("relative", relative).Msg("changing volume")
	res := l.Client.SetVolume(ctx, volume, relative)
	if res >= 400 {
		return fmt.Errorf("failed to set volume: daemon returned status %d", res)
	}
	logger.Log.Info().Int("status", res).Msg("volume response")
	return nil
}

func (p *Player) GetVolume(ctx context.Context) (*models.VolumeResponse, error) {
	l, err := p.requireLibrespot()
	if err != nil {
		return nil, err
	}
	return l.Client.GetVolume(ctx)
}

func (p *Player) GetPlaylistTracks(ctx context.Context, uri string, offset int, limit int) (*models.ResolveTracksResponse, error) {
	l, err := p.requireLibrespot()
	if err != nil {
		return nil, err
	}
	return l.Client.ResolvePlaylistTracks(ctx, uri, offset, limit)
}

func (p *Player) Start(ctx context.Context) error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	err = l.Deamon.StartDeamon()
	if err != nil {
		return err
	}
	l.Events.Start()
	return nil
}

func (p *Player) WaitTillReady() error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	return <-l.Ready
}

func (p *Player) WaitForDaemonFailure() error {
	l, err := p.requireLibrespot()
	if err != nil {
		return err
	}
	return <-l.Deamon.RestartFailErrorChannel
}

func (p *Player) Destroy(ctx context.Context) {
	l, err := p.requireLibrespot()
	if err != nil {
		logger.Log.Warn().Err(err).Msg("skipping player shutdown")
		return
	}
	if l.Events != nil {
		l.Events.Close()
	}
	l.Deamon.StopDeamon()
}

func (p *Player) Events() <-chan models.PlayerEvent {
	l, err := p.requireLibrespot()
	if err != nil {
		return nil
	}
	return l.EventStream()
}
