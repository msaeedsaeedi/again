package infra

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os/exec"
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
		cmd = exec.CommandContext(ctx, "sh", "-c", cfg.Command[0])
	} else {
		cmd = exec.CommandContext(ctx, cfg.Command[0], cfg.Command[1:]...)
	}

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

	err := cmd.Run()
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	if err != nil {
		if ctx.Err() == context.Canceled {
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
