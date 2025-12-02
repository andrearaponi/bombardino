package assertion

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/tidwall/gjson"
)

// Context holds all the information needed to evaluate assertions
type Context struct {
	StatusCode   int
	ResponseTime time.Duration
	Body         []byte
	Headers      http.Header
}

// NewContext creates a new assertion context
func NewContext(statusCode int, responseTime time.Duration, body []byte, headers http.Header) *Context {
	return &Context{
		StatusCode:   statusCode,
		ResponseTime: responseTime,
		Body:         body,
		Headers:      headers,
	}
}

// Result holds the outcome of an assertion evaluation
type Result struct {
	Assertion   models.Assertion
	Passed      bool
	ActualValue interface{}
	Message     string
}

// Evaluator evaluates assertions against response data
type Evaluator struct {
	verbose bool
}

// New creates a new assertion evaluator
func New(verbose bool) *Evaluator {
	return &Evaluator{
		verbose: verbose,
	}
}

// EvaluateAll evaluates all assertions and returns all results
func (e *Evaluator) EvaluateAll(assertions []models.Assertion, ctx *Context) []Result {
	results := make([]Result, 0, len(assertions))
	for _, assertion := range assertions {
		results = append(results, e.Evaluate(assertion, ctx))
	}
	return results
}

// Evaluate evaluates a single assertion against the context
func (e *Evaluator) Evaluate(assertion models.Assertion, ctx *Context) Result {
	result := Result{
		Assertion: assertion,
		Passed:    false,
	}

	switch assertion.Type {
	case "json_path":
		return e.evaluateJSONPath(assertion, ctx)
	case "response_time":
		return e.evaluateResponseTime(assertion, ctx)
	case "status":
		return e.evaluateStatus(assertion, ctx)
	case "header":
		return e.evaluateHeader(assertion, ctx)
	case "body_size":
		return e.evaluateBodySize(assertion, ctx)
	default:
		result.Message = fmt.Sprintf("unknown assertion type: %s", assertion.Type)
		return result
	}
}

// evaluateJSONPath evaluates a JSON path assertion
func (e *Evaluator) evaluateJSONPath(assertion models.Assertion, ctx *Context) Result {
	result := Result{
		Assertion: assertion,
		Passed:    false,
	}

	if len(ctx.Body) == 0 {
		result.Message = "empty response body"
		return result
	}

	// Check if JSON is valid
	if !gjson.ValidBytes(ctx.Body) {
		result.Message = "invalid JSON in response body"
		return result
	}

	// Handle exists/not_exists operators
	if assertion.Operator == "exists" || assertion.Operator == "not_exists" {
		exists := gjson.GetBytes(ctx.Body, assertion.Target).Exists()
		result.ActualValue = exists

		if assertion.Operator == "exists" {
			result.Passed = exists
			if !exists {
				result.Message = fmt.Sprintf("path '%s' not found in response", assertion.Target)
			}
		} else {
			result.Passed = !exists
			if exists {
				result.Message = fmt.Sprintf("path '%s' exists but should not", assertion.Target)
			}
		}
		return result
	}

	// Get the value at the path
	value := gjson.GetBytes(ctx.Body, assertion.Target)
	if !value.Exists() {
		result.Message = fmt.Sprintf("path '%s' not found in response", assertion.Target)
		return result
	}

	// Extract the actual value
	var actualValue interface{}
	switch value.Type {
	case gjson.String:
		actualValue = value.String()
	case gjson.Number:
		actualValue = value.Float()
	case gjson.True:
		actualValue = true
	case gjson.False:
		actualValue = false
	case gjson.Null:
		actualValue = nil
	default:
		actualValue = value.Raw
	}
	result.ActualValue = actualValue

	// Compare values
	passed, err := e.compare(assertion.Operator, actualValue, assertion.Value)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	result.Passed = passed
	if !passed {
		result.Message = fmt.Sprintf("assertion failed: %s %s %v, got %v",
			assertion.Target, assertion.Operator, assertion.Value, actualValue)
	}

	return result
}

