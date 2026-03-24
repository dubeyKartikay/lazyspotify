package librespot

import (
	"context"
	"fmt"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/deamon"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
)

type Librespot struct {
	Deamon deamon.DeamonManager
	Server *LibrespotApiServer
	Client *LibrespotApiClient
	Ready  chan error
}

func InitLibrespot(ctx context.Context, userId string, accessToken string,panicOnDaemonFailure bool) (*Librespot, error) {
	cfg := utils.GetConfig().Librespot
	librespotCommand := cfg.Daemon.Cmd
	librespotCommand = append(librespotCommand, "--config_dir", GetLibrespotConfigDir())
	err := InitLibrespotConfig(ctx, userId, accessToken)
  if err != nil {
    return nil, err
	}
	deamonManager, err := deamon.NewDeamonManager(librespotCommand)
	if err != nil {
		return nil, err
	}

	librespotApiServer := NewLibrespotApiServer(cfg.Host, cfg.Port)
	librespotApiClient := NewLibrespotApiClient(librespotApiServer)
	l := &Librespot{Deamon: deamonManager, Server: librespotApiServer, Client: librespotApiClient, Ready: make(chan error, 1)}
	go notifyWhenReady(l)
	return l, nil
}

func notifyWhenReady(l *Librespot) {
	for range 900 {
		healthRes, err := l.Client.GetHealth()

		if err == nil && healthRes.PlaybackReady {
			l.Ready <- nil
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	l.Ready <- fmt.Errorf("daemon did not become ready before timeout")
}


