<template>
    <OuiStack gap="lg">
      <!-- Header -->
      <OuiFlex justify="between" align="center">
        <OuiText as="h3" size="lg" weight="bold">Build History</OuiText>
        <OuiText v-if="!isLoading" size="sm" color="secondary">
          {{ total }} build{{ total !== 1 ? 's' : '' }}
        </OuiText>
      </OuiFlex>

      <!-- Loading State -->
      <div v-if="isLoading" class="flex justify-center items-center py-12">
        <OuiText size="sm" color="secondary">Loading builds...</OuiText>
      </div>

      <!-- Empty State -->
      <div v-else-if="builds.length === 0" class="flex flex-col items-center justify-center py-12">
        <CubeIcon class="h-12 w-12 text-secondary mb-4" />
        <OuiText size="md" weight="medium" class="mb-2">No builds yet</OuiText>
        <OuiText size="sm" color="secondary" class="text-center max-w-md">
          Build history will appear here once you trigger a deployment. Each build is automatically saved with logs and configuration.
        </OuiText>
      </div>

      <!-- Builds List -->
      <OuiStack v-else gap="md">
        <OuiCard
          v-for="build in builds"
          :key="build.id"
          variant="outline"
          :class="{
            'border-success/20': build.status === BuildStatus.BUILD_SUCCESS,
            'border-danger/20': build.status === BuildStatus.BUILD_FAILED,
            'border-warning/20': build.status === BuildStatus.BUILD_BUILDING || build.status === BuildStatus.BUILD_PENDING,
          }"
        >
          <OuiCardBody>
            <OuiStack gap="md">
              <!-- Build Header -->
              <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
                <OuiFlex align="center" gap="md">
                  <OuiBadge
                    :variant="getBuildStatusVariant(build.status)"
                    size="sm"
                  >
                    <span
                      v-if="build.status === BuildStatus.BUILD_BUILDING || build.status === BuildStatus.BUILD_PENDING"
                      class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5 animate-pulse"
                      :class="getBuildStatusDotClass(build.status)"
                    />
                    <span
                      v-else
                      class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                      :class="getBuildStatusDotClass(build.status)"
                    />
                    {{ getBuildStatusLabel(build.status) }}
                  </OuiBadge>
                  <OuiText size="sm" weight="semibold">
                    Build #{{ build.buildNumber }}
                  </OuiText>
                  <OuiText v-if="build.commitSha" size="xs" color="secondary" class="font-mono">
                    {{ build.commitSha.substring(0, 7) }}
                  </OuiText>
                </OuiFlex>

                <OuiFlex gap="sm" align="center" wrap="wrap">
                  <OuiText size="xs" color="secondary">
                    <OuiRelativeTime
                      :value="build.startedAt ? date(build.startedAt) : undefined"
                      :style="'short'"
                    />
                  </OuiText>
                  <OuiButton
                    v-if="build.status === BuildStatus.BUILD_SUCCESS"
                    variant="ghost"
                    size="xs"
                    @click="() => revertToBuild(build.id)"
                    :disabled="isReverting"
                  >
                    <ArrowPathIcon class="h-3 w-3 mr-1" />
                    Revert
                  </OuiButton>
                </OuiFlex>
              </OuiFlex>

              <!-- Build Details -->
              <OuiGrid :cols="{ sm: 1, md: 2, lg: 3 }" gap="sm">
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

              <!-- Build Configuration -->
              <OuiStack gap="xs" v-if="showBuildDetails(build)">
                <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">
                  Build Configuration
                </OuiText>
                <OuiStack gap="xs">
                  <div v-if="build.repositoryUrl" class="flex items-center gap-2">
                    <CodeBracketIcon class="h-3 w-3 text-secondary" />
                    <OuiText size="xs" class="font-mono truncate">{{ build.repositoryUrl }}</OuiText>
                  </div>
                  <div v-if="build.buildCommand" class="flex items-center gap-2">
                    <CommandLineIcon class="h-3 w-3 text-secondary" />
                    <OuiText size="xs" class="font-mono">{{ build.buildCommand }}</OuiText>
                  </div>
                  <div v-if="build.imageName" class="flex items-center gap-2">
                    <CubeIcon class="h-3 w-3 text-secondary" />
                    <OuiText size="xs" class="font-mono truncate">{{ build.imageName }}</OuiText>
                  </div>
                </OuiStack>
              </OuiStack>

              <!-- Error Message -->
              <OuiCard
                v-if="build.error"
                variant="outline"
                class="border-danger/20 bg-danger/5"
              >
                <OuiCardBody class="p-3">
                  <OuiText size="xs" color="danger" class="font-mono whitespace-pre-wrap">
                    {{ build.error }}
                  </OuiText>
                </OuiCardBody>
              </OuiCard>

              <!-- Actions -->
              <OuiFlex gap="sm" justify="end">
                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="() => viewBuildLogs(build.id)"
                >
                  <DocumentTextIcon class="h-4 w-4 mr-1" />
                  View Logs
                </OuiButton>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  color="danger"
                  @click="() => deleteBuild(build.id)"
                  :disabled="isDeleting"
                >
                  <TrashIcon class="h-4 w-4 mr-1" />
                  Delete
                </OuiButton>
              </OuiFlex>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Pagination -->
        <OuiFlex v-if="total > limit" justify="between" align="center" class="pt-4 border-t border-border-default">
          <OuiText size="xs" color="secondary">
            Showing {{ builds.length }} of {{ total }} builds
          </OuiText>
          <OuiFlex gap="sm">
            <OuiButton
              variant="ghost"
              size="sm"
              @click="loadMore"
              :disabled="isLoadingMore || builds.length >= total"
            >
              Load More
            </OuiButton>
          </OuiFlex>
        </OuiFlex>
      </OuiStack>
    </OuiStack>

</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from "vue";
import {
  CubeIcon,
  ArrowPathIcon,
  DocumentTextIcon,
  CodeBracketIcon,
  CommandLineIcon,
  TrashIcon,
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
  } catch (error: any) {
    console.error("Failed to load builds:", error);
    await showAlert({
      title: "Failed to Load Builds",
      message: error.message || "Failed to load build history. Please try again.",
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
  } catch (error: any) {
    console.error("Failed to revert build:", error);
    await showAlert({
      title: "Failed to Revert",
      message: error.message || "Failed to revert to the selected build. Please try again.",
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
  } catch (error: any) {
    console.error("Failed to delete build:", error);
    await showAlert({
      title: "Failed to Delete Build",
      message: error.message || "Failed to delete the build. Please try again.",
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

