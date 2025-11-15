import { ref, onUnmounted } from "vue";
import { useAuth } from "~/composables/useAuth";
import { useOrganizationsStore } from "~/stores/organizations";
import { useConfig } from "~/composables/useConfig";

interface UseGameServerCommandOptions {
  gameServerId: string;
  organizationId?: string;
}

export function useGameServerCommand(options: UseGameServerCommandOptions) {
  const auth = useAuth();
  const orgsStore = useOrganizationsStore();
  const config = useRuntimeConfig();
  const appConfig = useConfig();
  
  const organizationId = options.organizationId || orgsStore.currentOrgId || "";
  const isConnected = ref(false);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  
  let websocket: WebSocket | null = null;
  let connectionPromise: Promise<void> | null = null;
  let isConnecting = false;

  const connect = async (): Promise<void> => {
    // Return existing connection promise if already connecting
    if (connectionPromise) {
      return connectionPromise;
    }

    // If already connected, return immediately
    if (isConnected.value && websocket && websocket.readyState === WebSocket.OPEN) {
      return Promise.resolve();
    }

    connectionPromise = new Promise(async (resolve, reject) => {
      try {
        isConnecting = true;
        isLoading.value = true;
        error.value = null;

        const apiBase = config.public.apiHost || config.public.requestHost;
        const disableAuth = Boolean(appConfig.disableAuth.value);
        const wsUrlObject = new URL("/gameservers/terminal/ws", apiBase);
        wsUrlObject.protocol = wsUrlObject.protocol === "https:" ? "wss:" : "ws:";
        const wsUrl = wsUrlObject.toString();

        websocket = new WebSocket(wsUrl);

        websocket.onopen = async () => {
          try {
            if (!auth.ready) {
              await new Promise((resolve) => {
                const checkReady = () => {
                  if (auth.ready) {
                    resolve(undefined);
                  } else {
                    setTimeout(checkReady, 100);
                  }
                };
                checkReady();
              });
            }

            let token = await auth.getAccessToken();
            if (!token && disableAuth) {
              token = "dev-dummy-token";
            }

            if (!token) {
              const err = "Authentication required. Please log in.";
              error.value = err;
              isLoading.value = false;
              websocket?.close();
              reject(new Error(err));
              return;
            }

            const initMessage: any = {
              type: "init",
              gameServerId: options.gameServerId,
              organizationId: organizationId,
              token,
              cols: 80,
              rows: 24,
            };

            websocket!.send(JSON.stringify(initMessage));
          } catch (err: any) {
            error.value = err.message || "Failed to initialize connection";
            isLoading.value = false;
            websocket?.close();
            reject(err);
          }
        };

        websocket.onmessage = (event: MessageEvent) => {
          try {
            const message = JSON.parse(event.data);

            if (message.type === "connected") {
              isConnected.value = true;
              isLoading.value = false;
              error.value = null;
              isConnecting = false;
              connectionPromise = null;
              resolve();
            } else if (message.type === "error") {
              const errMsg = message.message || "Connection error";
              error.value = errMsg;
              isConnected.value = false;
              isLoading.value = false;
              isConnecting = false;
              connectionPromise = null;
              reject(new Error(errMsg));
            }
          } catch (err: any) {
            console.error("Error processing WebSocket message:", err);
          }
        };

        websocket.onerror = (err) => {
          console.error("WebSocket error:", err);
          const errMsg = "Failed to connect to game server";
          error.value = errMsg;
          isLoading.value = false;
          isConnecting = false;
          connectionPromise = null;
          reject(new Error(errMsg));
        };

        websocket.onclose = () => {
          isConnected.value = false;
          websocket = null;
          isLoading.value = false;
          isConnecting = false;
          connectionPromise = null;
        };
      } catch (err: any) {
        error.value = err.message || "Failed to connect";
        isLoading.value = false;
        isConnecting = false;
        connectionPromise = null;
        reject(err);
      }
    });

    return connectionPromise;
  };

  const sendCommand = async (command: string): Promise<void> => {
    // Ensure we're connected
    if (!isConnected.value || !websocket || websocket.readyState !== WebSocket.OPEN) {
      await connect();
    }

    if (!websocket || websocket.readyState !== WebSocket.OPEN) {
      throw new Error("Not connected to game server");
    }

    // Send command to game server stdin
    const encoder = new TextEncoder();
    const input = encoder.encode(command.trim() + "\n");
    
    try {
      websocket.send(
        JSON.stringify({
          type: "input",
          input: Array.from(input),
        })
      );
    } catch (err: any) {
      console.error("Failed to send command:", err);
      throw new Error(`Failed to send command: ${err.message || "Unknown error"}`);
    }
  };

  const disconnect = () => {
    if (websocket) {
      try {
        websocket.close();
      } catch {
        // ignore
      }
      websocket = null;
    }
    isConnected.value = false;
    isLoading.value = false;
    connectionPromise = null;
    isConnecting = false;
  };

  // Cleanup on unmount
  onUnmounted(() => {
    disconnect();
  });

  return {
    isConnected,
    isLoading,
    error,
    connect,
    sendCommand,
    disconnect,
  };
}

