package variables

import (
	"regexp"
)

// varPattern matches ${variable_name} patterns, including dotted names like ${data.username}
var varPattern = regexp.MustCompile(`\$\{([a-zA-Z_][a-zA-Z0-9_.]*)\}`)

// Substitutor replaces variable references with their values
type Substitutor struct {
	store *Store
}

// NewSubstitutor creates a new substitutor
func NewSubstitutor(store *Store) *Substitutor {
	return &Substitutor{
		store: store,
	}
}

// Substitute replaces all ${variable} patterns in the input string
func (s *Substitutor) Substitute(input string) string {
	return varPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name from ${name}
		varName := match[2 : len(match)-1]

		if value, ok := s.store.Get(varName); ok {
			return s.store.GetString(varName)
		} else {
			// Keep original if variable not found
			_ = value // Suppress unused warning
			return match
		}
	})
}

// SubstituteMap substitutes variables in all values of a string map
func (s *Substitutor) SubstituteMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = s.Substitute(v)
	}
	return result
}

// SubstituteBody substitutes variables in an arbitrary body structure
// Supports strings, maps, and arrays recursively
func (s *Substitutor) SubstituteBody(body interface{}) interface{} {
	if body == nil {
		return nil
	}

	switch v := body.(type) {
	case string:
		// Check if the entire string is a single variable reference
		// If so, return the actual value (preserving type for numbers, bools, etc.)
		if matches := varPattern.FindStringSubmatch(v); len(matches) == 2 && matches[0] == v {
			varName := matches[1]
			if value, ok := s.store.Get(varName); ok {
				return value
			}
			return v // Keep original if not found
		}
		// Otherwise do string substitution (for embedded variables)
		return s.Substitute(v)

	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			result[key] = s.SubstituteBody(val)
		}
		return result

	case map[string]string:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			result[key] = s.Substitute(val)
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = s.SubstituteBody(val)
		}
		return result

	case []string:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = s.Substitute(val)
		}
		return result

	default:
		// Return as-is for other types (int, float, bool, etc.)
		return v
	}
}
