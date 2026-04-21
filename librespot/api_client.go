package librespot

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

const (
	healthPath        = "/"
	playPath          = "/player/play"
	playpausePath     = "/player/playpause"
	seekPath          = "/player/seek"
	nextPath          = "/player/next"
	previousPath      = "/player/prev"
	volumePath        = "/player/volume"
	resolveTracksPath = "/resolver/tracks"
	shufflePath      = "/player/shuffle_context"
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

func (l *LibrespotApiClient) GetHealth() (*models.HealthResponse, error) {
	url := l.server.GetServerUrl() + healthPath
	req, err := http.NewRequest("GET", url, nil)
	logger.Log.Debug().Str("url", url).Msg("requesting health")
	if err != nil {
		return nil, err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	healthRes, err := models.DecodeHealthResponse(resData)
	if err != nil {
		return nil, err
	}
	return &healthRes, nil
}

func (l *LibrespotApiClient) doRequest(ctx context.Context, method string, path string, body []byte) (*http.Response, error) {
	url := l.server.GetServerUrl() + path
	var requestBody io.Reader
	if body != nil {
		requestBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, requestBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	logger.Log.Debug().Msgf("requesting %+v", req)

	cfg := utils.GetConfig().Librespot
	return DoWithRetry(l.client, req, cfg.MaxRetries, time.Duration(cfg.RetryDelay)*time.Millisecond)
}

func (l *LibrespotApiClient) doPostStatus(ctx context.Context, path string, body []byte) int {
	resp, err := l.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		logger.Log.Error().Err(err).Str("path", path).Msg("request failed")
		return http.StatusInternalServerError
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

func (l *LibrespotApiClient) Play(ctx context.Context, uri string, skip_to_uri string, paused bool) int {
	playRequestJson, err := models.NewPlayRequest(uri, skip_to_uri, paused)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to marshal play request")
		return http.StatusInternalServerError
	}

	return l.doPostStatus(ctx, playPath, playRequestJson)
}

func (l *LibrespotApiClient) PlayPause(ctx context.Context) int {
	return l.doPostStatus(ctx, playpausePath, nil)
}

func (l *LibrespotApiClient) Seek(ctx context.Context, position int, relative bool) int {
	seekRequestJSON, err := models.NewSeekRequest(position, relative)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to marshal seek request")
		return http.StatusInternalServerError
	}
	return l.doPostStatus(ctx, seekPath, seekRequestJSON)
}

func (l *LibrespotApiClient) Next(ctx context.Context) int {
	return l.doPostStatus(ctx, nextPath, nil)
}

func (l *LibrespotApiClient) Previous(ctx context.Context) int {
	return l.doPostStatus(ctx, previousPath, nil)
}

func (l *LibrespotApiClient) SetVolume(ctx context.Context, volume int, relative bool) int {
	volumeRequestJSON, err := models.NewVolumeRequest(volume, relative)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to marshal volume request")
		return http.StatusInternalServerError
	}

	return l.doPostStatus(ctx, volumePath, volumeRequestJSON)
}

func (l *LibrespotApiClient) GetVolume(ctx context.Context) (*models.VolumeResponse, error) {
	resp, err := l.doRequest(ctx, http.MethodGet, volumePath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("failed to get volume: daemon returned status %d", resp.StatusCode)
	}

	resData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	volumeRes, err := models.DecodeVolumeResponse(resData)
	if err != nil {
		return nil, err
	}

	return &volumeRes, nil
}

func (l *LibrespotApiClient) Shuffle(ctx context.Context, shuffle bool) int {
	shuffleRequestJSON, err := models.NewShuffleRequest(shuffle)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to marshal shuffle request")
		return http.StatusInternalServerError
	}
	return l.doPostStatus(ctx, shufflePath, shuffleRequestJSON)
}

func (l *LibrespotApiClient) ResolvePlaylistTracks(ctx context.Context, uri string, offset int, limit int) (*models.ResolveTracksResponse, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}

	query := url.Values{}
	query.Set("uri", uri)
	query.Set("offset", fmt.Sprintf("%d", offset))
	query.Set("limit", fmt.Sprintf("%d", limit))

	path := resolveTracksPath + "?" + query.Encode()
	resp, err := l.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("failed to resolve playlist tracks: daemon returned status %d", resp.StatusCode)
	}

	resData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resolved, err := models.DecodeResolveTracksResponse(resData)
	if err != nil {
		return nil, err
	}
	if uri != "" && resolved.URI == "" {
		return nil, fmt.Errorf("failed to resolve playlist tracks: daemon endpoint /resolver/tracks is not available on this binary")
	}
	logger.Log.Debug().Any("response", resolved).Msg("resolved playlist tracks")
	return &resolved, nil
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
		} else {
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
