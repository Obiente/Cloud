<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiText as="h3" size="md" weight="semibold">
        Game Server Logs
      </OuiText>
      <OuiFlex gap="sm">
        <OuiButton
          variant="ghost"
          size="sm"
          @click="toggleFollow"
          :class="{ 'text-primary': isFollowing && isConnected }"
          :disabled="isLoading"
        >
          <ArrowPathIcon
            class="h-4 w-4 mr-1"
            :class="{ 'animate-spin': isLoading || (isFollowing && !isConnected) }"
          />
          {{ isLoading ? "Connecting..." : isFollowing && isConnected ? "Connected" : "Disconnected" }}
        </OuiButton>
        <OuiButton variant="ghost" size="sm" @click="clearLogs">
          Clear
        </OuiButton>
        <OuiMenu>
          <template #trigger>
            <OuiButton variant="ghost" size="sm">
              <EllipsisVerticalIcon class="h-4 w-4" />
            </OuiButton>
          </template>
          <template #default>
            <OuiMenuItem>
              <OuiCheckbox
                v-model="showTimestamps"
                label="Show timestamps"
                @click.stop
              />
            </OuiMenuItem>
            <OuiMenuItem>
              <label class="flex items-center gap-2 px-4 py-2 text-sm cursor-pointer">
                <span>Tail lines:</span>
                <OuiInput
                  :model-value="tailLines.toString()"
                  type="number"
                  :min="10"
                  :max="10000"
                  size="sm"
                  style="width: 100px;"
                  @update:model-value="handleTailChange"
                  @click.stop
                />
              </label>
            </OuiMenuItem>
          </template>
        </OuiMenu>
      </OuiFlex>
    </OuiFlex>

    <OuiLogs
      ref="logsComponent"
      :logs="formattedLogs"
      :is-loading="isLoading"
      :show-timestamps="showTimestamps"
      :show-tail-controls="false"
      :enable-ansi="true"
      :auto-scroll="true"
      empty-message="No logs available. Start following to see real-time logs."
      loading-message="Connecting..."
      title="Game Server Logs"
    />

    <!-- Interactive Terminal (replaces command input for tab autocomplete support) -->
    <GameServerTerminal
      :game-server-id="props.gameServerId"
      :organization-id="props.organizationId"
      @log-output="handleTerminalOutput"
    />

    <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>
    <OuiText v-if="isConnected && !error" size="xs" color="success">
      ✓ Connected. Logs will appear here when the server is running.
    </OuiText>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { ArrowPathIcon, EllipsisVerticalIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import { useAuth } from "~/composables/useAuth";
import type { LogEntry } from "~/components/oui/Logs.vue";
import GameServerTerminal from "~/components/gameserver/GameServerTerminal.vue";

interface Props {
  gameServerId: string;
  organizationId: string;
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const auth = useAuth();
const client = useConnectClient(GameServerService);

const effectiveOrgId = computed(() => props.organizationId || orgsStore.currentOrgId || "");

const logsComponent = ref<any>(null);
const isLoading = ref(false);
const isFollowing = ref(false);
const isConnected = ref(false);
const error = ref<string | null>(null);
const showTimestamps = ref(true);
const tailLines = ref(100);

let logStream: any = null;
let streamController: AbortController | null = null;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 10;
const RECONNECT_DELAY = 2000; // Start with 2 seconds, exponential backoff
const logs = ref<Array<{ line: string; timestamp: string; level?: number }>>([]);
let terminalOutputBuffer = ""; // Buffer for partial lines from terminal WebSocket

// Helper function to check if a line is empty or only whitespace
function isEmptyOrWhitespace(line: string): boolean {
  return !line || line.trim().length === 0;
}

// Format logs for OuiLogs component
const formattedLogs = computed<LogEntry[]>(() => {
  return logs.value
    .filter((log) => !isEmptyOrWhitespace(log.line)) // Filter out empty/whitespace-only lines
    .map((log) => {
      // Map log level from backend to frontend format
      // OuiLogs expects: "info" | "warning" | "error" | "debug" | "trace"
      let level: "info" | "warning" | "error" | "debug" | "trace" = "info";
      if (log.level !== undefined) {
        // LogLevel enum: 0=UNSPECIFIED, 1=TRACE, 2=DEBUG, 3=INFO, 4=WARN, 5=ERROR
        switch (log.level) {
          case 5: // ERROR
            level = "error";
            break;
          case 4: // WARN -> "warning" (OuiLogs expects "warning", not "warn")
            level = "warning";
            break;
          case 2: // DEBUG
          case 1: // TRACE
            level = "debug";
            break;
          case 3: // INFO
          default:
            level = "info";
            break;
        }
      }
      return {
        line: log.line,
        timestamp: log.timestamp ? new Date(log.timestamp) : undefined,
        level,
      };
    });
});

const toggleFollow = async () => {
  if (isFollowing.value) {
    stopFollowing();
  } else {
    await startFollowing();
  }
};

// Auto-reconnect logic
const scheduleReconnect = () => {
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
  }
  
  if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
    error.value = "Failed to reconnect after multiple attempts. Please refresh the page.";
    isFollowing.value = false;
    isConnected.value = false;
    return;
  }

  reconnectAttempts++;
  const delay = Math.min(RECONNECT_DELAY * Math.pow(2, reconnectAttempts - 1), 30000); // Exponential backoff, max 30s
  
  reconnectTimeout = setTimeout(async () => {
    if (!isFollowing.value) {
      return; // User stopped following
    }
    console.log(`Attempting to reconnect (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`);
    error.value = `Reconnecting... (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`;
    await startFollowing();
  }, delay);
};

