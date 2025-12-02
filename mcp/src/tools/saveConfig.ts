import { writeFile } from "fs/promises";
import { dirname } from "path";
import { mkdir } from "fs/promises";

export interface SaveConfigInput {
  config: string;
  path: string;
}

export interface SaveConfigOutput {
  saved: boolean;
  path: string;
  error?: string;
}

/**
 * Saves a Bombardino configuration to a file
 */
export async function saveConfig(input: SaveConfigInput): Promise<SaveConfigOutput> {
  const { config, path } = input;

  // Validate JSON first
  try {
    const parsed = JSON.parse(config);
    // Pretty print the JSON
    const prettyConfig = JSON.stringify(parsed, null, 2);

    // Ensure directory exists
    const dir = dirname(path);
    await mkdir(dir, { recursive: true });

    // Write the file
    await writeFile(path, prettyConfig, "utf-8");

    return {
      saved: true,
      path,
    };
  } catch (e) {
    return {
      saved: false,
      path,
      error: (e as Error).message,
    };
  }
}
