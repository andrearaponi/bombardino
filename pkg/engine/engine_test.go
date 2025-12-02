package engine

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/andrearaponi/bombardino/pkg/progress"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_Run_SimpleGET(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "success"}`))
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Test Config",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Delay:      10 * time.Millisecond,
			Iterations: 2,
		},
		Tests: []models.TestCase{
			{
				Name:           "Simple GET test",
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
				Iterations:     1,
			},
		},
	}

	progressBar := progress.New(config.GetTotalRequests())
	engine := New(2, progressBar, false)

	summary := engine.Run(config)

	assert.Equal(t, 1, summary.TotalRequests)
	assert.Equal(t, 1, summary.SuccessfulReqs)
	assert.Equal(t, 0, summary.FailedReqs)
	assert.Equal(t, 1, summary.StatusCodes[200])
	assert.True(t, summary.AvgResponseTime > 0)
	assert.True(t, summary.RequestsPerSec > 0)
}

func TestEngine_Run_MultiplePOST(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/users", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(fmt.Sprintf(`{"id": %d, "created": true}`, requestCount)))
	}))
	defer server.Close()

	config := &models.Config{
		Name: "POST Test Config",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Delay:      5 * time.Millisecond,
			Iterations: 3,
		},
		Tests: []models.TestCase{
			{
				Name:   "Create user",
				Method: "POST",
				Path:   "/api/users",
				Body: map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
				},
				ExpectedStatus: []int{201},
			},
		},
	}

	progressBar := progress.New(config.GetTotalRequests())
	engine := New(1, progressBar, false)

	summary := engine.Run(config)

	assert.Equal(t, 3, summary.TotalRequests)
	assert.Equal(t, 3, summary.SuccessfulReqs)
	assert.Equal(t, 0, summary.FailedReqs)
	assert.Equal(t, 3, summary.StatusCodes[201])
	assert.Equal(t, 3, requestCount)
}

func TestEngine_Run_UnexpectedStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Error Test Config",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Delay:      1 * time.Millisecond,
			Iterations: 2,
		},
		Tests: []models.TestCase{
			{
				Name:           "Failing test",
				Method:         "GET",
				Path:           "/error",
				ExpectedStatus: []int{200},
			},
		},
	}

	progressBar := progress.New(config.GetTotalRequests())
	engine := New(1, progressBar, false)

	summary := engine.Run(config)

	assert.Equal(t, 2, summary.TotalRequests)
	assert.Equal(t, 0, summary.SuccessfulReqs)
	assert.Equal(t, 2, summary.FailedReqs)
	assert.Equal(t, 2, summary.StatusCodes[500])
	assert.Contains(t, summary.Errors, "Unexpected status code: 500 (expected: [200])")
}

func TestEngine_Run_WithCustomHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bombardino/1.0", r.Header.Get("User-Agent"))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"authenticated": true}`))
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Headers Test Config",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Delay:      1 * time.Millisecond,
			Iterations: 1,
			Headers: map[string]string{
				"User-Agent": "Bombardino/1.0",
			},
		},
		Tests: []models.TestCase{
			{
				Name:   "Authenticated request",
				Method: "GET",
				Path:   "/protected",
				Headers: map[string]string{
					"Authorization": "Bearer token123",
					"Content-Type":  "application/json",
				},
				ExpectedStatus: []int{200},
			},
		},
	}

	progressBar := progress.New(config.GetTotalRequests())
	engine := New(1, progressBar, false)

	summary := engine.Run(config)

	assert.Equal(t, 1, summary.TotalRequests)
	assert.Equal(t, 1, summary.SuccessfulReqs)
	assert.Equal(t, 0, summary.FailedReqs)
}

