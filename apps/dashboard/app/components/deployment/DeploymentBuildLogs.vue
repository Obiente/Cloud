<template>
  <OuiStack gap="sm">
    <!-- Historical build info bar -->
    <OuiCard v-if="!isStreaming && latestBuild && !isLoading" variant="outline">
      <OuiCardBody class="py-2.5! px-4!">
        <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
          <OuiFlex align="center" gap="sm">
            <OuiText size="sm" weight="medium">Build #{{ latestBuild.buildNumber }}</OuiText>
            <OuiBadge :variant="getBuildStatusVariant(latestBuild.status)" size="xs">
              {{ getBuildStatusLabel(latestBuild.status) }}
            </OuiBadge>
            <OuiText size="xs" color="tertiary">
              <OuiRelativeTime
                :value="latestBuild.startedAt ? date(latestBuild.startedAt) : undefined"
                :style="'short'"
              />
            </OuiText>
          </OuiFlex>
          <NuxtLink :to="`/deployments/${props.deploymentId}?tab=builds`">
            <OuiButton variant="ghost" size="sm">
              All Builds
              <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5 ml-1" />
            </OuiButton>
          </NuxtLink>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <!-- Toolbar -->
    <OuiCard variant="outline">
      <OuiCardBody class="py-2! px-4!">
        <OuiFlex align="center" justify="between" gap="md">
          <!-- Left: title + live/historical badge -->
          <OuiFlex align="center" gap="sm">
            <UiSectionHeader :icon="BeakerIcon" color="warning" size="sm">Build Logs</UiSectionHeader>
            <OuiBadge v-if="isStreaming" color="info" size="xs">
              <span class="animate-pulse mr-1">●</span>Live
            </OuiBadge>
            <OuiBadge v-else-if="latestBuild" color="secondary" size="xs">Historical</OuiBadge>
          </OuiFlex>
          <!-- Right: controls -->
          <OuiFlex align="center" gap="sm">
            <OuiFlex v-if="isStreaming" align="center" gap="xs" class="shrink-0">
              <span
                class="h-1.5 w-1.5 rounded-full transition-colors"
                :class="isFollowing ? 'bg-success animate-pulse' : 'bg-border-strong'"
              />
              <OuiText size="xs" color="tertiary" class="whitespace-nowrap">{{ isFollowing ? 'Following' : 'Paused' }}</OuiText>
            </OuiFlex>
            <OuiButton
              v-if="isStreaming"
              variant="ghost"
              size="sm"
              class="whitespace-nowrap shrink-0"
              @click="toggleFollow"
            >
              <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': isFollowing }" />
              {{ isFollowing ? 'Stop' : 'Follow' }}
            </OuiButton>
            <OuiButton variant="ghost" size="sm" :disabled="logs.length === 0" @click="clearLogs">
              Clear
            </OuiButton>
          </OuiFlex>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <!-- Log viewer -->
    <OuiLogs
      :logs="formattedLogs"
      :is-loading="isLoading"
      :show-timestamps="showTimestamps"
      :enable-ansi="true"
      :auto-scroll="isStreaming && isFollowing"
      empty-message="No build logs available. Logs appear here when a build is triggered."
      loading-message="Connecting to build log stream…"
    />

    <!-- Footer -->
    <OuiFlex justify="end" align="center">
      <OuiText size="xs" color="tertiary">
        {{ logs.length }} line{{ logs.length !== 1 ? 's' : '' }}
      </OuiText>
    </OuiFlex>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick, shallowRef } from "vue";
import { ArrowPathIcon, ArrowTopRightOnSquareIcon, BeakerIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService, BuildStatus, type Build } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import type { LogEntry } from "~/components/oui/Logs.vue";
import { date } from "@obiente/proto/utils";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";

interface Props {
  deploymentId: string;
  organizationId: string;
  autoStart?: boolean; // Automatically start streaming when component mounts
}

const props = withDefaults(defineProps<Props>(), {
  autoStart: false,
});

const orgsStore = useOrganizationsStore();
const effectiveOrgId = computed(
  () => props.organizationId || orgsStore.currentOrgId || ""
);

const client = useConnectClient(DeploymentService);
// Use shallowRef for logs to prevent deep reactivity issues during revalidation
const logs = shallowRef<LogLine[]>([]);
const isFollowing = ref(false);
const isStreaming = ref(false);
const isLoading = ref(false);
const showTimestamps = ref(true);
const latestBuild = ref<Build | null>(null);
let streamController: AbortController | null = null;
// Track if we're in the middle of streaming to prevent restart during revalidation
const isStreamingInternal = ref(false);

