<template>
  <OuiStack gap="sm">
    <!-- Toolbar -->
    <OuiCard variant="outline">
      <OuiCardBody class="py-2! px-4!">
        <OuiFlex align="center" justify="between" gap="md" wrap="wrap">
          <!-- Left: title + service selector -->
          <OuiFlex align="center" gap="md">
            <UiSectionHeader :icon="CommandLineIcon" color="secondary" size="sm">Container Logs</UiSectionHeader>
            <ContainerSelector
              :deployment-id="props.deploymentId"
              :organization-id="effectiveOrgId"
              :model-value="selectedService"
              :show-label="false"
              :include-all-option="true"
              placeholder="All services"
              :style="{ minWidth: '160px' }"
              @change="onServiceChange"
            />
          </OuiFlex>
          <!-- Right: controls -->
          <OuiFlex align="center" gap="sm">
            <OuiInput
              v-model="searchQuery"
              size="sm"
              placeholder="Filter logs..."
              :style="{ width: '170px' }"
            >
              <template #prefix>
                <MagnifyingGlassIcon class="h-3.5 w-3.5 text-tertiary" />
              </template>
              <template v-if="searchQuery" #suffix>
                <button class="text-tertiary hover:text-primary transition-colors" @click="searchQuery = ''">
                  <XMarkIcon class="h-3.5 w-3.5" />
                </button>
              </template>
            </OuiInput>
            <OuiFlex align="center" gap="xs" class="shrink-0">
              <span
                class="h-1.5 w-1.5 rounded-full transition-colors"
                :class="isFollowing ? 'bg-success animate-pulse' : 'bg-border-strong'"
              />
              <OuiText size="xs" color="tertiary" class="whitespace-nowrap">{{ isFollowing ? 'Live' : 'Stopped' }}</OuiText>
            </OuiFlex>
            <OuiButton variant="ghost" size="sm" class="whitespace-nowrap shrink-0" @click="toggleFollow">
              <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': isLoading }" />
              {{ isFollowing ? 'Stop' : 'Follow' }}
            </OuiButton>
            <OuiButton variant="ghost" size="sm" :disabled="logs.length === 0" @click="clearLogs">
              Clear
            </OuiButton>
            <OuiMenu>
              <template #trigger>
                <OuiButton variant="ghost" size="sm">
                  <EllipsisVerticalIcon class="h-3.5 w-3.5" />
                </OuiButton>
              </template>
              <OuiMenuItem>
                <OuiCheckbox v-model="showTimestamps" label="Show timestamps" @click.stop />
              </OuiMenuItem>
              <OuiMenuSeparator />
              <OuiMenuItem>
                <label class="flex items-center gap-2 px-1 py-1 text-sm cursor-pointer">
                  <span class="text-tertiary">Tail:</span>
                  <OuiInput
                    :model-value="tailLines.toString()"
                    type="number"
                    min="10"
                    max="10000"
                    size="sm"
                    :style="{ width: '80px' }"
                    @update:model-value="handleTailChange"
                    @click.stop
                  />
                </label>
              </OuiMenuItem>
            </OuiMenu>
          </OuiFlex>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <!-- Log viewer -->
    <OuiLogs
      ref="logsComponent"
      :logs="filteredLogs"
      :is-loading="isLoading"
      :show-timestamps="showTimestamps"
      :enable-ansi="true"
      :auto-scroll="true"
      empty-message="No logs yet — click Follow to start streaming."
      loading-message="Connecting to log stream…"
    />

    <!-- Footer -->
    <OuiFlex justify="end" align="center">
      <OuiText size="xs" color="tertiary">
        {{ logs.length }} line{{ logs.length !== 1 ? 's' : '' }}<template v-if="searchQuery && filteredLogs.length !== logs.length"> &middot; {{ filteredLogs.length }} matching</template>
      </OuiText>
    </OuiFlex>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, shallowRef, computed, watch, onMounted, onUnmounted } from "vue";
import { ArrowPathIcon, EllipsisVerticalIcon, CommandLineIcon, MagnifyingGlassIcon, XMarkIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService, type StreamDeploymentLogsRequest } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import { useAuth } from "~/composables/useAuth";
import type { LogEntry } from "~/components/oui/Logs.vue";
import ContainerSelector from "./ContainerSelector.vue";

interface Props {
  deploymentId: string;
  organizationId: string;
}

interface LogLine {
  line: string;
  timestamp: string;
  stderr: boolean;
  logLevel?: number; // LogLevel enum value from proto
}

const props = defineProps<Props>();

const orgsStore = useOrganizationsStore();
const effectiveOrgId = computed(
  () => props.organizationId || orgsStore.currentOrgId || ""
);
const auth = useAuth();

