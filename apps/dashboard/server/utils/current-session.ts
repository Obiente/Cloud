import type { H3Event } from "@nuxt/nitro-server/h3";
import { setHeader } from "@nuxt/nitro-server/h3";
import type { UserSession } from "@obiente/types";
import { getUserData } from "./auth";
import { getUserSession } from "./session";
import {
  captureAuthRuntimeConfig,
  type AuthRuntimeConfig,
} from "./auth-runtime-config";

// Use the page's response event during SSR so renewal cookies reach the browser
// and subsequent server RPCs see the same updated session.
export async function readCurrentSession(
  event: H3Event,
  signal?: AbortSignal,
  config?: AuthRuntimeConfig
): Promise<UserSession> {
  if (config) captureAuthRuntimeConfig(event, config);
  setHeader(event, "Cache-Control", "private, no-store");
  if (process.env.DISABLE_AUTH === "true") {
    return {
      id: "development-session",
      user: {
        sub: "mem-development",
        id: "mem-development",
        email: "dev@obiente.local",
        name: "Development User",
        given_name: "Development",
        family_name: "User",
        preferred_username: "dev",
        email_verified: true,
        locale: "en",
        updated_at: 0,
      } as UserSession["user"] & { id: string },
    };
  }
  const session = await getUserSession(event);
  try {
    await getUserData(event, session, signal);
  } catch (error) {
    // A signed, existing session is still known during a provider outage.
    // Resource authorization remains the API's responsibility.
    const status = (error as { statusCode?: number })?.statusCode;
    if (!session.user || (status !== 502 && status !== 503)) throw error;
  }
  const current = await getUserSession(event);
  return { id: current.id, user: current.user };
}
