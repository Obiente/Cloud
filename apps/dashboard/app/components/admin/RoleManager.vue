<template>
  <OuiStack gap="md">
    <OuiText size="xl" weight="semibold">{{ title }}</OuiText>
    
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
              v-if="showOrganizationSelector"
              label="Organization"
              v-model="selectedOrg"
              :items="orgItems"
              :disabled="editingRole !== null"
            />
            <OuiInput
              label="Role Name"
              v-model="name"
              :placeholder="namePlaceholder"
            />
            <OuiInput
              v-if="showDescription"
              label="Description"
              v-model="description"
              :placeholder="descriptionPlaceholder"
            />
          </OuiGrid>
          
          <OuiStack gap="md" mt="md">
            <PermissionTree
              v-if="hasPermissions && permissionsCatalog"
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
          <OuiButton variant="ghost" size="sm" @click="() => props.onRefreshRoles()">
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
            align="start"
            p="sm"
            class="border border-border-muted rounded hover:bg-background-muted transition-colors"
          >
            <OuiStack gap="xs" class="flex-1">
              <OuiText size="sm" weight="semibold">
                {{ r.name }}
              </OuiText>
              <OuiText v-if="showDescription && r.description" size="xs" color="secondary">
                {{ r.description }}
              </OuiText>
              <OuiFlex gap="xs" wrap="wrap" align="center">
                <OuiBadge
                  v-for="perm in getPermissionBadges(r.permissionsJson)"
                  :key="perm"
                  variant="secondary"
                  size="xs"
                  tone="soft"
                >
                  {{ perm }}
                </OuiBadge>
                <OuiText
                  v-if="getPermissionCount(r.permissionsJson) > getPermissionBadges(r.permissionsJson).length"
                  size="xs"
                  color="secondary"
                >
                  +{{ getPermissionCount(r.permissionsJson) - getPermissionBadges(r.permissionsJson).length }} more
                </OuiText>
                <OuiText
                  v-if="getPermissionCount(r.permissionsJson) === 0"
                  size="xs"
                  color="secondary"
                >
                  No permissions
                </OuiText>
              </OuiFlex>
            </OuiStack>
            <OuiFlex gap="xs" class="flex-shrink-0">
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
import PermissionTree from "~/components/admin/PermissionTree.vue";
import { useToast } from "~/composables/useToast";

export interface Role {
  id: string;
  name: string;
  description?: string;
  permissionsJson: string;
}

export interface RoleManagerProps {
  title?: string;
  showOrganizationSelector?: boolean;
  showDescription?: boolean;
  namePlaceholder?: string;
  descriptionPlaceholder?: string;
  permissionsCatalog: Array<{ id: string; description: string; resourceType: string }> | null;
  permissionsError: Error | null | undefined;
  roles: Role[];
  onRefreshRoles: () => Promise<void>;
  onCreateRole: (data: { name: string; description?: string; permissionsJson: string; organizationId?: string }) => Promise<void>;
  onUpdateRole: (data: { id: string; name: string; description?: string; permissionsJson: string; organizationId?: string }) => Promise<void>;
  onDeleteRole: (id: string) => Promise<void>;
  onSuccess?: (message: string) => void;
  onError?: (message: string) => void;
}

const props = withDefaults(defineProps<RoleManagerProps>(), {
  title: "Roles",
  showOrganizationSelector: false,
  showDescription: false,
  namePlaceholder: "e.g. devops",
  descriptionPlaceholder: "e.g. DevOps team role",
});

const { toast } = useToast();

// Organization management (only if showOrganizationSelector is true)
const orgStore = props.showOrganizationSelector ? useOrganizationsStore() : null;
if (orgStore) {
  orgStore.hydrate();
}
const { orgs, currentOrgId } = props.showOrganizationSelector && orgStore ? storeToRefs(orgStore) : { orgs: ref([]), currentOrgId: ref("") };

const orgItems = computed(() =>
  (orgs.value || []).map((o) => ({
    label: o.name ?? o.id,
    value: o.id,
  }))
);

const selectedOrg = computed({
  get: () => currentOrgId.value || "",
  set: (id: string) => {
    if (id && orgStore) orgStore.switchOrganization(id);
  },
});

// Form state
const name = ref("");
const description = ref("");
const selectedPerms = ref<string[]>([]);
const error = ref("");
const editingRole = ref<Role | null>(null);
const saving = ref(false);

