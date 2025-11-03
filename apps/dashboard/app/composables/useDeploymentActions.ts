import { ref } from "vue";
import { type Deployment, DeploymentStatus } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";
import { timestamp } from "@obiente/proto/utils";
import { useOrganizationsStore } from "~/stores/organizations";

export function useDeploymentActions(organizationId: string = "default") {
  const client = useConnectClient(DeploymentService);
  const isProcessing = ref(false);
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
      deployment.status = newStatus;
      return deployment;
    }
    return null;
  };

  /**
   * Start a stopped deployment
   */
  const startDeployment = async (
    deploymentId: string,
    deployments: Deployment | Deployment[] | null | undefined
  ) => {
    if (isProcessing.value) return;
    isProcessing.value = true;

    // Optimistic update
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

      // Simulate transition to RUNNING after a delay
      // (In real app, this would be handled by server via WebSocket/polling)
      setTimeout(() => {
        if (deployment) {
          deployment.status = DeploymentStatus.RUNNING;
        }
      }, 2000);

      return res.deployment;
    } catch (error) {
      console.error("Failed to start deployment:", error);
      // Revert optimistic update
      if (deployment) {
        deployment.status = DeploymentStatus.STOPPED;
      }
      throw error;
    } finally {
      isProcessing.value = false;
    }
  };

  /**
   * Stop a running deployment
   */
  const stopDeployment = async (
    deploymentId: string,
    deployments: Deployment | Deployment[] | null | undefined
  ) => {
    if (isProcessing.value) return;
    isProcessing.value = true;

    // Optimistic update
    const deployment = updateDeploymentStatus(
      deployments,
      deploymentId,
      DeploymentStatus.STOPPED
    );

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
      // Revert optimistic update
      if (deployment) {
        deployment.status = DeploymentStatus.RUNNING;
      }
      throw error;
    } finally {
      isProcessing.value = false;
    }
  };

  /**
   * Trigger a redeployment
   */
  const redeployDeployment = async (
    deploymentId: string,
    deployments: Deployment | Deployment[] | null | undefined
  ) => {
    if (isProcessing.value) return;
    isProcessing.value = true;

    // Optimistic update
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

      // Update with server response after a delay
      setTimeout(() => {
        if (deployment) {
          deployment.status = DeploymentStatus.RUNNING;
        }
      }, 2000);

      return res;
    } catch (error) {
      console.error("Failed to redeploy:", error);
      // Revert optimistic update
      if (deployment) {
        deployment.status = DeploymentStatus.FAILED;
      }
      throw error;
    } finally {
      isProcessing.value = false;
    }
  };

  /**
   * Delete a deployment
   */
  const deleteDeployment = async (
    deploymentId: string,
    deployments?: Deployment | Deployment[]
  ) => {
    if (isProcessing.value) return;
    isProcessing.value = true;

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
      throw error;
    } finally {
      isProcessing.value = false;
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
             githubIntegrationId?: string;
             environment?: number; // Environment enum
             groups?: string[];
           }
         ) => {
           if (isProcessing.value) return;
           isProcessing.value = true;

           try {
             // Build request object, only including fields that are explicitly provided
             const request: any = {
               organizationId: getOrgId(),
               deploymentId,
             };
             
             if (updates.name !== undefined) request.name = updates.name;
             // Include repositoryUrl if provided (convert empty string to undefined for backend)
             if (updates.repositoryUrl !== undefined) {
               request.repositoryUrl = updates.repositoryUrl && updates.repositoryUrl.trim() !== "" 
                 ? updates.repositoryUrl.trim() 
                 : undefined;
             }
             if (updates.branch !== undefined) request.branch = updates.branch;
             if (updates.buildStrategy !== undefined) request.buildStrategy = updates.buildStrategy;
             if (updates.buildCommand !== undefined) request.buildCommand = updates.buildCommand;
             if (updates.installCommand !== undefined) request.installCommand = updates.installCommand;
             if (updates.startCommand !== undefined) request.startCommand = updates.startCommand;
             if (updates.dockerfilePath !== undefined) request.dockerfilePath = updates.dockerfilePath;
             if (updates.composeFilePath !== undefined) request.composeFilePath = updates.composeFilePath;
             // Include githubIntegrationId if provided
             if (updates.githubIntegrationId !== undefined) {
               request.githubIntegrationId = updates.githubIntegrationId && updates.githubIntegrationId.trim() !== ""
                 ? updates.githubIntegrationId.trim()
                 : undefined;
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
             throw error;
           } finally {
             isProcessing.value = false;
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
    if (isProcessing.value) return;
    isProcessing.value = true;

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
      throw error;
    } finally {
      isProcessing.value = false;
    }
  };

  return {
    isProcessing,
    startDeployment,
    stopDeployment,
    redeployDeployment,
    deleteDeployment,
    updateDeployment,
    createDeployment,
  };
}
