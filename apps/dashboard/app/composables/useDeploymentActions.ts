import { ref } from "vue";
import { type Deployment, type DockerfileBuildOptions, type DockerfileVolume, DeploymentStatus } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { timestamp } from "@obiente/proto/utils";
import { useOrganizationsStore } from "~/stores/organizations";

export function useDeploymentActions(organizationId: string = "default") {
  const client = useConnectClient(DeploymentService);
  const isProcessing = ref(false);
  const currentOperation = ref<"start" | "stop" | "redeploy" | "restart" | "delete" | "update" | "create" | null>(null);
  const operationError = ref<string | null>(null);
  const orgsStore = useOrganizationsStore();

  const getOrgId = () => {
    // Prefer explicit org id when provided and not the legacy "default"
    if (organizationId && organizationId !== "default") return organizationId;
    // Fall back to global selected org id from store
    if (orgsStore?.currentOrgId) return orgsStore.currentOrgId;
    // Let API resolve if still empty
    return "";
  };

  /**
   * Optimistically update deployment status
   */
  const updateDeploymentStatus = (
    deployments: Deployment | Deployment[] | null | undefined,
    deploymentId: string,
    newStatus: DeploymentStatus
  ) => {
    if (!deployments) return null;

    // Handle both single deployment and array of deployments
    const deployment = Array.isArray(deployments)
      ? deployments.find((d) => d.id === deploymentId)
      : (deployments as Deployment).id === deploymentId
      ? (deployments as Deployment)
      : null;

    if (deployment) {
      // Use Object.assign to ensure reactivity is maintained
      Object.assign(deployment, { status: newStatus });
      return deployment;
    }
    return null;
  };

  const getErrorMessage = (error: unknown, fallback: string) => {
    if (error instanceof Error && error.message) return error.message;
    return fallback;
  };

  const beginOperation = (operation: typeof currentOperation.value) => {
    if (isProcessing.value) return false;
    isProcessing.value = true;
    currentOperation.value = operation;
    operationError.value = null;
    return true;
  };

  const finishOperation = () => {
    isProcessing.value = false;
    currentOperation.value = null;
  };

  /**
   * Start a stopped deployment. The UI moves into a transitional state, but the
   * final status must come from the backend response or follow-up refreshes.
   */
  const startDeployment = async (
    deploymentId: string,
    deployments: Deployment | Deployment[] | null | undefined
  ) => {
    if (!beginOperation("start")) return;

    const previousDeployment = Array.isArray(deployments)
      ? deployments.find((d) => d.id === deploymentId)
      : (deployments as Deployment)?.id === deploymentId
      ? (deployments as Deployment)
      : null;
    const previousStatus = previousDeployment?.status;
    const deployment = updateDeploymentStatus(
      deployments,
      deploymentId,
      DeploymentStatus.BUILDING
    );

    try {
      const res = await client.startDeployment({
        organizationId: getOrgId(),
        deploymentId,
      });

      if (deployment && res.deployment) {
        Object.assign(deployment, res.deployment);
      }

      return res.deployment;
    } catch (error) {
      console.error("Failed to start deployment:", error);
      if (deployment) {
        deployment.status = previousStatus ?? DeploymentStatus.STOPPED;
      }
      operationError.value = getErrorMessage(error, "Failed to start deployment.");
      throw error;
    } finally {
      finishOperation();
    }
  };

  /**
   * Stop a running deployment
   */
  const stopDeployment = async (
    deploymentId: string,
    deployments: Deployment | Deployment[] | null | undefined
  ) => {
    if (!beginOperation("stop")) return;

    const deployment = Array.isArray(deployments)
      ? deployments.find((d) => d.id === deploymentId)
      : (deployments as Deployment)?.id === deploymentId
      ? (deployments as Deployment)
      : null;

    try {
      const res = await client.stopDeployment({
        organizationId: getOrgId(),
        deploymentId,
      });

      // Update with server response
      if (deployment && res.deployment) {
        Object.assign(deployment, res.deployment);
      }

      return res.deployment;
    } catch (error) {
      console.error("Failed to stop deployment:", error);
      operationError.value = getErrorMessage(error, "Failed to stop deployment.");
      throw error;
    } finally {
      finishOperation();
    }
  };

  /**
   * Trigger a redeployment (rebuilds and redeploys)
   */
  const redeployDeployment = async (
    deploymentId: string,
    deployments: Deployment | Deployment[] | null | undefined
  ) => {
    if (!beginOperation("redeploy")) return;

    const previousDeployment = Array.isArray(deployments)
      ? deployments.find((d) => d.id === deploymentId)
      : (deployments as Deployment)?.id === deploymentId
      ? (deployments as Deployment)
      : null;
    const previousStatus = previousDeployment?.status;
    const deployment = updateDeploymentStatus(
      deployments,
      deploymentId,
      DeploymentStatus.DEPLOYING
    );

    if (deployment) {
      deployment.lastDeployedAt = timestamp(new Date());
    }

    try {
      const res = await client.triggerDeployment({
        organizationId: getOrgId(),
        deploymentId,
      });

      if (deployment && "deployment" in res && res.deployment) {
        Object.assign(deployment, res.deployment);
      }

      return res;
    } catch (error) {
      console.error("Failed to redeploy:", error);
      if (deployment) {
        deployment.status = previousStatus ?? DeploymentStatus.FAILED;
      }
      operationError.value = getErrorMessage(error, "Failed to redeploy.");
      throw error;
    } finally {
      finishOperation();
    }
  };

  /**
   * Reload a deployment (restarts containers without rebuilding)
   * This is useful when configs have been updated and you want to apply them
   */
  const reloadDeployment = async (
    deploymentId: string,
    deployments: Deployment | Deployment[] | null | undefined
  ) => {
    if (!beginOperation("restart")) return;

    // Optimistic update - keep current status but show it's reloading
    const deployment = Array.isArray(deployments)
      ? deployments.find((d) => d.id === deploymentId)
      : (deployments as Deployment)?.id === deploymentId
      ? (deployments as Deployment)
      : null;

    if (deployment) {
      deployment.lastDeployedAt = timestamp(new Date());
    }

    try {
      const res = await client.restartDeployment({
        organizationId: getOrgId(),
        deploymentId,
      });

      // Update with server response
      if (deployment && res.deployment) {
        Object.assign(deployment, res.deployment);
      }

      return res.deployment;
    } catch (error) {
      console.error("Failed to reload deployment:", error);
      operationError.value = getErrorMessage(error, "Failed to restart deployment.");
      throw error;
    } finally {
      finishOperation();
    }
  };

  /**
   * Delete a deployment
   */
  const deleteDeployment = async (
    deploymentId: string,
    deployments?: Deployment | Deployment[]
  ) => {
    if (!beginOperation("delete")) return;

    try {
      const res = await client.deleteDeployment({
        organizationId: getOrgId(),
        deploymentId,
      });

      // Remove from local state if it's an array
      if (deployments && Array.isArray(deployments)) {
        const index = deployments.findIndex((d) => d.id === deploymentId);
        if (index !== -1) {
          deployments.splice(index, 1);
        }
      }

      return res;
    } catch (error) {
      console.error("Failed to delete deployment:", error);
      operationError.value = getErrorMessage(error, "Failed to delete deployment.");
      throw error;
    } finally {
      finishOperation();
    }
  };

  /**
   * Update deployment configuration
   */
         const updateDeployment = async (
           deploymentId: string,
           updates: {
             name?: string;
             repositoryUrl?: string;
             branch?: string;
             buildStrategy?: number; // BuildStrategy enum
             buildCommand?: string;
             installCommand?: string;
             startCommand?: string;
             dockerfilePath?: string;
             composeFilePath?: string;
             buildPath?: string;
             buildOutputPath?: string;
             nginxConfig?: string;
             cpuLimit?: number | string | null;
             memoryLimit?: number | string | bigint | null;
             githubIntegrationId?: string;
             environment?: number; // Environment enum
             groups?: string[];
             healthcheckType?: number;
             healthcheckPort?: number;
             healthcheckPath?: string;
	             healthcheckExpectedStatus?: number;
	             healthcheckCustomCommand?: string;
	             autoDeploy?: boolean;
	             buildArgs?: Record<string, string>;
	             dockerfileVolumes?: DockerfileVolume[];
	             dockerfileBuildOptions?: DockerfileBuildOptions;
	           }
         ) => {
           if (!beginOperation("update")) return;

           try {
             // Build request object, only including fields that are explicitly provided
             const request: any = {
               organizationId: getOrgId(),
               deploymentId,
             };
             
             if (updates.name !== undefined) request.name = updates.name;
             // Include repositoryUrl if provided - send null for empty strings to clear it
             if (updates.repositoryUrl !== undefined) {
               request.repositoryUrl = updates.repositoryUrl === null || updates.repositoryUrl === "" 
                 ? null 
                 : updates.repositoryUrl.trim();
             }
             if (updates.branch !== undefined) request.branch = updates.branch;
            if (updates.buildStrategy !== undefined) request.buildStrategy = updates.buildStrategy;
            // Always include these fields - send empty string for empty/null values so protobuf includes them
            // Backend will clear fields when it receives empty strings
            if (updates.buildCommand !== undefined) {
              request.buildCommand = updates.buildCommand === null || updates.buildCommand === "" ? "" : updates.buildCommand;
            }
            if (updates.installCommand !== undefined) {
              request.installCommand = updates.installCommand === null || updates.installCommand === "" ? "" : updates.installCommand;
            }
            if (updates.startCommand !== undefined) {
              request.startCommand = updates.startCommand === null || updates.startCommand === "" ? "" : updates.startCommand;
            }
            if (updates.dockerfilePath !== undefined) {
              request.dockerfilePath = updates.dockerfilePath === null || updates.dockerfilePath === "" ? "" : updates.dockerfilePath;
            }
            if (updates.composeFilePath !== undefined) {
              request.composeFilePath = updates.composeFilePath === null || updates.composeFilePath === "" ? "" : updates.composeFilePath;
            }
            if (updates.buildPath !== undefined) {
              request.buildPath = updates.buildPath === null || updates.buildPath === "" ? "" : updates.buildPath;
            }
            if (updates.buildOutputPath !== undefined) {
              request.buildOutputPath = updates.buildOutputPath === null || updates.buildOutputPath === "" ? "" : updates.buildOutputPath;
            }
            if (updates.nginxConfig !== undefined) {
              request.nginxConfig = updates.nginxConfig === null || updates.nginxConfig === "" ? "" : updates.nginxConfig;
            }

            // Healthcheck configuration
            if (updates.healthcheckType !== undefined) {
              request.healthcheckType = updates.healthcheckType;
            }
            if (updates.healthcheckPort !== undefined) {
              request.healthcheckPort = updates.healthcheckPort;
            }
            if (updates.healthcheckPath !== undefined) {
              request.healthcheckPath = updates.healthcheckPath === null || updates.healthcheckPath === "" ? "" : updates.healthcheckPath;
            }
            if (updates.healthcheckExpectedStatus !== undefined) {
              request.healthcheckExpectedStatus = updates.healthcheckExpectedStatus;
            }
            if (updates.healthcheckCustomCommand !== undefined) {
              request.healthcheckCustomCommand = updates.healthcheckCustomCommand === null || updates.healthcheckCustomCommand === "" ? "" : updates.healthcheckCustomCommand;
            }
	            if (updates.autoDeploy !== undefined) {
	              request.autoDeploy = updates.autoDeploy;
	            }
	            if (updates.buildArgs !== undefined) {
	              request.buildArgs = updates.buildArgs;
	            }
	            if (updates.dockerfileVolumes !== undefined) {
	              request.dockerfileVolumes = updates.dockerfileVolumes;
	            }
	            if (updates.dockerfileBuildOptions !== undefined) {
	              request.dockerfileBuildOptions = updates.dockerfileBuildOptions;
	            }

	            // Per-deployment resource limits
            // Semantics: 0 clears override (backend falls back to defaults capped by plan)
            if (updates.cpuLimit !== undefined) {
              const raw = updates.cpuLimit;
              const parsed =
                raw === null || raw === "" ? 0 : Number(String(raw).trim());
              request.cpuLimit = Number.isFinite(parsed) ? parsed : 0;
            }
            if (updates.memoryLimit !== undefined) {
              const raw = updates.memoryLimit;
              const parsed =
                raw === null || raw === "" ? 0 : Number(String(raw).trim());
              const mb = Number.isFinite(parsed) ? Math.trunc(parsed) : 0;
              request.memoryLimit = BigInt(Math.max(0, mb));
            }
             // Include githubIntegrationId if provided - send null for empty strings to clear it
             if (updates.githubIntegrationId !== undefined) {
               request.githubIntegrationId = updates.githubIntegrationId === null || updates.githubIntegrationId === "" 
                 ? null 
                 : updates.githubIntegrationId.trim();
             }
             if (updates.environment !== undefined) request.environment = updates.environment;
             // Always include groups if provided (even if empty array) so backend can clear it
             if (updates.groups !== undefined) {
               request.groups = updates.groups;
             }
             
             const res = await client.updateDeployment(request);

             return res.deployment;
           } catch (error) {
             console.error("Failed to update deployment:", error);
             operationError.value = getErrorMessage(error, "Failed to update deployment.");
             throw error;
           } finally {
             finishOperation();
           }
         };

  /**
   * Create a new deployment
   */
         const createDeployment = async (deployment: {
           name: string;
           environment?: number;
           groups?: string[];
         }) => {
    if (!beginOperation("create")) return;

    try {
             const res = await client.createDeployment({
               organizationId: getOrgId(),
               name: deployment.name,
               environment: deployment.environment,
               groups: deployment.groups || [],
             });

      return res.deployment;
    } catch (error) {
      console.error("Failed to create deployment:", error);
      operationError.value = getErrorMessage(error, "Failed to create deployment.");
      throw error;
    } finally {
      finishOperation();
    }
  };

  return {
    isProcessing,
    currentOperation,
    operationError,
    startDeployment,
    stopDeployment,
    redeployDeployment,
    reloadDeployment,
    deleteDeployment,
    updateDeployment,
    createDeployment,
  };
}
