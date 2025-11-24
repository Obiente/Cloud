/**
 * Utility functions for handling ANSI escape codes and terminal control sequences
 */

/**
 * Strips ANSI escape codes and terminal control sequences from text
 * Handles:
 * - Standard ANSI escape sequences (ESC[...m, ESC[K, etc.)
 * - Terminal mode sequences ([?1h, [?2004h, etc.)
 * - Application keypad mode sequences ([=, [>, [<)
 * - Cursor control sequences ([K, [H, etc.)
 * - Color codes ([93m, [0m, etc.)
 * - Prompt continuation indicators (>....)
 * 
 * @param text - The text to clean
 * @returns The text with ANSI codes and control sequences removed
 */
export function stripAnsiCodes(text: string): string {
  if (!text) return text;

  let cleaned = text;

  // Pattern 1: Standard ANSI escape sequences with ESC prefix (\x1b)
  // Matches: ESC[ followed by parameters and command letter
  cleaned = cleaned.replace(/\x1b\[[0-9;?]*[a-zA-Z]/g, "");
  cleaned = cleaned.replace(/\x1b[=<>]/g, "");

  // Pattern 2: Iteratively remove escape sequences until no more changes occur
  // This handles cases where sequences are concatenated without spaces
  let previous: string;
  do {
    previous = cleaned;

    // Remove terminal mode sequences: [?1h, [?2004h, etc.
    cleaned = cleaned.replace(/\[\?[0-9]+[hHlLmM]/g, "");

    // Remove application keypad mode sequences: [=, [>, [<
    cleaned = cleaned.replace(/\[[=<>]/g, "");

    // Remove single-character CSI sequences: [K (clear to end of line), [H (cursor home), etc.
    // Common CSI single-letter commands: A-H, J, K, m, s, u
    cleaned = cleaned.replace(/\[K/g, "");
    cleaned = cleaned.replace(/\[[A-HJmsu]/g, "");

    // Remove formatting codes: [0m, [1m, [4m, [3m, [30m, etc.
    // Match [ followed by digits and 'm' (SGR - Select Graphic Rendition)
    // But exclude timestamps like [21:45:50] by requiring the 'm' suffix
    cleaned = cleaned.replace(/\[[0-9;]+m/g, "");

    // Remove prompt continuation indicators like ">...." anywhere in the line
    cleaned = cleaned.replace(/>\.+/g, "");

    // Remove any remaining malformed escape patterns
    cleaned = cleaned.replace(/\[=[?0-9]*/g, "");
  } while (cleaned !== previous);

  // Final pass: remove any remaining control sequences that might have been missed
  // Remove patterns at start/end of line
  cleaned = cleaned.replace(/^\[\?[0-9]*[hHlLmM]?/g, ""); // At start of line
  cleaned = cleaned.replace(/\[K$/g, ""); // At end of line
  cleaned = cleaned.replace(/\[K\s/g, " "); // Before whitespace

  // Remove any remaining control characters except newlines, carriage returns, and tabs
  cleaned = cleaned.replace(/[\x00-\x08\x0B-\x0C\x0E-\x1F\x7F]/g, "");

  // Don't trim here - preserve whitespace structure for proper line handling
  // The calling code can trim individual lines if needed
  return cleaned;
}

/**
 * Strips ANSI codes and also removes common timestamp patterns from log lines
 * Processes each line individually to handle multiline text
 * @param text - The text to clean
 * @returns The cleaned text
 */
export function stripAnsiAndTimestamps(text: string): string {
  let cleaned = stripAnsiCodes(text);

  // Split into lines, process each line, then rejoin
  const lines = cleaned.split(/\r?\n/);
  const cleanedLines = lines.map((line) => {
    let processed = line;
    
    // Remove Minecraft-style log prefixes: [HH:MM:SS LEVEL]: 
    // Examples: [22:24:10 INFO]:, [12:34:56 WARN]:, [01:23:45 ERROR]:
    // Try multiple patterns to catch all variations
    // Pattern 1: Standard format with space: [HH:MM:SS LEVEL]:
    processed = processed.replace(/\[\d{2}:\d{2}:\d{2}\s+(INFO|WARN|ERROR|DEBUG|TRACE|FATAL)\]:\s*/gi, "");
    
    // Pattern 2: Without space (in case space was stripped): [HH:MM:SSLEVEL]:
    processed = processed.replace(/\[\d{2}:\d{2}:\d{2}(INFO|WARN|ERROR|DEBUG|TRACE|FATAL)\]:\s*/gi, "");
    
    // Pattern 3: More flexible - any whitespace characters
    processed = processed.replace(/\[\d{2}:\d{2}:\d{2}[\s\u00A0\u1680\u2000-\u200B\u202F\u205F\u3000\uFEFF]*(INFO|WARN|ERROR|DEBUG|TRACE|FATAL)\]:\s*/gi, "");

    // Remove Minecraft-style timestamps without level: [HH:MM:SS]
    processed = processed.replace(/\[\d{2}:\d{2}:\d{2}\]\s*/g, "");

    // Remove ISO timestamps: 2025-11-05T01:57:06.052Z
    processed = processed.replace(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z\s+/g, "");

    // Remove plugin name prefixes at start of line: pluginname] or [pluginname]
    // Examples: park], [park], spark], etc.
    // Match word characters followed by ] at the start of the line
    processed = processed.replace(/^[a-zA-Z0-9_-]+\]\s*/g, "");
    processed = processed.replace(/^\[[a-zA-Z0-9_-]+\]\s*/g, "");

    return processed;
  });

  return cleanedLines.join("\n");
}

