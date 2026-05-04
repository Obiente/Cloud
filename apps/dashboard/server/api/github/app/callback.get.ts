import {
  clearGitHubAppInstallStateCookie,
  encodeGitHubAppInstallState,
  decodeGitHubAppInstallState,
  getGitHubAppInstallStateCookie,
  setGitHubAppInstallStateCookie,
  verifyGitHubAppInstallState,
} from "../../../utils/githubAppInstallState";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const state = typeof query.state === "string" ? query.state : "";
  const installationIdValue =
    typeof query.installation_id === "string" ? query.installation_id : "";
  const setupCode = typeof query.code === "string" ? query.code : "";
  const setupAction =
    typeof query.setup_action === "string" ? query.setup_action : "";

  const redirectToSettings = (reason: string) =>
    sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(
        reason
      )}`
    );

  if (!verifyGitHubAppInstallState(event, getGitHubAppInstallStateCookie(event), state)) {
    clearGitHubAppInstallStateCookie(event);
    return redirectToSettings("invalid_state");
  }

  let stateData: ReturnType<typeof decodeGitHubAppInstallState>;
  try {
    stateData = decodeGitHubAppInstallState(state);
  } catch {
    clearGitHubAppInstallStateCookie(event);
    return redirectToSettings("invalid_state");
  }
  clearGitHubAppInstallStateCookie(event);

  if (!stateData.orgId) {
    return redirectToSettings("missing_organization");
  }

  const installationIdFromQuery = Number.parseInt(installationIdValue, 10);
  const installationIdFromState = Number.parseInt(
    stateData.installationId || "",
    10
  );
  const installationId = Number.isFinite(installationIdFromQuery) && installationIdFromQuery > 0
    ? installationIdFromQuery
    : installationIdFromState;
  if (!Number.isFinite(installationId) || installationId <= 0) {
    clearGitHubAppInstallStateCookie(event);
    return redirectToSettings("missing_installation");
  }

  if (!setupCode) {
    const config = useRuntimeConfig(event);
    const clientId =
      (config.githubAppClientId as string) ||
      process.env.GITHUB_APP_CLIENT_ID ||
      "";
    if (!clientId) {
      clearGitHubAppInstallStateCookie(event);
      return redirectToSettings("github_app_client_not_configured");
    }

    const authState = encodeGitHubAppInstallState(event, {
      random: crypto.randomUUID().replace(/-/g, ""),
      orgId: stateData.orgId,
      installationId: String(installationId),
      repositorySelection: setupAction || stateData.repositorySelection || "",
    });
    setGitHubAppInstallStateCookie(event, authState);

    const authUrl = new URL("https://github.com/login/oauth/authorize");
    authUrl.searchParams.set("client_id", clientId);
    authUrl.searchParams.set("state", authState);
    authUrl.searchParams.set("redirect_uri", getGitHubAppCallbackUrl(event));
    return sendRedirect(event, authUrl.toString());
  }

  clearGitHubAppInstallStateCookie(event);

  const isAuthDisabled = process.env.DISABLE_AUTH === "true";
  const { getServerToken } = await import("../../../utils/serverAuth");
  let userAccessToken = await getServerToken(event);
  if (!userAccessToken && isAuthDisabled) {
    userAccessToken = "dev-dummy-token";
  }
  if (!userAccessToken) {
    return redirectToSettings("Please log in to connect your GitHub organization");
  }

  try {
    const config = useRuntimeConfig(event);
    const { createConnectTransport } = await import("@connectrpc/connect-node");
    const { createClient } = await import("@connectrpc/connect");
    const { createAuthInterceptor } = await import("~/lib/transport");
    const { AuthService, ConnectOrganizationGitHubAppRequestSchema } =
      await import("@obiente/proto");
    const { create } = await import("@bufbuild/protobuf");

    const getToken = () => Promise.resolve(userAccessToken || undefined);
    const authInterceptor = createAuthInterceptor(getToken);
    const createTransport = (baseUrl: string) =>
      createConnectTransport({
        baseUrl,
        httpVersion: "1.1",
        useBinaryFormat: false,
        interceptors: [authInterceptor],
      });

    let apiHost = (config.apiHostInternal as string) || config.public.apiHost;
    let client = createClient(AuthService, createTransport(apiHost));

    const request = create(ConnectOrganizationGitHubAppRequestSchema, {
      organizationId: stateData.orgId,
      installationId: BigInt(installationId),
      accountLogin: "",
      accountType: "Organization",
      repositorySelection: setupAction || stateData.repositorySelection || "",
      setupCode,
    });

    try {
      await client.connectOrganizationGitHubApp(request);
    } catch (err) {
      if (config.apiHostInternal && apiHost === (config.apiHostInternal as string)) {
        apiHost = config.public.apiHost;
        client = createClient(AuthService, createTransport(apiHost));
        await client.connectOrganizationGitHubApp(request);
      } else {
        throw err;
      }
    }

    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&success=true&orgId=${encodeURIComponent(
        stateData.orgId
      )}&installationId=${encodeURIComponent(String(installationId))}`
    );
  } catch (err: any) {
    console.error("[GitHub App] Failed to save installation:", {
      message: err?.message,
      code: err?.code,
    });
    return redirectToSettings(err?.message || "github_app_install_failed");
  }
});

function getGitHubAppCallbackUrl(event: any): string {
  const requestUrl = new URL(
    event.node.req.url || "/",
    `http://${event.node.req.headers.host || "localhost:3000"}`
  );
  const forwardedProto = event.node.req.headers["x-forwarded-proto"];
  const forwardedHost = event.node.req.headers["x-forwarded-host"];
  const protocolHeader = Array.isArray(forwardedProto)
    ? forwardedProto[0]
    : forwardedProto;
  const hostHeader = Array.isArray(forwardedHost)
    ? forwardedHost[0]
    : forwardedHost;
  const protocol =
    protocolHeader || (requestUrl.protocol === "https:" ? "https" : "http");
  const host =
    hostHeader?.split(",")[0]?.trim() ||
    event.node.req.headers.host ||
    requestUrl.host ||
    "localhost:3000";

  return `${protocol}://${host}/api/github/app/callback`;
}
