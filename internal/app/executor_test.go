package app

import (
	"bytes"
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/msaeedsaeedi/again/internal/domain"
	"github.com/msaeedsaeedi/again/internal/infra"
	"github.com/msaeedsaeedi/again/internal/ui"
)

// syncBuffer is a bytes.Buffer safe for concurrent use.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// recordingHandler wraps a handler and snapshots the output written so far
// whenever a run starts.
type recordingHandler struct {
	ResultHandler
	out            *syncBuffer
	outputAtStarts map[int]string
}

func (h *recordingHandler) OnStart(runID int) {
	h.outputAtStarts[runID] = h.out.String()
	h.ResultHandler.OnStart(runID)
}

func TestJSONLStreamsEachRunAsItCompletes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX shell command")
	}

	const times = 3
	out := &syncBuffer{}
	handler := &recordingHandler{
		ResultHandler:  ui.NewJSONLFormatter(out),
		out:            out,
		outputAtStarts: map[int]string{},
	}
	cfg := &domain.RunConfig{
		Command: []string{"printf 'hello\\nworld\\n'"},
		Times:   times,
		Format:  domain.FormatJSONL,
	}

	if err := NewSequentialExecutor(infra.NewCommandRunner()).Execute(context.Background(), cfg, handler); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// The first line must already be written before the last run starts.
	before := handler.outputAtStarts[times]
	if n := strings.Count(before, "\n"); n != times-1 {
		t.Fatalf("before run %d started, got %d lines, want %d: %q", times, n, times-1, before)
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != times {
		t.Fatalf("got %d lines, want %d: %q", len(lines), times, out.String())
	}
	for i, line := range lines {
		var rec struct {
			Type    string `json:"type"`
			ID      int    `json:"id"`
			Success bool   `json:"success"`
			Stdout  string `json:"stdout"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("line %d is not valid JSON: %v: %q", i+1, err, line)
		}
		if rec.Type != "run" || rec.ID != i+1 || !rec.Success || rec.Stdout != "hello\nworld\n" {
			t.Errorf("line %d: unexpected record %+v", i+1, rec)
		}
	}
}

func TestGetFormatterJSONL(t *testing.T) {
	if _, ok := getFormatter(&domain.RunConfig{Format: domain.FormatJSONL}).(*ui.JSONLFormatter); !ok {
		t.Fatal("getFormatter did not return a JSONLFormatter for jsonl")
	}
}
