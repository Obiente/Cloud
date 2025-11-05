<template>
  <OuiStack gap="sm">
    <OuiText as="h3" size="sm" weight="semibold">
      Command Input
    </OuiText>

    <div class="command-input-wrapper">
      <div class="command-input-container">
        <div class="command-prompt">$</div>
        <input
          ref="commandInput"
          v-model="command"
          type="text"
          class="command-input"
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
          class="command-send-button"
        >
          Send
        </OuiButton>
      </div>
    </div>

    <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>

    <OuiText v-if="isConnected && !error" size="xs" color="secondary">
      Press Enter to send commands to the game server stdin
    </OuiText>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { useAuth } from "~/composables/useAuth";
import { useOrganizationsStore } from "~/stores/organizations";

interface Props {
  gameServerId: string;
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
// Always attempt to reconnect - logs should always be connected
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

  // Send command to game server stdin
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
  // Prevent default tab behavior (focus navigation)
  event.preventDefault();
  
  // Send Tab character (ASCII 9) to the backend for tab completion
  if (!isConnected.value || !websocket || websocket.readyState !== WebSocket.OPEN) {
    return;
  }

  // Clear any existing timeout
  if (tabCompletionTimeout) {
    clearTimeout(tabCompletionTimeout);
  }

  // Store current command before tab for completion processing
  commandBeforeTab.value = command.value;
  awaitingTabCompletion.value = true;
  
  // Set timeout to reset awaitingTabCompletion if no response comes
  // Give more time for tab completion (some shells take longer)
  tabCompletionTimeout = setTimeout(() => {
    awaitingTabCompletion.value = false;
    tabCompletionTimeout = null;
  }, 2000);

  const encoder = new TextEncoder();
  
  // IMPORTANT: Send the current command text first, then Tab
  // The shell needs to know what text we're trying to complete
  // We need to send it as if it was typed character by character
  const currentCommand = command.value;
  const commandBytes = encoder.encode(currentCommand);
  const tabChar = encoder.encode("\t");
  
  // Combine command text + Tab character
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
    // Convert output data to string
    const output = new Uint8Array(outputData);
    const text = new TextDecoder().decode(output);
    
