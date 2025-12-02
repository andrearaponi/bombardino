# Bombardino Documentation

Bombardino is a fast, flexible REST API stress testing tool written in Go. It helps you test your APIs for performance, validate responses, and simulate real user behavior.

## What Can Bombardino Do?

- **Load Testing**: Send thousands of concurrent requests to your API
- **Response Validation**: Check status codes, JSON fields, headers, and response times
- **Request Chaining**: Use data from one request in the next (like getting an ID after creating a resource)
- **Realistic Simulation**: Add think time between requests to simulate real users
- **Data-Driven Testing**: Run the same test with different data sets
- **Multiple Reports**: Get results as text, JSON, or beautiful HTML reports

## Quick Example

Here's a simple test that checks if your API is responding:

```json
{
  "name": "Health Check",
  "global": {
    "base_url": "http://localhost:8080",
    "iterations": 10
  },
  "tests": [
    {
      "name": "Check API Health",
      "method": "GET",
      "path": "/health",
      "expected_status": [200]
    }
  ]
}
```

Run it:
```bash
bombardino -config health-check.json
```

## Documentation Guides

| Guide | Description |
|-------|-------------|
| [Getting Started](getting-started.md) | Install Bombardino and run your first test |
| [Configuration Reference](configuration-reference.md) | All configuration options explained |
| [Assertions](assertions.md) | Validate responses beyond status codes |
| [Request Chaining](request-chaining.md) | Variables, extraction, and test dependencies |
| [Think Time](think-time.md) | Simulate realistic user behavior |
| [Data-Driven Testing](data-driven-testing.md) | Test with multiple data sets |
| [Output Formats](output-formats.md) | Text, JSON, and HTML reports |
| [Tutorial: CRUD API](tutorial-crud-api.md) | Complete walkthrough with real examples |

## Installation

```bash
# Using Go
go install github.com/andrearaponi/bombardino/cmd/bombardino@latest

# Or build from source
git clone https://github.com/andrearaponi/bombardino.git
cd bombardino
make build
```

## Getting Help

- Run `bombardino -help` for command-line options
- Check the [examples/](../examples/) folder for sample configurations
- Open an issue on [GitHub](https://github.com/andrearaponi/bombardino/issues)
