<template>
  <OuiStack gap="sm">
    <OuiCard variant="outline">
      <OuiCardBody class="py-2! px-4!">
        <OuiFlex align="center" gap="xs" wrap="wrap">
          <OuiButton
            v-for="tab in sourceTabs"
            :key="tab.id"
            :variant="activeSource === tab.id ? 'primary' : 'ghost'"
            size="sm"
            @click="activeSource = tab.id"
          >
            <component :is="tab.icon" class="h-3.5 w-3.5" />
            {{ tab.label }}
          </OuiButton>
        </OuiFlex>
      </OuiCardBody>
    </OuiCard>

    <template v-if="activeSource === 'provisioning'">
      <OuiCard variant="outline">
        <OuiCardBody class="py-2! px-4!">
          <OuiFlex align="center" justify="between" gap="md" wrap="wrap">
            <UiSectionHeader
              :icon="CommandLineIcon"
              color="secondary"
              size="sm"
            >
              Provisioning Logs
            </UiSectionHeader>

            <OuiFlex align="center" gap="sm">
              <OuiInput
                v-model="provisioningSearchQuery"
                size="sm"
                placeholder="Filter logs…"
                :style="{ width: '170px' }"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-3.5 w-3.5 text-tertiary" />
                </template>
                <template v-if="provisioningSearchQuery" #suffix>
                  <button
                    class="text-tertiary hover:text-primary transition-colors"
                    @click="provisioningSearchQuery = ''"
                  >
                    <XMarkIcon class="h-3.5 w-3.5" />
                  </button>
                </template>
              </OuiInput>

              <OuiFlex align="center" gap="xs" class="shrink-0">
                <span
                  class="h-1.5 w-1.5 rounded-full transition-colors"
                  :class="
                    isFollowing
                      ? 'bg-success animate-pulse'
                      : 'bg-border-strong'
                  "
                />
                <OuiText size="xs" color="tertiary" class="whitespace-nowrap">
                  {{ isFollowing ? "Live" : "Stopped" }}
                </OuiText>
              </OuiFlex>

              <OuiButton
                variant="ghost"
                size="sm"
                class="whitespace-nowrap shrink-0"
                @click="toggleFollow"
              >
                <ArrowPathIcon
                  class="h-3.5 w-3.5"
                  :class="{ 'animate-spin': provisioningLoading }"
                />
                {{ isFollowing ? "Stop" : "Follow" }}
              </OuiButton>

              <OuiButton
                variant="ghost"
                size="sm"
                :disabled="provisioningLogs.length === 0"
                @click="clearProvisioningLogs"
              >
                Clear
              </OuiButton>

              <OuiMenu>
                <template #trigger>
                  <OuiButton variant="ghost" size="sm">
                    <EllipsisVerticalIcon class="h-3.5 w-3.5" />
                  </OuiButton>
                </template>
                <OuiMenuItem>
                  <OuiCheckbox
                    v-model="showTimestamps"
                    label="Show timestamps"
                    @click.stop
                  />
                </OuiMenuItem>
              </OuiMenu>
            </OuiFlex>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <OuiLogs
        :logs="filteredProvisioningLogs"
        :is-loading="provisioningLoading"
        :show-timestamps="showTimestamps"
        :enable-ansi="true"
        :auto-scroll="true"
        empty-message="No provisioning logs yet."
        loading-message="Connecting to provisioning log stream…"
      />

      <OuiFlex justify="between" align="center">
        <OuiText size="xs" color="tertiary">
          These logs come from the platform provisioning flow and are retained
          briefly for setup/debugging.
        </OuiText>
        <OuiText size="xs" color="tertiary">
          {{ provisioningLogs.length }} line{{
            provisioningLogs.length !== 1 ? "s" : ""
          }}
          <template
            v-if="
              provisioningSearchQuery &&
              filteredProvisioningLogs.length !== provisioningLogs.length
            "
          >
            &middot; {{ filteredProvisioningLogs.length }} matching
          </template>
        </OuiText>
      </OuiFlex>
    </template>

    <template v-else-if="activeSource === 'journal'">
      <OuiCard variant="outline">
        <OuiCardBody class="py-2! px-4!">
          <OuiFlex align="center" justify="between" gap="md" wrap="wrap">
            <UiSectionHeader
              :icon="DocumentTextIcon"
              color="secondary"
              size="sm"
            >
              Guest Journal
            </UiSectionHeader>

            <OuiFlex align="center" gap="sm" wrap="wrap">
              <OuiInput
                v-model="journalSearchQuery"
                size="sm"
                placeholder="Filter loaded logs…"
                :style="{ width: '170px' }"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-3.5 w-3.5 text-tertiary" />
                </template>
                <template v-if="journalSearchQuery" #suffix>
                  <button
                    class="text-tertiary hover:text-primary transition-colors"
                    @click="journalSearchQuery = ''"
                  >
                    <XMarkIcon class="h-3.5 w-3.5" />
                  </button>
                </template>
              </OuiInput>

              <OuiInput
                v-model="journalUnit"
                size="sm"
                placeholder="Unit e.g. nginx.service"
                :style="{ width: '220px' }"
                @keydown.enter.prevent="refreshJournalLogs"
              />

              <OuiInput
                v-model="journalLinesInput"
                type="number"
                min="25"
                max="1000"
                size="sm"
                placeholder="200"
                :style="{ width: '90px' }"
                @keydown.enter.prevent="refreshJournalLogs"
              />

              <OuiButton
                variant="ghost"
                size="sm"
                class="whitespace-nowrap shrink-0"
                @click="refreshJournalLogs"
              >
                <ArrowPathIcon
                  class="h-3.5 w-3.5"
                  :class="{ 'animate-spin': journalLoading }"
                />
                Refresh
              </OuiButton>

              <OuiMenu>
                <template #trigger>
                  <OuiButton variant="ghost" size="sm">
                    <EllipsisVerticalIcon class="h-3.5 w-3.5" />
                  </OuiButton>
                </template>
                <OuiMenuItem>
                  <OuiCheckbox
                    v-model="showTimestamps"
                    label="Show timestamps"
                    @click.stop
                  />
                </OuiMenuItem>
              </OuiMenu>
            </OuiFlex>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <OuiLogs
        :logs="filteredJournalLogs"
        :is-loading="journalLoading"
        :show-timestamps="showTimestamps"
        :enable-ansi="false"
        :auto-scroll="false"
        empty-message="No guest journal logs found for this VPS."
        loading-message="Loading journal logs from the guest OS…"
      />

      <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
        <OuiText v-if="journalError" size="xs" color="danger">{{
          journalError
        }}</OuiText>
        <OuiText v-else size="xs" color="tertiary">
          {{
            journalUnit?.trim()
              ? `Showing ${journalLines} lines for ${journalUnit.trim()}`
              : `Showing the latest ${journalLines} guest journal lines`
          }}
        </OuiText>
        <OuiText size="xs" color="tertiary">
          {{ journalLogs.length }} line{{ journalLogs.length !== 1 ? "s" : "" }}
          <template
            v-if="
              journalSearchQuery &&
              filteredJournalLogs.length !== journalLogs.length
            "
          >
            &middot; {{ filteredJournalLogs.length }} matching
          </template>
        </OuiText>
      </OuiFlex>
    </template>

    <template v-else>
      <OuiCard variant="outline">
        <OuiCardBody class="py-2! px-4!">
          <OuiFlex align="center" justify="between" gap="md" wrap="wrap">
            <UiSectionHeader
              :icon="ServerStackIcon"
              color="secondary"
              size="sm"
            >
              Guest Services
            </UiSectionHeader>

            <OuiFlex align="center" gap="sm" wrap="wrap">
              <OuiInput
                v-model="serviceSearchQuery"
                size="sm"
                placeholder="Search services…"
                :style="{ width: '200px' }"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-3.5 w-3.5 text-tertiary" />
                </template>
                <template v-if="serviceSearchQuery" #suffix>
                  <button
                    class="text-tertiary hover:text-primary transition-colors"
                    @click="serviceSearchQuery = ''"
                  >
                    <XMarkIcon class="h-3.5 w-3.5" />
                  </button>
                </template>
              </OuiInput>

              <OuiCheckbox
                v-model="includeInactiveServices"
                label="Include inactive"
              />

              <OuiButton
                variant="ghost"
                size="sm"
                class="whitespace-nowrap shrink-0"
                @click="refreshServices"
              >
                <ArrowPathIcon
                  class="h-3.5 w-3.5"
                  :class="{ 'animate-spin': servicesLoading }"
                />
                Refresh
              </OuiButton>
            </OuiFlex>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <OuiTable
        :columns="serviceColumns"
        :rows="filteredServices"
        :sortable="false"
        :resizable="true"
        :clickable="true"
        :loading="servicesLoading"
        empty-text="No guest services found."
        aria-label="Guest systemd services"
        row-key="name"
        @row-click="handleServiceRowClick"
      >
        <template #cell-name="{ row }">
          <OuiStack gap="xs">
            <OuiText size="sm" class="font-medium">{{ row.name }}</OuiText>
            <OuiText size="xs" color="tertiary">{{
              row.description || "No description available"
            }}</OuiText>
          </OuiStack>
        </template>

        <template #cell-state="{ row }">
          <OuiFlex align="center" gap="xs" wrap="wrap">
            <OuiBadge
              :variant="serviceBadgeVariant(row.activeState, row.subState)"
              size="xs"
            >
              {{ row.activeState || "unknown" }}
            </OuiBadge>
            <OuiText size="xs" color="tertiary">{{
              row.subState || "n/a"
            }}</OuiText>
          </OuiFlex>
        </template>

        <template #cell-load="{ row }">
          <OuiText size="sm">{{ row.loadState || "unknown" }}</OuiText>
        </template>

        <template #cell-action="{ row }">
          <OuiButton
            variant="ghost"
            size="sm"
            @click.stop="openServiceJournal(row)"
          >
            View Journal
          </OuiButton>
        </template>
      </OuiTable>

      <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
        <OuiText v-if="servicesError" size="xs" color="danger">{{
          servicesError
        }}</OuiText>
        <OuiText v-else size="xs" color="tertiary">
          Click a service to jump into its journal logs.
        </OuiText>
        <OuiText size="xs" color="tertiary">
          {{ filteredServices.length }} service{{
            filteredServices.length !== 1 ? "s" : ""
          }}
          <template v-if="filteredServices.length !== services.length">
            of {{ services.length }}</template
          >
        </OuiText>
      </OuiFlex>
    </template>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import {
  ArrowPathIcon,
  CommandLineIcon,
  DocumentTextIcon,
  EllipsisVerticalIcon,
  MagnifyingGlassIcon,
  ServerStackIcon,
  XMarkIcon,
} from "@heroicons/vue/24/outline";
import { VPSService } from "@obiente/proto";
import { date } from "@obiente/proto/utils";
import { useConnectClient } from "~/lib/connect-client";
import { useAuth } from "~/composables/useAuth";
import type { LogEntry } from "~/components/oui/Logs.vue";
import type { TableColumn } from "~/components/oui/Table.vue";
import { useOrganizationsStore } from "~/stores/organizations";