const startFollowing = async () => {
  if (isFollowing.value && isConnected.value) return; // Already connected
  
  // Cancel any pending reconnect
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
  
  isLoading.value = true;
  error.value = null;
  isFollowing.value = true;

  try {
    // Wait for auth to be ready before making the request
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

    // Ensure we have a token before streaming
    const token = await auth.getAccessToken();
    if (!token) {
      throw new Error("Authentication required. Please log in.");
    }

    // Get recent logs first (only on initial connection or reconnection)
    if (logs.value.length === 0 || reconnectAttempts > 0) {
      try {
        const response = await client.getGameServerLogs({
          gameServerId: props.gameServerId,
          limit: tailLines.value,
        });

        if (response.lines && response.lines.length > 0) {
          // Only add logs if we don't have recent ones to avoid duplicates
          const existingLogs = logs.value.map(l => l.line);
          const newLogs = response.lines
            .filter((line) => {
              const lineText = line.line || '';
              return !isEmptyOrWhitespace(lineText) && !existingLogs.includes(lineText);
            })
            .map((line) => {
              // Safely parse timestamp
              let timestamp: string;
              try {
                if (line.timestamp) {
                  const ts = typeof line.timestamp === 'string' 
                    ? line.timestamp 
                    : (line.timestamp as any)?.seconds 
                      ? new Date(Number((line.timestamp as any).seconds) * 1000).toISOString()
                      : new Date(line.timestamp as any).toISOString();
                  timestamp = ts;
                } else {
                  timestamp = new Date().toISOString();
                }
              } catch (err) {
                timestamp = new Date().toISOString();
              }
              return {
                line: line.line || '',
                timestamp,
                level: line.level,
              };
            });
          logs.value.push(...newLogs);
        }
      } catch (err) {
        // Non-fatal - continue with streaming even if initial fetch fails
        console.warn("Failed to fetch initial logs:", err);
      }
    }

    // Start streaming logs with abort controller
    streamController = new AbortController();
    logStream = client.streamGameServerLogs(
      {
        gameServerId: props.gameServerId,
        follow: true, // Always follow
        tail: tailLines.value,
      },
      { signal: streamController.signal }
    );

    isConnected.value = true;
    isLoading.value = false;
    reconnectAttempts = 0; // Reset on successful connection
    error.value = null;

    // Handle stream messages
    for await (const logLine of logStream) {
      // Safely parse timestamp
      let timestamp: string;
      try {
        if (logLine.timestamp) {
          const ts = typeof logLine.timestamp === 'string' 
            ? logLine.timestamp 
            : (logLine.timestamp as any)?.seconds 
              ? new Date(Number((logLine.timestamp as any).seconds) * 1000).toISOString()
              : new Date(logLine.timestamp as any).toISOString();
          timestamp = ts;
        } else {
          timestamp = new Date().toISOString();
        }
      } catch (err) {
        timestamp = new Date().toISOString();
      }

      // Only add non-empty lines
      const line = logLine.line || '';
      if (!isEmptyOrWhitespace(line)) {
        logs.value.push({
          line: line,
          timestamp,
          level: logLine.level,
        });
      }

      // Keep only last 10000 lines
      if (logs.value.length > 10000) {
        logs.value = logs.value.slice(-10000);
      }
    }
    
    // Stream ended (not aborted) - schedule reconnect
    if (isFollowing.value && !streamController.signal.aborted) {
      isConnected.value = false;
      scheduleReconnect();
    }
  } catch (err: any) {
    // Suppress abort-related errors
    const isAbortError = 
      err.name === "AbortError" || 
      err.message?.toLowerCase().includes("aborted") ||
      err.message?.toLowerCase().includes("canceled") ||
      err.message?.toLowerCase().includes("cancelled");

    if (isAbortError) {
      return;
    }

    // Suppress benign errors (missing trailer, etc.)
    const isBenignError =
      err.message?.toLowerCase().includes("missing trailer") ||
      err.message?.toLowerCase().includes("trailer") ||
      err.message?.toLowerCase().includes("missing endstreamresponse") ||
      err.message?.toLowerCase().includes("endstreamresponse") ||
      err.message?.toLowerCase().includes("unimplemented") ||
      err.message?.toLowerCase().includes("not fully implemented") ||
      err.code === "unknown";

    if (!isBenignError) {
      console.error("Failed to stream logs:", err);
      error.value = err.message || "Failed to connect to logs stream";
    }
    
    // Schedule reconnect if we're still supposed to be following
    if (isFollowing.value && !isAbortError) {
      isConnected.value = false;
      scheduleReconnect();
    } else {
      isFollowing.value = false;
      isConnected.value = false;
    }
  } finally {
    isLoading.value = false;
  }
};

