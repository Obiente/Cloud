import { createGrpcWebTransport } from "@connectrpc/connect-web";

/**
 * Pure factory for creating a connect transport instance.
 * Keep this file under `app/` so it can be imported in both
 * client plugins and server code without pulling in Nuxt server-only helpers.
 */
export function createTransport(baseUrl: string) {
  return createGrpcWebTransport({
    baseUrl,
  });
}

export default createTransport;
