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

      <!-- Loading State (only on initial load, not during refresh) -->
      <template v-else-if="pending && !vps">
        <!-- Header Skeleton -->
        <OuiStack gap="md" class="md:gap-xl">
          <OuiCard>
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
                  <OuiStack gap="sm" class="flex-1 min-w-0">
                    <OuiSkeleton width="12rem" height="2rem" variant="text" />
                    <OuiSkeleton width="20rem" height="1.25rem" variant="text" />
                  </OuiStack>
                  <OuiFlex gap="sm">
                    <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                    <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                    <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                  </OuiFlex>
                </OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Overview Cards Skeleton -->
          <ResourceDetailsGrid cols="1" cols-md="2" cols-lg="4">
            <OuiCard v-for="i in 4" :key="i">
              <OuiCardBody>
                <OuiStack gap="sm">
                  <OuiSkeleton width="6rem" height="0.875rem" variant="text" />
                  <OuiSkeleton width="4rem" height="1.5rem" variant="text" />
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </ResourceDetailsGrid>

          <!-- Tab Content Skeleton -->
          <OuiCard>
            <OuiCardHeader>
              <OuiSkeleton width="8rem" height="1.5rem" variant="text" />
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="lg">
                <OuiStack gap="md">
                  <OuiSkeleton width="100%" height="1rem" variant="text" />
                  <OuiSkeleton width="80%" height="1rem" variant="text" />
                  <OuiSkeleton width="90%" height="1rem" variant="text" />
                </OuiStack>
                <OuiSkeleton width="100%" height="12rem" variant="rectangle" rounded />
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiStack>
      </template>

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
                              <OuiBox
                                v-for="(ip, idx) in vps.ipv4Addresses"
                                :key="idx"
                                p="sm"
                                rounded="md"
                                class="bg-surface-muted/40 ring-1 ring-border-muted"
                              >
                                <OuiFlex justify="between" align="center" gap="sm">
                                  <OuiText size="sm" class="font-mono">{{ ip }}</OuiText>
                                  <OuiButton
                                    variant="ghost"
                                    size="xs"
                                    icon-only
                                    @click="copyToClipboard(ip)"
                                    title="Copy IP address"
                                  >
                                    <ClipboardDocumentListIcon class="h-3 w-3" />
                                  </OuiButton>
                                </OuiFlex>
                              </OuiBox>
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
                              <OuiBox
                                v-for="(ip, idx) in vps.ipv6Addresses"
                                :key="idx"
                                p="sm"
                                rounded="md"
                                class="bg-surface-muted/40 ring-1 ring-border-muted"
                              >
                                <OuiFlex justify="between" align="center" gap="sm">
                                  <OuiText size="sm" class="font-mono">{{ ip }}</OuiText>
                                  <OuiButton
                                    variant="ghost"
                                    size="xs"
                                    icon-only
                                    @click="copyToClipboard(ip)"
                                    title="Copy IP address"
                                  >
                                    <ClipboardDocumentListIcon class="h-3 w-3" />
                                  </OuiButton>
                                </OuiFlex>
                              </OuiBox>
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
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">Connection Information</OuiText>
                        <OuiText color="secondary" size="sm">
                          Access your VPS instance using SSH or the web terminal (available in the Terminal tab)
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                          <OuiStack gap="sm">
                            <OuiFlex align="center" gap="sm">
                              <CommandLineIcon class="h-5 w-5 text-primary" />
                              <OuiText size="sm" weight="semibold" color="primary">SSH Access</OuiText>
                            </OuiFlex>
                            <OuiText size="sm" color="secondary">
                              Connect to your VPS via SSH using the SSH proxy. SSH key authentication is tried first, then password authentication.
                            </OuiText>
                            <div v-if="sshInfo" class="mt-2">
                              <OuiStack gap="sm">
                                <div>
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
                                </div>
                                <div v-if="sshInfo.connectionInstructions">
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
                              </OuiStack>
                            </div>
                            <div v-else-if="sshInfoLoading" class="mt-2">
                              <OuiStack gap="sm">
                                <OuiText size="xs" weight="semibold">Quick Connect:</OuiText>
                                <OuiBox p="sm" rounded="md" class="bg-surface-muted font-mono text-xs">
                                  <OuiSkeleton width="24rem" height="1rem" variant="text" />
                                </OuiBox>
                                <OuiButton variant="ghost" size="xs" disabled class="self-start">
                                  <ClipboardDocumentListIcon class="h-3 w-3 mr-1" />
                                  Copy Command
                                </OuiButton>
                              </OuiStack>
                            </div>
                            <div v-else-if="sshInfoError" class="mt-2">
                              <OuiBox variant="danger" p="sm" rounded="md">
                                <OuiText size="xs" color="danger">
                                  Failed to load SSH connection info. {{ sshInfoError }}
                                </OuiText>
                              </OuiBox>
                            </div>
                          </OuiStack>
                        </OuiBox>
                        <OuiBox p="md" rounded="lg" class="bg-primary/5 ring-1 ring-primary/20">
                          <OuiStack gap="xs">
                            <OuiFlex align="center" gap="sm">
                              <CommandLineIcon class="h-5 w-5 text-primary" />
                              <OuiText size="sm" weight="semibold" color="primary">Web Terminal</OuiText>
                            </OuiFlex>
                            <OuiText size="sm" color="secondary">
                              Access your VPS directly from the browser using the built-in web terminal. Navigate to the <strong>Terminal</strong> tab to get started.
                            </OuiText>
                            <OuiButton
                              variant="outline"
                              size="sm"
                              @click="openTerminal"
                              class="self-start gap-2 mt-1"
                            >
                              <CommandLineIcon class="h-4 w-4" />
                              Go to Terminal Tab
                            </OuiButton>
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
              <template #settings>
                <OuiStack gap="xl">
                  <!-- General Settings -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">General Settings</OuiText>
                        <OuiText color="secondary" size="sm">
                          Manage basic VPS configuration and information
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="lg">
                        <!-- VPS Name -->
                        <OuiStack gap="sm">
                          <OuiText size="sm" weight="semibold">VPS Name</OuiText>
                          <OuiFlex gap="sm" align="end">
                            <OuiInput
                              v-model="vpsName"
                              placeholder="Enter VPS name"
                              class="flex-1"
                            />
                            <OuiButton
                              variant="solid"
                              size="sm"
                              @click="handleRename"
                              :disabled="isActioning || !vpsName || vpsName === vps?.name"
                            >
                              Save
                            </OuiButton>
                          </OuiFlex>
                          <OuiText size="xs" color="secondary">
                            A descriptive name to identify this VPS instance
                          </OuiText>
                        </OuiStack>

                        <!-- VPS Description -->
                        <OuiStack gap="sm">
                          <OuiText size="sm" weight="semibold">Description</OuiText>
                          <OuiFlex gap="sm" align="end">
                            <OuiInput
                              v-model="vpsDescription"
                              placeholder="Enter description (optional)"
                              class="flex-1"
                            />
                            <OuiButton
                              variant="solid"
                              size="sm"
                              @click="handleUpdateDescription"
                              :disabled="isActioning || vpsDescription === vps?.description"
                            >
                              Save
                            </OuiButton>
                          </OuiFlex>
                          <OuiText size="xs" color="secondary">
                            Optional description for this VPS instance
                          </OuiText>
                        </OuiStack>

                        <!-- VPS Information (Read-only) -->
                        <OuiStack gap="sm">
                          <OuiText size="sm" weight="semibold">VPS Information</OuiText>
                          <OuiStack gap="xs">
                            <OuiFlex justify="between" align="center">
                              <OuiText size="sm" color="secondary">VPS ID</OuiText>
                              <OuiText size="sm" weight="medium" class="font-mono text-xs">{{ vps?.id }}</OuiText>
                            </OuiFlex>
                            <OuiFlex justify="between" align="center" v-if="vps?.instanceId">
                              <OuiText size="sm" color="secondary">VM ID</OuiText>
                              <OuiText size="sm" weight="medium" class="font-mono text-xs">{{ vps.instanceId }}</OuiText>
                            </OuiFlex>
                            <OuiFlex justify="between" align="center" v-if="vps?.nodeId">
                              <OuiText size="sm" color="secondary">Node ID</OuiText>
                              <OuiText size="sm" weight="medium" class="font-mono text-xs">{{ vps.nodeId }}</OuiText>
                            </OuiFlex>
                            <OuiFlex justify="between" align="center">
                              <OuiText size="sm" color="secondary">Region</OuiText>
                              <OuiText size="sm" weight="medium">{{ vps?.region || "—" }}</OuiText>
                            </OuiFlex>
                            <OuiFlex justify="between" align="center">
                              <OuiText size="sm" color="secondary">Size</OuiText>
                              <OuiText size="sm" weight="medium">{{ vps?.size || "—" }}</OuiText>
                            </OuiFlex>
                          </OuiStack>
                        </OuiStack>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Security Settings -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">Security Settings</OuiText>
                        <OuiText color="secondary" size="sm">
                          Manage passwords and security configurations
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="lg">
                        <!-- Reset Root Password -->
                        <OuiStack gap="sm">
                          <OuiText size="sm" weight="semibold">Root Password</OuiText>
                          <OuiText size="xs" color="secondary">
                            Reset the root password for this VPS. The new password will be generated and shown once.
                            It will take effect after the VPS is rebooted or cloud-init is re-run.
                          </OuiText>
                          <OuiButton
                            variant="outline"
                            size="sm"
                            @click="resetPasswordDialogOpen = true"
                            :disabled="isActioning || !vps?.instanceId"
                            class="self-start"
                          >
                            Reset Root Password
                          </OuiButton>
                        </OuiStack>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- VM Management -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">VM Management</OuiText>
                        <OuiText color="secondary" size="sm">
                          Advanced VM operations and reinitialization
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="lg">
                        <!-- Reinitialize VM -->
                        <OuiStack gap="sm">
                          <OuiText size="sm" weight="semibold">Reinitialize VPS</OuiText>
                          <OuiText size="xs" color="secondary">
                            Reinstall the VPS with a fresh OS image. This will permanently delete all data on the VPS
                            and reinstall the operating system. The VPS will be reconfigured with cloud-init settings.
                          </OuiText>
                          <OuiButton
                            variant="outline"
                            color="warning"
                            size="sm"
                            @click="handleReinit"
                            :disabled="isActioning || !vps?.instanceId"
                            class="self-start"
                          >
                            Reinitialize VPS
                          </OuiButton>
                        </OuiStack>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Danger Zone -->
                  <OuiCard variant="outline" class="border-danger/20">
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiStack gap="xs">
                          <OuiText as="h3" size="lg" weight="semibold" color="danger">
                            Danger Zone
                          </OuiText>
                          <OuiText size="sm" color="secondary">
                            Irreversible and destructive actions
                          </OuiText>
                        </OuiStack>
                        <OuiStack gap="md">
                          <!-- Delete VPS -->
                          <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
                            <OuiStack gap="xs" class="flex-1 min-w-0">
                              <OuiText size="sm" weight="medium" color="primary">
                                Delete VPS Instance
                              </OuiText>
                              <OuiText size="xs" color="secondary">
                                Once you delete a VPS instance, there is no going back. This will
                                permanently remove the VPS and all associated data.
                              </OuiText>
                            </OuiStack>
                            <OuiButton
                              variant="outline"
                              color="danger"
                              size="sm"
                              @click="handleDelete"
                              :disabled="isActioning"
                              class="gap-2 shrink-0"
                            >
                              <TrashIcon class="h-4 w-4" />
                              <OuiText as="span" size="xs" weight="medium"
                                >Delete VPS</OuiText
                              >
                            </OuiButton>
                          </OuiFlex>
                        </OuiStack>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiStack>
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
                        <div v-if="sshKeysLoading" class="space-y-3">
                          <OuiBox
                            v-for="i in 3"
                            :key="i"
                            p="md"
                            rounded="lg"
                            class="bg-surface-muted/40 ring-1 ring-border-muted"
                          >
                            <OuiStack gap="sm">
                              <OuiFlex justify="between" align="start">
                                <OuiStack gap="xs" class="flex-1">
                                  <OuiSkeleton width="10rem" height="1rem" variant="text" />
                                  <OuiSkeleton width="8rem" height="0.875rem" variant="text" />
                                  <OuiSkeleton width="100%" height="1.5rem" variant="rectangle" rounded />
                                </OuiStack>
                                <OuiSkeleton width="4rem" height="1.75rem" variant="rectangle" rounded />
                              </OuiFlex>
                            </OuiStack>
                          </OuiBox>
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

                  <!-- SSH Alias -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">SSH Alias</OuiText>
                        <OuiText color="secondary" size="sm">
                          Set a memorable alias for easier SSH connections. Instead of using the long VPS ID, you can use a short alias like "prod-db" or "web-1".
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiBox variant="info" p="sm" rounded="md">
                          <OuiStack gap="xs">
                            <OuiText size="xs" weight="semibold">About SSH Aliases</OuiText>
                            <OuiText size="xs" color="secondary">
                              SSH aliases make it easier to connect to your VPS. Instead of typing the long VPS ID, you can use a short, memorable alias. Aliases must be unique and can only contain alphanumeric characters, hyphens, and underscores.
                            </OuiText>
                          </OuiStack>
                        </OuiBox>

                        <div v-if="sshAliasLoading" class="py-4">
                          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                            <OuiStack gap="sm">
                              <OuiSkeleton width="8rem" height="1rem" variant="text" />
                              <OuiSkeleton width="12rem" height="0.875rem" variant="text" />
                              <OuiSkeleton width="100%" height="1.5rem" variant="rectangle" rounded />
                            </OuiStack>
                          </OuiBox>
                        </div>
                        <div v-else-if="sshAliasError" class="py-4">
                          <OuiBox variant="danger" p="sm" rounded="md">
                            <OuiText color="danger" size="sm" class="text-center">
                              Failed to load SSH alias: {{ sshAliasError }}
                            </OuiText>
                          </OuiBox>
                        </div>
                        <div v-else-if="sshAlias" class="space-y-3">
                          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                            <OuiStack gap="sm">
                              <OuiFlex justify="between" align="start">
                                <OuiStack gap="xs">
                                  <OuiText size="sm" weight="semibold">SSH Alias Active</OuiText>
                                  <OuiText size="xs" color="secondary" class="font-mono">
                                    {{ sshAlias }}
                                  </OuiText>
                                  <OuiText size="xs" color="muted">
                                    Connect using: <code class="text-xs bg-surface-muted px-1 py-0.5 rounded">ssh -p {{ sshPort }} root@{{ sshAlias }}@{{ sshDomain }}</code>
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
                              @click="openSetSSHAliasDialog"
                              :disabled="settingSSHAlias || removingSSHAlias || !vps.instanceId"
                              class="flex-1"
                            >
                              <PencilIcon class="h-4 w-4 mr-2" />
                              Change Alias
                            </OuiButton>
                            <OuiButton
                              variant="outline"
                              color="danger"
                              @click="openRemoveSSHAliasDialog"
                              :disabled="settingSSHAlias || removingSSHAlias || !vps.instanceId"
                              class="flex-1"
                            >
                              <TrashIcon class="h-4 w-4 mr-2" />
                              Remove
                            </OuiButton>
                          </OuiFlex>
                        </div>
                        <div v-else class="py-4">
                          <OuiStack gap="md" align="center">
                            <OuiText color="secondary" class="text-center">
                              No SSH alias set. Set an alias to make SSH connections easier.
                            </OuiText>
                            <OuiButton
                              variant="outline"
                              color="primary"
                              @click="openSetSSHAliasDialog"
                              :disabled="settingSSHAlias || !vps.instanceId"
                            >
                              <PlusIcon class="h-4 w-4 mr-2" />
                              Set SSH Alias
                            </OuiButton>
                          </OuiStack>
                        </div>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- SSH Bastion Key -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">SSH Bastion Key</OuiText>
                        <OuiText color="secondary" size="sm">
                          Manage the SSH key used for SSH bastion host connections. This key is required for SSH access through the bastion host and is automatically generated.
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiBox variant="info" p="sm" rounded="md">
                          <OuiStack gap="xs">
                            <OuiText size="xs" weight="semibold">About Bastion Keys</OuiText>
                            <OuiText size="xs" color="secondary">
                              The bastion key is required for SSH proxy connections through the bastion host. The public key is automatically added to the root user via cloud-init. This key is essential for SSH access, so it cannot be removed.
                            </OuiText>
                          </OuiStack>
                        </OuiBox>

                        <div v-if="bastionKeyLoading" class="py-4">
                          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                            <OuiStack gap="sm">
                              <OuiSkeleton width="10rem" height="1rem" variant="text" />
                              <OuiSkeleton width="8rem" height="0.875rem" variant="text" />
                              <OuiSkeleton width="100%" height="1.5rem" variant="rectangle" rounded />
                              <OuiSkeleton width="6rem" height="1.75rem" variant="rectangle" rounded />
                            </OuiStack>
                          </OuiBox>
                        </div>
                        <div v-else-if="bastionKeyError" class="py-4">
                          <OuiBox variant="danger" p="sm" rounded="md">
                            <OuiText color="danger" size="sm" class="text-center">
                              Failed to load bastion key status: {{ bastionKeyError }}
                            </OuiText>
                          </OuiBox>
                        </div>
                        <div v-else-if="bastionKey" class="space-y-3">
                          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                            <OuiStack gap="sm">
                              <OuiFlex justify="between" align="start">
                                <OuiStack gap="xs">
                                  <OuiText size="sm" weight="semibold">Bastion Key Active</OuiText>
                                  <OuiText size="xs" color="secondary" class="font-mono">
                                    {{ bastionKey.fingerprint }}
                                  </OuiText>
                                  <OuiText size="xs" color="muted">
                                    Created {{ formatSSHKeyDate(bastionKey.createdAt) }}
                                  </OuiText>
                                </OuiStack>
                                <OuiBadge variant="success" size="sm">Active</OuiBadge>
                              </OuiFlex>
                            </OuiStack>
                          </OuiBox>

                          <OuiButton
                            variant="outline"
                            color="primary"
                            @click="rotateBastionKey"
                            :disabled="rotatingBastionKey || !vps.instanceId"
                            class="w-full"
                          >
                            <ArrowPathIcon class="h-4 w-4 mr-2" />
                            {{ rotatingBastionKey ? "Rotating..." : "Rotate Key" }}
                          </OuiButton>

                          <OuiText size="xs" color="secondary">
                            Rotating the key generates a new key pair. The new key will take effect after the VPS is rebooted or cloud-init is re-run.
                          </OuiText>
                        </div>
                        <div v-else class="py-4">
                          <OuiStack gap="md" align="center">
                            <OuiText color="secondary" class="text-center">
                              No bastion key found. SSH bastion access requires this key.
                            </OuiText>
                            <OuiButton
                              variant="outline"
                              color="primary"
                              @click="rotateBastionKey"
                              :disabled="rotatingBastionKey || !vps.instanceId"
                            >
                              <KeyIcon class="h-4 w-4 mr-2" />
                              {{ rotatingBastionKey ? "Creating..." : "Create Bastion Key" }}
                            </OuiButton>
                          </OuiStack>
                        </div>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Web Terminal Key -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiStack gap="xs">
                        <OuiText as="h2" class="oui-card-title">Web Terminal Key</OuiText>
                        <OuiText color="secondary" size="sm">
                          Manage the SSH key used for web terminal access. This key is optional and can be removed to disable web terminal access while keeping SSH bastion working.
                        </OuiText>
                      </OuiStack>
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiBox variant="info" p="sm" rounded="md">
                          <OuiStack gap="xs">
                            <OuiText size="xs" weight="semibold">About Web Terminal Keys</OuiText>
                            <OuiText size="xs" color="secondary">
                              The web terminal key enables browser-based terminal access to your VPS. The public key is automatically added to the root user via cloud-init. You can remove this key to disable web terminal access while keeping SSH bastion access working.
                            </OuiText>
                          </OuiStack>
                        </OuiBox>

                        <div v-if="terminalKeyLoading" class="py-4">
                          <OuiBox p="md" rounded="lg" class="bg-surface-muted/40 ring-1 ring-border-muted">
                            <OuiStack gap="sm">
                              <OuiSkeleton width="10rem" height="1rem" variant="text" />
                              <OuiSkeleton width="8rem" height="0.875rem" variant="text" />
                              <OuiSkeleton width="100%" height="1.5rem" variant="rectangle" rounded />
                              <OuiSkeleton width="6rem" height="1.75rem" variant="rectangle" rounded />
                            </OuiStack>
                          </OuiBox>
                        </div>
                        <div v-else-if="terminalKeyError" class="py-4">
                          <OuiBox variant="danger" p="sm" rounded="md">
                            <OuiText color="danger" size="sm" class="text-center">
                              Failed to load terminal key status: {{ terminalKeyError }}
                            </OuiText>
                          </OuiBox>
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
                            Rotating the key generates a new key pair. Removing the key disables web terminal access. The new key will take effect after the VPS is rebooted or cloud-init is re-run.
                          </OuiText>
                        </div>
                        <div v-else class="py-4">
                          <OuiStack gap="md" align="center">
                            <OuiText color="secondary" class="text-center">
                              No terminal key found. Web terminal access is disabled. Create a key to enable it.
                            </OuiText>
                            <OuiButton
                              variant="outline"
                              color="primary"
                              @click="rotateTerminalKey"
                              :disabled="rotatingTerminalKey || !vps.instanceId"
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
                  description="This will disable web terminal access for this VPS."
                  size="md"
                >
                  <OuiStack gap="md">
                    <OuiBox variant="warning" p="md" rounded="lg">
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="warning">
                          ⚠️ Warning
                        </OuiText>
                        <OuiText size="xs" color="secondary">
                          Removing the terminal key will disable web terminal access. SSH bastion access will continue to work using the separate bastion key.
                        </OuiText>
                        <OuiText size="xs" color="secondary">
                          The key will be removed from the VPS on the next reboot or when cloud-init is re-run. You can recreate the key at any time to re-enable web terminal access.
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

                <!-- Set SSH Alias Dialog -->
                <OuiDialog
                  v-model:open="setSSHAliasDialogOpen"
                  title="Set SSH Alias"
                  description="Set a memorable alias for easier SSH connections."
                  size="md"
                >
                  <OuiStack gap="md">
                    <OuiBox variant="info" p="sm" rounded="md">
                      <OuiStack gap="xs">
                        <OuiText size="xs" weight="semibold">Alias Requirements</OuiText>
                        <OuiText size="xs" color="secondary">
                          • 1-63 characters<br/>
                          • Alphanumeric characters, hyphens, and underscores only<br/>
                          • Cannot start with "vps-"<br/>
                          • Must be unique across all VPS instances
                        </OuiText>
                      </OuiStack>
                    </OuiBox>

                    <div>
                      <OuiInput
                        v-model="newSSHAlias"
                        label="SSH Alias"
                        placeholder="e.g., prod-db, web-1, api-server"
                        :disabled="settingSSHAlias"
                        @keyup.enter="setSSHAlias"
                      />
                      <OuiText v-if="newSSHAlias" size="xs" color="secondary" class="mt-1">
                        Connect using: <code class="text-xs bg-surface-muted px-1 py-0.5 rounded">ssh -p {{ sshPort }} root@{{ newSSHAlias }}@{{ sshDomain }}</code>
                      </OuiText>
                    </div>

                    <OuiFlex justify="end" gap="sm">
                      <OuiButton
                        variant="ghost"
                        @click="setSSHAliasDialogOpen = false"
                        :disabled="settingSSHAlias"
                      >
                        Cancel
                      </OuiButton>
                      <OuiButton
                        variant="solid"
                        @click="setSSHAlias"
                        :disabled="settingSSHAlias || !newSSHAlias || !isValidSSHAlias(newSSHAlias)"
                      >
                        {{ settingSSHAlias ? "Setting..." : "Set Alias" }}
                      </OuiButton>
                    </OuiFlex>
                  </OuiStack>
                </OuiDialog>

                <!-- Remove SSH Alias Dialog -->
                <OuiDialog
                  v-model:open="removeSSHAliasDialogOpen"
                  title="Remove SSH Alias"
                  description="This will remove the SSH alias for this VPS."
                  size="md"
                >
                  <OuiStack gap="md">
                    <OuiBox variant="info" p="sm" rounded="md">
                      <OuiText size="xs" color="secondary">
                        After removing the alias, you'll need to use the full VPS ID to connect: <code class="text-xs bg-surface-muted px-1 py-0.5 rounded">ssh -p {{ sshPort }} root@{{ vpsId }}@{{ sshDomain }}</code>
                      </OuiText>
                    </OuiBox>

                    <OuiFlex justify="end" gap="sm">
                      <OuiButton
                        variant="ghost"
                        @click="removeSSHAliasDialogOpen = false"
                        :disabled="removingSSHAlias"
                      >
                        Cancel
                      </OuiButton>
                      <OuiButton
                        variant="solid"
                        color="danger"
                        @click="removeSSHAlias"
                        :disabled="removingSSHAlias"
                      >
                        {{ removingSSHAlias ? "Removing..." : "Remove Alias" }}
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
import { computed, ref, watch, nextTick, defineAsyncComponent } from "vue";
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
import { VPSService, VPSConfigService, SuperadminService, VPSStatus, VPSImage, type VPSInstance } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import { useOrganizationsStore } from "~/stores/organizations";
import { useDialog } from "~/composables/useDialog";
import { useSuperAdmin } from "~/composables/useSuperAdmin";
import { ConnectError, Code } from "@connectrpc/connect";
import OuiByte from "~/components/oui/Byte.vue";
import OuiDate from "~/components/oui/Date.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import ErrorAlert from "~/components/ErrorAlert.vue";
import OuiSkeleton from "~/components/oui/Skeleton.vue";
// Lazy load tab components for better performance
const VPSFirewall = defineAsyncComponent(() => import("~/components/vps/VPSFirewall.vue"));
const VPSXTermTerminal = defineAsyncComponent(() => import("~/components/vps/VPSXTermTerminal.vue"));
const VPSUsersManagement = defineAsyncComponent(() => import("~/components/vps/VPSUsersManagement.vue"));
const VPSCloudInitSettings = defineAsyncComponent(() => import("~/components/vps/VPSCloudInitSettings.vue"));
const AuditLogs = defineAsyncComponent(() => import("~/components/audit/AuditLogs.vue"));
const ResourceHeader = defineAsyncComponent(() => import("~/components/resource/ResourceHeader.vue"));
const ResourceStatusBadge = defineAsyncComponent(() => import("~/components/resource/ResourceStatusBadge.vue"));
const ResourceDetailsGrid = defineAsyncComponent(() => import("~/components/resource/ResourceDetailsGrid.vue"));
const ResourceDetailCard = defineAsyncComponent(() => import("~/components/resource/ResourceDetailCard.vue"));
const ResourceTabs = defineAsyncComponent(() => import("~/components/resource/ResourceTabs.vue"));
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
const superAdmin = useSuperAdmin();

