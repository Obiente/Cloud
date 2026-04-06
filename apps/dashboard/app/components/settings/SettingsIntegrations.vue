<template>
  <OuiStack gap="lg">
      <!-- Success Message -->
      <OuiAlert v-if="successMessage && !error" variant="success">
        <OuiText size="sm">{{ successMessage }}</OuiText>
      </OuiAlert>

      <!-- Loading State -->
      <OuiCard v-if="isLoading" variant="outline">
        <OuiCardBody>
          <OuiFlex align="center" gap="sm">
            <div class="h-4 w-4 border-2 border-primary border-t-transparent rounded-full animate-spin" />
            <OuiText size="sm" color="tertiary">Loading connected accounts...</OuiText>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Error State -->
      <OuiAlert v-else-if="error && !isLoading" variant="danger">
        <OuiFlex align="center" justify="between">
          <OuiText size="sm">{{ error }}</OuiText>
          <OuiButton size="xs" variant="ghost" @click="loadIntegrations">Retry</OuiButton>
        </OuiFlex>
      </OuiAlert>

      <!-- Connected Accounts -->
      <template v-else-if="integrations.length > 0">
        <OuiCard v-for="integration in integrations" :key="integration.id" variant="outline">
          <OuiCardBody>
            <OuiFlex justify="between" align="center" gap="md">
              <OuiFlex align="center" gap="sm" class="flex-1 min-w-0">
                <img
                  :src="`https://avatars.githubusercontent.com/${integration.username}`"
                  :alt="integration.username"
                  class="h-8 w-8 rounded-full border border-default shrink-0"
                  @error="handleAvatarError"
                />
                <OuiStack gap="none" class="min-w-0">
                  <OuiFlex align="center" gap="sm" wrap="wrap">
                    <OuiText size="sm" weight="medium">@{{ integration.username }}</OuiText>
                    <OuiBadge :variant="integration.isUser ? 'primary' : 'secondary'" size="xs">
                      {{ integration.isUser ? 'Personal' : 'Organization' }}
                    </OuiBadge>
                    <OuiBadge v-if="!integration.isUser && integration.organizationName" variant="secondary" size="xs">
                      {{ integration.organizationName }}
                    </OuiBadge>
                  </OuiFlex>
                  <OuiText size="xs" color="tertiary">
                    Connected <OuiRelativeTime v-if="integration.connectedAt" :value="new Date((integration.connectedAt.seconds || 0) * 1000)" :style="'short'" />
                    <template v-if="integration.scope"> · {{ formatScopes(integration.scope) }}</template>
                  </OuiText>
                </OuiStack>
              </OuiFlex>
              <OuiButton
                @click="disconnectIntegration(integration)"
                :disabled="isDisconnecting"
                variant="ghost"
                size="xs"
                color="danger"
              >
                Disconnect
              </OuiButton>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </template>

      <!-- Empty State -->
      <OuiCard v-else variant="outline">
        <OuiCardBody>
          <OuiStack gap="sm" align="center" class="py-6">
            <Icon name="uil:github" class="h-10 w-10 text-text-tertiary" />
            <OuiStack gap="xs" align="center">
              <OuiText size="sm" weight="medium">No connected accounts</OuiText>
              <OuiText size="xs" color="tertiary" class="text-center max-w-sm">
                Connect your GitHub account to enable repository imports and deployments.
              </OuiText>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Connect Account -->
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiText size="sm" weight="semibold">Connect GitHub</OuiText>

            <OuiStack gap="sm">
              <OuiRadioGroup
                v-model="connectionType"
                :options="[
                  { label: 'Personal Account', value: 'user' },
                  { label: 'Organization', value: 'organization' },
                ]"
              />

              <OuiStack v-if="connectionType === 'organization'" gap="xs">
                <OuiSelect
                  v-model="selectedOrgId"
                  :items="organizationOptions"
                  placeholder="Select an organization"
                />
              </OuiStack>

              <OuiAlert
                v-if="connectionType === 'user' && integrations.some((i) => i.isUser)"
                variant="warning"
              >
                <OuiText size="xs">
                  Personal account already connected ({{ integrations.find((i) => i.isUser)?.username }}). This will update the existing connection.
                </OuiText>
              </OuiAlert>

              <OuiAlert
                v-if="connectionType === 'organization' && selectedOrgId && integrations.some((i) => !i.isUser && i.organizationId === selectedOrgId)"
                variant="warning"
              >
                <OuiText size="xs">
                  This organization already has a connection ({{ integrations.find((i) => !i.isUser && i.organizationId === selectedOrgId)?.username }}). This will update it.
                </OuiText>
              </OuiAlert>

              <OuiButton
                @click="connectGitHub"
                :disabled="isConnecting || (connectionType === 'organization' && !selectedOrgId)"
                variant="solid"
                size="sm"
              >
                {{ isConnecting ? "Connecting..." : "Connect GitHub" }}
              </OuiButton>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- GitLab placeholder -->
      <OuiCard variant="outline" class="opacity-50">
        <OuiCardBody>
          <OuiFlex align="center" gap="sm">
            <Icon name="uil:gitlab" class="h-5 w-5" />
            <OuiText size="sm" weight="medium">GitLab</OuiText>
            <OuiBadge variant="secondary" size="xs">Coming soon</OuiBadge>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
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
const successMessage = ref("");
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
  let handledCallback = false;

  const provider = route.query.provider;
  if (provider === "github") {
    // Handle OAuth callback results (success/error)
    const success = route.query.success;
    const errorParam = route.query.error;
    const username = route.query.username;
    const orgId = route.query.orgId;

    if (success === "true" && username) {
      handledCallback = true;
      // Successfully connected - reload integrations with a small delay to ensure backend processed it
      await new Promise((resolve) => setTimeout(resolve, 500));
      error.value = "";
      // Show success message briefly
      successMessage.value = orgId
        ? `Successfully connected GitHub organization to ${orgId}`
        : `Successfully connected GitHub account: ${username}`;
      await loadIntegrations({ preserveFeedback: true });
      // Clean up URL
      router.replace({ query: { tab: route.query.tab } });
    } else if (errorParam) {
      handledCallback = true;
      // Handle errors from callback
      const errorMsg = String(errorParam);
      isConnecting.value = false; // Reset on error
      successMessage.value = "";
      if (errorMsg === "missing_code") {
        error.value =
          "Authorization code missing. Please try connecting again.";
      } else if (errorMsg === "configuration_error") {
        error.value =
          "GitHub integration is not properly configured. Please contact your administrator.";
      } else if (errorMsg === "token_exchange_failed") {
        error.value = "Failed to complete GitHub connection. Please try again.";
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
  if (!handledCallback) {
    await loadIntegrations();
  }
});

const loadIntegrations = async (
  options: { preserveFeedback?: boolean } = {}
) => {
  isLoading.value = true;
  if (!options.preserveFeedback) {
    error.value = "";
    successMessage.value = "";
  }

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
  } catch (err: unknown) {
    console.error("Failed to load GitHub integrations:", err);
    error.value =
      (err as Error).message || "Failed to load connected accounts. Please try again.";
    integrations.value = [];
  } finally {
    isLoading.value = false;
  }
};

