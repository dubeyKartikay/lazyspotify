package deamon

import (
	"context"
	"fmt"
	"os/exec"
	"github.com/dubeyKartikay/lazyspotify/core/logger"
	"github.com/dubeyKartikay/lazyspotify/core/utils"
)

type DeamonProcess struct {
	cmd *exec.Cmd
	cancel context.CancelFunc
}

func NewDeamonProcess(ctx context.Context, args []string)(DeamonProcess,error){
  cmd := exec.CommandContext(ctx, args[0], args[1:]...)
  ctx, cancel := context.WithCancel(ctx)
  return DeamonProcess{cmd: cmd, cancel: cancel}, nil
}

func (d *DeamonProcess) StartDeamon() error {

  lumberjackLogger := utils.NewLumberjackLogger("deamon.log")
	d.cmd.Stdout = lumberjackLogger
	d.cmd.Stderr = lumberjackLogger

	if err := d.cmd.Start(); err != nil {
		d.cancel()
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	logger.Log.Info().Msgf("daemon process %v", d.cmd.Process)
	return nil
}

func (d *DeamonProcess) MonitorDeamon(channel chan error){
	err := d.cmd.Wait()
	if (err != nil){
    channel <- err
	}
}


