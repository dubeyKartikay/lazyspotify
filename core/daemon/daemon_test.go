package daemon

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestMonitorDaemonIncludesStartupFailure(t *testing.T) {
	process, err := NewDaemonProcess(context.Background(), []string{"sh", "-c", `printf '%s\n' 'initializing' 'clienttoken returned HTTP 400' >&2; exit 1`})
	if err != nil {
		t.Fatal(err)
	}
	defer process.cancel()
	process.cmd.Stderr = process.stderr
	if err := process.cmd.Start(); err != nil {
		t.Fatal(err)
	}
	errorsCh := make(chan error, 1)
	process.MonitorDaemon(errorsCh)
	err = <-errorsCh
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || !strings.Contains(err.Error(), "clienttoken returned HTTP 400") {
		t.Fatalf("daemon failure lost its cause: %v", err)
	}
}

func TestDaemonErrorOutputKeepsBoundedTail(t *testing.T) {
	output := &daemonErrorOutput{}
	for range 10 {
		_, _ = output.Write([]byte(strings.Repeat("x", 2000)))
	}
	_, _ = output.Write([]byte("\nfatal startup failure\n"))
	if len(output.tail) > 4096 || output.lastLine() != "fatal startup failure" {
		t.Fatal("stderr capture did not preserve the last line in bounded storage")
	}
	_, _ = output.Write([]byte(strings.Repeat("y", 10000) + "\nlast line\n"))
	if len(output.tail) != 4096 || output.lastLine() != "last line" {
		t.Fatal("large stderr write did not preserve the tail")
	}
}

func TestNewDaemonProcessBuildsCommand(t *testing.T) {
	process, err := NewDaemonProcess(context.Background(), []string{"librespot", "--config-dir", "/tmp/lazyspotify"})
	if err != nil {
		t.Fatalf("NewDaemonProcess() error = %v, want nil", err)
	}
	if process.cmd == nil {
		t.Fatal("process.cmd = nil, want command")
	}
	if process.cancel == nil {
		t.Fatal("process.cancel = nil, want cancel function")
	}
	wantArgs := []string{"librespot", "--config-dir", "/tmp/lazyspotify"}
	if len(process.cmd.Args) != len(wantArgs) {
		t.Fatalf("cmd.Args = %#v, want %#v", process.cmd.Args, wantArgs)
	}
	for i, want := range wantArgs {
		if process.cmd.Args[i] != want {
			t.Fatalf("cmd.Args[%d] = %q, want %q", i, process.cmd.Args[i], want)
		}
	}
}

func TestNewDaemonProcessRejectsEmptyCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing command", args: nil},
		{name: "empty command", args: []string{""}},
		{name: "blank command", args: []string{"  "}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			process, err := NewDaemonProcess(context.Background(), tt.args)
			if err == nil {
				t.Fatal("NewDaemonProcess() error = nil, want empty command error")
			}
			if process.cmd != nil {
				t.Fatalf("process.cmd = %#v, want nil command", process.cmd)
			}
			if !strings.Contains(err.Error(), "daemon command is empty") {
				t.Fatalf("NewDaemonProcess() error = %q, want empty command error", err.Error())
			}
		})
	}
}

func TestNewDaemonManagerRejectsEmptyCommand(t *testing.T) {
	manager, err := NewDaemonManager(nil)
	if err == nil {
		t.Fatal("NewDaemonManager(nil) error = nil, want empty command error")
	}
	if manager.daemonProcess.cmd != nil {
		t.Fatalf("manager.daemonProcess.cmd = %#v, want nil command", manager.daemonProcess.cmd)
	}
}
