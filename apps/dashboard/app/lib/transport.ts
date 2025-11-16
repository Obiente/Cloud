import { createConnectTransport } from "@connectrpc/connect-web";
import type { Transport } from "@connectrpc/connect";
import type { Interceptor } from "@connectrpc/connect";

/**
 * Create auth interceptor for adding tokens to requests
 */
export function createAuthInterceptor(
  getToken?: () => string | undefined | Promise<string | undefined>
): Interceptor {
  return (next) => async (req) => {
    // Only add auth header if getToken is provided and returns a token
    if (getToken) {
      try {
        const token = await Promise.resolve(getToken());
        if (token && typeof token === "string" && token.trim() !== "") {
          req.header.append("Authorization", `Bearer ${token}`);
        } else {
          // Only warn if token is missing and it's not a dev dummy token scenario
          // (dev dummy token is "dev-dummy-token", so empty means real missing token)
          const isDevDummy = process.env.DISABLE_AUTH === "true";
          if (!isDevDummy) {
            console.warn("[Auth Interceptor] No valid token available for request to:", req.url);
          }
          // Don't throw here - let the API handle authentication errors
          // This allows for better error messages from the API
        }
      } catch (error) {
        console.error("[Auth Interceptor] Error getting auth token:", error);
        // Continue without token - let the API return proper auth error
      }
    }
    return next(req);
  };
}

// Export client-side transport factory
export function createWebTransport(
  baseUrl: string,
  interceptor: Interceptor
): Transport {
  return createConnectTransport({
    baseUrl,
    interceptors: [interceptor],
    // Use JSON for better browser compatibility
    useBinaryFormat: false,
    defaultTimeoutMs: 5000, // 5 seconds timeout for client-side API calls
  });
}
