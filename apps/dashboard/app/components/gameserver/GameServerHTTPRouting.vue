<template>
  <OuiStack gap="lg">
    <!-- Custom Domain Management -->
    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiStack gap="lg">
          <OuiFlex justify="between" align="center">
            <OuiText as="h3" size="md" weight="semibold">Custom Domains</OuiText>
            <OuiButton @click="showAddDomain = true" size="sm" variant="solid">
              Add Domain
            </OuiButton>
          </OuiFlex>

          <OuiText size="xs" color="secondary">
            Add custom domains to your game server's HTTP routes. You'll need to verify
            ownership via DNS TXT record before using the domain in a routing rule.
          </OuiText>

          <!-- Domain List -->
          <OuiStack gap="md" v-if="verifiedDomains.length > 0 || pendingDomains.length > 0">
            <div
              v-for="domainInfo in [...pendingDomains, ...verifiedDomains]"
              :key="domainInfo.domain"
              class="border border-border-default rounded-lg p-4"
            >
              <OuiStack gap="md">
                <OuiFlex justify="between" align="center">
                  <OuiFlex align="center" gap="sm">
                    <OuiText size="sm" weight="semibold">{{ domainInfo.domain }}</OuiText>
                    <OuiBadge :variant="getStatusVariant(domainInfo.status)" size="sm">
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

                <!-- Verification Instructions - Show for pending or failed status -->
                <div
                  v-if="(domainInfo.status === 'pending' || domainInfo.status === 'failed') && domainInfo.txtRecordName"
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

                <!-- Verified - DNS CNAME instructions -->
                <div
                  v-if="domainInfo.status === 'verified'"
                  class="bg-success/5 border border-success/20 rounded-lg p-4"
                >
                  <OuiStack gap="sm">
                    <OuiFlex align="center" gap="sm">
                      <Icon name="uil:check-circle" class="h-4 w-4 text-success" />
                      <OuiText size="sm" weight="semibold" color="success">Domain Verified</OuiText>
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
                        <OuiButton variant="ghost" size="xs" @click="copyToClipboard(domainInfo.domain)" title="Copy">
                          <Icon name="uil:copy" class="h-3 w-3" />
                        </OuiButton>
                      </div>
                      <div class="flex items-center justify-between gap-2">
                        <div class="flex-1 break-all">
                          <span class="text-secondary">Value:</span> {{ defaultDomain }}
                        </div>
                        <OuiButton variant="ghost" size="xs" @click="copyToClipboard(defaultDomain)" title="Copy">
                          <Icon name="uil:copy" class="h-3 w-3" />
                        </OuiButton>
                      </div>
                    </div>
                    <OuiText size="xs" color="secondary">
                      SSL certificates will be issued automatically via Let's Encrypt once DNS is configured.
                    </OuiText>
                  </OuiStack>
                </div>

                <!-- Remove button (only for non-default domains) -->
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

    <!-- Routing Rules -->
    <OuiCard variant="outline">
      <OuiCardBody>
        <OuiStack gap="md">
          <OuiFlex justify="between" align="center">
            <OuiStack gap="none">
              <OuiText as="h3" size="md" weight="semibold">HTTP Routing Rules</OuiText>
              <OuiText size="sm" color="secondary">
                Route HTTP traffic to specific ports on your game server. Requires the game server
                to have an HTTP-capable service running on the configured port.
              </OuiText>
            </OuiStack>
            <OuiButton size="sm" @click="addRule">
              <PlusIcon class="h-4 w-4 mr-2" />
              Add Rule
            </OuiButton>
          </OuiFlex>

          <OuiFlex v-if="isLoadingRoutes" justify="center" class="py-8">
            <OuiText color="secondary">Loading routing rules...</OuiText>
          </OuiFlex>

          <OuiFlex
            v-else-if="!editingRule && routes.length === 0"
            direction="col"
            align="center"
            justify="center"
            class="py-12"
          >
            <OuiStack gap="md" align="center">
              <OuiText size="sm" color="secondary">
                No HTTP routing rules configured. Add a rule to expose HTTP services.
              </OuiText>
              <OuiButton size="sm" @click="addRule">Add First Rule</OuiButton>
            </OuiStack>
          </OuiFlex>

          <OuiStack v-else gap="md">
            <!-- Existing routes -->
            <OuiCard
              v-for="route in routes"
              :key="route.id"
              variant="outline"
              class="border-default"
            >
              <OuiCardBody>
                <OuiFlex justify="between" align="start">
                  <OuiStack gap="xs">
                    <OuiFlex align="center" gap="sm" wrap="wrap">
                      <OuiText size="sm" weight="semibold">{{ route.domain }}</OuiText>
                      <OuiBadge variant="secondary" size="xs">
                        {{ route.protocol?.toUpperCase() || "HTTP" }}
                      </OuiBadge>
                      <OuiBadge v-if="route.sslEnabled" variant="success" size="xs">SSL</OuiBadge>
                    </OuiFlex>
                    <OuiText size="xs" color="secondary">
                      Port {{ route.targetPort }}
                      <span v-if="route.pathPrefix"> • {{ route.pathPrefix }}</span>
                    </OuiText>
                  </OuiStack>
                  <OuiFlex gap="sm">
                    <OuiButton variant="ghost" size="sm" @click="startEditRule(route)">
                      <PencilIcon class="h-4 w-4" />
                    </OuiButton>
                    <OuiButton variant="ghost" size="sm" color="danger" @click="deleteRoute(route.id)">
                      <TrashIcon class="h-4 w-4" />
                    </OuiButton>
                  </OuiFlex>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Inline edit/add form -->
            <OuiCard v-if="editingRule" variant="outline" class="border-primary/30 bg-primary/5">
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText size="sm" weight="semibold">
                    {{ editingRule.id ? "Edit Rule" : "New Rule" }}
                  </OuiText>

                  <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
                    <OuiSelect
                      v-model="editingRule.domain"
                      :items="domainOptions"
                      label="Domain"
                      placeholder="Select domain"
                    />
                    <OuiInput
                      v-model="editingRule.targetPortStr"
                      type="number"
                      label="Target Port"
                      placeholder="80"
                      @update:model-value="(val) => { editingRule!.targetPort = parseInt(val) || 80; editingRule!.targetPortStr = val; }"
                    />
                  </OuiGrid>

                  <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
                    <OuiInput
                      v-model="editingRule.pathPrefix"
                      label="Path Prefix (optional)"
                      placeholder="/api"
                    />
                    <OuiSelect
                      v-model="editingRule.protocol"
                      :items="protocolOptions"
                      label="Protocol"
                      @update:model-value="(val) => {
                        editingRule!.protocol = val;
                        if (val === 'http') editingRule!.sslEnabled = false;
                        else if (val === 'https') editingRule!.sslEnabled = true;
                      }"
                    />
                  </OuiGrid>

                  <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
                    <OuiSwitch
                      v-model="editingRule.sslEnabled"
                      label="SSL Enabled"
                      :disabled="editingRule.protocol === 'http'"
                    />
                    <OuiSelect
                      v-if="editingRule.sslEnabled"
                      v-model="editingRule.sslCertResolver"
                      :items="sslResolverOptions"
                      label="SSL Certificate Resolver"
                    />
                  </OuiGrid>

                  <OuiFlex justify="end" gap="sm">
                    <OuiButton variant="ghost" size="sm" @click="cancelEdit">Cancel</OuiButton>
                    <OuiButton size="sm" :loading="isSavingRule" @click="saveRule">
                      {{ editingRule.id ? "Update Rule" : "Add Rule" }}
                    </OuiButton>
                  </OuiFlex>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>

  <!-- Add Domain Dialog -->
  <OuiDialog v-model:open="showAddDomain" title="Add Custom Domain">
    <template #footer>
      <OuiFlex justify="end" gap="sm">
        <OuiButton variant="ghost" @click="showAddDomain = false">Cancel</OuiButton>
        <OuiButton
          variant="solid"
          @click="addDomain"
          :disabled="!newDomain.trim() || isAdding"
          :loading="isAdding"
        >
          Add Domain
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
import { ref, computed, onMounted } from "vue";
import { PlusIcon, TrashIcon, PencilIcon } from "@heroicons/vue/24/outline";
import { useConnectClient } from "~/lib/connect-client";
import { GameServerService } from "@obiente/proto";
import type { GameServerHTTPRoute } from "@obiente/proto";
import { useOrganizationsStore } from "~/stores/organizations";
import { useDialog } from "~/composables/useDialog";
import { useToast } from "~/composables/useToast";

