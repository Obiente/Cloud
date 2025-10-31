import type { H3Event } from "h3";
import { getCookie } from "h3";
import { AUTH_COOKIE_NAME } from "./auth";

/**
 * Get the access token directly from the cookie on server-side
 * Falls back to session if cookie is not available
 * This is a simpler alternative to using the session when we just need the token
 */
export async function getServerToken(event: H3Event): Promise<string | undefined> {
  // Try to get the token directly from the cookie first
  const cookieToken = getCookie(event, AUTH_COOKIE_NAME);
  
  if (cookieToken && typeof cookieToken === "string" && cookieToken.trim() !== "") {
    return cookieToken;
  }

  // Fallback to session if cookie is not available
  try {
    const { getUserSession } = await import("./session");
    const session = await getUserSession(event);
    const sessionToken = session?.secure?.access_token;
    
    if (sessionToken && typeof sessionToken === "string" && sessionToken.trim() !== "") {
      return sessionToken;
    }
  } catch (err) {
    console.error("[getServerToken] Failed to get token from session:", err);
  }

  return undefined;
}
