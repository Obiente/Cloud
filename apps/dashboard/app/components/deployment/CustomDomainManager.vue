<template>
  <OuiCard variant="outline">
    <OuiCardBody>
      <OuiStack gap="lg">
        <OuiFlex justify="between" align="center">
          <OuiText as="h3" size="md" weight="semibold"
            >Custom Domains</OuiText
          >
          <OuiButton
            @click="showAddDomain = true"
            size="sm"
            variant="solid"
          >
            Add Domain
          </OuiButton>
        </OuiFlex>

        <OuiText size="xs" color="secondary">
          Add custom domains to your deployment. You'll need to verify ownership via DNS TXT record.
        </OuiText>

        <!-- Domain List -->
        <OuiStack gap="md" v-if="domains.length > 0">
          <div
            v-for="domainInfo in domains"
            :key="domainInfo.domain"
            class="border border-border-default rounded-lg p-4"
          >
            <OuiStack gap="md">
              <OuiFlex justify="between" align="center">
                <OuiFlex align="center" gap="sm">
                  <OuiText size="sm" weight="semibold">{{ domainInfo.domain }}</OuiText>
                  <OuiBadge
                    :variant="getStatusVariant(domainInfo.status)"
                    size="sm"
                  >
                    {{ getStatusLabel(domainInfo.status) }}
                  </OuiBadge>
                </OuiFlex>
                <OuiButton
                  v-if="domainInfo.status === 'pending' || domainInfo.status === 'failed'"
                  variant="ghost"
                  size="xs"
                  @click="verifyDomain(domainInfo.domain)"
                  :disabled="isVerifying"
                >
                  {{ isVerifying ? "Verifying..." : "Verify" }}
                </OuiButton>
              </OuiFlex>

              <!-- Failed Status - Show error banner -->
              <div
                v-if="domainInfo.status === 'failed' && domainInfo.errorMessage"
                class="bg-danger/5 border border-danger/20 rounded-lg p-3 mb-3"
              >
                <OuiFlex align="center" gap="sm">
                  <Icon name="uil:times-circle" class="h-4 w-4 text-danger" />
                  <OuiText size="xs" color="danger">
                    {{ domainInfo.errorMessage }}
                  </OuiText>
                </OuiFlex>
              </div>

              <!-- Verification Instructions - Show for pending or failed status -->
              <div
                v-if="(domainInfo.status === 'pending' || domainInfo.status === 'failed') && domainInfo.verificationToken"
                class="bg-background-muted rounded-lg p-4 border border-border-default"
              >
                <OuiStack gap="sm">
                  <OuiText size="sm" weight="semibold">Verify Domain Ownership</OuiText>
                  <OuiText size="xs" color="secondary">
                    Add the following TXT record to your DNS provider to verify ownership:
                  </OuiText>
                  
                  <div class="bg-background-default rounded p-3 font-mono text-xs border border-border-default">
                    <div class="mb-2">
                      <span class="text-secondary">Type:</span> TXT
                    </div>
                    <div class="mb-2 flex items-center justify-between gap-2">
                      <div class="flex-1">
                        <span class="text-secondary">Name:</span> {{ domainInfo.txtRecordName }}
                      </div>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click="copyToClipboard(domainInfo.txtRecordName || '')"
                        title="Copy"
                      >
                        <Icon name="uil:copy" class="h-3 w-3" />
                      </OuiButton>
                    </div>
                    <div class="flex items-center justify-between gap-2">
                      <div class="flex-1 break-all">
                        <span class="text-secondary">Value:</span> {{ domainInfo.txtRecordValue }}
                      </div>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click="copyToClipboard(domainInfo.txtRecordValue || '')"
                        title="Copy"
                      >
                        <Icon name="uil:copy" class="h-3 w-3" />
                      </OuiButton>
                    </div>
                  </div>

                  <OuiText size="xs" color="secondary">
                    After adding the record, wait a few minutes for DNS propagation, then click "Verify" above.
                  </OuiText>
                </OuiStack>
              </div>

              <!-- DNS Configuration Instructions -->
              <div
                v-if="domainInfo.status === 'verified'"
                class="bg-success/5 border border-success/20 rounded-lg p-4"
              >
                <OuiStack gap="sm">
                  <OuiFlex align="center" gap="sm">
                    <Icon name="uil:check-circle" class="h-4 w-4 text-success" />
                    <OuiText size="sm" weight="semibold" color="success">
                      Domain Verified
                    </OuiText>
                  </OuiFlex>
                  <OuiText size="xs" color="secondary">
                    Configure a CNAME record in your DNS provider:
                  </OuiText>
                  
                  <div class="bg-background-default rounded p-3 font-mono text-xs border border-border-default">
                    <div class="mb-2">
                      <span class="text-secondary">Type:</span> CNAME
                    </div>
                    <div class="mb-2 flex items-center justify-between gap-2">
                      <div class="flex-1">
                        <span class="text-secondary">Name:</span> {{ domainInfo.domain }}
                      </div>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click="copyToClipboard(domainInfo.domain)"
                        title="Copy"
                      >
                        <Icon name="uil:copy" class="h-3 w-3" />
                      </OuiButton>
                    </div>
                    <div class="flex items-center justify-between gap-2">
                      <div class="flex-1 break-all">
                        <span class="text-secondary">Value:</span> {{ defaultDomain }}
                      </div>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        @click="copyToClipboard(defaultDomain)"
                        title="Copy"
                      >
                        <Icon name="uil:copy" class="h-3 w-3" />
                      </OuiButton>
                    </div>
                  </div>

                  <OuiText size="xs" color="secondary">
                    SSL certificates will be issued automatically via Let's Encrypt once DNS is configured.
                  </OuiText>
                </OuiStack>
              </div>

              <!-- Remove Domain Button -->
              <OuiButton
                variant="ghost"
                color="danger"
                size="xs"
                @click="removeDomain(domainInfo.domain)"
                :disabled="isRemoving"
              >
                Remove Domain
              </OuiButton>
            </OuiStack>
          </div>
        </OuiStack>

        <!-- Empty State -->
        <div v-else class="text-center py-8">
          <Icon name="uil:globe" class="h-12 w-12 text-secondary mx-auto mb-4" />
          <OuiText size="sm" color="secondary">
            No custom domains configured. Add one to get started.
          </OuiText>
        </div>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>

  <!-- Add Domain Dialog -->
  <OuiDialog v-model:open="showAddDomain" title="Add Custom Domain">
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="showAddDomain = false">
          Cancel
        </OuiButton>
        <OuiButton
          variant="solid"
          @click="addDomain"
          :disabled="!newDomain || isAdding"
        >
          {{ isAdding ? "Adding..." : "Add Domain" }}
        </OuiButton>
      </OuiFlex>
    </template>
    <OuiStack gap="md">
      <OuiInput
        v-model="newDomain"
        label="Domain"
        placeholder="example.com"
        helper-text="Enter your domain name (e.g., example.com or www.example.com)"
      />
    </OuiStack>
  </OuiDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from "vue";
