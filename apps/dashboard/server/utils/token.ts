import type { H3Event } from "h3";
import type { ZitadelErrorResponse, ZitadelTokenClaims } from "@obiente/types";

// Import directly from the relative path to avoid module resolution issues
import { AUTH_COOKIE_NAME } from "./auth";

/**
 * Get the access token from a cookie
 */
export function getAccessToken(event: H3Event): string | undefined {
  return getCookie(event, AUTH_COOKIE_NAME);
}

/**
 * Helper to handle Zitadel errors
 */
export function handleZitadelError(error: unknown): ZitadelErrorResponse {
  if (typeof error === "string") {
    return {
      error: "server_error",
      error_description: error,
    };
  }

  if (error instanceof Error) {
    return {
      error: "server_error",
      error_description: error.message,
    };
  }

  return {
    error: "unknown_error",
    error_description: "An unknown error occurred",
  };
}