interface Props {
  vpsId: string;
  organizationId: string;
}

interface ProvisioningLogLine {
  line: string;
  timestamp: string;
  stderr: boolean;
}

interface JournalLogLine {
  line: string;
  timestamp: string;
  stderr: boolean;
  lineNumber: number;
}

interface ServiceRow {
  name: string;
  loadState: string;
  activeState: string;
  subState: string;
  description: string;
}

type SourceTab = "provisioning" | "journal" | "services";

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const auth = useAuth();
const client = useConnectClient(VPSService);

const effectiveOrgId = computed(
  () => props.organizationId || orgsStore.currentOrgId || ""
);

const activeSource = ref<SourceTab>("provisioning");
const showTimestamps = ref(true);

const sourceTabs = computed(() => [
  { id: "provisioning", label: "Provisioning", icon: CommandLineIcon },
  { id: "journal", label: "Journal", icon: DocumentTextIcon },
  { id: "services", label: "Services", icon: ServerStackIcon },
]);

const provisioningLogs = ref<ProvisioningLogLine[]>([]);
const provisioningLoading = ref(false);
const provisioningSearchQuery = ref("");
const isFollowing = ref(false);
let provisioningStreamController: AbortController | null = null;
let provisioningAbortRequested = false;

const journalLogs = ref<JournalLogLine[]>([]);
const journalLoading = ref(false);
const journalError = ref<string | null>(null);
const journalSearchQuery = ref("");
const journalUnit = ref("");
const journalLines = ref(200);
const journalLinesInput = computed({
  get: () => String(journalLines.value),
  set: (value: string) => {
    const parsed = Number.parseInt(value, 10);
    if (Number.isNaN(parsed)) {
      return;
    }
    journalLines.value = Math.min(1000, Math.max(25, parsed));
  },
});

