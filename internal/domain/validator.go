package domain

import (
	"errors"
	"fmt"
)

type ConfigValidator struct{}

func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

func validateFormat(format OutputFormat) error {
	switch format {
	case FormatRaw, FormatJSON, FormatTUI:
		return nil
	default:
		return fmt.Errorf("invalid output format: %s", format)
	}
}

func validateVerbosity(verbosity VerbosityLevel) error {
	switch verbosity {
	case VerbositySilent, VerbosityNormal, VerbosityVerbose:
		return nil
	default:
		return fmt.Errorf("invalid verbosity level: %s", verbosity)
	}
}

func (v *ConfigValidator) Validate(cfg *RunConfig) error {
	if len(cfg.Command) == 0 {
		return errors.New("command cannot be empty")
	}

	if cfg.Times < 1 {
		return errors.New("times must be at least 1")
	}

	if err := validateFormat(cfg.Format); err != nil {
		return err
	}

	if err := validateVerbosity(cfg.Verbosity); err != nil {
		return err
	}

	return nil
}
