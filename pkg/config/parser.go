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
	BaseURL            string                 `json:"base_url"`
	Timeout            string                 `json:"timeout"`
	Delay              string                 `json:"delay"`
	Iterations         int                    `json:"iterations,omitempty"`
	Duration           string                 `json:"duration,omitempty"`
	Headers            map[string]string      `json:"headers,omitempty"`
	InsecureSkipVerify bool                   `json:"insecure_skip_verify,omitempty"`
	Variables          map[string]interface{} `json:"variables,omitempty"`
	ThinkTime          string                 `json:"think_time,omitempty"`
	ThinkTimeMin       string                 `json:"think_time_min,omitempty"`
	ThinkTimeMax       string                 `json:"think_time_max,omitempty"`
}

type rawTestCase struct {
	Name               string                   `json:"name"`
	Method             string                   `json:"method"`
	Path               string                   `json:"path"`
	Headers            map[string]string        `json:"headers,omitempty"`
	Body               interface{}              `json:"body,omitempty"`
	ExpectedStatus     []int                    `json:"expected_status"`
	Timeout            string                   `json:"timeout,omitempty"`
	Delay              string                   `json:"delay,omitempty"`
	Iterations         int                      `json:"iterations,omitempty"`
	Duration           string                   `json:"duration,omitempty"`
	Assertions         []rawAssertion           `json:"assertions,omitempty"`
	InsecureSkipVerify *bool                    `json:"insecure_skip_verify,omitempty"`
	Extract            []rawExtraction          `json:"extract,omitempty"`
	DependsOn          []string                 `json:"depends_on,omitempty"`
	ThinkTime          string                   `json:"think_time,omitempty"`
	ThinkTimeMin       string                   `json:"think_time_min,omitempty"`
	ThinkTimeMax       string                   `json:"think_time_max,omitempty"`
	Data               []map[string]interface{} `json:"data,omitempty"`
	DataFile           string                   `json:"data_file,omitempty"`
	CompareWith        *rawCompareConfig        `json:"compare_with,omitempty"`
}

type rawExtraction struct {
	Name   string `json:"name"`
	Source string `json:"source"`
	Path   string `json:"path"`
}

