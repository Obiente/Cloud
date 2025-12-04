<template>
  <OuiDialog
    :open="modelValue"
    :title="props.initialValues ? 'Retry VPS Creation' : 'Create VPS Instance'"
    :description="props.initialValues ? 'Retry creating this VPS instance with the same settings.' : 'Provision a new virtual private server with full root access.'"
    @update:open="updateOpen"
  >
    <OuiStack gap="lg">
      <!-- Error Alert -->
      <ErrorAlert v-if="error" :error="error" title="Failed to create VPS" />

      <!-- Form -->
      <OuiStack gap="md">
        <OuiInput
          v-model="form.name"
          label="Name"
          placeholder="my-vps"
          required
          :error="errors.name"
        />

        <OuiTextarea
          v-model="form.description"
          label="Description"
          placeholder="Optional description"
          :rows="2"
        />

        <OuiSelect
          v-model="form.region"
          label="Region"
          :items="regionOptions"
          required
          :error="errors.region"
          :loading="loadingRegions"
        />

        <OuiSelect
          v-model="form.image"
          label="Operating System"
          :items="imageOptions"
          required
          :error="errors.image"
        />

        <OuiSelect
          v-model="form.size"
          label="Instance Size"
          description="Select the resource limits for your VPS. You'll be charged based on actual usage."
          :items="sizeOptions"
          required
          :error="errors.size"
          :loading="loadingSizes"
        >
          <template #item="{ item }">
            <OuiStack gap="xs">
              <OuiText weight="medium">{{ item.label }}</OuiText>
              <OuiText size="xs" color="secondary">
                {{ item.description }}
              </OuiText>
              <OuiText size="xs" color="secondary">
                {{ item.cpuCores }} CPU ·
                {{ formatMemory(item.memoryBytes) }} RAM ·
                {{ formatDisk(item.diskBytes) }} Storage
              </OuiText>
            </OuiStack>
          </template>
        </OuiSelect>
      </OuiStack>

      <!-- Advanced Configuration (Cloud-Init) -->
      <OuiCollapsible
        v-model="showAdvancedConfig"
        label="Advanced Configuration (Cloud-Init)"
      >
        <OuiStack gap="md" class="pt-2">
          <!-- Custom Root Password -->
          <OuiInput
            v-model="form.rootPassword"
            type="password"
            label="Custom Root Password"
            description="Optional: Set a custom root password. If not set, a random password will be generated."
            placeholder="Leave empty for auto-generated password"
          />

          <!-- System Configuration -->
          <OuiStack gap="sm">
            <OuiText size="sm" weight="semibold">System Configuration</OuiText>
            <OuiInput
              v-model="form.cloudInit.hostname"
              label="Hostname"
              placeholder="my-vps"
              description="System hostname (optional)"
            />
            <OuiInput
              v-model="form.cloudInit.timezone"
              label="Timezone"
              placeholder="America/New_York"
              description="Timezone (e.g., UTC, America/New_York, Europe/London)"
            />
            <OuiInput
              v-model="form.cloudInit.locale"
              label="Locale"
              placeholder="en_US.UTF-8"
              description="System locale (e.g., en_US.UTF-8)"
            />
          </OuiStack>

          <!-- Users -->
          <OuiStack gap="sm">
            <OuiFlex justify="between" align="center">
              <OuiText size="sm" weight="semibold">Users</OuiText>
              <OuiButton
                variant="outline"
                size="xs"
                @click="addUser"
                class="gap-1"
              >
                <PlusIcon class="h-4 w-4" />
                Add User
              </OuiButton>
            </OuiFlex>
            <OuiText size="xs" color="secondary">
              Create additional users with custom passwords and SSH keys
            </OuiText>
            <OuiStack gap="sm" v-if="form.cloudInit.users.length > 0">
              <OuiCard
                v-for="(user, index) in form.cloudInit.users"
                :key="index"
                variant="outline"
              >
                <OuiCardBody>
                  <OuiStack gap="sm">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" weight="medium">User {{ index + 1 }}</OuiText>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        color="danger"
                        @click="removeUser(index)"
                      >
                        Remove
                      </OuiButton>
                    </OuiFlex>
                    <OuiInput
                      v-model="user.name"
                      label="Username"
                      placeholder="username"
                      required
                    />
                    <OuiInput
                      v-model="user.password"
                      type="password"
                      label="Password"
                      placeholder="Leave empty for SSH key only"
                    />
                    <!-- SSH Keys Selection -->
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
                          No SSH keys available. Add SSH keys in your organization settings.
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
                          :model-value="user.selectedSSHKeyIds?.includes(key.id) || false"
                          @update:model-value="(checked) => toggleSSHKey(user, key.id, checked)"
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
                    <OuiFlex gap="sm">
                      <OuiCheckbox
                        v-model="user.sudo"
                        label="Grant sudo access"
                      />
                      <OuiCheckbox
                        v-model="user.sudoNopasswd"
                        label="Sudo without password"
                        :disabled="!user.sudo"
                      />
                    </OuiFlex>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>
          </OuiStack>

          <!-- Packages -->
          <OuiStack gap="sm">
            <OuiText size="sm" weight="semibold">Additional Packages</OuiText>
            <OuiTextarea
              v-model="form.cloudInit.packages"
              label="Packages to Install"
              placeholder="nginx&#10;docker.io&#10;git"
              description="One package name per line"
              :rows="3"
            />
            <OuiFlex gap="sm">
              <OuiCheckbox
                v-model="form.cloudInit.packageUpdate"
                label="Update package database"
              />
              <OuiCheckbox
                v-model="form.cloudInit.packageUpgrade"
                label="Upgrade packages"
              />
            </OuiFlex>
          </OuiStack>

          <!-- Custom Commands -->
          <OuiStack gap="sm">
            <OuiText size="sm" weight="semibold">Custom Commands</OuiText>
            <OuiTextarea
              v-model="form.cloudInit.runcmd"
              label="Commands to Run on First Boot"
              placeholder="echo 'Hello World' > /tmp/hello.txt&#10;systemctl enable my-service"
              description="One command per line. Commands run as root."
              :rows="3"
            />
          </OuiStack>

          <!-- SSH Configuration -->
          <OuiStack gap="sm">
            <OuiText size="sm" weight="semibold">SSH Configuration</OuiText>
            <OuiFlex gap="sm">
              <OuiCheckbox
                v-model="form.cloudInit.sshInstallServer"
                label="Install SSH server"
              />
              <OuiCheckbox
                v-model="form.cloudInit.sshAllowPw"
                label="Allow password authentication"
              />
            </OuiFlex>
          </OuiStack>
        </OuiStack>
      </OuiCollapsible>

      <!-- VPS Size Description -->
      <OuiCard
        variant="outline"
        v-if="selectedSize && selectedSize.catalogDescription"
      >
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText size="sm" weight="semibold">{{
              selectedSize.label
            }}</OuiText>
            <OuiText size="xs" color="secondary">
              {{ selectedSize.catalogDescription }}
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Summary -->
      <OuiCard variant="outline" v-if="selectedSize">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="sm" weight="semibold">Resource Limits</OuiText>
            <OuiText size="xs" color="secondary">
              These limits define the maximum resources your VPS can use. You'll
              be charged based on actual usage (pay-as-you-go).
            </OuiText>
            <OuiGrid cols="2" gap="sm">
              <OuiText size="xs" color="secondary">CPU Cores</OuiText>
              <OuiText size="xs">{{ selectedSize.cpuCores }}</OuiText>
              <OuiText size="xs" color="secondary">Memory</OuiText>
              <OuiText size="xs">{{
                formatMemory(selectedSize.memoryBytes)
              }}</OuiText>
              <OuiText size="xs" color="secondary">Storage</OuiText>
              <OuiText size="xs">{{
                formatDisk(selectedSize.diskBytes)
              }}</OuiText>
            </OuiGrid>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>

    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="updateOpen(false)">Cancel</OuiButton>
        <OuiButton
          color="primary"
          @click="handleCreate"
          :disabled="!isValid || isCreating"
        >
          {{ isCreating ? (props.initialValues ? "Retrying..." : "Creating...") : (props.initialValues ? "Retry VPS" : "Create VPS") }}
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>

  <!-- Password Display Dialog (One-time only) -->
  <OuiDialog
    :open="showPasswordDialog"
    title="VPS Created Successfully"
    description="Your VPS instance has been created. Please note down the root password below - it will not be shown again."
    @update:open="
      (val) => {
        if (!val) {
          showPasswordDialog = false;
          createdPassword = null;
          emit('created');
        }
      }
    "
    size="md"
  >
    <OuiStack gap="md">
      <OuiBox variant="warning" p="md" rounded="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" weight="semibold" color="warning">
            ⚠️ Important: Save This Password
          </OuiText>
          <OuiText size="xs" color="secondary">
            This password will only be shown once. If you lose it, you can reset
            it from the VPS settings, but you'll need to reboot the VPS for the
            new password to take effect.
          </OuiText>
        </OuiStack>
      </OuiBox>

      <OuiStack gap="xs">
        <OuiText size="sm" weight="medium">Root Password</OuiText>
        <OuiBox p="md" rounded="md" class="bg-surface-muted font-mono text-sm">
          <OuiFlex justify="between" align="center" gap="sm">
            <OuiText class="select-all">{{ createdPassword ?? "" }}</OuiText>
            <OuiButton
              variant="ghost"
              size="xs"
              @click="copyPassword"
              class="gap-1"
            >
              <ClipboardDocumentIcon class="h-4 w-4" />
              Copy
            </OuiButton>
          </OuiFlex>
        </OuiBox>
        <OuiText size="xs" color="secondary">
          Use this password to log in as root via SSH. We recommend using SSH
          keys instead for better security.
        </OuiText>
      </OuiStack>
    </OuiStack>

    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton
          color="primary"
          @click="
            () => {
              showPasswordDialog = false;
              createdPassword = null;
              emit('created');
              updateOpen(false);
            }
          "
        >
          I've Saved the Password
        </OuiButton>
      </OuiFlex>
    </template>
  </OuiDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch, nextTick } from "vue";
  import { VPSService, VPSImage } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import { useDialog } from "~/composables/useDialog";
  import { useToast } from "~/composables/useToast";
  import ErrorAlert from "~/components/ErrorAlert.vue";
  import { ClipboardDocumentIcon, PlusIcon, KeyIcon } from "@heroicons/vue/24/outline";

  interface Props {
    modelValue: boolean;
    initialValues?: {
      name?: string;
      description?: string;
      region?: string;
      image?: number;
      size?: string;
    } | null;
  }

  const props = withDefaults(defineProps<Props>(), {
    initialValues: null,
  });
  const emit = defineEmits<{
    "update:modelValue": [value: boolean];
    created: [];
  }>();

  const client = useConnectClient(VPSService);
  const organizationId = useOrganizationId();
  const { showAlert } = useDialog();
  const { toast } = useToast();

  const form = ref({
    name: "",
    description: "",
    region: "",
    image: VPSImage.UBUNTU_24_04,
    size: "",
    rootPassword: "",
    cloudInit: {
      hostname: "",
      timezone: "",
      locale: "",
      packages: "",
      packageUpdate: true,
      packageUpgrade: false,
      runcmd: "",
      sshInstallServer: true,
      sshAllowPw: true,
      users: [] as Array<{
        name: string;
        password: string;
        selectedSSHKeyIds: string[];
        sudo: boolean;
        sudoNopasswd: boolean;
      }>,
    },
  });

  const showAdvancedConfig = ref(false);

  const errors = ref<Record<string, string>>({});
  const error = ref<Error | null>(null);
  const isCreating = ref(false);
  const loadingRegions = ref(false);
  const loadingSizes = ref(false);
  const createdPassword = ref<string | null>(null);
  const showPasswordDialog = ref(false);

  const regionOptions = ref<Array<{ label: string; value: string }>>([]);
  const sizeOptions = ref<
    Array<{
      label: string;
      value: string;
      description: string;
      catalogDescription?: string;
      minimumPaymentCents: bigint | number;
      cpuCores: number;
      memoryBytes: bigint | number;
      diskBytes: bigint | number;
    }>
  >([]);

  const imageOptions = [
    { label: "Ubuntu 24.04 LTS", value: String(VPSImage.UBUNTU_24_04) },
    { label: "Ubuntu 22.04 LTS", value: String(VPSImage.UBUNTU_22_04) },
    { label: "Debian 13", value: String(VPSImage.DEBIAN_13) },
    { label: "Debian 12", value: String(VPSImage.DEBIAN_12) },
    { label: "Rocky Linux 9", value: String(VPSImage.ROCKY_LINUX_9) },
    { label: "AlmaLinux 9", value: String(VPSImage.ALMA_LINUX_9) },
  ];

  const selectedSize = computed(() => {
    if (!form.value.size) return null;
    return sizeOptions.value.find((s) => s.value === form.value.size);
  });

  const isValid = computed(() => {
    return (
      form.value.name.trim() !== "" &&
      form.value.region !== "" &&
      form.value.size !== ""
    );
  });

  const formatMemory = (bytes: bigint | number | undefined) => {
    if (!bytes) return "0 GB";
    const gb = Number(bytes) / (1024 * 1024 * 1024);
    return `${gb.toFixed(1)} GB`;
  };

  const formatDisk = (bytes: bigint | number | undefined) => {
    if (!bytes) return "0 GB";
    const gb = Number(bytes) / (1024 * 1024 * 1024);
    return `${gb.toFixed(0)} GB`;
  };

  const addUser = () => {
    form.value.cloudInit.users.push({
      name: "",
      password: "",
      selectedSSHKeyIds: [],
      sudo: false,
      sudoNopasswd: false,
    });
  };

  const toggleSSHKey = (
    user: {
      selectedSSHKeyIds: string[];
    },
    keyId: string,
    checked: boolean
  ) => {
    if (checked) {
      if (!user.selectedSSHKeyIds.includes(keyId)) {
        user.selectedSSHKeyIds.push(keyId);
      }
    } else {
      const index = user.selectedSSHKeyIds.indexOf(keyId);
      if (index > -1) {
        user.selectedSSHKeyIds.splice(index, 1);
      }
    }
  };

  const removeUser = (index: number) => {
    form.value.cloudInit.users.splice(index, 1);
  };

  const updateOpen = (value: boolean) => {
    emit("update:modelValue", value);
    if (!value) {
      // Reset form when closing (unless we have initial values for retry)
      if (!props.initialValues) {
        form.value = {
          name: "",
          description: "",
          region: "",
          image: VPSImage.UBUNTU_24_04,
          size: "",
          rootPassword: "",
          cloudInit: {
            hostname: "",
            timezone: "",
            locale: "",
            packages: "",
            packageUpdate: true,
            packageUpgrade: false,
            runcmd: "",
            sshInstallServer: true,
            sshAllowPw: true,
            users: [],
          },
        };
      }
      errors.value = {};
      error.value = null;
      // Don't clear password if we're about to show the password dialog
      // The password dialog's close handler will clear it
      if (!showPasswordDialog.value) {
      createdPassword.value = null;
      }
      showAdvancedConfig.value = false;
    }
  };

  // SSH Keys Management
  const availableSSHKeys = ref<Array<{
    id: string;
    name: string;
    publicKey: string;
    fingerprint: string;
    vpsId?: string;
  }>>([]);
  const sshKeysLoading = ref(false);

  const fetchSSHKeys = async () => {
    if (!organizationId.value) {
      availableSSHKeys.value = [];
      return;
    }

    sshKeysLoading.value = true;
    try {
      // Fetch organization-wide SSH keys (no vpsId filter since VPS doesn't exist yet)
      const res = await client.listSSHKeys({
        organizationId: organizationId.value,
      });
      availableSSHKeys.value = (res.keys || [])
        .filter((key) => !key.vpsId) // Only show org-wide keys for new VPS
        .map((key) => ({
          id: key.id || "",
          name: key.name || "",
          publicKey: key.publicKey || "",
          fingerprint: key.fingerprint || "",
          vpsId: key.vpsId,
        }));
    } catch (err: unknown) {
      console.error("Failed to fetch SSH keys:", err);
      availableSSHKeys.value = [];
    } finally {
      sshKeysLoading.value = false;
    }
  };

  // Load data when dialog opens
  watch(
    () => props.modelValue,
    async (isOpen) => {
      if (isOpen) {
        // Pre-fill form with initial values if provided (for retry)
        if (props.initialValues) {
          if (props.initialValues.name) {
            form.value.name = props.initialValues.name;
          }
          if (props.initialValues.description !== undefined) {
            form.value.description = props.initialValues.description;
          }
          if (props.initialValues.region) {
            form.value.region = props.initialValues.region;
          }
          if (props.initialValues.image !== undefined) {
            form.value.image = props.initialValues.image as VPSImage;
          }
          if (props.initialValues.size) {
            form.value.size = props.initialValues.size;
          }
        }

        await Promise.all([
          fetchSSHKeys(),
          loadRegions(),
          loadSizes(),
        ]);
      }
    }
  );

  const loadRegions = async () => {
    loadingRegions.value = true;
    try {
      const response = await client.listVPSRegions({});
      const availableRegions = (response.regions || []).filter(
        (r) => r.available
      );

      if (availableRegions.length > 0) {
        regionOptions.value = availableRegions.map((r) => ({
          label: r.name,
          value: r.id,
        }));
        // Auto-select first region if none selected
        if (
          !form.value.region &&
          availableRegions.length === 1 &&
          availableRegions[0]
        ) {
          form.value.region = availableRegions[0].id;
        }
      } else {
        // No regions available
        regionOptions.value = [];
      }
    } catch (err) {
      console.error("Failed to load regions:", err);
      // Show error but don't default
      regionOptions.value = [];
      error.value =
        err instanceof Error
          ? err
          : new Error(
              "Failed to load VPS regions. Please ensure VPS_REGIONS environment variable is configured."
            );
    } finally {
      loadingRegions.value = false;
    }
  };

  const loadSizes = async () => {
    loadingSizes.value = true;
    try {
      const response = await client.listVPSSizes({
        region: form.value.region || undefined,
      });
      sizeOptions.value = (response.sizes || [])
        .filter((s) => s.available)
        .map((s) => ({
          label: s.name,
          value: s.id,
          description: `${s.cpuCores} CPU • ${formatMemory(
            s.memoryBytes
          )} RAM • ${formatDisk(s.diskBytes)} Storage`,
          catalogDescription: s.description || undefined,
          minimumPaymentCents: s.minimumPaymentCents || 0,
          cpuCores: s.cpuCores,
          memoryBytes: s.memoryBytes,
          diskBytes: s.diskBytes,
        }));
    } catch (err) {
      console.error("Failed to load sizes:", err);
    } finally {
      loadingSizes.value = false;
    }
  };

  // Reload sizes when region changes
  watch(
    () => form.value.region,
    () => {
      if (form.value.region) {
        loadSizes();
        form.value.size = ""; // Reset size selection
      }
    }
  );

  const handleCreate = async () => {
    errors.value = {};
    error.value = null;

    // Validate
    if (!form.value.name.trim()) {
      errors.value.name = "Name is required";
      return;
    }
    if (!form.value.region) {
      errors.value.region = "Region is required";
      return;
    }
    if (!form.value.size) {
      errors.value.size = "Instance size is required";
      return;
    }

    isCreating.value = true;
    try {
      // Convert image string back to number (enum value)
      const imageValue =
        typeof form.value.image === "string"
          ? Number(form.value.image)
          : form.value.image;

      // Build cloud-init config if advanced options are used
      let cloudInitConfig = undefined;
      const hasCloudInitConfig =
        showAdvancedConfig.value &&
        (form.value.cloudInit.hostname ||
          form.value.cloudInit.timezone ||
          form.value.cloudInit.locale ||
          form.value.cloudInit.packages ||
          form.value.cloudInit.runcmd ||
          form.value.cloudInit.users.length > 0 ||
          !form.value.cloudInit.packageUpdate ||
          form.value.cloudInit.packageUpgrade ||
          !form.value.cloudInit.sshInstallServer ||
          !form.value.cloudInit.sshAllowPw);

      if (hasCloudInitConfig) {
        // Parse packages (one per line)
        const packages = form.value.cloudInit.packages
          .split("\n")
          .map((p) => p.trim())
          .filter((p) => p.length > 0);

        // Parse runcmd (one per line)
        const runcmd = form.value.cloudInit.runcmd
          .split("\n")
          .map((c) => c.trim())
          .filter((c) => c.length > 0);

        // Convert users
        const users = form.value.cloudInit.users
          .filter((u) => u.name.trim() !== "")
          .map((u) => {
            // Convert selected SSH key IDs to their public keys
            const sshAuthorizedKeys = (u.selectedSSHKeyIds || [])
              .map((keyId) => {
                const key = availableSSHKeys.value.find((k) => k.id === keyId);
                return key?.publicKey;
              })
              .filter((key): key is string => !!key); // Filter out undefined values

            return {
              name: u.name.trim(),
              password: u.password.trim() || undefined,
              sshAuthorizedKeys: sshAuthorizedKeys.length > 0 ? sshAuthorizedKeys : undefined,
              sudo: u.sudo || undefined,
              sudoNopasswd: u.sudoNopasswd || undefined,
            };
          });

        cloudInitConfig = {
          users: users,
          hostname: form.value.cloudInit.hostname.trim() || undefined,
          timezone: form.value.cloudInit.timezone.trim() || undefined,
          locale: form.value.cloudInit.locale.trim() || undefined,
          packages: packages.length > 0 ? packages : undefined,
          packageUpdate: form.value.cloudInit.packageUpdate,
          packageUpgrade: form.value.cloudInit.packageUpgrade,
          runcmd: runcmd.length > 0 ? runcmd : undefined,
          sshInstallServer: form.value.cloudInit.sshInstallServer,
          sshAllowPw: form.value.cloudInit.sshAllowPw,
        };
      }

      // Start VPS creation (async - don't wait for completion)
      // The backend creates the VPS record immediately with CREATING status,
      // so we can close the dialog and refresh the list right away
      const createPromise = client.createVPS({
        organizationId: organizationId.value || "",
        name: form.value.name.trim(),
        description: form.value.description.trim() || undefined,
        region: form.value.region,
        image: imageValue as VPSImage,
        size: form.value.size,
        ...(form.value.rootPassword.trim() && {
          rootPassword: form.value.rootPassword.trim(),
        }),
        ...(cloudInitConfig && { cloudInit: cloudInitConfig }),
      });

      // Capture VPS name before closing dialog (form gets reset on close)
      const vpsName = form.value.name.trim();
      
      // Close dialog and refresh list immediately so user can see progress
      emit("created");
      updateOpen(false);
      isCreating.value = false;
      
      // Show toast notification that VPS creation has started
      toast.success("VPS creation started", `Your VPS instance "${vpsName}" is now provisioning. You can track the progress in the VPS list.`);

      // Handle password display asynchronously (don't block UI)
      createPromise
        .then((response) => {
          // Capture password if provided (one-time only)
          const vps = response.vps;
          
          // Use a replacer to handle BigInt values (convert to string) for JSON serialization
          const bigIntReplacer = (key: string, value: any) => {
            if (typeof value === 'bigint') {
              return value.toString();
            }
            return value;
          };
          
          // Try multiple ways to access the password (defensive approach)
          let password: string | undefined = undefined;
          
          // Method 1: Direct access (camelCase - TypeScript convention)
          if (vps?.rootPassword !== undefined && vps?.rootPassword !== null) {
            const pwd = vps.rootPassword;
            if (typeof pwd === "string" && pwd.trim() !== "") {
              password = pwd.trim();
            } else if (pwd) {
              password = String(pwd).trim();
            }
          }
          
          // Method 2: Check snake_case (protobuf field name)
          if (!password && vps && (vps as any).root_password !== undefined && (vps as any).root_password !== null) {
            const pwd = (vps as any).root_password;
            if (typeof pwd === "string" && pwd.trim() !== "") {
              password = pwd.trim();
            } else if (pwd) {
              password = String(pwd).trim();
            }
          }
          
          // Method 3: Check if it's a getter method (protobuf optional fields)
          if (!password && vps && typeof (vps as any).getRootPassword === "function") {
            try {
              const getterResult = (vps as any).getRootPassword();
              if (getterResult && typeof getterResult === "string" && getterResult.trim() !== "") {
                password = getterResult.trim();
              }
            } catch (e) {
              // Ignore getter errors
            }
          }
          
          // Method 4: Deep search in response (last resort)
          if (!password) {
            const responseStr = JSON.stringify(response, bigIntReplacer);
            const patterns = [
              /"rootPassword"\s*:\s*"([^"]+)"/,
              /"root_password"\s*:\s*"([^"]+)"/,
              /"root[_-]?password"\s*:\s*"([^"]+)"/i,
            ];
            
            for (const pattern of patterns) {
              const match = responseStr.match(pattern);
              if (match && match[1]) {
                password = match[1].trim();
                break;
              }
            }
          }
          
          // Show password dialog if password was returned
          if (password && password.length > 0) {
            createdPassword.value = password;
            showPasswordDialog.value = true;
          }
        })
        .catch((err) => {
          // Handle errors that occur after dialog is closed
          error.value =
            err instanceof Error ? err : new Error("Failed to create VPS");
          
          // Check if this is a timeout/context cancellation error
          // In this case, the VPS might still have been created successfully
          const isTimeoutError = 
            err instanceof Error && (
              err.message.includes("context canceled") ||
              err.message.includes("timeout") ||
              err.message.includes("NetworkError") ||
              err.message.includes("Failed to fetch")
            );
          
          if (isTimeoutError) {
            // VPS might have been created despite the error
            // Show a warning but don't treat it as a complete failure
            toast.warning(
              "VPS creation may have completed",
              "The request timed out, but your VPS may have been created. Please check the VPS list."
            );
          } else {
            // Real error - show alert
            showAlert({
              title: "Failed to create VPS",
              message: error.value.message,
            });
          }
        });
    } catch (err) {
      error.value =
        err instanceof Error ? err : new Error("Failed to create VPS");
      await showAlert({
        title: "Failed to create VPS",
        message: error.value.message,
      });
    } finally {
      isCreating.value = false;
    }
  };

  const copyPassword = async () => {
    if (!createdPassword.value) return;
    try {
      await navigator.clipboard.writeText(createdPassword.value);
      toast.success("Password copied to clipboard");
    } catch (err) {
      toast.error("Failed to copy password");
    }
  };
</script>
