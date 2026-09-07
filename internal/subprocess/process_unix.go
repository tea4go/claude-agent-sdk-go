//go:build !windows

package subprocess

import (
	"errors"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

const processTreePollInterval = 10 * time.Millisecond

// unixProcessTree 基于进程组（pgid）在类 Unix 系统上实现 processTree。
type unixProcessTree struct {
	mu        sync.Mutex
	pgid      int
	forceOnce sync.Once
	forceErr  error
}

// configureProcessTree 在启动前设置 Setpgid，使子进程自成一个进程组，
// 以便后续向整个进程组发送信号。
func configureProcessTree(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

// attachProcessTree 在进程启动后以其 PID 作为进程组 ID 创建 processTree。
func attachProcessTree(cmd *exec.Cmd) (processTree, error) {
	if cmd == nil || cmd.Process == nil {
		return nil, errors.New("cannot attach process tree before process start")
	}
	return &unixProcessTree{pgid: cmd.Process.Pid}, nil
}

// gracefulStop 向整个进程组发送 SIGTERM 以请求优雅退出。
func (p *unixProcessTree) gracefulStop() error {
	return p.signal(syscall.SIGTERM)
}

// forceStop 以 sync.Once 保证仅向进程组发送一次 SIGKILL 强制终止。
func (p *unixProcessTree) forceStop() error {
	p.forceOnce.Do(func() {
		p.forceErr = p.signal(syscall.SIGKILL)
	})
	return p.forceErr
}

// signal 向进程组（-pgid）发送指定信号；进程已不存在（ESRCH）视为成功。
func (p *unixProcessTree) signal(sig syscall.Signal) error {
	p.mu.Lock()
	pgid := p.pgid
	p.mu.Unlock()
	if pgid <= 0 {
		return nil
	}

	err := syscall.Kill(-pgid, sig)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}

// wait 轮询等待进程组退出，直到退出或超时；退出返回 true，超时返回 false。
func (p *unixProcessTree) wait(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if !p.alive() {
			return true
		}
		if timeout <= 0 || !time.Now().Before(deadline) {
			return false
		}
		sleep := processTreePollInterval
		if remaining := time.Until(deadline); remaining < sleep {
			sleep = remaining
		}
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}
}

// alive 通过向进程组发送 0 信号探测其是否存活。
func (p *unixProcessTree) alive() bool {
	p.mu.Lock()
	pgid := p.pgid
	p.mu.Unlock()
	if pgid <= 0 {
		return false
	}

	err := syscall.Kill(-pgid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// close 释放进程组引用（将 pgid 置零）；Unix 下无需额外句柄清理。
func (p *unixProcessTree) close() error {
	p.mu.Lock()
	p.pgid = 0
	p.mu.Unlock()
	return nil
}