const services = ref<ServiceRow[]>([]);
const servicesLoading = ref(false);
const servicesError = ref<string | null>(null);
const serviceSearchQuery = ref("");
const includeInactiveServices = ref(true);

const serviceColumns: TableColumn<ServiceRow>[] = [
  { key: "name", label: "Service", defaultWidth: 360, minWidth: 260 },
  {
    key: "state",
    label: "State",
    defaultWidth: 180,
    minWidth: 160,
    accessor: (row) => row.activeState,
  },
  {
    key: "load",
    label: "Load",
    defaultWidth: 130,
    minWidth: 120,
    accessor: (row) => row.loadState,
  },
  {
    key: "action",
    label: "Action",
    defaultWidth: 140,
    minWidth: 130,
    sortable: false,
  },
];

const provisioningEntries = computed<LogEntry[]>(() =>
  provisioningLogs.value.map((log, idx) => ({
    id: idx,
    line: log.line,
    timestamp: log.timestamp,
    stderr: log.stderr,
    level: log.stderr ? ("error" as const) : undefined,
  }))
);

const filteredProvisioningLogs = computed<LogEntry[]>(() => {
  if (!provisioningSearchQuery.value) {
    return provisioningEntries.value;
  }
  const query = provisioningSearchQuery.value.toLowerCase();
  return provisioningEntries.value.filter((log) =>
    (log.line || "").toLowerCase().includes(query)
  );
});

