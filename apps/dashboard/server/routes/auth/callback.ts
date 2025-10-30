import type { ZitadelTokenResponse } from "@obiente/types";

export default defineEventHandler(async (event) => {
  try {
    const { code, state, error } = getQuery<{
      code?: string;
      state?: string;
      error?: string;
    }>(event);
    const config = useRuntimeConfig();
    if (!code) {
      throw createError({ statusCode: 400, message: "Missing code" });
    }
    const { code_verifier } = await handlePKCE(event);

    const tokenResponse = await $fetch<ZitadelTokenResponse>(
      `${config.public.oidcBase}/oauth/v2/token`,
      {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({
          grant_type: "authorization_code",
          code,
          code_verifier,
          redirect_uri: config.public.requestHost + "/auth/callback",
          client_id: config.public.oidcClientId,
        }),
      }
    );

    // Set the session
    await getUserData(
      event,
      await setUserSession(event, {
        secure: {
          scope: tokenResponse.scope,
          token_type: tokenResponse.token_type,
          expires_in: tokenResponse.expires_in,
          refresh_token: tokenResponse.refresh_token,
          access_token: tokenResponse.access_token,
        },
      })
    );

    // Also set the auth cookie directly for easier access
    // Calculate expiry time (in seconds)
    const expirySeconds = tokenResponse.expires_in || 3600;
    const maxAge = expirySeconds - 60; // Subtract a minute for safety

    // Import the cookie name from auth utils
    const { AUTH_COOKIE_NAME } = await import("../../utils/auth");

    // Set the cookie with the token - using specific settings for SSR compatibility
    setCookie(event, AUTH_COOKIE_NAME, tokenResponse.access_token, {
      httpOnly: false, // Allow JavaScript access
      path: "/",
      maxAge,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      // Make sure domain is set correctly for SSR
      domain: undefined, // Let the browser determine the domain
    });
  } finally {
    return `<!DOCTYPE html>
<html>
<body>
<script>
localStorage.setItem('auth-completed', Date.now().toString());
window.close();
</script>
</body>
</html>`;
  }
});
