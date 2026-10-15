package librespot

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/daemon"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

type Librespot struct {
	Daemon          daemon.DaemonManager
	Server          *LibrespotApiServer
	Client          *LibrespotApiClient
	Events          *eventSocket
	Ready           chan error
	AuthCodes       chan *models.DeviceAuth
	cancelReadiness context.CancelFunc
}

func InitLibrespot(ctx context.Context) (*Librespot, error) {
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
	readyCtx, cancel := context.WithCancel(ctx)
	l := &Librespot{Daemon: daemonManager, Server: librespotApiServer, Client: librespotApiClient, Events: librespotWs, Ready: make(chan error, 1), AuthCodes: make(chan *models.DeviceAuth, 1), cancelReadiness: cancel}
	go notifyWhenReady(readyCtx, l)
	return l, nil
}

func notifyWhenReady(ctx context.Context, l *Librespot) {
	defer close(l.AuthCodes)
	deadline := time.Now().Add(3 * time.Minute)
	var previous *models.DeviceAuth
	for time.Now().Before(deadline) {
		pollCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		code, authErr := l.Client.GetAuthCode(pollCtx)
		cancel()
		if errors.Is(authErr, ErrDeviceAuthUnsupported) {
			l.Ready <- authErr
			return
		}
		if authErr == nil {
			deadline = readinessDeadline(deadline, code)
			if (code == nil) != (previous == nil) || (code != nil && previous != nil && *code != *previous) {
				l.publishAuthCode(code)
				previous = code
			}
		}
		// /auth/code is served before the session exists. The root health
		// request may block during device authorization, so do not send it
		// until approval has cleared the pending code.
		if code == nil {
			healthCtx, cancelHealth := context.WithTimeout(ctx, 2*time.Second)
			health, err := l.Client.GetHealthContext(healthCtx)
			cancelHealth()
			if err == nil && health.PlaybackReady {
				l.publishAuthCode(nil)
				l.Ready <- nil
				return
			}
		}
		select {
		case <-ctx.Done():
			l.Ready <- ctx.Err()
			return
		case <-time.After(time.Second):
		}
	}
	l.Ready <- fmt.Errorf("device authorization or daemon startup timed out; restart to request a new pairing code")
}

func (l *Librespot) EventStream() <-chan models.PlayerEvent {
	if l.Events == nil {
		return nil
	}
	return l.Events.Events()
}
