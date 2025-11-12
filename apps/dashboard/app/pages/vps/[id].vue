<template>
  <OuiContainer size="full" py="sm" class="md:py-6">
    <OuiStack gap="md" class="md:gap-xl">
      <!-- Access Error State -->
      <OuiCard v-if="accessError" variant="outline" class="border-danger/20">
        <OuiCardBody>
          <OuiStack gap="lg" align="center">
            <ErrorAlert
              :error="accessError"
              title="Access Denied"
              :hint="errorHint"
            />
            <OuiButton
              variant="solid"
              color="primary"
              @click="router.push('/vps')"
            >
              Go to VPS Instances
            </OuiButton>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Loading State -->
      <OuiCard v-else-if="pending">
        <OuiCardBody>
          <OuiStack align="center" gap="md" class="py-16">
            <OuiSpinner size="lg" />
            <OuiText color="secondary">Loading VPS instance...</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Error State -->
      <OuiCard v-else-if="error" variant="outline" class="border-danger/20">
        <OuiCardBody>
          <OuiStack gap="lg" align="center">
            <ErrorAlert
              :error="error"
              title="Failed to load VPS instance"
              hint="Please try refreshing the page. If the problem persists, contact support."
            />
            <OuiButton @click="refreshVPS()" variant="outline" class="gap-2">
              <ArrowPathIcon class="h-4 w-4" />
              Try Again
            </OuiButton>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- VPS Content -->
      <template v-else-if="vps">
        <!-- Header -->
        <ResourceHeader
          :title="vps.name"
          :icon="ServerIcon"
        >
          <template #badges>
            <ResourceStatusBadge
              :label="statusLabel"
              :badge="statusBadgeColor"
              :dot-class="statusDotClass"
            />
            <OuiBadge
              v-if="vps.instanceId"
              variant="secondary"
              size="xs"
              class="md:size-sm"
            >
              <OuiText as="span" size="xs" weight="medium">
                VM ID: {{ vps.instanceId }}
              </OuiText>
            </OuiBadge>
          </template>
          <template #subtitle>
            <span v-if="vps.description">{{ vps.description }} • </span>
            <span>Last updated </span>
            <OuiRelativeTime
              :value="vps.updatedAt ? date(vps.updatedAt) : undefined"
              :style="'short'"
            />
          </template>
          <template #actions>
            <OuiButton
              variant="ghost"
              color="secondary"
              size="sm"
              @click="refreshVPS"
              :loading="isRefreshing"
              class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
            >
              <ArrowPathIcon
                class="h-4 w-4"
                :class="{ 'animate-spin': isRefreshing }"
              />
              <OuiText
                as="span"
                size="xs"
                weight="medium"
                class="hidden sm:inline"
              >
                Refresh
              </OuiText>
            </OuiButton>
            <OuiButton
              v-if="vps.status === VPSStatus.STOPPED"
              variant="solid"
              color="success"
              size="sm"
              @click="handleStart"
              :disabled="isActioning"
              class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
            >
              <PlayIcon class="h-4 w-4" />
              <OuiText
                as="span"
                size="xs"
                weight="medium"
                class="hidden sm:inline"
              >
                Start
              </OuiText>
            </OuiButton>
            <OuiButton
              v-if="vps.status === VPSStatus.RUNNING"
              variant="solid"
              color="danger"
              size="sm"
              @click="handleStop"
              :disabled="isActioning"
              class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
            >
              <StopIcon class="h-4 w-4" />
              <OuiText
                as="span"
                size="xs"
                weight="medium"
                class="hidden sm:inline"
              >
                Stop
              </OuiText>
            </OuiButton>
            <OuiButton
              v-if="vps.status === VPSStatus.RUNNING"
              variant="outline"
              color="secondary"
              size="sm"
              @click="handleReboot"
              :disabled="isActioning"
              class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
            >
              <ArrowPathIcon class="h-4 w-4" />
              <OuiText
                as="span"
                size="xs"
                weight="medium"
                class="hidden sm:inline"
              >
                Reboot
              </OuiText>
            </OuiButton>
            <OuiButton
              variant="outline"
              color="danger"
              size="sm"
              @click="handleDelete"
              :disabled="isActioning"
              class="gap-1.5 md:gap-2 flex-1 md:flex-initial"
            >
              <TrashIcon class="h-4 w-4" />
              <OuiText
                as="span"
                size="xs"
                weight="medium"
                class="hidden sm:inline"
              >
                Delete
              </OuiText>
            </OuiButton>
          </template>
        </ResourceHeader>

        <!-- Overview Cards -->
        <ResourceDetailsGrid cols="1" cols-md="2" cols-lg="4">
          <ResourceDetailCard
            label="CPU Cores"
            :icon="CpuChipIcon"
          >
            {{ vps.cpuCores || "N/A" }}
          </ResourceDetailCard>
          <ResourceDetailCard
            label="Memory"
            :icon="CircleStackIcon"
          >
            <OuiByte :value="vps.memoryBytes" />
          </ResourceDetailCard>
          <ResourceDetailCard
            label="Storage"
            :icon="ServerIcon"
          >
            <OuiByte :value="vps.diskBytes" />
          </ResourceDetailCard>
          <ResourceDetailCard
            label="Instance Size"
            :icon="CubeIcon"
          >
            {{ vps.size || "N/A" }}
          </ResourceDetailCard>
        </ResourceDetailsGrid>

        <!-- Tabbed Content -->
        <ResourceTabs ref="tabsRef" :tabs="tabs" default-tab="overview">
              <template #overview>
                <OuiStack gap="xl">
                  <!-- VPS Information Grid -->
                  <ResourceDetailsGrid cols="1" cols-md="2" cols-lg="2">
                    <!-- VPS Details Card -->
                    <OuiCard variant="default">
                      <OuiCardHeader>
                        <OuiText as="h2" class="oui-card-title">VPS Details</OuiText>
                      </OuiCardHeader>
                      <OuiCardBody>
                        <OuiStack gap="md">
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" color="secondary">Status</OuiText>
                            <ResourceStatusBadge
                              :label="statusLabel"
                              :badge="statusBadgeColor"
                              :dot-class="statusDotClass"
                            />
                          </OuiFlex>
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" color="secondary">Region</OuiText>
                            <OuiText size="sm" weight="medium">{{ vps.region || "—" }}</OuiText>
                          </OuiFlex>
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" color="secondary">Operating System</OuiText>
                            <OuiText size="sm" weight="medium">{{ imageLabel }}</OuiText>
                          </OuiFlex>
                          <OuiFlex justify="between" align="center" v-if="vps.instanceId">
                            <OuiText size="sm" color="secondary">VM ID</OuiText>
                            <OuiText size="sm" weight="medium" class="font-mono text-xs">{{ vps.instanceId }}</OuiText>
                          </OuiFlex>
                          <OuiFlex justify="between" align="center" v-if="vps.nodeId">
                            <OuiText size="sm" color="secondary">Node ID</OuiText>
                            <OuiText size="sm" weight="medium" class="font-mono text-xs">{{ vps.nodeId }}</OuiText>
                          </OuiFlex>
                          <OuiFlex justify="between" align="center">
                            <OuiText size="sm" color="secondary">Created</OuiText>
                            <OuiText size="sm" weight="medium">
                              <OuiRelativeTime
                                :value="vps.createdAt ? date(vps.createdAt) : undefined"
                                :style="'short'"
                              />
                            </OuiText>
                          </OuiFlex>
                          <OuiFlex justify="between" align="center" v-if="vps.lastStartedAt">
                            <OuiText size="sm" color="secondary">Last Started</OuiText>
                            <OuiText size="sm" weight="medium">
                              <OuiDate :value="vps.lastStartedAt" />
                            </OuiText>
                          </OuiFlex>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>

                    <!-- Network Information Card -->
                    <OuiCard variant="default">
                      <OuiCardHeader>
                        <OuiText as="h2" class="oui-card-title">Network Information</OuiText>
                      </OuiCardHeader>
                      <OuiCardBody>
                        <OuiStack gap="md">
                          <div v-if="vps.ipv4Addresses && vps.ipv4Addresses.length > 0">
                            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold" class="mb-2">
                              IPv4 Addresses
                            </OuiText>
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
                            <OuiText size="sm" color="secondary">No IPv4 addresses assigned</OuiText>
                          </div>
                          <div v-if="vps.ipv6Addresses && vps.ipv6Addresses.length > 0">
                            <OuiText size="xs" color="muted" transform="uppercase" weight="semibold" class="mb-2">
                              IPv6 Addresses
                            </OuiText>
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
                            <OuiText size="sm" color="secondary">No IPv6 addresses assigned</OuiText>
                          </div>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                  </ResourceDetailsGrid>

                  <!-- Connection Information -->
                  <OuiCard v-if="vps.status === VPSStatus.RUNNING" variant="default">
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
                              Connect to your VPS via SSH using the SSH proxy. SSH key authentication is tried first, then password authentication.
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

                </OuiStack>
              </template>
              <template #terminal>
                <VPSXTermTerminal
                  :vps-id="vpsId"
                  :organization-id="orgId"
                />
              </template>
              <template #firewall>
                <VPSFirewall
                  v-if="vps.instanceId"
                  :vps-id="vpsId"
                  :organization-id="orgId"
                />
                <OuiCard v-else variant="outline">
                  <OuiCardBody>
                    <OuiStack gap="md" align="center" class="py-8">
                      <ShieldExclamationIcon class="h-12 w-12 text-secondary" />
                      <OuiText color="secondary" size="sm">
                        Firewall settings are only available after the VPS is provisioned.
                      </OuiText>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </template>
              <template #users>
                <VPSUsersManagement
                  :vps-id="vpsId"
                  :organization-id="orgId"
                  :vps="vps"
                />
              </template>
              <template #cloud-init>
                <VPSCloudInitSettings
                  :vps-id="vpsId"
                  :organization-id="orgId"
                  :vps="vps"
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

                  <!-- Password Management -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">Root Password</OuiText>
                        <OuiText color="secondary" size="sm">
                          Reset the root password for this VPS instance. The new password will be shown once and must be saved immediately.
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiBox variant="info" p="sm" rounded="md">
                          <OuiStack gap="xs">
                            <OuiText size="xs" weight="semibold">Security Note</OuiText>
                            <OuiText size="xs" color="secondary">
                              SSH key authentication is preferred and tried first. Passwords are only used as a fallback. For better security, use SSH keys instead of passwords.
                            </OuiText>
                          </OuiStack>
                        </OuiBox>

                        <OuiButton
                          variant="outline"
                          color="warning"
                          @click="openResetPasswordDialog"
                          :disabled="resettingPassword || !vps.instanceId"
                          class="self-start gap-2"
                        >
                          <KeyIcon class="h-4 w-4" />
                          {{ resettingPassword ? "Resetting..." : "Reset Root Password" }}
                        </OuiButton>

                        <OuiText size="xs" color="secondary">
                          After resetting, you'll need to reboot the VPS for the new password to take effect. The password will only be shown once.
                        </OuiText>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Web Terminal Access -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">Web Terminal Access</OuiText>
                        <OuiText color="secondary" size="sm">
                          Manage the SSH key used for web terminal access. This key is automatically generated and allows secure terminal access without passwords.
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiBox variant="info" p="sm" rounded="md">
                          <OuiStack gap="xs">
                            <OuiText size="xs" weight="semibold">About Web Terminal Keys</OuiText>
                            <OuiText size="xs" color="secondary">
                              Each VPS has a dedicated SSH key pair for web terminal access. The public key is automatically added to the root user via cloud-init. You can rotate or remove this key at any time.
                            </OuiText>
                          </OuiStack>
                        </OuiBox>

                        <div v-if="terminalKeyLoading" class="py-4">
                          <OuiText color="secondary" class="text-center">Loading terminal key status...</OuiText>
                        </div>
                        <div v-else-if="terminalKeyError" class="py-4">
                          <OuiText color="danger" class="text-center">
                            Failed to load terminal key status: {{ terminalKeyError }}
                          </OuiText>
                        </div>
                        <div v-else-if="terminalKey" class="space-y-3">
                          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                            <OuiStack gap="sm">
                              <OuiFlex justify="between" align="start">
                                <OuiStack gap="xs">
                                  <OuiText size="sm" weight="semibold">Terminal Key Active</OuiText>
                                  <OuiText size="xs" color="secondary" class="font-mono">
                                    {{ terminalKey.fingerprint }}
                                  </OuiText>
                                  <OuiText size="xs" color="muted">
                                    Created {{ formatSSHKeyDate(terminalKey.createdAt) }}
                                  </OuiText>
                                </OuiStack>
                                <OuiBadge variant="success" size="sm">Active</OuiBadge>
                              </OuiFlex>
                            </OuiStack>
                          </OuiBox>

                          <OuiFlex gap="sm">
                            <OuiButton
                              variant="outline"
                              color="primary"
                              @click="rotateTerminalKey"
                              :disabled="rotatingTerminalKey || removingTerminalKey || !vps.instanceId"
                              class="flex-1"
                            >
                              <ArrowPathIcon class="h-4 w-4 mr-2" />
                              {{ rotatingTerminalKey ? "Rotating..." : "Rotate Key" }}
                            </OuiButton>
                            <OuiButton
                              variant="outline"
                              color="danger"
                              @click="openRemoveTerminalKeyDialog"
                              :disabled="rotatingTerminalKey || removingTerminalKey || !vps.instanceId"
                              class="flex-1"
                            >
                              <TrashIcon class="h-4 w-4 mr-2" />
                              {{ removingTerminalKey ? "Removing..." : "Remove Key" }}
                            </OuiButton>
                          </OuiFlex>

                          <OuiText size="xs" color="secondary">
                            Rotating the key generates a new key pair. Removing the key disables web terminal access until a new key is created (requires VPS reboot).
                          </OuiText>
                        </div>
                        <div v-else class="py-4">
                          <OuiStack gap="md">
                            <OuiText color="secondary" class="text-center">
                              No terminal key found. Web terminal access is not available.
                            </OuiText>
                            <OuiButton
                              variant="outline"
                              color="primary"
                              @click="rotateTerminalKey"
                              :disabled="rotatingTerminalKey || !vps.instanceId"
                              class="self-center"
                            >
                              <KeyIcon class="h-4 w-4 mr-2" />
                              {{ rotatingTerminalKey ? "Creating..." : "Create Terminal Key" }}
                            </OuiButton>
                          </OuiStack>
                        </div>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiStack>

                <!-- Remove Terminal Key Dialog -->
                <OuiDialog
                  v-model:open="removeTerminalKeyDialogOpen"
                  title="Remove Terminal Key"
                  description="This will disable web terminal access for this VPS. You can recreate the key later."
                  size="md"
                >
                  <OuiStack gap="md">
                    <OuiBox variant="warning" p="md" rounded="lg">
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="warning">
                          ⚠️ Warning
                        </OuiText>
                        <OuiText size="xs" color="secondary">
                          Removing the terminal key will disable web terminal access. The key will be removed from the VPS on the next reboot or when cloud-init is re-run.
                        </OuiText>
                        <OuiText size="xs" color="secondary">
                          You can recreate the key at any time, but web terminal access will not work until the VPS is rebooted after recreating the key.
                        </OuiText>
                      </OuiStack>
                    </OuiBox>

                    <OuiFlex justify="end" gap="sm">
                      <OuiButton
                        variant="ghost"
                        @click="removeTerminalKeyDialogOpen = false"
                        :disabled="removingTerminalKey"
                      >
                        Cancel
                      </OuiButton>
                      <OuiButton
                        variant="solid"
                        color="danger"
                        @click="removeTerminalKey"
                        :disabled="removingTerminalKey"
                      >
                        {{ removingTerminalKey ? "Removing..." : "Remove Key" }}
                      </OuiButton>
                    </OuiFlex>
                  </OuiStack>
                </OuiDialog>

                <!-- Reset Password Dialog -->
                <OuiDialog
                  v-model:open="resetPasswordDialogOpen"
                  title="Reset Root Password"
                  description="A new root password will be generated and shown once. Please save it immediately."
                  size="md"
                >
                  <OuiStack gap="md" v-if="!newPassword">
                    <OuiBox variant="warning" p="md" rounded="lg">
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="warning">
                          ⚠️ Important
                        </OuiText>
                        <OuiText size="xs" color="secondary">
                          The new password will only be shown once. After closing this dialog, you won't be able to see it again. Make sure to save it securely.
                        </OuiText>
                        <OuiText size="xs" color="secondary">
                          The password will take effect after the VPS is rebooted or cloud-init is re-run.
                        </OuiText>
                      </OuiStack>
                    </OuiBox>

                    <OuiText color="secondary" size="sm">
                      Are you sure you want to reset the root password for this VPS instance?
                    </OuiText>
                  </OuiStack>

                  <OuiStack gap="md" v-else>
                    <OuiBox variant="warning" p="md" rounded="lg">
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="warning">
                          ⚠️ Save This Password Now
                        </OuiText>
                        <OuiText size="xs" color="secondary">
                          This password will only be shown once. If you lose it, you'll need to reset it again.
                        </OuiText>
                      </OuiStack>
                    </OuiBox>

                    <OuiStack gap="xs">
                      <OuiText size="sm" weight="medium">New Root Password</OuiText>
                      <OuiBox p="md" rounded="md" class="bg-surface-muted font-mono text-sm">
                        <OuiFlex justify="between" align="center" gap="sm">
                          <OuiText class="select-all">{{ newPassword }}</OuiText>
                          <OuiButton
                            variant="ghost"
                            size="xs"
                            @click="copyNewPassword"
                            class="gap-1"
                          >
                            <ClipboardDocumentListIcon class="h-4 w-4" />
                            Copy
                          </OuiButton>
                        </OuiFlex>
                      </OuiBox>
                      <OuiText size="xs" color="secondary">
                        {{ resetPasswordMessage || "The password will take effect after the VPS is rebooted or cloud-init is re-run." }}
                      </OuiText>
                    </OuiStack>
                  </OuiStack>

                  <template #footer>
                    <OuiFlex justify="end" gap="sm">
                      <OuiButton
                        v-if="!newPassword"
                        variant="ghost"
                        @click="resetPasswordDialogOpen = false"
                        :disabled="resettingPassword"
                      >
                        Cancel
                      </OuiButton>
                      <OuiButton
                        v-if="!newPassword"
                        color="warning"
                        @click="handleResetPassword"
                        :disabled="resettingPassword || !vps.instanceId"
                      >
                        {{ resettingPassword ? "Resetting..." : "Reset Password" }}
                      </OuiButton>
                      <template v-else>
                        <OuiButton
                          v-if="!passwordRebooted"
                          variant="outline"
                          color="primary"
                          @click="handleRebootFromDialog"
                          :disabled="isActioning || !vps.instanceId"
                          class="gap-2"
                        >
                          <ArrowPathIcon class="h-4 w-4" />
                          {{ isActioning ? "Rebooting..." : "Reboot VPS" }}
                        </OuiButton>
                        <OuiButton
                          color="primary"
                          @click="() => {
                            resetPasswordDialogOpen = false;
                            newPassword = null;
                            resetPasswordMessage = null;
                            passwordRebooted = false;
                          }"
                        >
                          I've Saved the Password
                        </OuiButton>
                      </template>
                    </OuiFlex>
                  </template>
                </OuiDialog>

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
                    <OuiInput
                      v-model="newSSHKeyName"
                      label="Key Name"
                      required
                      placeholder="e.g., My Laptop, Work Computer"
                      :disabled="addingSSHKey"
                    />
                    <OuiTextarea
                      v-model="newSSHKeyValue"
                      label="Public Key"
                      required
                      placeholder="ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQ..."
                      :rows="4"
                      :disabled="addingSSHKey"
                      helper-text="Paste your SSH public key (usually from ~/.ssh/id_rsa.pub or ~/.ssh/id_ed25519.pub)"
                    />
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
        </ResourceTabs>

      </template>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowPathIcon,
  CommandLineIcon,
  PlayIcon,
  ServerIcon,
  StopIcon,
  TrashIcon,
  InformationCircleIcon,
  ClipboardDocumentListIcon,
  KeyIcon,
  PencilIcon,
  CpuChipIcon,
  CircleStackIcon,
  CubeIcon,
  ShieldExclamationIcon,
  UserIcon,
  CogIcon,
  PlusIcon,
  CheckIcon,
  XMarkIcon,
} from "@heroicons/vue/24/outline";
import { VPSService, VPSConfigService, VPSStatus, VPSImage, type VPSInstance } from "@obiente/proto";
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
import VPSUsersManagement from "~/components/vps/VPSUsersManagement.vue";
import VPSCloudInitSettings from "~/components/vps/VPSCloudInitSettings.vue";
import AuditLogs from "~/components/audit/AuditLogs.vue";
import ErrorAlert from "~/components/ErrorAlert.vue";
import ResourceHeader from "~/components/resource/ResourceHeader.vue";
import ResourceStatusBadge from "~/components/resource/ResourceStatusBadge.vue";
import ResourceDetailsGrid from "~/components/resource/ResourceDetailsGrid.vue";
import ResourceDetailCard from "~/components/resource/ResourceDetailCard.vue";
import ResourceTabs from "~/components/resource/ResourceTabs.vue";
import OuiSpinner from "~/components/oui/Spinner.vue";
import type { TabItem } from "~/components/oui/Tabs.vue";
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
const configClient = useConnectClient(VPSConfigService);
const accessError = ref<Error | null>(null);
const isActioning = ref(false);
const isRefreshing = ref(false);

