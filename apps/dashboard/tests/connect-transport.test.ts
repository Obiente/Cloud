import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { effectScope, reactive, ref, nextTick, type EffectScope } from "vue";
import { Code, ConnectError } from "@connectrpc/connect";
const mocks = vi.hoisted(() => ({ auth: null as any, getToken: null as any, organizations: null as any, restriction: null as any, interceptor: null as any }));
vi.mock("~/composables/useAuth", () => ({ useAuth: () => mocks.auth }));
vi.mock("~/stores/organizations", () => ({ useOrganizationsStore: () => mocks.organizations }));
vi.mock("~/lib/transport", () => ({
  createAuthInterceptor: (get: any) => {
    mocks.getToken = get;
    return {};
  },
  createWebTransport: (_url: string, _auth: any, interceptor: any) => { mocks.interceptor = interceptor; return {}; },
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
  mocks.organizations = reactive({ currentOrgId: "org-a", reset: vi.fn() });
  mocks.restriction = ref(null);
  vi.stubGlobal("useAccessRestriction", () => mocks.restriction);
  vi.stubGlobal("clearNuxtData", vi.fn());
  vi.stubGlobal("defineNuxtPlugin", (plugin: any) => plugin);
  vi.stubGlobal("useRuntimeConfig", () => ({
    public: { apiHost: "https://api.example.test" },
  }));
  const plugin = (await import("../app/plugins/connect-transport.client"))
    .default;
  scope.run(() => (plugin as any).setup({}));
});

function restricted(scope: string, organizationId = "org-a") {
  return new ConnectError("Restricted", Code.PermissionDenied, {
    "X-Obiente-Restriction": scope,
    "X-Obiente-Organization-ID": organizationId,
  });
}

describe("current moderation state", () => {
  it("clears cached organization data when the account is restricted", async () => {
    const error = restricted("account");
    await expect(mocks.interceptor(async () => { throw error; })({})).rejects.toBe(error);
    expect(mocks.restriction.value).toEqual({ scope: "account" });
    expect(mocks.organizations.reset).toHaveBeenCalled();
    expect(clearNuxtData).toHaveBeenCalled();
  });

  it("does not treat ordinary admin permission errors as an organization ban", async () => {
    const error = new ConnectError("Action denied", Code.PermissionDenied);
    await expect(mocks.interceptor(async () => { throw error; })({})).rejects.toBe(error);
    expect(mocks.restriction.value).toBeNull();
  });

  it("ignores a late account denial after switching identities", async () => {
    let reject!: (error: unknown) => void;
    const pending = mocks.interceptor(() => new Promise((_resolve, fail) => { reject = fail; }))({});
    mocks.auth.session = { id: "b" };
    reject(restricted("account"));
    await expect(pending).rejects.toThrow();
    expect(mocks.restriction.value).toBeNull();
  });

  it("ignores a late denial from the previous organization", async () => {
    let reject!: (error: unknown) => void;
    const pending = mocks.interceptor(() => new Promise((_resolve, fail) => { reject = fail; }))({});
    mocks.organizations.currentOrgId = "org-b";
    reject(restricted("organization"));
    await expect(pending).rejects.toThrow();
    expect(mocks.restriction.value).toBeNull();
  });

  it("clears an organization restriction after selecting another organization", async () => {
    await expect(mocks.interceptor(async () => { throw restricted("organization"); })({})).rejects.toThrow();
    expect(mocks.restriction.value).toEqual({ scope: "organization", organizationId: "org-a" });
    mocks.organizations.currentOrgId = "org-b";
    await nextTick();
    expect(mocks.restriction.value).toBeNull();
  });

  it("handles a ban received after a stream has started", async () => {
    const response = await mocks.interceptor(async () => ({
      stream: true,
      message: (async function* () { yield "first"; throw restricted("account"); })(),
    }))({});
    const iterator = response.message[Symbol.asyncIterator]();
    expect((await iterator.next()).value).toBe("first");
    await expect(iterator.next()).rejects.toThrow();
    expect(mocks.restriction.value).toEqual({ scope: "account" });
  });
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
