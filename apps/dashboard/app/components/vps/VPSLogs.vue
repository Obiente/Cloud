<template>
  <OuiStack gap="sm">

    <!-- Unified toolbar -->
    <OuiCard variant="outline">
      <OuiCardBody class="py-2! px-3!">
        <OuiStack gap="xs">

          <!-- Row 1: source tabs + section controls -->
          <OuiFlex align="center" justify="between" gap="md" wrap="wrap">

            <!-- Source tabs -->
            <OuiFlex align="center" gap="xs" class="shrink-0">
              <button
                v-for="tab in sourceTabs"
                :key="tab.id"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium transition-colors whitespace-nowrap border"
                :class="activeSource === tab.id
                  ? 'bg-primary/10 text-primary border-primary/20'
                  : 'bg-transparent border-transparent text-text-secondary hover:bg-surface-muted hover:text-text-primary'"
                @click="activeSource = tab.id"
              >
                <component :is="tab.icon" class="h-3.5 w-3.5" />
                {{ tab.label }}
              </button>
            </OuiFlex>

            <!-- Provisioning controls -->
            <OuiFlex v-if="activeSource === 'provisioning'" align="center" gap="sm" wrap="wrap">
              <OuiInput
                v-model="provisioningSearchQuery"
                size="sm"
                placeholder="Filter logs…"
                class="w-40"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-3.5 w-3.5 text-text-tertiary" />
                </template>
                <template v-if="provisioningSearchQuery" #suffix>
                  <button class="text-text-tertiary hover:text-text-primary transition-colors" @click="provisioningSearchQuery = ''">
                    <XMarkIcon class="h-3.5 w-3.5" />
                  </button>
                </template>
              </OuiInput>

              <OuiFlex align="center" gap="xs" class="shrink-0">
                <span
                  class="h-1.5 w-1.5 rounded-full transition-colors"
                  :class="isFollowing ? 'bg-success animate-pulse' : 'bg-border-strong'"
                />
                <OuiText size="xs" color="tertiary" class="whitespace-nowrap">
                  {{ isFollowing ? "Live" : "Stopped" }}
                </OuiText>
              </OuiFlex>

              <OuiButton variant="ghost" size="sm" class="whitespace-nowrap shrink-0 gap-1.5" @click="toggleFollow">
                <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': provisioningLoading }" />
                {{ isFollowing ? "Stop" : "Follow" }}
              </OuiButton>

              <OuiButton variant="ghost" size="sm" :disabled="provisioningLogs.length === 0" @click="clearProvisioningLogs">
                Clear
              </OuiButton>

              <OuiMenu>
                <template #trigger>
                  <OuiButton variant="ghost" size="sm"><EllipsisVerticalIcon class="h-3.5 w-3.5" /></OuiButton>
                </template>
                <OuiMenuItem>
                  <OuiCheckbox v-model="showTimestamps" label="Show timestamps" @click.stop />
                </OuiMenuItem>
              </OuiMenu>
            </OuiFlex>

            <!-- Journal controls -->
            <OuiFlex v-else-if="activeSource === 'journal'" align="center" gap="sm" wrap="wrap">
              <OuiInput
                v-model="journalSearchQuery"
                size="sm"
                placeholder="Filter loaded logs…"
                class="w-40"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-3.5 w-3.5 text-text-tertiary" />
                </template>
                <template v-if="journalSearchQuery" #suffix>
                  <button class="text-text-tertiary hover:text-text-primary transition-colors" @click="journalSearchQuery = ''">
                    <XMarkIcon class="h-3.5 w-3.5" />
                  </button>
                </template>
              </OuiInput>

              <OuiButton variant="ghost" size="sm" class="whitespace-nowrap shrink-0 gap-1.5" @click="refreshJournalLogs">
                <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': journalLoading }" />
                Refresh
              </OuiButton>

              <OuiMenu>
                <template #trigger>
                  <OuiButton variant="ghost" size="sm"><EllipsisVerticalIcon class="h-3.5 w-3.5" /></OuiButton>
                </template>
                <OuiMenuItem>
                  <OuiCheckbox v-model="showTimestamps" label="Show timestamps" @click.stop />
                </OuiMenuItem>
              </OuiMenu>
            </OuiFlex>

            <!-- Services controls -->
            <OuiFlex v-else align="center" gap="sm" wrap="wrap">
              <OuiInput
                v-model="serviceSearchQuery"
                size="sm"
                placeholder="Search services…"
                class="w-48"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-3.5 w-3.5 text-text-tertiary" />
                </template>
                <template v-if="serviceSearchQuery" #suffix>
                  <button class="text-text-tertiary hover:text-text-primary transition-colors" @click="serviceSearchQuery = ''">
                    <XMarkIcon class="h-3.5 w-3.5" />
                  </button>
                </template>
              </OuiInput>

              <OuiButton variant="ghost" size="sm" class="whitespace-nowrap shrink-0 gap-1.5" @click="refreshServices">
                <ArrowPathIcon class="h-3.5 w-3.5" :class="{ 'animate-spin': servicesLoading }" />
                Refresh
              </OuiButton>
            </OuiFlex>

          </OuiFlex>

          <!-- Row 2 (journal only): unit + lines config -->
          <OuiFlex
            v-if="activeSource === 'journal'"
            align="center"
            gap="sm"
            wrap="wrap"
            class="border-t border-border-muted pt-2"
          >
            <OuiText size="xs" color="tertiary" class="shrink-0">Filter by:</OuiText>
            <OuiInput
              v-model="journalUnit"
              size="sm"
              placeholder="Unit (e.g. nginx.service)"
              class="flex-1 min-w-[160px] max-w-[260px]"
              @keydown.enter.prevent="refreshJournalLogs"
            />
            <OuiText size="xs" color="tertiary" class="shrink-0">Lines:</OuiText>
            <OuiInput
              v-model="journalLinesInput"
              type="number"
              min="25"
              max="1000"
              size="sm"
              placeholder="200"
              class="w-20"
              @keydown.enter.prevent="refreshJournalLogs"
            />
            <OuiButton variant="soft" color="primary" size="sm" class="shrink-0" @click="refreshJournalLogs">
              Apply
            </OuiButton>
            <OuiText v-if="journalUnit" size="xs" color="tertiary" class="ml-auto">
              Showing {{ journalLines }} lines for
              <span class="font-mono text-text-primary">{{ journalUnit }}</span>
            </OuiText>
            <OuiText v-else size="xs" color="tertiary" class="ml-auto">
              Showing latest {{ journalLines }} journal lines
            </OuiText>
          </OuiFlex>

          <!-- Row 2 (services only): include inactive toggle -->
          <OuiFlex
            v-if="activeSource === 'services'"
            align="center"
            gap="sm"
            class="border-t border-border-muted pt-2"
          >
            <OuiCheckbox v-model="includeInactiveServices" label="Include inactive services" />
            <OuiText v-if="services.length > 0" size="xs" color="tertiary" class="ml-auto">
              {{ filteredServices.length }}
              <template v-if="filteredServices.length !== services.length"> of {{ services.length }}</template>
              service{{ services.length !== 1 ? 's' : '' }}
            </OuiText>
          </OuiFlex>

        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Provisioning log viewer -->
    <template v-if="activeSource === 'provisioning'">
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
          Provisioning logs are retained briefly for setup and debugging.
        </OuiText>
        <OuiText size="xs" color="tertiary">
          {{ provisioningLogs.length }} line{{ provisioningLogs.length !== 1 ? "s" : "" }}
          <template v-if="provisioningSearchQuery && filteredProvisioningLogs.length !== provisioningLogs.length">
            · {{ filteredProvisioningLogs.length }} matching
          </template>
        </OuiText>
      </OuiFlex>
    </template>

    <!-- Journal log viewer -->
    <template v-else-if="activeSource === 'journal'">
      <OuiLogs
        :logs="filteredJournalLogs"
        :is-loading="journalLoading"
        :show-timestamps="showTimestamps"
        :enable-ansi="false"
        :auto-scroll="false"
        empty-message="No guest journal logs found for this VPS."
        loading-message="Loading journal logs from the guest OS…"
      >
        <template #loading>
          <div class="flex flex-col items-center gap-4 py-10">
            <ArrowPathIcon class="h-5 w-5 animate-spin text-primary" />
            <div class="flex flex-col items-center gap-2 text-center">
              <p class="text-sm font-medium text-text-primary">{{ journalConnStatus.message }}</p>
              <p class="text-xs text-text-tertiary max-w-xs">{{ journalConnStatus.sub }}</p>
            </div>
            <!-- Step indicators -->
            <div class="flex items-center gap-1.5">
              <template v-for="(step, i) in CONN_PHASES" :key="i">
                <div
                  class="h-1.5 w-6 rounded-full transition-colors duration-500"
                  :class="i < journalConnStatus.phase
                    ? 'bg-primary'
                    : i === journalConnStatus.phase
                      ? 'bg-primary/50'
                      : 'bg-border-strong'"
                />
              </template>
            </div>
          </div>
        </template>
      </OuiLogs>
      <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
        <OuiText v-if="journalError" size="xs" color="danger">{{ journalError }}</OuiText>
        <OuiText v-else size="xs" color="tertiary">Journal logs fetched from the guest OS via the hypervisor.</OuiText>
        <OuiText size="xs" color="tertiary">
          {{ journalLogs.length }} line{{ journalLogs.length !== 1 ? "s" : "" }}
          <template v-if="journalSearchQuery && filteredJournalLogs.length !== journalLogs.length">
            · {{ filteredJournalLogs.length }} matching
          </template>
        </OuiText>
      </OuiFlex>
    </template>

    <!-- Services table -->
    <template v-else>
      <!-- Connection status banner shown while loading services -->
      <OuiCard v-if="servicesLoading" variant="outline">
        <OuiCardBody class="py-5!">
          <OuiFlex align="center" justify="center" gap="md">
            <ArrowPathIcon class="h-4 w-4 shrink-0 animate-spin text-primary" />
            <div class="flex flex-col gap-0.5 min-w-0">
              <p class="text-sm font-medium text-text-primary">{{ servicesConnStatus.message }}</p>
              <p class="text-xs text-text-tertiary">{{ servicesConnStatus.sub }}</p>
            </div>
            <div class="flex items-center gap-1.5 shrink-0 ml-2">
              <template v-for="(step, i) in CONN_PHASES" :key="i">
                <div
                  class="h-1.5 w-6 rounded-full transition-colors duration-500"
                  :class="i < servicesConnStatus.phase
                    ? 'bg-primary'
                    : i === servicesConnStatus.phase
                      ? 'bg-primary/50'
                      : 'bg-border-strong'"
                />
              </template>
            </div>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <OuiTable
        v-else
        :columns="serviceColumns"
        :rows="filteredServices"
        :sortable="false"
        :resizable="true"
        :clickable="true"
        :loading="false"
        empty-text="No guest services found."
        aria-label="Guest systemd services"
        row-key="name"
        @row-click="handleServiceRowClick"
      >
        <template #cell-name="{ row }">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="semibold">{{ row.name }}</OuiText>
            <OuiText size="xs" color="tertiary">{{ row.description || "No description available" }}</OuiText>
          </OuiStack>
        </template>

        <template #cell-state="{ row }">
          <OuiFlex align="center" gap="xs">
            <OuiBadge :variant="serviceBadgeVariant(row.activeState, row.subState)" size="xs">
              {{ row.activeState || "unknown" }}
            </OuiBadge>
            <OuiText size="xs" color="tertiary">{{ row.subState || "—" }}</OuiText>
          </OuiFlex>
        </template>

        <template #cell-load="{ row }">
          <OuiText size="sm" color="tertiary">{{ row.loadState || "unknown" }}</OuiText>
        </template>

        <template #cell-action="{ row }">
          <OuiButton variant="ghost" size="sm" class="gap-1" @click.stop="openServiceJournal(row)">
            Journal
            <ArrowTopRightOnSquareIcon class="h-3 w-3" />
          </OuiButton>
        </template>
      </OuiTable>

      <OuiFlex justify="between" align="center" gap="md" wrap="wrap">
        <OuiText v-if="servicesError" size="xs" color="danger">{{ servicesError }}</OuiText>
        <OuiText v-else size="xs" color="tertiary">Click any service row to open its journal logs.</OuiText>
      </OuiFlex>
    </template>

  </OuiStack>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import {
  ArrowPathIcon,
  ArrowTopRightOnSquareIcon,
  CommandLineIcon,
  DocumentTextIcon,
  EllipsisVerticalIcon,
  MagnifyingGlassIcon,
  ServerStackIcon,
  XMarkIcon,
} from "@heroicons/vue/24/outline";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { VPSService } from "@obiente/proto";
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

