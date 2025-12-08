<template>
  <OuiContainer size="full">
    <OuiStack gap="xl">
      <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
        <OuiStack gap="sm" class="max-w-xl">
          <OuiFlex align="center" gap="md">
            <OuiBox
              p="sm"
              rounded="xl"
              bg="accent-primary"
              class="bg-primary/10 ring-1 ring-primary/20"
            >
              <CubeIcon class="w-6 h-6 text-primary" />
            </OuiBox>
            <OuiText as="h1" size="3xl" weight="bold"> Game Servers </OuiText>
          </OuiFlex>
          <OuiText color="secondary" size="md">
            Manage and monitor your game server instances with pay-as-you-go pricing.
            Low costs when idle or offline.
          </OuiText>
        </OuiStack>

        <OuiButton
          color="primary"
          class="gap-2 shadow-lg shadow-primary/20 hover:shadow-xl hover:shadow-primary/30 transition-all"
          @click="showCreateDialog = true"
        >
          <PlusIcon class="h-4 w-4" />
          <OuiText as="span" size="sm" weight="medium">New Game Server</OuiText>
        </OuiButton>
      </OuiFlex>

      <!-- Show error alert if there was a problem loading game servers -->
      <ErrorAlert
        v-if="listError"
        :error="listError"
        title="Failed to load game servers"
        hint="Please try refreshing the page. If the problem persists, contact support."
      />

      <OuiCard
        variant="default"
        class="backdrop-blur-sm border border-border-muted/60"
      >
        <OuiCardBody>
          <OuiGrid :cols="{ sm: 1, md: 3 }" gap="md">
            <OuiInput
              v-model="searchQuery"
              placeholder="Search by name or game type..."
              clearable
            >
              <template #prefix>
                <MagnifyingGlassIcon class="h-4 w-4 text-secondary" />
              </template>
            </OuiInput>

            <OuiSelect
              v-model="statusFilter"
              :items="statusFilterOptions"
              placeholder="All Status"
            />

            <OuiSelect
              v-model="gameTypeFilter"
              :items="gameTypeFilterOptions"
              placeholder="All Game Types"
            />
          </OuiGrid>
        </OuiCardBody>
      </OuiCard>

      <!-- Loading State with Skeleton Cards -->
      <OuiGrid v-if="pending && !gameServersData" :cols="{ sm: 1, md: 2, lg: 3 }" gap="lg">
        <GameServerCard
          v-for="i in 6"
          :key="i"
          :loading="true"
        />
      </OuiGrid>

      <!-- Empty State -->
      <OuiStack
        v-else-if="filteredGameServers.length === 0"
        align="center"
        gap="lg"
        class="text-center py-20"
      >
        <OuiBox
          class="inline-flex items-center justify-center w-20 h-20 rounded-xl bg-surface-muted/50 ring-1 ring-border-muted"
        >
          <CubeIcon class="h-10 w-10 text-secondary" />
        </OuiBox>
        <OuiStack align="center" gap="sm">
          <OuiText as="h3" size="xl" weight="semibold" color="primary">
            No game servers found
          </OuiText>
          <OuiBox class="max-w-md">
            <OuiText color="secondary">
              {{
                searchQuery || statusFilter || gameTypeFilter
                  ? "Try adjusting your filters to see more results."
                  : "Get started by creating your first game server."
              }}
            </OuiText>
          </OuiBox>
        </OuiStack>
        <OuiButton
          color="primary"
          class="gap-2 shadow-lg shadow-primary/20"
          @click="showCreateDialog = true"
        >
          <PlusIcon class="h-4 w-4" />
          <OuiText as="span" size="sm" weight="medium"
            >Create Your First Game Server</OuiText
          >
        </OuiButton>
      </OuiStack>

      <OuiGrid v-else :cols="{ sm: 1, md: 2, lg: 3 }" gap="lg">
        <GameServerCard
          v-for="gameServer in filteredGameServers"
          :key="gameServer.id"
          :game-server="gameServer"
          @refresh="refreshGameServers"
        />
      </OuiGrid>
    </OuiStack>

    <!-- Create Game Server Dialog -->
    <OuiDialog
      v-model:open="showCreateDialog"
      title="Create New Game Server"
      description="Deploy a game server with pay-as-you-go pricing"
    >
      <form @submit.prevent="createGameServer">
        <OuiStack gap="lg">
          <!-- Error display -->
          <ErrorAlert
            v-if="createError"
            :error="createError"
            title="Unable to create game server"
            :hint="createErrorHint"
          />

          <OuiInput
            v-model="newGameServer.name"
            label="Server Name"
            placeholder="my-minecraft-server"
            required
          />

          <OuiSelect
            v-model="newGameServer.gameType"
            label="Game Type"
            :items="gameTypeOptions"
            required
          />

          <OuiInput
            v-model="newGameServer.memoryGBStr"
            label="Memory (GB) - Max Limit"
            type="number"
            min="0.5"
            max="32"
            step="0.5"
            placeholder="2"
            required
          />

          <OuiInput
            v-model="newGameServer.cpuCoresStr"
            label="vCPU Cores - Max Limit"
            type="number"
            min="0.25"
            max="8"
            step="0.25"
            placeholder="1"
            required
          />

          <OuiInput
            v-model="newGameServer.serverVersion"
            label="Server Version (Optional)"
            placeholder="e.g., 1.20.1 for Minecraft"
          />

          <OuiTextarea
            v-model="newGameServer.description"
            label="Description (Optional)"
            placeholder="A brief description of your game server"
          />
        </OuiStack>
      </form>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="showCreateDialog = false">
            Cancel
          </OuiButton>
          <OuiButton
            color="primary"
            :loading="isCreating"
            @click="createGameServer"
            class="gap-2 shadow-lg shadow-primary/20"
          >
            <CubeIcon class="h-4 w-4" />
            <OuiText as="span" size="sm" weight="medium">Create Server</OuiText>
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowPathIcon,
  CpuChipIcon,
  EyeIcon,
  MagnifyingGlassIcon,
  PauseCircleIcon,
  PlayIcon,
  PlusIcon,
  CubeIcon,
  ServerIcon,
  StopIcon,
  CircleStackIcon,
} from "@heroicons/vue/24/outline";

