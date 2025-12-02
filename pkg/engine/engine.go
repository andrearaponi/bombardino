package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/andrearaponi/bombardino/pkg/assertion"
	"github.com/andrearaponi/bombardino/pkg/progress"
	"github.com/andrearaponi/bombardino/pkg/variables"
	"github.com/google/uuid"
)

type Engine struct {
	workers            int
	progressBar        *progress.ProgressBar
	verbose            bool
	logChan            chan models.DebugLog
	debugLogs          []models.DebugLog
	logMutex           sync.Mutex
	assertionEvaluator *assertion.Evaluator
	varStore           *variables.Store
	varExtractor       *variables.Extractor
	varSubstitutor     *variables.Substitutor
}

func New(workers int, progressBar *progress.ProgressBar, verbose bool) *Engine {
	varStore := variables.NewStore()
	e := &Engine{
		workers:            workers,
		progressBar:        progressBar,
		verbose:            verbose,
		assertionEvaluator: assertion.New(verbose),
		varStore:           varStore,
		varExtractor:       variables.NewExtractor(varStore),
		varSubstitutor:     variables.NewSubstitutor(varStore),
	}
	if verbose {
		e.logChan = make(chan models.DebugLog, 100)
	}
	return e
}

func (e *Engine) Run(config *models.Config) *models.Summary {
	// Load global variables into store
	if config.Global.Variables != nil {
		e.varStore.SetFromMap(config.Global.Variables)
	}

	// Check if we need DAG-based execution (tests have dependencies)
	if e.hasDependencies(config) {
		return e.runWithDAG(config)
	}

	jobs := make(chan Job, 1000)
	results := make(chan models.TestResult, 1000)

	// Start logger goroutine if verbose mode is enabled
	if e.verbose {
		go e.logger()
	}

	// Create context with timeout for duration-based tests
	var ctx context.Context
	var cancel context.CancelFunc

	if config.IsDurationBased() || config.HasMixedMode() {
		// Find the maximum duration among all tests
		maxDuration := config.Global.Duration
		for _, test := range config.Tests {
			if test.Duration > maxDuration {
				maxDuration = test.Duration
			}
		}
		ctx, cancel = context.WithTimeout(context.Background(), maxDuration)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < e.workers; i++ {
		wg.Add(1)
		go e.worker(ctx, jobs, results, &wg)
	}

	go func() {
		defer close(jobs)
		e.generateJobs(config, jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	summary := e.collectResults(results, config.GetTotalRequests())
	if e.progressBar != nil {
		e.progressBar.Finish()
	}

	// Close log channel if verbose mode is enabled
	if e.verbose {
		close(e.logChan)
		// Give logger time to flush remaining messages
		time.Sleep(100 * time.Millisecond)
		
		// Add debug logs to summary
		e.logMutex.Lock()
		summary.DebugLogs = e.debugLogs
		e.logMutex.Unlock()
	}

	return summary
}

type Job struct {
	Config   *models.Config
	TestCase models.TestCase
	URL      string
	DataRow  map[string]interface{} // Data row for data-driven testing
}

type TestMode int

const (
	IterationMode TestMode = iota
	DurationMode
)

func (e *Engine) generateJobs(config *models.Config, jobs chan<- Job) {
	if config.HasMixedMode() {
		e.generateMixedModeJobs(config, jobs)
	} else if config.IsDurationBased() {
		e.generateDurationBasedJobs(config, jobs)
	} else {
		e.generateIterationBasedJobs(config, jobs)
	}
}

func (e *Engine) generateIterationBasedJobs(config *models.Config, jobs chan<- Job) {
	for _, test := range config.Tests {
		iterations := test.Iterations
		if iterations == 0 {
			iterations = config.Global.Iterations
		}

		baseURL := strings.TrimSuffix(config.Global.BaseURL, "/")
		testPath := strings.TrimPrefix(test.Path, "/")
		fullURL := baseURL + "/" + testPath

		// Get data rows (from inline data, file, or empty)
		dataRows := e.getDataRows(test)

		if len(dataRows) > 0 {
			// Data-driven test: run iterations for each data row
			for _, dataRow := range dataRows {
				for i := 0; i < iterations; i++ {
					jobs <- Job{
						Config:   config,
						TestCase: test,
						URL:      fullURL,
						DataRow:  dataRow,
					}
				}
			}
		} else {
			// Regular test without data
			for i := 0; i < iterations; i++ {
				jobs <- Job{
					Config:   config,
					TestCase: test,
					URL:      fullURL,
				}
			}
		}
	}
}

func (e *Engine) generateDurationBasedJobs(config *models.Config, jobs chan<- Job) {
	startTime := time.Now()

	// Create separate goroutines for each test to handle individual durations
	var wg sync.WaitGroup

	for _, test := range config.Tests {
		wg.Add(1)
		go func(testCase models.TestCase) {
			defer wg.Done()

			// Determine test duration
			testDuration := testCase.Duration
			if testDuration == 0 {
				testDuration = config.Global.Duration
			}

			endTime := startTime.Add(testDuration)

			baseURL := strings.TrimSuffix(config.Global.BaseURL, "/")
			testPath := strings.TrimPrefix(testCase.Path, "/")
			fullURL := baseURL + "/" + testPath

			// Generate jobs as fast as possible - let workers handle delays
			for time.Now().Before(endTime) {
				select {
				case jobs <- Job{
					Config:   config,
					TestCase: testCase,
					URL:      fullURL,
				}:
					// Job sent successfully
				case <-time.After(10 * time.Millisecond):
					// Prevent busy waiting if channel is full
				}
			}
		}(test)
	}

	wg.Wait()
}

func (e *Engine) generateMixedModeJobs(config *models.Config, jobs chan<- Job) {
	var wg sync.WaitGroup

	for _, test := range config.Tests {
		wg.Add(1)

		if test.Duration > 0 || (test.Duration == 0 && config.Global.Duration > 0 && test.Iterations == 0) {
			// Duration-based test
			go func(testCase models.TestCase) {
				defer wg.Done()

				testDuration := testCase.Duration
				if testDuration == 0 {
					testDuration = config.Global.Duration
				}

				endTime := time.Now().Add(testDuration)

				baseURL := strings.TrimSuffix(config.Global.BaseURL, "/")
				testPath := strings.TrimPrefix(testCase.Path, "/")
				fullURL := baseURL + "/" + testPath

				for time.Now().Before(endTime) {
					select {
					case jobs <- Job{
						Config:   config,
						TestCase: testCase,
						URL:      fullURL,
					}:
						// Job sent successfully
					case <-time.After(10 * time.Millisecond):
						// Prevent busy waiting if channel is full
					}
				}
			}(test)
		} else {
			// Iteration-based test
			go func(testCase models.TestCase) {
				defer wg.Done()

				iterations := testCase.Iterations
				if iterations == 0 {
					iterations = config.Global.Iterations
				}

				baseURL := strings.TrimSuffix(config.Global.BaseURL, "/")
				testPath := strings.TrimPrefix(testCase.Path, "/")
				fullURL := baseURL + "/" + testPath

				for i := 0; i < iterations; i++ {
					jobs <- Job{
						Config:   config,
						TestCase: testCase,
						URL:      fullURL,
					}
				}
			}(test)
		}
	}

	wg.Wait()
}

func (e *Engine) worker(ctx context.Context, jobs <-chan Job, results chan<- models.TestResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Context timeout reached, stop processing
			return
		case job, ok := <-jobs:
			if !ok {
				// Jobs channel closed, no more work
				return
			}

			// Apply think time before executing the request (simulates user thinking)
			thinkTime := e.calculateThinkTime(job)
			if thinkTime > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(thinkTime):
					// Think time completed, continue
				}
			}

			// Set data variables for data-driven tests
			if job.DataRow != nil {
				e.setDataVariables(job.DataRow)
			}

			result := e.executeTest(job)
			results <- result
			if e.progressBar != nil {
				e.progressBar.Increment()
			}

			// Apply delay after processing the job (only for workers)
			delay := job.TestCase.Delay
			if delay == 0 {
				delay = job.Config.Global.Delay
			}
			if delay > 0 {
				select {
				case <-ctx.Done():
					// Context timeout reached during delay, stop processing
					return
				case <-time.After(delay):
					// Delay completed, continue
				}
			}
		}
	}
}

