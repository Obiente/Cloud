<template>
  <OuiContainer size="full">
    <OuiStack gap="xl">
      <!-- Back Button -->
      <OuiButton
        variant="ghost"
        @click="router.push('/vps')"
        class="self-start gap-2"
        size="sm"
      >
        <ArrowLeftIcon class="h-4 w-4" />
        Back to VPS Instances
      </OuiButton>

      <!-- Loading State -->
      <OuiCard v-if="pending">
        <OuiCardBody>
          <div class="text-center py-16">
            <OuiSpinner size="lg" />
            <OuiText color="secondary" class="mt-4">Loading VPS instance...</OuiText>
          </div>
        </OuiCardBody>
      </OuiCard>

      <!-- Error State -->
      <OuiCard v-else-if="error || accessError">
        <OuiCardBody>
          <div class="text-center py-16">
            <ExclamationCircleIcon class="h-12 w-12 text-danger mx-auto mb-4" />
            <OuiText color="danger" size="lg" weight="semibold" class="mb-2">
              Failed to load VPS instance
            </OuiText>
            <OuiText color="secondary" class="mb-4">
              {{ error || accessError?.message || "Unknown error" }}
            </OuiText>
            <OuiButton @click="refreshVPS()" variant="outline" class="gap-2">
              <ArrowPathIcon class="h-4 w-4" />
              Try Again
            </OuiButton>
          </div>
        </OuiCardBody>
      </OuiCard>

      <!-- VPS Content -->
      <template v-else-if="vps">
        <!-- Header -->
        <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
          <OuiStack gap="sm" class="max-w-xl">
            <OuiFlex align="center" gap="md">
              <OuiBox
                p="sm"
                rounded="xl"
                bg="accent-primary"
                :class="statusClass"
              >
                <ServerIcon class="w-6 h-6" :class="statusIconClass" />
              </OuiBox>
              <OuiStack gap="xs">
                <OuiText as="h1" size="3xl" weight="bold">{{ vps.name }}</OuiText>
                <OuiText color="secondary" size="sm">
                  {{ vps.id }}
                </OuiText>
              </OuiStack>
            </OuiFlex>
            <OuiText v-if="vps.description" color="secondary" size="md">
              {{ vps.description }}
            </OuiText>
          </OuiStack>

          <OuiFlex gap="sm" wrap="wrap">
            <OuiButton
              v-if="vps.status === VPSStatus.STOPPED"
              color="success"
              @click="handleStart"
              :disabled="isActioning"
              class="gap-2"
            >
              <PlayIcon class="h-4 w-4" />
              Start
            </OuiButton>
            <OuiButton
              v-if="vps.status === VPSStatus.RUNNING"
              color="warning"
              @click="handleStop"
              :disabled="isActioning"
              class="gap-2"
            >
              <StopIcon class="h-4 w-4" />
              Stop
            </OuiButton>
            <OuiButton
              v-if="vps.status === VPSStatus.RUNNING"
              variant="outline"
              @click="handleReboot"
              :disabled="isActioning"
              class="gap-2"
            >
              <ArrowPathIcon class="h-4 w-4" />
              Reboot
            </OuiButton>
            <OuiButton
              variant="outline"
              color="danger"
              @click="handleDelete"
              :disabled="isActioning"
              class="gap-2"
            >
              <TrashIcon class="h-4 w-4" />
              Delete
            </OuiButton>
          </OuiFlex>
        </OuiFlex>

        <!-- Status Badge -->
        <OuiCard>
          <OuiCardBody>
            <OuiFlex align="center" gap="md" wrap="wrap">
              <OuiText weight="medium">Status:</OuiText>
              <OuiBadge :color="statusBadgeColor" size="md">
                {{ statusLabel }}
              </OuiBadge>
              <OuiText v-if="vps.instanceId" color="secondary" size="sm">
                VM ID: {{ vps.instanceId }}
              </OuiText>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>

        <!-- Details Grid -->
        <OuiGrid cols="1" cols-md="2" gap="md">
          <!-- Resource Specifications -->
          <OuiCard>
            <OuiCardHeader>
              <OuiText as="h2" class="oui-card-title">Resource Specifications</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex justify="between">
                  <OuiText color="secondary">CPU Cores</OuiText>
                  <OuiText weight="medium">{{ vps.cpuCores }}</OuiText>
                </OuiFlex>
                <OuiFlex justify="between">
                  <OuiText color="secondary">Memory</OuiText>
                  <OuiText weight="medium">
                    <OuiByte :value="vps.memoryBytes" />
                  </OuiText>
                </OuiFlex>
                <OuiFlex justify="between">
                  <OuiText color="secondary">Storage</OuiText>
                  <OuiText weight="medium">
                    <OuiByte :value="vps.diskBytes" />
                  </OuiText>
                </OuiFlex>
                <OuiFlex justify="between">
                  <OuiText color="secondary">Instance Size</OuiText>
                  <OuiText weight="medium">{{ vps.size }}</OuiText>
                </OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Network Information -->
          <OuiCard>
            <OuiCardHeader>
              <OuiText as="h2" class="oui-card-title">Network Information</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <div v-if="vps.ipv4Addresses && vps.ipv4Addresses.length > 0">
                  <OuiText color="secondary" size="sm" class="mb-1">IPv4 Addresses</OuiText>
                  <OuiStack gap="xs">
                    <OuiText
                      v-for="(ip, idx) in vps.ipv4Addresses"
                      :key="idx"
                      size="sm"
                      class="font-mono"
                    >
                      {{ ip }}
                    </OuiText>
                  </OuiStack>
                </div>
                <div v-else>
                  <OuiText color="secondary" size="sm">No IPv4 addresses assigned</OuiText>
                </div>
                <div v-if="vps.ipv6Addresses && vps.ipv6Addresses.length > 0">
                  <OuiText color="secondary" size="sm" class="mb-1">IPv6 Addresses</OuiText>
                  <OuiStack gap="xs">
                    <OuiText
                      v-for="(ip, idx) in vps.ipv6Addresses"
                      :key="idx"
                      size="sm"
                      class="font-mono"
                    >
                      {{ ip }}
                    </OuiText>
                  </OuiStack>
                </div>
                <div v-else>
                  <OuiText color="secondary" size="sm">No IPv6 addresses assigned</OuiText>
                </div>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Configuration -->
          <OuiCard>
            <OuiCardHeader>
              <OuiText as="h2" class="oui-card-title">Configuration</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex justify="between">
                  <OuiText color="secondary">Region</OuiText>
                  <OuiText weight="medium">{{ vps.region || "—" }}</OuiText>
                </OuiFlex>
                <OuiFlex justify="between">
                  <OuiText color="secondary">Operating System</OuiText>
                  <OuiText weight="medium">{{ imageLabel }}</OuiText>
                </OuiFlex>
                <OuiFlex justify="between" v-if="vps.nodeId">
                  <OuiText color="secondary">Node ID</OuiText>
                  <OuiText weight="medium" class="font-mono text-xs">{{ vps.nodeId }}</OuiText>
                </OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Timestamps -->
          <OuiCard>
            <OuiCardHeader>
              <OuiText as="h2" class="oui-card-title">Timestamps</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex justify="between">
                  <OuiText color="secondary">Created</OuiText>
                  <OuiText weight="medium">
                    <OuiRelativeTime
                      :value="vps.createdAt ? date(vps.createdAt) : undefined"
                      :style="'short'"
                    />
                  </OuiText>
                </OuiFlex>
                <OuiFlex justify="between">
                  <OuiText color="secondary">Last Updated</OuiText>
                  <OuiText weight="medium">
                    <OuiRelativeTime
                      :value="vps.updatedAt ? date(vps.updatedAt) : undefined"
                      :style="'short'"
                    />
                  </OuiText>
                </OuiFlex>
                <OuiFlex justify="between" v-if="vps.lastStartedAt">
                  <OuiText color="secondary">Last Started</OuiText>
                  <OuiText weight="medium">
                    <OuiDate :value="vps.lastStartedAt" />
                  </OuiText>
                </OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <!-- Tabbed Content -->
        <OuiStack gap="sm" class="md:gap-md">
          <OuiTabs v-model="activeTab" :tabs="tabs" />
          <OuiCard variant="default">
            <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
              <template #overview>
                <!-- Connection Information -->
                <OuiCard v-if="vps.status === VPSStatus.RUNNING">
                  <OuiCardHeader>
                    <OuiText as="h2" class="oui-card-title">Connection Information</OuiText>
                  </OuiCardHeader>
                  <OuiCardBody>
                    <OuiStack gap="md">
                      <OuiText color="secondary" size="sm">
                        Access your VPS instance using one of the following methods:
                      </OuiText>
                      <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                        <OuiStack gap="sm">
                          <OuiText size="sm" weight="semibold" color="primary">Web Terminal</OuiText>
                          <OuiText size="sm" color="secondary">
                            Use the built-in web terminal to access your VPS directly from the browser.
                          </OuiText>
                          <OuiButton
                            variant="outline"
                            size="sm"
                            @click="openTerminal"
                            class="self-start gap-2"
                          >
                            <CommandLineIcon class="h-4 w-4" />
                            Open Terminal
                          </OuiButton>
                        </OuiStack>
                      </OuiBox>
                      <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                        <OuiStack gap="sm">
                          <OuiText size="sm" weight="semibold" color="primary">SSH Access</OuiText>
                          <OuiText size="sm" color="secondary">
                            Connect to your VPS via SSH using the SSH proxy.
                          </OuiText>
                          <div v-if="sshInfo" class="mt-2">
                            <OuiText size="xs" weight="semibold" class="mb-1">SSH Command:</OuiText>
                            <OuiBox p="sm" rounded="md" class="bg-surface-muted font-mono text-xs overflow-x-auto">
                              <code>{{ sshInfo.sshProxyCommand }}</code>
                            </OuiBox>
                            <OuiButton
                              variant="ghost"
                              size="xs"
                              @click="copySSHCommand"
                              class="mt-2"
                            >
                              <ClipboardDocumentListIcon class="h-3 w-3 mr-1" />
                              Copy Command
                            </OuiButton>
                            <div v-if="sshInfo.connectionInstructions" class="mt-4">
                              <OuiText size="xs" weight="semibold" class="mb-2">Full Connection Instructions:</OuiText>
                              <OuiBox p="sm" rounded="md" class="bg-surface-muted font-mono text-xs whitespace-pre-wrap overflow-x-auto">
                                <code>{{ sshInfo.connectionInstructions }}</code>
                              </OuiBox>
                              <OuiButton
                                variant="ghost"
                                size="xs"
                                @click="copyConnectionInstructions"
                                class="mt-2"
                              >
                                <ClipboardDocumentListIcon class="h-3 w-3 mr-1" />
                                Copy Instructions
                              </OuiButton>
                            </div>
                          </div>
                          <div v-else-if="sshInfoLoading" class="mt-2">
                            <OuiText size="xs" color="secondary">Loading SSH connection info...</OuiText>
                          </div>
                          <div v-else-if="sshInfoError" class="mt-2">
                            <OuiText size="xs" color="danger">
                              Failed to load SSH connection info. {{ sshInfoError }}
                            </OuiText>
                          </div>
                        </OuiStack>
                      </OuiBox>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>

                <!-- Firewall Management -->
                <VPSFirewall
                  v-if="vps.instanceId"
                  :vps-id="vpsId"
                  :organization-id="orgId"
                />
              </template>
              <template #terminal>
                <VPSXTermTerminal
                  :vps-id="vpsId"
                  :organization-id="orgId"
                />
              </template>
              <template #ssh-settings>
                <!-- SSH Keys Management -->
                <OuiStack gap="md">
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">SSH Keys</OuiText>
                        <OuiText color="secondary" size="sm">
                          Manage SSH public keys for accessing your VPS instances. These keys are automatically added to new VPS instances via cloud-init.
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <!-- Add SSH Key Button -->
                        <OuiFlex justify="end">
                          <OuiButton
                            variant="solid"
                            size="sm"
                            @click="openAddSSHKeyDialog"
                          >
                            <KeyIcon class="h-4 w-4 mr-2" />
                            Add SSH Key
                          </OuiButton>
                        </OuiFlex>

                        <!-- SSH Keys List -->
                        <div v-if="sshKeysLoading" class="py-8">
                          <OuiText color="secondary" class="text-center">Loading SSH keys...</OuiText>
                        </div>
                        <div v-else-if="sshKeysError" class="py-8">
                          <OuiText color="danger" class="text-center">
                            Failed to load SSH keys: {{ sshKeysError }}
                          </OuiText>
                        </div>
                        <div v-else-if="sshKeys.length === 0" class="py-8">
                          <OuiText color="secondary" class="text-center">
                            No SSH keys found. Add your first SSH key to get started.
                          </OuiText>
                        </div>
                        <div v-else class="space-y-3">
                          <OuiBox
                            v-for="key in sshKeys"
                            :key="key.id"
                            p="md"
                            rounded="lg"
                            class="bg-surface-muted/40 ring-1 ring-border-muted"
                          >
                            <OuiStack gap="sm">
                              <OuiFlex justify="between" align="start">
                                <OuiStack gap="xs">
                                  <OuiFlex align="center" gap="sm">
                                  <OuiText size="sm" weight="semibold">{{ key.name }}</OuiText>
                                    <OuiButton
                                      variant="ghost"
                                      size="xs"
                                      @click="openEditSSHKeyDialog(key)"
                                      :disabled="editingSSHKey === key.id"
                                    >
                                      <PencilIcon class="h-3 w-3" />
                                    </OuiButton>
                                    <OuiBadge v-if="!key.vpsId" variant="primary" size="sm">Organization-wide</OuiBadge>
                                  </OuiFlex>
                                  <OuiText size="xs" color="secondary" class="font-mono">
                                    {{ key.fingerprint }}
                                  </OuiText>
                                  <OuiText size="xs" color="muted">
                                    Added {{ formatSSHKeyDate(key.createdAt) }}
                                  </OuiText>
                                </OuiStack>
                                <OuiButton
                                  variant="ghost"
                                  color="danger"
                                  size="xs"
                                  @click="removeSSHKey(key.id)"
                                  :disabled="removingSSHKey === key.id"
                                >
                                  <TrashIcon class="h-3 w-3 mr-1" />
                                  Remove
                                </OuiButton>
                              </OuiFlex>
                              <OuiBox p="sm" rounded="md" class="bg-surface-muted font-mono text-xs overflow-x-auto">
                                <code>{{ key.publicKey }}</code>
                              </OuiBox>
                            </OuiStack>
                          </OuiBox>
                        </div>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiStack>

                <!-- Edit SSH Key Dialog -->
                <OuiDialog
                  v-model:open="editSSHKeyDialogOpen"
                  title="Edit SSH Key Name"
                  size="md"
                >
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
                <OuiDialog
                  v-model:open="addSSHKeyDialogOpen"
                  title="Add SSH Key"
                  size="md"
                >
                  <OuiStack gap="md">
                    <OuiText color="secondary" size="sm">
                      Paste your SSH public key below. This key will be added to all VPS instances in your organization.
                    </OuiText>
                    <OuiFormField label="Key Name" required>
                      <OuiInput
                        v-model="newSSHKeyName"
                        placeholder="e.g., My Laptop, Work Computer"
                        :disabled="addingSSHKey"
                      />
                    </OuiFormField>
                    <OuiFormField label="Public Key" required>
                      <OuiTextarea
                        v-model="newSSHKeyValue"
                        placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQ..."
                        :rows="4"
                        :disabled="addingSSHKey"
                      />
                      <OuiText size="xs" color="secondary" class="mt-1">
                        Paste your SSH public key (usually from ~/.ssh/id_rsa.pub or ~/.ssh/id_ed25519.pub)
                      </OuiText>
                    </OuiFormField>
                    <div v-if="addSSHKeyError" class="mt-2">
                      <OuiText size="sm" color="danger">{{ addSSHKeyError }}</OuiText>
                    </div>
                  </OuiStack>
                  <template #footer>
                    <OuiFlex justify="end" gap="sm">
                      <OuiButton
                        variant="ghost"
                        @click="addSSHKeyDialogOpen = false"
                        :disabled="addingSSHKey"
                      >
                        Cancel
                      </OuiButton>
                      <OuiButton
                        variant="solid"
                        @click="addSSHKey"
                        :disabled="addingSSHKey || !newSSHKeyName || !newSSHKeyValue"
                      >
                        <span v-if="addingSSHKey">Adding...</span>
                        <span v-else>Add Key</span>
                      </OuiButton>
                    </OuiFlex>
                  </template>
                </OuiDialog>
              </template>
              <template #audit-logs>
                <AuditLogs
                  :organization-id="orgId"
                  resource-type="vps"
                  :resource-id="vpsId"
                />
              </template>
            </OuiTabs>
          </OuiCard>
        </OuiStack>

      </template>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowLeftIcon,
  ArrowPathIcon,
  CommandLineIcon,
  ExclamationCircleIcon,
  PlayIcon,
  ServerIcon,
  StopIcon,
  TrashIcon,
  InformationCircleIcon,
  ClipboardDocumentListIcon,
  KeyIcon,
  PencilIcon,
} from "@heroicons/vue/24/outline";
import { VPSService, VPSStatus, VPSImage, type VPSInstance } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import { useOrganizationsStore } from "~/stores/organizations";
import { useDialog } from "~/composables/useDialog";
import { ConnectError, Code } from "@connectrpc/connect";
import OuiByte from "~/components/oui/Byte.vue";
import OuiDate from "~/components/oui/Date.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import VPSFirewall from "~/components/vps/VPSFirewall.vue";
import VPSXTermTerminal from "~/components/vps/VPSXTermTerminal.vue";
import AuditLogs from "~/components/audit/AuditLogs.vue";
import type { TabItem } from "~/components/oui/Tabs.vue";
import { useTabQuery } from "~/composables/useTabQuery";
import { date } from "@obiente/proto/utils";
import { formatDate } from "~/utils/common";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();
const { toast } = useToast();
const { showAlert, showConfirm } = useDialog();
const orgsStore = useOrganizationsStore();

