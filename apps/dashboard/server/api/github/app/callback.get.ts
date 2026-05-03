import {
  clearGitHubOAuthStateCookie,
  decodeGitHubOAuthState,
  getGitHubOAuthStateCookie,
  verifyGitHubOAuthState,
} from "../../../utils/githubOAuth";

export default defineEventHandler(async (event) => {
  const query = getQuery(event);
  const state = typeof query.state === "string" ? query.state : "";
  const installationIdValue =
    typeof query.installation_id === "string" ? query.installation_id : "";
  const setupAction =
    typeof query.setup_action === "string" ? query.setup_action : "";

  const redirectToSettings = (reason: string) =>
    sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&error=${encodeURIComponent(
        reason
      )}`
    );

  const installationId = Number.parseInt(installationIdValue, 10);
  if (!Number.isFinite(installationId) || installationId <= 0) {
    return redirectToSettings("missing_installation");
  }

  if (!state) {
    clearGitHubOAuthStateCookie(event);
    return sendRedirect(
      event,
      `/settings?tab=integrations&provider=github&success=true&installationUpdated=true&installationId=${encodeURIComponent(
        String(installationId)
      )}`
    );
  }

  if (!verifyGitHubOAuthState(getGitHubOAuthStateCookie(event), state)) {
    clearGitHubOAuthStateCookie(event);
    return redirectToSettings("invalid_state");
  }

  let stateData: ReturnType<typeof decodeGitHubOAuthState>;
  try {
    stateData = decodeGitHubOAuthState(state);
  } catch {
    clearGitHubOAuthStateCookie(event);
    return redirectToSettings("invalid_state");
  }
  clearGitHubOAuthStateCookie(event);

  if (stateData.type !== "organization" || !stateData.orgId) {
    return redirectToSettings("missing_organization");
  }

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
      repositorySelection: setupAction || "",
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
