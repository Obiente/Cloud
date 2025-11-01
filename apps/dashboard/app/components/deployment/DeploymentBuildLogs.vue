<template>
  <OuiCardBody>
    <OuiLogs
      :logs="formattedLogs"
      :is-loading="isLoading"
      :show-timestamps="showTimestamps"
      :enable-ansi="true"
      :auto-scroll="true"
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
        </OuiFlex>
      </template>
      <template #actions>
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
      </template>
      <template #footer>
        <div class="oui-logs-controls">
          <OuiText size="xs" color="secondary" class="logs-count">
            {{ logs.length }} line{{ logs.length !== 1 ? 's' : '' }}
          </OuiText>
        </div>
      </template>
    </OuiLogs>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import type { LogEntry } from "~/components/oui/Logs.vue";

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
let streamController: AbortController | null = null;

interface LogLine {
  line: string;
  timestamp: string;
  stderr: boolean;
}

// Convert internal logs to LogEntry format for OuiLogs component
const formattedLogs = computed<LogEntry[]>(() => {
  return logs.value.map((log, idx) => ({
    id: idx,
    line: log.line,
    timestamp: log.timestamp,
    stderr: log.stderr,
    level: log.stderr ? "error" : "info",
  }));
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
        });
      }
    }
  } catch (error: any) {
    // Suppress "missing trailer" errors if we successfully received logs
    // This is a benign Connect/gRPC-Web quirk where streams can end without
    // proper HTTP trailers, but the stream itself worked correctly
    const isMissingTrailerError =
      error.message?.toLowerCase().includes("missing trailer") ||
      error.message?.toLowerCase().includes("trailer") ||
      error.code === "unknown";

    if (error.name === "AbortError") {
      // User intentionally cancelled - no error needed
      return;
    }

    // Only show error if it's not a missing trailer error after successful streaming,
    // or if it's a real error before receiving any logs
    if (!isMissingTrailerError || !hasReceivedLogs) {
      console.error("Build log stream error:", error);
      logs.value.push({
        line: `[error] Failed to stream build logs: ${error.message}`,
        timestamp: new Date().toISOString(),
        stderr: true,
      });
    } else {
      // Log to console but don't show to user - this is expected behavior
      console.debug("Stream ended with missing trailer (benign):", error.message);
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

onMounted(() => {
  // Auto-start following if autoStart is enabled
  if (props.autoStart) {
    startStream();
  }
});

onUnmounted(() => {
  stopStream();
});

watch(() => props.deploymentId, () => {
  stopStream();
  logs.value = [];
  if (props.autoStart) {
    startStream();
  }
});
</script>

<style scoped>
.logs-count {
  white-space: nowrap;
  user-select: none;
}
</style>

