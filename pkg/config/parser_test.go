package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile_ValidConfig(t *testing.T) {
	configContent := `{
		"name": "Test Config",
		"description": "Test description",
		"global": {
			"base_url": "https://api.example.com",
			"timeout": "30s",
			"delay": "100ms",
			"iterations": 10,
			"headers": {
				"Authorization": "Bearer token123",
				"Content-Type": "application/json"
			}
		},
		"tests": [
			{
				"name": "Get users",
				"method": "GET",
				"path": "/users",
				"expected_status": [200, 201],
				"timeout": "5s",
				"delay": "50ms",
				"iterations": 5,
				"assertions": [
					{
						"type": "response_time",
						"operator": "lt",
						"value": "1s"
					}
				]
			}
		]
	}`

	tmpFile := createTempFile(t, configContent)
	defer os.Remove(tmpFile)

	config, err := LoadFromFile(tmpFile)
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "Test Config", config.Name)
	assert.Equal(t, "Test description", config.Description)

	assert.Equal(t, "https://api.example.com", config.Global.BaseURL)
	assert.Equal(t, 30*time.Second, config.Global.Timeout)
	assert.Equal(t, 100*time.Millisecond, config.Global.Delay)
	assert.Equal(t, 10, config.Global.Iterations)
	assert.Equal(t, "Bearer token123", config.Global.Headers["Authorization"])

	require.Len(t, config.Tests, 1)
	test := config.Tests[0]
	assert.Equal(t, "Get users", test.Name)
	assert.Equal(t, "GET", test.Method)
	assert.Equal(t, "/users", test.Path)
	assert.Equal(t, []int{200, 201}, test.ExpectedStatus)
	assert.Equal(t, 5*time.Second, test.Timeout)
	assert.Equal(t, 50*time.Millisecond, test.Delay)
	assert.Equal(t, 5, test.Iterations)

	require.Len(t, test.Assertions, 1)
	assertion := test.Assertions[0]
	assert.Equal(t, "response_time", assertion.Type)
	assert.Equal(t, "lt", assertion.Operator)
	assert.Equal(t, "1s", assertion.Value)
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	configContent := `{
		"name": "Invalid JSON"
		"missing_comma": true
	}`

	tmpFile := createTempFile(t, configContent)
	defer os.Remove(tmpFile)

	config, err := LoadFromFile(tmpFile)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	config, err := LoadFromFile("nonexistent-file.json")
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadFromFile_InvalidTimeout(t *testing.T) {
	configContent := `{
		"name": "Invalid Timeout",
		"global": {
			"base_url": "https://api.example.com",
			"timeout": "invalid-duration",
			"delay": "100ms",
			"iterations": 10
		},
		"tests": []
	}`

	tmpFile := createTempFile(t, configContent)
	defer os.Remove(tmpFile)

	config, err := LoadFromFile(tmpFile)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "invalid global timeout")
}

func TestLoadFromFile_InvalidDelay(t *testing.T) {
	configContent := `{
		"name": "Invalid Delay",
		"global": {
			"base_url": "https://api.example.com",
			"timeout": "30s",
			"delay": "invalid-duration",
			"iterations": 10
		},
		"tests": []
	}`

	tmpFile := createTempFile(t, configContent)
	defer os.Remove(tmpFile)

	config, err := LoadFromFile(tmpFile)
	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "invalid global delay")
}

func TestValidateConfig_MissingName(t *testing.T) {
	config := &models.Config{
		Global: models.GlobalConfig{
			BaseURL:    "https://api.example.com",
			Iterations: 10,
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

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config name is required")
}

func TestValidateConfig_MissingBaseURL(t *testing.T) {
	config := &models.Config{
		Name: "Test Config",
		Global: models.GlobalConfig{
			Iterations: 10,
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

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global base_url is required")
}

func TestValidateConfig_InvalidIterations(t *testing.T) {
	config := &models.Config{
		Name: "Test Config",
		Global: models.GlobalConfig{
			BaseURL:    "https://api.example.com",
			Iterations: 0,
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

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "global iterations must be greater than 0")
}

func TestValidateConfig_NoTests(t *testing.T) {
	config := &models.Config{
		Name: "Test Config",
		Global: models.GlobalConfig{
			BaseURL:    "https://api.example.com",
			Iterations: 10,
		},
		Tests: []models.TestCase{},
	}

	err := validateConfig(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one test case is required")
}

func TestValidateConfig_InvalidTestCase(t *testing.T) {
	tests := []struct {
		name        string
		testCase    models.TestCase
		expectedErr string
	}{
		{
			name: "missing name",
			testCase: models.TestCase{
				Method:         "GET",
				Path:           "/test",
				ExpectedStatus: []int{200},
			},
			expectedErr: "name is required",
		},
		{
			name: "missing method",
			testCase: models.TestCase{
				Name:           "Test",
				Path:           "/test",
				ExpectedStatus: []int{200},
			},
			expectedErr: "method is required",
		},
		{
			name: "missing path",
			testCase: models.TestCase{
				Name:           "Test",
				Method:         "GET",
				ExpectedStatus: []int{200},
			},
			expectedErr: "path is required",
		},
		{
			name: "missing expected status",
			testCase: models.TestCase{
				Name:   "Test",
				Method: "GET",
				Path:   "/test",
			},
			expectedErr: "at least one expected status is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &models.Config{
				Name: "Test Config",
				Global: models.GlobalConfig{
					BaseURL:    "https://api.example.com",
					Iterations: 10,
				},
				Tests: []models.TestCase{tt.testCase},
			}

			err := validateConfig(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestGetTotalRequests(t *testing.T) {
	config := &models.Config{
		Global: models.GlobalConfig{
			Iterations: 10,
		},
		Tests: []models.TestCase{
			{
				Name:           "Test1",
				Method:         "GET",
				Path:           "/test1",
				ExpectedStatus: []int{200},
			},
			{
				Name:           "Test2",
				Method:         "GET",
				Path:           "/test2",
				ExpectedStatus: []int{200},
				Iterations:     5,
			},
			{
				Name:           "Test3",
				Method:         "GET",
				Path:           "/test3",
				ExpectedStatus: []int{200},
				Iterations:     20,
			},
		},
	}

	total := config.GetTotalRequests()
	assert.Equal(t, 35, total) // 10 + 5 + 20
}

func createTempFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")

	err := os.WriteFile(tmpFile, []byte(content), 0644)
	require.NoError(t, err)

	return tmpFile
}
