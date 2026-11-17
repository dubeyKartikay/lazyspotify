package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"golang.org/x/oauth2"
)

const (
	defaultTokenFileName = "token.json"
	tokenFilePermission  = 0o600
	// tokenDirPermission applies only to directories this store creates; an
	// existing config directory keeps whatever mode it already has.
	tokenDirPermission = 0o700
	// staleTokenTempAge is how old a leftover .token-*.tmp must be before it
	// is swept. The margin keeps a concurrent writer's file safe.
	staleTokenTempAge = time.Hour
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
	path = strings.TrimSpace(path)
	resolved, err := expandHome(path)
	if err != nil {
		return nil, err
	}
	if resolved == "" {
		resolved = defaultTokenFileName
	}
	if !filepath.IsAbs(resolved) {
		configDir := utils.ConfigDir()
		if configDir == "" {
			return nil, errors.New("failed to resolve user config directory")
		}
		resolved = filepath.Join(configDir, resolved)
	}
	return &FileTokenStore{path: resolved}, nil
}

func (f *FileTokenStore) Get(ctx context.Context, key string) (*oauth2.Token, error) {
	_ = ctx
	values, err := f.read()
	if err != nil {
		return nil, err
	}
	value, ok := values[key]
	if !ok || value == "" {
		return nil, ErrTokenNotFound
	}
	return decodeToken(key, value)
}

func (f *FileTokenStore) Set(ctx context.Context, key string, token *oauth2.Token) error {
	_ = ctx
	encoded, err := encodeToken(token)
	if err != nil {
		return err
	}

	values, err := f.read()
	if err != nil && !errors.Is(err, ErrTokenNotFound) {
		return err
	}
	if values == nil {
		values = map[string]string{}
	}
	values[key] = encoded

	data, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token file %s: %w", f.path, err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, tokenDirPermission); err != nil {
		return fmt.Errorf("create token directory %s: %w", dir, err)
	}
	sweepStaleTokenTemps(dir)

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
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write token file %s: %w", tmpName, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync token file %s: %w", tmpName, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close token file %s: %w", tmpName, err)
	}
	if err := os.Rename(tmpName, f.path); err != nil {
		return fmt.Errorf("replace token file %s: %w", f.path, err)
	}
	syncDir(dir)
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
		quarantined, quarantineErr := quarantineTokenFile(f.path)
		if quarantineErr != nil {
			return nil, fmt.Errorf(
				"parse token file %s: %w; delete the file and log in again to rebuild it",
				f.path,
				err,
			)
		}
		logger.Log.Warn().
			Err(err).
			Str("path", f.path).
			Str("quarantined_path", quarantined).
			Msg("token file was unreadable; moved it aside so the next login can rebuild it")
		return nil, ErrTokenNotFound
	}
	return values, nil
}

// quarantineTokenFile moves an unreadable token file out of the way so the
// next login can write a fresh one without destroying the evidence.
func quarantineTokenFile(path string) (string, error) {
	base := fmt.Sprintf("%s.corrupt-%d", path, time.Now().UTC().Unix())
	for attempt := 0; attempt < 10; attempt++ {
		target := base
		if attempt > 0 {
			target = fmt.Sprintf("%s-%d", base, attempt)
		}
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		if err := os.Rename(path, target); err != nil {
			return "", err
		}
		return target, nil
	}
	return "", fmt.Errorf("could not find a free quarantine path for %s", path)
}

// sweepStaleTokenTemps removes temporary files left behind by a crash between
// create and rename. They hold token material, so they must not linger.
func sweepStaleTokenTemps(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-staleTokenTempAge)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, ".token-") || !strings.HasSuffix(name, ".tmp") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		full := filepath.Join(dir, name)
		if err := os.Remove(full); err == nil {
			logger.Log.Debug().Str("path", full).Msg("removed stale token temporary file")
		}
	}
}

// syncDir flushes the directory entry after a rename so the new file survives
// a crash. Directory fsync is best effort: some filesystems reject it.
func syncDir(dir string) {
	handle, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = handle.Sync()
	_ = handle.Close()
}

func warnLoosePermissions(path string) {
	info, err := os.Stat(path)
	if err != nil || !hasLoosePermissions(info.Mode()) {
		return
	}
	logger.Log.Warn().
		Str("path", path).
		Str("mode", info.Mode().Perm().String()).
		Msg("token file is readable by other users; run chmod 600 on it")
}

// hasLoosePermissions reports whether group or other users can access the file.
func hasLoosePermissions(mode fs.FileMode) bool {
	return mode.Perm()&0o077 != 0
}

func expandHome(path string) (string, error) {
	if path == "" || !strings.HasPrefix(path, "~") {
		return path, nil
	}
	if path == "~" || path == "~/" {
		return "", fmt.Errorf(
			"invalid config value `auth.token-store.file.path: %s`; set a file path such as ~/token.json",
			path,
		)
	}
	if !strings.HasPrefix(path, "~/") {
		return "", fmt.Errorf(
			"invalid config value `auth.token-store.file.path: %s`; only ~/ home expansion is supported",
			path,
		)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory for token file path %q: %w", path, err)
	}
	return filepath.Join(home, path[2:]), nil
}
