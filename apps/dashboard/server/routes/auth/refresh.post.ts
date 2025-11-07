import type { ZitadelTokenResponse } from "@obiente/types";
import { handleZitadelError } from "../../utils/token";

/**
 * API endpoint to refresh the access token using the refresh token
 * This is used when the access token expires
 */
export default defineEventHandler(async (event) => {
  if (process.env.DISABLE_AUTH === "true") {
    return {
      accessToken: "dev-dummy-token",
      expiresIn: 3600,
    };
  }

  try {
    const session = await getUserSession(event);
    const refreshToken = session.secure?.refresh_token;

    if (!refreshToken) {
      throw createError({
        statusCode: 401,
        message: "No refresh token available",
      });
    }

    const config = useRuntimeConfig();

    // Exchange refresh token for new tokens
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

    // Update session with new tokens (including id_token if present)
    await setUserSession(event, {
      secure: {
        scope: tokenResponse.scope,
        token_type: tokenResponse.token_type,
        expires_in: tokenResponse.expires_in,
        refresh_token: tokenResponse.refresh_token, // Zitadel uses rotating refresh tokens
        access_token: tokenResponse.access_token,
        id_token: tokenResponse.id_token, // Store id_token for logout
      },
    });

    // Validate the token before returning
    if (
      !tokenResponse.access_token ||
      typeof tokenResponse.access_token !== "string" ||
      tokenResponse.access_token.trim() === ""
    ) {
      console.warn("Invalid token received from Zitadel");
      throw createError({
        statusCode: 500,
        message: "Invalid token received from authentication provider",
      });
    }

    // Update the auth cookie with new access token
    // Always use long expiry to remember the user (unless they explicitly logout)
    const { AUTH_COOKIE_NAME } = await import("../../utils/auth");
    const expirySeconds = tokenResponse.expires_in || 3600;
    // Use refresh token expiry (typically 30 days) or 7 days, whichever is longer
    // This ensures the user stays logged in unless they explicitly logout
    const maxAge = Math.max(expirySeconds * 7, 7 * 24 * 60 * 60); // At least 7 days

    setCookie(event, AUTH_COOKIE_NAME, tokenResponse.access_token, {
      httpOnly: false,
      path: "/",
      maxAge,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      domain: undefined,
    });

    // Return the new access token
    return {
      accessToken: tokenResponse.access_token,
      expiresIn: tokenResponse.expires_in,
    };
  } catch (error) {
    // Handle errors
    const zitadelError = handleZitadelError(error);

    // If refresh token is invalid or expired, clear session
    if (
      zitadelError.error === "invalid_grant" ||
      zitadelError.error === "invalid_token"
    ) {
      await clearUserSession(event);
      throw createError({
        statusCode: 401,
        message: "Session expired. Please log in again.",
      });
    }

    // For other errors
    throw createError({
      statusCode: 500,
      message: `Token refresh failed: ${zitadelError.error_description}`,
    });
  }
});
