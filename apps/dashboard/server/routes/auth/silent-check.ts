export default eventHandler(async (event) => {
  const config = useRuntimeConfig();
  const OIDC = {
    authority: config.public.oidcBase + "/oauth/v2",
    redirectPath: "/auth/callback",
    postLogoutRedirectUri: "/",
    scope: "openid profile email offline_access",
    responseType: "code",
    clientId: config.public.oidcClientId,
  };

  const { code_challenge, code_challenge_method } = await handlePKCE(event);

  // Always use silent auth for this endpoint (prompt: "none")
  // This is called from a popup window to check if user is logged into Zitadel
  // Use the same redirect_uri as regular auth (must match Zitadel config exactly)
  const params = new URLSearchParams({
    prompt: "none", // Force silent auth
    client_id: OIDC.clientId,
    redirect_uri: config.public.requestHost + OIDC.redirectPath,
    response_type: OIDC.responseType,
    scope: OIDC.scope,
    code_challenge: code_challenge!,
    code_challenge_method: code_challenge_method!,
    // Use state to identify silent auth requests
    state: "silent-auth",
  });

  // Redirect to Zitadel with silent auth
  // If user has active session, Zitadel will redirect back to callback with code
  // If no session, Zitadel will redirect to its login page (which we can't intercept)
  // The popup handler will timeout after 3 seconds and close the popup
  sendRedirect(event, `${OIDC.authority}/authorize?${params.toString()}`);
});
