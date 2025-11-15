<template>
  <OuiStack gap="md">
    <OuiFlex justify="between" align="center">
      <OuiStack gap="xs">
        <OuiText as="h3" size="md" weight="semibold">
          Game Server Logs
        </OuiText>
        <OuiText size="xs" color="muted">
          Real-time logs from your game server
        </OuiText>
      </OuiStack>
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

    <!-- Loading older logs indicator -->
    <div v-if="isLoadingOlderLogs" class="logs-loading-indicator">
      <OuiFlex align="center" justify="center" gap="sm">
        <ArrowPathIcon class="h-4 w-4 animate-spin" />
        <OuiText size="xs" color="muted">Loading older logs...</OuiText>
      </OuiFlex>
    </div>

    <!-- No more logs indicator -->
    <div v-if="hasLoadedAllLogs && logs.length > 0" class="logs-end-indicator">
      <OuiText size="xs" color="muted" align="center">
        No more logs available
      </OuiText>
    </div>

    <div ref="logsContainer" class="logs-container-wrapper" @scroll="handleScroll">
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
      >
        <template #footer>
          <!-- Empty footer to hide inline controls since we have them in the menu -->
        </template>
      </OuiLogs>
    </div>

    <!-- Interactive Terminal -->
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
import { timestamp } from "@obiente/proto/utils";
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
const logsContainer = ref<HTMLDivElement | null>(null);
const isLoading = ref(false);
const isLoadingOlderLogs = ref(false);
const isFollowing = ref(false);
const isConnected = ref(false);
const error = ref<string | null>(null);
const showTimestamps = ref(true);
const tailLines = ref(100);
const hasLoadedAllLogs = ref(false);

let logStream: any = null;
let streamController: AbortController | null = null;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
let reconnectAttempts = 0;
const MAX_RECONNECT_ATTEMPTS = 10;
const RECONNECT_DELAY = 2000;
const logs = ref<Array<{ line: string; timestamp: string; level?: number }>>([]);
let terminalOutputBuffer = "";
let scrollPositionBeforeLoad = 0;
let isLoadingOlderLogsDebounce: ReturnType<typeof setTimeout> | null = null;

// Helper function to check if a line is empty or only whitespace
function isEmptyOrWhitespace(line: string): boolean {
  return !line || line.trim().length === 0;
}

