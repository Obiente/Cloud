<template>
  <ResourceCard
    :title="vps?.name || ''"
    :subtitle="vps ? `${vps.region} • ${sizeLabel}` : ''"
    :status-meta="statusMeta"
    :resources="resources"
    :created-at="createdAtDate"
    :detail-url="vps ? `/vps/${vps.id}` : undefined"
    :is-actioning="isActioning"
    :loading="loading"
  >
    <template #actions>
            <OuiButton
              v-if="canRetry"
              variant="ghost"
              size="sm"
              icon-only
              @click.stop="handleRetry"
              title="Retry Creation"
            >
              <ArrowPathIcon class="h-4 w-4" />
            </OuiButton>
            <OuiButton
              v-if="canStart"
              variant="ghost"
              size="sm"
              icon-only
              @click.stop="handleStart"
              title="Start"
            >
              <PlayIcon class="h-4 w-4" />
            </OuiButton>
            <OuiButton
              v-if="canStop"
              variant="ghost"
              size="sm"
              icon-only
              @click.stop="handleStop"
              title="Stop"
            >
              <StopIcon class="h-4 w-4" />
            </OuiButton>
            <OuiButton
              v-if="canReboot"
              variant="ghost"
              size="sm"
              icon-only
              @click.stop="handleReboot"
              title="Reboot"
            >
              <ArrowPathIcon class="h-4 w-4" />
            </OuiButton>
            <OuiButton
              v-if="!loading && vps && vps.status !== VPSStatus.DELETED && vps.status !== VPSStatus.DELETING"
              variant="ghost"
              size="sm"
              icon-only
              @click.stop="handleDelete"
              title="Delete"
            >
              <TrashIcon class="h-4 w-4" />
            </OuiButton>
            <OuiButton
              v-if="canDeleteDeleted"
              variant="ghost"
              size="sm"
              icon-only
              @click.stop="handleDelete"
              title="Delete Record"
            >
              <TrashIcon class="h-4 w-4" />
            </OuiButton>
    </template>

    <template #info>
      <!-- Provisioning Progress Status -->
      <OuiBox
        v-if="showProgress"
        p="md"
        rounded="xl"
        class="border backdrop-blur-sm"
        :class="progressClass"
      >
        <OuiStack gap="sm">
          <OuiFlex
            align="center"
            gap="sm"
            class="text-xs font-bold uppercase tracking-wider"
            :class="progressTextClass"
          >
            <Cog6ToothIcon
              v-if="!isProgressFailed"
              class="h-4 w-4 animate-spin"
            />
            <ExclamationTriangleIcon
              v-else
              class="h-4 w-4 text-danger"
            />
            <span :class="progressTextClass">
              {{ progressPhase || "Starting server setup..." }}
            </span>
          </OuiFlex>
          <div
            class="relative h-2 w-full overflow-hidden rounded-full"
            :class="progressBarBgClass"
          >
            <div
              v-if="isProgressFailed"
              class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
              :class="progressBarFillClass"
              :style="{ width: '100%' }"
            />
            <div
              v-else
              class="absolute inset-y-0 left-0 rounded-full transition-all duration-300"
              :class="progressBarFillClass"
              :style="{ width: `${progressValue || 0}%` }"
            />
          </div>
        </OuiStack>
      </OuiBox>
      
      <!-- IP Addresses Skeleton -->
      <OuiStack v-else-if="loading" gap="xs">
        <OuiText size="xs" weight="medium" color="secondary" class="opacity-50">IP Addresses</OuiText>
        <OuiFlex gap="xs" wrap="wrap">
          <OuiBadge variant="secondary" size="xs" class="opacity-30">
            <OuiSkeleton :width="randomTextWidthByType('short')" height="0.875rem" variant="text" class="bg-transparent" />
          </OuiBadge>
          <OuiBadge variant="secondary" size="xs" class="opacity-30">
            <OuiSkeleton :width="randomTextWidthByType('short')" height="0.875rem" variant="text" class="bg-transparent" />
          </OuiBadge>
        </OuiFlex>
      </OuiStack>
      
      <!-- IP Addresses Actual -->
      <OuiStack v-else-if="ipAddresses.length > 0" gap="xs">
        <OuiText size="xs" weight="medium" color="secondary">IP Addresses</OuiText>
        <OuiFlex gap="xs" wrap="wrap">
          <OuiBadge
            v-for="ip in ipAddresses.slice(0, 2)"
            :key="ip"
            variant="secondary"
            size="xs"
          >
            {{ ip }}
          </OuiBadge>
          <OuiBadge
            v-if="ipAddresses.length > 2"
            variant="secondary"
            size="xs"
          >
            +{{ ipAddresses.length - 2 }}
          </OuiBadge>
        </OuiFlex>
      </OuiStack>
    </template>
  </ResourceCard>
