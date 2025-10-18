<template>
  <OuiContainer size="7xl" py="xl" class="min-h-screen">
    <OuiStack gap="xl">
      <!-- Header -->
      <OuiFlex justify="between" align="start" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="min-w-0">
          <OuiFlex align="center" gap="md">
            <OuiBox p="sm" rounded="xl" bg="accent-primary" class="bg-primary/10 ring-1 ring-primary/20">
              <RocketLaunchIcon class="w-6 h-6 text-primary" />
            </OuiBox>
            <OuiText as="h1" size="2xl" weight="bold" truncate>
              {{ deployment.name }}
            </OuiText>
          </OuiFlex>
          <OuiFlex align="center" gap="md" wrap="wrap">
            <OuiBadge :variant="statusMeta.badge">
              <span class="inline-flex h-1.5 w-1.5 rounded-full" :class="statusMeta.dotClass" />
              <OuiText as="span" size="xs" weight="semibold" transform="uppercase">{{ statusMeta.label }}</OuiText>
            </OuiBadge>
            <OuiText size="sm" color="secondary">Last deployed {{ formatRelativeTime(deployment.lastDeployedAt) }}</OuiText>
          </OuiFlex>
        </OuiStack>

        <OuiFlex gap="sm" wrap="wrap">
          <OuiButton variant="ghost" size="sm" @click="openDomain" class="gap-2">
            <ArrowTopRightOnSquareIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Open</OuiText>
          </OuiButton>
          <OuiButton variant="ghost" color="warning" size="sm" @click="redeploy" class="gap-2">
            <ArrowPathIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Redeploy</OuiText>
          </OuiButton>
          <OuiButton v-if="deployment.status==='RUNNING'" variant="solid" color="danger" size="sm" @click="stop" class="gap-2">
            <StopIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Stop</OuiText>
          </OuiButton>
          <OuiButton v-else variant="solid" color="success" size="sm" @click="start" class="gap-2">
            <PlayIcon class="h-4 w-4" />
            <OuiText as="span" size="xs" weight="medium">Start</OuiText>
          </OuiButton>
        </OuiFlex>
      </OuiFlex>

      <!-- Content layout -->
      <OuiGrid cols="1" :cols-xl="3" gap="lg">
        <!-- Main column -->
        <div class="xl:col-span-2 space-y-6">
          <!-- Overview -->
          <OuiCard variant="raised">
            <OuiCardHeader>
              <OuiText as="h2" size="lg" weight="semibold">Overview</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiGrid cols="1" :cols-md="2" gap="md">
                <OuiBox p="md" rounded="xl" class="ring-1 ring-border-muted bg-surface-muted/30">
                  <OuiText size="xs" color="secondary" transform="uppercase" weight="bold">Domain</OuiText>
                  <OuiFlex align="center" gap="sm" class="mt-1">
                    <Icon name="uil:globe" class="h-4 w-4 text-secondary" />
                    <OuiText size="sm" weight="medium">{{ deployment.domain }}</OuiText>
                  </OuiFlex>
                </OuiBox>
                <OuiBox p="md" rounded="xl" class="ring-1 ring-border-muted bg-surface-muted/30">
                  <OuiText size="xs" color="secondary" transform="uppercase" weight="bold">Framework</OuiText>
                  <OuiFlex align="center" gap="sm" class="mt-1">
                    <CodeBracketIcon class="h-4 w-4 text-primary" />
                    <OuiText size="sm" weight="medium">{{ deployment.framework }}</OuiText>
                  </OuiFlex>
                </OuiBox>
                <OuiBox p="md" rounded="xl" class="ring-1 ring-border-muted bg-surface-muted/30">
                  <OuiText size="xs" color="secondary" transform="uppercase" weight="bold">Environment</OuiText>
                  <OuiFlex align="center" gap="sm" class="mt-1">
                    <CpuChipIcon class="h-4 w-4 text-secondary" />
                    <OuiText size="sm" weight="medium">{{ deployment.environment }}</OuiText>
                  </OuiFlex>
                </OuiBox>
                <OuiBox p="md" rounded="xl" class="ring-1 ring-border-muted bg-surface-muted/30">
                  <OuiText size="xs" color="secondary" transform="uppercase" weight="bold">Build Time</OuiText>
                  <OuiText size="lg" weight="bold">{{ deployment.buildTime }}s</OuiText>
                </OuiBox>
              </OuiGrid>
            </OuiCardBody>
          </OuiCard>

          <!-- Configuration -->
          <OuiCard variant="raised">
            <OuiCardHeader>
              <OuiFlex justify="between" align="center">
                <OuiText as="h3" size="lg" weight="semibold">Configuration</OuiText>
                <OuiButton size="sm" variant="ghost" @click="saveConfig" :disabled="!isConfigDirty">Save</OuiButton>
              </OuiFlex>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiInput v-model="config.repositoryUrl" label="Repository URL" placeholder="https://github.com/org/repo" />
                <OuiGrid cols="1" :cols-md="2" gap="md">
                  <OuiInput v-model="config.branch" label="Branch" placeholder="main" />
                  <OuiSelect v-model="config.runtime" :items="runtimeOptions" label="Runtime" placeholder="Select runtime" />
                </OuiGrid>
                <OuiGrid cols="1" :cols-md="2" gap="md">
                  <OuiInput v-model="config.installCommand" label="Install Command" placeholder="pnpm install" />
                  <OuiInput v-model="config.buildCommand" label="Build Command" placeholder="pnpm build" />
                </OuiGrid>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Logs -->
          <OuiCard variant="raised">
            <OuiCardHeader>
              <OuiText as="h3" size="lg" weight="semibold">Recent Logs</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <pre class="bg-black text-green-400 p-4 rounded-xl text-xs overflow-auto max-h-64">{{ logs }}</pre>
            </OuiCardBody>
          </OuiCard>
        </div>

        <!-- Sidebar column -->
        <div class="space-y-6">
          <OuiCard variant="raised">
            <OuiCardHeader>
              <OuiText as="h3" size="base" weight="semibold">Actions</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiButton size="sm" color="warning" variant="solid" @click="redeploy" class="gap-2">
                  <ArrowPathIcon class="h-4 w-4" /> Redeploy
                </OuiButton>
                <OuiButton size="sm" color="secondary" variant="ghost" @click="copyDomain" class="gap-2">
                  <Icon name="uil:copy" class="h-4 w-4" /> Copy Domain
                </OuiButton>
                <OuiButton size="sm" color="danger" variant="ghost" @click="deleteDeployment" class="gap-2">
                  <Icon name="uil:trash" class="h-4 w-4" /> Delete
                </OuiButton>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <OuiCard variant="raised">
            <OuiCardHeader>
              <OuiText as="h3" size="base" weight="semibold">Build Info</OuiText>
            </OuiCardHeader>
            <OuiCardBody>
              <OuiStack gap="sm">
                <OuiFlex justify="between"><OuiText size="sm" color="secondary">Size</OuiText><OuiText size="sm" weight="medium">{{ deployment.size }}</OuiText></OuiFlex>
                <OuiFlex justify="between"><OuiText size="sm" color="secondary">Framework</OuiText><OuiText size="sm" weight="medium">{{ deployment.framework }}</OuiText></OuiFlex>
                <OuiFlex justify="between"><OuiText size="sm" color="secondary">Environment</OuiText><OuiText size="sm" weight="medium">{{ deployment.environment }}</OuiText></OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </div>
      </OuiGrid>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRoute } from 'vue-router'