// evaluateResponseTime evaluates a response time assertion
func (e *Evaluator) evaluateResponseTime(assertion models.Assertion, ctx *Context) Result {
	result := Result{
		Assertion:   assertion,
		ActualValue: ctx.ResponseTime,
		Passed:      false,
	}

	// Parse expected duration from value
	valueStr, ok := assertion.Value.(string)
	if !ok {
		result.Message = fmt.Sprintf("invalid duration value: %v (expected string like '100ms')", assertion.Value)
		return result
	}

	expected, err := time.ParseDuration(valueStr)
	if err != nil {
		result.Message = fmt.Sprintf("invalid duration format: %v", err)
		return result
	}

	// Compare durations
	passed, err := e.compareDurations(assertion.Operator, ctx.ResponseTime, expected)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	result.Passed = passed
	if !passed {
		result.Message = fmt.Sprintf("response time assertion failed: %v %s %v",
			ctx.ResponseTime, assertion.Operator, expected)
	}

	return result
}

// evaluateStatus evaluates a status code assertion
func (e *Evaluator) evaluateStatus(assertion models.Assertion, ctx *Context) Result {
	result := Result{
		Assertion:   assertion,
		ActualValue: ctx.StatusCode,
		Passed:      false,
	}

	// Convert expected value to float64 (JSON numbers are float64)
	expected, ok := assertion.Value.(float64)
	if !ok {
		result.Message = fmt.Sprintf("invalid status code value: %v", assertion.Value)
		return result
	}

	passed, err := e.compare(assertion.Operator, float64(ctx.StatusCode), expected)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	result.Passed = passed
	if !passed {
		result.Message = fmt.Sprintf("status assertion failed: %d %s %v",
			ctx.StatusCode, assertion.Operator, int(expected))
	}

	return result
}

// evaluateHeader evaluates a header assertion
func (e *Evaluator) evaluateHeader(assertion models.Assertion, ctx *Context) Result {
	result := Result{
		Assertion: assertion,
		Passed:    false,
	}

	if ctx.Headers == nil {
		result.Message = "no headers in response"
		return result
	}

	// Get header value (case-insensitive)
	headerValue := ctx.Headers.Get(assertion.Target)
	result.ActualValue = headerValue

	// Handle exists/not_exists operators
	if assertion.Operator == "exists" || assertion.Operator == "not_exists" {
		exists := headerValue != ""

		if assertion.Operator == "exists" {
			result.Passed = exists
			if !exists {
				result.Message = fmt.Sprintf("header '%s' not found", assertion.Target)
			}
		} else {
			result.Passed = !exists
			if exists {
				result.Message = fmt.Sprintf("header '%s' exists but should not", assertion.Target)
			}
		}
		return result
	}

	if headerValue == "" {
		result.Message = fmt.Sprintf("header '%s' not found", assertion.Target)
		return result
	}

	// Compare values
	passed, err := e.compare(assertion.Operator, headerValue, assertion.Value)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	result.Passed = passed
	if !passed {
		result.Message = fmt.Sprintf("header assertion failed: %s %s %v, got '%s'",
			assertion.Target, assertion.Operator, assertion.Value, headerValue)
	}

	return result
}

// evaluateBodySize evaluates a body size assertion
func (e *Evaluator) evaluateBodySize(assertion models.Assertion, ctx *Context) Result {
	result := Result{
		Assertion:   assertion,
		ActualValue: len(ctx.Body),
		Passed:      false,
	}

	expected, ok := assertion.Value.(float64)
	if !ok {
		result.Message = fmt.Sprintf("invalid body size value: %v", assertion.Value)
		return result
	}

	passed, err := e.compare(assertion.Operator, float64(len(ctx.Body)), expected)
	if err != nil {
		result.Message = err.Error()
		return result
	}

	result.Passed = passed
	if !passed {
		result.Message = fmt.Sprintf("body size assertion failed: %d %s %v",
			len(ctx.Body), assertion.Operator, int(expected))
	}

	return result
}

