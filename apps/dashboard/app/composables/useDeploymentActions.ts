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
      branch?: string;
      buildCommand?: string;
      installCommand?: string;
    }
  ) => {
    if (isProcessing.value) return;
    isProcessing.value = true;

    try {
      const res = await client.updateDeployment({
        organizationId: getOrgId(),
        deploymentId,
        branch: updates.branch,
        buildCommand: updates.buildCommand,
        installCommand: updates.installCommand,
      });

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
    type: any; // DeploymentType
    repositoryUrl?: string;
    branch: string;
    buildCommand?: string;
    installCommand?: string;
  }) => {
    if (isProcessing.value) return;
    isProcessing.value = true;

    try {
      const res = await client.createDeployment({
        organizationId: getOrgId(),
        name: deployment.name,
        type: deployment.type,
        repositoryUrl: deployment.repositoryUrl,
        branch: deployment.branch,
        buildCommand: deployment.buildCommand,
        installCommand: deployment.installCommand,
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
