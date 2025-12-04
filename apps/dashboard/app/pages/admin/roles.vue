<template>
  <RoleManager
    title="Roles"
    :show-organization-selector="true"
    :permissions-catalog="permissionsCatalog"
    :permissions-error="permissionsError || null"
    :roles="roles"
    :on-refresh-roles="refreshRoles"
    :on-create-role="createRole"
    :on-update-role="updateRole"
    :on-delete-role="deleteRole"
  />
</template>

<script setup lang="ts">
import { computed } from "vue";
import { AdminService, OrganizationService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import RoleManager, { type Role } from "~/components/admin/RoleManager.vue";
import { storeToRefs } from "pinia";
import { useOrganizationsStore } from "~/stores/organizations";

definePageMeta({ layout: "admin", middleware: "auth" });

const orgStore = useOrganizationsStore();
orgStore.hydrate();
const { orgs, currentOrgId } = storeToRefs(orgStore);

const orgClient = useConnectClient(OrganizationService);
const adminClient = useConnectClient(AdminService);

if (!orgs.value.length) {
  try {
    const res = await orgClient.listOrganizations({});
    orgStore.setOrganizations(res.organizations || []);
  } catch (e) {
    console.error("Failed to load organizations", e);
  }
}

const selectedOrg = computed({
  get: () => currentOrgId.value || "",
  set: (id: string) => {
    if (id) orgStore.switchOrganization(id);
  },
});

const { data: permissionsCatalog, error: permissionsError } = await useClientFetch(
  "admin-permissions",
  async () => {
    try {
      const res = await adminClient.listPermissions({});
      const perms = (res.permissions || []).map((p) => ({
        id: p.id,
        description: p.description ?? "",
        resourceType: p.resourceType ?? "admin",
      }));
      return perms;
    } catch (e: any) {
      console.error("[Roles] Failed to load permissions:", e);
      throw e;
    }
  }
);

const { data: roles, refresh: refreshRoles } = await useClientFetch(
  () =>
    selectedOrg.value
      ? `admin-roles-${selectedOrg.value}`
      : "admin-roles-none",
  async () => {
    if (!selectedOrg.value) return [];
    const res = await adminClient.listRoles({
      organizationId: selectedOrg.value,
    });
    return (res.roles || []).map((r) => ({
      id: r.id,
      name: r.name,
      permissionsJson: r.permissionsJson,
    })) as Role[];
  },
  { watch: [selectedOrg] }
);

async function createRole(data: { name: string; permissionsJson: string; organizationId?: string }) {
  if (!data.organizationId) throw new Error("Organization ID is required");
  await adminClient.createRole({
    organizationId: data.organizationId,
    name: data.name,
    permissionsJson: data.permissionsJson,
  });
}

async function updateRole(data: { id: string; name: string; permissionsJson: string; organizationId?: string }) {
  if (!data.organizationId) throw new Error("Organization ID is required");
  await adminClient.updateRole({
    id: data.id,
    organizationId: data.organizationId,
    name: data.name,
    permissionsJson: data.permissionsJson,
  });
}

async function deleteRole(id: string) {
  await adminClient.deleteRole({ id });
}
</script>
