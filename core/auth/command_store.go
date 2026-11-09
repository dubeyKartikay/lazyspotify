package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
)

const (
	defaultCommandTimeout = 10 * time.Second
	// commandStderrLimit caps how much command output ends up in errors so a
	// chatty secret manager cannot flood the log or the auth screen.
	commandStderrLimit = 300
	// tokenKeyEnv lets command backends use the configured token key without
	// it being baked into their argv.
	tokenKeyEnv = "LAZYSPOTIFY_TOKEN_KEY"
)

// CommandTokenStore keeps tokens in an external secret manager. The get
// command must print the secret on stdout (a trailing newline is fine) and
// exit non-zero when no secret is stored yet; the set command receives the
// secret on stdin. This is how `pass`, `oo7`, and similar tools integrate.
type CommandTokenStore struct {
	getCmd  []string
	setCmd  []string
	timeout time.Duration
}

func NewCommandTokenStore(getCmd, setCmd []string, timeout time.Duration) (*CommandTokenStore, error) {
	if len(getCmd) == 0 {
		return nil, fmt.Errorf("missing command token store get command")
	}
	if len(setCmd) == 0 {
		return nil, fmt.Errorf("missing command token store set command")
	}
	if timeout <= 0 {
		timeout = defaultCommandTimeout
	}
	return &CommandTokenStore{
		getCmd:  append([]string(nil), getCmd...),
		setCmd:  append([]string(nil), setCmd...),
		timeout: timeout,
	}, nil
}

func (c *CommandTokenStore) GetString(key string) (string, error) {
	stdout, stderr, err := c.run(c.getCmd, key, "")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Secret managers exit non-zero for "no such entry", which is the
			// same state as an empty store and should lead to a fresh login.
			logger.Log.Debug().
				Err(err).
				Str("command", c.getCmd[0]).
				Str("stderr", truncate(stderr)).
				Msg("token get command exited non-zero; treating token as missing")
			return "", ErrTokenNotFound
		}
		return "", fmt.Errorf("run token get command %q: %w", c.getCmd[0], err)
	}

	value := strings.TrimRight(stdout, "\r\n")
	if value == "" {
		logger.Log.Debug().
			Str("command", c.getCmd[0]).
			Msg("token get command returned an empty secret; treating token as missing")
		return "", ErrTokenNotFound
	}
	return value, nil
}

func (c *CommandTokenStore) SetString(key, value string) error {
	_, stderr, err := c.run(c.setCmd, key, value)
	if err != nil {
		if message := truncate(stderr); message != "" {
			return fmt.Errorf("run token set command %q: %w: %s", c.setCmd[0], err, message)
		}
		return fmt.Errorf("run token set command %q: %w", c.setCmd[0], err)
	}
	return nil
}

// run executes argv with the token key exported and returns both output
// streams. Exit errors are returned unwrapped so callers can tell a missing
// secret from a command that could not run at all.
func (c *CommandTokenStore) run(argv []string, key, input string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), tokenKeyEnv+"="+key)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return "", stderr.String(), fmt.Errorf("timed out after %s", c.timeout)
		}
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

func truncate(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= commandStderrLimit {
		return value
	}
	return value[:commandStderrLimit] + "…"
}
