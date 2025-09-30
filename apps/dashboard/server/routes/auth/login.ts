export default eventHandler(async event => {
  const config = useRuntimeConfig();
  const OIDC = {
    authority: config.public.oidcBase + '/oauth/v2',
    redirectPath: '/auth/callback',
    postLogoutRedirectUri: '/',
    scope: 'openid profile email',
    responseType: 'code',
    clientId: config.public.oidcClientId,
  };
  const { code_challenge, code_challenge_method } = await handlePKCE(event);
  const params = new URLSearchParams({
    prompt: 'none',
    client_id: OIDC.clientId,
    redirect_uri: useRuntimeConfig().requestHost + '/auth/callback',
    response_type: OIDC.responseType,
    scope: OIDC.scope,
    code_challenge: code_challenge!,
    code_challenge_method: code_challenge_method!,
  });
  sendRedirect(event, `${OIDC.authority}/authorize?${params.toString()}`);
});
