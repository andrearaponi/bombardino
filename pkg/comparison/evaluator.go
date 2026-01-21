package comparison

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"reflect"
	"strings"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/tidwall/gjson"
)

// Evaluator performs response comparisons
type Evaluator struct {
	verbose      bool
	ignoreFields map[string]bool
	mode         string // "full", "partial", "structural"
}

// New creates a new comparison evaluator
func New(verbose bool) *Evaluator {
	return &Evaluator{
		verbose:      verbose,
		ignoreFields: make(map[string]bool),
		mode:         "full",
	}
}

// SetIgnoreFields sets the fields to ignore during comparison
func (e *Evaluator) SetIgnoreFields(fields []string) {
	e.ignoreFields = make(map[string]bool)
	for _, f := range fields {
		e.ignoreFields[f] = true
	}
}

// SetMode sets the comparison mode
func (e *Evaluator) SetMode(mode string) {
	if mode == "" {
		mode = "full"
	}
	e.mode = mode
}

// Compare performs the comparison based on configured assertions
func (e *Evaluator) Compare(ctx *Context, assertions []models.CompareAssertion) *Result {
	result := &Result{
		Success:     true,
		StatusMatch: ctx.PrimaryStatusCode == ctx.CompareStatusCode,
	}

	// Check status codes first
	if !result.StatusMatch {
		result.Success = false
		result.FieldDiffs = append(result.FieldDiffs, FieldDiff{
			Path:         "_status_code",
			DiffType:     DiffValueMismatch,
			PrimaryValue: ctx.PrimaryStatusCode,
			CompareValue: ctx.CompareStatusCode,
			Message:      fmt.Sprintf("Status code mismatch: primary=%d, compare=%d", ctx.PrimaryStatusCode, ctx.CompareStatusCode),
		})
	}

	// If no specific assertions, do full body comparison
	if len(assertions) == 0 {
		diffs := e.compareJSONBodies(ctx.PrimaryBody, ctx.CompareBody, "")
		result.FieldDiffs = append(result.FieldDiffs, diffs...)
		if len(diffs) > 0 {
			result.Success = false
		}
	} else {
		// Evaluate specific assertions
		for _, assertion := range assertions {
			ar := e.evaluateAssertion(assertion, ctx)
			result.AssertionResults = append(result.AssertionResults, ar)
			if !ar.Passed {
				result.Success = false
			}
		}
	}

	return result
}

// evaluateAssertion evaluates a single comparison assertion
func (e *Evaluator) evaluateAssertion(assertion models.CompareAssertion, ctx *Context) AssertionResult {
	switch assertion.Type {
	case "field_match":
		return e.evaluateFieldMatch(assertion, ctx)
	case "field_tolerance":
		return e.evaluateFieldTolerance(assertion, ctx)
	case "structure_match":
		return e.evaluateStructureMatch(ctx)
	case "status_match":
		return e.evaluateStatusMatch(ctx)
	case "response_time_tolerance":
		return e.evaluateResponseTimeTolerance(assertion, ctx)
	case "header_match":
		return e.evaluateHeaderMatch(assertion, ctx)
	default:
		return AssertionResult{
			Type:    assertion.Type,
			Target:  assertion.Target,
			Passed:  false,
			Message: fmt.Sprintf("unknown comparison type: %s", assertion.Type),
		}
	}
}

// evaluateFieldMatch checks if a specific field matches between responses
func (e *Evaluator) evaluateFieldMatch(assertion models.CompareAssertion, ctx *Context) AssertionResult {
	result := AssertionResult{
		Type:   assertion.Type,
		Target: assertion.Target,
	}

	primaryVal := gjson.GetBytes(ctx.PrimaryBody, assertion.Target)
	compareVal := gjson.GetBytes(ctx.CompareBody, assertion.Target)

	result.PrimaryValue = primaryVal.Value()
	result.CompareValue = compareVal.Value()

	if !primaryVal.Exists() && !compareVal.Exists() {
		result.Passed = true
		result.Message = fmt.Sprintf("field '%s' does not exist in either response", assertion.Target)
		return result
	}

	if !primaryVal.Exists() {
		result.Passed = false
		result.Message = fmt.Sprintf("field '%s' missing in primary response", assertion.Target)
		return result
	}

	if !compareVal.Exists() {
		result.Passed = false
		result.Message = fmt.Sprintf("field '%s' missing in compare response", assertion.Target)
		return result
	}

	// Apply operator
	operator := assertion.Operator
	if operator == "" {
		operator = "eq"
	}

	switch operator {
	case "eq":
		result.Passed = reflect.DeepEqual(primaryVal.Value(), compareVal.Value())
	case "contains":
		result.Passed = strings.Contains(
			fmt.Sprintf("%v", compareVal.Value()),
			fmt.Sprintf("%v", primaryVal.Value()),
		)
	default:
		result.Passed = primaryVal.Raw == compareVal.Raw
	}

	if !result.Passed {
		result.Message = fmt.Sprintf("field '%s' mismatch: primary=%v, compare=%v",
			assertion.Target, primaryVal.Value(), compareVal.Value())
	}

	return result
}

