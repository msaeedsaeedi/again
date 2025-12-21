package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/msaeedsaeedi/again/internal/app"
	"github.com/msaeedsaeedi/again/internal/domain"
	"github.com/spf13/cobra"
)

const version = "0.1.0-beta"

type options struct {
	times     int
	json      bool
	raw       bool
	tui       bool
	verbosity string
}

func parseCommand(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}
	return args
}

func buildRunConfig(command []string, opts *options) *domain.RunConfig {
	var format string
	switch {
	case opts.json:
		format = "json"
	case opts.raw:
		format = "raw"
	case opts.tui:
		format = "tui"
	default:
		format = "tui"
	}
	cfg := &domain.RunConfig{
		Command:   command,
		Times:     opts.times,
		Verbosity: domain.VerbosityLevel(opts.verbosity),
		Format:    domain.OutputFormat(format),
	}
	return cfg
}

func run(args []string, opts *options) error {
	command := parseCommand(args)
	cfg := buildRunConfig(command, opts)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	orchestrator := app.NewOrchestrator()
	if err := orchestrator.Execute(ctx, cfg); err != nil {
		if err == context.Canceled {
			println("\n\nExecution cancelled")
			return nil
		}
		return err
	}

	return nil
}

func newRootCmd(opts *options) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "again [flags] -- <command>",
		Short:              "Run commands multiple times",
		Long:               "again - A powerful CLI tool to execute commands multiple times",
		SilenceErrors:      true,
		DisableFlagParsing: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, opts)
		},
		SilenceUsage: true,
	}

	cmd.Flags().IntVarP(&opts.times, "times", "n", 1, "Number of times to run a command")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.raw, "raw", false, "Output in raw format")
	cmd.Flags().BoolVar(&opts.tui, "tui", false, "Output in TUI format (default)")
	cmd.Flags().StringVarP(&opts.verbosity, "verbosity", "v", "normal", "Verbosity level (silent|normal|verbose)")
	cmd.Version = version

	return cmd
}

func main() {
	opts := &options{}
	rootCmd := newRootCmd(opts)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		fmt.Fprintln(os.Stderr)
		rootCmd.Usage()
		os.Exit(1)
	}
}
