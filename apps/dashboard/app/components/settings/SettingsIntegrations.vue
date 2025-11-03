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

      <!-- Error State -->
      <OuiCard v-else-if="error && !isLoading" variant="outline" class="border-danger">
        <OuiCardBody>
          <OuiStack gap="sm">
            <OuiText size="sm" weight="medium" color="danger">
              Error Loading Accounts
            </OuiText>
            <OuiText size="xs" color="secondary">{{ error }}</OuiText>
            <OuiButton size="sm" variant="ghost" @click="loadIntegrations">
              Retry
            </OuiButton>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Connected Accounts List -->
      <div v-else-if="integrations.length > 0">
        <OuiStack gap="md">
          <OuiText as="h3" size="lg" weight="semibold"
            >Active Connections</OuiText
          >

          <!-- Single account - no nested cards to avoid whitespace -->
          <OuiCard
            v-if="integrations.length === 1 && integrations[0]"
            variant="outline"
            class="border-default"
          >
            <OuiCardBody>
              <template v-if="integrations[0]">
                <OuiFlex justify="between" align="center" gap="md">
                  <OuiFlex align="center" gap="md" class="flex-1">
                    <img
                      :src="`https://avatars.githubusercontent.com/${integrations[0].username}`"
                      :alt="integrations[0].username"
                      class="h-12 w-12 rounded-full border border-default"
                      @error="handleAvatarError"
                    />
                    <OuiStack gap="xs" class="flex-1">
                      <OuiStack gap="xs">
                        <OuiFlex align="center" gap="sm" wrap="wrap">
                          <OuiText size="sm" weight="semibold" class="text-text-primary">
                            @{{ integrations[0].username }}
                          </OuiText>
                          <OuiBox
                            v-if="integrations[0].isUser"
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-primary/10 text-primary text-xs font-medium"
                          >
                            Personal
                          </OuiBox>
                          <OuiBox
                            v-else
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-secondary/10 text-secondary text-xs font-medium"
                          >
                            Organization
                          </OuiBox>
                          <OuiBox
                            v-if="!integrations[0].isUser && integrations[0].organizationName"
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-text-secondary/10 text-text-secondary text-xs font-medium"
                          >
                            {{ integrations[0].organizationName }}
                          </OuiBox>
                        </OuiFlex>
                        <OuiFlex align="center" gap="xs" wrap="wrap">
                          <OuiText size="xs" color="secondary">
                            Connected
                            <OuiRelativeTime
                              v-if="integrations[0].connectedAt"
                              :value="new Date((integrations[0].connectedAt.seconds || 0) * 1000)"
                              :style="'short'"
                            />
                          </OuiText>
                          <template v-if="integrations[0].scope">
                            <span class="text-text-tertiary">•</span>
                            <OuiText size="xs" color="secondary">
                              Scopes: {{ formatScopes(integrations[0].scope) }}
                            </OuiText>
                          </template>
                        </OuiFlex>
                      </OuiStack>
                    </OuiStack>
                  </OuiFlex>
                  <OuiButton
                    @click="disconnectIntegration(integrations[0])"
                    :disabled="isDisconnecting"
                    variant="ghost"
                    size="sm"
                    color="danger"
                  >
                    Disconnect
                  </OuiButton>
                </OuiFlex>
              </template>
            </OuiCardBody>
          </OuiCard>

          <!-- Multiple accounts - use list layout -->
          <OuiStack v-else gap="sm">
            <OuiCard
              v-for="integration in integrations"
              :key="integration.id"
              variant="outline"
              class="border-default"
            >
              <OuiCardBody>
                <OuiFlex justify="between" align="center" gap="md">
                  <OuiFlex align="center" gap="md" class="flex-1">
                    <img
                      :src="`https://avatars.githubusercontent.com/${integration.username}`"
                      :alt="integration.username"
                      class="h-10 w-10 rounded-full border border-default"
                      @error="handleAvatarError"
                    />
                    <OuiStack gap="xs" class="flex-1">
                      <OuiStack gap="xs">
                        <OuiFlex align="center" gap="sm" wrap="wrap">
                          <OuiText size="sm" weight="semibold" class="text-text-primary">
                            @{{ integration.username }}
                          </OuiText>
                          <OuiBox
                            v-if="integration.isUser"
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-primary/10 text-primary text-xs font-medium"
                          >
                            Personal
                          </OuiBox>
                          <OuiBox
                            v-else
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-secondary/10 text-secondary text-xs font-medium"
                          >
                            Organization
                          </OuiBox>
                          <OuiBox
                            v-if="!integration.isUser && integration.organizationName"
                            px="xs"
                            py="xs"
                            rounded="sm"
                            class="bg-text-secondary/10 text-text-secondary text-xs font-medium"
                          >
                            {{ integration.organizationName }}
                          </OuiBox>
                        </OuiFlex>
                        <OuiFlex align="center" gap="xs" wrap="wrap">
                          <OuiText size="xs" color="secondary">
                            Connected
                            <OuiRelativeTime
                              v-if="integration.connectedAt"
                              :value="new Date((integration.connectedAt.seconds || 0) * 1000)"
                              :style="'short'"
                            />
                          </OuiText>
                          <template v-if="integration.scope">
                            <span class="text-text-tertiary">•</span>
                            <OuiText size="xs" color="secondary">
                              Scopes: {{ formatScopes(integration.scope) }}
                            </OuiText>
                          </template>
                        </OuiFlex>
                      </OuiStack>
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
      </div>

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

            <!-- Error is shown in dedicated error card above -->
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
  import OuiRelativeTime from "~/components/oui/RelativeTime.vue";

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
  const isLoading = ref(false);
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
    // Reset connecting state in case user navigated back
    isConnecting.value = false;
    
    const provider = route.query.provider;
    if (provider === "github") {
      // Handle OAuth callback results (success/error)
      const success = route.query.success;
      const errorParam = route.query.error;
      const username = route.query.username;
      const orgId = route.query.orgId;

      if (success === "true" && username) {
        // Successfully connected - reload integrations with a small delay to ensure backend processed it
        await new Promise(resolve => setTimeout(resolve, 500));
        await loadIntegrations();
        error.value = "";
        // Show success message briefly
        const successMsg = orgId 
          ? `Successfully connected GitHub organization to ${orgId}`
          : `Successfully connected GitHub account: ${username}`;
        console.log("[SettingsIntegrations]", successMsg);
        // Clean up URL
        router.replace({ query: { tab: route.query.tab } });
      } else if (errorParam) {
        // Handle errors from callback
        const errorMsg = String(errorParam);
        isConnecting.value = false; // Reset on error
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

      integrations.value = (response.integrations || []).map((i) => ({
        id: i.id,
        username: i.username || "Unknown",
        scope: i.scope || "",
        isUser: i.isUser || false,
        organizationId: i.organizationId || undefined,
        organizationName: i.organizationName || undefined,
        connectedAt: i.connectedAt
          ? {
              seconds: Number(i.connectedAt.seconds || 0),
              nanos: Number(i.connectedAt.nanos || 0),
            }
          : undefined,
      }));

      // Log for debugging
      console.log(`[SettingsIntegrations] Loaded ${integrations.value.length} GitHub integration(s)`);
    } catch (err: any) {
      console.error("Failed to load GitHub integrations:", err);
      error.value = err.message || "Failed to load connected accounts. Please try again.";
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
    
    // Log for debugging - ensure this matches what callback handler uses
    console.log("[GitHub OAuth] Redirect URI being sent to GitHub:", redirectUri);
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

  const formatScopes = (scope: string): string => {
    if (!scope) return "None";
    // Common GitHub scopes with readable names
    const scopeMap: Record<string, string> = {
      repo: "Repository access",
      "read:user": "Read user info",
      "admin:repo_hook": "Manage webhooks",
      "read:org": "Read organization",
      "write:org": "Write organization",
    };
    
    const scopes = scope.split(" ").filter((s) => s.trim());
    return scopes
      .map((s) => scopeMap[s] || s)
      .join(", ");
  };

  const handleAvatarError = (event: Event) => {
    // Fallback to a default GitHub icon if avatar fails to load
    const target = event.target as HTMLImageElement;
    if (target) {
      target.src = "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%23666'%3E%3Cpath d='M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z'/%3E%3C/svg%3E";
    }
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

</script>
