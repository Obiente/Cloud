<template>
  <SuperadminPageLayout
    title="Nodes"
    description="View and configure cluster nodes independently."
    :columns="tableColumns"
    :rows="tableRows"
    :filters="filterConfigs"
    :search="search"
    :empty-text="isLoading ? 'Loading nodes…' : 'No nodes match your filters.'"
    :loading="isLoading"
    search-placeholder="Search by hostname, ID, IP, role, region…"
    @update:search="search = $event"
    @filter-change="handleFilterChange"
    @refresh="refresh"
  >
    <template #cell-node="{ value, row }">
      <SuperadminResourceCell
        :name="row.hostname"
        :subtitle="row.ip"
        :id="row.id"
      />
      <div v-if="row.region" class="text-xs text-text-muted mt-0.5">
        Region: {{ row.region }}
      </div>
    </template>
    <template #cell-role="{ value }">
      <SuperadminStatusBadge
        :status="value?.toLowerCase()"
        :status-map="roleStatusMap"
      />
    </template>
    <template #cell-status="{ value }">
      <SuperadminStatusBadge
        :status="value?.toLowerCase()"
        :status-map="nodeStatusMap"
      />
    </template>
    <template #cell-resources="{ value, row }">
      <div class="text-sm">
        <div>CPU: {{ row.usedCpu.toFixed(1) }}% / {{ row.totalCpu }} cores</div>
        <div class="text-text-muted">
          Memory: {{ formatBytes(row.usedMemory) }} / {{ formatBytes(row.totalMemory) }}
        </div>
      </div>
    </template>
    <template #cell-deployments="{ value, row }">
      <span class="font-mono">{{ row.deploymentCount }} / {{ row.maxDeployments }}</span>
    </template>
    <template #cell-config="{ value, row }">
      <div class="text-sm space-y-1">
        <template v-if="isSwarmNode(row)">
          <div v-if="row.config?.subdomain" class="text-text-secondary">
            Subdomain: <span class="font-mono">{{ row.config.subdomain }}</span>
          </div>
          <div v-if="row.config?.useNodeSpecificDomains !== undefined" class="text-text-secondary">
            Node-specific domains: {{ row.config.useNodeSpecificDomains ? 'Enabled' : 'Disabled' }}
          </div>
          <div v-if="row.config?.serviceDomainPattern" class="text-text-secondary">
            Pattern: <span class="font-mono">{{ row.config.serviceDomainPattern }}</span>
          </div>
          <div v-if="!row.config?.subdomain && row.config?.useNodeSpecificDomains === undefined" class="text-text-muted text-xs">
            Using defaults
          </div>
        </template>
        <template v-else>
          <div class="text-text-muted text-xs">
            Compose deployment
          </div>
          <div class="text-text-muted text-xs mt-1">
            Configure via env vars: NODE_SUBDOMAIN, USE_NODE_SPECIFIC_DOMAINS
          </div>
        </template>
      </div>
    </template>
    <template #cell-actions="{ row }">
      <SuperadminActionsCell :actions="getNodeActions(row)" />
    </template>
  </SuperadminPageLayout>

  <!-- Edit Node Config Dialog -->
  <OuiDialog v-model:open="editNodeDialogOpen" :title="`Configure Node: ${editingNode?.hostname || ''}`">
    <OuiStack gap="lg" v-if="editingNode">
      <OuiStack gap="md">
        <!-- Node subdomain configuration - only for Swarm stack services -->
        <template v-if="isSwarmNode(editingNode)">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Node Subdomain</OuiText>
            <OuiText size="xs" color="muted">
              Identifier for this node (e.g., "node1", "us-east-1"). Used for node-specific domains in Swarm deployments.
            </OuiText>
            <OuiInput
              v-model="editNodeForm.subdomain"
              type="text"
              placeholder="node1"
            />
          </OuiStack>

          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Use Node-Specific Domains</OuiText>
            <OuiText size="xs" color="muted">
              When enabled, microservices on this node use node-specific subdomains (e.g., "node1-auth-service.domain").
              Only applies to Swarm stack services.
            </OuiText>
            <OuiSwitch
              v-model="editNodeForm.useNodeSpecificDomains"
              label="Enable node-specific domains"
            />
          </OuiStack>

          <OuiStack gap="xs" v-if="editNodeForm.useNodeSpecificDomains">
            <OuiText size="sm" weight="medium">Service Domain Pattern</OuiText>
            <OuiText size="xs" color="muted">
              Pattern for constructing node-specific domains.
            </OuiText>
            <OuiSelect
              v-model="editNodeForm.serviceDomainPattern"
              :items="domainPatternOptions"
            />
          </OuiStack>
        </template>
        <template v-else>
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium" color="muted">Node Subdomain Configuration</OuiText>
            <OuiText size="xs" color="muted">
              Node subdomain configuration is only available for Swarm stack services.
              For compose deployments, configure via environment variables in your docker-compose file.
            </OuiText>
            <div class="mt-2 p-3 bg-surface-secondary rounded-md">
              <OuiText size="xs" weight="medium" class="mb-2">Required Environment Variables:</OuiText>
              <div class="space-y-1 font-mono text-xs">
                <div><span class="text-text-secondary">NODE_SUBDOMAIN</span>=<span class="text-text-primary">node1</span></div>
                <div><span class="text-text-secondary">USE_NODE_SPECIFIC_DOMAINS</span>=<span class="text-text-primary">true</span></div>
                <div><span class="text-text-secondary">SERVICE_DOMAIN_PATTERN</span>=<span class="text-text-primary">node-service</span></div>
              </div>
            </div>
          </OuiStack>
        </template>

        <OuiStack gap="xs">
          <OuiText size="sm" weight="medium">Region</OuiText>
          <OuiText size="xs" color="muted">
            Region identifier for this node (e.g., "us-east-1", "eu-west-1").
          </OuiText>
          <OuiInput
            v-model="editNodeForm.region"
            type="text"
            placeholder="us-east-1"
          />
        </OuiStack>

        <OuiStack gap="xs">
          <OuiText size="sm" weight="medium">Max Deployments</OuiText>
          <OuiText size="xs" color="muted">
            Maximum number of deployments allowed on this node.
          </OuiText>
          <OuiInput
            v-model="editNodeForm.maxDeployments"
            type="number"
            min="1"
            placeholder="50"
          />
        </OuiStack>
      </OuiStack>

      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="editNodeDialogOpen = false">
          Cancel
        </OuiButton>
        <OuiButton
          variant="solid"
          :loading="isSaving"
          @click="saveNodeConfig"
        >
          Save Configuration
        </OuiButton>
      </OuiFlex>
    </OuiStack>
  </OuiDialog>
