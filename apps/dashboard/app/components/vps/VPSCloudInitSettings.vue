<template>
  <OuiStack gap="lg">
    <OuiCard>
      <OuiCardHeader>
        <OuiStack gap="xs">
          <OuiText as="h2" class="oui-card-title">Cloud-Init Configuration</OuiText>
          <OuiText color="secondary" size="sm">
            View and edit cloud-init settings for this VPS instance. Changes will be applied on the next reboot.
          </OuiText>
        </OuiStack>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack v-if="loading" gap="lg" class="py-4">
          <!-- System Configuration Skeleton -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">System Configuration</OuiText>
            <OuiGrid cols="1" cols-md="2" gap="md">
              <OuiSkeleton width="100%" height="3.5rem" variant="rectangle" rounded />
              <OuiSkeleton width="100%" height="3.5rem" variant="rectangle" rounded />
              <OuiSkeleton width="100%" height="3.5rem" variant="rectangle" rounded />
            </OuiGrid>
          </OuiStack>

          <!-- Package Management Skeleton -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Package Management</OuiText>
            <OuiSkeleton width="100%" height="6rem" variant="rectangle" rounded />
            <OuiFlex gap="sm">
              <OuiSkeleton width="8rem" height="1.5rem" variant="rectangle" rounded />
              <OuiSkeleton width="8rem" height="1.5rem" variant="rectangle" rounded />
            </OuiFlex>
          </OuiStack>

          <!-- Custom Commands Skeleton -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Custom Commands</OuiText>
            <OuiSkeleton width="100%" height="6rem" variant="rectangle" rounded />
          </OuiStack>

          <!-- SSH Configuration Skeleton -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">SSH Configuration</OuiText>
            <OuiFlex gap="sm">
              <OuiSkeleton width="8rem" height="1.5rem" variant="rectangle" rounded />
              <OuiSkeleton width="12rem" height="1.5rem" variant="rectangle" rounded />
            </OuiFlex>
          </OuiStack>

          <!-- Actions -->
          <OuiFlex justify="end" gap="sm">
            <OuiButton variant="outline" disabled>Reset</OuiButton>
            <OuiButton variant="solid" disabled>Saving...</OuiButton>
          </OuiFlex>
        </OuiStack>
        <div v-else-if="error" class="py-8 text-center">
          <OuiText color="danger">{{ error }}</OuiText>
          <OuiButton variant="outline" @click="loadConfig" class="mt-4 gap-2">
            <ArrowPathIcon class="h-4 w-4" />
            Retry
          </OuiButton>
        </div>
        <OuiStack v-else gap="lg">
          <!-- System Configuration -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">System Configuration</OuiText>
            <OuiGrid cols="1" cols-md="2" gap="md">
              <OuiInput
                v-model="config.hostname"
                label="Hostname"
                placeholder="my-vps"
                description="System hostname"
              />
              <OuiInput
                v-model="config.timezone"
                label="Timezone"
                placeholder="UTC"
                description="System timezone (e.g., UTC, America/New_York)"
              />
              <OuiInput
                v-model="config.locale"
                label="Locale"
                placeholder="en_US.UTF-8"
                description="System locale"
              />
            </OuiGrid>
          </OuiStack>

          <!-- Package Management -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Package Management</OuiText>
            <OuiTextarea
              v-model="config.packages"
              label="Packages to Install"
              placeholder="nginx&#10;docker.io&#10;git"
              description="One package name per line"
              :rows="4"
            />
            <OuiFlex gap="sm">
              <OuiCheckbox
                v-model="config.packageUpdate"
                label="Update package database"
              />
              <OuiCheckbox
                v-model="config.packageUpgrade"
                label="Upgrade packages"
              />
            </OuiFlex>
          </OuiStack>

          <!-- Custom Commands -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Custom Commands</OuiText>
            <OuiTextarea
              v-model="config.runcmd"
              label="Commands to Run on First Boot"
              placeholder="echo 'Hello World' > /tmp/hello.txt&#10;systemctl enable my-service"
              description="One command per line. Commands run as root."
              :rows="4"
            />
          </OuiStack>

          <!-- SSH Configuration -->
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">SSH Configuration</OuiText>
            <OuiFlex gap="sm">
              <OuiCheckbox
                v-model="config.sshInstallServer"
                label="Install SSH server"
              />
              <OuiCheckbox
                v-model="config.sshAllowPw"
                label="Allow password authentication"
              />
            </OuiFlex>
          </OuiStack>

          <!-- Raw YAML View -->
          <OuiStack gap="md">
            <OuiFlex justify="between" align="center">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="semibold">Raw Cloud-Init YAML</OuiText>
                <OuiText size="xs" color="secondary">
                  {{ showActualUserData ? "Current cloud-init config" : "Preview (from config only)" }}
                </OuiText>
              </OuiStack>
              <OuiFlex gap="xs">
                <OuiButton
                  variant="ghost"
                  size="xs"
                  @click="toggleUserDataView"
                  :disabled="loadingUserData"
                  class="gap-1"
                >
                  {{ showActualUserData ? "Show Preview" : "Show Current" }}
                </OuiButton>
                <OuiButton
                  variant="ghost"
                  size="xs"
                  @click="showRawYAML = !showRawYAML"
                  class="gap-1"
                >
                  {{ showRawYAML ? "Hide" : "Show" }} YAML
                </OuiButton>
              </OuiFlex>
            </OuiFlex>
            <OuiBox
              v-if="showRawYAML"
              p="md"
              rounded="md"
              class="bg-surface-muted font-mono text-xs overflow-x-auto"
            >
              <div v-if="loadingUserData && showActualUserData" class="py-4">
                <OuiStack gap="sm">
                  <OuiSkeleton width="100%" height="1rem" variant="text" />
                  <OuiSkeleton width="100%" height="1rem" variant="text" />
                  <OuiSkeleton width="90%" height="1rem" variant="text" />
                  <OuiSkeleton width="100%" height="1rem" variant="text" />
                  <OuiSkeleton width="85%" height="1rem" variant="text" />
                </OuiStack>
              </div>
              <div v-else-if="userDataError && showActualUserData" class="py-4">
                <OuiText color="danger" size="sm">
                  Failed to load actual userData: {{ userDataError }}
                </OuiText>
              </div>
              <pre v-else>{{ displayYAML }}</pre>
            </OuiBox>
          </OuiStack>

          <!-- Actions -->
          <OuiFlex justify="end" gap="sm">
            <OuiButton variant="outline" @click="resetConfig">Reset</OuiButton>
            <OuiButton
              variant="solid"
              @click="saveConfig"
              :disabled="saving"
            >
              {{ saving ? "Saving..." : "Save Configuration" }}
            </OuiButton>
          </OuiFlex>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiStack gap="xs">
          <OuiText size="sm" weight="semibold">Important Notes</OuiText>
          <OuiText size="xs" color="secondary">
            • Cloud-init configuration changes will only take effect after the VPS is rebooted.
          </OuiText>
          <OuiText size="xs" color="secondary">
            • User management should be done in the Users tab.
          </OuiText>
          <OuiText size="xs" color="secondary">
            • SSH keys are managed separately in the SSH Settings tab.
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { VPSConfigService, type VPSInstance, type CloudInitConfig, CloudInitConfigSchema } from "@obiente/proto";
import { create } from "@bufbuild/protobuf";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import OuiSkeleton from "~/components/oui/Skeleton.vue";

