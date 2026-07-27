//go:build windows

package subprocess

import (
	"context"
	"testing"
	"time"

	"github.com/tea4go/claude-agent-sdk-go/internal/cli"
)

func TestWindowsProcessTreeForceStop(t *testing.T) {
	processCtx, processCancel := context.WithCancel(context.Background())
	defer processCancel()

	cmd := cli.NewExecCommandContext(processCtx, []string{
		"cmd.exe", "/c", "ping", "127.0.0.1", "-n", "30",
	})
	configureProcessTree(cmd)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start Windows helper: %v", err)
	}

	tree, err := attachProcessTree(cmd)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("attach process tree: %v", err)
	}
	defer tree.close() //nolint:errcheck

	if err := tree.forceStop(); err != nil {
		processCancel()
		_ = cmd.Wait()
		t.Fatalf("force process tree: %v", err)
	}
	if !tree.wait(2 * time.Second) {
		processCancel()
		_ = cmd.Wait()
		t.Fatal("Windows process tree did not stop")
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("forced Windows helper unexpectedly exited successfully")
	}
}

func TestDurationToWindowsMilliseconds(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want uint32
	}{
		{name: "poll", in: 0, want: 0},
		{name: "round up sub-millisecond", in: time.Nanosecond, want: 1},
		{name: "seconds", in: 2 * time.Second, want: 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := durationToWindowsMilliseconds(tt.in); got != tt.want {
				t.Fatalf("durationToWindowsMilliseconds(%s) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
