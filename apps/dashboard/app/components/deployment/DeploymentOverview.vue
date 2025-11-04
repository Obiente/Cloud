<template>
    <OuiStack gap="xl">
      <!-- Key Metrics Grid -->
      <OuiGrid cols="1" cols-md="2" cols-lg="3" cols-xl="4" gap="md">
        <!-- Status Card -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Status
              </OuiText>
              <OuiFlex align="center" gap="sm">
                <span
                  class="h-2 w-2 rounded-full"
                  :class="getStatusDotClass(deployment.status)"
                />
                <OuiText size="lg" weight="bold">
                  {{ getStatusLabel(deployment.status) }}
                </OuiText>
              </OuiFlex>
              <OuiText v-if="deployment.lastDeployedAt" size="xs" color="muted">
                Last deployed
                <OuiRelativeTime
                  :value="date(deployment.lastDeployedAt)"
                  :style="'short'"
                />
              </OuiText>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Containers Card -->
        <OuiCard v-if="deployment.containersTotal !== undefined">
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Containers
              </OuiText>
              <OuiFlex align="center" gap="md">
                <OuiText size="2xl" weight="bold">
                  {{ deployment.containersRunning ?? 0 }}/{{ deployment.containersTotal }}
                </OuiText>
                <OuiBadge
                  :variant="getContainerStatusVariant()"
                  size="sm"
                >
                  <OuiText as="span" size="xs" weight="medium">
                    {{ getContainerStatusLabel() }}
                  </OuiText>
                </OuiBadge>
          </OuiFlex>
              <OuiText size="xs" color="muted">
                {{ deployment.containersTotal === 1 ? "Container" : "Containers" }} total
              </OuiText>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Build Time Card -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Build Time
              </OuiText>
              <OuiFlex align="center" gap="sm">
                <ClockIcon class="h-5 w-5 text-secondary" />
                <OuiText size="2xl" weight="bold">
                  {{ formatBuildTime(deployment.buildTime ?? 0) }}
                </OuiText>
          </OuiFlex>
              <OuiText size="xs" color="muted">
                Last build duration
              </OuiText>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Size Card -->
        <OuiCard v-if="deployment.size && deployment.size !== '--'">
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Bundle Size
              </OuiText>
              <OuiFlex align="center" gap="sm">
                <ArchiveBoxIcon class="h-5 w-5 text-secondary" />
                <OuiText size="2xl" weight="bold">
                  <OuiByte :value="deployment.size ?? 0" unit-display="short" />
                </OuiText>
          </OuiFlex>
              <OuiText size="xs" color="muted">
                Compressed size
              </OuiText>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Storage Card -->
        <OuiCard v-if="deployment.storageUsage !== undefined && deployment.storageUsage !== null && Number(deployment.storageUsage) > 0">
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Storage Usage
              </OuiText>
              <OuiFlex align="center" gap="sm">
                <CubeIcon class="h-5 w-5 text-secondary" />
                <OuiText size="2xl" weight="bold">
                  <OuiByte :value="deployment.storageUsage" />
                </OuiText>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                Image + Volumes + Disk
              </OuiText>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Health Status Card -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Health Status
              </OuiText>
              <template v-if="deployment.healthStatus">
                <OuiFlex align="center" gap="sm">
                  <component
                    :is="getHealthIcon(deployment.healthStatus)"
                    :class="`h-5 w-5 ${getHealthIconClass(deployment.healthStatus)}`"
                  />
                  <OuiText size="2xl" weight="bold">
                    {{ deployment.healthStatus }}
                  </OuiText>
                </OuiFlex>
                <OuiBadge
                  :variant="getHealthVariant(deployment.healthStatus)"
                  size="sm"
                >
                  <OuiText as="span" size="xs" weight="medium">
                    {{ getHealthLabel(deployment.healthStatus) }}
                  </OuiText>
                </OuiBadge>
              </template>
              <template v-else>
                <OuiFlex align="center" gap="sm">
                  <HeartIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="2xl" weight="bold">Unknown</OuiText>
                </OuiFlex>
                <OuiText size="xs" color="muted">
                  No health data available
                </OuiText>
              </template>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Created Card -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Created
              </OuiText>
              <template v-if="deployment.createdAt">
                <OuiFlex align="center" gap="sm">
                  <CalendarIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="lg" weight="bold">
                    <OuiRelativeTime
                      :value="date(deployment.createdAt)"
                      :style="'short'"
                    />
                  </OuiText>
                </OuiFlex>
                <OuiText size="xs" color="muted">
                  {{ formatDate(deployment.createdAt) }}
                </OuiText>
              </template>
              <template v-else>
                <OuiFlex align="center" gap="sm">
                  <CalendarIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="lg" weight="bold">Unknown</OuiText>
                </OuiFlex>
                <OuiText size="xs" color="muted">
                  Creation date unavailable
                </OuiText>
              </template>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Environment Card -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Environment
              </OuiText>
              <OuiFlex align="center" gap="sm">
                <CpuChipIcon class="h-5 w-5 text-secondary" />
                <OuiBadge
                  :variant="getEnvironmentVariant(deployment.environment)"
                  size="sm"
                >
                  <OuiText as="span" size="sm" weight="medium">
                    {{ getEnvironmentLabel(deployment.environment) }}
                  </OuiText>
                </OuiBadge>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                Deployment environment
              </OuiText>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Port Card -->
        <OuiCard>
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Port
              </OuiText>
              <template v-if="deployment.port">
                <OuiFlex align="center" gap="sm">
                  <SignalIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="2xl" weight="bold">
                    {{ deployment.port }}
                  </OuiText>
                </OuiFlex>
                <OuiText size="xs" color="muted">
                  Application port
                </OuiText>
              </template>
              <template v-else>
                <OuiFlex align="center" gap="sm">
                  <SignalIcon class="h-5 w-5 text-secondary" />
                  <OuiText size="lg" weight="bold">Default</OuiText>
                </OuiFlex>
                <OuiText size="xs" color="muted">
                  No port specified
                </OuiText>
              </template>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Usage Statistics Section -->
      <OuiCard v-if="usageData && usageData.current">
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <OuiStack gap="xs">
              <OuiText size="lg" weight="bold">Monthly Usage</OuiText>
              <OuiText size="xs" color="muted">
                Current month resource usage and estimated costs
              </OuiText>
            </OuiStack>
            <OuiBadge variant="secondary">
              {{ usageData.month }}
            </OuiBadge>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
            <!-- CPU Usage -->
            <OuiStack gap="sm">
              <OuiFlex align="center" justify="between">
                <OuiText size="sm" weight="medium" color="primary">CPU</OuiText>
                <OuiText size="xs" color="muted">
                  {{ formatCPUUsage(Number(usageData.current.cpuCoreSeconds)) }}
                </OuiText>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                {{ formatCurrency(Number(usageData.current.estimatedCostCents) / 100) }} estimated
              </OuiText>
            </OuiStack>

            <!-- Memory Usage -->
            <OuiStack gap="sm">
              <OuiFlex align="center" justify="between">
                <OuiText size="sm" weight="medium" color="primary">Memory</OuiText>
                <OuiText size="xs" color="muted">
                  <OuiByte :value="Number(usageData.current.memoryByteSeconds) / 3600" unit-display="short" />/hr avg
                </OuiText>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                {{ formatUptime(Number(usageData.current.uptimeSeconds)) }} uptime
              </OuiText>
            </OuiStack>

            <!-- Bandwidth Usage -->
            <OuiStack gap="sm">
              <OuiFlex align="center" justify="between">
                <OuiText size="sm" weight="medium" color="primary">Bandwidth</OuiText>
                <OuiText size="xs" color="muted">
                  <OuiByte :value="Number(usageData.current.bandwidthRxBytes) + Number(usageData.current.bandwidthTxBytes)" unit-display="short" />
                </OuiText>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                {{ formatNumber(Number(usageData.current.requestCount)) }} requests
              </OuiText>
            </OuiStack>

            <!-- Storage Usage -->
            <OuiStack gap="sm">
              <OuiFlex align="center" justify="between">
                <OuiText size="sm" weight="medium" color="primary">Storage</OuiText>
                <OuiText size="xs" color="muted">
                  <OuiByte :value="Number(usageData.current.storageBytes)" unit-display="short" />
                </OuiText>
              </OuiFlex>
              <OuiText size="xs" color="muted">
                {{ formatNumber(Number(usageData.current.errorCount)) }} errors
              </OuiText>
            </OuiStack>
          </OuiGrid>
        </OuiCardBody>
      </OuiCard>

      <!-- Cost Breakdown -->
      <OuiCard v-if="usageData && usageData.estimatedMonthly && usageData.current">
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <OuiStack gap="xs">
              <OuiText size="lg" weight="bold">Estimated Monthly Cost</OuiText>
              <OuiText size="xs" color="muted">
                Cost breakdown by resource type
              </OuiText>
            </OuiStack>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiFlex align="center" justify="between" class="pb-3 border-b border-border-muted">
              <OuiStack gap="xs">
                <OuiText size="sm" color="muted">Total Estimated</OuiText>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ formatCurrency(Number(usageData.estimatedMonthly.estimatedCostCents) / 100) }}
                </OuiText>
              </OuiStack>
              <OuiText size="xs" color="muted">
                Current: {{ formatCurrency(Number(usageData.current.estimatedCostCents) / 100) }}
              </OuiText>
            </OuiFlex>
            <OuiGrid cols="1" cols-md="2" gap="sm">
              <OuiBox
                v-for="cost in costBreakdown"
                :key="cost.label"
                p="sm"
                rounded="lg"
                class="bg-surface-muted/40 ring-1 ring-border-muted"
              >
                <OuiFlex align="center" justify="between" gap="md">
                  <OuiFlex align="center" gap="sm" class="flex-1">
                    <OuiBox
                      class="w-3 h-3 rounded-full"
                      :class="cost.color"
                    />
                    <OuiText size="sm" weight="medium" color="primary">
                      {{ cost.label }}
                    </OuiText>
                  </OuiFlex>
                  <OuiText size="sm" weight="semibold" color="primary">
                    {{ cost.value }}
                  </OuiText>
                </OuiFlex>
              </OuiBox>
            </OuiGrid>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Enhanced Live Metrics -->
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <OuiStack gap="xs">
              <OuiText size="lg" weight="bold">Live Metrics</OuiText>
              <OuiText size="xs" color="muted">
                Real-time resource usage from running containers
              </OuiText>
            </OuiStack>
            <OuiFlex align="center" gap="sm">
              <span
                class="h-2 w-2 rounded-full"
                :class="isStreaming ? 'bg-success animate-pulse' : 'bg-secondary'"
              />
              <OuiText size="xs" color="muted">
                {{ isStreaming ? "Live" : "Stopped" }}
              </OuiText>
            </OuiFlex>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="md">
            <!-- CPU Usage -->
            <OuiBox
              p="md"
              rounded="xl"
              class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
            >
              <OuiStack gap="xs">
                <OuiFlex align="center" justify="between">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    CPU Usage
                  </OuiText>
                  <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                    Live
                  </OuiBadge>
                </OuiFlex>
                <OuiText size="2xl" weight="bold">
                  {{ currentCpuUsage.toFixed(1) }}%
                </OuiText>
                <OuiBox
                  w="full"
                  class="h-1 bg-surface-muted overflow-hidden rounded-full mt-1"
                >
                  <OuiBox
                    class="h-full bg-accent-primary transition-all duration-300"
                    :style="{ width: `${Math.min(currentCpuUsage, 100)}%` }"
                  />
                </OuiBox>
                <OuiText size="xs" color="muted">
                  {{ latestMetric ? "Current" : "No data" }}
                </OuiText>
              </OuiStack>
            </OuiBox>

            <!-- Memory Usage -->
            <OuiBox
              p="md"
              rounded="xl"
              class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
            >
              <OuiStack gap="xs">
                <OuiFlex align="center" justify="between">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Memory Usage
                  </OuiText>
                  <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                    Live
                  </OuiBadge>
                </OuiFlex>
                <OuiText size="2xl" weight="bold">
                  <OuiByte :value="currentMemoryUsage" unit-display="short" />
                </OuiText>
                <OuiText size="xs" color="muted">
                  {{ latestMetric ? "Current" : "No data" }}
                </OuiText>
              </OuiStack>
            </OuiBox>

            <!-- Network Rx -->
            <OuiBox
              p="md"
              rounded="xl"
              class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
            >
              <OuiStack gap="xs">
                <OuiFlex align="center" justify="between">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Network Rx
                  </OuiText>
                  <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                    Live
                  </OuiBadge>
                </OuiFlex>
                <OuiText size="2xl" weight="bold">
                  <OuiByte :value="currentNetworkRx" unit-display="short" />
                </OuiText>
                <OuiText size="xs" color="muted">
                  {{ latestMetric ? "Total received" : "No data" }}
                </OuiText>
              </OuiStack>
            </OuiBox>

            <!-- Network Tx -->
            <OuiBox
              p="md"
              rounded="xl"
              class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
            >
              <OuiStack gap="xs">
                <OuiFlex align="center" justify="between">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Network Tx
                  </OuiText>
                  <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                    Live
                  </OuiBadge>
                </OuiFlex>
                <OuiText size="2xl" weight="bold">
                  <OuiByte :value="currentNetworkTx" unit-display="short" />
                </OuiText>
                <OuiText size="xs" color="muted">
                  {{ latestMetric ? "Total sent" : "No data" }}
                </OuiText>
              </OuiStack>
            </OuiBox>
          </OuiGrid>
        </OuiCardBody>
      </OuiCard>

      <!-- Main Information Grid -->
      <OuiGrid cols="1" cols-lg="2" gap="lg">
        <!-- Deployment Details Card -->
        <OuiCard>
          <OuiCardHeader>
            <OuiText size="lg" weight="bold">Deployment Details</OuiText>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiStack gap="md">
              <!-- Domain -->
              <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <GlobeAltIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Domain
                    </OuiText>
                    <OuiText size="sm" weight="medium" truncate>
                      {{ deployment.domain }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
                <OuiButton
                  variant="ghost"
                  size="xs"
                  @click="openDomain"
                  :disabled="!deployment.domain"
                >
                  <ArrowTopRightOnSquareIcon class="h-3 w-3" />
                </OuiButton>
              </div>

              <!-- Custom Domains -->
              <div
                v-if="deployment.customDomains && deployment.customDomains.length > 0"
                class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
              >
                <OuiStack gap="xs" class="min-w-0 flex-1">
                  <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                    Custom Domains
                  </OuiText>
                  <OuiFlex gap="xs" wrap="wrap">
                    <OuiBadge
                      v-for="domain in getDisplayDomains(deployment.customDomains)"
                      :key="domain.domain"
                      :variant="getDomainStatusVariant(domain.status)"
                      size="sm"
                    >
                      {{ domain.domain }}
                      <span v-if="domain.status === 'pending'" class="ml-1">(pending)</span>
                    </OuiBadge>
                  </OuiFlex>
                </OuiStack>
              </div>

              <!-- Environment -->
              <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <CpuChipIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Environment
                    </OuiText>
                    <OuiBadge
                      :variant="getEnvironmentVariant(deployment.environment)"
                      size="sm" 
                    >
                      {{ getEnvironmentLabel(deployment.environment) }}
                    </OuiBadge>
                  </OuiStack>
                </OuiFlex>
              </div>

              <!-- Type & Build Strategy -->
              <div class="flex items-start justify-between gap-4 py-2 border-b border-border-default">
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <CodeBracketIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Type
                    </OuiText>
                    <OuiText size="sm" weight="medium">
                      {{ getTypeLabel((deployment as any).type) }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </div>

              <!-- Build Strategy -->
              <div
                v-if="deployment.buildStrategy !== undefined"
                class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <WrenchScrewdriverIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Build Strategy
                    </OuiText>
                    <OuiText size="sm" weight="medium">
                      {{ getBuildStrategyLabel(deployment.buildStrategy) }}
                    </OuiText>
                  </OuiStack>
                  </OuiFlex>
              </div>

              <!-- Groups -->
              <div
                v-if="deployment.groups && deployment.groups.length > 0"
                class="flex items-start justify-between gap-4 py-2"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <TagIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Groups
                    </OuiText>
                    <OuiFlex gap="xs" wrap="wrap">
                      <OuiBadge
                        v-for="group in deployment.groups"
                        :key="group"
                        variant="secondary"
                        size="sm"
                      >
                        {{ group }}
                      </OuiBadge>
                  </OuiFlex>
                  </OuiStack>
                </OuiFlex>
              </div>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

        <!-- Repository & Runtime Card -->
        <OuiCard>
          <OuiCardHeader>
            <OuiText size="lg" weight="bold">Repository & Runtime</OuiText>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiStack gap="md">
              <!-- Repository URL -->
              <div
                v-if="deployment.repositoryUrl"
                class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <CodeBracketSquareIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Repository
                    </OuiText>
                    <OuiText size="sm" weight="medium" truncate>
                      {{ deployment.repositoryUrl }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
                      <OuiButton
                        variant="ghost"
                  size="xs"
                  @click="openRepository"
                  :disabled="!deployment.repositoryUrl"
                      >
                  <ArrowTopRightOnSquareIcon class="h-3 w-3" />
                      </OuiButton>
              </div>

              <!-- Branch -->
              <div
                v-if="deployment.branch"
                class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <TagIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Branch
                    </OuiText>
                    <OuiText size="sm" weight="medium">
                      {{ deployment.branch }}
                    </OuiText>
                  </OuiStack>
                    </OuiFlex>
              </div>

              <!-- Dockerfile Path -->
              <div
                v-if="deployment.dockerfilePath"
                class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <DocumentTextIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Dockerfile
                    </OuiText>
                    <OuiText size="sm" weight="medium" truncate>
                      {{ deployment.dockerfilePath }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </div>

              <!-- Compose File Path -->
              <div
                v-if="deployment.composeFilePath"
                class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <DocumentTextIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Compose File
                    </OuiText>
                    <OuiText size="sm" weight="medium" truncate>
                      {{ deployment.composeFilePath }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </div>

              <!-- Port -->
              <div
                v-if="deployment.port"
                class="flex items-start justify-between gap-4 py-2 border-b border-border-default"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <SignalIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Port
                    </OuiText>
                    <OuiText size="sm" weight="medium">
                      {{ deployment.port }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </div>

              <!-- Image -->
              <div
                v-if="deployment.image"
                class="flex items-start justify-between gap-4 py-2"
              >
                <OuiFlex align="center" gap="sm" class="min-w-0 flex-1">
                  <CubeIcon class="h-4 w-4 text-secondary shrink-0" />
                  <OuiStack gap="xs" class="min-w-0 flex-1">
                    <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                      Image
                    </OuiText>
                    <OuiText size="sm" weight="medium" truncate>
                      {{ deployment.image }}
                    </OuiText>
              </OuiStack>
                </OuiFlex>
              </div>

              <!-- No Repository Message -->
              <div
                v-if="!deployment.repositoryUrl"
                class="py-4 text-center"
              >
                <OuiText size="sm" color="muted">
                  No repository configured
                </OuiText>
              </div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
            </OuiGrid>
            
      <!-- Quick Links -->
      <OuiCard>
        <OuiCardHeader>
          <OuiText size="lg" weight="bold">Quick Links</OuiText>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiGrid cols="2" cols-md="4" gap="md">
            <OuiButton
              variant="ghost"
              size="sm"
              class="justify-start"
              @click="$emit('navigate', 'metrics')"
            >
              <ChartBarIcon class="h-4 w-4 mr-2" />
              <OuiText as="span" size="sm">Metrics</OuiText>
            </OuiButton>
            <OuiButton
              variant="ghost"
              size="sm"
              class="justify-start"
              @click="$emit('navigate', 'logs')"
            >
              <DocumentTextIcon class="h-4 w-4 mr-2" />
              <OuiText as="span" size="sm">Logs</OuiText>
            </OuiButton>
            <OuiButton
              variant="ghost"
              size="sm"
              class="justify-start"
              @click="$emit('navigate', 'terminal')"
            >
              <CommandLineIcon class="h-4 w-4 mr-2" />
              <OuiText as="span" size="sm">Terminal</OuiText>
            </OuiButton>
            <OuiButton
              variant="ghost"
              size="sm"
              class="justify-start"
              @click="$emit('navigate', 'files')"
            >
              <FolderIcon class="h-4 w-4 mr-2" />
              <OuiText as="span" size="sm">Files</OuiText>
            </OuiButton>
            </OuiGrid>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted, watch } from "vue";
import {
  CodeBracketIcon,
  CpuChipIcon,
  ClockIcon,
  ArchiveBoxIcon,
  GlobeAltIcon,
  ArrowTopRightOnSquareIcon,
  WrenchScrewdriverIcon,
  TagIcon,
  CodeBracketSquareIcon,
  DocumentTextIcon,
  SignalIcon,
  CubeIcon,
  ChartBarIcon,
  CommandLineIcon,
  FolderIcon,
  CalendarIcon,
  HeartIcon,
  CheckCircleIcon,
  ExclamationTriangleIcon,
  XCircleIcon,
} from "@heroicons/vue/24/outline";
import type { Deployment } from "@obiente/proto";
import {
  DeploymentType,
  DeploymentStatus,
  Environment as EnvEnum,
  BuildStrategy,
  DeploymentService,
} from "@obiente/proto";
import { date } from "@obiente/proto/utils";
import { useConnectClient } from "~/lib/connect-client";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import OuiByte from "~/components/oui/Byte.vue";

interface Props {
  deployment: Deployment;
  organizationId?: string;
}

const props = defineProps<Props>();

defineEmits<{
  navigate: [tab: string];
}>();

const client = useConnectClient(DeploymentService);

// Fetch deployment usage data
const { data: usageData } = await useAsyncData(
  () => `deployment-usage-${props.deployment.id}`,
  async () => {
    if (!props.deployment?.id || !props.organizationId) return null;
    try {
      const res = await client.getDeploymentUsage({
        deploymentId: props.deployment.id,
        organizationId: props.organizationId,
      });
      return res;
    } catch (err) {
      console.error("Failed to fetch deployment usage:", err);
      return null;
    }
  },
  { watch: [() => props.deployment.id, () => props.organizationId], server: false }
);

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

// Format bytes helper
const formatBytes = (bytes: number | bigint) => {
  const b = Number(bytes);
  if (b === 0) return "0 B";
  if (b < 1024) return `${b} B`;
  if (b < 1024 * 1024) return `${(b / 1024).toFixed(2)} KB`;
  if (b < 1024 * 1024 * 1024) return `${(b / (1024 * 1024)).toFixed(2)} MB`;
  return `${(b / (1024 * 1024 * 1024)).toFixed(2)} GB`;
};

const formatCurrency = (amount: number) =>
  new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount);

const formatNumber = (value: number) =>
  new Intl.NumberFormat("en-US").format(value);

const formatCPUUsage = (coreSeconds: number): string => {
  const hours = coreSeconds / 3600;
  if (hours < 1) {
    return `${(coreSeconds / 60).toFixed(1)} min`;
  }
  return `${hours.toFixed(1)} core-hrs`;
};

const formatUptime = (seconds: number): string => {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
};


// Cost breakdown
const costBreakdown = computed(() => {
  if (!usageData.value?.estimatedMonthly) return [];
  const estimated = usageData.value.estimatedMonthly;
  const breakdown = [];
  
  const totalCents = Number(estimated.estimatedCostCents);
  if (totalCents > 0) {
    // Use actual cost breakdown if available, otherwise estimate
    breakdown.push(
      { 
        label: "CPU", 
        value: formatCurrency(estimated.cpuCostCents ? Number(estimated.cpuCostCents) / 100 : totalCents * 0.4 / 100), 
        color: "bg-accent-primary" 
      },
      { 
        label: "Memory", 
        value: formatCurrency(estimated.memoryCostCents ? Number(estimated.memoryCostCents) / 100 : totalCents * 0.3 / 100), 
        color: "bg-success" 
      },
      { 
        label: "Bandwidth", 
        value: formatCurrency(estimated.bandwidthCostCents ? Number(estimated.bandwidthCostCents) / 100 : totalCents * 0.2 / 100), 
        color: "bg-accent-secondary" 
      },
      { 
        label: "Storage", 
        value: formatCurrency(estimated.storageCostCents ? Number(estimated.storageCostCents) / 100 : totalCents * 0.1 / 100), 
        color: "bg-warning" 
      },
    );
  }
  
  return breakdown;
});

// Start streaming metrics
const startStreaming = async () => {
  if (isStreaming.value || streamController.value || !props.deployment?.id) {
    return;
  }

  isStreaming.value = true;
  streamController.value = new AbortController();

  try {
    const request: any = {
      deploymentId: props.deployment.id,
      organizationId: props.organizationId || "",
      intervalSeconds: 5,
      aggregate: true, // Get aggregated metrics for all containers
    };

    if (!request.organizationId) {
      console.warn("No organizationId provided for metrics streaming");
      isStreaming.value = false;
      streamController.value = null;
      return;
    }

    const stream = await (client as any).streamDeploymentMetrics(request, {
      signal: streamController.value.signal,
    });

    for await (const metric of stream) {
      if (streamController.value?.signal.aborted) {
        break;
      }
      latestMetric.value = metric;
    }
  } catch (err: any) {
    if (err.name === "AbortError") {
      return;
    }
    // Suppress "missing trailer" errors
    const isMissingTrailerError =
      err.message?.toLowerCase().includes("missing trailer") ||
      err.message?.toLowerCase().includes("trailer") ||
      err.code === "unknown";

    if (!isMissingTrailerError) {
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

// Start streaming when component mounts if deployment is running
onMounted(() => {
  if (props.deployment?.status === DeploymentStatus.RUNNING) {
    startStreaming();
  }
});

// Watch deployment status and start/stop streaming accordingly
watch(
  () => props.deployment?.status,
  (status) => {
    if (status === DeploymentStatus.RUNNING && !isStreaming.value) {
      startStreaming();
    } else if (status !== DeploymentStatus.RUNNING && isStreaming.value) {
      stopStreaming();
    }
  }
);

// Clean up on unmount
onUnmounted(() => {
  stopStreaming();
});

const getTypeLabel = (t: DeploymentType | number | undefined) => {
  switch (t) {
    case DeploymentType.DOCKER:
      return "Docker";
    case DeploymentType.STATIC:
      return "Static Site";
    case DeploymentType.NODE:
      return "Node.js";
    case DeploymentType.GO:
      return "Go";
    case DeploymentType.PYTHON:
      return "Python";
    case DeploymentType.RUBY:
      return "Ruby";
    case DeploymentType.RUST:
      return "Rust";
    case DeploymentType.JAVA:
      return "Java";
    case DeploymentType.PHP:
      return "PHP";
    case DeploymentType.GENERIC:
      return "Generic";
    default:
      return "Custom";
  }
};

const getBuildStrategyLabel = (strategy: BuildStrategy | number | undefined) => {
  switch (strategy) {
    case BuildStrategy.RAILPACK:
      return "Railpack";
    case BuildStrategy.NIXPACKS:
      return "Nixpacks";
    case BuildStrategy.DOCKERFILE:
      return "Dockerfile";
    case BuildStrategy.PLAIN_COMPOSE:
      return "Docker Compose";
    case BuildStrategy.COMPOSE_REPO:
      return "Compose from Repo";
    case BuildStrategy.STATIC_SITE:
      return "Static Site";
    default:
      return "Unknown";
  }
};

const getEnvironmentLabel = (env: string | EnvEnum | number) => {
  if (typeof env === "number") {
    switch (env) {
      case EnvEnum.PRODUCTION:
        return "Production";
      case EnvEnum.STAGING:
        return "Staging";
      case EnvEnum.DEVELOPMENT:
        return "Development";
      default:
        return "Environment";
    }
  }
  return String(env);
};

const getEnvironmentVariant = (env: string | EnvEnum | number): "success" | "warning" | "secondary" => {
  if (typeof env === "number") {
    switch (env) {
      case EnvEnum.PRODUCTION:
        return "success";
      case EnvEnum.STAGING:
        return "warning";
      case EnvEnum.DEVELOPMENT:
        return "secondary";
      default:
        return "secondary";
    }
  }
  return "secondary";
};

const getDisplayDomains = (customDomains: string[]) => {
  return customDomains.map((entry) => {
    const parts = entry.split(":");
    const domain = parts[0] || "";
    let status = "pending";
    
    if (parts.length >= 4 && parts[1] === "token" && parts[3]) {
      status = parts[3];
    } else if (parts.length >= 2 && parts[1] === "verified") {
      status = "verified";
    }
    
    return { domain, status };
  });
};

const getDomainStatusVariant = (status: string): "success" | "warning" | "danger" | "secondary" => {
  switch (status) {
    case "verified":
      return "success";
    case "failed":
      return "danger";
    case "expired":
      return "warning";
    default:
      return "secondary";
  }
};

const getStatusLabel = (status: DeploymentStatus | number) => {
  switch (status) {
    case DeploymentStatus.RUNNING:
      return "Running";
    case DeploymentStatus.STOPPED:
      return "Stopped";
    case DeploymentStatus.BUILDING:
      return "Building";
    case DeploymentStatus.DEPLOYING:
      return "Deploying";
    case DeploymentStatus.FAILED:
      return "Failed";
    case DeploymentStatus.CREATED:
      return "Created";
    default:
      return "Unknown";
  }
};

const getStatusDotClass = (status: DeploymentStatus | number) => {
  switch (status) {
    case DeploymentStatus.RUNNING:
      return "bg-success animate-pulse";
    case DeploymentStatus.STOPPED:
      return "bg-danger";
    case DeploymentStatus.BUILDING:
    case DeploymentStatus.DEPLOYING:
      return "bg-warning animate-pulse";
    case DeploymentStatus.FAILED:
      return "bg-danger";
    default:
      return "bg-secondary";
  }
};

const getContainerStatusVariant = () => {
  const running = props.deployment.containersRunning ?? 0;
  const total = props.deployment.containersTotal ?? 0;
  
  if (running === 0) return "danger";
  if (running === total) return "success";
  return "warning";
};

const getContainerStatusLabel = () => {
  const running = props.deployment.containersRunning ?? 0;
  const total = props.deployment.containersTotal ?? 0;
  
  if (running === 0) return "Stopped";
  if (running === total) return "Running";
  return "Partial";
};

const formatBuildTime = (seconds: number) => {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
};

const getHealthIcon = (healthStatus: string) => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return CheckCircleIcon;
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return XCircleIcon;
  }
  if (status.includes("warning") || status === "degraded") {
    return ExclamationTriangleIcon;
  }
  return HeartIcon;
};

const getHealthIconClass = (healthStatus: string) => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return "text-success";
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return "text-danger";
  }
  if (status.includes("warning") || status === "degraded") {
    return "text-warning";
  }
  return "text-secondary";
};

const getHealthVariant = (healthStatus: string): "success" | "warning" | "danger" | "secondary" => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return "success";
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return "danger";
  }
  if (status.includes("warning") || status === "degraded") {
    return "warning";
  }
  return "secondary";
};

const getHealthLabel = (healthStatus: string) => {
  const status = healthStatus.toLowerCase();
  if (status.includes("healthy") || status === "ok" || status === "up") {
    return "Healthy";
  }
  if (status.includes("unhealthy") || status.includes("down") || status === "error") {
    return "Unhealthy";
  }
  if (status.includes("warning") || status === "degraded") {
    return "Degraded";
  }
  return "Unknown";
};

const formatDate = (timestamp: any) => {
  if (!timestamp) return "";
  const d = date(timestamp);
  if (!d) return "";
  return d.toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
};

const openDomain = () => {
  if (props.deployment.domain) {
    window.open(`https://${props.deployment.domain}`, "_blank");
  }
};

const openRepository = () => {
  if (props.deployment.repositoryUrl) {
    window.open(props.deployment.repositoryUrl, "_blank");
  }
};
</script>