// evaluateFieldTolerance checks if numeric fields are within tolerance
func (e *Evaluator) evaluateFieldTolerance(assertion models.CompareAssertion, ctx *Context) AssertionResult {
	result := AssertionResult{
		Type:   assertion.Type,
		Target: assertion.Target,
	}

	primaryVal := gjson.GetBytes(ctx.PrimaryBody, assertion.Target)
	compareVal := gjson.GetBytes(ctx.CompareBody, assertion.Target)

	result.PrimaryValue = primaryVal.Value()
	result.CompareValue = compareVal.Value()

	if !primaryVal.Exists() || !compareVal.Exists() {
		result.Passed = false
		result.Message = fmt.Sprintf("field '%s' missing in one or both responses", assertion.Target)
		return result
	}

	primaryNum := primaryVal.Float()
	compareNum := compareVal.Float()

	tolerance := e.parseTolerance(assertion.Tolerance)

	var diff float64
	if tolerance.isPercentage {
		// Percentage tolerance
		if primaryNum == 0 {
			diff = math.Abs(compareNum)
		} else {
			diff = math.Abs((compareNum - primaryNum) / primaryNum)
		}
		result.Passed = diff <= tolerance.value
		if !result.Passed {
			result.Message = fmt.Sprintf("field '%s' exceeds tolerance: diff=%.2f%%, tolerance=%.2f%%",
				assertion.Target, diff*100, tolerance.value*100)
		}
	} else {
		// Absolute tolerance
		diff = math.Abs(compareNum - primaryNum)
		result.Passed = diff <= tolerance.value
		if !result.Passed {
			result.Message = fmt.Sprintf("field '%s' exceeds tolerance: diff=%.4f, tolerance=%.4f",
				assertion.Target, diff, tolerance.value)
		}
	}

	return result
}

type toleranceValue struct {
	value        float64
	isPercentage bool
}

func (e *Evaluator) parseTolerance(val interface{}) toleranceValue {
	switch v := val.(type) {
	case float64:
		// If less than 1, treat as percentage (0.1 = 10%)
		if v > 0 && v < 1 {
			return toleranceValue{value: v, isPercentage: true}
		}
		return toleranceValue{value: v, isPercentage: false}
	case int:
		return toleranceValue{value: float64(v), isPercentage: false}
	case string:
		if strings.HasSuffix(v, "%") {
			var pct float64
			fmt.Sscanf(v, "%f%%", &pct)
			return toleranceValue{value: pct / 100, isPercentage: true}
		}
		var num float64
		fmt.Sscanf(v, "%f", &num)
		return toleranceValue{value: num, isPercentage: false}
	default:
		return toleranceValue{value: 0, isPercentage: false}
	}
}

// evaluateStructureMatch checks if JSON structures match (ignoring values)
func (e *Evaluator) evaluateStructureMatch(ctx *Context) AssertionResult {
	result := AssertionResult{
		Type: "structure_match",
	}

	var primaryData, compareData interface{}
	if err := json.Unmarshal(ctx.PrimaryBody, &primaryData); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("failed to parse primary body: %v", err)
		return result
	}
	if err := json.Unmarshal(ctx.CompareBody, &compareData); err != nil {
		result.Passed = false
		result.Message = fmt.Sprintf("failed to parse compare body: %v", err)
		return result
	}

	result.Passed = e.structuresMatch(primaryData, compareData, "")
	if !result.Passed {
		result.Message = "JSON structure mismatch detected"
	}

	return result
}

// structuresMatch recursively checks if two JSON structures have the same shape
func (e *Evaluator) structuresMatch(a, b interface{}, path string) bool {
	if e.isIgnored(path) {
		return true
	}

	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)

	if aType != bType {
		return false
	}

	switch aVal := a.(type) {
	case map[string]interface{}:
		bVal, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		for key := range aVal {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			if _, ok := bVal[key]; !ok {
				if !e.isIgnored(newPath) {
					return false
				}
			}
			if !e.structuresMatch(aVal[key], bVal[key], newPath) {
				return false
			}
		}
		// Check for extra keys in b
		for key := range bVal {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			if _, ok := aVal[key]; !ok {
				if !e.isIgnored(newPath) {
					return false
				}
			}
		}
	case []interface{}:
		bVal, ok := b.([]interface{})
		if !ok {
			return false
		}
		if len(aVal) > 0 && len(bVal) > 0 {
			// Check first element structure
			return e.structuresMatch(aVal[0], bVal[0], path+"[0]")
		}
	}

	return true
}