const journalEntries = computed<LogEntry[]>(() =>
  journalLogs.value.map((log) => ({
    id: log.lineNumber,
    line: log.line,
    timestamp: log.timestamp,
    stderr: log.stderr,
    level: log.stderr ? ("error" as const) : undefined,
  }))
);

const filteredJournalLogs = computed<LogEntry[]>(() => {
  if (!journalSearchQuery.value) {
    return journalEntries.value;
  }
  const query = journalSearchQuery.value.toLowerCase();
  return journalEntries.value.filter((log) =>
    (log.line || "").toLowerCase().includes(query)
  );
});

const filteredServices = computed<ServiceRow[]>(() => {
  const query = serviceSearchQuery.value.trim().toLowerCase();
  if (!query) {
    return services.value;
  }
  return services.value.filter((service) =>
    [
      service.name,
      service.description,
      service.activeState,
      service.subState,
      service.loadState,
    ]
      .join(" ")
      .toLowerCase()
      .includes(query)
  );
});

async function ensureAuthReady() {
  if (auth.ready) {
    return;
  }
  await new Promise<void>((resolve) => {
    const check = () => (auth.ready ? resolve() : setTimeout(check, 100));
    check();
  });
}

function formatTimestamp(
  value: { seconds: bigint; nanos: number } | undefined
) {
  if (!value) {
    return new Date().toISOString();
  }
  return date(value).toISOString();
}

function clearProvisioningLogs() {
  provisioningLogs.value = [];
}

function toggleFollow() {
  if (isFollowing.value) {
    stopProvisioningStream();
    return;
  }
  void startProvisioningStream();
}

