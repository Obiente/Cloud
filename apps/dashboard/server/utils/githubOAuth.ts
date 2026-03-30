import { timingSafeEqual } from "node:crypto";
import type { H3Event } from "h3";
import { createError, deleteCookie, getCookie, setCookie } from "h3";

export const GITHUB_OAUTH_STATE_COOKIE = "github_oauth_state";

const GITHUB_OAUTH_STATE_MAX_AGE_SECONDS = 10 * 60;

export type GitHubConnectionType = "user" | "organization";

export interface GitHubOAuthState {
  random: string;
  type: GitHubConnectionType;
  orgId?: string;
}

export function buildGitHubCallbackUrl(event: H3Event): string {
  const requestUrl = new URL(
    event.node.req.url || "/",
    `http://${event.node.req.headers.host || "localhost:3000"}`
  );
  const forwardedProto = event.node.req.headers["x-forwarded-proto"];
  const forwardedHost = event.node.req.headers["x-forwarded-host"];
  const forwardedPort = event.node.req.headers["x-forwarded-port"];
  const protocolHeader = Array.isArray(forwardedProto)
    ? forwardedProto[0]
    : forwardedProto;
  const hostHeader = Array.isArray(forwardedHost)
    ? forwardedHost[0]
    : forwardedHost;
  const portHeader = Array.isArray(forwardedPort)
    ? forwardedPort[0]
    : forwardedPort;
  const protocol =
    protocolHeader || (requestUrl.protocol === "https:" ? "https" : "http");
  const normalizedHost = hostHeader?.split(",")[0]?.trim();
  const normalizedPort = portHeader?.split(",")[0]?.trim();
  const fallbackHost =
    event.node.req.headers.host || requestUrl.host || "localhost:3000";

  let host = normalizedHost || fallbackHost;
  if (
    normalizedPort &&
    normalizedHost &&
    !normalizedHost.includes(":") &&
    normalizedPort !== "80" &&
    normalizedPort !== "443"
  ) {
    host = `${normalizedHost}:${normalizedPort}`;
  }

  return `${protocol}://${host}/api/github/callback`;
}

export function encodeGitHubOAuthState(state: GitHubOAuthState): string {
  return Buffer.from(JSON.stringify(state), "utf-8").toString("base64");
}

export function decodeGitHubOAuthState(state: string): GitHubOAuthState {
  let parsed: unknown;

  try {
    parsed = JSON.parse(Buffer.from(state, "base64").toString("utf-8"));
  } catch {
    throw createError({
      statusCode: 400,
      statusMessage: "invalid GitHub OAuth state encoding",
    });
  }

  if (!parsed || typeof parsed !== "object") {
    throw createError({
      statusCode: 400,
      statusMessage: "invalid GitHub OAuth state payload",
    });
  }

  const { random, type, orgId } = parsed as {
    random?: unknown;
    type?: unknown;
    orgId?: unknown;
  };

  if (typeof random !== "string" || random.length < 16) {
    throw createError({
      statusCode: 400,
      statusMessage: "invalid GitHub OAuth nonce",
    });
  }

  if (type !== "user" && type !== "organization") {
    throw createError({
      statusCode: 400,
      statusMessage: "invalid GitHub OAuth connection type",
    });
  }

  if (
    type === "organization" &&
    (typeof orgId !== "string" || orgId.length === 0)
  ) {
    throw createError({
      statusCode: 400,
      statusMessage: "organization GitHub OAuth state is missing orgId",
    });
  }

  return {
    random,
    type,
    orgId: typeof orgId === "string" && orgId.length > 0 ? orgId : undefined,
  };
}

export function getGitHubOAuthStateCookie(event: H3Event): string | undefined {
  return getCookie(event, GITHUB_OAUTH_STATE_COOKIE);
}

export function setGitHubOAuthStateCookie(event: H3Event, state: string): void {
  setCookie(event, GITHUB_OAUTH_STATE_COOKIE, state, {
    httpOnly: true,
    maxAge: GITHUB_OAUTH_STATE_MAX_AGE_SECONDS,
    path: "/api/github",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

export function clearGitHubOAuthStateCookie(event: H3Event): void {
  deleteCookie(event, GITHUB_OAUTH_STATE_COOKIE, {
    path: "/api/github",
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
  });
}

export function verifyGitHubOAuthState(
  expectedState: string | undefined,
  actualState: string | undefined
): boolean {
  if (!expectedState || !actualState) {
    return false;
  }

  const expected = Buffer.from(expectedState, "utf-8");
  const actual = Buffer.from(actualState, "utf-8");
  if (expected.length != actual.length) {
    return false;
  }

  return timingSafeEqual(expected, actual);
}
