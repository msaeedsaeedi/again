//go:build !windows

package infra

import (
	"os/exec"
	"syscall"
)

// setupProcessGroup sets up a process group for Unix systems
// This allows killing the entire process tree on cancellation
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcess kills the process group on Unix systems
func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		pgid := cmd.Process.Pid
		// Kill the entire process group
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}
}
