package librespot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/dubeyKartikay/lazyspotify/librespot/models"
)

var ErrDeviceAuthUnsupported = errors.New("daemon does not support device authorization; rebuild the updated go-librespot fork")

func (l *LibrespotApiClient) GetAuthCode(ctx context.Context) (*models.DeviceAuth, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.server.GetServerUrl()+"/auth/code", nil)
	if err != nil {
		return nil, err
	}
	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrDeviceAuthUnsupported
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device authorization returned HTTP %d", resp.StatusCode)
	}
	var code models.DeviceAuth
	if err := json.NewDecoder(resp.Body).Decode(&code); err != nil {
		return nil, fmt.Errorf("decode device authorization: %w", err)
	}
	if code.URL == "" || code.Code == "" || code.ExpiresAt.IsZero() {
		return nil, ErrDeviceAuthUnsupported
	}
	return &code, nil
}

func readinessDeadline(deadline time.Time, code *models.DeviceAuth) time.Time {
	if code != nil && code.ExpiresAt.Add(30*time.Second).After(deadline) {
		return code.ExpiresAt.Add(30 * time.Second)
	}
	return deadline
}

func (l *Librespot) StopReadiness() {
	if l.cancelReadiness != nil {
		l.cancelReadiness()
	}
}

func (l *Librespot) publishAuthCode(code *models.DeviceAuth) {
	// Keep the latest code even if the UI has not yet subscribed.
	select {
	case <-l.AuthCodes:
	default:
	}
	l.AuthCodes <- code
}
