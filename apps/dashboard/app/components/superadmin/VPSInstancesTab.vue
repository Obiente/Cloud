<template>
  <OuiStack gap="xl">
    <OuiStack gap="xs">
      <OuiText tag="h2" size="2xl" weight="extrabold">All VPS Instances</OuiText>
      <OuiText color="muted">View and manage VPS instances across all organizations.</OuiText>
    </OuiStack>

    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="tableColumns"
          :rows="tableRows"
          :empty-text="isLoading ? 'Loading instances…' : 'No VPS instances found.'"
        >
          <template #cell-name="{ row }">
            <div>
              <div class="font-medium text-text-primary">{{ row.vps?.name || 'N/A' }}</div>
              <div v-if="row.vps?.description" class="text-xs text-text-muted mt-0.5">{{ row.vps.description }}</div>
            </div>
          </template>
          <template #cell-organization="{ row }">
            <span class="text-sm">{{ row.organizationName || 'N/A' }}</span>
          </template>
          <template #cell-status="{ row }">
            <OuiBadge :variant="getStatusColor(row.vps?.status)">
              {{ getStatusLabel(row.vps?.status) }}
            </OuiBadge>
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
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Pagination -->
    <OuiFlex v-if="pagination && pagination.totalPages > 1" justify="center" gap="md">
      <OuiButton
        variant="ghost"
        size="sm"
        :disabled="!pagination || pagination.page <= 1"
        @click="goToPage(pagination.page - 1)"
      >
        Previous
      </OuiButton>
      <OuiText size="sm" color="muted">
        Page {{ pagination?.page || 1 }} of {{ pagination?.totalPages || 1 }}
      </OuiText>
      <OuiButton
        variant="ghost"
        size="sm"
        :disabled="!pagination || pagination.page >= pagination.totalPages"
        @click="goToPage(pagination.page + 1)"
      >
        Next
      </OuiButton>
    </OuiFlex>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import { useConnectClient } from "~/lib/connect-client";
import { SuperadminService, VPSStatus, VPSImage } from "@obiente/proto";
import { useToast } from "~/composables/useToast";

const { toast } = useToast();
const client = useConnectClient(SuperadminService);

const instances = ref<any[]>([]);
const isLoading = ref(false);
const pagination = ref<any>(null);
const currentPage = ref(1);
const perPage = 20;

const tableColumns = computed(() => [
  { key: "name", label: "Name", defaultWidth: 200, minWidth: 150 },
  { key: "organization", label: "Organization", defaultWidth: 180, minWidth: 150 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "specs", label: "Specifications", defaultWidth: 200, minWidth: 180 },
  { key: "region", label: "Region", defaultWidth: 150, minWidth: 120 },
  { key: "image", label: "Image", defaultWidth: 150, minWidth: 120 },
  { key: "created", label: "Created", defaultWidth: 150, minWidth: 120 },
]);

const tableRows = computed(() => instances.value);

const { formatBytes } = useUtils();

const getStatusLabel = (status: any): string => {
  if (status === undefined || status === null) return "Unknown";
  const statusMap: Record<number, string> = {
    [VPSStatus.CREATING]: "Creating",
    [VPSStatus.RUNNING]: "Running",
    [VPSStatus.STOPPED]: "Stopped",
    [VPSStatus.FAILED]: "Failed",
    [VPSStatus.DELETING]: "Deleting",
  };
  return statusMap[status] || "Unknown";
};

type BadgeVariant = "primary" | "secondary" | "success" | "warning" | "danger" | "outline";

const getStatusColor = (status: any): BadgeVariant => {
  if (status === undefined || status === null) return "secondary";
  const colorMap: Record<number, BadgeVariant> = {
    [VPSStatus.CREATING]: "warning",
    [VPSStatus.RUNNING]: "success",
    [VPSStatus.STOPPED]: "secondary",
    [VPSStatus.FAILED]: "danger",
    [VPSStatus.DELETING]: "warning",
  };
  return colorMap[status] || "secondary";
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

onMounted(() => {
  fetchInstances(1);
});
</script>