// calculateThinkTime returns the think time to apply before a request
// It handles both fixed think time and random range
func (e *Engine) calculateThinkTime(job Job) time.Duration {
	// Check test-level think time first
	if job.TestCase.ThinkTime > 0 {
		return job.TestCase.ThinkTime
	}

	// Check test-level random range
	if job.TestCase.ThinkTimeMin > 0 && job.TestCase.ThinkTimeMax > 0 {
		return e.randomDuration(job.TestCase.ThinkTimeMin, job.TestCase.ThinkTimeMax)
	}

	// Fall back to global settings
	if job.Config.Global.ThinkTime > 0 {
		return job.Config.Global.ThinkTime
	}

	// Check global random range
	if job.Config.Global.ThinkTimeMin > 0 && job.Config.Global.ThinkTimeMax > 0 {
		return e.randomDuration(job.Config.Global.ThinkTimeMin, job.Config.Global.ThinkTimeMax)
	}

	return 0
}

// randomDuration returns a random duration between min and max
func (e *Engine) randomDuration(min, max time.Duration) time.Duration {
	if min >= max {
		return min
	}
	return min + time.Duration(rand.Int63n(int64(max-min)))
}

// getDataRows returns the data rows for a test (from inline data or file)
func (e *Engine) getDataRows(test models.TestCase) []map[string]interface{} {
	// First check inline data
	if len(test.Data) > 0 {
		return test.Data
	}

	// Check for data file
	if test.DataFile != "" {
		data, err := e.loadDataFromFile(test.DataFile)
		if err != nil {
			// Log error but continue - test will run without data
			if e.verbose {
				fmt.Printf("Warning: Failed to load data file %s: %v\n", test.DataFile, err)
			}
			return nil
		}
		return data
	}

	return nil
}

