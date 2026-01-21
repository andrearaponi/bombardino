package models

import (
	"encoding/json"
	"time"
)

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
	CompareWith        *CompareConfig           `json:"compare_with,omitempty"`
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

// CompareConfig defines configuration for tap compare feature
type CompareConfig struct {
	Endpoint     string              `json:"endpoint"`
	Path         string              `json:"path,omitempty"`
	Headers      map[string]string   `json:"headers,omitempty"`
	Timeout      time.Duration       `json:"timeout,omitempty"`
	Assertions   []CompareAssertion  `json:"assertions,omitempty"`
	IgnoreFields []string            `json:"ignore_fields,omitempty"`
	Mode         string              `json:"mode,omitempty"` // "full", "partial", "structural"
}

// CompareAssertion defines how to compare specific fields between responses
type CompareAssertion struct {
	Type      string      `json:"type"`               // "field_match", "field_tolerance", "structure_match", "status_match", "response_time_tolerance"
	Target    string      `json:"target,omitempty"`   // JSON path to compare
	Operator  string      `json:"operator,omitempty"` // "eq", "contains", "matches"
	Tolerance interface{} `json:"tolerance,omitempty"` // For numeric tolerance (percentage or absolute)
}

// ComparisonResult holds the outcome of a tap compare operation
type ComparisonResult struct {
	Success          bool                     `json:"success"`
	PrimaryResponse  ResponseData             `json:"primary_response"`
	CompareResponse  ResponseData             `json:"compare_response"`
	FieldDiffs       []FieldDiff              `json:"field_diffs,omitempty"`
	AssertionResults []CompareAssertionResult `json:"assertion_results,omitempty"`
	Error            string                   `json:"error,omitempty"`
}

// ResponseData captures relevant response information for comparison
type ResponseData struct {
	StatusCode   int             `json:"status_code"`
	ResponseTime time.Duration   `json:"response_time"`
	BodySize     int64           `json:"body_size"`
	Body         json.RawMessage `json:"body,omitempty"`
}

// FieldDiff represents a difference between two fields
type FieldDiff struct {
	Path         string      `json:"path"`
	PrimaryValue interface{} `json:"primary_value"`
	CompareValue interface{} `json:"compare_value"`
	Type         string      `json:"type"` // "missing", "extra", "type_mismatch", "value_mismatch"
	Message      string      `json:"message"`
}

// CompareAssertionResult holds the outcome of a single comparison assertion
type CompareAssertionResult struct {
	Assertion    CompareAssertion `json:"assertion"`
	Passed       bool             `json:"passed"`
	PrimaryValue interface{}      `json:"primary_value,omitempty"`
	CompareValue interface{}      `json:"compare_value,omitempty"`
	Message      string           `json:"message,omitempty"`
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
	ComparisonResult *ComparisonResult
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
	TotalComparisons   int
	ComparisonsPassed  int
	ComparisonsFailed  int
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
	Name              string
	URL               string
	TotalRequests     int
	SuccessfulReqs    int
	FailedReqs        int
	SkippedReqs       int
	AvgResponseTime   time.Duration
	P50ResponseTime   time.Duration
	P95ResponseTime   time.Duration
	P99ResponseTime   time.Duration
	StatusCodes       map[int]int
	Errors            []string
	TotalAssertions   int
	AssertionsPassed  int
	AssertionsFailed  int
	FirstExecutedAt   time.Time // Track execution order
	TotalComparisons  int
	ComparisonsPassed int
	ComparisonsFailed int
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