// Computed error hint message
const errorHint = computed(() => {
  if (!accessError.value || !(accessError.value instanceof ConnectError)) {
    return "You don't have permission to view this VPS instance, or it doesn't exist.";
  }

  if (accessError.value.code === Code.PermissionDenied) {
    return "You don't have permission to view this VPS instance. Please contact your organization administrator if you believe you should have access.";
  }

  if (accessError.value.code === Code.NotFound) {
    return "This VPS instance doesn't exist or may have been deleted.";
  }

  return "You don't have permission to view this VPS instance, or it doesn't exist.";
});

// Fetch VPS data
const {
  data: vpsData,
  pending,
  error,
  refresh: refreshVPSData,
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

// Refresh function with loading state
const refreshVPS = async () => {
  if (isRefreshing.value) return;
  isRefreshing.value = true;
  try {
    await refreshVPSData();
  } finally {
    isRefreshing.value = false;
  }
};

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

// Terminal key management
const terminalKey = ref<{ fingerprint: string; createdAt: { seconds: number | bigint; nanos: number } } | null>(null);
const terminalKeyLoading = ref(false);
const terminalKeyError = ref<string | null>(null);
const rotatingTerminalKey = ref(false);
const removingTerminalKey = ref(false);
const removeTerminalKeyDialogOpen = ref(false);

// Password Reset
const resetPasswordDialogOpen = ref(false);
const resettingPassword = ref(false);
const newPassword = ref<string | null>(null);
const resetPasswordMessage = ref<string | null>(null);
const passwordRebooted = ref(false);

// Terminal key functions
// Note: We don't have a GetTerminalKey endpoint yet, so we infer key existence
// from VPS status. If VPS is provisioned, assume key might exist.
// Users can try to rotate/remove, and we'll handle errors appropriately.
const fetchTerminalKey = async () => {
  if (!orgId.value || !vpsId.value || !vps.value?.instanceId) {
    terminalKey.value = null;
    terminalKeyLoading.value = false;
    return;
  }

  terminalKeyLoading.value = true;
  terminalKeyError.value = null;
  try {
    // For now, assume key exists if VPS is provisioned
    // In the future, we can add a GetTerminalKey endpoint for accurate status
    // For MVP, we'll show "unknown" state and let users interact
    // The rotate/remove actions will provide feedback on actual key status
    // For now, show placeholder - actual key info will be updated after rotate/remove actions
    // Use VPS creation date as placeholder
    const createdAt = vps.value.createdAt 
      ? (typeof vps.value.createdAt === 'object' && 'seconds' in vps.value.createdAt
          ? vps.value.createdAt
          : date(vps.value.createdAt))
      : { seconds: Math.floor(Date.now() / 1000), nanos: 0 };
    
    terminalKey.value = {
      fingerprint: "Unknown",
      createdAt: createdAt as { seconds: number | bigint; nanos: number },
    };
    terminalKeyLoading.value = false;
  } catch (err: any) {
    terminalKeyError.value = err.message || "Failed to load terminal key status";
    terminalKeyLoading.value = false;
  }
};

// Fetch terminal key when VPS is loaded
watch(() => vps.value?.instanceId, async (instanceId) => {
  if (instanceId) {
    await fetchTerminalKey();
  } else {
    terminalKey.value = null;
  }
}, { immediate: true });

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

// Password Reset Functions
const openResetPasswordDialog = () => {
  resetPasswordDialogOpen.value = true;
  newPassword.value = null;
  resetPasswordMessage.value = null;
  passwordRebooted.value = false;
};

const rotateTerminalKey = async () => {
  if (!orgId.value || !vpsId.value) {
    return;
  }

  rotatingTerminalKey.value = true;
  try {
    const response = await configClient.rotateTerminalKey({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    toast.success("Terminal key rotated successfully. The new key will take effect after reboot.");
    
    // Update terminal key info
    const now = Math.floor(Date.now() / 1000);
    terminalKey.value = {
      fingerprint: response.fingerprint,
      createdAt: { seconds: now, nanos: 0 },
    };
    
    // Refresh VPS to ensure UI is up to date
    await refreshVPS();
  } catch (err: any) {
    if (err instanceof ConnectError) {
      if (err.code === Code.NotFound) {
        // Key doesn't exist - this shouldn't happen with rotate, but handle it
        toast.error("Terminal key not found. The key may need to be created first.");
        terminalKey.value = null;
      } else {
        toast.error(`Failed to rotate terminal key: ${err.message}`);
      }
    } else {
      toast.error(`Failed to rotate terminal key: ${err.message || "Unknown error"}`);
    }
  } finally {
    rotatingTerminalKey.value = false;
  }
};

const openRemoveTerminalKeyDialog = () => {
  removeTerminalKeyDialogOpen.value = true;
};

const removeTerminalKey = async () => {
  if (!orgId.value || !vpsId.value) {
    return;
  }

  removingTerminalKey.value = true;
  try {
    await configClient.removeTerminalKey({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    toast.success("Terminal key removed. Web terminal access will be disabled after reboot.");
    
    // Clear terminal key info
    terminalKey.value = null;
    removeTerminalKeyDialogOpen.value = false;
    
    // Refresh VPS to ensure UI is up to date
    await refreshVPS();
  } catch (err: any) {
    if (err instanceof ConnectError) {
      if (err.code === Code.NotFound) {
        toast.error("Terminal key not found.");
      } else {
        toast.error(`Failed to remove terminal key: ${err.message}`);
      }
    } else {
      toast.error(`Failed to remove terminal key: ${err.message || "Unknown error"}`);
    }
  } finally {
    removingTerminalKey.value = false;
  }
};

const handleResetPassword = async () => {
  if (!vps.value || !orgId.value) return;

  resettingPassword.value = true;
  try {
    const res = await client.resetVPSPassword({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    newPassword.value = res.rootPassword || null;
    resetPasswordMessage.value = res.message || null;
    toast.success("Password reset successfully", "Please save the new password - it will not be shown again.");
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to reset password", message);
    resetPasswordDialogOpen.value = false;
  } finally {
    resettingPassword.value = false;
  }
};

const copyNewPassword = async () => {
  if (!newPassword.value) return;
  try {
    await navigator.clipboard.writeText(newPassword.value);
    toast.success("Password copied to clipboard");
  } catch (err) {
    toast.error("Failed to copy password");
  }
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

const getStatusMeta = (status: number) => {
  switch (status) {
    case VPSStatus.RUNNING:
      return {
        badge: "success" as const,
        label: "Running",
        dotClass: "bg-success",
      };
    case VPSStatus.CREATING:
    case VPSStatus.STARTING:
    case VPSStatus.REBOOTING:
      return {
        badge: "warning" as const,
        label: status === VPSStatus.CREATING ? "Creating" : status === VPSStatus.STARTING ? "Starting" : "Rebooting",
        dotClass: "bg-warning",
      };
    case VPSStatus.STOPPED:
    case VPSStatus.STOPPING:
      return {
        badge: "secondary" as const,
        label: status === VPSStatus.STOPPING ? "Stopping" : "Stopped",
        dotClass: "bg-secondary",
      };
    case VPSStatus.FAILED:
      return {
        badge: "danger" as const,
        label: "Failed",
        dotClass: "bg-danger",
      };
    default:
      return {
        badge: "secondary" as const,
        label: "Unknown",
        dotClass: "bg-secondary",
      };
  }
};

const statusMeta = computed(() => {
  if (!vps.value) {
    return {
      badge: "secondary" as const,
      label: "Unknown",
      dotClass: "bg-secondary",
    };
  }
  return getStatusMeta(vps.value.status);
});

const statusBadgeColor = computed(() => statusMeta.value.badge);
const statusDotClass = computed(() => statusMeta.value.dotClass);

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

  await performReboot();
}

async function handleRebootFromDialog() {
  // Reboot without confirmation (user already confirmed by resetting password)
  // Don't close dialog or clear password - user needs to confirm they saved it
  await performReboot();
  // Mark as rebooted so we can hide the reboot button
  passwordRebooted.value = true;
}

async function performReboot() {
  if (!vps.value) return;

  isActioning.value = true;
  try {
    await client.rebootVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    // Only show password-related message if we're in the password reset flow
    if (newPassword.value) {
      toast.success("VPS instance rebooting", "The VPS instance is rebooting. The new password will be active after reboot.");
    } else {
      toast.success("VPS instance rebooting");
    }
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
    // Redirect immediately to prevent any refetch attempts
    await router.push("/vps");
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
  { id: "firewall", label: "Firewall", icon: ShieldExclamationIcon },
  { id: "users", label: "Users", icon: UserIcon },
  { id: "cloud-init", label: "Cloud-Init", icon: CogIcon },
  { id: "ssh-settings", label: "SSH Settings", icon: KeyIcon },
  { id: "audit-logs", label: "Audit Logs", icon: ClipboardDocumentListIcon },
]);

// Get activeTab from ResourceTabs component
const tabsRef = ref<{ activeTab: { value: string } } | null>(null);
const activeTab = computed({
  get: () => tabsRef.value?.activeTab.value || "overview",
  set: (value: string) => {
    if (tabsRef.value) {
      tabsRef.value.activeTab.value = value;
    }
  },
});

function openTerminal() {
  if (tabsRef.value) {
    tabsRef.value.activeTab.value = "terminal";
  }
}
</script>

