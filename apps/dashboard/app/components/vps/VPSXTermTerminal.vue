<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiText as="h3" size="md" weight="semibold"> VPS Terminal </OuiText>
      <OuiFlex gap="sm">
        <OuiButton
          v-if="terminal"
          variant="ghost"
          size="sm"
          @click="clearTerminal"
          :disabled="!terminal"
        >
          Clear
        </OuiButton>
        <OuiButton
          v-if="isConnected"
          variant="ghost"
          size="sm"
          color="danger"
          @click="disconnectTerminal"
        >
          Disconnect
        </OuiButton>
        <OuiButton
          v-else
          variant="ghost"
          size="sm"
          @click="connectTerminal"
          :disabled="isLoading"
        >
          {{ isLoading ? "Connecting..." : "Connect" }}
        </OuiButton>
      </OuiFlex>
    </OuiFlex>

    <OuiText size="sm" color="secondary">
      Access an interactive terminal session to run commands directly on your
      VPS.
    </OuiText>

    <div class="terminal-wrapper">
      <div class="terminal-container">
        <div ref="terminalContainer" class="terminal-content" />
        <div v-if="showSpinner" class="terminal-overlay">
          <div class="terminal-spinner" aria-hidden="true"></div>
          <OuiText size="sm" color="secondary" class="spinner-text">
            Connecting to terminal...
          </OuiText>
        </div>
      </div>
    </div>

    <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>

    <OuiText v-if="isConnected && !error" size="xs" color="success">
      ✓ Terminal connected. Type commands to interact with your VPS.
    </OuiText>
  </OuiStack>
</template>

