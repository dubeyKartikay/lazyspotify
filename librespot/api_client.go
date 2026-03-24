package librespot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

const (
  healthPath = "/"
	playPath = "/player/play"
	playpausePath = "/playpause"
)

type LibrespotApiServer struct {
	host string
	port int
}

type LibrespotApiClient struct {
	server *LibrespotApiServer
  client *http.Client
}

func NewLibrespotApiServer(host string, port int) *LibrespotApiServer {
  return &LibrespotApiServer{
    host: host,
    port: port,
  }
}

func (l *LibrespotApiServer) GetServerUrl() string {
	return fmt.Sprintf("http://%s:%d", l.host, l.port)
}

func NewLibrespotApiClient(server *LibrespotApiServer) *LibrespotApiClient {
	cfg := utils.GetConfig().Librespot
	client := http.Client{
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}
	return &LibrespotApiClient{
		client: &client,
		server: server,
	}
}

func (l *LibrespotApiClient) GetHealth() (*models.HealthResponse,error) {
	url := l.server.GetServerUrl() + healthPath;
	req, err := http.NewRequest("GET", url, nil)
	logger.Log.Debug().Str("url", url).Msg("requesting health")
	if err != nil {
		return nil,err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resData, err := io.ReadAll(resp.Body)
	if err != nil {
    return nil,err
  }
	healthRes, err := models.DecodeHealthResponse(resData)
	if err != nil {
    return nil, err
  }
	return &healthRes, nil
}

func (l *LibrespotApiClient) Play(ctx context.Context ,uri string, skip_to_uri string, paused bool) int{
  url := l.server.GetServerUrl() + playPath;
	playRequestJson,err := models.NewPlayRequest(uri, skip_to_uri, paused)
	if (err != nil) {
    return 500
	}
  req, err := http.NewRequestWithContext(ctx,"POST", url, bytes.NewReader(playRequestJson))
	req.Header.Set("Content-Type", "application/json")
  logger.Log.Debug().Msgf("requesting %+v", req)
  if err != nil {
    return 500
  }
	cfg := utils.GetConfig().Librespot
	resp, err := DoWithRetry(l.client, req, cfg.MaxRetries, time.Duration(cfg.RetryDelay)*time.Millisecond)
  if err != nil {
    logger.Log.Error().Err(err).Msg("play request failed")
    return 500
  }
  defer resp.Body.Close()
  return resp.StatusCode
}

func (l *LibrespotApiClient) PlayPause(ctx context.Context) int{
  url := l.server.GetServerUrl() + playpausePath;
  req, err := http.NewRequestWithContext(ctx,"POST", url, nil)
  logger.Log.Debug().Msgf("requesting %+v", req)
  if err != nil {
    return 500
  }
	cfg := utils.GetConfig().Librespot
	resp, err := DoWithRetry(l.client, req, cfg.MaxRetries, time.Duration(cfg.RetryDelay)*time.Millisecond)
  if err != nil {
    logger.Log.Error().Err(err).Msg("playpause request failed")
    return 500
  }
  defer resp.Body.Close()
  return resp.StatusCode
}

func DoWithRetry(client *http.Client, req *http.Request, maxRetries int, retryDelay time.Duration) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i <= maxRetries; i++ {

		if req.GetBody != nil {
			req.Body, _ = req.GetBody()
		}

		resp, err = client.Do(req)

		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		} else{
			logger.Log.Error().Err(err).Msg("request error")
		}

		if resp != nil {
			logger.Log.Debug().Msgf("%+v", resp)
			resp.Body.Close()
		}

		if i >= maxRetries {
			break
		}

		backoffDuration := time.Duration(math.Pow(2, float64(i))) * retryDelay
		logger.Log.Warn().Dur("backoff", backoffDuration).Msg("request failed, retrying")
		time.Sleep(backoffDuration)
	}

	return resp, fmt.Errorf("request failed after %d retries. Last error: %v", maxRetries, err)
}
