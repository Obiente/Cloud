import type { ZitadelTokenResponse } from "@obiente/types";
import { handleZitadelError } from "../../utils/token";

/**
 * API endpoint to refresh the access token using the refresh token
 * This is used when the access token expires
 */
export default defineEventHandler(async (event) => {
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

    // Update session with new tokens
    await setUserSession(event, {
      secure: {
        scope: tokenResponse.scope,
        token_type: tokenResponse.token_type,
        expires_in: tokenResponse.expires_in,
        refresh_token: tokenResponse.refresh_token,
        access_token: tokenResponse.access_token,
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
