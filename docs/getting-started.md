# Getting Started

This guide will help you install Bombardino and run your first API test in under 5 minutes.

## Installation

### Option 1: Using Go (Recommended)

If you have Go installed:

```bash
go install github.com/andrearaponi/bombardino/cmd/bombardino@latest
```

### Option 2: Build from Source

```bash
git clone https://github.com/andrearaponi/bombardino.git
cd bombardino
make build
```

This creates a `bombardino` binary in the current directory.

### Option 3: Download Binary

Download the latest release from the [GitHub releases page](https://github.com/andrearaponi/bombardino/releases).

## Verify Installation

Check that Bombardino is installed correctly:

```bash
bombardino -version
```

You should see output like:
```
Bombardino v1.0.0 (commit: abc1234)
Built: 2025-01-27_10:30:00
```

## Your First Test

Let's create a simple test against a public API.

### Step 1: Create a Configuration File

Create a file named `first-test.json`:

```json
{
  "name": "My First Test",
  "description": "Testing a public API",
  "global": {
    "base_url": "https://jsonplaceholder.typicode.com",
    "timeout": "10s",
    "iterations": 5
  },
  "tests": [
    {
      "name": "Get Users",
      "method": "GET",
      "path": "/users",
      "expected_status": [200]
    }
  ]
}
```

This configuration tells Bombardino to:
- Connect to `https://jsonplaceholder.typicode.com`
- Make 5 GET requests to `/users`
- Expect a 200 status code

### Step 2: Run the Test

```bash
bombardino -config first-test.json
```

### Step 3: Understand the Output

You'll see a progress bar during execution, followed by a summary:

```
╔════════════════════════════════════════════════════════════════╗
║                        TEST SUMMARY                            ║
╚════════════════════════════════════════════════════════════════╝

Total Requests:     5
Successful:         5 (100.0%)
Failed:             0 (0.0%)
Total Time:         1.234s
Requests/sec:       4.05

╔════════════════════════════════════════════════════════════════╗
║                      RESPONSE TIMES                            ║
╚════════════════════════════════════════════════════════════════╝

Minimum:            145ms
Average:            189ms
Maximum:            267ms
P50 (Median):       178ms
P95:                245ms
P99:                267ms
```

**What the numbers mean:**
- **Total Requests**: How many requests were sent
- **Successful**: Requests that got the expected status code
- **Failed**: Requests that didn't match expected status or had errors
- **Requests/sec**: Throughput (how many requests per second)
- **P50/P95/P99**: Percentiles - P95 means 95% of requests were faster than this

## Adding More Options

### Use More Workers

By default, Bombardino uses 10 concurrent workers. You can change this:

```bash
bombardino -config first-test.json -workers 50
```

### Run More Iterations

Change `iterations` in your config:

```json
"iterations": 100
```

### Run for a Duration

Instead of a fixed number of iterations, run for a specific time:

```json
{
  "global": {
    "base_url": "https://jsonplaceholder.typicode.com",
    "duration": "30s"
  }
}
```

This runs the test for 30 seconds.

## Next Steps

Now that you've run your first test, explore these guides:

- [Assertions](assertions.md) - Validate more than just status codes
- [Request Chaining](request-chaining.md) - Use data from one request in another
- [Configuration Reference](configuration-reference.md) - See all available options
- [Tutorial: CRUD API](tutorial-crud-api.md) - Complete real-world example

## Common Issues

### "command not found: bombardino"

Make sure the binary is in your PATH:
```bash
# If installed with go install
export PATH=$PATH:$(go env GOPATH)/bin

# Or move the binary
sudo mv bombardino /usr/local/bin/
```

### "connection refused"

The API you're testing isn't running or the URL is wrong. Check:
- Is the server running?
- Is the `base_url` correct?
- Is the port correct?

### "certificate error"

For self-signed certificates, add to your global config:
```json
"insecure_skip_verify": true
```
