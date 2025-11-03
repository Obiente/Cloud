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
                  {{ deployment.size }}
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

      <!-- Quick Metrics Overview -->
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <OuiText size="lg" weight="bold">Live Metrics</OuiText>
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
              class="ring-1 ring-border-muted bg-surface-muted/30"
            >
              <OuiStack gap="xs">
                <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                  CPU Usage
                </OuiText>
                <OuiText size="2xl" weight="bold">
                  {{ currentCpuUsage.toFixed(1) }}%
                </OuiText>
                <OuiText size="xs" color="muted">
                  {{ latestMetric ? "Current" : "No data" }}
                </OuiText>
              </OuiStack>
            </OuiBox>

            <!-- Memory Usage -->
            <OuiBox
              p="md"
              rounded="xl"
              class="ring-1 ring-border-muted bg-surface-muted/30"
            >
              <OuiStack gap="xs">
                <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                  Memory Usage
                </OuiText>
                <OuiText size="2xl" weight="bold">
                  {{ formatBytes(currentMemoryUsage) }}
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
              class="ring-1 ring-border-muted bg-surface-muted/30"
            >
              <OuiStack gap="xs">
                <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                  Network Rx
                </OuiText>
                <OuiText size="2xl" weight="bold">
                  {{ formatBytes(currentNetworkRx) }}
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
              class="ring-1 ring-border-muted bg-surface-muted/30"
            >
              <OuiStack gap="xs">
                <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                  Network Tx
                </OuiText>
                <OuiText size="2xl" weight="bold">
                  {{ formatBytes(currentNetworkTx) }}
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
                      v-for="domain in deployment.customDomains"
                      :key="domain"
                      variant="outline"
                      size="sm"
                    >
                      {{ domain }}
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
    case BuildStrategy.RAILPACKS:
      return "Railpacks";
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