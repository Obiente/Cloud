<template>
  <OuiContainer>
    <OuiStack gap="xl">
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
              @click="router.push('/gameservers')"
            >
              Go to Game Servers
            </OuiButton>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Game Server Content (only show if no access error and game server exists) -->
      <template v-else-if="gameServer">
        <!-- Header -->
        <OuiCard variant="outline" class="border-border-default/50">
          <OuiCardBody>
            <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
              <OuiStack gap="md" class="flex-1 min-w-0">
                <OuiFlex align="center" gap="md" wrap="wrap">
                  <OuiBox
                    p="sm"
                    rounded="xl"
                    bg="accent-primary"
                    class="bg-primary/10 ring-1 ring-primary/20 shrink-0"
                  >
                    <CubeIcon class="w-6 h-6 text-primary" />
                  </OuiBox>
                  <OuiStack gap="none" class="min-w-0 flex-1">
                    <OuiFlex align="center" gap="md">
                      <OuiText as="h1" size="2xl" weight="bold" truncate>
                        {{ gameServer.name || "Loading..." }}
                      </OuiText>
                      <OuiBadge v-if="gameServer.status" :variant="statusMeta.badge" size="xs">
                        <span
                          class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                          :class="statusMeta.dotClass"
                        />
                        <OuiText
                          as="span"
                          size="xs"
                          weight="semibold"
                          transform="uppercase"
                          >{{ statusMeta.label }}</OuiText
                        >
                      </OuiBadge>
                    </OuiFlex>
                    <OuiText size="sm" color="secondary" class="hidden sm:inline">
                      <span v-if="gameServer.gameType !== undefined">{{ getGameTypeLabel(gameServer.gameType) }} • </span>
                      Last updated
                      <OuiRelativeTime
                        :value="gameServer.updatedAt ? date(gameServer.updatedAt) : undefined"
                        :style="'short'"
                      />
                    </OuiText>
                  </OuiStack>

                  <OuiFlex gap="sm" wrap="wrap" class="shrink-0">
                    <OuiButton
                      variant="ghost"
                      color="secondary"
                      size="sm"
                      @click="refreshAll"
                      :loading="isRefreshing"
                      class="gap-2"
                    >
                      <ArrowPathIcon
                        class="h-4 w-4"
                        :class="{ 'animate-spin': isRefreshing }"
                      />
                      <OuiText as="span" size="xs" weight="medium">Refresh</OuiText>
                    </OuiButton>
                    <OuiButton
                      :color="gameServer.status === 'RUNNING' ? 'danger' : 'success'"
                      variant="outline"
                      size="sm"
                      class="gap-2"
                      :loading="isStarting || isStopping || isRestarting"
                      :disabled="isActionDisabled(gameServer.status)"
                      @click="toggleServerStatus"
                    >
                      <template v-if="gameServer.status === 'RUNNING'">
                        <StopIcon class="h-4 w-4" />
                        <OuiText as="span" size="xs" weight="medium">Stop</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'STOPPED' || gameServer.status === 'FAILED'">
                        <PlayIcon class="h-4 w-4" />
                        <OuiText as="span" size="xs" weight="medium">Start</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'STARTING'">
                        <PlayIcon class="h-4 w-4 animate-pulse" />
                        <OuiText as="span" size="xs" weight="medium">Starting...</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'STOPPING'">
                        <StopIcon class="h-4 w-4 animate-pulse" />
                        <OuiText as="span" size="xs" weight="medium">Stopping...</OuiText>
                      </template>
                      <template v-else-if="gameServer.status === 'RESTARTING'">
                        <ArrowPathIcon class="h-4 w-4 animate-spin" />
                        <OuiText as="span" size="xs" weight="medium">Restarting...</OuiText>
                      </template>
                      <template v-else>
                        <PlayIcon class="h-4 w-4" />
                        <OuiText as="span" size="xs" weight="medium">Start</OuiText>
                      </template>
                    </OuiButton>
                    <OuiButton
                      variant="outline"
                      color="secondary"
                      size="sm"
                      class="gap-2"
                      :disabled="isActionDisabled(gameServer.status)"
                      @click="restartServer"
                    >
                      <ArrowPathIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium">Restart</OuiText>
                    </OuiButton>
                    <OuiButton
                      variant="outline"
                      color="danger"
                      size="sm"
                      class="gap-2"
                      @click="showDeleteDialog = true"
                    >
                      <TrashIcon class="h-4 w-4" />
                      <OuiText as="span" size="xs" weight="medium">Delete</OuiText>
                    </OuiButton>
                  </OuiFlex>
                </OuiFlex>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>

        <!-- Overview Cards -->
        <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <CpuChipIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">vCPU Cores (Max)</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ gameServer.cpuCores || "N/A" }}
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <CircleStackIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">Memory (Max)</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  <OuiByte :value="getMemoryBytesValue(gameServer.memoryBytes)" />
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <ServerIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">Port</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ gameServer.port || "N/A" }}
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <ChartBarIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="sm" color="secondary">Monthly Cost</OuiText>
                </OuiFlex>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ formatCurrency(estimatedMonthlyCost) }}
                </OuiText>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <!-- Tabs -->
        <OuiStack gap="md">
          <OuiTabs v-model="activeTab" :tabs="tabs" />
          <OuiCard variant="default">
            <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
              <template #overview>
              <GameServerOverview
                :game-server="gameServer"
                :usage-data="usageData"
                :is-streaming="isStreaming"
                :latest-metric="latestMetric"
                :current-cpu-usage="currentCpuUsage"
                :current-memory-usage="currentMemoryUsage"
                :current-network-rx="currentNetworkRx"
                :current-network-tx="currentNetworkTx"
              />
              </template>
            <template #logs>
              <GameServerLogs
                :game-server-id="gameServerId"
                :organization-id="gameServer?.organizationId || ''"
              />
            </template>
              <template #metrics>
                <GameServerMetrics
                  :game-server-id="gameServerId"
                  :organization-id="gameServer?.organizationId || ''"
                  :game-server-status="(() => {
                    const status = gameServerData?.status;
                    if (status === undefined || status === null) return undefined;
                    if (typeof status === 'number') return status;
                    // If it's a string, try to map it back to number
                    const statusMap: Record<string, number> = {
                      'CREATED': 1,
                      'STARTING': 2,
                      'RUNNING': 3,
                      'STOPPING': 4,
                      'STOPPED': 5,
                      'FAILED': 6,
                      'RESTARTING': 7,
                    };
                    return statusMap[status as string] ?? undefined;
                  })()"
                />
              </template>
            <template #files>
              <GameServerFiles :game-server-id="gameServerId" />
            </template>
            <template #eula>
              <MinecraftFileEditor
                :game-server-id="gameServerId"
                file-path="eula.txt"
                :editor-component="MinecraftEULAEditor"
              />
            </template>
            <template #server-properties>
              <MinecraftFileEditor
                :game-server-id="gameServerId"
                file-path="server.properties"
                :editor-component="MinecraftServerPropertiesEditor"
                :editor-props="{ serverVersion: gameServer?.serverVersion }"
              />
            </template>
            <template #users>
              <MinecraftUsersEditor :game-server-id="gameServerId" />
            </template>
            <template #settings>
              <GameServerSettings
                :game-server="gameServerData as any"
                @saved="refreshAll"
                @delete="showDeleteDialog = true"
              />
            </template>
            <template #audit-logs>
              <AuditLogs
                :organization-id="gameServer?.organizationId || ''"
                resource-type="game_server"
                :resource-id="gameServerId"
              />
            </template>
            </OuiTabs>
          </OuiCard>
        </OuiStack>
      </template>
      
      <!-- Loading State -->
      <template v-else-if="showLoadingSkeleton">
        <!-- Header Skeleton -->
        <OuiCard variant="outline" class="border-border-default/50">
          <OuiCardBody>
            <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
              <OuiStack gap="md" class="flex-1 min-w-0">
                <OuiFlex align="center" gap="md" wrap="wrap">
                  <OuiSkeleton width="3rem" height="3rem" variant="rectangle" :rounded="true" class="rounded-xl shrink-0" />
                  <OuiStack gap="none" class="min-w-0 flex-1">
                    <OuiFlex align="center" gap="md">
                      <OuiSkeleton width="16rem" height="2rem" variant="text" />
                      <OuiSkeleton width="5rem" height="1.5rem" variant="rectangle" rounded />
                    </OuiFlex>
                    <OuiSkeleton width="12rem" height="1rem" variant="text" class="mt-1" />
                  </OuiStack>
                  <OuiFlex gap="sm" wrap="wrap" class="shrink-0">
                    <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                    <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                    <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                    <OuiSkeleton width="5rem" height="2rem" variant="rectangle" rounded />
                  </OuiFlex>
                </OuiFlex>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>

        <!-- Overview Cards Skeleton -->
        <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                  <OuiSkeleton width="8rem" height="1rem" variant="text" />
                </OuiFlex>
                <OuiSkeleton width="4rem" height="2rem" variant="text" />
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                  <OuiSkeleton width="8rem" height="1rem" variant="text" />
                </OuiFlex>
                <OuiSkeleton width="5rem" height="2rem" variant="text" />
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                  <OuiSkeleton width="4rem" height="1rem" variant="text" />
                </OuiFlex>
                <OuiSkeleton width="3rem" height="2rem" variant="text" />
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex align="center" gap="sm">
                  <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                  <OuiSkeleton width="8rem" height="1rem" variant="text" />
                </OuiFlex>
                <OuiSkeleton width="6rem" height="2rem" variant="text" />
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <!-- Tabs Skeleton -->
        <OuiStack gap="md">
          <!-- Tab Navigation Skeleton (outside card, matching actual structure) -->
          <OuiFlex gap="sm" wrap="wrap" class="px-1">
            <OuiSkeleton width="6rem" height="2.5rem" variant="rectangle" rounded />
            <OuiSkeleton width="5rem" height="2.5rem" variant="rectangle" rounded />
            <OuiSkeleton width="6rem" height="2.5rem" variant="rectangle" rounded />
            <OuiSkeleton width="5rem" height="2.5rem" variant="rectangle" rounded />
            <OuiSkeleton width="6rem" height="2.5rem" variant="rectangle" rounded />
          </OuiFlex>
          <!-- Tab Content Skeleton (Overview tab structure) -->
          <OuiCard variant="default">
            <OuiCardBody>
              <OuiStack gap="xl">
                <!-- Key Metrics Grid Skeleton -->
                <OuiGrid cols="1" cols-md="2" cols-lg="3" cols-xl="4" gap="md">
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="sm">
                        <OuiSkeleton width="4rem" height="0.75rem" variant="text" />
                        <OuiFlex align="center" gap="sm">
                          <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                          <OuiSkeleton width="3rem" height="1.5rem" variant="text" />
                        </OuiFlex>
                        <OuiSkeleton width="8rem" height="0.75rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="sm">
                        <OuiSkeleton width="5rem" height="0.75rem" variant="text" />
                        <OuiFlex align="center" gap="sm">
                          <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                          <OuiSkeleton width="3rem" height="1.5rem" variant="text" />
                        </OuiFlex>
                        <OuiSkeleton width="10rem" height="0.75rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="sm">
                        <OuiSkeleton width="5rem" height="0.75rem" variant="text" />
                        <OuiFlex align="center" gap="sm">
                          <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                          <OuiSkeleton width="4rem" height="1.5rem" variant="text" />
                        </OuiFlex>
                        <OuiSkeleton width="8rem" height="0.75rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="sm">
                        <OuiSkeleton width="6rem" height="0.75rem" variant="text" />
                        <OuiFlex align="center" gap="sm">
                          <OuiSkeleton width="1.25rem" height="1.25rem" variant="circle" />
                          <OuiSkeleton width="4rem" height="1.5rem" variant="text" />
                        </OuiFlex>
                        <OuiSkeleton width="10rem" height="0.75rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiGrid>

                <!-- Usage Statistics Skeleton -->
                <OuiCard>
                  <OuiCardHeader>
                    <OuiSkeleton width="12rem" height="1.5rem" variant="text" />
                  </OuiCardHeader>
                  <OuiCardBody>
                    <OuiStack gap="md">
                      <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="md">
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                      </OuiGrid>
                      <OuiSkeleton width="100%" height="8rem" variant="rectangle" rounded />
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>

                <!-- Cost Breakdown Skeleton -->
                <OuiCard>
                  <OuiCardHeader>
                    <OuiSkeleton width="10rem" height="1.5rem" variant="text" />
                  </OuiCardHeader>
                  <OuiCardBody>
                    <OuiStack gap="md">
                      <OuiSkeleton width="100%" height="6rem" variant="rectangle" rounded />
                      <OuiGrid cols="1" cols-md="2" gap="md">
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                      </OuiGrid>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>

                <!-- Live Metrics Skeleton -->
                <OuiCard>
                  <OuiCardHeader>
                    <OuiSkeleton width="8rem" height="1.5rem" variant="text" />
                  </OuiCardHeader>
                  <OuiCardBody>
                    <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="md">
                      <OuiSkeleton width="100%" height="6rem" variant="rectangle" rounded />
                      <OuiSkeleton width="100%" height="6rem" variant="rectangle" rounded />
                      <OuiSkeleton width="100%" height="6rem" variant="rectangle" rounded />
                      <OuiSkeleton width="100%" height="6rem" variant="rectangle" rounded />
                    </OuiGrid>
                  </OuiCardBody>
                </OuiCard>

                <!-- Main Information Grid Skeleton -->
                <OuiGrid cols="1" cols-lg="2" gap="lg">
                  <!-- Game Server Details Skeleton -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiSkeleton width="10rem" height="1.5rem" variant="text" />
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
                          <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                            <OuiSkeleton width="1rem" height="1rem" variant="circle" />
                            <OuiStack gap="xs" class="flex-1">
                              <OuiSkeleton width="5rem" height="0.75rem" variant="text" />
                              <OuiSkeleton width="8rem" height="1rem" variant="text" />
                            </OuiStack>
                          </OuiFlex>
                        </div>
                        <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
                          <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                            <OuiSkeleton width="1rem" height="1rem" variant="circle" />
                            <OuiStack gap="xs" class="flex-1">
                              <OuiSkeleton width="4rem" height="0.75rem" variant="text" />
                              <OuiSkeleton width="6rem" height="1rem" variant="text" />
                            </OuiStack>
                          </OuiFlex>
                        </div>
                        <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
                          <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                            <OuiSkeleton width="1rem" height="1rem" variant="circle" />
                            <OuiStack gap="xs" class="flex-1">
                              <OuiSkeleton width="6rem" height="0.75rem" variant="text" />
                              <OuiSkeleton width="10rem" height="1rem" variant="text" />
                            </OuiStack>
                          </OuiFlex>
                        </div>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Connection Info Skeleton -->
                  <OuiCard>
                    <OuiCardHeader>
                      <OuiSkeleton width="12rem" height="1.5rem" variant="text" />
                    </OuiCardHeader>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                        <OuiSkeleton width="100%" height="4rem" variant="rectangle" rounded />
                        <OuiStack gap="sm">
                          <OuiSkeleton width="8rem" height="1rem" variant="text" />
                          <OuiSkeleton width="100%" height="3rem" variant="rectangle" rounded />
                        </OuiStack>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiGrid>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiStack>
      </template>
      
      <!-- Not Found State (only show when not loading, no data, and no access error) -->
      <template v-else-if="!pending && !gameServerData && !accessError">
        <OuiStack align="center" gap="lg" class="text-center py-20">
          <OuiBox
            class="inline-flex items-center justify-center w-20 h-20 rounded-xl bg-surface-muted/50 ring-1 ring-border-muted"
          >
            <CubeIcon class="h-10 w-10 text-secondary" />
          </OuiBox>
          <OuiStack align="center" gap="sm">
            <OuiText as="h3" size="xl" weight="semibold" color="primary">
              Game server not found
            </OuiText>
            <OuiText color="secondary">
              The game server you are looking for does not exist or you do not have access.
            </OuiText>
          </OuiStack>
          <OuiButton
            color="primary"
            class="gap-2 shadow-lg shadow-primary/20"
            @click="router.push('/gameservers')"
          >
            <ArrowLeftIcon class="h-4 w-4" />
            <OuiText as="span" size="sm" weight="medium"
              >Go to Game Servers List</OuiText
            >
          </OuiButton>
        </OuiStack>
      </template>
      
      <!-- Fallback: Show loading if we have an ID but none of the above conditions matched -->
      <!-- This prevents blank pages during navigation when pending might be false briefly -->
      <template v-else-if="gameServerId && !gameServerData && !accessError">
        <!-- Use the same loading skeleton as above -->
        <OuiCard variant="outline" class="border-border-default/50">
          <OuiCardBody>
            <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
              <OuiStack gap="md" class="flex-1 min-w-0">
                <OuiFlex align="center" gap="md" wrap="wrap">
                  <OuiSkeleton width="3rem" height="3rem" variant="rectangle" :rounded="true" class="rounded-xl shrink-0" />
                  <OuiStack gap="none" class="min-w-0 flex-1">
                    <OuiFlex align="center" gap="md">
                      <OuiSkeleton width="16rem" height="2rem" variant="text" />
                      <OuiSkeleton width="5rem" height="1.5rem" variant="rectangle" rounded />
                    </OuiFlex>
                    <OuiSkeleton width="12rem" height="1rem" variant="text" class="mt-1" />
                  </OuiStack>
                </OuiFlex>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </template>
    </OuiStack>

    <!-- Delete Confirmation Dialog -->
    <OuiDialog v-model:open="showDeleteDialog" title="Delete Game Server">
      <OuiText color="secondary">
        Are you sure you want to delete the game server
        <OuiText as="span" weight="semibold" color="primary">{{ gameServer?.name }}</OuiText>?
        This action cannot be undone.
      </OuiText>
      <template #footer>
        <OuiButton variant="ghost" @click="showDeleteDialog = false">Cancel</OuiButton>
        <OuiButton color="danger" :loading="isDeleting" @click="deleteGameServer">Delete</OuiButton>
      </template>
    </OuiDialog>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, defineAsyncComponent } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowPathIcon,
  ArrowLeftIcon,
  ChartBarIcon,
  Cog6ToothIcon,
  CpuChipIcon,
  CubeIcon,
  PlayIcon,
  ServerIcon,
  StopIcon,
  CircleStackIcon,
  TrashIcon,
  DocumentTextIcon,
  GlobeAltIcon,
  ClipboardIcon,
  FolderIcon,
  ClipboardDocumentListIcon,
  DocumentCheckIcon,
  UserGroupIcon,
  UserMinusIcon,
  ShieldCheckIcon,
} from "@heroicons/vue/24/outline";

