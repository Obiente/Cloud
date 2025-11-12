<template>
  <OuiStack gap="lg">
    <!-- Users List -->
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex justify="between" align="center">
          <OuiStack gap="xs">
            <OuiText as="h2" class="oui-card-title">Users</OuiText>
            <OuiText color="secondary" size="sm">
              Manage users on this VPS instance. Users are configured via cloud-init and will be applied on the next reboot.
            </OuiText>
          </OuiStack>
          <OuiButton
            variant="solid"
            size="sm"
            @click="openCreateUserDialog"
            :disabled="!vps?.instanceId"
            class="gap-2"
          >
            <PlusIcon class="h-4 w-4" />
            Add User
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <div v-if="loading" class="py-8 text-center">
          <OuiSpinner size="lg" />
          <OuiText color="secondary" class="mt-4">Loading users...</OuiText>
        </div>
        <div v-else-if="error" class="py-8 text-center">
          <OuiText color="danger">{{ error }}</OuiText>
          <OuiButton variant="outline" @click="loadUsers" class="mt-4 gap-2">
            <ArrowPathIcon class="h-4 w-4" />
            Retry
          </OuiButton>
        </div>
        <div v-else-if="users.length === 0" class="py-8 text-center">
          <UserIcon class="h-12 w-12 text-secondary mx-auto mb-4" />
          <OuiText color="secondary" class="mb-4">No custom users configured</OuiText>
          <OuiText size="sm" color="muted">
            The root user is always available. Add custom users to grant access to specific accounts.
          </OuiText>
        </div>
        <OuiStack v-else gap="md">
          <OuiBox
            v-for="user in users.filter(u => u.name !== 'root')"
            :key="user.name"
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="md">
              <OuiFlex justify="between" align="start">
                <OuiStack gap="xs">
                  <OuiFlex align="center" gap="sm">
                    <UserIcon class="h-5 w-5 text-secondary" />
                    <OuiText size="sm" weight="semibold">{{ user.name }}</OuiText>
                    <OuiBadge v-if="user.sudo" variant="primary" size="xs">Sudo</OuiBadge>
                    <OuiBadge v-if="user.sudoNopasswd" variant="success" size="xs">Sudo (NOPASSWD)</OuiBadge>
                  </OuiFlex>
                  <OuiText v-if="user.groups && user.groups.length > 0" size="xs" color="secondary">
                    Groups: {{ user.groups.join(", ") }}
                  </OuiText>
                  <OuiText v-if="user.shell" size="xs" color="secondary">
                    Shell: {{ user.shell }}
                  </OuiText>
                </OuiStack>
                <OuiButton
                  variant="ghost"
                  color="danger"
                  size="xs"
                  @click="deleteUser(user.name)"
                  :disabled="deletingUser === user.name"
                  class="gap-1"
                >
                  <TrashIcon class="h-4 w-4" />
                  Delete
                </OuiButton>
              </OuiFlex>

              <!-- SSH Keys -->
              <OuiStack gap="xs" v-if="user.sshAuthorizedKeys && user.sshAuthorizedKeys.length > 0">
                <OuiText size="xs" weight="medium" color="secondary">SSH Keys</OuiText>
                <OuiStack gap="xs">
                  <OuiBox
                    v-for="(key, idx) in user.sshAuthorizedKeys"
                    :key="idx"
                    p="sm"
                    rounded="md"
                    class="bg-surface-muted font-mono text-xs"
                  >
                    <OuiFlex justify="between" align="center" gap="sm">
                      <OuiText class="truncate">{{ key.substring(0, 60) }}...</OuiText>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click="removeSSHKeyFromUser(user.name, idx)"
                        class="gap-1"
                      >
                        <XMarkIcon class="h-3 w-3" />
                      </OuiButton>
                    </OuiFlex>
                  </OuiBox>
                </OuiStack>
              </OuiStack>
              <OuiText v-else size="xs" color="muted">No SSH keys configured</OuiText>

              <!-- Actions -->
              <OuiFlex gap="sm">
                <OuiButton
                  variant="outline"
                  size="xs"
                  @click="openManageSSHKeysDialog(user)"
                  class="gap-1"
                >
                  <KeyIcon class="h-4 w-4" />
                  Manage SSH Keys
                </OuiButton>
                <OuiButton
                  v-if="user.name !== 'root'"
                  variant="outline"
                  size="xs"
                  @click="openResetPasswordDialog(user)"
                  class="gap-1"
                >
                  <KeyIcon class="h-4 w-4" />
                  {{ user.hasPassword ? "Reset Password" : "Set Password" }}
                </OuiButton>
              </OuiFlex>
            </OuiStack>
          </OuiBox>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Root User Card -->
    <OuiCard variant="outline">
      <OuiCardHeader>
        <OuiStack gap="xs">
          <OuiFlex align="center" gap="sm">
            <UserIcon class="h-5 w-5 text-secondary" />
            <OuiText as="h2" class="oui-card-title">Root User</OuiText>
            <OuiBadge variant="primary" size="xs">System</OuiBadge>
          </OuiFlex>
          <OuiText color="secondary" size="sm">
            The root user is always available and has full system access. SSH keys can be managed in SSH Settings.
          </OuiText>
        </OuiStack>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiFlex justify="between" align="center">
            <OuiText size="sm" color="secondary">Password</OuiText>
            <OuiButton
              variant="outline"
              size="xs"
              @click="openResetRootPasswordDialog"
              class="gap-1"
            >
              <KeyIcon class="h-4 w-4" />
              Reset Password
            </OuiButton>
          </OuiFlex>
          <OuiText size="xs" color="muted">
            Root password management is available in the SSH Settings tab.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Create User Dialog -->
    <OuiDialog
      v-model:open="createUserDialogOpen"
      title="Create User"
      description="Add a new user to this VPS instance. The user will be created on the next reboot via cloud-init."
      size="md"
    >
      <OuiStack gap="md">
        <OuiInput
          v-model="newUser.name"
          label="Username"
          placeholder="username"
          required
          :error="userFormErrors.name"
        />
        <OuiInput
          v-model="newUser.password"
          type="password"
          label="Password"
          placeholder="Leave empty for SSH key only"
          :error="userFormErrors.password"
        />
        <OuiCheckbox
          v-model="newUser.sudo"
          label="Grant sudo access"
        />
        <OuiCheckbox
          v-model="newUser.sudoNopasswd"
          label="Sudo without password"
          :disabled="!newUser.sudo"
        />
        <OuiInput
          v-model="newUser.shell"
          label="Shell (optional)"
          placeholder="/bin/bash"
        />
        <OuiInput
          v-model="newUser.groups"
          label="Groups (comma-separated, optional)"
          placeholder="docker, www-data"
        />
        <OuiStack gap="xs">
          <OuiText size="sm" weight="medium">SSH Keys</OuiText>
          <OuiText size="xs" color="secondary">
            Select SSH keys from your organization to assign to this user
          </OuiText>
          <OuiBox
            v-if="availableSSHKeys.length === 0"
            p="sm"
            rounded="md"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiText size="xs" color="secondary">
              No SSH keys available. Add SSH keys in SSH Settings.
            </OuiText>
          </OuiBox>
          <OuiStack
            v-else
            gap="xs"
            class="max-h-48 overflow-y-auto"
          >
            <OuiCheckbox
              v-for="key in availableSSHKeys"
              :key="key.id"
              :model-value="newUserSSHKeys.includes(key.id)"
              @update:model-value="(checked) => {
                if (checked) {
                  if (!newUserSSHKeys.includes(key.id)) {
                    newUserSSHKeys.push(key.id);
                  }
                } else {
                  const index = newUserSSHKeys.indexOf(key.id);
                  if (index > -1) {
                    newUserSSHKeys.splice(index, 1);
                  }
                }
              }"
            >
              <OuiFlex align="center" gap="xs">
                <KeyIcon class="h-4 w-4 text-secondary" />
                <OuiText size="sm">{{ key.name }}</OuiText>
                <OuiBadge
                  v-if="!key.vpsId"
                  variant="primary"
                  size="xs"
                >
                  Org-wide
                </OuiBadge>
              </OuiFlex>
            </OuiCheckbox>
          </OuiStack>
        </OuiStack>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="closeCreateUserDialog">Cancel</OuiButton>
          <OuiButton
            variant="solid"
            @click="createUser"
            :disabled="!newUser.name.trim() || creatingUser"
          >
            {{ creatingUser ? "Creating..." : "Create User" }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Manage SSH Keys Dialog -->
    <OuiDialog
      v-model:open="manageSSHKeysDialogOpen"
      :title="`Manage SSH Keys for ${selectedUser?.name || 'User'}`"
      size="md"
    >
      <OuiStack gap="md" v-if="selectedUser">
        <OuiText color="secondary" size="sm">
          Select SSH keys from your organization to assign to this user.
        </OuiText>
        <OuiStack gap="xs" v-if="availableSSHKeys.length === 0">
          <OuiText size="sm" color="muted">No SSH keys available. Add SSH keys in SSH Settings.</OuiText>
        </OuiStack>
        <OuiStack gap="xs" v-else class="max-h-64 overflow-y-auto">
          <OuiCheckbox
            v-for="key in availableSSHKeys"
            :key="key.id"
            :model-value="selectedUserSSHKeys.includes(key.id)"
            @update:model-value="(checked) => toggleUserSSHKey(key.id, checked)"
          >
            <OuiFlex align="center" gap="xs">
              <KeyIcon class="h-4 w-4 text-secondary" />
              <OuiText size="sm">{{ key.name }}</OuiText>
              <OuiBadge v-if="!key.vpsId" variant="primary" size="xs">Org-wide</OuiBadge>
            </OuiFlex>
          </OuiCheckbox>
        </OuiStack>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="manageSSHKeysDialogOpen = false">Cancel</OuiButton>
          <OuiButton variant="solid" @click="saveUserSSHKeys" :disabled="savingSSHKeys">
            {{ savingSSHKeys ? "Saving..." : "Save" }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Reset Password Dialog -->
    <OuiDialog
      v-model:open="resetPasswordDialogOpen"
      :title="`${selectedUser?.name === 'root' ? 'Reset' : selectedUser?.hasPassword ? 'Reset' : 'Set'} Password for ${selectedUser?.name || 'User'}`"
      size="md"
    >
      <OuiStack gap="md" v-if="selectedUser">
        <OuiInput
          v-model="newPassword"
          type="password"
          label="New Password"
          placeholder="Enter new password"
          required
        />
        <OuiText size="xs" color="secondary">
          The password will be applied on the next reboot via cloud-init.
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="resetPasswordDialogOpen = false">Cancel</OuiButton>
          <OuiButton
            variant="solid"
            @click="saveUserPassword"
            :disabled="!newPassword.trim() || savingPassword"
          >
            {{ savingPassword ? "Saving..." : "Save Password" }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import {
  PlusIcon,
  UserIcon,
  KeyIcon,
  TrashIcon,
  XMarkIcon,
  ArrowPathIcon,
} from "@heroicons/vue/24/outline";
import { VPSService, VPSConfigService, type VPSInstance, type VPSUser } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import { useDialog } from "~/composables/useDialog";
import OuiSpinner from "~/components/oui/Spinner.vue";

interface Props {
  vpsId: string;
  organizationId: string;
  vps: VPSInstance | null | undefined;
}

const props = defineProps<Props>();
const { toast } = useToast();
const { showConfirm } = useDialog();
const vpsClient = useConnectClient(VPSService);
const configClient = useConnectClient(VPSConfigService);

const loading = ref(false);
const error = ref<string | null>(null);
const users = ref<VPSUser[]>([]);
const availableSSHKeys = ref<Array<{
  id: string;
  name: string;
  publicKey: string;
  fingerprint: string;
  vpsId?: string;
}>>([]);

// Dialogs
const createUserDialogOpen = ref(false);
const manageSSHKeysDialogOpen = ref(false);
const resetPasswordDialogOpen = ref(false);
const selectedUser = ref<VPSUser | null>(null);
const selectedUserSSHKeys = ref<string[]>([]);
const newPassword = ref("");

// Form state
const newUser = ref({
  name: "",
  password: "",
  sudo: false,
  sudoNopasswd: false,
  shell: "",
  groups: "",
});
const newUserSSHKeys = ref<string[]>([]);
const userFormErrors = ref<Record<string, string>>({});
const creatingUser = ref(false);
const deletingUser = ref<string | null>(null);
const savingSSHKeys = ref(false);
const savingPassword = ref(false);

const loadUsers = async () => {
  loading.value = true;
  error.value = null;
  try {
    const res = await configClient.listVPSUsers({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
    });
    users.value = res.users || [];
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : "Failed to load users";
  } finally {
    loading.value = false;
  }
};

const loadSSHKeys = async () => {
  try {
    const res = await vpsClient.listSSHKeys({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
    });
    availableSSHKeys.value = (res.keys || []).map((key) => ({
      id: key.id || "",
      name: key.name || "",
      publicKey: key.publicKey || "",
      fingerprint: key.fingerprint || "",
      vpsId: key.vpsId,
    }));
  } catch (err: unknown) {
    console.error("Failed to load SSH keys:", err);
  }
};

const openCreateUserDialog = () => {
  newUser.value = {
    name: "",
    password: "",
    sudo: false,
    sudoNopasswd: false,
    shell: "",
    groups: "",
  };
  newUserSSHKeys.value = [];
  userFormErrors.value = {};
  createUserDialogOpen.value = true;
};

const closeCreateUserDialog = () => {
  createUserDialogOpen.value = false;
  newUser.value = {
    name: "",
    password: "",
    sudo: false,
    sudoNopasswd: false,
    shell: "",
    groups: "",
  };
  newUserSSHKeys.value = [];
  userFormErrors.value = {};
};

const createUser = async () => {
  creatingUser.value = true;
  try {
    // Validate
    if (!newUser.value.name.trim()) {
      userFormErrors.value.name = "Username is required";
      creatingUser.value = false;
      return;
    }

    const groups = newUser.value.groups
      .split(",")
      .map((g) => g.trim())
      .filter((g) => g.length > 0);

    const res = await configClient.createVPSUser({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      name: newUser.value.name.trim(),
      password: newUser.value.password.trim() || undefined,
      sshAuthorizedKeys: newUserSSHKeys.value.length > 0
        ? availableSSHKeys.value
            .filter((k) => newUserSSHKeys.value.includes(k.id))
            .map((k) => k.publicKey)
        : [],
      sudo: newUser.value.sudo || undefined,
      sudoNopasswd: newUser.value.sudoNopasswd || undefined,
      groups: groups.length > 0 ? groups : undefined,
      shell: newUser.value.shell.trim() || undefined,
    });

    toast.success("User created", res.message || "The user will be created on the next reboot.");
    closeCreateUserDialog();
    await loadUsers();
  } catch (err: unknown) {
    toast.error("Failed to create user", err instanceof Error ? err.message : "Unknown error");
  } finally {
    creatingUser.value = false;
  }
};

const deleteUser = async (username: string) => {
  if (username === "root") {
    toast.error("Cannot delete root user");
    return;
  }

  const confirmed = await showConfirm({
    title: "Delete User",
    message: `Are you sure you want to delete user "${username}"? This action cannot be undone.`,
    variant: "danger",
  });

  if (!confirmed) return;

  deletingUser.value = username;
  try {
    const res = await configClient.deleteVPSUser({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      name: username,
    });
    toast.success("User deleted", res.message || "The user will be removed on the next reboot.");
    await loadUsers();
  } catch (err: unknown) {
    toast.error("Failed to delete user", err instanceof Error ? err.message : "Unknown error");
  } finally {
    deletingUser.value = null;
  }
};

const openManageSSHKeysDialog = (user: VPSUser) => {
  selectedUser.value = user;
  // Map current SSH keys to their IDs
  selectedUserSSHKeys.value = user.sshAuthorizedKeys
    .map((pubKey) => {
      const key = availableSSHKeys.value.find((k) => k.publicKey === pubKey);
      return key?.id;
    })
    .filter((id): id is string => !!id);
  manageSSHKeysDialogOpen.value = true;
};

const toggleUserSSHKey = (keyId: string, checked: boolean) => {
  if (checked) {
    if (!selectedUserSSHKeys.value.includes(keyId)) {
      selectedUserSSHKeys.value.push(keyId);
    }
  } else {
    const index = selectedUserSSHKeys.value.indexOf(keyId);
    if (index > -1) {
      selectedUserSSHKeys.value.splice(index, 1);
    }
  }
};

const saveUserSSHKeys = async () => {
  if (!selectedUser.value) return;

  savingSSHKeys.value = true;
  try {
    const res = await configClient.updateUserSSHKeys({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      userName: selectedUser.value.name,
      sshKeyIds: selectedUserSSHKeys.value,
    });
    toast.success("SSH keys updated", res.message || "Changes will be applied on the next reboot.");
    manageSSHKeysDialogOpen.value = false;
    await loadUsers();
  } catch (err: unknown) {
    toast.error("Failed to update SSH keys", err instanceof Error ? err.message : "Unknown error");
  } finally {
    savingSSHKeys.value = false;
  }
};

const openResetPasswordDialog = (user: VPSUser) => {
  selectedUser.value = user;
  newPassword.value = "";
  resetPasswordDialogOpen.value = true;
};

const openResetRootPasswordDialog = () => {
  // Redirect to SSH Settings tab for root password reset
  toast.info("Root password reset", "Please use the SSH Settings tab to reset the root password.");
};

const saveUserPassword = async () => {
  if (!selectedUser.value || !newPassword.value.trim()) return;

  savingPassword.value = true;
  try {
    const res = await configClient.setUserPassword({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      userName: selectedUser.value.name,
      password: newPassword.value.trim(),
    });
    toast.success("Password updated", res.message || "The password will be applied on the next reboot.");
    resetPasswordDialogOpen.value = false;
    newPassword.value = "";
    await loadUsers();
  } catch (err: unknown) {
    toast.error("Failed to update password", err instanceof Error ? err.message : "Unknown error");
  } finally {
    savingPassword.value = false;
  }
};

const removeSSHKeyFromUser = async (username: string, keyIndex: number) => {
  const user = users.value.find((u) => u.name === username);
  if (!user) return;

  // Remove the key at the specified index
  const updatedKeys = user.sshAuthorizedKeys.filter((_, idx) => idx !== keyIndex);
  
  // Map to key IDs
  const keyIds = updatedKeys
    .map((pubKey) => {
      const key = availableSSHKeys.value.find((k) => k.publicKey === pubKey);
      return key?.id;
    })
    .filter((id): id is string => !!id);

  try {
    await configClient.updateUserSSHKeys({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      userName: username,
      sshKeyIds: keyIds,
    });
    toast.success("SSH key removed", "Changes will be applied on the next reboot.");
    await loadUsers();
  } catch (err: unknown) {
    toast.error("Failed to remove SSH key", err instanceof Error ? err.message : "Unknown error");
  }
};

watch(() => props.vpsId, () => {
  if (props.vpsId) {
    loadUsers();
    loadSSHKeys();
  }
}, { immediate: true });

onMounted(() => {
  if (props.vpsId) {
    loadUsers();
    loadSSHKeys();
  }
});
</script>