import type { Deployment } from "@obiente/proto";
import { DeploymentService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useOrganizationsStore } from "~/stores/organizations";
import { useDialog } from "~/composables/useDialog";

const { showAlert, showConfirm } = useDialog();

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    // Show a subtle toast-like notification instead of alert
    const toast = document.createElement("div");
    toast.className = "fixed top-4 right-4 bg-success text-white px-4 py-2 rounded-lg shadow-lg z-50";
    toast.textContent = "Copied to clipboard!";
    document.body.appendChild(toast);
    setTimeout(() => {
      document.body.removeChild(toast);
    }, 2000);
  } catch (err) {
    console.error("Failed to copy:", err);
    await showAlert({
      title: "Copy Failed",
      message: "Failed to copy to clipboard. Please copy manually.",
    });
  }
};

interface Props {
  deployment: Deployment;
}

interface DomainInfo {
  domain: string;
  status: "pending" | "verified" | "failed" | "expired";
  verificationToken?: string;
  txtRecordName?: string;
  txtRecordValue?: string;
  errorMessage?: string;
}

const props = defineProps<Props>();
const orgsStore = useOrganizationsStore();
const client = useConnectClient(DeploymentService);
const organizationId = computed(() => orgsStore.currentOrgId || "");

const showAddDomain = ref(false);
const newDomain = ref("");
const isAdding = ref(false);
const isVerifying = ref(false);
const isRemoving = ref(false);
const domains = ref<DomainInfo[]>([]);

const defaultDomain = computed(() => props.deployment.domain || "");

// Parse domains from deployment.customDomains
const parseDomains = () => {
  const parsed: DomainInfo[] = [];
  
  if (props.deployment.customDomains && props.deployment.customDomains.length > 0) {
    for (const entry of props.deployment.customDomains) {
      const parts = entry.split(":");
      const domain = parts[0];
      if (!domain) continue; // Skip invalid entries
      
      let status: DomainInfo["status"] = "pending";

      if (parts.length >= 4 && parts[1] === "token" && parts[2] && parts[3]) {
        status = parts[3] as DomainInfo["status"];
        parsed.push({
          domain,
          status,
          verificationToken: parts[2],
          txtRecordName: `_obiente-verification.${domain}`,
          txtRecordValue: `obiente-verification=${parts[2]}`,
        });
      } else if (parts.length >= 2 && parts[1] === "verified") {
        parsed.push({
          domain,
          status: "verified",
        });
      } else {
        parsed.push({
          domain,
          status: "pending",
        });
      }
    }
  }
  return parsed;
};

const getStatusVariant = (status: string) => {
  switch (status) {
    case "verified":
      return "success";
    case "failed":
      return "danger";
    case "expired":
      return "warning";
    default:
      return "secondary";
  }
};

