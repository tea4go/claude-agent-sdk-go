//go:build !windows

package subprocess

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/tea4go/claude-agent-sdk-go/internal/control"
	"github.com/tea4go/claude-agent-sdk-go/internal/shared"
)

func TestTransportConfiguresDedicatedProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := setupTransportForTest(t, newTransportMockCLI())
	defer func() {
		cancel()
		_ = transport.Close()
	}()

	connectTransportSafely(ctx, t, transport)

	if transport.cmd.SysProcAttr == nil || !transport.cmd.SysProcAttr.Setpgid {
		t.Fatal("Claude CLI command is not configured with a dedicated process group")
	}
}

func TestConnectContextCancellationKillsProcessGroup(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "process-tree.pids")
	script := createTransportTempScript(fmt.Sprintf(`#!/bin/bash
if [ "$1" = "-v" ]; then echo "3.0.0"; exit 0; fi
sleep 30 &
child=$!
echo "$$ $child" > %q
wait "$child"
`, pidFile), "")

	options := &shared.Options{}
	ctx, cancel := context.WithCancel(context.Background())
	transport := New(script, options, false, "sdk-go")

	var pids []int
	defer func() {
		cancel()
		for _, pid := range pids {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		_ = transport.Close()
	}()

	connectTransportSafely(ctx, t, transport)
	pids = waitForRecordedPIDs(t, pidFile, 2*time.Second)

	cancel()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		allStopped := true
		for _, pid := range pids {
			if processExists(pid) {
				allStopped = false
				break
			}
		}
		if allStopped {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	var alive []int
	for _, pid := range pids {
		if processExists(pid) {
			alive = append(alive, pid)
		}
	}
	t.Fatalf("context cancellation left process-tree members alive: %v", alive)
}

func TestCloseUsesSingleGracePeriod(t *testing.T) {
	script := createTransportTempScript(`#!/bin/bash
if [ "$1" = "-v" ]; then echo "3.0.0"; exit 0; fi
trap '' TERM
sh -c 'trap "" TERM; while true; do sleep 1; done' &
wait
`, "")

	originalGrace := processTerminationGracePeriod
	processTerminationGracePeriod = 250 * time.Millisecond
	defer func() {
		processTerminationGracePeriod = originalGrace
	}()

	transport := setupTransportForTest(t, script)
	connectTransportSafely(context.Background(), t, transport)

	start := time.Now()
	if err := transport.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	elapsed := time.Since(start)

	// Reader shutdown and process termination share one bound. The old order
	// consumed the fixed five-second reader timeout before starting process
	// termination, despite this test's 250ms grace period.
	if elapsed > 2*time.Second {
		t.Fatalf("Close() took %s; reader wait created a second timeout", elapsed)
	}
}

func TestAbortImmediatelyKillsProcessGroup(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "abort-process-tree.pids")
	script := createTransportTempScript(fmt.Sprintf(`#!/bin/bash
if [ "$1" = "-v" ]; then echo "3.0.0"; exit 0; fi
trap '' TERM
sh -c 'trap "" TERM; while true; do sleep 1; done' &
child=$!
echo "$$ $child" > %q
wait "$child"
`, pidFile), "")

	originalGrace := processTerminationGracePeriod
	processTerminationGracePeriod = 3 * time.Second
	defer func() {
		processTerminationGracePeriod = originalGrace
	}()

	transport := setupTransportForTest(t, script)
	connectTransportSafely(context.Background(), t, transport)
	pids := waitForRecordedPIDs(t, pidFile, 2*time.Second)
	defer func() {
		for _, pid := range pids {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		_ = transport.Close()
	}()

	start := time.Now()
	if err := transport.Abort(); err != nil {
		t.Fatalf("Abort() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed >= processTerminationGracePeriod {
		t.Fatalf("Abort() took %s, want less than graceful period %s", elapsed, processTerminationGracePeriod)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		allStopped := true
		for _, pid := range pids {
			if processExists(pid) {
				allStopped = false
				break
			}
		}
		if allStopped {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("Abort() left process-tree members alive: %v", pids)
}

func TestAbortEscalatesConcurrentGracefulClose(t *testing.T) {
	script := createTransportTempScript(`#!/bin/bash
if [ "$1" = "-v" ]; then echo "3.0.0"; exit 0; fi
trap '' TERM
while true; do sleep 1; done
`, "")

	originalGrace := processTerminationGracePeriod
	processTerminationGracePeriod = 3 * time.Second
	defer func() {
		processTerminationGracePeriod = originalGrace
	}()

	transport := setupTransportForTest(t, script)
	connectTransportSafely(context.Background(), t, transport)

	closeDone := make(chan error, 1)
	go func() {
		closeDone <- transport.Close()
	}()
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	if err := transport.Abort(); err != nil {
		t.Fatalf("Abort() error = %v", err)
	}
	if elapsed := time.Since(start); elapsed >= processTerminationGracePeriod {
		t.Fatalf("Abort() waited for full graceful period: %s", elapsed)
	}
	if err := <-closeDone; err != nil {
		t.Fatalf("concurrent Close() error = %v", err)
	}
}

func TestConnectRollbackKillsStartedProcessTree(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "rollback-tree.pids")
	script := createTransportTempScript(fmt.Sprintf(`#!/bin/bash
if [ "$1" = "-v" ]; then echo "3.0.0"; exit 0; fi
sleep 30 &
child=$!
echo "$$ $child" > %q
wait "$child"
`, pidFile), "")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	transport := New(
		script,
		&shared.Options{Hooks: map[control.HookEvent][]control.HookMatcher{}},
		false,
		"sdk-go",
	)

	err := transport.Connect(ctx)
	if err == nil {
		_ = transport.Close()
		t.Fatal("Connect() unexpectedly succeeded without initialize response")
	}
	t.Logf("Connect() failed as expected: %v", err)

	pids := waitForRecordedPIDs(t, pidFile, time.Second)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		allStopped := true
		for _, pid := range pids {
			if processExists(pid) {
				allStopped = false
				break
			}
		}
		if allStopped {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGKILL)
	}
	t.Fatalf("failed Connect left process-tree members alive: %v", pids)
}

func waitForRecordedPIDs(t *testing.T, path string, timeout time.Duration) []int {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			fields := strings.Fields(string(data))
			pids := make([]int, 0, len(fields))
			for _, field := range fields {
				pid, convErr := strconv.Atoi(field)
				if convErr != nil {
					t.Fatalf("invalid recorded PID %q: %v", field, convErr)
				}
				pids = append(pids, pid)
			}
			if len(pids) >= 2 {
				return pids
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for process PIDs in %s", path)
	return nil
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
