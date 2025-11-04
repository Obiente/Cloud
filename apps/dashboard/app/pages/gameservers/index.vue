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
          <OuiGrid cols="1" cols-md="3" gap="md">
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

      <OuiStack
        v-if="filteredGameServers.length === 0"
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

      <OuiGrid v-else cols="1" cols-md="2" :cols-2xl="3" gap="lg">
        <OuiCard
          v-for="gameServer in filteredGameServers"
          :key="gameServer.id"
          variant="default"
          hoverable
          :data-status="gameServer.status"
          :class="[
            'group relative overflow-hidden transition-all duration-300 hover:shadow-2xl',
            getStatusMeta(gameServer.status).cardClass,
            getStatusMeta(gameServer.status).beforeGradient,
          ]"
        >
          <div
            class="absolute top-0 left-0 right-0 h-1"
            :class="getStatusMeta(gameServer.status).barClass"
          />

          <OuiFlex direction="col" h="full" class="relative">
            <OuiCardHeader>
              <OuiFlex justify="between" align="center" gap="lg" wrap="wrap">
                <OuiStack gap="xs" class="min-w-0">
                  <OuiText
                    as="h3"
                    size="xl"
                    weight="semibold"
                    color="primary"
                    truncate
                    class="transition-colors group-hover:text-primary/90"
                  >
                    {{ gameServer.name }}
                  </OuiText>
                  <OuiFlex align="center" gap="xs">
                    <OuiText size="sm" color="secondary">
                      {{ gameServer.gameType || "Unknown" }}
                    </OuiText>
                  </OuiFlex>
                  <OuiFlex v-if="gameServer.port" align="center" gap="xs" class="mt-0.5">
                    <ServerIcon class="h-3 w-3 text-secondary" />
                    <OuiText size="xs" color="secondary"
                      >Port: {{ gameServer.port }}</OuiText
                    >
                  </OuiFlex>
                </OuiStack>
                <OuiFlex gap="sm" justify="end" wrap="wrap">
                  <OuiBadge :variant="getStatusMeta(gameServer.status).badge">
                    <span
                      class="inline-flex h-1.5 w-1.5 rounded-full"
                      :class="[
                        getStatusMeta(gameServer.status).dotClass,
                        getStatusMeta(gameServer.status).pulseDot
                          ? 'animate-pulse'
                          : '',
                      ]"
                    />
                    <OuiText
                      as="span"
                      size="xs"
                      weight="semibold"
                      transform="uppercase"
                      class="text-[11px]"
                    >
                      {{ getStatusMeta(gameServer.status).label }}
                    </OuiText>
                  </OuiBadge>
                </OuiFlex>
              </OuiFlex>
            </OuiCardHeader>

            <OuiCardBody class="flex-1">
              <OuiStack gap="md">
                <!-- Resource Usage -->
                <OuiStack gap="sm">
                  <OuiText size="sm" weight="semibold" color="primary">
                    Resources
                  </OuiText>
                  <OuiGrid cols="2" gap="sm">
                    <OuiStack gap="xs">
                      <OuiFlex align="center" gap="xs">
                        <CpuChipIcon class="h-3.5 w-3.5 text-secondary" />
                        <OuiText size="xs" color="secondary">vCPU</OuiText>
                      </OuiFlex>
                      <OuiText size="sm" weight="semibold" color="primary">
                        {{ gameServer.cpuCores || "N/A" }}
                      </OuiText>
                    </OuiStack>
                    <OuiStack gap="xs">
                      <OuiFlex align="center" gap="xs">
                        <CircleStackIcon class="h-3.5 w-3.5 text-secondary" />
                        <OuiText size="xs" color="secondary">Memory</OuiText>
                      </OuiFlex>
                      <OuiText size="sm" weight="semibold" color="primary">
                        <OuiByte :bytes="gameServer.memoryBytes || 0" />
                      </OuiText>
                    </OuiStack>
                  </OuiGrid>
                </OuiStack>

                <!-- Status Info -->
                <OuiFlex align="center" justify="between" class="pt-2 border-t border-border-muted">
                  <OuiText size="xs" color="secondary">
                    Updated
                    <OuiRelativeTime :timestamp="gameServer.updatedAt" />
                  </OuiText>
                </OuiFlex>
              </OuiStack>
            </OuiCardBody>

            <OuiCardFooter class="border-t border-border-muted">
              <OuiFlex justify="between" align="center" gap="sm" class="w-full">
                <OuiButton
                  variant="ghost"
                  size="sm"
                  class="gap-2"
                  @click="navigateTo(`/gameservers/${gameServer.id}`)"
                >
                  <EyeIcon class="h-4 w-4" />
                  View Details
                </OuiButton>
                <OuiFlex gap="xs">
                  <OuiButton
                    v-if="gameServer.status === 'RUNNING'"
                    variant="ghost"
                    size="sm"
                    class="gap-2"
                    @click.stop="handleStop(gameServer.id)"
                  >
                    <StopIcon class="h-4 w-4" />
                  </OuiButton>
                  <OuiButton
                    v-if="gameServer.status === 'STOPPED'"
                    variant="ghost"
                    size="sm"
                    class="gap-2"
                    @click.stop="handleStart(gameServer.id)"
                  >
                    <PlayIcon class="h-4 w-4" />
                  </OuiButton>
                  <OuiButton
                    variant="ghost"
                    size="sm"
                    class="gap-2"
                    @click.stop="handleRefresh(gameServer.id)"
                  >
                    <ArrowPathIcon class="h-4 w-4" />
                  </OuiButton>
                </OuiFlex>
              </OuiFlex>
            </OuiCardFooter>
          </OuiFlex>
        </OuiCard>
      </OuiGrid>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch } from "vue";
