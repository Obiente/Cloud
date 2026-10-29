import { beforeEach, expect, it, vi } from "vitest";
import { useFileExplorer } from "../app/composables/useFileExplorer";

const { listContainerFiles, writeContainerFile } = vi.hoisted(() => ({ listContainerFiles: vi.fn(), writeContainerFile: vi.fn() }));
vi.mock("~/lib/connect-client", () => ({ useConnectClient: () => ({ listContainerFiles, writeContainerFile }) }));
const explorer = () => useFileExplorer({ type: "deployment", organizationId: "org-example", deploymentId: "dep-example" });
const file = (name: string) => ({ name, path: `/${name}`, size: 4n });
beforeEach(() => { listContainerFiles.mockReset(); writeContainerFile.mockReset(); });

it("saves to the selected container and preserves a captured file target", async () => {
  const e = explorer();
  e.setContainer("container-b", "worker");
  e.switchToVolume("data");
  await e.writeFile({ path: "/record.txt", volumeName: "data", content: "edited" });
  expect(writeContainerFile).toHaveBeenLastCalledWith(expect.objectContaining({ containerId: "container-b", serviceName: "worker", volumeName: "data" }));
  e.setContainer("container-a", "api");
  await e.writeFile({ path: "/record.txt", volumeName: "data", content: "edited", containerId: "container-b", serviceName: "worker" });
  expect(writeContainerFile).toHaveBeenLastCalledWith(expect.objectContaining({ containerId: "container-b", serviceName: "worker", volumeName: "data" }));
});

it("uses the selected container for volume discovery and files", async () => {
  const e = explorer();
  e.setContainer("container-a", "api");
  listContainerFiles.mockResolvedValueOnce({ volumes: [{ name: "data", mountPoint: "/data" }], containerRunning: true });
  await e.fetchVolumes();
  expect(listContainerFiles).toHaveBeenLastCalledWith(expect.objectContaining({ listVolumes: true, containerId: "container-a", serviceName: "api" }));
  e.switchToVolume("data");
  listContainerFiles.mockResolvedValueOnce({ files: [file("record")], containerRunning: true });
  await e.refreshRoot();
  expect(listContainerFiles).toHaveBeenLastCalledWith(expect.objectContaining({ volumeName: "data", containerId: "container-a", serviceName: "api" }));
  expect(e.root.children?.map(f => f.name)).toEqual(["record"]);
  e.setContainer("container-b", "worker");
  expect(e.volumes.value).toEqual([]);
  expect(e.root.children).toEqual([]);
  expect(e.source.type).toBe("container");
});

it("keeps volume errors visible after file refresh and recovers on retry", async () => {
  const e = explorer();
  listContainerFiles.mockRejectedValueOnce(new Error("offline")).mockResolvedValueOnce({ files: [file("app")], containerRunning: true });
  await e.fetchVolumes();
  await e.refreshRoot();
  expect(e.volumeError.value).toBe("Unable to load storage volumes. Try refreshing.");
  expect(e.containerRunning.value).toBe(true);
  listContainerFiles.mockResolvedValueOnce({ volumes: [{ name: "data" }], containerRunning: true });
  await e.fetchVolumes();
  expect(e.volumeError.value).toBeNull();
  expect(e.volumes.value).toHaveLength(1);
});

it("discards stale volume and tree responses after switching containers", async () => {
  const e = explorer();
  e.setContainer("old");
  let finishVolumes!: (value: unknown) => void;
  let finishFiles!: (value: unknown) => void;
  listContainerFiles.mockImplementationOnce(() => new Promise(resolve => { finishVolumes = resolve; }))
    .mockImplementationOnce(() => new Promise(resolve => { finishFiles = resolve; }));
  const volumes = e.fetchVolumes();
  const files = e.refreshRoot();
  e.setContainer("new");
  listContainerFiles.mockResolvedValueOnce({ files: [file("current")], containerRunning: true });
  await e.refreshRoot();
  finishVolumes({ volumes: [{ name: "old-data" }], containerRunning: false });
  finishFiles({ files: [file("stale")], containerRunning: false });
  await Promise.all([volumes, files]);
  expect(e.root.children?.map(f => f.name)).toEqual(["current"]);
  expect(e.volumes.value).toEqual([]);
  expect(e.containerRunning.value).toBe(true);
});

it("clears stale files on failure without treating errors as entries", async () => {
  const e = explorer();
  listContainerFiles.mockResolvedValueOnce({ files: [file("old")] })
    .mockRejectedValueOnce(new Error("Unable to read directory"));
  await e.refreshRoot();
  await e.refreshRoot();
  expect(e.root.children).toEqual([]);
  expect(e.root.hasLoaded).toBe(false);
  expect(e.errorMessage.value).toBe("Unable to read directory");
});

it("preserves the selected volume during refresh and ignores a previous source", async () => {
  const e = explorer();
  let finish!: (value: unknown) => void;
  listContainerFiles.mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
  const old = e.refreshRoot();
  e.switchToVolume("data");
  listContainerFiles.mockResolvedValueOnce({ volumes: [{ name: "data" }], containerRunning: true })
    .mockResolvedValueOnce({ files: [file("volume-record")], containerRunning: true });
  await e.fetchVolumes();
  await e.refreshRoot();
  finish({ files: [file("container-record")] });
  await old;
  expect(e.source.volumeName).toBe("data");
  expect(e.root.children?.map(f => f.name)).toEqual(["volume-record"]);
});
