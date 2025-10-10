/**
 * Authentication middleware for Nuxt server routes
 */
export default defineEventHandler(async (event) => {
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

  try {
    const { payload } = await verifyAccessToken(accessToken);
    event.context.user = payload;
  } catch (error) {
    throw createError({
      statusCode: 401,
      message: "Invalid or expired token",
    });
  }
});
