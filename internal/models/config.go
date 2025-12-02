package models

import "time"

type Config struct {
	Name        string       `json:"name"`
	Description string       `json:"description,omitempty"`
	Global      GlobalConfig `json:"global"`
	Tests       []TestCase   `json:"tests"`
}

type GlobalConfig struct {
	BaseURL            string                 `json:"base_url"`
	Timeout            time.Duration          `json:"timeout"`
	Delay              time.Duration          `json:"delay"`
	Iterations         int                    `json:"iterations,omitempty"`
	Duration           time.Duration          `json:"duration,omitempty"`
	Headers            Headers                `json:"headers,omitempty"`
	InsecureSkipVerify bool                   `json:"insecure_skip_verify,omitempty"`
	Variables          map[string]interface{} `json:"variables,omitempty"`
	ThinkTime          time.Duration          `json:"think_time,omitempty"`
	ThinkTimeMin       time.Duration          `json:"think_time_min,omitempty"`
	ThinkTimeMax       time.Duration          `json:"think_time_max,omitempty"`
}

type TestCase struct {
	Name               string                   `json:"name"`
	Method             string                   `json:"method"`
	Path               string                   `json:"path"`
	Headers            Headers                  `json:"headers,omitempty"`
	Body               interface{}              `json:"body,omitempty"`
	ExpectedStatus     []int                    `json:"expected_status"`
	Timeout            time.Duration            `json:"timeout,omitempty"`
	Delay              time.Duration            `json:"delay,omitempty"`
	Iterations         int                      `json:"iterations,omitempty"`
	Duration           time.Duration            `json:"duration,omitempty"`
	Assertions         []Assertion              `json:"assertions,omitempty"`
	InsecureSkipVerify *bool                    `json:"insecure_skip_verify,omitempty"`
	Extract            []ExtractionRule         `json:"extract,omitempty"`
	DependsOn          []string                 `json:"depends_on,omitempty"`
	ThinkTime          time.Duration            `json:"think_time,omitempty"`
	ThinkTimeMin       time.Duration            `json:"think_time_min,omitempty"`
	ThinkTimeMax       time.Duration            `json:"think_time_max,omitempty"`
	Data               []map[string]interface{} `json:"data,omitempty"`
	DataFile           string                   `json:"data_file,omitempty"`
}

// ExtractionRule defines how to extract a variable from a response
type ExtractionRule struct {
	Name   string `json:"name"`   // Variable name to store
	Source string `json:"source"` // "body", "header", "status"
	Path   string `json:"path"`   // JSON path for body, header name for header
}

type Headers map[string]string

type Assertion struct {
	Type     string      `json:"type"`
	Target   string      `json:"target"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type TestResult struct {
	TestName         string
	URL              string
	Method           string
	StatusCode       int
	ResponseTime     time.Duration
	Success          bool
	Error            string
	ResponseSize     int64
	RequestSize      int64
	Timestamp        time.Time
	AssertionsPassed int
	AssertionsFailed int
	AssertionErrors  []string
	Skipped          bool
	SkipReason       string
}

type Summary struct {
	TotalRequests      int
	SuccessfulReqs     int
	FailedReqs         int
	SkippedReqs        int
	TotalTime          time.Duration
	AvgResponseTime    time.Duration
	MinResponseTime    time.Duration
	MaxResponseTime    time.Duration
	P50ResponseTime    time.Duration
	P95ResponseTime    time.Duration
	P99ResponseTime    time.Duration
	RequestsPerSec     float64
	StatusCodes        map[int]int
	Errors             map[string]int
	EndpointResults    map[string]*EndpointSummary
	DebugLogs          []DebugLog // Added for verbose mode
	TotalAssertions    int
	AssertionsPassed   int
	AssertionsFailed   int
}

type DebugLog struct {
	Timestamp   time.Time         `json:"timestamp"`
	RequestID   string            `json:"request_id,omitempty"`
	Type        string            `json:"type"` // "request" or "response"
	TestName    string            `json:"test_name"`
	Method      string            `json:"method,omitempty"`
	URL         string            `json:"url,omitempty"`
	StatusCode  int               `json:"status_code,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Body        string            `json:"body,omitempty"`
	ResponseTime time.Duration    `json:"response_time,omitempty"`
	Error       string            `json:"error,omitempty"`
}

type EndpointSummary struct {
	Name             string
	URL              string
	TotalRequests    int
	SuccessfulReqs   int
	FailedReqs       int
	SkippedReqs      int
	AvgResponseTime  time.Duration
	P50ResponseTime  time.Duration
	P95ResponseTime  time.Duration
	P99ResponseTime  time.Duration
	StatusCodes      map[int]int
	Errors           []string
	TotalAssertions  int
	AssertionsPassed int
	AssertionsFailed int
	FirstExecutedAt  time.Time // Track execution order
}

func (c *Config) GetTotalRequests() int {
	// For duration-based tests, we can't know the exact number in advance
	// Return estimated number for progress bar (can be adjusted during execution)
	if c.Global.Duration > 0 {
		// Rough estimate: assume 1 request per second per test
		estimatedRPS := len(c.Tests)
		return int(c.Global.Duration.Seconds()) * estimatedRPS
	}

	total := 0
	for _, test := range c.Tests {
		if test.Duration > 0 {
			// Duration-based test: estimate requests
			total += int(test.Duration.Seconds())
		} else {
			// Iteration-based test
			iterations := test.Iterations
			if iterations == 0 {
				iterations = c.Global.Iterations
			}
			total += iterations
		}
	}
	return total
}

func (c *Config) IsDurationBased() bool {
	return c.Global.Duration > 0
}

func (c *Config) HasMixedMode() bool {
	hasDuration := c.Global.Duration > 0
	hasIterations := c.Global.Iterations > 0

	for _, test := range c.Tests {
		if test.Duration > 0 {
			hasDuration = true
		}
		if test.Iterations > 0 {
			hasIterations = true
		}
	}

	return hasDuration && hasIterations
}
