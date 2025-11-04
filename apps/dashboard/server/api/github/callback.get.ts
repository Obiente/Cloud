/**
 * GitHub OAuth Callback Handler
 * 
 * This endpoint receives the OAuth code from GitHub, exchanges it for an access token,
 * stores it securely, and redirects the user back to the settings page.
 * 
 * Callback URL format: /api/github/callback?code=...&state=...
 * 
 * This URL must be registered in your GitHub OAuth App settings:
 * - Production: https://your-domain.com/api/github/callback
 * - Development: http://localhost:3000/api/github/callback
 */
import type { ZitadelTokenResponse } from "@obiente/types";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const { code, state, error } = query as {
    code?: string;
    state?: string;
    error?: string;
  };

  // Check for OAuth errors from GitHub
  if (error) {
    console.error("[GitHub OAuth] Error from GitHub:", error);
    // Redirect to settings page with error
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(error)}`
    );
  }

  // Validate code is present
  if (!code) {
    console.error("[GitHub OAuth] Missing authorization code");
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent("missing_code")}`
    );
  }

  // Verify state parameter (CSRF protection)
  // TODO: Retrieve stored state from session and verify it matches
  // For now, we'll skip strict validation but log it
  if (state) {
    console.log("[GitHub OAuth] State parameter received:", state);
    // TODO: Verify against sessionStorage or server-side session
  }

  try {
    const runtimeConfig = useRuntimeConfig();
    const githubClientId = runtimeConfig.public.githubClientId;
    const githubClientSecret = runtimeConfig.githubClientSecret; // Server-side only

    if (!githubClientId || !githubClientSecret) {
      console.error("[GitHub OAuth] Missing GitHub credentials in config");
      return sendRedirect(
        event,
        `/settings?tab=integrations&provider=github&error=${encodeURIComponent("configuration_error")}`
      );
    }

    // Exchange authorization code for access token
    // Get the redirect URI from the request (must match EXACTLY what frontend sent to GitHub)
    // The frontend uses window.location.origin, so we need to construct the same value
    // Use the request URL to get the actual origin used by the browser
    const requestUrl = new URL(event.node.req.url || '/', `http://${event.node.req.headers.host || 'localhost:3000'}`);
    const protocol = event.node.req.headers['x-forwarded-proto'] || (requestUrl.protocol === 'https:' ? 'https' : 'http');
    const host = event.node.req.headers.host || requestUrl.host || 'localhost:3000';
    const redirectUri = `${protocol}://${host}/api/github/callback`;
    
    console.log("[GitHub OAuth] Redirect URI being used for token exchange:", redirectUri);
    console.log("[GitHub OAuth] Authorization code received:", code ? `${code.substring(0, 10)}...` : "none");
    
    let tokenResponse: {
      access_token?: string;
      token_type?: string;
      scope?: string;
      error?: string;
      error_description?: string;
      error_uri?: string;
    };
    
    try {
      tokenResponse = await $fetch("https://github.com/login/oauth/access_token", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        body: {
          client_id: githubClientId,
          client_secret: githubClientSecret,
          code,
          // redirect_uri must match EXACTLY what was sent in the authorization request
          redirect_uri: redirectUri,
        },
      }) as any;
      
      console.log("[GitHub OAuth] Token exchange response received:", {
        hasAccessToken: !!tokenResponse.access_token,
        hasError: !!tokenResponse.error,
        error: tokenResponse.error,
        errorDescription: tokenResponse.error_description,
      });
    } catch (fetchError: any) {
      console.error("[GitHub OAuth] Token exchange request failed:", {
        message: fetchError.message,
        status: fetchError.status,
        statusText: fetchError.statusText,
        data: fetchError.data,
        response: fetchError.response,
      });
      throw new Error(`Token exchange failed: ${fetchError.message || "Unknown error"}`);
    }

    // Check for GitHub API errors
    if (tokenResponse.error) {
      console.error("[GitHub OAuth] GitHub API error:", {
        error: tokenResponse.error,
        errorDescription: tokenResponse.error_description,
        errorUri: tokenResponse.error_uri,
        redirectUri,
        code: code ? `${code.substring(0, 10)}...` : "none",
      });
      
      // Provide more helpful error messages based on common issues
      let errorMessage = tokenResponse.error;
      if (tokenResponse.error_description) {
        errorMessage += `: ${tokenResponse.error_description}`;
      }
      
      if (tokenResponse.error === "bad_verification_code") {
        errorMessage = "The authorization code has expired or is invalid. Please try connecting again.";
      } else if (tokenResponse.error === "redirect_uri_mismatch") {
        errorMessage = `Redirect URI mismatch. Expected redirect_uri to match the one registered with GitHub. Used: ${redirectUri}`;
      }
      
      throw new Error(errorMessage);
    }

    if (!tokenResponse.access_token) {
      console.error("[GitHub OAuth] No access token in response:", {
        response: JSON.stringify(tokenResponse),
        redirectUri,
        code: code ? `${code.substring(0, 10)}...` : "none",
      });
      throw new Error("No access token received from GitHub. Please check that the authorization code is valid and hasn't expired.");
    }

    // TODO: Store the access token securely
    // Options:
    // 1. Store in database linked to user account
    // 2. Store in encrypted session
    // 3. Store in secure cookie (less secure)
    

    // Get user info from GitHub to verify connection
    const userResponse = await $fetch<{
      login: string;
      name?: string;
      email?: string;
      avatar_url?: string;
    }>("https://api.github.com/user", {
      headers: {
        Authorization: `Bearer ${tokenResponse.access_token}`,
        Accept: "application/vnd.github.v3+json",
      },
    });

    console.log("[GitHub OAuth] Connected as:", userResponse.login);

    // Check if auth is disabled (development mode)
    const isAuthDisabled = process.env.DISABLE_AUTH === "true";
    
    // Get user's access token to authenticate API request
    // Try cookie first, then fall back to session
    // If auth is disabled, we can skip this requirement
    const { getServerToken } = await import("../../utils/serverAuth");
    let userAccessToken = await getServerToken(event);

    // Log token availability for debugging
    if (userAccessToken) {
      console.log("[GitHub OAuth] User access token found (length):", userAccessToken.length);
    } else {
      console.warn("[GitHub OAuth] No user access token found");
      // Try to get all cookies for debugging
      const allCookies = event.node.req.headers.cookie;
      console.log("[GitHub OAuth] Available cookies:", allCookies ? "present" : "none");
    }

    // Always try to refresh the token proactively, even if we have one
    // This ensures we have a fresh token before making the API call
    if (!isAuthDisabled) {
      try {
        const { getUserSession } = await import("../../utils/session");
        const session = await getUserSession(event);
        const refreshToken = session?.secure?.refresh_token;
        
        if (refreshToken) {
          console.log("[GitHub OAuth] Proactively refreshing token...");
          const config = useRuntimeConfig();
          
          // Exchange refresh token for new tokens
          const tokenResponse = await $fetch<ZitadelTokenResponse>(
            `${config.public.oidcBase}/oauth/v2/token`,
            {
              method: "POST",
              headers: { "Content-Type": "application/x-www-form-urlencoded" },
              body: new URLSearchParams({
                grant_type: "refresh_token",
                refresh_token: refreshToken,
                client_id: config.public.oidcClientId,
              }),
            }
          );
          
          if (tokenResponse?.access_token) {
            // Update session with new tokens
            const { setUserSession } = await import("../../utils/session");
            await setUserSession(event, {
              secure: {
                scope: tokenResponse.scope,
                token_type: tokenResponse.token_type,
                expires_in: tokenResponse.expires_in,
                refresh_token: tokenResponse.refresh_token,
                access_token: tokenResponse.access_token,
              },
            });
            
            userAccessToken = tokenResponse.access_token;
            console.log("[GitHub OAuth] Token refreshed proactively");
            
            // Update the cookie with the new token
            const { AUTH_COOKIE_NAME } = await import("../../utils/auth");
            setCookie(event, AUTH_COOKIE_NAME, tokenResponse.access_token, {
              httpOnly: false,
              path: "/",
              maxAge: (tokenResponse.expires_in || 3600) - 60,
              secure: process.env.NODE_ENV === "production",
              sameSite: "lax",
              domain: undefined,
            });
          } else {
            console.warn("[GitHub OAuth] Token refresh returned no access token, using existing token");
          }
        } else {
          console.warn("[GitHub OAuth] No refresh token available, using existing token if available");
        }
      } catch (refreshError: any) {
        console.warn("[GitHub OAuth] Proactive token refresh failed, using existing token:", refreshError.message);
        // Continue with existing token if refresh fails
      }
    }

    if (!userAccessToken) {
      if (isAuthDisabled) {
        // In development mode, use a dummy token - the Go API will ignore it
        // and use the mock dev user instead
        console.log("[GitHub OAuth] Auth disabled - using dummy token for API call");
        userAccessToken = "dev-dummy-token";
      } else {
        console.error("[GitHub OAuth] No user access token available - user must be logged in");
        // Redirect to login with a helpful message
        return sendRedirect(
          event,
          `/auth/login?redirect=${encodeURIComponent("/settings?tab=integrations&provider=github&error=" + encodeURIComponent("Please log in to connect your GitHub account"))}`
        );
      }
    }

    // Parse state parameter to get connection type and org ID
    // The state contains encoded JSON with connection info
    let connectionType: string | undefined;
    let orgId: string | undefined;
    
    if (state) {
      try {
        // Decode the state parameter (base64 JSON)
        const decodedState = Buffer.from(state, "base64").toString("utf-8");
        const stateData = JSON.parse(decodedState);
        connectionType = stateData.type;
        orgId = stateData.orgId;
      } catch (err) {
        console.error("[GitHub OAuth] Failed to parse state parameter:", err);
        // Fall back to user connection if state parsing fails
        connectionType = "user";
      }
    } else {
      // Default to user connection if no state
      connectionType = "user";
    }

    // Call Go API to store the GitHub token in database
    try {
      // Create Connect client manually for server API route
      // The plugin transport isn't available in API routes, so we create it here
      const config = useRuntimeConfig();
      const { createConnectTransport } = await import("@connectrpc/connect-node");
      const { createAuthInterceptor } = await import("~/lib/transport");
      
      // Create an interceptor that uses the token we already have
      const getToken = () => Promise.resolve(userAccessToken || undefined);
      const authInterceptor = createAuthInterceptor(getToken);
      
      const transport = createConnectTransport({
        baseUrl: config.public.apiHost,
        httpVersion: "1.1",
        useBinaryFormat: false,
        interceptors: [authInterceptor],
      });

      const { createClient } = await import("@connectrpc/connect");
      const { AuthService } = await import("@obiente/proto");
      const client = createClient(AuthService, transport);

      const authHeaders = new Headers();
      if (userAccessToken) {
        authHeaders.set("Authorization", `Bearer ${userAccessToken}`);
      }

      let success = false;
      let apiError: Error | null = null;
      const baseRedirectUrl = `/settings?tab=integrations&provider=github&success=true&username=${encodeURIComponent(userResponse.login)}`;
      let redirectUrl = baseRedirectUrl;

      console.log("[GitHub OAuth] Storing token in database:", {
        connectionType,
        orgId: orgId || "none",
        username: userResponse.login,
        hasToken: !!tokenResponse.access_token,
      });

      try {
        if (connectionType === "organization" && orgId) {
          // Connect as organization
          const response = await client.connectOrganizationGitHub({
            organizationId: orgId,
            accessToken: tokenResponse.access_token,
            username: userResponse.login,
            scope: tokenResponse.scope || "",
          }, { headers: authHeaders });
          success = response.success;
          if (success && orgId) {
            redirectUrl += `&orgId=${encodeURIComponent(orgId)}`;
          }
          console.log("[GitHub OAuth] Organization connection response:", { success, orgId });
        } else {
          // Connect as user
          const response = await client.connectGitHub({
            accessToken: tokenResponse.access_token,
            username: userResponse.login,
            scope: tokenResponse.scope || "",
          }, { headers: authHeaders });
          success = response.success;
          console.log("[GitHub OAuth] User connection response:", { success, username: response.username });
        }
      } catch (apiCallError: any) {
        console.error("[GitHub OAuth] API call failed:", {
          message: apiCallError.message,
          code: apiCallError.code,
          codeType: typeof apiCallError.code,
          details: apiCallError.details,
          errorObject: JSON.stringify(apiCallError, Object.getOwnPropertyNames(apiCallError)),
        });
        
        // If the error is authentication-related, try refreshing the token once
        // Connect-RPC error codes: 16 = UNAUTHENTICATED
        const errorCode = apiCallError.code;
        const errorMessage = apiCallError.message || "";
        const isUnauthenticated = 
          errorCode === "UNAUTHENTICATED" || 
          errorCode === 16 || 
          String(errorCode) === "16" ||
          errorMessage.toLowerCase().includes("unauthenticated") ||
          errorMessage.toLowerCase().includes("invalid authorization token");
        
        console.log("[GitHub OAuth] Error analysis:", {
          errorCode,
          errorCodeType: typeof errorCode,
          errorMessage,
          isUnauthenticated,
          isAuthDisabled,
          willAttemptRefresh: isUnauthenticated && !isAuthDisabled,
        });
        
        // If we get an authentication error, it means the proactive refresh didn't work
        // This could happen if the refresh token wasn't available or expired
        // In this case, we can't retry - the user needs to log in again
        if (isUnauthenticated && !isAuthDisabled) {
          console.error("[GitHub OAuth] Authentication error after proactive refresh - user may need to log in again");
          apiError = apiCallError;
        } else {
          apiError = apiCallError;
        }
      }

      if (!success) {
        const errorMsg = apiError?.message || "Failed to save GitHub token to database";
        console.error("[GitHub OAuth] Token storage failed:", errorMsg);
        throw new Error(errorMsg);
      }

      console.log("[GitHub OAuth] Token saved to database successfully");

      // Redirect back to settings page with success
      return sendRedirect(event, redirectUrl);
    } catch (apiErr: any) {
      console.error("[GitHub OAuth] Failed to save token to API:", apiErr);
      const apiMessage = typeof apiErr?.message === "string" ? apiErr.message : "failed_to_save_token";
      return sendRedirect(
        event,
        `/settings?tab=integrations&provider=github&error=${encodeURIComponent(apiMessage)}`
      );
    }
  } catch (err: any) {
    console.error("[GitHub OAuth] Token exchange failed:", err);
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(err.message || "token_exchange_failed")}`
    );
  }
});