interface Props {
  gameServerId: string;
  /** The default domain, e.g. gs-1234567890.my.obiente.cloud */
  defaultDomain: string;
}

interface DomainInfo {
  domain: string;
  status: "pending" | "verified" | "failed" | "expired";
  txtRecordName?: string;
  txtRecordValue?: string;
}

interface LocalRule {
  id?: string;
  domain: string;
  pathPrefix: string;
  targetPort: number;
  targetPortStr: string;
  protocol: string;
  sslEnabled: boolean;
  sslCertResolver: string;
}

const props = defineProps<Props>();

const client = useConnectClient(GameServerService);
const orgsStore = useOrganizationsStore();
const organizationId = computed(() => orgsStore.currentOrgId || "");
const { showConfirm, showAlert } = useDialog();
const { toast } = useToast();

// Domain management state
const showAddDomain = ref(false);
const newDomain = ref("");
const isAdding = ref(false);
const isVerifying = ref(false);
const isRemoving = ref(false);
const pendingDomains = ref<DomainInfo[]>([]);
const verifiedDomains = ref<DomainInfo[]>([]);

// Route management state
const routes = ref<GameServerHTTPRoute[]>([]);
const isLoadingRoutes = ref(false);
const editingRule = ref<LocalRule | null>(null);
const isSavingRule = ref(false);

