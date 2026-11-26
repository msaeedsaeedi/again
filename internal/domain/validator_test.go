package domain

import (
	"strings"
	"testing"
)

func TestValidateFormatAcceptsKnownFormats(t *testing.T) {
	for _, format := range []OutputFormat{FormatTUI, FormatJSON, FormatJSONL, FormatRaw} {
		if err := validateFormat(format); err != nil {
			t.Errorf("validateFormat(%q) returned error: %v", format, err)
		}
	}
}

func TestValidateFormatRejectsUnknownFormat(t *testing.T) {
	err := validateFormat(OutputFormat("xml"))
	if err == nil {
		t.Fatal("validateFormat(\"xml\") returned nil, want error")
	}
	msg := err.Error()
	if !strings.Contains(msg, "xml") {
		t.Errorf("error %q does not mention the rejected format", msg)
	}
	for _, valid := range []string{"tui", "json", "jsonl", "raw"} {
		if !strings.Contains(msg, valid) {
			t.Errorf("error %q does not list valid format %q", msg, valid)
		}
	}
}

func TestValidateAcceptsJSONLConfig(t *testing.T) {
	cfg := &RunConfig{
		Command:   []string{"echo", "hi"},
		Times:     1,
		Verbosity: VerbosityNormal,
		Format:    FormatJSONL,
	}
	if err := NewConfigValidator().Validate(cfg); err != nil {
		t.Fatalf("Validate returned error for jsonl config: %v", err)
	}
}
