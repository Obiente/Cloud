<script setup lang="ts">
// Import OUI components
import OuiText from "../components/oui/Text.vue";

definePageMeta({
  layout: false,
});

interface LogEntry {
  type: "connected" | "disconnected" | "error" | "stdout" | "stderr";
  data: string;
  timestamp: string;
}

interface ContainerListItem {
  id: string;
  names: string[];
  image: string;
  state: string;
  status: string;
  created: number;
}

const selectedContainer = ref<ContainerListItem | null>(null);
const logs = ref<LogEntry[]>([]);
const isConnected = ref(false);
const isLoading = ref(false);
const error = ref("");
const eventSource = ref<EventSource | null>(null);
const logContainer = ref<HTMLElement | null>(null);
const autoScroll = ref(true);
const availableContainers = ref<ContainerListItem[]>([]);
const isLoadingContainers = ref(false);

// Load available containers on mount
onMounted(async () => {
  await loadContainers();
});

const loadContainers = async () => {
  isLoadingContainers.value = true;
  try {
    const response = await $fetch("/api/docker/list-containers?all=true");
    if (response.success) {
      availableContainers.value = response.containers.map((container: any) => ({
        id: container.id || "",
        names: container.names || [],
        image: container.image || "",
        state: container.state || "",
        status: container.status || "",
        created: container.created || 0,
      }));
    }
  } catch (err) {
    console.error("Failed to load containers:", err);
    error.value = "Failed to load containers: " + (err as Error).message;
  } finally {
    isLoadingContainers.value = false;
  }
};

const selectContainer = (container: ContainerListItem) => {
  selectedContainer.value = container;
  error.value = "";
};

const connectToContainer = () => {
  if (!selectedContainer.value) {
    error.value = "Please select a container first";
    return;
  }

  if (isConnected.value) {
    disconnect();
  }

  isLoading.value = true;
  error.value = "";
  logs.value = [];

  startLogStream();
};

const startLogStream = () => {
  if (!selectedContainer.value) return;

  const url = `/api/docker/attach-stream?id=${encodeURIComponent(
    selectedContainer.value.id
  )}&follow=true&timestamps=true&tail=100`;

  eventSource.value = new EventSource(url);

  eventSource.value.onmessage = (event) => {
    try {
      const data: LogEntry = JSON.parse(event.data);
      logs.value.push(data);

      if (data.type === "connected") {
        isConnected.value = true;
        isLoading.value = false;
      } else if (data.type === "disconnected") {
        isConnected.value = false;
        isLoading.value = false;
      } else if (data.type === "error") {
        error.value = data.data || "Unknown error occurred";
        disconnect();
      }

      scrollToBottom();
    } catch (err) {
      console.error("Error parsing SSE message:", err);
    }
  };

  eventSource.value.onerror = (err) => {
    console.error("EventSource error:", err);
    error.value = "Connection lost";
    disconnect();
  };
};

const disconnect = () => {
  if (eventSource.value) {
    eventSource.value.close();
    eventSource.value = null;
  }
  isConnected.value = false;
  isLoading.value = false;

  if (logs.value.length > 0) {
    logs.value.push({
      type: "disconnected",
      data: "Disconnected from container",
      timestamp: new Date().toISOString(),
    });
  }
};

const scrollToBottom = () => {
  if (logContainer.value && autoScroll.value) {
    nextTick(() => {
      logContainer.value!.scrollTop = logContainer.value!.scrollHeight;
    });
  }
};

const clearLogs = () => {
  logs.value = [];
};

const formatTimestamp = (timestamp: string) => {
  return new Date(timestamp).toLocaleTimeString();
};

const getLogTypeClass = (type: string) => {
  switch (type) {
    case "connected":
      return "text-success";
    case "disconnected":
      return "text-warning";
    case "error":
      return "text-danger";
    case "stdout":
      return "text-success";
    case "stderr":
      return "text-danger";
    default:
      return "text-success";
  }
};

// Cleanup on unmount
onUnmounted(() => {
  disconnect();
});
</script>

