package app

import (
	"context"

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
	executor.Execute(ctx, cfg, handler)

	return nil
}

func getFormatter(cfg *domain.RunConfig) ResultHandler {
	switch cfg.Format {
	case domain.FormatRaw:
		return ui.NewRawFormatter()
	case domain.FormatJSON:
		return ui.NewJSONFormatter(cfg)
	default:
		return ui.NewRawFormatter()
	}
}