const client = useConnectClient(DeploymentService);
const logs = shallowRef<LogLine[]>([]);
const isFollowing = ref(false);
const isLoading = ref(false);
const tailLines = ref(200);
const showTimestamps = ref(true);
const logsComponent = ref<any>(null);
let streamController: AbortController | null = null;
let isAborting = false; // Track if we're intentionally aborting
let streamRunId = 0;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let flushTimer: ReturnType<typeof requestAnimationFrame> | null = null;
let pendingLogs: LogLine[] = [];
const seenLogKeys = new Set<string>();
const maxLogLines = 10000;

// Search / filter
const searchQuery = ref("");
const filteredLogs = computed<LogEntry[]>(() => {
  if (!searchQuery.value) return formattedLogs.value;
  const q = searchQuery.value.toLowerCase();
  return formattedLogs.value.filter((l) =>
    (l.line || l.content || l.data || "").toLowerCase().includes(q)
  );
});

// Service selection
const selectedService = ref<string>(""); // Empty string means "first container" (default)
const selectedContainer = ref<{ containerId: string; serviceName?: string } | null>(null);

// Convert LogLevel enum to string for OuiLogs
const logLevelToString = (level?: number): "error" | "warning" | "info" | "debug" | "trace" => {
  if (!level) return "info";
  // LogLevel enum: 0=UNSPECIFIED, 1=TRACE, 2=DEBUG, 3=INFO, 4=WARN, 5=ERROR
  switch (level) {
    case 5: return "error";
    case 4: return "warning";
    case 3: return "info";
    case 2: return "debug";
    case 1: return "trace";
    default: return "info";
  }
};

// Convert internal logs to LogEntry format for OuiLogs component
const formattedLogs = computed<LogEntry[]>(() => {
  return logs.value.map((log, idx) => {
    // Determine log level - use detected level if available, otherwise infer from stderr
    let level: "error" | "warning" | "info" | "debug" | "trace" | undefined;
    if (log.logLevel !== undefined) {
      level = logLevelToString(log.logLevel);
    } else {
      level = log.stderr ? "error" : "info";
    }
    
    return {
      id: idx,
      line: log.line,
      timestamp: log.timestamp,
      stderr: log.stderr,
      level,
    };
  });
});

const handleTailChange = (value: string | number) => {
  const numValue = typeof value === "string" ? parseInt(value, 10) : value;
  if (isNaN(numValue) || numValue < 10) {
    tailLines.value = 10;
  } else if (numValue > 10000) {
    tailLines.value = 10000;
  } else {
    tailLines.value = numValue;
  }
  if (isFollowing.value) {
    restartStream();
  }
};

const clearLogs = () => {
  logs.value = [];
  pendingLogs = [];
  seenLogKeys.clear();
};

const toggleFollow = () => {
  if (isFollowing.value) {
    stopStream();
  } else {
    startStream(true);
  }
};

const logKey = (log: LogLine) => `${log.timestamp}|${log.stderr ? "1" : "0"}|${log.line}`;

const flushPendingLogs = () => {
  flushTimer = null;
  if (pendingLogs.length === 0) return;

  const batch = pendingLogs;
  pendingLogs = [];

  let nextLogs = logs.value.length > 0 ? logs.value.slice() : [];
  let outOfOrder = false;
  const lastExisting = nextLogs[nextLogs.length - 1];
  const lastExistingTime = lastExisting ? Date.parse(lastExisting.timestamp) : Number.NEGATIVE_INFINITY;

  for (const log of batch) {
    const key = logKey(log);
    if (seenLogKeys.has(key)) continue;
    seenLogKeys.add(key);
    if (Date.parse(log.timestamp) < lastExistingTime) {
      outOfOrder = true;
    }
    nextLogs.push(log);
  }

  if (outOfOrder) {
    nextLogs.sort((a, b) => Date.parse(a.timestamp) - Date.parse(b.timestamp));
  }

  if (nextLogs.length > maxLogLines) {
    const removed = nextLogs.length - maxLogLines;
    for (const log of nextLogs.slice(0, removed)) {
      seenLogKeys.delete(logKey(log));
    }
    nextLogs = nextLogs.slice(-maxLogLines);
  }

  logs.value = nextLogs;
};

const queueLog = (log: LogLine) => {
  pendingLogs.push(log);
  if (flushTimer === null) {
    flushTimer = requestAnimationFrame(flushPendingLogs);
  }
};

