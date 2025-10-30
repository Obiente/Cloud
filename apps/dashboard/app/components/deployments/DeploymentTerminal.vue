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
        class="w-full h-96 rounded-lg bg-black border border-border-default overflow-hidden"
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
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";
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

let terminal: Terminal | null = null;
let fitAddon: FitAddon | null = null;
let terminalStream: any = null;
let reconnectAttempts = 0;

const initTerminal = async () => {
  if (!terminalContainer.value) return;

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: "#000000",
      foreground: "#00ff00",
      cursor: "#00ff00",
      selection: "#ffffff40",
    },
  });

  fitAddon = new FitAddon();
  terminal.loadAddon(fitAddon);

  terminal.open(terminalContainer.value);
  fitAddon.fit();

  // Handle terminal resize
  const resizeObserver = new ResizeObserver(() => {
    if (fitAddon && terminal) {
      fitAddon.fit();
      if (terminalStream && terminal) {
        const dims = terminal.getOption("cols") && terminal.getOption("rows");
        if (dims) {
          terminalStream.send({
            organizationId: organizationId.value,
            deploymentId: props.deploymentId,
            input: new Uint8Array(0),
            cols: terminal.getOption("cols") || 80,
            rows: terminal.getOption("rows") || 24,
          });
        }
      }
    }
  });

  if (terminalContainer.value) {
    resizeObserver.observe(terminalContainer.value);
  }

  // Handle input
  terminal.onData((data) => {
    if (terminalStream && isConnected.value) {
      const input = new TextEncoder().encode(data);
      terminalStream.send({
        organizationId: organizationId.value,
        deploymentId: props.deploymentId,
        input: Array.from(input),
        cols: terminal?.getOption("cols") || 80,
        rows: terminal?.getOption("rows") || 24,
      });
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

    // Create bidirectional stream
    terminalStream = client.streamTerminal();

    // Get terminal dimensions
    const cols = terminal.getOption("cols") || 80;
    const rows = terminal.getOption("rows") || 24;

    // Send initial connection message
    await terminalStream.send({
      organizationId: organizationId.value,
      deploymentId: props.deploymentId,
      input: new Uint8Array(0),
      cols: cols,
      rows: rows,
    });

    terminal.write("\r\n\x1b[32mConnecting to container...\x1b[0m\r\n");

    // Handle output from container
    terminalStream.onMessage((output: any) => {
      if (terminal && output.output) {
        const text = new TextDecoder().decode(output.output);
        terminal.write(text);
      }
      if (output.exit) {
        terminal?.write("\r\n\x1b[31m[Terminal session ended]\x1b[0m\r\n");
        isConnected.value = false;
      }
    });

    terminalStream.onError((err: any) => {
      console.error("Terminal stream error:", err);
      error.value = "Connection lost. Please reconnect.";
      isConnected.value = false;
      terminal?.write("\r\n\x1b[31m[Connection error]\x1b[0m\r\n");
    });

    isConnected.value = true;
    reconnectAttempts = 0;
  } catch (err: any) {
    console.error("Failed to connect terminal:", err);
    error.value = err.message || "Failed to connect terminal. Please try again.";
    terminal?.write(`\r\n\x1b[31mError: ${error.value}\x1b[0m\r\n`);
  } finally {
    isLoading.value = false;
  }
};

const disconnectTerminal = () => {
  if (terminalStream) {
    terminalStream.close();
    terminalStream = null;
  }
  isConnected.value = false;
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
