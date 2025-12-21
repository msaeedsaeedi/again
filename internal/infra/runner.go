package infra

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"runtime"
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

	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	var cmd *exec.Cmd
	if len(cfg.Command) == 1 {
		if runtime.GOOS == "windows" {
			cmd = exec.Command("cmd", "/C", cfg.Command[0])
		} else {
			cmd = exec.Command("sh", "-c", cfg.Command[0])
		}
	} else {
		cmd = exec.Command(cfg.Command[0], cfg.Command[1:]...)
	}

	setupProcessGroup(cmd)

	// Limit output buffer size to prevent OOM (10 MB per stream)
	const maxOutputSize = 10 * 1024 * 1024
	stdout := &limitedBuffer{buf: &bytes.Buffer{}, limit: maxOutputSize}
	stderr := &limitedBuffer{buf: &bytes.Buffer{}, limit: maxOutputSize}

	if stdoutWriter != nil {
		cmd.Stdout = io.MultiWriter(stdout, stdoutWriter)
	} else {
		cmd.Stdout = stdout
	}

	if stderrWriter != nil {
		cmd.Stderr = io.MultiWriter(stderr, stderrWriter)
	} else {
		cmd.Stderr = stderr
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
		// Kill the process or process group
		killProcess(cmd)
		err = ctx.Err()
		<-done
	}
	result.FinishedAt = time.Now()
	result.Duration = result.FinishedAt.Sub(result.StartedAt)
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()

	// Add warning if output was truncated
	if stdout.truncated {
		truncMsg := []byte("\n[OUTPUT TRUNCATED: exceeded 10MB limit]\n")
		result.Stdout = append(result.Stdout, truncMsg...)
	}
	if stderr.truncated {
		truncMsg := []byte("\n[OUTPUT TRUNCATED: exceeded 10MB limit]\n")
		result.Stderr = append(result.Stderr, truncMsg...)
	}

	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
			result.Error = errors.New("cancelled")
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			result.Error = fmt.Errorf("timeout: command exceeded %v", cfg.Timeout)
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

type limitedBuffer struct {
	buf       *bytes.Buffer
	limit     int
	truncated bool
}

func (lb *limitedBuffer) Write(p []byte) (n int, err error) {
	if lb.buf.Len() >= lb.limit {
		lb.truncated = true
		return len(p), nil
	}

	available := lb.limit - lb.buf.Len()
	if len(p) > available {
		lb.buf.Write(p[:available])
		lb.truncated = true
		return len(p), nil
	}

	return lb.buf.Write(p)
}

func (lb *limitedBuffer) Bytes() []byte {
	return lb.buf.Bytes()
}
