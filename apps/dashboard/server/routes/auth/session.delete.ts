import { eventHandler, setCookie } from "h3";
import { clearUserSession } from "../../utils/session";
import { AUTH_COOKIE_NAME, REFRESH_COOKIE_NAME } from "../../utils/auth";

export default eventHandler(async (event) => {
  // Clear the session
  const result = await clearUserSession(event);

  // Also explicitly clear the direct auth cookies
  setCookie(event, AUTH_COOKIE_NAME, "", {
    httpOnly: false,
    path: "/",
    maxAge: 0,
    expires: new Date(0),
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
  });

  setCookie(event, REFRESH_COOKIE_NAME, "", {
    httpOnly: true,
    path: "/",
    maxAge: 0,
    expires: new Date(0),
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
  });

  return result;
});
