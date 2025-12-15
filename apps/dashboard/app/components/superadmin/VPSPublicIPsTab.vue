<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h2" size="2xl" weight="extrabold">Public IP Addresses</OuiText>
        <OuiText color="muted">Manage public IP addresses and assign them to VPS instances.</OuiText>
      </OuiStack>
      <OuiButton @click="openCreateDialog">
        <PlusIcon class="h-4 w-4" />
        Create IP
      </OuiButton>
    </OuiFlex>

    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="tableColumns"
          :rows="tableRows"
          :empty-text="isLoading ? 'Loading IPs…' : 'No public IPs found. Create your first IP to get started.'"
        >
          <template #cell-ipAddress="{ value }">
            <OuiCode :code="value" inline />
          </template>
          <template #cell-vps="{ row }">
            <OuiStack v-if="row.vpsId" gap="xs">
              <OuiText weight="medium">{{ row.vpsName || row.vpsId }}</OuiText>
              <OuiText v-if="row.organizationName" size="xs" color="muted">
                {{ row.organizationName }}
              </OuiText>
            </OuiStack>
            <OuiText v-else size="sm" color="muted">Unassigned</OuiText>
          </template>
          <template #cell-cost="{ value }">
            <OuiFlex gap="xs" align="center">
              <OuiCurrency :value="value" />
              <OuiText size="xs" color="muted">/month</OuiText>
            </OuiFlex>
          </template>
          <template #cell-assignedAt="{ value }">
            <OuiDate v-if="value" :value="value" />
            <OuiText v-else size="sm" color="muted">—</OuiText>
          </template>
          <template #cell-createdAt="{ value }">
            <OuiDate v-if="value" :value="value" />
            <OuiText v-else size="sm" color="muted">—</OuiText>
          </template>
          <template #cell-actions="{ row }">
            <OuiFlex gap="sm" justify="end">
              <OuiButton 
                v-if="!row.vpsId" 
                size="xs" 
                variant="ghost" 
                @click="openAssignDialog(row)"
              >
                Assign
              </OuiButton>
              <OuiButton 
                v-else 
                size="xs" 
                variant="ghost" 
                @click="openUnassignDialog(row)"
              >
                Unassign
              </OuiButton>
              <OuiButton size="xs" variant="ghost" @click="openEditDialog(row)">
                Edit
              </OuiButton>
              <OuiButton 
                v-if="!row.vpsId" 
                size="xs" 
                variant="ghost" 
                color="danger" 
                @click="openDeleteDialog(row)"
              >
                Delete
              </OuiButton>
            </OuiFlex>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Create IP Dialog -->
    <OuiDialog v-model:open="createDialogOpen" title="Create Public IP">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Add a new public IP address to the pool. You can assign it to a VPS instance later.
        </OuiText>
        <OuiField label="IP Address" required>
          <OuiInput 
            v-model="ipForm.ipAddress" 
            placeholder="e.g., 203.0.113.1 or 2001:db8::1" 
          />
          <OuiText size="xs" color="muted" class="mt-1">
            IPv4 or IPv6 address
          </OuiText>
        </OuiField>
        <OuiField label="Monthly Cost (USD)" required>
          <OuiInput 
            v-model="ipForm.monthlyCostUSD" 
            type="number" 
            min="0" 
            step="0.01" 
            placeholder="0.00" 
          />
          <OuiText size="xs" color="muted" class="mt-1">
            Monthly flat rate cost for this IP address
          </OuiText>
        </OuiField>
        <OuiField label="Gateway (optional)">
          <OuiInput 
            v-model="ipForm.gateway" 
            placeholder="e.g., 203.0.113.1" 
          />
          <OuiText size="xs" color="muted" class="mt-1">
            Gateway IP for this public IP. If not set, will default to IP with last octet set to .1
          </OuiText>
        </OuiField>
        <OuiField label="Netmask (optional)">
          <OuiInput 
            v-model="ipForm.netmask" 
            placeholder="e.g., 24 or 255.255.255.0" 
          />
          <OuiText size="xs" color="muted" class="mt-1">
            Netmask or CIDR notation (e.g., "24" for /24 or "255.255.255.0"). Defaults to "24" if not set.
          </OuiText>
        </OuiField>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="closeCreateDialog">Cancel</OuiButton>
          <OuiButton @click="createIP" :loading="isCreating">
            Create IP
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Edit Cost Dialog -->
    <OuiDialog v-model:open="editDialogOpen" :title="`Edit IP ${editingIP?.ipAddress || ''}`">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Update the monthly cost, gateway, and netmask for this IP address.
        </OuiText>
        <OuiField label="Monthly Cost (USD)" required>
          <OuiInput 
            v-model="editForm.monthlyCostUSD" 
            type="number" 
            min="0" 
            step="0.01" 
            placeholder="0.00" 
          />
        </OuiField>
        <OuiField label="Gateway (optional)">
          <OuiInput 
            v-model="editForm.gateway" 
            placeholder="e.g., 203.0.113.1" 
          />
          <OuiText size="xs" color="muted" class="mt-1">
            Gateway IP for this public IP. If not set, will default to IP with last octet set to .1
          </OuiText>
        </OuiField>
        <OuiField label="Netmask (optional)">
          <OuiInput 
            v-model="editForm.netmask" 
            placeholder="e.g., 24 or 255.255.255.0" 
          />
          <OuiText size="xs" color="muted" class="mt-1">
            Netmask or CIDR notation (e.g., "24" for /24 or "255.255.255.0"). Defaults to "24" if not set.
          </OuiText>
        </OuiField>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="closeEditDialog">Cancel</OuiButton>
          <OuiButton @click="updateIP" :loading="isUpdating">
            Update IP
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Assign IP Dialog -->
    <OuiDialog v-model:open="assignDialogOpen" :title="`Assign IP ${assigningIP?.ipAddress || ''}`">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Assign this public IP address to a VPS instance.
        </OuiText>
        <OuiField label="VPS Instance" required>
          <OuiSelect
            v-model="assignForm.vpsId"
            label="Select VPS"
            :items="vpsOptions"
            :loading="loadingVPS"
            placeholder="Search for a VPS..."
          />
        </OuiField>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="closeAssignDialog">Cancel</OuiButton>
          <OuiButton @click="assignIP" :loading="isAssigning">
            Assign IP
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Unassign IP Dialog -->
    <OuiDialog v-model:open="unassignDialogOpen" :title="`Unassign IP ${unassigningIP?.ipAddress || ''}`">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Unassign this public IP address from the VPS instance. The IP will be available for reassignment.
        </OuiText>
        <OuiText size="sm" v-if="unassigningIP?.vpsName">
          Currently assigned to: <OuiText as="strong" weight="semibold">{{ unassigningIP.vpsName }}</OuiText>
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="closeUnassignDialog">Cancel</OuiButton>
          <OuiButton color="danger" @click="unassignIP" :loading="isUnassigning">
            Unassign IP
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Delete IP Dialog -->
    <OuiDialog v-model:open="deleteDialogOpen" :title="`Delete IP ${ipToDelete?.ipAddress || ''}`">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Are you sure you want to delete this public IP address? This action cannot be undone.
        </OuiText>
        <OuiText size="sm" v-if="ipToDelete?.vpsId" color="warning">
          Warning: This IP is currently assigned to a VPS. Unassign it first before deleting.
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="closeDeleteDialog">Cancel</OuiButton>
          <OuiButton color="danger" @click="deleteIP" :loading="isDeleting">
            Delete IP
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
import { SuperadminService, type VPSPublicIP, type CreateVPSPublicIPRequest, type UpdateVPSPublicIPRequest } from "@obiente/proto";
import { useToast } from "~/composables/useToast";

