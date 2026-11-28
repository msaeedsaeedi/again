//go:build !windows

package infra

import (
	"os/exec"
	"syscall"

	"github.com/msaeedsaeedi/again/internal/domain"
)

type processGuard struct{}

func (g *processGuard) Attach(cmd *exec.Cmd) {}

func (g *processGuard) Kill(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil && cmd.Process.Pid > 0 {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}

func (g *processGuard) Close() {}

func setupProcessGroup(cmd *exec.Cmd) *processGuard {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return &processGuard{}
}

func createCmd(cfg *domain.RunConfig) *exec.Cmd {
	var cmd *exec.Cmd
	if len(cfg.Command) == 1 {
		cmd = exec.Command("sh", "-c", cfg.Command[0])
	} else {
		cmd = exec.Command(cfg.Command[0], cfg.Command[1:]...)
	}
	return cmd
}
