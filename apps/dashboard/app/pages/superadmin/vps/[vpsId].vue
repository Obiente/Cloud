<template>
  <OuiContainer size="full">
    <OuiStack gap="2xl">
      <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
        <OuiStack gap="xs">
          <OuiFlex gap="sm" align="center">
            <OuiButton
              variant="ghost"
              size="sm"
              @click="router.back()"
            >
              <ArrowLeftIcon class="h-4 w-4 mr-1" />
              Back
            </OuiButton>
            <OuiText tag="h1" size="3xl" weight="extrabold">VPS Details</OuiText>
          </OuiFlex>
          <OuiText color="muted">View detailed information and manage this VPS instance.</OuiText>
        </OuiStack>
        <OuiFlex gap="sm" align="center">
          <OuiButton
            v-if="vps?.id"
            variant="outline"
            @click="goToVPSPage"
          >
            Open VPS Page
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <OuiGrid cols="1" colsLg="2" gap="lg">
        <!-- VPS Info & Specifications Card -->
        <OuiCard class="border border-border-muted rounded-xl">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">VPS Information</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-6">
            <OuiStack gap="lg">
              <OuiStack gap="xs">
                <OuiText size="xl" weight="bold">
                  {{ vps?.name || vps?.id || "Loading..." }}
                </OuiText>
                <OuiText v-if="vps?.description" color="muted" size="sm">
                  {{ vps.description }}
                </OuiText>
                <OuiText v-if="vps?.id" color="muted" size="xs" class="font-mono">
                  {{ vps.id }}
                </OuiText>
              </OuiStack>

              <OuiGrid cols="1" colsMd="2" gap="md" class="border-t border-border-muted pt-4">
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Status</OuiText>
                  <SuperadminStatusBadge
                    :status="vps?.status"
                    :status-map="vpsStatusMap"
                  />
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Organization</OuiText>
                  <NuxtLink
                    v-if="vps?.organizationId"
                    :to="`/superadmin/organizations?organizationId=${vps.organizationId}`"
                    class="font-medium text-text-primary hover:text-primary transition-colors"
                  >
                    {{ organizationName || vps.organizationId }}
                  </NuxtLink>
                  <OuiText v-else color="muted" size="sm">—</OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="createdBy">
                  <OuiText size="sm" weight="medium" color="muted">Created By</OuiText>
                  <NuxtLink
                    v-if="createdBy.id"
                    :to="`/superadmin/users/${createdBy.id}`"
                    class="font-medium text-text-primary hover:text-primary transition-colors"
                  >
                    <OuiFlex gap="sm" align="center">
                      <OuiAvatar
                        v-if="createdBy.avatarUrl"
                        :name="createdBy.name || createdBy.email || createdBy.id"
                        :src="createdBy.avatarUrl"
                        size="sm"
                      />
                      <OuiText>
                        {{ createdBy.name || createdBy.email || createdBy.preferredUsername || createdBy.id }}
                      </OuiText>
                    </OuiFlex>
                  </NuxtLink>
                  <OuiText v-else color="muted" size="sm">—</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Region</OuiText>
                  <OuiText>{{ vps?.region || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Node</OuiText>
                  <OuiText class="font-mono">{{ vps?.nodeId || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium" color="muted">Instance ID</OuiText>
                  <OuiText class="font-mono">{{ vps?.instanceId || "—" }}</OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="vps?.createdAt">
                  <OuiText size="sm" weight="medium" color="muted">Created</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="vps.createdAt" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="vps?.updatedAt">
                  <OuiText size="sm" weight="medium" color="muted">Last Updated</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="vps.updatedAt" />
                  </OuiText>
                </OuiStack>

                <OuiStack gap="xs" v-if="vps?.lastStartedAt">
                  <OuiText size="sm" weight="medium" color="muted">Last Started</OuiText>
                  <OuiText size="sm">
                    <OuiDate :value="vps.lastStartedAt" />
                  </OuiText>
                </OuiStack>
              </OuiGrid>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Specifications Card -->
        <OuiCard class="border border-border-muted rounded-xl">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">Specifications</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-6">
            <OuiGrid cols="1" colsMd="2" gap="md">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="muted">Size</OuiText>
                <OuiText>{{ vps?.size || "—" }}</OuiText>
              </OuiStack>

              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="muted">CPU Cores</OuiText>
                <OuiText>{{ vps?.cpuCores || 0 }} cores</OuiText>
              </OuiStack>

              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="muted">Memory</OuiText>
                <OuiText>{{ formatMemory(vps?.memoryBytes) }}</OuiText>
              </OuiStack>

              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="muted">Disk</OuiText>
                <OuiText>{{ formatDisk(vps?.diskBytes) }}</OuiText>
              </OuiStack>

              <OuiStack gap="xs" class="col-span-1 md:col-span-2">
                <OuiText size="sm" weight="medium" color="muted">Image</OuiText>
                <OuiText>{{ formatImage(vps?.image, vps?.imageId) }}</OuiText>
              </OuiStack>
            </OuiGrid>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Network & Actions Row -->
      <OuiGrid cols="1" colsLg="2" gap="lg">
        <!-- Network & SSH Card -->
        <OuiCard class="border border-border-muted rounded-xl">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">Network & Access</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-6">
            <OuiGrid cols="1" colsMd="2" gap="md">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="muted">IPv4 Addresses</OuiText>
                <OuiStack gap="xs">
                  <OuiText
                    v-for="ip in vps?.ipv4Addresses"
                    :key="ip"
                    class="font-mono"
                    size="sm"
                  >
                    {{ ip }}
                  </OuiText>
                  <OuiText v-if="!vps?.ipv4Addresses?.length" color="muted" size="sm">
                    No IPv4 addresses
                  </OuiText>
                </OuiStack>
              </OuiStack>

              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium" color="muted">IPv6 Addresses</OuiText>
                <OuiStack gap="xs">
                  <OuiText
                    v-for="ip in vps?.ipv6Addresses"
                    :key="ip"
                    class="font-mono"
                    size="sm"
                  >
                    {{ ip }}
                  </OuiText>
                  <OuiText v-if="!vps?.ipv6Addresses?.length" color="muted" size="sm">
                    No IPv6 addresses
                  </OuiText>
                </OuiStack>
              </OuiStack>

              <OuiStack gap="xs" v-if="vps?.sshAlias">
                <OuiText size="sm" weight="medium" color="muted">SSH Alias</OuiText>
                <OuiText class="font-mono">{{ vps.sshAlias }}</OuiText>
                <OuiText size="xs" color="muted">
                  Use this alias for SSH connections
                </OuiText>
              </OuiStack>

              <OuiStack gap="xs" v-if="vps?.sshKeyId">
                <OuiText size="sm" weight="medium" color="muted">SSH Key ID</OuiText>
                <OuiText class="font-mono">{{ vps.sshKeyId }}</OuiText>
              </OuiStack>

              <OuiStack gap="xs" v-if="hasMetadata" class="col-span-1 md:col-span-2">
                <OuiText size="sm" weight="medium" color="muted">Metadata</OuiText>
                <OuiGrid cols="1" colsMd="2" gap="xs">
                  <OuiText
                    v-for="(value, key) in parsedMetadata"
                    :key="key"
                    size="sm"
                  >
                    <span class="font-medium">{{ key }}:</span>
                    <span class="ml-1">{{ value }}</span>
                  </OuiText>
                </OuiGrid>
              </OuiStack>
            </OuiGrid>
          </OuiCardBody>
        </OuiCard>

        <!-- Actions Card -->
        <OuiCard class="border border-border-muted rounded-xl">
          <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
            <OuiText tag="h2" size="lg" weight="bold">Superadmin Actions</OuiText>
          </OuiCardHeader>
          <OuiCardBody class="p-6">
            <OuiStack gap="md">
              <OuiText size="sm" color="muted">
                Perform administrative actions on this VPS instance. Use with caution.
              </OuiText>
              <OuiFlex gap="sm" wrap="wrap">
                <OuiButton
                  variant="outline"
                  color="primary"
                  @click="openResizeDialog"
                  :disabled="loading || !vps"
                >
                  Resize
                </OuiButton>
              <OuiButton
                v-if="vps?.status !== VPSStatus.SUSPENDED"
                variant="outline"
                color="warning"
                @click="openSuspendDialog"
                :disabled="loading || !vps"
              >
                Suspend
              </OuiButton>
              <OuiButton
                v-else
                variant="outline"
                color="warning"
                @click="handleUnsuspend"
                :disabled="loading || isUnsuspending || !vps"
              >
                {{ isUnsuspending ? "Unsuspending..." : "Unsuspend" }}
              </OuiButton>
              <OuiButton
                variant="outline"
                @click="openCloudInitDialog"
                :disabled="loading || !vps"
              >
                Update CloudInit
              </OuiButton>
              <OuiButton
                variant="outline"
                color="danger"
                @click="openForceStopDialog"
                :disabled="loading || !vps"
              >
                Force Stop
              </OuiButton>
              <OuiButton
                variant="outline"
                color="danger"
                @click="openForceDeleteDialog"
                :disabled="loading || !vps"
              >
                Force Delete
              </OuiButton>
            </OuiFlex>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
      </OuiGrid>
    </OuiStack>

    <!-- Resize Dialog -->
    <OuiDialog v-model:open="resizeDialogOpen" title="Resize VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Change the VPS instance size. CPU, memory, and disk will be updated.
        </OuiText>
        <OuiSelect
          v-model="resizeForm.newSize"
          label="New Size"
          :items="sizeOptionsWithCustom"
          :loading="loadingSizes"
          required
        >
          <template #item="{ item }">
            <OuiStack gap="xs">
              <OuiText weight="medium">{{ item.label }}</OuiText>
              <OuiText v-if="item.value !== 'custom'" size="xs" color="secondary">
                {{ item.cpuCores }} CPU · {{ formatMemory(item.memoryBytes) }} RAM · {{ formatDisk(item.diskBytes) }} Storage
              </OuiText>
              <OuiText v-else size="xs" color="secondary">
                Configure custom CPU, memory, and disk sizes
              </OuiText>
            </OuiStack>
          </template>
        </OuiSelect>
        
        <!-- Custom Size Inputs -->
        <OuiStack v-if="resizeForm.newSize === 'custom'" gap="md" class="border border-border-muted rounded-lg p-4">
          <OuiText size="sm" weight="medium">Custom Size Configuration</OuiText>
          <OuiInput
            v-model="resizeForm.customCpuCores"
            type="number"
            label="CPU Cores"
            placeholder="e.g., 4"
            :min="1"
            :max="128"
            required
          />
          <OuiInput
            v-model="resizeForm.customMemoryGB"
            type="number"
            label="Memory (GB)"
            placeholder="e.g., 8"
            :min="0.5"
            :step="0.5"
            required
          />
          <OuiInput
            v-model="resizeForm.customDiskGB"
            type="number"
            label="Disk (GB)"
            placeholder="e.g., 100"
            :min="1"
            required
          />
        </OuiStack>
        
        <OuiCheckbox v-model="resizeForm.growDisk" label="Grow disk to new size" />
        <OuiCheckbox v-model="resizeForm.applyCloudInit" label="Apply cloud-init for disk growth" />
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="resizeDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="primary" 
            @click="handleResize"
            :disabled="!canResize || isResizing"
          >
            {{ isResizing ? 'Resizing...' : 'Resize' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Suspend Dialog -->
    <OuiDialog v-model:open="suspendDialogOpen" title="Suspend VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Suspend this VPS instance. The VPS will be marked as suspended and normal operations will be prevented.
        </OuiText>
        <OuiInput
          v-model="suspendForm.reason"
          label="Reason (Optional)"
          placeholder="Reason for suspension"
        />
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="suspendDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="warning" 
            @click="handleSuspend"
            :disabled="isSuspending"
          >
            {{ isSuspending ? 'Suspending...' : 'Suspend' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- CloudInit Dialog -->
    <OuiDialog v-model:open="cloudInitDialogOpen" title="Update CloudInit Configuration" size="lg">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Update the cloud-init configuration for this VPS. Changes will take effect on the next reboot.
        </OuiText>
        <OuiTextarea
          v-model="cloudInitForm.userData"
          label="CloudInit User Data (YAML)"
          placeholder="#cloud-config&#10;users:&#10;  - name: user1&#10;    ssh_authorized_keys:&#10;      - ssh-rsa ..."
          :rows="15"
        />
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="cloudInitDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="primary" 
            @click="handleUpdateCloudInit"
            :disabled="isUpdatingCloudInit"
          >
            {{ isUpdatingCloudInit ? 'Updating...' : 'Update CloudInit' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Force Stop Dialog -->
    <OuiDialog v-model:open="forceStopDialogOpen" title="Force Stop VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Forcefully stop this VPS instance. This action cannot be undone.
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="forceStopDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="danger" 
            @click="handleForceStop"
            :disabled="isForceStopping"
          >
            {{ isForceStopping ? 'Stopping...' : 'Force Stop' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Force Delete Dialog -->
    <OuiDialog v-model:open="forceDeleteDialogOpen" title="Force Delete VPS">
      <OuiStack gap="lg">
        <OuiText size="sm" color="muted">
          Permanently delete this VPS instance. This action cannot be undone and will remove all data.
        </OuiText>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="forceDeleteDialogOpen = false">Cancel</OuiButton>
          <OuiButton 
            color="danger" 
            @click="handleForceDelete"
            :disabled="isForceDeleting"
          >
            {{ isForceDeleting ? 'Deleting...' : 'Force Delete' }}
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ArrowLeftIcon } from "@heroicons/vue/24/outline";
import { SuperadminService, VPSStatus, VPSImage } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import SuperadminStatusBadge from "~/components/superadmin/SuperadminStatusBadge.vue";
import type { BadgeVariant } from "~/components/oui/Badge.vue";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const route = useRoute();
const router = useRouter();
const client = useConnectClient(SuperadminService);
const { toast } = useToast();

const vpsId = computed(() => route.params.vpsId as string);
const vps = ref<any>(null);
const organizationName = ref<string>("");
const createdBy = ref<any>(null);

const vpsStatusMap: Record<number, { label: string; variant: BadgeVariant }> = {
  [VPSStatus.CREATING]: { label: "Creating", variant: "warning" },
  [VPSStatus.STARTING]: { label: "Starting", variant: "warning" },
  [VPSStatus.RUNNING]: { label: "Running", variant: "success" },
  [VPSStatus.STOPPING]: { label: "Stopping", variant: "warning" },
  [VPSStatus.STOPPED]: { label: "Stopped", variant: "secondary" },
  [VPSStatus.REBOOTING]: { label: "Rebooting", variant: "warning" },
  [VPSStatus.FAILED]: { label: "Failed", variant: "danger" },
  [VPSStatus.DELETING]: { label: "Deleting", variant: "warning" },
  [VPSStatus.DELETED]: { label: "Deleted", variant: "secondary" },
  [VPSStatus.SUSPENDED]: { label: "Suspended", variant: "warning" },
};

// Dialog states
const resizeDialogOpen = ref(false);
const suspendDialogOpen = ref(false);
const cloudInitDialogOpen = ref(false);
const forceStopDialogOpen = ref(false);
const forceDeleteDialogOpen = ref(false);

// Form states
const isResizing = ref(false);
const isSuspending = ref(false);
const isUnsuspending = ref(false);
const isUpdatingCloudInit = ref(false);
const isForceStopping = ref(false);
const isForceDeleting = ref(false);
const loadingSizes = ref(false);
const sizeOptions = ref<any[]>([]);

const resizeForm = ref<{
  newSize: string;
  growDisk: boolean;
  applyCloudInit: boolean;
  customCpuCores: string;
  customMemoryGB: string;
  customDiskGB: string;
}>({
  newSize: "",
  growDisk: true,
  applyCloudInit: true,
  customCpuCores: "1",
  customMemoryGB: "1",
  customDiskGB: "10",
});

const suspendForm = ref({
  reason: "",
});

const cloudInitForm = ref({
  userData: "",
});

function formatStatus(status: number | undefined): string {
  if (status === undefined) return "Unknown";
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
    case VPSStatus.SUSPENDED:
      return "Suspended";
    default:
      return "Unknown";
  }
}

function formatMemory(bytes: bigint | number | undefined): string {
  if (!bytes) return "0 GB";
  const gb = Number(bytes) / (1024 * 1024 * 1024);
  return `${gb.toFixed(1)} GB`;
}

function formatDisk(bytes: bigint | number | undefined): string {
  if (!bytes) return "0 GB";
  const gb = Number(bytes) / (1024 * 1024 * 1024);
  return `${gb.toFixed(0)} GB`;
}

function formatImage(image: number | undefined, imageId?: string | null): string {
  if (imageId) return imageId;
  if (image === undefined || image === null) return "Unknown";
  switch (image) {
    case VPSImage.UBUNTU_22_04:
      return "Ubuntu 22.04";
    case VPSImage.UBUNTU_24_04:
      return "Ubuntu 24.04";
    case VPSImage.DEBIAN_12:
      return "Debian 12";
    case VPSImage.CUSTOM:
      return "Custom";
    default:
      return "Unknown";
  }
}

const parsedMetadata = computed(() => {
  if (!vps.value?.metadata) return {};
  try {
    const meta = typeof vps.value.metadata === "string" 
      ? JSON.parse(vps.value.metadata) 
      : vps.value.metadata;
    return meta && typeof meta === "object" ? meta : {};
  } catch {
    return {};
  }
});

const hasMetadata = computed(() => {
  const meta = parsedMetadata.value;
  return meta && typeof meta === "object" && Object.keys(meta).length > 0;
});

async function loadVPS() {
  if (!vpsId.value) return null;
  try {
    const response = await client.superadminGetVPS({
      vpsId: vpsId.value,
    });
    return {
      vps: response.vps,
      organizationName: response.organizationName,
      createdBy: response.createdBy,
    };
  } catch (error: any) {
    console.error("Failed to load VPS:", error);
    toast.error(error?.message || "Failed to load VPS");
    throw error;
  }
}

// Use client-side fetching for non-blocking navigation
const { data: vpsData, pending: loading } = useClientFetch(
  () => `superadmin-vps-${vpsId.value}`,
  loadVPS
);

// Update refs when data is loaded
watch(vpsData, (newData) => {
  if (newData) {
    vps.value = newData.vps;
    organizationName.value = newData.organizationName;
    createdBy.value = newData.createdBy;
  }
}, { immediate: true });

// Dialog handlers
function openResizeDialog() {
  resizeForm.value.newSize = "";
  resizeForm.value.growDisk = true;
  resizeForm.value.applyCloudInit = true;
  // Initialize custom values with current VPS values
  if (vps.value) {
    resizeForm.value.customCpuCores = String(vps.value.cpuCores || 1);
    resizeForm.value.customMemoryGB = String(Math.round((Number(vps.value.memoryBytes || 0) / (1024 * 1024 * 1024)) * 10) / 10);
    resizeForm.value.customDiskGB = String(Math.round(Number(vps.value.diskBytes || 0) / (1024 * 1024 * 1024)));
  } else {
    resizeForm.value.customCpuCores = "1";
    resizeForm.value.customMemoryGB = "1";
    resizeForm.value.customDiskGB = "10";
  }
  loadSizes();
  resizeDialogOpen.value = true;
}

function openSuspendDialog() {
  suspendForm.value.reason = "";
  suspendDialogOpen.value = true;
}

function openCloudInitDialog() {
  cloudInitForm.value.userData = "";
  // TODO: Load existing cloud-init config if available
  cloudInitDialogOpen.value = true;
}

function openForceStopDialog() {
  forceStopDialogOpen.value = true;
}

function openForceDeleteDialog() {
  forceDeleteDialogOpen.value = true;
}

async function loadSizes() {
  loadingSizes.value = true;
  try {
    const response = await client.listVPSSizes({});
    sizeOptions.value = (response.sizes || []).map((size) => ({
      label: size.name,
      value: size.id,
      cpuCores: size.cpuCores,
      memoryBytes: size.memoryBytes,
      diskBytes: size.diskBytes,
    }));
  } catch (error: any) {
    toast.error(`Failed to load sizes: ${error?.message || "Unknown error"}`);
  } finally {
    loadingSizes.value = false;
  }
}

const sizeOptionsWithCustom = computed(() => {
  const options = [...sizeOptions.value];
  options.push({
    label: "Custom",
    value: "custom",
    cpuCores: 0,
    memoryBytes: 0,
    diskBytes: 0,
  });
  return options;
});

const canResize = computed(() => {
  if (!resizeForm.value.newSize) return false;
  if (resizeForm.value.newSize === "custom") {
    const cpuCores = Number(resizeForm.value.customCpuCores);
    const memoryGB = Number(resizeForm.value.customMemoryGB);
    const diskGB = Number(resizeForm.value.customDiskGB);
    return (
      !isNaN(cpuCores) && cpuCores > 0 &&
      !isNaN(memoryGB) && memoryGB > 0 &&
      !isNaN(diskGB) && diskGB > 0
    );
  }
  return true;
});

async function handleResize() {
  if (!vps.value || !resizeForm.value.newSize) return;
  isResizing.value = true;
  try {
    const request: any = {
      vpsId: vps.value.id,
      newSize: resizeForm.value.newSize,
      growDisk: resizeForm.value.growDisk,
      applyCloudinit: resizeForm.value.applyCloudInit,
    };

    // Add custom size parameters if using custom size
    if (resizeForm.value.newSize === "custom") {
      const cpuCores = Number(resizeForm.value.customCpuCores);
      const memoryGB = Number(resizeForm.value.customMemoryGB);
      const diskGB = Number(resizeForm.value.customDiskGB);
      
      if (isNaN(cpuCores) || cpuCores <= 0) {
        toast.error("CPU cores must be a positive number");
        return;
      }
      if (isNaN(memoryGB) || memoryGB <= 0) {
        toast.error("Memory must be a positive number");
        return;
      }
      if (isNaN(diskGB) || diskGB <= 0) {
        toast.error("Disk size must be a positive number");
        return;
      }
      
      request.customCpuCores = cpuCores;
      request.customMemoryBytes = BigInt(Math.round(memoryGB * 1024 * 1024 * 1024));
      request.customDiskBytes = BigInt(Math.round(diskGB * 1024 * 1024 * 1024));
    }

    await client.superadminResizeVPS(request);
    toast.success("VPS resized successfully");
    resizeDialogOpen.value = false;
    await loadVPS(); // Refresh VPS data
  } catch (error: any) {
    toast.error(`Failed to resize VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isResizing.value = false;
  }
}

async function handleSuspend() {
  if (!vps.value) return;
  isSuspending.value = true;
  try {
    await client.superadminSuspendVPS({
      vpsId: vps.value.id,
      reason: suspendForm.value.reason || undefined,
    });
    toast.success("VPS suspended successfully");
    suspendDialogOpen.value = false;
    await loadVPS(); // Refresh VPS data
  } catch (error: any) {
    toast.error(`Failed to suspend VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isSuspending.value = false;
  }
}

async function handleUnsuspend() {
  if (!vps.value) return;
  isUnsuspending.value = true;
  try {
    await client.superadminUnsuspendVPS({
      vpsId: vps.value.id,
    });
    toast.success("VPS unsuspended successfully");
    await loadVPS(); // Refresh VPS data
  } catch (error: any) {
    toast.error(`Failed to unsuspend VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isUnsuspending.value = false;
  }
}

async function handleUpdateCloudInit() {
  if (!vps.value) return;
  isUpdatingCloudInit.value = true;
  try {
    // TODO: Parse YAML and convert to CloudInitConfig proto
    // For now, show an error that this needs proper YAML parsing
    toast.error("CloudInit update requires proper YAML parsing. This feature is not yet fully implemented.");
    cloudInitDialogOpen.value = false;
  } catch (error: any) {
    toast.error(`Failed to update CloudInit: ${error?.message || "Unknown error"}`);
  } finally {
    isUpdatingCloudInit.value = false;
  }
}

async function handleForceStop() {
  if (!vps.value) return;
  isForceStopping.value = true;
  try {
    await client.superadminForceStopVPS({
      vpsId: vps.value.id,
    });
    toast.success("VPS force stopped successfully");
    forceStopDialogOpen.value = false;
    await loadVPS(); // Refresh VPS data
  } catch (error: any) {
    toast.error(`Failed to force stop VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isForceStopping.value = false;
  }
}

async function handleForceDelete() {
  if (!vps.value) return;
  isForceDeleting.value = true;
  try {
    await client.superadminForceDeleteVPS({
      vpsId: vps.value.id,
    });
    toast.success("VPS force deleted successfully");
    forceDeleteDialogOpen.value = false;
    router.push("/superadmin/vps");
  } catch (error: any) {
    toast.error(`Failed to force delete VPS: ${error?.message || "Unknown error"}`);
  } finally {
    isForceDeleting.value = false;
  }
}

function goToVPSPage() {
  if (!vps.value?.id) return;
  // Navigate to the regular VPS page
  // The regular VPS page will automatically switch to the correct organization if needed
  router.push(`/vps/${vps.value.id}`);
}

await loadVPS();
</script>

