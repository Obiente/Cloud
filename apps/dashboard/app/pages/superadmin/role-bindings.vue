<template>
  <OuiStack gap="md">
    <OuiText size="xl" weight="semibold">Superadmin Role Bindings</OuiText>
    
    <!-- Create Binding Form -->
    <OuiCard>
      <OuiCardHeader>
        <OuiText size="lg" weight="semibold">Assign Role to User</OuiText>
      </OuiCardHeader>
      <OuiCardBody>
        <form @submit.prevent="createBinding">
          <OuiGrid cols="1" colsMd="2" gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">User ID</OuiText>
              <OuiInput
                v-model="userId"
                placeholder="Enter user ID"
              />
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">Role</OuiText>
              <OuiSelect v-model="roleId" :items="roleItems" />
            </OuiStack>
          </OuiGrid>
          <OuiFlex mt="md" gap="md" align="center">
            <OuiButton type="submit" :loading="creating">Assign Role</OuiButton>
            <OuiText v-if="error" color="danger">{{ error }}</OuiText>
          </OuiFlex>
        </form>
      </OuiCardBody>
    </OuiCard>

    <!-- Existing Bindings List -->
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center">
          <OuiText size="lg" weight="semibold">Existing Bindings</OuiText>
          <OuiButton variant="ghost" size="sm" @click="refreshBindings">
            Refresh
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="xs">
          <OuiFlex
            v-for="b in bindingItems"
            :key="b.id"
            justify="between"
            align="center"
            p="sm"
            class="border border-border-muted rounded hover:bg-background-muted transition-colors"
          >
            <OuiStack gap="xs" class="flex-1">
              <OuiText size="sm" weight="semibold">
                User: {{ b.userId }}
              </OuiText>
              <OuiText size="xs" color="secondary">
                Role: {{ b.roleName }}
              </OuiText>
            </OuiStack>
            <OuiButton
              variant="ghost"
              color="danger"
              size="xs"
              @click="removeBinding(b.id)"
            >
              Remove
            </OuiButton>
          </OuiFlex>
          <OuiFlex v-if="!bindingItems || bindingItems.length === 0" justify="center" py="lg">
            <OuiText color="secondary">No bindings found</OuiText>
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const userId = ref("");
const roleId = ref("");
const error = ref("");
const creating = ref(false);

const superadminClient = useConnectClient(SuperadminService);
const { toast } = useToast();

// Load roles for dropdown
const { data: roles, refresh: refreshRoles } = await useClientFetch(
  "superadmin-roles-for-bindings",
  async () => {
    const res = await superadminClient.listSuperadminRoles({});
    return res.roles || [];
  }
);

const roleItems = computed(() =>
  (roles.value || []).map((r) => ({ label: r.name, value: r.id }))
);

const roleLabelMap = computed(() => {
  const map = new Map<string, string>();
  (roles.value || []).forEach((r) => map.set(r.id, r.name));
  return map;
});

// Load bindings
const { data: bindings, refresh: refreshBindings } = await useClientFetch(
  "superadmin-role-bindings",
  async () => {
    const res = await superadminClient.listSuperadminRoleBindings({});
    return res.bindings || [];
  }
);

const bindingItems = computed(() =>
  (bindings.value || []).map((b) => ({
    id: b.id,
    userId: b.userId,
    roleId: b.roleId,
    roleName: roleLabelMap.value.get(b.roleId) || b.roleId,
  }))
);

async function createBinding() {
  if (!userId.value || !roleId.value) {
    error.value = "User ID and Role are required";
    return;
  }
  error.value = "";
  creating.value = true;
  try {
    await superadminClient.createSuperadminRoleBinding({
      userId: userId.value,
      roleId: roleId.value,
    });
    toast.success("Role binding created successfully");
    userId.value = "";
    roleId.value = "";
    await refreshBindings();
  } catch (e: any) {
    error.value = e?.message || "Error";
    toast.error(error.value);
  } finally {
    creating.value = false;
  }
}

async function removeBinding(id: string) {
  if (!confirm("Are you sure you want to remove this role binding?")) {
    return;
  }
  try {
    await superadminClient.deleteSuperadminRoleBinding({ id });
    toast.success("Role binding removed successfully");
    await refreshBindings();
  } catch (e: any) {
    error.value = e?.message || "Error";
    toast.error(error.value);
  }
}
</script>

