import type { User, UserSession, ZitadelTokenResponse } from "@obiente/types";
import type { H3Event } from "@nuxt/nitro-server/h3";
import { setCookie, createError } from "@nuxt/nitro-server/h3";
import { setUserSession, clearUserSession } from "./session";
import { getAuthRuntimeConfig } from "./auth-runtime-config";

export const AUTH_COOKIE_NAME = "obiente_auth";
export const REFRESH_COOKIE_NAME = "obiente_refresh";

export async function exchangeCodeForTokens(
  code: string,
  code_verifier: string,
  redirect_uri: string
): Promise<ZitadelTokenResponse> {
  const config = getAuthRuntimeConfig();
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
    const errorMessage =
      error instanceof Error
        ? error.message
        : typeof error === "string"
        ? error
        : error?.message || String(error) || "Unknown error";
    throw new Error(errorMessage);
  });
  return response;
}

// Keep transient provider failures distinct from an invalid/expired grant.
export async function refreshUserSession(
  event: H3Event,
  session: UserSession,
  signal?: AbortSignal
): Promise<UserSession> {
  const refreshToken = session.secure?.refresh_token;
  if (!refreshToken) {
    await clearUserSession(event);
    throw createError({
      statusCode: 401,
      message: "Session expired. Please log in again.",
    });
  }
  const config = getAuthRuntimeConfig(event);
  let response: ZitadelTokenResponse;
  try {
    response = await $fetch<ZitadelTokenResponse>(
      `${config.public.oidcBase}/oauth/v2/token`,
      {
        method: "POST",
        retry: false,
        timeout: 5000,
        signal,
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: new URLSearchParams({
          grant_type: "refresh_token",
          refresh_token: refreshToken,
          client_id: config.public.oidcClientId,
        }),
      }
    );
  } catch (error) {
    const code = (error as { data?: { error?: string } })?.data?.error;
    if (code === "invalid_grant" || code === "invalid_token") {
      await clearUserSession(event);
      throw createError({
        statusCode: 401,
        message: "Session expired. Please log in again.",
      });
    }
    throw createError({
      statusCode: 503,
      message: "Authentication is temporarily unavailable. Please try again.",
    });
  }
  if (
    typeof response?.access_token !== "string" ||
    !response.access_token.trim() ||
    !Number.isFinite(response.expires_in) ||
    response.expires_in <= 0
  ) {
    throw createError({
      statusCode: 502,
      message: "Authentication provider returned an invalid token.",
    });
  }
  const updated = await setUserSession(event, {
    secure: {
      scope: response.scope,
      token_type: response.token_type,
      expires_in: response.expires_in,
      expires_at: Date.now() + response.expires_in * 1000,
      refresh_token: response.refresh_token || refreshToken,
      access_token: response.access_token,
      id_token: response.id_token || session.secure?.id_token,
    },
  });
  setCookie(event, AUTH_COOKIE_NAME, response.access_token, {
    httpOnly: false,
    path: "/",
    maxAge: Math.max(response.expires_in * 7, 7 * 24 * 60 * 60),
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
  });
  return updated;
}

export async function getUserData(
  event: H3Event,
  session: UserSession,
  signal?: AbortSignal
): Promise<void> {
  if (!session.secure?.access_token) return;
  const config = getAuthRuntimeConfig(event);
  const getUser = (token: string) =>
    $fetch<User>(`${config.public.oidcBase}/oidc/v1/userinfo`, {
      headers: { Authorization: `Bearer ${token}` },
      retry: false,
      timeout: 5000,
      signal,
    });
  const unauthorized = (error: unknown) => {
    const value = error as { statusCode?: number; status?: number };
    return (value?.statusCode ?? value?.status) === 401;
  };
  let currentSession = session;
  let response: User;
  try {
    response = await getUser(session.secure.access_token);
  } catch (error) {
    if (!unauthorized(error))
      throw createError({
        statusCode: 503,
        message: "Authentication is temporarily unavailable. Please try again.",
      });
    currentSession = await refreshUserSession(event, session, signal);
    try {
      response = await getUser(currentSession.secure!.access_token);
    } catch (retryError) {
      if (unauthorized(retryError)) {
        await clearUserSession(event);
        throw createError({
          statusCode: 401,
          message: "Session expired. Please log in again.",
        });
      }
      throw createError({
        statusCode: 503,
        message: "Authentication is temporarily unavailable. Please try again.",
      });
    }
  }
  await setUserSession(event, {
    user: response,
    secure: currentSession.secure,
  });
}
