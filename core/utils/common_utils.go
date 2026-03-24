package utils

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)



func SpotifyURLToURI(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	parts := strings.Split(u.Path, "/")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid spotify url")
	}

	typ := parts[1]
	id := parts[2]

	return fmt.Sprintf("spotify:%s:%s", typ, id), nil
}

func EnsureExists(dir string) error {
  if _, err := os.Stat(dir); os.IsNotExist(err) {
    return os.MkdirAll(dir, 0755)
  }
  return nil
}

