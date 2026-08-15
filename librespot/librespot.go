package librespot

import (
	"context"
	"fmt"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/daemon"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

type Librespot struct {
	Daemon daemon.DaemonManager
	Server *LibrespotApiServer
	Client *LibrespotApiClient
	Events *eventSocket
	Ready  chan error
}

func InitLibrespot(ctx context.Context, userId string, accessToken string, panicOnDaemonFailure bool) (*Librespot, error) {
	cfg := utils.GetConfig().Librespot
	logger.Log.Info().Str("config", fmt.Sprintf("%+v", cfg)).Msg("librespot config")
	librespotCommand, err := utils.ResolveLibrespotDaemonCmd(cfg.Daemon.Cmd)
	if err != nil {
		return nil, err
	}
	librespotCommand = append(append([]string{}, librespotCommand...), "--config_dir", GetLibrespotConfigDir())
	err = InitLibrespotConfig(ctx)
	if err != nil {
		return nil, err
	}
	daemonManager, err := daemon.NewDaemonManager(librespotCommand)
	if err != nil {
		return nil, err
	}

	librespotApiServer := NewLibrespotApiServer(cfg.Host, cfg.Port)
	librespotApiClient := NewLibrespotApiClient(librespotApiServer)
	librespotWs := newEventSocket(librespotApiServer.GetServerUrl())
	l := &Librespot{Daemon: daemonManager, Server: librespotApiServer, Client: librespotApiClient, Events: librespotWs, Ready: make(chan error, 1)}
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

func (l *Librespot) EventStream() <-chan models.PlayerEvent {
	if l.Events == nil {
		return nil
	}
	return l.Events.Events()
}
