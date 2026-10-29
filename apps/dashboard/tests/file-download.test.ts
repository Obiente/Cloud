import { beforeEach, expect, it, vi } from "vitest";
import { useDeploymentFileBrowserClient } from "../app/composables/useDeploymentFileBrowserClient";
import { useGameServerFileBrowserClient } from "../app/composables/useGameServerFileBrowserClient";

const { getContainerFile, getGameServerFile } = vi.hoisted(() => ({ getContainerFile: vi.fn(), getGameServerFile: vi.fn() }));
vi.mock("~/lib/connect-client", () => ({ useConnectClient: () => ({ getContainerFile, getGameServerFile }) }));
beforeEach(() => {
  getContainerFile.mockReset().mockResolvedValue({ content: "file", size: 4n });
  getGameServerFile.mockReset().mockResolvedValue({ content: "file", size: 4n });
});

it("distinguishes previews from complete downloads in both file adapters", async () => {
  for (const [client, request] of [
    [useDeploymentFileBrowserClient("dep-example", () => "org-example"), getContainerFile],
    [useGameServerFileBrowserClient("game-example"), getGameServerFile],
  ] as const) {
    await client.getFile({ path: "/large.zip", volumeName: "data" });
    expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ path: "/large.zip", volumeName: "data", download: undefined }));
    await client.getFile({ path: "/large.zip", volumeName: "data", containerId: "worker", download: true });
    expect(request).toHaveBeenLastCalledWith(expect.objectContaining({ path: "/large.zip", volumeName: "data", download: true }));
  }
  expect(getContainerFile).toHaveBeenLastCalledWith(expect.objectContaining({ containerId: "worker" }));
});
