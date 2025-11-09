<template>
  <ResourceCard
    :title="vps.name"
    :subtitle="`${vps.region} • ${sizeLabel}`"
    :status-meta="statusMeta"
    :icon="statusMeta.icon"
    :icon-class="statusMeta.iconClass"
    :resources="resources"
    :created-at="createdAtDate"
    :detail-url="`/vps/${vps.id}`"
    :is-actioning="isActioning"
  >
    <template #actions>
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
        variant="ghost"
        size="sm"
        icon-only
        @click.stop="handleDelete"
        title="Delete"
      >
        <TrashIcon class="h-4 w-4" />
      </OuiButton>
    </template>

    <template #info>
      <!-- IP Addresses -->
      <OuiStack v-if="ipAddresses.length > 0" gap="xs">
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
  import { computed, ref } from "vue";
  import {
    ServerIcon,
    PlayIcon,
    StopIcon,
    ArrowPathIcon,
    TrashIcon,
    CircleStackIcon,
  } from "@heroicons/vue/24/outline";
  import { VPSStatus, type VPSInstance } from "@obiente/proto";
  import { date } from "@obiente/proto/utils";
  import { useConnectClient } from "~/lib/connect-client";
  import { VPSService } from "@obiente/proto";
  import { useDialog } from "~/composables/useDialog";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import ResourceCard from "~/components/shared/ResourceCard.vue";

  interface Props {
    vps: VPSInstance;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<{
    refresh: [];
    delete: [];
  }>();

  const client = useConnectClient(VPSService);
  const { showAlert, showConfirm } = useDialog();
  const organizationId = useOrganizationId();
  const isActioning = ref(false);

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
  } as const;

  const statusMeta = computed(() => {
    const status = props.vps.status as VPSStatus;
    // Handle all status values, defaulting to STOPPED for unknown statuses
    if (status in STATUS_META) {
      return STATUS_META[status as keyof typeof STATUS_META];
    }
    // Default fallback for DELETING, DELETED, VPS_STATUS_UNSPECIFIED, etc.
    return STATUS_META[VPSStatus.STOPPED];
  });

  const sizeLabel = computed(() => {
    return props.vps.size || "Unknown";
  });

  const ipAddresses = computed(() => {
    return [...(props.vps.ipv4Addresses || []), ...(props.vps.ipv6Addresses || [])];
  });

  const canStart = computed(() => props.vps.status === VPSStatus.STOPPED);
  const canStop = computed(() => props.vps.status === VPSStatus.RUNNING);
  const canReboot = computed(() => props.vps.status === VPSStatus.RUNNING);

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
    if (!props.vps.createdAt) return new Date();
    return date(props.vps.createdAt);
  });

  const resources = computed(() => [
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
  ]);

  const handleStart = async () => {
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

  const handleDelete = async () => {
    const confirmed = await showConfirm({
      title: "Delete VPS Instance",
      message: `Are you sure you want to delete "${props.vps.name}"? This action cannot be undone.`,
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
      emit("delete");
    } catch (error) {
      await showAlert({
        title: "Failed to delete VPS",
        message: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };
</script>

