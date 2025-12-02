import { runTest as runBombardino } from "../utils/bombardino.js";

export interface RunTestInput {
  config: string;
  workers?: number;
  verbose?: boolean;
}

export interface RunTestOutput {
  success: boolean;
  summary: {
    total_requests: number;
    successful_requests: number;
    failed_requests: number;
    success_rate_percent: number;
    total_time: string;
    avg_response_time: string;
    min_response_time: string;
    max_response_time: string;
    p50_response_time: string;
    p95_response_time: string;
    p99_response_time: string;
    requests_per_sec: number;
  };
  endpoints: Record<string, unknown>;
  error?: string;
}

/**
 * Runs a Bombardino test and returns results
 */
export async function runTest(input: RunTestInput): Promise<RunTestOutput> {
  const { config, workers, verbose } = input;

  // First validate the config
  try {
    JSON.parse(config);
  } catch (e) {
    return {
      success: false,
      summary: {
        total_requests: 0,
        successful_requests: 0,
        failed_requests: 0,
        success_rate_percent: 0,
        total_time: "0s",
        avg_response_time: "0s",
        min_response_time: "0s",
        max_response_time: "0s",
        p50_response_time: "0s",
        p95_response_time: "0s",
        p99_response_time: "0s",
        requests_per_sec: 0,
      },
      endpoints: {},
      error: `Invalid JSON: ${(e as Error).message}`,
    };
  }

  try {
    const result = await runBombardino(config, { workers, verbose });
    return {
      success: result.success,
      summary: result.summary as RunTestOutput["summary"],
      endpoints: result.endpoints,
    };
  } catch (e) {
    return {
      success: false,
      summary: {
        total_requests: 0,
        successful_requests: 0,
        failed_requests: 0,
        success_rate_percent: 0,
        total_time: "0s",
        avg_response_time: "0s",
        min_response_time: "0s",
        max_response_time: "0s",
        p50_response_time: "0s",
        p95_response_time: "0s",
        p99_response_time: "0s",
        requests_per_sec: 0,
      },
      endpoints: {},
      error: (e as Error).message,
    };
  }
}