import { useRouter } from "vue-router";
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
import { useOrganizationsStore } from "~/stores/organizations";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
import { useToast } from "~/composables/useToast";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();
const { toast } = useToast();

// Error handling
const listError = ref<Error | null>(null);

// Organizations
const orgsStore = useOrganizationsStore();

// Check for organizationId in query params (from superadmin navigation)
if (route.query.organizationId && typeof route.query.organizationId === "string") {
  orgsStore.switchOrganization(route.query.organizationId);
}

const effectiveOrgId = computed(() => {
  return orgsStore.currentOrgId || "";
});

// Filters
const searchQuery = ref("");
const statusFilter = ref<string | undefined>(undefined);
const gameTypeFilter = ref<string | undefined>(undefined);
const showCreateDialog = ref(false);

// Status filter options
const statusFilterOptions = [
  { label: "All Status", value: undefined },
  { label: "Running", value: "RUNNING" },
  { label: "Stopped", value: "STOPPED" },
  { label: "Starting", value: "STARTING" },
  { label: "Stopping", value: "STOPPING" },
  { label: "Error", value: "ERROR" },
];

// Game type filter options (placeholder - will be populated from API)
const gameTypeFilterOptions = [
  { label: "All Game Types", value: undefined },
  { label: "Minecraft", value: "MINECRAFT" },
  { label: "Valheim", value: "VALHEIM" },
  { label: "Terraria", value: "TERRARIA" },
  { label: "Rust", value: "RUST" },
  { label: "CS2", value: "CS2" },
  { label: "Other", value: "OTHER" },
];

// Placeholder game servers data (will be replaced with API call)
const gameServers = ref<
  Array<{
    id: string;
    name: string;
    gameType?: string;
    status: string;
    port?: number;
    cpuCores?: number;
    memoryBytes?: number;
    updatedAt: string;
  }>
>([]);

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

// Watch for organization changes
watch(
  () => effectiveOrgId.value,
  () => {
    // TODO: Reload game servers when organization changes
    // loadGameServers();
  }
);
</script>