const stopFollowing = () => {
  // Cancel reconnect attempts
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
  reconnectAttempts = 0;
  
  if (streamController) {
    streamController.abort();
    streamController = null;
  }
  if (logStream) {
    try {
      logStream.return?.();
    } catch (err) {
      // Ignore errors when closing stream
    }
    logStream = null;
  }
  isFollowing.value = false;
  isConnected.value = false;
};

const clearLogs = () => {
  logs.value = [];
  if (logsComponent.value) {
    logsComponent.value.scrollToTop?.();
  }
};

const handleTailChange = (value: string) => {
  const num = parseInt(value, 10);
  if (!isNaN(num) && num >= 10 && num <= 10000) {
    tailLines.value = num;
  }
};

// Handle output from terminal WebSocket
const handleTerminalOutput = (text: string) => {
  if (!text) return;
  
  console.log("[GameServer Logs] handleTerminalOutput called with:", text.length, "chars, first 100:", text.substring(0, 100));
  
  // Add to buffer
  terminalOutputBuffer += text;
  
  // Split by newlines - keep incomplete line in buffer
  const lines = terminalOutputBuffer.split(/\r?\n/);
  
  // If buffer ends with newline, last element will be empty string
  // Otherwise, last element is an incomplete line that stays in buffer
  if (terminalOutputBuffer.endsWith("\n") || terminalOutputBuffer.endsWith("\r")) {
    terminalOutputBuffer = "";
  } else {
    terminalOutputBuffer = lines.pop() || ""; // Keep last incomplete line
  }
  
  // Add complete lines to logs (filter out empty/whitespace-only lines)
  let linesAdded = 0;
  for (const line of lines) {
    // Skip empty or whitespace-only lines
    if (isEmptyOrWhitespace(line)) {
      continue;
    }
    
    logs.value.push({
      line: line,
      timestamp: new Date().toISOString(),
      level: undefined, // Terminal output doesn't have a log level
    });
    linesAdded++;
    
    // Keep only last 10000 lines
    if (logs.value.length > 10000) {
      logs.value = logs.value.slice(-10000);
    }
  }
  
  console.log("[GameServer Logs] Added", linesAdded, "lines to logs. Total logs:", logs.value.length);
  
  // Also flush buffer if it gets too large (in case we never get a newline)
  if (terminalOutputBuffer.length > 1000) {
    console.log("[GameServer Logs] Flushing large buffer:", terminalOutputBuffer.length, "chars");
    logs.value.push({
      line: terminalOutputBuffer,
      timestamp: new Date().toISOString(),
      level: undefined,
    });
    terminalOutputBuffer = "";
    
    if (logs.value.length > 10000) {
      logs.value = logs.value.slice(-10000);
    }
  }
};

onMounted(() => {
  // Auto-start following logs when component mounts
  startFollowing();
});

onUnmounted(() => {
  // Clean up when component unmounts
  stopFollowing();
});

// Watch for gameServerId changes and restart streaming
watch(() => props.gameServerId, () => {
  if (isFollowing.value) {
    stopFollowing();
    nextTick(() => {
      startFollowing();
    });
  }
});
</script>

