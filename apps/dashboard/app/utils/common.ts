/**
 * Common utility functions used across the application
 */

/**
 * Generates initials from a name string
 * @param name - The name to generate initials from
 * @returns Initials string (e.g., "John Doe" -> "JD", "Alice" -> "AL")
 */
export function getInitials(name: string): string {
  if (!name) return "??";
  
  // Remove extra whitespace and split by spaces
  const parts = name.trim().split(/\s+/).filter((p) => p.length > 0);
  
  if (parts.length === 0) return "??";
  
  if (parts.length === 1) {
    // Single word - take first 2 characters
    const word = parts[0];
    if (!word) return "??";
    if (word.length >= 2) {
      return word.substring(0, 2).toUpperCase();
    }
    return word.substring(0, 1).toUpperCase();
  }
  
  // Multiple words - take first letter of first and last word
  const firstPart = parts[0];
  const lastPart = parts[parts.length - 1];
  if (!firstPart || !lastPart) return "??";
  const first = firstPart[0];
  const last = lastPart[0];
  if (!first || !last) return "??";
  return first.toUpperCase() + last.toUpperCase();
}

/**
 * Formats a timestamp (from proto or Date) to a readable date string
 * @param timestamp - Timestamp object with seconds/nanos properties or Date object
 * @returns Formatted date string
 */
export function formatDate(timestamp: any): string {
  if (!timestamp) return "—";
  
  let date: Date;
  if (timestamp.seconds !== undefined) {
    // Proto timestamp format
    const seconds = typeof timestamp.seconds === "bigint" ? Number(timestamp.seconds) : timestamp.seconds;
    const millis = seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
    date = new Date(millis);
  } else if (timestamp instanceof Date) {
    date = timestamp;
  } else {
    date = new Date(timestamp);
  }
  
  if (isNaN(date.getTime())) return "—";
  
  return date.toLocaleDateString() + " " + date.toLocaleTimeString();
}

/**
 * Formats a timestamp to a date-only string
 * @param timestamp - Timestamp object with seconds property or Date object
 * @returns Formatted date string (date only)
 */
export function formatDateOnly(timestamp: any): string {
  if (!timestamp) return "—";
  
  const date = timestamp.seconds
    ? new Date(Number(timestamp.seconds) * 1000)
    : timestamp instanceof Date
    ? timestamp
    : new Date(timestamp);
  
  if (isNaN(date.getTime())) return "—";
  
  return date.toLocaleDateString();
}

/**
 * Formats bytes to a human-readable string
 * @param bytes - Number of bytes (can be number, bigint, null, or undefined)
 * @returns Formatted string (e.g., "1.5 MB")
 */
export function formatBytes(bytes: number | bigint | null | undefined): string {
  if (bytes === null || bytes === undefined) return "0 B";
  const numBytes = typeof bytes === "bigint" ? Number(bytes) : bytes;
  if (numBytes === 0 || !Number.isFinite(numBytes) || numBytes < 0) return "0 B";
  
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(numBytes) / Math.log(k));
  
  return Math.round((numBytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
}

/**
 * Formats a number as currency
 * @param cents - Amount in cents
 * @param currency - Currency code (default: "USD")
 * @returns Formatted currency string
 */
export function formatCurrency(cents: number | bigint, currency: string = "USD"): string {
  const amount = typeof cents === "bigint" ? Number(cents) : cents;
  const dollars = amount / 100;
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency,
  }).format(dollars);
}

