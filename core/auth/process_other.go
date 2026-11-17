//go:build !unix

package auth

import "os/exec"

// configureProcessGroup is a no-op on platforms without POSIX process groups;
// exec.CommandContext still terminates the direct child on deadline.
func configureProcessGroup(cmd *exec.Cmd) {}
