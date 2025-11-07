import handlePKCE from "../../utils/handlePKCE";

/**
 * Login endpoint - redirects to Zitadel Authorization Code flow
 * This creates a persistent session in Zitadel (unlike ROPC grant)
 *
 * Note: ROPC (Resource Owner Password Credentials) grant does NOT create
 * persistent sessions in Zitadel. For session persistence, we must use
 * the Authorization Code flow with PKCE.
 *
 * We don't use prompt: "login" because that forces re-authentication every time.
 * Without a prompt parameter, Zitadel will:
 * - Use existing session if available (persistent)
 * - Prompt for login if no session exists
 */
export default defineEventHandler(async (event) => {
  try {
    const config = useRuntimeConfig();
    const OIDC = {
      authority: config.public.oidcBase + "/oauth/v2",
      redirectPath: "/auth/callback",
      scope: "openid profile email offline_access", // offline_access required for refresh tokens
      responseType: "code",
      clientId: config.public.oidcClientId,
    };

    const { code_challenge, code_challenge_method } = await handlePKCE(event);

    // Use Authorization Code flow WITHOUT prompt parameter
    // This allows Zitadel to use existing sessions (persistent) or prompt if needed
    // NOT using prompt: "login" because that prevents session persistence
    const params = new URLSearchParams({
      // No prompt parameter - let Zitadel handle session persistence naturally
      client_id: OIDC.clientId,
      redirect_uri: config.public.requestHost + OIDC.redirectPath,
      response_type: OIDC.responseType,
      scope: OIDC.scope,
      code_challenge: code_challenge!,
      code_challenge_method: code_challenge_method!,
      state: "login", // Identify this as a login flow
    });

    // Redirect to Zitadel for authentication
    // This will create/use a persistent session in Zitadel
    sendRedirect(event, `${OIDC.authority}/authorize?${params.toString()}`);
  } catch (error: any) {
    console.error("Login error:", error);
    throw createError({
      statusCode: error.statusCode || 500,
      message: error.message || "Login failed",
    });
  }
});
