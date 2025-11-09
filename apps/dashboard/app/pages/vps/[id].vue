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
                          <OuiText size="sm" weight="semibold" color="primary">SSH Proxy</OuiText>
                          <OuiText size="sm" color="secondary">
                            Connect via SSH through a jump host proxy. Connection instructions will be available
                            once the VPS is fully provisioned.
                          </OuiText>
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
import { computed, ref } from "vue";
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
  { id: "audit-logs", label: "Audit Logs", icon: ClipboardDocumentListIcon },
]);

// Use composable for tab query parameter management
const activeTab = useTabQuery(tabs, "overview");

function openTerminal() {
  activeTab.value = "terminal";
}
</script>

