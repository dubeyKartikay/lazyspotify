package lyrics

import (
	"net/http"
	"time"
)

type staticBearerRoundTripper struct {
	token string
	rt    http.RoundTripper
}

func (t *staticBearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	if t.rt == nil {
		t.rt = http.DefaultTransport
	}
	return t.rt.RoundTrip(r)
}

// HTTPClientForLyrics returns an HTTP client that attaches accessToken as a
// Bearer header and does not run OAuth2 token refresh. Using the oauth2
// auto-refresh transport for lyrics can persist a new refresh_token / access
// token while go-librespot still uses the previous access token from startup,
// which breaks playback.
func HTTPClientForLyrics(accessToken string) *http.Client {
	return &http.Client{
		Timeout: 20 * time.Second,
		Transport: &staticBearerRoundTripper{
			token: accessToken,
			rt:    http.DefaultTransport,
		},
	}
}