// loadDataFromFile loads data from a JSON or CSV file
func (e *Engine) loadDataFromFile(filePath string) ([]map[string]interface{}, error) {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".json":
		return e.loadJSONData(filePath)
	case ".csv":
		return e.loadCSVData(filePath)
	default:
		return nil, fmt.Errorf("unsupported data file format: %s", ext)
	}
}

// loadJSONData loads an array of objects from a JSON file
func (e *Engine) loadJSONData(filePath string) ([]map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return result, nil
}

// loadCSVData loads data from a CSV file (first row is header)
func (e *Engine) loadCSVData(filePath string) ([]map[string]interface{}, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must have at least a header and one data row")
	}

	// First row is the header
	headers := records[0]
	var result []map[string]interface{}

	// Convert each row to a map
	for i := 1; i < len(records); i++ {
		row := make(map[string]interface{})
		for j, header := range headers {
			if j < len(records[i]) {
				row[header] = records[i][j]
			}
		}
		result = append(result, row)
	}

	return result, nil
}

// setDataVariables sets the data row variables in the store with "data." prefix
func (e *Engine) setDataVariables(dataRow map[string]interface{}) {
	if dataRow == nil {
		return
	}

	// Set each field with "data." prefix
	for key, value := range dataRow {
		e.varStore.Set("data."+key, value)

		// Handle nested maps
		if nested, ok := value.(map[string]interface{}); ok {
			e.setNestedDataVariables("data."+key, nested)
		}
	}
}

// setNestedDataVariables recursively sets nested data variables
func (e *Engine) setNestedDataVariables(prefix string, data map[string]interface{}) {
	for key, value := range data {
		fullKey := prefix + "." + key
		e.varStore.Set(fullKey, value)

		// Handle nested maps
		if nested, ok := value.(map[string]interface{}); ok {
			e.setNestedDataVariables(fullKey, nested)
		}
	}
}