import type { TabItem } from "~/components/oui/Tabs.vue";
import { useTabQuery } from "~/composables/useTabQuery";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";
// Lazy load tab components for better performance
const GameServerMetrics = defineAsyncComponent(() => import("~/components/gameserver/GameServerMetrics.vue"));
const GameServerLogs = defineAsyncComponent(() => import("~/components/gameserver/GameServerLogs.vue"));
const GameServerFiles = defineAsyncComponent(() => import("~/components/gameserver/GameServerFiles.vue"));
const GameServerSettings = defineAsyncComponent(() => import("~/components/gameserver/GameServerSettings.vue"));
const GameServerOverview = defineAsyncComponent(() => import("~/components/gameserver/GameServerOverview.vue"));
const MinecraftFileEditor = defineAsyncComponent(() => import("~/components/gameserver/MinecraftFileEditor.vue"));
const MinecraftEULAEditor = defineAsyncComponent(() => import("~/components/gameserver/MinecraftEULAEditor.vue"));
const MinecraftServerPropertiesEditor = defineAsyncComponent(() => import("~/components/gameserver/MinecraftServerPropertiesEditor.vue"));
const MinecraftUsersEditor = defineAsyncComponent(() => import("~/components/gameserver/MinecraftUsersEditor.vue"));
const AuditLogs = defineAsyncComponent(() => import("~/components/audit/AuditLogs.vue"));
import { date } from "@obiente/proto/utils";
import { useToast } from "~/composables/useToast";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService, GameType } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import { SuperadminService } from "@obiente/proto";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();
const { toast } = useToast();
const orgsStore = useOrganizationsStore();
const client = useConnectClient(GameServerService);
const superadminClient = useConnectClient(SuperadminService);

