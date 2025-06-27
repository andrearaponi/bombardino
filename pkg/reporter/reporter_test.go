package reporter

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestReporter_New(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
	}{
		{"verbose reporter", true},
		{"non-verbose reporter", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := New(tt.verbose)
			assert.NotNil(t, reporter)
			assert.Equal(t, tt.verbose, reporter.verbose)
		})
	}
}

func TestReporter_GenerateReport_BasicSummary(t *testing.T) {
	summary := &models.Summary{
		TotalRequests:   100,
		SuccessfulReqs:  95,
		FailedReqs:      5,
		TotalTime:       10 * time.Second,
		AvgResponseTime: 150 * time.Millisecond,
		MinResponseTime: 50 * time.Millisecond,
		MaxResponseTime: 500 * time.Millisecond,
		RequestsPerSec:  10.0,
		StatusCodes: map[int]int{
			200: 90,
			201: 5,
			500: 5,
		},
		Errors: map[string]int{
			"Unexpected status code: 500": 5,
		},
	}

	output := captureOutput(func() {
		reporter := New(false)
		reporter.GenerateReport(summary)
	})

	// Check for header
	assert.Contains(t, output, "BOMBARDINO RESULTS")

	// Check for summary section
	assert.Contains(t, output, "📊 SUMMARY")
	assert.Contains(t, output, "Total Requests:      100")
	assert.Contains(t, output, "Successful:          95 (95.0%)")
	assert.Contains(t, output, "Failed:              5 (5.0%)")
	assert.Contains(t, output, "Requests/sec:        10.00")

	// Check for response times
	assert.Contains(t, output, "⏱️  RESPONSE TIMES")
	assert.Contains(t, output, "Average:             150ms")
	assert.Contains(t, output, "Minimum:             50ms")
	assert.Contains(t, output, "Maximum:             500ms")

	// Check for status codes
	assert.Contains(t, output, "📈 STATUS CODES")
	assert.Contains(t, output, "✅ 200:")
	assert.Contains(t, output, "✅ 201:")
	assert.Contains(t, output, "❌ 500:")

	// Check for errors
	assert.Contains(t, output, "❌ ERRORS")
	assert.Contains(t, output, "Unexpected status code: 500")

	// Check for footer
	assert.Contains(t, output, "🚀 Test completed successfully!")
}

func TestReporter_GenerateReport_NoErrors(t *testing.T) {
	summary := &models.Summary{
		TotalRequests:   50,
		SuccessfulReqs:  50,
		FailedReqs:      0,
		TotalTime:       5 * time.Second,
		AvgResponseTime: 100 * time.Millisecond,
		MinResponseTime: 80 * time.Millisecond,
		MaxResponseTime: 120 * time.Millisecond,
		RequestsPerSec:  10.0,
		StatusCodes: map[int]int{
			200: 50,
		},
		Errors: map[string]int{}, // No errors
	}

	output := captureOutput(func() {
		reporter := New(false)
		reporter.GenerateReport(summary)
	})

	// Should contain summary and status codes but not errors section
	assert.Contains(t, output, "📊 SUMMARY")
	assert.Contains(t, output, "📈 STATUS CODES")
	assert.NotContains(t, output, "❌ ERRORS")
	assert.Contains(t, output, "Successful:          50 (100.0%)")
	assert.Contains(t, output, "Failed:              0 (0.0%)")
}

func TestReporter_GenerateReport_VerboseMode(t *testing.T) {
	summary := &models.Summary{
		TotalRequests:   10,
		SuccessfulReqs:  8,
		FailedReqs:      2,
		TotalTime:       2 * time.Second,
		AvgResponseTime: 200 * time.Millisecond,
		MinResponseTime: 100 * time.Millisecond,
		MaxResponseTime: 300 * time.Millisecond,
		RequestsPerSec:  5.0,
		StatusCodes: map[int]int{
			200: 8,
			404: 2,
		},
		Errors: map[string]int{
			"Unexpected status code: 404": 2,
		},
	}

	verboseOutput := captureOutput(func() {
		reporter := New(true)
		reporter.GenerateReport(summary)
	})

	nonVerboseOutput := captureOutput(func() {
		reporter := New(false)
		reporter.GenerateReport(summary)
	})

	// Both should contain the same basic information
	assert.Contains(t, verboseOutput, "📊 SUMMARY")
	assert.Contains(t, nonVerboseOutput, "📊 SUMMARY")

	// For now, verbose mode doesn't add extra content, but structure is ready
	assert.Equal(t, len(strings.Split(verboseOutput, "\n")), len(strings.Split(nonVerboseOutput, "\n")))
}

