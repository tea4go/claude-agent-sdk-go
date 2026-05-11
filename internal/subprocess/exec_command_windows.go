//go:build windows

package subprocess

import (
	"context"
	"os/exec"
	"syscall"
)

const createNoWindow uint32 = 0x08000000

func NewExecCommandContext(ctx context.Context, args []string) *exec.Cmd {
	//nolint:gosec // G204: This is the core CLI SDK functionality - subprocess execution is required
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd
}
