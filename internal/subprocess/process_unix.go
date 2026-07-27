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

type unixProcessTree struct {
	mu        sync.Mutex
	pgid      int
	forceOnce sync.Once
	forceErr  error
}

func configureProcessTree(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func attachProcessTree(cmd *exec.Cmd) (processTree, error) {
	if cmd == nil || cmd.Process == nil {
		return nil, errors.New("cannot attach process tree before process start")
	}
	return &unixProcessTree{pgid: cmd.Process.Pid}, nil
}

func (p *unixProcessTree) gracefulStop() error {
	return p.signal(syscall.SIGTERM)
}

func (p *unixProcessTree) forceStop() error {
	p.forceOnce.Do(func() {
		p.forceErr = p.signal(syscall.SIGKILL)
	})
	return p.forceErr
}

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

func (p *unixProcessTree) close() error {
	p.mu.Lock()
	p.pgid = 0
	p.mu.Unlock()
	return nil
}
