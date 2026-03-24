package v1

import (
	"context"
	"fmt"

	"github.com/dubeyKartikay/lazyspotify/core/auth"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/player"
	"github.com/dubeyKartikay/lazyspotify/spotify"
)


type Model struct {
	authModel *AuthModel
	playing bool
  songInfo SongInfo
  volumeInfo VolumeInfo
	player *player.Player
	spotifyClient *spotify.SpotifyClient
	width int
	height int
}

type SongInfo struct {
  title string
	artist string
	album string
	duration int
}

type VolumeInfo struct {
  volume int
}


func (m *Model) playPause() {
  m.playing = !m.playing
}

func (m *Model) seekForward() {
  
}

func (m *Model) seekBackward() {
  
}
func (m *Model) next(){

}

func (m *Model) previous(){
  
}

func (m *Model) decrementVolume(){

}

func (m *Model) incrementVolume(){
  
}

func (m *Model) setSize(width, height int) {
  m.width = width
  m.height = height
	if(m.authModel != nil){
    m.authModel.SetSize(width, height)
  }
}

func (m *Model) start() error {
	ctx := context.Background()
	var err error
	m.authModel = newAuthModel()
	if(m.width != 0 || m.height != 0){
  	m.authModel.SetSize(m.width, m.height)
	}
	m.spotifyClient,err = spotify.NewSpotifyClient(ctx,m.authModel.auth)
  if err != nil{
		if spotify.IsAuthError(err){
      m.authModel.needsAuth = true
    }
    logger.Log.Error().Err(err).Msg("failed to create spotify client")
    return err
	}
	userId, err := m.spotifyClient.GetUserID(ctx)
	logger.Log.Info().Str("user id", userId).Msg("got user id")
  if err != nil{
		if spotify.IsAuthError(err){
      m.authModel.needsAuth = true
		}
    return err
  }

	tkn,err := auth.New().GetAuthToken(ctx)

  if err != nil || tkn == nil{
    m.authModel.needsAuth = true
    return err
  }

	m.player = player.NewPlayer(ctx, userId, tkn.AccessToken)

	m.player.Start(ctx)

	if m.player == nil{
		logger.Log.Error().Msg("failed to create player")
    return fmt.Errorf("failed to create player")
	}

	err = m.player.WaitTillReady()

	if err != nil {
    logger.Log.Error().Err(err).Msg("failed to wait for player to be ready")
    return err
  }

  return nil

}
