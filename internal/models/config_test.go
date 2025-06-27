package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig_GetTotalRequests_GlobalIterations(t *testing.T) {
	config := &Config{
		Global: GlobalConfig{
			Iterations: 10,
		},
		Tests: []TestCase{
			{
				Name:           "Test1",
				Method:         "GET",
				Path:           "/test1",
				ExpectedStatus: []int{200},
				// No iterations specified - should use global
			},
			{
				Name:           "Test2",
				Method:         "GET",
				Path:           "/test2",
				ExpectedStatus: []int{200},
				// No iterations specified - should use global
			},
		},
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 20, total) // 10 + 10
}

func TestConfig_GetTotalRequests_MixedIterations(t *testing.T) {
	config := &Config{
		Global: GlobalConfig{
			Iterations: 10,
		},
		Tests: []TestCase{
			{
				Name:           "Test1",
				Method:         "GET",
				Path:           "/test1",
				ExpectedStatus: []int{200},
				// Uses global iterations (10)
			},
			{
				Name:           "Test2",
				Method:         "GET",
				Path:           "/test2",
				ExpectedStatus: []int{200},
				Iterations:     5, // Override global
			},
			{
				Name:           "Test3",
				Method:         "POST",
				Path:           "/test3",
				ExpectedStatus: []int{201},
				Iterations:     15, // Override global
			},
		},
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 30, total) // 10 + 5 + 15
}

func TestConfig_GetTotalRequests_ZeroGlobalIterations(t *testing.T) {
	config := &Config{
		Global: GlobalConfig{
			Iterations: 0, // Zero global iterations
		},
		Tests: []TestCase{
			{
				Name:           "Test1",
				Method:         "GET",
				Path:           "/test1",
				ExpectedStatus: []int{200},
				// Should use 0 from global
			},
			{
				Name:           "Test2",
				Method:         "GET",
				Path:           "/test2",
				ExpectedStatus: []int{200},
				Iterations:     5, // Override with specific value
			},
		},
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 5, total) // 0 + 5
}

func TestConfig_GetTotalRequests_AllSpecificIterations(t *testing.T) {
	config := &Config{
		Global: GlobalConfig{
			Iterations: 100, // This shouldn't be used
		},
		Tests: []TestCase{
			{
				Name:           "Test1",
				Method:         "GET",
				Path:           "/test1",
				ExpectedStatus: []int{200},
				Iterations:     3,
			},
			{
				Name:           "Test2",
				Method:         "GET",
				Path:           "/test2",
				ExpectedStatus: []int{200},
				Iterations:     7,
			},
			{
				Name:           "Test3",
				Method:         "POST",
				Path:           "/test3",
				ExpectedStatus: []int{201},
				Iterations:     12,
			},
		},
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 22, total) // 3 + 7 + 12
}

func TestConfig_GetTotalRequests_NoTests(t *testing.T) {
	config := &Config{
		Global: GlobalConfig{
			Iterations: 10,
		},
		Tests: []TestCase{}, // No tests
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 0, total)
}

func TestConfig_GetTotalRequests_SingleTest(t *testing.T) {
	config := &Config{
		Global: GlobalConfig{
			Iterations: 5,
		},
		Tests: []TestCase{
			{
				Name:           "SingleTest",
				Method:         "GET",
				Path:           "/single",
				ExpectedStatus: []int{200},
				Iterations:     42,
			},
		},
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 42, total)
}

func TestConfig_GetTotalRequests_LargeNumbers(t *testing.T) {
	config := &Config{
		Global: GlobalConfig{
			Iterations: 10000,
		},
		Tests: []TestCase{
			{
				Name:           "Test1",
				Method:         "GET",
				Path:           "/test1",
				ExpectedStatus: []int{200},
				// Uses global 10000
			},
			{
				Name:           "Test2",
				Method:         "GET",
				Path:           "/test2",
				ExpectedStatus: []int{200},
				Iterations:     50000,
			},
			{
				Name:           "Test3",
				Method:         "POST",
				Path:           "/test3",
				ExpectedStatus: []int{201},
				Iterations:     25000,
			},
		},
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 85000, total) // 10000 + 50000 + 25000
}

func TestTestCase_DefaultValues(t *testing.T) {
	testCase := TestCase{
		Name:           "Test",
		Method:         "GET",
		Path:           "/test",
		ExpectedStatus: []int{200},
	}

	// Test that default values are zero values
	assert.Equal(t, time.Duration(0), testCase.Timeout)
	assert.Equal(t, time.Duration(0), testCase.Delay)
	assert.Equal(t, 0, testCase.Iterations)
	assert.Nil(t, testCase.Headers)
	assert.Nil(t, testCase.Body)
	assert.Empty(t, testCase.Assertions)
}

