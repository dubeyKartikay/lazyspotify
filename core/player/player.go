package player

import (
	"context"
	"fmt"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/librespot"
)


type Player struct{
	librespot *librespot.Librespot
}
func NewPlayer(ctx context.Context,userId string, accessToken string) *Player{
	l, err := librespot.InitLibrespot(ctx, userId, accessToken, true)
  if err != nil {
    logger.Log.Error().Err(err).Msg("failed to init librespot")
    return nil
  }
  return &Player{
    librespot: l,
  }
}

func (p *Player) Play(ctx context.Context) error {
  return p.PlayTrack(ctx, "")
}

func (p *Player) PlayTrack(ctx context.Context, uri string) error {
	l := p.librespot
	logger.Log.Info().Str("uri", uri).Msg("playing track")
	res := l.Client.Play(ctx, uri, "", false)
	if res >= 400 {
		return fmt.Errorf("failed to play track: daemon returned status %d", res)
	}
	logger.Log.Info().Int("status", res).Msg("play response")
	return nil
}

func (p *Player) PlayPause(ctx context.Context) error {
  l := p.librespot
  logger.Log.Info().Msg("pausing track")
  res := l.Client.PlayPause(ctx)
  if res >= 400 {
    return fmt.Errorf("failed to pause track: daemon returned status %d", res)
  }
  logger.Log.Info().Int("status", res).Msg("pause response")
  return nil
}

func (p *Player) Start(ctx context.Context) error {
  l := p.librespot
	err := l.Deamon.StartDeamon()
  if err != nil {
    return err
  }
  return nil
}

func (p *Player) WaitTillReady() error {
  l := p.librespot
  return <- l.Ready
}

func (p *Player) Destroy(ctx context.Context) {
  l := p.librespot
  l.Deamon.StopDeamon()
}