interface Props {
  vpsId: string;
  organizationId: string;
  vps: VPSInstance | null | undefined;
}

const props = defineProps<Props>();
const { toast } = useToast();
const client = useConnectClient(VPSConfigService);

const loading = ref(false);
const error = ref<string | null>(null);
const saving = ref(false);
const showRawYAML = ref(false);
const showActualUserData = ref(false);
const actualUserData = ref<string | null>(null);
const loadingUserData = ref(false);
const userDataError = ref<string | null>(null);

const config = ref({
  hostname: "",
  timezone: "",
  locale: "",
  packages: "",
  packageUpdate: true,
  packageUpgrade: false,
  runcmd: "",
  sshInstallServer: true,
  sshAllowPw: true,
});

const rawYAML = computed(() => {
  // Generate YAML representation of the config
  let yaml = "#cloud-config\n\n";
  
  if (config.value.hostname) {
    yaml += `hostname: ${config.value.hostname}\n`;
    yaml += `fqdn: ${config.value.hostname}\n\n`;
  }
  
  if (config.value.timezone) {
    yaml += `timezone: ${config.value.timezone}\n\n`;
  }
  
  if (config.value.locale) {
    yaml += `locale: ${config.value.locale}\n\n`;
  }
  
  yaml += "ssh:\n";
  yaml += `  install-server: ${config.value.sshInstallServer}\n`;
  yaml += `  allow-pw: ${config.value.sshAllowPw}\n\n`;
  
  yaml += `package_update: ${config.value.packageUpdate}\n`;
  yaml += `package_upgrade: ${config.value.packageUpgrade}\n`;
  
  if (config.value.packages.trim()) {
    const packages = config.value.packages
      .split("\n")
      .map((p) => p.trim())
      .filter((p) => p.length > 0);
    if (packages.length > 0) {
      yaml += "packages:\n";
      packages.forEach((pkg) => {
        yaml += `  - ${pkg}\n`;
      });
    }
  }
  
  if (config.value.runcmd.trim()) {
    const commands = config.value.runcmd
      .split("\n")
      .map((c) => c.trim())
      .filter((c) => c.length > 0);
    if (commands.length > 0) {
      yaml += "\nruncmd:\n";
      commands.forEach((cmd) => {
        yaml += `  - ${cmd}\n`;
      });
    }
  }
  
  return yaml;
});

