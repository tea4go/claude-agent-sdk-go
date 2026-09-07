//go:build !windows

package cli

import (
	"context"
	"os/exec"
)

// NewExecCommandContext 在类 Unix 系统上创建绑定到给定 context 的 CLI 子进程命令。
// args[0] 为可执行文件路径，其余为命令行参数。
func NewExecCommandContext(ctx context.Context, args []string) *exec.Cmd {
	//nolint:gosec // G204: This is the core CLI SDK functionality - subprocess execution is required
	// G204：这是 CLI SDK 的核心功能——必须执行子进程
	return exec.CommandContext(ctx, args[0], args[1:]...)
}
