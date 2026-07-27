package subprocess

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"
)

var processTerminationGracePeriod = 5 * time.Second

// processTree owns the Claude CLI root process and all ordinary descendants.
// Implementations must tolerate concurrent forceStop calls from context
// cancellation and explicit Close.
type processTree interface {
	gracefulStop() error
	forceStop() error
	wait(time.Duration) bool
	close() error
}

// isProcessAlreadyFinishedError checks if an error indicates the process has
// already terminated. These conditions are successful cleanup outcomes.
func isProcessAlreadyFinishedError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "process already finished") ||
		strings.Contains(errStr, "process already released") ||
		strings.Contains(errStr, "no child processes") ||
		strings.Contains(errStr, "signal: killed") ||
		strings.Contains(errStr, "signal: terminated")
}

func (t *Transport) watchCallerCancellation(
	ctxDone <-chan struct{},
	stop <-chan struct{},
	done chan<- struct{},
	tree processTree,
	cmd *exec.Cmd,
	processCancel func(),
	ioCancel func(),
) {
	defer close(done)

	select {
	case <-ctxDone:
		atomic.StoreUint32(&t.cancellationRequested, 1)
		if tree != nil {
			_ = tree.forceStop()
		}
		if processCancel != nil {
			processCancel()
		}
		t.startProcessWaiter(cmd)
		if ioCancel != nil {
			ioCancel()
		}
	case <-stop:
	}
}

// terminateProcess performs one graceful process-tree wait followed by a hard
// tree kill. Caller-context cancellation and explicit Abort skip the grace period.
func (t *Transport) terminateProcess() error {
	if t.cmd == nil || t.cmd.Process == nil {
		return nil
	}

	waitDone := t.startProcessWaiter(t.cmd)
	tree := t.processTree
	var cleanupErr error

	if tree != nil {
		if atomic.LoadUint32(&t.cancellationRequested) != 0 {
			if err := tree.forceStop(); err != nil && !isProcessAlreadyFinishedError(err) {
				cleanupErr = fmt.Errorf("force process tree: %w", err)
			}
		} else {
			if err := tree.gracefulStop(); err != nil && !isProcessAlreadyFinishedError(err) {
				cleanupErr = fmt.Errorf("gracefully stop process tree: %w", err)
			}
			if !tree.wait(processTerminationGracePeriod) {
				if err := tree.forceStop(); err != nil &&
					!isProcessAlreadyFinishedError(err) &&
					cleanupErr == nil {
					cleanupErr = fmt.Errorf("force process tree after timeout: %w", err)
				}
			}
		}
	}

	if t.processCancel != nil {
		t.processCancel()
	}

	<-waitDone
	waitErr := t.processWaitErr
	if waitErr != nil &&
		!isProcessAlreadyFinishedError(waitErr) &&
		!strings.Contains(waitErr.Error(), "signal:") &&
		cleanupErr == nil {
		var exitErr *exec.ExitError
		if !errors.As(waitErr, &exitErr) {
			cleanupErr = fmt.Errorf("wait for Claude CLI process: %w", waitErr)
		}
	}

	if tree != nil {
		if err := tree.close(); err != nil &&
			!isProcessAlreadyFinishedError(err) &&
			cleanupErr == nil {
			cleanupErr = fmt.Errorf("release process tree: %w", err)
		}
	}

	return cleanupErr
}

func (t *Transport) startProcessWaiter(cmd *exec.Cmd) <-chan struct{} {
	t.processWaitOnce.Do(func() {
		go func() {
			t.processWaitErr = cmd.Wait()
			close(t.processWaitDone)
		}()
	})
	return t.processWaitDone
}

// cleanup releases pipes, temporary files, and platform process handles.
func (t *Transport) cleanup() {
	if t.stdin != nil {
		_ = t.stdin.Close()
		t.stdin = nil
	}

	if t.stdout != nil {
		_ = t.stdout.Close()
		t.stdout = nil
	}

	if t.stderrPipe != nil {
		_ = t.stderrPipe.Close()
		t.stderrPipe = nil
	}

	if t.stderr != nil {
		_ = t.stderr.Close()
		_ = os.Remove(t.stderr.Name())
		t.stderr = nil
	}

	if t.mcpConfigFile != nil {
		_ = t.mcpConfigFile.Close()
		_ = os.Remove(t.mcpConfigFile.Name())
		t.mcpConfigFile = nil
	}

	t.cleanupSkillRegistryDirs()

	if t.processTree != nil {
		_ = t.processTree.close()
		t.processTree = nil
	}
	if t.processCancel != nil {
		t.processCancel()
		t.processCancel = nil
	}

	t.cmd = nil
	t.watcherDone = nil
	t.processWaitDone = nil
	t.processWaitErr = nil
}

func (t *Transport) cleanupSkillRegistryDirs() {
	for _, dir := range t.skillRegistryDirs {
		_ = os.RemoveAll(dir)
	}
	t.skillRegistryDirs = nil
}
