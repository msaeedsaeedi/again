package infra

import (
	"bytes"
	"context"
	"os/exec"
	"time"

	"github.com/msaeedsaeedi/again/internal/domain"
)

type CommandRunner struct{}

func NewCommandRunner() *CommandRunner {
	return &CommandRunner{}
}

func (r *CommandRunner) Run(ctx context.Context, cfg *domain.RunConfig, runID int) domain.RunResult {
	result := domain.RunResult{
		ID:        runID,
		StartedAt: time.Now(),
	}

	var cmd *exec.Cmd
	if len(cfg.Command) == 1 {
		cmd = exec.CommandContext(ctx, "sh", "-c", cfg.Command[0])
	} else {
		cmd = exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	if err != nil {
		result.Error = err
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
