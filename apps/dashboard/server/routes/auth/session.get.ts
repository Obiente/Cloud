import { eventHandler } from "h3";
import { getUserData } from "../../utils/auth";

export default eventHandler(async (event) => {
  // If DISABLE_AUTH is enabled, return mock dev user
  if (process.env.DISABLE_AUTH === "true") {
    console.log("⚠️  WARNING: DISABLE_AUTH=true, returning mock development user");
    return {
      user: {
        id: "mem-development",
        email: "dev@obiente.local",
        name: "Development User",
        given_name: "Development",
        family_name: "User",
        preferred_username: "dev",
        email_verified: true,
        locale: "en",
        picture: "",
      },
    };
  }

  const session = await getUserSession(event);
  // Populate user data if session exists
  if (Object.keys(session).length > 0) {
    try {
      // getUserData will refresh the token if needed and update the session
      await getUserData(event, session);
      // Re-fetch session after getUserData (it may have updated it with refreshed token)
      const updatedSession = await getUserSession(event);
      const { secure, ...data } = updatedSession;
      return data;
    } catch (error) {
      // If getUserData fails (e.g., token expired and refresh failed), return empty session
      console.error("[session.get] Failed to get user data:", error);
      // Return empty session so client knows to redirect to login
      return { user: null };
    }
  }
  // Exclude secure (server-only) data from response
  const { secure, ...data } = session;
  return data;
});