<template>
  <div class="min-h-screen bg-background p-6">
    <div class="max-w-6xl mx-auto">
      <div class="bg-surface-base rounded-xl shadow-lg">
        <!-- Header -->
        <div class="border-b border-border-muted p-6">
          <OuiText
            as="h1"
            size="2xl"
            weight="bold"
            color="primary"
            class="mb-6"
          >
            Docker Container Log Viewer
          </OuiText>

          <!-- Container Selection -->
          <div v-if="!selectedContainer" class="space-y-4">
            <div class="flex items-center justify-between">
              <OuiText as="h2" size="lg" weight="medium" color="primary">
                Select a Container
              </OuiText>
              <button
                @click="loadContainers"
                :disabled="isLoadingContainers"
                class="px-4 py-2 bg-primary text-foreground rounded-xl hover:bg-primary/90 disabled:opacity-50"
              >
                {{ isLoadingContainers ? "Loading..." : "Refresh" }}
              </button>
            </div>

            <OuiText
              v-if="isLoadingContainers"
              align="center"
              color="secondary"
              class="py-8"
            >
              Loading containers...
            </OuiText>

            <OuiText
              v-else-if="availableContainers.length === 0"
              align="center"
              color="secondary"
              class="py-8"
            >
              No containers found
            </OuiText>

            <div v-else class="grid gap-4">
              <div
                v-for="container in availableContainers"
                :key="container.id"
                @click="selectContainer(container)"
                class="p-4 border border-border-muted rounded-xl hover:border-primary hover:bg-surface-muted/40 cursor-pointer transition-colors"
              >
                <div class="flex items-center justify-between">
                  <div class="flex-1">
                    <OuiText weight="medium" color="primary">
                      {{
                        container.names[0]?.replace(/^\//, "") ||
                        "Unnamed Container"
                      }}
                    </OuiText>
                    <OuiText size="sm" color="secondary" class="mt-1">
                      ID: {{ container.id.substring(0, 12) }} • Image:
                      {{ container.image }}
                    </OuiText>
                  </div>
                  <div class="text-right">
                    <span
                      class="inline-flex px-3 py-1 text-xs font-semibold rounded-full"
                      :class="
                        container.state === 'running'
                          ? 'bg-success/10 text-success'
                          : container.state === 'exited'
                          ? 'bg-danger/10 text-danger'
                          : 'bg-surface-muted/40 text-secondary'
                      "
                    >
                      {{ container.state }}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Selected Container Info -->
          <div v-else class="space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <OuiText as="h2" size="lg" weight="medium" color="primary">
                  {{
                    selectedContainer.names[0]?.replace(/^\//, "") ||
                    "Unnamed Container"
                  }}
                </OuiText>
                <OuiText size="sm" color="secondary">
                  {{ selectedContainer.id.substring(0, 12) }} •
                  {{ selectedContainer.image }}
                </OuiText>
              </div>
              <div class="flex items-center gap-4">
                <span
                  class="inline-flex px-3 py-1 text-xs font-semibold rounded-full"
                  :class="
                    selectedContainer.state === 'running'
                      ? 'bg-success/10 text-success'
                      : selectedContainer.state === 'exited'
                      ? 'bg-danger/10 text-danger'
                      : 'bg-surface-muted/40 text-secondary'
                  "
                >
                  {{ selectedContainer.state }}
                </span>
                <button
                  @click="
                    selectedContainer = null;
                    disconnect();
                  "
                  class="px-4 py-2 bg-secondary text-foreground rounded-xl hover:bg-secondary-dark"
                >
                  Change Container
                </button>
              </div>
            </div>

            <div class="flex gap-4">
              <button
                @click="connectToContainer"
                :disabled="isLoading || isConnected"
                class="px-6 py-2 bg-primary text-foreground rounded-xl hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <span v-if="isLoading">Connecting...</span>
                <span v-else-if="isConnected">Connected</span>
                <span v-else>Connect to Logs</span>
              </button>

              <button
                v-if="isConnected"
                @click="disconnect"
                class="px-6 py-2 bg-danger text-foreground rounded-xl hover:bg-danger/90"
              >
                Disconnect
              </button>
            </div>
          </div>

          <!-- Error Display -->
          <div
            v-if="error"
            class="bg-danger/10 border border-danger/20 rounded-xl p-4 mt-4"
          >
            <div class="flex">
              <div class="text-danger"><strong>Error:</strong> {{ error }}</div>
            </div>
          </div>
        </div>

        <!-- Log Controls -->
        <div
          v-if="selectedContainer"
          class="border-b border-border-muted p-4 bg-surface-muted/30"
        >
          <div class="flex justify-between items-center">
            <div class="flex items-center gap-4">
              <div class="flex items-center">
                <span class="text-sm text-secondary">Status:</span>
                <span class="ml-2 inline-flex items-center">
                  <div
                    class="w-2 h-2 rounded-full mr-2"
                    :class="isConnected ? 'bg-success' : 'bg-danger'"
                  ></div>
                  {{ isConnected ? "Connected" : "Disconnected" }}
                </span>
              </div>

              <div class="text-sm text-secondary">
                Messages: {{ logs.length }}
              </div>
            </div>

            <div class="flex gap-2">
              <label class="flex items-center text-sm text-secondary">
                <input type="checkbox" v-model="autoScroll" class="mr-2" />
                Auto-scroll
              </label>

              <button
                @click="clearLogs"
                class="px-3 py-1 bg-secondary text-foreground text-sm rounded hover:bg-secondary-dark"
              >
                Clear Logs
              </button>
            </div>
          </div>
        </div>

        <!-- Log Display -->
        <div v-if="selectedContainer" class="relative">
          <div
            ref="logContainer"
            class="h-96 overflow-auto bg-surface-overlay text-success font-mono text-sm p-4"
          >
            <OuiText
              v-if="logs.length === 0 && !isLoading"
              color="secondary"
              align="center"
              class="py-8"
            >
              No logs to display. Click "Connect to Logs" to start streaming.
            </OuiText>

            <OuiText
              v-if="isLoading"
              color="warning"
              align="center"
              class="py-8"
            >
              Connecting to container...
            </OuiText>

            <div
              v-for="(log, index) in logs"
              :key="index"
              class="mb-1 leading-relaxed"
              :class="getLogTypeClass(log.type)"
            >
              <span class="text-secondary mr-2">
                [{{ formatTimestamp(log.timestamp) }}]
              </span>

              <span class="text-primary mr-2"> [{{ log.type }}] </span>

              <span class="whitespace-pre-wrap">{{ log.data }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Custom scrollbar for dark log container */
.overflow-auto::-webkit-scrollbar {
  width: 8px;
}

.overflow-auto::-webkit-scrollbar-track {
  background: var(--oui-surface-muted);
}

.overflow-auto::-webkit-scrollbar-thumb {
  background: var(--oui-border-default);
  border-radius: 4px;
}

.overflow-auto::-webkit-scrollbar-thumb:hover {
  background: var(--oui-border-strong);
}
</style>
