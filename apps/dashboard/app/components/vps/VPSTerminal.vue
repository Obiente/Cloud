<template>
  <OuiStack gap="sm">
    <OuiText as="h3" size="sm" weight="semibold">
      Terminal Access
    </OuiText>

    <div class="w-full">
      <div class="flex items-center gap-2 p-2 border border-border-muted rounded-lg bg-background-base">
        <div class="text-secondary font-mono text-sm">$</div>
        <input
          ref="commandInput"
          v-model="command"
          type="text"
          class="flex-1 bg-transparent border-none outline-none text-primary font-mono text-sm"
          placeholder="Type command and press Enter..."
          :disabled="false"
          :readonly="false"
          autocomplete="off"
          @keydown.enter="sendCommand"
          @keydown.up="navigateHistory('up')"
          @keydown.down="navigateHistory('down')"
          @keydown.tab.prevent="handleTab"
        />
        <OuiButton
          v-if="command.trim()"
          variant="ghost"
          size="sm"
          @click="sendCommand"
          class="shrink-0"
        >
          Send
        </OuiButton>
      </div>
    </div>

    <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>

    <OuiText v-if="isConnected && !error" size="xs" color="secondary">
      Press Enter to send commands to the VPS
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
const emit = defineEmits<{
  "log-output": [text: string];
}>();
const orgsStore = useOrganizationsStore();
const organizationId = computed(
  () => props.organizationId || orgsStore.currentOrgId || ""
);
const auth = useAuth();

const commandInput = ref<HTMLInputElement | null>(null);
const command = ref("");
const isConnected = ref(false);
const isLoading = ref(false);
const error = ref("");
const commandHistory = ref<string[]>([]);
const historyIndex = ref(-1);
const awaitingTabCompletion = ref(false);
const commandBeforeTab = ref("");
let tabCompletionTimeout: ReturnType<typeof setTimeout> | null = null;

let websocket: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempts = 0;
const shouldAttemptReconnect = ref(true);

const sendCommand = () => {
  if (!command.value.trim()) {
    return;
  }

  const cmd = command.value.trim();
  
  // Add to history
  if (cmd && (commandHistory.value.length === 0 || commandHistory.value[commandHistory.value.length - 1] !== cmd)) {
    commandHistory.value.push(cmd);
    // Keep only last 50 commands
    if (commandHistory.value.length > 50) {
      commandHistory.value.shift();
    }
  }
  historyIndex.value = -1;

  // Check if connected before sending
  if (!isConnected.value || !websocket || websocket.readyState !== WebSocket.OPEN) {
    error.value = "Not connected. Commands will be sent when connection is established.";
    // Try to reconnect if not already connecting
    if (!isLoading.value && shouldAttemptReconnect.value) {
      connectTerminal();
    }
    command.value = "";
    return;
  }

  // Send command to VPS stdin
  const encoder = new TextEncoder();
  const input = encoder.encode(cmd + "\n");
  try {
    websocket.send(
      JSON.stringify({
        type: "input",
        input: Array.from(input),
      })
    );
    command.value = "";
    error.value = "";
  } catch (err) {
    console.error("Failed to send command:", err);
    error.value = "Failed to send command. Please try again.";
  }
};

const navigateHistory = (direction: "up" | "down") => {
  if (commandHistory.value.length === 0) return;

  if (direction === "up") {
    if (historyIndex.value < commandHistory.value.length - 1) {
      historyIndex.value++;
      const cmd = commandHistory.value[commandHistory.value.length - 1 - historyIndex.value];
      command.value = cmd ?? "";
    }
  } else {
    if (historyIndex.value > 0) {
      historyIndex.value--;
      const cmd = commandHistory.value[commandHistory.value.length - 1 - historyIndex.value];
      command.value = cmd ?? "";
    } else {
      historyIndex.value = -1;
      command.value = "";
    }
  }
};

const handleTab = (event: KeyboardEvent) => {
  event.preventDefault();
  
  if (!isConnected.value || !websocket || websocket.readyState !== WebSocket.OPEN) {
    return;
  }

  if (tabCompletionTimeout) {
    clearTimeout(tabCompletionTimeout);
  }

  commandBeforeTab.value = command.value;
  awaitingTabCompletion.value = true;
  
  tabCompletionTimeout = setTimeout(() => {
    awaitingTabCompletion.value = false;
    tabCompletionTimeout = null;
  }, 2000);

  const encoder = new TextEncoder();
  const currentCommand = command.value;
  const commandBytes = encoder.encode(currentCommand);
  const tabChar = encoder.encode("\t");
  
  const combinedInput = new Uint8Array(commandBytes.length + tabChar.length);
  combinedInput.set(commandBytes, 0);
  combinedInput.set(tabChar, commandBytes.length);
  
  try {
    websocket.send(
      JSON.stringify({
        type: "input",
        input: Array.from(combinedInput),
      })
    );
  } catch (err) {
    console.error("Failed to send tab completion:", err);
    awaitingTabCompletion.value = false;
    if (tabCompletionTimeout) {
      clearTimeout(tabCompletionTimeout);
      tabCompletionTimeout = null;
    }
  }
};

