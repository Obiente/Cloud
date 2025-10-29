import type { H3Event } from 'h3';
import { getCookie } from 'h3';
import { AUTH_COOKIE_NAME } from './auth';

/**
 * Get the access token directly from the cookie on server-side
 * This is a simpler alternative to using the session when we just need the token
 */
export function getServerToken(event: H3Event): string | undefined {
  // Try to get the token directly from the cookie
  const token = getCookie(event, AUTH_COOKIE_NAME);
  
  // Validate the token
  if (!token || typeof token !== 'string' || token.trim() === '') {
    return undefined;
  }
  
  return token;
}
