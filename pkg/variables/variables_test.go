package variables

import (
	"net/http"
	"sync"
	"testing"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Store Tests
// =============================================================================

func TestNewStore(t *testing.T) {
	s := NewStore()
	assert.NotNil(t, s)
}

func TestStore_SetAndGet(t *testing.T) {
	s := NewStore()

	s.Set("token", "abc123")
	val, ok := s.Get("token")

	assert.True(t, ok)
	assert.Equal(t, "abc123", val)
}

func TestStore_GetString(t *testing.T) {
	s := NewStore()

	s.Set("token", "abc123")
	s.Set("count", 42)
	s.Set("rate", 3.14)

	assert.Equal(t, "abc123", s.GetString("token"))
	assert.Equal(t, "42", s.GetString("count"))
	assert.Equal(t, "3.14", s.GetString("rate"))
	assert.Equal(t, "", s.GetString("missing"))
}

func TestStore_GetMissing(t *testing.T) {
	s := NewStore()

	val, ok := s.Get("missing")

	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestStore_Delete(t *testing.T) {
	s := NewStore()

	s.Set("token", "abc123")
	s.Delete("token")
	_, ok := s.Get("token")

	assert.False(t, ok)
}

func TestStore_Clear(t *testing.T) {
	s := NewStore()

	s.Set("a", 1)
	s.Set("b", 2)
	s.Set("c", 3)
	s.Clear()

	assert.Equal(t, 0, len(s.All()))
}

func TestStore_All(t *testing.T) {
	s := NewStore()

	s.Set("a", 1)
	s.Set("b", "two")

	all := s.All()
	assert.Equal(t, 2, len(all))
	assert.Equal(t, 1, all["a"])
	assert.Equal(t, "two", all["b"])
}

func TestStore_ThreadSafety(t *testing.T) {
	s := NewStore()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.Set("key", i)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Get("key")
		}()
	}

	wg.Wait()
	// If we get here without race conditions, test passes
}

func TestStore_SetFromMap(t *testing.T) {
	s := NewStore()

	data := map[string]interface{}{
		"user_id": 123,
		"email":   "test@example.com",
		"active":  true,
	}

	s.SetFromMap(data)

	assert.Equal(t, 123, s.All()["user_id"])
	assert.Equal(t, "test@example.com", s.All()["email"])
	assert.Equal(t, true, s.All()["active"])
}

// =============================================================================
// Extractor Tests
// =============================================================================

func TestNewExtractor(t *testing.T) {
	s := NewStore()
	e := NewExtractor(s)
	assert.NotNil(t, e)
}

func TestExtractor_ExtractFromBody(t *testing.T) {
	s := NewStore()
	e := NewExtractor(s)

	body := []byte(`{"token": "jwt-token-123", "user": {"id": 42, "email": "test@example.com"}}`)

	rules := []models.ExtractionRule{
		{Name: "auth_token", Source: "body", Path: "token"},
		{Name: "user_id", Source: "body", Path: "user.id"},
		{Name: "user_email", Source: "body", Path: "user.email"},
	}

	err := e.Extract(rules, body, nil, 200)
	require.NoError(t, err)

	assert.Equal(t, "jwt-token-123", s.GetString("auth_token"))
	assert.Equal(t, "42", s.GetString("user_id"))
	assert.Equal(t, "test@example.com", s.GetString("user_email"))
}

func TestExtractor_ExtractFromHeader(t *testing.T) {
	s := NewStore()
	e := NewExtractor(s)

	headers := http.Header{
		"X-Request-Id":  []string{"req-12345"},
		"X-Rate-Limit":  []string{"100"},
		"Authorization": []string{"Bearer secret-token"},
	}

	rules := []models.ExtractionRule{
		{Name: "request_id", Source: "header", Path: "X-Request-Id"},
		{Name: "rate_limit", Source: "header", Path: "X-Rate-Limit"},
	}

	err := e.Extract(rules, nil, headers, 200)
	require.NoError(t, err)

	assert.Equal(t, "req-12345", s.GetString("request_id"))
	assert.Equal(t, "100", s.GetString("rate_limit"))
}

func TestExtractor_ExtractFromStatus(t *testing.T) {
	s := NewStore()
	e := NewExtractor(s)

	rules := []models.ExtractionRule{
		{Name: "status", Source: "status", Path: ""},
	}

	err := e.Extract(rules, nil, nil, 201)
	require.NoError(t, err)

	assert.Equal(t, "201", s.GetString("status"))
}

func TestExtractor_ExtractNestedJSON(t *testing.T) {
	s := NewStore()
	e := NewExtractor(s)

	body := []byte(`{
		"data": {
			"users": [
				{"id": 1, "name": "Alice"},
				{"id": 2, "name": "Bob"}
			]
		}
	}`)

	rules := []models.ExtractionRule{
		{Name: "first_user_id", Source: "body", Path: "data.users.0.id"},
		{Name: "first_user_name", Source: "body", Path: "data.users.0.name"},
		{Name: "user_count", Source: "body", Path: "data.users.#"},
	}

	err := e.Extract(rules, body, nil, 200)
	require.NoError(t, err)

	assert.Equal(t, "1", s.GetString("first_user_id"))
	assert.Equal(t, "Alice", s.GetString("first_user_name"))
	assert.Equal(t, "2", s.GetString("user_count"))
}

