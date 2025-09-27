export default defineEventHandler(async event => {
  const body = await readBody(event);
  const { code, code_verifier, redirect_uri } = body;

  if (!code || !code_verifier) {
    throw createError({ statusCode: 400, message: 'Missing code or code_verifier' });
  }

  const tokenResponse = await exchangeCodeForTokens(code, code_verifier, redirect_uri);

  // Set secure HTTP-only cookies
  setAuthCookies(event, tokenResponse);

  // Return minimal response (tokens are in cookies)
});
