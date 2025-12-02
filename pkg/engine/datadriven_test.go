package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Data-Driven Testing Tests
// =============================================================================

func TestEngine_DataDriven_InlineData(t *testing.T) {
	var receivedBodies []map[string]interface{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		receivedBodies = append(receivedBodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Inline Data Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Create Users",
				Method:         "POST",
				Path:           "/users",
				ExpectedStatus: []int{201},
				Data: []map[string]interface{}{
					{"username": "alice", "email": "alice@test.com"},
					{"username": "bob", "email": "bob@test.com"},
					{"username": "charlie", "email": "charlie@test.com"},
				},
				Body: map[string]interface{}{
					"username": "${data.username}",
					"email":    "${data.email}",
				},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	// Should run 3 times (one per data row)
	assert.Equal(t, 3, summary.TotalRequests)
	assert.Equal(t, 3, summary.SuccessfulReqs)

	// Verify each body was correctly substituted
	require.Len(t, receivedBodies, 3)
	usernames := []string{}
	for _, body := range receivedBodies {
		usernames = append(usernames, body["username"].(string))
	}
	assert.ElementsMatch(t, []string{"alice", "bob", "charlie"}, usernames)
}

func TestEngine_DataDriven_InURL(t *testing.T) {
	var receivedPaths []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedPaths = append(receivedPaths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Data in URL Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Get Users",
				Method:         "GET",
				Path:           "/users/${data.id}",
				ExpectedStatus: []int{200},
				Data: []map[string]interface{}{
					{"id": 1},
					{"id": 2},
					{"id": 3},
				},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.TotalRequests)
	assert.Equal(t, 3, summary.SuccessfulReqs)

	// Verify paths
	require.Len(t, receivedPaths, 3)
	assert.ElementsMatch(t, []string{"/users/1", "/users/2", "/users/3"}, receivedPaths)
}

func TestEngine_DataDriven_InHeaders(t *testing.T) {
	var receivedTokens []string
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedTokens = append(receivedTokens, r.Header.Get("Authorization"))
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Data in Headers Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:   "Auth Request",
				Method: "GET",
				Path:   "/protected",
				Headers: map[string]string{
					"Authorization": "Bearer ${data.token}",
				},
				ExpectedStatus: []int{200},
				Data: []map[string]interface{}{
					{"token": "token-user-1"},
					{"token": "token-user-2"},
				},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 2, summary.TotalRequests)
	assert.Equal(t, 2, summary.SuccessfulReqs)

	require.Len(t, receivedTokens, 2)
	assert.ElementsMatch(t, []string{"Bearer token-user-1", "Bearer token-user-2"}, receivedTokens)
}

func TestEngine_DataDriven_FromJSONFile(t *testing.T) {
	var receivedBodies []map[string]interface{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		receivedBodies = append(receivedBodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Create a temporary JSON data file
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "users.json")
	dataContent := `[
		{"username": "from_file_1", "email": "user1@file.com"},
		{"username": "from_file_2", "email": "user2@file.com"}
	]`
	err := os.WriteFile(dataFile, []byte(dataContent), 0644)
	require.NoError(t, err)

	config := &models.Config{
		Name: "JSON File Data Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Create Users from File",
				Method:         "POST",
				Path:           "/users",
				ExpectedStatus: []int{201},
				DataFile:       dataFile,
				Body: map[string]interface{}{
					"username": "${data.username}",
					"email":    "${data.email}",
				},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 2, summary.TotalRequests)
	assert.Equal(t, 2, summary.SuccessfulReqs)

	require.Len(t, receivedBodies, 2)
	assert.Equal(t, "from_file_1", receivedBodies[0]["username"])
	assert.Equal(t, "from_file_2", receivedBodies[1]["username"])
}

func TestEngine_DataDriven_FromCSVFile(t *testing.T) {
	var receivedBodies []map[string]interface{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		receivedBodies = append(receivedBodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Create a temporary CSV data file
	tmpDir := t.TempDir()
	dataFile := filepath.Join(tmpDir, "users.csv")
	csvContent := `username,email,age
csv_user_1,csv1@test.com,25
csv_user_2,csv2@test.com,30
csv_user_3,csv3@test.com,35`
	err := os.WriteFile(dataFile, []byte(csvContent), 0644)
	require.NoError(t, err)

	config := &models.Config{
		Name: "CSV File Data Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Create Users from CSV",
				Method:         "POST",
				Path:           "/users",
				ExpectedStatus: []int{201},
				DataFile:       dataFile,
				Body: map[string]interface{}{
					"username": "${data.username}",
					"email":    "${data.email}",
					"age":      "${data.age}",
				},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 3, summary.TotalRequests)
	assert.Equal(t, 3, summary.SuccessfulReqs)

	require.Len(t, receivedBodies, 3)
	usernames := []string{}
	for _, body := range receivedBodies {
		usernames = append(usernames, body["username"].(string))
	}
	assert.ElementsMatch(t, []string{"csv_user_1", "csv_user_2", "csv_user_3"}, usernames)
}

func TestEngine_DataDriven_WithIterations(t *testing.T) {
	var requestCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Data with Iterations Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 2, // Each data row runs 2 times
		},
		Tests: []models.TestCase{
			{
				Name:           "Test",
				Method:         "GET",
				Path:           "/test/${data.id}",
				ExpectedStatus: []int{200},
				Data: []map[string]interface{}{
					{"id": 1},
					{"id": 2},
				},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	// 2 data rows * 2 iterations = 4 total requests
	assert.Equal(t, 4, summary.TotalRequests)
	assert.Equal(t, 4, summary.SuccessfulReqs)
	assert.Equal(t, 4, requestCount)
}

func TestEngine_DataDriven_NoData(t *testing.T) {
	var requestCount int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "No Data Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 3,
		},
		Tests: []models.TestCase{
			{
				Name:           "Normal Test",
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
				// No Data field - should run normally with iterations
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	// Regular iteration-based test
	assert.Equal(t, 3, summary.TotalRequests)
	assert.Equal(t, 3, summary.SuccessfulReqs)
	assert.Equal(t, 3, requestCount)
}

func TestEngine_DataDriven_NestedData(t *testing.T) {
	var receivedBodies []map[string]interface{}
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		receivedBodies = append(receivedBodies, body)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	config := &models.Config{
		Name: "Nested Data Test",
		Global: models.GlobalConfig{
			BaseURL:    server.URL,
			Timeout:    5 * time.Second,
			Iterations: 1,
		},
		Tests: []models.TestCase{
			{
				Name:           "Create with Nested",
				Method:         "POST",
				Path:           "/items",
				ExpectedStatus: []int{201},
				Data: []map[string]interface{}{
					{
						"name": "Item 1",
						"meta": map[string]interface{}{
							"category": "books",
							"price":    29.99,
						},
					},
				},
				Body: map[string]interface{}{
					"name":     "${data.name}",
					"category": "${data.meta.category}",
				},
			},
		},
	}

	engine := New(1, nil, false)
	summary := engine.Run(config)

	assert.Equal(t, 1, summary.TotalRequests)
	assert.Equal(t, 1, summary.SuccessfulReqs)

	require.Len(t, receivedBodies, 1)
	assert.Equal(t, "Item 1", receivedBodies[0]["name"])
	assert.Equal(t, "books", receivedBodies[0]["category"])
}
