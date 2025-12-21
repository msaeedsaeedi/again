package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/msaeedsaeedi/again/internal/domain"
)

type ResultJSON struct {
	ID       int     `json:"id"`
	ExitCode int     `json:"exit_code"`
	Success  bool    `json:"success"`
	Duration float64 `json:"duration_ms"`
	Stdout   string  `json:"stdout,omitempty"`
	Stderr   string  `json:"stderr,omitempty"`
	Error    string  `json:"error,omitempty"`
}

type JSONFormatter struct {
	config  *domain.RunConfig
	results []domain.RunResult
	mu      sync.Mutex
}

func NewJSONFormatter(cfg *domain.RunConfig) *JSONFormatter {
	return &JSONFormatter{
		config:  cfg,
		results: make([]domain.RunResult, 0),
	}
}

func (f *JSONFormatter) GetOutputWriters() (stdout, stderr io.Writer) {
	return nil, nil
}

func (f *JSONFormatter) OnStart(runID int) {
	// JSON formatter doesn't output progress
}

func (f *JSONFormatter) OnComplete(result domain.RunResult) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results = append(f.results, result)
}

func (f *JSONFormatter) OnFinish() {
	f.mu.Lock()
	defer f.mu.Unlock()

	results := make([]ResultJSON, 0, len(f.results))

	for _, res := range f.results {
		resultJSON := ResultJSON{
			ID:       res.ID,
			ExitCode: res.ExitCode,
			Success:  res.Success,
			Duration: float64(res.Duration.Milliseconds()),
			Stdout:   string(res.Stdout),
			Stderr:   string(res.Stderr),
		}
		if res.Error != nil {
			resultJSON.Error = res.Error.Error()
		}
		results = append(results, resultJSON)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(map[string][]ResultJSON{"results": results}); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON output: %v\n", err)
		os.Exit(1)
	}
}
