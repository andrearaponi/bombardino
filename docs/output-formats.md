# Output Formats

Bombardino generates reports in three formats: text (default), JSON, and HTML. Each format serves different needs.

## Text Output (Default)

Text output is designed for the terminal. It includes colors, unicode characters, and a clean layout.

### Usage

```bash
bombardino -config test.json
# or explicitly
bombardino -config test.json -output text
```

### Example Output

```
╔════════════════════════════════════════════════════════════════╗
║                        TEST SUMMARY                            ║
╚════════════════════════════════════════════════════════════════╝

Total Requests:     100
Successful:         98 (98.0%)
Failed:             2 (2.0%)
Total Time:         15.234s
Requests/sec:       6.57

╔════════════════════════════════════════════════════════════════╗
║                        ASSERTIONS                              ║
╚════════════════════════════════════════════════════════════════╝

Total: 15    Passed: 14    Failed: 1

✅ status eq 200
✅ json_path id exists
✅ json_path name eq Mario
❌ response_time lt 100ms (actual: 245ms)

╔════════════════════════════════════════════════════════════════╗
║                      RESPONSE TIMES                            ║
╚════════════════════════════════════════════════════════════════╝

Minimum:            45ms
Average:            156ms
Maximum:            892ms
P50 (Median):       145ms
P95:                389ms
P99:                654ms

╔════════════════════════════════════════════════════════════════╗
║                      STATUS CODES                              ║
╚════════════════════════════════════════════════════════════════╝

✅ 200 OK:              98 requests
❌ 500 Internal Error:  2 requests

╔════════════════════════════════════════════════════════════════╗
║                    ENDPOINT RESULTS                            ║
╚════════════════════════════════════════════════════════════════╝

Test: Create User (POST /api/users)
────────────────────────────────────────────────────────────────
Total Requests:     50
Success Rate:       100.0%
Avg Response:       123ms
P95:                287ms
```

### Status Code Icons

| Icon | Status Range | Meaning |
|------|--------------|---------|
| ✅ | 2xx | Success |
| 🔄 | 3xx | Redirect |
| ⚠️ | 4xx | Client Error |
| ❌ | 5xx | Server Error |

## JSON Output

JSON output is machine-readable, perfect for CI/CD pipelines and automation.

### Usage

```bash
bombardino -config test.json -output json
# Save to file
bombardino -config test.json -output json > results.json
```

### Example Output

```json
{
  "summary": {
    "total_requests": 100,
    "successful": 98,
    "failed": 2,
    "total_time": "15.234s",
    "avg_response_time": "156ms",
    "min_response_time": "45ms",
    "max_response_time": "892ms",
    "p50_response_time": "145ms",
    "p95_response_time": "389ms",
    "p99_response_time": "654ms",
    "requests_per_sec": 6.57,
    "status_codes": {
      "200": 98,
      "500": 2
    },
    "errors": {
      "connection timeout": 2
    }
  },
  "assertions": {
    "total": 15,
    "passed": 14,
    "failed": 1,
    "results": [
      {"type": "status", "target": "response", "operator": "eq", "expected": 200, "passed": true},
      {"type": "response_time", "target": "response", "operator": "lt", "expected": "100ms", "actual": "245ms", "passed": false}
    ]
  },
  "endpoints": {
    "Create User": {
      "name": "Create User",
      "method": "POST",
      "url": "/api/users",
      "total_requests": 50,
      "successful": 50,
      "failed": 0,
      "success_rate": 100.0,
      "avg_response_time": "123ms",
      "min_response_time": "67ms",
      "max_response_time": "456ms",
      "p50_response_time": "115ms",
      "p95_response_time": "287ms",
      "p99_response_time": "432ms",
      "requests_per_sec": 3.28,
      "status_codes": {
        "201": 50
      }
    }
  },
  "debug_logs": [],
  "success": false
}
```

### Key Fields

| Field | Description |
|-------|-------------|
| `summary.total_requests` | Total number of requests sent |
| `summary.successful` | Requests matching expected status |
| `summary.failed` | Requests not matching or with errors |
| `summary.requests_per_sec` | Throughput |
| `assertions.passed` | Number of passing assertions |
| `assertions.failed` | Number of failing assertions |
| `endpoints` | Per-endpoint breakdown |
| `success` | `true` if all tests passed, `false` otherwise |

