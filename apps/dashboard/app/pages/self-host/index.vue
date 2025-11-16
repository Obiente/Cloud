<template>
  <OuiStack gap="2xl">
    <!-- Page Header -->
    <OuiStack gap="md">
      <OuiStack gap="xs">
        <OuiFlex align="center" gap="md">
          <OuiBox class="w-12 h-12 bg-primary/10 rounded-xl flex items-center justify-center">
            <ServerIcon class="h-6 w-6 text-primary" />
          </OuiBox>
          <OuiStack gap="none">
            <OuiText tag="h1" size="3xl" weight="extrabold">Self-Hosted DNS Delegation</OuiText>
            <OuiText size="sm" color="muted">
              Manage DNS delegation for your self-hosted Obiente Cloud instance
            </OuiText>
          </OuiStack>
        </OuiFlex>
        <OuiText size="sm" color="muted" class="mt-2">
          Use the main <OuiCode code="my.obiente.cloud" language="text" padding="xs" inline /> DNS 
          service without exposing DNS port 53 locally.
        </OuiText>
      </OuiStack>
    </OuiStack>

    <!-- Organization Selector -->
    <OuiCard>
      <OuiCardHeader>
        <OuiStack gap="xs">
          <OuiText tag="h2" size="lg" weight="semibold">Organization</OuiText>
          <OuiText size="sm" color="muted">
            DNS delegation is managed per organization. Select the organization that owns your self-hosted instance.
          </OuiText>
        </OuiStack>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiSelect
          v-model="selectedOrg"
          placeholder="Choose organization"
          :items="organizationSelectItems"
        />
      </OuiCardBody>
    </OuiCard>

    <template v-if="!selectedOrg">
      <OuiCard variant="outline">
        <OuiCardBody>
          <OuiStack gap="md" align="center" class="py-12">
            <OuiBox class="w-16 h-16 bg-surface-subtle rounded-full flex items-center justify-center">
              <ServerIcon class="h-8 w-8 text-muted" />
            </OuiBox>
            <OuiStack gap="xs" align="center">
              <OuiText size="xl" weight="semibold">Select an Organization</OuiText>
              <OuiText size="sm" color="muted" align="center" class="max-w-md">
                Please select an organization to manage DNS delegation.
              </OuiText>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </template>

    <template v-else>
      <!-- Subscription Status -->
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="start">
            <OuiStack gap="xs">
              <OuiFlex align="center" gap="sm">
                <CreditCardIcon class="h-5 w-5 text-muted" />
                <OuiText tag="h2" size="xl" weight="bold">Subscription</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="muted">
                DNS delegation requires an active subscription ($2/month)
              </OuiText>
            </OuiStack>
            <OuiFlex gap="sm">
              <OuiButton 
                variant="solid" 
                size="sm"
                @click="subscribeToDNSDelegation"
                :disabled="dnsDelegationSubscribing || hasActiveSubscription"
                v-if="!hasActiveSubscription && currentUserIsOwner"
              >
                <CreditCardIcon class="h-4 w-4 mr-2" />
                Subscribe ($2/month)
              </OuiButton>
              <OuiButton 
                color="danger"
                variant="outline" 
                size="sm"
                @click="cancelSubscriptionDialogOpen = true"
                :disabled="cancelingSubscription || !currentUserIsOwner"
                v-if="hasActiveSubscription && !subscriptionCanceling"
              >
                <TrashIcon class="h-4 w-4 mr-2" />
                Cancel
              </OuiButton>
            </OuiFlex>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <OuiBox 
              v-if="subscriptionCanceling" 
              p="md" 
              rounded="md" 
              class="bg-warning/10 border border-warning/30"
            >
              <OuiFlex align="start" gap="sm">
                <ExclamationTriangleIcon class="h-5 w-5 text-warning shrink-0 mt-0.5" />
                <OuiText size="sm">
                  Your subscription will be canceled at the end of the current billing period.
                  <span v-if="subscriptionCanceledAt">
                    Access will end on {{ formatDate(subscriptionCanceledAt) }}.
                  </span>
                </OuiText>
              </OuiFlex>
            </OuiBox>
            
            <OuiGrid columns="2" gap="md">
              <OuiCard variant="outline">
                <OuiCardBody>
                  <OuiStack gap="sm">
                    <OuiText size="sm" color="muted" weight="medium">Status</OuiText>
                    <OuiBadge 
                      v-if="hasActiveSubscription" 
                      variant="success"
                      size="md"
                    >
                      Active
                    </OuiBadge>
                    <OuiBadge 
                      v-else 
                      variant="secondary"
                      size="md"
                    >
                      Not Subscribed
                    </OuiBadge>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
              
              <OuiCard variant="outline">
                <OuiCardBody>
                  <OuiStack gap="sm">
                    <OuiText size="sm" color="muted" weight="medium">Price</OuiText>
                    <OuiText size="lg" weight="bold">$2.00/month</OuiText>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiGrid>

            <OuiText size="sm" color="muted" v-if="!hasActiveSubscription">
              Subscribe to enable DNS delegation. Once subscribed, you'll be able to create API keys 
              for pushing DNS records from your self-hosted instances.
            </OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- API Key Management -->
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="start">
            <OuiStack gap="xs">
              <OuiFlex align="center" gap="sm">
                <KeyIcon class="h-5 w-5 text-muted" />
                <OuiText tag="h2" size="xl" weight="bold">API Key</OuiText>
              </OuiFlex>
              <OuiText size="sm" color="muted">
                API key for pushing DNS records from your self-hosted API
              </OuiText>
            </OuiStack>
            <OuiFlex gap="sm">
              <OuiButton 
                variant="ghost" 
                size="sm"
                @click="refreshAPIKey"
                :disabled="refreshingAPIKey"
              >
                <ArrowPathIcon class="h-4 w-4 mr-2" :class="{ 'animate-spin': refreshingAPIKey }" />
                Refresh
              </OuiButton>
              <OuiButton 
                variant="solid" 
                size="sm"
                @click="createAPIKeyDialogOpen = true"
                :disabled="!hasActiveSubscription || creatingAPIKey"
                v-if="!currentAPIKey"
              >
                <KeyIcon class="h-4 w-4 mr-2" />
                Create API Key
              </OuiButton>
              <OuiButton 
                color="danger"
                variant="outline"
                size="sm"
                @click="revokeAPIKey"
                :disabled="revokingAPIKey || !currentAPIKey"
                v-if="currentAPIKey"
              >
                <TrashIcon class="h-4 w-4 mr-2" />
                Revoke
              </OuiButton>
            </OuiFlex>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="md">
            <template v-if="loadingAPIKey">
              <OuiStack gap="sm" align="center" class="py-8">
                <ArrowPathIcon class="h-8 w-8 text-muted animate-spin" />
                <OuiText size="sm" color="muted">Loading API key...</OuiText>
              </OuiStack>
            </template>
            
            <template v-else-if="currentAPIKey && currentAPIKey !== '***KEY_EXISTS***'">
              <OuiCard variant="outline" class="border-success/30 bg-success/5">
                <OuiCardBody>
                  <OuiStack gap="md">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" weight="medium">Your API Key</OuiText>
                      <OuiBadge variant="success" size="sm">Active</OuiBadge>
                    </OuiFlex>
                    <OuiText size="sm" color="muted" v-if="apiKeyDescription">
                      Description: {{ apiKeyDescription }}
                    </OuiText>
                    <OuiCode 
                      :code="currentAPIKey"
                      language="text"
                      padding="md"
                      copyable
                    />
                    <OuiBox 
                      p="md" 
                      rounded="md" 
                      class="bg-warning/10 border border-warning/30"
                    >
                      <OuiFlex align="start" gap="sm">
                        <ExclamationTriangleIcon class="h-5 w-5 text-warning shrink-0 mt-0.5" />
                        <OuiText size="sm">
                          <strong>Important:</strong> Save this API key securely. It will not be shown again after you leave this page.
                        </OuiText>
                      </OuiFlex>
                    </OuiBox>
                    <OuiText size="xs" color="muted">
                      Created: {{ apiKeyCreatedAt ? formatDate(apiKeyCreatedAt) : 'Unknown' }}
                    </OuiText>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </template>
            
            <template v-else-if="currentAPIKey === '***KEY_EXISTS***'">
              <OuiCard variant="outline">
                <OuiCardBody>
                  <OuiStack gap="md">
                    <OuiFlex justify="between" align="center">
                      <OuiText size="sm" weight="medium">API Key Status</OuiText>
                      <OuiBadge variant="success" size="sm">Active</OuiBadge>
                    </OuiFlex>
                    <OuiText size="sm" color="muted" v-if="apiKeyDescription">
                      Description: {{ apiKeyDescription }}
                    </OuiText>
                    <OuiText size="sm" color="muted" v-else>
                      Your organization has an active API key.
                    </OuiText>
                    <OuiText size="xs" color="muted">
                      Created: {{ apiKeyCreatedAt ? formatDate(apiKeyCreatedAt) : 'Unknown' }}
                    </OuiText>
                    <OuiBox 
                      p="md" 
                      rounded="md" 
                      class="bg-info/10 border border-info/30"
                    >
                      <OuiFlex align="start" gap="sm">
                        <InformationCircleIcon class="h-5 w-5 text-info shrink-0 mt-0.5" />
                        <OuiText size="sm">
                          The API key value is not displayed for security reasons. If you need a new key, 
                          revoke the existing one and create a new one.
                        </OuiText>
                      </OuiFlex>
                    </OuiBox>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </template>
            
            <template v-else>
              <OuiCard variant="outline">
                <OuiCardBody>
                  <OuiStack gap="md" align="center" class="py-8">
                    <OuiBox class="w-16 h-16 bg-surface-subtle rounded-full flex items-center justify-center">
                      <KeyIcon class="h-8 w-8 text-muted" />
                    </OuiBox>
                    <OuiStack gap="xs" align="center">
                      <OuiText size="lg" weight="semibold">No API Key</OuiText>
                      <OuiText size="sm" color="muted" align="center">
                        <template v-if="!hasActiveSubscription">
                          Subscribe to DNS delegation to create an API key.
                        </template>
                        <template v-else>
                          Create an API key to start pushing DNS records from your self-hosted instance.
                        </template>
                      </OuiText>
                    </OuiStack>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </template>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Configuration Guide -->
      <OuiCard>
        <OuiCardHeader>
          <OuiStack gap="xs">
            <OuiFlex align="center" gap="sm">
              <ServerIcon class="h-5 w-5 text-muted" />
              <OuiText tag="h2" size="xl" weight="bold">Configuration</OuiText>
            </OuiFlex>
            <OuiText size="sm" color="muted">
              Use this API key to configure your self-hosted Obiente Cloud API to push DNS records.
            </OuiText>
          </OuiStack>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiStack gap="md">
              <OuiText size="sm" weight="semibold">Environment Variables</OuiText>
              <OuiCode 
                :code="envVarsCode"
                language="bash"
                padding="md"
                copyable
              />
            </OuiStack>

            <OuiStack gap="md">
              <OuiText size="sm" weight="semibold">Docker Compose Example</OuiText>
              <OuiCode 
                :code="dockerComposeCode"
                language="yaml"
                padding="md"
                copyable
              />
            </OuiStack>

            <OuiStack gap="sm">
              <OuiText size="sm" weight="semibold">Testing DNS Resolution</OuiText>
              <OuiText size="xs" color="muted">
                After configuring your self-hosted API, test DNS resolution:
              </OuiText>
              <OuiCode 
                code="dig deploy-123.my.obiente.cloud"
                language="bash"
                padding="sm"
                copyable
              />
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </template>

    <!-- Create API Key Dialog -->
    <OuiDialog v-model:open="createAPIKeyDialogOpen" title="Create API Key">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Create a new API key for DNS delegation. This will replace any existing API key.
          </OuiText>
          <OuiBox 
            v-if="error" 
            p="md" 
            rounded="md" 
            class="bg-danger/10 border border-danger/30"
          >
            <OuiFlex align="start" gap="sm">
              <ExclamationTriangleIcon class="h-5 w-5 text-danger shrink-0 mt-0.5" />
              <OuiText size="sm" color="danger">{{ error }}</OuiText>
            </OuiFlex>
          </OuiBox>
        </OuiStack>
        
        <OuiStack gap="md">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Description</OuiText>
            <OuiInput
              v-model="apiKeyDescription"
              placeholder="e.g., Self-hosted instance at example.com"
            />
          </OuiStack>
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Source API URL (Optional)</OuiText>
            <OuiInput
              v-model="apiKeySourceAPI"
              placeholder="https://selfhosted-api.example.com"
              type="url"
            />
          </OuiStack>
        </OuiStack>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="createAPIKeyDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="solid" 
            @click="createAPIKey"
            :disabled="creatingAPIKey || !apiKeyDescription"
          >
            {{ creatingAPIKey ? "Creating..." : "Create API Key" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>

    <!-- Cancel Subscription Dialog -->
    <OuiDialog v-model:open="cancelSubscriptionDialogOpen" title="Cancel Subscription">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Are you sure you want to cancel your DNS delegation subscription? Your subscription will remain active until the end of the current billing period, then it will be canceled.
          </OuiText>
          <OuiBox 
            v-if="error" 
            p="md" 
            rounded="md" 
            class="bg-danger/10 border border-danger/30"
          >
            <OuiFlex align="start" gap="sm">
              <ExclamationTriangleIcon class="h-5 w-5 text-danger shrink-0 mt-0.5" />
              <OuiText size="sm" color="danger">{{ error }}</OuiText>
            </OuiFlex>
          </OuiBox>
        </OuiStack>
        
        <OuiBox 
          p="md" 
          rounded="md" 
          class="bg-warning/10 border border-warning/30"
        >
          <OuiFlex align="start" gap="sm">
            <ExclamationTriangleIcon class="h-5 w-5 text-warning shrink-0 mt-0.5" />
            <OuiText size="sm">
              <strong>Important:</strong> After cancellation, your API keys will be revoked and DNS delegation will stop working. You can resubscribe at any time.
            </OuiText>
          </OuiFlex>
        </OuiBox>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="cancelSubscriptionDialogOpen = false">
            Keep Subscription
          </OuiButton>
          <OuiButton 
            color="danger"
            variant="solid" 
            @click="cancelSubscription"
            :disabled="cancelingSubscription"
          >
            {{ cancelingSubscription ? "Canceling..." : "Cancel Subscription" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
definePageMeta({ layout: "self-host", middleware: "auth" });

import {
  OrganizationService,
  BillingService,
  SuperadminService,
} from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { onMounted, watch, computed, ref } from "vue";
import { useOrganizationLabels } from "~/composables/useOrganizationLabels";
import {
  ServerIcon,
  CreditCardIcon,
  ArrowPathIcon,
  KeyIcon,
  TrashIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
} from "@heroicons/vue/24/outline";

const error = ref("");
const auth = useAuth();
const orgClient = useConnectClient(OrganizationService);
const billingClient = useConnectClient(BillingService);
const superadminClient = useConnectClient(SuperadminService);
const { toast } = useToast();

const organizations = computed(() => auth.organizations || []);
const { organizationSelectItems } = useOrganizationLabels(organizations);
const selectedOrg = computed({
  get: () => auth.currentOrganizationId,
  set: (id: string) => {
    if (id) auth.switchOrganization(id);
  },
});

// Load organizations if not already loaded
if (!organizations.value.length && auth.isAuthenticated) {
  const res = await orgClient.listOrganizations({ onlyMine: true });
  auth.setOrganizations(res.organizations || []);
}

// Get current user identifiers (similar to billing page)
const currentUserIdentifiers = computed(() => {
  const identifiers = new Set<string>();
  const sessionUser: any = auth.user || null;
  if (!sessionUser) {
    return identifiers;
  }
  [sessionUser.id, sessionUser.sub, sessionUser.userId].forEach((id) => {
    if (id) {
      identifiers.add(String(id));
    }
  });
  return identifiers;
});

// Get current member record for permission checks
const { data: membersData } = await useClientFetch(
  () =>
    selectedOrg.value
      ? `org-members-${selectedOrg.value}`
      : "org-members-none",
  async () => {
    if (!selectedOrg.value) return [];
    try {
      const res = await orgClient.listMembers({
        organizationId: selectedOrg.value,
      });
      return res.members || [];
    } catch {
      return [];
    }
  },
  { watch: [selectedOrg], server: false }
);

const members = computed(() => membersData.value || []);

const currentMemberRecord = computed(
  () =>
    members.value.find((member) => {
      const memberUserId = member.user?.id;
      if (!memberUserId) return false;
      return currentUserIdentifiers.value.has(memberUserId);
    }) || null
);

const currentUserIsOwner = computed(() => {
  return currentMemberRecord.value?.role === "owner";
});

// Billing account
const { data: billingAccountData, refresh: refreshBillingAccount } = await useClientFetch(
  () => selectedOrg.value ? `billing-${selectedOrg.value}` : "billing-none",
  async () => {
    if (!selectedOrg.value) return null;
    try {
      const res = await billingClient.getBillingAccount({
        organizationId: selectedOrg.value,
      });
      return res.account;
    } catch (err) {
      console.error("Failed to fetch billing account:", err);
      return null;
    }
  },
  { watch: [selectedOrg], server: false }
);

const billingAccount = computed(() => billingAccountData.value);

// DNS Delegation subscription
const dnsDelegationSubscribing = ref(false);
const hasActiveSubscription = ref(false);
const subscriptionStatusLoading = ref(false);
const subscriptionCanceling = ref(false);
const subscriptionCanceledAt = ref<string | null>(null);
const cancelSubscriptionDialogOpen = ref(false);
const cancelingSubscription = ref(false);

// Load subscription status
async function loadSubscriptionStatus() {
  if (!selectedOrg.value) return;
  
  subscriptionStatusLoading.value = true;
  try {
    const response = await billingClient.getDNSDelegationSubscriptionStatus({
      organizationId: selectedOrg.value,
    });
    
    hasActiveSubscription.value = response.hasActiveSubscription || false;
    subscriptionCanceling.value = response.cancelAtPeriodEnd || false;
    
    if (response.currentPeriodEnd) {
      const seconds = typeof response.currentPeriodEnd.seconds === "bigint" 
        ? Number(response.currentPeriodEnd.seconds) 
        : (response.currentPeriodEnd.seconds || 0);
      const nanos = response.currentPeriodEnd.nanos || 0;
      const millis = seconds * 1000 + Math.floor(nanos / 1_000_000);
      subscriptionCanceledAt.value = new Date(millis).toISOString();
    }
  } catch (err) {
    console.error("Failed to load subscription status:", err);
    hasActiveSubscription.value = false;
    subscriptionCanceling.value = false;
  } finally {
    subscriptionStatusLoading.value = false;
  }
}

async function subscribeToDNSDelegation() {
  if (!selectedOrg.value) return;
  
  dnsDelegationSubscribing.value = true;
  error.value = "";
  try {
    const config = useRuntimeConfig();
    const baseUrl = config.public.requestHost || window.location.origin;
    const successUrl = `${baseUrl}/subscription-callback?status=success`;
    const cancelUrl = `${baseUrl}/subscription-callback?status=canceled`;
    
    const response = await billingClient.createDNSDelegationSubscriptionCheckout({
      organizationId: selectedOrg.value,
      successUrl,
      cancelUrl,
    });
    
    if (response.checkoutUrl) {
      // Open Stripe Checkout in a popup window
      const width = 500;
      const height = 700;
      const top = (window.top?.outerHeight ?? 0) / 2 + (window.top?.screenY ?? 0) - height / 2;
      const left = (window.top?.outerWidth ?? 0) / 2 + (window.top?.screenX ?? 0) - width / 2;
      
      const popup = window.open(
        response.checkoutUrl,
        "stripe-checkout",
        `width=${width}, height=${height}, top=${top}, left=${left}, toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=yes, resizable=yes`
      );
      
      if (!popup) {
        throw new Error("Popup blocked. Please allow popups for this site.");
      }
      
      // Listen for message from popup callback page
      const messageListener = (event: MessageEvent) => {
        if (event.origin !== window.location.origin) return;
        
        if (event.data?.type === "stripe-checkout-complete") {
          const status = event.data.status;
          window.removeEventListener("message", messageListener);
          
          dnsDelegationSubscribing.value = false;
          
          if (status === "success") {
            // Poll subscription status until it's active
            let pollCount = 0;
            const maxPolls = 20; // Increased to 20 seconds for webhook processing
            const pollInterval = setInterval(async () => {
              pollCount++;
              await loadSubscriptionStatus();
              
              // If subscription is now active, stop polling
              if (hasActiveSubscription.value || pollCount >= maxPolls) {
                clearInterval(pollInterval);
                if (hasActiveSubscription.value) {
                  toast.success("Subscription successful! You can now create an API key.");
                  // Also reload API key status
                  await loadAPIKey();
                } else {
                  toast.warning("Subscription processing. Please refresh the page in a moment.");
                }
              }
            }, 1000); // Poll every second
          } else {
            toast.info("Subscription canceled");
          }
        }
      };
      
      window.addEventListener("message", messageListener);
      
      // Fallback: check if popup closes without message (user closes manually)
      const checkPopup = setInterval(() => {
        if (popup.closed) {
          clearInterval(checkPopup);
          // Only handle if we haven't already received a message
          if (dnsDelegationSubscribing.value) {
            dnsDelegationSubscribing.value = false;
            // Poll once to check status
            loadSubscriptionStatus().then(() => {
              if (hasActiveSubscription.value) {
                toast.success("Subscription successful!");
                loadAPIKey();
              }
            });
          }
        }
      }, 500);
    } else {
      throw new Error("No checkout URL received");
    }
  } catch (err: any) {
    error.value = err.message || "Failed to create subscription checkout";
    toast.error(error.value);
    dnsDelegationSubscribing.value = false;
  }
}

async function openCustomerPortal() {
  if (!selectedOrg.value) return;
  try {
    const response = await billingClient.createPortalSession({
      organizationId: selectedOrg.value,
    });
    if (response.portalUrl) {
      window.location.href = response.portalUrl;
    }
  } catch (err: any) {
    error.value = err.message || "Failed to open customer portal";
    toast.error(error.value);
  }
}

async function cancelSubscription() {
  if (!selectedOrg.value) return;
  
  cancelingSubscription.value = true;
  error.value = "";
  try {
    const response = await billingClient.cancelDNSDelegationSubscription({
      organizationId: selectedOrg.value,
    });
    
    if (response.success) {
      cancelSubscriptionDialogOpen.value = false;
      subscriptionCanceling.value = true;
      if (response.canceledAt) {
        const seconds = typeof response.canceledAt.seconds === "bigint" 
          ? Number(response.canceledAt.seconds) 
          : (response.canceledAt.seconds || 0);
        const nanos = response.canceledAt.nanos || 0;
        const millis = seconds * 1000 + Math.floor(nanos / 1_000_000);
        subscriptionCanceledAt.value = new Date(millis).toISOString();
      }
      toast.success(response.message || "Subscription canceled successfully");
      // Reload subscription status
      await loadSubscriptionStatus();
    } else {
      throw new Error(response.message || "Failed to cancel subscription");
    }
  } catch (err: any) {
    error.value = err.message || "Failed to cancel subscription";
    toast.error(error.value);
  } finally {
    cancelingSubscription.value = false;
  }
}

// API Key management
const loadingAPIKey = ref(false);
const refreshingAPIKey = ref(false);
const creatingAPIKey = ref(false);
const revokingAPIKey = ref(false);
const currentAPIKey = ref<string | null>(null);
const apiKeyCreatedAt = ref<string | null>(null);
const createAPIKeyDialogOpen = ref(false);
const apiKeyDescription = ref("");
const apiKeySourceAPI = ref("");

async function loadAPIKey() {
  if (!selectedOrg.value) return;
  
  loadingAPIKey.value = true;
  error.value = "";
  try {
    // Load subscription status to check if user can create API key
    const statusResponse = await billingClient.getDNSDelegationSubscriptionStatus({
      organizationId: selectedOrg.value,
    });
    
    hasActiveSubscription.value = statusResponse.hasActiveSubscription || false;
    
    // Check if organization has an API key
    if (statusResponse.hasApiKey) {
      currentAPIKey.value = "***KEY_EXISTS***"; // Placeholder - we can't retrieve the actual key
      if (statusResponse.apiKeyCreatedAt) {
        // Convert Timestamp to ISO string
        const seconds = typeof statusResponse.apiKeyCreatedAt.seconds === "bigint" 
          ? Number(statusResponse.apiKeyCreatedAt.seconds) 
          : (statusResponse.apiKeyCreatedAt.seconds || 0);
        const nanos = statusResponse.apiKeyCreatedAt.nanos || 0;
        const millis = seconds * 1000 + Math.floor(nanos / 1_000_000);
        apiKeyCreatedAt.value = new Date(millis).toISOString();
      }
      if (statusResponse.apiKeyDescription) {
        apiKeyDescription.value = statusResponse.apiKeyDescription;
      }
    } else {
      currentAPIKey.value = null;
      apiKeyCreatedAt.value = null;
      apiKeyDescription.value = "";
    }
  } catch (err: any) {
    console.error("Failed to load API key:", err);
    currentAPIKey.value = null;
    hasActiveSubscription.value = false;
  } finally {
    loadingAPIKey.value = false;
  }
}

async function refreshAPIKey() {
  await loadAPIKey();
}

async function createAPIKey() {
  if (!selectedOrg.value || !apiKeyDescription.value) return;
  
  creatingAPIKey.value = true;
  error.value = "";
  try {
    const response = await superadminClient.createDNSDelegationAPIKey({
      description: apiKeyDescription.value,
      sourceApi: apiKeySourceAPI.value || undefined,
      organizationId: selectedOrg.value,
    });
    
    if (response.apiKey) {
      currentAPIKey.value = response.apiKey;
      apiKeyCreatedAt.value = new Date().toISOString();
      // Save description from response before clearing form
      const savedDescription = response.description || apiKeyDescription.value;
      createAPIKeyDialogOpen.value = false;
      // Clear form inputs after successful creation
      apiKeyDescription.value = "";
      apiKeySourceAPI.value = "";
      // Restore description for display (will be overwritten by loadAPIKey anyway)
      apiKeyDescription.value = savedDescription;
      toast.success("API key created successfully. Save it securely!");
      // Reload API key status to ensure it's properly saved (this will populate description from DB)
      await loadAPIKey();
    } else {
      throw new Error("No API key received");
    }
  } catch (err: any) {
    error.value = err.message || "Failed to create API key";
    toast.error(error.value);
  } finally {
    creatingAPIKey.value = false;
  }
}

async function revokeAPIKey() {
  if (!selectedOrg.value) return;
  
  if (!confirm("Are you sure you want to revoke this API key? This will stop DNS delegation from working.")) {
    return;
  }
  
  revokingAPIKey.value = true;
  error.value = "";
  try {
    await superadminClient.revokeDNSDelegationAPIKeyForOrganization({
      organizationId: selectedOrg.value,
    });
    
    currentAPIKey.value = null;
    apiKeyCreatedAt.value = null;
    toast.success("API key revoked successfully");
  } catch (err: any) {
    error.value = err.message || "Failed to revoke API key";
    toast.error(error.value);
  } finally {
    revokingAPIKey.value = false;
  }
}

function formatDate(date: string | Date): string {
  try {
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(date));
  } catch {
    return String(date);
  }
}

// Computed code blocks for display
const envVarsCode = computed(() => {
  return `# Production API URL
DNS_DELEGATION_PRODUCTION_API_URL="https://api.obiente.cloud"

# API key (from above)
DNS_DELEGATION_API_KEY="${currentAPIKey.value || 'your-api-key-here'}"

# Optional: How often to push DNS records (default: 2m)
DNS_DELEGATION_PUSH_INTERVAL="2m"

# Optional: TTL for pushed DNS records (default: 300s = 5 minutes)
DNS_DELEGATION_TTL="300s"`;
});

const dockerComposeCode = computed(() => {
  return `services:
  api:
    environment:
      DNS_DELEGATION_PRODUCTION_API_URL: "https://api.obiente.cloud"
      DNS_DELEGATION_API_KEY: "${currentAPIKey.value || 'your-api-key-here'}"
      DNS_DELEGATION_PUSH_INTERVAL: "2m"
      DNS_DELEGATION_TTL: "300s"`;
});

// Watch for organization changes
watch(selectedOrg, () => {
  if (selectedOrg.value) {
    loadSubscriptionStatus();
    loadAPIKey();
    refreshBillingAccount();
  }
});

// Load on mount
onMounted(() => {
  if (selectedOrg.value) {
    loadSubscriptionStatus();
    loadAPIKey();
    refreshBillingAccount();
  }
});
</script>

