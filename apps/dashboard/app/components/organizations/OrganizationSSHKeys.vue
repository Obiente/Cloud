<script setup lang="ts">
import { ref, watch, computed } from "vue";
import { VPSService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import { useDialog } from "~/composables/useDialog";
import { ConnectError } from "@connectrpc/connect";
import { formatDate } from "~/utils/common";
import {
  PlusIcon,
  TrashIcon,
  KeyIcon,
  PencilIcon,
} from "@heroicons/vue/24/outline";

const props = defineProps<{
  organizationId: string;
}>();

const client = useConnectClient(VPSService);
const { toast } = useToast();
const { showConfirm } = useDialog();

const sshKeys = ref<Array<{
  id: string;
  name: string;
  publicKey: string;
  fingerprint: string;
  vpsId?: string;
  createdAt: { seconds: number | bigint; nanos: number } | undefined;
}>>([]);
const sshKeysLoading = ref(false);
const sshKeysError = ref<string | null>(null);
const addSSHKeyDialogOpen = ref(false);
const newSSHKeyName = ref("");
const newSSHKeyValue = ref("");
const addingSSHKey = ref(false);
const addSSHKeyError = ref("");
const removingSSHKey = ref<string | null>(null);
const editSSHKeyDialogOpen = ref(false);
const editingSSHKey = ref<string | null>(null);
const editingSSHKeyName = ref("");
const editingSSHKeyId = ref<string | null>(null);
const editingSSHKeyError = ref("");

const fetchSSHKeys = async () => {
  if (!props.organizationId) {
    sshKeys.value = [];
    return;
  }

  sshKeysLoading.value = true;
  sshKeysError.value = null;
  try {
    // List only org-wide keys (no vpsId)
    const res = await client.listSSHKeys({
      organizationId: props.organizationId,
    });
    sshKeys.value = (res.keys || []).map((key) => ({
      id: key.id || "",
      name: key.name || "",
      publicKey: key.publicKey || "",
      fingerprint: key.fingerprint || "",
      vpsId: key.vpsId,
      createdAt: key.createdAt as { seconds: number | bigint; nanos: number } | undefined,
    }));
  } catch (err: unknown) {
    sshKeysError.value = err instanceof Error ? err.message : "Unknown error";
    sshKeys.value = [];
  } finally {
    sshKeysLoading.value = false;
  }
};

const openAddSSHKeyDialog = () => {
  addSSHKeyDialogOpen.value = true;
  newSSHKeyName.value = "";
  newSSHKeyValue.value = "";
  addSSHKeyError.value = "";
};

const addSSHKey = async () => {
  if (!props.organizationId || !newSSHKeyName.value || !newSSHKeyValue.value) {
    return;
  }

  addingSSHKey.value = true;
  addSSHKeyError.value = "";
  try {
    // Clean the SSH key: remove all newlines and carriage returns
    // SSH keys should be a single continuous line
    const cleanedKey = newSSHKeyValue.value
      .trim()
      .replace(/\r\n/g, "")
      .replace(/\n/g, "")
      .replace(/\r/g, "")
      .trim();
    
    // Add as org-wide key (no vpsId)
    await client.addSSHKey({
      organizationId: props.organizationId,
      name: newSSHKeyName.value.trim(),
      publicKey: cleanedKey,
    });
    toast.success("SSH key added successfully");
    addSSHKeyDialogOpen.value = false;
    await fetchSSHKeys();
  } catch (err: unknown) {
    if (err instanceof ConnectError) {
      addSSHKeyError.value = err.message || "Failed to add SSH key";
    } else {
      addSSHKeyError.value = err instanceof Error ? err.message : "Unknown error";
    }
    toast.error("Failed to add SSH key", addSSHKeyError.value);
  } finally {
    addingSSHKey.value = false;
  }
};

const openEditSSHKeyDialog = (key: { id: string; name: string }) => {
  editingSSHKeyId.value = key.id;
  editingSSHKeyName.value = key.name;
  editingSSHKeyError.value = "";
  editSSHKeyDialogOpen.value = true;
};

const updateSSHKey = async () => {
  if (!props.organizationId || !editingSSHKeyId.value || !editingSSHKeyName.value.trim()) {
    return;
  }

  editingSSHKey.value = editingSSHKeyId.value;
  editingSSHKeyError.value = "";
  try {
    await client.updateSSHKey({
      organizationId: props.organizationId,
      keyId: editingSSHKeyId.value,
      name: editingSSHKeyName.value.trim(),
    });
    toast.success("SSH key name updated successfully");
    editSSHKeyDialogOpen.value = false;
    await fetchSSHKeys();
  } catch (err: unknown) {
    if (err instanceof ConnectError) {
      editingSSHKeyError.value = err.message || "Failed to update SSH key";
    } else {
      editingSSHKeyError.value = err instanceof Error ? err.message : "Unknown error";
    }
    toast.error("Failed to update SSH key", editingSSHKeyError.value);
  } finally {
    editingSSHKey.value = null;
  }
};

const removeSSHKey = async (keyId: string) => {
  if (!props.organizationId) {
    return;
  }

  // Find the key to check if it's org-wide
  const key = sshKeys.value.find((k) => k.id === keyId);
  const isOrgWide = key && !key.vpsId;

  let message = "Are you sure you want to remove this SSH key?";
  let affectedVPSList: string[] = [];
  
  if (isOrgWide) {
    // For org-wide keys, fetch the list of VPS instances that will be affected
    try {
      const vpsRes = await client.listVPS({
        organizationId: props.organizationId,
        page: 1,
        perPage: 100, // Get up to 100 VPS instances
      });
      
      // Filter to only VPS instances that are provisioned (have instance_id)
      affectedVPSList = (vpsRes.vpsInstances || [])
        .filter((vps) => vps.instanceId) // Only VPS instances that are provisioned
        .map((vps) => vps.name || vps.id)
        .slice(0, 20); // Limit to 20 for display
      
      if (affectedVPSList.length > 0) {
        const vpsCount = vpsRes.pagination?.total || affectedVPSList.length;
        let vpsListText = affectedVPSList.map((name) => `  • ${name}`).join("\n");
        if (vpsCount > affectedVPSList.length) {
          vpsListText += `\n  ... and ${vpsCount - affectedVPSList.length} more`;
        }
        message = `Are you sure you want to remove this organization-wide SSH key?\n\nThis will remove the key from ${vpsCount} VPS instance(s) in this organization:\n\n${vpsListText}`;
      } else {
        message = "Are you sure you want to remove this organization-wide SSH key? It will be removed from all VPS instances in this organization.";
      }
    } catch (err) {
      // If we can't fetch VPS list, show generic message
      message = "Are you sure you want to remove this organization-wide SSH key? It will be removed from all VPS instances in this organization.";
    }
  }

  const confirmed = await showConfirm({
    title: "Remove SSH Key",
    message: message,
    confirmLabel: "Remove",
    cancelLabel: "Cancel",
    variant: "danger",
  });

  if (!confirmed) {
    return;
  }

  removingSSHKey.value = keyId;
  try {
    const res = await client.removeSSHKey({
      organizationId: props.organizationId,
      keyId: keyId,
    });
    
    // Show success message with affected VPS count
    if (isOrgWide && res.affectedVpsIds && res.affectedVpsIds.length > 0) {
      toast.success(
        `SSH key removed successfully from ${res.affectedVpsIds.length} VPS instance(s)`
      );
    } else {
    toast.success("SSH key removed successfully");
    }
    await fetchSSHKeys();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to remove SSH key", message);
  } finally {
    removingSSHKey.value = null;
  }
};

const formatSSHKeyDate = (timestamp: { seconds: number | bigint; nanos: number } | undefined) => {
  if (!timestamp) return "Unknown";
  return formatDate(timestamp);
};

// Fetch SSH keys when organization changes
watch(() => props.organizationId, () => {
  fetchSSHKeys();
}, { immediate: true });
</script>

<template>
  <OuiStack gap="lg">
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex align="center" justify="between">
          <OuiStack gap="xs">
            <OuiText size="lg" weight="semibold">SSH Keys</OuiText>
            <OuiText color="muted" size="sm">
              Manage organization-wide SSH keys. These keys will be available to all VPS instances in this organization.
            </OuiText>
          </OuiStack>
          <OuiButton
            color="primary"
            size="sm"
            @click="openAddSSHKeyDialog"
            :disabled="!organizationId"
          >
            <PlusIcon class="h-4 w-4 mr-1" />
            Add SSH Key
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="md">
          <!-- Loading State -->
          <div v-if="sshKeysLoading" class="text-center py-8">
            <OuiText color="muted">Loading SSH keys...</OuiText>
          </div>

          <!-- Error State -->
          <OuiBox
            v-else-if="sshKeysError"
            variant="danger"
            p="md"
            rounded="md"
          >
            <OuiText color="danger">
              Failed to load SSH keys: {{ sshKeysError }}
            </OuiText>
          </OuiBox>

          <!-- Empty State -->
          <div v-else-if="sshKeys.length === 0" class="text-center py-8">
            <KeyIcon class="h-12 w-12 mx-auto text-text-muted mb-4" />
            <OuiText size="lg" weight="medium" class="mb-2">No SSH keys</OuiText>
            <OuiText color="muted" size="sm" class="mb-4">
              Add an organization-wide SSH key to enable access to all VPS instances.
            </OuiText>
            <OuiButton color="primary" size="sm" @click="openAddSSHKeyDialog">
              <PlusIcon class="h-4 w-4 mr-1" />
              Add SSH Key
            </OuiButton>
          </div>

          <!-- SSH Keys List -->
          <OuiStack v-else gap="sm">
            <OuiBox
              v-for="key in sshKeys"
              :key="key.id"
              variant="outline"
              p="md"
              rounded="md"
            >
              <OuiFlex align="start" justify="between" gap="md">
                <OuiStack gap="xs" class="flex-1">
                  <OuiFlex align="center" gap="sm">
                    <OuiText weight="medium">{{ key.name }}</OuiText>
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      @click="openEditSSHKeyDialog(key)"
                      :disabled="editingSSHKey === key.id"
                    >
                      <PencilIcon class="h-3 w-3" />
                    </OuiButton>
                    <OuiBadge variant="primary" size="sm">Organization-wide</OuiBadge>
                  </OuiFlex>
                  <OuiText size="xs" color="muted" class="font-mono break-all">
                    {{ key.fingerprint }}
                  </OuiText>
                  <OuiText size="xs" color="muted" class="font-mono break-all">
                    {{ key.publicKey }}
                  </OuiText>
                  <OuiText size="xs" color="muted">
                    Added {{ formatSSHKeyDate(key.createdAt) }}
                  </OuiText>
                </OuiStack>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="removeSSHKey(key.id)"
                  :disabled="removingSSHKey === key.id"
                >
                  <TrashIcon class="h-4 w-4" />
                </OuiButton>
              </OuiFlex>
            </OuiBox>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Edit SSH Key Dialog -->
    <OuiDialog v-model:open="editSSHKeyDialogOpen" title="Edit SSH Key Name">
      <OuiStack gap="md">
        <OuiText color="muted" size="sm">
          Update the name for this SSH key. The name will be synced to Proxmox.
        </OuiText>

        <OuiStack gap="xs">
          <OuiText size="sm" weight="medium">Name</OuiText>
          <OuiInput
            v-model="editingSSHKeyName"
            placeholder="My SSH Key"
            :disabled="editingSSHKey !== null"
          />
        </OuiStack>

        <OuiBox v-if="editingSSHKeyError" variant="danger" p="sm" rounded="md">
          <OuiText size="sm" color="danger">{{ editingSSHKeyError }}</OuiText>
        </OuiBox>

        <OuiFlex justify="end" gap="sm">
          <OuiButton
            variant="ghost"
            @click="editSSHKeyDialogOpen = false"
            :disabled="editingSSHKey !== null"
          >
            Cancel
          </OuiButton>
          <OuiButton
            color="primary"
            @click="updateSSHKey"
            :disabled="!editingSSHKeyName.trim() || editingSSHKey !== null"
          >
            <span v-if="editingSSHKey">Updating...</span>
            <span v-else>Update</span>
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>

    <!-- Add SSH Key Dialog -->
    <OuiDialog v-model:open="addSSHKeyDialogOpen" title="Add SSH Key">
      <OuiStack gap="md">
        <OuiText color="muted" size="sm">
          Add an organization-wide SSH key. This key will be available to all VPS instances in this organization.
        </OuiText>

        <OuiStack gap="xs">
          <OuiText size="sm" weight="medium">Name</OuiText>
          <OuiInput
            v-model="newSSHKeyName"
            placeholder="My SSH Key"
            :disabled="addingSSHKey"
          />
        </OuiStack>

        <OuiStack gap="xs">
          <OuiText size="sm" weight="medium">Public Key</OuiText>
          <OuiTextarea
            v-model="newSSHKeyValue"
            placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQ..."
            :rows="4"
            :disabled="addingSSHKey"
          />
          <OuiText size="xs" color="muted">
            Paste your SSH public key here (starts with ssh-rsa, ssh-ed25519, etc.)
          </OuiText>
        </OuiStack>

        <OuiBox v-if="addSSHKeyError" variant="danger" p="sm" rounded="md">
          <OuiText size="sm" color="danger">{{ addSSHKeyError }}</OuiText>
        </OuiBox>

        <OuiFlex justify="end" gap="sm">
          <OuiButton
            variant="ghost"
            @click="addSSHKeyDialogOpen = false"
            :disabled="addingSSHKey"
          >
            Cancel
          </OuiButton>
          <OuiButton
            color="primary"
            @click="addSSHKey"
            :disabled="!newSSHKeyName || !newSSHKeyValue || addingSSHKey"
          >
            Add Key
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>
  </OuiStack>
</template>

