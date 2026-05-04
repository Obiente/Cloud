import { createHmac, timingSafeEqual } from "node:crypto";
import type { H3Event } from "h3";
import { createError, deleteCookie, getCookie, setCookie } from "h3";

export const GITHUB_APP_INSTALL_STATE_COOKIE = "github_app_install_state";

const GITHUB_APP_INSTALL_STATE_MAX_AGE_SECONDS = 10 * 60;

export interface GitHubAppInstallState {
  random: string;
  orgId: string;
  installationId?: string;
  repositorySelection?: string;
}

export function encodeGitHubAppInstallState(
  event: H3Event,
  state: GitHubAppInstallState
): string {
  const payload = Buffer.from(JSON.stringify(state), "utf-8").toString(
    "base64url"
  );
  const signature = signGitHubAppInstallState(event, payload);
  return `${payload}.${signature}`;
}

export function decodeGitHubAppInstallState(state: string): GitHubAppInstallState {
  let parsed: unknown;

  try {
    const payload = state.includes(".") ? state.split(".")[0] || "" : state;
    parsed = JSON.parse(Buffer.from(payload, "base64").toString("utf-8"));
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

  const { random, orgId, installationId, repositorySelection } = parsed as {
    random?: unknown;
    orgId?: unknown;
    installationId?: unknown;
    repositorySelection?: unknown;
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

  return {
    random,
    orgId,
    installationId:
      typeof installationId === "string" && installationId.length > 0
        ? installationId
        : undefined,
    repositorySelection:
      typeof repositorySelection === "string" && repositorySelection.length > 0
        ? repositorySelection
        : undefined,
  };
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
  event: H3Event,
  expectedState: string | undefined,
  actualState: string | undefined
): boolean {
  if (!actualState) {
    return false;
  }

  if (expectedState) {
    const expected = Buffer.from(expectedState, "utf-8");
    const actual = Buffer.from(actualState, "utf-8");
    if (expected.length === actual.length && timingSafeEqual(expected, actual)) {
      return true;
    }
  }

  return verifySignedGitHubAppInstallState(event, actualState);
}

function signGitHubAppInstallState(event: H3Event, payload: string): string {
  return createHmac("sha256", getGitHubAppInstallStateSecret(event))
    .update(payload)
    .digest("base64url");
}

function verifySignedGitHubAppInstallState(
  event: H3Event,
  state: string
): boolean {
  const [payload, signature, extra] = state.split(".");
  if (!payload || !signature || extra !== undefined) {
    return false;
  }

  const expectedSignature = signGitHubAppInstallState(event, payload);
  const expected = Buffer.from(expectedSignature, "utf-8");
  const actual = Buffer.from(signature, "utf-8");
  return expected.length === actual.length && timingSafeEqual(expected, actual);
}

function getGitHubAppInstallStateSecret(event: H3Event): string {
  const config = useRuntimeConfig(event);
  const secret =
    config.session?.password ||
    process.env.NUXT_SESSION_PASSWORD ||
    process.env.SESSION_SECRET ||
    process.env.SECRET ||
    "";

  if (typeof secret !== "string" || secret.length < 32) {
    throw createError({
      statusCode: 500,
      statusMessage:
        "GitHub App install state signing requires NUXT_SESSION_PASSWORD or another strong server secret",
    });
  }

  return secret;
}
