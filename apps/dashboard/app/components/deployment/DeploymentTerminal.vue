<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">Interactive Terminal</OuiText>
        <OuiFlex gap="sm">
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
        Access an interactive terminal session to run commands directly in your container.
      </OuiText>

      <div
        ref="terminalContainer"
        class="w-full h-96 rounded-lg border border-border-default overflow-hidden"
        style="background-color: var(--oui-surface-base);"
      />

      <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>

      <OuiText v-if="isConnected && !error" size="xs" color="success">
        ✓ Terminal connected. Type commands to interact with your container.
      </OuiText>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";

interface Props {
  deploymentId: string;
  organizationId?: string;
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => props.organizationId || orgsStore.currentOrgId || "");

const client = useConnectClient(DeploymentService);
const terminalContainer = ref<HTMLElement | null>(null);
const isConnected = ref(false);
const isLoading = ref(false);
const error = ref("");

let terminal: any = null;
let fitAddon: any = null;
let terminalStream: any = null;
let reconnectAttempts = 0;
let Terminal: any = null;
let FitAddon: any = null;
let streamController: AbortController | null = null;

const initTerminal = async () => {
  // Only run on client side
  if (typeof window === 'undefined' || !terminalContainer.value) return;

  // Lazy load xterm only on client side
  // Using dynamic imports so Nuxt can code-split and optimize the bundle
  if (!Terminal || !FitAddon) {
    try {
      // Load CSS first (fire and forget to avoid blocking)
      import("@xterm/xterm/css/xterm.css").catch(() => {
        // CSS import failed silently, terminal will still work
      });
      
      // Parallel imports for better performance - Nuxt will optimize these as separate chunks
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

  // Get OUI theme colors with chroma.js for better color manipulation
  async function getOUITerminalTheme() {
    if (typeof window === "undefined") {
      return {
        background: "var(--oui-surface-base)",
        foreground: "var(--oui-text-primary)",
        cursor: "var(--oui-accent-primary)",
        selection: "#10b98140",
      };
    }

    // Lazy load chroma-js for color manipulation
    let chroma: any = null;
    try {
      const chromaModule = await import("chroma-js");
      // chroma-js exports the default function directly or as default
      chroma = (chromaModule as any).default || chromaModule;
    } catch (err) {
      console.warn("Failed to load chroma-js, using fallback colors:", err);
    }

    const root = document.documentElement;
    const getStyle = (prop: string) => {
      const value = getComputedStyle(root).getPropertyValue(prop).trim();
      return value || "";
    };

    // Helper function to adjust lightness and saturation for bright colors
    const lighten = (color: string, amount: number = 0.3): string => {
      if (!chroma) {
        return color;
      }
      try {
        return chroma(color).brighten(amount).saturate(0.2).hex();
      } catch {
        return color;
      }
    };

    // Get all colors
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

    // Create bright versions of colors using chroma.js
    const brightRed = lighten(accentDanger, 0.4);
    const brightGreen = lighten(accentSuccess, 0.3);
    const brightYellow = lighten(accentWarning, 0.3);
    const brightBlue = lighten(accentInfo, 0.3);
    const brightMagenta = lighten(accentPrimary, 0.3);
    const brightCyan = lighten(accentSecondary, 0.3);
    const brightWhite = lighten(foreground, 0.2);
    const brightBlack = lighten(textTertiary, 0.4);

    return {
      background,
      foreground,
      cursor: accentPrimary,
      selection: accentPrimary + "40",
      cursorAccent: ouiBackground,
      // Standard ANSI colors
      black: ouiBackground,
      red: accentDanger,
      green: accentSuccess,
      yellow: accentWarning,
      blue: accentInfo,
      magenta: accentPrimary,
      cyan: accentSecondary,
      white: foreground,
      // Bright ANSI colors - actually brighter using chroma.js
      brightBlack: brightBlack,
      brightRed: brightRed,
      brightGreen: brightGreen,
      brightYellow: brightYellow,
      brightBlue: brightBlue,
      brightMagenta: brightMagenta,
      brightCyan: brightCyan,
      brightWhite: brightWhite,
    };
  }

  const terminalTheme = await getOUITerminalTheme();

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: terminalTheme,
  });

  // Ensure theme is applied after terminal creation (xterm.js sometimes needs this)
  if (terminal.options) {
    terminal.options.theme = terminalTheme;
  }

  fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);

  terminal.open(terminalContainer.value);
  fitAddon.fit();

  // Apply theme after terminal is opened (some versions of xterm.js require this)
  terminal.options.theme = terminalTheme;
  
  // Force a refresh to ensure theme is applied
  terminal.refresh(0, terminal.rows - 1);

  // Handle terminal resize
  const resizeObserver = new ResizeObserver(async () => {
    if (fitAddon && terminal) {
      fitAddon.fit();
      if (isConnected.value) {
        // Send resize info via unary RPC
        try {
          await client.sendTerminalInput({
            organizationId: organizationId.value,
            deploymentId: props.deploymentId,
            input: new Uint8Array(0), // No input, just resize
            cols: terminal.cols || 80,
            rows: terminal.rows || 24,
          });
        } catch (err) {
          console.error("Failed to send resize:", err);
          // Don't disconnect on resize errors
        }
      }
    }
  });

  if (terminalContainer.value) {
    resizeObserver.observe(terminalContainer.value);
  }

  // Handle user input - send via unary RPC
  terminal.onData(async (data: string) => {
    if (isConnected.value) {
      try {
        // Convert string to Uint8Array
        const encoder = new TextEncoder();
        const input = encoder.encode(data);
        
        // Send input via unary RPC call
        await client.sendTerminalInput({
          organizationId: organizationId.value,
          deploymentId: props.deploymentId,
          input: input,
          cols: terminal?.cols || 80,
          rows: terminal?.rows || 24,
        });
      } catch (err) {
        console.error("Failed to send terminal input:", err);
        // Don't disconnect on input errors, just log
      }
    }
  });
};

