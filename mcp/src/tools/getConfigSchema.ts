/**
 * Returns the complete Bombardino configuration schema
 */
export const configSchema = {
  $schema: "http://json-schema.org/draft-07/schema#",
  title: "Bombardino Configuration",
  description: "Configuration file for Bombardino API load testing",
  type: "object",
  required: ["name", "global", "tests"],
  properties: {
    name: {
      type: "string",
      description: "Name of the test suite",
    },
    description: {
      type: "string",
      description: "Optional description of the test suite",
    },
    global: {
      type: "object",
      description: "Global settings applied to all tests",
      required: ["base_url"],
      properties: {
        base_url: {
          type: "string",
          description: "Base URL for all requests (e.g., http://localhost:8080)",
        },
        timeout: {
          type: "string",
          description: "Request timeout in Go duration format (e.g., '30s', '1m')",
          default: "30s",
        },
        delay: {
          type: "string",
          description: "Delay between requests (e.g., '100ms')",
          default: "0",
        },
        iterations: {
          type: "integer",
          description: "Number of times to run each test",
          default: 1,
        },
        duration: {
          type: "string",
          description: "Run tests for this duration (e.g., '5m'). Alternative to iterations.",
        },
        headers: {
          type: "object",
          description: "HTTP headers added to all requests",
          additionalProperties: { type: "string" },
        },
        insecure_skip_verify: {
          type: "boolean",
          description: "Skip TLS certificate verification",
          default: false,
        },
        variables: {
          type: "object",
          description: "Global variables accessible with ${name} syntax",
          additionalProperties: true,
        },
        think_time: {
          type: "string",
          description: "Fixed pause after each test iteration (e.g., '500ms')",
        },
        think_time_min: {
          type: "string",
          description: "Minimum random think time",
        },
        think_time_max: {
          type: "string",
          description: "Maximum random think time",
        },
      },
    },
    tests: {
      type: "array",
      description: "Array of test definitions",
      minItems: 1,
      items: {
        type: "object",
        required: ["name", "method", "path", "expected_status"],
        properties: {
          name: {
            type: "string",
            description: "Unique test name (used for depends_on references)",
          },
          method: {
            type: "string",
            enum: ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"],
            description: "HTTP method",
          },
          path: {
            type: "string",
            description: "URL path (appended to base_url). Supports ${variable} syntax.",
          },
          expected_status: {
            type: "array",
            items: { type: "integer" },
            description: "Expected HTTP status codes (e.g., [200, 201])",
          },
          headers: {
            type: "object",
            description: "Additional headers for this test",
            additionalProperties: { type: "string" },
          },
          body: {
            description: "Request body (object, array, or string)",
          },
          timeout: {
            type: "string",
            description: "Override global timeout for this test",
          },
          delay: {
            type: "string",
            description: "Override global delay for this test",
          },
          iterations: {
            type: "integer",
            description: "Override global iterations for this test",
          },
          duration: {
            type: "string",
            description: "Override global duration for this test",
          },
          depends_on: {
            type: "array",
            items: { type: "string" },
            description: "Test names that must complete before this one",
          },
          assertions: {
            type: "array",
            description: "Response validations",
            items: {
              type: "object",
              required: ["type", "target", "operator", "value"],
              properties: {
                type: {
                  type: "string",
                  enum: ["status", "json_path", "response_time", "header", "body_size"],
                  description: "Assertion type",
                },
                target: {
                  type: "string",
                  description: "What to check (e.g., 'response', JSON path, header name)",
                },
                operator: {
                  type: "string",
                  enum: ["eq", "neq", "gt", "gte", "lt", "lte", "contains", "starts_with", "ends_with", "exists", "not_exists", "matches"],
                  description: "Comparison operator",
                },
                value: {
                  description: "Expected value",
                },
              },
            },
          },
          extract: {
            type: "array",
            description: "Values to extract from response for use in later tests",
            items: {
              type: "object",
              required: ["name", "source", "path"],
              properties: {
                name: {
                  type: "string",
                  description: "Variable name (use as ${name})",
                },
                source: {
                  type: "string",
                  enum: ["body", "header", "status"],
                  description: "Where to extract from",
                },
                path: {
                  type: "string",
                  description: "JSON path for body, header name for header",
                },
              },
            },
          },
          data: {
            type: "array",
            description: "Inline data for data-driven testing",
            items: { type: "object" },
          },
          data_file: {
            type: "string",
            description: "Path to external data file (JSON or CSV)",
          },
          think_time: {
            type: "string",
            description: "Override global think time for this test",
          },
          think_time_min: {
            type: "string",
            description: "Override global think_time_min for this test",
          },
          think_time_max: {
            type: "string",
            description: "Override global think_time_max for this test",
          },
          insecure_skip_verify: {
            type: "boolean",
            description: "Override global insecure_skip_verify for this test",
          },
          compare_with: {
            type: "object",
            description: "Compare response with another endpoint (tap compare)",
            properties: {
              endpoint: {
                type: "string",
                description: "Base URL of comparison endpoint (required)",
              },
              path: {
                type: "string",
                description: "Path override (defaults to primary path)",
              },
              headers: {
                type: "object",
                description: "Additional headers for comparison request",
                additionalProperties: { type: "string" },
              },
              timeout: {
                type: "string",
                description: "Custom timeout for comparison request",
              },
              assertions: {
                type: "array",
                description: "Comparison assertions",
                items: {
                  type: "object",
                  properties: {
                    type: {
                      type: "string",
                      enum: [
                        "status_match",
                        "field_match",
                        "field_tolerance",
                        "structure_match",
                        "response_time_tolerance",
                        "header_match",
                      ],
                      description: "Comparison assertion type",
                    },
                    target: {
                      type: "string",
                      description: "JSON path for field assertions",
                    },
                    operator: {
                      type: "string",
                      enum: ["eq", "contains", "exists"],
                      description: "Comparison operator (eq/contains for field_match, eq/contains/exists for header_match)",
                    },
                    tolerance: {
                      description:
                        "Tolerance value: 0.1 = 10%, '15%' = 15%, 10 = absolute value",
                    },
                  },
                },
              },
              ignore_fields: {
                type: "array",
                items: { type: "string" },
                description:
                  "Fields to skip during comparison (e.g., timestamp, request_id)",
              },
              mode: {
                type: "string",
                enum: ["full", "partial", "structural"],
                description: "Comparison mode",
                default: "full",
              },
            },
            required: ["endpoint"],
          },
        },
      },
    },
  },
};