// Format logs for OuiLogs component
const formattedLogs = computed<LogEntry[]>(() => {
  return logs.value
    .filter((log) => !isEmptyOrWhitespace(log.line))
    .map((log) => {
      let level: "info" | "warning" | "error" | "debug" | "trace" = "info";
      if (log.level !== undefined) {
        switch (log.level) {
          case 5: // ERROR
            level = "error";
            break;
          case 4: // WARN
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

// Handle scroll to detect when user scrolls to top for lazy loading
const handleScroll = () => {
  if (!logsContainer.value || isLoadingOlderLogs.value || hasLoadedAllLogs.value) return;

  const container = logsContainer.value;
  const scrollTop = container.scrollTop;
  
  // If user scrolls within 200px of the top, load older logs
  if (scrollTop < 200 && logs.value.length > 0) {
    // Debounce to avoid multiple rapid requests
    if (isLoadingOlderLogsDebounce) {
      clearTimeout(isLoadingOlderLogsDebounce);
    }
    
    isLoadingOlderLogsDebounce = setTimeout(() => {
      loadOlderLogs();
    }, 300);
  }
};

// Load older logs using the since parameter
const loadOlderLogs = async () => {
  if (isLoadingOlderLogs.value || hasLoadedAllLogs.value || logs.value.length === 0) return;

  isLoadingOlderLogs.value = true;
  
  try {
    // Get the oldest timestamp we have
    const oldestLog = logs.value[0];
    if (!oldestLog || !oldestLog.timestamp) {
      hasLoadedAllLogs.value = true;
      return;
    }

    // Save current scroll position
    if (logsContainer.value) {
      scrollPositionBeforeLoad = logsContainer.value.scrollHeight - logsContainer.value.scrollTop;
    }

    // Convert timestamp to protobuf Timestamp format
    const oldestDate = new Date(oldestLog.timestamp);
    const sinceTimestamp = timestamp(oldestDate);

    // Fetch logs before this timestamp
    const response = await client.getGameServerLogs({
      gameServerId: props.gameServerId,
      limit: 500, // Load 500 older lines at a time
      since: sinceTimestamp,
    });

    if (response.lines && response.lines.length > 0) {
      // Parse and add older logs to the beginning
      const olderLogs = response.lines
        .filter((line) => {
          const lineText = line.line || '';
          return !isEmptyOrWhitespace(lineText);
        })
        .map((line) => {
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

      // Prepend older logs (they should be in chronological order, oldest first)
      logs.value = [...olderLogs, ...logs.value];

      // Restore scroll position after DOM update
      await nextTick();
      if (logsContainer.value) {
        const newScrollHeight = logsContainer.value.scrollHeight;
        logsContainer.value.scrollTop = newScrollHeight - scrollPositionBeforeLoad;
      }

      // If we got fewer logs than requested, we've reached the beginning
      if (response.lines.length < 500) {
        hasLoadedAllLogs.value = true;
      }
    } else {
      // No more logs available
      hasLoadedAllLogs.value = true;
    }
  } catch (err: any) {
    console.error("Failed to load older logs:", err);
    // Don't set hasLoadedAllLogs on error - user can try again
  } finally {
    isLoadingOlderLogs.value = false;
  }
};

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
  const delay = Math.min(RECONNECT_DELAY * Math.pow(2, reconnectAttempts - 1), 30000);
  
  reconnectTimeout = setTimeout(async () => {
    if (!isFollowing.value) {
      return;
    }
    console.log(`Attempting to reconnect (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})...`);
    error.value = `Reconnecting... (attempt ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`;
    await startFollowing();
  }, delay);
};

const startFollowing = async () => {
  if (isFollowing.value && isConnected.value) return;
  
  if (reconnectTimeout) {
    clearTimeout(reconnectTimeout);
    reconnectTimeout = null;
  }
  
  isLoading.value = true;
  error.value = null;
  isFollowing.value = true;
  hasLoadedAllLogs.value = false; // Reset when reconnecting

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
          const existingLogs = logs.value.map(l => l.line);
          const newLogs = response.lines
            .filter((line) => {
              const lineText = line.line || '';
              return !isEmptyOrWhitespace(lineText) && !existingLogs.includes(lineText);
            })
            .map((line) => {
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
        console.warn("Failed to fetch initial logs:", err);
      }
    }

    // Start streaming logs
    streamController = new AbortController();
    logStream = client.streamGameServerLogs(
      {
        gameServerId: props.gameServerId,
        follow: true,
        tail: tailLines.value,
      },
      { signal: streamController.signal }
    );

    isConnected.value = true;
    isLoading.value = false;
    reconnectAttempts = 0;
    error.value = null;

    // Handle stream messages
    for await (const logLine of logStream) {
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
    
    if (isFollowing.value && !streamController.signal.aborted) {
      isConnected.value = false;
      scheduleReconnect();
    }
  } catch (err: any) {
    const isAbortError = 
      err.name === "AbortError" || 
      err.message?.toLowerCase().includes("aborted") ||
      err.message?.toLowerCase().includes("canceled") ||
      err.message?.toLowerCase().includes("cancelled");

    if (isAbortError) {
      return;
    }

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
  hasLoadedAllLogs.value = false;
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
  
  terminalOutputBuffer += text;
  
  const lines = terminalOutputBuffer.split(/\r?\n/);
  
  if (terminalOutputBuffer.endsWith("\n") || terminalOutputBuffer.endsWith("\r")) {
    terminalOutputBuffer = "";
  } else {
    terminalOutputBuffer = lines.pop() || "";
  }
  
  for (const line of lines) {
    if (isEmptyOrWhitespace(line)) {
      continue;
    }
    
    logs.value.push({
      line: line,
      timestamp: new Date().toISOString(),
      level: undefined,
    });
    
    if (logs.value.length > 10000) {
      logs.value = logs.value.slice(-10000);
    }
  }
  
  if (terminalOutputBuffer.length > 1000) {
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
  startFollowing();
});

onUnmounted(() => {
  stopFollowing();
  if (isLoadingOlderLogsDebounce) {
    clearTimeout(isLoadingOlderLogsDebounce);
  }
});

watch(() => props.gameServerId, () => {
  if (isFollowing.value) {
    stopFollowing();
    nextTick(() => {
      startFollowing();
    });
  }
});
</script>

<style scoped>
.logs-container-wrapper {
  position: relative;
}

.logs-loading-indicator {
  padding: 0.5rem;
  background: var(--oui-surface-muted, rgba(255, 255, 255, 0.05));
  border-radius: 0.375rem;
  margin-bottom: 0.5rem;
}

.logs-end-indicator {
  padding: 0.5rem;
  background: var(--oui-surface-muted, rgba(255, 255, 255, 0.05));
  border-radius: 0.375rem;
  margin-bottom: 0.5rem;
}
</style>
