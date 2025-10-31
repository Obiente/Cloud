<template>
  <OuiStack gap="md">
    <OuiText size="xl" weight="semibold">Roles</OuiText>
    <OuiCard>
      <OuiCardBody>
        <form @submit.prevent="create">
          <OuiGrid cols="1" colsMd="2" gap="md">
            <OuiSelect
              label="Organization"
              v-model="selectedOrg"
              :items="orgItems"
            />
            <OuiInput
              label="Role Name"
              v-model="name"
              placeholder="e.g. devops"
            />
          </OuiGrid>
          <div class="mt-4">
            <OuiText size="md" weight="medium">Permissions</OuiText>
            <OuiGrid cols="1" colsMd="3" gap="sm" class="mt-2">
              <div
                v-for="p in permissionsCatalog"
                :key="p.id"
                class="flex items-start gap-2"
              >
                <OuiCheckbox
                  :label="p.id"
                  :model-value="selectedPerms.includes(p.id)"
                  @update:model-value="(val: boolean) => onTogglePerm(p.id, val)"
                >
                  <template #default>
                    <OuiText class="font-medium">{{ p.id }}</OuiText>
                    <OuiText class="block text-sm text-secondary">{{
                      p.description
                    }}</OuiText>
                  </template>
                </OuiCheckbox>
              </div>
            </OuiGrid>
          </div>
          <OuiFlex class="mt-4" gap="md">
            <OuiButton type="submit">Create</OuiButton>
            <OuiText v-if="error" color="danger">{{ error }}</OuiText>
          </OuiFlex>
        </form>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardHeader>Existing Roles</OuiCardHeader>
      <OuiCardBody>
        <OuiFlex gap="sm" class="mb-2">
          <OuiButton variant="ghost" @click="refreshRoles">Refresh</OuiButton>
        </OuiFlex>
        <table class="w-full text-sm">
          <thead>
            <tr>
              <th class="text-left">Name</th>
              <th class="text-left">Permissions</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="r in roles"
              :key="r.id"
              class="border-t border-border-default"
            >
              <td class="py-2">{{ r.name }}</td>
              <td class="py-2 truncate">{{ r.permissionsJson }}</td>
              <td class="py-2 text-right">
                <OuiButton
                  variant="ghost"
                  color="danger"
                  @click="removeRole(r.id)"
                  >Delete</OuiButton
                >
              </td>
            </tr>
          </tbody>
        </table>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, watch } from "vue";
import { storeToRefs } from "pinia";
import { useOrganizationsStore } from "~/stores/organizations";
import { OrganizationService, AdminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

definePageMeta({ layout: "admin", middleware: "auth" });

const orgStore = useOrganizationsStore();
orgStore.hydrate();
const { orgs, currentOrgId } = storeToRefs(orgStore);

const name = ref("");
const selectedPerms = ref<string[]>([]);
const error = ref("");

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

const { data: permissionsCatalog } = await useAsyncData(
  "admin-permissions",
  async () => {
    const res = await adminClient.listPermissions({});
    return (res.permissions || []).map((p) => ({
      id: p.id,
      description: p.description ?? "",
      resourceType: p.resourceType ?? "admin",
    }));
  }
);

const { data: roles, refresh: refreshRoles } = await useAsyncData(
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
  { watch: [selectedOrg], server: true }
);

watch(
  () => selectedOrg.value,
  () => {
    name.value = "";
    selectedPerms.value = [];
  },
  { immediate: true }
);

function onTogglePerm(id: string, checked: boolean) {
  const set = new Set(selectedPerms.value);
  if (checked) set.add(id);
  else set.delete(id);
  selectedPerms.value = Array.from(set);
}

async function create() {
  error.value = "";
  try {
    await adminClient.createRole({
      organizationId: selectedOrg.value,
      name: name.value,
      permissionsJson: JSON.stringify(selectedPerms.value),
    });
    name.value = "";
    selectedPerms.value = [];
    await refreshRoles();
  } catch (e: any) {
    error.value = e?.message || "Error";
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