const connectTerminal = async () => {
  if (!terminal) {
    await initTerminal();
  }

  isLoading.value = true;
  error.value = "";

  try {
    if (!terminal) {
      throw new Error("Terminal not initialized");
    }

    // Create server stream for terminal output (gRPC-Web supports server streaming!)
    const controller = new AbortController();
    streamController = controller;

    terminal.write("\r\n\x1b[32mConnecting to container...\x1b[0m\r\n");
    
    // Start server stream for terminal output
    terminalStream = client.streamTerminalOutput(
      {
        organizationId: organizationId.value,
        deploymentId: props.deploymentId,
        cols: terminal.cols || 80,
        rows: terminal.rows || 24,
      },
      { signal: controller.signal }
    );

    isConnected.value = true;
    reconnectAttempts = 0;

    // Listen for messages from the server (async iteration)
    // Server streaming works perfectly in browsers with gRPC-Web!
    (async () => {
      try {
        for await (const output of terminalStream) {
          if (!isConnected.value) break;
          
          if (terminal && output.output) {
            const text = new TextDecoder().decode(output.output);
            terminal.write(text);
          }
          
          if (output.exit) {
            terminal?.write("\r\n\x1b[31m[Terminal session ended]\x1b[0m\r\n");
            isConnected.value = false;
            break;
          }
        }
      } catch (err: any) {
        if (isConnected.value) {
          console.error("Terminal stream error:", err);
          error.value = "Connection lost. Please reconnect.";
          isConnected.value = false;
          terminal?.write("\r\n\x1b[31m[Connection error]\x1b[0m\r\n");
        }
      }
    })();

  } catch (err: any) {
    console.error("Failed to connect terminal:", err);
    const errMsg = err.message || "Failed to connect terminal. Please try again.";
    error.value = errMsg;
    
    if (terminal) {
      terminal.write(`\r\n\x1b[31m[ERROR]\x1b[0m ${errMsg}\r\n`);
    }
    
    isConnected.value = false;
  } finally {
    isLoading.value = false;
  }
};

const disconnectTerminal = () => {
  isConnected.value = false;
  
  if (streamController) {
    streamController.abort();
    streamController = null;
  }
  if (terminalStream) {
    try {
      terminalStream.close?.();
    } catch (err) {
      // Ignore close errors
    }
    terminalStream = null;
  }
  if (terminal) {
    terminal.write("\r\n\x1b[33m[Disconnected]\x1b[0m\r\n");
  }
};

watch(() => props.deploymentId, () => {
  if (isConnected.value) {
    disconnectTerminal();
  }
});

onMounted(async () => {
  await nextTick();
  await initTerminal();
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