    // Remove ANSI escape codes (including cursor movement codes)
    const ansiRegex = /\x1b\[[0-9;]*[a-zA-Z]/g;
    let cleanText = text.replace(ansiRegex, "");
    
    // Also remove control characters except newlines
    cleanText = cleanText.replace(/[\x00-\x08\x0B-\x1F\x7F]/g, "");
    
    // Get the prefix we had before Tab
    const prefix = commandBeforeTab.value.trim();
    
    // For tab completion, shells typically output the completed text
    // This might be mixed with other output, so we need to extract it carefully
    
    // Try to find the completed command in the output
    // Look for text that starts with our prefix
    const lines = cleanText.split(/\r?\n/);
    
    // Also check the raw text for completion patterns
    // Tab completion often shows: "prefix" + "completion" or just the completion part
    for (const line of lines) {
      const trimmedLine = line.trim();
      
      // If line starts with our prefix and is longer, extract completion
      if (trimmedLine.startsWith(prefix) && trimmedLine.length > prefix.length) {
        const completion = trimmedLine.substring(prefix.length);
        // Update command with completion
        command.value = commandBeforeTab.value + completion;
        
        // Clear timeout and reset flag
        if (tabCompletionTimeout) {
          clearTimeout(tabCompletionTimeout);
          tabCompletionTimeout = null;
        }
        awaitingTabCompletion.value = false;
        
        // Set cursor position after completion
        nextTick(() => {
          const input = commandInput.value;
          if (input) {
            input.setSelectionRange(command.value.length, command.value.length);
          }
        });
        return;
      }
      
      // If we see a single word that could be a completion
      if (lines.length === 1 && trimmedLine.length > 0 && !trimmedLine.includes(" ")) {
        // Check if it's a continuation of our prefix
        if (trimmedLine.startsWith(prefix)) {
          command.value = trimmedLine;
          
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
    }
    
    // If we see common completion patterns (multiple options separated by spaces)
    // This happens when there are multiple matches - we can't auto-complete
    // But we'll reset the flag so next output messages aren't treated as completion
    if (cleanText.includes("  ") || lines.length > 3) {
      // Multiple options shown - don't auto-complete but reset flag
      if (tabCompletionTimeout) {
        clearTimeout(tabCompletionTimeout);
        tabCompletionTimeout = null;
      }
      awaitingTabCompletion.value = false;
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
  // Don't connect if already connected
  if (isConnected.value) {
    return;
  }

  shouldAttemptReconnect.value = true;
  isLoading.value = true;
  error.value = "";

  try {
    const config = useRuntimeConfig();
    const apiBase = config.public.apiHost || config.public.requestHost;
    const disableAuth = Boolean(config.public.disableAuth);
    const wsUrlObject = new URL("/gameservers/terminal/ws", apiBase);
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
        // Still attempt reconnect in case auth becomes available
        return;
      }

      const initMessage: any = {
        type: "init",
        gameServerId: props.gameServerId,
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
          // Handle tab completion output - capture output for a short time after Tab press
          if (awaitingTabCompletion.value && message.data && Array.isArray(message.data)) {
            handleTabCompletionOutput(message.data);
          }
          // Convert output data to string and emit to parent logs component
          if (message.data && Array.isArray(message.data)) {
            try {
              const outputBytes = new Uint8Array(message.data);
              const outputText = new TextDecoder().decode(outputBytes);
              console.log("[GameServer Terminal] Received output:", outputText.length, "chars, first 100:", outputText.substring(0, 100));
              // Emit log output to parent component
              emit("log-output", outputText);
            } catch (err) {
              console.error("Error decoding output:", err);
            }
          }
        } else if (message.type === "closed") {
          // Connection closed - will auto-reconnect
          isConnected.value = false;
        } else if (message.type === "error") {
          error.value = message.message || "Connection error";
          isConnected.value = false;
          // Will auto-reconnect via onclose handler
        }
      } catch (err) {
        console.error("Error processing WebSocket message:", err);
      }
    };

    websocket.onerror = (err) => {
      console.error("WebSocket error:", err);
      if (!isConnected.value) {
        error.value = "Failed to connect. Please try again.";
        isLoading.value = false;
      }
    };

    websocket.onclose = () => {
      const wasConnected = isConnected.value;
      isConnected.value = false;
      websocket = null;

      if (shouldAttemptReconnect.value && wasConnected) {
        isLoading.value = true;
      }

      if (shouldAttemptReconnect.value) {
        scheduleReconnect();
      } else {
        isLoading.value = false;
      }
    };
  } catch (err: any) {
    console.error("Failed to connect:", err);
    const errMsg = err.message || "Failed to connect. Please try again.";
    error.value = errMsg;
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
  // Only used for cleanup on unmount - don't stop reconnection
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
};

watch(
  () => props.gameServerId,
  async () => {
    // Reconnect when game server ID changes
    disconnectTerminal();
    await nextTick();
    await connectTerminal();
  }
);

onMounted(async () => {
  await nextTick();
  // Auto-connect when component mounts
  await connectTerminal();
});

onUnmounted(() => {
  // Stop reconnection when component unmounts
  shouldAttemptReconnect.value = false;
  disconnectTerminal();
});
</script>

<style scoped>
.command-input-wrapper {
  width: 100%;
}

.command-input-container {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  background: var(--oui-surface-base);
  border: 1px solid var(--oui-border-default);
  border-radius: var(--oui-radius-md);
  padding: 0.5rem 0.75rem;
  transition: border-color 0.2s;
}

.command-input-container:focus-within {
  border-color: var(--oui-accent-primary);
  outline: 2px solid var(--oui-accent-primary);
  outline-offset: 2px;
}

.command-prompt {
  color: var(--oui-accent-primary);
  font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 0.875rem;
  font-weight: 600;
  user-select: none;
  flex-shrink: 0;
}

.command-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: var(--oui-text-primary);
  font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
  font-size: 0.875rem;
  padding: 0;
  min-width: 0;
  pointer-events: auto;
  cursor: text;
}

.command-input::placeholder {
  color: var(--oui-text-secondary);
  opacity: 0.6;
}

.command-input:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  pointer-events: none;
}

.command-send-button {
  flex-shrink: 0;
  padding: 0.25rem 0.75rem;
  height: auto;
}
</style>