func TestTestCase_WithAllFields(t *testing.T) {
	headers := Headers{
		"Authorization": "Bearer token",
		"Content-Type":  "application/json",
	}

	body := map[string]interface{}{
		"name":  "John",
		"email": "john@example.com",
	}

	assertions := []Assertion{
		{
			Type:     "response_time",
			Operator: "lt",
			Value:    "1s",
		},
		{
			Type:     "json_path",
			Target:   "$.id",
			Operator: "exists",
			Value:    true,
		},
	}

	testCase := TestCase{
		Name:           "Complete Test",
		Method:         "POST",
		Path:           "/api/users",
		Headers:        headers,
		Body:           body,
		ExpectedStatus: []int{201, 202},
		Timeout:        5 * time.Second,
		Delay:          100 * time.Millisecond,
		Iterations:     10,
		Assertions:     assertions,
	}

	assert.Equal(t, "Complete Test", testCase.Name)
	assert.Equal(t, "POST", testCase.Method)
	assert.Equal(t, "/api/users", testCase.Path)
	assert.Equal(t, headers, testCase.Headers)
	assert.Equal(t, body, testCase.Body)
	assert.Equal(t, []int{201, 202}, testCase.ExpectedStatus)
	assert.Equal(t, 5*time.Second, testCase.Timeout)
	assert.Equal(t, 100*time.Millisecond, testCase.Delay)
	assert.Equal(t, 10, testCase.Iterations)
	assert.Equal(t, assertions, testCase.Assertions)
}

func TestGlobalConfig_DefaultValues(t *testing.T) {
	global := GlobalConfig{
		BaseURL:    "https://api.example.com",
		Iterations: 10,
	}

	assert.Equal(t, "https://api.example.com", global.BaseURL)
	assert.Equal(t, 10, global.Iterations)
	assert.Equal(t, time.Duration(0), global.Timeout)
	assert.Equal(t, time.Duration(0), global.Delay)
	assert.Nil(t, global.Headers)
}

func TestHeaders_Type(t *testing.T) {
	headers := Headers{
		"Authorization": "Bearer token123",
		"Accept":        "application/json",
		"User-Agent":    "Bombardino/1.0",
	}

	assert.Equal(t, "Bearer token123", headers["Authorization"])
	assert.Equal(t, "application/json", headers["Accept"])
	assert.Equal(t, "Bombardino/1.0", headers["User-Agent"])
	assert.Equal(t, 3, len(headers))
}

func TestAssertion_AllFields(t *testing.T) {
	assertion := Assertion{
		Type:     "json_path",
		Target:   "$.data[0].id",
		Operator: "eq",
		Value:    123,
	}

	assert.Equal(t, "json_path", assertion.Type)
	assert.Equal(t, "$.data[0].id", assertion.Target)
	assert.Equal(t, "eq", assertion.Operator)
	assert.Equal(t, 123, assertion.Value)
}

func TestTestResult_AllFields(t *testing.T) {
	timestamp := time.Now()
	result := TestResult{
		TestName:     "Test API",
		URL:          "https://api.example.com/test",
		Method:       "GET",
		StatusCode:   200,
		ResponseTime: 150 * time.Millisecond,
		Success:      true,
		Error:        "",
		ResponseSize: 1024,
		RequestSize:  256,
		Timestamp:    timestamp,
	}

	assert.Equal(t, "Test API", result.TestName)
	assert.Equal(t, "https://api.example.com/test", result.URL)
	assert.Equal(t, "GET", result.Method)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, 150*time.Millisecond, result.ResponseTime)
	assert.True(t, result.Success)
	assert.Equal(t, "", result.Error)
	assert.Equal(t, int64(1024), result.ResponseSize)
	assert.Equal(t, int64(256), result.RequestSize)
	assert.Equal(t, timestamp, result.Timestamp)
}

func TestSummary_AllFields(t *testing.T) {
	statusCodes := map[int]int{
		200: 85,
		201: 10,
		500: 5,
	}

	errors := map[string]int{
		"Connection timeout":          3,
		"Unexpected status code: 500": 2,
	}

	summary := Summary{
		TotalRequests:   100,
		SuccessfulReqs:  95,
		FailedReqs:      5,
		TotalTime:       10 * time.Second,
		AvgResponseTime: 120 * time.Millisecond,
		MinResponseTime: 50 * time.Millisecond,
		MaxResponseTime: 300 * time.Millisecond,
		RequestsPerSec:  10.0,
		StatusCodes:     statusCodes,
		Errors:          errors,
	}

	assert.Equal(t, 100, summary.TotalRequests)
	assert.Equal(t, 95, summary.SuccessfulReqs)
	assert.Equal(t, 5, summary.FailedReqs)
	assert.Equal(t, 10*time.Second, summary.TotalTime)
	assert.Equal(t, 120*time.Millisecond, summary.AvgResponseTime)
	assert.Equal(t, 50*time.Millisecond, summary.MinResponseTime)
	assert.Equal(t, 300*time.Millisecond, summary.MaxResponseTime)
	assert.Equal(t, 10.0, summary.RequestsPerSec)
	assert.Equal(t, statusCodes, summary.StatusCodes)
	assert.Equal(t, errors, summary.Errors)
}
