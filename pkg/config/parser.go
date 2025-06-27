package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
)

func LoadFromFile(filename string) (*models.Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var rawConfig rawConfig
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	config, err := parseConfig(&rawConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

type rawConfig struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Global      rawGlobalConfig `json:"global"`
	Tests       []rawTestCase   `json:"tests"`
}

type rawGlobalConfig struct {
	BaseURL            string            `json:"base_url"`
	Timeout            string            `json:"timeout"`
	Delay              string            `json:"delay"`
	Iterations         int               `json:"iterations,omitempty"`
	Duration           string            `json:"duration,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	InsecureSkipVerify bool              `json:"insecure_skip_verify,omitempty"`
}

type rawTestCase struct {
	Name               string            `json:"name"`
	Method             string            `json:"method"`
	Path               string            `json:"path"`
	Headers            map[string]string `json:"headers,omitempty"`
	Body               interface{}       `json:"body,omitempty"`
	ExpectedStatus     []int             `json:"expected_status"`
	Timeout            string            `json:"timeout,omitempty"`
	Delay              string            `json:"delay,omitempty"`
	Iterations         int               `json:"iterations,omitempty"`
	Duration           string            `json:"duration,omitempty"`
	Assertions         []rawAssertion    `json:"assertions,omitempty"`
	InsecureSkipVerify *bool             `json:"insecure_skip_verify,omitempty"`
}

type rawAssertion struct {
	Type     string      `json:"type"`
	Target   string      `json:"target"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

func parseConfig(raw *rawConfig) (*models.Config, error) {
	globalTimeout, err := time.ParseDuration(raw.Global.Timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid global timeout: %w", err)
	}

	globalDelay, err := time.ParseDuration(raw.Global.Delay)
	if err != nil {
		return nil, fmt.Errorf("invalid global delay: %w", err)
	}

	var globalDuration time.Duration
	if raw.Global.Duration != "" {
		globalDuration, err = time.ParseDuration(raw.Global.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid global duration: %w", err)
		}
	}

	config := &models.Config{
		Name:        raw.Name,
		Description: raw.Description,
		Global: models.GlobalConfig{
			BaseURL:            raw.Global.BaseURL,
			Timeout:            globalTimeout,
			Delay:              globalDelay,
			Iterations:         raw.Global.Iterations,
			Duration:           globalDuration,
			Headers:            raw.Global.Headers,
			InsecureSkipVerify: raw.Global.InsecureSkipVerify,
		},
	}

	for i, rawTest := range raw.Tests {
		test := models.TestCase{
			Name:               rawTest.Name,
			Method:             rawTest.Method,
			Path:               rawTest.Path,
			Headers:            rawTest.Headers,
			Body:               rawTest.Body,
			ExpectedStatus:     rawTest.ExpectedStatus,
			Iterations:         rawTest.Iterations,
			InsecureSkipVerify: rawTest.InsecureSkipVerify,
		}

		if rawTest.Timeout != "" {
			timeout, err := time.ParseDuration(rawTest.Timeout)
			if err != nil {
				return nil, fmt.Errorf("invalid timeout for test %d: %w", i, err)
			}
			test.Timeout = timeout
		}

		if rawTest.Delay != "" {
			delay, err := time.ParseDuration(rawTest.Delay)
			if err != nil {
				return nil, fmt.Errorf("invalid delay for test %d: %w", i, err)
			}
			test.Delay = delay
		}

		if rawTest.Duration != "" {
			duration, err := time.ParseDuration(rawTest.Duration)
			if err != nil {
				return nil, fmt.Errorf("invalid duration for test %d: %w", i, err)
			}
			test.Duration = duration
		}

		for _, rawAssertion := range rawTest.Assertions {
			assertion := models.Assertion{
				Type:     rawAssertion.Type,
				Target:   rawAssertion.Target,
				Operator: rawAssertion.Operator,
				Value:    rawAssertion.Value,
			}
			test.Assertions = append(test.Assertions, assertion)
		}

		config.Tests = append(config.Tests, test)
	}

	return config, nil
}

func validateConfig(config *models.Config) error {
	if config.Name == "" {
		return fmt.Errorf("config name is required")
	}

	if config.Global.BaseURL == "" {
		return fmt.Errorf("global base_url is required")
	}

	// Validate that either duration or iterations is specified at global level
	if config.Global.Duration <= 0 && config.Global.Iterations <= 0 {
		return fmt.Errorf("either global duration or global iterations must be greater than 0")
	}

	// Warn if both are specified (duration takes precedence)
	if config.Global.Duration > 0 && config.Global.Iterations > 0 {
		fmt.Printf("Warning: Both global duration and iterations specified. Duration will take precedence.\n")
	}

	if len(config.Tests) == 0 {
		return fmt.Errorf("at least one test case is required")
	}

	for i, test := range config.Tests {
		if test.Name == "" {
			return fmt.Errorf("test %d: name is required", i)
		}

		if test.Method == "" {
			return fmt.Errorf("test %d: method is required", i)
		}

		if test.Path == "" {
			return fmt.Errorf("test %d: path is required", i)
		}

		if len(test.ExpectedStatus) == 0 {
			return fmt.Errorf("test %d: at least one expected status is required", i)
		}
	}

	return nil
}
