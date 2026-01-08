<template>
    <!-- Service Selector -->
    <div class="mb-4">
      <ContainerSelector
        :deployment-id="props.deploymentId"
        :organization-id="effectiveOrgId"
        :model-value="selectedService"
        :show-selected-info="true"
        selected-info-text="Viewing logs for"
        @change="onServiceChange"
      />
    </div>

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
      title="Deployment Logs"
    >
      <template #actions>
        <OuiFlex gap="sm" align="center">
          <OuiButton
            variant="ghost"
            size="sm"
            @click="toggleFollow"
            :class="{ 'text-primary': isFollowing }"
          >
            <ArrowPathIcon
              class="h-4 w-4 mr-1"
              :class="{ 'animate-spin': isFollowing }"
            />
            {{ isFollowing ? "Following" : "Follow" }}
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
      </template>
    </OuiLogs>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { ArrowPathIcon, EllipsisVerticalIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
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
const logs = ref<LogLine[]>([]);
const isFollowing = ref(false);
const isLoading = ref(false);
const tailLines = ref(200);
const showTimestamps = ref(true);
const logsComponent = ref<any>(null);
let streamController: AbortController | null = null;
let isAborting = false; // Track if we're intentionally aborting

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
};

const toggleFollow = () => {
  if (isFollowing.value) {
    stopStream();
  } else {
    startStream();
  }
};

const startStream = async () => {
  if (isFollowing.value) return;
  isFollowing.value = true;
  isLoading.value = true;
  logs.value = [];

  let hasReceivedLogs = false;

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
    const request: any = {
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
      request,
      { signal: streamController.signal }
    );

    isLoading.value = false;

    for await (const update of stream) {
      if (update.line) {
        hasReceivedLogs = true;
          logs.value.push({
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
  } catch (error: any) {
    // Suppress all abort-related errors - they're intentional when switching services
    const isAbortError = 
      error.name === "AbortError" || 
      error.message?.toLowerCase().includes("aborted") ||
      error.message?.toLowerCase().includes("canceled") ||
      error.message?.toLowerCase().includes("cancelled") ||
      isAborting;

    if (isAbortError) {
      // User intentionally cancelled or we're switching services - no error needed
      return;
    }

    // Suppress benign errors if we successfully received logs:
    // - "missing trailer": Connect/gRPC-Web quirk where streams end without HTTP trailers
    // - "missing EndStreamResponse": Connect/gRPC-Web quirk where streams end without explicit end marker
    // - "invalid UTF-8": Docker logs can contain binary data (now sanitized server-side)
    const isBenignError =
      error.message?.toLowerCase().includes("missing trailer") ||
      error.message?.toLowerCase().includes("trailer") ||
      error.message?.toLowerCase().includes("missing endstreamresponse") ||
      error.message?.toLowerCase().includes("endstreamresponse") ||
      error.message?.toLowerCase().includes("invalid utf-8") ||
      error.message?.toLowerCase().includes("marshal message") ||
      error.code === "unknown";

    // Only show error if it's not a benign error after successful streaming,
    // or if it's a real error before receiving any logs
    if (!isBenignError || !hasReceivedLogs) {
      console.error("Log stream error:", error);
      logs.value.push({
        line: `[error] Failed to stream logs: ${error.message}`,
        timestamp: new Date().toISOString(),
        stderr: true,
        logLevel: 5, // ERROR
      });
    } else {
      // Log to console but don't show to user - this is expected behavior
      console.debug("Stream ended with benign error:", error.message);
    }
  } finally {
    isLoading.value = false;
    isFollowing.value = false;
    isAborting = false; // Reset abort flag
  }
};

const stopStream = () => {
  if (streamController) {
    isAborting = true; // Mark that we're intentionally aborting
    streamController.abort();
    streamController = null;
  }
  isFollowing.value = false;
};

const restartStream = () => {
  stopStream();
  setTimeout(() => startStream(), 100);
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
    logs.value = [];
    
    // Always auto-follow when service changes
    startStream();
  }, 50);
};

onMounted(() => {
  // Auto-start following if deployment is running
  startStream();
});

onUnmounted(() => {
  stopStream();
});

watch(() => props.deploymentId, () => {
  stopStream();
  logs.value = [];
  selectedService.value = ""; // Reset to default
  selectedContainer.value = null;
  if (isFollowing.value) {
    startStream();
  }
});
</script>
