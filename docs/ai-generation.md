# AI-Powered Test Generation with Bombardino

This guide explains how to use AI assistants (Claude, GPT, etc.) to generate Bombardino test configurations from OpenAPI specifications and user stories.

## Overview

Bombardino includes an MCP (Model Context Protocol) server that allows AI assistants to:

1. **Get Configuration Schema** - Understand the complete Bombardino configuration format
2. **Validate Configurations** - Test configurations before running them
3. **Run Tests** - Execute tests and get results
4. **Save Configurations** - Persist configurations to files

This enables a powerful workflow where you can describe what you want to test in natural language, and the AI generates valid, runnable Bombardino configurations.

---

## The AI Generation Workflow

### Step 1: Provide Context

Give the AI assistant:
- **OpenAPI Specification** - The API definition (JSON or YAML)
- **User Story** - What you want to test in natural language
- **Base URL** - The target API endpoint

### Step 2: AI Generates Configuration

The AI uses the schema from `get_config_schema` to generate a valid configuration that:
- Matches your OpenAPI spec
- Implements your user story
- Follows Bombardino best practices

### Step 3: Validation Loop

The AI uses `validate_config` to verify the configuration:
- If valid: Proceed to execution or save
- If invalid: AI fixes the errors and revalidates

### Step 4: Execute or Save

- Use `run_test` to execute immediately and see results
- Use `save_config` to persist for later use

---

## Example Prompts

### Basic API Testing

```
I have an API at https://api.example.com with these endpoints:
- GET /users - list all users
- POST /users - create a user
- GET /users/{id} - get specific user
- DELETE /users/{id} - delete a user

Generate a Bombardino test that:
1. Creates a new user
2. Retrieves the created user
3. Deletes the user
4. Verifies the user is deleted

Use 10 iterations and include assertions to verify responses.
```

### From OpenAPI Spec

```
Here's my OpenAPI spec:
[paste your OpenAPI JSON/YAML]

Generate a Bombardino load test for the user registration flow:
1. First, call the health endpoint to verify the API is up
2. Register a new user with random data
3. Login with the created credentials
4. Access a protected endpoint using the auth token
5. Logout

Run for 2 minutes with 50 concurrent workers.
```

### User Story Based

```
User Story: As a customer, I want to add items to my cart and checkout

Given the e-commerce API at https://shop.example.com:
- POST /cart/items - add item to cart
- GET /cart - view cart
- POST /checkout - process payment
- GET /orders/{id} - view order status

Generate a test that simulates this complete flow with proper assertions
and extracted variables for IDs and tokens.
```

---

## Effective Prompting Tips

### Be Specific About Test Goals

**Less effective:**
```
Test my API
```

**More effective:**
```
Test the authentication flow with these specific validations:
- Login should return 200 and include an access_token
- Token should be valid for protected endpoints
- Invalid credentials should return 401
- Rate limiting should kick in after 5 failed attempts
```

### Provide Example Data

```
Test user creation with this data:
- Valid user: {"name": "John", "email": "john@test.com"}
- Invalid user (missing email): {"name": "Jane"}

Verify that:
- Valid user returns 201 with an ID
- Invalid user returns 400 with validation errors
```

### Specify Performance Requirements

```
Generate a load test that:
- Runs for 5 minutes
- Uses 100 concurrent workers
- Includes a 50ms delay between requests
- Expects P95 response time under 500ms
```

### Request Dependencies and Extraction

```
Create a CRUD test flow where:
1. POST creates a resource and extracts the ID
2. GET uses the extracted ID to fetch the resource
3. PUT updates using the same ID
4. DELETE removes the resource
5. GET verifies 404 after deletion

Use depends_on to ensure correct ordering.
```

---

## Common Patterns

### Authentication Flow Pattern

```json
{
  "name": "Auth Flow Test",
  "global": {
    "base_url": "https://api.example.com",
    "iterations": 1
  },
  "tests": [
    {
      "name": "Login",
      "method": "POST",
      "path": "/auth/login",
      "expected_status": [200],
      "body": {
        "email": "${email}",
        "password": "${password}"
      },
      "extract": [
        {"name": "access_token", "source": "body", "path": "token"}
      ]
    },
    {
      "name": "Access Protected",
      "method": "GET",
      "path": "/api/profile",
      "expected_status": [200],
      "headers": {
        "Authorization": "Bearer ${access_token}"
      },
      "depends_on": ["Login"]
    }
  ]
}
```

### CRUD Operations Pattern

```json
{
  "name": "Resource CRUD",
  "global": {
    "base_url": "https://api.example.com",
    "headers": {"Content-Type": "application/json"},
    "iterations": 1
  },
  "tests": [
    {
      "name": "Create",
      "method": "POST",
      "path": "/api/items",
      "expected_status": [201],
      "body": {"name": "Test Item"},
      "extract": [
        {"name": "item_id", "source": "body", "path": "id"}
      ]
    },
    {
      "name": "Read",
      "method": "GET",
      "path": "/api/items/${item_id}",
      "expected_status": [200],
      "depends_on": ["Create"],
      "assertions": [
        {"type": "json_path", "target": "name", "operator": "eq", "value": "Test Item"}
      ]
    },
    {
      "name": "Update",
      "method": "PUT",
      "path": "/api/items/${item_id}",
      "expected_status": [200],
      "depends_on": ["Read"],
      "body": {"name": "Updated Item"}
    },
    {
      "name": "Delete",
      "method": "DELETE",
      "path": "/api/items/${item_id}",
      "expected_status": [204],
      "depends_on": ["Update"]
    },
    {
      "name": "Verify Deleted",
      "method": "GET",
      "path": "/api/items/${item_id}",
      "expected_status": [404],
      "depends_on": ["Delete"]
    }
  ]
}
```

