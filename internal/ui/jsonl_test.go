package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/msaeedsaeedi/again/internal/domain"
)

func splitLines(t *testing.T, out string) []string {
	t.Helper()
	if !strings.HasSuffix(out, "\n") {
		t.Fatalf("output %q does not end with a newline", out)
	}
	return strings.Split(strings.TrimSuffix(out, "\n"), "\n")
}

func TestJSONLFormatterWritesOneLinePerCompletedRun(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONLFormatter(&buf)

	f.OnStart(1)
	if buf.Len() != 0 {
		t.Fatalf("OnStart wrote output: %q", buf.String())
	}

	f.OnComplete(domain.RunResult{ID: 1, Success: true, Duration: 7 * time.Millisecond, Stdout: []byte("a\n")})
	if got := splitLines(t, buf.String()); len(got) != 1 {
		t.Fatalf("after first run got %d lines, want 1: %q", len(got), buf.String())
	}

	f.OnStart(2)
	f.OnComplete(domain.RunResult{ID: 2, ExitCode: 1, Duration: 3 * time.Millisecond, Stderr: []byte("bad"), Error: errors.New("exit status 1")})
	f.OnFinish()

	lines := splitLines(t, buf.String())
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}

	want := []string{
		`{"type":"run","id":1,"exit_code":0,"success":true,"duration_ms":7,"stdout":"a\n"}`,
		`{"type":"run","id":2,"exit_code":1,"success":false,"duration_ms":3,"stderr":"bad","error":"exit status 1"}`,
	}
	for i := range want {
		if lines[i] != want[i] {
			t.Errorf("line %d:\ngot:  %s\nwant: %s", i+1, lines[i], want[i])
		}
	}
}

func TestJSONLFormatterEscapesMultilineOutput(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONLFormatter(&buf)

	stdout := "first\nsecond\r\nthird\n"
	stderr := "warn\n\tindented\n"
	f.OnComplete(domain.RunResult{ID: 1, Success: true, Stdout: []byte(stdout), Stderr: []byte(stderr)})

	lines := splitLines(t, buf.String())
	if len(lines) != 1 {
		t.Fatalf("got %d lines, want 1: %q", len(lines), buf.String())
	}

	var rec struct {
		Type     string  `json:"type"`
		ID       int     `json:"id"`
		ExitCode int     `json:"exit_code"`
		Success  bool    `json:"success"`
		Duration float64 `json:"duration_ms"`
		Stdout   string  `json:"stdout"`
		Stderr   string  `json:"stderr"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("line is not valid JSON: %v: %q", err, lines[0])
	}
	if rec.Type != "run" || rec.ID != 1 || !rec.Success {
		t.Errorf("unexpected record: %+v", rec)
	}
	if rec.Stdout != stdout || rec.Stderr != stderr {
		t.Errorf("stdout/stderr did not round-trip: %+v", rec)
	}
}

func TestJSONLRecordHasSameFieldsAsJSONResults(t *testing.T) {
	res := domain.RunResult{ID: 4, ExitCode: 2, Duration: time.Second, Stdout: []byte("o"), Stderr: []byte("e"), Error: errors.New("x")}

	jsonBytes, err := json.Marshal(newResultJSON(res))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	NewJSONLFormatter(&buf).OnComplete(res)

	var fromJSON, fromJSONL map[string]any
	if err := json.Unmarshal(jsonBytes, &fromJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(buf.Bytes(), &fromJSONL); err != nil {
		t.Fatal(err)
	}
	if fromJSONL["type"] != "run" {
		t.Errorf("type = %v, want run", fromJSONL["type"])
	}
	delete(fromJSONL, "type")
	if len(fromJSON) != len(fromJSONL) {
		t.Fatalf("field sets differ: json=%v jsonl=%v", fromJSON, fromJSONL)
	}
	for k, v := range fromJSON {
		if fromJSONL[k] != v {
			t.Errorf("field %q: json=%v jsonl=%v", k, v, fromJSONL[k])
		}
	}
}
