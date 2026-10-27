import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
const mocks = vi.hoisted(() => ({
  session: null as any,
  setSession: vi.fn(),
  clear: vi.fn(),
  fetch: vi.fn(),
  cookie: vi.fn(),
}));
vi.mock("h3", () => ({
  setCookie: mocks.cookie,
  setHeader: vi.fn(),
  eventHandler: (handler: any) => handler,
  createError: (options: any) =>
    Object.assign(new Error(options.message), options),
}));
vi.mock("@nuxt/nitro-server/h3", () => ({
  setCookie: mocks.cookie,
  setHeader: vi.fn(),
  eventHandler: (handler: any) => handler,
  createError: (options: any) =>
    Object.assign(new Error(options.message), options),
}));
vi.mock("../server/utils/session", () => ({
  getUserSession: async () => mocks.session,
  setUserSession: mocks.setSession,
  clearUserSession: mocks.clear,
}));
beforeEach(() => {
  vi.resetModules();
  vi.clearAllMocks();
  vi.stubEnv("DISABLE_AUTH", "false");
  mocks.session = {
    id: "example-session",
    user: { sub: "example-user" },
    secure: {
      access_token: "old-access",
      refresh_token: "example-refresh",
      id_token: "example-id",
      expires_in: 3600,
    },
  };
  mocks.setSession.mockImplementation(async (_event: any, data: any) => {
    mocks.session = { ...mocks.session, ...data };
    return mocks.session;
  });
  vi.stubGlobal("getUserSession", async () => mocks.session);
  vi.stubGlobal("setUserSession", mocks.setSession);
  vi.stubGlobal("clearUserSession", mocks.clear);
  vi.stubGlobal("setCookie", mocks.cookie);
  vi.stubGlobal("defineEventHandler", (handler: any) => handler);
  vi.stubGlobal("createError", (options: any) =>
    Object.assign(new Error(options.message), options)
  );
  vi.stubGlobal("useRuntimeConfig", () => ({
    public: {
      oidcBase: "https://identity.example.test",
      oidcClientId: "example-client",
    },
  }));
  vi.stubGlobal("$fetch", mocks.fetch);
  vi.spyOn(console, "error").mockImplementation(() => {});
  vi.spyOn(console, "warn").mockImplementation(() => {});
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
  vi.restoreAllMocks();
});
describe("server authentication recovery", () => {
  it("reports remaining token lifetime instead of restarting its original lifetime", async () => {
    const route = (await import("../server/routes/auth/token.get")).default;
    mocks.session.secure.expires_at = Date.now() + 90000;
    const result = await route({} as never);
    expect(result.expiresIn).toBeGreaterThanOrEqual(89);
    expect(result.expiresIn).toBeLessThanOrEqual(90);
  });
  it("does not return an already expired token", async () => {
    mocks.session.secure.expires_at = Date.now() - 1000;
    const route = (await import("../server/routes/auth/token.get")).default;
    expect((await route({} as never)).accessToken).toBeNull();
  });
  it("preserves the session when renewal temporarily fails", async () => {
    mocks.fetch
      .mockRejectedValueOnce({ statusCode: 401 })
      .mockRejectedValueOnce({ statusCode: 503 });
    const { getUserData } = await import("../server/utils/auth");
    await expect(getUserData({} as never, mocks.session)).rejects.toMatchObject(
      { statusCode: 503 }
    );
    expect(mocks.clear).not.toHaveBeenCalled();
    expect(mocks.setSession).not.toHaveBeenCalled();
  });
  it("clears an invalid refresh grant and returns unauthorized", async () => {
    mocks.fetch.mockRejectedValueOnce(
      Object.assign(new Error("Bad Request"), {
        statusCode: 400,
        data: { error: "invalid_grant" },
      })
    );
    const route = (await import("../server/routes/auth/refresh.post")).default;
    await expect(route({} as never)).rejects.toMatchObject({ statusCode: 401 });
    expect(mocks.clear).toHaveBeenCalledOnce();
  });
  it("validates provider tokens before saving session or cookies", async () => {
    mocks.fetch.mockResolvedValueOnce({ access_token: "", expires_in: 3600 });
    const route = (await import("../server/routes/auth/refresh.post")).default;
    await expect(route({} as never)).rejects.toBeDefined();
    expect(mocks.setSession).not.toHaveBeenCalled();
    expect(mocks.cookie).not.toHaveBeenCalled();
  });
  it("preserves the refresh and ID tokens if a successful renewal omits them", async () => {
    mocks.fetch.mockResolvedValueOnce({
      access_token: "new-access",
      expires_in: 600,
      token_type: "Bearer",
      scope: "openid",
    });
    const route = (await import("../server/routes/auth/refresh.post")).default;
    expect((await route({} as never)).accessToken).toBe("new-access");
    expect(mocks.session.secure.refresh_token).toBe("example-refresh");
    expect(mocks.session.secure.id_token).toBe("example-id");
    expect(mocks.session.secure.expires_at).toBeGreaterThan(
      Date.now() + 590000
    );
  });
  it("renews on the provided response event and never exposes secure session data", async () => {
    mocks.fetch
      .mockRejectedValueOnce({ statusCode: 401 })
      .mockResolvedValueOnce({
        access_token: "renewed-access",
        expires_in: 600,
        token_type: "Bearer",
        scope: "openid",
      })
      .mockResolvedValueOnce({ sub: "example-user", name: "Example User" });
    const { readCurrentSession } = await import(
      "../server/utils/current-session"
    );
    const event = {} as never;
    const session = await readCurrentSession(event);
    expect(mocks.cookie).toHaveBeenCalledWith(
      event,
      "obiente_auth",
      "renewed-access",
      expect.any(Object)
    );
    expect(
      mocks.setSession.mock.calls.every(
        ([usedEvent]: any) => usedEvent === event
      )
    ).toBe(true);
    expect(session.user?.sub).toBe("example-user");
    expect(session).not.toHaveProperty("secure");
  });

  it("retains a known server session during a provider outage without exposing its tokens", async () => {
    mocks.fetch.mockRejectedValueOnce({ statusCode: 503 });
    const { readCurrentSession } = await import(
      "../server/utils/current-session"
    );
    const result = await readCurrentSession({} as never);
    expect(result.user?.sub).toBe("example-user");
    expect(result).not.toHaveProperty("secure");
    expect(mocks.clear).not.toHaveBeenCalled();
  });
});
