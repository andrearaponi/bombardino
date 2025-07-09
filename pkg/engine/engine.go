package engine

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/andrearaponi/bombardino/internal/models"
	"github.com/andrearaponi/bombardino/pkg/progress"
	"github.com/google/uuid"
)

type Engine struct {
	workers     int
	progressBar *progress.ProgressBar
	verbose     bool
	logChan     chan models.DebugLog
	debugLogs   []models.DebugLog
	logMutex    sync.Mutex
}

func New(workers int, progressBar *progress.ProgressBar, verbose bool) *Engine {
	e := &Engine{
		workers:     workers,
		progressBar: progressBar,
		verbose:     verbose,
	}
	if verbose {
		e.logChan = make(chan models.DebugLog, 100)
	}
	return e
}

func (e *Engine) Run(config *models.Config) *models.Summary {
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

		for i := 0; i < iterations; i++ {
			jobs <- Job{
				Config:   config,
				TestCase: test,
				URL:      fullURL,
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

	return result
}

func (e *Engine) createRequest(job Job) (*http.Request, error) {
	var body io.Reader

	if job.TestCase.Body != nil {
		jsonBody, err := json.Marshal(job.TestCase.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		body = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(job.TestCase.Method, job.URL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range job.Config.Global.Headers {
		req.Header.Set(key, value)
	}

	for key, value := range job.TestCase.Headers {
		req.Header.Set(key, value)
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