const protocolOptions = [
  { label: "HTTP", value: "http" },
  { label: "HTTPS", value: "https" },
  { label: "gRPC", value: "grpc" },
];

const sslResolverOptions = [
  { label: "Let's Encrypt", value: "letsencrypt" },
  { label: "Internal (Handled by App)", value: "internal" },
];

const domainOptions = computed(() => {
  const options: Array<{ label: string; value: string }> = [
    { label: `${props.defaultDomain} (default)`, value: props.defaultDomain },
  ];
  for (const d of verifiedDomains.value) {
    options.push({ label: d.domain, value: d.domain });
  }
  return options;
});

// ── helpers ──────────────────────────────────────────────────────────────────

const getStatusVariant = (status: string) => {
  switch (status) {
    case "verified": return "success";
    case "failed": return "danger";
    case "expired": return "warning";
    default: return "secondary";
  }
};

const getStatusLabel = (status: string) => {
  switch (status) {
    case "verified": return "Verified";
    case "failed": return "Failed";
    case "expired": return "Expired";
    default: return "Pending Verification";
  }
};

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("Copied to clipboard");
  } catch {
    await showAlert({ title: "Copy Failed", message: "Please copy manually." });
  }
};

// ── domain management ─────────────────────────────────────────────────────────

const loadRoutes = async () => {
  isLoadingRoutes.value = true;
  try {
    const res = await client.getGameServerHTTPRoutes({
      gameServerId: props.gameServerId,
      organizationId: organizationId.value,
    });
    routes.value = res.routes || [];

    // Build the domain lists from the existing routes' domains (excluding default)
    const seenDomains = new Set<string>();
    const pending: DomainInfo[] = [];
    const verified: DomainInfo[] = [];

    for (const route of routes.value) {
      if (!route.domain || route.domain === props.defaultDomain) continue;
      if (seenDomains.has(route.domain)) continue;
      seenDomains.add(route.domain);
      // We don't know verification status from routes alone; fetch token to get status
    }

    // Refresh verification status for any custom domains in routes
    for (const domain of seenDomains) {
      try {
        const tokenRes = await client.getGameServerDomainVerificationToken({
          gameServerId: props.gameServerId,
          organizationId: organizationId.value,
          domain,
        });
        const info: DomainInfo = {
          domain: tokenRes.domain,
          status: tokenRes.status as DomainInfo["status"],
          txtRecordName: tokenRes.txtRecordName,
          txtRecordValue: tokenRes.txtRecordValue,
        };
        if (tokenRes.status === "verified") {
          verified.push(info);
        } else {
          pending.push(info);
        }
      } catch {
        pending.push({ domain, status: "pending" });
      }
    }

    pendingDomains.value = pending;
    verifiedDomains.value = verified;
  } catch (err) {
    console.error("Failed to load game server HTTP routes:", err);
  } finally {
    isLoadingRoutes.value = false;
  }
};

