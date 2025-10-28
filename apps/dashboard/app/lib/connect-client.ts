import { useNuxtApp } from "#app";
import { createClient, type Client } from "@connectrpc/connect";
import type { DescService } from "@bufbuild/protobuf";

/**
 * Creates a Connect RPC client for a given service.
 * This utility uses the transport provided by the connect-transport plugin,
 * which properly handles both SSR and client-side requests.
 *
 * @param service - The service definition from proto generation
 * @returns A configured client instance
 */
export function useConnectClient<T extends DescService>(service: T): Client<T> {
  const nuxtApp = useNuxtApp();
  const transport = nuxtApp.$connect;
  
  return createClient(service, transport);
}
