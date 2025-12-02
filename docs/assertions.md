# Assertions Guide

Assertions let you validate API responses beyond just checking the status code. You can verify response content, check headers, measure performance, and more.

## Why Use Assertions?

A 200 OK status doesn't mean your API is working correctly. Consider these scenarios:

- The API returns 200, but the JSON is empty
- The API returns 200, but a required field is missing
- The API returns 200, but it takes 5 seconds
- The API returns 200, but a security header is missing

Assertions catch these issues.

## Assertion Structure

Every assertion has four parts:

```json
{
  "type": "json_path",
  "target": "user.name",
  "operator": "eq",
  "value": "Mario"
}
```

| Field | Description |
|-------|-------------|
| `type` | What to check (status, json_path, header, etc.) |
| `target` | Where to look (depends on type) |
| `operator` | How to compare (eq, contains, exists, etc.) |
| `value` | What to expect |

## Assertion Types

### 1. Status Code (`status`)

Check the HTTP status code:

```json
{
  "type": "status",
  "target": "response",
  "operator": "eq",
  "value": 200
}
```

You can also check for ranges:

```json
{
  "type": "status",
  "target": "response",
  "operator": "lt",
  "value": 300
}
```

### 2. JSON Path (`json_path`)

Check values in the JSON response body. The `target` is the path to the field.

**Simple field:**
```json
{
  "type": "json_path",
  "target": "name",
  "operator": "eq",
  "value": "Mario"
}
```

For response: `{"name": "Mario", "age": 30}`

**Nested field:**
```json
{
  "type": "json_path",
  "target": "user.address.city",
  "operator": "eq",
  "value": "Rome"
}
```

For response: `{"user": {"address": {"city": "Rome"}}}`

**Array element:**
```json
{
  "type": "json_path",
  "target": "0.id",
  "operator": "exists",
  "value": ""
}
```

For response: `[{"id": 1, "name": "Mario"}, {"id": 2, "name": "Luigi"}]`

**Check field exists:**
```json
{
  "type": "json_path",
  "target": "id",
  "operator": "exists",
  "value": ""
}
```

### 3. Response Time (`response_time`)

Ensure your API responds fast enough:

```json
{
  "type": "response_time",
  "target": "response",
  "operator": "lt",
  "value": "500ms"
}
```

Common thresholds:
- Fast APIs: `< 100ms`
- Normal APIs: `< 500ms`
- Slow operations: `< 2000ms`

### 4. Header (`header`)

Check response headers:

```json
{
  "type": "header",
  "target": "Content-Type",
  "operator": "contains",
  "value": "application/json"
}
```

Check security headers:
```json
{
  "type": "header",
  "target": "X-Frame-Options",
  "operator": "exists",
  "value": ""
}
```

### 5. Body Size (`body_size`)

Check the response size in bytes:

```json
{
  "type": "body_size",
  "target": "response",
  "operator": "lt",
  "value": 10000
}
```

Useful for:
- Ensuring responses aren't too large
- Detecting empty responses
- Checking pagination works

## Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `eq` | Equals | `"eq", 200` |
| `neq` | Not equals | `"neq", 500` |
| `gt` | Greater than | `"gt", 0` |
| `gte` | Greater than or equal | `"gte", 1` |
| `lt` | Less than | `"lt", 500` |
| `lte` | Less than or equal | `"lte", 100` |
| `contains` | String contains | `"contains", "error"` |
| `starts_with` | String starts with | `"starts_with", "user_"` |
| `ends_with` | String ends with | `"ends_with", "@test.com"` |
| `exists` | Field exists | `"exists", ""` |
| `not_exists` | Field doesn't exist | `"not_exists", ""` |
| `matches` | Regex match | `"matches", "^[0-9]+$"` |

## Real Example: Testing Person API

Here's a complete test that creates a person and validates the response:

```json
{
  "name": "Create Person Test",
  "global": {
    "base_url": "http://localhost:8080",
    "iterations": 1
  },
  "tests": [
    {
      "name": "Create Person",
      "method": "POST",
      "path": "/api/persons",
      "expected_status": [201],
      "headers": {
        "Content-Type": "application/json"
      },
      "body": {
        "name": "Mario",
        "surname": "Rossi",
        "age": 30,
        "mail": "mario@test.com"
      },
      "assertions": [
        {
          "type": "status",
          "target": "response",
          "operator": "eq",
          "value": 201
        },
        {
          "type": "json_path",
          "target": "id",
          "operator": "exists",
          "value": ""
        },
        {
          "type": "json_path",
          "target": "name",
          "operator": "eq",
          "value": "Mario"
        },
        {
          "type": "json_path",
          "target": "surname",
          "operator": "eq",
          "value": "Rossi"
        },
        {
          "type": "json_path",
          "target": "age",
          "operator": "eq",
          "value": 30
        },
        {
          "type": "response_time",
          "target": "response",
          "operator": "lt",
          "value": "500ms"
        }
      ]
    }
  ]
}
```

## Viewing Assertion Results

### Text Output

```
╔════════════════════════════════════════════════════════════════╗
║                        ASSERTIONS                              ║
╚════════════════════════════════════════════════════════════════╝

Total: 6    Passed: 6    Failed: 0

✅ status eq 201
✅ json_path id exists
✅ json_path name eq Mario
✅ json_path surname eq Rossi
✅ json_path age eq 30
✅ response_time lt 500ms
```

### HTML Output

The HTML report shows assertions with color-coded indicators and progress bars.

## Tips

1. **Start simple**: Begin with status code assertions, then add more
2. **Check required fields**: Use `exists` to ensure important fields are present
3. **Be specific**: Use `eq` for exact matches, `contains` for partial matches
4. **Set realistic thresholds**: Don't set response time too low for complex operations
5. **Test failure cases**: Check that errors return correct status codes and messages

## Next Steps

- [Request Chaining](request-chaining.md) - Use response data in other tests
- [Data-Driven Testing](data-driven-testing.md) - Run assertions with different data
- [Configuration Reference](configuration-reference.md) - All assertion options
