import { refreshUserSession } from "../../utils/auth";
import { remainingTokenLifetime } from "../../utils/token-lifetime";

export default defineEventHandler(async (event) => {
  if (process.env.DISABLE_AUTH === "true")
    return { accessToken: "dev-dummy-token", expiresIn: 3600 };
  const session = await refreshUserSession(event, await getUserSession(event));
  return {
    accessToken: session.secure!.access_token,
    expiresIn: remainingTokenLifetime(session.secure),
  };
});