const gameServerId = computed(() => route.params.id as string);
const effectiveOrgId = computed(() => orgsStore.currentOrgId || "");

// State
const accessError = ref<Error | null>(null);
const errorHint = computed(() => {
  return "You may not have permission to view this game server, or it may not exist.";
});

// Fetch game server data
// Use immediate: true to ensure fetch starts immediately when route changes
const { data: gameServerData, pending, refresh: refreshGameServer, error: fetchError } = useClientFetch(
  () => `game-server-${gameServerId.value}`,
  async () => {
    try {
      const res = await client.getGameServer({
        gameServerId: gameServerId.value,
      });
      // Clear any previous errors on success
      accessError.value = null;
      
      // If response is successful but gameServer is null/undefined, treat as not found
      if (!res.gameServer) {
        const notFoundError = new Error("Game server not found") as any;
        notFoundError.code = "not_found";
        accessError.value = notFoundError;
        return null;
      }
      
      return res.gameServer;
    } catch (err: any) {
      // Check if it's a permission denied or not found error
      if (err && (err.code === "permission_denied" || err.code === "not_found")) {
        accessError.value = err;
        return null;
      }
      // Re-throw other errors
      throw err;
    }
  },
  {
    watch: [gameServerId],
    immediate: true, // Ensure fetch starts immediately when gameServerId changes
  }
);