</template>

<script setup lang="ts">
definePageMeta({
  middleware: ["auth", "superadmin"],
});

import { ref, computed, onMounted } from "vue";
import { useSuperAdmin } from "~/composables/useSuperAdmin";
import { useToast } from "~/composables/useToast";
import { formatBytes } from "~/utils/common";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import SuperadminResourceCell from "~/components/superadmin/SuperadminResourceCell.vue";
import SuperadminStatusBadge from "~/components/superadmin/SuperadminStatusBadge.vue";
import SuperadminActionsCell from "~/components/superadmin/SuperadminActionsCell.vue";
import type { TableColumn } from "~/components/oui/Table.vue";
import type { FilterConfig } from "~/components/superadmin/SuperadminFilterBar.vue";
import type { BadgeVariant } from "~/components/oui/Badge.vue";
import { ServerIcon, Cog6ToothIcon } from "@heroicons/vue/24/outline";

const superAdmin = useSuperAdmin();
const { toast } = useToast();

const isLoading = ref(false);
const isSaving = ref(false);
const search = ref("");
const nodes = ref<any[]>([]);
const editNodeDialogOpen = ref(false);
const editingNode = ref<any | null>(null);
const editNodeForm = ref({
  subdomain: "",
  useNodeSpecificDomains: false,
  serviceDomainPattern: "node-service",
  region: "",
  maxDeployments: "50",
});

const domainPatternOptions = [
  { key: "node-service", value: "node-service", label: "node-service (node1-auth-service.domain)" },
  { key: "service-node", value: "service-node", label: "service-node (auth-service.node1.domain)" },
];

const roleStatusMap: Record<string, { label: string; variant: BadgeVariant }> = {
  manager: { label: "Manager", variant: "primary" },
  worker: { label: "Worker", variant: "secondary" },
};

const nodeStatusMap: Record<string, { label: string; variant: BadgeVariant }> = {
  ready: { label: "Ready", variant: "success" },
  down: { label: "Down", variant: "danger" },
};

