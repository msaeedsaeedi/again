package app

import (
	"context"
	"io"

	"github.com/msaeedsaeedi/again/internal/domain"
	"github.com/msaeedsaeedi/again/internal/infra"
)

type ResultHandler interface {
	OnStart(runID int)
	OnComplete(result domain.RunResult)
	OnFinish()
	GetOutputWriters() (stdout, stderr io.Writer)
}

type Executor interface {
	Execute(ctx context.Context, cfg *domain.RunConfig, handler ResultHandler) error
}

type SequentialExecutor struct {
	runner *infra.CommandRunner
}

func NewSequentialExecutor(runner *infra.CommandRunner) *SequentialExecutor {
	return &SequentialExecutor{
		runner: runner,
	}
}

func (e *SequentialExecutor) Execute(ctx context.Context, cfg *domain.RunConfig, handler ResultHandler) error {
	results := make([]domain.RunResult, 0, cfg.Times)

	for i := 1; i <= cfg.Times; i++ {
		select {
		case <-ctx.Done():
			handler.OnFinish()
			return ctx.Err()
		default:
		}

		handler.OnStart(i)

		stdoutWriter, stderrWriter := handler.GetOutputWriters()
		result := e.runner.Run(ctx, cfg, i, stdoutWriter, stderrWriter)
		results = append(results, result)

		handler.OnComplete(result)
	}

	handler.OnFinish()
	return nil
}

func NewExecutor(cfg *domain.RunConfig, runner *infra.CommandRunner) Executor {
	return NewSequentialExecutor(runner)
}
