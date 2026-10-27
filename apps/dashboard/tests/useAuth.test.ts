import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  computed,
  effectScope,
  reactive,
  readonly,
  ref,
  watch,
  type EffectScope,
  type Ref,
} from "vue";

vi.mock("@nuxt/nitro-server/h3", () => ({ appendResponseHeader: vi.fn() }));
const mocks = vi.hoisted(() => ({ reset: vi.fn() }));
vi.mock("~/stores/organizations", () => ({
  useOrganizationsStore: () => ({
    reset: mocks.reset,
    orgs: [],
    currentOrgId: "org-a",
  }),
}));
const deferred = <T>() => {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
};
const identity = (sub: string) => ({ id: `session-${sub}`, user: { sub } });
let values: Map<string, Ref<any>>;
let scope: EffectScope;
let request: ReturnType<typeof vi.fn>;
let tokenRequest: ReturnType<typeof vi.fn>;
let useAuth: typeof import("../app/composables/useAuth").useAuth;
const auth = () => scope.run(() => useAuth())!;
const switchIdentity = () => {
  values.get("obiente-session")!.value = identity("b");
};

beforeEach(async () => {
  vi.resetModules();
  vi.clearAllMocks();
  values = new Map([["obiente-session", ref(identity("a"))]]);
  scope = effectScope();
  request = vi.fn();
  tokenRequest = vi.fn();
  for (const [key, value] of Object.entries({
    computed,
    reactive,
    readonly,
    ref,
    watch,
  }))
    vi.stubGlobal(key, value);
  vi.stubGlobal("useState", (key: string, init: () => unknown) => {
    if (!values.has(key)) values.set(key, ref(init()));
    return values.get(key);
  });
  vi.stubGlobal("useRequestFetch", () => request);
  vi.stubGlobal("$fetch", tokenRequest);
  vi.stubGlobal("onNuxtReady", vi.fn());
  vi.stubGlobal("requestAnimationFrame", vi.fn());
  vi.stubGlobal("sessionStorage", { setItem: vi.fn() });
  vi.stubGlobal("useRuntimeConfig", () => ({
    public: {
      requestHost: "https://dashboard.example.test",
      oidcBase: "https://identity.example.test",
      oidcClientId: "example",
    },
  }));
  vi.stubGlobal("window", {
    location: { origin: "https://dashboard.example.test", href: "" },
  });
  useAuth = (await import("../app/composables/useAuth")).useAuth;
});
afterEach(() => {
  scope.stop();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("authentication response ordering", () => {
  it("discards an old token response after the account changes", async () => {
    const old = deferred<any>();
    tokenRequest.mockReturnValueOnce(old.promise);
    const client = auth();
    const pending = client.getAccessToken();
    switchIdentity();
    expect(tokenRequest.mock.calls[0]?.[1].signal.aborted).toBe(true);
    old.resolve({ accessToken: "old-account-token", expiresIn: 3600 });
    expect(await pending).toBeNull();
    expect(values.get("auth-token")!.value).toBeNull();
    expect(mocks.reset).toHaveBeenCalledTimes(1);
  });
  it("clears a cached token synchronously on an account switch", () => {
    auth();
    values.get("auth-token")!.value = "old-token";
    switchIdentity();
    expect(values.get("auth-token")!.value).toBeNull();
  });
  it("does not let a stale refresh failure log out a newer account", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const old = deferred<any>();
    tokenRequest.mockReturnValue(old.promise);
    const client = auth();
    const pending = client.refreshAccessToken(true);
    switchIdentity();
    values.get("auth-token")!.value = "new-account-token";
    old.reject({ statusCode: 401 });
    await pending;
    expect(request).not.toHaveBeenCalled();
    expect(values.get("auth-token")!.value).toBe("new-account-token");
  });
  it("shares concurrent refreshes across composable instances", async () => {
    const response = deferred<any>();
    tokenRequest.mockReturnValue(response.promise);
    const a = auth().refreshAccessToken(true),
      b = auth().refreshAccessToken(true);
    response.resolve({ accessToken: "latest", expiresIn: 3600 });
    await Promise.all([a, b]);
    expect(tokenRequest).toHaveBeenCalledTimes(1);
    expect(values.get("auth-token")!.value).toBe("latest");
  });
  it("discards a delayed session response from a previous identity", async () => {
    const old = deferred<any>();
    request.mockReturnValueOnce(old.promise);
    const client = auth();
    const pending = client.fetch();
    switchIdentity();
    old.resolve(identity("a"));
    await pending;
    expect(client.user?.sub).toBe("b");
  });
  it("uses the latest session request regardless of response order", async () => {
    const first = deferred<any>(),
      second = deferred<any>();
    request
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(second.promise);
    const client = auth();
    const a = client.fetch(),
      b = client.fetch();
    second.resolve(identity("b"));
    await b;
    first.resolve(identity("a"));
    await a;
    expect(client.user?.sub).toBe("b");
  });
  it("preserves the session during a temporary session endpoint outage", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    request.mockRejectedValueOnce({ statusCode: 503 });
    const client = auth();
    await client.fetch();
    expect(client.user?.sub).toBe("a");
    expect(client.ready).toBe(true);
  });
  it("invalidates in-flight session and token reads as soon as logout begins", async () => {
    const sessionRead = deferred<any>(),
      tokenRead = deferred<any>(),
      deletion = deferred<any>();
    request
      .mockReturnValueOnce(sessionRead.promise)
      .mockReturnValueOnce(deletion.promise);
    tokenRequest.mockReturnValueOnce(tokenRead.promise);
    const client = auth();
    const a = client.fetch(),
      b = client.getAccessToken(),
      c = client.logout();
    expect(client.user).toBeNull();
    sessionRead.resolve(identity("a"));
    tokenRead.resolve({ accessToken: "old", expiresIn: 3600 });
    await Promise.all([a, b]);
    expect(client.user).toBeNull();
    expect(values.get("auth-token")!.value).toBeNull();
    deletion.resolve({});
    await c;
  });
  it("forces an actual refresh instead of rereading an expired token", async () => {
    tokenRequest.mockResolvedValue({ accessToken: "fresh", expiresIn: 3600 });
    await auth().refreshAccessToken(true);
    expect(tokenRequest.mock.calls[0]?.[0]).toBe("/auth/refresh");
  });
});
