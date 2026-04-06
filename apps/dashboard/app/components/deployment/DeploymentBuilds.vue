<template>
    <OuiStack gap="md">
      <!-- Header -->
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="sm" weight="semibold">Build History</OuiText>
        <OuiBadge v-if="!isLoading && total > 0" variant="secondary" size="xs">
          {{ total }} build{{ total !== 1 ? 's' : '' }}
        </OuiBadge>
      </OuiFlex>

      <!-- Loading State -->
      <div v-if="isLoading" class="flex justify-center items-center py-12">
        <OuiStack gap="sm" align="center">
          <ArrowPathIcon class="h-5 w-5 text-secondary animate-spin" />
          <OuiText size="sm" color="tertiary">Loading builds...</OuiText>
        </OuiStack>
      </div>

      <!-- Empty State -->
      <OuiCard v-else-if="builds.length === 0" variant="outline">
        <OuiCardBody>
          <OuiStack gap="md" align="center" class="py-8">
            <div class="h-12 w-12 rounded-xl bg-surface-muted flex items-center justify-center">
              <CubeIcon class="h-6 w-6 text-secondary" />
            </div>
            <OuiStack gap="xs" align="center">
              <OuiText size="sm" weight="semibold">No builds yet</OuiText>
              <OuiText size="xs" color="tertiary" class="text-center max-w-sm">
                Build history will appear here once you trigger a deployment.
              </OuiText>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Builds Timeline -->
      <div v-else class="relative">
        <!-- Timeline line -->
        <div class="absolute left-[15px] top-4 bottom-4 w-px bg-border-default" />

        <OuiStack gap="none">
          <div
            v-for="(build, index) in builds"
            :key="build.id"
            class="relative pl-10 pb-6 last:pb-0 group"
          >
            <!-- Timeline dot -->
            <div class="absolute left-[8px] top-1.5 z-10">
              <span
                class="flex h-[14px] w-[14px] items-center justify-center rounded-full border-2 border-surface-base"
                :class="getBuildDotColor(build.status)"
              >
                <span
                  v-if="build.status === BuildStatus.BUILD_BUILDING || build.status === BuildStatus.BUILD_PENDING"
                  class="h-1.5 w-1.5 rounded-full bg-white/80 animate-pulse"
                />
              </span>
            </div>

            <!-- Build card -->
            <OuiCard
              variant="outline"
              class="transition-colors duration-150"
              :class="{
                'border-success/30': build.status === BuildStatus.BUILD_SUCCESS,
                'border-danger/30': build.status === BuildStatus.BUILD_FAILED,
                'border-warning/30 shadow-[0_0_12px_-3px] shadow-warning/10': build.status === BuildStatus.BUILD_BUILDING || build.status === BuildStatus.BUILD_PENDING,
              }"
            >
              <OuiCardBody>
                <OuiStack gap="sm">
                  <!-- Top row: build number + status + time -->
                  <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
                    <OuiFlex align="center" gap="sm">
                      <OuiText size="sm" weight="semibold">
                        #{{ build.buildNumber }}
                      </OuiText>
                      <OuiBadge :variant="getBuildStatusVariant(build.status)" size="xs">
                        {{ getBuildStatusLabel(build.status) }}
                      </OuiBadge>
                      <OuiText v-if="build.commitSha" size="xs" color="tertiary" class="font-mono">
                        {{ build.commitSha.substring(0, 7) }}
                      </OuiText>
                    </OuiFlex>

                    <OuiText size="xs" color="tertiary">
                      <OuiRelativeTime
                        :value="build.startedAt ? date(build.startedAt) : undefined"
                        :style="'short'"
                      />
                    </OuiText>
                  </OuiFlex>

                  <!-- Build stats row -->
                  <OuiFlex gap="md" align="center" wrap="wrap">
                    <OuiFlex v-if="build.branch" align="center" gap="xs">
                      <svg class="h-3 w-3 text-tertiary shrink-0" viewBox="0 0 16 16" fill="currentColor"><path d="M9.5 3.25a2.25 2.25 0 1 1 3 2.122V6A2.5 2.5 0 0 1 10 8.5H6a1 1 0 0 0-1 1v1.128a2.251 2.251 0 1 1-1.5 0V5.372a2.25 2.25 0 1 1 1.5 0v1.836A2.492 2.492 0 0 1 6 7h4a1 1 0 0 0 1-1v-.628A2.25 2.25 0 0 1 9.5 3.25Z" /></svg>
                      <OuiText size="xs" color="tertiary" class="font-mono">{{ build.branch }}</OuiText>
                    </OuiFlex>
                    <OuiFlex v-if="build.buildTime > 0" align="center" gap="xs">
                      <ClockIcon class="h-3 w-3 text-tertiary shrink-0" />
                      <OuiText size="xs" color="tertiary">{{ formatBuildTime(build.buildTime) }}</OuiText>
                    </OuiFlex>
                    <OuiFlex v-if="build.size" align="center" gap="xs">
                      <ArchiveBoxIcon class="h-3 w-3 text-tertiary shrink-0" />
                      <OuiText size="xs" color="tertiary">{{ build.size }}</OuiText>
                    </OuiFlex>
                  </OuiFlex>

                  <!-- Error Message (inline, no card-in-card) -->
                  <div
                    v-if="build.error"
                    class="rounded-lg bg-danger/5 border border-danger/20 px-3 py-2"
                  >
                    <OuiText size="xs" color="danger" class="font-mono whitespace-pre-wrap line-clamp-3">
                      {{ build.error }}
                    </OuiText>
                  </div>

                  <!-- Actions -->
                  <OuiFlex gap="sm" justify="end">
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      @click="() => viewBuildLogs(build.id)"
                    >
                      <DocumentTextIcon class="h-3.5 w-3.5" />
                      Logs
                    </OuiButton>
                    <OuiButton
                      v-if="build.status === BuildStatus.BUILD_SUCCESS"
                      variant="ghost"
                      size="xs"
                      @click="() => revertToBuild(build.id)"
                      :disabled="isReverting"
                    >
                      <ArrowPathIcon class="h-3.5 w-3.5" />
                      Revert
                    </OuiButton>
                    <OuiButton
                      variant="ghost"
                      size="xs"
                      color="danger"
                      @click="() => deleteBuild(build.id)"
                      :disabled="isDeleting"
                    >
                      <TrashIcon class="h-3.5 w-3.5" />
                      Delete
                    </OuiButton>
                  </OuiFlex>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </div>
        </OuiStack>
      </div>

      <!-- Pagination -->
      <OuiFlex v-if="total > limit" justify="center">
        <OuiButton
          variant="outline"
          size="sm"
          @click="loadMore"
          :disabled="isLoadingMore || builds.length >= total"
        >
          {{ isLoadingMore ? 'Loading...' : `Load more (${builds.length}/${total})` }}
        </OuiButton>
      </OuiFlex>
    </OuiStack>