func (e *Engine) executeTest(job Job) models.TestResult {
	start := time.Now()
	
	// Generate a unique request ID for tracking in verbose mode
	requestID := ""
	if e.verbose {
		requestID = uuid.New().String()[:8] // Use first 8 chars for readability
	}

	req, err := e.createRequest(job)
	if err != nil {
		return models.TestResult{
			TestName:  job.TestCase.Name,
			URL:       job.URL,
			Method:    job.TestCase.Method,
			Success:   false,
			Error:     err.Error(),
			Timestamp: start,
		}
	}

	timeout := job.TestCase.Timeout
	if timeout == 0 {
		timeout = job.Config.Global.Timeout
	}

	skipVerify := job.Config.Global.InsecureSkipVerify
	if job.TestCase.InsecureSkipVerify != nil {
		skipVerify = *job.TestCase.InsecureSkipVerify
	}

	var transport *http.Transport
	if skipVerify {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				MinVersion:         tls.VersionTLS10,
				MaxVersion:         tls.VersionTLS13,
				CipherSuites: []uint16{
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				},
			},
		}
	} else {
		transport = &http.Transport{}
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	
	// Log request details in verbose mode
	if e.verbose {
		log := models.DebugLog{
			Timestamp: start,
			RequestID: requestID,
			Type:      "request",
			TestName:  job.TestCase.Name,
			Method:    req.Method,
			URL:       req.URL.String(),
			Headers:   make(map[string]string),
		}
		
		// Convert headers
		for key, values := range req.Header {
			if len(values) > 0 {
				log.Headers[key] = strings.Join(values, "; ")
			}
		}
		
		if req.Body != nil {
			// Read and restore body for logging
			bodyBytes, _ := io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			log.Body = string(bodyBytes)
		}
		
		e.logChan <- log
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return models.TestResult{
			TestName:     job.TestCase.Name,
			URL:          job.URL,
			Method:       job.TestCase.Method,
			ResponseTime: time.Since(start),
			Success:      false,
			Error:        err.Error(),
			Timestamp:    start,
		}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	responseTime := time.Since(start)
	
	// Log response details in verbose mode
	if e.verbose {
		log := models.DebugLog{
			Timestamp:    time.Now(),
			RequestID:    requestID,
			Type:         "response",
			TestName:     job.TestCase.Name,
			StatusCode:   resp.StatusCode,
			Headers:      make(map[string]string),
			Body:         string(body),
			ResponseTime: responseTime,
		}
		
		// Convert headers
		for key, values := range resp.Header {
			if len(values) > 0 {
				log.Headers[key] = strings.Join(values, "; ")
			}
		}
		
		e.logChan <- log
	}

	success := e.isExpectedStatus(resp.StatusCode, job.TestCase.ExpectedStatus)

	result := models.TestResult{
		TestName:     job.TestCase.Name,
		URL:          job.URL,
		Method:       job.TestCase.Method,
		StatusCode:   resp.StatusCode,
		ResponseTime: responseTime,
		Success:      success,
		ResponseSize: int64(len(body)),
		RequestSize:  req.ContentLength,
		Timestamp:    start,
	}

	if !success {
		if e.verbose {
			// In verbose mode, include more details in the error message
			result.Error = fmt.Sprintf("Unexpected status code: %d (expected: %v)\nResponse body: %s",
				resp.StatusCode, job.TestCase.ExpectedStatus, string(body))
		} else {
			result.Error = fmt.Sprintf("Unexpected status code: %d (expected: %v)",
				resp.StatusCode, job.TestCase.ExpectedStatus)
		}
	}

	// Extract variables from response if extraction rules are defined
	if len(job.TestCase.Extract) > 0 && success {
		if err := e.varExtractor.Extract(job.TestCase.Extract, body, resp.Header, resp.StatusCode); err != nil {
			result.Error = fmt.Sprintf("Variable extraction failed: %v", err)
			result.Success = false
		}
	}

	// Evaluate assertions if any are defined
	if len(job.TestCase.Assertions) > 0 {
		ctx := assertion.NewContext(resp.StatusCode, responseTime, body, resp.Header)
		assertionResults := e.assertionEvaluator.EvaluateAll(job.TestCase.Assertions, ctx)

		for _, ar := range assertionResults {
			if ar.Passed {
				result.AssertionsPassed++
			} else {
				result.AssertionsFailed++
				result.AssertionErrors = append(result.AssertionErrors, ar.Message)
				result.Success = false // Assertion failure means test failure
			}
		}
	}

	return result
}

func (e *Engine) createRequest(job Job) (*http.Request, error) {
	// Substitute variables in URL
	url := e.varSubstitutor.Substitute(job.URL)

	var body io.Reader
	if job.TestCase.Body != nil {
		// Substitute variables in body
		substitutedBody := e.varSubstitutor.SubstituteBody(job.TestCase.Body)
		jsonBody, err := json.Marshal(substitutedBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		body = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(job.TestCase.Method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Substitute variables in global headers
	for key, value := range job.Config.Global.Headers {
		req.Header.Set(key, e.varSubstitutor.Substitute(value))
	}

	// Substitute variables in test-specific headers
	for key, value := range job.TestCase.Headers {
		req.Header.Set(key, e.varSubstitutor.Substitute(value))
	}

	if job.TestCase.Body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func (e *Engine) isExpectedStatus(statusCode int, expectedStatuses []int) bool {
	for _, expected := range expectedStatuses {
		if statusCode == expected {
			return true
		}
	}
	return false
}

func (e *Engine) collectResults(results <-chan models.TestResult, totalRequests int) *models.Summary {
	summary := &models.Summary{
		StatusCodes:     make(map[int]int),
		Errors:          make(map[string]int),
		EndpointResults: make(map[string]*models.EndpointSummary),
	}

	var allResults []models.TestResult

	for result := range results {
		allResults = append(allResults, result)

		summary.TotalRequests++
		if result.Success {
			summary.SuccessfulReqs++
		} else {
			summary.FailedReqs++
			if result.Error != "" {
				summary.Errors[result.Error]++
			}
		}

		summary.StatusCodes[result.StatusCode]++

		if summary.MinResponseTime == 0 || result.ResponseTime < summary.MinResponseTime {
			summary.MinResponseTime = result.ResponseTime
		}

		if result.ResponseTime > summary.MaxResponseTime {
			summary.MaxResponseTime = result.ResponseTime
		}

		// Collect endpoint-specific results
		key := result.TestName
		if summary.EndpointResults[key] == nil {
			summary.EndpointResults[key] = &models.EndpointSummary{
				Name:        result.TestName,
				URL:         result.URL,
				StatusCodes: make(map[int]int),
				Errors:      []string{},
			}
		}

		endpoint := summary.EndpointResults[key]
		endpoint.TotalRequests++
		if result.Success {
			endpoint.SuccessfulReqs++
		} else {
			endpoint.FailedReqs++
			if result.Error != "" {
				endpoint.Errors = append(endpoint.Errors, result.Error)
			}
		}
		endpoint.StatusCodes[result.StatusCode]++

		// Aggregate assertion results
		summary.AssertionsPassed += result.AssertionsPassed
		summary.AssertionsFailed += result.AssertionsFailed
		summary.TotalAssertions += result.AssertionsPassed + result.AssertionsFailed
		endpoint.AssertionsPassed += result.AssertionsPassed
		endpoint.AssertionsFailed += result.AssertionsFailed
		endpoint.TotalAssertions += result.AssertionsPassed + result.AssertionsFailed
	}

	if len(allResults) > 0 {
		var totalResponseTime time.Duration
		var allTimes []time.Duration
		endpointTimes := make(map[string][]time.Duration)

		for _, result := range allResults {
			totalResponseTime += result.ResponseTime
			allTimes = append(allTimes, result.ResponseTime)
			endpointTimes[result.TestName] = append(endpointTimes[result.TestName], result.ResponseTime)
		}

		summary.AvgResponseTime = totalResponseTime / time.Duration(len(allResults))
		summary.TotalTime = allResults[len(allResults)-1].Timestamp.Sub(allResults[0].Timestamp) + allResults[len(allResults)-1].ResponseTime

		if summary.TotalTime > 0 {
			summary.RequestsPerSec = float64(len(allResults)) / summary.TotalTime.Seconds()
		}

		// Calculate global percentiles
		summary.P50ResponseTime = calculatePercentile(allTimes, 50)
		summary.P95ResponseTime = calculatePercentile(allTimes, 95)
		summary.P99ResponseTime = calculatePercentile(allTimes, 99)

		// Calculate average response times and percentiles for each endpoint
		for testName, times := range endpointTimes {
			if endpoint, exists := summary.EndpointResults[testName]; exists {
				var total time.Duration
				for _, t := range times {
					total += t
				}
				endpoint.AvgResponseTime = total / time.Duration(len(times))
				endpoint.P50ResponseTime = calculatePercentile(times, 50)
				endpoint.P95ResponseTime = calculatePercentile(times, 95)
				endpoint.P99ResponseTime = calculatePercentile(times, 99)
			}
		}
	}

	return summary
}

func calculatePercentile(times []time.Duration, percentile float64) time.Duration {
	if len(times) == 0 {
		return 0
	}

	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})

	index := percentile * float64(len(times)-1) / 100.0
	lowerIndex := int(index)
	upperIndex := lowerIndex + 1

	if upperIndex >= len(times) {
		return times[len(times)-1]
	}

	if lowerIndex == int(index) {
		return times[lowerIndex]
	}

	// Linear interpolation
	weight := index - float64(lowerIndex)
	lower := times[lowerIndex]
	upper := times[upperIndex]

	return time.Duration(float64(lower) + weight*float64(upper-lower))
}

// logger is a goroutine that handles all verbose logging sequentially
func (e *Engine) logger() {
	for log := range e.logChan {
		if e.progressBar != nil {
			// Text mode: print formatted output
			e.printDebugLog(log)
		}
		// Always store for potential JSON output
		e.logMutex.Lock()
		e.debugLogs = append(e.debugLogs, log)
		e.logMutex.Unlock()
	}
}

// hasDependencies checks if any test has dependencies requiring DAG execution
func (e *Engine) hasDependencies(config *models.Config) bool {
	for _, test := range config.Tests {
		if len(test.DependsOn) > 0 {
			return true
		}
	}
	return false
}

// runWithDAG executes tests using DAG-based ordering for dependencies
func (e *Engine) runWithDAG(config *models.Config) *models.Summary {
	// Start logger goroutine if verbose mode is enabled
	if e.verbose {
		go e.logger()
	}

	startTime := time.Now()

	// Build DAG from test dependencies
	var testDeps []variables.TestDependency
	for _, test := range config.Tests {
		testDeps = append(testDeps, variables.TestDependency{
			Name:      test.Name,
			DependsOn: test.DependsOn,
		})
	}

	plan, err := variables.BuildDAG(testDeps)
	if err != nil {
		// Return summary with error
		summary := &models.Summary{
			StatusCodes:     make(map[int]int),
			Errors:          make(map[string]int),
			EndpointResults: make(map[string]*models.EndpointSummary),
		}
		summary.Errors[err.Error()] = 1
		return summary
	}

	// Create test lookup map
	testByName := make(map[string]models.TestCase)
	for _, test := range config.Tests {
		testByName[test.Name] = test
	}

	// Execute phases sequentially, tests within each phase in parallel
	var allResults []models.TestResult
	failedTests := make(map[string]bool) // Track tests that failed

	for _, phase := range plan.Phases {
		var wg sync.WaitGroup

		// Separate tests into executable and skipped
		var executableTests []string
		var skippedResults []models.TestResult

		for _, testName := range phase {
			test := testByName[testName]
			// Check if any dependency has failed
			var failedDep string
			for _, dep := range test.DependsOn {
				if failedTests[dep] {
					failedDep = dep
					break
				}
			}

			if failedDep != "" {
				// Skip this test - create skipped result(s)
				baseURL := strings.TrimSuffix(config.Global.BaseURL, "/")
				testPath := strings.TrimPrefix(test.Path, "/")
				fullURL := baseURL + "/" + testPath

				dataRows := e.getDataRows(test)
				iterations := config.Global.Iterations
				if test.Iterations > 0 {
					iterations = test.Iterations
				}
				if iterations <= 0 {
					iterations = 1
				}

				numSkipped := iterations
				if len(dataRows) > 0 {
					numSkipped = len(dataRows) * iterations
				}

				for i := 0; i < numSkipped; i++ {
					skippedResults = append(skippedResults, models.TestResult{
						TestName:   test.Name,
						URL:        fullURL,
						Method:     test.Method,
						Skipped:    true,
						SkipReason: fmt.Sprintf("dependency '%s' failed", failedDep),
						Timestamp:  time.Now(),
					})
				}
				// Mark this test as failed too (so its dependents are also skipped)
				failedTests[testName] = true
			} else {
				executableTests = append(executableTests, testName)
			}
		}

		// Add skipped results immediately
		for _, result := range skippedResults {
			allResults = append(allResults, result)
			if e.progressBar != nil {
				e.progressBar.Increment()
			}
		}

		// If no executable tests, continue to next phase
		if len(executableTests) == 0 {
			continue
		}

		// Calculate total jobs for executable tests
		totalPhaseJobs := 0
		for _, testName := range executableTests {
			test := testByName[testName]
			dataRows := e.getDataRows(test)
			iterations := config.Global.Iterations
			if test.Iterations > 0 {
				iterations = test.Iterations
			}
			if iterations <= 0 {
				iterations = 1
			}
			if len(dataRows) > 0 {
				totalPhaseJobs += len(dataRows) * iterations
			} else {
				totalPhaseJobs += iterations
			}
		}

		// Create channels with proper buffer sizes
		phaseResults := make(chan models.TestResult, totalPhaseJobs)
		phaseJobs := make(chan Job, totalPhaseJobs)

		// Limit workers to min(available workers, total jobs in phase)
		workers := e.workers
		if totalPhaseJobs < workers {
			workers = totalPhaseJobs
		}
		if workers < 1 {
			workers = 1
		}

		// Start workers for this phase
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range phaseJobs {
					// Apply think time before executing the request
					thinkTime := e.calculateThinkTime(job)
					if thinkTime > 0 {
						time.Sleep(thinkTime)
					}

					// Set data variables for data-driven tests
					if job.DataRow != nil {
						e.setDataVariables(job.DataRow)
					}

					result := e.executeTestWithExtraction(job)
					phaseResults <- result
				}
			}()
		}

		// Send jobs for executable tests
		for _, testName := range executableTests {
			test := testByName[testName]
			baseURL := strings.TrimSuffix(config.Global.BaseURL, "/")
			testPath := strings.TrimPrefix(test.Path, "/")
			fullURL := baseURL + "/" + testPath

			// Get data rows for data-driven testing
			dataRows := e.getDataRows(test)

			// Determine iterations
			iterations := config.Global.Iterations
			if test.Iterations > 0 {
				iterations = test.Iterations
			}
			if iterations <= 0 {
				iterations = 1
			}

			if len(dataRows) > 0 {
				// Data-driven test: run iterations for each data row
				for _, dataRow := range dataRows {
					for i := 0; i < iterations; i++ {
						phaseJobs <- Job{
							Config:   config,
							TestCase: test,
							URL:      fullURL,
							DataRow:  dataRow,
						}
					}
				}
			} else {
				// Regular test without data
				for i := 0; i < iterations; i++ {
					phaseJobs <- Job{
						Config:   config,
						TestCase: test,
						URL:      fullURL,
					}
				}
			}
		}
		close(phaseJobs)

		// Wait for all tests in this phase to complete
		wg.Wait()
		close(phaseResults)

		// Collect results for this phase and track failures
		for result := range phaseResults {
			allResults = append(allResults, result)
			if e.progressBar != nil {
				e.progressBar.Increment()
			}
			// Mark test as failed if it didn't succeed
			if !result.Success {
				failedTests[result.TestName] = true
			}
		}
	}

	// Calculate summary from all results
	summary := e.calculateSummaryFromResults(allResults, startTime)

	if e.progressBar != nil {
		e.progressBar.Finish()
	}

	// Close log channel if verbose mode is enabled
	if e.verbose {
		close(e.logChan)
		time.Sleep(100 * time.Millisecond)

		e.logMutex.Lock()
		summary.DebugLogs = e.debugLogs
		e.logMutex.Unlock()
	}

	return summary
}

// executeTestWithExtraction executes a test and extracts variables from the response
// Note: extraction is now handled directly in executeTest(), so this is a simple wrapper
func (e *Engine) executeTestWithExtraction(job Job) models.TestResult {
	return e.executeTest(job)
}

// calculateSummaryFromResults creates a summary from a slice of results
func (e *Engine) calculateSummaryFromResults(allResults []models.TestResult, startTime time.Time) *models.Summary {
	summary := &models.Summary{
		StatusCodes:     make(map[int]int),
		Errors:          make(map[string]int),
		EndpointResults: make(map[string]*models.EndpointSummary),
	}

	for _, result := range allResults {
		summary.TotalRequests++

		// Collect endpoint-specific results
		key := result.TestName
		if summary.EndpointResults[key] == nil {
			summary.EndpointResults[key] = &models.EndpointSummary{
				Name:            result.TestName,
				URL:             result.URL,
				StatusCodes:     make(map[int]int),
				Errors:          []string{},
				FirstExecutedAt: result.Timestamp,
			}
		}
		endpoint := summary.EndpointResults[key]
		endpoint.TotalRequests++
		// Track earliest execution time
		if endpoint.FirstExecutedAt.IsZero() || result.Timestamp.Before(endpoint.FirstExecutedAt) {
			endpoint.FirstExecutedAt = result.Timestamp
		}

		// Handle skipped tests separately
		if result.Skipped {
			summary.SkippedReqs++
			endpoint.SkippedReqs++
			if result.SkipReason != "" {
				summary.Errors[result.SkipReason]++
				endpoint.Errors = append(endpoint.Errors, result.SkipReason)
			}
			continue // Don't count skipped in response times or status codes
		}

		if result.Success {
			summary.SuccessfulReqs++
			endpoint.SuccessfulReqs++
		} else {
			summary.FailedReqs++
			endpoint.FailedReqs++
			if result.Error != "" {
				summary.Errors[result.Error]++
				endpoint.Errors = append(endpoint.Errors, result.Error)
			}
		}

		summary.StatusCodes[result.StatusCode]++
		endpoint.StatusCodes[result.StatusCode]++

		if summary.MinResponseTime == 0 || result.ResponseTime < summary.MinResponseTime {
			summary.MinResponseTime = result.ResponseTime
		}

		if result.ResponseTime > summary.MaxResponseTime {
			summary.MaxResponseTime = result.ResponseTime
		}

		// Aggregate assertion results
		summary.AssertionsPassed += result.AssertionsPassed
		summary.AssertionsFailed += result.AssertionsFailed
		summary.TotalAssertions += result.AssertionsPassed + result.AssertionsFailed
		endpoint.AssertionsPassed += result.AssertionsPassed
		endpoint.AssertionsFailed += result.AssertionsFailed
		endpoint.TotalAssertions += result.AssertionsPassed + result.AssertionsFailed
	}

	// Calculate response time stats (excluding skipped)
	executedCount := summary.SuccessfulReqs + summary.FailedReqs
	if executedCount > 0 {
		var totalResponseTime time.Duration
		var allTimes []time.Duration
		endpointTimes := make(map[string][]time.Duration)

		for _, result := range allResults {
			if result.Skipped {
				continue // Skip from response time calculations
			}
			totalResponseTime += result.ResponseTime
			allTimes = append(allTimes, result.ResponseTime)
			endpointTimes[result.TestName] = append(endpointTimes[result.TestName], result.ResponseTime)
		}

		summary.AvgResponseTime = totalResponseTime / time.Duration(executedCount)
		summary.TotalTime = time.Since(startTime)

		if summary.TotalTime > 0 {
			summary.RequestsPerSec = float64(executedCount) / summary.TotalTime.Seconds()
		}

		// Calculate global percentiles
		summary.P50ResponseTime = calculatePercentile(allTimes, 50)
		summary.P95ResponseTime = calculatePercentile(allTimes, 95)
		summary.P99ResponseTime = calculatePercentile(allTimes, 99)

		// Calculate average response times and percentiles for each endpoint
		for testName, times := range endpointTimes {
			if endpoint, exists := summary.EndpointResults[testName]; exists {
				var total time.Duration
				for _, t := range times {
					total += t
				}
				endpoint.AvgResponseTime = total / time.Duration(len(times))
				endpoint.P50ResponseTime = calculatePercentile(times, 50)
				endpoint.P95ResponseTime = calculatePercentile(times, 95)
				endpoint.P99ResponseTime = calculatePercentile(times, 99)
			}
		}
	}

	return summary
}

// printDebugLog formats and prints debug log for text output
func (e *Engine) printDebugLog(log models.DebugLog) {
	if log.Type == "request" {
		fmt.Printf("\n=== REQUEST DEBUG ===")
		fmt.Printf("\nRequest ID: %s", log.RequestID)
		fmt.Printf("\nTimestamp: %s", log.Timestamp.Format(time.RFC3339))
		fmt.Printf("\nTest: %s", log.TestName)
		fmt.Printf("\nMethod: %s", log.Method)
		fmt.Printf("\nURL: %s", log.URL)
		if len(log.Headers) > 0 {
			fmt.Printf("\nHeaders:")
			for key, value := range log.Headers {
				fmt.Printf("\n  %s: %s", key, value)
			}
		}
		if log.Body != "" {
			fmt.Printf("\nBody: %s", log.Body)
		}
		fmt.Printf("\n===================\n")
	} else if log.Type == "response" {
		fmt.Printf("\n=== RESPONSE DEBUG ===")
		fmt.Printf("\nRequest ID: %s", log.RequestID)
		fmt.Printf("\nTimestamp: %s", log.Timestamp.Format(time.RFC3339))
		fmt.Printf("\nTest: %s", log.TestName)
		fmt.Printf("\nStatus: %d", log.StatusCode)
		if len(log.Headers) > 0 {
			fmt.Printf("\nHeaders:")
			for key, value := range log.Headers {
				fmt.Printf("\n  %s: %s", key, value)
			}
		}
		if log.Body != "" {
			fmt.Printf("\nBody (%d bytes):", len(log.Body))
			if len(log.Body) > 1000 {
				fmt.Printf("\n%s... (truncated)", log.Body[:1000])
			} else {
				fmt.Printf("\n%s", log.Body)
			}
		}
		fmt.Printf("\nResponse Time: %v", log.ResponseTime)
		fmt.Printf("\n===================\n")
	}
}
