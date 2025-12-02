package variables

import (
	"fmt"
	"net/http"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/tidwall/gjson"
)

// Extractor extracts variables from HTTP responses
type Extractor struct {
	store *Store
}

// NewExtractor creates a new extractor
func NewExtractor(store *Store) *Extractor {
	return &Extractor{
		store: store,
	}
}

// Extract extracts variables from a response based on the given rules
func (e *Extractor) Extract(rules []models.ExtractionRule, body []byte, headers http.Header, statusCode int) error {
	for _, rule := range rules {
		var value interface{}
		var found bool

		switch rule.Source {
		case "body":
			value, found = e.extractFromBody(body, rule.Path)
		case "header":
			value, found = e.extractFromHeader(headers, rule.Path)
		case "status":
			value = statusCode
			found = true
		default:
			return fmt.Errorf("unknown source: %s", rule.Source)
		}

		if found {
			e.store.Set(rule.Name, value)
		}
	}

	return nil
}

// extractFromBody extracts a value from JSON body using gjson path
func (e *Extractor) extractFromBody(body []byte, path string) (interface{}, bool) {
	if len(body) == 0 {
		return nil, false
	}

	result := gjson.GetBytes(body, path)
	if !result.Exists() {
		return nil, false
	}

	// Return the appropriate type
	switch result.Type {
	case gjson.String:
		return result.String(), true
	case gjson.Number:
		// Return as int if it's a whole number, otherwise float
		if result.Float() == float64(int(result.Float())) {
			return int(result.Float()), true
		}
		return result.Float(), true
	case gjson.True:
		return true, true
	case gjson.False:
		return false, true
	case gjson.Null:
		return nil, true
	default:
		// For arrays and objects, return raw JSON
		return result.Raw, true
	}
}

// extractFromHeader extracts a value from HTTP headers
func (e *Extractor) extractFromHeader(headers http.Header, headerName string) (interface{}, bool) {
	if headers == nil {
		return nil, false
	}

	value := headers.Get(headerName)
	if value == "" {
		return nil, false
	}

	return value, true
}
