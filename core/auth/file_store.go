package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
)

const (
	defaultTokenFileName = "token.json"
	tokenFilePermission  = 0o600
	tokenDirPermission   = 0o700
)

// FileTokenStore keeps tokens in a plaintext JSON file. It exists for headless
// environments without Secret Service and D-Bus; anyone who can read the file
// can read the stored token.
type FileTokenStore struct {
	path string
}

// NewFileTokenStore builds a plaintext token store. An empty path resolves to
// token.json inside the lazyspotify config directory, and relative paths
// resolve against that directory so the file does not depend on the working
// directory lazyspotify was started from.
func NewFileTokenStore(path string) (*FileTokenStore, error) {
	resolved, err := expandHome(path)
	if err != nil {
		return nil, err
	}
	if resolved == "" {
		resolved = defaultTokenFileName
	}
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(utils.SafeGetConfigDir(), resolved)
	}
	return &FileTokenStore{path: resolved}, nil
}

// Path reports where the token file lives.
func (f *FileTokenStore) Path() string {
	return f.path
}

func (f *FileTokenStore) GetString(key string) (string, error) {
	values, err := f.read()
	if err != nil {
		return "", err
	}
	value, ok := values[key]
	if !ok || value == "" {
		return "", ErrTokenNotFound
	}
	return value, nil
}

func (f *FileTokenStore) SetString(key, value string) error {
	values, err := f.read()
	if err != nil && !errors.Is(err, ErrTokenNotFound) {
		return err
	}
	if values == nil {
		values = map[string]string{}
	}
	values[key] = value

	encoded, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token file %s: %w", f.path, err)
	}
	encoded = append(encoded, '\n')

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, tokenDirPermission); err != nil {
		return fmt.Errorf("create token directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".token-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary token file in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if err := tmp.Chmod(tokenFilePermission); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set permissions on %s: %w", tmpName, err)
	}
	if _, err := tmp.Write(encoded); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write token file %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close token file %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, f.path); err != nil {
		return fmt.Errorf("replace token file %s: %w", f.path, err)
	}
	return nil
}

// read loads the whole token file. A missing file reads as ErrTokenNotFound so
// that setup behaves like an empty store.
func (f *FileTokenStore) read() (map[string]string, error) {
	data, err := os.ReadFile(f.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, ErrTokenNotFound
		}
		return nil, fmt.Errorf("read token file %s: %w", f.path, err)
	}
	warnLoosePermissions(f.path)

	values := map[string]string{}
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf(
			"parse token file %s: %w; delete the file and log in again to rebuild it",
			f.path,
			err,
		)
	}
	return values, nil
}

func warnLoosePermissions(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.Mode().Perm()&0o077 == 0 {
		return
	}
	logger.Log.Warn().
		Str("path", path).
		Str("mode", info.Mode().Perm().String()).
		Msg("token file is readable by other users; run chmod 600 on it")
}

func expandHome(path string) (string, error) {
	if path != "~" && !strings.HasPrefix(path, "~/") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for token file path %q: %w", path, err)
	}
	return filepath.Join(home, path[1:]), nil
}
