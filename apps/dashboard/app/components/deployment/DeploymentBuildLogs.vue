<template>
    <OuiStack gap="md">
      <!-- Show latest build info and link if no active build -->
      <OuiCard v-if="!isStreaming && latestBuild && !isLoading" variant="outline">
        <OuiCardBody>
          <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
            <OuiStack gap="xs">
              <OuiFlex align="center" gap="sm">
                <OuiText as="h4" size="sm" weight="semibold">
                  Latest Build #{{ latestBuild.buildNumber }}
                </OuiText>
                <OuiBadge
                  :variant="getBuildStatusVariant(latestBuild.status)"
                  size="xs"
                >
                  {{ getBuildStatusLabel(latestBuild.status) }}
                </OuiBadge>
              </OuiFlex>
              <OuiText size="xs" color="secondary">
                <OuiRelativeTime
                  :value="latestBuild.startedAt ? date(latestBuild.startedAt) : undefined"
                  :style="'short'"
                />
              </OuiText>
            </OuiStack>
            <NuxtLink
              :to="`/deployments/${props.deploymentId}?tab=builds`"
              class="text-primary hover:underline"
            >
              <OuiButton variant="ghost" size="sm">
                View All Builds
                <ArrowTopRightOnSquareIcon class="h-4 w-4 ml-1" />
              </OuiButton>
            </NuxtLink>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <OuiLogs
        :logs="formattedLogs"
        :is-loading="isLoading"
        :show-timestamps="showTimestamps"
        :enable-ansi="true"
        :auto-scroll="isStreaming"
        empty-message="No build logs available. Build logs will appear here when a deployment is triggered."
        loading-message="Connecting to build logs..."
        title="Build Logs"
        @update:show-timestamps="showTimestamps = $event"
      >
        <template #title>
          <OuiFlex align="center" gap="sm">
            <OuiText as="h3" size="md" weight="semibold">Build Logs</OuiText>
            <OuiBadge v-if="isStreaming" color="info" size="sm">
              <span class="animate-pulse">●</span> Live
            </OuiBadge>
            <OuiBadge v-else-if="latestBuild" color="secondary" size="sm">
              Historical
            </OuiBadge>
          </OuiFlex>
        </template>
        <template #actions>
          <OuiButton
            v-if="isStreaming"
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
        </template>
        <template #footer>
          <div class="oui-logs-controls">
            <OuiText size="xs" color="secondary" class="logs-count">
              {{ logs.length }} line{{ logs.length !== 1 ? 's' : '' }}
            </OuiText>
          </div>
        </template>
      </OuiLogs>
    </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { ArrowPathIcon, ArrowTopRightOnSquareIcon } from "@heroicons/vue/24/outline";
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
const logs = ref<LogLine[]>([]);
const isFollowing = ref(false);
const isStreaming = ref(false);
const isLoading = ref(false);
const showTimestamps = ref(true);
const latestBuild = ref<Build | null>(null);
let streamController: AbortController | null = null;

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
  if (isFollowing.value) return;
  isFollowing.value = true;
  isLoading.value = true;
  isStreaming.value = true;

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
    // Suppress benign stream closure errors if we successfully received logs
    // These are Connect/gRPC-Web quirks where streams can end without
    // proper HTTP trailers or end-of-stream markers, but the stream itself worked correctly
    const isBenignStreamError =
      error.message?.toLowerCase().includes("missing trailer") ||
      error.message?.toLowerCase().includes("trailer") ||
      error.message?.toLowerCase().includes("missing endstreamresponse") ||
      error.message?.toLowerCase().includes("endstreamresponse") ||
      error.code === "unknown";

    if (error.name === "AbortError") {
      // User intentionally cancelled - no error needed
      return;
    }

    // Only show error if it's not a benign stream error after successful streaming,
    // or if it's a real error before receiving any logs
    if (!isBenignStreamError || !hasReceivedLogs) {
      console.error("Build log stream error:", error);
      logs.value.push({
        line: `[error] Failed to stream build logs: ${error.message}`,
        timestamp: new Date().toISOString(),
        stderr: true,
        logLevel: 5, // ERROR
      });
    } else {
      // Log to console but don't show to user - this is expected behavior
      console.debug("Stream ended with benign error (expected):", error.message);
    }
  } finally {
    isLoading.value = false;
    isFollowing.value = false;
    isStreaming.value = false;
  }
};

const stopStream = () => {
  if (streamController) {
    streamController.abort();
    streamController = null;
  }
  isFollowing.value = false;
  isStreaming.value = false;
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
  
  // Don't load if we're streaming (active build)
  if (isStreaming.value || props.autoStart) return;
  
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
  } catch (error: any) {
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

