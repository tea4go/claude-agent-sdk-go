//go:build windows

package subprocess

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/tea4go/claude-agent-sdk-go/internal/cli"
	"golang.org/x/sys/windows"
)

const windowsTreeKillTimeout = 5 * time.Second

type windowsProcessTree struct {
	mu sync.Mutex

	pid         int
	job         windows.Handle
	process     windows.Handle
	useTaskkill bool
	forceOnce   sync.Once
	forceErr    error
	closeOnce   sync.Once
	closeErr    error
}

func configureProcessTree(_ *exec.Cmd) {
	// Windows process-tree ownership is attached after Start with a Job Object.
	// Hidden-window attributes are applied by cli.NewExecCommandContext.
}

func attachProcessTree(cmd *exec.Cmd) (processTree, error) {
	if cmd == nil || cmd.Process == nil {
		return nil, errors.New("cannot attach process tree before process start")
	}

	tree := &windowsProcessTree{pid: cmd.Process.Pid}
	if err := tree.attachJob(); err != nil {
		// Some hosts place the SDK process in a restrictive Job Object. Keep the
		// session usable and retain a hidden taskkill /T /F fallback.
		tree.useTaskkill = true
	}
	return tree, nil
}

func (p *windowsProcessTree) attachJob() error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return fmt.Errorf("create Job Object: %w", err)
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	ret, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	)
	if ret == 0 {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("configure Job Object: %w", err)
	}

	process, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE,
		false,
		uint32(p.pid),
	)
	if err != nil {
		_ = windows.CloseHandle(job)
		return fmt.Errorf("open Claude process: %w", err)
	}

	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		_ = windows.CloseHandle(process)
		_ = windows.CloseHandle(job)
		return fmt.Errorf("assign Claude process to Job Object: %w", err)
	}

	p.job = job
	p.process = process
	return nil
}

func (p *windowsProcessTree) gracefulStop() error {
	// Hidden processes do not have a console to receive CTRL_BREAK_EVENT.
	// Close closes stdin before waiting, allowing Claude to exit naturally.
	return nil
}

func (p *windowsProcessTree) forceStop() error {
	p.forceOnce.Do(func() {
		p.mu.Lock()
		job := p.job
		useTaskkill := p.useTaskkill
		pid := p.pid
		p.mu.Unlock()

		switch {
		case job != 0:
			p.forceErr = windows.TerminateJobObject(job, 1)
		case useTaskkill && pid > 0:
			p.forceErr = forceKillWindowsProcessTree(pid)
		}
	})
	return p.forceErr
}

func forceKillWindowsProcessTree(pid int) error {
	ctx, cancel := context.WithTimeout(context.Background(), windowsTreeKillTimeout)
	defer cancel()

	cmd := cli.NewExecCommandContext(ctx, []string{
		"taskkill.exe",
		"/PID", strconv.Itoa(pid),
		"/T",
		"/F",
	})
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill process tree %d: %w: %s", pid, err, output)
	}
	return nil
}

func (p *windowsProcessTree) wait(timeout time.Duration) bool {
	p.mu.Lock()
	process := p.process
	pid := p.pid
	p.mu.Unlock()

	temporaryHandle := false
	if process == 0 && pid > 0 {
		var err error
		process, err = windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
		if err != nil {
			return true
		}
		temporaryHandle = true
	}
	if process == 0 {
		return true
	}
	if temporaryHandle {
		defer windows.CloseHandle(process) //nolint:errcheck
	}

	waitMillis := durationToWindowsMilliseconds(timeout)
	event, err := windows.WaitForSingleObject(process, waitMillis)
	if err != nil {
		return false
	}
	return event == windows.WAIT_OBJECT_0
}

func durationToWindowsMilliseconds(timeout time.Duration) uint32 {
	if timeout <= 0 {
		return 0
	}
	millis := timeout / time.Millisecond
	if millis >= time.Duration(windows.INFINITE) {
		return windows.INFINITE - 1
	}
	if millis == 0 {
		return 1
	}
	return uint32(millis)
}

func (p *windowsProcessTree) close() error {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		process := p.process
		job := p.job
		p.process = 0
		p.job = 0
		p.pid = 0
		p.mu.Unlock()

		if process != 0 {
			if err := windows.CloseHandle(process); err != nil {
				p.closeErr = err
			}
		}
		if job != 0 {
			if err := windows.CloseHandle(job); err != nil && p.closeErr == nil {
				p.closeErr = err
			}
		}
	})
	return p.closeErr
}