// evaluateStatusMatch checks if status codes match
func (e *Evaluator) evaluateStatusMatch(ctx *Context) AssertionResult {
	result := AssertionResult{
		Type:         "status_match",
		PrimaryValue: ctx.PrimaryStatusCode,
		CompareValue: ctx.CompareStatusCode,
		Passed:       ctx.PrimaryStatusCode == ctx.CompareStatusCode,
	}

	if !result.Passed {
		result.Message = fmt.Sprintf("status mismatch: primary=%d, compare=%d",
			ctx.PrimaryStatusCode, ctx.CompareStatusCode)
	}

	return result
}

// evaluateResponseTimeTolerance checks response time within tolerance
func (e *Evaluator) evaluateResponseTimeTolerance(assertion models.CompareAssertion, ctx *Context) AssertionResult {
	result := AssertionResult{
		Type:         "response_time_tolerance",
		PrimaryValue: ctx.PrimaryResponseTime.String(),
		CompareValue: ctx.CompareResponseTime.String(),
	}

	tolerance := e.parseTolerance(assertion.Tolerance)
	primaryMs := float64(ctx.PrimaryResponseTime.Milliseconds())
	compareMs := float64(ctx.CompareResponseTime.Milliseconds())

	var diff float64
	if tolerance.isPercentage {
		if primaryMs == 0 {
			diff = math.Abs(compareMs)
		} else {
			diff = math.Abs((compareMs - primaryMs) / primaryMs)
		}
		result.Passed = diff <= tolerance.value
	} else {
		diff = math.Abs(compareMs - primaryMs)
		result.Passed = diff <= tolerance.value
	}

	if !result.Passed {
		result.Message = fmt.Sprintf("response time diff exceeds tolerance: primary=%v, compare=%v",
			ctx.PrimaryResponseTime, ctx.CompareResponseTime)
	}

	return result
}

// evaluateHeaderMatch compares specific headers between responses
func (e *Evaluator) evaluateHeaderMatch(assertion models.CompareAssertion, ctx *Context) AssertionResult {
	result := AssertionResult{
		Type:   assertion.Type,
		Target: assertion.Target,
	}

	// Get header values (case-insensitive)
	headerName := assertion.Target
	primaryVals := ctx.PrimaryHeaders[headerName]
	compareVals := ctx.CompareHeaders[headerName]

	// Try canonical form if not found
	if len(primaryVals) == 0 {
		primaryVals = ctx.PrimaryHeaders[http.CanonicalHeaderKey(headerName)]
	}
	if len(compareVals) == 0 {
		compareVals = ctx.CompareHeaders[http.CanonicalHeaderKey(headerName)]
	}

	result.PrimaryValue = primaryVals
	result.CompareValue = compareVals

	// Apply operator
	operator := assertion.Operator
	if operator == "" {
		operator = "eq"
	}

	// For "exists" operator, only check if header exists in compare response
	if operator == "exists" {
		result.Passed = len(compareVals) > 0
		if !result.Passed {
			result.Message = fmt.Sprintf("header '%s' does not exist in compare response", headerName)
		}
		return result
	}

	// Both missing is a match (for eq/contains)
	if len(primaryVals) == 0 && len(compareVals) == 0 {
		result.Passed = true
		result.Message = fmt.Sprintf("header '%s' not present in either response", headerName)
		return result
	}

	// One missing
	if len(primaryVals) == 0 {
		result.Passed = false
		result.Message = fmt.Sprintf("header '%s' missing in primary response", headerName)
		return result
	}
	if len(compareVals) == 0 {
		result.Passed = false
		result.Message = fmt.Sprintf("header '%s' missing in compare response", headerName)
		return result
	}

	primaryVal := strings.Join(primaryVals, ", ")
	compareVal := strings.Join(compareVals, ", ")

	switch operator {
	case "eq":
		result.Passed = primaryVal == compareVal
	case "contains":
		result.Passed = strings.Contains(compareVal, primaryVal)
	default:
		result.Passed = primaryVal == compareVal
	}

	if !result.Passed {
		result.Message = fmt.Sprintf("header '%s' mismatch: primary=%s, compare=%s",
			headerName, primaryVal, compareVal)
	}

	return result
}