const getStatusLabel = (status: string) => {
  switch (status) {
    case "verified":
      return "Verified";
    case "failed":
      return "Failed";
    case "expired":
      return "Expired";
    default:
      return "Pending Verification";
  }
};

const addDomain = async () => {
  if (!newDomain.value.trim()) return;

  isAdding.value = true;
  try {
    const currentDomains = props.deployment.customDomains || [];
    const updatedDomains = [...currentDomains, newDomain.value.trim()];

    await client.updateDeployment({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      customDomains: updatedDomains,
    });

    await refreshNuxtData(`deployment-${props.deployment.id}`);
    // Refresh domain info
    await refreshDomainInfo(newDomain.value.trim());
    newDomain.value = "";
    showAddDomain.value = false;
  } catch (err: any) {
    await showAlert({
      title: "Failed to Add Domain",
      message: err.message || "Failed to add domain. Please try again.",
    });
  } finally {
    isAdding.value = false;
  }
};

const refreshDomainInfo = async (domain: string) => {
  try {
    const response = await client.getDomainVerificationToken({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      domain,
    });

    const domainInfo: DomainInfo = {
      domain: response.domain,
      status: response.status as DomainInfo["status"],
      verificationToken: response.token,
      txtRecordName: response.txtRecordName,
      txtRecordValue: response.txtRecordValue,
    };

    const index = domains.value.findIndex((d) => d.domain === domain);
    if (index >= 0) {
      domains.value[index] = domainInfo;
    } else {
      domains.value.push(domainInfo);
    }
  } catch (err) {
    console.error("Failed to refresh domain info:", err);
  }
};

const verifyDomain = async (domain: string) => {
  isVerifying.value = true;
  try {
    const response = await client.verifyDomainOwnership({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      domain,
    });

    if (response.verified) {
      // Refresh deployment data so routing component sees the updated domains
      await refreshNuxtData(`deployment-${props.deployment.id}`);
      await refreshDomainInfo(domain);
      // Success is shown inline in the UI (verified badge and DNS instructions)
    } else {
      const domainInfo = domains.value.find((d) => d.domain === domain);
      if (domainInfo) {
        // Update status but preserve verification token info so user can see instructions
        domainInfo.status = "failed";
        domainInfo.errorMessage = response.message || "Verification failed";
        // Don't clear verificationToken, txtRecordName, txtRecordValue - user needs them to retry
      }
      // Error is shown inline in the UI, no need for alert dialog
    }
  } catch (err: any) {
    const domainInfo = domains.value.find((d) => d.domain === domain);
    if (domainInfo) {
      domainInfo.status = "failed";
      domainInfo.errorMessage = err.message || "Failed to verify domain. Please try again.";
    }
  } finally {
    isVerifying.value = false;
  }
};

const removeDomain = async (domain: string) => {
  const confirmed = await showConfirm({
    title: "Remove Domain",
    message: `Are you sure you want to remove ${domain}?`,
    confirmLabel: "Remove",
    cancelLabel: "Cancel",
    variant: "danger",
  });

  if (!confirmed) return;

  isRemoving.value = true;
  try {
    const currentDomains = props.deployment.customDomains || [];
    // Filter out the domain (case-insensitive comparison)
    const updatedDomains = currentDomains.filter((d) => {
      const parts = d.split(":");
      const domainName = parts[0] || "";
      return domainName.toLowerCase() !== domain.toLowerCase();
    });

    await client.updateDeployment({
      organizationId: organizationId.value,
      deploymentId: props.deployment.id,
      customDomains: updatedDomains,
    });

    // Refresh deployment data - the watch will automatically update domains.value
    await refreshNuxtData(`deployment-${props.deployment.id}`);
    
    await showAlert({
      title: "Domain Removed",
      message: `${domain} has been removed successfully.`,
    });
  } catch (err: any) {
    console.error("Failed to remove domain:", err);
    await showAlert({
      title: "Failed to Remove Domain",
      message: err.message || "Failed to remove domain. Please try again.",
    });
  } finally {
    isRemoving.value = false;
  }
};

// Initialize domains on mount and when deployment changes
watch(
  () => props.deployment.customDomains,
  (newDomains) => {
    domains.value = parseDomains();
    // Only fetch verification info for domains that don't have tokens yet (pending plain entries)
    domains.value.forEach(async (domainInfo) => {
      // Only call API if we don't have a token yet AND status is pending
      // If we already have a token from parseDomains, don't regenerate it
      if (domainInfo.status === "pending" && !domainInfo.verificationToken) {
        await refreshDomainInfo(domainInfo.domain);
      }
    });
  },
  { immediate: true, deep: true }
);

onMounted(() => {
  domains.value = parseDomains();
  // Only fetch verification info for domains that don't have tokens yet
  domains.value.forEach(async (domainInfo) => {
    if (domainInfo.status === "pending" && !domainInfo.verificationToken) {
      await refreshDomainInfo(domainInfo.domain);
    }
  });
});
</script>

