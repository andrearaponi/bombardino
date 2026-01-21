package comparison

import (
	"time"
)

// Context holds both responses for comparison
type Context struct {
	PrimaryStatusCode   int
	PrimaryResponseTime time.Duration
	PrimaryBody         []byte
	PrimaryHeaders      map[string][]string

	CompareStatusCode   int
	CompareResponseTime time.Duration
	CompareBody         []byte
	CompareHeaders      map[string][]string
}

// NewContext creates a new comparison context
func NewContext(
	primaryStatus int, primaryTime time.Duration, primaryBody []byte, primaryHeaders map[string][]string,
	compareStatus int, compareTime time.Duration, compareBody []byte, compareHeaders map[string][]string,
) *Context {
	return &Context{
		PrimaryStatusCode:   primaryStatus,
		PrimaryResponseTime: primaryTime,
		PrimaryBody:         primaryBody,
		PrimaryHeaders:      primaryHeaders,
		CompareStatusCode:   compareStatus,
		CompareResponseTime: compareTime,
		CompareBody:         compareBody,
		CompareHeaders:      compareHeaders,
	}
}

// DiffType represents the type of difference found
type DiffType string

const (
	DiffMissing       DiffType = "missing"        // Field exists in primary but not compare
	DiffExtra         DiffType = "extra"          // Field exists in compare but not primary
	DiffTypeMismatch  DiffType = "type_mismatch"  // Different types
	DiffValueMismatch DiffType = "value_mismatch" // Different values
)

// Result holds comprehensive comparison results
type Result struct {
	Success          bool
	StatusMatch      bool
	StructureMatch   bool
	FieldDiffs       []FieldDiff
	AssertionResults []AssertionResult
	Error            error
}

// FieldDiff describes a single field difference
type FieldDiff struct {
	Path         string
	DiffType     DiffType
	PrimaryValue interface{}
	CompareValue interface{}
	Message      string
}

// AssertionResult holds the result of a single comparison assertion
type AssertionResult struct {
	Type         string
	Target       string
	Passed       bool
	PrimaryValue interface{}
	CompareValue interface{}
	Message      string
}
