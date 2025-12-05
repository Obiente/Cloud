<template>
  <OuiStack gap="md">
    <OuiText size="xl" weight="semibold">Role Bindings</OuiText>
    <OuiCard>
      <OuiCardBody>
        <form @submit.prevent="create">
          <OuiStack gap="lg">
            <!-- Basic Binding Info -->
            <OuiStack gap="md">
              <OuiText size="sm" weight="semibold" transform="uppercase" class="tracking-wide" color="secondary">
                Basic Information
              </OuiText>
              <OuiGrid cols="1" colsMd="3" gap="md">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Organization</OuiText>
                  <OuiSelect v-model="selectedOrg" :items="orgItems" />
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Member</OuiText>
                  <OuiCombobox
                    v-model="userId"
                    :options="memberItems"
                    placeholder="Search for a member..."
                  />
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Role</OuiText>
                  <OuiCombobox
                    v-model="roleId"
                    :options="roleItemsForCombobox"
                    placeholder="Search for a role..."
                  />
                  <OuiText v-if="selectedRolePermissions.length > 0" size="xs" color="secondary" class="mt-1">
                    {{ selectedRolePermissions.length }} permission{{ selectedRolePermissions.length !== 1 ? 's' : '' }}
                  </OuiText>
                </OuiStack>
              </OuiGrid>
            </OuiStack>

            <!-- Resource Scoping (Optional) -->
            <OuiStack gap="md">
              <OuiFlex justify="between" align="center">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="semibold" transform="uppercase" class="tracking-wide" color="secondary">
                    Resource Scoping (Optional)
                  </OuiText>
                  <OuiText size="xs" color="secondary">
                    Leave empty for organization-wide access, or scope to specific resources
                  </OuiText>
                </OuiStack>
                <OuiButton
                  v-if="resourceType"
                  variant="ghost"
                  size="xs"
                  @click.prevent="clearResourceScope"
                >
                  Clear
                </OuiButton>
              </OuiFlex>
              
              <OuiGrid cols="1" colsMd="2" gap="md">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">Resource Type</OuiText>
                  <OuiSelect 
                    v-model="resourceType" 
                    :items="resourceTypeItems"
                    placeholder="Select resource type..."
                  />
                  <OuiText v-if="resourceType === 'deployment'" size="xs" color="secondary" class="mt-1">
                    Grant access to a specific deployment
                  </OuiText>
                  <OuiText v-else-if="resourceType === 'environment'" size="xs" color="secondary" class="mt-1">
                    Grant access to all deployments in selected environment(s)
                  </OuiText>
                  <OuiText v-else-if="resourceType === 'vps'" size="xs" color="secondary" class="mt-1">
                    Grant access to a specific VPS instance
                  </OuiText>
                  <OuiText v-else-if="resourceType === 'gameserver'" size="xs" color="secondary" class="mt-1">
                    Grant access to a specific game server
                  </OuiText>
                </OuiStack>
                <OuiStack gap="xs" v-if="resourceType">
                  <OuiText size="sm" weight="medium">
                    {{ resourceType === 'deployment' ? 'Deployment' : resourceType === 'environment' ? 'Environment(s)' : resourceType === 'vps' ? 'VPS Instance' : resourceType === 'gameserver' ? 'Game Server' : 'Resource ID' }}
                  </OuiText>
                  <OuiCombobox
                    v-if="resourceType === 'deployment'"
                    v-model="deploymentId"
                    :options="deploymentItems"
                    placeholder="Search for a deployment..."
                  />
                  <OuiCombobox
                    v-else-if="resourceType === 'vps'"
                    v-model="vpsId"
                    :options="vpsItems"
                    placeholder="Search for a VPS instance..."
                  />
                  <OuiCombobox
                    v-else-if="resourceType === 'gameserver'"
                    v-model="gameserverId"
                    :options="gameserverItems"
                    placeholder="Search for a game server..."
                  />
                  <OuiSelect
                    v-else-if="resourceType === 'environment'"
                    multiple
                    v-model="resourceIds"
                    :items="environmentItems"
                    placeholder="Select environments..."
                  />
                  <OuiInput
                    v-else
                    v-model="resourceIdsString"
                    placeholder="Enter resource ID or * for all"
                  />
                  <OuiText v-if="resourceType === 'deployment' && deploymentId" size="xs" color="secondary" class="mt-1">
                    Selected: {{ getDeploymentName(deploymentId) }}
                  </OuiText>
                  <OuiText v-else-if="resourceType === 'vps' && vpsId" size="xs" color="secondary" class="mt-1">
                    Selected: {{ getVPSName(vpsId) }}
                  </OuiText>
                  <OuiText v-else-if="resourceType === 'gameserver' && gameserverId" size="xs" color="secondary" class="mt-1">
                    Selected: {{ getGameServerName(gameserverId) }}
                  </OuiText>
                  <OuiText v-else-if="resourceType === 'environment' && Array.isArray(resourceIds) && resourceIds.length > 0" size="xs" color="secondary" class="mt-1">
                    {{ resourceIds.length }} environment{{ resourceIds.length !== 1 ? 's' : '' }} selected
                  </OuiText>
                </OuiStack>
              </OuiGrid>
            </OuiStack>

            <!-- Binding Preview -->
            <OuiCard v-if="bindingPreview" variant="outline" class="bg-surface-subtle">
              <OuiCardBody>
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="semibold">Binding Preview</OuiText>
                  <OuiText size="xs" color="secondary">
                    {{ bindingPreview }}
                  </OuiText>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <OuiFlex class="mt-6" gap="md" justify="start">
              <OuiButton type="submit">Bind</OuiButton>
            </OuiFlex>
          </OuiStack>
        </form>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>Bindings</OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiFlex gap="xs">
            <OuiButton variant="ghost" size="sm" @click="refreshAll">Refresh</OuiButton>
          </OuiFlex>
          <OuiStack gap="xs">
            <OuiFlex
              v-for="b in bindingItems"
              :key="b.id"
              align="center"
              gap="xs"
            >
              <OuiText size="sm">
                {{ b.user }} → {{ b.role }} ({{ b.resourceType }} {{ b.resource }})
              </OuiText>
            </OuiFlex>
            <OuiFlex v-if="!bindingItems || bindingItems.length === 0" justify="center" py="lg">
              <OuiText color="secondary" size="sm">No bindings found</OuiText>
            </OuiFlex>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { storeToRefs } from "pinia";
