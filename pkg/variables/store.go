package variables

import (
	"fmt"
	"sync"
)

// Store provides thread-safe storage for variables
type Store struct {
	mu        sync.RWMutex
	variables map[string]interface{}
}

// NewStore creates a new variable store
func NewStore() *Store {
	return &Store{
		variables: make(map[string]interface{}),
	}
}

// Set stores a variable with the given key and value
func (s *Store) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.variables[key] = value
}

// Get retrieves a variable by key
func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.variables[key]
	return val, ok
}

// GetString retrieves a variable as a string
func (s *Store) GetString(key string) string {
	val, ok := s.Get(key)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%v", val)
}

// Delete removes a variable by key
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.variables, key)
}

// Clear removes all variables
func (s *Store) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.variables = make(map[string]interface{})
}

// All returns a copy of all variables
func (s *Store) All() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]interface{}, len(s.variables))
	for k, v := range s.variables {
		result[k] = v
	}
	return result
}

// SetFromMap sets multiple variables from a map
func (s *Store) SetFromMap(data map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k, v := range data {
		s.variables[k] = v
	}
}
