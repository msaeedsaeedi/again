package domain

import (
	"time"
)

type VerbosityLevel string
type OutputFormat string

const (
	Silent  VerbosityLevel = "silent"
	Default VerbosityLevel = "default"
	Verbose VerbosityLevel = "verbose"
)

const (
	FormatTUI  OutputFormat = "tui"
	FormatJSON OutputFormat = "json"
	FormatRaw  OutputFormat = "raw"
)

type RunConfig struct {
	Command   []string
	Times     int
	Verbosity VerbosityLevel
	Format    OutputFormat
}

type RunResult struct {
	ID         int
	ExitCode   int
	Stdout     []byte
	Stderr     []byte
	Duration   time.Duration
	StartedAt  time.Time
	FinishedAt time.Time
	Success    bool
	Error      error
}