func TestReporter_getStatusEmoji(t *testing.T) {
	reporter := &Reporter{}

	tests := []struct {
		statusCode int
		expected   string
	}{
		{200, "✅"},
		{201, "✅"},
		{299, "✅"},
		{300, "🔄"},
		{301, "🔄"},
		{399, "🔄"},
		{400, "⚠️"},
		{404, "⚠️"},
		{499, "⚠️"},
		{500, "❌"},
		{503, "❌"},
		{599, "❌"},
		{100, "❓"}, // 1xx status codes
		{600, "❓"}, // Invalid status codes
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.statusCode), func(t *testing.T) {
			result := reporter.getStatusEmoji(tt.statusCode)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReporter_printStatusCodes_Sorting(t *testing.T) {
	summary := &models.Summary{
		TotalRequests: 100,
		StatusCodes: map[int]int{
			500: 10,
			200: 70,
			404: 15,
			201: 5,
		},
	}

	output := captureOutput(func() {
		reporter := New(false)
		reporter.printStatusCodes(summary)
	})

	lines := strings.Split(output, "\n")
	var statusLines []string
	for _, line := range lines {
		if strings.Contains(line, ":") && (strings.Contains(line, "✅") || strings.Contains(line, "⚠️") || strings.Contains(line, "❌")) {
			statusLines = append(statusLines, line)
		}
	}

	// Should be sorted by status code (200, 201, 404, 500)
	assert.True(t, len(statusLines) >= 4)
	assert.Contains(t, statusLines[0], "200")
	assert.Contains(t, statusLines[1], "201")
	assert.Contains(t, statusLines[2], "404")
	assert.Contains(t, statusLines[3], "500")
}

func TestReporter_printErrors_Sorting(t *testing.T) {
	summary := &models.Summary{
		TotalRequests: 100,
		Errors: map[string]int{
			"Connection timeout":          5,
			"Unexpected status code: 500": 15,
			"DNS resolution failed":       3,
			"Connection refused":          10,
		},
	}

	output := captureOutput(func() {
		reporter := New(false)
		reporter.printErrors(summary)
	})

	lines := strings.Split(output, "\n")
	var errorLines []string
	for _, line := range lines {
		if strings.Contains(line, "•") {
			errorLines = append(errorLines, line)
		}
	}

	// Should be sorted by count (descending): 15, 10, 5, 3
	assert.True(t, len(errorLines) >= 4)
	assert.Contains(t, errorLines[0], "Unexpected status code: 500")
	assert.Contains(t, errorLines[1], "Connection refused")
	assert.Contains(t, errorLines[2], "Connection timeout")
	assert.Contains(t, errorLines[3], "DNS resolution failed")
}

func TestReporter_GenerateReport_EmptyStatusCodes(t *testing.T) {
	summary := &models.Summary{
		TotalRequests:   0,
		SuccessfulReqs:  0,
		FailedReqs:      0,
		TotalTime:       0,
		AvgResponseTime: 0,
		MinResponseTime: 0,
		MaxResponseTime: 0,
		RequestsPerSec:  0,
		StatusCodes:     map[int]int{}, // Empty
		Errors:          map[string]int{},
	}

	output := captureOutput(func() {
		reporter := New(false)
		reporter.GenerateReport(summary)
	})

	// Should contain header and summary but no status codes section
	assert.Contains(t, output, "📊 SUMMARY")
	assert.NotContains(t, output, "📈 STATUS CODES")
	assert.NotContains(t, output, "❌ ERRORS")
}

func TestReporter_GenerateReport_LargeNumbers(t *testing.T) {
	summary := &models.Summary{
		TotalRequests:   1000000,
		SuccessfulReqs:  999500,
		FailedReqs:      500,
		TotalTime:       1 * time.Hour,
		AvgResponseTime: 50 * time.Millisecond,
		MinResponseTime: 10 * time.Millisecond,
		MaxResponseTime: 2 * time.Second,
		RequestsPerSec:  277.78,
		StatusCodes: map[int]int{
			200: 999500,
			500: 500,
		},
		Errors: map[string]int{
			"Unexpected status code: 500": 500,
		},
	}

	output := captureOutput(func() {
		reporter := New(false)
		reporter.GenerateReport(summary)
	})

	assert.Contains(t, output, "Total Requests:      1000000")
	assert.Contains(t, output, "Successful:          999500 (100.0%)")
	assert.Contains(t, output, "Failed:              500 (0.0%)")
	assert.Contains(t, output, "Requests/sec:        277.78")
}

// Helper function to capture stdout output
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
