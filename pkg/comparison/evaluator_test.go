package comparison

import (
	"testing"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFieldMatch_ExactMatch(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"id": 1, "name": "test"}`), nil,
		200, 100*time.Millisecond, []byte(`{"id": 1, "name": "test"}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:   "field_match",
		Target: "id",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
	assert.Len(t, result.AssertionResults, 1)
	assert.True(t, result.AssertionResults[0].Passed)
}

func TestFieldMatch_ValueMismatch(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"id": 1}`), nil,
		200, 100*time.Millisecond, []byte(`{"id": 2}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:   "field_match",
		Target: "id",
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
	assert.Len(t, result.AssertionResults, 1)
	assert.False(t, result.AssertionResults[0].Passed)
}

func TestFieldMatch_NestedField(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"data": {"value": 42}}`), nil,
		200, 100*time.Millisecond, []byte(`{"data": {"value": 42}}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:   "field_match",
		Target: "data.value",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}

func TestFieldMatch_MissingInPrimary(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`), nil,
		200, 100*time.Millisecond, []byte(`{"id": 1}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:   "field_match",
		Target: "id",
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
}

func TestFieldTolerance_WithinTolerance(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"value": 100}`), nil,
		200, 100*time.Millisecond, []byte(`{"value": 105}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:      "field_tolerance",
		Target:    "value",
		Tolerance: 0.10, // 10% tolerance
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}

func TestFieldTolerance_ExceedsTolerance(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"value": 100}`), nil,
		200, 100*time.Millisecond, []byte(`{"value": 150}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:      "field_tolerance",
		Target:    "value",
		Tolerance: 0.10, // 10% tolerance
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
}

func TestStatusMatch_Match(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`), nil,
		200, 100*time.Millisecond, []byte(`{}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type: "status_match",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
	assert.True(t, result.StatusMatch)
}

func TestStatusMatch_Mismatch(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`), nil,
		500, 100*time.Millisecond, []byte(`{}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type: "status_match",
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
	assert.False(t, result.StatusMatch)
}

func TestStructureMatch_SameStructure(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"users": [{"id": 1, "name": "Alice"}]}`), nil,
		200, 100*time.Millisecond, []byte(`{"users": [{"id": 2, "name": "Bob"}]}`), nil,
	)

	assertions := []models.CompareAssertion{{Type: "structure_match"}}
	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}

func TestStructureMatch_DifferentStructure(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"users": [{"id": 1}]}`), nil,
		200, 100*time.Millisecond, []byte(`{"users": [{"id": 1, "extra": "field"}]}`), nil,
	)

	assertions := []models.CompareAssertion{{Type: "structure_match"}}
	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
}

func TestIgnoreFields(t *testing.T) {
	e := New(false)
	e.SetIgnoreFields([]string{"timestamp", "request_id"})

	ctx := NewContext(
		200, 100*time.Millisecond,
		[]byte(`{"id": 1, "timestamp": "2024-01-01", "request_id": "abc"}`), nil,
		200, 100*time.Millisecond,
		[]byte(`{"id": 1, "timestamp": "2024-02-02", "request_id": "xyz"}`), nil,
	)

	result := e.Compare(ctx, nil)
	assert.True(t, result.Success)
}

func TestIgnoreFields_NestedPath(t *testing.T) {
	e := New(false)
	e.SetIgnoreFields([]string{"meta.timestamp"})

	ctx := NewContext(
		200, 100*time.Millisecond,
		[]byte(`{"id": 1, "meta": {"timestamp": "2024-01-01"}}`), nil,
		200, 100*time.Millisecond,
		[]byte(`{"id": 1, "meta": {"timestamp": "2024-02-02"}}`), nil,
	)

	result := e.Compare(ctx, nil)
	assert.True(t, result.Success)
}

func TestResponseTimeTolerance_WithinTolerance(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`), nil,
		200, 110*time.Millisecond, []byte(`{}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:      "response_time_tolerance",
		Tolerance: 0.20, // 20% tolerance
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}

func TestResponseTimeTolerance_ExceedsTolerance(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`), nil,
		200, 200*time.Millisecond, []byte(`{}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:      "response_time_tolerance",
		Tolerance: 0.10, // 10% tolerance
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
}

