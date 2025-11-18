/**
 * Authentication middleware for Nuxt server routes
 * Access tokens are bearer tokens, not JWTs, so validation is handled by the API
 */
export default defineEventHandler(async (event) => {
  // Check if auth is disabled via public config endpoint
  // First check environment variable as fallback
  if (process.env.DISABLE_AUTH === "true") {
    console.log("⚠️  WARNING: DISABLE_AUTH=true (env), skipping authentication middleware");
    return;
  }

  // Try to fetch public config to check disableAuth
  try {
    const config = useRuntimeConfig();
    const { AuthService } = await import("@obiente/proto");
    const { createClient } = await import("@connectrpc/connect");
    const { createConnectTransport } = await import("@connectrpc/connect-node");
    
    // Use internal API host for server-side (Docker internal networking)
    let apiHost = (config.apiHostInternal as string) || config.public.apiHost;
    let publicTransport = createConnectTransport({
      baseUrl: apiHost,
      httpVersion: "1.1",
      useBinaryFormat: false,
      defaultTimeoutMs: 1000, // 1 second timeout for faster failure detection
    });
    
    let client = createClient(AuthService, publicTransport);
    let publicConfig;
    
    try {
      publicConfig = await client.getPublicConfig({});
    } catch (err: any) {
      // If internal API fails and we have a fallback, try public API
      if (config.apiHostInternal && apiHost === (config.apiHostInternal as string)) {
        console.warn(`[Server Auth Middleware] Internal API (${apiHost}) failed, trying public API as fallback:`, err?.code || err?.message);
        apiHost = config.public.apiHost;
        publicTransport = createConnectTransport({
          baseUrl: apiHost,
          httpVersion: "1.1",
          useBinaryFormat: false,
          defaultTimeoutMs: 1000,
        });
        client = createClient(AuthService, publicTransport);
        publicConfig = await client.getPublicConfig({});
      } else {
        throw err;
      }
    }
    
    // If auth is disabled via public config, skip all authentication checks
    if (publicConfig.disableAuth === true) {
      console.log("⚠️  WARNING: Auth disabled via public config, skipping authentication middleware");
      return;
    }
  } catch (err) {
    // If we can't fetch config, log warning but continue with auth checks
    // This ensures we don't accidentally disable auth if the API is down
    console.warn("[Server Auth Middleware] Failed to fetch public config, continuing with auth checks:", err);
  }

  const { getAccessToken } = await import("../utils/token");
  // Skip auth for public routes
  const publicRoutes = ["/auth/callback", "/auth/oauth-login", "/auth/oauth-signup", "/auth/silent-check", "/api/github/callback"];
  if (publicRoutes.includes(event.path)) {
    return;
  }
  const accessToken = getAccessToken(event);
  if (!accessToken) {
    throw createError({
      statusCode: 401,
      message: "Not authenticated",
    });
  }

  // Access token is a bearer token, not a JWT, so we don't verify it here
  // The API backend will validate it when making requests
  // We just need to ensure it exists
});
