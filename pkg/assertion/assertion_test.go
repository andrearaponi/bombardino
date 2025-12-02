package assertion

import (
	"net/http"
	"testing"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Context Tests
// =============================================================================

func TestNewContext(t *testing.T) {
	ctx := NewContext(
		200,
		100*time.Millisecond,
		[]byte(`{"id": 1, "name": "test"}`),
		http.Header{"Content-Type": []string{"application/json"}},
	)

	assert.Equal(t, 200, ctx.StatusCode)
	assert.Equal(t, 100*time.Millisecond, ctx.ResponseTime)
	assert.Equal(t, []byte(`{"id": 1, "name": "test"}`), ctx.Body)
	assert.Equal(t, "application/json", ctx.Headers.Get("Content-Type"))
}

// =============================================================================
// Evaluator Tests
// =============================================================================

func TestNewEvaluator(t *testing.T) {
	e := New(true)
	assert.NotNil(t, e)
}

// =============================================================================
// JSON Path Assertion Tests
// =============================================================================

func TestJSONPathAssertion_SimpleField(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"id": 42, "name": "test"}`), nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "eq operator matches integer",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "id",
				Operator: "eq",
				Value:    float64(42), // JSON numbers are float64
			},
			wantPass: true,
		},
		{
			name: "eq operator matches string",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "name",
				Operator: "eq",
				Value:    "test",
			},
			wantPass: true,
		},
		{
			name: "eq operator fails on mismatch",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "id",
				Operator: "eq",
				Value:    float64(99),
			},
			wantPass: false,
		},
		{
			name: "neq operator passes on mismatch",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "id",
				Operator: "neq",
				Value:    float64(99),
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

func TestJSONPathAssertion_NestedField(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{
		"user": {
			"profile": {
				"email": "test@example.com",
				"age": 30
			}
		}
	}`), nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "nested path with dot notation",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "user.profile.email",
				Operator: "eq",
				Value:    "test@example.com",
			},
			wantPass: true,
		},
		{
			name: "nested numeric value",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "user.profile.age",
				Operator: "eq",
				Value:    float64(30),
			},
			wantPass: true,
		},
		{
			name: "contains operator on nested string",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "user.profile.email",
				Operator: "contains",
				Value:    "@example",
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

func TestJSONPathAssertion_ArrayAccess(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{
		"items": [
			{"id": 1, "name": "first"},
			{"id": 2, "name": "second"},
			{"id": 3, "name": "third"}
		]
	}`), nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "array index access",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "items.0.name",
				Operator: "eq",
				Value:    "first",
			},
			wantPass: true,
		},
		{
			name: "array length check",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "items.#",
				Operator: "eq",
				Value:    float64(3),
			},
			wantPass: true,
		},
		{
			name: "last item access",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "items.2.id",
				Operator: "eq",
				Value:    float64(3),
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

func TestJSONPathAssertion_NonExistentPath(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"id": 1}`), nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "json_path",
		Target:   "nonexistent.field",
		Operator: "eq",
		Value:    "anything",
	}, ctx)

	assert.False(t, result.Passed)
	assert.Contains(t, result.Message, "not found")
}