const displayYAML = computed(() => {
  if (showActualUserData.value && actualUserData.value) {
    return actualUserData.value;
  }
  return rawYAML.value;
});

const toggleUserDataView = async () => {
  showActualUserData.value = !showActualUserData.value;
  
  // Fetch actual userData when switching to actual view
  if (showActualUserData.value && !actualUserData.value) {
    await fetchActualUserData();
  }
};

const fetchActualUserData = async () => {
  if (!props.vpsId || !props.organizationId) {
    return;
  }
  
  loadingUserData.value = true;
  userDataError.value = null;
  try {
    const res = await client.getCloudInitUserData({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
    });
    actualUserData.value = res.userData || "";
  } catch (err: unknown) {
    userDataError.value = err instanceof Error ? err.message : "Unknown error";
    actualUserData.value = null;
  } finally {
    loadingUserData.value = false;
  }
};

const loadConfig = async () => {
  loading.value = true;
  error.value = null;
  try {
    const res = await client.getCloudInitConfig({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
    });

    const cloudInit = res.cloudInit;
    if (cloudInit) {
      config.value = {
        hostname: cloudInit.hostname || "",
        timezone: cloudInit.timezone || "",
        locale: cloudInit.locale || "",
        packages: (cloudInit.packages || []).join("\n"),
        packageUpdate: cloudInit.packageUpdate ?? true,
        packageUpgrade: cloudInit.packageUpgrade ?? false,
        runcmd: (cloudInit.runcmd || []).join("\n"),
        sshInstallServer: cloudInit.sshInstallServer ?? true,
        sshAllowPw: cloudInit.sshAllowPw ?? true,
      };
    } else {
      // Default config
      config.value = {
        hostname: "",
        timezone: "",
        locale: "",
        packages: "",
        packageUpdate: true,
        packageUpgrade: false,
        runcmd: "",
        sshInstallServer: true,
        sshAllowPw: true,
      };
    }
  } catch (err: unknown) {
    error.value = err instanceof Error ? err.message : "Failed to load cloud-init configuration";
  } finally {
    loading.value = false;
  }
};

const saveConfig = async () => {
  saving.value = true;
  try {
    // First, get current config to preserve users
    const currentRes = await client.getCloudInitConfig({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
    });

    const packages = config.value.packages
      .split("\n")
      .map((p) => p.trim())
      .filter((p) => p.length > 0);

    const runcmd = config.value.runcmd
      .split("\n")
      .map((c) => c.trim())
      .filter((c) => c.length > 0);

    const cloudInitConfig = create(CloudInitConfigSchema, {
      hostname: config.value.hostname.trim() || undefined,
      timezone: config.value.timezone.trim() || undefined,
      locale: config.value.locale.trim() || undefined,
      packages: packages, // Always an array (empty if no packages)
      packageUpdate: config.value.packageUpdate,
      packageUpgrade: config.value.packageUpgrade,
      runcmd: runcmd, // Always an array (empty if no commands)
      sshInstallServer: config.value.sshInstallServer,
      sshAllowPw: config.value.sshAllowPw,
      // Preserve existing users - don't modify them in this tab
      users: currentRes.cloudInit?.users || [],
      writeFiles: currentRes.cloudInit?.writeFiles || [], // Preserve write_files as well
    });

    const res = await client.updateCloudInitConfig({
      organizationId: props.organizationId,
      vpsId: props.vpsId,
      cloudInit: cloudInitConfig,
    });

    toast.success("Configuration saved", res.message || "Changes will be applied on the next reboot.");
  } catch (err: unknown) {
    toast.error("Failed to save configuration", err instanceof Error ? err.message : "Unknown error");
  } finally {
    saving.value = false;
  }
};

const resetConfig = () => {
  loadConfig();
};

watch(() => props.vpsId, () => {
  if (props.vpsId) {
    loadConfig();
    // Reset userData view when VPS changes
    showActualUserData.value = false;
    actualUserData.value = null;
  }
}, { immediate: true });

watch(() => showActualUserData.value, (newValue) => {
  // Fetch actual userData when switching to actual view
  if (newValue && !actualUserData.value && props.vpsId) {
    fetchActualUserData();
  }
});

onMounted(() => {
  if (props.vpsId) {
    loadConfig();
  }
});
</script>