// Computed to ensure reactivity
const hasPermissions = computed(() => {
  const perms = props.permissionsCatalog;
  return perms && Array.isArray(perms) && perms.length > 0;
});

// Watch for organization changes (only if organization selector is shown)
if (props.showOrganizationSelector) {
  watch(
    () => selectedOrg.value,
    () => {
      if (!editingRole.value) {
        name.value = "";
        description.value = "";
        selectedPerms.value = [];
      }
    },
    { immediate: true }
  );
}

function getPermissionCount(permissionsJson: string): number {
  try {
    const perms = JSON.parse(permissionsJson);
    return Array.isArray(perms) ? perms.length : 0;
  } catch {
    return 0;
  }
}

function getPermissionBadges(permissionsJson: string, maxDisplay: number = 8): string[] {
  try {
    const perms = JSON.parse(permissionsJson);
    if (!Array.isArray(perms)) return [];
    
    // Format permissions for display
    const formatted = perms.slice(0, maxDisplay).map((perm: string) => {
      // Try to find the permission in the catalog for a better description
      const catalogPerm = props.permissionsCatalog?.find((p) => p.id === perm);
      if (catalogPerm?.description) {
        // Use the description if available, but shorten it if too long
        const desc = catalogPerm.description;
        if (desc.length > 30) {
          return desc.substring(0, 27) + "...";
        }
        return desc;
      }
      
      // Fallback: format the permission ID
      // e.g., "deployment.read" -> "Deployment: Read"
      // e.g., "organization.members.invite" -> "Members: Invite"
      // e.g., "admin.roles.create" -> "Admin Roles: Create"
      const parts = perm.split(".");
      if (parts.length >= 2) {
        const resource = parts[0];
        const action = parts[parts.length - 1];
        const middle = parts.slice(1, -1);
        
        if (!resource || !action) {
          return perm;
        }
        
        // Capitalize and format
        const resourceFormatted = resource.charAt(0).toUpperCase() + resource.slice(1);
        const actionFormatted = action.charAt(0).toUpperCase() + action.slice(1);
        
        if (middle.length > 0) {
          const middleFormatted = middle.map((m: string) => 
            m.charAt(0).toUpperCase() + m.slice(1)
          ).join(" ");
          return `${middleFormatted} ${actionFormatted}`;
        }
        return `${resourceFormatted}: ${actionFormatted}`;
      }
      return perm;
    });
    
    return formatted;
  } catch {
    return [];
  }
}

function editRole(role: Role) {
  editingRole.value = role;
  name.value = role.name;
  description.value = role.description || "";
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
  description.value = "";
  selectedPerms.value = [];
  error.value = "";
}

async function saveRole() {
  error.value = "";
  saving.value = true;
  try {
    const roleData: any = {
      name: name.value,
      permissionsJson: JSON.stringify(selectedPerms.value),
    };
    
    if (props.showDescription) {
      roleData.description = description.value;
    }
    
    if (props.showOrganizationSelector && selectedOrg.value) {
      roleData.organizationId = selectedOrg.value;
    }
    
    if (editingRole.value) {
      // Update existing role
      roleData.id = editingRole.value.id;
      await props.onUpdateRole(roleData);
      const message = "Role updated successfully";
      if (props.onSuccess) {
        props.onSuccess(message);
      } else {
        toast.success(message);
      }
      cancelEdit();
    } else {
      // Create new role
      await props.onCreateRole(roleData);
      const message = "Role created successfully";
      if (props.onSuccess) {
        props.onSuccess(message);
      } else {
        toast.success(message);
      }
      name.value = "";
      description.value = "";
      selectedPerms.value = [];
    }
    await props.onRefreshRoles();
  } catch (e: any) {
    error.value = e?.message || "Error";
    if (props.onError) {
      props.onError(error.value);
    } else {
      toast.error(error.value);
    }
  } finally {
    saving.value = false;
  }
}

async function removeRole(id: string) {
  if (!confirm("Are you sure you want to delete this role?")) {
    return;
  }
  try {
    await props.onDeleteRole(id);
    const message = "Role deleted successfully";
    if (props.onSuccess) {
      props.onSuccess(message);
    } else {
      toast.success(message);
    }
    await props.onRefreshRoles();
  } catch (e: any) {
    const errorMsg = e?.message || "Error";
    if (props.onError) {
      props.onError(errorMsg);
    } else {
      toast.error(errorMsg);
    }
  }
}
</script>

