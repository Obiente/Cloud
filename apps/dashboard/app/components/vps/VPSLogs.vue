<template>
  <OuiStack gap="sm">
    <!-- Toolbar -->
    <OuiCard variant="outline">
      <OuiCardBody class="py-2! px-4!">
        <OuiFlex align="center" justify="between" gap="md" wrap="wrap">
          <!-- Left: title -->
          <UiSectionHeader :icon="CommandLineIcon" color="secondary" size="sm">System Logs</UiSectionHeader>
          <!-- Right: controls -->
          <OuiFlex align="center" gap="sm">
            <OuiInput
              v-model="searchQuery"
              size="sm"
              placeholder="Filter logs…"
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
import { ref, computed, onMounted, onUnmounted } from "vue";
import {
  ArrowPathIcon,
  EllipsisVerticalIcon,
  CommandLineIcon,
  MagnifyingGlassIcon,
  XMarkIcon,
} from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { VPSService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import { useAuth } from "~/composables/useAuth";
import type { LogEntry } from "~/components/oui/Logs.vue";

interface Props {
  vpsId: string;
  organizationId: string;
}

interface LogLine {
  line: string;
  timestamp: string;
  stderr: boolean;
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const auth = useAuth();
const client = useConnectClient(VPSService);

const effectiveOrgId = computed(
  () => props.organizationId || orgsStore.currentOrgId || ""
);

const logsComponent = ref<any>(null);
const logs = ref<LogLine[]>([]);
const isFollowing = ref(false);
const isLoading = ref(false);
const showTimestamps = ref(true);
const searchQuery = ref("");
let streamController: AbortController | null = null;
let isAborting = false;

const formattedLogs = computed<LogEntry[]>(() =>
  logs.value.map((log, idx) => ({
    id: idx,
    line: log.line,
    timestamp: log.timestamp,
    stderr: log.stderr,
    level: log.stderr ? ("error" as const) : undefined,
  }))
);

const filteredLogs = computed<LogEntry[]>(() => {
  if (!searchQuery.value) return formattedLogs.value;
  const q = searchQuery.value.toLowerCase();
  return formattedLogs.value.filter((l) =>
    (l.line || "").toLowerCase().includes(q)
  );
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
  logs.value = [];

  let hasReceivedLogs = false;

  try {
    if (!auth.ready) {
      await new Promise<void>((resolve) => {
        const check = () => (auth.ready ? resolve() : setTimeout(check, 100));
        check();
      });
    }

    const token = await auth.getAccessToken();
    if (!token) throw new Error("Authentication required.");

    streamController = new AbortController();

    const stream = client.streamVPSLogs(
      {
        organizationId: effectiveOrgId.value,
        vpsId: props.vpsId,
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
  } catch (error: unknown) {
    const isAbortError =
      (error as any).name === "AbortError" ||
      (error as Error).message?.toLowerCase().includes("aborted") ||
      (error as Error).message?.toLowerCase().includes("canceled") ||
      (error as Error).message?.toLowerCase().includes("cancelled") ||
      isAborting;

    if (isAbortError) return;

    const isBenignError =
      (error as Error).message?.toLowerCase().includes("missing trailer") ||
      (error as Error).message?.toLowerCase().includes("trailer") ||
      (error as Error).message?.toLowerCase().includes("missing endstreamresponse") ||
      (error as Error).message?.toLowerCase().includes("endstreamresponse") ||
      (error as any).code === "unknown";

    if (!isBenignError || !hasReceivedLogs) {
      logs.value.push({
        line: `[error] Failed to stream logs: ${(error as Error).message}`,
        timestamp: new Date().toISOString(),
        stderr: true,
      });
    }
  } finally {
    isLoading.value = false;
    isFollowing.value = false;
    isAborting = false;
  }
};

const stopStream = () => {
  if (streamController) {
    isAborting = true;
    streamController.abort();
    streamController = null;
  }
  isFollowing.value = false;
};

onMounted(() => {
  startStream();
});

onUnmounted(() => {
  stopStream();
});
</script>