// compare compares two values using the specified operator
func (e *Evaluator) compare(operator string, actual, expected interface{}) (bool, error) {
	switch operator {
	case "eq":
		return e.equals(actual, expected), nil
	case "neq":
		return !e.equals(actual, expected), nil
	case "gt":
		return e.greaterThan(actual, expected)
	case "gte":
		return e.greaterThanOrEqual(actual, expected)
	case "lt":
		return e.lessThan(actual, expected)
	case "lte":
		return e.lessThanOrEqual(actual, expected)
	case "contains":
		return e.contains(actual, expected)
	case "starts_with":
		return e.startsWith(actual, expected)
	case "ends_with":
		return e.endsWith(actual, expected)
	case "matches":
		return e.matches(actual, expected)
	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// compareDurations compares two durations using the specified operator
func (e *Evaluator) compareDurations(operator string, actual, expected time.Duration) (bool, error) {
	switch operator {
	case "eq":
		return actual == expected, nil
	case "neq":
		return actual != expected, nil
	case "gt":
		return actual > expected, nil
	case "gte":
		return actual >= expected, nil
	case "lt":
		return actual < expected, nil
	case "lte":
		return actual <= expected, nil
	default:
		return false, fmt.Errorf("unknown operator for duration: %s", operator)
	}
}

// equals checks if two values are equal
func (e *Evaluator) equals(actual, expected interface{}) bool {
	// Handle numeric comparison
	if actualFloat, ok := toFloat64(actual); ok {
		if expectedFloat, ok := toFloat64(expected); ok {
			return actualFloat == expectedFloat
		}
	}

	// Handle boolean comparison
	if actualBool, ok := actual.(bool); ok {
		if expectedBool, ok := expected.(bool); ok {
			return actualBool == expectedBool
		}
	}

	// Default string comparison
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

// greaterThan checks if actual > expected
func (e *Evaluator) greaterThan(actual, expected interface{}) (bool, error) {
	actualFloat, ok1 := toFloat64(actual)
	expectedFloat, ok2 := toFloat64(expected)
	if !ok1 || !ok2 {
		return false, fmt.Errorf("cannot compare non-numeric values: %v, %v", actual, expected)
	}
	return actualFloat > expectedFloat, nil
}

// greaterThanOrEqual checks if actual >= expected
func (e *Evaluator) greaterThanOrEqual(actual, expected interface{}) (bool, error) {
	actualFloat, ok1 := toFloat64(actual)
	expectedFloat, ok2 := toFloat64(expected)
	if !ok1 || !ok2 {
		return false, fmt.Errorf("cannot compare non-numeric values: %v, %v", actual, expected)
	}
	return actualFloat >= expectedFloat, nil
}

// lessThan checks if actual < expected
func (e *Evaluator) lessThan(actual, expected interface{}) (bool, error) {
	actualFloat, ok1 := toFloat64(actual)
	expectedFloat, ok2 := toFloat64(expected)
	if !ok1 || !ok2 {
		return false, fmt.Errorf("cannot compare non-numeric values: %v, %v", actual, expected)
	}
	return actualFloat < expectedFloat, nil
}

// lessThanOrEqual checks if actual <= expected
func (e *Evaluator) lessThanOrEqual(actual, expected interface{}) (bool, error) {
	actualFloat, ok1 := toFloat64(actual)
	expectedFloat, ok2 := toFloat64(expected)
	if !ok1 || !ok2 {
		return false, fmt.Errorf("cannot compare non-numeric values: %v, %v", actual, expected)
	}
	return actualFloat <= expectedFloat, nil
}

// contains checks if actual contains expected (string)
func (e *Evaluator) contains(actual, expected interface{}) (bool, error) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return strings.Contains(actualStr, expectedStr), nil
}

// startsWith checks if actual starts with expected (string)
func (e *Evaluator) startsWith(actual, expected interface{}) (bool, error) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return strings.HasPrefix(actualStr, expectedStr), nil
}

// endsWith checks if actual ends with expected (string)
func (e *Evaluator) endsWith(actual, expected interface{}) (bool, error) {
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	return strings.HasSuffix(actualStr, expectedStr), nil
}

// matches checks if actual matches expected regex pattern
func (e *Evaluator) matches(actual, expected interface{}) (bool, error) {
	actualStr := fmt.Sprintf("%v", actual)
	patternStr := fmt.Sprintf("%v", expected)

	re, err := regexp.Compile(patternStr)
	if err != nil {
		return false, fmt.Errorf("invalid regex pattern: %v", err)
	}

	return re.MatchString(actualStr), nil
}

// toFloat64 attempts to convert a value to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case int32:
		return float64(val), true
	default:
		return 0, false
	}
}