// compareJSONBodies performs deep comparison of two JSON bodies
func (e *Evaluator) compareJSONBodies(primary, compare []byte, basePath string) []FieldDiff {
	var primaryData, compareData interface{}
	if err := json.Unmarshal(primary, &primaryData); err != nil {
		return []FieldDiff{{Path: basePath, DiffType: DiffTypeMismatch, Message: "invalid primary JSON"}}
	}
	if err := json.Unmarshal(compare, &compareData); err != nil {
		return []FieldDiff{{Path: basePath, DiffType: DiffTypeMismatch, Message: "invalid compare JSON"}}
	}

	return e.compareValues(primaryData, compareData, basePath)
}

// compareValues recursively compares two values
func (e *Evaluator) compareValues(primary, compare interface{}, path string) []FieldDiff {
	var diffs []FieldDiff

	if e.isIgnored(path) {
		return diffs
	}

	if primary == nil && compare == nil {
		return diffs
	}

	if primary == nil {
		return []FieldDiff{{
			Path:         path,
			DiffType:     DiffExtra,
			CompareValue: compare,
			Message:      fmt.Sprintf("field '%s' only exists in compare response", path),
		}}
	}

	if compare == nil {
		return []FieldDiff{{
			Path:         path,
			DiffType:     DiffMissing,
			PrimaryValue: primary,
			Message:      fmt.Sprintf("field '%s' only exists in primary response", path),
		}}
	}

	primaryType := reflect.TypeOf(primary)
	compareType := reflect.TypeOf(compare)

	if primaryType != compareType {
		return []FieldDiff{{
			Path:         path,
			DiffType:     DiffTypeMismatch,
			PrimaryValue: primary,
			CompareValue: compare,
			Message:      fmt.Sprintf("type mismatch at '%s': primary=%T, compare=%T", path, primary, compare),
		}}
	}

	switch pVal := primary.(type) {
	case map[string]interface{}:
		cVal := compare.(map[string]interface{})

		// Check primary keys
		for key, pv := range pVal {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			if cv, ok := cVal[key]; ok {
				diffs = append(diffs, e.compareValues(pv, cv, newPath)...)
			} else if !e.isIgnored(newPath) {
				diffs = append(diffs, FieldDiff{
					Path:         newPath,
					DiffType:     DiffMissing,
					PrimaryValue: pv,
					Message:      fmt.Sprintf("field '%s' missing in compare response", newPath),
				})
			}
		}

		// Check for extra keys in compare
		for key, cv := range cVal {
			newPath := key
			if path != "" {
				newPath = path + "." + key
			}
			if _, ok := pVal[key]; !ok && !e.isIgnored(newPath) {
				diffs = append(diffs, FieldDiff{
					Path:         newPath,
					DiffType:     DiffExtra,
					CompareValue: cv,
					Message:      fmt.Sprintf("field '%s' only in compare response", newPath),
				})
			}
		}

	case []interface{}:
		cVal := compare.([]interface{})
		if e.mode == "structural" {
			// Only compare structure, not individual array elements
			if len(pVal) > 0 && len(cVal) > 0 {
				diffs = append(diffs, e.compareValues(pVal[0], cVal[0], path+"[0]")...)
			}
		} else {
			// Compare arrays element by element
			maxLen := len(pVal)
			if len(cVal) > maxLen {
				maxLen = len(cVal)
			}
			for i := 0; i < maxLen; i++ {
				elemPath := fmt.Sprintf("%s[%d]", path, i)
				if i >= len(pVal) {
					diffs = append(diffs, FieldDiff{
						Path:         elemPath,
						DiffType:     DiffExtra,
						CompareValue: cVal[i],
						Message:      fmt.Sprintf("extra element at '%s'", elemPath),
					})
				} else if i >= len(cVal) {
					diffs = append(diffs, FieldDiff{
						Path:         elemPath,
						DiffType:     DiffMissing,
						PrimaryValue: pVal[i],
						Message:      fmt.Sprintf("missing element at '%s'", elemPath),
					})
				} else {
					diffs = append(diffs, e.compareValues(pVal[i], cVal[i], elemPath)...)
				}
			}
		}

	default:
		if !reflect.DeepEqual(primary, compare) {
			diffs = append(diffs, FieldDiff{
				Path:         path,
				DiffType:     DiffValueMismatch,
				PrimaryValue: primary,
				CompareValue: compare,
				Message:      fmt.Sprintf("value mismatch at '%s': primary=%v, compare=%v", path, primary, compare),
			})
		}
	}

	return diffs
}

// isIgnored checks if a path should be ignored
func (e *Evaluator) isIgnored(path string) bool {
	if path == "" {
		return false
	}

	// Check exact match
	if e.ignoreFields[path] {
		return true
	}

	// Check if any parent path is ignored (for nested fields)
	parts := strings.Split(path, ".")
	for i := range parts {
		parentPath := strings.Join(parts[:i+1], ".")
		if e.ignoreFields[parentPath] {
			return true
		}
	}

	return false
}