const filterConfigs: FilterConfig[] = [
  {
    key: "role",
    placeholder: "Role",
    items: [
      { key: "", value: "", label: "All Roles" },
      { key: "manager", value: "manager", label: "Manager" },
      { key: "worker", value: "worker", label: "Worker" },
    ],
  },
  {
    key: "availability",
    placeholder: "Availability",
    items: [
      { key: "", value: "", label: "All" },
      { key: "active", value: "active", label: "Active" },
      { key: "pause", value: "pause", label: "Pause" },
      { key: "drain", value: "drain", label: "Drain" },
    ],
  },
  {
    key: "status",
    placeholder: "Status",
    items: [
      { key: "", value: "", label: "All" },
      { key: "ready", value: "ready", label: "Ready" },
      { key: "down", value: "down", label: "Down" },
    ],
  },
];

const tableColumns: TableColumn[] = [
  { key: "node", label: "Node", sortable: true },
  { key: "role", label: "Role", sortable: true },
  { key: "status", label: "Status", sortable: true },
  { key: "resources", label: "Resources" },
  { key: "deployments", label: "Deployments", sortable: true },
  { key: "config", label: "Configuration" },
  { key: "actions", label: "Actions" },
];

const tableRows = computed(() => {
  const term = search.value.trim().toLowerCase();
  let filtered = nodes.value;

  if (term) {
    filtered = filtered.filter((node) => {
      const searchable = [
        node.hostname,
        node.id,
        node.ip,
        node.role,
        node.region,
        node.config?.subdomain,
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();
      return searchable.includes(term);
    });
  }

  return filtered;
});

// Check if node is a Swarm node (not a compose deployment)
// Swarm nodes have node IDs that don't start with "local-"
const isSwarmNode = (node: any) => {
  return node?.id && !node.id.startsWith("local-");
};

const getNodeActions = (node: any) => {
  return [
    {
      label: "Configure",
      icon: Cog6ToothIcon,
      onClick: () => openEditDialog(node),
    },
  ];
};

const openEditDialog = (node: any) => {
  editingNode.value = node;
  editNodeForm.value = {
    subdomain: node.config?.subdomain || "",
    useNodeSpecificDomains: node.config?.useNodeSpecificDomains ?? false,
    serviceDomainPattern: node.config?.serviceDomainPattern || "node-service",
    region: node.region || "",
    maxDeployments: String(node.maxDeployments || 50),
  };
  editNodeDialogOpen.value = true;
};

const saveNodeConfig = async () => {
  if (!editingNode.value) return;

  isSaving.value = true;
  try {
    // Only send subdomain config for Swarm nodes
    const isSwarm = isSwarmNode(editingNode.value);
    await superAdmin.updateNodeConfig({
      nodeId: editingNode.value.id,
      subdomain: isSwarm ? (editNodeForm.value.subdomain || undefined) : undefined,
      useNodeSpecificDomains: isSwarm ? (editNodeForm.value.useNodeSpecificDomains || undefined) : undefined,
      serviceDomainPattern: isSwarm ? (editNodeForm.value.serviceDomainPattern || undefined) : undefined,
      region: editNodeForm.value.region || undefined,
      maxDeployments: editNodeForm.value.maxDeployments ? parseInt(editNodeForm.value.maxDeployments, 10) : undefined,
    });

    toast.success(
      "Node configuration updated",
      `Configuration for ${editingNode.value.hostname} has been updated.`
    );

    editNodeDialogOpen.value = false;
    await refresh();
  } catch (error: any) {
    toast.error(
      "Failed to update node configuration",
      error.message || "An error occurred"
    );
  } finally {
    isSaving.value = false;
  }
};

const handleFilterChange = (key: string, value: string) => {
  // Filter logic would go here if needed
  refresh();
};

const refresh = async () => {
  isLoading.value = true;
  try {
    const response = await superAdmin.listNodes({
      role: undefined,
      availability: undefined,
      status: undefined,
      region: undefined,
    });
    nodes.value = response.nodes || [];
  } catch (error: any) {
    toast.error(
      "Failed to load nodes",
      error.message || "An error occurred"
    );
  } finally {
    isLoading.value = false;
  }
};

onMounted(() => {
  refresh();
});
</script>