const sourceTabs = computed<
  Array<{ id: SourceTab; label: string; icon: typeof CommandLineIcon }>
>(() => [
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

// ── Connection status for slow guest-OS calls (journal & services) ───────────
// The backend tries SSH (~12 s timeout) then falls back to the hypervisor agent.
// We mirror those phases in the UI so users understand what's happening.

type ConnPhase = 0 | 1 | 2; // 0=connect, 1=ssh, 2=agent

interface ConnStatus {
  phase: ConnPhase;
  message: string;
  sub: string;
}

const CONN_PHASES: Array<{ msg: string; sub: string; delay: number }> = [
  {
    msg: "Querying hypervisor agent…",
    sub: "Fetching data via Proxmox QEMU guest agent",
    delay: 0,
  },
  {
    msg: "Agent slow — trying SSH via gateway…",
    sub: "Falling back to SSH connection through the VPS gateway",
    delay: 5000,
  },
  {
    msg: "Still connecting…",
    sub: "This is taking longer than expected",
    delay: 12000,
  },
];

function makeConnStatus(): ConnStatus {
  return {
    phase: 0,
    message: CONN_PHASES[0]!.msg,
    sub: CONN_PHASES[0]!.sub,
  };
}

function startConnTimer(
  status: ReturnType<typeof ref<ConnStatus>>
): ReturnType<typeof setTimeout>[] {
  const timers: ReturnType<typeof setTimeout>[] = [];
  for (let i = 1; i < CONN_PHASES.length; i++) {
    const phaseEntry = CONN_PHASES[i]!;
    const idx = i as ConnPhase;
    timers.push(
      setTimeout(() => {
        status.value = { phase: idx, message: phaseEntry.msg, sub: phaseEntry.sub };
      }, phaseEntry.delay)
    );
  }
  return timers;
}

function clearConnTimers(timers: ReturnType<typeof setTimeout>[]) {
  timers.forEach(clearTimeout);
  timers.length = 0;
}

// ─────────────────────────────────────────────────────────────────────────────

const journalLogs = ref<JournalLogLine[]>([]);
const journalLoading = ref(false);
const journalError = ref<string | null>(null);
const journalConnStatus = ref<ConnStatus>(makeConnStatus());
const journalConnTimers: ReturnType<typeof setTimeout>[] = [];
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
const servicesConnStatus = ref<ConnStatus>(makeConnStatus());
const servicesConnTimers: ReturnType<typeof setTimeout>[] = [];
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

function formatTimestamp(value: Timestamp | undefined) {
  if (!value) {
    return new Date().toISOString();
  }
  const seconds =
    typeof value.seconds === "bigint"
      ? value.seconds
      : BigInt(value.seconds ?? 0);
  const nanos = typeof value.nanos === "number" ? value.nanos : 0;
  const millis = seconds * 1000n + BigInt(Math.floor(nanos / 1_000_000));
  return new Date(Number(millis)).toISOString();
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
  journalConnStatus.value = makeConnStatus();
  clearConnTimers(journalConnTimers);
  journalConnTimers.push(...startConnTimer(journalConnStatus));

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
    clearConnTimers(journalConnTimers);
    journalLoading.value = false;
  }
}

async function refreshServices() {
  servicesLoading.value = true;
  servicesError.value = null;
  servicesConnStatus.value = makeConnStatus();
  clearConnTimers(servicesConnTimers);
  servicesConnTimers.push(...startConnTimer(servicesConnStatus));

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
    clearConnTimers(servicesConnTimers);
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