</template>

<script setup lang="ts">
  import { computed, ref, watch, onMounted, onBeforeUnmount, type ComputedRef } from "vue";
  import {
    ServerIcon,
    PlayIcon,
    StopIcon,
    ArrowPathIcon,
    TrashIcon,
    CircleStackIcon,
    Cog6ToothIcon,
    ExclamationTriangleIcon,
  } from "@heroicons/vue/24/outline";
  import { VPSStatus, type VPSInstance } from "@obiente/proto";
  import { date } from "@obiente/proto/utils";
  import { useConnectClient } from "~/lib/connect-client";
  import { VPSService } from "@obiente/proto";
  import { useDialog } from "~/composables/useDialog";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import ResourceCard from "~/components/shared/ResourceCard.vue";
  import OuiSkeleton from "~/components/oui/Skeleton.vue";
  import OuiBadge from "~/components/oui/Badge.vue";
  import OuiBox from "~/components/oui/Box.vue";
  import OuiStack from "~/components/oui/Stack.vue";
  import OuiFlex from "~/components/oui/Flex.vue";
  import OuiText from "~/components/oui/Text.vue";
  import { randomTextWidthByType, randomIconVariation } from "~/composables/useSkeletonVariations";
  import { useVPSProgress } from "~/composables/useVPSProgress";

  interface Props {
    vps?: VPSInstance;
    loading?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), {
    loading: false,
  });
  const emit = defineEmits<{
    refresh: [];
    delete: [vps: VPSInstance];
    retry: [vps: VPSInstance];
  }>();

  const client = useConnectClient(VPSService);
  const { showAlert, showConfirm } = useDialog();
  const organizationId = useOrganizationId();
  const isActioning = ref(false);

  // Generate random variations for skeleton icons
  const iconVar = randomIconVariation();

  // VPS progress tracking - initialize with placeholder values
  const vpsProgress = ref<ReturnType<typeof useVPSProgress> | null>(null);

  // Show progress when VPS is creating, starting, or recently created (within last 2 minutes)
  const showProgress = computed(() => {
    if (!props.vps || props.loading) return false;
    const status = props.vps.status as VPSStatus;
    
    // Show progress for CREATING or STARTING status
    if (status === VPSStatus.CREATING || status === VPSStatus.STARTING) {
      return true;
    }
    
    // Also show progress for RUNNING VPS that was just created (within last 2 minutes)
    // This handles cases where provisioning completes quickly and VPS is already RUNNING
    if (status === VPSStatus.RUNNING && props.vps.createdAt) {
      const createdAt = date(props.vps.createdAt);
      const twoMinutesAgo = Date.now() - 2 * 60 * 1000;
      if (createdAt.getTime() > twoMinutesAgo) {
        return true;
      }
    }
    
    return false;
  });

  // Progress values - access computed refs correctly
  const progressValue = computed(() => {
    if (!vpsProgress.value) return 0;
    // TypeScript auto-unwraps computed refs in some contexts, so we need to access .value
    const prog = vpsProgress.value.progress;
    // Check if it's already unwrapped (number) or still a ComputedRef
    if (typeof prog === 'number') return prog;
    return (prog as unknown as ComputedRef<number>).value;
  });

  const progressPhase = computed(() => {
    if (!vpsProgress.value) return "Starting server setup...";
    const phase = vpsProgress.value.currentPhase;
    if (typeof phase === 'string') return phase;
    return (phase as unknown as ComputedRef<string>).value;
  });

  const isProgressFailed = computed(() => {
    if (!vpsProgress.value) return false;
    const failed = vpsProgress.value.isFailed;
    if (typeof failed === 'boolean') return failed;
    return (failed as unknown as ComputedRef<boolean>).value;
  });

  // Progress styling
  const progressClass = computed(() => {
    if (isProgressFailed.value) {
      return "border-danger/20 bg-danger/5";
    }
    return "border-warning/20 bg-warning/5";
  });

  const progressTextClass = computed(() => {
    if (isProgressFailed.value) {
      return "text-danger";
    }
    return "text-warning";
  });

  const progressBarBgClass = computed(() => {
    if (isProgressFailed.value) {
      return "bg-danger/10";
    }
    return "bg-warning/10";
  });

  const progressBarFillClass = computed(() => {
    if (isProgressFailed.value) {
      return "bg-danger";
    }
    return "bg-warning";
  });

  // Initialize and manage progress tracking
  watch(
    () => [showProgress.value, props.vps?.id, organizationId.value],
    ([shouldShow, vpsId, orgId]) => {
      // Clean up previous progress tracker
      if (vpsProgress.value) {
        vpsProgress.value.stopStreaming();
        vpsProgress.value = null;
      }

      // Initialize new progress tracker if needed
      if (shouldShow && vpsId && orgId) {
        const progress = useVPSProgress({
          vpsId: vpsId as string,
          organizationId: orgId as string,
        });
        vpsProgress.value = progress;
        progress.startStreaming();
      }
    },
    { immediate: true }
  );

  onBeforeUnmount(() => {
    if (vpsProgress.value) {
      vpsProgress.value.stopStreaming();
    }
  });

  const STATUS_META = {
    [VPSStatus.RUNNING]: {
      badge: "success",
      label: "Running",
      cardClass: "hover:ring-1 hover:ring-success/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-success/20 before:via-success/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-success to-success/70",
      icon: ServerIcon,
      iconClass: "text-success",
    },
    [VPSStatus.STOPPED]: {
      badge: "danger",
      label: "Stopped",
      cardClass: "hover:ring-1 hover:ring-danger/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-danger/20 before:via-danger/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-danger to-danger/60",
      icon: StopIcon,
      iconClass: "text-danger",
    },
    [VPSStatus.CREATING]: {
      badge: "warning",
      label: "Creating",
      cardClass: "hover:ring-1 hover:ring-warning/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
      icon: ServerIcon,
      iconClass: "text-warning",
    },
    [VPSStatus.STARTING]: {
      badge: "warning",
      label: "Starting",
      cardClass: "hover:ring-1 hover:ring-warning/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
      icon: PlayIcon,
      iconClass: "text-warning",
    },
    [VPSStatus.STOPPING]: {
      badge: "warning",
      label: "Stopping",
      cardClass: "hover:ring-1 hover:ring-warning/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
      icon: StopIcon,
      iconClass: "text-warning",
    },
    [VPSStatus.REBOOTING]: {
      badge: "warning",
      label: "Rebooting",
      cardClass: "hover:ring-1 hover:ring-warning/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
      icon: ArrowPathIcon,
      iconClass: "text-warning",
    },
    [VPSStatus.FAILED]: {
      badge: "danger",
      label: "Failed",
      cardClass: "hover:ring-1 hover:ring-danger/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-danger/20 before:via-danger/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-danger to-danger/60",
      icon: ServerIcon,
      iconClass: "text-danger",
    },
    [VPSStatus.DELETING]: {
      badge: "warning",
      label: "Deleting",
      cardClass: "hover:ring-1 hover:ring-warning/30",
      beforeGradient:
        "before:absolute before:inset-0 before:-z-10 before:rounded-[inherit] before:bg-gradient-to-br before:from-warning/20 before:via-warning/10 before:to-transparent before:opacity-0 before:transition-opacity before:duration-300 hover:before:opacity-100",
      barClass: "bg-gradient-to-r from-warning to-warning/60 animate-pulse",
      icon: TrashIcon,
      iconClass: "text-warning",
    },
    [VPSStatus.DELETED]: {
      badge: "secondary",
      label: "Deleted",
      cardClass: "hover:ring-1 hover:ring-secondary/30 opacity-60",
      beforeGradient: "",
      barClass: "bg-gradient-to-r from-secondary to-secondary/60",
      icon: ServerIcon,
      iconClass: "text-secondary",
    },
  } as const;

  const statusMeta = computed(() => {
    if (!props.vps || props.loading) {
      return STATUS_META[VPSStatus.STOPPED];
    }
    const status = props.vps.status as VPSStatus;
    // Handle all status values, defaulting to STOPPED for unknown statuses
    if (status in STATUS_META) {
      return STATUS_META[status as keyof typeof STATUS_META];
    }
    // Default fallback for VPS_STATUS_UNSPECIFIED, etc.
    return STATUS_META[VPSStatus.STOPPED];
  });

  const sizeLabel = computed(() => {
    if (!props.vps || props.loading) return "Unknown";
    return props.vps.size || "Unknown";
  });

  const ipAddresses = computed(() => {
    if (!props.vps || props.loading) return [];
    return [...(props.vps.ipv4Addresses || []), ...(props.vps.ipv6Addresses || [])];
  });

  const canStart = computed(() => {
    if (props.loading || !props.vps) return false;
    return props.vps.status === VPSStatus.STOPPED;
  });
  const canStop = computed(() => {
    if (props.loading || !props.vps) return false;
    return props.vps.status === VPSStatus.RUNNING;
  });
  const canReboot = computed(() => {
    if (props.loading || !props.vps) return false;
    return props.vps.status === VPSStatus.RUNNING;
  });
  const canRetry = computed(() => {
    if (props.loading || !props.vps) return false;
    return props.vps.status === VPSStatus.FAILED;
  });
  const canDeleteDeleted = computed(() => {
    if (props.loading || !props.vps) return false;
    return props.vps.status === VPSStatus.DELETED;
  });

  const formatMemory = (bytes: bigint | number | undefined) => {
    if (!bytes) return "0 GB";
    const gb = Number(bytes) / (1024 * 1024 * 1024);
    return `${gb.toFixed(1)} GB`;
  };

  const formatDisk = (bytes: bigint | number | undefined) => {
    if (!bytes) return "0 GB";
    const gb = Number(bytes) / (1024 * 1024 * 1024);
    return `${gb.toFixed(0)} GB`;
  };

  const createdAtDate = computed(() => {
    if (!props.vps || props.loading) return new Date();
    if (!props.vps.createdAt) return new Date();
    return date(props.vps.createdAt);
  });

  const resources = computed(() => {
    if (props.loading || !props.vps) {
      return [
        { icon: ServerIcon, label: "CPU" },
        { icon: CircleStackIcon, label: "Memory" },
      ];
    }
    return [
      {
        icon: ServerIcon,
        label: `${props.vps.cpuCores} CPU`,
      },
    {
      icon: CircleStackIcon,
      label: formatMemory(props.vps.memoryBytes),
    },
    {
      icon: CircleStackIcon,
      label: formatDisk(props.vps.diskBytes),
    },
    ];
  });

  const handleStart = async () => {
    if (!props.vps) return;
    isActioning.value = true;
    try {
      await client.startVPS({
        organizationId: organizationId.value || "",
        vpsId: props.vps.id,
      });
      emit("refresh");
    } catch (error) {
      await showAlert({
        title: "Failed to start VPS",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      isActioning.value = false;
    }
  };

  const handleStop = async () => {
    if (!props.vps) return;
    const confirmed = await showConfirm({
      title: "Stop VPS Instance",
      message: `Are you sure you want to stop "${props.vps.name}"?`,
      confirmLabel: "Stop",
      cancelLabel: "Cancel",
    });

    if (!confirmed) return;

    isActioning.value = true;
    try {
      await client.stopVPS({
        organizationId: organizationId.value || "",
        vpsId: props.vps.id,
      });
      emit("refresh");
    } catch (error) {
      await showAlert({
        title: "Failed to stop VPS",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      isActioning.value = false;
    }
  };

  const handleReboot = async () => {
    if (!props.vps) return;
    const confirmed = await showConfirm({
      title: "Reboot VPS Instance",
      message: `Are you sure you want to reboot "${props.vps.name}"?`,
      confirmLabel: "Reboot",
      cancelLabel: "Cancel",
    });

    if (!confirmed) return;

    isActioning.value = true;
    try {
      await client.rebootVPS({
        organizationId: organizationId.value || "",
        vpsId: props.vps.id,
      });
      emit("refresh");
    } catch (error) {
      await showAlert({
        title: "Failed to reboot VPS",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    } finally {
      isActioning.value = false;
    }
  };

  const handleRetry = () => {
    if (!props.vps) return;
    emit("retry", props.vps);
  };

  const handleDelete = async () => {
    if (!props.vps) return;
    
    const isDeleted = props.vps.status === VPSStatus.DELETED;
    const title = isDeleted ? "Delete VPS Record" : "Delete VPS Instance";
    const message = isDeleted
      ? `Are you sure you want to permanently delete the record for "${props.vps.name}"? The VPS has already been removed from the infrastructure. This action cannot be undone.`
      : `Are you sure you want to delete "${props.vps.name}"? This action cannot be undone.`;
    
    const confirmed = await showConfirm({
      title,
      message,
      confirmLabel: "Delete",
      cancelLabel: "Cancel",
      variant: "danger",
    });

    if (!confirmed) return;

    try {
      await client.deleteVPS({
        organizationId: organizationId.value || "",
        vpsId: props.vps.id,
        force: false,
      });
      emit("delete", props.vps);
    } catch (error) {
      await showAlert({
        title: "Failed to delete VPS",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };
</script>

