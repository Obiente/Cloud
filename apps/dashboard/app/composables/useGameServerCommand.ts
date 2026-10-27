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
  const isConnected = ref(false);
  const isLoading = ref(false);
  const error = ref<string | null>(null);
  let websocket: WebSocket | null = null;
  let connectionPromise: Promise<void> | null = null;
  let cancelConnection: (() => void) | null = null;

  const connect = async (): Promise<void> => {
    if (connectionPromise) return connectionPromise;
    if (isConnected.value && websocket?.readyState === WebSocket.OPEN) return;
    isLoading.value = true;
    error.value = null;
    const pending = new Promise<void>((resolve, reject) => {
      let settled = false;
      let socket: WebSocket | null = null;
      const fail = (reason: unknown) => {
        if (settled) return;
        settled = true;
        clearTimeout(deadline);
        const failure =
          reason instanceof Error ? reason : new Error(String(reason));
        if (websocket === socket) {
          error.value = failure.message;
          isLoading.value = false;
          isConnected.value = false;
          websocket = null;
        }
        socket?.close();
        reject(failure);
      };
      const deadline = setTimeout(
        () =>
          fail(
            new Error("Game server connection timed out. Please try again.")
          ),
        10000
      );
      cancelConnection = () =>
        fail(new Error("Game server connection was closed."));
      try {
        const url = new URL(
          "/gameservers/terminal/ws",
          config.public.apiHost || config.public.requestHost
        );
        url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
        socket = new WebSocket(url.toString());
        websocket = socket;
        const current = socket;
        current.onopen = async () => {
          try {
            const started = Date.now();
            while (
              !auth.ready &&
              !settled &&
              websocket === current &&
              Date.now() - started < 4000
            ) {
              await new Promise<void>((resolve) => setTimeout(resolve, 100));
            }
            if (settled || websocket !== current) return;
            if (!auth.ready)
              throw new Error("Authentication is not ready. Please try again.");
            let token = await auth.getAccessToken();
            if (
              settled ||
              websocket !== current ||
              current.readyState !== WebSocket.OPEN
            )
              return;
            if (!token && appConfig.disableAuth.value)
              token = "dev-dummy-token";
            if (!token)
              throw new Error("Authentication required. Please log in.");
            current.send(
              JSON.stringify({
                type: "init",
                gameServerId: options.gameServerId,
                organizationId:
                  options.organizationId || orgsStore.currentOrgId || "",
                token,
                cols: 80,
                rows: 24,
              })
            );
          } catch (reason) {
            fail(reason);
          }
        };
        current.onmessage = (event) => {
          if (websocket !== current) return;
          try {
            const message = JSON.parse(event.data);
            if (message.type === "connected" && !settled) {
              settled = true;
              clearTimeout(deadline);
              cancelConnection = null;
              isConnected.value = true;
              isLoading.value = false;
              error.value = null;
              resolve();
            } else if (message.type === "error") {
              if (!settled)
                fail(new Error(message.message || "Connection failed"));
              else {
                error.value = message.message || "Connection failed";
                current.close();
              }
            }
          } catch {
            fail(new Error("Invalid game server connection response."));
          }
        };
        current.onerror = () => {
          if (!settled) fail(new Error("Failed to connect to game server."));
          else if (websocket === current) {
            error.value = "Game server connection failed.";
            current.close();
          }
        };
        current.onclose = () => {
          if (!settled)
            fail(
              new Error("Game server connection closed before it was ready.")
            );
          if (websocket === current) {
            websocket = null;
            isConnected.value = false;
            isLoading.value = false;
          }
        };
      } catch (reason) {
        fail(reason);
      }
    });
    connectionPromise = pending;
    try {
      await pending;
    } finally {
      if (connectionPromise === pending) {
        connectionPromise = null;
        cancelConnection = null;
      }
    }
  };
  const sendCommand = async (command: string): Promise<void> => {
    if (!isConnected.value || websocket?.readyState !== WebSocket.OPEN)
      await connect();
    if (!websocket || websocket.readyState !== WebSocket.OPEN)
      throw new Error("Not connected to game server");
    websocket.send(
      JSON.stringify({
        type: "input",
        input: Array.from(new TextEncoder().encode(command.trim() + "\n")),
      })
    );
  };
  const disconnect = () => {
    cancelConnection?.();
    cancelConnection = null;
    websocket?.close();
    websocket = null;
    connectionPromise = null;
    isConnected.value = false;
    isLoading.value = false;
  };
  onUnmounted(disconnect);
  return { isConnected, isLoading, error, connect, sendCommand, disconnect };
}