<script setup lang="ts">
  import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
  import { useAuth } from "~/composables/useAuth";
  import { useOrganizationsStore } from "~/stores/organizations";

  interface Props {
    vpsId: string;
    organizationId?: string;
  }

  const props = defineProps<Props>();
  const orgsStore = useOrganizationsStore();
  const organizationId = computed(
    () => props.organizationId || orgsStore.currentOrgId || ""
  );
  const auth = useAuth();

  const terminalContainer = ref<HTMLElement | null>(null);
  const isConnected = ref(false);
  const isLoading = ref(false);
  const error = ref("");

  let terminal: any = null;
  let fitAddon: any = null;
  let websocket: WebSocket | null = null;
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  let reconnectAttempts = 0;
  let Terminal: any = null;
  let FitAddon: any = null;

  const showSpinner = computed(() => isLoading.value && !isConnected.value);
  const shouldAttemptReconnect = ref(true);

  async function initTerminal() {
    if (typeof window === "undefined" || !terminalContainer.value) return;

    if (!Terminal || !FitAddon) {
      try {
        import("@xterm/xterm/css/xterm.css").catch(() => {});
        const [xtermModule, xtermAddonModule] = await Promise.all([
          import("@xterm/xterm"),
          import("@xterm/addon-fit"),
        ]);
        Terminal = xtermModule.Terminal;
        FitAddon = xtermAddonModule.FitAddon;
      } catch (err) {
        console.error("Failed to load xterm:", err);
        error.value = "Failed to load terminal. Please refresh the page.";
        return;
      }
    }

    if (!Terminal || !FitAddon) return;

    async function getOUITerminalTheme() {
      if (typeof window === "undefined") {
        return {
          background: "var(--oui-surface-base)",
          foreground: "var(--oui-text-primary)",
          cursor: "var(--oui-accent-primary)",
          selection: "#10b98140",
        };
      }

      let chroma: any = null;
      try {
        const chromaModule = await import("chroma-js");
        chroma = (chromaModule as any).default || chromaModule;
      } catch (err) {
        console.warn("Failed to load chroma-js, using fallback colors:", err);
      }

      const root = document.documentElement;
      const getStyle = (prop: string) =>
        getComputedStyle(root).getPropertyValue(prop).trim() || "";

      const lighten = (color: string, amount = 0.3): string => {
        if (!chroma) return color;
        try {
          return chroma(color).brighten(amount).saturate(0.2).hex();
        } catch {
          return color;
        }
      };

      const background = getStyle("--oui-surface-base") || "#111a16";
      const foreground = getStyle("--oui-text-primary") || "#e9fff8";
      const accentPrimary = getStyle("--oui-accent-primary") || "#10b981";
      const accentDanger = getStyle("--oui-accent-danger") || "#f43f5e";
      const accentSuccess = getStyle("--oui-accent-success") || "#22c55e";
      const accentWarning = getStyle("--oui-accent-warning") || "#fbbf24";
      const accentInfo = getStyle("--oui-accent-info") || "#38bdf8";
      const accentSecondary = getStyle("--oui-accent-secondary") || "#67e8f9";
      const textTertiary = getStyle("--oui-text-tertiary") || "#5ce1a6";
      const ouiBackground = getStyle("--oui-background") || "#0a0f0c";

      return {
        background,
        foreground,
        cursor: accentPrimary,
        selection: accentPrimary + "40",
        cursorAccent: ouiBackground,
        black: ouiBackground,
        red: accentDanger,
        green: accentSuccess,
        yellow: accentWarning,
        blue: accentInfo,
        magenta: accentPrimary,
        cyan: accentSecondary,
        white: foreground,
        brightBlack: lighten(textTertiary, 0.4),
        brightRed: lighten(accentDanger, 0.4),
        brightGreen: lighten(accentSuccess, 0.3),
        brightYellow: lighten(accentWarning, 0.3),
        brightBlue: lighten(accentInfo, 0.3),
        brightMagenta: lighten(accentPrimary, 0.3),
        brightCyan: lighten(accentSecondary, 0.3),
        brightWhite: lighten(foreground, 0.2),
      };
    }

    const terminalTheme = await getOUITerminalTheme();

    terminal = new Terminal({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: terminalTheme,
      allowProposedApi: true,
      disableStdin: false,
    });

    if (terminal.options) {
      terminal.options.theme = terminalTheme;
    }

    fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(terminalContainer.value);

    await nextTick();
    fitAddon.fit();
    terminal.focus();
    terminal.options.theme = terminalTheme;
    terminal.refresh(0, terminal.rows - 1);

    const resizeObserver = new ResizeObserver(async () => {
      if (!fitAddon || !terminal) return;
      fitAddon.fit();
      if (
        isConnected.value &&
        websocket &&
        websocket.readyState === WebSocket.OPEN
      ) {
        try {
          websocket.send(
            JSON.stringify({
              type: "resize",
              cols: terminal.cols || 80,
              rows: terminal.rows || 24,
            })
          );
        } catch (err) {
          console.error("Failed to send resize:", err);
        }
      }
    });

    if (terminalContainer.value) {
      resizeObserver.observe(terminalContainer.value);
    }

    terminal.onData((data: string) => {
      if (!websocket || websocket.readyState !== WebSocket.OPEN) {
        return;
      }

      // Send input as JSON message (backend expects this format)
      const encoder = new TextEncoder();
      const input = encoder.encode(data);
      try {
        websocket.send(
          JSON.stringify({
            type: "input",
            input: Array.from(input),
            cols: terminal?.cols || 80,
            rows: terminal?.rows || 24,
          })
        );
      } catch (err) {
        console.error("Failed to send terminal input:", err);
      }
    });
  }

  const connectTerminal = async () => {
    if (isConnected.value) {
      return;
    }

    shouldAttemptReconnect.value = true;
    isLoading.value = true;
    error.value = "";

    if (!terminal) {
      await initTerminal();
    }

    if (!terminal) {
      error.value = "Failed to initialize terminal";
      isLoading.value = false;
      return;
    }

    try {
      if (!terminal) {
        throw new Error("Terminal not initialized");
      }

      const config = useRuntimeConfig();
      const appConfig = useConfig();
      const apiBase = config.public.apiHost || config.public.requestHost;
      const disableAuth = Boolean(appConfig.disableAuth.value);
      const wsUrlObject = new URL(`/vps/${props.vpsId}/terminal/ws`, apiBase);
      wsUrlObject.protocol = wsUrlObject.protocol === "https:" ? "wss:" : "ws:";
      const wsUrl = wsUrlObject.toString();

      websocket = new WebSocket(wsUrl);

      websocket.onopen = async () => {
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
          error.value = "Authentication required. Please log in.";
          isLoading.value = false;
          websocket?.close();
          return;
        }

        const initMessage: any = {
          type: "init",
          vpsId: props.vpsId,
          organizationId: organizationId.value,
          token,
          cols: terminal.cols || 80,
          rows: terminal.rows || 24,
        };

        websocket!.send(JSON.stringify(initMessage));
      };

      websocket.onmessage = async (event: MessageEvent) => {
        try {
          // Handle binary messages (preferred for terminal output)
          if (event.data instanceof ArrayBuffer) {
            if (terminal) {
              const data = new Uint8Array(event.data);
              terminal.write(data);
            }
            return;
          }

          if (event.data instanceof Blob) {
            if (terminal) {
              const arrayBuffer = await event.data.arrayBuffer();
              const data = new Uint8Array(arrayBuffer);
              terminal.write(data);
            }
            return;
          }

          // Handle JSON messages (for control messages)
          const message =
            typeof event.data === "string"
              ? JSON.parse(event.data)
              : event.data;

          if (message.type === "connected") {
            isConnected.value = true;
            isLoading.value = false;
            reconnectAttempts = 0;
            shouldAttemptReconnect.value = true;
            terminal?.focus();
          } else if (message.type === "output" && terminal) {
            // Fallback: handle JSON output format (for guest agent)
            if (
              message.data &&
              Array.isArray(message.data) &&
              message.data.length > 0
            ) {
              const output = new Uint8Array(message.data);
              terminal.write(output);
            }
          } else if (message.type === "closed") {
            disconnectTerminal();
          } else if (message.type === "error") {
            error.value = message.message || "Terminal error";
            isLoading.value = false;
            if (terminal) {
              terminal.write(
                `\r\n\x1b[31m[ERROR]\x1b[0m ${message.message}\r\n`
              );
            }
            if (message.message.includes("Authentication required")) {
              disconnectTerminal();
            }
          }
        } catch (err) {
          console.error("Error processing WebSocket message:", err);
        }
      };

      websocket.onerror = (err) => {
        console.error("WebSocket error:", err);
        if (!isConnected.value) {
          error.value = "Failed to connect to terminal. Please try again.";
          isLoading.value = false;
        }
      };

      websocket.onclose = () => {
        const wasConnected = isConnected.value;
        isConnected.value = false;
        websocket = null;

        if (shouldAttemptReconnect.value) {
          if (wasConnected) {
            isLoading.value = true;
          }
          // Keep isLoading true if we're going to reconnect (it should already be true)
          scheduleReconnect();
        } else {
          isLoading.value = false;
        }
      };
    } catch (err: any) {
      console.error("Failed to connect terminal:", err);
      const errMsg =
        err.message || "Failed to connect terminal. Please try again.";
      error.value = errMsg;

      if (terminal) {
        terminal.write(`\r\n\x1b[31m[ERROR]\x1b[0m ${errMsg}\r\n`);
      }

      isConnected.value = false;
      isLoading.value = false;
      if (websocket) {
        websocket.close();
        websocket = null;
      }
    }
  };

  function scheduleReconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
    }

    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000);
    reconnectAttempts += 1;

    reconnectTimer = setTimeout(async () => {
      reconnectTimer = null;
      await connectTerminal();
    }, delay);
  }

  const disconnectTerminal = () => {
    shouldAttemptReconnect.value = false;
    isConnected.value = false;
    isLoading.value = false;

    if (reconnectTimer) {
      clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }

    if (websocket) {
      try {
        websocket.close();
      } catch {
        // ignore
      }
      websocket = null;
    }

    if (terminal) {
      terminal.write("\r\n\x1b[33m[Disconnected]\x1b[0m\r\n");
    }
  };

  const clearTerminal = () => {
    if (terminal) {
      terminal.clear();
    }
  };

  watch(
    () => props.vpsId,
    async () => {
      if (isConnected.value) {
        disconnectTerminal();
      }
      await nextTick();
      if (terminal) {
        await connectTerminal();
      }
    }
  );

  onMounted(async () => {
    await nextTick();
    await initTerminal();
    if (terminal) {
      await connectTerminal();
    }
  });

  onUnmounted(() => {
    disconnectTerminal();
    if (terminal) {
      terminal.dispose();
    }
    if (fitAddon) {
      fitAddon.dispose();
    }
  });
