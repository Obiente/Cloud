<template>
  <div class="p-6">
    <OuiStack gap="lg">
      <!-- Header -->
      <OuiStack gap="xs">
        <OuiText as="h2" size="xl" weight="bold">Connected Accounts</OuiText>
        <OuiText size="sm" color="secondary">
          Manage GitHub connections for your account and organizations
        </OuiText>
      </OuiStack>

      <!-- Loading State -->
      <OuiCard v-if="isLoading" variant="outline">
        <OuiCardBody>
          <OuiFlex align="center" gap="md">
            <div
              class="h-4 w-4 border-2 border-primary border-t-transparent rounded-full animate-spin"
            />
            <OuiText size="sm" color="secondary"
              >Loading connected accounts...</OuiText
            >
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Connected Accounts List -->
      <OuiCard v-else-if="integrations.length > 0" variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiText as="h3" size="lg" weight="semibold"
              >Active Connections</OuiText
            >

            <OuiStack gap="sm">
              <OuiCard
                v-for="integration in integrations"
                :key="integration.id"
                variant="outline"
                class="border-default"
              >
                <OuiCardBody>
                  <OuiFlex justify="between" align="center">
                    <OuiFlex align="center" gap="md">
                      <Icon name="uil:github" class="h-5 w-5" />
                      <OuiStack gap="xs">
                        <OuiFlex align="center" gap="sm">
                          <OuiText size="sm" weight="medium">{{
                            integration.username
                          }}</OuiText>
                          <OuiBox
                            v-if="integration.isUser"
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-primary/10 text-primary text-xs"
                          >
                            Personal
                          </OuiBox>
                          <OuiBox
                            v-else
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-secondary/10 text-secondary text-xs"
                          >
                            {{
                              integration.organizationName ||
                              integration.organizationId
                            }}
                          </OuiBox>
                        </OuiFlex>
                        <OuiText size="xs" color="secondary">
                          Connected {{ formatDate(integration.connectedAt) }}
                        </OuiText>
                      </OuiStack>
                    </OuiFlex>
                    <OuiButton
                      @click="disconnectIntegration(integration)"
                      :disabled="isDisconnecting"
                      variant="ghost"
                      size="sm"
                      color="danger"
                    >
                      Disconnect
                    </OuiButton>
                  </OuiFlex>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Empty State -->
      <OuiCard v-else variant="outline">
        <OuiCardBody>
          <OuiStack gap="md" align="center" class="py-8">
            <Icon name="uil:github" class="h-12 w-12 text-text-secondary" />
            <OuiStack gap="xs" align="center">
              <OuiText size="lg" weight="semibold"
                >No connected accounts</OuiText
              >
              <OuiText size="sm" color="secondary" class="text-center max-w-md">
                Connect your GitHub account or an organization account to enable
                repository imports and deployments
              </OuiText>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Connect New Account -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiStack gap="xs">
              <OuiText as="h3" size="lg" weight="semibold"
                >Connect GitHub Account</OuiText
              >
              <OuiText size="sm" color="secondary">
                Connect a GitHub account or organization to enable repository
                imports and deployments. You can connect multiple accounts.
              </OuiText>
            </OuiStack>

            <!-- Account Type Selection -->
            <OuiStack gap="md">
              <OuiText size="sm" weight="medium">Connect as:</OuiText>
              <OuiRadioGroup
                v-model="connectionType"
                :options="[
                  { label: 'Personal Account', value: 'user' },
                  { label: 'Organization', value: 'organization' },
                ]"
              />

              <!-- Organization Selector -->
              <OuiStack v-if="connectionType === 'organization'" gap="sm">
                <OuiText size="sm" weight="medium"
                  >Select Obiente Organization:</OuiText
                >
                <OuiSelect
                  v-model="selectedOrgId"
                  :items="organizationOptions"
                  placeholder="Select an organization"
                />
                <OuiText size="xs" color="secondary">
                  This will link a GitHub organization to your selected Obiente
                  cloud organization.
                </OuiText>
              </OuiStack>

              <!-- Check if already connected -->
              <OuiCard
                v-if="
                  connectionType === 'user' &&
                  integrations.some((i) => i.isUser)
                "
                variant="outline"
                class="bg-warning/5 border-warning/20"
              >
                <OuiCardBody>
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="medium" color="warning">
                      Personal account already connected
                    </OuiText>
                    <OuiText size="xs" color="secondary">
                      Your personal GitHub account ({{
                        integrations.find((i) => i.isUser)?.username
                      }}) is already connected. Clicking "Connect GitHub" will
                      update the existing connection.
                    </OuiText>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>

              <OuiCard
                v-if="
                  connectionType === 'organization' &&
                  selectedOrgId &&
                  integrations.some(
                    (i) => !i.isUser && i.organizationId === selectedOrgId
                  )
                "
                variant="outline"
                class="bg-warning/5 border-warning/20"
              >
                <OuiCardBody>
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="medium" color="warning">
                      Organization already connected
                    </OuiText>
                    <OuiText size="xs" color="secondary">
                      This organization already has a GitHub integration ({{
                        integrations.find(
                          (i) => !i.isUser && i.organizationId === selectedOrgId
                        )?.username
                      }}). Clicking "Connect GitHub" will update the existing
                      connection.
                    </OuiText>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>

              <OuiFlex gap="sm" align="center">
                <OuiButton
                  @click="connectGitHub"
                  :disabled="
                    isConnecting ||
                    (connectionType === 'organization' && !selectedOrgId)
                  "
                  variant="solid"
                >
                  {{ isConnecting ? "Connecting..." : "Connect GitHub" }}
                </OuiButton>
              </OuiFlex>
            </OuiStack>

            <!-- Connection Info -->
            <OuiStack gap="sm">
              <OuiText
                size="xs"
                weight="semibold"
                transform="uppercase"
                color="secondary"
              >
                Benefits
              </OuiText>
              <ul
                class="list-disc list-inside space-y-1 text-sm text-text-secondary"
              >
                <li>Import repositories directly from GitHub</li>
                <li>Auto-detect branches and load docker-compose.yml files</li>
                <li>Automatic deployments on push (coming soon)</li>
                <li>Access to private repositories</li>
              </ul>
            </OuiStack>

            <OuiText v-if="error" size="xs" color="danger">{{ error }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Other integrations placeholder -->
      <OuiCard variant="outline" class="opacity-50">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiFlex align="center" gap="sm">
              <Icon name="uil:gitlab" class="h-6 w-6" />
              <OuiText as="h2" size="lg" weight="semibold">GitLab</OuiText>
            </OuiFlex>
            <OuiText size="sm" color="secondary">
              Coming soon - Connect your GitLab account for repository
              integration
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </div>
</template>

<script setup lang="ts">
  import { ref, onMounted, computed } from "vue";
  import { useRoute, useRouter } from "vue-router";
  import { useOrganizationsStore } from "~/stores/organizations";

  const route = useRoute();
  const router = useRouter();
  const orgsStore = useOrganizationsStore();

  const integrations = ref<
    Array<{
      id: string;
      username: string;
      scope: string;
      isUser: boolean;
      organizationId?: string;
      organizationName?: string;
      connectedAt?: { seconds: number; nanos: number };
    }>
  >([]);
  const isLoading = ref(true);
  const isConnecting = ref(false);
  const isDisconnecting = ref(false);
  const error = ref("");
  const connectionType = ref<"user" | "organization">("user");
  const selectedOrgId = ref<string>("");

  const organizationOptions = computed(() => {
    return orgsStore.orgs.map((org) => ({
      label: org.name || org.id,
      value: org.id,
    }));
  });

  // Check if we're coming from a callback
  onMounted(async () => {
    const provider = route.query.provider;
    if (provider === "github") {
      // Handle OAuth callback results (success/error)
      const success = route.query.success;
      const errorParam = route.query.error;
      const username = route.query.username;
      const orgId = route.query.orgId;

      if (success === "true" && username) {
        // Successfully connected - reload integrations
        await loadIntegrations();
        error.value = "";
        // Clean up URL
        router.replace({ query: { tab: route.query.tab } });
      } else if (errorParam) {
        // Handle errors from callback
        const errorMsg = String(errorParam);
        if (errorMsg === "missing_code") {
          error.value =
            "Authorization code missing. Please try connecting again.";
        } else if (errorMsg === "configuration_error") {
          error.value =
            "GitHub integration is not properly configured. Please contact your administrator.";
        } else if (errorMsg === "token_exchange_failed") {
          error.value =
            "Failed to complete GitHub connection. Please try again.";
        } else if (errorMsg.includes("Please log in")) {
          error.value = errorMsg;
        } else {
          error.value = `GitHub connection failed: ${errorMsg}`;
        }
        // Clean up URL
        router.replace({ query: { tab: route.query.tab } });
      }
    }

    // Load integrations
    await loadIntegrations();
  });

  const loadIntegrations = async () => {
    isLoading.value = true;
    error.value = "";

    try {
      const { useConnectClient } = await import("~/lib/connect-client");
      const { AuthService, ListGitHubIntegrationsRequestSchema } = await import(
        "@obiente/proto"
      );
      const { create } = await import("@bufbuild/protobuf");

      const client = useConnectClient(AuthService);

      const request = create(ListGitHubIntegrationsRequestSchema, {});
      const response = await client.listGitHubIntegrations(request);

      integrations.value = response.integrations.map((i) => ({
        id: i.id,
        username: i.username,
        scope: i.scope,
        isUser: i.isUser,
        organizationId: i.organizationId || undefined,
        organizationName: i.organizationName || undefined,
        connectedAt: i.connectedAt
          ? {
              seconds: Number(i.connectedAt.seconds),
              nanos: Number(i.connectedAt.nanos),
            }
          : undefined,
      }));
    } catch (err) {
      console.error("Failed to load GitHub integrations:", err);
      error.value = "Failed to load connected accounts. Please try again.";
      integrations.value = [];
    } finally {
      isLoading.value = false;
    }
  };

  const connectGitHub = () => {
    error.value = "";
    isConnecting.value = true;

    const config = useRuntimeConfig();
    const githubClientId = config.public.githubClientId;

    if (!githubClientId || githubClientId === "") {
      error.value =
        "GitHub Client ID not configured. Please set NUXT_PUBLIC_GITHUB_CLIENT_ID in your .env file.";
      isConnecting.value = false;
      return;
    }

    // Callback URL - must match EXACTLY what's configured in GitHub OAuth App
    // Do NOT add query parameters here - GitHub will reject it
    const redirectUri = `${window.location.origin}/api/github/callback`;
    // Required scopes:
    // - repo: Full repository access (read/write, webhooks, deployments)
    // - read:user: Read user profile information
    // - admin:repo_hook: Full control of repository hooks (needed for autodeploy webhooks)
    const scope = "repo read:user admin:repo_hook";

    // Generate a random state for CSRF protection
    const randomState = generateState();

    // Encode connection type and org ID in the state parameter
    // GitHub will pass this back to us in the callback
    const stateData = {
      random: randomState,
      type: connectionType.value,
      orgId:
        connectionType.value === "organization" && selectedOrgId.value
          ? selectedOrgId.value
          : undefined,
    };

    // Encode state data as base64 JSON for transmission
    const state = btoa(JSON.stringify(stateData));

    // Also store in sessionStorage as backup (for client-side verification)
    sessionStorage.setItem("github_oauth_state", state);

    // Build GitHub OAuth authorization URL
    // Add prompt=select_account to force GitHub to show account/organization selection screen
    // This ensures users can select which account or organization to authorize, even if already logged in
    const authUrl = `https://github.com/login/oauth/authorize?client_id=${githubClientId}&redirect_uri=${encodeURIComponent(
      redirectUri
    )}&scope=${encodeURIComponent(scope)}&state=${encodeURIComponent(
      state
    )}&prompt=select_account`;

    window.location.href = authUrl;
  };

  const generateState = () => {
    return (
      Math.random().toString(36).substring(2, 15) +
      Math.random().toString(36).substring(2, 15)
    );
  };

  const disconnectIntegration = async (
    integration: (typeof integrations.value)[0]
  ) => {
    const accountName = integration.isUser
      ? `your personal account (${integration.username})`
      : `organization ${
          integration.organizationName || integration.organizationId
        } (${integration.username})`;

    const { showConfirm } = useDialog();
    const confirmed = await showConfirm({
      title: "Disconnect GitHub Account",
      message: `Are you sure you want to disconnect ${accountName}?`,
      confirmLabel: "Disconnect",
      cancelLabel: "Cancel",
      variant: "danger",
    });

    if (!confirmed) {
      return;
    }

    isDisconnecting.value = true;
    error.value = "";

    try {
      const { useConnectClient } = await import("~/lib/connect-client");
      const {
        AuthService,
        DisconnectGitHubRequestSchema,
        DisconnectOrganizationGitHubRequestSchema,
      } = await import("@obiente/proto");
      const { create } = await import("@bufbuild/protobuf");

      const client = useConnectClient(AuthService);

      if (integration.isUser) {
        const request = create(DisconnectGitHubRequestSchema, {});
        await client.disconnectGitHub(request);
      } else if (integration.organizationId) {
        const request = create(DisconnectOrganizationGitHubRequestSchema, {
          organizationId: integration.organizationId,
        });
        await client.disconnectOrganizationGitHub(request);
      }

      // Reload integrations list
      await loadIntegrations();
    } catch (err: any) {
      console.error("Failed to disconnect GitHub:", err);
      error.value =
        err.message || "Failed to disconnect GitHub account. Please try again.";
    } finally {
      isDisconnecting.value = false;
    }
  };

  const formatDate = (timestamp?: { seconds: number; nanos: number }) => {
    if (!timestamp) return "recently";

    const date = new Date(timestamp.seconds * 1000);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return "today";
    if (diffDays === 1) return "yesterday";
    if (diffDays < 7) return `${diffDays} days ago`;
    if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`;
    if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`;
    return `${Math.floor(diffDays / 365)} years ago`;
  };
</script>
