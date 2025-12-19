package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/msaeedsaeedi/again/internal/domain"
)

type RawFormatter struct{}

func NewRawFormatter() *RawFormatter {
	return &RawFormatter{}
}

func (f *RawFormatter) GetOutputWriters() (stdout, stderr io.Writer) {
	return os.Stdout, os.Stderr
}

func (f *RawFormatter) OnStart(runID int) {
	fmt.Fprintf(os.Stderr, "[ Run %d ]\n", runID)
}

func (f *RawFormatter) OnComplete(result domain.RunResult) {
	fmt.Fprintf(os.Stderr, "[ Run %d completed in %v", result.ID, result.Duration)

	if !result.Success {
		if result.Error != nil {
			fmt.Fprintf(os.Stderr, " - FAILED: exit code %d, error: %v", result.ExitCode, result.Error)
		} else {
			fmt.Fprintf(os.Stderr, " - FAILED: exit code %d", result.ExitCode)
		}
	} else {
		fmt.Fprintf(os.Stderr, " - SUCCESS")
	}
	fmt.Println(" ]")
}

func (f *RawFormatter) OnFinish() {
	// No-op
}
