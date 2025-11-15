<template>
  <ResourceCard
    :title="gameServer.name"
    :subtitle="gameTypeLabel"
    :status-meta="statusMeta"
    :resources="resources"
    :created-at="updatedAtDate"
    :detail-url="`/gameservers/${gameServer.id}`"
    :is-actioning="isActioning"
  >
    <template #subtitle>
      <OuiStack gap="xs">
        <OuiText size="sm" color="secondary">
          {{ gameTypeLabel }}
        </OuiText>
        <OuiFlex v-if="gameServer.port" align="center" gap="xs">
          <ServerIcon class="h-3 w-3 text-secondary" />
          <OuiText size="xs" color="secondary">Port: {{ gameServer.port }}</OuiText>
        </OuiFlex>
      </OuiStack>
    </template>

    <template #actions>
      <OuiButton
        v-if="gameServer.status === 'RUNNING'"
        variant="ghost"
        size="sm"
        icon-only
        @click.stop="handleStop"
        title="Stop"
      >
        <StopIcon class="h-4 w-4" />
      </OuiButton>
      <OuiButton
        v-if="gameServer.status === 'STOPPED'"
        variant="ghost"
        size="sm"
        icon-only
        @click.stop="handleStart"
        title="Start"
      >
        <PlayIcon class="h-4 w-4" />
      </OuiButton>
      <OuiButton
        variant="ghost"
        size="sm"
        icon-only
        @click.stop="handleRefresh"
        title="Refresh"
      >
        <ArrowPathIcon class="h-4 w-4" />
      </OuiButton>
    </template>
  </ResourceCard>
</template>

<script setup lang="ts">
  import { computed, ref } from "vue";
  import {
    ServerIcon,
    PlayIcon,
    StopIcon,
    ArrowPathIcon,
    CpuChipIcon,
    CircleStackIcon,
  } from "@heroicons/vue/24/outline";
  import { GameType } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import { GameServerService } from "@obiente/proto";
  import { useDialog } from "~/composables/useDialog";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import ResourceCard from "~/components/shared/ResourceCard.vue";

  interface GameServer {
    id: string;
    name: string;
    gameType?: string;
    status: string;
    port?: number;
    cpuCores?: number;
    memoryBytes?: number;
    updatedAt?: string;
  }

  interface Props {
    gameServer: GameServer;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<{
    refresh: [];
  }>();

  const client = useConnectClient(GameServerService);
  const { showAlert } = useDialog();
  const isActioning = ref(false);

  const getStatusMeta = (status: string) => {
    const statusMap: Record<string, any> = {
      RUNNING: {
        badge: "success" as const,
        label: "Running",
        cardClass: "hover:ring-1 hover:ring-success/30",
        beforeGradient:
          "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-success/20 before:via-success/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
        barClass: "bg-gradient-to-r from-success to-success/70",
        iconClass: "text-success",
      },
      STOPPED: {
        badge: "danger" as const,
        label: "Stopped",
        cardClass: "hover:ring-1 hover:ring-danger/30",
        beforeGradient:
          "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-danger/20 before:via-danger/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
        barClass: "bg-gradient-to-r from-danger to-danger/60",
        iconClass: "text-danger",
      },
      CREATED: {
        badge: "warning" as const,
        label: "Created",
        cardClass: "hover:ring-1 hover:ring-warning/30",
        beforeGradient:
          "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
        barClass: "bg-gradient-to-r from-warning to-warning/60",
        iconClass: "text-warning",
      },
    };

    return statusMap[status] || statusMap.STOPPED;
  };

  const statusMeta = computed(() => getStatusMeta(props.gameServer.status));

  const gameTypeLabel = computed((): string => {
    const gameType = props.gameServer.gameType;
    if (!gameType) return "Unknown";
    
    const typeMap: Record<string, string> = {
      [GameType.MINECRAFT]: "Minecraft",
      [GameType.MINECRAFT_JAVA]: "Minecraft Java",
      [GameType.MINECRAFT_BEDROCK]: "Minecraft Bedrock",
      [GameType.VALHEIM]: "Valheim",
      [GameType.TERRARIA]: "Terraria",
      [GameType.RUST]: "Rust",
      [GameType.CS2]: "Counter-Strike 2",
      [GameType.TF2]: "Team Fortress 2",
      [GameType.ARK]: "ARK",
      [GameType.CONAN]: "Conan Exiles",
      [GameType.SEVEN_DAYS]: "7 Days to Die",
      [GameType.FACTORIO]: "Factorio",
      [GameType.SPACED_ENGINEERS]: "Space Engineers",
      [GameType.OTHER]: "Other",
    };

    return typeMap[gameType] || String(gameType);
  });

  const updatedAtDate = computed(() => {
    if (!props.gameServer.updatedAt) return new Date();
    return new Date(props.gameServer.updatedAt);
  });

  const formatMemory = (bytes: number | undefined) => {
    if (!bytes) return "0 GB";
    const gb = bytes / (1024 * 1024 * 1024);
    return `${gb.toFixed(1)} GB`;
  };

  const resources = computed(() => [
    {
      icon: CpuChipIcon,
      label: `${props.gameServer.cpuCores || "N/A"} vCPU`,
    },
    {
      icon: CircleStackIcon,
      label: formatMemory(props.gameServer.memoryBytes),
    },
  ]);

  const handleStart = async () => {
    isActioning.value = true;
    try {
      await client.startGameServer({
        gameServerId: props.gameServer.id,
      });
      emit("refresh");
    } catch (error: any) {
      const errorMessage = error?.message || "Unknown error";
      
      // Check for common configuration errors
      let hint = "";
      if (errorMessage.includes("exited immediately") || errorMessage.includes("container exit")) {
        hint = "The container may be missing required environment variables. Check the game server settings.";
        
        // Add specific hint for CS2 servers
        const gameTypeNum = typeof props.gameServer.gameType === 'number' 
          ? props.gameServer.gameType 
          : (props.gameServer.gameType ? Number(props.gameServer.gameType) : undefined);
        if (gameTypeNum === GameType.CS2 && errorMessage.includes("exit")) {
          hint = "CS2 servers require a Steam Game Server Login Token (SRCDS_TOKEN). Configure it in the game server settings.";
        }
      }
      
      const message = hint ? `${hint}\n\nError: ${errorMessage}` : errorMessage;
      await showAlert({
        title: "Failed to start game server",
        message: message,
      });
    } finally {
      isActioning.value = false;
    }
  };

  const handleStop = async () => {
    isActioning.value = true;
    try {
      await client.stopGameServer({
        gameServerId: props.gameServer.id,
      });
      emit("refresh");
    } catch (error) {
      await showAlert({
        title: "Failed to stop game server",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      isActioning.value = false;
    }
  };

  const handleRefresh = () => {
    emit("refresh");
  };
</script>

