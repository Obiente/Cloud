<template>
  <OuiFlex align="center" gap="sm">
    <OuiText v-if="showLabel" size="sm" color="muted">{{ label }}:</OuiText>
    <OuiSelect
      v-model="selectedValue"
      :items="containerOptions"
      :placeholder="placeholder"
      :disabled="isLoading || containers.length === 0 || disabled"
      :style="style"
      @update:model-value="handleChange"
    />
    <OuiFlex v-if="showSelectedInfo && selectedContainer" align="center" gap="sm" wrap="wrap">
      <OuiText size="sm" color="muted">{{ selectedInfoText }}</OuiText>
      <OuiText size="sm" color="muted">{{ getContainerName(selectedContainer) }}</OuiText>
      <OuiBadge
        :variant="getStatusVariant(selectedContainer.status)"
        size="xs"
        :pill="false"
        style="border-radius: 4px;"
      >
        {{ getStatusLabel(selectedContainer.status) }}
      </OuiBadge>
    </OuiFlex>
  </OuiFlex>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, nextTick } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";

interface Props {
  deploymentId: string;
  organizationId?: string;
  modelValue?: string; // Empty string means "first container" (default)
  label?: string;
  showLabel?: boolean;
  placeholder?: string;
  showSelectedInfo?: boolean;
  selectedInfoText?: string; // Custom text for selected info, defaults to "Viewing: {service}"
  style?: string | Record<string, string>;
  disabled?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  label: "Service",
  showLabel: true,
  placeholder: "Select service",
  showSelectedInfo: false,
  style: () => ({ minWidth: "200px" }),
  disabled: false,
});

const emit = defineEmits<{
  "update:modelValue": [value: string];
  change: [container: { containerId: string; serviceName?: string } | null];
}>();

const orgsStore = useOrganizationsStore();
const effectiveOrgId = computed(
  () => props.organizationId || orgsStore.currentOrgId || ""
);

const client = useConnectClient(DeploymentService);
const containers = ref<Array<{ containerId: string; serviceName?: string; status?: string }>>([]);
const isLoading = ref(false);
const selectedValue = ref<string>(props.modelValue || "");

// Container options for select dropdown
const containerOptions = computed(() => {
  const options: Array<{ label: string; value: string }> = [];
  
  // Add first container as default option if containers exist
  if (containers.value.length > 0) {
    const firstContainer = containers.value[0];
    if (firstContainer) {
      const firstLabel = formatContainerLabel(firstContainer);
      options.push({ label: firstLabel, value: "" });
      
      // Add remaining containers
      for (let i = 1; i < containers.value.length; i++) {
        const container = containers.value[i];
        if (container) {
          const label = formatContainerLabel(container);
          const value = container.serviceName || container.containerId;
          options.push({ label, value });
        }
      }
    }
  } else {
    // No containers available - show placeholder option
    options.push({ label: "No containers available", value: "" });
  }
  
  return options;
});

// Format container label with status indicator
const formatContainerLabel = (container: { containerId: string; serviceName?: string; status?: string }) => {
  const name = container.serviceName || container.containerId.substring(0, 12);
  const status = container.status || "unknown";
  
  // Format status text - capitalize first letter, handle common statuses
  let statusText = status.toLowerCase();
  if (statusText === "exited") {
    statusText = "exited";
  } else if (statusText === "stopped") {
    statusText = "stopped";
  } else if (statusText === "running") {
    statusText = "running";
  } else if (statusText === "starting" || statusText === "restarting") {
    statusText = statusText;
  } else {
    // Capitalize first letter
    statusText = statusText.charAt(0).toUpperCase() + statusText.slice(1);
  }
  
  // Always show status for clarity
  return `${name} (${statusText})`;
};

// Selected container info
const selectedContainer = computed(() => {
  if (!selectedValue.value) {
    // Return first container if no value selected (default)
    return containers.value.length > 0 ? containers.value[0] : null;
  }
  return (
    containers.value.find(
      (c) => (c.serviceName || c.containerId) === selectedValue.value
    ) || null
  );
});