// Computed property to determine if we should show loading skeleton
// Show skeleton when we're actively fetching
// This prevents blank pages during navigation
const showLoadingSkeleton = computed(() => {
  const hasError = Boolean(accessError.value || fetchError.value);
  const hasData = Boolean(gameServerData.value);
  
  // Don't show loading if we have an error or data
  if (hasError || hasData) {
    return false;
  }
  
  // Show loading if we're actively pending
  // With lazy loading, pending should be true when a fetch is in progress
  return pending.value;
});

// Watch for fetch errors and handle access errors
watch(fetchError, (err) => {
  if (err && ((err as any)?.code === "permission_denied" || (err as any)?.code === "not_found")) {
    accessError.value = err as Error;
  }
});

// Watch for route changes and clear stale errors
watch(gameServerId, (newId, oldId) => {
  if (newId !== oldId && newId) {
    // Clear any previous errors when navigating to a new game server
    accessError.value = null;
  }
});

// Computed game server from fetched data
const gameServer = computed(() => {
  const data = gameServerData.value;
  if (!data) return null;
  
  // Convert status from enum number to string if needed
  let status: string = 'CREATED';
  if (data.status !== undefined && data.status !== null) {
    if (typeof data.status === 'number') {
      // Map GameServerStatus enum values (from proto)
      // 0: GAME_SERVER_STATUS_UNSPECIFIED, 1: CREATED, 2: STARTING, 3: RUNNING, 4: STOPPING, 5: STOPPED, 6: FAILED, 7: RESTARTING
      const statusMap: Record<number, string> = {
        0: 'CREATED', // GAME_SERVER_STATUS_UNSPECIFIED -> treat as CREATED
        1: 'CREATED',
        2: 'STARTING',
        3: 'RUNNING',
        4: 'STOPPING',
        5: 'STOPPED',
        6: 'FAILED',
        7: 'RESTARTING',
      };
      status = statusMap[data.status] || 'CREATED';
    } else if (typeof data.status === 'string') {
      status = data.status;
    }
  }
  
  return {
    ...data,
    status: status,
    gameType: typeof data.gameType === 'number' ? data.gameType : (data.gameType ? Number(data.gameType) : undefined),
  };
});
const usageData = ref<any>(null);

