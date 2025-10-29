import { eventHandler } from "h3";

/**
 * API endpoint to get the current access token from session
 * This is needed for client-side access to the token
 */
export default eventHandler(async (event) => {
  const session = await getUserSession(event);

  if (!session || !session.secure?.access_token) {
    return { accessToken: null };
  }

  const token = session.secure.access_token;
  // Validate that the token is a non-empty string
  if (typeof token !== "string" || token.trim() === "") {
    console.warn("Invalid token found in session");
    return { accessToken: null };
  }

  // Get expiry information from session
  const expiresIn = session.secure?.expires_in || 3600; // Default to 1 hour

  return {
    accessToken: token,
    expiresIn,
  };
});
