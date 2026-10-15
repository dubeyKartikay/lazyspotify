package daemon

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"os/exec"
)

type DaemonProcess struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	stderr *daemonErrorOutput
}

// Keep a bounded tail of stderr so a startup error includes the daemon's
// actual failure reason rather than only an exit status. Read after cmd.Wait,
// which waits for the stderr copying goroutine to finish.
type daemonErrorOutput struct{ tail []byte }

func (o *daemonErrorOutput) Write(p []byte) (int, error) {
	const maxBytes = 4096
	n := len(p)
	if len(p) >= maxBytes {
		o.tail = append(o.tail[:0], p[len(p)-maxBytes:]...)
	} else {
		if len(o.tail)+len(p) > maxBytes {
			o.tail = o.tail[len(o.tail)+len(p)-maxBytes:]
		}
		o.tail = append(o.tail, p...)
	}
	return n, nil
}

func (o *daemonErrorOutput) lastLine() string {
	output := strings.TrimSpace(string(o.tail))
	if i := strings.LastIndexByte(output, '\n'); i >= 0 {
		output = output[i+1:]
	}
	return output
}

func NewDaemonProcess(ctx context.Context, args []string) (DaemonProcess, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return DaemonProcess{}, fmt.Errorf("daemon command is empty")
	}
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	return DaemonProcess{cmd: cmd, cancel: cancel, stderr: &daemonErrorOutput{}}, nil
}

func (d *DaemonProcess) StartDaemon() error {

	lumberjackLogger := utils.NewLumberjackLogger("daemon.log")
	d.cmd.Stdout = lumberjackLogger
	d.cmd.Stderr = io.MultiWriter(lumberjackLogger, d.stderr)

	if err := d.cmd.Start(); err != nil {
		d.cancel()
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	logger.Log.Info().Msgf("daemon process %v", d.cmd.Process)
	return nil
}

func (d *DaemonProcess) MonitorDaemon(channel chan error) {
	err := d.cmd.Wait()
	if err != nil {
		if reason := d.stderr.lastLine(); reason != "" {
			err = fmt.Errorf("%w: %s", err, reason)
		}
		channel <- err
	}
}