// Live metrics state
const isStreaming = ref(false);
const latestMetric = ref<any>(null);
const streamController = ref<AbortController | null>(null);

// Computed metrics from latest data
const currentCpuUsage = computed(() => {
  return latestMetric.value?.cpuUsagePercent ?? 0;
});

const currentMemoryUsage = computed(() => {
  return latestMetric.value?.memoryUsageBytes ?? 0;
});

const currentNetworkRx = computed(() => {
  return latestMetric.value?.networkRxBytes ?? 0;
});

const currentNetworkTx = computed(() => {
  return latestMetric.value?.networkTxBytes ?? 0;
});

// State variables
const isRefreshing = ref(false);
const isStarting = ref(false);
const isStopping = ref(false);
const isRestarting = ref(false);
const isDeleting = ref(false);
const showDeleteDialog = ref(false);

// Tabs definition
const isMinecraft = computed(() => {
  const gameType = gameServer.value?.gameType;
  if (typeof gameType === 'number') {
    return gameType === GameType.MINECRAFT || 
           gameType === GameType.MINECRAFT_JAVA || 
           gameType === GameType.MINECRAFT_BEDROCK;
  }
  return false;
});

const tabs = computed<TabItem[]>(() => {
  const baseTabs: TabItem[] = [
    { id: "overview", label: "Overview", icon: CubeIcon },
    { id: "logs", label: "Logs", icon: DocumentTextIcon },
    { id: "metrics", label: "Metrics", icon: ChartBarIcon },
    { id: "files", label: "Files", icon: FolderIcon },
  ];

  // Add Minecraft-specific tabs
  if (isMinecraft.value) {
    baseTabs.push(
      { id: "server-properties", label: "Server Properties", icon: Cog6ToothIcon },
      { id: "users", label: "Users", icon: UserGroupIcon }
    );
  }

  baseTabs.push(
    { id: "settings", label: "Settings", icon: Cog6ToothIcon },
    { id: "audit-logs", label: "Audit Logs", icon: ClipboardDocumentListIcon }
  );

  // Add EULA tab at the end
  if (isMinecraft.value) {
    baseTabs.push(
      { id: "eula", label: "EULA", icon: DocumentCheckIcon }
    );
  }

  return baseTabs;
});

