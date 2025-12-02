import { validateConfig as validateBombardino } from "../utils/bombardino.js";

export interface ValidateConfigInput {
  config: string;
}

export interface ValidateConfigOutput {
  valid: boolean;
  error?: string;
  testCount?: number;
  configName?: string;
}

/**
 * Validates a Bombardino configuration JSON
 */
export async function validateConfig(input: ValidateConfigInput): Promise<ValidateConfigOutput> {
  const { config } = input;

  // First, try to parse as JSON to give better error messages
  try {
    JSON.parse(config);
  } catch (e) {
    return {
      valid: false,
      error: `Invalid JSON: ${(e as Error).message}`,
    };
  }

  // Then validate with Bombardino CLI
  return validateBombardino(config);
}
