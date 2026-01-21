# Tap Compare

Compare API responses between two endpoints to validate migrations, staging environments, or API versioning.

## Use Cases

- **Production vs Staging** - Validate deployments before release
- **API v1 vs v2** - Ensure backward compatibility during upgrades
- **Migration validation** - Compare legacy vs new system responses
- **A/B testing** - Compare different implementations

## Basic Usage

Add `compare_with` to any test:

```json
{
  "name": "Compare Users",
  "method": "GET",
  "path": "/api/users",
  "expected_status": [200],
  "compare_with": {
    "endpoint": "https://staging.api.com",
    "assertions": [
      {"type": "status_match"},
      {"type": "structure_match"}
    ]
  }
}
```

The primary request goes to `base_url` + `path`, and the comparison request goes to `compare_with.endpoint` + `path` (or `compare_with.path` if specified).

## Configuration

### compare_with Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `endpoint` | string | Yes | Base URL of comparison endpoint |
| `path` | string | No | Path override (defaults to primary path) |
| `headers` | object | No | Additional headers for comparison request |
| `timeout` | string | No | Custom timeout (e.g., "30s") |
| `assertions` | array | No | Comparison assertions |
| `ignore_fields` | array | No | Fields to skip during comparison |
| `mode` | string | No | Comparison mode: "full", "partial", "structural" |

## Assertion Types

### status_match

Verifies both endpoints return the same HTTP status code.

```json
{"type": "status_match"}
```

### field_match

Compares specific JSON field values using JSON path syntax.

```json
{"type": "field_match", "target": "data.id"}
{"type": "field_match", "target": "items.0.name"}
{"type": "field_match", "target": "user.email", "operator": "contains"}
```

**Operators:**
- `eq` (default) - Exact match
- `contains` - Primary value contained in compare value

### field_tolerance

Allows numeric values to differ within a tolerance threshold.

```json
{"type": "field_tolerance", "target": "price", "tolerance": 0.05}
{"type": "field_tolerance", "target": "count", "tolerance": "10%"}
```

**Tolerance formats:**
- `0.05` - 5% tolerance (values < 1 are treated as percentages)
- `"10%"` - 10% tolerance (explicit percentage)
- `10` - Absolute tolerance of 10 (values >= 1 are treated as absolute)

### structure_match

Validates that JSON structure is identical (keys and types), ignoring actual values.

```json
{"type": "structure_match"}
```

### response_time_tolerance

Compares response times within a tolerance threshold.

```json
{"type": "response_time_tolerance", "tolerance": 0.20}
```

### header_match

Compares specific HTTP headers between responses. Essential for strangler fig migrations.

```json
{"type": "header_match", "target": "Content-Type"}
{"type": "header_match", "target": "Cache-Control", "operator": "eq"}
{"type": "header_match", "target": "X-Request-Id", "operator": "exists"}
```

**Operators:**
- `eq` (default) - Exact header value match
- `contains` - Primary header value contained in compare value
- `exists` - Header exists in compare response

**Use case - Strangler Fig migration:**
```json
{
  "compare_with": {
    "endpoint": "https://new-microservice.internal",
    "assertions": [
      {"type": "status_match"},
      {"type": "header_match", "target": "Content-Type"},
      {"type": "header_match", "target": "Cache-Control"},
      {"type": "header_match", "target": "ETag"},
      {"type": "structure_match"}
    ]
  }
}
```

### Why Compare Headers?

HTTP headers carry critical metadata that affects client behavior. During migrations (monolith to microservices, framework upgrades, cloud migrations), ensuring header parity is essential for:

1. **Caching consistency** - Different `Cache-Control` or `ETag` headers can cause cache invalidation issues
2. **Content negotiation** - `Content-Type` mismatches break client parsers
3. **Security policies** - CORS headers, CSP, and security headers must match
4. **Rate limiting** - `X-RateLimit-*` headers inform clients about quotas
5. **Debugging** - `X-Request-Id` propagation for distributed tracing

### Common Headers to Compare

| Header | Why It Matters |
|--------|----------------|
| `Content-Type` | Ensures response format matches (application/json vs text/plain) |
| `Cache-Control` | Caching behavior must be identical to avoid stale data |
| `ETag` | Entity tag for conditional requests and cache validation |
| `X-Request-Id` | Verify request tracing is propagated correctly |
| `Access-Control-*` | CORS headers must match for browser clients |
| `X-RateLimit-*` | Rate limit info must be consistent across services |
| `Set-Cookie` | Session handling must be preserved |
| `Location` | Redirect URLs must match |

### Migration Scenarios

**Scenario 1: Monolith to Microservice (Strangler Fig)**

Gradually route traffic from monolith to new microservice while validating equivalence:

