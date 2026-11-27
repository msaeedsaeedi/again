package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/msaeedsaeedi/again/internal/domain"
)

// RecordTypeRun is the "type" value of a jsonl line describing a completed run.
const RecordTypeRun = "run"

// RunRecordJSONL is a single jsonl line for a completed run. It carries the
// same fields as the entries of the json formatter's "results" array, plus a
// "type" discriminator.
type RunRecordJSONL struct {
	Type string `json:"type"`
	ResultJSON
}

// JSONLFormatter streams one JSON object per line as each run completes.
type JSONLFormatter struct {
	encoder *json.Encoder
	mu      sync.Mutex
}

func NewJSONLFormatter(out io.Writer) *JSONLFormatter {
	return &JSONLFormatter{
		encoder: json.NewEncoder(out),
	}
}

func (f *JSONLFormatter) GetOutputWriters() (stdout, stderr io.Writer) {
	return nil, nil
}

func (f *JSONLFormatter) OnStart(runID int) {
	// JSONL formatter doesn't output progress
}

func (f *JSONLFormatter) OnComplete(result domain.RunResult) {
	f.mu.Lock()
	defer f.mu.Unlock()

	record := RunRecordJSONL{
		Type:       RecordTypeRun,
		ResultJSON: newResultJSON(result),
	}

	// Encode writes compact JSON followed by a newline; newlines and other
	// control characters inside stdout/stderr are escaped, so each record
	// always occupies exactly one line.
	if err := f.encoder.Encode(record); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSONL output: %v\n", err)
		os.Exit(1)
	}
}

func (f *JSONLFormatter) OnFinish() {
	// Each run is written as it completes; nothing is buffered.
}
