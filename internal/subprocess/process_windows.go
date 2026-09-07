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

// windowsProcessTree 基于 Job Object（首选）或 taskkill /T /F（回退）
// 在 Windows 上实现 processTree。
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
	// Windows 下的进程树接管在 Start 后通过 Job Object 完成；
	// 隐藏窗口属性由 cli.NewExecCommandContext 应用。
}

// attachProcessTree 在进程启动后尝试将其纳入 Job Object；
// 若失败（如宿主已施加限制性 Job）则回退到隐藏的 taskkill /T /F。
func attachProcessTree(cmd *exec.Cmd) (processTree, error) {
	if cmd == nil || cmd.Process == nil {
		return nil, errors.New("cannot attach process tree before process start")
	}

	tree := &windowsProcessTree{pid: cmd.Process.Pid}
	if err := tree.attachJob(); err != nil {
		// Some hosts place the SDK process in a restrictive Job Object. Keep the
		// session usable and retain a hidden taskkill /T /F fallback.
		// 某些宿主会将 SDK 进程置于受限的 Job Object 中；保持会话可用，
		// 并保留隐藏的 taskkill /T /F 回退方案。
		tree.useTaskkill = true
	}
	return tree, nil
}

// attachJob 创建 Job Object（设置 KILL_ON_JOB_CLOSE）并将 Claude 进程纳入，
// 以便一次性终止整棵进程树。
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
	// 隐藏进程没有控制台可接收 CTRL_BREAK_EVENT。
	// Close 会在等待前先关闭 stdin，从而让 Claude 自然退出。
	return nil
}

// forceStop 以 sync.Once 保证仅强制终止一次：优先终止 Job Object，
// 否则回退到 taskkill 杀死整棵进程树。
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

// forceKillWindowsProcessTree 通过 taskkill /T /F 杀死指定 PID 及其整棵进程树（回退方案）。
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

// wait 等待进程在给定超时内退出；退出返回 true，超时返回 false。
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

// durationToWindowsMilliseconds 将 time.Duration 转换为 Windows 等待 API 所需的毫秒数。
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

// close 以 sync.Once 关闭并释放进程与 Job Object 句柄。
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