async function startProvisioningStream() {
  if (isFollowing.value || activeSource.value !== "provisioning") {
    return;
  }

  provisioningLoading.value = true;
  isFollowing.value = true;
  provisioningLogs.value = [];

  try {
    await ensureAuthReady();
    const token = await auth.getAccessToken();
    if (!token) {
      throw new Error("Authentication required.");
    }

    provisioningStreamController = new AbortController();
    const stream = client.streamVPSLogs(
      {
        organizationId: effectiveOrgId.value,
        vpsId: props.vpsId,
      },
      { signal: provisioningStreamController.signal }
    );

    provisioningLoading.value = false;

    for await (const update of stream) {
      if (!update.line) {
        continue;
      }

      provisioningLogs.value.push({
        line: update.line,
        timestamp: formatTimestamp(update.timestamp),
        stderr: update.stderr || false,
      });
    }
  } catch (error: unknown) {
    const message = error instanceof Error ? error.message.toLowerCase() : "";
    const isAbortError =
      (error as { name?: string })?.name === "AbortError" ||
      provisioningAbortRequested ||
      message.includes("aborted") ||
      message.includes("canceled") ||
      message.includes("cancelled");

    if (!isAbortError) {
      provisioningLogs.value.push({
        line: `[error] Failed to stream provisioning logs: ${
          error instanceof Error ? error.message : "Unknown error"
        }`,
        timestamp: new Date().toISOString(),
        stderr: true,
      });
    }
  } finally {
    provisioningLoading.value = false;
    isFollowing.value = false;
    provisioningAbortRequested = false;
  }
}

function stopProvisioningStream() {
  if (provisioningStreamController) {
    provisioningAbortRequested = true;
    provisioningStreamController.abort();
    provisioningStreamController = null;
  }
  isFollowing.value = false;
}

async function refreshJournalLogs() {
  journalLoading.value = true;
  journalError.value = null;

  try {
    await ensureAuthReady();
    const response = await client.getVPSJournalLogs({
      organizationId: effectiveOrgId.value,
      vpsId: props.vpsId,
      unit: journalUnit.value.trim() || undefined,
      lines: journalLines.value,
    });

    journalLogs.value = (response.logs || []).map((log, index) => ({
      line: log.line,
      timestamp: formatTimestamp(log.timestamp),
      stderr: log.stderr || false,
      lineNumber: log.lineNumber || index + 1,
    }));
  } catch (error: unknown) {
    journalLogs.value = [];
    journalError.value =
      error instanceof Error ? error.message : "Failed to load journal logs.";
  } finally {
    journalLoading.value = false;
  }
}

async function refreshServices() {
  servicesLoading.value = true;
  servicesError.value = null;

  try {
    await ensureAuthReady();
    const response = await client.listVPSServices({
      organizationId: effectiveOrgId.value,
      vpsId: props.vpsId,
      includeInactive: includeInactiveServices.value,
    });

    services.value = (response.services || []).map((service) => ({
      name: service.name,
      loadState: service.loadState,
      activeState: service.activeState,
      subState: service.subState,
      description: service.description,
    }));
  } catch (error: unknown) {
    services.value = [];
    servicesError.value =
      error instanceof Error ? error.message : "Failed to load services.";
  } finally {
    servicesLoading.value = false;
  }
}

function serviceBadgeVariant(activeState: string, subState: string) {
  if (activeState === "active") {
    return "success";
  }
  if (activeState === "failed" || subState === "failed") {
    return "danger";
  }
  if (activeState === "activating" || activeState === "reloading") {
    return "warning";
  }
  return "secondary";
}

async function openServiceJournal(service: ServiceRow) {
  journalUnit.value = service.name;
  activeSource.value = "journal";
  await refreshJournalLogs();
}

function handleServiceRowClick(row: ServiceRow) {
  void openServiceJournal(row);
}

watch(activeSource, (source) => {
  if (source === "provisioning") {
    void startProvisioningStream();
    return;
  }

  stopProvisioningStream();

  if (
    source === "journal" &&
    journalLogs.value.length === 0 &&
    !journalLoading.value
  ) {
    void refreshJournalLogs();
  }
  if (
    source === "services" &&
    services.value.length === 0 &&
    !servicesLoading.value
  ) {
    void refreshServices();
  }
});

watch(includeInactiveServices, () => {
  if (activeSource.value === "services") {
    void refreshServices();
  }
});

watch(
  () => [props.vpsId, effectiveOrgId.value] as const,
  () => {
    stopProvisioningStream();
    provisioningLogs.value = [];
    journalLogs.value = [];
    journalError.value = null;
    services.value = [];
    servicesError.value = null;

    if (activeSource.value === "provisioning") {
      void startProvisioningStream();
    } else if (activeSource.value === "journal") {
      void refreshJournalLogs();
    } else {
      void refreshServices();
    }
  }
);

onMounted(() => {
  void startProvisioningStream();
});

onUnmounted(() => {
  stopProvisioningStream();
});
</script>
