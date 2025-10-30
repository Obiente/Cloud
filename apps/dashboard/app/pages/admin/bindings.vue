<template>
  <OuiStack gap="md">
    <OuiText size="xl" weight="semibold">Role Bindings</OuiText>
    <OuiCard>
      <OuiCardBody>
        <form @submit.prevent="create">
          <OuiGrid cols="1" colsMd="3" gap="md">
            <div>
              <OuiText size="sm" weight="medium">Organization</OuiText>
              <OuiSelect v-model="selectedOrg" :items="orgItems" />
            </div>
            <div>
              <OuiText size="sm" weight="medium">Member</OuiText>
              <OuiSelect v-model="userId" :items="memberItems" />
            </div>
            <div>
              <OuiText size="sm" weight="medium">Role</OuiText>
              <OuiSelect v-model="roleId" :items="roleItems" />
            </div>
            <div>
              <OuiText size="sm" weight="medium">
                Resource Type (optional)
              </OuiText>
              <OuiSelect v-model="resourceType" :items="resourceTypeItems" />
            </div>
            <div>
              <OuiText size="sm" weight="medium">Resource (optional)</OuiText>
              <OuiSelect
                v-if="resourceType === 'deployment'"
                multiple
                v-model="resourceIds"
                :items="deploymentItems"
              />
              <OuiSelect
                v-else-if="resourceType === 'environment'"
                multiple
                v-model="resourceIds"
                :items="environmentItems"
              />
              <OuiInput
                v-else
                v-model="(resourceIds as any)"
                placeholder="* for all"
              />
            </div>
          </OuiGrid>
          <OuiFlex class="mt-4" gap="md">
            <OuiButton type="submit">Bind</OuiButton>
          </OuiFlex>
        </form>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>Bindings</OuiCardHeader>
      <OuiCardBody>
        <OuiFlex gap="sm">
          <OuiButton variant="ghost" @click="refreshAll">Refresh</OuiButton>
        </OuiFlex>
        <ul class="list-disc pl-6 mt-2">
          <li v-for="b in bindingItems" :key="b.id">
            {{ b.user }} → {{ b.role }} ({{ b.resourceType }} {{ b.resource }})
          </li>
        </ul>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { storeToRefs } from "pinia";
import { useOrganizationsStore } from "~/stores/organizations";
import { OrganizationService, DeploymentService, AdminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

definePageMeta({ layout: "admin", middleware: "auth" });

const orgStore = useOrganizationsStore();
orgStore.hydrate();
const { orgs, currentOrgId } = storeToRefs(orgStore);

const userId = ref("");
const roleId = ref("");
const resourceType = ref("");
const resourceIds = ref<string[] | string>("");

const resourceTypeItems = [
  { label: "Deployment", value: "deployment" },
  { label: "Environment", value: "environment" },
  { label: "Admin", value: "admin" },
];
const environmentItems = [
  { label: "Production", value: "production" },
  { label: "Staging", value: "staging" },
  { label: "Development", value: "development" },
];

const orgClient = useConnectClient(OrganizationService);
const depClient = useConnectClient(DeploymentService);
const adminClient = useConnectClient(AdminService);

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
const selectedOrg = computed({
  get: () => currentOrgId.value || "",
  set: (id: string) => {
    if (id) orgStore.switchOrganization(id);
  },
});

const { data: bindingsData, refresh: refreshBindings } = await useAsyncData(
  () =>
    selectedOrg.value
      ? `admin-bindings-${selectedOrg.value}`
      : "admin-bindings-none",
  async () => {
    if (!selectedOrg.value) return [];
    const res = await adminClient.listRoleBindings({
      organizationId: selectedOrg.value,
    });
    return res.bindings || [];
  },
  { watch: [selectedOrg], server: true }
);
const bindings = computed(() => bindingsData.value || []);

const { data: roleOptionsData, refresh: refreshRoleOptions } = await useAsyncData(
  () =>
    selectedOrg.value
      ? `admin-binding-roles-${selectedOrg.value}`
      : "admin-binding-roles-none",
  async () => {
    if (!selectedOrg.value) return [];
    const res = await adminClient.listRoles({
      organizationId: selectedOrg.value,
    });
    const roles = res.roles || [];
    if (!roleId.value && roles.length) {
      roleId.value = roles[0]?.id ?? "";
    }
    return roles.map((r) => ({ id: r.id, name: r.name }));
  },
  { watch: [selectedOrg], server: true }
);
const roleLabelMap = computed(() => {
  const map = new Map<string, string>();
  (roleOptionsData.value || []).forEach((r) => map.set(r.id, r.name));
  return map;
});
const roleItems = computed(() =>
  (roleOptionsData.value || []).map((r) => ({ label: r.name, value: r.id }))
);

const { data: memberOptionsData, refresh: refreshMemberOptions } = await useAsyncData(
  () =>
    selectedOrg.value
      ? `admin-binding-members-${selectedOrg.value}`
      : "admin-binding-members-none",
  async () => {
    if (!selectedOrg.value) return [];
    const res = await orgClient.listMembers({
      organizationId: selectedOrg.value,
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
  { watch: [selectedOrg], server: true }
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

const { data: deploymentOptionsData, refresh: refreshDeploymentOptions } =
  await useAsyncData(
    () =>
      selectedOrg.value
        ? `admin-binding-deployments-${selectedOrg.value}`
        : "admin-binding-deployments-none",
    async () => {
      if (!selectedOrg.value) return [];
      const res = await depClient.listDeployments({
        organizationId: selectedOrg.value,
      });
      return (res.deployments || []).map((d) => ({
        id: d.id,
        name: d.name || d.id,
      }));
    },
    { watch: [selectedOrg], server: true }
  );
const deploymentItems = computed(() =>
  (deploymentOptionsData.value || []).map((d) => ({
    label: d.name,
    value: d.id,
  }))
);

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
    await refreshAll();
  },
  { immediate: true }
);

watch(resourceType, () => {
  resourceIds.value = Array.isArray(resourceIds.value) ? [] : "";
});

async function create() {
  if (!selectedOrg.value || !userId.value || !roleId.value) return;
  const ids = Array.isArray(resourceIds.value)
    ? resourceIds.value
    : [resourceIds.value || ""];
  for (const rid of ids) {
    await adminClient.createRoleBinding({
      organizationId: selectedOrg.value,
      userId: userId.value,
      roleId: roleId.value,
      resourceType: resourceType.value as any,
      resourceId: rid,
    });
  }
  await refreshBindings();
}

async function refreshAll() {
  await Promise.all([
    refreshBindings(),
    refreshRoleOptions(),
    refreshMemberOptions(),
    refreshDeploymentOptions(),
  ]);
}
</script>