</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import {
  CubeIcon,
  ArrowPathIcon,
  DocumentTextIcon,
  TrashIcon,
  ClockIcon,
  ArchiveBoxIcon,
} from "@heroicons/vue/24/outline";
import {
  DeploymentService,
  BuildStatus,
  type Build,
} from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { date } from "@obiente/proto/utils";
import { useDialog } from "~/composables/useDialog";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";

interface Props {
  deploymentId: string;
  organizationId: string;
}

const props = defineProps<Props>();
const client = useConnectClient(DeploymentService);
const { showConfirm, showAlert } = useDialog();

const builds = ref<Build[]>([]);
const isLoading = ref(false);
const isLoadingMore = ref(false);
const total = ref(0);
const limit = ref(50);
const offset = ref(0);
const isReverting = ref(false);
const isDeleting = ref(false);

const loadBuilds = async (reset = false) => {
  if (reset) {
    offset.value = 0;
    isLoading.value = true;
  } else {
    isLoadingMore.value = true;
  }

  try {
    const response = await client.listBuilds({
      organizationId: props.organizationId,
      deploymentId: props.deploymentId,
      limit: limit.value,
      offset: offset.value,
    });

    if (reset) {
      builds.value = response.builds;
    } else {
      builds.value.push(...response.builds);
    }
    total.value = response.total;
    offset.value += response.builds.length;
  } catch (error: unknown) {
    console.error("Failed to load builds:", error);
    await showAlert({
      title: "Failed to Load Builds",
      message: (error as Error).message || "Failed to load build history. Please try again.",
    });
  } finally {
    isLoading.value = false;
    isLoadingMore.value = false;
  }
};