const addDomain = async () => {
  const domain = newDomain.value.trim();
  if (!domain) return;

  isAdding.value = true;
  try {
    const res = await client.getGameServerDomainVerificationToken({
      gameServerId: props.gameServerId,
      organizationId: organizationId.value,
      domain,
    });

    const info: DomainInfo = {
      domain: res.domain,
      status: res.status as DomainInfo["status"],
      txtRecordName: res.txtRecordName,
      txtRecordValue: res.txtRecordValue,
    };

    // Remove existing entry for this domain if any
    pendingDomains.value = pendingDomains.value.filter((d) => d.domain !== domain);
    verifiedDomains.value = verifiedDomains.value.filter((d) => d.domain !== domain);

    if (res.status === "verified") {
      verifiedDomains.value.push(info);
    } else {
      pendingDomains.value.push(info);
    }

    newDomain.value = "";
    showAddDomain.value = false;
    toast.success("Domain added", "Add the TXT record to verify ownership.");
  } catch (err: unknown) {
    await showAlert({
      title: "Failed to add domain",
      message: (err as Error | undefined)?.message || "An error occurred while adding the domain.",
    });
  } finally {
    isAdding.value = false;
  }
};

const verifyDomain = async (domain: string) => {
  isVerifying.value = true;
  try {
    const res = await client.verifyGameServerDomain({
      gameServerId: props.gameServerId,
      organizationId: organizationId.value,
      domain,
    });

    if (res.verified) {
      // Move from pending to verified
      const pendingIdx = pendingDomains.value.findIndex((d) => d.domain === domain);
      if (pendingIdx !== -1) {
        const [entry] = pendingDomains.value.splice(pendingIdx, 1);
        if (entry) {
          verifiedDomains.value.push({ ...entry, status: "verified" });
        }
      }
      toast.success("Domain verified!");
    } else {
      // Update status in pending list
      const entry = pendingDomains.value.find((d) => d.domain === domain);
      if (entry) entry.status = res.status as DomainInfo["status"];
      toast.error("Verification failed", "TXT record not found. Ensure it's added and allow DNS propagation.");
    }
  } catch (err: unknown) {
    toast.error("Verification error", (err as Error | undefined)?.message || "Could not verify domain.");
  } finally {
    isVerifying.value = false;
  }
};