import { useConnectClient } from "~/lib/connect-client";
import ErrorAlert from "~/components/ErrorAlert.vue";
import GameServerCard from "~/components/gameserver/GameServerCard.vue";
import { useOrganizationsStore } from "~/stores/organizations";
import { useOrganizationId } from "~/composables/useOrganizationId";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import { date } from "@obiente/proto/utils";
import { useToast } from "~/composables/useToast";
import { GameServerService, GameType } from "@obiente/proto";
import { ConnectError, Code } from "@connectrpc/connect";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();
const { toast } = useToast();
const auth = useAuth();
const orgsStore = useOrganizationsStore();
const client = useConnectClient(GameServerService);

// Error handling
const listError = ref<Error | null>(null);

// Compute hint for create error - only show "logged in" hint if user is not authenticated
const createErrorHint = computed(() => {
  if (!createError.value) return undefined;
  
  // If error is PermissionDenied and user is authenticated, don't mention logging in
  if (createError.value instanceof ConnectError) {
    if (createError.value.code === Code.PermissionDenied && auth.isAuthenticated) {
      return "You don't have permission to create game servers. Contact your organization administrator to grant you the necessary permissions.";
    }
    if (createError.value.code === Code.Unauthenticated) {
      return "Please log in and try again.";
    }
  }
  
  // Default hint for other errors
  return "Please try again. If the problem persists, contact support.";
});

// Check for organizationId in query params (from superadmin navigation)
if (route.query.organizationId && typeof route.query.organizationId === "string") {
  orgsStore.switchOrganization(route.query.organizationId);
}

// Get organizationId using SSR-compatible composable
const organizationId = useOrganizationId();

// Fetch game servers via optimized client fetch
const { data: gameServersData, pending, refresh: refreshGameServers } = await useClientFetch(
  () => `game-servers-list-${organizationId.value}`,
  async () => {
    try {
      const response = await client.listGameServers({
        organizationId: organizationId.value || undefined,
      });
      return response.gameServers || [];
    } catch (error) {
      console.error("Failed to list game servers:", error);
      listError.value = error as Error;
      return [];
    }
  }
);

// Convert game servers to local format
const gameServers = computed(() => {
  return (gameServersData.value || []).map((gs) => {
    const updatedAt = gs.updatedAt 
      ? (typeof gs.updatedAt === 'string' 
          ? gs.updatedAt 
          : date(gs.updatedAt)?.toISOString() || new Date().toISOString())
      : new Date().toISOString();
    
    // Convert status from enum number to string if needed
    let status: string = "CREATED";
    if (gs.status !== undefined && gs.status !== null) {
      if (typeof gs.status === 'number') {
        // Map GameServerStatus enum values (from proto)
        // Note: The detail page uses a different mapping, but proto shows:
        // 0: GAME_SERVER_STATUS_UNSPECIFIED, 1: CREATED, 2: STARTING, 3: RUNNING, 4: STOPPING, 5: STOPPED, 6: FAILED, 7: RESTARTING
        const statusMap: Record<number, string> = {
          0: 'CREATED', // GAME_SERVER_STATUS_UNSPECIFIED -> treat as CREATED
          1: 'CREATED',
          2: 'STARTING',
          3: 'RUNNING',
          4: 'STOPPING',
          5: 'STOPPED',
          6: 'FAILED',
          7: 'RESTARTING',
        };
        status = statusMap[gs.status] || 'CREATED';
      } else if (typeof gs.status === 'string') {
        status = gs.status;
      }
    }
    
    return {
      id: gs.id,
      name: gs.name,
      gameType: gs.gameType?.toString(),
      status: status,
      port: gs.port,
      cpuCores: gs.cpuCores,
      memoryBytes: gs.memoryBytes ? Number(gs.memoryBytes) : undefined,
      updatedAt: updatedAt,
    };
  });
});

