import { createGrpcWebTransport } from "@connectrpc/connect-web";
import type { Transport } from "@connectrpc/connect";
import type { Interceptor } from "@connectrpc/connect";

/**
 * Pure factory for creating a connect transport instance.
 * Keep this file under `app/` so it can be imported in both
 * client plugins and server code without pulling in Nuxt server-only helpers.
 * 
 * @param baseUrl - The base URL for the API
 * @param getToken - Optional function to retrieve the auth token
 */
export function createTransport(
  baseUrl: string, 
  getToken?: () => string | undefined | Promise<string | undefined>
): Transport {
  // Create interceptor to add auth token if available
  const authInterceptor: Interceptor = (next) => async (req) => {
    // Only add auth header if getToken is provided and returns a token
    if (getToken) {
      try {
        const token = await Promise.resolve(getToken());
        if (token && typeof token === 'string' && token.trim() !== '') {
          req.header.append("Authorization", `Bearer ${token}`);
        } else {
          console.debug('No valid token available for request');
        }
      } catch (error) {
        console.error('Error getting auth token:', error);
        // Continue without token
      }
    }
    return next(req);
  };
  
  // Create transport with auth interceptor
  return createGrpcWebTransport({
    baseUrl,
    interceptors: [authInterceptor],
  });
}

export default createTransport;
