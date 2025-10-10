import type { H3Event } from "h3";
import type { ZitadelErrorResponse, ZitadelTokenClaims } from "@obiente/types";
import { jwtVerify, decodeJwt, createRemoteJWKSet } from "jose";

/**
 * Get the access token from a cookie
 */
export function getAccessToken(event: H3Event): string | undefined {
  return getCookie(event, AUTH_COOKIE_NAME);
}

/**
 * Verify and decode a Zitadel access token
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
 * Decode token without verifying (useful for getting expiration)
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
