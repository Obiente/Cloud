<template>
  <OuiCardBody>
    <OuiLogs
      ref="logsComponent"
      :logs="formattedLogs"
      :is-loading="isLoading"
      :show-timestamps="showTimestamps"
      :tail-lines="tailLines"
      :show-tail-controls="true"
      :enable-ansi="true"
      :auto-scroll="true"
      empty-message="No logs available. Start following to see real-time logs."
      loading-message="Connecting..."
      title="Deployment Logs"
      @tail-change="handleTailChange"
      @update:show-timestamps="showTimestamps = $event"
    >
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
    </OuiLogs>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import { useAuth } from "~/composables/useAuth";
import type { LogEntry } from "~/components/oui/Logs.vue";

interface Props {
  deploymentId: string;
  organizationId: string;
}

interface LogLine {
  line: string;
  timestamp: string;
  stderr: boolean;
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

const handleTailChange = (value: number) => {
  tailLines.value = value;
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
    const stream = await client.streamDeploymentLogs(
      {
        organizationId: effectiveOrgId.value,
        deploymentId: props.deploymentId,
        tail: tailLines.value,
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
      console.error("Log stream error:", error);
      logs.value.push({
        line: `[error] Failed to stream logs: ${error.message}`,
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
  }
};

const stopStream = () => {
  if (streamController) {
    streamController.abort();
    streamController = null;
  }
  isFollowing.value = false;
};

const restartStream = () => {
  stopStream();
  setTimeout(() => startStream(), 100);
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
  if (isFollowing.value) {
    startStream();
  }
});
</script>