const connectGitHub = () => {
  error.value = "";
  successMessage.value = "";
  isConnecting.value = true;

  const config = useRuntimeConfig();
  const githubClientId = config.public.githubClientId;

  if (!githubClientId || githubClientId === "") {
    error.value =
      "GitHub Client ID not configured. Please set NUXT_PUBLIC_GITHUB_CLIENT_ID in your .env file.";
    isConnecting.value = false;
    return;
  }

  if (connectionType.value === "organization" && !selectedOrgId.value) {
    error.value = "Select an organization before connecting GitHub.";
    isConnecting.value = false;
    return;
  }

  const connectUrl = new URL("/api/github/connect", window.location.origin);
  connectUrl.searchParams.set("type", connectionType.value);
  if (connectionType.value === "organization" && selectedOrgId.value) {
    connectUrl.searchParams.set("orgId", selectedOrgId.value);
  }

  window.location.href = connectUrl.toString();
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
  return scopes.map((s) => scopeMap[s] || s).join(", ");
};

const handleAvatarError = (event: Event) => {
  // Fallback to a default GitHub icon if avatar fails to load
  const target = event.target as HTMLImageElement;
  if (target) {
    target.src =
      "data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%23666'%3E%3Cpath d='M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z'/%3E%3C/svg%3E";
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
  successMessage.value = "";

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
  } catch (err: unknown) {
    console.error("Failed to disconnect GitHub:", err);
    error.value =
      (err as Error).message || "Failed to disconnect GitHub account. Please try again.";
  } finally {
    isDisconnecting.value = false;
  }
};
</script>