const handleTabCompletionOutput = (outputData: number[]) => {
  if (!awaitingTabCompletion.value) return;
  
  try {
    const output = new Uint8Array(outputData);
    const text = new TextDecoder().decode(output);
    
    const ansiRegex = /\x1b\[[0-9;]*[a-zA-Z]/g;
    let cleanText = text.replace(ansiRegex, "");
    cleanText = cleanText.replace(/[\x00-\x08\x0B-\x1F\x7F]/g, "");
    
    const prefix = commandBeforeTab.value.trim();
    const lines = cleanText.split(/\r?\n/);
    
    for (const line of lines) {
      const trimmedLine = line.trim();
      
      if (trimmedLine.startsWith(prefix) && trimmedLine.length > prefix.length) {
        const completion = trimmedLine.substring(prefix.length);
        command.value = commandBeforeTab.value + completion;
        
        if (tabCompletionTimeout) {
          clearTimeout(tabCompletionTimeout);
          tabCompletionTimeout = null;
        }
        awaitingTabCompletion.value = false;
        
        nextTick(() => {
          const input = commandInput.value;
          if (input) {
            input.setSelectionRange(command.value.length, command.value.length);
          }
        });
        return;
      }
    }
  } catch (err) {
    console.error("Failed to process tab completion:", err);
    awaitingTabCompletion.value = false;
    if (tabCompletionTimeout) {
      clearTimeout(tabCompletionTimeout);
      tabCompletionTimeout = null;
    }
  }
};

const connectTerminal = async () => {
  if (isConnected.value) {
    return;
  }

  shouldAttemptReconnect.value = true;
  isLoading.value = true;
  error.value = "";

  try {
    const config = useRuntimeConfig();
    const apiBase = config.public.apiHost || config.public.requestHost;
    const appConfig = useConfig();
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
        cols: 80,
        rows: 24,
      };

      websocket!.send(JSON.stringify(initMessage));
    };

    websocket.onmessage = (event: MessageEvent) => {
      try {
        const message = JSON.parse(event.data);

        if (message.type === "connected") {
          isConnected.value = true;
          isLoading.value = false;
          reconnectAttempts = 0;
          shouldAttemptReconnect.value = true;
          nextTick(() => {
            commandInput.value?.focus();
          });
        } else if (message.type === "output") {
          if (awaitingTabCompletion.value && message.data && Array.isArray(message.data)) {
            handleTabCompletionOutput(message.data);
          }
          if (message.data && Array.isArray(message.data)) {
            try {
              const outputBytes = new Uint8Array(message.data);
              const outputText = new TextDecoder().decode(outputBytes);
              emit("log-output", outputText);
            } catch (err) {
              console.error("Error decoding output:", err);
            }
          }
        } else if (message.type === "closed") {
          isConnected.value = false;
        } else if (message.type === "error") {
          error.value = message.message || "Connection error";
          isConnected.value = false;
        }
      } catch (err) {
        console.error("Error processing WebSocket message:", err);
      }
    };

    websocket.onerror = (err) => {
      console.error("WebSocket error:", err);
      error.value = "Connection error. Attempting to reconnect...";
      isConnected.value = false;
    };

    websocket.onclose = () => {
      isConnected.value = false;
      isLoading.value = false;

      if (shouldAttemptReconnect.value && reconnectAttempts < 5) {
        reconnectAttempts++;
        reconnectTimer = setTimeout(() => {
          connectTerminal();
        }, Math.min(1000 * reconnectAttempts, 5000));
      } else if (reconnectAttempts >= 5) {
        error.value = "Failed to connect after multiple attempts. Please refresh the page.";
      }
    };
  } catch (err) {
    console.error("Failed to connect terminal:", err);
    error.value = err instanceof Error ? err.message : "Failed to connect";
    isLoading.value = false;
  }
};

onMounted(() => {
  connectTerminal();
});

onUnmounted(() => {
  shouldAttemptReconnect.value = false;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
  }
  if (tabCompletionTimeout) {
    clearTimeout(tabCompletionTimeout);
  }
  websocket?.close();
});
</script>

