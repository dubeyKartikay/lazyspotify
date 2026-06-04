package daemon

import (
	"context"
	"fmt"
	"strings"

	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
	"os/exec"
)

type DaemonProcess struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
}

func NewDaemonProcess(ctx context.Context, args []string) (DaemonProcess, error) {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return DaemonProcess{}, fmt.Errorf("daemon command is empty")
	}
	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	return DaemonProcess{cmd: cmd, cancel: cancel}, nil
}

func (d *DaemonProcess) StartDaemon() error {

	lumberjackLogger := utils.NewLumberjackLogger("daemon.log")
	d.cmd.Stdout = lumberjackLogger
	d.cmd.Stderr = lumberjackLogger

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
		channel <- err
	}
}