export const examples = [
  {
    name: "Simple GET Test",
    description: "Basic test with a single GET request",
    config: {
      name: "Simple GET Test",
      global: {
        base_url: "https://api.example.com",
        iterations: 10,
      },
      tests: [
        {
          name: "Get Users",
          method: "GET",
          path: "/users",
          expected_status: [200],
        },
      ],
    },
  },
  {
    name: "CRUD Flow with Dependencies",
    description: "Create, Read, Update, Delete with variable extraction",
    config: {
      name: "CRUD Flow",
      global: {
        base_url: "http://localhost:8080",
        timeout: "30s",
        iterations: 1,
        headers: { "Content-Type": "application/json" },
      },
      tests: [
        {
          name: "Create",
          method: "POST",
          path: "/api/users",
          expected_status: [201],
          body: { name: "Test User", email: "test@example.com" },
          extract: [{ name: "user_id", source: "body", path: "id" }],
        },
        {
          name: "Read",
          method: "GET",
          path: "/api/users/${user_id}",
          expected_status: [200],
          depends_on: ["Create"],
        },
        {
          name: "Update",
          method: "PUT",
          path: "/api/users/${user_id}",
          expected_status: [200],
          depends_on: ["Read"],
          body: { name: "Updated User" },
        },
        {
          name: "Delete",
          method: "DELETE",
          path: "/api/users/${user_id}",
          expected_status: [204],
          depends_on: ["Update"],
        },
        {
          name: "Verify Deleted",
          method: "GET",
          path: "/api/users/${user_id}",
          expected_status: [404],
          depends_on: ["Delete"],
        },
      ],
    },
  },
  {
    name: "Test with Assertions",
    description: "Validate response content with assertions",
    config: {
      name: "API Validation Test",
      global: {
        base_url: "https://api.example.com",
        iterations: 5,
      },
      tests: [
        {
          name: "Get User Details",
          method: "GET",
          path: "/users/1",
          expected_status: [200],
          assertions: [
            { type: "json_path", target: "id", operator: "exists", value: "" },
            { type: "json_path", target: "name", operator: "eq", value: "John Doe" },
            { type: "response_time", target: "response", operator: "lt", value: "500ms" },
            { type: "header", target: "Content-Type", operator: "contains", value: "json" },
          ],
        },
      ],
    },
  },
  {
    name: "Tap Compare - Production vs Staging",
    description: "Compare API responses between two endpoints to validate migrations or deployments",
    config: {
      name: "Production vs Staging Validation",
      global: {
        base_url: "https://jsonplaceholder.typicode.com",
        timeout: "30s",
        iterations: 3,
      },
      tests: [
        {
          name: "Compare Users API",
          method: "GET",
          path: "/users/1",
          expected_status: [200],
          compare_with: {
            endpoint: "https://jsonplaceholder.typicode.com",
            ignore_fields: ["website"],
            assertions: [
              { type: "status_match" },
              { type: "field_match", target: "id" },
              { type: "field_match", target: "email" },
              { type: "header_match", target: "Content-Type" },
              { type: "structure_match" },
            ],
          },
        },
      ],
    },
  },
];
