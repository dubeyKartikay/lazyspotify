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
	"golang.org/x/oauth2"
)

const (
	defaultCommandTimeout = 10 * time.Second
	// commandStdoutLimit and commandStderrLimit cap how much output is kept so
	// a chatty or hostile secret manager cannot exhaust memory.
	commandStdoutLimit = 64 << 10
	commandStderrLimit = 4 << 10
	// commandErrorLimit is how many stderr runes end up in an error message.
	commandErrorLimit = 300
	// tokenKeyEnv lets command backends use the configured token key without
	// it being baked into their argv.
	tokenKeyEnv = "LAZYSPOTIFY_TOKEN_KEY"
)

// CommandTokenStore keeps tokens in an external secret manager. The get
// command prints the token's JSON on stdout and exits with one of the
// configured not-found exit codes when no token is stored yet; the set command
// receives the token JSON on stdin. This is how `pass`, `oo7`, and similar
// tools integrate.
type CommandTokenStore struct {
	getCmd            []string
	setCmd            []string
	notFoundExitCodes map[int]struct{}
	anyExitIsNotFound bool
	timeout           time.Duration
}

func NewCommandTokenStore(getCmd, setCmd []string, notFoundExitCodes []int, timeout time.Duration) (*CommandTokenStore, error) {
	if len(getCmd) == 0 {
		return nil, errors.New("missing required config value `auth.token-store.command.get`")
	}
	if len(setCmd) == 0 {
		return nil, errors.New("missing required config value `auth.token-store.command.set`")
	}

	codes := make(map[int]struct{}, len(notFoundExitCodes))
	for _, code := range notFoundExitCodes {
		if code < 0 {
			return nil, fmt.Errorf(
				"invalid config value `auth.token-store.command.not-found-exit-codes`: %d; exit codes must be zero or positive",
				code,
			)
		}
		codes[code] = struct{}{}
	}
	if timeout <= 0 {
		timeout = defaultCommandTimeout
	}

	return &CommandTokenStore{
		getCmd: append([]string(nil), getCmd...),
		setCmd: append([]string(nil), setCmd...),
		// An empty list keeps the historical behaviour: any non-zero exit
		// means "no token yet".
		notFoundExitCodes: codes,
		anyExitIsNotFound: len(codes) == 0,
		timeout:           timeout,
	}, nil
}

func (c *CommandTokenStore) Get(ctx context.Context, key string) (*oauth2.Token, error) {
	stdout, stderr, err := c.run(ctx, c.getCmd, key, "")
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && c.isNotFoundExitCode(exitErr.ExitCode()) {
			logger.Log.Warn().
				Err(err).
				Str("command", strings.Join(c.getCmd, " ")).
				Int("exit_code", exitErr.ExitCode()).
				Str("stderr", truncate(stderr)).
				Msg("token get command reported a missing token; login required")
			return nil, ErrTokenNotFound
		}
		return nil, commandError("get", c.getCmd, err, stderr)
	}

	value := strings.TrimRight(stdout, "\r\n")
	if value == "" {
		logger.Log.Warn().
			Str("command", strings.Join(c.getCmd, " ")).
			Msg("token get command returned an empty secret; treating token as missing")
		return nil, ErrTokenNotFound
	}
	return decodeToken(key, value)
}

func (c *CommandTokenStore) Set(ctx context.Context, key string, token *oauth2.Token) error {
	encoded, err := encodeToken(token)
	if err != nil {
		return err
	}
	_, stderr, err := c.run(ctx, c.setCmd, key, encoded)
	if err != nil {
		return commandError("set", c.setCmd, err, stderr)
	}
	return nil
}

func (c *CommandTokenStore) isNotFoundExitCode(code int) bool {
	if c.anyExitIsNotFound {
		return true
	}
	_, ok := c.notFoundExitCodes[code]
	return ok
}

// run executes argv in its own process group with the token key exported and
// returns both capped output streams. Exit errors are returned unwrapped so
// callers can map them to "not found" or a real failure.
func (c *CommandTokenStore) run(ctx context.Context, argv []string, key, input string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), tokenKeyEnv+"="+key)
	cmd.Stdin = strings.NewReader(input)
	stdout := newCappedBuffer(commandStdoutLimit)
	stderr := newCappedBuffer(commandStderrLimit)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	// Keep descendants from holding the pipes open past the deadline.
	cmd.WaitDelay = 2 * time.Second
	configureProcessGroup(cmd)

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return stdout.String(), stderr.String(), fmt.Errorf("timed out after %s", c.timeout)
		}
		return stdout.String(), stderr.String(), err
	}
	return stdout.String(), stderr.String(), nil
}

// commandError describes a failing token command without leaking more output
// than a person needs to diagnose it.
func commandError(action string, argv []string, err error, stderr string) error {
	command := strings.Join(argv, " ")
	if message := truncate(stderr); message != "" {
		return fmt.Errorf("token %s command %q failed: %w: %s", action, command, err, message)
	}
	return fmt.Errorf("token %s command %q failed: %w", action, command, err)
}

// cappedBuffer keeps at most limit bytes and records that it truncated.
type cappedBuffer struct {
	limit     int
	buf       bytes.Buffer
	truncated bool
}

func newCappedBuffer(limit int) *cappedBuffer {
	return &cappedBuffer{limit: limit}
}

func (c *cappedBuffer) Write(p []byte) (int, error) {
	remaining := c.limit - c.buf.Len()
	if remaining <= 0 {
		if len(p) > 0 {
			c.truncated = true
		}
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = c.buf.Write(p[:remaining])
		c.truncated = true
		return len(p), nil
	}
	return c.buf.Write(p)
}

func (c *cappedBuffer) String() string {
	if !c.truncated {
		return c.buf.String()
	}
	return strings.ToValidUTF8(c.buf.String(), "") + "…"
}

func truncate(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= commandErrorLimit {
		return value
	}
	return string(runes[:commandErrorLimit]) + "…"
}
