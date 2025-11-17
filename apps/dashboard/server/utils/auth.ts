import type { User, UserSession, ZitadelTokenResponse } from "@obiente/types";
import type { H3Event } from "h3";
import { setCookie } from "h3";
import { setUserSession, clearUserSession } from "./session";

export const AUTH_COOKIE_NAME = "obiente_auth";
export const REFRESH_COOKIE_NAME = "obiente_refresh";

export async function exchangeCodeForTokens(
  code: string,
  code_verifier: string,
  redirect_uri: string
): Promise<ZitadelTokenResponse> {
  const config = useRuntimeConfig();
  const response = await $fetch<ZitadelTokenResponse>(
    `${config.public.oidcBase}/oauth/v2/token`,
    {
      method: "POST",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: new URLSearchParams({
        grant_type: "authorization_code",
        code,
        code_verifier,
        redirect_uri,
        client_id: config.public.oidcClientId,
      }),
    }
  ).catch((error) => {
    const errorMessage = error instanceof Error 
      ? error.message 
      : typeof error === 'string' 
        ? error 
        : error?.message || String(error) || 'Unknown error';
    throw new Error(errorMessage);
  });
  return response;
}

/**
 * Refresh the access token using the refresh token
 */
async function refreshAccessToken(
  event: H3Event,
  session: UserSession
): Promise<UserSession | null> {
  const refreshToken = session.secure?.refresh_token;
  if (!refreshToken) {
    return null;
  }

  try {
    const config = useRuntimeConfig();
    const tokenResponse = await $fetch<ZitadelTokenResponse>(
      `${config.public.oidcBase}/oauth/v2/token`,
      {
        method: "POST",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({
          grant_type: "refresh_token",
          refresh_token: refreshToken,
          client_id: config.public.oidcClientId,
        }),
      }
    );

    // Update session with new tokens
    const updatedSession = await setUserSession(event, {
      secure: {
        scope: tokenResponse.scope,
        token_type: tokenResponse.token_type,
        expires_in: tokenResponse.expires_in,
        refresh_token: tokenResponse.refresh_token,
        access_token: tokenResponse.access_token,
        id_token: tokenResponse.id_token || session.secure?.id_token,
      },
    });

    // Update auth cookie
    const expirySeconds = tokenResponse.expires_in || 3600;
    const maxAge = Math.max(expirySeconds * 7, 7 * 24 * 60 * 60);
    setCookie(event, AUTH_COOKIE_NAME, tokenResponse.access_token, {
      httpOnly: false,
      path: "/",
      maxAge,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      domain: undefined,
    });

    return updatedSession;
  } catch (error) {
    console.error("Failed to refresh token:", error);
    return null;
  }
}

export async function getUserData(
  event: H3Event,
  session: UserSession
): Promise<void> {
  if (!session.secure?.access_token) return;
  
  const config = useRuntimeConfig();
  let accessToken = session.secure.access_token;
  let currentSession = session;

  // Try to fetch user data
  let response = await $fetch<User>(
    `${config.public.oidcBase}/oidc/v1/userinfo`,
    {
      headers: {
        Authorization: `Bearer ${accessToken}`,
      },
    }
  ).catch(async (e: any) => {
    // If we get a 401, try to refresh the token
    if (e?.statusCode === 401 || e?.status === 401) {
      console.log("[getUserData] Token expired, attempting refresh...");
      const refreshedSession = await refreshAccessToken(event, currentSession);
      
      if (refreshedSession?.secure?.access_token) {
        // Update currentSession to use the refreshed token
        currentSession = refreshedSession;
        accessToken = refreshedSession.secure.access_token;
        
        // Retry with new token
        try {
          const userData = await $fetch<User>(
            `${config.public.oidcBase}/oidc/v1/userinfo`,
            {
              headers: {
                Authorization: `Bearer ${accessToken}`,
              },
            }
          );
          // Update session with user data AND ensure refreshed session is persisted
          await setUserSession(event, { 
            user: userData,
            secure: refreshedSession.secure 
          });
          return userData;
        } catch (retryError: any) {
          console.error("[getUserData] Failed to fetch user data after refresh:", retryError);
          // If refresh token is also invalid, clear session
          if (retryError?.statusCode === 401 || retryError?.status === 401) {
            await clearUserSession(event);
          }
          return null;
        }
      } else {
        // Refresh failed, clear session
        console.error("[getUserData] Token refresh failed, clearing session");
        await clearUserSession(event);
        return null;
      }
    } else {
      console.error("[getUserData] Failed to fetch user data:", e);
      return null;
    }
  });

  if (response) {
    // Ensure we persist the current session (which may have been refreshed)
    await setUserSession(event, { 
      user: response,
      secure: currentSession.secure 
    });
  }
}
