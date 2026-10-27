import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { computed, reactive, readonly, ref } from "vue";
const mocks = vi.hoisted(() => ({
  event: {} as any,
  current: vi.fn(),
  fetch: vi.fn(),
  append: vi.fn(),
}));
vi.mock("@nuxt/nitro-server/h3", () => ({
  appendResponseHeader: mocks.append,
}));
vi.mock("~/stores/organizations", () => ({
  useOrganizationsStore: () => ({ orgs: [], reset: vi.fn() }),
}));
vi.mock("../server/utils/current-session", () => ({
  readCurrentSession: mocks.current,
}));
beforeEach(() => {
  vi.resetModules();
  vi.clearAllMocks();
  const states = new Map();
  mocks.event = { context: {} };
  for (const [key, value] of Object.entries({
    computed,
    reactive,
    readonly,
    ref,
  }))
    vi.stubGlobal(key, value);
  vi.stubGlobal("useState", (key: string, init: () => unknown) => {
    if (!states.has(key)) states.set(key, ref(init()));
    return states.get(key);
  });
  vi.stubGlobal("useRequestEvent", () => mocks.event);
  vi.stubGlobal("useRuntimeConfig", () => ({
    session: { password: "synthetic-password" },
    public: {},
  }));
  vi.stubGlobal("useRequestFetch", () => mocks.fetch);
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});
describe("server-rendered session", () => {
  it("renews the session on the actual response event and strips secrets from hydration", async () => {
    mocks.current.mockImplementation(async (event: any) => {
      event.context.renewed = true;
      return { id: "example-session", user: { sub: "example-user" } };
    });
    const { useAuth } = await import("../app/composables/useAuth");
    const client = useAuth();
    await client.fetch();
    expect(mocks.current).toHaveBeenCalledWith(
      mocks.event,
      expect.any(AbortSignal),
      { session: { password: "synthetic-password" }, public: {} }
    );
    expect(mocks.event.context.renewed).toBe(true);
    expect(mocks.fetch).not.toHaveBeenCalled();
    expect(client.session?.user?.sub).toBe("example-user");
    expect(client.session).not.toHaveProperty("secure");
  });
});