### Load Testing Pattern

```json
{
  "name": "Load Test",
  "global": {
    "base_url": "https://api.example.com",
    "duration": "5m",
    "delay": "100ms",
    "think_time_min": "500ms",
    "think_time_max": "2s"
  },
  "tests": [
    {
      "name": "Homepage",
      "method": "GET",
      "path": "/",
      "expected_status": [200],
      "assertions": [
        {"type": "response_time", "target": "response", "operator": "lt", "value": "500ms"}
      ]
    },
    {
      "name": "Search",
      "method": "GET",
      "path": "/search?q=test",
      "expected_status": [200],
      "assertions": [
        {"type": "response_time", "target": "response", "operator": "lt", "value": "1s"}
      ]
    }
  ]
}
```

### Error Handling Pattern

```json
{
  "name": "Error Handling Tests",
  "global": {
    "base_url": "https://api.example.com",
    "iterations": 1
  },
  "tests": [
    {
      "name": "Invalid Input",
      "method": "POST",
      "path": "/api/users",
      "expected_status": [400],
      "body": {"invalid": "data"},
      "assertions": [
        {"type": "json_path", "target": "error", "operator": "exists", "value": ""}
      ]
    },
    {
      "name": "Not Found",
      "method": "GET",
      "path": "/api/users/nonexistent-id",
      "expected_status": [404]
    },
    {
      "name": "Unauthorized",
      "method": "GET",
      "path": "/api/protected",
      "expected_status": [401]
    }
  ]
}
```

---

## OpenAPI to Bombardino Mapping

When generating from OpenAPI specs, the AI maps concepts as follows:

| OpenAPI | Bombardino |
|---------|------------|
| `servers[0].url` | `global.base_url` |
| `paths.{path}.{method}` | `tests[].path` + `tests[].method` |
| `requestBody.content.application/json.schema` | `tests[].body` structure |
| `responses.200` | `tests[].expected_status: [200]` |
| `parameters` (path) | `${variable}` in path |
| `parameters` (query) | Appended to path as query string |
| `parameters` (header) | `tests[].headers` |
| `securitySchemes.bearerAuth` | `headers.Authorization: "Bearer ${token}"` |

### Example OpenAPI to Bombardino

**OpenAPI:**
```yaml
paths:
  /users:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                email:
                  type: string
      responses:
        '201':
          description: Created
```

**Generated Bombardino:**
```json
{
  "tests": [
    {
      "name": "Create User",
      "method": "POST",
      "path": "/users",
      "expected_status": [201],
      "body": {
        "name": "Test User",
        "email": "test@example.com"
      }
    }
  ]
}
```

---

## Validation Feedback Loop

The AI uses the validation tool to ensure correctness. When validation fails, common fixes include:

### Missing Required Fields
```
Error: "test case missing required field: expected_status"
Fix: Add expected_status array to each test
```

### Invalid Duration Format
```
Error: "invalid duration format"
Fix: Use Go duration format (e.g., "30s", "5m", "100ms")
```

### Invalid JSON in Body
```
Error: "invalid JSON"
Fix: Ensure body is valid JSON (proper quotes, no trailing commas)
```

### Empty Tests Array
```
Error: "at least one test required"
Fix: Add at least one test case to the tests array
```

---

## Best Practices for AI-Generated Tests

1. **Start Simple** - Generate basic tests first, then add complexity
2. **Use Dependencies** - Always use `depends_on` for sequential flows
3. **Extract Important Values** - Use `extract` for IDs, tokens, and dynamic data
4. **Add Assertions** - Don't just check status codes; validate response content
5. **Set Realistic Timeouts** - Configure appropriate timeouts for your API
6. **Use Think Time** - Add realistic delays for load testing
7. **Validate Before Running** - Always validate configurations before execution
8. **Review Generated Configs** - AI-generated configs should be reviewed before production use

---

## Troubleshooting

### "bombardino not found"
Set the `BOMBARDINO_PATH` environment variable to the path of your Bombardino binary.

### Configuration Validation Fails
Ask the AI to show the validation error and fix the specific issue.

### Tests Fail Unexpectedly
- Check if the API is running
- Verify the base_url is correct
- Ensure authentication tokens are valid
- Check if expected status codes match actual responses

### Performance Issues
- Reduce worker count for limited APIs
- Add delay between requests
- Use duration mode instead of high iteration counts

---

## Related Documentation

- [Configuration Reference](./configuration-reference.md) - Complete field documentation
- [MCP Server README](../mcp/README.md) - Installation and setup
- [Main README](../README.md) - General Bombardino usage
