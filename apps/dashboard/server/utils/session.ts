import type { H3Event, SessionConfig } from "h3";
import { useSession, createError, isEvent } from "h3";
import { defu } from "defu";
import type { UserSession } from "@obiente/types";

type UseSessionEvent = Parameters<typeof useSession>[0];

/**
 * Get the user session from the current request
 * @param event The Request (h3) event
 * @returns The user session
 */
export async function getUserSession(
  event: UseSessionEvent
): Promise<UserSession> {
  const session = await _useSession(event);
  return {
    ...session.data,
    id: session.id!,
  };
}
/**
 * Set a user session
 * @param event The Request (h3) event
 * @param data User session data, please only store public information since it can be decoded with API calls
 * @see https://github.com/atinux/nuxt-auth-utils
 */
export async function setUserSession(
  event: H3Event,
  data: Omit<UserSession, "id">,
  config?: Partial<SessionConfig>
): Promise<UserSession> {
  const session = await _useSession(event, config);
  await session.update(defu(data, session.data as Omit<UserSession, "id">));

  return session.data;
}

/**
 * Replace a user session
 * @param event The Request (h3) event
 * @param data User session data, please only store public information since it can be decoded with API calls
 */
export async function replaceUserSession(
  event: H3Event,
  data: Omit<UserSession, "id">,
  config?: Partial<SessionConfig>
): Promise<UserSession> {
  const session = await _useSession(event, config);

  await session.clear();
  await session.update(data);

  return session.data;
}

/**
 * Clear the user session and removing the session cookie
 * @param event The Request (h3) event
 * @returns true if the session was cleared
 */
export async function clearUserSession(
  event: H3Event,
  config?: Partial<SessionConfig>
): Promise<boolean> {
  const session = await _useSession(event, config);

  await session.clear();

  return true;
}

/**
 * Require a user session, throw a 401 error if the user is not logged in
 * @param event
 * @param opts Options to customize the error message and status code
 * @param opts.statusCode The status code to use for the error (defaults to 401)
 * @param opts.message The message to use for the error (defaults to Unauthorized)
 * @see https://github.com/atinux/nuxt-auth-utils
 */
export async function requireUserSession(
  event: UseSessionEvent,
  opts: { statusCode?: number; message?: string } = {}
): Promise<UserSession> {
  const userSession = await getUserSession(event);

  if (!userSession.user) {
    if (isEvent(event)) {
      throw createError({
        statusCode: opts.statusCode || 401,
        message: opts.message || "Unauthorized",
      });
    } else {
      throw new Response(opts.message || "Unauthorized", {
        status: opts.statusCode || 401,
      });
    }
  }

  return userSession;
}

let sessionConfig: SessionConfig;

export function _useSession<T extends Record<string, any> = UserSession>(
  event: UseSessionEvent,
  config: Partial<SessionConfig> = {}
) {
  if (!sessionConfig) {
    const runtimeConfig = useRuntimeConfig(isEvent(event) ? event : undefined);
    sessionConfig = defu(
      { password: runtimeConfig.session.password },
      runtimeConfig.session
    );
    if (!sessionConfig.password) {
      console.error(`[obiente-auth] NUXT_SESSION_PASSWORD was not set.`);
    }
    if (sessionConfig.password.startsWith("changeme_")) {
      console.warn(
        `[obiente-auth] NUXT_SESSION_PASSWORD was set to the default value.`
      );
    }
  }
  const finalConfig = defu(config, sessionConfig) as SessionConfig;
  return useSession<T>(event, finalConfig);
}
