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

  const params = new URLSearchParams({
    client_id: OIDC.clientId,
    redirect_uri: config.public.requestHost + OIDC.redirectPath,
    response_type: OIDC.responseType,
    scope: OIDC.scope,
    code_challenge: code_challenge!,
    code_challenge_method: code_challenge_method!,
  });

  // This route is opened by an explicit user action, so it must remain
  // interactive. Omitting prompt lets Zitadel reuse a valid session or show its
  // login UI. prompt=none is reserved for /auth/silent-check; with Login V2 it
  // returns a raw "No active session found" response when the browser only has
  // stale or unusable Zitadel sessions.
  sendRedirect(event, `${OIDC.authority}/authorize?${params.toString()}`);
});