const startStream = async (resetLogs = false) => {
  if (isFollowing.value) return;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  const runId = ++streamRunId;
  isFollowing.value = true;
  isLoading.value = true;
  isAborting = false;
  if (resetLogs) {
    clearLogs();
  }

  let hasReceivedLogs = false;
  let reconnectAllowed = true;

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

    streamController = new AbortController();
    
    // Build request with service/container filter
    const request: Partial<StreamDeploymentLogsRequest> = {
      organizationId: effectiveOrgId.value,
      deploymentId: props.deploymentId,
      tail: tailLines.value,
    };
    
    // Add service filter if a specific service is selected
    if (selectedContainer.value) {
      if (selectedContainer.value.serviceName) {
        request.serviceName = selectedContainer.value.serviceName;
      } else {
        request.containerId = selectedContainer.value.containerId;
      }
    }
    
    const stream = await client.streamDeploymentLogs(
      request as StreamDeploymentLogsRequest,
      { signal: streamController.signal }
    );

    isLoading.value = false;

    for await (const update of stream) {
      if (runId !== streamRunId) return;
      if (update.line) {
        hasReceivedLogs = true;
        queueLog({
          line: update.line,
          timestamp: update.timestamp
            ? new Date(
                Number(update.timestamp.seconds) * 1000 +
                  Number(update.timestamp.nanos || 0) / 1e6
              ).toISOString()
            : new Date().toISOString(),
          stderr: update.stderr || false,
          logLevel: update.logLevel || undefined, // Include log level from proto
        });
      }
    }
  } catch (error: unknown) {
    // Suppress all abort-related errors - they're intentional when switching services
    const isAbortError = 
      (error as any).name === "AbortError" || 
      (error as Error).message?.toLowerCase().includes("aborted") ||
      (error as Error).message?.toLowerCase().includes("canceled") ||
      (error as Error).message?.toLowerCase().includes("cancelled") ||
      isAborting;

    if (isAbortError) {
      // User intentionally cancelled or we're switching services - no error needed
      return;
    }

    const errorMessage = (error as Error).message || "";
    if (errorMessage.toLowerCase().includes("authentication required")) {
      reconnectAllowed = false;
    }

    // Suppress benign errors if we successfully received logs:
    // - "missing trailer": Connect/gRPC-Web quirk where streams end without HTTP trailers
    // - "missing EndStreamResponse": Connect/gRPC-Web quirk where streams end without explicit end marker
    // - "invalid UTF-8": Docker logs can contain binary data (now sanitized server-side)
    const isBenignError =
      (error as Error).message?.toLowerCase().includes("missing trailer") ||
      (error as Error).message?.toLowerCase().includes("trailer") ||
      (error as Error).message?.toLowerCase().includes("missing endstreamresponse") ||
      (error as Error).message?.toLowerCase().includes("endstreamresponse") ||
      (error as Error).message?.toLowerCase().includes("invalid utf-8") ||
      (error as Error).message?.toLowerCase().includes("marshal message") ||
      (error as any).code === "unknown";

    // Only show error if it's not a benign error after successful streaming,
    // or if it's a real error before receiving any logs
    if (!isBenignError || !hasReceivedLogs) {
      console.error("Log stream error:", error);
      queueLog({
        line: `[error] Failed to stream logs: ${errorMessage}`,
        timestamp: new Date().toISOString(),
        stderr: true,
        logLevel: 5, // ERROR
      });
    } else {
      // Log to console but don't show to user - this is expected behavior
      console.debug("Stream ended with benign error:", errorMessage);
    }
  } finally {
    if (runId === streamRunId) {
      isLoading.value = false;
      isFollowing.value = false;
      streamController = null;
      if (!isAborting && reconnectAllowed) {
        reconnectTimer = setTimeout(() => {
          startStream(false);
        }, hasReceivedLogs ? 1000 : 3000);
      }
    }
    isAborting = false; // Reset abort flag
  }
};

const stopStream = () => {
  streamRunId++;
  if (reconnectTimer) {
    clearTimeout(reconnectTimer);
    reconnectTimer = null;
  }
  if (streamController) {
    isAborting = true; // Mark that we're intentionally aborting
    streamController.abort();
    streamController = null;
  }
  isFollowing.value = false;
};

const restartStream = () => {
  stopStream();
  setTimeout(() => startStream(false), 100);
};

// Handle service change
const onServiceChange = (container: { containerId: string; serviceName?: string } | null) => {
  selectedContainer.value = container;
  // Update selectedService to match the container selection
  if (container) {
    selectedService.value = container.serviceName || container.containerId;
  } else {
    selectedService.value = "";
  }
  
  // Stop current stream (this will set isAborting = true)
  stopStream();
  
  // Wait a brief moment to let the abort error be handled silently
  // Clear logs after a short delay to avoid race conditions
  setTimeout(() => {
    clearLogs();
    
    // Always auto-follow when service changes
    startStream(false);
  }, 50);
};

onMounted(() => {
  // Auto-start following if deployment is running
  startStream(true);
});

onUnmounted(() => {
  stopStream();
  if (flushTimer !== null) {
    cancelAnimationFrame(flushTimer);
    flushTimer = null;
  }
});

watch(() => props.deploymentId, () => {
  const wasFollowing = isFollowing.value;
  stopStream();
  clearLogs();
  selectedService.value = ""; // Reset to default
  selectedContainer.value = null;
  if (wasFollowing) {
    startStream(false);
  }
});
</script>
