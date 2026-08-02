import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  installState: "signed-install-state",
  setInstallStateCookie: vi.fn(),
}));

vi.mock("../server/utils/githubAppInstallState", () => ({
  createGitHubAppInstallNonce: () => "n".repeat(43),
  encodeGitHubAppInstallState: () => mocks.installState,
  setGitHubAppInstallStateCookie: mocks.setInstallStateCookie,
}));

describe("GitHub App install route", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.clearAllMocks();
    vi.stubGlobal("defineEventHandler", <T>(handler: T) => handler);
    vi.stubGlobal("getQuery", () => ({ orgId: "workspace-a" }));
    vi.stubGlobal("useRuntimeConfig", () => ({
      public: { githubAppSlug: "obiente-cloud" },
    }));
    vi.stubGlobal(
      "sendRedirect",
      vi.fn((_event, location: string) => location)
    );
  });

  it("uses GitHub's supported state-preserving installation URL", async () => {
    const { default: installGitHubApp } = await import(
      "../server/api/github/app/install.get"
    );

    const location = await installGitHubApp({} as never);
    const url = new URL(String(location));

    expect(url.origin).toBe("https://github.com");
    expect(url.pathname).toBe("/apps/obiente-cloud/installations/new");
    expect(url.searchParams.get("state")).toBe(mocks.installState);
    expect(mocks.setInstallStateCookie).toHaveBeenCalledOnce();
  });
});
