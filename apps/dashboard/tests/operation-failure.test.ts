import { beforeEach, afterEach, describe, it, expect, vi } from "vitest";
import { ref } from "vue";
const mocks = vi.hoisted(() => ({
  upload: vi.fn(),
  auth: null as any,
  cleanup: null as any,
}));
vi.mock("~/lib/connect-client", () => ({
  useConnectClient: () => ({ uploadContainerFiles: mocks.upload }),
}));
vi.mock("~/composables/useAuth", () => ({ useAuth: () => mocks.auth }));
vi.mock("~/stores/organizations", () => ({
  useOrganizationsStore: () => ({ currentOrgId: "org-example" }),
}));
vi.mock("~/composables/useConfig", () => ({
  useConfig: () => ({ disableAuth: { value: false } }),
}));
vi.mock("vue", async (original) => ({
  ...(await original<typeof import("vue")>()),
  onUnmounted: (fn: any) => {
    mocks.cleanup = fn;
  },
}));
class Socket {
  static OPEN = 1;
  static instances: Socket[] = [];
  readyState = 0;
  onopen: any;
  onclose: any;
  onmessage: any;
  onerror: any;
  sent: string[] = [];
  constructor(_url: string) {
    Socket.instances.push(this);
  }
  send(data: string) {
    if (this.readyState !== 1) throw new Error("closed");
    this.sent.push(data);
  }
  close() {
    this.readyState = 3;
    this.onclose?.();
  }
  async open() {
    this.readyState = 1;
    await this.onopen?.();
  }
  connected() {
    this.onmessage?.({ data: JSON.stringify({ type: "connected" }) });
  }
}
beforeEach(() => {
  vi.clearAllMocks();
  vi.useFakeTimers();
  Socket.instances = [];
  mocks.auth = {
    ready: true,
    getAccessToken: vi.fn().mockResolvedValue("synthetic-token"),
  };
  vi.stubGlobal("ref", ref);
  vi.stubGlobal("WebSocket", Socket);
  vi.stubGlobal("useRuntimeConfig", () => ({
    public: { apiHost: "https://api.example.test" },
  }));
});
afterEach(() => {
  vi.clearAllTimers();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});
describe("operation failure recovery", () => {
  it("reports an unreadable file and releases upload state", async () => {
    const { useDeploymentUpload } = await import(
      "../app/composables/useDeploymentUpload"
    );
    const upload = useDeploymentUpload();
    const file = {
      name: "example.txt",
      size: 10,
      arrayBuffer: () =>
        Promise.reject(new Error("File is no longer available")),
    } as File;
    expect(
      await upload.uploadFile(file, { deploymentId: "example-deployment" })
    ).toBe(false);
    expect(upload.isUploading.value).toBe(false);
    expect(upload.error.value).toContain("no longer available");
    expect(mocks.upload).not.toHaveBeenCalled();
  });
  it("rejects a socket closed before the handshake and permits retry", async () => {
    const { useGameServerCommand } = await import(
      "../app/composables/useGameServerCommand"
    );
    const command = useGameServerCommand({ gameServerId: "example-server" });
    const first = command.connect().then(
      () => "connected",
      () => "failed"
    );
    Socket.instances[0]!.close();
    await vi.advanceTimersByTimeAsync(1);
    expect(await Promise.race([first, Promise.resolve("pending")])).toBe(
      "failed"
    );
    const second = command.connect();
    await Socket.instances[1]!.open();
    Socket.instances[1]!.connected();
    await second;
    await command.sendCommand("status");
    expect(Socket.instances[1]!.sent.at(-1)).toContain('"type":"input"');
    command.disconnect();
  });
  it("bounds authentication readiness and cleans up the failed handshake", async () => {
    const { useGameServerCommand } = await import(
      "../app/composables/useGameServerCommand"
    );
    mocks.auth.ready = false;
    const command = useGameServerCommand({ gameServerId: "example-server" });
    const pending = command.connect().then(
      () => "connected",
      () => "failed"
    );
    void Socket.instances[0]!.open();
    await vi.advanceTimersByTimeAsync(11000);
    expect(await Promise.race([pending, Promise.resolve("pending")])).toBe(
      "failed"
    );
    expect(command.isLoading.value).toBe(false);
    expect(vi.getTimerCount()).toBe(0);
  });
  it("does not initialize a disconnected socket when token retrieval finishes late", async () => {
    const { useGameServerCommand } = await import(
      "../app/composables/useGameServerCommand"
    );
    let complete!: (token: string) => void;
    mocks.auth.getAccessToken.mockReturnValue(
      new Promise((r) => {
        complete = r;
      })
    );
    const command = useGameServerCommand({ gameServerId: "example-server" });
    const pending = command.connect().catch(() => {});
    const opening = Socket.instances[0]!.open();
    command.disconnect();
    complete("old-token");
    await opening;
    await pending;
    expect(Socket.instances[0]!.sent).toEqual([]);
    expect(command.isConnected.value).toBe(false);
  });
});
