<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h2" size="2xl" weight="extrabold">VPS Size Catalog</OuiText>
        <OuiText color="muted">Manage VPS instance sizes and minimum payment requirements.</OuiText>
      </OuiStack>
      <OuiButton @click="openCreateDialog">
        <PlusIcon class="h-4 w-4" />
        Create Size
      </OuiButton>
    </OuiFlex>

    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="tableColumns"
          :rows="tableRows"
          :empty-text="isLoading ? 'Loading sizes…' : 'No sizes found. Create your first size to get started.'"
        >
          <template #cell-name="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ value }}</div>
              <div v-if="row.description" class="text-xs text-text-muted mt-0.5">{{ row.description }}</div>
            </div>
          </template>
          <template #cell-specs="{ row }">
            <OuiStack gap="xs">
              <div class="text-sm">
                <span class="text-text-secondary">CPU:</span>
                <span class="font-mono ml-1">{{ row.cpuCores || 0 }} cores</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Memory:</span>
                <span class="font-mono ml-1">{{ formatBytes(Number(row.memoryBytes || 0)) }}</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Disk:</span>
                <span class="font-mono ml-1">{{ formatBytes(Number(row.diskBytes || 0)) }}</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Bandwidth:</span>
                <span class="font-mono ml-1">
                  {{ row.bandwidthBytesMonth === 0n || row.bandwidthBytesMonth === 0 ? 'Unlimited' : formatBytes(Number(row.bandwidthBytesMonth || 0)) + '/mo' }}
                </span>
              </div>
            </OuiStack>
          </template>
          <template #cell-minimumPayment="{ value }">
            <span class="font-mono">
              <OuiCurrency :value="typeof value === 'bigint' ? Number(value) / 100 : (Number(value) || 0) / 100" />
            </span>
            <span v-if="!value || value === 0n || value === 0" class="text-text-muted text-xs ml-1">(no requirement)</span>
          </template>
          <template #cell-status="{ row }">
            <OuiBadge :variant="row.available ? 'success' : 'secondary'">
              {{ row.available ? 'Available' : 'Unavailable' }}
            </OuiBadge>
          </template>
          <template #cell-region="{ value }">
            <span class="text-sm">{{ value || 'All Regions' }}</span>
          </template>
          <template #cell-actions="{ row }">
            <OuiFlex gap="sm" justify="end">
              <OuiButton size="xs" variant="ghost" @click="openEditDialog(row)">
                Edit
              </OuiButton>
              <OuiButton size="xs" variant="ghost" color="danger" @click="openDeleteDialog(row)">
                Delete
              </OuiButton>
            </OuiFlex>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Create/Edit Size Dialog -->
    <OuiDialog v-model:open="sizeDialogOpen" :title="editingSize ? 'Edit VPS Size' : 'Create VPS Size'">
      <OuiStack gap="lg">
        <OuiStack gap="md">
          <OuiField label="Size ID" required>
            <OuiInput 
              v-model="sizeForm.id" 
              placeholder="e.g., small, medium, custom-1" 
              :disabled="!!editingSize"
            />
            <OuiText size="xs" color="muted" class="mt-1">
              Unique identifier for this size. Cannot be changed after creation.
            </OuiText>
          </OuiField>

          <OuiField label="Display Name" required>
            <OuiInput v-model="sizeForm.name" placeholder="e.g., Small VPS, Medium VPS" />
          </OuiField>
          
          <OuiField label="Description">
            <OuiTextarea v-model="sizeForm.description" placeholder="Optional description of the size" :rows="3" />
          </OuiField>

          <OuiField label="Region">
            <OuiInput v-model="sizeForm.region" placeholder="Leave empty for all regions" />
            <OuiText size="xs" color="muted" class="mt-1">
              Leave empty to make this size available in all regions, or specify a region ID (e.g., us-east-1).
            </OuiText>
          </OuiField>

          <OuiField label="Available">
            <OuiSwitch v-model="sizeForm.available" />
            <OuiText size="xs" color="muted" class="mt-1">
              Whether this size is available for new VPS instances.
            </OuiText>
          </OuiField>
        </OuiStack>

        <OuiDivider />

        <OuiStack gap="md">
          <OuiText size="sm" weight="medium">Resource Specifications</OuiText>
          
          <OuiField label="CPU Cores" required>
            <OuiInput v-model="sizeForm.cpuCores" type="number" min="1" placeholder="1" />
          </OuiField>

          <OuiField label="Memory (GB)" required>
            <OuiInput v-model="sizeForm.memoryGB" type="number" min="0.1" step="0.1" placeholder="1" />
            <OuiText size="xs" color="muted" class="mt-1">
              Memory in gigabytes (will be converted to bytes)
            </OuiText>
          </OuiField>

          <OuiField label="Disk Space (GB)" required>
            <OuiInput v-model="sizeForm.diskGB" type="number" min="1" placeholder="10" />
            <OuiText size="xs" color="muted" class="mt-1">
              Disk space in gigabytes (will be converted to bytes)
            </OuiText>
          </OuiField>

          <OuiField label="Bandwidth per Month (GB)">
            <OuiInput v-model="sizeForm.bandwidthGB" type="number" min="0" placeholder="0" />
            <OuiText size="xs" color="muted" class="mt-1">
              Monthly bandwidth limit in gigabytes. Set to 0 for unlimited (will be converted to bytes).
            </OuiText>
          </OuiField>
        </OuiStack>

        <OuiDivider />

        <OuiStack gap="md">
          <OuiText size="sm" weight="medium">Minimum Payment Requirement</OuiText>
          
          <OuiField label="Minimum Payment (USD)">
            <OuiInput
              v-model="sizeForm.minimumPaymentUSD"
              type="number"
              step="0.01"
              min="0"
              placeholder="0.00"
            />
            <OuiText size="xs" color="muted" class="mt-1">
              Minimum total payment (in USD) required for organizations to create VPS instances of this size. Set to 0 for no requirement. This helps ensure payment security before allowing larger VPS instances.
            </OuiText>
          </OuiField>
        </OuiStack>
      </OuiStack>

      <template #footer>
        <OuiFlex gap="sm" justify="end">
          <OuiButton variant="ghost" @click="closeSizeDialog">Cancel</OuiButton>
          <OuiButton @click="saveSize" :disabled="isSaving">
            {{ isSaving ? 'Saving...' : (editingSize ? 'Update' : 'Create') }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Delete Confirmation Dialog -->
    <OuiDialog v-model:open="deleteDialogOpen" title="Delete VPS Size">
      <OuiStack gap="md">
        <OuiText>
          Are you sure you want to delete the VPS size <strong>{{ sizeToDelete?.name }}</strong>?
        </OuiText>
        <OuiText size="sm" color="danger">
          This action cannot be undone. If any VPS instances are using this size, deletion may cause issues.
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex gap="sm" justify="end">
          <OuiButton variant="ghost" @click="deleteDialogOpen = false">Cancel</OuiButton>
          <OuiButton color="danger" @click="confirmDelete" :disabled="isDeleting">
            {{ isDeleting ? 'Deleting...' : 'Delete' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { PlusIcon } from "@heroicons/vue/24/outline";
import { computed, ref, onMounted } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { SuperadminService } from "@obiente/proto";
import { useToast } from "~/composables/useToast";

const { toast } = useToast();
const client = useConnectClient(SuperadminService);

const sizes = ref<any[]>([]);
const isLoading = ref(false);
const isSaving = ref(false);
const isDeleting = ref(false);
const sizeDialogOpen = ref(false);
const deleteDialogOpen = ref(false);
const editingSize = ref<any>(null);
const sizeToDelete = ref<any>(null);

const sizeForm = ref({
  id: "",
  name: "",
  description: "",
  cpuCores: "",
  memoryGB: "",
  diskGB: "",
  bandwidthGB: "",
  minimumPaymentUSD: "",
  available: true,
  region: "",
});

const tableColumns = computed(() => [
  { key: "name", label: "Name", defaultWidth: 200, minWidth: 150 },
  { key: "specs", label: "Specifications", defaultWidth: 250, minWidth: 200 },
  { key: "minimumPayment", label: "Min Payment", defaultWidth: 150, minWidth: 120 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "region", label: "Region", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 150, minWidth: 120, resizable: false },
]);

const tableRows = computed(() => 
  sizes.value.map(size => ({
    ...size,
    minimumPayment: size.minimumPaymentCents || 0,
  }))
);

const { formatBytes } = useUtils();

const fetchSizes = async () => {
  isLoading.value = true;
  try {
    const response = await client.listVPSSizes({ includeUnavailable: true });
    sizes.value = response.sizes || [];
  } catch (error: any) {
    toast.error(`Failed to load sizes: ${error?.message || "Unknown error"}`);
  } finally {
    isLoading.value = false;
  }
};

const openCreateDialog = () => {
  editingSize.value = null;
  sizeForm.value = {
    id: "",
    name: "",
    description: "",
    cpuCores: "",
    memoryGB: "",
    diskGB: "",
    bandwidthGB: "",
    minimumPaymentUSD: "",
    available: true,
    region: "",
  };
  sizeDialogOpen.value = true;
};

const openEditDialog = (size: any) => {
  editingSize.value = size;
  sizeForm.value = {
    id: size.id || "",
    name: size.name || "",
    description: size.description || "",
    cpuCores: size.cpuCores != null ? String(size.cpuCores) : "",
    memoryGB: size.memoryBytes != null ? String(Number(size.memoryBytes) / (1024 * 1024 * 1024)) : "",
    diskGB: size.diskBytes != null ? String(Number(size.diskBytes) / (1024 * 1024 * 1024)) : "",
    bandwidthGB: size.bandwidthBytesMonth != null && size.bandwidthBytesMonth !== 0n && size.bandwidthBytesMonth !== 0
      ? String(Number(size.bandwidthBytesMonth) / (1024 * 1024 * 1024))
      : "",
    minimumPaymentUSD: size.minimumPaymentCents != null ? String(Number(size.minimumPaymentCents) / 100) : "",
    available: size.available ?? true,
    region: size.region || "",
  };
  sizeDialogOpen.value = true;
};

const closeSizeDialog = () => {
  sizeDialogOpen.value = false;
  editingSize.value = null;
};

const saveSize = async () => {
  if (!sizeForm.value.id) {
    toast.error("Size ID is required");
    return;
  }
  if (!sizeForm.value.name) {
    toast.error("Display name is required");
    return;
  }
  if (!sizeForm.value.cpuCores || Number(sizeForm.value.cpuCores) <= 0) {
    toast.error("CPU cores must be greater than 0");
    return;
  }
  if (!sizeForm.value.memoryGB || Number(sizeForm.value.memoryGB) <= 0) {
    toast.error("Memory must be greater than 0");
    return;
  }
  if (!sizeForm.value.diskGB || Number(sizeForm.value.diskGB) <= 0) {
    toast.error("Disk space must be greater than 0");
    return;
  }
  if (sizeForm.value.minimumPaymentUSD && Number(sizeForm.value.minimumPaymentUSD) < 0) {
    toast.error("Minimum payment must be 0 or greater");
    return;
  }

  isSaving.value = true;
  try {
    // Convert inputs to proper types
    const cpuCores = Number(sizeForm.value.cpuCores);
    const memoryBytes = BigInt(Math.round(Number(sizeForm.value.memoryGB) * 1024 * 1024 * 1024));
    const diskBytes = BigInt(Math.round(Number(sizeForm.value.diskGB) * 1024 * 1024 * 1024));
    const bandwidthBytesMonth = sizeForm.value.bandwidthGB && Number(sizeForm.value.bandwidthGB) > 0
      ? BigInt(Math.round(Number(sizeForm.value.bandwidthGB) * 1024 * 1024 * 1024))
      : BigInt(0);
    const minimumPaymentCents = sizeForm.value.minimumPaymentUSD
      ? BigInt(Math.round(Number(sizeForm.value.minimumPaymentUSD) * 100))
      : BigInt(0);

    if (editingSize.value) {
      const updateRequest: any = {
        id: editingSize.value.id,
      };
      if (sizeForm.value.name) updateRequest.name = sizeForm.value.name;
      if (sizeForm.value.description !== undefined) updateRequest.description = sizeForm.value.description;
      if (cpuCores > 0) updateRequest.cpuCores = cpuCores;
      if (memoryBytes > 0n) updateRequest.memoryBytes = memoryBytes;
      if (diskBytes > 0n) updateRequest.diskBytes = diskBytes;
      if (bandwidthBytesMonth !== undefined) updateRequest.bandwidthBytesMonth = bandwidthBytesMonth;
      if (minimumPaymentCents !== undefined) updateRequest.minimumPaymentCents = minimumPaymentCents;
      if (sizeForm.value.available !== undefined) updateRequest.available = sizeForm.value.available;
      if (sizeForm.value.region !== undefined) updateRequest.region = sizeForm.value.region;

      await client.updateVPSSize(updateRequest);
      toast.success("VPS size updated successfully");
    } else {
      await client.createVPSSize({
        id: sizeForm.value.id,
        name: sizeForm.value.name,
        description: sizeForm.value.description || "",
        cpuCores: cpuCores,
        memoryBytes: memoryBytes,
        diskBytes: diskBytes,
        bandwidthBytesMonth: bandwidthBytesMonth,
        minimumPaymentCents: minimumPaymentCents,
        available: sizeForm.value.available,
        region: sizeForm.value.region || "",
      });
      toast.success("VPS size created successfully");
    }
    closeSizeDialog();
    await fetchSizes();
  } catch (error: any) {
    toast.error(`Failed to save size: ${error?.message || "Unknown error"}`);
  } finally {
    isSaving.value = false;
  }
};

const openDeleteDialog = (size: any) => {
  sizeToDelete.value = size;
  deleteDialogOpen.value = true;
};

const confirmDelete = async () => {
  if (!sizeToDelete.value) return;

  isDeleting.value = true;
  try {
    await client.deleteVPSSize({ id: sizeToDelete.value.id });
    toast.success("VPS size deleted successfully");
    deleteDialogOpen.value = false;
    sizeToDelete.value = null;
    await fetchSizes();
  } catch (error: any) {
    toast.error(`Failed to delete size: ${error?.message || "Unknown error"}`);
  } finally {
    isDeleting.value = false;
  }
};

onMounted(() => {
  fetchSizes();
});
</script>