import { useOrganizationsStore } from "~/stores/organizations";
import { useOrganizationId } from "~/composables/useOrganizationId";
import { OrganizationService, DeploymentService, AdminService, VPSService, GameServerService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import OuiCombobox from "~/components/oui/Combobox.vue";

definePageMeta({ layout: "admin", middleware: "auth" });

const orgStore = useOrganizationsStore();
orgStore.hydrate();
const { orgs, currentOrgId } = storeToRefs(orgStore);

const userId = ref("");
const roleId = ref("");
const resourceType = ref("");
const resourceIds = ref<string[] | string>("");
const deploymentId = ref("");
const vpsId = ref("");
const gameserverId = ref("");

// Computed property for OuiInput (string) binding
const resourceIdsString = computed({
  get: () => {
    const value = resourceIds.value;
    return Array.isArray(value) ? value.join(",") : value || "";
  },
  set: (val: string) => {
    resourceIds.value = val;
  },
});

const resourceTypeItems = [
  { label: "Deployment", value: "deployment" },
  { label: "Environment", value: "environment" },
  { label: "VPS", value: "vps" },
  { label: "Game Server", value: "gameserver" },
];
const environmentItems = [
  { label: "Production", value: "production" },
  { label: "Staging", value: "staging" },
  { label: "Development", value: "development" },
];

const orgClient = useConnectClient(OrganizationService);
const depClient = useConnectClient(DeploymentService);
const adminClient = useConnectClient(AdminService);
const vpsClient = useConnectClient(VPSService);
const gameserverClient = useConnectClient(GameServerService);

if (!orgs.value.length) {
  try {
    const res = await orgClient.listOrganizations({});
    orgStore.setOrganizations(res.organizations || []);
  } catch (e) {
    console.error("Failed to load organizations", e);
  }
}

const orgItems = computed(() =>
  (orgs.value || []).map((o) => ({
    label: o.name ?? o.id,
    value: o.id,
  }))
);

// Get organizationId using SSR-compatible composable
const organizationId = useOrganizationId();

const selectedOrg = computed({
  get: () => organizationId.value || currentOrgId.value || "",
  set: (id: string) => {
    if (id) orgStore.switchOrganization(id);
  },
});

const { data: bindingsData, refresh: refreshBindings } = await useClientFetch(
  () =>
    organizationId.value
      ? `admin-bindings-${organizationId.value}`
      : "admin-bindings-none",
  async () => {
    const orgId = organizationId.value;
    if (!orgId) return [];
    const res = await adminClient.listRoleBindings({
      organizationId: orgId,
    });
    return res.bindings || [];
  },
  { watch: [selectedOrg] }
);
const bindings = computed(() => bindingsData.value || []);

const { data: roleOptionsData, refresh: refreshRoleOptions } = await useClientFetch(
  () =>
    organizationId.value
      ? `admin-binding-roles-${organizationId.value}`
      : "admin-binding-roles-none",
  async () => {
    const orgId = organizationId.value;
    if (!orgId) return [];
    const res = await adminClient.listRoles({
      organizationId: orgId,
    });
    const roles = res.roles || [];
    if (!roleId.value && roles.length) {
      roleId.value = roles[0]?.id ?? "";
    }
    return roles.map((r) => ({ 
      id: r.id, 
      name: r.name,
      permissionsJson: r.permissionsJson || "[]"
    }));
  },
  { watch: [selectedOrg] }
);
const roleLabelMap = computed(() => {
  const map = new Map<string, string>();
  (roleOptionsData.value || []).forEach((r) => map.set(r.id, r.name));
  return map;
});
const roleItems = computed(() =>
  (roleOptionsData.value || []).map((r) => ({ label: r.name, value: r.id }))
);

const { data: memberOptionsData, refresh: refreshMemberOptions } = await useClientFetch(
  () =>
    organizationId.value
      ? `admin-binding-members-${organizationId.value}`
      : "admin-binding-members-none",
  async () => {
    const orgId = organizationId.value;
    if (!orgId) return [];
    const res = await orgClient.listMembers({
      organizationId: orgId,
    });
    const members = res.members || [];
    if (!userId.value && members.length) {
      userId.value = members[0]?.user?.id || members[0]?.id || "";
    }
    return members.map((m) => ({
      id: m.user?.id || m.id,
      label: m.user?.name || m.user?.email || m.id,
    }));
  },
  { watch: [selectedOrg] }
);
const memberLabelMap = computed(() => {
  const map = new Map<string, string>();
  (memberOptionsData.value || []).forEach((m) => map.set(m.id, m.label));
  return map;
});
const memberItems = computed(() =>
  (memberOptionsData.value || []).map((m) => ({
    label: m.label,
    value: m.id,
  }))
);

const roleItemsForCombobox = computed(() =>
  (roleOptionsData.value || []).map((r) => ({
    label: r.name,
    value: r.id,
  }))
);

// Get selected role's permissions for preview
const selectedRole = computed(() => {
  if (!roleId.value) return null;
  return (roleOptionsData.value || []).find((r) => r.id === roleId.value);
});

const selectedRolePermissions = computed(() => {
  if (!selectedRole.value) return [];
  try {
    const perms = JSON.parse(selectedRole.value.permissionsJson || "[]");
    return Array.isArray(perms) ? perms : [];
  } catch {
    return [];
  }
});

// Get deployment name for preview
const getDeploymentName = (id: string) => {
  if (!id) return "";
  const deployment = (deploymentOptionsData.value || []).find((d) => d.id === id);
  return deployment?.name || id;
};

// Get VPS name for preview
const getVPSName = (id: string) => {
  if (!id) return "";
  const vps = (vpsOptionsData.value || []).find((v) => v.id === id);
  return vps?.name || id;
};

// Get game server name for preview
const getGameServerName = (id: string) => {
  if (!id) return "";
  const gameserver = (gameserverOptionsData.value || []).find((gs) => gs.id === id);
  return gameserver?.name || id;
};

// Binding preview text
const bindingPreview = computed(() => {
  if (!userId.value || !roleId.value) return null;
  
  const memberName = memberLabelMap.value.get(userId.value) || userId.value;
  const roleName = roleLabelMap.value.get(roleId.value) || roleId.value;
  
  let scope = "organization-wide";
  if (resourceType.value === "deployment" && deploymentId.value) {
    scope = `deployment "${getDeploymentName(deploymentId.value)}"`;
  } else if (resourceType.value === "vps" && vpsId.value) {
    scope = `VPS "${getVPSName(vpsId.value)}"`;
  } else if (resourceType.value === "gameserver" && gameserverId.value) {
    scope = `game server "${getGameServerName(gameserverId.value)}"`;
  } else if (resourceType.value === "environment" && Array.isArray(resourceIds.value) && resourceIds.value.length > 0) {
    const envNames = resourceIds.value.map((e) => {
      const env = environmentItems.find((item) => item.value === e);
      return env?.label || e;
    });
    scope = `${envNames.join(", ")} environment${envNames.length !== 1 ? "s" : ""}`;
  } else if (resourceType.value && resourceIdsString.value) {
    scope = `${resourceType.value} "${resourceIdsString.value}"`;
  }
  
  return `${memberName} will have the "${roleName}" role with ${scope} access`;
});

function clearResourceScope() {
  resourceType.value = "";
  deploymentId.value = "";
  vpsId.value = "";
  gameserverId.value = "";
  resourceIds.value = "";
  resourceIdsString.value = "";
}

const { data: deploymentOptionsData, refresh: refreshDeploymentOptions } =
  await useClientFetch(
    () =>
      organizationId.value
        ? `admin-binding-deployments-${organizationId.value}`
        : "admin-binding-deployments-none",
    async () => {
      const orgId = organizationId.value;
      if (!orgId) return [];
      const res = await depClient.listDeployments({
        organizationId: orgId,
      });
      return (res.deployments || []).map((d) => ({
        id: d.id,
        name: d.name || d.id,
      }));
    },
    { watch: [selectedOrg] }
  );
const deploymentItems = computed(() =>
  (deploymentOptionsData.value || []).map((d) => ({
    label: d.name,
    value: d.id,
  }))
);

// Load VPS instances for combobox
const { data: vpsOptionsData, refresh: refreshVPSOptions } = await useClientFetch(
  () =>
    organizationId.value
      ? `admin-binding-vps-${organizationId.value}`
      : "admin-binding-vps-none",
  async () => {
    const orgId = organizationId.value;
    if (!orgId) return [];
    const res = await vpsClient.listVPS({
      organizationId: orgId,
      page: 1,
      perPage: 100,
    });
    return (res.vpsInstances || []).map((v) => ({
      id: v.id,
      name: v.name || v.id,
    }));
  },
  { watch: [selectedOrg] }
);
const vpsItems = computed(() =>
  (vpsOptionsData.value || []).map((v) => ({
    label: v.name,
    value: v.id,
  }))
);

// Load game server instances for combobox
const { data: gameserverOptionsData, refresh: refreshGameserverOptions } = await useClientFetch(
  () =>
    organizationId.value
      ? `admin-binding-gameservers-${organizationId.value}`
      : "admin-binding-gameservers-none",
  async () => {
    const orgId = organizationId.value;
    if (!orgId) return [];
    const res = await gameserverClient.listGameServers({
      organizationId: orgId,
    });
    return (res.gameServers || []).map((gs) => ({
      id: gs.id,
      name: gs.name || gs.id,
    }));
  },
  { watch: [selectedOrg] }
);
const gameserverItems = computed(() =>
  (gameserverOptionsData.value || []).map((gs) => ({
    label: gs.name,
    value: gs.id,
  }))
);

// Handle deployment selection - auto-set resource type if not set
watch(deploymentId, (val) => {
  if (val && !resourceType.value) {
    resourceType.value = 'deployment';
  }
  if (resourceType.value === 'deployment') {
    resourceIds.value = val ? [val] : [];
  }
});

// Handle VPS selection - auto-set resource type if not set
watch(vpsId, (val) => {
  if (val && !resourceType.value) {
    resourceType.value = 'vps';
  }
  if (resourceType.value === 'vps') {
    resourceIds.value = val ? [val] : [];
  }
});

// Handle game server selection - auto-set resource type if not set
watch(gameserverId, (val) => {
  if (val && !resourceType.value) {
    resourceType.value = 'gameserver';
  }
  if (resourceType.value === 'gameserver') {
    resourceIds.value = val ? [val] : [];
  }
});

watch(() => resourceType.value, (newType, oldType) => {
  // Only clear if switching to a different resource type
  if (newType !== 'deployment') {
    deploymentId.value = "";
  }
  if (newType !== 'vps') {
    vpsId.value = "";
  }
  if (newType !== 'gameserver') {
    gameserverId.value = "";
  }
  if (newType !== 'environment') {
    resourceIds.value = Array.isArray(resourceIds.value) ? [] : "";
  }
  // If switching to deployment and we have a deployment selected, keep it
  if (newType === 'deployment' && deploymentId.value) {
    resourceIds.value = [deploymentId.value];
  }
  // If switching to VPS and we have a VPS selected, keep it
  if (newType === 'vps' && vpsId.value) {
    resourceIds.value = [vpsId.value];
  }
  // If switching to gameserver and we have a gameserver selected, keep it
  if (newType === 'gameserver' && gameserverId.value) {
    resourceIds.value = [gameserverId.value];
  }
});

const bindingItems = computed(() =>
  (bindings.value as any[]).map((b) => ({
    id: b.id,
    user:
      memberLabelMap.value.get(b.userId ?? b.user_id) ??
      b.userId ??
      b.user_id ??
      "",
    role:
      roleLabelMap.value.get(b.roleId ?? b.role_id) ??
      b.roleId ??
      b.role_id ??
      "",
    resourceType: b.resourceType ?? b.resource_type ?? "org",
    resource: b.resourceId ?? b.resource_id ?? "",
  }))
);

watch(
  () => selectedOrg.value,
  async (org) => {
    if (!org) return;
    userId.value = "";
    roleId.value = "";
    resourceType.value = "";
    resourceIds.value = "";
    deploymentId.value = "";
    vpsId.value = "";
    gameserverId.value = "";
    await refreshAll();
  },
  { immediate: true }
);

async function create() {
  if (!selectedOrg.value || !userId.value || !roleId.value) return;
  
  // Determine resource ID based on resource type
  let resourceIdToUse = "";
  if (resourceType.value === "deployment" && deploymentId.value) {
    resourceIdToUse = deploymentId.value;
  } else if (resourceType.value === "vps" && vpsId.value) {
    resourceIdToUse = vpsId.value;
  } else if (resourceType.value === "gameserver" && gameserverId.value) {
    resourceIdToUse = gameserverId.value;
  } else if (resourceType.value === "environment" && Array.isArray(resourceIds.value) && resourceIds.value.length > 0) {
    // For environment, we need to create bindings for each environment
    // But actually, environment uses ResourceSelector, not ResourceID
    // Let's handle this properly
    for (const envId of resourceIds.value) {
      await adminClient.createRoleBinding({
        organizationId: selectedOrg.value,
        userId: userId.value,
        roleId: roleId.value,
        resourceType: resourceType.value as any,
        resourceId: envId,
      });
    }
    await refreshBindings();
    // Reset form
    userId.value = "";
    roleId.value = "";
    resourceType.value = "";
    resourceIds.value = "";
    deploymentId.value = "";
    vpsId.value = "";
    return;
  } else if (resourceIdsString.value) {
    resourceIdToUse = resourceIdsString.value;
  }
  
  await adminClient.createRoleBinding({
    organizationId: selectedOrg.value,
    userId: userId.value,
    roleId: roleId.value,
    resourceType: resourceType.value as any,
    resourceId: resourceIdToUse,
  });
  await refreshBindings();
  // Reset form
  userId.value = "";
  roleId.value = "";
  resourceType.value = "";
  resourceIds.value = "";
  deploymentId.value = "";
  vpsId.value = "";
  gameserverId.value = "";
}

async function refreshAll() {
  await Promise.all([
    refreshBindings(),
    refreshRoleOptions(),
    refreshMemberOptions(),
    refreshDeploymentOptions(),
    refreshVPSOptions(),
    refreshGameserverOptions(),
  ]);
}
</script>
