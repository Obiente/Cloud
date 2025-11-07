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
    prompt: "create", // Force signup/registration UI (Zitadel uses "create" not "register")
    client_id: OIDC.clientId,
    redirect_uri: config.public.requestHost + OIDC.redirectPath,
    response_type: OIDC.responseType,
    scope: OIDC.scope,
    code_challenge: code_challenge!,
    code_challenge_method: code_challenge_method!,
    // Add state to identify this as a signup flow
    state: "signup",
  });

  sendRedirect(event, `${OIDC.authority}/authorize?${params.toString()}`);
});