const { toast } = useToast();
const client = useConnectClient(SuperadminService);

const ips = ref<VPSPublicIP[]>([]);
const isLoading = ref(false);
const isCreating = ref(false);
const isUpdating = ref(false);
const isAssigning = ref(false);
const isUnassigning = ref(false);
const isDeleting = ref(false);
const loadingVPS = ref(false);

const createDialogOpen = ref(false);
const editDialogOpen = ref(false);
const assignDialogOpen = ref(false);
const unassignDialogOpen = ref(false);
const deleteDialogOpen = ref(false);

const editingIP = ref<VPSPublicIP | null>(null);
const assigningIP = ref<VPSPublicIP | null>(null);
const unassigningIP = ref<VPSPublicIP | null>(null);
const ipToDelete = ref<VPSPublicIP | null>(null);
const vpsList = ref<Array<{
  id: string;
  name: string;
  organizationId: string;
  organizationName: string;
  region: string;
  status: number;
}>>([]);

const ipForm = ref({
  ipAddress: "",
  monthlyCostUSD: "",
  gateway: "",
  netmask: "",
});

const editForm = ref({
  monthlyCostUSD: "",
  gateway: "",
  netmask: "",
});

const assignForm = ref({
  vpsId: "",
});

const tableColumns = computed(() => [
  { key: "ipAddress", label: "IP Address", defaultWidth: 180, minWidth: 150 },
  { key: "vps", label: "Assigned To", defaultWidth: 250, minWidth: 200 },
  { key: "cost", label: "Monthly Cost", defaultWidth: 150, minWidth: 120 },
  { key: "assignedAt", label: "Assigned At", defaultWidth: 150, minWidth: 120 },
  { key: "createdAt", label: "Created At", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 200, minWidth: 180, resizable: false },
]);

