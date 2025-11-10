<template>
  <OuiDialog :open="modelValue" title="Create VPS Instance" description="Provision a new virtual private server with full root access." @update:open="updateOpen">
    <OuiStack gap="lg">
        <!-- Error Alert -->
        <ErrorAlert
          v-if="error"
          :error="error"
          title="Failed to create VPS"
        />

        <!-- Form -->
        <OuiStack gap="md">
          <OuiInput
            v-model="form.name"
            label="Name"
            placeholder="my-vps"
            required
            :error="errors.name"
          />

          <OuiTextarea
            v-model="form.description"
            label="Description"
            placeholder="Optional description"
            :rows="2"
          />

          <OuiSelect
            v-model="form.region"
            label="Region"
            :items="regionOptions"
            required
            :error="errors.region"
            :loading="loadingRegions"
          />

          <OuiSelect
            v-model="form.image"
            label="Operating System"
            :items="imageOptions"
            required
            :error="errors.image"
          />

          <OuiSelect
            v-model="form.size"
            label="Instance Size"
            description="Select the resource limits for your VPS. You'll be charged based on actual usage."
            :items="sizeOptions"
            required
            :error="errors.size"
            :loading="loadingSizes"
          >
            <template #item="{ item }">
              <OuiStack gap="xs">
                <OuiText weight="medium">{{ item.label }}</OuiText>
                <OuiText size="xs" color="secondary">
                  {{ item.description }}
                </OuiText>
                <OuiText size="xs" color="secondary">
                  {{ item.cpuCores }} CPU · {{ formatMemory(item.memoryBytes) }} RAM · {{ formatDisk(item.diskBytes) }} Storage
                </OuiText>
              </OuiStack>
            </template>
          </OuiSelect>
        </OuiStack>

        <!-- Summary -->
        <OuiCard variant="outline" v-if="selectedSize">
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold">Resource Limits</OuiText>
              <OuiText size="xs" color="secondary">
                These limits define the maximum resources your VPS can use. You'll be charged based on actual usage (pay-as-you-go).
              </OuiText>
              <OuiGrid cols="2" gap="sm">
                <OuiText size="xs" color="secondary">CPU Cores</OuiText>
                <OuiText size="xs">{{ selectedSize.cpuCores }}</OuiText>
                <OuiText size="xs" color="secondary">Memory</OuiText>
                <OuiText size="xs">{{ formatMemory(selectedSize.memoryBytes) }}</OuiText>
                <OuiText size="xs" color="secondary">Storage</OuiText>
                <OuiText size="xs">{{ formatDisk(selectedSize.diskBytes) }}</OuiText>
              </OuiGrid>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiStack>

    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="updateOpen(false)">Cancel</OuiButton>
        <OuiButton
          color="primary"
          @click="handleCreate"
          :disabled="!isValid || isCreating"
        >
          {{ isCreating ? "Creating..." : "Create VPS" }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from "vue";
  import { VPSService, VPSImage } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import { useDialog } from "~/composables/useDialog";
  import ErrorAlert from "~/components/ErrorAlert.vue";

  interface Props {
    modelValue: boolean;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<{
    "update:modelValue": [value: boolean];
    created: [];
  }>();

  const client = useConnectClient(VPSService);
  const organizationId = useOrganizationId();
  const { showAlert } = useDialog();

  const form = ref({
    name: "",
    description: "",
    region: "",
    image: VPSImage.UBUNTU_24_04,
    size: "",
  });

  const errors = ref<Record<string, string>>({});
  const error = ref<Error | null>(null);
  const isCreating = ref(false);
  const loadingRegions = ref(false);
  const loadingSizes = ref(false);

  const regionOptions = ref<Array<{ label: string; value: string }>>([]);
  const sizeOptions = ref<Array<{
    label: string;
    value: string;
    description: string;
    minimumPaymentCents: bigint | number;
    cpuCores: number;
    memoryBytes: bigint | number;
    diskBytes: bigint | number;
  }>>([]);

  const imageOptions = [
    { label: "Ubuntu 24.04 LTS", value: String(VPSImage.UBUNTU_24_04) },
    { label: "Ubuntu 22.04 LTS", value: String(VPSImage.UBUNTU_22_04) },
    { label: "Debian 13", value: String(VPSImage.DEBIAN_13) },
    { label: "Debian 12", value: String(VPSImage.DEBIAN_12) },
    { label: "Rocky Linux 9", value: String(VPSImage.ROCKY_LINUX_9) },
    { label: "AlmaLinux 9", value: String(VPSImage.ALMA_LINUX_9) },
  ];

  const selectedSize = computed(() => {
    if (!form.value.size) return null;
    return sizeOptions.value.find((s) => s.value === form.value.size);
  });

  const isValid = computed(() => {
    return (
      form.value.name.trim() !== "" &&
      form.value.region !== "" &&
      form.value.size !== ""
    );
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

  const updateOpen = (value: boolean) => {
    emit("update:modelValue", value);
    if (!value) {
      // Reset form when closing
      form.value = {
        name: "",
        description: "",
        region: "",
        image: VPSImage.UBUNTU_24_04,
        size: "",
      };
      errors.value = {};
      error.value = null;
    }
  };

  // Load regions and sizes when dialog opens
  watch(() => props.modelValue, async (isOpen) => {
    if (isOpen) {
      await loadRegions();
      await loadSizes();
    }
  });

  const loadRegions = async () => {
    loadingRegions.value = true;
    try {
      const response = await client.listVPSRegions({});
      const availableRegions = (response.regions || []).filter((r) => r.available);
      
      if (availableRegions.length > 0) {
        regionOptions.value = availableRegions.map((r) => ({ label: r.name, value: r.id }));
        // Auto-select first region if none selected
        if (!form.value.region && availableRegions.length === 1 && availableRegions[0]) {
          form.value.region = availableRegions[0].id;
        }
      } else {
        // No regions available
        regionOptions.value = [];
      }
    } catch (err) {
      console.error("Failed to load regions:", err);
      // Show error but don't default
      regionOptions.value = [];
      error.value = err instanceof Error ? err : new Error("Failed to load VPS regions. Please ensure VPS_REGIONS environment variable is configured.");
    } finally {
      loadingRegions.value = false;
    }
  };

  const loadSizes = async () => {
    loadingSizes.value = true;
    try {
      const response = await client.listVPSSizes({
        region: form.value.region || undefined,
      });
      sizeOptions.value = (response.sizes || [])
        .filter((s) => s.available)
        .map((s) => ({
          label: s.name,
          value: s.id,
          description: `${s.cpuCores} CPU • ${formatMemory(s.memoryBytes)} RAM • ${formatDisk(s.diskBytes)} Storage`,
          minimumPaymentCents: s.minimumPaymentCents || 0,
          cpuCores: s.cpuCores,
          memoryBytes: s.memoryBytes,
          diskBytes: s.diskBytes,
        }));
    } catch (err) {
      console.error("Failed to load sizes:", err);
    } finally {
      loadingSizes.value = false;
    }
  };

  // Reload sizes when region changes
  watch(() => form.value.region, () => {
    if (form.value.region) {
      loadSizes();
      form.value.size = ""; // Reset size selection
    }
  });

  const handleCreate = async () => {
    errors.value = {};
    error.value = null;

    // Validate
    if (!form.value.name.trim()) {
      errors.value.name = "Name is required";
      return;
    }
    if (!form.value.region) {
      errors.value.region = "Region is required";
      return;
    }
    if (!form.value.size) {
      errors.value.size = "Instance size is required";
      return;
    }

    isCreating.value = true;
    try {
      // Convert image string back to number (enum value)
      const imageValue = typeof form.value.image === "string" 
        ? Number(form.value.image) 
        : form.value.image;
      
      await client.createVPS({
        organizationId: organizationId.value || "",
        name: form.value.name.trim(),
        description: form.value.description.trim() || undefined,
        region: form.value.region,
        image: imageValue as VPSImage,
        size: form.value.size,
      });

      emit("created");
      updateOpen(false);
    } catch (err) {
      error.value = err instanceof Error ? err : new Error("Failed to create VPS");
      await showAlert({
        title: "Failed to create VPS",
        message: error.value.message,
      });
    } finally {
      isCreating.value = false;
    }
  };
</script>

