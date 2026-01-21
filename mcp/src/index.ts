#!/usr/bin/env node

import { Server } from "@modelcontextprotocol/sdk/server/index.js";
import { StdioServerTransport } from "@modelcontextprotocol/sdk/server/stdio.js";
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from "@modelcontextprotocol/sdk/types.js";

import { configSchema, examples } from "./tools/getConfigSchema.js";
import { validateConfig } from "./tools/validateConfig.js";
import { runTest } from "./tools/runTest.js";
import { saveConfig } from "./tools/saveConfig.js";
import { getVersion } from "./utils/bombardino.js";

const server = new Server(
  {
    name: "bombardino-mcp",
    version: "1.0.0",
  },
  {
    capabilities: {
      tools: {},
    },
  }
);

// List available tools
server.setRequestHandler(ListToolsRequestSchema, async () => {
  return {
    tools: [
      {
        name: "get_config_schema",
        description:
          "Get the JSON schema and examples for Bombardino configuration files. Use this to understand the format before generating a config.",
        inputSchema: {
          type: "object",
          properties: {},
          required: [],
        },
      },
      {
        name: "validate_config",
        description:
          "Validate a Bombardino configuration JSON without running tests. Returns validation status and any errors.",
        inputSchema: {
          type: "object",
          properties: {
            config: {
              type: "string",
              description: "The Bombardino configuration as a JSON string",
            },
          },
          required: ["config"],
        },
      },
      {
        name: "run_test",
        description:
          "Run a Bombardino test configuration and return the results. The config must be valid JSON.",
        inputSchema: {
          type: "object",
          properties: {
            config: {
              type: "string",
              description: "The Bombardino configuration as a JSON string",
            },
            workers: {
              type: "number",
              description: "Number of concurrent workers (default: 10)",
            },
            verbose: {
              type: "boolean",
              description: "Enable verbose logging (default: false)",
            },
          },
          required: ["config"],
        },
      },
      {
        name: "save_config",
        description:
          "Save a Bombardino configuration to a file. The config will be pretty-printed.",
        inputSchema: {
          type: "object",
          properties: {
            config: {
              type: "string",
              description: "The Bombardino configuration as a JSON string",
            },
            path: {
              type: "string",
              description: "File path to save the configuration",
            },
          },
          required: ["config", "path"],
        },
      },
    ],
  };
});

// Handle tool calls
server.setRequestHandler(CallToolRequestSchema, async (request) => {
  const { name, arguments: args } = request.params;

  switch (name) {
    case "get_config_schema": {
      const version = await getVersion();
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(
              {
                bombardino_version: version,
                schema: configSchema,
                examples,
                tips: [
                  "Always use depends_on when a test needs data from a previous test",
                  "Use extract to capture values (like IDs) for use in subsequent tests",
                  "Reference extracted values with ${variable_name} syntax",
                  "For CRUD flows: Create → Read → Update → Delete → Verify Deleted",
                  "Use assertions to validate response content, not just status codes",
                  "Set appropriate expected_status for each operation (201 for create, 204 for delete, etc.)",
                  "Use compare_with to validate API migrations or staging deployments",
                  "Use ignore_fields in compare_with for dynamic values like timestamps",
                  "Use header_match in compare_with to verify headers during strangler fig migrations",
                ],
              },
              null,
              2
            ),
          },
        ],
      };
    }

    case "validate_config": {
      const result = await validateConfig({
        config: args?.config as string,
      });
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(result, null, 2),
          },
        ],
      };
    }

    case "run_test": {
      const result = await runTest({
        config: args?.config as string,
        workers: args?.workers as number | undefined,
        verbose: args?.verbose as boolean | undefined,
      });
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(result, null, 2),
          },
        ],
      };
    }

    case "save_config": {
      const result = await saveConfig({
        config: args?.config as string,
        path: args?.path as string,
      });
      return {
        content: [
          {
            type: "text",
            text: JSON.stringify(result, null, 2),
          },
        ],
      };
    }

    default:
      throw new Error(`Unknown tool: ${name}`);
  }
});

// Start the server
async function main() {
  const transport = new StdioServerTransport();
  await server.connect(transport);
  console.error("Bombardino MCP server running on stdio");
}

main().catch((error) => {
  console.error("Fatal error:", error);
  process.exit(1);
});