const removeDomain = async (domain: string) => {
  const confirmed = await showConfirm({
    title: "Remove Domain",
    message: `Remove "${domain}"? Any routing rules using this domain will also be deleted.`,
  });
  if (!confirmed) return;

  isRemoving.value = true;
  try {
    // Delete all routes with this domain
    const routesToDelete = routes.value.filter((r) => r.domain === domain);
    for (const route of routesToDelete) {
      await client.deleteGameServerHTTPRoute({
        gameServerId: props.gameServerId,
        organizationId: organizationId.value,
        routeId: route.id,
      });
    }
    pendingDomains.value = pendingDomains.value.filter((d) => d.domain !== domain);
    verifiedDomains.value = verifiedDomains.value.filter((d) => d.domain !== domain);
    routes.value = routes.value.filter((r) => r.domain !== domain);
    toast.success("Domain removed");
  } catch (err: unknown) {
    toast.error("Failed to remove domain", (err as Error | undefined)?.message);
  } finally {
    isRemoving.value = false;
  }
};

// ── routing rules ─────────────────────────────────────────────────────────────

const addRule = () => {
  editingRule.value = {
    domain: props.defaultDomain,
    pathPrefix: "",
    targetPort: 8080,
    targetPortStr: "8080",
    protocol: "http",
    sslEnabled: false,
    sslCertResolver: "letsencrypt",
  };
};

const startEditRule = (route: GameServerHTTPRoute) => {
  editingRule.value = {
    id: route.id,
    domain: route.domain,
    pathPrefix: route.pathPrefix || "",
    targetPort: route.targetPort,
    targetPortStr: String(route.targetPort),
    protocol: route.protocol || "http",
    sslEnabled: route.sslEnabled ?? false,
    sslCertResolver: route.sslCertResolver || "letsencrypt",
  };
};

const cancelEdit = () => {
  editingRule.value = null;
};

const saveRule = async () => {
  const rule = editingRule.value;
  if (!rule) return;

  if (!rule.domain.trim()) {
    await showAlert({ title: "Validation Error", message: "Please select a domain." });
    return;
  }
  if (!rule.targetPort || rule.targetPort < 1 || rule.targetPort > 65535) {
    await showAlert({ title: "Validation Error", message: "Port must be between 1 and 65535." });
    return;
  }

  isSavingRule.value = true;
  try {
    const protocol = rule.protocol || "http";
    const sslEnabled = protocol === "http" ? false : protocol === "https" ? true : rule.sslEnabled;

    const res = await client.upsertGameServerHTTPRoute({
      gameServerId: props.gameServerId,
      organizationId: organizationId.value,
      routeId: rule.id || undefined,
      domain: rule.domain.trim(),
      pathPrefix: rule.pathPrefix || undefined,
      targetPort: rule.targetPort,
      protocol,
      sslEnabled,
      sslCertResolver: sslEnabled ? (rule.sslCertResolver || "letsencrypt") : undefined,
    });

    if (res.route) {
      if (rule.id) {
        const idx = routes.value.findIndex((r) => r.id === rule.id);
        if (idx !== -1) routes.value[idx] = res.route;
      } else {
        routes.value.push(res.route);
      }
    }

    editingRule.value = null;
    toast.success(rule.id ? "Rule updated" : "Rule added", "Server will restart to apply new routing.");
  } catch (err: unknown) {
    await showAlert({
      title: "Failed to save rule",
      message: (err as Error | undefined)?.message || "An error occurred.",
    });
  } finally {
    isSavingRule.value = false;
  }
};

const deleteRoute = async (routeId: string) => {
  const confirmed = await showConfirm({
    title: "Delete Rule",
    message: "Remove this routing rule? The server will restart to apply the change.",
  });
  if (!confirmed) return;

  try {
    await client.deleteGameServerHTTPRoute({
      gameServerId: props.gameServerId,
      organizationId: organizationId.value,
      routeId,
    });
    routes.value = routes.value.filter((r) => r.id !== routeId);
    toast.success("Rule deleted");
  } catch (err: unknown) {
    toast.error("Failed to delete rule", (err as Error | undefined)?.message);
  }
};

// ── lifecycle ─────────────────────────────────────────────────────────────────

onMounted(() => {
  loadRoutes();
});
</script>