// Filters
const searchQuery = ref("");
const statusFilter = ref<string>("");
const gameTypeFilter = ref<string>("");
const showCreateDialog = ref(false);
const isCreating = ref(false);
const createError = ref<Error | null>(null);

// New game server form
const newGameServer = ref({
  name: "",
  gameType: GameType.MINECRAFT,
  memoryGBStr: "2",
  cpuCoresStr: "1",
  serverVersion: "",
  description: "",
});

// Game type options
const gameTypeOptions = [
  { label: "Minecraft", value: GameType.MINECRAFT },
  { label: "Minecraft Java", value: GameType.MINECRAFT_JAVA },
  { label: "Minecraft Bedrock", value: GameType.MINECRAFT_BEDROCK },
  { label: "Valheim", value: GameType.VALHEIM },
  { label: "Terraria", value: GameType.TERRARIA },
  { label: "Rust", value: GameType.RUST },
  { label: "Counter-Strike 2", value: GameType.CS2 },
  { label: "Team Fortress 2", value: GameType.TF2 },
  { label: "ARK: Survival Evolved", value: GameType.ARK },
  { label: "Conan Exiles", value: GameType.CONAN },
  { label: "7 Days to Die", value: GameType.SEVEN_DAYS },
  { label: "Factorio", value: GameType.FACTORIO },
  { label: "Space Engineers", value: GameType.SPACED_ENGINEERS },
  { label: "Other", value: GameType.OTHER },
];

// Status filter options
const statusFilterOptions = [
  { label: "All Status", value: "" },
  { label: "Running", value: "RUNNING" },
  { label: "Stopped", value: "STOPPED" },
  { label: "Starting", value: "STARTING" },
  { label: "Stopping", value: "STOPPING" },
  { label: "Error", value: "ERROR" },
];

// Game type filter options
const gameTypeFilterOptions = [
  { label: "All Game Types", value: "" },
  { label: "Minecraft", value: "MINECRAFT" },
  { label: "Minecraft Java", value: "MINECRAFT_JAVA" },
  { label: "Minecraft Bedrock", value: "MINECRAFT_BEDROCK" },
  { label: "Valheim", value: "VALHEIM" },
  { label: "Terraria", value: "TERRARIA" },
  { label: "Rust", value: "RUST" },
  { label: "Counter-Strike 2", value: "CS2" },
  { label: "Team Fortress 2", value: "TF2" },
  { label: "ARK", value: "ARK" },
  { label: "Conan Exiles", value: "CONAN" },
  { label: "7 Days to Die", value: "SEVEN_DAYS" },
  { label: "Factorio", value: "FACTORIO" },
  { label: "Space Engineers", value: "SPACED_ENGINEERS" },
  { label: "Other", value: "OTHER" },
];

// Filtered game servers
const filteredGameServers = computed(() => {
  let filtered = [...gameServers.value];

  // Search filter
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter(
      (gs) =>
        gs.name.toLowerCase().includes(query) ||
        (gs.gameType && gs.gameType.toLowerCase().includes(query))
    );
  }

  // Status filter
  if (statusFilter.value) {
    filtered = filtered.filter((gs) => gs.status === statusFilter.value);
  }

  // Game type filter
  if (gameTypeFilter.value) {
    filtered = filtered.filter((gs) => gs.gameType === gameTypeFilter.value);
  }

  return filtered;
});

// Status metadata helper
const getStatusMeta = (status: string) => {
  const statusMap: Record<string, any> = {
    RUNNING: {
      label: "Running",
      badge: "success" as const,
      dotClass: "bg-success",
      cardClass: "border-success/20",
      barClass: "bg-success",
      pulseDot: false,
      beforeGradient: "",
    },
    STOPPED: {
      label: "Stopped",
      badge: "muted" as const,
      dotClass: "bg-muted",
      cardClass: "border-muted/20",
      barClass: "bg-muted",
      pulseDot: false,
      beforeGradient: "",
    },
    STARTING: {
      label: "Starting",
      badge: "warning" as const,
      dotClass: "bg-warning",
      cardClass: "border-warning/20",
      barClass: "bg-warning",
      pulseDot: true,
      beforeGradient: "",
    },
    STOPPING: {
      label: "Stopping",
      badge: "warning" as const,
      dotClass: "bg-warning",
      cardClass: "border-warning/20",
      barClass: "bg-warning",
      pulseDot: true,
      beforeGradient: "",
    },
    ERROR: {
      label: "Error",
      badge: "danger" as const,
      dotClass: "bg-danger",
      cardClass: "border-danger/20",
      barClass: "bg-danger",
      pulseDot: false,
      beforeGradient: "",
    },
  };

  return (
    statusMap[status] || {
      label: "Unknown",
      badge: "muted" as const,
      dotClass: "bg-muted",
      cardClass: "border-muted/20",
      barClass: "bg-muted",
      pulseDot: false,
      beforeGradient: "",
    }
  );
};

