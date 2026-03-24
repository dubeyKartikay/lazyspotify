package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
)

type AuthServerErr struct {
	Err error
}

type AuthServerSuccess struct{}

type AuthServer struct {
	host             string
	port             int
	redirectEndpoint string
	timeout          time.Duration
	httpServer       *http.Server
	Started          atomic.Bool
}


func NewAuthServer() *AuthServer {
	cfg := utils.GetConfig().Auth
	return &AuthServer{
		host:             cfg.Host,
		port:             cfg.Port,
		redirectEndpoint: cfg.RedirectEndpoint,
		timeout:          time.Duration(cfg.Timeout) * time.Second,
	}
}

func (authServer *AuthServer) GetOauthRedirectURI() string {
	return fmt.Sprintf("http://%s:%d%s", authServer.host, authServer.port, authServer.redirectEndpoint)
}

func (authServer *AuthServer) GetAuthServerAddress() string {
	return fmt.Sprintf("%s:%d", authServer.host, authServer.port)
}

func (authServer *AuthServer) Start() chan error {
	return startServer(authServer)
}

func (authServer *AuthServer) Shutdown() error {
	if(!authServer.Started.Load()) {
    return nil
  }
	authServer.Started.Store(false)
	ctx, cancel := context.WithTimeout(context.Background(), authServer.timeout)
	defer cancel()
	if authServer.httpServer != nil {
		err := authServer.httpServer.Shutdown(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (authServer *AuthServer) InitAuthServer(oauthRedirectCallbackFunc func(w http.ResponseWriter, r *http.Request)) {
	mux := http.NewServeMux()
	registerRoutes(mux, authServer.redirectEndpoint, oauthRedirectCallbackFunc)
	server := &http.Server{
		Addr:    authServer.GetAuthServerAddress(),
		Handler: mux,
	}
	authServer.httpServer = server
}

func registerRoutes(mux *http.ServeMux, redirectEndpoint string, oauthRedirectCallbackFunc func(w http.ResponseWriter, r *http.Request)) {
	mux.HandleFunc(redirectEndpoint, oauthRedirectCallbackFunc)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		logger.Log.Debug().Str("url", r.URL.String()).Msg("got request")
	})
}

func startServer(authServer *AuthServer) chan error {
	if(authServer.Started.Load()) {
  	return nil
	}
	errCh := make(chan error, 1)
	authServer.Started.Store(true)
	go func() {
		err := authServer.httpServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()
	return errCh
}
