import { eventHandler } from "h3";
import { remainingTokenLifetime } from "../../utils/token-lifetime";

/**
 * API endpoint to get the current access token from session
 * This is needed for client-side access to the token
 */
export default eventHandler(async (event) => {
  if (process.env.DISABLE_AUTH === "true") {
    return {
      accessToken: "dev-dummy-token",
      expiresIn: 3600,
    };
  }

  const session = await getUserSession(event);

  if (!session?.secure?.access_token) {
    return { accessToken: null };
  }

  const token = session.secure.access_token;
  // Validate that the token is a non-empty string
  if (typeof token !== "string" || token.trim() === "") {
    console.warn("Invalid token found in session");
    return { accessToken: null };
  }

  // Get expiry information from session
  const expiresIn = remainingTokenLifetime(session.secure);
  if (expiresIn <= 0) return { accessToken: null, expiresIn: 0 };

  return {
    accessToken: token,
    expiresIn,
  };
});
