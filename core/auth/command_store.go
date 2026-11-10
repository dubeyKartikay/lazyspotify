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

// NewCommandTokenStore builds a command store. The checks here repeat the ones
// in utils.validateTokenStoreConfig, which is authoritative for startup errors;
// these protect callers that build a store without loading config.
func NewCommandTokenStore(getCmd, setCmd []string, notFoundExitCodes []int, timeout time.Duration) (*CommandTokenStore, error) {
	if len(getCmd) == 0 {
		return nil, errors.New("missing required config value `auth.token-store.command.get`")
	}
	if len(setCmd) == 0 {
		return nil, errors.New("missing required config value `auth.token-store.command.set`")
	}

	codes := make(map[int]struct{}, len(notFoundExitCodes))
	for _, code := range notFoundExitCodes {
		// Exit 0 is success and can never be an ExitError; listing it would
		// silently disable the default any-non-zero mapping.
		if code <= 0 {
			return nil, fmt.Errorf(
				"invalid config value `auth.token-store.command.not-found-exit-codes`: %d; exit codes must be positive",
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
	out, err := c.run(ctx, c.getCmd, key, "")
	if err != nil {
		var exitErr *exec.ExitError
		// A negative exit code means the process died from a signal. That is
		// a failure, never "no token yet", even under the default mapping.
		if errors.As(err, &exitErr) && exitErr.ExitCode() >= 0 && c.isNotFoundExitCode(exitErr.ExitCode()) {
			logger.Log.Warn().
				Err(err).
				Str("command", strings.Join(c.getCmd, " ")).
				Int("exit_code", exitErr.ExitCode()).
				Str("stderr", truncate(out.stderr)).
				Msg("token get command reported a missing token; login required")
			return nil, ErrTokenNotFound
		}
		return nil, commandError("get", c.getCmd, err, out.stderr)
	}

	// Truncated output cannot be trusted: decoding it would look like a
	// corrupt token and start a login that overwrites a valid entry.
	if out.stdoutTruncated {
		return nil, fmt.Errorf(
			"token get command %q wrote more than %d bytes to stdout; refusing to read a truncated token",
			strings.Join(c.getCmd, " "),
			commandStdoutLimit,
		)
	}

	value := strings.TrimRight(out.stdout, "\r\n")
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
	out, err := c.run(ctx, c.setCmd, key, encoded)
	if err != nil {
		return commandError("set", c.setCmd, err, out.stderr)
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

// commandOutput carries capped command output plus whether either stream was
// cut off, so callers can refuse to parse incomplete data.
type commandOutput struct {
	stdout          string
	stderr          string
	stdoutTruncated bool
	stderrTruncated bool
}

// run executes argv in its own process group with the token key exported and
// returns both capped output streams. Exit errors are returned unwrapped so
// callers can map them to "not found" or a real failure.
func (c *CommandTokenStore) run(ctx context.Context, argv []string, key, input string) (commandOutput, error) {
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

	runErr := cmd.Run()
	out := commandOutput{
		stdout:          stdout.String(),
		stderr:          stderr.String(),
		stdoutTruncated: stdout.Truncated(),
		stderrTruncated: stderr.Truncated(),
	}
	if runErr != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			if errors.Is(ctxErr, context.DeadlineExceeded) {
				return out, fmt.Errorf("timed out after %s", c.timeout)
			}
			return out, fmt.Errorf("token command canceled: %w", ctxErr)
		}
		return out, runErr
	}
	return out, nil
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

// String returns the raw prefix that fit in the buffer. It never adds
// presentation characters: callers that need to display it use truncate.
func (c *cappedBuffer) String() string {
	return c.buf.String()
}

// Truncated reports whether bytes were dropped.
func (c *cappedBuffer) Truncated() bool {
	return c.truncated
}

func truncate(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= commandErrorLimit {
		return value
	}
	return string(runes[:commandErrorLimit]) + "…"
}