const tableRows = computed(() => ips.value.map(ip => ({
  ...ip,
  // Transform monthlyCostCents to cost in cents for OuiCurrency
  cost: Number(ip.monthlyCostCents || 0n),
})));

const vpsOptions = computed(() => 
  vpsList.value
    .filter(vps => vps.status !== 8 && vps.status !== 9) // Exclude DELETING and DELETED
    .map(vps => ({
      value: vps.id,
      label: `${vps.name || vps.id} (${vps.organizationName || "Unknown"} · ${vps.organizationId || ""} · ${vps.region || "N/A"})`,
      organizationId: vps.organizationId || "",
      organizationName: vps.organizationName || "Unknown",
      region: vps.region || "N/A",
    }))
);

const fetchIPs = async () => {
  isLoading.value = true;
  try {
    const response = await client.listVPSPublicIPs({
      includeUnassigned: true,
      page: 1,
      perPage: 1000, // Get all IPs for now
    });
    ips.value = response.ips || [];
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    toast.error(`Failed to load IPs: ${message}`);
  } finally {
    isLoading.value = false;
  }
};

const fetchVPSList = async () => {
  loadingVPS.value = true;
  try {
    const response = await client.listAllVPS({
      page: 1,
      perPage: 1000, // Get all VPS for assignment
    });
    vpsList.value = (response.vpsInstances || []).map((inst) => ({
      id: inst.vps?.id || "",
      name: inst.vps?.name || "",
      organizationId: inst.vps?.organizationId || "",
      organizationName: inst.organizationName || "Unknown",
      region: inst.vps?.region || "N/A",
      status: inst.vps?.status || 0,
    }));
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    toast.error(`Failed to load VPS list: ${message}`);
  } finally {
    loadingVPS.value = false;
  }
};

const openCreateDialog = () => {
  ipForm.value = {
    ipAddress: "",
    monthlyCostUSD: "",
    gateway: "",
    netmask: "",
  };
  createDialogOpen.value = true;
};

const closeCreateDialog = () => {
  createDialogOpen.value = false;
  ipForm.value = {
    ipAddress: "",
    monthlyCostUSD: "",
    gateway: "",
    netmask: "",
  };
};

const createIP = async () => {
  if (!ipForm.value.ipAddress || !ipForm.value.monthlyCostUSD) {
    toast.error("Please fill in all required fields");
    return;
  }

  const monthlyCostCents = Math.round(parseFloat(ipForm.value.monthlyCostUSD) * 100);
  if (isNaN(monthlyCostCents) || monthlyCostCents < 0) {
    toast.error("Invalid monthly cost");
    return;
  }

  isCreating.value = true;
  try {
    const request: Partial<CreateVPSPublicIPRequest> = {
      ipAddress: ipForm.value.ipAddress.trim(),
      monthlyCostCents: BigInt(monthlyCostCents),
    };
    
    if (ipForm.value.gateway?.trim()) {
      request.gateway = ipForm.value.gateway.trim();
    }
    
    if (ipForm.value.netmask?.trim()) {
      request.netmask = ipForm.value.netmask.trim();
    }
    
    await client.createVPSPublicIP(request as CreateVPSPublicIPRequest);
    toast.success("Public IP created successfully");
    closeCreateDialog();
    await fetchIPs();
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    toast.error(`Failed to create IP: ${message}`);
  } finally {
    isCreating.value = false;
  }
};

