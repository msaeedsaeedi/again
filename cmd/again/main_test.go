package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "no double dash returns all args unchanged",
			args: []string{"echo", "hello"},
			want: []string{"echo", "hello"},
		},
		{
			name: "double dash returns all args after it",
			args: []string{"-f", "jsonl", "--", "echo", "a", "--", "echo", "b"},
			want: []string{"echo", "a", "--", "echo", "b"},
		},
		{
			name: "nested double dash is preserved",
			args: []string{"echo", "a", "--", "echo", "b"},
			want: []string{"echo", "a", "--", "echo", "b"},
		},
		{
			name: "empty args returns empty",
			args: []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCommand(tt.args); !equalStrings(got, tt.want) {
				t.Errorf("parseCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMainCommandWithDoubleDash(t *testing.T) {
	// The first "--" is handled by cobra (it is removed before run is called),
	// so parseCommand must not split on it again. Nested "--" must reach the
	// command unchanged.
	opts := &options{Format: "raw"}
	cmd := newRootCmd(opts)
	cmd.SetArgs([]string{"-f", "raw", "--", "echo", "a", "--", "echo", "b"})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "a -- echo b") {
		t.Fatalf("output = %q, want to contain %q", buf.String(), "a -- echo b")
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
