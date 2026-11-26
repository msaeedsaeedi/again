package ui

import (
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/msaeedsaeedi/again/internal/domain"
)

// captureStdout runs fn with os.Stdout redirected and returns what was written.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()

	fn()

	w.Close()
	out := <-done
	r.Close()
	return string(out)
}

func TestJSONFormatterOutputUnchanged(t *testing.T) {
	f := NewJSONFormatter(&domain.RunConfig{Times: 2})

	out := captureStdout(t, func() {
		f.OnStart(1)
		f.OnComplete(domain.RunResult{
			ID:       1,
			ExitCode: 0,
			Success:  true,
			Duration: 5 * time.Millisecond,
			Stdout:   []byte("line1\nline2\n"),
		})
		f.OnStart(2)
		f.OnComplete(domain.RunResult{
			ID:       2,
			ExitCode: 3,
			Success:  false,
			Duration: 12 * time.Millisecond,
			Stderr:   []byte("boom <&>\n"),
			Error:    errors.New("exit status 3"),
		})
		f.OnFinish()
	})

	want := `{
  "results": [
    {
      "id": 1,
      "exit_code": 0,
      "success": true,
      "duration_ms": 5,
      "stdout": "line1\nline2\n"
    },
    {
      "id": 2,
      "exit_code": 3,
      "success": false,
      "duration_ms": 12,
      "stderr": "boom \u003c\u0026\u003e\n",
      "error": "exit status 3"
    }
  ]
}
`
	if out != want {
		t.Errorf("json output changed.\ngot:\n%s\nwant:\n%s", out, want)
	}
}
