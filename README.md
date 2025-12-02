<div align="center">
  <h1>Bombardino</h1>
  <p>
    <img src="https://img.shields.io/badge/version-1.0.0-green.svg" alt="Version">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
    <img src="https://img.shields.io/badge/Go-1.24+-00ADD8.svg" alt="Go Version">
  </p>
  <p>
    <img src="bombardino.png" alt="Bombardino Logo">
  </p>
  <p>
    <strong>Fast, flexible REST API stress testing tool written in Go.</strong>
  </p>
</div>

## Features

- **Load Testing** - Iteration-based, duration-based, or mixed mode
- **Assertions** - Validate status codes, JSON fields, headers, response times
- **Request Chaining** - Extract values and use them in subsequent requests
- **Test Dependencies** - DAG-based execution order with `depends_on`
- **Data-Driven Testing** - Run tests with multiple data sets
- **Think Time** - Simulate realistic user behavior with pauses
- **Multiple Reports** - Text, JSON, and HTML output formats
- **Concurrent Workers** - Configurable worker pool for high throughput
- **SSL/TLS Support** - Skip verification for self-signed certificates
- **AI-Powered Generation** - MCP server for AI assistants to generate tests

## Quick Start

```bash
# Install
go install github.com/andrearaponi/bombardino/cmd/bombardino@latest

# Or build from source
git clone https://github.com/andrearaponi/bombardino.git
cd bombardino && make build
```

Create `test.json`:
```json
{
  "name": "Quick Test",
  "global": {
    "base_url": "https://jsonplaceholder.typicode.com",
    "iterations": 10
  },
  "tests": [
    {
      "name": "Get Posts",
      "method": "GET",
      "path": "/posts",
      "expected_status": [200],
      "assertions": [
        {"type": "status", "target": "response", "operator": "eq", "value": 200},
        {"type": "response_time", "target": "response", "operator": "lt", "value": "1s"}
      ]
    }
  ]
}
```

Run:
```bash
bombardino -config test.json
```

## Usage

```bash
bombardino -config <file> [options]

Options:
  -config string    Path to JSON configuration file (required)
  -workers int      Number of concurrent workers (default: 10)
  -output string    Output format: text, json, html (default: text)
  -verbose          Enable debug logging
  -t                Validate configuration and exit
  -version          Show version
```

### Examples

```bash
# Basic test
bombardino -config test.json

# Validate configuration (like nginx -t)
bombardino -t -config test.json

# High concurrency
bombardino -config test.json -workers 100

# JSON output for CI/CD
bombardino -config test.json -output json > results.json

# HTML report
bombardino -config test.json -output html > report.html

# Debug mode
bombardino -config test.json -verbose
```

## Documentation

Full documentation is available in the [`docs/`](docs/) folder:

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | Installation and first test |
| [Configuration Reference](docs/configuration-reference.md) | All configuration options |
| [Assertions](docs/assertions.md) | Validate API responses |
| [Request Chaining](docs/request-chaining.md) | Variables, extraction, dependencies |
| [Think Time](docs/think-time.md) | Realistic user simulation |
| [Data-Driven Testing](docs/data-driven-testing.md) | Test with multiple data sets |
| [Output Formats](docs/output-formats.md) | Text, JSON, HTML reports |
| [AI Generation](docs/ai-generation.md) | Generate tests with AI assistants |
| [Tutorial: CRUD API](docs/tutorial-crud-api.md) | Complete walkthrough |

## Example Configuration

```json
{
  "name": "API Test Suite",
  "global": {
    "base_url": "https://jsonplaceholder.typicode.com",
    "timeout": "30s",
    "iterations": 1,
    "headers": {"Content-Type": "application/json"}
  },
  "tests": [
    {
      "name": "Get Post",
      "method": "GET",
      "path": "/posts/1",
      "expected_status": [200],
      "assertions": [
        {"type": "json_path", "target": "id", "operator": "eq", "value": 1},
        {"type": "json_path", "target": "userId", "operator": "exists", "value": ""}
      ],
      "extract": [
        {"name": "post_id", "source": "body", "path": "id"}
      ]
    },
    {
      "name": "Get Post Comments",
      "method": "GET",
      "path": "/posts/${post_id}/comments",
      "expected_status": [200],
      "depends_on": ["Get Post"],
      "assertions": [
        {"type": "json_path", "target": "0.postId", "operator": "eq", "value": 1}
      ]
    }
  ]
}
```

## Project Structure

```
bombardino/
├── cmd/bombardino/     # CLI entry point
├── pkg/
│   ├── config/         # Configuration parser
│   ├── engine/         # Test execution engine
│   ├── variables/      # Variable substitution
│   ├── assertions/     # Assertion system
│   ├── progress/       # Progress bar
│   └── reporter/       # Report generation
├── internal/models/    # Data structures
├── mcp/                # MCP server for AI assistants
├── examples/           # Example configurations
└── docs/               # Documentation
```

## MCP Server (AI Integration)

Bombardino includes an MCP (Model Context Protocol) server that enables AI assistants like Claude to generate, validate, and run tests.

### Setup with Claude Code

```bash
# Build everything
make build

# Add to Claude Code
claude mcp add bombardino \
  -e BOMBARDINO_PATH=$(pwd)/bin/bombardino \
  -- node $(pwd)/mcp/dist/index.js
```

### Available Tools

| Tool | Description |
|------|-------------|
| `get_config_schema` | Get JSON schema and examples |
| `validate_config` | Validate a configuration |
| `run_test` | Execute tests and get results |
| `save_config` | Save configuration to file |

See [mcp/README.md](mcp/README.md) for full documentation.

## Contributing

1. Fork the repo
2. Create your feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) for details.
