<template>
  <SuperadminPageLayout
    title="Game Servers"
    description="View and manage game server instances across all organizations."
    :columns="tableColumns"
    :rows="filteredTableRows"
    :filters="filterConfigs"
    :search="search"
    :empty-text="isLoading ? 'Loading game servers…' : 'No game servers match your filters.'"
    :loading="isLoading"
    search-placeholder="Search by name, ID, organization, image…"
    :pagination="pagination ? {
      page: pagination.page,
      totalPages: pagination.totalPages,
      total: pagination.total,
      perPage: perPage,
    } : undefined"
    @update:search="search = $event"
    @filter-change="handleFilterChange"
    @refresh="() => fetchGameServers(currentPage)"
    @page-change="goToPage"
  >
    <template #cell-name="{ row }">
      <OuiStack gap="xs">
        <OuiFlex gap="xs" align="center">
          <span class="font-medium text-text-primary">{{ row.gameServer?.name || "—" }}</span>
          <OuiBadge v-if="row.isSuspended" variant="warning" tone="soft" size="xs">SUSPENDED</OuiBadge>
          <OuiBadge v-if="row.flaggedAsMiner" variant="danger" tone="soft" size="xs">⚠ MINER SUSPECTED</OuiBadge>
        </OuiFlex>
        <OuiText color="tertiary" size="xs" class="font-mono">{{ row.gameServer?.id }}</OuiText>
        <OuiText v-if="row.isSuspended && row.suspensionReason" color="tertiary" size="xs">
          Reason: {{ row.suspensionReason }}
        </OuiText>
      </OuiStack>
    </template>
    <template #cell-organization="{ row }">
      <SuperadminOrganizationCell
        :organization-name="row.organizationName"
        :organization-id="row.gameServer?.organizationId"
        :owner-name="row.ownerName"
        :owner-id="row.ownerId"
      />
    </template>
    <template #cell-status="{ row }">
      <SuperadminStatusBadge
        :status="row.gameServer?.status"
        :status-map="gameServerStatusMap"
      />
    </template>
    <template #cell-gameType="{ row }">
      <span class="text-sm">{{ getGameTypeLabel(row.gameServer?.gameType) }}</span>
    </template>
    <template #cell-resources="{ row }">
      <OuiStack gap="xs">
        <div class="text-sm">
          <span class="text-text-secondary">CPU:</span>
          <span class="font-mono ml-1">{{ row.gameServer?.cpuCores || 0 }} cores</span>
        </div>
        <div class="text-sm">
          <span class="text-text-secondary">Memory:</span>
          <span class="font-mono ml-1">{{ formatBytes(Number(row.gameServer?.memoryBytes || 0)) }}</span>
        </div>
      </OuiStack>
    </template>
    <template #cell-image="{ row }">
      <span class="text-xs font-mono break-all">{{ row.gameServer?.dockerImage || "—" }}</span>
    </template>
    <template #cell-created="{ row }">
      <OuiDate v-if="row.gameServer?.createdAt" :value="row.gameServer.createdAt" format="short" />
      <span v-else class="text-sm text-text-muted">—</span>
    </template>
    <template #cell-actions="{ row }">
      <SuperadminActionsCell :actions="getGameServerActions(row)" />
    </template>
  </SuperadminPageLayout>

  <!-- Suspend Dialog -->
  <OuiDialog v-model:open="suspendDialogOpen" title="Suspend Game Server">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Suspend this game server. The container will be stopped and further operations will be prevented.
      </OuiText>
      <OuiInput
        v-model="suspendForm.reason"
        label="Reason (Optional)"
        placeholder="Reason for suspension (e.g. Cryptomining abuse)"
      />
    </OuiStack>
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="suspendDialogOpen = false">Cancel</OuiButton>
        <OuiButton color="warning" @click="handleSuspend" :disabled="isSuspending">
          {{ isSuspending ? 'Suspending...' : 'Suspend' }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>

  <!-- Unsuspend Dialog -->
  <OuiDialog v-model:open="unsuspendDialogOpen" title="Unsuspend Game Server">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Lift the suspension on this game server. The owner will be able to start it again.
      </OuiText>
    </OuiStack>
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="unsuspendDialogOpen = false">Cancel</OuiButton>
        <OuiButton color="primary" @click="handleUnsuspend" :disabled="isUnsuspending">
          {{ isUnsuspending ? 'Unsuspending...' : 'Unsuspend' }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>

  <!-- Force Stop Dialog -->
  <OuiDialog v-model:open="forceStopDialogOpen" title="Force Stop Game Server">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Force stop this game server container immediately. This performs a hard shutdown.
      </OuiText>
    </OuiStack>
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="forceStopDialogOpen = false">Cancel</OuiButton>
        <OuiButton color="danger" @click="handleForceStop" :disabled="isForceStopping">
          {{ isForceStopping ? 'Stopping...' : 'Force Stop' }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>

  <!-- Force Delete Dialog -->
  <OuiDialog v-model:open="forceDeleteDialogOpen" title="Force Delete Game Server">
    <OuiStack gap="lg">
      <OuiText size="sm" color="tertiary">
        Permanently delete this game server. This action cannot be undone.
      </OuiText>
      <OuiCheckbox v-model="forceDeleteForm.hardDelete" label="Hard delete (remove container and all data immediately)" />
    </OuiStack>
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="forceDeleteDialogOpen = false">Cancel</OuiButton>
        <OuiButton color="danger" @click="handleForceDelete" :disabled="isForceDeleting">
          {{ isForceDeleting ? 'Deleting...' : 'Delete' }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { SuperadminService, GameServerStatus, GameType } from "@obiente/proto";
import { useToast } from "~/composables/useToast";
import { useUtils } from "~/composables/useUtils";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import SuperadminOrganizationCell from "~/components/superadmin/SuperadminOrganizationCell.vue";
import SuperadminStatusBadge from "~/components/superadmin/SuperadminStatusBadge.vue";
import SuperadminActionsCell, { type Action } from "~/components/superadmin/SuperadminActionsCell.vue";
import type { FilterConfig } from "~/components/superadmin/SuperadminFilterBar.vue";
import type { BadgeVariant } from "~/components/oui/Badge.vue";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const { toast } = useToast();
const client = useConnectClient(SuperadminService);
const { formatBytes } = useUtils();

const gameServers = ref<any[]>([]);
const isLoading = ref(false);
const pagination = ref<any>(null);
const currentPage = ref(1);
const perPage = 20;
const search = ref("");
const statusFilter = ref<string>("all");
const flaggedOnlyFilter = ref<string>("all");

const tableColumns = computed(() => [
  { key: "name", label: "Name", defaultWidth: 220, minWidth: 180 },
  { key: "organization", label: "Organization", defaultWidth: 180, minWidth: 150 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "gameType", label: "Game Type", defaultWidth: 130, minWidth: 100 },
  { key: "resources", label: "Resources", defaultWidth: 160, minWidth: 140 },
  { key: "image", label: "Docker Image", defaultWidth: 200, minWidth: 160 },
  { key: "created", label: "Created", defaultWidth: 140, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 100, minWidth: 80 },
]);

const gameServerStatusMap: Record<number, { label: string; variant: BadgeVariant }> = {
  [GameServerStatus.GAME_SERVER_STATUS_UNSPECIFIED]: { label: "Unknown", variant: "secondary" },
  [GameServerStatus.CREATED]: { label: "Created", variant: "secondary" },
  [GameServerStatus.STARTING]: { label: "Starting", variant: "warning" },
  [GameServerStatus.RUNNING]: { label: "Running", variant: "success" },
  [GameServerStatus.STOPPING]: { label: "Stopping", variant: "warning" },
  [GameServerStatus.STOPPED]: { label: "Stopped", variant: "secondary" },
  [GameServerStatus.FAILED]: { label: "Failed", variant: "danger" },
  [GameServerStatus.RESTARTING]: { label: "Restarting", variant: "warning" },
};

const gameTypeMap: Record<number, string> = {
  [GameType.GAME_TYPE_UNSPECIFIED]: "Unknown",
  [GameType.MINECRAFT]: "Minecraft",
  [GameType.MINECRAFT_JAVA]: "Minecraft Java",
  [GameType.MINECRAFT_BEDROCK]: "Minecraft Bedrock",
  [GameType.VALHEIM]: "Valheim",
  [GameType.TERRARIA]: "Terraria",
  [GameType.RUST]: "Rust",
  [GameType.CS2]: "CS2",
  [GameType.TF2]: "TF2",
  [GameType.ARK]: "ARK",
  [GameType.CONAN]: "Conan Exiles",
  [GameType.SEVEN_DAYS]: "7 Days to Die",
  [GameType.FACTORIO]: "Factorio",
  [GameType.SPACED_ENGINEERS]: "Space Engineers",
  [GameType.OTHER]: "Other",
};

function getGameTypeLabel(gameType: number | undefined): string {
  if (gameType === undefined || gameType === null) return "—";
  return gameTypeMap[gameType] || `Type ${gameType}`;
}

const statusOptions = computed(() => [
  { label: "All statuses", value: "all" },
  { label: "Running", value: String(GameServerStatus.RUNNING) },
  { label: "Stopped", value: String(GameServerStatus.STOPPED) },
  { label: "Failed", value: String(GameServerStatus.FAILED) },
  { label: "Starting", value: String(GameServerStatus.STARTING) },
  { label: "Stopping", value: String(GameServerStatus.STOPPING) },
  { label: "Created", value: String(GameServerStatus.CREATED) },
]);

const filterConfigs = computed(() => [
  {
    key: "status",
    placeholder: "Status",
    items: statusOptions.value,
  },
  {
    key: "flaggedOnly",
    placeholder: "Flagged",
    items: [
      { label: "All servers", value: "all" },
      { label: "Flagged only", value: "flagged" },
    ],
  },
] as FilterConfig[]);

const enrichedRows = computed(() =>
  gameServers.value.map((gs) => ({
    ...gs,
    flaggedAsMiner: gs.suspensionReason
      ? gs.suspensionReason.toLowerCase().includes("miner") ||
        gs.suspensionReason.toLowerCase().includes("cryptomin")
      : false,
  }))
);

const filteredTableRows = computed(() => {
  const term = search.value.trim().toLowerCase();
  const status = statusFilter.value;
  const flagged = flaggedOnlyFilter.value;

  return enrichedRows.value.filter((gs) => {
    if (status !== "all" && String(gs.gameServer?.status) !== status) return false;
    if (flagged === "flagged" && !gs.isSuspended && !gs.flaggedAsMiner) return false;

    if (!term) return true;
    const searchable = [
      gs.gameServer?.name,
      gs.gameServer?.id,
      gs.organizationName,
      gs.gameServer?.organizationId,
      gs.gameServer?.dockerImage,
      getGameTypeLabel(gs.gameServer?.gameType),
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    return searchable.includes(term);
  });
});

function handleFilterChange(key: string, value: string) {
  if (key === "status") statusFilter.value = value;
  else if (key === "flaggedOnly") flaggedOnlyFilter.value = value;
}

const fetchGameServers = async (page: number = 1) => {
  isLoading.value = true;
  try {
    const response = await client.listAllGameServers({
      page,
      perPage,
    });
    gameServers.value = response.gameServers || [];
    pagination.value = response.pagination;
    currentPage.value = page;
  } catch (error: unknown) {
    toast.error(`Failed to load game servers: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isLoading.value = false;
  }
};

const goToPage = (page: number) => {
  if (page >= 1 && (!pagination.value || page <= pagination.value.totalPages)) {
    fetchGameServers(page);
  }
};

onMounted(() => fetchGameServers());

// Suspend
const suspendDialogOpen = ref(false);
const isSuspending = ref(false);
const suspendForm = ref({ gameServerId: "", reason: "" });

function openSuspendDialog(row: any) {
  suspendForm.value.gameServerId = row.gameServer?.id || "";
  suspendForm.value.reason = "";
  suspendDialogOpen.value = true;
}

const handleSuspend = async () => {
  if (!suspendForm.value.gameServerId) return;
  isSuspending.value = true;
  try {
    await client.superadminSuspendGameServer({
      gameServerId: suspendForm.value.gameServerId,
      reason: suspendForm.value.reason || undefined,
    });
    toast.success("Game server suspended.");
    suspendDialogOpen.value = false;
    await fetchGameServers(currentPage.value);
  } catch (error: unknown) {
    toast.error(`Failed to suspend: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isSuspending.value = false;
  }
};

// Unsuspend
const unsuspendDialogOpen = ref(false);
const isUnsuspending = ref(false);
const unsuspendGameServerId = ref("");

function openUnsuspendDialog(row: any) {
  unsuspendGameServerId.value = row.gameServer?.id || "";
  unsuspendDialogOpen.value = true;
}

const handleUnsuspend = async () => {
  if (!unsuspendGameServerId.value) return;
  isUnsuspending.value = true;
  try {
    await client.superadminUnsuspendGameServer({
      gameServerId: unsuspendGameServerId.value,
    });
    toast.success("Game server unsuspended.");
    unsuspendDialogOpen.value = false;
    await fetchGameServers(currentPage.value);
  } catch (error: unknown) {
    toast.error(`Failed to unsuspend: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isUnsuspending.value = false;
  }
};

// Force Stop
const forceStopDialogOpen = ref(false);
const isForceStopping = ref(false);
const forceStopGameServerId = ref("");

function openForceStopDialog(row: any) {
  forceStopGameServerId.value = row.gameServer?.id || "";
  forceStopDialogOpen.value = true;
}

const handleForceStop = async () => {
  if (!forceStopGameServerId.value) return;
  isForceStopping.value = true;
  try {
    await client.superadminForceStopGameServer({
      gameServerId: forceStopGameServerId.value,
    });
    toast.success("Game server stopped.");
    forceStopDialogOpen.value = false;
    await fetchGameServers(currentPage.value);
  } catch (error: unknown) {
    toast.error(`Failed to force stop: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isForceStopping.value = false;
  }
};

// Force Delete
const forceDeleteDialogOpen = ref(false);
const isForceDeleting = ref(false);
const forceDeleteForm = ref({ gameServerId: "", hardDelete: false });

function openForceDeleteDialog(row: any) {
  forceDeleteForm.value.gameServerId = row.gameServer?.id || "";
  forceDeleteForm.value.hardDelete = false;
  forceDeleteDialogOpen.value = true;
}

const handleForceDelete = async () => {
  if (!forceDeleteForm.value.gameServerId) return;
  isForceDeleting.value = true;
  try {
    await client.superadminForceDeleteGameServer({
      gameServerId: forceDeleteForm.value.gameServerId,
      hardDelete: forceDeleteForm.value.hardDelete,
    });
    toast.success("Game server deleted.");
    forceDeleteDialogOpen.value = false;
    await fetchGameServers(currentPage.value);
  } catch (error: unknown) {
    toast.error(`Failed to delete: ${(error as any)?.message || "Unknown error"}`);
  } finally {
    isForceDeleting.value = false;
  }
};

const getGameServerActions = (row: any): Action[] => {
  const actions: Action[] = [];

  if (row.isSuspended) {
    actions.push({
      key: "unsuspend",
      label: "Unsuspend",
      onClick: () => openUnsuspendDialog(row),
    });
  } else {
    actions.push({
      key: "suspend",
      label: "Suspend",
      onClick: () => openSuspendDialog(row),
    });
  }

  actions.push(
    {
      key: "force-stop",
      label: "Force Stop",
      onClick: () => openForceStopDialog(row),
      color: "danger",
    },
    {
      key: "force-delete",
      label: "Force Delete",
      onClick: () => openForceDeleteDialog(row),
      color: "danger",
    }
  );

  return actions;
};
</script>
