package app

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/msaeedsaeedi/again/internal/domain"
	"github.com/msaeedsaeedi/again/internal/infra"
	"github.com/msaeedsaeedi/again/internal/ui"
)

type Orchestrator struct {
	validator *domain.ConfigValidator
	runner    *infra.CommandRunner
}

func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		validator: domain.NewConfigValidator(),
		runner:    infra.NewCommandRunner(),
	}
}

func (o *Orchestrator) Execute(ctx context.Context, cfg *domain.RunConfig) error {
	if err := o.validator.Validate(cfg); err != nil {
		return err
	}

	executor := NewExecutor(cfg, o.runner)
	handler := getFormatter(cfg)

	if cfg.Format == domain.FormatTUI {
		tuiHandler, ok := handler.(*ui.TUIFormatter)
		if !ok {
			return fmt.Errorf("tui formatter not available")
		}
		return o.executeTUI(ctx, executor, cfg, tuiHandler)
	}

	return executor.Execute(ctx, cfg, handler)
}

func (o *Orchestrator) executeTUI(ctx context.Context, executor Executor, cfg *domain.RunConfig, tui *ui.TUIFormatter) error {
	ctxRun, cancel := context.WithCancel(ctx)
	defer cancel()

	g, gctx := errgroup.WithContext(ctxRun)

	// Start TUI; when it exits (quit or finish), cancel to stop executor
	g.Go(func() error {
		defer cancel()
		return tui.Run(gctx)
	})

	// Ensure the TUI program is initialized before starting executor so streaming works
	if err := tui.WaitReady(gctx); err != nil {
		return err
	}

	// Start executor
	g.Go(func() error {
		if err := executor.Execute(gctx, cfg, tui); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func getFormatter(cfg *domain.RunConfig) ResultHandler {
	switch cfg.Format {
	case domain.FormatRaw:
		return ui.NewRawFormatter()
	case domain.FormatJSON:
		return ui.NewJSONFormatter(cfg)
	case domain.FormatTUI:
		return ui.NewTUIFormatter(cfg)
	default:
		return ui.NewTUIFormatter(cfg)
	}
}
