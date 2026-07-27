//go:build windows

package cli

import (
	"context"
	"testing"
)

func TestNewExecCommandContextAlwaysHidesWindow(t *testing.T) {
	cmd := NewExecCommandContext(context.Background(), []string{"cmd.exe", "/c", "exit", "0"})
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("HideWindow is false")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("CreationFlags = %#x, want CREATE_NO_WINDOW", cmd.SysProcAttr.CreationFlags)
	}
}