```json
{
  "name": "Monolith vs Microservice Validation",
  "global": {
    "base_url": "https://monolith.company.com"
  },
  "tests": [
    {
      "name": "User Service Migration",
      "method": "GET",
      "path": "/api/users/1",
      "expected_status": [200],
      "compare_with": {
        "endpoint": "https://users-service.internal",
        "assertions": [
          {"type": "status_match"},
          {"type": "header_match", "target": "Content-Type"},
          {"type": "header_match", "target": "Cache-Control"},
          {"type": "header_match", "target": "X-Request-Id", "operator": "exists"},
          {"type": "structure_match"},
          {"type": "field_match", "target": "id"},
          {"type": "field_match", "target": "email"}
        ],
        "ignore_fields": ["timestamp", "generated_at"]
      }
    }
  ]
}
```

**Scenario 2: Framework Migration (e.g., Spring to Go)**

Verify header behavior when rewriting services:

```json
{
  "compare_with": {
    "endpoint": "https://new-go-service.internal",
    "assertions": [
      {"type": "header_match", "target": "Content-Type"},
      {"type": "header_match", "target": "Content-Encoding"},
      {"type": "header_match", "target": "Transfer-Encoding", "operator": "exists"}
    ]
  }
}
```

**Scenario 3: CDN/Proxy Migration**

When moving from one CDN to another, validate caching headers:

```json
{
  "compare_with": {
    "endpoint": "https://new-cdn.example.com",
    "assertions": [
      {"type": "header_match", "target": "Cache-Control"},
      {"type": "header_match", "target": "Vary"},
      {"type": "header_match", "target": "Age", "operator": "exists"},
      {"type": "header_match", "target": "X-Cache", "operator": "exists"}
    ]
  }
}
```

### Header Comparison Best Practices

1. **Start with critical headers** - `Content-Type` and `Cache-Control` are usually the most important
2. **Use `exists` for optional headers** - Some headers may be present in one system but not another
3. **Ignore timestamps** - Headers like `Date` change on every request
4. **Check CORS early** - CORS mismatches break browser clients silently
5. **Validate security headers** - `X-Frame-Options`, `X-XSS-Protection`, CSP headers

## Ignoring Fields

Skip dynamic fields like timestamps or request IDs that change between requests:

```json
"ignore_fields": ["timestamp", "request_id", "meta.generated_at"]
```

Supports nested paths using dot notation.

## Complete Example

```json
{
  "name": "Production vs Staging Validation",
  "global": {
    "base_url": "https://api.production.com",
    "timeout": "30s",
    "iterations": 5
  },
  "tests": [
    {
      "name": "Compare Users API",
      "method": "GET",
      "path": "/api/v1/users",
      "expected_status": [200],
      "compare_with": {
        "endpoint": "https://api.staging.com",
        "ignore_fields": ["timestamp", "request_id"],
        "assertions": [
          {"type": "status_match"},
          {"type": "field_match", "target": "data.0.id"},
          {"type": "field_match", "target": "pagination.total"},
          {"type": "field_tolerance", "target": "meta.count", "tolerance": 0.10},
          {"type": "structure_match"}
        ]
      }
    },
    {
      "name": "Compare User Details",
      "method": "GET",
      "path": "/api/v1/users/1",
      "expected_status": [200],
      "compare_with": {
        "endpoint": "https://api.staging.com",
        "assertions": [
          {"type": "status_match"},
          {"type": "field_match", "target": "id"},
          {"type": "field_match", "target": "email"}
        ]
      }
    }
  ]
}
```

## Output

Comparison results appear in all output formats.

### Text Output

```
COMPARISONS
------------------------------------------------------------
Total Comparisons:   10
Passed:              8 (80.0%)
Failed:              2 (20.0%)
```

### JSON Output

Includes `comparison_result` object with detailed diff information:

```json
{
  "comparison_result": {
    "success": false,
    "primary_response": {
      "status_code": 200,
      "response_time": "45ms"
    },
    "compare_response": {
      "status_code": 200,
      "response_time": "52ms"
    },
    "field_diffs": [
      {
        "path": "data.count",
        "primary_value": 100,
        "compare_value": 95,
        "type": "value_mismatch"
      }
    ]
  }
}
```

### HTML Report

Dedicated "Tap Compare Results" section with:
- Visual pass/fail indicators
- Progress bar showing comparison success rate
- Detailed diff information per endpoint

## Comparison Modes

### full (default)

Compares the entire response body field by field.

### partial

Only compares fields present in the primary response.

### structural

Compares JSON structure only, ignoring actual values.

## Tips

1. **Start simple** - Begin with `status_match` and `structure_match`, then add specific field assertions.

2. **Use ignore_fields** - Dynamic fields like timestamps should always be ignored.

3. **Combine with assertions** - Tap compare works alongside regular assertions. Both must pass for the test to succeed.

4. **Different paths** - Use `compare_with.path` to compare different API versions:
   ```json
   {
     "path": "/api/v2/users",
     "compare_with": {
       "endpoint": "https://api.example.com",
       "path": "/api/v1/users"
     }
   }
   ```

5. **Custom headers** - Add authentication or version headers for the comparison endpoint:
   ```json
   {
     "compare_with": {
       "endpoint": "https://staging.api.com",
       "headers": {
         "X-API-Version": "2.0",
         "Authorization": "Bearer staging-token"
       }
     }
   }
   ```
