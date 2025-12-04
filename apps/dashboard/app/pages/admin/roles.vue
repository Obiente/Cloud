<template>
  <OuiStack gap="md">
    <OuiText size="xl" weight="semibold">Roles</OuiText>
    
    <!-- Create/Edit Role Form -->
    <OuiCard>
      <OuiCardHeader>
        <OuiText size="lg" weight="semibold">
          {{ editingRole ? "Edit Role" : "Create Role" }}
        </OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <form @submit.prevent="saveRole">
          <OuiGrid cols="1" colsMd="2" gap="md">
            <OuiSelect
              label="Organization"
              v-model="selectedOrg"
              :items="orgItems"
              :disabled="editingRole !== null"
            />
            <OuiInput
              label="Role Name"
              v-model="name"
              placeholder="e.g. devops"
            />
          </OuiGrid>
          
          <OuiStack gap="md" mt="md">
            <PermissionTree
              v-if="hasPermissions"
              :permissions="permissionsCatalog"
              v-model="selectedPerms"
              :default-expanded="false"
            />
            <OuiFlex v-else-if="!permissionsCatalog" justify="center" py="lg">
              <OuiText color="secondary">Loading permissions...</OuiText>
            </OuiFlex>
            <OuiFlex v-else-if="permissionsError" justify="center" py="lg">
              <OuiText color="danger">Failed to load permissions: {{ permissionsError }}</OuiText>
            </OuiFlex>
            <OuiFlex v-else justify="center" py="lg">
              <OuiText color="secondary">No permissions available</OuiText>
            </OuiFlex>
          </OuiStack>
          
          <OuiFlex class="mt-6" gap="md" align="center">
            <OuiButton type="submit" :loading="saving">
              {{ editingRole ? "Update Role" : "Create Role" }}
            </OuiButton>
            <OuiButton
              v-if="editingRole"
              variant="ghost"
              @click="cancelEdit"
            >
              Cancel
            </OuiButton>
            <OuiText v-if="error" color="danger">{{ error }}</OuiText>
          </OuiFlex>
        </form>
      </OuiCardBody>
    </OuiCard>

    <!-- Existing Roles List -->
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center">
          <OuiText size="lg" weight="semibold">Existing Roles</OuiText>
          <OuiButton variant="ghost" size="sm" @click="refreshRoles">
            Refresh
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="xs">
          <OuiFlex
            v-for="r in roles"
            :key="r.id"
            justify="between"
            align="center"
            p="sm"
            class="border border-border-muted rounded hover:bg-background-muted transition-colors"
          >
            <OuiStack gap="xs" class="flex-1">
              <OuiText size="sm" weight="semibold">
                {{ r.name }}
              </OuiText>
              <OuiText size="xs" color="secondary">
                {{ getPermissionCount(r.permissionsJson) }} permission(s)
              </OuiText>
            </OuiStack>
            <OuiFlex gap="xs">
              <OuiButton
                variant="ghost"
                size="xs"
                @click="editRole(r)"
              >
                Edit
              </OuiButton>
              <OuiButton
                variant="ghost"
                color="danger"
                size="xs"
                @click="removeRole(r.id)"
              >
                Delete
              </OuiButton>
            </OuiFlex>
          </OuiFlex>
          <OuiFlex v-if="!roles || roles.length === 0" justify="center" py="lg">
            <OuiText color="secondary">No roles found</OuiText>
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useOrganizationsStore } from "~/stores/organizations";
import { OrganizationService, AdminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import PermissionTree from "~/components/admin/PermissionTree.vue";

definePageMeta({ layout: "admin", middleware: "auth" });

const orgStore = useOrganizationsStore();
orgStore.hydrate();
const { orgs, currentOrgId } = storeToRefs(orgStore);

const name = ref("");
const selectedPerms = ref<string[]>([]);
const error = ref("");
const editingRole = ref<{ id: string; name: string; permissionsJson: string } | null>(null);
const saving = ref(false);

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

// Computed to ensure reactivity
const hasPermissions = computed(() => {
  const perms = permissionsCatalog.value;
  return perms && Array.isArray(perms) && perms.length > 0;
});

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
    return res.roles || [];
  },
  { watch: [selectedOrg] }
);

watch(
  () => selectedOrg.value,
  () => {
    if (!editingRole.value) {
      name.value = "";
      selectedPerms.value = [];
    }
  },
  { immediate: true }
);

function getPermissionCount(permissionsJson: string): number {
  try {
    const perms = JSON.parse(permissionsJson);
    return Array.isArray(perms) ? perms.length : 0;
  } catch {
    return 0;
  }
}

function editRole(role: { id: string; name: string; permissionsJson: string }) {
  editingRole.value = role;
  name.value = role.name;
  try {
    const perms = JSON.parse(role.permissionsJson);
    selectedPerms.value = Array.isArray(perms) ? perms : [];
  } catch {
    selectedPerms.value = [];
  }
  // Scroll to top of form
  window.scrollTo({ top: 0, behavior: "smooth" });
}

function cancelEdit() {
  editingRole.value = null;
  name.value = "";
  selectedPerms.value = [];
  error.value = "";
}

async function saveRole() {
  error.value = "";
  saving.value = true;
  try {
    if (editingRole.value) {
      // Update existing role
      await adminClient.updateRole({
        id: editingRole.value.id,
        organizationId: selectedOrg.value,
        name: name.value,
        permissionsJson: JSON.stringify(selectedPerms.value),
      });
      cancelEdit();
    } else {
      // Create new role
      await adminClient.createRole({
        organizationId: selectedOrg.value,
        name: name.value,
        permissionsJson: JSON.stringify(selectedPerms.value),
      });
      name.value = "";
      selectedPerms.value = [];
    }
    await refreshRoles();
  } catch (e: any) {
    error.value = e?.message || "Error";
  } finally {
    saving.value = false;
  }
}

async function removeRole(id: string) {
  try {
    await adminClient.deleteRole({ id });
    await refreshRoles();
  } catch (e: any) {
    error.value = e?.message || "Error";
  }
}
</script>