// Use composable for tab query parameter management
const activeTab = useTabQuery(tabs);

// Status metadata helper
const statusMeta = computed(() => {
  if (!gameServer.value) {
    return {
      label: "Unknown",
      badge: "muted" as const,
      dotClass: "bg-muted",
    };
  }

  const statusMap: Record<string, any> = {
    RUNNING: {
      label: "Running",
      badge: "success" as const,
      dotClass: "bg-success",
    },
    STOPPED: {
      label: "Stopped",
      badge: "muted" as const,
      dotClass: "bg-muted",
    },
    STARTING: {
      label: "Starting",
      badge: "warning" as const,
      dotClass: "bg-warning",
    },
    STOPPING: {
      label: "Stopping",
      badge: "warning" as const,
      dotClass: "bg-warning",
    },
    RESTARTING: {
      label: "Restarting",
      badge: "warning" as const,
      dotClass: "bg-warning",
    },
    FAILED: {
      label: "Failed",
      badge: "danger" as const,
      dotClass: "bg-danger",
    },
  };

  return (
    statusMap[gameServer.value.status] || {
      label: "Unknown",
      badge: "muted" as const,
      dotClass: "bg-muted",
    }
  );
});

// Estimated monthly cost based on actual usage and pricing
const estimatedMonthlyCost = computed(() => {
  // Use actual usage data if available (current price of usage, like deployments)
  if (usageData.value?.estimatedMonthly?.estimatedCostCents) {
    return Number(usageData.value.estimatedMonthly.estimatedCostCents) / 100;
  }
  
  // If no usage data available yet, return 0
  return 0;
});

// Helper function to convert BigInt to number for memoryBytes
const getMemoryBytesValue = (value: bigint | number | undefined | null): number => {
  if (!value) return 0;
  if (typeof value === 'bigint') return Number(value);
  return value;
};

// Format helpers
const formatMemory = (bytes: number) => {
  if (!bytes) return "0 B";
  const gb = bytes / (1024 * 1024 * 1024);
  if (gb >= 1) return `${gb.toFixed(2)} GB`;
  const mb = bytes / (1024 * 1024);
  if (mb >= 1) return `${mb.toFixed(2)} MB`;
  return `${bytes} B`;
};

