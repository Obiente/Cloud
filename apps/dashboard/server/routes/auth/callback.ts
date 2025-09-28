import type { ZitadelTokenResponse } from '@obiente/types';

export default defineEventHandler(async event => {
  const { code, state, error } = getQuery<{ code?: string; state?: string; error?: string }>(event);
  const config = useRuntimeConfig();
  if (!code) {
    throw createError({ statusCode: 400, message: 'Missing code' });
  }
  const { code_verifier } = await handlePKCE(event);
  console.log(code_verifier, code);

  const tokenResponse = await $fetch<ZitadelTokenResponse>(
    `${config.public.oidcBase}/oauth/v2/token`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        grant_type: 'authorization_code',
        code,
        code_verifier,
        redirect_uri: config.requestHost + '/auth/callback',
        client_id: config.public.oidcClientId,
      }),
    }
  );

  // Set secure HTTP-only cookies

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
  return `<!DOCTYPE html>
<html>
<body>
  <script>
    window.close();
  </script>
</body>
</html>`;
});
