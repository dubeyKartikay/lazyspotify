//go:build unix

package auth

import (
	"os/exec"
	"syscall"
)

// configureProcessGroup puts the command in its own process group so a
// timeout kills helpers the secret manager spawned too.
func configureProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