const formatStorage = (bytes: number) => {
  return formatMemory(bytes);
};

const formatCurrency = (dollars: number) => {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(dollars);
};

const getGameTypeLabel = (gameType: number) => {
  // Map GameType enum values to labels
  const types: Record<number, string> = {
    [GameType.MINECRAFT]: "Minecraft",
    [GameType.MINECRAFT_JAVA]: "Minecraft Java",
    [GameType.MINECRAFT_BEDROCK]: "Minecraft Bedrock",
    [GameType.VALHEIM]: "Valheim",
    [GameType.TERRARIA]: "Terraria",
    [GameType.RUST]: "Rust",
    [GameType.CS2]: "Counter-Strike 2",
    [GameType.TF2]: "Team Fortress 2",
    [GameType.ARK]: "ARK: Survival Evolved",
    [GameType.CONAN]: "Conan Exiles",
    [GameType.SEVEN_DAYS]: "7 Days to Die",
    [GameType.FACTORIO]: "Factorio",
    [GameType.SPACED_ENGINEERS]: "Space Engineers",
    [GameType.OTHER]: "Other",
  };
  return types[gameType] || "Unknown";
};

// Connection domain helpers
const connectionDomain = computed(() => {
  if (!gameServer.value?.id) return "";
  // Format: gs-123.my.obiente.cloud
  return `${gameServer.value.id}.my.obiente.cloud`;
});

// Get SRV domains based on game type
const srvDomains = computed(() => {
  if (!gameServer.value?.id || !gameServer.value?.gameType) return [];
  
  const gameType = typeof gameServer.value.gameType === 'number'
    ? gameServer.value.gameType as GameType
    : gameServer.value.gameType;
  const id = gameServer.value.id;
  const domains: Array<{ label: string; domain: string; description: string }> = [];
  
  // GameType enum values: MINECRAFT = 1, MINECRAFT_JAVA = 2, MINECRAFT_BEDROCK = 3, RUST = 6
  if (gameType === GameType.MINECRAFT || gameType === GameType.MINECRAFT_JAVA) {
    // Minecraft Java Edition - TCP SRV record
    domains.push({
      label: "Minecraft Java (SRV)",
      domain: `_minecraft._tcp.${id}.my.obiente.cloud`,
      description: "Use this domain in Minecraft Java Edition for automatic port resolution"
    });
  }
  
  if (gameType === GameType.MINECRAFT || gameType === GameType.MINECRAFT_BEDROCK) {
    // Minecraft Bedrock Edition - UDP SRV record
    domains.push({
      label: "Minecraft Bedrock (SRV)",
      domain: `_minecraft._udp.${id}.my.obiente.cloud`,
      description: "Use this domain in Minecraft Bedrock Edition for automatic port resolution"
    });
  }
  
  if (gameType === GameType.RUST) {
    // Rust - UDP SRV record
    domains.push({
      label: "Rust (SRV)",
      domain: `_rust._udp.${id}.my.obiente.cloud`,
      description: "Use this domain in Rust for automatic port resolution"
    });
  }
  
  return domains;
});

const isMinecraftServer = computed(() => {
  if (!gameServer.value?.gameType) return false;
  const gameType = typeof gameServer.value.gameType === 'number'
    ? gameServer.value.gameType as GameType
    : gameServer.value.gameType;
  return (
    gameType === GameType.MINECRAFT ||
    gameType === GameType.MINECRAFT_JAVA ||
    gameType === GameType.MINECRAFT_BEDROCK
  );
});

const hasSRVRecords = computed(() => {
  return srvDomains.value.length > 0;
});

// Copy to clipboard helper
const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  } catch (error) {
    console.error("Failed to copy:", error);
    toast.error("Failed to copy to clipboard");
  }
};

const isActionDisabled = (status: string) => {
  return ["STARTING", "STOPPING", "RESTARTING"].includes(status);
};

// Load game server data
const loadGameServer = async () => {
  await refreshGameServer();
  await loadUsage();
};

// Load usage data
const loadUsage = async () => {
  if (!gameServer.value) return;
  
  try {
    const month = new Date().toISOString().slice(0, 7); // YYYY-MM
    const res = await client.getGameServerUsage({
      gameServerId: gameServerId.value,
      organizationId: gameServer.value.organizationId || effectiveOrgId.value,
      month,
    });
    usageData.value = res;
  } catch (error) {
    console.error("Failed to load usage:", error);
    // Don't show error toast for usage - it's optional
  }
};

// Watch for game server data to load usage
watch(() => gameServer.value, (newValue) => {
  if (newValue) {
    loadUsage();
  }
}, { immediate: true });

