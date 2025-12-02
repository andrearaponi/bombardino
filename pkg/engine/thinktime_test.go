package engine

import (
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
// Think Time Tests
// =============================================================================

func TestEngine_ThinkTime_Fixed(t *testing.T) {
	var requestTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Fixed Think Time Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 3,
			ThinkTime:  100 * time.Millisecond, // Fixed think time
		},
		Tests: []models.TestCase{
			{
				Name:           "Test",
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(1, nil, false) // Single worker for predictable timing
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.SuccessfulReqs)
	require.Len(t, requestTimes, 3)

	// Verify think time is applied between requests
	for i := 1; i < len(requestTimes); i++ {
		gap := requestTimes[i].Sub(requestTimes[i-1])
		// Allow some tolerance (think time should be at least 80ms)
		assert.True(t, gap >= 80*time.Millisecond,
			"Gap between requests should be at least 80ms, got %v", gap)
	}
}

func TestEngine_ThinkTime_Random(t *testing.T) {
	var requestTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Random Think Time Test",
		Global: models.GlobalConfig{
			BaseURL:        server.URL,
			Timeout:        5 * time.Second,
			Iterations:     5,
			ThinkTimeMin:   50 * time.Millisecond,
			ThinkTimeMax:   150 * time.Millisecond,
		},
		Tests: []models.TestCase{
			{
				Name:           "Test",
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 5, summary.SuccessfulReqs)
	require.Len(t, requestTimes, 5)

	// Verify think time is within range
	for i := 1; i < len(requestTimes); i++ {
		gap := requestTimes[i].Sub(requestTimes[i-1])
		// Allow tolerance for request execution time
		assert.True(t, gap >= 40*time.Millisecond,
			"Gap should be at least 40ms, got %v", gap)
		assert.True(t, gap <= 300*time.Millisecond,
			"Gap should be at most 300ms, got %v", gap)
	}
}

func TestEngine_ThinkTime_TestLevelOverride(t *testing.T) {
	var requestTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Think Time Override Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 3,
			ThinkTime:  10 * time.Millisecond, // Global think time
		},
		Tests: []models.TestCase{
			{
				Name:           "Test with Override",
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
				ThinkTime:      100 * time.Millisecond, // Override with longer think time
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.SuccessfulReqs)
	require.Len(t, requestTimes, 3)

	// Verify test-level think time is applied (should be ~100ms, not 10ms)
	for i := 1; i < len(requestTimes); i++ {
		gap := requestTimes[i].Sub(requestTimes[i-1])
		assert.True(t, gap >= 80*time.Millisecond,
			"Gap should be at least 80ms (test override), got %v", gap)
	}
}

func TestEngine_ThinkTime_ZeroMeansNone(t *testing.T) {
	var requestTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "No Think Time Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 3,
			// No think time configured
		},
		Tests: []models.TestCase{
			{
				Name:           "Fast Test",
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(1, nil, false)
	startTime := time.Now()
	summary := engine.Run(config)
	totalTime := time.Since(startTime)

	assert.Equal(t, 3, summary.SuccessfulReqs)
	// Without think time, requests should complete very quickly
	assert.True(t, totalTime < 500*time.Millisecond,
		"Without think time, 3 requests should complete in < 500ms, got %v", totalTime)
}

func TestEngine_ThinkTime_WithDelay(t *testing.T) {
	// Test that think time and delay work together
	// Delay is applied after each request
	// Think time is applied before each request (simulating user "thinking")
	var requestTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Think Time with Delay Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 3,
			Delay:      50 * time.Millisecond,  // Delay after each request
			ThinkTime:  50 * time.Millisecond,  // Think time before each request
		},
		Tests: []models.TestCase{
			{
				Name:           "Test",
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.SuccessfulReqs)
	require.Len(t, requestTimes, 3)

	// Each gap should be at least delay + think time (~100ms)
	for i := 1; i < len(requestTimes); i++ {
		gap := requestTimes[i].Sub(requestTimes[i-1])
		assert.True(t, gap >= 80*time.Millisecond,
			"Gap should be at least 80ms (delay + think time), got %v", gap)
	}
}

func TestEngine_ThinkTime_DAGMode(t *testing.T) {
	// Think time should also work in DAG mode
	var requestTimes []time.Time
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestTimes = append(requestTimes, time.Now())
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"token": "abc"}`))
	}))
	defer server.Close()

	config := &models.Config{
		Name: "DAG with Think Time",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
			ThinkTime:  100 * time.Millisecond,
		},
		Tests: []models.TestCase{
			{
				Name:           "Login",
				Method:         "POST",
				Path:           "/login",
				ExpectedStatus: []int{200},
				Extract: []models.ExtractionRule{
					{Name: "token", Source: "body", Path: "token"},
				},
			},
			{
				Name:           "GetData",
				Method:         "GET",
				Path:           "/data",
				ExpectedStatus: []int{200},
				DependsOn:      []string{"Login"},
			},
		},
	}

	engine := New(1, nil, false)
	startTime := time.Now()
	summary := engine.Run(config)
	totalTime := time.Since(startTime)

	assert.Equal(t, 2, summary.SuccessfulReqs)
	// With think time, the second request should wait ~100ms
	// Total time should be at least 100ms (think time before second request)
	assert.True(t, totalTime >= 80*time.Millisecond,
		"Total time should include think time, got %v", totalTime)
}
