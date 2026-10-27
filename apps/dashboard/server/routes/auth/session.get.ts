import { eventHandler } from "h3";
import { readCurrentSession } from "../../utils/current-session";

export default eventHandler((event) => readCurrentSession(event));
