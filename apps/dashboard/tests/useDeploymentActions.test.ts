import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeploymentStatus, type Deployment } from "@obiente/proto";
import { useDeploymentActions } from "../app/composables/useDeploymentActions";

const mockClient = {
  startDeployment: vi.fn(),
  stopDeployment: vi.fn(),
  triggerDeployment: vi.fn(),
  restartDeployment: vi.fn(),
  deleteDeployment: vi.fn(),
  updateDeployment: vi.fn(),
  createDeployment: vi.fn(),
};

vi.mock("~/lib/connect-client", () => ({
  useConnectClient: () => mockClient,
}));

vi.mock("~/stores/organizations", () => ({
  useOrganizationsStore: () => ({
    currentOrgId: "org-a",
  }),
}));

describe("useDeploymentActions", () => {
  beforeEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  it("does not fake a running status after start; final state must come from backend refreshes", async () => {
    vi.useFakeTimers();
    const deployment = {
      id: "deployment-a",
      status: DeploymentStatus.STOPPED,
    } as Deployment;

    mockClient.startDeployment.mockResolvedValueOnce({});

    const actions = useDeploymentActions("org-a");
    await actions.startDeployment("deployment-a", deployment);

    expect(deployment.status).toBe(DeploymentStatus.BUILDING);

    await vi.advanceTimersByTimeAsync(3_000);

    expect(deployment.status).toBe(DeploymentStatus.BUILDING);
  });

  it("does not fake a stopped status until stop returns deployment state", async () => {
    const deployment = {
      id: "deployment-a",
      status: DeploymentStatus.RUNNING,
    } as Deployment;

    mockClient.stopDeployment.mockResolvedValueOnce({});

    const actions = useDeploymentActions("org-a");
    await actions.stopDeployment("deployment-a", deployment);

    expect(deployment.status).toBe(DeploymentStatus.RUNNING);
  });

  it("stores command failures for the UI and restores the previous start status", async () => {
    const deployment = {
      id: "deployment-a",
      status: DeploymentStatus.STOPPED,
    } as Deployment;

    mockClient.startDeployment.mockRejectedValueOnce(new Error("Docker host unavailable"));

    const actions = useDeploymentActions("org-a");

    await expect(actions.startDeployment("deployment-a", deployment)).rejects.toThrow("Docker host unavailable");
    expect(deployment.status).toBe(DeploymentStatus.STOPPED);
    expect(actions.operationError.value).toBe("Docker host unavailable");
  });
});