func TestExtractor_MissingPath(t *testing.T) {
	s := NewStore()
	e := NewExtractor(s)

	body := []byte(`{"id": 1}`)

	rules := []models.ExtractionRule{
		{Name: "missing", Source: "body", Path: "nonexistent.path"},
	}

	err := e.Extract(rules, body, nil, 200)
	// Should not error, just not set the variable
	assert.NoError(t, err)

	_, ok := s.Get("missing")
	assert.False(t, ok)
}

func TestExtractor_InvalidSource(t *testing.T) {
	s := NewStore()
	e := NewExtractor(s)

	rules := []models.ExtractionRule{
		{Name: "test", Source: "invalid", Path: "path"},
	}

	err := e.Extract(rules, nil, nil, 200)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown source")
}

// =============================================================================
// Substitutor Tests
// =============================================================================

func TestNewSubstitutor(t *testing.T) {
	s := NewStore()
	sub := NewSubstitutor(s)
	assert.NotNil(t, sub)
}

func TestSubstitutor_SubstituteString(t *testing.T) {
	s := NewStore()
	s.Set("user_id", "123")
	s.Set("token", "abc")

	sub := NewSubstitutor(s)

	tests := []struct {
		input    string
		expected string
	}{
		{"/users/${user_id}", "/users/123"},
		{"Bearer ${token}", "Bearer abc"},
		{"${user_id}/${token}", "123/abc"},
		{"no variables here", "no variables here"},
		{"${missing}", "${missing}"}, // Missing variables stay as-is
		{"", ""},
	}

	for _, tt := range tests {
		result := sub.Substitute(tt.input)
		assert.Equal(t, tt.expected, result, "Input: %s", tt.input)
	}
}

func TestSubstitutor_SubstituteMap(t *testing.T) {
	s := NewStore()
	s.Set("token", "secret123")
	s.Set("content_type", "application/json")

	sub := NewSubstitutor(s)

	headers := map[string]string{
		"Authorization": "Bearer ${token}",
		"Content-Type":  "${content_type}",
		"Accept":        "text/html",
	}

	result := sub.SubstituteMap(headers)

	assert.Equal(t, "Bearer secret123", result["Authorization"])
	assert.Equal(t, "application/json", result["Content-Type"])
	assert.Equal(t, "text/html", result["Accept"])
}