const vpsId = computed(() => String(route.params.id));
const orgId = computed(() => orgsStore.currentOrgId || "");

const client = useConnectClient(VPSService);
const accessError = ref<Error | null>(null);
const isActioning = ref(false);

// Fetch VPS data
const {
  data: vpsData,
  pending,
  error,
  refresh: refreshVPS,
} = await useAsyncData(
  () => `vps-${vpsId.value}`,
  async () => {
    try {
      const res = await client.getVPS({
        organizationId: orgId.value,
        vpsId: vpsId.value,
      });
      accessError.value = null;
      return res.vps ?? null;
    } catch (err: unknown) {
      if (err instanceof ConnectError) {
        if (err.code === Code.NotFound || err.code === Code.PermissionDenied) {
          accessError.value = err;
          return null;
        }
      }
      throw err;
    }
  },
  {
    watch: [vpsId, orgId],
  }
);

const vps = computed(() => vpsData.value);

// Fetch SSH connection info
const sshInfo = ref<{ sshProxyCommand: string; connectionInstructions: string } | null>(null);
const sshInfoLoading = ref(false);
const sshInfoError = ref<string | null>(null);

const fetchSSHInfo = async () => {
  if (!vps.value || vps.value.status !== VPSStatus.RUNNING) {
    sshInfo.value = null;
    return;
  }

  sshInfoLoading.value = true;
  sshInfoError.value = null;
  try {
    const res = await client.getVPSProxyInfo({
      vpsId: vpsId.value,
    });
    sshInfo.value = {
      sshProxyCommand: res.sshProxyCommand || "",
      connectionInstructions: res.connectionInstructions || "",
    };
  } catch (err: unknown) {
    sshInfoError.value = err instanceof Error ? err.message : "Unknown error";
    sshInfo.value = null;
  } finally {
    sshInfoLoading.value = false;
  }
};

