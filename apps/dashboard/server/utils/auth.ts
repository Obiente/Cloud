import type { H3Event } from 'h3';
import type { ZitadelTokenResponse } from '@obiente/types';

export const AUTH_COOKIE_NAME = 'obiente_auth';
export const REFRESH_COOKIE_NAME = 'obiente_refresh';

export async function exchangeCodeForTokens(
  code: string,
  code_verifier: string,
  redirect_uri: string
): Promise<ZitadelTokenResponse> {
  const config = useRuntimeConfig();
  const response: ZitadelTokenResponse = await $fetch(`${config.public.oidcBase}/oauth/v2/token`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({
      grant_type: 'authorization_code',
      code,
      code_verifier,
      redirect_uri,
      client_id: config.public.oidcClientId,
    }),
  });

  return response;
}

export function setAuthCookies(event: H3Event, tokens: ZitadelTokenResponse) {
  const secure = process.env.NODE_ENV === 'production';

  // Set access token
  setCookie(event, AUTH_COOKIE_NAME, tokens.access_token, {
    httpOnly: true,
    secure,
    sameSite: 'lax',
    expires: new Date(Date.now() + tokens.expires_in * 1000),
    path: '/',
  });

  // Set refresh token if available
  if (tokens.refresh_token) {
    setCookie(event, REFRESH_COOKIE_NAME, tokens.refresh_token, {
      httpOnly: true,
      secure,
      sameSite: 'lax',
      expires: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days
      path: '/',
    });
  }
}

export function clearAuthCookies(event: H3Event) {
  deleteCookie(event, AUTH_COOKIE_NAME);
  deleteCookie(event, REFRESH_COOKIE_NAME);
}
