<template>
  <RoleManager
    title="Superadmin Roles"
    :show-organization-selector="false"
    :show-description="true"
    name-placeholder="e.g. support"
    description-placeholder="e.g. Support team role"
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
import { SuperadminService, type SuperadminPermissionDefinition } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import RoleManager, { type Role } from "~/components/admin/RoleManager.vue";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const superadminClient = useConnectClient(SuperadminService);

// Load superadmin-only permissions
const { data: permissionsCatalog, error: permissionsError } = await useClientFetch(
  "superadmin-permissions",
  async () => {
    try {
      const res = await superadminClient.listSuperadminPermissions({});
      const perms = (res.permissions || []).map((p: SuperadminPermissionDefinition) => ({
        id: p.id,
        description: p.description ?? "",
        resourceType: p.resourceType ?? "admin",
      }));
      return perms;
    } catch (e: any) {
      console.error("[SuperadminRoles] Failed to load permissions:", e);
      throw e;
    }
  }
);

const { data: roles, refresh: refreshRoles } = await useClientFetch(
  "superadmin-roles",
  async () => {
    const res = await superadminClient.listSuperadminRoles({});
    return (res.roles || []).map((r) => ({
      id: r.id,
      name: r.name,
      description: r.description,
      permissionsJson: r.permissionsJson,
    })) as Role[];
  }
);

async function createRole(data: { name: string; description?: string; permissionsJson: string }) {
  await superadminClient.createSuperadminRole({
    name: data.name,
    description: data.description || "",
    permissionsJson: data.permissionsJson,
  });
}

async function updateRole(data: { id: string; name: string; description?: string; permissionsJson: string }) {
  await superadminClient.updateSuperadminRole({
    id: data.id,
    name: data.name,
    description: data.description || "",
    permissionsJson: data.permissionsJson,
  });
}

async function deleteRole(id: string) {
  await superadminClient.deleteSuperadminRole({ id });
}
</script>