// Fetch SSH info when VPS is running
watch(
  () => vps.value?.status,
  (status) => {
    if (status === VPSStatus.RUNNING) {
      fetchSSHInfo();
    } else {
      sshInfo.value = null;
    }
  },
  { immediate: true }
);

// SSH Keys Management
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
  if (!orgId.value) {
    sshKeys.value = [];
    return;
  }

  sshKeysLoading.value = true;
  sshKeysError.value = null;
  try {
    const res = await client.listSSHKeys({
      organizationId: orgId.value,
      vpsId: vpsId.value,
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
  if (!orgId.value || !newSSHKeyName.value || !newSSHKeyValue.value) {
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
    
    await client.addSSHKey({
      organizationId: orgId.value,
      name: newSSHKeyName.value.trim(),
      publicKey: cleanedKey,
      vpsId: vpsId.value,
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
  if (!orgId.value || !editingSSHKeyId.value || !editingSSHKeyName.value.trim()) {
    return;
  }

  editingSSHKey.value = editingSSHKeyId.value;
  editingSSHKeyError.value = "";
  try {
    await client.updateSSHKey({
      organizationId: orgId.value,
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
  if (!orgId.value) {
    return;
  }

  // Find the key to check if it's org-wide
  const key = sshKeys.value.find((k) => k.id === keyId);
  const isOrgWide = key && !key.vpsId;

  let message = "Are you sure you want to remove this SSH key?";
  if (isOrgWide) {
    // For org-wide keys, fetch the list of VPS instances that will be affected
    try {
      const vpsRes = await client.listVPS({
        organizationId: orgId.value,
        page: 1,
        perPage: 100, // Get up to 100 VPS instances
      });
      
      // Filter to only VPS instances that are provisioned (have instance_id)
      const affectedVPSList = (vpsRes.vpsInstances || [])
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
  } else {
    message = "Are you sure you want to remove this SSH key? You will no longer be able to use it to access this VPS instance.";
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
      organizationId: orgId.value,
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
watch(orgId, () => {
  fetchSSHKeys();
}, { immediate: true });

const copySSHCommand = async () => {
  if (!sshInfo.value?.sshProxyCommand) return;
  try {
    await navigator.clipboard.writeText(sshInfo.value.sshProxyCommand);
    toast.success("SSH command copied to clipboard");
  } catch (err) {
    toast.error("Failed to copy SSH command");
  }
};

const copyConnectionInstructions = async () => {
  if (!sshInfo.value?.connectionInstructions) return;
  try {
    await navigator.clipboard.writeText(sshInfo.value.connectionInstructions);
    toast.success("Connection instructions copied to clipboard");
  } catch (err) {
    toast.error("Failed to copy connection instructions");
  }
};

// Status helpers
const statusLabel = computed(() => {
  if (!vps.value) return "Unknown";
  const status = vps.value.status;
  switch (status) {
    case VPSStatus.CREATING:
      return "Creating";
    case VPSStatus.STARTING:
      return "Starting";
    case VPSStatus.RUNNING:
      return "Running";
    case VPSStatus.STOPPING:
      return "Stopping";
    case VPSStatus.STOPPED:
      return "Stopped";
    case VPSStatus.REBOOTING:
      return "Rebooting";
    case VPSStatus.FAILED:
      return "Failed";
    case VPSStatus.DELETING:
      return "Deleting";
    case VPSStatus.DELETED:
      return "Deleted";
    default:
      return "Unknown";
  }
});

const statusBadgeColor = computed(() => {
  if (!vps.value) return "secondary";
  const status = vps.value.status;
  switch (status) {
    case VPSStatus.RUNNING:
      return "success";
    case VPSStatus.CREATING:
    case VPSStatus.STARTING:
    case VPSStatus.REBOOTING:
      return "info";
    case VPSStatus.STOPPED:
    case VPSStatus.STOPPING:
      return "secondary";
    case VPSStatus.FAILED:
      return "danger";
    default:
      return "secondary";
  }
});

const statusClass = computed(() => {
  if (!vps.value) return "bg-secondary/10 ring-1 ring-secondary/20";
  const status = vps.value.status;
  switch (status) {
    case VPSStatus.RUNNING:
      return "bg-success/10 ring-1 ring-success/20";
    case VPSStatus.CREATING:
    case VPSStatus.STARTING:
    case VPSStatus.REBOOTING:
      return "bg-info/10 ring-1 ring-info/20";
    case VPSStatus.STOPPED:
    case VPSStatus.STOPPING:
      return "bg-secondary/10 ring-1 ring-secondary/20";
    case VPSStatus.FAILED:
      return "bg-danger/10 ring-1 ring-danger/20";
    default:
      return "bg-secondary/10 ring-1 ring-secondary/20";
  }
});

const statusIconClass = computed(() => {
  if (!vps.value) return "text-secondary";
  const status = vps.value.status;
  switch (status) {
    case VPSStatus.RUNNING:
      return "text-success";
    case VPSStatus.CREATING:
    case VPSStatus.STARTING:
    case VPSStatus.REBOOTING:
      return "text-info";
    case VPSStatus.STOPPED:
    case VPSStatus.STOPPING:
      return "text-secondary";
    case VPSStatus.FAILED:
      return "text-danger";
    default:
      return "text-secondary";
  }
});

const imageLabel = computed(() => {
  if (!vps.value) return "—";
  const image = vps.value.image;
  switch (image) {
    case VPSImage.UBUNTU_22_04:
      return "Ubuntu 22.04 LTS";
    case VPSImage.UBUNTU_24_04:
      return "Ubuntu 24.04 LTS";
    case VPSImage.DEBIAN_12:
      return "Debian 12";
    case VPSImage.DEBIAN_13:
      return "Debian 13";
    case VPSImage.ROCKY_LINUX_9:
      return "Rocky Linux 9";
    case VPSImage.ALMA_LINUX_9:
      return "AlmaLinux 9";
    case VPSImage.CUSTOM:
      return vps.value.imageId || "Custom Image";
    default:
      return "Unknown";
  }
});

// Actions
async function handleStart() {
  if (!vps.value) return;
  const confirmed = await showConfirm({
    title: "Start VPS Instance",
    message: `Are you sure you want to start "${vps.value.name}"?`,
  });
  if (!confirmed) return;

  isActioning.value = true;
  try {
    await client.startVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    toast.success("VPS instance started", "The VPS instance is starting up.");
    await refreshVPS();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to start VPS", message);
  } finally {
    isActioning.value = false;
  }
}

async function handleStop() {
  if (!vps.value) return;
  const confirmed = await showConfirm({
    title: "Stop VPS Instance",
    message: `Are you sure you want to stop "${vps.value.name}"? The instance will be stopped and will not consume resources.`,
  });
  if (!confirmed) return;

  isActioning.value = true;
  try {
    await client.stopVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    toast.success("VPS instance stopped", "The VPS instance has been stopped.");
    await refreshVPS();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to stop VPS", message);
  } finally {
    isActioning.value = false;
  }
}

async function handleReboot() {
  if (!vps.value) return;
  const confirmed = await showConfirm({
    title: "Reboot VPS Instance",
    message: `Are you sure you want to reboot "${vps.value.name}"? The instance will restart.`,
  });
  if (!confirmed) return;

  isActioning.value = true;
  try {
    await client.rebootVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    toast.success("VPS instance rebooting", "The VPS instance is rebooting.");
    await refreshVPS();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to reboot VPS", message);
  } finally {
    isActioning.value = false;
  }
}

async function handleDelete() {
  if (!vps.value) return;
  const confirmed = await showConfirm({
    title: "Delete VPS Instance",
    message: `Are you sure you want to delete "${vps.value.name}"? This action cannot be undone. All data on the VPS will be permanently lost.`,
    confirmLabel: "Delete",
    cancelLabel: "Cancel",
    variant: "danger",
  });
  if (!confirmed) return;

  isActioning.value = true;
  try {
    await client.deleteVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
      force: false,
    });
    toast.success("VPS instance deleted", "The VPS instance has been deleted.");
    router.push("/vps");
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to delete VPS", message);
    isActioning.value = false;
  }
}

// Tabs configuration
const tabs = computed<TabItem[]>(() => [
  { id: "overview", label: "Overview", icon: InformationCircleIcon },
  { id: "terminal", label: "Terminal", icon: CommandLineIcon },
  { id: "ssh-settings", label: "SSH Settings", icon: KeyIcon },
  { id: "audit-logs", label: "Audit Logs", icon: ClipboardDocumentListIcon },
]);

// Use composable for tab query parameter management
const activeTab = useTabQuery(tabs, "overview");

function openTerminal() {
  activeTab.value = "terminal";
}
</script>

