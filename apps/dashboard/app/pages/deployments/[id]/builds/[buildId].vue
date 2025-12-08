<template>
  <OuiContainer>
    <OuiStack gap="lg">
      <!-- Header -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
              <OuiStack gap="xs">
                <OuiFlex align="center" gap="sm">
                  <NuxtLink
                    :to="`/deployments/${deploymentId}?tab=builds`"
                    class="text-secondary hover:text-primary transition-colors"
                  >
                    <ArrowLeftIcon class="h-5 w-5" />
                  </NuxtLink>
                  <OuiText as="h1" size="2xl" weight="bold">
                    Build #{{ buildNumber }} Logs
                  </OuiText>
                </OuiFlex>
                <OuiText size="sm" color="secondary">
                  Deployment: {{ deploymentId }}
                </OuiText>
              </OuiStack>

              <OuiFlex gap="sm" align="center" wrap="wrap">
                <OuiBadge
                  :variant="getBuildStatusVariant(build?.status)"
                  size="sm"
                  v-if="build"
                >
                  <span
                    class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                    :class="getBuildStatusDotClass(build.status)"
                  />
                  {{ getBuildStatusLabel(build.status) }}
                </OuiBadge>
                <OuiText size="xs" color="secondary" v-if="build?.startedAt">
                  <OuiRelativeTime
                    :value="build.startedAt ? date(build.startedAt) : undefined"
                    :style="'short'"
                  />
                </OuiText>
              </OuiFlex>
            </OuiFlex>

            <!-- Build Info -->
            <OuiGrid :cols="{ sm: 1, md: 2, lg: 3 }" gap="sm" v-if="build">
              <div v-if="build.branch">
                <OuiText size="xs" color="muted" class="mb-1">Branch</OuiText>
                <OuiText size="sm" weight="medium" class="font-mono">{{ build.branch }}</OuiText>
              </div>
              <div v-if="build.buildTime > 0">
                <OuiText size="xs" color="muted" class="mb-1">Duration</OuiText>
                <OuiText size="sm" weight="medium">{{ formatBuildTime(build.buildTime) }}</OuiText>
              </div>
              <div v-if="build.size">
                <OuiText size="xs" color="muted" class="mb-1">Size</OuiText>
                <OuiText size="sm" weight="medium">{{ build.size }}</OuiText>
              </div>
            </OuiGrid>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Logs -->
      <OuiCard>
        <OuiCardBody>
          <OuiLogs
            :logs="formattedLogs"
            :is-loading="isLoading"
            :show-timestamps="showTimestamps"
            :enable-ansi="true"
            :auto-scroll="false"
            height="600px"
            empty-message="No logs available for this build."
            loading-message="Loading build logs..."
          >
            <template #footer>
              <OuiFlex justify="between" align="center" class="pt-4 border-t border-border-default">
                <OuiText size="xs" color="secondary">
                  {{ logs.length }} log line{{ logs.length !== 1 ? 's' : '' }}
                </OuiText>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="showTimestamps = !showTimestamps"
                >
                  {{ showTimestamps ? 'Hide' : 'Show' }} Timestamps
                </OuiButton>
              </OuiFlex>
            </template>
          </OuiLogs>
        </OuiCardBody>
      </OuiCard>

      <!-- Error Message -->
      <OuiCard
        v-if="build?.error"
        variant="outline"
        class="border-danger/20 bg-danger/5"
      >
        <OuiCardBody>
          <OuiText size="sm" weight="semibold" color="danger" class="mb-2">
            Build Error
          </OuiText>
          <OuiText size="xs" color="danger" class="font-mono whitespace-pre-wrap">
            {{ build.error }}
          </OuiText>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowLeftIcon } from "@heroicons/vue/24/outline";
import {
  DeploymentService,
  BuildStatus,
  type Build,
  type DeploymentLogLine,
} from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { date } from "@obiente/proto/utils";
import type { LogEntry } from "~/components/oui/Logs.vue";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import { useOrganizationsStore } from "~/stores/organizations";

