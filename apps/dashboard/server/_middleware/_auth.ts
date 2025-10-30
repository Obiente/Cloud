/**
 * Authentication middleware for Nuxt server routes
 * Access tokens are bearer tokens, not JWTs, so validation is handled by the API
 */
export default defineEventHandler(async (event) => {
  const { getAccessToken } = await import("../utils/token");
  // Skip auth for public routes
  const publicRoutes = ["/auth/callback", "/auth/login"];
  if (publicRoutes.includes(event.path)) {
    return;
  }
  const accessToken = getAccessToken(event);
  if (!accessToken) {
    throw createError({
      statusCode: 401,
      message: "Not authenticated",
    });
  }

  // Access token is a bearer token, not a JWT, so we don't verify it here
  // The API backend will validate it when making requests
  // We just need to ensure it exists
});