const openEditDialog = (ip: VPSPublicIP) => {
  editingIP.value = ip;
  editForm.value = {
    monthlyCostUSD: (Number(ip.monthlyCostCents) / 100).toFixed(2),
    gateway: ip.gateway || "",
    netmask: ip.netmask || "",
  };
  editDialogOpen.value = true;
};

const closeEditDialog = () => {
  editDialogOpen.value = false;
  editingIP.value = null;
  editForm.value = {
    monthlyCostUSD: "",
    gateway: "",
    netmask: "",
  };
};

const updateIP = async () => {
  if (!editingIP.value || !editForm.value.monthlyCostUSD) {
    toast.error("Please fill in all required fields");
    return;
  }

  const monthlyCostCents = Math.round(parseFloat(editForm.value.monthlyCostUSD) * 100);
  if (isNaN(monthlyCostCents) || monthlyCostCents < 0) {
    toast.error("Invalid monthly cost");
    return;
  }

  isUpdating.value = true;
  try {
    const request: Partial<UpdateVPSPublicIPRequest> = {
      id: editingIP.value.id,
      monthlyCostCents: BigInt(monthlyCostCents),
    };
    
    // Always include gateway and netmask (empty string to clear if not provided)
    const gateway = editForm.value.gateway?.trim();
    const netmask = editForm.value.netmask?.trim();
    if (gateway) {
      request.gateway = gateway;
    }
    if (netmask) {
      request.netmask = netmask;
    }
    
    await client.updateVPSPublicIP(request as UpdateVPSPublicIPRequest);
    toast.success("IP updated successfully");
    closeEditDialog();
    await fetchIPs();
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    toast.error(`Failed to update IP: ${message}`);
  } finally {
    isUpdating.value = false;
  }
};

const openAssignDialog = (ip: VPSPublicIP) => {
  assigningIP.value = ip;
  assignForm.value = {
    vpsId: "",
  };
  assignDialogOpen.value = true;
};

const closeAssignDialog = () => {
  assignDialogOpen.value = false;
  assigningIP.value = null;
  assignForm.value = {
    vpsId: "",
  };
};

const assignIP = async () => {
  if (!assigningIP.value || !assignForm.value.vpsId) {
    toast.error("Please select a VPS instance");
    return;
  }

  isAssigning.value = true;
    try {
    await client.assignVPSPublicIP({
      publicIp: assigningIP.value.ipAddress,
      vpsId: assignForm.value.vpsId,
    });
    toast.success("IP assigned successfully");
    closeAssignDialog();
    await fetchIPs();
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    toast.error(`Failed to assign IP: ${message}`);
  } finally {
    isAssigning.value = false;
  }
};

const openUnassignDialog = (ip: VPSPublicIP) => {
  unassigningIP.value = ip;
  unassignDialogOpen.value = true;
};

const closeUnassignDialog = () => {
  unassignDialogOpen.value = false;
  unassigningIP.value = null;
};

const unassignIP = async () => {
  if (!unassigningIP.value) return;

  isUnassigning.value = true;
    try {
    await client.unassignVPSPublicIP({
      publicIp: unassigningIP.value.ipAddress,
    });
    toast.success("IP unassigned successfully");
    closeUnassignDialog();
    await fetchIPs();
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    toast.error(`Failed to unassign IP: ${message}`);
  } finally {
    isUnassigning.value = false;
  }
};

const openDeleteDialog = (ip: VPSPublicIP) => {
  ipToDelete.value = ip;
  deleteDialogOpen.value = true;
};

const closeDeleteDialog = () => {
  deleteDialogOpen.value = false;
  ipToDelete.value = null;
};

const deleteIP = async () => {
  if (!ipToDelete.value) return;

  if (ipToDelete.value.vpsId) {
    toast.error("Cannot delete IP: it is currently assigned to a VPS. Unassign it first.");
    closeDeleteDialog();
    return;
  }

  isDeleting.value = true;
  try {
    await client.deleteVPSPublicIP({
      id: ipToDelete.value.id,
    });
    toast.success("IP deleted successfully");
    closeDeleteDialog();
    await fetchIPs();
  } catch (error) {
    const message = error instanceof Error ? error.message : "Unknown error";
    toast.error(`Failed to delete IP: ${message}`);
  } finally {
    isDeleting.value = false;
  }
};

onMounted(() => {
  fetchIPs();
  fetchVPSList();
});
</script>

