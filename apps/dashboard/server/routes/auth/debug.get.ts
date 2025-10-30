import { eventHandler } from "h3";
import { getServerToken } from "../../utils/serverAuth";

/**
 * Debug endpoint to check authentication state
 * This is useful for debugging auth issues
 * NOTE: Remove this in production!
 */
export default eventHandler(async (event) => {
  const session = await getUserSession(event);
  const directToken = getServerToken(event);

  // Don't return the actual tokens in production!
  return {
    hasSession: !!session,
    hasUser: !!session?.user,
    hasSecureData: !!session?.secure,
    hasSessionToken: !!session?.secure?.access_token,
    hasDirectToken: !!directToken,
    cookiePresent: !!getCookie(event, "obiente_auth"),
    tokenLength: directToken ? directToken.length : 0,
    // Only return token details in development
    tokenDetails:
      process.env.NODE_ENV !== "production"
        ? {
            sessionTokenPrefix: session?.secure?.access_token
              ? session.secure.access_token.substring(0, 10) + "..."
              : null,
            directTokenPrefix: directToken
              ? directToken.substring(0, 10) + "..."
              : null,
          }
        : undefined,
  };
});