const loadMore = () => {
  loadBuilds(false);
};

const revertToBuild = async (buildId: string) => {
  const build = builds.value.find(b => b.id === buildId);
  if (!build) return;

  const confirmed = await showConfirm({
    title: "Revert to Build",
    message: `Are you sure you want to revert to Build #${build.buildNumber}? This will restore the deployment configuration from that build and trigger a new deployment.`,
    confirmLabel: "Revert",
    cancelLabel: "Cancel",
    variant: "danger",
  });

  if (!confirmed) return;

  isReverting.value = true;
  try {
    const response = await client.revertToBuild({
      organizationId: props.organizationId,
      deploymentId: props.deploymentId,
      buildId: buildId,
    });

    await showAlert({
      title: "Reverted Successfully",
      message: `Deployment configuration has been restored from Build #${build.buildNumber}. A new build will be triggered.`,
    });

    // Refresh builds and deployment data
    await loadBuilds(true);
    await refreshNuxtData(`deployment-${props.deploymentId}`);
  } catch (error: unknown) {
    console.error("Failed to revert build:", error);
    await showAlert({
      title: "Failed to Revert",
      message: (error as Error).message || "Failed to revert to the selected build. Please try again.",
    });
  } finally {
    isReverting.value = false;
  }
};

const viewBuildLogs = (buildId: string) => {
  navigateTo(`/deployments/${props.deploymentId}/builds/${buildId}`);
};

const deleteBuild = async (buildId: string) => {
  const build = builds.value.find(b => b.id === buildId);
  if (!build) return;

  const confirmed = await showConfirm({
    title: "Delete Build",
    message: `Are you sure you want to delete Build #${build.buildNumber}? This action cannot be undone and will also delete all associated logs.`,
    confirmLabel: "Delete",
    cancelLabel: "Cancel",
    variant: "danger",
  });

  if (!confirmed) return;

  isDeleting.value = true;
  try {
    await client.deleteBuild({
      organizationId: props.organizationId,
      deploymentId: props.deploymentId,
      buildId: buildId,
    });

    await showAlert({
      title: "Build Deleted",
      message: `Build #${build.buildNumber} has been deleted successfully.`,
    });

    // Remove the build from the list
    builds.value = builds.value.filter(b => b.id !== buildId);
    total.value = Math.max(0, total.value - 1);
  } catch (error: unknown) {
    console.error("Failed to delete build:", error);
    await showAlert({
      title: "Failed to Delete Build",
      message: (error as Error).message || "Failed to delete the build. Please try again.",
    });
  } finally {
    isDeleting.value = false;
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

const getBuildStatusVariant = (status: BuildStatus): "success" | "warning" | "danger" | "secondary" => {
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

const getBuildDotColor = (status: BuildStatus) => {
  switch (status) {
    case BuildStatus.BUILD_SUCCESS:
      return "bg-success";
    case BuildStatus.BUILD_FAILED:
      return "bg-danger";
    case BuildStatus.BUILD_BUILDING:
    case BuildStatus.BUILD_PENDING:
      return "bg-warning";
    default:
      return "bg-border-strong";
  }
};

const formatBuildTime = (seconds: number) => {
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return remainingSeconds > 0 ? `${minutes}m ${remainingSeconds}s` : `${minutes}m`;
};

const showBuildDetails = (build: Build) => {
  return !!(build.repositoryUrl || build.buildCommand || build.imageName);
};

onMounted(() => {
  loadBuilds(true);
});

watch(() => props.deploymentId, () => {
  loadBuilds(true);
});
</script>