func TestEngine_Run_MultipleTests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
		case "/posts":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{"id": 1, "title": "Post 1"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Multiple Tests Config",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Delay:      1 * time.Millisecond,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Get users",
				Method:         "GET",
				Path:           "/users",
				ExpectedStatus: []int{200},
				Iterations:     2,
			},
			{
				Name:           "Get posts",
				Method:         "GET",
				Path:           "/posts",
				ExpectedStatus: []int{200},
				Iterations:     3,
			},
		},
	}

	progressBar := progress.New(config.GetTotalRequests())
	engine := New(2, progressBar, false)

	summary := engine.Run(config)

	assert.Equal(t, 5, summary.TotalRequests) // 2 + 3
	assert.Equal(t, 5, summary.SuccessfulReqs)
	assert.Equal(t, 0, summary.FailedReqs)
	assert.Equal(t, 5, summary.StatusCodes[200])
}

func TestEngine_Run_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // Simulate slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Timeout Test Config",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    100 * time.Millisecond, // Short timeout
			Delay:      1 * time.Millisecond,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Slow endpoint",
				Method:         "GET",
				Path:           "/slow",
				ExpectedStatus: []int{200},
			},
		},
	}

	progressBar := progress.New(config.GetTotalRequests())
	engine := New(1, progressBar, false)

	summary := engine.Run(config)

	assert.Equal(t, 1, summary.TotalRequests)
	assert.Equal(t, 0, summary.SuccessfulReqs)
	assert.Equal(t, 1, summary.FailedReqs)
	assert.True(t, len(summary.Errors) > 0)
}

func TestEngine_isExpectedStatus(t *testing.T) {
	engine := &Engine{}

	tests := []struct {
		name           string
		statusCode     int
		expectedStatus []int
		expected       bool
	}{
		{
			name:           "exact match",
			statusCode:     200,
			expectedStatus: []int{200},
			expected:       true,
		},
		{
			name:           "multiple expected - match first",
			statusCode:     200,
			expectedStatus: []int{200, 201, 202},
			expected:       true,
		},
		{
			name:           "multiple expected - match middle",
			statusCode:     201,
			expectedStatus: []int{200, 201, 202},
			expected:       true,
		},
		{
			name:           "multiple expected - match last",
			statusCode:     202,
			expectedStatus: []int{200, 201, 202},
			expected:       true,
		},
		{
			name:           "no match",
			statusCode:     404,
			expectedStatus: []int{200, 201, 202},
			expected:       false,
		},
		{
			name:           "empty expected status",
			statusCode:     200,
			expectedStatus: []int{},
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.isExpectedStatus(tt.statusCode, tt.expectedStatus)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEngine_createRequest_WithBody(t *testing.T) {
	engine := &Engine{}

	job := Job{
		Config: &models.Config{
			Global: models.GlobalConfig{
				Headers: map[string]string{
					"User-Agent": "Bombardino/1.0",
				},
			},
		},
		TestCase: models.TestCase{
			Method: "POST",
			Headers: map[string]string{
				"Authorization": "Bearer token123",
			},
			Body: map[string]interface{}{
				"name":  "John",
				"email": "john@example.com",
			},
		},
		URL: "https://api.example.com/users",
	}

	req, err := engine.createRequest(job)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "https://api.example.com/users", req.URL.String())
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.Equal(t, "Bombardino/1.0", req.Header.Get("User-Agent"))
	assert.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
	assert.NotNil(t, req.Body)
}

func TestEngine_createRequest_WithoutBody(t *testing.T) {
	engine := &Engine{}

	job := Job{
		Config: &models.Config{
			Global: models.GlobalConfig{
				Headers: map[string]string{
					"Accept": "application/json",
				},
			},
		},
		TestCase: models.TestCase{
			Method: "GET",
		},
		URL: "https://api.example.com/users",
	}

	req, err := engine.createRequest(job)
	require.NoError(t, err)
	require.NotNil(t, req)

	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "https://api.example.com/users", req.URL.String())
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
	assert.Equal(t, "", req.Header.Get("Content-Type"))
	assert.Nil(t, req.Body)
}