const vpsId = computed(() => String(route.params.id));
const orgId = computed(() => orgsStore.currentOrgId || "");
const isSuperAdmin = computed(() => superAdmin.allowed.value === true);

const client = useConnectClient(VPSService);
const superadminClient = useConnectClient(SuperadminService);
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
  execute: executeVPSData,
} = await useClientFetch(
  () => `vps-${vpsId.value}-${orgId.value}-${isSuperAdmin.value}`,
  async () => {
    try {
      let res;
      // Try regular endpoint first if we have an orgId, otherwise use superadmin endpoint
      if (orgId.value && !isSuperAdmin.value) {
        // Regular user - must use regular endpoint
        res = await client.getVPS({
          organizationId: orgId.value,
          vpsId: vpsId.value,
        });
        accessError.value = null;
        return res.vps ?? null;
      } else if (isSuperAdmin.value) {
        // Superadmin - try regular endpoint first if orgId is set, fallback to superadmin endpoint
        if (orgId.value) {
          try {
            res = await client.getVPS({
              organizationId: orgId.value,
              vpsId: vpsId.value,
            });
            accessError.value = null;
            return res.vps ?? null;
          } catch (regularErr: unknown) {
            // If regular endpoint fails, use superadmin endpoint
            if (regularErr instanceof ConnectError && 
                (regularErr.code === Code.NotFound || regularErr.code === Code.PermissionDenied)) {
              res = await superadminClient.superadminGetVPS({
                vpsId: vpsId.value,
              });
              accessError.value = null;
              // Switch to the VPS's organization for proper context
              if (res.vps?.organizationId && res.vps.organizationId !== orgId.value) {
                // Use nextTick to avoid triggering watch during fetch
                await nextTick();
                orgsStore.switchOrganization(res.vps.organizationId);
              }
              return res.vps ?? null;
            }
            throw regularErr;
          }
        } else {
          // No orgId - use superadmin endpoint
          res = await superadminClient.superadminGetVPS({
            vpsId: vpsId.value,
          });
          accessError.value = null;
          // Switch to the VPS's organization for proper context
          if (res.vps?.organizationId) {
            await nextTick();
            orgsStore.switchOrganization(res.vps.organizationId);
          }
          return res.vps ?? null;
        }
      } else {
        // No orgId and not superadmin - error
        throw new Error("No organization context available");
      }
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

// Settings form data
const vpsName = ref("");
const vpsDescription = ref("");

// Watch VPS data to update form fields
watch(vps, (newVps) => {
  if (newVps) {
    vpsName.value = newVps.name || "";
    vpsDescription.value = newVps.description || "";
  }
}, { immediate: true });

// Refresh function with loading state
// This keeps existing data visible while refreshing in the background
const refreshVPS = async () => {
  if (isRefreshing.value) return;
  isRefreshing.value = true;
  try {
    // Use execute instead of refresh to avoid setting pending to true
    // This keeps the existing UI visible while data is being fetched
    await executeVPSData();
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

// Extract domain and port from SSH proxy command
// Format: ssh -p {port} root@{vpsId}@{domain}
const sshDomain = computed(() => {
  if (!sshInfo.value?.sshProxyCommand) {
    return "localhost";
  }
  // Parse: ssh -p {port} root@{vpsId}@{domain}
  const match = sshInfo.value.sshProxyCommand.match(/@([^@]+)$/);
  return match ? match[1] : "localhost";
});

const sshPort = computed(() => {
  if (!sshInfo.value?.sshProxyCommand) {
    return "2323";
  }
  // Parse: ssh -p {port} root@{vpsId}@{domain}
  const match = sshInfo.value.sshProxyCommand.match(/-p\s+(\d+)/);
  return match ? match[1] : "2323";
});

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
const terminalKey = ref<{ fingerprint: string; createdAt: { seconds: number | bigint; nanos: number }; updatedAt: { seconds: number | bigint; nanos: number } } | null>(null);
const terminalKeyLoading = ref(false);
const terminalKeyError = ref<string | null>(null);
const rotatingTerminalKey = ref(false);
const removingTerminalKey = ref(false);
const removeTerminalKeyDialogOpen = ref(false);

// Bastion key management
const bastionKey = ref<{ fingerprint: string; createdAt: { seconds: number | bigint; nanos: number }; updatedAt: { seconds: number | bigint; nanos: number } } | null>(null);
const bastionKeyLoading = ref(false);
const bastionKeyError = ref<string | null>(null);
const rotatingBastionKey = ref(false);

// SSH alias management
const sshAlias = ref<string | null>(null);
const sshAliasLoading = ref(false);
const sshAliasError = ref<string | null>(null);
const setSSHAliasDialogOpen = ref(false);
const removeSSHAliasDialogOpen = ref(false);
const newSSHAlias = ref("");
const settingSSHAlias = ref(false);
const removingSSHAlias = ref(false);

// Password Reset
const resetPasswordDialogOpen = ref(false);
const resettingPassword = ref(false);
const newPassword = ref<string | null>(null);
const resetPasswordMessage = ref<string | null>(null);
const passwordRebooted = ref(false);

// Terminal key functions
const fetchTerminalKey = async () => {
  if (!orgId.value || !vpsId.value || !vps.value?.instanceId) {
    terminalKey.value = null;
    terminalKeyLoading.value = false;
    return;
  }

  terminalKeyLoading.value = true;
  terminalKeyError.value = null;
  try {
    const res = await configClient.getTerminalKey({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    terminalKey.value = {
      fingerprint: res.fingerprint,
      createdAt: res.createdAt as { seconds: number | bigint; nanos: number },
      updatedAt: res.updatedAt as { seconds: number | bigint; nanos: number },
    };
    terminalKeyLoading.value = false;
  } catch (err: any) {
    if (err instanceof ConnectError && err.code === Code.NotFound) {
      // Key doesn't exist - this is fine, just set to null
      terminalKey.value = null;
    } else {
      terminalKeyError.value = err instanceof Error ? err.message : "Failed to load terminal key status";
    }
    terminalKeyLoading.value = false;
  }
};

// Bastion key functions
const fetchBastionKey = async () => {
  if (!orgId.value || !vpsId.value || !vps.value?.instanceId) {
    bastionKey.value = null;
    bastionKeyLoading.value = false;
    return;
  }

  bastionKeyLoading.value = true;
  bastionKeyError.value = null;
  try {
    const res = await configClient.getBastionKey({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    bastionKey.value = {
      fingerprint: res.fingerprint,
      createdAt: res.createdAt as { seconds: number | bigint; nanos: number },
      updatedAt: res.updatedAt as { seconds: number | bigint; nanos: number },
    };
    bastionKeyLoading.value = false;
  } catch (err: any) {
    if (err instanceof ConnectError && err.code === Code.NotFound) {
      // Key doesn't exist - this is fine, just set to null
      bastionKey.value = null;
    } else {
      bastionKeyError.value = err instanceof Error ? err.message : "Failed to load bastion key status";
    }
    bastionKeyLoading.value = false;
  }
};

// SSH alias functions
const fetchSSHAlias = async () => {
  if (!orgId.value || !vpsId.value || !vps.value?.instanceId) {
    sshAlias.value = null;
    sshAliasLoading.value = false;
    return;
  }

  sshAliasLoading.value = true;
  sshAliasError.value = null;
  try {
    const res = await configClient.getSSHAlias({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    sshAlias.value = res.alias || null;
    sshAliasLoading.value = false;
  } catch (err: any) {
    if (err instanceof ConnectError && err.code === Code.NotFound) {
      sshAlias.value = null;
    } else {
      sshAliasError.value = err instanceof Error ? err.message : "Failed to load SSH alias";
    }
    sshAliasLoading.value = false;
  }
};

const isValidSSHAlias = (alias: string): boolean => {
  if (!alias || alias.length === 0 || alias.length > 63) {
    return false;
  }
  // Check if contains only allowed characters
  const validPattern = /^[a-zA-Z0-9_-]+$/;
  if (!validPattern.test(alias)) {
    return false;
  }
  // Cannot start with "vps-"
  if (alias.length >= 4 && alias.substring(0, 4) === "vps-") {
    return false;
  }
  return true;
};

const openSetSSHAliasDialog = () => {
  newSSHAlias.value = sshAlias.value || "";
  setSSHAliasDialogOpen.value = true;
};

const openRemoveSSHAliasDialog = () => {
  removeSSHAliasDialogOpen.value = true;
};

const setSSHAlias = async () => {
  if (!orgId.value || !vpsId.value || !newSSHAlias.value || !isValidSSHAlias(newSSHAlias.value)) {
    return;
  }

  settingSSHAlias.value = true;
  try {
    const response = await configClient.setSSHAlias({
      organizationId: orgId.value,
      vpsId: vpsId.value,
      alias: newSSHAlias.value,
    });
    
    toast.success(response.message || `SSH alias '${response.alias}' has been set.`);
    sshAlias.value = response.alias;
    setSSHAliasDialogOpen.value = false;
    newSSHAlias.value = "";
  } catch (err: any) {
    const errorMsg = err instanceof Error ? err.message : "Failed to set SSH alias";
    toast.error(errorMsg);
  } finally {
    settingSSHAlias.value = false;
  }
};

const removeSSHAlias = async () => {
  if (!orgId.value || !vpsId.value) {
    return;
  }

  removingSSHAlias.value = true;
  try {
    const response = await configClient.removeSSHAlias({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    toast.success(response.message || "SSH alias has been removed.");
    sshAlias.value = null;
    removeSSHAliasDialogOpen.value = false;
  } catch (err: any) {
    const errorMsg = err instanceof Error ? err.message : "Failed to remove SSH alias";
    toast.error(errorMsg);
  } finally {
    removingSSHAlias.value = false;
  }
};

// Fetch both keys and alias when VPS is loaded
watch(() => vps.value?.instanceId, async (instanceId) => {
  if (instanceId) {
    await Promise.all([fetchTerminalKey(), fetchBastionKey(), fetchSSHAlias()]);
  } else {
    terminalKey.value = null;
    bastionKey.value = null;
    sshAlias.value = null;
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
    
    // Refresh terminal key info
    await fetchTerminalKey();
    
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

const rotateBastionKey = async () => {
  if (!orgId.value || !vpsId.value) {
    return;
  }

  rotatingBastionKey.value = true;
  try {
    const response = await configClient.rotateBastionKey({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    toast.success("Bastion key rotated successfully. The new key will take effect after reboot.");
    
    // Refresh bastion key info
    await fetchBastionKey();
    
    // Refresh VPS to ensure UI is up to date
    await refreshVPS();
  } catch (err: any) {
    if (err instanceof ConnectError) {
      if (err.code === Code.NotFound) {
        toast.error("Bastion key not found. The key may need to be created first.");
        bastionKey.value = null;
      } else {
        toast.error(`Failed to rotate bastion key: ${err.message}`);
      }
    } else {
      toast.error(`Failed to rotate bastion key: ${err.message || "Unknown error"}`);
    }
  } finally {
    rotatingBastionKey.value = false;
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
    
    // Clear terminal key info and refresh
    terminalKey.value = null;
    removeTerminalKeyDialogOpen.value = false;
    await fetchTerminalKey(); // Refresh to confirm removal
    
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

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  } catch (err) {
    toast.error("Failed to copy to clipboard");
  }
};

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
    case VPSStatus.DELETING:
      return {
        badge: "warning" as const,
        label: "Deleting",
        dotClass: "bg-warning",
      };
    case VPSStatus.DELETED:
      return {
        badge: "secondary" as const,
        label: "Deleted",
        dotClass: "bg-secondary",
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

async function handleRename() {
  if (!vps.value || !vpsName.value.trim() || vpsName.value === vps.value.name) return;

  isActioning.value = true;
  try {
    await client.updateVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
      name: vpsName.value.trim(),
    });
    toast.success("VPS renamed", "The VPS name has been updated.");
    await refreshVPS();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to rename VPS", message);
  } finally {
    isActioning.value = false;
  }
}

async function handleUpdateDescription() {
  if (!vps.value || vpsDescription.value === vps.value.description) return;

  isActioning.value = true;
  try {
    await client.updateVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
      description: vpsDescription.value.trim() || undefined,
    });
    toast.success("Description updated", "The VPS description has been updated.");
    await refreshVPS();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to update description", message);
  } finally {
    isActioning.value = false;
  }
}

async function handleReinit() {
  if (!vps.value) return;
  
  const confirmed = await showConfirm({
    title: "Reinitialize VPS",
    message: `Are you sure you want to reinitialize "${vps.value.name}"? This will permanently delete all data on the VPS and reinstall the operating system. The VPS will be reconfigured with the same cloud-init settings. This action cannot be undone.`,
    confirmLabel: "Reinitialize",
    cancelLabel: "Cancel",
    variant: "danger",
  });
  if (!confirmed) return;

  isActioning.value = true;
  try {
    const res = await client.reinitializeVPS({
      organizationId: orgId.value,
      vpsId: vpsId.value,
    });
    
    toast.success("VPS reinitialized", res.message || "The VPS is being reinitialized.");
    
    // Show password dialog if password was returned
    if (res.rootPassword) {
      await showAlert({
        title: "VPS Reinitialized",
        message: `The VPS has been reinitialized. Please save this root password as it will not be shown again:\n\n${res.rootPassword}\n\nThe VPS is being provisioned and cloud-init will be reapplied.`,
      });
    }
    
    // Refresh VPS data
    await refreshVPS();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : "Unknown error";
    toast.error("Failed to reinitialize VPS", message);
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
  { id: "settings", label: "Settings", icon: CogIcon },
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

