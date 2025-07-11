<div align="center">
  <h1>Bombardino</h1>
  <p>
    <img src="https://img.shields.io/badge/status-active-green.svg" alt="Status">
    <img src="https://img.shields.io/badge/release-0.2.0beta-green.svg" alt="Release">
    <img src="https://img.shields.io/badge/license-Mit-blue.svg" alt="License">
    <img src="https://img.shields.io/badge/language-Go-blue.svg" alt="Language">
  </p>
  <p>
    <img src="bombardino.png" alt="Bombardino Logo">

  </p>
  <p>
    <strong>Bombardino</strong> is a powerful REST API stress testing tool written in Go. It offers JSON file configuration, concurrent execution, real-time progress bar, and detailed reports.
  </p>

</div>


## Features

*   **Layer 7 REST API Testing:** Handles HTTP and HTTPS requests with comprehensive validation.
*   **Duration-Based Testing:** Run tests for specific time periods rather than fixed iterations.
*   **Mixed Mode Support:** Combine duration and iteration-based tests in the same configuration.
*   **Advanced Metrics:** Detailed percentiles (P50, P95, P99) for performance analysis.
*   **Concurrent Execution:** Configurable worker pools for optimal throughput.
*   **Real-Time Monitoring:** Live progress bar with statistics and ETA calculations.
*   **Flexible Configuration:** JSON-based configuration with per-endpoint customization.
*   **SSL/TLS Support:** Custom certificate handling and insecure skip verification.
*   **CI/CD Integration:** JSON output format with proper exit codes for automation.
*   **Per-Endpoint Reporting:** Detailed metrics and success rates for each test case.
*   **Custom Headers & Payloads:** Full HTTP request customization per endpoint.
*   **Response Validation:** Expected status code validation with comprehensive error reporting.
*   **Debug Mode:** Detailed request/response logging with UUID tracking for debugging.
*   **Assertion System:** Advanced response validation (planned for future release).

## Installation

Ensure you have Go (>= 1.21) and `make` installed.

### Quick Start (Recommended)

```bash
# Clone repo
git clone https://github.com/andrearaponi/bombardino.git && cd bombardino

# One-command setup & start
make run-example
```
This will:
1.  Build the Bombardino binary
2.  Run a stress test with the example configuration
3.  Display comprehensive results with percentiles

### Step-by-Step

```bash
# 1. Clone repo
git clone https://github.com/andrearaponi/bombardino.git && cd bombardino

# 2. Build binary
make build

# 3. Run test
./bin/bombardino -config=examples/example-config.json
```

### Makefile Commands

| Category | Command | Description |
|----------|---------|-------------|
| **Quick** | `run-example` | Build and run with example configuration (recommended) |
| | `run` | Build and show version information |
| **Build** | `build` | Build Bombardino binary for current platform |
| | `build-all` | Build binaries for all platforms (Linux, macOS, Windows) |
| | `install` | Install binary to $GOPATH/bin |
| | `release` | Prepare release artifacts with cross-platform builds |
| **Testing** | `test` | Run all tests |
| | `test-coverage` | Run tests with coverage report |
| | `test-short` | Run only short tests |
| | `bench` | Run benchmarks |
| **Quality** | `fmt` | Format Go code |
| | `vet` | Run go vet |
| | `lint` | Run golangci-lint (requires golangci-lint installation) |
| | `check` | Run all code quality checks |
| **Dependencies** | `deps` | Download dependencies |
| | `deps-update` | Update dependencies |
| **Cleanup** | `clean` | Remove all build artifacts |
| | `clean-all` | Clean all artifacts including dependencies |
| **Info** | `help` | Show all commands with detailed descriptions |
| | `version` | Show version information |
| | `info` | Show detailed build information |

### Manual Installation (Advanced)

1.  **Build Bombardino:**
    ```bash
    go build -o bin/bombardino ./cmd/bombardino
    ```

2.  **Run with example config:**
    ```bash
    ./bin/bombardino -config=examples/example-config.json
    ```

