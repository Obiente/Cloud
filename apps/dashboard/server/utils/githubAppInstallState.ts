import { timingSafeEqual } from "node:crypto";
import type { H3Event } from "h3";
import { createError, deleteCookie, getCookie, setCookie } from "h3";

export const GITHUB_APP_INSTALL_STATE_COOKIE = "github_app_install_state";

const GITHUB_APP_INSTALL_STATE_MAX_AGE_SECONDS = 10 * 60;

export interface GitHubAppInstallState {
  random: string;
  orgId: string;
}

export function encodeGitHubAppInstallState(state: GitHubAppInstallState): string {
  return Buffer.from(JSON.stringify(state), "utf-8").toString("base64");
}

export function decodeGitHubAppInstallState(state: string): GitHubAppInstallState {
  let parsed: unknown;

  try {
    parsed = JSON.parse(Buffer.from(state, "base64").toString("utf-8"));
  } catch {
    throw createError({
      statusCode: 400,
      statusMessage: "invalid GitHub App install state encoding",
    });
  }

  if (!parsed || typeof parsed !== "object") {
    throw createError({
      statusCode: 400,
      statusMessage: "invalid GitHub App install state payload",
    });
  }

  const { random, orgId } = parsed as {
    random?: unknown;
    orgId?: unknown;
  };

  if (typeof random !== "string" || random.length < 16) {
    throw createError({
      statusCode: 400,
      statusMessage: "invalid GitHub App install nonce",
    });
  }

  if (typeof orgId !== "string" || orgId.length === 0) {
    throw createError({
      statusCode: 400,
      statusMessage: "GitHub App install state is missing orgId",
    });
  }

  return { random, orgId };
}

export function getGitHubAppInstallStateCookie(event: H3Event): string | undefined {
  return getCookie(event, GITHUB_APP_INSTALL_STATE_COOKIE);
}

export function setGitHubAppInstallStateCookie(event: H3Event, state: string): void {
  setCookie(event, GITHUB_APP_INSTALL_STATE_COOKIE, state, {
    httpOnly: true,
    maxAge: GITHUB_APP_INSTALL_STATE_MAX_AGE_SECONDS,
    path: "/api/github",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

export function clearGitHubAppInstallStateCookie(event: H3Event): void {
  deleteCookie(event, GITHUB_APP_INSTALL_STATE_COOKIE, {
    path: "/api/github",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

export function verifyGitHubAppInstallState(
  expectedState: string | undefined,
  actualState: string | undefined
): boolean {
  if (!expectedState || !actualState) {
    return false;
  }

  const expected = Buffer.from(expectedState, "utf-8");
  const actual = Buffer.from(actualState, "utf-8");
  if (expected.length !== actual.length) {
    return false;
  }

  return timingSafeEqual(expected, actual);
}
