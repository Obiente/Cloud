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
    await getUserData(event, session);
  }
  // Exclude secure (server-only) data from response
  const { secure, ...data } = session;
  return data;
});