### CI/CD Integration

Use the `success` field and exit code:

```bash
#!/bin/bash
bombardino -config test.json -output json > results.json

if [ $? -eq 0 ]; then
    echo "All tests passed!"
else
    echo "Tests failed!"
    exit 1
fi
```

Or parse the JSON:

```bash
# Using jq
SUCCESS=$(jq -r '.success' results.json)
if [ "$SUCCESS" = "true" ]; then
    echo "All tests passed!"
else
    echo "Tests failed!"
    FAILED=$(jq -r '.assertions.failed' results.json)
    echo "$FAILED assertions failed"
    exit 1
fi
```

## HTML Output

HTML output creates a visual report with charts and interactive elements.

### Usage

```bash
bombardino -config test.json -output html > report.html
# Open in browser
open report.html  # macOS
xdg-open report.html  # Linux
start report.html  # Windows
```

### Features

- **Dark/Light Mode**: Toggle between color schemes
- **Summary Cards**: Quick overview of key metrics
- **Assertions Section**: Color-coded pass/fail indicators
- **Response Time Chart**: Visual bar chart of percentiles
- **Endpoint Breakdown**: Per-test metrics with expandable details
- **Errors Section**: Grouped errors with counts

### Screenshots

The HTML report includes:

1. **Header**: Test suite name and overall status
2. **Summary Stats**: Total requests, success rate, timing metrics
3. **Assertions Panel**: All assertions with pass/fail status
4. **Response Time Chart**: Min, Avg, Max, P50, P95, P99 as bars
5. **Endpoint Cards**: Each test with its own metrics
6. **Errors List**: Any errors that occurred

## Verbose Mode

Add detailed request/response logging to any output format.

### Usage

```bash
bombardino -config test.json -verbose
```

### Output

```
[12:34:56] [a1b2c3d4] REQUEST  POST /api/users
[12:34:56] [a1b2c3d4] Headers: Content-Type: application/json
[12:34:56] [a1b2c3d4] Body: {"name":"Mario","age":30}
[12:34:57] [a1b2c3d4] RESPONSE 201 Created (123ms)
[12:34:57] [a1b2c3d4] Headers: Content-Type: application/json
[12:34:57] [a1b2c3d4] Body: {"id":42,"name":"Mario","age":30}
[12:34:57] [a1b2c3d4] EXTRACT user_id = 42
```

**Request ID** (`a1b2c3d4`): Links requests and responses together.

### When to Use Verbose

- Debugging assertion failures
- Checking request/response bodies
- Verifying variable extraction
- Understanding test flow

## Exit Codes

Bombardino uses exit codes for automation:

| Exit Code | Meaning |
|-----------|---------|
| `0` | All tests passed (all requests got expected status) |
| `1` | Tests failed (status mismatch, errors, or assertion failures) |

### Example

```bash
bombardino -config test.json
if [ $? -eq 0 ]; then
    echo "Deploy to production"
else
    echo "Fix tests first!"
    exit 1
fi
```

## Choosing the Right Format

| Use Case | Recommended Format |
|----------|-------------------|
| Interactive testing | `text` (default) |
| CI/CD pipelines | `json` |
| Sharing reports | `html` |
| Debugging | `text` with `-verbose` |
| Data analysis | `json` |
| Presentations | `html` |

## Combining Options

```bash
# Verbose text for debugging
bombardino -config test.json -verbose

# JSON with verbose (debug logs included)
bombardino -config test.json -output json -verbose > debug.json

# More workers for load testing
bombardino -config test.json -workers 100 -output html > load-test.html
```

## Tips

1. **Use JSON for automation**: Parse it with `jq`, Python, or your CI tool
2. **Save HTML reports**: They're self-contained and easy to share
3. **Use verbose sparingly**: It adds significant output
4. **Check exit codes**: Always verify `$?` in scripts
5. **Redirect to files**: `> report.html` or `> results.json`

## Next Steps

- [Getting Started](getting-started.md) - Run your first test
- [Tutorial: CRUD API](tutorial-crud-api.md) - See all output formats in action