### Download Pre-built Binaries

Download the latest release from [GitHub Releases](https://github.com/andrearaponi/bombardino/releases)

## Usage

Start with default config:
```bash
make run-example
```

Or directly:
```bash
./bin/bombardino -config=examples/example-config.json -workers=20 -output=json
```

### Command Line Options
```bash
bombardino [options]

Options:
  -config string     Path to JSON configuration file (required)
  -workers int       Number of concurrent workers (default: 10)
  -verbose          Enable verbose debug output with request/response details (default: false)
  -output string    Output format: text, json, or html (default: text)
  -version          Show version information
```

### Common Usage Patterns

```bash
# Quick performance test with example configuration
make run-example

# High-throughput test with 50 concurrent workers
./bin/bombardino -config=examples/example-config.json -workers=50

# Duration-based load testing for 30 seconds
./bin/bombardino -config=examples/duration-test.json -workers=10

# CI/CD integration with JSON output
./bin/bombardino -config=ci-tests.json -output=json -workers=20

# Generate HTML report for sharing
./bin/bombardino -config=examples/example-config.json -output=html > report.html

# Mixed mode testing (duration + iterations)
./bin/bombardino -config=examples/mixed-mode-test.json -workers=5

# Verbose debugging output with detailed request/response logging
./bin/bombardino -config=examples/example-config.json -verbose

# Show version and build information
./bin/bombardino -version
```

## Configuration

The JSON configuration file defines all test parameters:

```json
{
  "name": "API Stress Test Example",
  "description": "Example configuration for testing REST APIs",
  "global": {
    "base_url": "https://jsonplaceholder.typicode.com",
    "timeout": "30s",
    "delay": "100ms",
    "iterations": 100,
    "headers": {
      "User-Agent": "Bombardino/1.0",
      "Accept": "application/json"
    }
  },
  "tests": [
    {
      "name": "Get all posts",
      "method": "GET",
      "path": "/posts",
      "expected_status": [200],
      "iterations": 50,
      "delay": "50ms"
    },
    {
      "name": "Create new post",
      "method": "POST",
      "path": "/posts",
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "title": "Test Post",
        "body": "Test content",
        "userId": 1
      },
      "expected_status": [201],
      "iterations": 25
    }
  ]
}
```

### Configuration Structure

#### Global Configuration
- `base_url`: Base URL for all tests
- `timeout`: Global timeout for requests (Go duration format)
- `delay`: Global delay between requests (Go duration format)
- `iterations`: Global number of iterations for tests (optional if using duration)
- `duration`: Global test duration (Go duration format, e.g., "30s", "5m")
- `headers`: Global HTTP headers
- `insecure_skip_verify`: Skip SSL/TLS certificate verification (default: false)

#### Test Configuration
- `name`: Descriptive name of the test
- `method`: HTTP method (GET, POST, PUT, DELETE, etc.)
- `path`: Path relative to the base URL
- `headers`: Test-specific HTTP headers (override global ones)
- `body`: JSON payload for POST/PUT tests
- `expected_status`: Array of accepted HTTP status codes
- `timeout`: Test-specific timeout (optional)
- `delay`: Test-specific delay between requests (optional)
- `iterations`: Test-specific number of iterations (optional, default: global iterations)
- `duration`: Test-specific duration (optional, overrides global duration)
- `insecure_skip_verify`: Override global SSL skip setting (optional)
- `assertions`: Array of assertions to validate the response (planned feature)

## Duration-Based Testing

Bombardino supports duration-based testing for realistic load testing scenarios. Instead of specifying a fixed number of iterations, you can run tests for a specific amount of time.

### Test Modes

Bombardino supports three distinct testing modes:

#### 1. Iteration Mode (Classic)
Run a specific number of requests per test:

```json
{
  "global": {
    "base_url": "https://api.example.com",
    "iterations": 100
  },
  "tests": [
    {
      "name": "Fixed iterations test",
      "method": "GET",
      "path": "/api",
      "iterations": 50
    }
  ]
}
```

#### 2. Duration Mode (Load Testing)
Run tests for a specific duration:

```json
{
  "global": {
    "base_url": "https://api.example.com",
    "duration": "2m"
  },
  "tests": [
    {
      "name": "2-minute load test",
      "method": "GET",
      "path": "/api"
    },
    {
      "name": "Custom 30s test",
      "method": "POST",
      "path": "/api/data",
      "duration": "30s"
    }
  ]
}
```

#### 3. Mixed Mode (Advanced)
Combine both duration and iteration-based tests in the same configuration:

```json
{
  "global": {
    "base_url": "https://api.example.com",
    "duration": "1m",
    "iterations": 50
  },
  "tests": [
    {
      "name": "Duration-based test",
      "method": "GET",
      "path": "/continuous-load"
    },
    {
      "name": "Short burst test",
      "method": "GET",
      "path": "/quick-check",
      "duration": "15s"
    },
    {
      "name": "Fixed iterations test",
      "method": "POST", 
      "path": "/specific-test",
      "iterations": 25
    }
  ]
}
```

### Duration Examples and Use Cases

| Duration | Use Case | Example Scenario |
|----------|----------|------------------|
| `"30s"` | Quick smoke test | API health check after deployment |
| `"2m"` | Short load test | Validate performance under normal load |
| `"5m"` | Sustained load test | Test API stability over time |
| `"30m"` | Endurance test | Long-running performance validation |
| `"1h"` | Stress test | Extended load for capacity planning |

### Advanced Duration Testing

```json
{
  "name": "Production Load Test",
  "description": "Realistic production load simulation",
  "global": {
    "base_url": "https://api.production.com",
    "duration": "10m",
    "delay": "100ms",
    "timeout": "30s"
  },
  "tests": [
    {
      "name": "User authentication",
      "method": "POST",
      "path": "/auth/login",
      "duration": "10m",
      "delay": "200ms"
    },
    {
      "name": "Data retrieval",
      "method": "GET", 
      "path": "/api/data",
      "duration": "8m",
      "delay": "50ms"
    },
    {
      "name": "Health check (quick)",
      "method": "GET",
      "path": "/health",
      "iterations": 20
    }
  ]
}
```

### Key Benefits

*   **Realistic Load Testing**: Simulate actual production traffic patterns
*   **Capacity Planning**: Test how your API performs under sustained load
*   **SLA Validation**: Verify performance over time, not just burst capacity
*   **Resource Monitoring**: Observe system behavior during extended periods
*   **Flexible Testing**: Mix different test types in a single run

## Debug Mode

Bombardino includes a powerful debug mode that provides detailed request and response logging for troubleshooting API issues. This feature is especially useful when you receive unexpected status codes or need to inspect the exact data being sent and received.

### Enabling Debug Mode

Use the `-verbose` flag to enable debug mode:

```bash
# Text output with debug information
./bin/bombardino -config=test.json -verbose

# JSON output with structured debug logs
./bin/bombardino -config=test.json -verbose -output=json
```

### Text Debug Output

When using verbose mode with text output, you'll see detailed request and response information:

```
=== REQUEST DEBUG ===
Request ID: a4e140ee
Timestamp: 2025-07-09T15:26:23+02:00
Test: Test API Endpoint
Method: GET
URL: https://api.example.com/users/123
Headers:
  Authorization: Bearer token123
  Content-Type: application/json
  User-Agent: Bombardino/1.0
Body: {"userId": 123}
===================

=== RESPONSE DEBUG ===
Request ID: a4e140ee
Timestamp: 2025-07-09T15:26:24+02:00
Test: Test API Endpoint
Status: 404 Not Found
Headers:
  Content-Type: application/json
  Content-Length: 45
Body (45 bytes):
{"error": "User not found", "code": 404}
Response Time: 234ms
===================
```

### JSON Debug Output

When using verbose mode with JSON output (`-verbose -output=json`), debug information is included as structured data:

```json
{
  "summary": {
    "total_requests": 2,
    "successful_requests": 1,
    "failed_requests": 1,
    "success_rate_percent": 50.0,
    "...": "..."
  },
  "endpoints": {
    "...": "..."
  },
  "debug_logs": [
    {
      "timestamp": "2025-07-09T15:26:23+02:00",
      "request_id": "a4e140ee",
      "type": "request",
      "test_name": "Test API Endpoint",
      "method": "GET",
      "url": "https://api.example.com/users/123",
      "headers": {
        "Authorization": "Bearer token123",
        "Content-Type": "application/json",
        "User-Agent": "Bombardino/1.0"
      },
      "body": "{\"userId\": 123}"
    },
    {
      "timestamp": "2025-07-09T15:26:24+02:00",
      "request_id": "a4e140ee",
      "type": "response",
      "test_name": "Test API Endpoint",
      "status_code": 404,
      "headers": {
        "Content-Type": "application/json",
        "Content-Length": "45"
      },
      "body": "{\"error\": \"User not found\", \"code\": 404}",
      "response_time": 234000000
    }
  ],
  "success": false
}
```

### Debug Features

- **Request ID Tracking**: Each request/response pair is linked with a unique 8-character UUID
- **Parallel Request Handling**: Debug logs remain organized even with multiple concurrent workers
- **Body Truncation**: Large response bodies are truncated at 1000 characters for readability in text mode
- **Structured Logging**: JSON mode provides machine-readable debug information
- **Enhanced Error Messages**: Verbose mode includes response body content in error messages

### Use Cases

Debug mode is particularly useful for:

- **API Development**: Verify request format and headers
- **Error Debugging**: See exact error responses when tests fail
- **Integration Testing**: Validate API contract compliance
- **Performance Analysis**: Correlate request/response details with timing
- **CI/CD Troubleshooting**: Capture detailed logs for automated testing

### Performance Impact

Debug mode adds minimal overhead but generates substantial output. For production load testing, disable verbose mode to maximize performance.

## Report Output

Bombardino generates detailed reports in three formats:

### Text Format (Default)

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                              BOMBARDINO RESULTS                              ║
╚══════════════════════════════════════════════════════════════════════════════╝

📊 SUMMARY
────────────────────────────────────────────────────────────────────────────────
Total Requests:      150
Successful:          150 (100.0%)
Failed:              0 (0.0%)
Requests/sec:        13.13
Total Duration:      11.421188s

⏱️  RESPONSE TIMES
────────────────────────────────────────────────────────────────────────────────
Average:             49ms
Minimum:             20ms
Maximum:             342ms
P50 (median):        45ms
P95:                 156ms
P99:                 298ms

📈 STATUS CODES
────────────────────────────────────────────────────────────────────────────────
✅ 200:              125 (83.3%)
✅ 201:              25 (16.7%)

🎯 ENDPOINT RESULTS
────────────────────────────────────────────────────────────────────────────────
✅ Get all posts
   URL: https://jsonplaceholder.typicode.com/posts
   Requests: 125 | Success: 125 (100.0%) | Failed: 0
   Response Times: Avg=42ms | P50=40ms | P95=89ms | P99=156ms
   Status Codes: 200 (125)

✅ Create new post
   URL: https://jsonplaceholder.typicode.com/posts
   Requests: 25 | Success: 25 (100.0%) | Failed: 0
   Response Times: Avg=78ms | P50=75ms | P95=142ms | P99=189ms
   Status Codes: 201 (25)

════════════════════════════════════════════════════════════════════════════════
🚀 Test completed successfully!
```

### HTML Format (Report Sharing)

Use the `-output=html` flag to generate a formatted HTML report perfect for sharing with teams:

```bash
./bin/bombardino -config=test.json -output=html > report.html
```

The HTML report includes:
- Interactive charts and visualizations
- Detailed performance metrics with color coding
- Professional formatting suitable for presentations
- Responsive design that works on all devices
- Complete test results including per-endpoint analysis

### JSON Format (CI/CD Friendly)

Use the `-output=json` flag to get machine-readable output perfect for CI/CD integration:

```json
{
  "summary": {
    "total_requests": 150,
    "successful_requests": 150,
    "failed_requests": 0,
    "success_rate_percent": 100.0,
    "total_time": "11.421188s",
    "avg_response_time": "49ms",
    "min_response_time": "20ms",
    "max_response_time": "342ms",
    "p50_response_time": "45ms",
    "p95_response_time": "156ms",
    "p99_response_time": "298ms",
    "requests_per_sec": 13.13,
    "status_codes": {
      "200": 125,
      "201": 25
    },
    "errors": {}
  },
  "endpoints": {
    "Get all posts": {
      "name": "Get all posts",
      "url": "https://jsonplaceholder.typicode.com/posts",
      "total_requests": 125,
      "successful_requests": 125,
      "failed_requests": 0,
      "success_rate_percent": 100.0,
      "avg_response_time": "42ms",
      "p50_response_time": "40ms",
      "p95_response_time": "89ms",
      "p99_response_time": "156ms",
      "status_codes": {
        "200": 125
      },
      "errors": [],
      "success": true
    },
    "Create new post": {
      "name": "Create new post",
      "url": "https://jsonplaceholder.typicode.com/posts",
      "total_requests": 25,
      "successful_requests": 25,
      "failed_requests": 0,
      "success_rate_percent": 100.0,
      "avg_response_time": "78ms",
      "p50_response_time": "75ms",
      "p95_response_time": "142ms",
      "p99_response_time": "189ms",
      "status_codes": {
        "201": 25
      },
      "errors": [],
      "success": true
    }
  },
  "debug_logs": [
    {
      "timestamp": "2025-07-09T15:26:23+02:00",
      "request_id": "a4e140ee",
      "type": "request",
      "test_name": "Get all posts",
      "method": "GET",
      "url": "https://jsonplaceholder.typicode.com/posts",
      "headers": {
        "User-Agent": "Bombardino/1.0",
        "Accept": "application/json"
      }
    },
    {
      "timestamp": "2025-07-09T15:26:24+02:00",
      "request_id": "a4e140ee",
      "type": "response",
      "test_name": "Get all posts",
      "status_code": 200,
      "headers": {
        "Content-Type": "application/json; charset=utf-8",
        "Content-Length": "6178"
      },
      "body": "[{\"userId\":1,\"id\":1,\"title\":\"sunt aut facere...\"}]",
      "response_time": 234000000
    }
  ],
  "success": true
}
```

### Performance Metrics Explained

- **P50 (Median)**: 50% of requests were faster than this time
- **P95**: 95% of requests were faster than this time (identifies slowest 5%)
- **P99**: 99% of requests were faster than this time (identifies outliers)

These percentiles are crucial for understanding your API's performance distribution and SLA compliance.

## SSL/TLS Configuration

Bombardino supports flexible SSL/TLS configuration for testing APIs with self-signed certificates or internal services:

### Example with SSL Skip Verification

```json
{
  "name": "Internal API Test",
  "global": {
    "base_url": "https://internal-api.company.com",
    "timeout": "30s",
    "iterations": 10,
    "insecure_skip_verify": true,
    "headers": {
      "Authorization": "Bearer your-token"
    }
  },
  "tests": [
    {
      "name": "Health check",
      "method": "GET",
      "path": "/health",
      "expected_status": [200]
    },
    {
      "name": "Secure endpoint with valid cert",
      "method": "GET", 
      "path": "/secure-data",
      "insecure_skip_verify": false,
      "expected_status": [200]
    }
  ]
}
```

### SSL Options

- **Global level**: Set `insecure_skip_verify: true` in global config
- **Per-test level**: Override global setting for specific tests
- **Use cases**: Self-signed certificates, internal development environments, testing environments

### Exit Codes

- **0**: All tests passed successfully
- **1**: Configuration or execution errors
- **Test failures**: Use JSON output to check `success` field

## CI/CD Integration

Bombardino's JSON output and proper exit codes make it perfect for CI/CD pipelines:

### GitHub Actions Example

```yaml
name: API Performance Tests
on: [push, pull_request]

jobs:
  performance-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Setup Go
        uses: actions/setup-go@v3
        with:
          go-version: '1.21'
          
      - name: Build Bombardino
        run: make build
        
      - name: Run Tests
        run: make test
        
      - name: Run Performance Tests
        run: ./bin/bombardino -config=tests/api-performance.json -output=json > results.json
        
      - name: Run Debug Tests (on failure)
        if: failure()
        run: ./bin/bombardino -config=tests/api-performance.json -verbose -output=json > debug-results.json
        
      - name: Check Results
        run: |
          SUCCESS=$(cat results.json | jq -r '.success')
          P95=$(cat results.json | jq -r '.summary.p95_response_time')
          echo "Test Success: $SUCCESS"
          echo "P95 Response Time: $P95"
          
          if [ "$SUCCESS" != "true" ]; then
            echo "Performance tests failed!"
            if [ -f debug-results.json ]; then
              echo "Debug information:"
              cat debug-results.json | jq -r '.debug_logs[] | select(.type == "response" and .status_code != 200) | "Request ID: \(.request_id) - Status: \(.status_code) - Body: \(.body)"'
            fi
            exit 1
          fi
          
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: performance-results
          path: results.json
```

### Jenkins Pipeline Example

```groovy
pipeline {
    agent any
    stages {
        stage('Build') {
            steps {
                sh 'make build'
            }
        }
        stage('Test') {
            steps {
                sh 'make test'
            }
        }
        stage('Performance Test') {
            steps {
                sh './bin/bombardino -config=performance-tests.json -output=json > results.json'
                script {
                    def results = readJSON file: 'results.json'
                    if (!results.success) {
                        error "Performance tests failed!"
                    }
                    echo "P95 Response Time: ${results.summary.p95_response_time}"
                    archiveArtifacts artifacts: 'results.json', fingerprint: true
                }
            }
        }
    }
}
```

### Makefile Integration

```bash
# Run performance tests in your CI pipeline
make build
make test
./bin/bombardino -config=ci-tests.json -output=json -workers=10
```

## Project Structure

The project is structured following Go best practices:

```
bombardino/
├── cmd/bombardino/          # Main application
├── pkg/                     # Public libraries
│   ├── config/             # JSON configuration parser
│   ├── engine/             # Test execution engine
│   ├── progress/           # Progress bar
│   └── reporter/           # Reporting system  
├── internal/models/        # Internal data models
└── examples/              # Configuration examples
```

## Future Roadmap

*   **Ramp-up/Ramp-down scenarios** (gradually increase/decrease load)
*   **Target RPS control** (maintain specific requests per second)
*   **Think time simulation** for realistic user behavior
*   **Test scenarios with dependencies** (login → API calls → logout)
*   **Assertion system** (JSON Path, response time, header validation, etc.)
*   **WebSocket support** for real-time applications
*   **Web frontend for visualization**
*   **Advanced metrics and charts**
*   **Enhanced report formats** (improved HTML styling and features)
*   **Load balancing** across multiple endpoints
*   **Real-time dashboard**
*   **Enhanced debug filtering** (filter by status code, test name, etc.)
*   **Debug log export** (save debug logs to separate files)

## Contributing

1.  Fork the repo.
2.  Create your feature branch (`git checkout -b feature/AmazingFeature`).
3.  Commit your changes (`git commit -m 'Add some AmazingFeature'`).
4.  Push to the branch (`git push origin feature/AmazingFeature`).
5.  Open a Pull Request.


## License

This project is distributed under the MIT license. See the `LICENSE` file for details.

---
