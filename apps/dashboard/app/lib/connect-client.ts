import { useNuxtApp } from "#app";
import { inject } from "vue";
import { createClient, type Client } from "@connectrpc/connect";
import type { DescService } from "@bufbuild/protobuf";

// Injection key for scoping preview transport to PreviewProviders subtree
export const PREVIEW_CONNECT_KEY: unique symbol = Symbol("OBIENTE_PREVIEW_CONNECT");

/**
 * Creates a Connect RPC client for a given service.
 * Uses live transport by default, but swaps to preview transport
 * when running inside PreviewProviders (which sets a global hook).
 *
 * @param service - The service definition from proto generation
 * @returns A configured client instance connected to live or preview transport
 */
export function useConnectClient<T extends DescService>(service: T): Client<T> {
  const nuxtApp = useNuxtApp();
  const injectedTransport = inject<any | null>(PREVIEW_CONNECT_KEY, null);
  const transport = injectedTransport || nuxtApp.$connect;
  return createClient(service, transport);
}