const selectedLabel = computed(() => {
  if (!selectedValue.value) {
    // Return first container's name with status if no selection
    if (containers.value.length > 0) {
      const firstContainer = containers.value[0];
      if (firstContainer) {
        return formatContainerLabel(firstContainer);
      }
    }
    return "No containers";
  }
  const container = selectedContainer.value;
  if (!container) return "Unknown";
  return formatContainerLabel(container);
});

// Get container name (without status)
const getContainerName = (container: { containerId: string; serviceName?: string; status?: string } | null) => {
  if (!container) return "Unknown";
  return container.serviceName || container.containerId.substring(0, 12);
};

// Get status label text
const getStatusLabel = (status?: string) => {
  const statusText = (status || "unknown").toLowerCase();
  if (statusText === "exited") return "exited";
  if (statusText === "stopped") return "stopped";
  if (statusText === "running") return "running";
  if (statusText === "starting" || statusText === "restarting") return statusText;
  // Capitalize first letter
  return statusText.charAt(0).toUpperCase() + statusText.slice(1);
};

// Get badge variant based on status
const getStatusVariant = (status?: string): "success" | "warning" | "danger" | "secondary" => {
  const statusText = (status || "unknown").toLowerCase();
  if (statusText === "running") return "success";
  if (statusText === "stopped" || statusText === "exited") return "danger";
  if (statusText === "starting" || statusText === "restarting") return "warning";
  return "secondary";
};

// Load containers for this deployment
const loadContainers = async () => {
  if (!props.deploymentId || !effectiveOrgId.value) return;

  isLoading.value = true;
  try {
    const res = await (client as any).listDeploymentContainers({
      deploymentId: props.deploymentId,
      organizationId: effectiveOrgId.value,
    });

    if (res?.containers) {
      containers.value = res.containers
        .map((c: any) => ({
          containerId: c.containerId,
          serviceName: c.serviceName || undefined,
          status: c.status || "unknown",
        }));
      // Sort: running containers first, then starting/restarting, then stopped/exited, then others
      containers.value.sort((a, b) => {
        const statusA = (a.status || "unknown").toLowerCase();
        const statusB = (b.status || "unknown").toLowerCase();
        
        // Priority order: running > starting/restarting > stopped/exited > others
        const priority = (status: string) => {
          if (status === "running") return 0;
          if (status === "starting" || status === "restarting") return 1;
          if (status === "stopped" || status === "exited") return 2;
          return 3;
        };
        
        const priorityA = priority(statusA);
        const priorityB = priority(statusB);
        
        if (priorityA !== priorityB) {
          return priorityA - priorityB;
        }
        
        // If same priority, sort alphabetically by name
        const nameA = a.serviceName || a.containerId;
        const nameB = b.serviceName || b.containerId;
        return nameA.localeCompare(nameB);
      });
    }
    
    // Emit initial selection after containers load
    // Use nextTick to ensure parent components are ready
    nextTick(() => {
      // If no explicit modelValue was provided (or it's empty string), emit the first container
      if (!props.modelValue || props.modelValue === "") {
        const firstContainer = containers.value[0] || null;
        emit("change", firstContainer);
      } else {
        // If a specific modelValue was provided, emit the corresponding container
        const container = selectedContainer.value;
        emit("change", container ?? null);
      }
    });
  } catch (err) {
    console.error("Failed to load containers:", err);
    containers.value = [];
  } finally {
    isLoading.value = false;
  }
};

// Handle selection change
const handleChange = (value: string) => {
  selectedValue.value = value;
  emit("update:modelValue", value);
  const container = selectedContainer.value;
  emit("change", container ?? null);
};

// Watch for external modelValue changes
watch(
  () => props.modelValue,
  (newValue) => {
    if (newValue !== selectedValue.value) {
      selectedValue.value = newValue || "";
    }
  }
);

// Watch deployment changes
watch(
  () => props.deploymentId,
  async () => {
    selectedValue.value = "";
    await loadContainers();
    emit("update:modelValue", "");
    emit("change", null);
  }
);

onMounted(() => {
  loadContainers();
});
</script>

