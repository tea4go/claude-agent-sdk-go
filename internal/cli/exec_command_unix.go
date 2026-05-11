//go:build !windows

package cli

import (
	"context"
	"os/exec"
)

func NewExecCommandContext(ctx context.Context, args []string) *exec.Cmd {
	//nolint:gosec // G204: This is the core CLI SDK functionality - subprocess execution is required
	return exec.CommandContext(ctx, args[0], args[1:]...)
}
