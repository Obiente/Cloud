<template>
  <OuiCardBody>
    <OuiStack gap="md">
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="md" weight="semibold">Deployment Logs</OuiText>
        <OuiFlex align="center" gap="sm">
          <OuiButton
            variant="ghost"
            size="sm"
            @click="toggleFollow"
            :class="{ 'text-primary': isFollowing }"
          >
            <ArrowPathIcon class="h-4 w-4 mr-1" :class="{ 'animate-spin': isFollowing }" />
            {{ isFollowing ? "Following" : "Follow" }}
          </OuiButton>
          <OuiButton variant="ghost" size="sm" @click="clearLogs">
            Clear
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <div
        ref="logContainer"
        class="bg-black text-green-400 p-4 rounded-xl text-xs font-mono overflow-auto"
        :style="{ height: '600px' }"
      >
        <div v-if="logs.length === 0 && !isLoading" class="text-gray-500">
          No logs available. Start following to see real-time logs.
        </div>
        <div v-else-if="isLoading" class="text-gray-500">Connecting...</div>
        <div v-else>
          <div
            v-for="(log, idx) in logs"
            :key="idx"
            :class="[
              'log-line',
              log.stderr ? 'text-red-400' : 'text-green-400',
            ]"
          >
            <span v-if="showTimestamps" class="text-gray-500 mr-2">{{
              formatTimestamp(log.timestamp)
            }}</span>
            {{ log.line }}
          </div>
        </div>
      </div>

      <OuiFlex align="center" gap="sm" justify="between">
        <OuiFlex align="center" gap="sm">
          <OuiText size="xs" color="secondary">Tail:</OuiText>
          <OuiInput
            v-model.number="tailLines"
            type="number"
            min="10"
            max="1000"
            class="w-20"
            size="sm"
            @update:model-value="handleTailChange"
          />
        </OuiFlex>
        <OuiCheckbox v-model="showTimestamps" label="Show timestamps" />
      </OuiFlex>
    </OuiStack>
  </OuiCardBody>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from "vue";
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";

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

const client = useConnectClient(DeploymentService);
const logs = ref<LogLine[]>([]);
const isFollowing = ref(false);
const isLoading = ref(false);
const tailLines = ref(200);
const showTimestamps = ref(true);
const logContainer = ref<HTMLElement | null>(null);
let streamController: AbortController | null = null;

const handleTailChange = () => {
  if (isFollowing.value) {
    restartStream();
  }
};

const clearLogs = () => {
  logs.value = [];
};

const formatTimestamp = (ts: string) => {
  return new Date(ts).toLocaleTimeString();
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

  try {
    streamController = new AbortController();
    const stream = await client.streamDeploymentLogs(
      {
        organizationId: props.organizationId,
        deploymentId: props.deploymentId,
        tail: tailLines.value,
      },
      { signal: streamController.signal }
    );

    isLoading.value = false;

    for await (const update of stream) {
      if (update.line) {
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
        await nextTick();
        scrollToBottom();
      }
    }
  } catch (error: any) {
    if (error.name !== "AbortError") {
      console.error("Log stream error:", error);
      logs.value.push({
        line: `[error] Failed to stream logs: ${error.message}`,
        timestamp: new Date().toISOString(),
        stderr: true,
      });
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

const scrollToBottom = () => {
  if (logContainer.value) {
    logContainer.value.scrollTop = logContainer.value.scrollHeight;
  }
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

<style scoped>
.log-line {
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
