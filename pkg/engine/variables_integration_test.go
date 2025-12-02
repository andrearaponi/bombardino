package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Variable Substitution Tests
// =============================================================================

func TestEngine_VariableSubstitution_InURL(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Variable Substitution Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
			Variables: map[string]interface{}{
				"user_id": "123",
			},
		},
		Tests: []models.TestCase{
			{
				Name:           "Get User",
				Method:         "GET",
				Path:           "/users/${user_id}",
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 1, summary.SuccessfulReqs)
	assert.Equal(t, "/users/123", receivedPath)
}

func TestEngine_VariableSubstitution_InHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Header Substitution Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
			Variables: map[string]interface{}{
				"token": "secret-jwt-token",
			},
		},
		Tests: []models.TestCase{
			{
				Name:   "Auth Request",
				Method: "GET",
				Path:   "/protected",
				Headers: map[string]string{
					"Authorization": "Bearer ${token}",
				},
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 1, summary.SuccessfulReqs)
	assert.Equal(t, "Bearer secret-jwt-token", receivedAuth)
}

func TestEngine_VariableSubstitution_InBody(t *testing.T) {
	var receivedBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Body Substitution Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
			Variables: map[string]interface{}{
				"username": "john_doe",
				"email":    "john@example.com",
			},
		},
		Tests: []models.TestCase{
			{
				Name:   "Create User",
				Method: "POST",
				Path:   "/users",
				Body: map[string]interface{}{
					"username": "${username}",
					"email":    "${email}",
					"active":   true,
				},
				ExpectedStatus: []int{201},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 1, summary.SuccessfulReqs)
	assert.Equal(t, "john_doe", receivedBody["username"])
	assert.Equal(t, "john@example.com", receivedBody["email"])
	assert.Equal(t, true, receivedBody["active"])
}

// =============================================================================
// Variable Extraction Tests
// =============================================================================