interface LogLine {
  line: string;
  timestamp: string;
  stderr: boolean;
  logLevel?: number; // LogLevel enum value from proto
}

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
  // Prevent starting a new stream if we're already streaming (e.g., during revalidation)
  if (isFollowing.value || isStreamingInternal.value) return;
  
  isFollowing.value = true;
  isLoading.value = true;
  isStreaming.value = true;
  isStreamingInternal.value = true;

  // Clear existing logs when starting a new stream to prevent mixing old and new logs
  logs.value = [];
  latestBuild.value = null;

  let hasReceivedLogs = false;

  try {
    streamController = new AbortController();
    const stream = await client.streamBuildLogs(
      {
        organizationId: effectiveOrgId.value,
        deploymentId: props.deploymentId,
      },
      { signal: streamController.signal }
    );

    isLoading.value = false;

    for await (const update of stream) {
      // Check if stream was aborted (e.g., during revalidation)
      if (streamController?.signal.aborted) {
        break;
      }
      
      if (update.line) {
        hasReceivedLogs = true;
        // Create a new array to trigger reactivity without deep watching
        const newLogs = [...logs.value];
        newLogs.push({
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
        logs.value = newLogs;
      }
    }
  } catch (error: unknown) {
    // Suppress benign stream closure errors if we successfully received logs
    // These are Connect/gRPC-Web quirks where streams can end without
    // proper HTTP trailers or end-of-stream markers, but the stream itself worked correctly
    const isBenignStreamError =
      (error as Error).message?.toLowerCase().includes("missing trailer") ||
      (error as Error).message?.toLowerCase().includes("trailer") ||
      (error as Error).message?.toLowerCase().includes("missing endstreamresponse") ||
      (error as Error).message?.toLowerCase().includes("endstreamresponse") ||
      (error as any).code === "unknown";

    if ((error as any).name === "AbortError") {
      // User intentionally cancelled - no error needed
      return;
    }

    // Only show error if it's not a benign stream error after successful streaming,
    // or if it's a real error before receiving any logs
    if (!isBenignStreamError || !hasReceivedLogs) {
      console.error("Build log stream error:", error);
      logs.value.push({
        line: `[error] Failed to stream build logs: ${(error as Error).message}`,
        timestamp: new Date().toISOString(),
        stderr: true,
        logLevel: 5, // ERROR
      });
    } else {
      // Log to console but don't show to user - this is expected behavior
      console.debug("Stream ended with benign error (expected):", (error as Error).message);
    }
  } finally {
    isLoading.value = false;
    isFollowing.value = false;
    isStreaming.value = false;
    isStreamingInternal.value = false;
  }
};

const stopStream = () => {
  if (streamController) {
    streamController.abort();
    streamController = null;
  }
  isFollowing.value = false;
  isStreaming.value = false;
  isStreamingInternal.value = false;
};

const restartStream = () => {
  stopStream();
  setTimeout(() => startStream(), 100);
};

// Expose methods for parent components
defineExpose({
  startStream,
  stopStream,
  clearLogs,
});

// Load latest build and its logs if no active build
const loadLatestBuildLogs = async () => {
  if (!effectiveOrgId.value || !props.deploymentId) return;
  
  // Don't load if we're streaming (active build) - this prevents reset during revalidation
  if (isStreaming.value || isStreamingInternal.value || props.autoStart) return;
  
  isLoading.value = true;
  try {
    // Get latest build
    const buildsResponse = await client.listBuilds({
      organizationId: effectiveOrgId.value,
      deploymentId: props.deploymentId,
      limit: 1,
      offset: 0,
    });
    
    if (buildsResponse.builds && buildsResponse.builds.length > 0) {
      const build = buildsResponse.builds[0];
      if (build) {
        latestBuild.value = build;
        
        // Load logs for this build
        if (build.id) {
          const logsResponse = await client.getBuildLogs({
            organizationId: effectiveOrgId.value,
            deploymentId: props.deploymentId,
            buildId: build.id,
            limit: 10000, // Large limit to get all logs
          });
          
          if (logsResponse.logs) {
            logs.value = logsResponse.logs.map((log) => ({
              line: log.line,
              timestamp: log.timestamp
                ? new Date(
                    Number(log.timestamp.seconds) * 1000 +
                      Number(log.timestamp.nanos || 0) / 1e6
                  ).toISOString()
                : new Date().toISOString(),
              stderr: log.stderr || false,
              logLevel: log.logLevel || undefined,
            }));
          }
        }
      }
    }
  } catch (error: unknown) {
    console.error("Failed to load latest build logs:", error);
    // Don't show error to user, just log it
  } finally {
    isLoading.value = false;
  }
};

// Helper functions for build status
const getBuildStatusVariant = (status: BuildStatus): "success" | "danger" | "warning" | "secondary" => {
  switch (status) {
    case BuildStatus.BUILD_SUCCESS:
      return "success";
    case BuildStatus.BUILD_FAILED:
      return "danger";
    case BuildStatus.BUILD_BUILDING:
    case BuildStatus.BUILD_PENDING:
      return "warning";
    default:
      return "secondary";
  }
};

const getBuildStatusLabel = (status: BuildStatus): string => {
  switch (status) {
    case BuildStatus.BUILD_SUCCESS:
      return "Success";
    case BuildStatus.BUILD_FAILED:
      return "Failed";
    case BuildStatus.BUILD_BUILDING:
      return "Building";
    case BuildStatus.BUILD_PENDING:
      return "Pending";
    default:
      return "Unknown";
  }
};

onMounted(() => {
  // Only start if we're not already streaming (prevents restart during revalidation)
  if (isStreamingInternal.value) return;
  
  // Auto-start following if autoStart is enabled
  if (props.autoStart) {
    startStream();
  } else {
    // Otherwise, load latest build logs
    loadLatestBuildLogs();
  }
});

onUnmounted(() => {
  stopStream();
});

watch(() => props.deploymentId, () => {
  stopStream();
  logs.value = [];
  latestBuild.value = null;
  if (props.autoStart) {
    startStream();
  } else {
    loadLatestBuildLogs();
  }
});

// Watch for autoStart changes to switch between streaming and historical logs
watch(() => props.autoStart, (newValue) => {
  if (newValue) {
    // Start streaming
    stopStream();
    logs.value = [];
    latestBuild.value = null;
    startStream();
  } else {
    // Stop streaming and load latest build
    stopStream();
    logs.value = [];
    loadLatestBuildLogs();
  }
});
</script>

<style scoped>
.logs-count {
  white-space: nowrap;
  user-select: none;
}
</style>

