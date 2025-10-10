import { eventHandler } from "h3";
import { clearUserSession } from "../../utils/session";

export default eventHandler(async (event) => {
  return await clearUserSession(event);
});