func TestEngine_VariableExtraction_FromBody(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex
	var receivedUserID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		count := requestCount
		mu.Unlock()

		if count == 1 {
			// First request: login, return token and user_id
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token": "extracted-token-xyz", "user": {"id": 42}}`))
		} else {
			// Second request: should use extracted user_id
			receivedUserID = r.URL.Path
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"name": "John"}`))
		}
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Extraction Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Login",
				Method:         "POST",
				Path:           "/auth/login",
				ExpectedStatus: []int{200},
				Extract: []models.ExtractionRule{
					{Name: "auth_token", Source: "body", Path: "token"},
					{Name: "user_id", Source: "body", Path: "user.id"},
				},
			},
			{
				Name:           "Get Profile",
				Method:         "GET",
				Path:           "/users/${user_id}",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Login"},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 2, summary.TotalRequests)
	assert.Equal(t, 2, summary.SuccessfulReqs)
	assert.Equal(t, "/users/42", receivedUserID)
}

func TestEngine_VariableExtraction_FromHeader(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex
	var receivedRequestID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		count := requestCount
		mu.Unlock()

		if count == 1 {
			w.Header().Set("X-Request-Id", "req-abc-123")
			w.WriteHeader(http.StatusOK)
		} else {
			receivedRequestID = r.Header.Get("X-Correlation-Id")
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Header Extraction Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "First Request",
				Method:         "GET",
				Path:           "/first",
				ExpectedStatus: []int{200},
				Extract: []models.ExtractionRule{
					{Name: "request_id", Source: "header", Path: "X-Request-Id"},
				},
			},
			{
				Name:   "Second Request",
				Method: "GET",
				Path:   "/second",
				Headers: map[string]string{
					"X-Correlation-Id": "${request_id}",
				},
				ExpectedStatus: []int{200},
				DependsOn:      []string{"First Request"},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 2, summary.SuccessfulReqs)
	assert.Equal(t, "req-abc-123", receivedRequestID)
}

// =============================================================================
// DAG Execution Tests
// =============================================================================

func TestEngine_DAG_LinearDependencies(t *testing.T) {
	var executionOrder []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		executionOrder = append(executionOrder, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Linear DAG Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Step 1",
				Method:         "GET",
				Path:           "/step1",
				ExpectedStatus: []int{200},
			},
			{
				Name:           "Step 2",
				Method:         "GET",
				Path:           "/step2",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Step 1"},
			},
			{
				Name:           "Step 3",
				Method:         "GET",
				Path:           "/step3",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Step 2"},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.SuccessfulReqs)
	// Verify execution order
	require.Len(t, executionOrder, 3)
	assert.Equal(t, "/step1", executionOrder[0])
	assert.Equal(t, "/step2", executionOrder[1])
	assert.Equal(t, "/step3", executionOrder[2])
}

func TestEngine_DAG_ParallelExecution(t *testing.T) {
	var executionTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		executionTimes = append(executionTimes, time.Now())
		mu.Unlock()
		// Small delay to make timing measurable
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Parallel DAG Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Login",
				Method:         "GET",
				Path:           "/login",
				ExpectedStatus: []int{200},
			},
			{
				Name:           "Get Profile",
				Method:         "GET",
				Path:           "/profile",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Login"},
			},
			{
				Name:           "Get Settings",
				Method:         "GET",
				Path:           "/settings",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Login"},
			},
		},
	}

	engine := New(2, nil, false) // 2 workers for parallel execution
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.SuccessfulReqs)
	// Get Profile and Get Settings should start around the same time
	// (after Login completes)
	require.Len(t, executionTimes, 3)
}

func TestEngine_DAG_NoDependencies_AllParallel(t *testing.T) {
	var requestPaths []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestPaths = append(requestPaths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "No Dependencies Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Test A",
				Method:         "GET",
				Path:           "/a",
				ExpectedStatus: []int{200},
			},
			{
				Name:           "Test B",
				Method:         "GET",
				Path:           "/b",
				ExpectedStatus: []int{200},
			},
			{
				Name:           "Test C",
				Method:         "GET",
				Path:           "/c",
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(3, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.SuccessfulReqs)
	assert.Len(t, requestPaths, 3)
	// All paths should be present (order may vary due to parallel execution)
	assert.Contains(t, requestPaths, "/a")
	assert.Contains(t, requestPaths, "/b")
	assert.Contains(t, requestPaths, "/c")
}

// =============================================================================
// Complex Flow Tests
// =============================================================================

func TestEngine_CompleteAuthFlow(t *testing.T) {
	var loginReceived, profileReceived, updateReceived bool
	var profileAuth, updateAuth string
	var updateBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/login":
			loginReceived = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"token": "jwt-secret-123", "user_id": 999}`))

		case "/users/999":
			profileReceived = true
			profileAuth = r.Header.Get("Authorization")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 999, "name": "John", "email": "john@test.com"}`))

		case "/users/999/update":
			updateReceived = true
			updateAuth = r.Header.Get("Authorization")
			json.NewDecoder(r.Body).Decode(&updateBody)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success": true}`))
		}
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Complete Auth Flow",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
			Variables: map[string]interface{}{
				"username": "testuser",
				"password": "secret123",
			},
		},
		Tests: []models.TestCase{
			{
				Name:   "Login",
				Method: "POST",
				Path:   "/auth/login",
				Body: map[string]interface{}{
					"username": "${username}",
					"password": "${password}",
				},
				ExpectedStatus: []int{200},
				Extract: []models.ExtractionRule{
					{Name: "auth_token", Source: "body", Path: "token"},
					{Name: "user_id", Source: "body", Path: "user_id"},
				},
			},
			{
				Name:   "Get Profile",
				Method: "GET",
				Path:   "/users/${user_id}",
				Headers: map[string]string{
					"Authorization": "Bearer ${auth_token}",
				},
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Login"},
				Extract: []models.ExtractionRule{
					{Name: "user_email", Source: "body", Path: "email"},
				},
			},
			{
				Name:   "Update Profile",
				Method: "PUT",
				Path:   "/users/${user_id}/update",
				Headers: map[string]string{
					"Authorization": "Bearer ${auth_token}",
				},
				Body: map[string]interface{}{
					"email": "${user_email}",
					"name":  "Updated Name",
				},
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Get Profile"},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	// Verify all requests were made
	assert.True(t, loginReceived, "Login should be called")
	assert.True(t, profileReceived, "Get Profile should be called")
	assert.True(t, updateReceived, "Update Profile should be called")

	// Verify authentication was properly extracted and used
	assert.Equal(t, "Bearer jwt-secret-123", profileAuth)
	assert.Equal(t, "Bearer jwt-secret-123", updateAuth)

	// Verify body substitution worked
	assert.Equal(t, "john@test.com", updateBody["email"])
	assert.Equal(t, "Updated Name", updateBody["name"])

	// Verify summary
	assert.Equal(t, 3, summary.TotalRequests)
	assert.Equal(t, 3, summary.SuccessfulReqs)
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestEngine_DAG_CyclicDependency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Cyclic Dependency Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "A",
				Method:         "GET",
				Path:           "/a",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"C"},
			},
			{
				Name:           "B",
				Method:         "GET",
				Path:           "/b",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"A"},
			},
			{
				Name:           "C",
				Method:         "GET",
				Path:           "/c",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"B"},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	// Should fail due to cyclic dependency
	assert.Equal(t, 0, summary.SuccessfulReqs)
	assert.True(t, len(summary.Errors) > 0, "Should have errors for cyclic dependency")
}

func TestEngine_MissingVariable_StaysAsIs(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Missing Variable Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
			// No variables defined
		},
		Tests: []models.TestCase{
			{
				Name:           "Test",
				Method:         "GET",
				Path:           "/users/${missing_var}",
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(1, nil, false)
	engine.Run(config)

	// Missing variable should stay as-is
	assert.Equal(t, "/users/${missing_var}", receivedPath)
}