definePageMeta({
  layout: "default",
  middleware: "auth",
});

const route = useRoute();
const router = useRouter();
const orgsStore = useOrganizationsStore();

const deploymentId = computed(() => String(route.params.id));
const buildId = computed(() => String(route.params.buildId));
const organizationId = computed(() => orgsStore.currentOrgId || "");

const client = useConnectClient(DeploymentService);
const build = ref<Build | null>(null);
const logs = ref<DeploymentLogLine[]>([]);
const isLoading = ref(false);
const showTimestamps = ref(true);

// Convert logs to LogEntry format
const formattedLogs = computed<LogEntry[]>(() => {
  return logs.value.map((log, idx) => ({
    id: idx,
    line: log.line,
    timestamp: log.timestamp
      ? new Date(
          Number(log.timestamp.seconds) * 1000 +
            Number(log.timestamp.nanos || 0) / 1e6
        ).toISOString()
      : new Date().toISOString(),
    stderr: log.stderr || false,
    level: log.logLevel === 5 ? "error" : log.logLevel === 4 ? "warning" : "info",
  }));
});

const buildNumber = computed(() => build.value?.buildNumber || 0);

const loadBuild = async () => {
  if (!organizationId.value || !deploymentId.value || !buildId.value) return;

  isLoading.value = true;
  try {
    const buildResponse = await client.getBuild({
      organizationId: organizationId.value,
      deploymentId: deploymentId.value,
      buildId: buildId.value,
    });

    if (buildResponse.build) {
      build.value = buildResponse.build;
    }
  } catch (error: any) {
    console.error("Failed to load build:", error);
    router.push(`/deployments/${deploymentId.value}?tab=builds`);
  } finally {
    isLoading.value = false;
  }
};

const loadBuildLogs = async () => {
  if (!organizationId.value || !deploymentId.value || !buildId.value) return;

  isLoading.value = true;
  try {
    const logsResponse = await client.getBuildLogs({
      organizationId: organizationId.value,
      deploymentId: deploymentId.value,
      buildId: buildId.value,
      limit: 10000, // Large limit to get all logs
    });

    logs.value = logsResponse.logs;
  } catch (error: any) {
    console.error("Failed to load build logs:", error);
    logs.value = [];
  } finally {
    isLoading.value = false;
  }
};

const getBuildStatusLabel = (status: BuildStatus) => {
  switch (status) {
    case BuildStatus.BUILD_PENDING:
      return "Pending";
    case BuildStatus.BUILD_BUILDING:
      return "Building";
    case BuildStatus.BUILD_SUCCESS:
      return "Success";
    case BuildStatus.BUILD_FAILED:
      return "Failed";
    default:
      return "Unknown";
  }
};

const getBuildStatusVariant = (
  status: BuildStatus
): "success" | "warning" | "danger" | "secondary" => {
  switch (status) {
    case BuildStatus.BUILD_SUCCESS:
      return "success";
    case BuildStatus.BUILD_FAILED:
      return "danger";
    case BuildStatus.BUILD_BUILDING:
    case BuildStatus.BUILD_PENDING:
      return "warning";
    default:
      return "secondary";
  }
};

const getBuildStatusDotClass = (status: BuildStatus) => {
  switch (status) {
    case BuildStatus.BUILD_SUCCESS:
      return "bg-success";
    case BuildStatus.BUILD_FAILED:
      return "bg-danger";
    case BuildStatus.BUILD_BUILDING:
    case BuildStatus.BUILD_PENDING:
      return "bg-warning";
    default:
      return "bg-secondary";
  }
};

const formatBuildTime = (seconds: number) => {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0
    ? `${minutes}m ${remainingSeconds}s`
    : `${minutes}m`;
};

onMounted(async () => {
  await Promise.all([loadBuild(), loadBuildLogs()]);
});

watch(
  () => [deploymentId.value, buildId.value, organizationId.value],
  () => {
    if (organizationId.value && deploymentId.value && buildId.value) {
      loadBuild();
      loadBuildLogs();
    }
  }
);
</script>