// Actions
const handleStart = async (id: string) => {
  // TODO: Implement start game server API call
  toast.success("Starting game server...");
  console.log("Start game server:", id);
};

const handleStop = async (id: string) => {
  // TODO: Implement stop game server API call
  toast.success("Stopping game server...");
  console.log("Stop game server:", id);
};

const handleRefresh = async (id: string) => {
  // TODO: Implement refresh game server status API call
  toast.success("Refreshing game server status...");
  console.log("Refresh game server:", id);
};

// Create game server
const createGameServer = async () => {
  if (!newGameServer.value.name || !organizationId.value) {
    toast.error("Please fill in all required fields");
    return;
  }

  isCreating.value = true;
  createError.value = null;

  try {
    const memoryGB = parseFloat(newGameServer.value.memoryGBStr) || 2;
    const cpuCores = parseFloat(newGameServer.value.cpuCoresStr) || 1;

    const request: any = {
      organizationId: organizationId.value,
      name: newGameServer.value.name,
      gameType: newGameServer.value.gameType,
      memoryBytes: BigInt(Math.floor(memoryGB * 1024 * 1024 * 1024)),
      cpuCores: cpuCores,
      envVars: {},
    };

    if (newGameServer.value.serverVersion) {
      request.serverVersion = newGameServer.value.serverVersion;
    }

    if (newGameServer.value.description) {
      request.description = newGameServer.value.description;
    }

    const response = await client.createGameServer(request);
    
    if (!response.gameServer) {
      throw new Error("No game server returned from API");
    }
    
    toast.success("Game server created successfully!");
    showCreateDialog.value = false;
    
    // Reset form
        newGameServer.value = {
          name: "",
          gameType: GameType.MINECRAFT,
          memoryGBStr: "2",
          cpuCoresStr: "1",
          serverVersion: "",
          description: "",
        };

    // Add to local list if not already there
    const gameServer = response.gameServer;
    if (!gameServers.value.find((gs) => gs.id === gameServer.id)) {
      const updatedAt = gameServer.updatedAt 
        ? (typeof gameServer.updatedAt === 'string' 
            ? gameServer.updatedAt 
            : date(gameServer.updatedAt)?.toISOString() || new Date().toISOString())
        : new Date().toISOString();
      
      // Refresh the list to include the new game server
      await refreshGameServers();
    }

    // Navigate to the detail page
    router.push(`/gameservers/${gameServer.id}`);
  } catch (error: any) {
    console.error("Failed to create game server:", error);
    createError.value = error;
    toast.error("Failed to create game server");
  } finally {
    isCreating.value = false;
  }
};

// Watch for organization changes
watch(
  () => organizationId.value,
  () => {
    // Reload game servers when organization changes
    refreshGameServers();
  }
);

// Helper function to get game type label
const getGameTypeLabel = (gameType: string | number | undefined): string => {
  if (gameType === undefined || gameType === null) {
    return "Unknown";
  }
  
  // Convert to number if it's a string
  const typeNum = typeof gameType === "string" ? parseInt(gameType, 10) : gameType;
  
  // Map GameType enum values to labels
  const types: Record<number, string> = {
    [GameType.MINECRAFT]: "Minecraft",
    [GameType.MINECRAFT_JAVA]: "Minecraft Java",
    [GameType.MINECRAFT_BEDROCK]: "Minecraft Bedrock",
    [GameType.VALHEIM]: "Valheim",
    [GameType.TERRARIA]: "Terraria",
    [GameType.RUST]: "Rust",
    [GameType.CS2]: "Counter-Strike 2",
    [GameType.TF2]: "Team Fortress 2",
    [GameType.ARK]: "ARK: Survival Evolved",
    [GameType.CONAN]: "Conan Exiles",
    [GameType.SEVEN_DAYS]: "7 Days to Die",
    [GameType.FACTORIO]: "Factorio",
    [GameType.SPACED_ENGINEERS]: "Space Engineers",
    [GameType.OTHER]: "Other",
  };
  
  return types[typeNum] || "Unknown";
};
</script>