func TestSubstitutor_SubstituteBody(t *testing.T) {
	s := NewStore()
	s.Set("username", "john")
	s.Set("email", "john@example.com")

	sub := NewSubstitutor(s)

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "simple string",
			input:    "${username}",
			expected: "john",
		},
		{
			name: "map with variables",
			input: map[string]interface{}{
				"user":  "${username}",
				"email": "${email}",
				"count": 42,
			},
			expected: map[string]interface{}{
				"user":  "john",
				"email": "john@example.com",
				"count": 42,
			},
		},
		{
			name: "nested map",
			input: map[string]interface{}{
				"data": map[string]interface{}{
					"name": "${username}",
				},
			},
			expected: map[string]interface{}{
				"data": map[string]interface{}{
					"name": "john",
				},
			},
		},
		{
			name:     "array of strings",
			input:    []interface{}{"${username}", "${email}", "literal"},
			expected: []interface{}{"john", "john@example.com", "literal"},
		},
		{
			name:     "integer passthrough",
			input:    42,
			expected: 42,
		},
		{
			name:     "nil passthrough",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sub.SubstituteBody(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubstitutor_NestedVariables(t *testing.T) {
	s := NewStore()
	s.Set("base", "api")
	s.Set("version", "v1")

	sub := NewSubstitutor(s)

	// Multiple variables in one string
	result := sub.Substitute("/${base}/${version}/users")
	assert.Equal(t, "/api/v1/users", result)
}

// =============================================================================
// DAG (Dependency Graph) Tests
// =============================================================================

func TestBuildDAG_NoDependencies(t *testing.T) {
	tests := []TestDependency{
		{Name: "TestA"},
		{Name: "TestB"},
		{Name: "TestC"},
	}

	plan, err := BuildDAG(tests)
	require.NoError(t, err)

	// All tests should be in the first phase (can run in parallel)
	require.Len(t, plan.Phases, 1)
	assert.ElementsMatch(t, []string{"TestA", "TestB", "TestC"}, plan.Phases[0])
}

func TestBuildDAG_LinearDependencies(t *testing.T) {
	tests := []TestDependency{
		{Name: "Login"},
		{Name: "GetProfile", DependsOn: []string{"Login"}},
		{Name: "UpdateProfile", DependsOn: []string{"GetProfile"}},
	}

	plan, err := BuildDAG(tests)
	require.NoError(t, err)

	// Should have 3 phases
	require.Len(t, plan.Phases, 3)
	assert.Equal(t, []string{"Login"}, plan.Phases[0])
	assert.Equal(t, []string{"GetProfile"}, plan.Phases[1])
	assert.Equal(t, []string{"UpdateProfile"}, plan.Phases[2])
}

func TestBuildDAG_ParallelWithDependencies(t *testing.T) {
	tests := []TestDependency{
		{Name: "Login"},
		{Name: "HealthCheck"}, // No dependency, can run with Login
		{Name: "GetProfile", DependsOn: []string{"Login"}},
		{Name: "GetSettings", DependsOn: []string{"Login"}}, // Can run parallel with GetProfile
	}

	plan, err := BuildDAG(tests)
	require.NoError(t, err)

	// Phase 1: Login, HealthCheck (no deps)
	// Phase 2: GetProfile, GetSettings (both depend on Login)
	require.Len(t, plan.Phases, 2)
	assert.ElementsMatch(t, []string{"Login", "HealthCheck"}, plan.Phases[0])
	assert.ElementsMatch(t, []string{"GetProfile", "GetSettings"}, plan.Phases[1])
}

func TestBuildDAG_MultipleDependencies(t *testing.T) {
	tests := []TestDependency{
		{Name: "Login"},
		{Name: "GetConfig"},
		{Name: "DoAction", DependsOn: []string{"Login", "GetConfig"}},
	}

	plan, err := BuildDAG(tests)
	require.NoError(t, err)

	// Phase 1: Login, GetConfig
	// Phase 2: DoAction (depends on both)
	require.Len(t, plan.Phases, 2)
	assert.ElementsMatch(t, []string{"Login", "GetConfig"}, plan.Phases[0])
	assert.Equal(t, []string{"DoAction"}, plan.Phases[1])
}

func TestBuildDAG_CyclicDependency(t *testing.T) {
	tests := []TestDependency{
		{Name: "A", DependsOn: []string{"C"}},
		{Name: "B", DependsOn: []string{"A"}},
		{Name: "C", DependsOn: []string{"B"}},
	}

	_, err := BuildDAG(tests)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cyclic dependency")
}

func TestBuildDAG_SelfDependency(t *testing.T) {
	tests := []TestDependency{
		{Name: "A", DependsOn: []string{"A"}},
	}

	_, err := BuildDAG(tests)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cyclic dependency")
}

func TestBuildDAG_MissingDependency(t *testing.T) {
	tests := []TestDependency{
		{Name: "A", DependsOn: []string{"NonExistent"}},
	}

	_, err := BuildDAG(tests)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown dependency")
}

func TestBuildDAG_ComplexGraph(t *testing.T) {
	// Complex graph:
	//     A
	//    / \
	//   B   C
	//    \ / \
	//     D   E
	//      \ /
	//       F
	tests := []TestDependency{
		{Name: "A"},
		{Name: "B", DependsOn: []string{"A"}},
		{Name: "C", DependsOn: []string{"A"}},
		{Name: "D", DependsOn: []string{"B", "C"}},
		{Name: "E", DependsOn: []string{"C"}},
		{Name: "F", DependsOn: []string{"D", "E"}},
	}

	plan, err := BuildDAG(tests)
	require.NoError(t, err)

	// Phase 1: A
	// Phase 2: B, C
	// Phase 3: D, E
	// Phase 4: F
	require.Len(t, plan.Phases, 4)
	assert.Equal(t, []string{"A"}, plan.Phases[0])
	assert.ElementsMatch(t, []string{"B", "C"}, plan.Phases[1])
	assert.ElementsMatch(t, []string{"D", "E"}, plan.Phases[2])
	assert.Equal(t, []string{"F"}, plan.Phases[3])
}

func TestBuildDAG_EmptyTests(t *testing.T) {
	tests := []TestDependency{}

	plan, err := BuildDAG(tests)
	require.NoError(t, err)
	assert.Empty(t, plan.Phases)
}

// =============================================================================
// Integration Test
// =============================================================================

func TestVariablesIntegration(t *testing.T) {
	// Simulate a login -> get profile -> update profile flow
	s := NewStore()
	e := NewExtractor(s)
	sub := NewSubstitutor(s)

	// Set initial variables (from config)
	s.Set("username", "testuser")
	s.Set("password", "secret123")

	// Simulate login response
	loginBody := []byte(`{"token": "jwt-abc123", "user_id": 42}`)
	e.Extract([]models.ExtractionRule{
		{Name: "auth_token", Source: "body", Path: "token"},
		{Name: "user_id", Source: "body", Path: "user_id"},
	}, loginBody, nil, 200)

	// Build next request using extracted variables
	profileURL := sub.Substitute("/users/${user_id}")
	authHeader := sub.Substitute("Bearer ${auth_token}")

	assert.Equal(t, "/users/42", profileURL)
	assert.Equal(t, "Bearer jwt-abc123", authHeader)

	// Build update request body
	updateBody := sub.SubstituteBody(map[string]interface{}{
		"user_id": "${user_id}",
		"name":    "Updated Name",
	})

	bodyMap, ok := updateBody.(map[string]interface{})
	require.True(t, ok)
	// user_id preserves its original type (int from JSON extraction for whole numbers)
	assert.Equal(t, 42, bodyMap["user_id"])
	assert.Equal(t, "Updated Name", bodyMap["name"])
}
