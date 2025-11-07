import type { ZitadelTokenResponse } from "@obiente/types";

export default defineEventHandler(async (event) => {
  try {
    const { code, state, error, error_description } = getQuery<{
      code?: string;
      state?: string;
      error?: string;
      error_description?: string;
    }>(event);
    const config = useRuntimeConfig();
    
    // Detect if this is a silent auth request (from popup)
    // Check state parameter or referer to see if it came from silent-check
    const isSilentAuth = state === "silent-auth" || getHeader(event, "referer")?.includes("silent-check") || false;
    
    // Detect if this is a signup flow
    const isSignup = state === "signup";
    
    // Detect if this is a login flow (from /api/auth/login endpoint)
    const isLogin = state === "login";
    
    // Handle OAuth errors (e.g., from silent auth when user is not logged in)
    if (error) {
      console.log("[OAuth Callback] OAuth error:", error, error_description);
      
      // Handle silent auth failures - when prompt: "none" is used but no session exists
      if (error === "login_required" || error === "interaction_required" || error === "no_session") {
        // Check if this is from silent auth iframe (detected by state or referer)
        if (isSilentAuth) {
          // Silent auth iframe - just silently fail, no popup
          return `<!DOCTYPE html>
<html>
<body>
<script>
// Silent auth failed in iframe - communicate to parent silently
if (window.parent && window.parent !== window) {
  window.parent.postMessage({ 
    type: 'silent-auth-error', 
    error: '${error}',
    message: 'No active session found'
  }, window.location.origin);
}
</script>
</body>
</html>`;
        } else {
          // Popup context - close and notify opener
          return `<!DOCTYPE html>
<html>
<body>
<script>
if (window.opener) {
  window.opener.postMessage({ 
    type: 'oauth-error', 
    error: '${error}',
    message: 'No active session found'
  }, window.location.origin);
  window.close();
} else {
  // No opener - just close
  window.close();
}
</script>
</body>
</html>`;
        }
      }
      
      // For other errors, show error page
      throw createError({ statusCode: 400, message: `OAuth error: ${error}${error_description ? ` - ${error_description}` : ''}` });
    }
    
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

    // For signup flow, don't automatically log in - show success message instead
    if (isSignup) {
      // Don't set session or cookie - user needs to login manually after signup
      // Close popup and show success message
      return `<!DOCTYPE html>
<html>
<body>
<script>
if (window.opener) {
  window.opener.postMessage({ 
    type: 'signup-success',
    message: 'Account created successfully. Please log in.'
  }, window.location.origin);
  window.close();
} else {
  // Not in popup - redirect to homepage with success message
  window.location.href = '/?signup=success';
}
</script>
</body>
</html>`;
    }

    // Set the session (for login flows, not signup)
    await getUserData(
      event,
      await setUserSession(event, {
        secure: {
          scope: tokenResponse.scope,
          token_type: tokenResponse.token_type,
          expires_in: tokenResponse.expires_in,
          refresh_token: tokenResponse.refresh_token,
          access_token: tokenResponse.access_token,
          id_token: tokenResponse.id_token, // Store id_token for logout
        },
      })
    );

    // Also set the auth cookie directly for easier access
    // Always use long expiry to remember the user (unless they explicitly logout)
    const expirySeconds = tokenResponse.expires_in || 3600;
    // Use refresh token expiry (typically 30 days) or 7 days, whichever is longer
    // This ensures the user stays logged in unless they explicitly logout
    const maxAge = Math.max(expirySeconds * 7, 7 * 24 * 60 * 60); // At least 7 days

    const { AUTH_COOKIE_NAME } = await import("../../utils/auth");

    setCookie(event, AUTH_COOKIE_NAME, tokenResponse.access_token, {
      httpOnly: false,
      path: "/",
      maxAge,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      domain: undefined,
    });

    // For silent auth (iframe), always use postMessage to parent
    if (isSilentAuth) {
      return `<!DOCTYPE html>
<html>
<body>
<script>
// Silent auth succeeded - communicate to parent (iframe)
if (window.parent && window.parent !== window) {
  window.parent.postMessage({ 
    type: 'silent-auth-success'
  }, window.location.origin);
} else if (window.opener) {
  // Fallback for popup
  window.opener.postMessage({ 
    type: 'silent-auth-success'
  }, window.location.origin);
  window.close();
} else {
  // Fallback - use localStorage
  localStorage.setItem('auth-completed', Date.now().toString());
  window.close();
}
</script>
</body>
</html>`;
    }

    // For login flow from /api/auth/login (redirect, not popup)
    if (isLogin) {
      // Redirect to dashboard after successful login
      sendRedirect(event, "/dashboard");
      return;
    }

    // Regular popup auth
    return `<!DOCTYPE html>
<html>
<body>
<script>
localStorage.setItem('auth-completed', Date.now().toString());
window.close();
</script>
</body>
</html>`;
  } catch (error: any) {
    console.error("[OAuth Callback] Error:", error);
    return `<!DOCTYPE html>
<html>
<body>
<script>
if (window.parent && window.parent !== window) {
  window.parent.postMessage({ 
    type: 'silent-auth-error', 
    error: 'callback_error',
    message: 'Authentication callback failed'
  }, window.location.origin);
} else if (window.opener) {
  window.opener.postMessage({ 
    type: 'silent-auth-error', 
    error: 'callback_error',
    message: 'Authentication callback failed'
  }, window.location.origin);
  window.close();
} else {
  // No parent or opener - this shouldn't happen in popup context
  // Just close the window
  window.close();
}
</script>
</body>
</html>`;
  }
});
