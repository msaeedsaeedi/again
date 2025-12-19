package infra

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
	"syscall"
	"time"

	"github.com/msaeedsaeedi/again/internal/domain"
)

type CommandRunner struct{}

func NewCommandRunner() *CommandRunner {
	return &CommandRunner{}
}

func (r *CommandRunner) Run(ctx context.Context, cfg *domain.RunConfig, runID int, stdoutWriter, stderrWriter io.Writer) domain.RunResult {
	result := domain.RunResult{
		ID:        runID,
		StartedAt: time.Now(),
	}

	var cmd *exec.Cmd
	if len(cfg.Command) == 1 {
		cmd = exec.Command("sh", "-c", cfg.Command[0])
	} else {
		cmd = exec.Command(cfg.Command[0], cfg.Command[1:]...)
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	var stdout, stderr bytes.Buffer

	if stdoutWriter != nil {
		cmd.Stdout = io.MultiWriter(&stdout, stdoutWriter)
	} else {
		cmd.Stdout = &stdout
	}

	if stderrWriter != nil {
		cmd.Stderr = io.MultiWriter(&stderr, stderrWriter)
	} else {
		cmd.Stderr = &stderr
	}

	if err := cmd.Start(); err != nil {
		result.FinishedAt = time.Now()
		result.Duration = result.FinishedAt.Sub(result.StartedAt)
		result.Stdout = stdout.Bytes()
		result.Stderr = stderr.Bytes()
		result.Error = err
		result.Success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var err error
	select {
	case err = <-done:
		// normal exit
	case <-ctx.Done():
		// kill whole process group
		pgid := cmd.Process.Pid
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
		err = ctx.Err()
		<-done
	}
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			result.Error = errors.New("cancelled")
		} else {
			result.Error = err
		}
		result.Success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	} else {
		result.Success = true
		result.ExitCode = 0
	}

	return result
}
