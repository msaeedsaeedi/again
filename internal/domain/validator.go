package domain

import "errors"

type ConfigValidator struct{}

func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

func (v *ConfigValidator) Validate(cfg *RunConfig) error {
	if len(cfg.Command) == 0 {
		return errors.New("command cannot be empty")
	}

	if cfg.Times < 1 {
		return errors.New("times must be at least 1")
	}

	return nil
}
