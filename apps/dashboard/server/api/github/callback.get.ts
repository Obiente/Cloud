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
    const tokenResponse = await $fetch<{
      access_token: string;
      token_type: string;
      scope: string;
    }>("https://github.com/login/oauth/access_token", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: {
        client_id: githubClientId,
        client_secret: githubClientSecret,
        code,
        // State should match what we sent
        // redirect_uri should match what we registered with GitHub
        redirect_uri: `${runtimeConfig.public.requestHost}/api/github/callback`,
      },
    });

    if (!tokenResponse.access_token) {
      throw new Error("No access token received from GitHub");
    }

    // TODO: Store the access token securely
    // Options:
    // 1. Store in database linked to user account
    // 2. Store in encrypted session
    // 3. Store in secure cookie (less secure)
    
    // For now, we'll need to implement token storage
    // This is a placeholder - implement based on your security requirements
    console.log("[GitHub OAuth] Token received (length):", tokenResponse.access_token.length);
    console.log("[GitHub OAuth] Scopes:", tokenResponse.scope);

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

    if (!userAccessToken) {
      if (isAuthDisabled) {
        // In development mode, use a dummy token - the Go API will ignore it
        // and use the mock dev user instead
        console.log("[GitHub OAuth] Auth disabled - using dummy token for API call");
        userAccessToken = "dev-dummy-token";
      } else {
        console.error("[GitHub OAuth] No user access token available - user must be logged in");
        // Redirect with a more helpful error message
        return sendRedirect(
          event,
          `/settings?tab=integrations&provider=github&error=${encodeURIComponent("Please log in to connect your GitHub account")}`
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
      const config = useRuntimeConfig();
      const apiHost = config.public.apiHost;

      // Import Connect client and message types
      const { AuthService, ConnectGitHubRequestSchema, ConnectOrganizationGitHubRequestSchema } = await import("@obiente/proto");
      const { create } = await import("@bufbuild/protobuf");
      const { createConnectTransport } = await import("@connectrpc/connect-node");
      const { createAuthInterceptor } = await import("~/lib/transport");
      const { createClient } = await import("@connectrpc/connect");
      
      // Create transport with auth interceptor
      const transport = createConnectTransport({
        baseUrl: `${apiHost}`,
        httpVersion: "2",
        interceptors: [
          createAuthInterceptor(() => Promise.resolve(userAccessToken)),
        ],
      });

      // Create client
      const client = createClient(AuthService, transport);

      let success = false;
      const baseRedirectUrl = `/settings?tab=integrations&provider=github&success=true&username=${encodeURIComponent(userResponse.login)}`;
      let redirectUrl = baseRedirectUrl;

      if (connectionType === "organization" && orgId) {
        // Connect as organization
        const request = create(ConnectOrganizationGitHubRequestSchema, {
          organizationId: orgId,
          accessToken: tokenResponse.access_token,
          username: userResponse.login,
          scope: tokenResponse.scope,
        });

        const connectResponse = await client.connectOrganizationGitHub(request);
        success = connectResponse.success;
      } else {
        // Connect as user
        const request = create(ConnectGitHubRequestSchema, {
          accessToken: tokenResponse.access_token,
          username: userResponse.login,
          scope: tokenResponse.scope,
        });

        const connectResponse = await client.connectGitHub(request);
        success = connectResponse.success;
      }

      if (!success) {
        throw new Error("Failed to save GitHub token to database");
      }

      console.log("[GitHub OAuth] Token saved to database successfully");

      // Redirect back to settings page with success
      return sendRedirect(event, redirectUrl);
    } catch (apiErr: any) {
      console.error("[GitHub OAuth] Failed to save token to API:", apiErr);
      return sendRedirect(
        event,
        `/settings?tab=integrations&provider=github&error=${encodeURIComponent("failed_to_save_token")}`
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

