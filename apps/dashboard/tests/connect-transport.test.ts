import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { effectScope, reactive, type EffectScope } from "vue";
const mocks = vi.hoisted(() => ({ auth: null as any, getToken: null as any }));
vi.mock("~/composables/useAuth", () => ({ useAuth: () => mocks.auth }));
vi.mock("~/lib/transport", () => ({
  createAuthInterceptor: (get: any) => {
    mocks.getToken = get;
    return {};
  },
  createWebTransport: () => ({}),
}));
const deferred = () => {
  let resolve!: (token: string | null) => void;
  const promise = new Promise<string | null>((r) => {
    resolve = r;
  });
  return { promise, resolve };
};
let scope: EffectScope;
beforeEach(async () => {
  vi.resetModules();
  scope = effectScope();
  mocks.auth = reactive({
    session: { id: "a" },
    ready: true,
    getAccessToken: vi.fn(),
  });
  vi.stubGlobal("defineNuxtPlugin", (plugin: any) => plugin);
  vi.stubGlobal("useRuntimeConfig", () => ({
    public: { apiHost: "https://api.example.test" },
  }));
  const plugin = (await import("../app/plugins/connect-transport.client"))
    .default;
  scope.run(() => (plugin as any).setup({}));
});
afterEach(() => {
  scope.stop();
  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});
describe("transport token cache", () => {
  it("keeps an in-flight token read when the same account's session is refreshed", async () => {
    const pendingToken = deferred();
    mocks.auth.getAccessToken.mockReturnValue(pendingToken.promise);
    const pending = mocks.getToken();
    mocks.auth.session = { id: "a" };
    pendingToken.resolve("current-token");
    expect(await pending).toBe("current-token");
  });

  it("does not send or cache a token from the previous account", async () => {
    const old = deferred();
    mocks.auth.getAccessToken
      .mockReturnValueOnce(old.promise)
      .mockResolvedValue("new-token");
    const pending = mocks.getToken();
    mocks.auth.session = { id: "b" };
    old.resolve("old-token");
    expect(await pending).toBeUndefined();
    expect(await mocks.getToken()).toBe("new-token");
    expect(await mocks.getToken()).toBe("new-token");
    expect(mocks.auth.getAccessToken).toHaveBeenCalledTimes(3);
  });
  it("does not let old completion clear the next identity's pending request", async () => {
    const old = deferred(),
      current = deferred();
    mocks.auth.getAccessToken
      .mockReturnValueOnce(old.promise)
      .mockReturnValueOnce(current.promise);
    const a = mocks.getToken();
    mocks.auth.session = { id: "b" };
    const b = mocks.getToken();
    old.resolve("old");
    await a;
    const c = mocks.getToken();
    current.resolve("current");
    expect(await b).toBe("current");
    expect(await c).toBe("current");
    expect(mocks.auth.getAccessToken).toHaveBeenCalledTimes(2);
  });
  it("uses a renewed token immediately even when the session identity is unchanged", async () => {
    mocks.auth.getAccessToken
      .mockResolvedValueOnce("near-expiry")
      .mockResolvedValueOnce("renewed");
    expect(await mocks.getToken()).toBe("near-expiry");
    expect(await mocks.getToken()).toBe("renewed");
  });
  it("stops readiness polling at its deadline", async () => {
    vi.useFakeTimers();
    vi.spyOn(console, "warn").mockImplementation(() => {});
    mocks.auth.ready = false;
    mocks.auth.getAccessToken.mockResolvedValue(null);
    const pending = mocks.getToken();
    await vi.advanceTimersByTimeAsync(550);
    await pending;
    expect(vi.getTimerCount()).toBe(0);
  });
});
