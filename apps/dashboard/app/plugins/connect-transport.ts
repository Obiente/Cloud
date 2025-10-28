import { defineNuxtPlugin, useRuntimeConfig } from "#imports";
import { createTransport } from "~/lib/transport";

// This plugin runs on the client and creates a transport instance using
// the public runtime config. It then injects it as `$transport` on the Nuxt app.
export default defineNuxtPlugin((nuxtApp) => {
  const config = useRuntimeConfig();
  return {
    provide: {
      connect: createTransport(
        config.public.requestHost + config.public.apiBaseUrl
      ),
    },
  };
});