type rawAssertion struct {
	Type     string      `json:"type"`
	Target   string      `json:"target"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type rawCompareConfig struct {
	Endpoint     string                `json:"endpoint"`
	Path         string                `json:"path,omitempty"`
	Headers      map[string]string     `json:"headers,omitempty"`
	Timeout      string                `json:"timeout,omitempty"`
	Assertions   []rawCompareAssertion `json:"assertions,omitempty"`
	IgnoreFields []string              `json:"ignore_fields,omitempty"`
	Mode         string                `json:"mode,omitempty"`
}

type rawCompareAssertion struct {
	Type      string      `json:"type"`
	Target    string      `json:"target,omitempty"`
	Operator  string      `json:"operator,omitempty"`
	Tolerance interface{} `json:"tolerance,omitempty"`
}

func parseConfig(raw *rawConfig) (*models.Config, error) {
	globalTimeout := 30 * time.Second // default
	var err error
	if raw.Global.Timeout != "" {
		globalTimeout, err = time.ParseDuration(raw.Global.Timeout)
		if err != nil {
			return nil, fmt.Errorf("invalid global timeout: %w", err)
		}
	}

	var globalDelay time.Duration
	if raw.Global.Delay != "" {
		globalDelay, err = time.ParseDuration(raw.Global.Delay)
		if err != nil {
			return nil, fmt.Errorf("invalid global delay: %w", err)
		}
	}

	var globalDuration time.Duration
	if raw.Global.Duration != "" {
		globalDuration, err = time.ParseDuration(raw.Global.Duration)
		if err != nil {
			return nil, fmt.Errorf("invalid global duration: %w", err)
		}
	}

	var globalThinkTime time.Duration
	if raw.Global.ThinkTime != "" {
		globalThinkTime, err = time.ParseDuration(raw.Global.ThinkTime)
		if err != nil {
			return nil, fmt.Errorf("invalid global think_time: %w", err)
		}
	}

	var globalThinkTimeMin time.Duration
	if raw.Global.ThinkTimeMin != "" {
		globalThinkTimeMin, err = time.ParseDuration(raw.Global.ThinkTimeMin)
		if err != nil {
			return nil, fmt.Errorf("invalid global think_time_min: %w", err)
		}
	}

	var globalThinkTimeMax time.Duration
	if raw.Global.ThinkTimeMax != "" {
		globalThinkTimeMax, err = time.ParseDuration(raw.Global.ThinkTimeMax)
		if err != nil {
			return nil, fmt.Errorf("invalid global think_time_max: %w", err)
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
			Variables:          raw.Global.Variables,
			ThinkTime:          globalThinkTime,
			ThinkTimeMin:       globalThinkTimeMin,
			ThinkTimeMax:       globalThinkTimeMax,
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

		// Parse extraction rules
		for _, rawExtract := range rawTest.Extract {
			extraction := models.ExtractionRule{
				Name:   rawExtract.Name,
				Source: rawExtract.Source,
				Path:   rawExtract.Path,
			}
			test.Extract = append(test.Extract, extraction)
		}

		// Copy dependencies
		test.DependsOn = rawTest.DependsOn

		// Parse think time settings
		if rawTest.ThinkTime != "" {
			thinkTime, err := time.ParseDuration(rawTest.ThinkTime)
			if err != nil {
				return nil, fmt.Errorf("invalid think_time for test %d: %w", i, err)
			}
			test.ThinkTime = thinkTime
		}

		if rawTest.ThinkTimeMin != "" {
			thinkTimeMin, err := time.ParseDuration(rawTest.ThinkTimeMin)
			if err != nil {
				return nil, fmt.Errorf("invalid think_time_min for test %d: %w", i, err)
			}
			test.ThinkTimeMin = thinkTimeMin
		}

		if rawTest.ThinkTimeMax != "" {
			thinkTimeMax, err := time.ParseDuration(rawTest.ThinkTimeMax)
			if err != nil {
				return nil, fmt.Errorf("invalid think_time_max for test %d: %w", i, err)
			}
			test.ThinkTimeMax = thinkTimeMax
		}

		// Copy data-driven test data
		test.Data = rawTest.Data
		test.DataFile = rawTest.DataFile

		// Parse compare_with configuration
		if rawTest.CompareWith != nil {
			compareConfig := &models.CompareConfig{
				Endpoint:     rawTest.CompareWith.Endpoint,
				Path:         rawTest.CompareWith.Path,
				Headers:      rawTest.CompareWith.Headers,
				IgnoreFields: rawTest.CompareWith.IgnoreFields,
				Mode:         rawTest.CompareWith.Mode,
			}

			if rawTest.CompareWith.Timeout != "" {
				timeout, err := time.ParseDuration(rawTest.CompareWith.Timeout)
				if err != nil {
					return nil, fmt.Errorf("invalid compare_with timeout for test %d: %w", i, err)
				}
				compareConfig.Timeout = timeout
			}

			for _, rawAssertion := range rawTest.CompareWith.Assertions {
				compareConfig.Assertions = append(compareConfig.Assertions, models.CompareAssertion{
					Type:      rawAssertion.Type,
					Target:    rawAssertion.Target,
					Operator:  rawAssertion.Operator,
					Tolerance: rawAssertion.Tolerance,
				})
			}

			test.CompareWith = compareConfig
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

		// Validate compare_with configuration
		if test.CompareWith != nil {
			if test.CompareWith.Endpoint == "" {
				return fmt.Errorf("test %d: compare_with.endpoint is required when compare_with is specified", i)
			}

			for j, assertion := range test.CompareWith.Assertions {
				if assertion.Type == "" {
					return fmt.Errorf("test %d: compare_with.assertions[%d].type is required", i, j)
				}
				// Target is required for all types except structure_match and status_match
				if assertion.Target == "" && assertion.Type != "structure_match" && assertion.Type != "status_match" {
					return fmt.Errorf("test %d: compare_with.assertions[%d].target is required for type %s", i, j, assertion.Type)
				}
			}
		}
	}

	return nil
}
