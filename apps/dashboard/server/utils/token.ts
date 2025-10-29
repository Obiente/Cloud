import type { H3Event } from "h3";
import type { ZitadelErrorResponse, ZitadelTokenClaims } from "@obiente/types";
import { jwtVerify, decodeJwt, createRemoteJWKSet } from "jose";

// Import directly from the relative path to avoid module resolution issues
import { AUTH_COOKIE_NAME } from "./auth";

/**
 * Get the access token from a cookie
 */
export function getAccessToken(event: H3Event): string | undefined {
  return getCookie(event, AUTH_COOKIE_NAME);
}

/**
 * Verify and decode a Zitadel access token (JWT)
 * NOTE: If access tokens are bearer tokens (not JWTs), this function should NOT be used.
 * Bearer tokens should be validated by the API backend, not in the middleware.
 */
export async function verifyAccessToken(token: string) {
  const config = useRuntimeConfig();
  const { oidcBase } = config.public;

  // Get JWKS from Zitadel
  const jwks = createRemoteJWKSet(new URL(`${oidcBase}/oauth/v2/keys`));

  // Verify token
  return jwtVerify(token, jwks, {
    issuer: config.public.oidcIssuer,
    audience: config.public.oidcClientId,
  });
}

/**
 * Decode JWT token without verifying (useful for getting expiration)
 * NOTE: This only works for JWT tokens. Bearer tokens cannot be decoded.
 */
export function decodeToken<T extends ZitadelTokenClaims>(token: string): T {
  return decodeJwt<T>(token);
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