func TestJSONPathAssertion_ExistsOperator(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"id": 1, "name": null}`), nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "exists on present field",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "id",
				Operator: "exists",
				Value:    true,
			},
			wantPass: true,
		},
		{
			name: "exists on missing field",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "missing",
				Operator: "exists",
				Value:    true,
			},
			wantPass: false,
		},
		{
			name: "not_exists on missing field",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "missing",
				Operator: "not_exists",
				Value:    true,
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

// =============================================================================
// Response Time Assertion Tests
// =============================================================================

func TestResponseTimeAssertion(t *testing.T) {
	ctx := NewContext(200, 150*time.Millisecond, nil, nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "lt passes when response is faster",
			assertion: models.Assertion{
				Type:     "response_time",
				Operator: "lt",
				Value:    "200ms",
			},
			wantPass: true,
		},
		{
			name: "lt fails when response is slower",
			assertion: models.Assertion{
				Type:     "response_time",
				Operator: "lt",
				Value:    "100ms",
			},
			wantPass: false,
		},
		{
			name: "lte passes on equal",
			assertion: models.Assertion{
				Type:     "response_time",
				Operator: "lte",
				Value:    "150ms",
			},
			wantPass: true,
		},
		{
			name: "gt passes when response is slower",
			assertion: models.Assertion{
				Type:     "response_time",
				Operator: "gt",
				Value:    "100ms",
			},
			wantPass: true,
		},
		{
			name: "gte passes on equal",
			assertion: models.Assertion{
				Type:     "response_time",
				Operator: "gte",
				Value:    "150ms",
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

func TestResponseTimeAssertion_InvalidDuration(t *testing.T) {
	ctx := NewContext(200, 150*time.Millisecond, nil, nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "response_time",
		Operator: "lt",
		Value:    "invalid",
	}, ctx)

	assert.False(t, result.Passed)
	assert.Contains(t, result.Message, "invalid duration")
}

// =============================================================================
// Status Code Assertion Tests
// =============================================================================

func TestStatusAssertion(t *testing.T) {
	ctx := NewContext(201, 100*time.Millisecond, nil, nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "eq matches status code",
			assertion: models.Assertion{
				Type:     "status",
				Operator: "eq",
				Value:    float64(201),
			},
			wantPass: true,
		},
		{
			name: "neq passes on different status",
			assertion: models.Assertion{
				Type:     "status",
				Operator: "neq",
				Value:    float64(200),
			},
			wantPass: true,
		},
		{
			name: "gte for 2xx range",
			assertion: models.Assertion{
				Type:     "status",
				Operator: "gte",
				Value:    float64(200),
			},
			wantPass: true,
		},
		{
			name: "lt for success range",
			assertion: models.Assertion{
				Type:     "status",
				Operator: "lt",
				Value:    float64(300),
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

// =============================================================================
// Header Assertion Tests
// =============================================================================

func TestHeaderAssertion(t *testing.T) {
	headers := http.Header{
		"Content-Type":  []string{"application/json; charset=utf-8"},
		"X-Custom":      []string{"custom-value"},
		"Cache-Control": []string{"no-cache"},
	}
	ctx := NewContext(200, 100*time.Millisecond, nil, headers)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "contains matches partial header value",
			assertion: models.Assertion{
				Type:     "header",
				Target:   "Content-Type",
				Operator: "contains",
				Value:    "application/json",
			},
			wantPass: true,
		},
		{
			name: "eq matches exact header value",
			assertion: models.Assertion{
				Type:     "header",
				Target:   "X-Custom",
				Operator: "eq",
				Value:    "custom-value",
			},
			wantPass: true,
		},
		{
			name: "exists on present header",
			assertion: models.Assertion{
				Type:     "header",
				Target:   "Cache-Control",
				Operator: "exists",
				Value:    true,
			},
			wantPass: true,
		},
		{
			name: "exists on missing header fails",
			assertion: models.Assertion{
				Type:     "header",
				Target:   "X-Missing",
				Operator: "exists",
				Value:    true,
			},
			wantPass: false,
		},
		{
			name: "case insensitive header name",
			assertion: models.Assertion{
				Type:     "header",
				Target:   "content-type",
				Operator: "contains",
				Value:    "json",
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

// =============================================================================
// Body Size Assertion Tests
// =============================================================================

func TestBodySizeAssertion(t *testing.T) {
	body := []byte(`{"id": 1, "name": "test", "description": "a longer description"}`)
	ctx := NewContext(200, 100*time.Millisecond, body, nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "lt passes when body is smaller",
			assertion: models.Assertion{
				Type:     "body_size",
				Operator: "lt",
				Value:    float64(1000),
			},
			wantPass: true,
		},
		{
			name: "gt passes when body is larger",
			assertion: models.Assertion{
				Type:     "body_size",
				Operator: "gt",
				Value:    float64(10),
			},
			wantPass: true,
		},
		{
			name: "eq matches exact size",
			assertion: models.Assertion{
				Type:     "body_size",
				Operator: "eq",
				Value:    float64(len(body)),
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

// =============================================================================
// Operator Tests
// =============================================================================

func TestComparisonOperators(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"value": 50}`), nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "gt greater than",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "value",
				Operator: "gt",
				Value:    float64(40),
			},
			wantPass: true,
		},
		{
			name: "gt not greater than",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "value",
				Operator: "gt",
				Value:    float64(60),
			},
			wantPass: false,
		},
		{
			name: "gte greater than or equal (greater)",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "value",
				Operator: "gte",
				Value:    float64(40),
			},
			wantPass: true,
		},
		{
			name: "gte greater than or equal (equal)",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "value",
				Operator: "gte",
				Value:    float64(50),
			},
			wantPass: true,
		},
		{
			name: "lt less than",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "value",
				Operator: "lt",
				Value:    float64(60),
			},
			wantPass: true,
		},
		{
			name: "lte less than or equal (less)",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "value",
				Operator: "lte",
				Value:    float64(60),
			},
			wantPass: true,
		},
		{
			name: "lte less than or equal (equal)",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "value",
				Operator: "lte",
				Value:    float64(50),
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

func TestStringOperators(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"text": "Hello, World!"}`), nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "contains substring",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "text",
				Operator: "contains",
				Value:    "World",
			},
			wantPass: true,
		},
		{
			name: "contains fails on missing substring",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "text",
				Operator: "contains",
				Value:    "Universe",
			},
			wantPass: false,
		},
		{
			name: "starts_with prefix",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "text",
				Operator: "starts_with",
				Value:    "Hello",
			},
			wantPass: true,
		},
		{
			name: "ends_with suffix",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "text",
				Operator: "ends_with",
				Value:    "World!",
			},
			wantPass: true,
		},
		{
			name: "matches regex",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "text",
				Operator: "matches",
				Value:    "^Hello.*!$",
			},
			wantPass: true,
		},
		{
			name: "matches regex fails",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "text",
				Operator: "matches",
				Value:    "^Goodbye.*",
			},
			wantPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

// =============================================================================
// EvaluateAll Tests
// =============================================================================

func TestEvaluateAll(t *testing.T) {
	ctx := NewContext(
		200,
		100*time.Millisecond,
		[]byte(`{"id": 1, "name": "test"}`),
		http.Header{"Content-Type": []string{"application/json"}},
	)
	e := New(false)

	assertions := []models.Assertion{
		{Type: "status", Operator: "eq", Value: float64(200)},
		{Type: "json_path", Target: "id", Operator: "eq", Value: float64(1)},
		{Type: "header", Target: "Content-Type", Operator: "contains", Value: "json"},
		{Type: "response_time", Operator: "lt", Value: "500ms"},
	}

	results := e.EvaluateAll(assertions, ctx)

	require.Len(t, results, 4)
	for i, r := range results {
		assert.True(t, r.Passed, "Assertion %d failed: %s", i, r.Message)
	}
}

func TestEvaluateAll_PartialFailure(t *testing.T) {
	ctx := NewContext(
		404,
		100*time.Millisecond,
		[]byte(`{"error": "not found"}`),
		nil,
	)
	e := New(false)

	assertions := []models.Assertion{
		{Type: "status", Operator: "eq", Value: float64(200)},  // Will fail
		{Type: "json_path", Target: "error", Operator: "eq", Value: "not found"}, // Will pass
	}

	results := e.EvaluateAll(assertions, ctx)

	require.Len(t, results, 2)
	assert.False(t, results[0].Passed, "Status assertion should fail")
	assert.True(t, results[1].Passed, "JSON path assertion should pass")
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestInvalidAssertionType(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, nil, nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "unknown_type",
		Operator: "eq",
		Value:    "test",
	}, ctx)

	assert.False(t, result.Passed)
	assert.Contains(t, result.Message, "unknown assertion type")
}

func TestInvalidOperator(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"id": 1}`), nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "json_path",
		Target:   "id",
		Operator: "unknown_operator",
		Value:    float64(1),
	}, ctx)

	assert.False(t, result.Passed)
	assert.Contains(t, result.Message, "unknown operator")
}

