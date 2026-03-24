package auth

import (
	"encoding/json"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)


type Keyring struct {
	service string
}

func NewSpotifyKeyring() *Keyring {
	return &Keyring{service: utils.GetConfig().Auth.Keyring.Service}
}

func (k *Keyring) GetString(key string) (string, error) {
  return keyring.Get(k.service, key)
}

func (k *Keyring) SetString(key, value string) error {
	return keyring.Set(k.service, key, value)
}

func (k *Keyring) GetToken(key string) (*oauth2.Token, error) {
	savedToken := oauth2.Token{}
  ser_token, err := k.GetString(key)
	if(err != nil) {
    return nil, err
  }
  err = json.Unmarshal([]byte(ser_token), &savedToken)
	if (err != nil){
		return nil , err
	}
	return &savedToken, nil
}

func (k *Keyring) SetToken(key string, token *oauth2.Token) error {
  ser_token, err := json.Marshal(token)
  if (err != nil){
    logger.Log.Error().Err(err).Msg("error marshaling token")
    return err
  }
  return k.SetString(key, string(ser_token))
}
