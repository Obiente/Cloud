<template>
  <SuperadminPageLayout
    title="All VPS Instances"
    description="View and manage VPS instances across all organizations."
    :columns="tableColumns"
    :rows="filteredTableRows"
    :filters="filterConfigs"
    :search="search"
    :empty-text="isLoading ? 'Loading instances…' : 'No VPS instances match your filters.'"
    :loading="isLoading"
    search-placeholder="Search by name, ID, organization, region, image…"
    :pagination="pagination ? {
      page: pagination.page,
      totalPages: pagination.totalPages,
      total: pagination.total,
      perPage: perPage,
    } : undefined"
    @update:search="search = $event"
    @filter-change="handleFilterChange"
    @refresh="() => fetchInstances(currentPage)"
    @row-click="openVPS"
    @page-change="goToPage"
  >
          <template #cell-name="{ row }">
            <SuperadminResourceCell
              :name="row.vps?.name"
              :description="row.vps?.description"
              :id="row.vps?.id"
            />
          </template>
          <template #cell-organization="{ row }">
            <SuperadminOrganizationCell
              :organization-name="row.organizationName"
              :organization-id="row.vps?.organizationId"
            />
          </template>
          <template #cell-status="{ row }">
            <SuperadminStatusBadge
              :status="row.vps?.status"
              :status-map="vpsStatusMap"
            />
          </template>
          <template #cell-specs="{ row }">
            <OuiStack gap="xs">
              <div class="text-sm">
                <span class="text-text-secondary">CPU:</span>
                <span class="font-mono ml-1">{{ row.vps?.cpuCores || 0 }} cores</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Memory:</span>
                <span class="font-mono ml-1">{{ formatBytes(Number(row.vps?.memoryBytes || 0)) }}</span>
              </div>
              <div class="text-sm">
                <span class="text-text-secondary">Disk:</span>
                <span class="font-mono ml-1">{{ formatBytes(Number(row.vps?.diskBytes || 0)) }}</span>
              </div>
            </OuiStack>
          </template>
          <template #cell-region="{ row }">
            <span class="text-sm">{{ row.vps?.region || 'N/A' }}</span>
          </template>
          <template #cell-image="{ row }">
            <span class="text-sm">{{ getImageLabel(row.vps?.image) }}</span>
          </template>
          <template #cell-created="{ row }">
            <OuiDate v-if="row.vps?.createdAt" :value="row.vps.createdAt" format="short" />
            <span v-else class="text-sm text-text-muted">—</span>
          </template>
          <template #cell-actions="{ row }">
            <SuperadminActionsCell :actions="getVPSActions(row)" />
          </template>
  </SuperadminPageLayout>

    <!-- Resize Dialog -->
    <OuiDialog v-model:open="resizeDialogOpen" title="Resize VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Change the VPS instance size. CPU, memory, and disk will be updated.
        </OuiText>
        <OuiSelect
          v-model="resizeForm.newSize"
          label="New Size"
          :items="sizeOptions"
          :loading="loadingSizes"
          required
        >
          <template #item="{ item }">
            <OuiStack gap="xs">
              <OuiText weight="medium">{{ item.label }}</OuiText>
              <OuiText size="xs" color="secondary">
                {{ item.cpuCores }} CPU · {{ formatMemory(item.memoryBytes) }} RAM · {{ formatDisk(item.diskBytes) }} Storage
              </OuiText>
            </OuiStack>
          </template>
        </OuiSelect>
        <OuiCheckbox v-model="resizeForm.growDisk" label="Grow disk to new size" />
        <OuiCheckbox v-model="resizeForm.applyCloudInit" label="Apply cloud-init for disk growth" />
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="resizeDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="primary" 
            @click="handleResize"
            :disabled="!resizeForm.newSize || isResizing"
          >
            {{ isResizing ? 'Resizing...' : 'Resize' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Suspend Dialog -->
    <OuiDialog v-model:open="suspendDialogOpen" title="Suspend VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Suspend this VPS instance. The VPS will be marked as suspended and normal operations will be prevented.
        </OuiText>
        <OuiInput
          v-model="suspendForm.reason"
          label="Reason (Optional)"
          placeholder="Reason for suspension"
        />
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="suspendDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="warning" 
            @click="handleSuspend"
            :disabled="isSuspending"
          >
            {{ isSuspending ? 'Suspending...' : 'Suspend' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- CloudInit Dialog -->
    <OuiDialog v-model:open="cloudInitDialogOpen" title="Update CloudInit Configuration" size="lg">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Update the cloud-init configuration for this VPS. Changes will take effect on the next reboot.
        </OuiText>
        <OuiTextarea
          v-model="cloudInitForm.userData"
          label="CloudInit User Data (YAML)"
          placeholder="#cloud-config&#10;users:&#10;  - name: user1&#10;    ssh_authorized_keys:&#10;      - ssh-rsa ..."
          :rows="15"
        />
        <OuiCheckbox v-model="cloudInitForm.growDiskIfNeeded" label="Grow disk if needed" />
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="cloudInitDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="primary" 
            @click="handleUpdateCloudInit"
            :disabled="isUpdatingCloudInit"
          >
            {{ isUpdatingCloudInit ? 'Updating...' : 'Update' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Force Stop Dialog -->
    <OuiDialog v-model:open="forceStopDialogOpen" title="Force Stop VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Force stop this VPS instance immediately. This will perform a hard shutdown.
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="forceStopDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="danger" 
            @click="handleForceStop"
            :disabled="isForceStopping"
          >
            {{ isForceStopping ? 'Stopping...' : 'Force Stop' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Force Delete Dialog -->
    <OuiDialog v-model:open="forceDeleteDialogOpen" title="Force Delete VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Permanently delete this VPS instance. This action cannot be undone.
        </OuiText>
        <OuiCheckbox v-model="forceDeleteForm.hardDelete" label="Hard delete (permanently remove from Proxmox)" />
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="forceDeleteDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="danger" 
            @click="handleForceDelete"
            :disabled="isForceDeleting"
          >
            {{ isForceDeleting ? 'Deleting...' : 'Delete' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, watch } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { SuperadminService, VPSStatus, VPSImage } from "@obiente/proto";
import { useToast } from "~/composables/useToast";
import { useRouter } from "vue-router";
import { useOrganizationsStore } from "~/stores/organizations";
import SuperadminPageLayout from "./SuperadminPageLayout.vue";
import SuperadminResourceCell from "./SuperadminResourceCell.vue";
import SuperadminOrganizationCell from "./SuperadminOrganizationCell.vue";
import SuperadminStatusBadge from "./SuperadminStatusBadge.vue";
import SuperadminActionsCell, { type Action } from "./SuperadminActionsCell.vue";
import type { FilterConfig } from "./SuperadminFilterBar.vue";
import type { BadgeVariant } from "~/components/oui/Badge.vue";

const { toast } = useToast();
const client = useConnectClient(SuperadminService);
const router = useRouter();
const organizationsStore = useOrganizationsStore();

const instances = ref<any[]>([]);
const isLoading = ref(false);
const pagination = ref<any>(null);
const currentPage = ref(1);
const perPage = 20;
const search = ref("");
const statusFilter = ref<string>("all");
const regionFilter = ref<string>("all");
const imageFilter = ref<string>("all");

const tableColumns = computed(() => [
  { key: "name", label: "Name", defaultWidth: 200, minWidth: 150 },
  { key: "organization", label: "Organization", defaultWidth: 180, minWidth: 150 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "specs", label: "Specifications", defaultWidth: 200, minWidth: 180 },
  { key: "region", label: "Region", defaultWidth: 150, minWidth: 120 },
  { key: "image", label: "Image", defaultWidth: 150, minWidth: 120 },
  { key: "created", label: "Created", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 100, minWidth: 80 },
]);

const vpsStatusMap: Record<number, { label: string; variant: BadgeVariant }> = {
  [VPSStatus.CREATING]: { label: "Creating", variant: "warning" },
  [VPSStatus.STARTING]: { label: "Starting", variant: "warning" },
  [VPSStatus.RUNNING]: { label: "Running", variant: "success" },
  [VPSStatus.STOPPING]: { label: "Stopping", variant: "warning" },
  [VPSStatus.STOPPED]: { label: "Stopped", variant: "secondary" },
  [VPSStatus.REBOOTING]: { label: "Rebooting", variant: "warning" },
  [VPSStatus.FAILED]: { label: "Failed", variant: "danger" },
  [VPSStatus.DELETING]: { label: "Deleting", variant: "warning" },
  [VPSStatus.DELETED]: { label: "Deleted", variant: "secondary" },
  [VPSStatus.SUSPENDED]: { label: "Suspended", variant: "warning" },
};

const statusOptions = computed(() => {
  const statuses = new Set<number>();
  instances.value.forEach((inst) => {
    if (inst.vps?.status !== undefined && inst.vps?.status !== null) {
      statuses.add(inst.vps.status);
    }
  });
  const sortedStatuses = Array.from(statuses).sort();
  return [
    { label: "All statuses", value: "all" },
    ...sortedStatuses.map((status) => ({
      label: vpsStatusMap[status]?.label || `Status ${status}`,
      value: String(status),
    })),
  ];
});

const regionOptions = computed(() => {
  const regions = new Set<string>();
  instances.value.forEach((inst) => {
    if (inst.vps?.region) {
      regions.add(inst.vps.region);
    }
  });
  const sortedRegions = Array.from(regions).sort();
  return [
    { label: "All regions", value: "all" },
    ...sortedRegions.map((region) => ({ label: region, value: region })),
  ];
});

const imageOptions = computed(() => {
  const images = new Set<number>();
  instances.value.forEach((inst) => {
    if (inst.vps?.image !== undefined && inst.vps?.image !== null) {
      images.add(inst.vps.image);
    }
  });
  const sortedImages = Array.from(images).sort();
  return [
    { label: "All images", value: "all" },
    ...sortedImages.map((image) => ({
      label: getImageLabel(image),
      value: String(image),
    })),
  ];
});

const filterConfigs = computed(() => [
  {
    key: "status",
    placeholder: "Status",
    items: statusOptions.value,
  },
  {
    key: "region",
    placeholder: "Region",
    items: regionOptions.value,
  },
  {
    key: "image",
    placeholder: "Image",
    items: imageOptions.value,
  },
] as FilterConfig[]);

const filteredTableRows = computed(() => {
  const term = search.value.trim().toLowerCase();
  const status = statusFilter.value;
  const region = regionFilter.value;
  const image = imageFilter.value;

  return instances.value.filter((inst) => {
    // Status filter
    if (status !== "all" && String(inst.vps?.status) !== status) {
      return false;
    }

    // Region filter
    if (region !== "all" && inst.vps?.region !== region) {
      return false;
    }

    // Image filter
    if (image !== "all" && String(inst.vps?.image) !== image) {
      return false;
    }

    // Search filter
    if (!term) return true;

    const searchable = [
      inst.vps?.name,
      inst.vps?.id,
      inst.organizationName,
      inst.vps?.organizationId,
      inst.vps?.region,
      getImageLabel(inst.vps?.image),
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();

    return searchable.includes(term);
  });
});

function handleFilterChange(key: string, value: string) {
  if (key === "status") {
    statusFilter.value = value;
  } else if (key === "region") {
    regionFilter.value = value;
  } else if (key === "image") {
    imageFilter.value = value;
  }
}

const { formatBytes } = useUtils();

const formatMemory = (bytes: bigint | number): string => {
  const numBytes = typeof bytes === 'bigint' ? Number(bytes) : bytes;
  return formatBytes(numBytes);
};

const formatDisk = (bytes: bigint | number): string => {
  const numBytes = typeof bytes === 'bigint' ? Number(bytes) : bytes;
  return formatBytes(numBytes);
};

const getVPSActions = (row: any): Action[] => {
  const actions: Action[] = [
    {
      key: "resize",
      label: "Resize",
      onClick: () => openResizeDialog(row),
    },
  ];

  if (row.vps?.status !== VPSStatus.SUSPENDED) {
    actions.push({
      key: "suspend",
      label: "Suspend",
      onClick: () => openSuspendDialog(row),
    });
  } else {
    actions.push({
      key: "unsuspend",
      label: "Unsuspend",
      onClick: () => handleUnsuspend(row),
    });
  }

  actions.push({
    key: "cloudinit",
    label: "Update CloudInit",
    onClick: () => openCloudInitDialog(row),
  });

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

const getImageLabel = (image: any): string => {
  if (image === undefined || image === null) return "N/A";
  const imageMap: Record<number, string> = {
    [VPSImage.UBUNTU_22_04]: "Ubuntu 22.04",
    [VPSImage.UBUNTU_24_04]: "Ubuntu 24.04",
    [VPSImage.DEBIAN_12]: "Debian 12",
    [VPSImage.DEBIAN_13]: "Debian 13",
    [VPSImage.ROCKY_LINUX_9]: "Rocky Linux 9",
    [VPSImage.ALMA_LINUX_9]: "AlmaLinux 9",
  };
  return imageMap[image] || "Unknown";
};

const fetchInstances = async (page: number = 1) => {
  isLoading.value = true;
  try {
    const response = await client.listAllVPS({
      page: page,
      perPage: perPage,
    });
    instances.value = response.vpsInstances || [];
    pagination.value = response.pagination;
    currentPage.value = page;
  } catch (error: any) {
    toast.error(`Failed to load VPS instances: ${error?.message || "Unknown error"}`);
  } finally {
    isLoading.value = false;
  }
};

const goToPage = (page: number) => {
  if (page >= 1 && (!pagination.value || page <= pagination.value.totalPages)) {
    fetchInstances(page);
  }
};

const openVPS = (row: any) => {
  const vpsId = row.vps?.id;
  const organizationId = row.vps?.organizationId;
  
  if (vpsId && organizationId) {
    // Switch to the VPS's organization and navigate to the VPS detail page
    organizationsStore.switchOrganization(organizationId);
    router.push(`/vps/${vpsId}`);
  }
};

// Resize Dialog
const resizeDialogOpen = ref(false);
const isResizing = ref(false);
const loadingSizes = ref(false);
const sizeOptions = ref<any[]>([]);
const resizeForm = ref({
  vpsId: "",
  newSize: "",
  growDisk: true,
  applyCloudInit: true,
});

const openResizeDialog = async (row: any) => {
  resizeForm.value.vpsId = row.vps?.id || "";
  resizeForm.value.newSize = row.vps?.size || "";
  resizeForm.value.growDisk = true;
  resizeForm.value.applyCloudInit = true;
  resizeDialogOpen.value = true;
  
  // Load sizes
  loadingSizes.value = true;
  try {
    const response = await client.listVPSSizes({
      region: row.vps?.region || undefined,
    });
    sizeOptions.value = (response.sizes || [])
      .filter((s) => s.available)
      .map((s) => ({
        label: s.name,
        value: s.id,
        cpuCores: s.cpuCores,
        memoryBytes: s.memoryBytes,
        diskBytes: s.diskBytes,
      }));
  } catch (error: any) {
    toast.error(`Failed to load sizes: ${error?.message || "Unknown error"}`);
  } finally {
    loadingSizes.value = false;
  }
};

const handleResize = async () => {
  if (!resizeForm.value.newSize) return;
  
  isResizing.value = true;
  try {
    const response = await client.superadminResizeVPS({
      vpsId: resizeForm.value.vpsId,
      newSize: resizeForm.value.newSize,
      growDisk: resizeForm.value.growDisk,
      applyCloudinit: resizeForm.value.applyCloudInit,
    });
    toast.success(response.message || "VPS resized successfully");
    resizeDialogOpen.value = false;
    await fetchInstances(currentPage.value);
  } catch (error: any) {
    toast.error(`Failed to resize VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isResizing.value = false;
  }
};

// Suspend Dialog
const suspendDialogOpen = ref(false);
const isSuspending = ref(false);
const suspendForm = ref({
  vpsId: "",
  reason: "",
});

const openSuspendDialog = (row: any) => {
  suspendForm.value.vpsId = row.vps?.id || "";
  suspendForm.value.reason = "";
  suspendDialogOpen.value = true;
};

const handleSuspend = async () => {
  isSuspending.value = true;
  try {
    const response = await client.superadminSuspendVPS({
      vpsId: suspendForm.value.vpsId,
      reason: suspendForm.value.reason || undefined,
    });
    toast.success(response.message || "VPS suspended successfully");
    suspendDialogOpen.value = false;
    await fetchInstances(currentPage.value);
  } catch (error: any) {
    toast.error(`Failed to suspend VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isSuspending.value = false;
  }
};

const handleUnsuspend = async (row: any) => {
  const vpsId = row.vps?.id;
  if (!vpsId) return;
  
  try {
    const response = await client.superadminUnsuspendVPS({
      vpsId: vpsId,
    });
    toast.success(response.message || "VPS unsuspended successfully");
    await fetchInstances(currentPage.value);
  } catch (error: any) {
    toast.error(`Failed to unsuspend VPS: ${error?.message || "Unknown error"}`);
  }
};

// CloudInit Dialog
const cloudInitDialogOpen = ref(false);
const isUpdatingCloudInit = ref(false);
const cloudInitForm = ref({
  vpsId: "",
  userData: "",
  growDiskIfNeeded: true,
});

const openCloudInitDialog = async (row: any) => {
  cloudInitForm.value.vpsId = row.vps?.id || "";
  cloudInitForm.value.userData = "";
  cloudInitForm.value.growDiskIfNeeded = true;
  cloudInitDialogOpen.value = true;
  
  // TODO: Load existing cloud-init config if available
};

const handleUpdateCloudInit = async () => {
  // TODO: Parse YAML and convert to CloudInitConfig proto
  // For now, show an error that this needs proper YAML parsing
  toast.error("CloudInit update requires proper YAML parsing. This feature is not yet fully implemented.");
  cloudInitDialogOpen.value = false;
};

// Force Stop Dialog
const forceStopDialogOpen = ref(false);
const isForceStopping = ref(false);
const forceStopVpsId = ref("");

const openForceStopDialog = (row: any) => {
  forceStopVpsId.value = row.vps?.id || "";
  forceStopDialogOpen.value = true;
};

const handleForceStop = async () => {
  isForceStopping.value = true;
  try {
    const response = await client.superadminForceStopVPS({
      vpsId: forceStopVpsId.value,
    });
    toast.success(response.message || "VPS force stopped successfully");
    forceStopDialogOpen.value = false;
    await fetchInstances(currentPage.value);
  } catch (error: any) {
    toast.error(`Failed to force stop VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isForceStopping.value = false;
  }
};

// Force Delete Dialog
const forceDeleteDialogOpen = ref(false);
const isForceDeleting = ref(false);
const forceDeleteForm = ref({
  vpsId: "",
  hardDelete: false,
});

const openForceDeleteDialog = (row: any) => {
  forceDeleteForm.value.vpsId = row.vps?.id || "";
  forceDeleteForm.value.hardDelete = false;
  forceDeleteDialogOpen.value = true;
};

const handleForceDelete = async () => {
  isForceDeleting.value = true;
  try {
    const response = await client.superadminForceDeleteVPS({
      vpsId: forceDeleteForm.value.vpsId,
      hardDelete: forceDeleteForm.value.hardDelete,
    });
    toast.success(response.message || "VPS deleted successfully");
    forceDeleteDialogOpen.value = false;
    await fetchInstances(currentPage.value);
  } catch (error: any) {
    toast.error(`Failed to delete VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isForceDeleting.value = false;
  }
};

onMounted(() => {
  fetchInstances(1);
});
</script>