func TestEmptyBody(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte{}, nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "json_path",
		Target:   "id",
		Operator: "eq",
		Value:    float64(1),
	}, ctx)

	assert.False(t, result.Passed)
}

func TestInvalidJSON(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`not valid json`), nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "json_path",
		Target:   "id",
		Operator: "eq",
		Value:    float64(1),
	}, ctx)

	assert.False(t, result.Passed)
}

func TestNilHeaders(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, nil, nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "header",
		Target:   "Content-Type",
		Operator: "exists",
		Value:    true,
	}, ctx)

	assert.False(t, result.Passed)
}

// =============================================================================
// Boolean Assertions
// =============================================================================

func TestBooleanAssertions(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"active": true, "deleted": false}`), nil)
	e := New(false)

	tests := []struct {
		name      string
		assertion models.Assertion
		wantPass  bool
	}{
		{
			name: "eq true boolean",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "active",
				Operator: "eq",
				Value:    true,
			},
			wantPass: true,
		},
		{
			name: "eq false boolean",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "deleted",
				Operator: "eq",
				Value:    false,
			},
			wantPass: true,
		},
		{
			name: "neq boolean",
			assertion: models.Assertion{
				Type:     "json_path",
				Target:   "active",
				Operator: "neq",
				Value:    false,
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := e.Evaluate(tt.assertion, ctx)
			assert.Equal(t, tt.wantPass, result.Passed, "Message: %s", result.Message)
		})
	}
}

// =============================================================================
// Result Message Tests
// =============================================================================

func TestResultContainsActualValue(t *testing.T) {
	ctx := NewContext(200, 100*time.Millisecond, []byte(`{"id": 42}`), nil)
	e := New(false)

	result := e.Evaluate(models.Assertion{
		Type:     "json_path",
		Target:   "id",
		Operator: "eq",
		Value:    float64(99),
	}, ctx)

	assert.False(t, result.Passed)
	assert.Equal(t, float64(42), result.ActualValue)
	assert.Contains(t, result.Message, "42")
	assert.Contains(t, result.Message, "99")
}