import {
  ArrowPathIcon,
  ArrowTopRightOnSquareIcon,
  CodeBracketIcon,
  CpuChipIcon,
  PlayIcon,
  RocketLaunchIcon,
  StopIcon,
} from '@heroicons/vue/24/outline'

const route = useRoute()
const id = computed(() => String(route.params.id))

// Mock initial state; in real app fetch by id
const deployment = reactive({
  id: id.value,
  name: 'New Deployment',
  domain: `${id.value}.obiente.cloud`,
  status: 'BUILDING',
  lastDeployedAt: new Date(),
  framework: 'Custom',
  environment: 'development',
  buildTime: 0,
  size: '--',
})

const STATUS_META = {
  RUNNING: { badge: 'success', label: 'Running', dotClass: 'bg-success' },
  STOPPED: { badge: 'danger', label: 'Stopped', dotClass: 'bg-danger' },
  BUILDING: { badge: 'warning', label: 'Building', dotClass: 'bg-warning' },
  FAILED: { badge: 'danger', label: 'Failed', dotClass: 'bg-danger' },
  DEFAULT: { badge: 'secondary', label: 'Unknown', dotClass: 'bg-secondary' },
} as const

type StatusKey = keyof typeof STATUS_META
const statusMeta = computed(() => STATUS_META[(deployment.status as StatusKey) || 'DEFAULT'] || STATUS_META.DEFAULT)

const config = reactive({
  repositoryUrl: '',
  branch: 'main',
  runtime: 'node',
  installCommand: 'pnpm install',
  buildCommand: 'pnpm build',
})

const initialConfig = JSON.stringify(config)
const isConfigDirty = computed(() => JSON.stringify(config) !== initialConfig)

const runtimeOptions = [
  { label: 'Node.js', value: 'node' },
  { label: 'Go', value: 'go' },
  { label: 'Docker', value: 'docker' },
  { label: 'Static', value: 'static' },
]

const logs = ref('[info] Initializing build...\n[info] Installing dependencies...\n')

const formatRelativeTime = (date: Date) => {
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)
  const diffDay = Math.floor(diffHour / 24)
  if (diffSec < 60) return 'just now'
  if (diffMin < 60) return `${diffMin}m ago`
  if (diffHour < 24) return `${diffHour}h ago`
  if (diffDay < 7) return `${diffDay}d ago`
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric' }).format(date)
}

function openDomain() {
  window.open(`https://${deployment.domain}`, '_blank')
}

function copyDomain() {
  navigator.clipboard?.writeText(deployment.domain)
}

function start() {
  deployment.status = 'BUILDING'
  setTimeout(() => (deployment.status = 'RUNNING'), 1500)
}

function stop() {
  deployment.status = 'STOPPED'
}

function redeploy() {
  deployment.status = 'BUILDING'
  deployment.lastDeployedAt = new Date()
  logs.value += `[info] Triggering redeploy at ${new Date().toISOString()}\n`
  setTimeout(() => (deployment.status = 'RUNNING'), 2000)
}

function deleteDeployment() {
  // TODO: call API and navigate back
  navigateTo('/deployments')
}

function saveConfig() {
  // TODO: persist config via API
}
</script>

<style scoped>
.log-line-enter-active,
.log-line-leave-active { transition: opacity .2s; }
.log-line-enter-from,
.log-line-leave-to { opacity: 0; }
</style>