// Start streaming metrics
const startStreaming = async () => {
  // Don't start if already streaming, no game server ID, or no game server data
  if (isStreaming.value || streamController.value || !gameServerId.value || !gameServer.value?.id) {
    return;
  }

  isStreaming.value = true;
  streamController.value = new AbortController();

  try {
    const stream = await client.streamGameServerMetrics({
      gameServerId: gameServerId.value,
    });

    for await (const metric of stream) {
      if (streamController.value?.signal.aborted) {
        break;
      }
      latestMetric.value = metric;
    }
  } catch (err: any) {
    // Handle abort errors silently
    if (err?.name === "AbortError" || err?.code === "aborted") {
      return;
    }
    
    // Suppress common stream errors that are expected/non-critical
    const errorMessage = err?.message?.toLowerCase() || "";
    const errorCode = err?.code || "";
    const isExpectedError =
      errorMessage.includes("missing trailer") ||
      errorMessage.includes("trailer") ||
      errorMessage.includes("error in input stream") ||
      errorMessage.includes("input stream") ||
      errorCode === "unknown" ||
      errorCode === "cancelled" ||
      errorCode === "aborted";

    // Only log unexpected errors
    if (!isExpectedError && err) {
      console.error("Failed to stream metrics:", err);
    }
  } finally {
    isStreaming.value = false;
    streamController.value = null;
  }
};

// Stop streaming
const stopStreaming = () => {
  if (streamController.value) {
    streamController.value.abort();
    streamController.value = null;
  }
  isStreaming.value = false;
};

// Start streaming when component mounts if server is running
onMounted(() => {
  if (gameServer.value?.status === "RUNNING") {
    startStreaming().catch((err) => {
      // Silently handle streaming errors on mount - they're non-critical
      // The error is already handled in startStreaming
    });
  }
});

// Watch game server status and start/stop streaming accordingly
watch(
  () => gameServer.value?.status,
  (status) => {
    if (status === "RUNNING" && !isStreaming.value) {
      startStreaming().catch((err) => {
        // Silently handle streaming errors - they're non-critical
        // The error is already handled in startStreaming
      });
    } else if (status !== "RUNNING" && isStreaming.value) {
      stopStreaming();
    }
  }
);

// Clean up on unmount
onUnmounted(() => {
  stopStreaming();
});

// Actions
const refreshAll = async () => {
  isRefreshing.value = true;
  try {
    await loadGameServer();
    toast.success("Game server refreshed");
  } catch (error) {
    toast.error("Failed to refresh game server");
  } finally {
    isRefreshing.value = false;
  }
};

const toggleServerStatus = async () => {
  if (!gameServer.value) return;
  
  if (gameServer.value.status === "RUNNING") {
    await stopServer();
  } else {
    await startServer();
  }
};

const startServer = async () => {
  if (!gameServer.value) return;
  isStarting.value = true;
  try {
    await client.startGameServer({
      gameServerId: gameServerId.value,
    });
    await loadGameServer();
    toast.success("Game server started");
  } catch (error: any) {
    // Extract error message from backend
    const errorMessage = error?.message || "Unknown error";
    
    // Check for common configuration errors
    let hint = "";
    if (errorMessage.includes("exited immediately") || errorMessage.includes("container exit")) {
      hint = "The container may be missing required environment variables or configuration. Check the Logs tab for details.";
      
      // Add specific hint for CS2 servers
      if (gameServer.value?.gameType === GameType.CS2 && errorMessage.includes("exit")) {
        hint = "CS2 servers require a Steam Game Server Login Token (SRCDS_TOKEN). Add it in the Settings tab under Environment Variables.";
      }
    }
    
    const description = hint ? `${hint}\n\nError: ${errorMessage}` : errorMessage;
    toast.error("Failed to start game server", description);
  } finally {
    isStarting.value = false;
  }
};

const stopServer = async () => {
  if (!gameServer.value) return;
  isStopping.value = true;
  try {
    await client.stopGameServer({
      gameServerId: gameServerId.value,
    });
    await loadGameServer();
    toast.success("Game server stopped");
  } catch (error) {
    toast.error("Failed to stop game server");
  } finally {
    isStopping.value = false;
  }
};

const restartServer = async () => {
  if (!gameServer.value) return;
  isRestarting.value = true;
  try {
    await client.restartGameServer({
      gameServerId: gameServerId.value,
    });
    await loadGameServer();
    toast.success("Game server restarted");
  } catch (error) {
    toast.error("Failed to restart game server");
  } finally {
    isRestarting.value = false;
  }
};

const deleteGameServer = async () => {
  if (!gameServer.value) return;
  isDeleting.value = true;
  try {
    await client.deleteGameServer({
      gameServerId: gameServerId.value,
    });
    toast.success("Game server deleted");
    router.push("/gameservers");
  } catch (error) {
    toast.error("Failed to delete game server");
  } finally {
    isDeleting.value = false;
    showDeleteDialog.value = false;
  }
};
</script>

