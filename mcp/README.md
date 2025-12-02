# Bombardino MCP Server

An MCP (Model Context Protocol) server that enables AI assistants to generate, validate, and run Bombardino API tests.

## Overview

This MCP server provides four tools for AI-powered test generation:

| Tool | Description |
|------|-------------|
| `get_config_schema` | Get the complete JSON schema and examples for Bombardino configurations |
| `validate_config` | Validate a configuration without running tests |
| `run_test` | Execute a Bombardino test and return results |
| `save_config` | Save a configuration to a file |

## Prerequisites

- **Node.js** 18 or later
- **Bombardino** binary installed and accessible in PATH (or set via `BOMBARDINO_PATH`)

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/andrearaponi/bombardino.git
cd bombardino/mcp

# Install dependencies
npm install

# Build
npm run build
```

### Build Bombardino Binary

Make sure the Bombardino binary is built:

```bash
cd ..
make build
```

## Configuration

### Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "bombardino": {
      "command": "node",
      "args": ["/path/to/bombardino/mcp/dist/index.js"],
      "env": {
        "BOMBARDINO_PATH": "/path/to/bombardino/bin/bombardino"
      }
    }
  }
}
```

### Using npx (if published to npm)

```json
{
  "mcpServers": {
    "bombardino": {
      "command": "npx",
      "args": ["bombardino-mcp"],
      "env": {
        "BOMBARDINO_PATH": "/path/to/bombardino"
      }
    }
  }
}
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `BOMBARDINO_PATH` | Path to Bombardino binary | `bombardino` (uses PATH) |

## Tools Reference

### get_config_schema

Returns the complete JSON schema for Bombardino configurations, along with examples and tips.

**Input**: None

**Output**: JSON object containing:
- `bombardino_version`: Current Bombardino version
- `schema`: Full JSON Schema for configurations
- `examples`: Array of example configurations
- `tips`: Best practices for writing tests

**Example Usage**:
```
AI: "What is the format for Bombardino configurations?"
→ Calls get_config_schema to retrieve the schema
```

### validate_config

Validates a Bombardino configuration without executing it.

**Input**:
```json
{
  "config": "{ JSON configuration as string }"
}
```

**Output**:
```json
{
  "valid": true,
  "configName": "My Test Suite",
  "testCount": 5
}
```

Or on error:
```json
{
  "valid": false,
  "error": "test case missing required field: expected_status"
}
```

**Example Usage**:
```
AI: "Is this configuration valid?"
→ Calls validate_config with the configuration
→ Returns validation result
```

### run_test

Executes a Bombardino test configuration and returns results.

**Input**:
```json
{
  "config": "{ JSON configuration as string }",
  "workers": 10,
  "verbose": false
}
```

**Output**:
```json
{
  "success": true,
  "summary": {
    "total_requests": 100,
    "successful_requests": 98,
    "failed_requests": 2,
    "success_rate_percent": 98.0,
    "total_time": "5.234s",
    "avg_response_time": "52.34ms",
    "min_response_time": "12ms",
    "max_response_time": "234ms",
    "p50_response_time": "45ms",
    "p95_response_time": "120ms",
    "p99_response_time": "200ms",
    "requests_per_sec": 19.1
  },
  "endpoints": {
    "Get Users": { ... },
    "Create User": { ... }
  }
}
```

**Example Usage**:
```
AI: "Run this test configuration"
→ Calls run_test with the configuration
→ Returns test results with metrics
```

### save_config

Saves a configuration to a file with pretty formatting.

**Input**:
```json
{
  "config": "{ JSON configuration as string }",
  "path": "/path/to/save/config.json"
}
```

**Output**:
```json
{
  "saved": true,
  "path": "/path/to/save/config.json"
}
```

**Example Usage**:
```
AI: "Save this configuration to tests/api-test.json"
→ Calls save_config with config and path
→ File is created with pretty-printed JSON
```

## Usage Examples

### Generate Test from OpenAPI

```
User: "Here's my OpenAPI spec for a user API. Generate a CRUD test."

AI:
1. Calls get_config_schema to understand the format
2. Generates a configuration based on the OpenAPI spec
3. Calls validate_config to verify correctness
4. If valid, calls save_config to persist
5. Optionally calls run_test to execute
```

### Iterative Test Development

```
User: "Create a load test for my authentication endpoint"

AI:
1. Gets schema via get_config_schema
2. Generates initial configuration
3. Validates with validate_config
4. If errors, fixes and revalidates
5. Runs with run_test to verify
6. Adjusts based on results
```

## Development

### Project Structure

```
mcp/
├── src/
│   ├── index.ts           # MCP server entry point
│   ├── tools/
│   │   ├── getConfigSchema.ts
│   │   ├── validateConfig.ts
│   │   ├── runTest.ts
│   │   └── saveConfig.ts
│   └── utils/
│       └── bombardino.ts  # CLI wrapper utilities
├── dist/                  # Compiled JavaScript
├── package.json
├── tsconfig.json
└── README.md
```

### Building

```bash
npm run build
```

### Running Locally

```bash
# Start the MCP server
node dist/index.js

# The server communicates via stdio
# Send JSON-RPC messages to interact
```

### Testing Tools Manually

```bash
# Test validation
echo '{"jsonrpc":"2.0","method":"tools/call","params":{"name":"validate_config","arguments":{"config":"{\"name\":\"Test\",\"global\":{\"base_url\":\"http://localhost\",\"iterations\":1},\"tests\":[{\"name\":\"Test\",\"method\":\"GET\",\"path\":\"/\",\"expected_status\":[200]}]}"}},"id":1}' | node dist/index.js
```

## Troubleshooting

### "bombardino not found" Error

Set the `BOMBARDINO_PATH` environment variable:

```bash
export BOMBARDINO_PATH=/path/to/bombardino/bin/bombardino
```

Or in Claude Desktop config:
```json
{
  "env": {
    "BOMBARDINO_PATH": "/absolute/path/to/bombardino"
  }
}
```

### Server Not Starting

1. Check Node.js version (requires 18+)
2. Rebuild: `npm run build`
3. Check for TypeScript errors

### Validation Always Fails

1. Ensure Bombardino binary is built: `make build`
2. Test Bombardino directly: `bombardino -version`
3. Check the `-t` flag works: `bombardino -t -config test.json`

### Tests Timeout

Increase the default timeout in your configuration:
```json
{
  "global": {
    "timeout": "60s"
  }
}
```

## License

MIT License - See [LICENSE](../LICENSE) for details.

## Related Documentation

- [AI Generation Guide](../docs/ai-generation.md) - Patterns for AI-powered test generation
- [Configuration Reference](../docs/configuration-reference.md) - Complete field documentation
- [Bombardino README](../README.md) - Main project documentation
