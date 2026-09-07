//go:build windows

package cli

import (
	"context"
	"os/exec"
	"syscall"
)

// createNoWindow 是 Windows CREATE_NO_WINDOW 标志，用于避免子进程弹出控制台窗口。
const createNoWindow uint32 = 0x08000000

// NewExecCommandContext 在 Windows 系统上创建绑定到给定 context 的 CLI 子进程命令。
// 通过设置 HideWindow 与 CREATE_NO_WINDOW 标志，避免弹出多余的控制台窗口。
func NewExecCommandContext(ctx context.Context, args []string) *exec.Cmd {
	//nolint:gosec // G204: This is the core CLI SDK functionality - subprocess execution is required
	// G204：这是 CLI SDK 的核心功能——必须执行子进程
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd
}
