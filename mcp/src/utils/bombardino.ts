import { exec } from "child_process";
import { promisify } from "util";
import { writeFile, unlink } from "fs/promises";
import { tmpdir } from "os";
import { join } from "path";
import { randomUUID } from "crypto";

const execAsync = promisify(exec);

// Get bombardino path from environment or use default
const BOMBARDINO_PATH = process.env.BOMBARDINO_PATH || "bombardino";

export interface ValidationResult {
  valid: boolean;
  error?: string;
  testCount?: number;
  configName?: string;
}

export interface TestResult {
  success: boolean;
  summary: {
    total_requests: number;
    successful_requests: number;
    failed_requests: number;
    success_rate_percent: number;
    total_time: string;
    avg_response_time: string;
    requests_per_sec: number;
  };
  endpoints: Record<string, unknown>;
  debug_logs?: unknown[];
}

/**
 * Creates a temporary file with the given content
 */
async function createTempConfig(config: string): Promise<string> {
  const tempPath = join(tmpdir(), `bombardino-${randomUUID()}.json`);
  await writeFile(tempPath, config, "utf-8");
  return tempPath;
}

/**
 * Cleans up a temporary file
 */
async function cleanupTempConfig(path: string): Promise<void> {
  try {
    await unlink(path);
  } catch {
    // Ignore cleanup errors
  }
}

/**
 * Validates a Bombardino configuration
 */
export async function validateConfig(config: string): Promise<ValidationResult> {
  const tempPath = await createTempConfig(config);

  try {
    const { stdout } = await execAsync(`${BOMBARDINO_PATH} -t -config "${tempPath}"`);

    // Parse success output: "✅ Configuration valid: Name (X tests)"
    const match = stdout.match(/Configuration valid: (.+) \((\d+) tests\)/);
    if (match) {
      return {
        valid: true,
        configName: match[1],
        testCount: parseInt(match[2], 10),
      };
    }

    return { valid: true };
  } catch (error: unknown) {
    const execError = error as { stderr?: string; stdout?: string; message?: string };
    // Parse error output: "❌ Configuration invalid: error message"
    const errorOutput = execError.stderr || execError.stdout || execError.message || "Unknown error";
    const errorMatch = errorOutput.match(/Configuration invalid: (.+)/);

    return {
      valid: false,
      error: errorMatch ? errorMatch[1].trim() : errorOutput.trim(),
    };
  } finally {
    await cleanupTempConfig(tempPath);
  }
}

/**
 * Runs a Bombardino test and returns JSON results
 */
export async function runTest(
  config: string,
  options: { workers?: number; verbose?: boolean } = {}
): Promise<TestResult> {
  const tempPath = await createTempConfig(config);
  const { workers = 10, verbose = false } = options;

  try {
    const verboseFlag = verbose ? "-verbose" : "";
    const cmd = `${BOMBARDINO_PATH} -config "${tempPath}" -workers ${workers} -output json ${verboseFlag}`;

    const { stdout } = await execAsync(cmd, { maxBuffer: 10 * 1024 * 1024 }); // 10MB buffer

    const result = JSON.parse(stdout) as TestResult;
    return result;
  } catch (error: unknown) {
    const execError = error as { stdout?: string; stderr?: string; message?: string };

    // Try to parse JSON from stdout even on non-zero exit (tests failed)
    if (execError.stdout) {
      try {
        return JSON.parse(execError.stdout) as TestResult;
      } catch {
        // Not valid JSON
      }
    }

    throw new Error(`Failed to run test: ${execError.stderr || execError.message}`);
  } finally {
    await cleanupTempConfig(tempPath);
  }
}

/**
 * Gets the Bombardino version
 */
export async function getVersion(): Promise<string> {
  try {
    const { stdout } = await execAsync(`${BOMBARDINO_PATH} -version`);
    return stdout.trim();
  } catch {
    return "unknown";
  }
}