</script>

<style scoped>
  .terminal-wrapper {
    width: 100%;
    height: 600px;
    position: relative;
  }

  .terminal-container {
    width: 100%;
    height: 100%;
    background-color: var(--oui-surface-base, #111a16);
    border-radius: 0.75rem;
    border: 1px solid var(--oui-border-default, rgba(255, 255, 255, 0.1));
    box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.3),
      0 2px 4px -1px rgba(0, 0, 0, 0.2), 0 0 0 1px rgba(255, 255, 255, 0.05),
      inset 0 1px 0 0 rgba(255, 255, 255, 0.05);
    padding: 1rem;
    position: relative;
    overflow: hidden;
  }

  .terminal-container::before {
    content: "";
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 1px;
    background: linear-gradient(
      90deg,
      transparent,
      rgba(255, 255, 255, 0.1) 20%,
      rgba(255, 255, 255, 0.1) 80%,
      transparent
    );
    pointer-events: none;
    z-index: 1;
  }

  .terminal-content {
    width: 100%;
    height: 100%;
    position: relative;
    overflow: hidden;
    z-index: 1;
  }

  .terminal-overlay {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    background: color-mix(
      in srgb,
      var(--oui-surface-overlay, #201b3f) 25%,
      transparent
    );
    backdrop-filter: blur(8px) saturate(180%);
    pointer-events: none;
    z-index: 1000;
    border-radius: 0.75rem;
  }

  .terminal-spinner {
    width: 3rem;
    height: 3rem;
    border-radius: 50%;
    border: 4px solid
      color-mix(in srgb, var(--oui-text-primary, #f5f3ff) 20%, transparent);
    border-top-color: var(--oui-accent-primary, #a855f7);
    animation: terminal-spin 0.8s linear infinite;
    flex-shrink: 0;
  }

  .spinner-text {
    margin-top: 0.5rem;
    text-align: center;
  }

  @keyframes terminal-spin {
    to {
      transform: rotate(360deg);
    }
  }

  .terminal-content :deep(.xterm) {
    height: 100%;
    width: 100%;
    position: relative;
    z-index: 1;
  }

  .terminal-content :deep(.xterm-viewport) {
    background-color: transparent !important;
  }

  .terminal-container :deep(.xterm-viewport)::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  .terminal-container :deep(.xterm-viewport)::-webkit-scrollbar-track {
    background: rgba(0, 0, 0, 0.2);
    border-radius: 4px;
  }

  .terminal-container :deep(.xterm-viewport)::-webkit-scrollbar-thumb {
    background: rgba(255, 255, 255, 0.2);
    border-radius: 4px;
    transition: background 0.2s ease;
  }

  .terminal-container :deep(.xterm-viewport)::-webkit-scrollbar-thumb:hover {
    background: rgba(255, 255, 255, 0.3);
  }

  :deep(.xterm-cursor) {
    box-shadow: none;
  }
</style>
