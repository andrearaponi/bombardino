# Configuration Reference

Complete guide to all Bombardino configuration fields.

---

## File Structure

Bombardino uses JSON configuration files with this structure:

```json
{
  "name": "Test Suite Name",
  "description": "Optional description",
  "global": {
    // Default settings for all tests
  },
  "tests": [
    // Array of test definitions
  ]
}
```

---

## Top-Level Fields

### `name` (required)

**Type:** `string`

Identifier name for the test suite. Displayed in reports and logs.

```json
{
  "name": "API Stress Test - Production"
}
```

**Notes:**
- Must be non-empty
- Used to identify the test suite in reports
- Recommendation: use descriptive names that indicate the test purpose

---

### `description` (optional)

**Type:** `string`
**Default:** `""`

Text description of the test suite. Useful for documenting what the tests do.

```json
{
  "name": "User API Tests",
  "description": "Load testing for user APIs - target 1000 req/s"
}
```

---

### `global` (required)

**Type:** `object`

Contains default settings applied to all tests. Each test can override these settings.

See [Global Settings](#global-settings) section for all available fields.

---

### `tests` (required)

**Type:** `array`

Array of test objects. Must contain at least one test.

```json
{
  "tests": [
    { "name": "Test 1", ... },
    { "name": "Test 2", ... }
  ]
}
```

See [Test Settings](#test-settings) section for all available fields.

---

## Global Settings

Settings in the `global` section that apply to all tests.

### `base_url` (required)

**Type:** `string`

Base URL for all requests. Test paths are concatenated to this URL.

```json
{
  "global": {
    "base_url": "https://api.example.com"
  },
  "tests": [
    {
      "path": "/users"  // Results in: https://api.example.com/users
    }
  ]
}
```

**Notes:**
- Should not end with `/` (handled automatically)
- Can include port: `http://localhost:8080`
- Supports HTTPS with valid or self-signed certificates (see `insecure_skip_verify`)

---

### `timeout` (optional)

**Type:** `duration`
**Default:** `30s`

Maximum timeout for each HTTP request. If the request doesn't complete within this time, it's considered failed.

```json
{
  "global": {
    "timeout": "10s"
  }
}
```

**Duration format:**
| Value | Meaning |
|-------|---------|
| `100ms` | 100 milliseconds |
| `1s` | 1 second |
| `1.5s` | 1.5 seconds |
| `1m` | 1 minute |
| `1m30s` | 1 minute and 30 seconds |
| `1h` | 1 hour |

**Notes:**
- Includes connection time + response time
- If omitted, defaults to 30 seconds
- Can be overridden per test

---

### `delay` (optional)

**Type:** `duration`
**Default:** `0`

Fixed delay between one request and the next. Useful for rate limiting.

```json
{
  "global": {
    "delay": "100ms"  // Max ~10 requests/second per worker
  }
}
```

**Difference with `think_time`:**
- `delay`: Applied between every request, used for rate limiting
- `think_time`: Simulates user pause, applied after each completed test

**Notes:**
- With 10 workers and delay=100ms, you get ~100 theoretical req/s
- Very low delays can saturate the target server

---

### `iterations` (conditional)

**Type:** `integer`
**Default:** `1`

Number of times each test is executed. Required if `duration` is not specified.

```json
{
  "global": {
    "iterations": 100  // Each test executed 100 times
  }
}
```

**Notes:**
- If `iterations: 0` and no `duration`, validation fails
- Can be overridden per test
- In mixed mode (with duration), the test stops when the first limit is reached

---

### `duration` (conditional)

**Type:** `duration`
**Default:** none

Total test duration. Tests are executed repeatedly until time expires.

```json
{
  "global": {
    "duration": "5m"  // Run tests for 5 minutes
  }
}
```

**Notes:**
- Alternative to `iterations` for prolonged load testing
- If specified together with `iterations`, stops at first limit reached (mixed mode)
- You cannot know the exact number of requests in advance

---

### `headers` (optional)

**Type:** `object` (map string → string)
**Default:** `{}`

HTTP headers added to all requests. Tests can add additional headers.

```json
{
  "global": {
    "headers": {
      "Content-Type": "application/json",
      "Authorization": "Bearer eyJhbGciOiJIUzI1NiIs...",
      "X-Api-Key": "my-api-key",
      "Accept-Language": "en-US"
    }
  }
}
```

**Notes:**
- Test headers are ADDED to these (not replaced)
- If a test defines the same header, it overrides the global value
- Supports variables: `"Authorization": "Bearer ${token}"`

---

### `insecure_skip_verify` (optional)

**Type:** `boolean`
**Default:** `false`

Disables TLS certificate verification. Useful for development environments with self-signed certificates.

```json
{
  "global": {
    "base_url": "https://dev.internal.company.com",
    "insecure_skip_verify": true
  }
}
```

**Warning:**
- Do NOT use in production
- Exposes to man-in-the-middle attacks
- Use only for internal tests or development

---

### `variables` (optional)

**Type:** `object` (map string → any)
**Default:** `{}`

Global variables available in all tests. Accessible with `${name}` syntax.

```json
{
  "global": {
    "variables": {
      "api_version": "v2",
      "default_limit": 100,
      "admin_email": "admin@test.com"
    }
  },
  "tests": [
    {
      "path": "/api/${api_version}/users?limit=${default_limit}"
    }
  ]
}
```

**Supported types:**
- Strings: `"value"`
- Numbers: `123` or `45.67`
- Booleans: `true` or `false`

**Notes:**
- Variables extracted from tests (with `extract`) override global ones
- Useful for parameterizing configurations

---

### `think_time` (optional)

**Type:** `duration`
**Default:** `0`

Fixed pause after each test iteration. Simulates the "thinking time" of a real user.

```json
{
  "global": {
    "think_time": "500ms"  // Pause 500ms after each test
  }
}
```

**Typical use:**
- Simulate real users who don't hammer the server
- Make the test more realistic
- Reduce load spikes

---

### `think_time_min` and `think_time_max` (optional)

**Type:** `duration`
**Default:** none

Define a range for random think time. More realistic than a fixed value.

```json
{
  "global": {
    "think_time_min": "200ms",
    "think_time_max": "2s"
  }
}
```

**Notes:**
- If defined, `think_time` is ignored
- The value is randomly chosen within the range at each iteration
- Both must be specified together

---

## Test Settings

Each object in the `tests` array supports these fields.

### `name` (required)

**Type:** `string`

Unique test name. Used in reports, logs, and for dependencies.

```json
{
  "name": "Create User"
}
```

**Notes:**
- Must be unique within the test suite
- Used in `depends_on` to reference this test
- Displayed in reports to identify results

---

### `method` (required)

**Type:** `string`

HTTP method to use.

```json
{
  "method": "POST"
}
```

**Supported values:**
- `GET` - Retrieve resources
- `POST` - Create new resources
- `PUT` - Update resources (full replacement)
- `PATCH` - Update resources (partial)
- `DELETE` - Delete resources
- `HEAD` - Like GET but without body
- `OPTIONS` - Get communication options

---

### `path` (required)

**Type:** `string`

URL path to concatenate to `base_url`. Can contain variables.

```json
{
  "path": "/api/users/${user_id}/posts"
}
```

**Examples:**
```json
// Simple path
"path": "/api/users"

// With query parameters
"path": "/api/users?page=1&limit=10"

// With variables
"path": "/api/users/${user_id}"

// With multiple variables
"path": "/api/${version}/users/${user_id}/posts/${post_id}"
```

**Notes:**
- Should not start with `/` if `base_url` already ends with `/`
- Unresolved variables cause an error
- Supports query parameters

---

### `expected_status` (required)

**Type:** `array` of `integer`

HTTP status codes considered "success". If the response has a different status, the test fails.

```json
{
  "expected_status": [200, 201]
}
```

**Common examples:**
```json
// Only 200 OK
"expected_status": [200]

// Resource creation
"expected_status": [201]

// GET can return 200 or 404
"expected_status": [200, 404]

// DELETE can be 200 or 204
"expected_status": [200, 204]
```

**Notes:**
- At least one status code required
- The request is considered "success" if the status is in the list
- Use assertions for more sophisticated validations

---

### `headers` (optional)

**Type:** `object`
**Default:** `{}`

Additional HTTP headers for this test. Merged with global headers.

```json
{
  "headers": {
    "X-Request-ID": "test-123",
    "Accept": "application/xml"
  }
}
```

**Merge behavior:**
```json
// Global
"headers": {"Content-Type": "application/json", "Authorization": "Bearer abc"}

// Test
"headers": {"Authorization": "Bearer xyz", "X-Custom": "value"}

// Result: Content-Type: application/json
//         Authorization: Bearer xyz (overridden)
//         X-Custom: value (added)
```

---

### `body` (optional)

**Type:** `object` | `string` | `array`
**Default:** none

HTTP request body. If it's an object or array, it's automatically serialized to JSON.

```json
// Object (converted to JSON)
{
  "body": {
    "name": "Mario",
    "email": "mario@test.com",
    "age": 30
  }
}

// String (sent as-is)
{
  "body": "raw string body"
}

// With variables
{
  "body": {
    "user_id": "${extracted_id}",
    "name": "${data.name}"
  }
}
```

**Notes:**
- If `body` is an object, `Content-Type: application/json` is automatically added
- Supports variables with `${name}`
- For non-JSON bodies, use a string

---

### `timeout` (optional)

**Type:** `duration`
**Default:** global value

Specific timeout for this test. Overrides the global timeout.

```json
{
  "name": "Slow Report Generation",
  "timeout": "2m"  // This endpoint is slow
}
```

---

### `delay` (optional)

**Type:** `duration`
**Default:** global value

Specific delay for this test.

```json
{
  "name": "Rate Limited API",
  "delay": "1s"  // This endpoint has a low rate limit
}
```

---

### `iterations` (optional)

**Type:** `integer`
**Default:** global value

Number of iterations for this specific test.

```json
{
  "name": "Heavy Test",
  "iterations": 10  // Only 10 iterations for this expensive test
}
```

---

### `duration` (optional)

**Type:** `duration`
**Default:** global value

Specific duration for this test.

```json
{
  "name": "Endurance Test",
  "duration": "10m"
}
```

---

### `assertions` (optional)

**Type:** `array`
**Default:** `[]`

Validations to perform on the response. See [Assertions](#assertions-1) section for details.

```json
{
  "assertions": [
    {"type": "status", "target": "response", "operator": "eq", "value": 200},
    {"type": "json_path", "target": "name", "operator": "eq", "value": "Mario"},
    {"type": "response_time", "target": "response", "operator": "lt", "value": "500ms"}
  ]
}
```

---

### `extract` (optional)

**Type:** `array`
**Default:** `[]`

Values to extract from the response for use in subsequent tests.

```json
{
  "extract": [
    {"name": "user_id", "source": "body", "path": "id"},
    {"name": "auth_token", "source": "header", "path": "Authorization"}
  ]
}
```

See [Extraction](#extraction) section for details.

---

### `depends_on` (optional)

**Type:** `array` of `string`
**Default:** `[]`

List of test names that must complete before this one.

```json
{
  "name": "Get User",
  "depends_on": ["Create User"],
  "path": "/api/users/${user_id}"
}
```

**Behavior:**
- Tests without dependencies run in parallel
- Tests with dependencies wait for all dependencies to complete
- If a dependency fails, dependent tests are **skipped**
- Variables extracted from dependencies are available

**DAG Example:**
```
Create User ──────┬──── Get User ──── Delete User
                  │
Create Post ──────┘
```

```json
[
  {"name": "Create User", ...},
  {"name": "Create Post", "depends_on": ["Create User"]},
  {"name": "Get User", "depends_on": ["Create User", "Create Post"]},
  {"name": "Delete User", "depends_on": ["Get User"]}
]
```

---

### `insecure_skip_verify` (optional)

**Type:** `boolean`
**Default:** global value

Override of the TLS setting for this test.

```json
{
  "name": "Internal API Call",
  "insecure_skip_verify": true
}
```

---

### `think_time`, `think_time_min`, `think_time_max` (optional)

Override of global think times for this test.

```json
{
  "name": "User Browse",
  "think_time_min": "2s",
  "think_time_max": "5s"
}
```

---

### `data` (optional)

**Type:** `array` of `object`
**Default:** none

Inline data for data-driven testing. The test is executed once for each element.

```json
{
  "name": "Create Multiple Users",
  "data": [
    {"name": "Mario", "email": "mario@test.com"},
    {"name": "Luigi", "email": "luigi@test.com"},
    {"name": "Peach", "email": "peach@test.com"}
  ],
  "body": {
    "name": "${data.name}",
    "email": "${data.email}"
  }
}
```

**Notes:**
- Each data row generates a separate request
- Accessible with `${data.field}`
- Types are preserved (numbers stay numbers)

---

### `data_file` (optional)

**Type:** `string`
**Default:** none

Path to external file with data. Alternative to inline `data`.

```json
{
  "data_file": "users.json"
}
```

**Supported formats:**

**JSON** (`users.json`):
```json
[
  {"name": "Mario", "age": 30},
  {"name": "Luigi", "age": 28}
]
```

**CSV** (`users.csv`):
```csv
name,age
Mario,30
Luigi,28
```

---

## Assertions

Assertions validate responses beyond simple status codes.

### Assertion Structure

```json
{
  "type": "assertion_type",
  "target": "what_to_check",
  "operator": "how_to_compare",
  "value": "expected_value"
}
```

### Assertion Types

#### `status`

Validates the HTTP status code.

```json
{"type": "status", "target": "response", "operator": "eq", "value": 200}
{"type": "status", "target": "response", "operator": "gte", "value": 200}
{"type": "status", "target": "response", "operator": "lt", "value": 300}
```

---

#### `json_path`

Validates values in the JSON body. The `target` is the JSON path.

```json
// Simple field
{"type": "json_path", "target": "id", "operator": "exists", "value": ""}

// Nested field
{"type": "json_path", "target": "user.name", "operator": "eq", "value": "Mario"}

// Array
{"type": "json_path", "target": "items.0.id", "operator": "eq", "value": 1}

// Array length
{"type": "json_path", "target": "items", "operator": "gt", "value": 0}
```

**Path syntax:**
- `field` - Root field
- `parent.child` - Nested field
- `array.0` - First array element
- `array.0.field` - Field of first element

---

#### `response_time`

Validates response time.

```json
{"type": "response_time", "target": "response", "operator": "lt", "value": "500ms"}
{"type": "response_time", "target": "response", "operator": "lte", "value": "1s"}
```

**Notes:**
- The value is a duration string
- Useful for SLA (Service Level Agreement)

---

#### `header`

Validates response headers. The `target` is the header name.

```json
{"type": "header", "target": "Content-Type", "operator": "contains", "value": "json"}
{"type": "header", "target": "X-Request-Id", "operator": "exists", "value": ""}
{"type": "header", "target": "Cache-Control", "operator": "eq", "value": "no-cache"}
```

---

#### `body_size`

Validates body size in bytes.

```json
{"type": "body_size", "target": "response", "operator": "lt", "value": 10000}
{"type": "body_size", "target": "response", "operator": "gt", "value": 0}
```

---

### Operators

| Operator | Description | Compatible Types |
|----------|-------------|------------------|
| `eq` | Equals | All |
| `neq` | Not equals | All |
| `gt` | Greater than | Numbers, duration |
| `gte` | Greater than or equal | Numbers, duration |
| `lt` | Less than | Numbers, duration |
| `lte` | Less than or equal | Numbers, duration |
| `contains` | Contains substring | Strings |
| `starts_with` | Starts with | Strings |
| `ends_with` | Ends with | Strings |
| `exists` | Field exists | All (value ignored) |
| `not_exists` | Field doesn't exist | All (value ignored) |
| `matches` | Regex match | Strings |

---

### Complete Examples

```json
"assertions": [
  // Exact status code
  {"type": "status", "target": "response", "operator": "eq", "value": 201},

  // ID field must exist
  {"type": "json_path", "target": "id", "operator": "exists", "value": ""},

  // Name must match what was sent
  {"type": "json_path", "target": "name", "operator": "eq", "value": "Mario"},

  // Email must contain @
  {"type": "json_path", "target": "email", "operator": "contains", "value": "@"},

  // Response under 500ms
  {"type": "response_time", "target": "response", "operator": "lt", "value": "500ms"},

  // Correct Content-Type header
  {"type": "header", "target": "Content-Type", "operator": "contains", "value": "json"},

  // Body not too large
  {"type": "body_size", "target": "response", "operator": "lt", "value": 50000},

  // Email regex match
  {"type": "json_path", "target": "email", "operator": "matches", "value": "^[a-z]+@[a-z]+\\.[a-z]+$"}
]
```

---

## Extraction

Extract values from responses to use in subsequent tests.

### Structure

```json
{
  "name": "variable_name",
  "source": "body|header|status",
  "path": "extraction_path"
}
```

### Fields

| Field | Description |
|-------|-------------|
| `name` | Variable name (used as `${name}`) |
| `source` | Where to extract: `body`, `header`, `status` |
| `path` | For `body`: JSON path. For `header`: header name. For `status`: ignored |

### Examples

```json
"extract": [
  // ID from JSON body
  {"name": "user_id", "source": "body", "path": "id"},

  // Nested token
  {"name": "token", "source": "body", "path": "data.auth.token"},

  // First array element
  {"name": "first_item_id", "source": "body", "path": "items.0.id"},

  // Location header (for redirects)
  {"name": "redirect_url", "source": "header", "path": "Location"},

  // Status code as variable
  {"name": "status", "source": "status", "path": ""}
]
```

### Using Extracted Variables

```json
{
  "tests": [
    {
      "name": "Create User",
      "method": "POST",
      "path": "/api/users",
      "extract": [
        {"name": "user_id", "source": "body", "path": "id"}
      ]
    },
    {
      "name": "Get User",
      "depends_on": ["Create User"],
      "method": "GET",
      "path": "/api/users/${user_id}"  // Uses extracted ID
    }
  ]
}
```

---

## Test Modes

### Iteration Mode

Run a fixed number of requests.

```json
{
  "global": {
    "iterations": 100
  }
}
```

**Pros:** Predictable results, easy to compare
**Cons:** Variable duration

---

### Duration Mode

Run for a specific time.

```json
{
  "global": {
    "duration": "5m"
  }
}
```

**Pros:** Predictable duration, simulates constant load
**Cons:** Variable number of requests

---

### Mixed Mode

Stop when either limit is reached.

```json
{
  "global": {
    "iterations": 10000,
    "duration": "5m"
  }
}
```

**Behavior:** Runs up to 10000 requests OR 5 minutes (whichever comes first)

---

## Override Hierarchy

Test settings override global settings:

```
Global defaults
    ↓
Test-level overrides (win)
```

```json
{
  "global": {
    "timeout": "30s",
    "iterations": 100
  },
  "tests": [
    {
      "name": "Fast Test"
      // Uses timeout: 30s, iterations: 100 (global)
    },
    {
      "name": "Slow Test",
      "timeout": "2m",      // Override: 2 minutes
      "iterations": 10       // Override: only 10 iterations
    }
  ]
}
```

---

## Command-Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-config` | Required | Path to configuration file |
| `-workers` | `10` | Number of concurrent workers |
| `-output` | `text` | Output format: `text`, `json`, `html` |
| `-verbose` | `false` | Enable detailed logging |
| `-t` | - | Validate configuration and exit (like `nginx -t`) |
| `-version` | - | Show version |

### Examples

```bash
# Basic test
bombardino -config test.json

# Validate configuration
bombardino -t -config test.json

# High load
bombardino -config test.json -workers 100

# JSON output for CI/CD
bombardino -config test.json -output json > results.json

# HTML report
bombardino -config test.json -output html > report.html

# Debug
bombardino -config test.json -verbose
```

---

## Complete Example

```json
{
  "name": "Complete API Test Suite",
  "description": "Full test suite for user APIs",
  "global": {
    "base_url": "http://localhost:8080",
    "timeout": "30s",
    "delay": "50ms",
    "iterations": 1,
    "think_time_min": "100ms",
    "think_time_max": "500ms",
    "headers": {
      "Content-Type": "application/json",
      "Accept": "application/json"
    },
    "variables": {
      "api_version": "v1"
    }
  },
  "tests": [
    {
      "name": "Health Check",
      "method": "GET",
      "path": "/health",
      "expected_status": [200],
      "assertions": [
        {"type": "status", "target": "response", "operator": "eq", "value": 200},
        {"type": "response_time", "target": "response", "operator": "lt", "value": "100ms"}
      ]
    },
    {
      "name": "Create User",
      "method": "POST",
      "path": "/api/${api_version}/users",
      "expected_status": [201],
      "body": {
        "name": "Mario Rossi",
        "email": "mario@test.com",
        "age": 30
      },
      "assertions": [
        {"type": "json_path", "target": "id", "operator": "exists", "value": ""},
        {"type": "json_path", "target": "name", "operator": "eq", "value": "Mario Rossi"}
      ],
      "extract": [
        {"name": "user_id", "source": "body", "path": "id"}
      ]
    },
    {
      "name": "Get User",
      "method": "GET",
      "path": "/api/${api_version}/users/${user_id}",
      "expected_status": [200],
      "depends_on": ["Create User"],
      "assertions": [
        {"type": "json_path", "target": "id", "operator": "eq", "value": "${user_id}"},
        {"type": "json_path", "target": "email", "operator": "eq", "value": "mario@test.com"}
      ]
    },
    {
      "name": "Update User",
      "method": "PUT",
      "path": "/api/${api_version}/users/${user_id}",
      "expected_status": [200],
      "depends_on": ["Get User"],
      "body": {
        "name": "Mario Rossi Updated",
        "email": "mario.updated@test.com",
        "age": 31
      }
    },
    {
      "name": "Delete User",
      "method": "DELETE",
      "path": "/api/${api_version}/users/${user_id}",
      "expected_status": [204],
      "depends_on": ["Update User"]
    },
    {
      "name": "Verify Deletion",
      "method": "GET",
      "path": "/api/${api_version}/users/${user_id}",
      "expected_status": [404],
      "depends_on": ["Delete User"]
    }
  ]
}
```

---

## Validation

Use the `-t` flag to validate configuration without running tests:

```bash
$ bombardino -t -config test.json
✅ Configuration valid: Complete API Test Suite (6 tests)

$ bombardino -t -config invalid.json
❌ Configuration invalid: test 3: name is required
```

**What gets validated:**
- Required fields present
- Correct duration format
- At least one test defined
- Iterations or duration > 0
- Valid JSON structure
