import type { H3Event, SessionConfig } from "@nuxt/nitro-server/h3";

export interface AuthRuntimeConfig {
  session: Partial<SessionConfig> & { password: string };
  public: { oidcBase: string; oidcClientId: string };
}

// SSR helpers also run after asynchronous boundaries, where Nuxt's app context
// is no longer available. Keep the captured configuration on this request only.
export function captureAuthRuntimeConfig(
  event: H3Event,
  config: AuthRuntimeConfig
) {
  event.context.obienteAuthRuntimeConfig = config;
}

export function getAuthRuntimeConfig(event?: H3Event): AuthRuntimeConfig {
  return event?.context?.obienteAuthRuntimeConfig ?? useRuntimeConfig(event);
}
