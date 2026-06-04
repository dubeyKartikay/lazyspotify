package daemon

import (
	"context"
	"strings"
	"testing"
)

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