func TestFullBodyComparison_NoAssertions(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"id": 1, "name": "test"}`), nil,
		200, 100*time.Millisecond, []byte(`{"id": 1, "name": "test"}`), nil,
	)

	// No assertions - should do full body comparison
	result := e.Compare(ctx, nil)
	assert.True(t, result.Success)
}

func TestFullBodyComparison_Difference(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"id": 1, "name": "test"}`), nil,
		200, 100*time.Millisecond, []byte(`{"id": 1, "name": "different"}`), nil,
	)

	// No assertions - should do full body comparison and find difference
	result := e.Compare(ctx, nil)
	assert.False(t, result.Success)
	assert.True(t, len(result.FieldDiffs) > 0)
}

func TestParseTolerance_Percentage(t *testing.T) {
	e := New(false)

	// Value < 1 should be treated as percentage
	tol := e.parseTolerance(0.10)
	assert.True(t, tol.isPercentage)
	assert.Equal(t, 0.10, tol.value)
}

func TestParseTolerance_Absolute(t *testing.T) {
	e := New(false)

	// Value >= 1 should be treated as absolute
	tol := e.parseTolerance(10.0)
	assert.False(t, tol.isPercentage)
	assert.Equal(t, 10.0, tol.value)
}

func TestParseTolerance_StringPercentage(t *testing.T) {
	e := New(false)

	tol := e.parseTolerance("15%")
	assert.True(t, tol.isPercentage)
	assert.Equal(t, 0.15, tol.value)
}

func TestCompareArrays(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"items": [1, 2, 3]}`), nil,
		200, 100*time.Millisecond, []byte(`{"items": [1, 2, 3]}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:   "field_match",
		Target: "items",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}

func TestCompareArrays_DifferentContent(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"items": [1, 2, 3]}`), nil,
		200, 100*time.Millisecond, []byte(`{"items": [1, 2, 4]}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:   "field_match",
		Target: "items",
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
}

func TestContainsOperator(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{"message": "hello"}`), nil,
		200, 100*time.Millisecond, []byte(`{"message": "hello world"}`), nil,
	)

	assertions := []models.CompareAssertion{{
		Type:     "field_match",
		Target:   "message",
		Operator: "contains",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}

func TestHeaderMatch_ExactMatch(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"Content-Type": {"application/json"}},
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"Content-Type": {"application/json"}},
	)

	assertions := []models.CompareAssertion{{
		Type:   "header_match",
		Target: "Content-Type",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
	assert.Len(t, result.AssertionResults, 1)
	assert.True(t, result.AssertionResults[0].Passed)
}

func TestHeaderMatch_Mismatch(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"Content-Type": {"application/json"}},
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"Content-Type": {"text/html"}},
	)

	assertions := []models.CompareAssertion{{
		Type:   "header_match",
		Target: "Content-Type",
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
	assert.False(t, result.AssertionResults[0].Passed)
}

func TestHeaderMatch_MissingInCompare(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"X-Custom": {"value"}},
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{},
	)

	assertions := []models.CompareAssertion{{
		Type:   "header_match",
		Target: "X-Custom",
	}}

	result := e.Compare(ctx, assertions)
	assert.False(t, result.Success)
}

func TestHeaderMatch_MissingInBoth(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{},
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{},
	)

	assertions := []models.CompareAssertion{{
		Type:   "header_match",
		Target: "X-Missing",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success) // Both missing is considered a match
}

func TestHeaderMatch_Contains(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"Content-Type": {"json"}},
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"Content-Type": {"application/json; charset=utf-8"}},
	)

	assertions := []models.CompareAssertion{{
		Type:     "header_match",
		Target:   "Content-Type",
		Operator: "contains",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}

func TestHeaderMatch_Exists(t *testing.T) {
	e := New(false)
	ctx := NewContext(
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{},
		200, 100*time.Millisecond, []byte(`{}`),
		map[string][]string{"ETag": {"\"abc123\""}},
	)

	assertions := []models.CompareAssertion{{
		Type:     "header_match",
		Target:   "ETag",
		Operator: "exists",
	}}

	result := e.Compare(ctx, assertions)
	assert.True(t, result.Success)
}
