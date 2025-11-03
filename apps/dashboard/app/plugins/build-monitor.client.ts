import { DeploymentService } from "@obiente/proto";
import { useNotifications } from "~/composables/useNotifications";
import { useToast } from "~/composables/useToast";
import { useOrganizationsStore } from "~/stores/organizations";
import { useConnectClient } from "~/lib/connect-client";

// BuildStatus enum - defined in the proto but not re-exported from main index
enum BuildStatus {
  BUILD_STATUS_UNSPECIFIED = 0,
  BUILD_PENDING = 1,
  BUILD_BUILDING = 2,
  BUILD_SUCCESS = 3,
  BUILD_FAILED = 4,
}

// DeploymentStatus enum
enum DeploymentStatus {
  DEPLOYMENT_STATUS_UNSPECIFIED = 0,
  CREATED = 1,
  BUILDING = 2,
  RUNNING = 3,
  STOPPED = 4,
  FAILED = 5,
  DEPLOYING = 6,
}

export default defineNuxtPlugin({
  name: "build-monitor",
  dependsOn: ["connect-transport-client"],
  setup() {
    if (import.meta.server) return;

    const nuxtApp = useNuxtApp();
    const { addNotification } = useNotifications();
    const { toast } = useToast();
    const orgsStore = useOrganizationsStore();

    // Track build statuses to detect changes
    const buildStatusCache = new Map<string, BuildStatus>();
    // Track deployment statuses to detect deployment failures
    const deploymentStatusCache = new Map<string, DeploymentStatus>();

    let pollingInterval: ReturnType<typeof setInterval> | null = null;

  // Get client lazily to ensure transport is initialized
  const getClient = () => {
    const baseClient = useConnectClient(DeploymentService);
    // listBuilds is defined in DeploymentService (proto line 4447) and works at runtime
    // TypeScript doesn't infer it, so we use a type-safe intersection assertion
    type ListBuildsRequest = { organizationId: string; deploymentId: string; limit?: number; offset?: number };
    type ListBuildsResponse = { builds?: Array<{ id: string; buildNumber: number; status: BuildStatus }> };
    return baseClient as typeof baseClient & {
      listBuilds: (req: ListBuildsRequest) => Promise<ListBuildsResponse>;
    };
  };

  const checkBuildStatuses = async () => {
    try {
      // Ensure transport is available before creating client
      if (!nuxtApp.$connect) {
        console.debug("Connect transport not ready, skipping build status check");
        return;
      }

      const organizationId = orgsStore.currentOrgId;
      if (!organizationId) {
        return;
      }

      const client = getClient();

      // Get all deployments
      const deploymentsResponse = await client.listDeployments({
        organizationId,
      });

      if (!deploymentsResponse.deployments || deploymentsResponse.deployments.length === 0) {
        return;
      }

      // For each deployment, check the latest build
      for (const deployment of deploymentsResponse.deployments) {
        try {
          // Get latest build - listBuilds is defined in the DeploymentService
          const buildsResponse = await client.listBuilds({
            organizationId,
            deploymentId: deployment.id,
            limit: 1,
            offset: 0,
          });

          if (!buildsResponse.builds || buildsResponse.builds.length === 0) {
            continue;
          }

          const latestBuild = buildsResponse.builds[0];
          if (!latestBuild) {
            continue;
          }

          const buildKey = `${deployment.id}-${latestBuild.id}`;
          const previousStatus = buildStatusCache.get(buildKey);
          const currentStatus = latestBuild.status;

          // If this is the first time we see this build, cache it but don't notify
          if (previousStatus === undefined) {
            buildStatusCache.set(buildKey, currentStatus);
            continue;
          }

          const deploymentName = deployment.name || deployment.domain || deployment.id;
          
          // Notify if build status changed from building/pending to success/failed
          if (
            (previousStatus === BuildStatus.BUILD_BUILDING ||
              previousStatus === BuildStatus.BUILD_PENDING) &&
            (currentStatus === BuildStatus.BUILD_SUCCESS ||
              currentStatus === BuildStatus.BUILD_FAILED)
          ) {
            // Build completed!
            if (currentStatus === BuildStatus.BUILD_SUCCESS) {
              toast.success(
                "Build Completed Successfully",
                `Build #${latestBuild.buildNumber} for ${deploymentName} completed successfully.`
              );
              addNotification({
                title: "Build Completed Successfully",
                message: `Build #${latestBuild.buildNumber} for ${deploymentName} completed successfully.`,
              });
            } else if (currentStatus === BuildStatus.BUILD_FAILED) {
              toast.error(
                "Build Failed",
                `Build #${latestBuild.buildNumber} for ${deploymentName} failed.`
              );
              addNotification({
                title: "Build Failed",
                message: `Build #${latestBuild.buildNumber} for ${deploymentName} failed.`,
              });
            }
          }
          
          // Also check for deployment failures after successful build
          // This happens when build succeeds but deployment fails
          if (
            previousStatus === BuildStatus.BUILD_SUCCESS &&
            currentStatus === BuildStatus.BUILD_FAILED
          ) {
            toast.error(
              "Deployment Failed",
              `Deployment failed for ${deploymentName} after successful build.`
            );
            addNotification({
              title: "Deployment Failed",
              message: `Build #${latestBuild.buildNumber} succeeded, but deployment failed for ${deploymentName}.`,
            });
          }
          
          // Check deployment status changes separately
          const deploymentKey = deployment.id;
          const previousDeploymentStatus = deploymentStatusCache.get(deploymentKey);
          const currentDeploymentStatus = deployment.status as DeploymentStatus;
          
          // Notify if deployment failed (especially after successful build)
          if (
            previousDeploymentStatus !== undefined &&
            previousDeploymentStatus !== DeploymentStatus.FAILED &&
            currentDeploymentStatus === DeploymentStatus.FAILED
          ) {
            toast.error(
              "Deployment Failed",
              `Deployment failed for ${deploymentName}.`
            );
            addNotification({
              title: "Deployment Failed",
              message: `Deployment failed for ${deploymentName}.`,
            });
          }
          
          // Update deployment status cache
          deploymentStatusCache.set(deploymentKey, currentDeploymentStatus);

          // Update cache
          buildStatusCache.set(buildKey, currentStatus);

          // Clean up old entries (keep only last 100)
          if (buildStatusCache.size > 100) {
            const entries = Array.from(buildStatusCache.entries());
            entries.slice(0, buildStatusCache.size - 100).forEach(([key]) => {
              buildStatusCache.delete(key);
            });
          }
        } catch (error) {
          console.error(`Failed to check builds for deployment ${deployment.id}:`, error);
        }
      }
    } catch (error) {
      console.error("Failed to check build statuses:", error);
    }
  };

  // Start polling when organization is set
  const startPolling = () => {
    if (pollingInterval) return;

    // Poll every 5 seconds
    pollingInterval = setInterval(checkBuildStatuses, 5000);
    
    // Also check immediately
    checkBuildStatuses();
  };

  const stopPolling = () => {
    if (pollingInterval) {
      clearInterval(pollingInterval);
      pollingInterval = null;
    }
  };

  // Watch for organization changes
  watch(
    () => orgsStore.currentOrgId,
    (newOrgId) => {
      if (newOrgId) {
        // Clear cache when switching organizations
        buildStatusCache.clear();
        deploymentStatusCache.clear();
        startPolling();
      } else {
        stopPolling();
      }
    },
    { immediate: true }
  );

    // Cleanup on unmount
    if (import.meta.client) {
      onUnmounted(() => {
        stopPolling();
        buildStatusCache.clear();
        deploymentStatusCache.clear();
      });
    }
  },
});
