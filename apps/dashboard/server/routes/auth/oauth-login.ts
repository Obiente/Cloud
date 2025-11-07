export default eventHandler(async (event) => {
  const config = useRuntimeConfig();
  const OIDC = {
    authority: config.public.oidcBase + "/oauth/v2",
    redirectPath: "/auth/callback",
    postLogoutRedirectUri: "/",
    scope: "openid profile email offline_access", // offline_access is required for refresh tokens
    responseType: "code",
    clientId: config.public.oidcClientId,
  };

  const { code_challenge, code_challenge_method } = await handlePKCE(event);

  // Check for existing session cookie to determine if we should use silent auth
  const { AUTH_COOKIE_NAME } = await import("../../utils/auth");
  const sessionCookie = getCookie(event, AUTH_COOKIE_NAME);
  // If no session cookie exists, try silent auth (check if user is logged into Zitadel)
  // If session cookie exists, don't use prompt (let Zitadel use existing session)
  const isSilent = !sessionCookie;

  const params = new URLSearchParams({
    client_id: OIDC.clientId,
    redirect_uri: config.public.requestHost + OIDC.redirectPath,
    response_type: OIDC.responseType,
    scope: OIDC.scope,
    code_challenge: code_challenge!,
    code_challenge_method: code_challenge_method!,
  });

  // Use silent auth if no local session, otherwise let Zitadel handle it naturally
  // NOT using prompt: "login" because that prevents session persistence
  if (isSilent) {
    params.set("prompt", "none");
  }

  sendRedirect(event, `${OIDC.authority}/authorize?${params.toString()}`);
});
