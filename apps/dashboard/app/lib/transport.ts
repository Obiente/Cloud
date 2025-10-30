import { createGrpcWebTransport } from "@connectrpc/connect-web";
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
          console.debug("No valid token available for request");
        }
      } catch (error) {
        console.error("Error getting auth token:", error);
        // Continue without token
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
  return createGrpcWebTransport({
    baseUrl,
    interceptors: [interceptor],
  });
}
