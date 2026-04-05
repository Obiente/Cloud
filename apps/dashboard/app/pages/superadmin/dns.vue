<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">
          {{ hasDelegatedDNS ? "Delegated DNS Management" : "DNS Management" }}
        </OuiText>
        <OuiText color="muted" size="sm">
          <span v-if="hasDelegatedDNS">
            View and manage your delegated DNS records. Your organization is using DNS delegation to push DNS records to the production DNS server.
          </span>
          <span v-else>
            Query DNS records and view DNS configuration for deployments, databases, game
            servers, and delegated DNS.
          </span>
        </OuiText>
      </OuiStack>
      <OuiButton
        variant="ghost"
        size="sm"
        @click="refresh"
        :disabled="isLoading"
      >
        <span class="flex items-center gap-2">
          <ArrowPathIcon
            class="h-4 w-4"
            :class="{ 'animate-spin': isLoading }"
          />
          Refresh
        </span>
      </OuiButton>
    </OuiFlex>

    <!-- Delegated DNS View (for users with delegated DNS) -->
    <template v-if="hasDelegatedDNS">
      <OuiAlert variant="info" size="sm">
        <OuiText size="xs">
          <strong>Delegated DNS Active:</strong> Your organization ({{ delegatedDNSInfo?.organizationId }}) is using DNS delegation. 
          Records are filtered to your organization by default. Clear the organization filter to see all records.
        </OuiText>
      </OuiAlert>

      <!-- Delegated DNS Records List (simplified view) -->
      <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
        <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
          <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
            <OuiStack gap="xs">
              <OuiText tag="h2" size="xl" weight="bold">Your Delegated DNS Records</OuiText>
              <OuiText color="muted" size="sm">
                {{ filteredDelegatedRecords.length }} of {{ delegatedDNSRecords.length }} records
              </OuiText>
            </OuiStack>
            <OuiFlex gap="sm" wrap="wrap">
              <div class="w-72 max-w-full">
                <OuiInput
                  v-model="delegatedRecordsSearch"
                  type="search"
                  placeholder="Search by domain, record type..."
                  clearable
                  size="sm"
                />
              </div>
              <div class="min-w-[160px]">
                <OuiSelect
                  v-model="delegatedRecordsRecordTypeFilter"
                  :items="recordTypeFilterOptions"
                  placeholder="Record Type"
                  clearable
                  size="sm"
                />
              </div>
              <div class="min-w-[160px]">
                <OuiSelect
                  v-model="delegatedRecordsOrgFilter"
                  :items="delegatedOrgFilterOptions"
                  placeholder="Organization"
                  clearable
                  size="sm"
                />
              </div>
            </OuiFlex>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody class="p-0">
          <OuiTable
            :columns="delegatedTableColumns"
            :rows="delegatedTableRows"
            :empty-text="
              delegatedRecordsLoading
                ? 'Loading delegated DNS records…'
                : 'No delegated DNS records found.'
            "
          >
            <template #cell-recordType="{ value }">
              <OuiBadge
                :variant="value === 'SRV' ? 'primary' : 'secondary'"
                size="sm"
              >
                {{ value }}
              </OuiBadge>
            </template>
            <template #cell-domain="{ value }">
              <div class="font-mono text-sm">{{ value }}</div>
            </template>
            <template #cell-records="{ value }">
              <div v-if="value && value.length > 0" class="flex flex-wrap gap-1">
                <span
                  v-for="record in value"
                  :key="record"
                  class="font-mono text-xs px-2 py-0.5 bg-surface-subtle rounded border border-border-muted"
                >
                  {{ record }}
                </span>
              </div>
              <span v-else class="text-text-tertiary text-sm">—</span>
            </template>
            <template #cell-ttl="{ value }">
              <OuiText size="sm">{{ value }}s</OuiText>
            </template>
            <template #cell-expiresAt="{ value }">
              <OuiText size="xs" color="muted">{{ formatDate(value) }}</OuiText>
            </template>
            <template #cell-lastUpdated="{ value }">
              <OuiText size="xs" color="muted">{{ formatDate(value) }}</OuiText>
            </template>
          </OuiTable>
        </OuiCardBody>
      </OuiCard>
    </template>

    <!-- Full DNS Management View (for superadmins or users without delegated DNS) -->
    <template v-else>
    <!-- DNS Query Tool -->
    <OuiCard class="border border-border-muted rounded-xl">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">DNS Query</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-6">
        <OuiStack gap="md">
          <OuiFlex gap="md" wrap="wrap" align="end">
            <div class="flex-1 min-w-[300px]">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Domain</OuiText>
                <OuiInput
                  v-model="queryDomain"
                  placeholder="deploy-123.my.obiente.cloud, db-123e4567e89b12d3.my.obiente.cloud, or _minecraft._tcp.gs-123.my.obiente.cloud"
                  @keyup.enter="queryDNS"
                />
              </OuiStack>
            </div>
            <div class="min-w-[120px]">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Record Type</OuiText>
                <OuiSelect
                  v-model="queryRecordType"
                  :items="recordTypeOptions"
                />
              </OuiStack>
            </div>
            <OuiButton
              @click="queryDNS"
              :disabled="queryLoading || !queryDomain"
            >
              Query
            </OuiButton>
          </OuiFlex>

          <!-- Query Results -->
          <div v-if="queryResult" class="mt-4">
            <OuiCard
              :class="
                queryResult.error
                  ? 'bg-danger/5 border-danger/20'
                  : 'bg-success/5 border-success/20'
              "
              class="border rounded-lg p-4"
            >
              <OuiStack gap="sm">
                <OuiFlex align="center" justify="between">
                  <OuiText size="sm" weight="medium">
                    {{ queryResult.domain }} ({{ queryResult.recordType }})
                  </OuiText>
                  <OuiText size="xs" color="muted">
                    TTL: {{ queryResult.ttl }}s
                  </OuiText>
                </OuiFlex>
                <div v-if="queryResult.error" class="text-danger">
                  <OuiText size="sm">{{ queryResult.error }}</OuiText>
                </div>
                <div
                  v-else-if="
                    queryResult.records && queryResult.records.length > 0
                  "
                >
                  <OuiStack gap="xs">
                    <OuiText size="xs" color="muted" weight="medium"
                      >Records:</OuiText
                    >
                    <div
                      v-for="(record, idx) in queryResult.records"
                      :key="idx"
                      class="font-mono text-sm"
                    >
                      {{ record }}
                    </div>
                  </OuiStack>
                </div>
                <div v-else>
                  <OuiText size="sm" color="muted">No records found</OuiText>
                </div>
              </OuiStack>
            </OuiCard>
          </div>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- DNS Configuration -->
    <OuiCard class="border border-border-muted rounded-xl">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">DNS Configuration</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-6">
        <div v-if="dnsConfigLoading" class="text-center py-8">
          <OuiText color="muted">Loading configuration...</OuiText>
        </div>
        <div v-else-if="dnsConfig">
          <OuiStack gap="lg">
            <!-- Traefik IPs by Region -->
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold"
                >Traefik IPs by Region</OuiText
              >
              <div v-if="traefikIPsByRegion.length === 0" class="text-muted">
                <OuiText size="sm">No regions configured</OuiText>
              </div>
              <div v-else class="space-y-3">
                <div
                  v-for="region in traefikIPsByRegion"
                  :key="region.region"
                  class="border border-border-muted rounded-lg p-4 bg-surface-subtle"
                >
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="medium">{{
                      region.region || "default"
                    }}</OuiText>
                    <div class="flex flex-wrap gap-2">
                      <span
                        v-for="ip in region.ips"
                        :key="ip"
                        class="font-mono text-sm px-2 py-1 bg-surface-raised rounded border border-border-muted"
                      >
                        {{ ip }}
                      </span>
                    </div>
                  </OuiStack>
                </div>
              </div>
            </OuiStack>

            <!-- DNS Server Info -->
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">DNS Server Info</OuiText>
              <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
                <div>
                  <OuiText
                    size="xs"
                    color="muted"
                    transform="uppercase"
                    class="tracking-wide"
                    >DNS Port</OuiText
                  >
                  <OuiText size="sm" weight="medium" class="font-mono">{{
                    dnsConfig.dnsPort || "53"
                  }}</OuiText>
                </div>
                <div>
                  <OuiText
                    size="xs"
                    color="muted"
                    transform="uppercase"
                    class="tracking-wide"
                    >Cache TTL</OuiText
                  >
                  <OuiText size="sm" weight="medium"
                    >{{ dnsConfig.cacheTtlSeconds }} seconds</OuiText
                  >
                </div>
              </OuiGrid>
            </OuiStack>

            <!-- DNS Server IPs -->
            <OuiStack
              gap="md"
              v-if="dnsConfig.dnsServerIps && dnsConfig.dnsServerIps.length > 0"
            >
              <OuiText size="lg" weight="semibold">DNS Server IPs</OuiText>
              <div class="flex flex-wrap gap-2">
                <span
                  v-for="ip in dnsConfig.dnsServerIps"
                  :key="ip"
                  class="font-mono text-sm px-2 py-1 bg-surface-raised rounded border border-border-muted"
                >
                  {{ ip }}
                </span>
              </div>
            </OuiStack>
          </OuiStack>
        </div>
        <div v-else class="text-center py-8">
          <OuiText color="muted">Failed to load DNS configuration</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>

    <!-- DNS Delegation API Key Management -->
    <OuiCard class="border border-border-muted rounded-xl">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiFlex justify="between" align="center">
          <OuiText tag="h2" size="xl" weight="bold"
            >DNS Delegation API Keys</OuiText
          >
          <OuiButton
            variant="solid"
            size="sm"
            @click="createAPIKeyDialogOpen = true"
          >
            <KeyIcon class="h-4 w-4 mr-2" />
            Create API Key
          </OuiButton>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-6">
        <OuiStack gap="md">
          <OuiText size="sm" color="muted">
            Create API keys for DNS delegation. These keys allow self-hosted
            Obiente Cloud instances to push DNS records to the production DNS
            server. The Obiente Cloud Team can create keys for any organization without
            requiring a subscription.
          </OuiText>

          <OuiAlert variant="info" size="sm">
            <OuiText size="xs">
              <strong>Note:</strong> Regular users can create API keys via the
              Self-Hosted DNS page if they have an active subscription.
            </OuiText>
          </OuiAlert>

          <OuiStack gap="sm">
            <OuiText size="sm" weight="medium">API Keys</OuiText>
            <div v-if="apiKeysLoading" class="text-center py-4">
              <OuiText color="muted" size="sm">Loading API keys...</OuiText>
            </div>
            <div v-else-if="apiKeys.length === 0" class="text-center py-4">
              <OuiText color="muted" size="sm">No API keys found. Create one to get started.</OuiText>
            </div>
            <div v-else>
              <OuiTable
                :columns="apiKeyColumns"
                :rows="apiKeyRows"
                :empty-text="'No API keys'"
              >
                <template #cell-is_active="{ value }">
                  <OuiBadge :variant="value ? 'success' : 'danger'" size="sm">
                    {{ value ? 'Active' : 'Revoked' }}
                  </OuiBadge>
                </template>
                <template #cell-description="{ value }">
                  <OuiText size="sm">{{ value || '—' }}</OuiText>
                </template>
                <template #cell-organization_id="{ value }">
                  <code class="text-xs">{{ value || '—' }}</code>
                </template>
                <template #cell-created_at="{ value }">
                  <OuiText size="xs" color="muted">{{ formatDate(value) }}</OuiText>
                </template>
                <template #cell-revoked_at="{ value }">
                  <OuiText size="xs" color="muted" v-if="value">{{ formatDate(value) }}</OuiText>
                  <span v-else class="text-text-tertiary text-xs">—</span>
                </template>
                <template #cell-actions="{ row }">
                  <OuiButton
                    v-if="row.is_active"
                    variant="ghost"
                    size="xs"
                    color="danger"
                    @click="revokeAPIKeyByID(row.id)"
                    :disabled="revokingAPIKey === row.id"
                  >
                    {{ revokingAPIKey === row.id ? 'Revoking...' : 'Revoke' }}
                  </OuiButton>
                  <span v-else class="text-text-tertiary text-xs">—</span>
                </template>
              </OuiTable>
            </div>
          </OuiStack>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- DNS Records List -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
          <OuiStack gap="xs">
            <OuiText tag="h2" size="xl" weight="bold">DNS Records</OuiText>
            <OuiText color="muted" size="sm">
              {{ filteredRecords.length }} of {{ dnsRecords.length }} records
            </OuiText>
          </OuiStack>
          <OuiFlex gap="sm" wrap="wrap">
            <div class="w-72 max-w-full">
              <OuiInput
                v-model="recordsSearch"
                type="search"
                placeholder="Search by domain, resource name, resource ID, or organization ID..."
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="recordsResourceTypeFilter"
                :items="resourceTypeFilterOptions"
                placeholder="Resource Type"
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="recordsRecordTypeFilter"
                :items="recordTypeFilterOptions"
                placeholder="Record Type"
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="recordsResourceFilter"
                :items="resourceFilterOptions"
                placeholder="Resource"
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="recordsOrgFilter"
                :items="orgFilterOptions"
                placeholder="Organization"
                clearable
                size="sm"
              />
            </div>
          </OuiFlex>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="tableColumns"
          :rows="tableRows"
          :empty-text="
            recordsLoading
              ? 'Loading DNS records…'
              : 'No DNS records match your filters.'
          "
        >
          <template #cell-recordType="{ value }">
            <OuiBadge
              :variant="value === 'SRV' ? 'primary' : 'secondary'"
              size="sm"
            >
              {{ value }}
            </OuiBadge>
          </template>
          <template #cell-domain="{ value }">
            <div class="font-mono text-sm">{{ value }}</div>
          </template>
          <template #cell-resourceType="{ value }">
            <OuiBadge size="sm" variant="outline">
              {{ value }}
            </OuiBadge>
          </template>
          <template #cell-resource="{ row }">
            <div>
              <div class="font-medium text-text-primary">
                {{ row.resourceName || row.resourceId || row.domain }}
              </div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">
                {{ row.resourceId || "—" }}
              </div>
            </div>
          </template>
          <template #cell-ips="{ value }">
            <div v-if="value && value.length > 0" class="flex flex-wrap gap-1">
              <span
                v-for="ip in value"
                :key="ip"
                class="font-mono text-xs px-2 py-0.5 bg-surface-subtle rounded border border-border-muted"
              >
                {{ ip }}
              </span>
            </div>
            <span v-else class="text-text-tertiary text-sm">—</span>
          </template>
          <template #cell-srvTarget="{ value, row }">
            <div v-if="value" class="flex flex-wrap gap-1">
              <span
                class="font-mono text-xs px-2 py-0.5 bg-surface-subtle rounded border border-border-muted"
              >
                {{ value }}:{{ row.port }}
              </span>
            </div>
            <span v-else class="text-text-tertiary text-sm">—</span>
          </template>
          <template #cell-region="{ value }">
            <span v-if="value" class="text-text-secondary uppercase text-xs">{{
              value
            }}</span>
            <span v-else class="text-text-tertiary text-sm">—</span>
          </template>
          <template #cell-status="{ value }">
            <OuiBadge :variant="getStatusBadgeVariant(value)">
              <span
                class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
                :class="getStatusDotClass(value)"
              />
              <OuiText
                as="span"
                size="xs"
                weight="semibold"
                transform="uppercase"
                class="text-[11px]"
              >
                {{ getStatusLabel(value) }}
              </OuiText>
            </OuiBadge>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Delegated DNS Records List (only for superadmins without delegated DNS) -->
    <OuiCard v-if="!hasDelegatedDNS" class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
          <OuiStack gap="xs">
            <OuiText tag="h2" size="xl" weight="bold">Delegated DNS Records</OuiText>
            <OuiText color="muted" size="sm">
              {{ filteredDelegatedRecords.length }} of {{ delegatedDNSRecords.length }} records
            </OuiText>
          </OuiStack>
          <OuiFlex gap="sm" wrap="wrap">
            <div class="w-72 max-w-full">
              <OuiInput
                v-model="delegatedRecordsSearch"
                type="search"
                placeholder="Search by domain, organization ID, API key ID..."
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="delegatedRecordsRecordTypeFilter"
                :items="recordTypeFilterOptions"
                placeholder="Record Type"
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="delegatedRecordsOrgFilter"
                :items="delegatedOrgFilterOptions"
                placeholder="Organization"
                clearable
                size="sm"
              />
            </div>
            <div class="min-w-[160px]">
              <OuiSelect
                v-model="delegatedRecordsAPIKeyFilter"
                :items="delegatedAPIKeyFilterOptions"
                placeholder="API Key"
                clearable
                size="sm"
              />
            </div>
          </OuiFlex>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="delegatedTableColumns"
          :rows="delegatedTableRows"
          :empty-text="
            delegatedRecordsLoading
              ? 'Loading delegated DNS records…'
              : 'No delegated DNS records match your filters.'
          "
        >
          <template #cell-recordType="{ value }">
            <OuiBadge
              :variant="value === 'SRV' ? 'primary' : 'secondary'"
              size="sm"
            >
              {{ value }}
            </OuiBadge>
          </template>
          <template #cell-domain="{ value }">
            <div class="font-mono text-sm">{{ value }}</div>
          </template>
          <template #cell-records="{ value }">
            <div v-if="value && value.length > 0" class="flex flex-wrap gap-1">
              <span
                v-for="record in value"
                :key="record"
                class="font-mono text-xs px-2 py-0.5 bg-surface-subtle rounded border border-border-muted"
              >
                {{ record }}
              </span>
            </div>
            <span v-else class="text-text-tertiary text-sm">—</span>
          </template>
          <template #cell-organizationId="{ value }">
            <code class="text-xs">{{ value || '—' }}</code>
          </template>
          <template #cell-apiKeyId="{ value }">
            <code class="text-xs">{{ value || '—' }}</code>
          </template>
          <template #cell-sourceApi="{ value }">
            <code class="text-xs">{{ value || '—' }}</code>
          </template>
          <template #cell-expiresAt="{ value }">
            <OuiText size="xs" color="muted">{{ formatDate(value) }}</OuiText>
          </template>
          <template #cell-lastUpdated="{ value }">
            <OuiText size="xs" color="muted">{{ formatDate(value) }}</OuiText>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>
    </template>

    <!-- Create API Key Dialog -->
    <OuiDialog
      v-model:open="createAPIKeyDialogOpen"
      title="Create DNS Delegation API Key"
    >
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Create a new API key for DNS delegation. This key can be used by
            self-hosted instances to push DNS records to the production DNS
            server.
          </OuiText>
          <OuiText v-if="apiKeyError" size="sm" color="danger">{{
            apiKeyError
          }}</OuiText>
        </OuiStack>

        <template v-if="createdAPIKey">
          <OuiAlert variant="warning" size="sm">
            <OuiText size="xs">
              <strong>Important:</strong> Save this API key securely. It will
              not be shown again.
            </OuiText>
          </OuiAlert>
          <OuiBox
            p="md"
            class="bg-surface-subtle border border-border-muted rounded"
          >
            <OuiFlex justify="between" align="center" gap="md">
              <code class="text-xs font-mono break-all flex-1">{{
                createdAPIKey
              }}</code>
              <OuiButton variant="ghost" size="xs" @click="copyAPIKey">
                Copy
              </OuiButton>
            </OuiFlex>
          </OuiBox>
        </template>

        <template v-else>
          <OuiStack gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">Description</OuiText>
              <OuiInput
                v-model="apiKeyDescription"
                placeholder="e.g., Self-hosted instance at example.com"
              />
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium"
                >Source API URL (Optional)</OuiText
              >
              <OuiInput
                v-model="apiKeySourceAPI"
                placeholder="https://selfhosted-api.example.com"
                type="url"
              />
            </OuiStack>
          </OuiStack>
        </template>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="createAPIKeyDialogOpen = false">
            {{ createdAPIKey ? "Close" : "Cancel" }}
          </OuiButton>
          <OuiButton
            v-if="!createdAPIKey"
            variant="solid"
            @click="createAPIKey"
            :disabled="creatingAPIKey || !apiKeyDescription"
          >
            {{ creatingAPIKey ? "Creating..." : "Create API Key" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
  definePageMeta({
    middleware: ["auth", "superadmin"],
  });

  import { ArrowPathIcon, KeyIcon } from "@heroicons/vue/24/outline";
  import { computed, ref, onMounted, watch } from "vue";
  import { SuperadminService, type DNSDelegationAPIKeyInfo } from "@obiente/proto";
  import type { Timestamp } from "@bufbuild/protobuf/wkt";
  import { useConnectClient } from "~/lib/connect-client";
  import { useToast } from "~/composables/useToast";

  const config = useConfig();
  const isSelfHosted = computed(() => config.selfHosted.value === true);

  const client = useConnectClient(SuperadminService);
  const { toast } = useToast();

  const isLoading = ref(false);
  const recordsLoading = ref(false);
  const dnsConfigLoading = ref(false);
  const queryLoading = ref(false);

  const queryDomain = ref("");
  const queryRecordType = ref("A");
  const queryResult = ref<any>(null);

  const recordsSearch = ref("");
  const recordsRecordTypeFilter = ref<string | null>(null);
  const recordsResourceTypeFilter = ref<string | null>(null);
  const recordsResourceFilter = ref<string | null>(null);
  const recordsOrgFilter = ref<string | null>(null);

  const dnsRecords = ref<any[]>([]);
  const dnsConfig = ref<any>(null);

  // Delegated DNS Records
  const delegatedRecordsLoading = ref(false);
  const delegatedRecordsSearch = ref("");
  const delegatedRecordsRecordTypeFilter = ref<string | null>(null);
  const delegatedRecordsOrgFilter = ref<string | null>(null);
  const delegatedRecordsAPIKeyFilter = ref<string | null>(null);
  const delegatedDNSRecords = ref<any[]>([]);
  const hasDelegatedDNS = ref(false);
  const delegatedDNSInfo = ref<any>(null);

  // API Key management
  const createAPIKeyDialogOpen = ref(false);
  const creatingAPIKey = ref(false);
  const apiKeyDescription = ref("");
  const apiKeySourceAPI = ref("");
  const apiKeyError = ref("");
  const createdAPIKey = ref<string | null>(null);
  const apiKeys = ref<any[]>([]);
  const apiKeysLoading = ref(false);
  const revokingAPIKey = ref<string | null>(null);

  const recordTypeOptions = [
    { label: "A", value: "A" },
    { label: "SRV", value: "SRV" },
  ];

  const traefikIPsByRegion = computed(() => {
    if (!dnsConfig.value?.traefikIpsByRegion) return [];
    return Object.entries(dnsConfig.value.traefikIpsByRegion).map(
      ([regionKey, traefikIPs]: [string, any]) => ({
        region: traefikIPs?.region || regionKey || "default",
        ips: traefikIPs?.ips || traefikIPs?.Ips || [],
      })
    );
  });

  const recordTypeFilterOptions = [
    { label: "A", value: "A" },
    { label: "SRV", value: "SRV" },
  ];

  function inferResourceType(record: any): string {
    if (record.recordType === "SRV" || record.gameServerId?.startsWith("gs-")) {
      return "Game Server";
    }

    const deploymentLikeID = record.deploymentId || "";
    const label = String(record.domain || "").split(".")[0] || "";

    if (deploymentLikeID.startsWith("db-") || label.startsWith("db-")) {
      return "Database";
    }
    if (deploymentLikeID.startsWith("deploy-") || record.deploymentId) {
      return "Deployment";
    }
    if (label.startsWith("gs-")) {
      return "Game Server";
    }
    return "Custom";
  }

  function normalizeDNSRecord(record: any) {
    const resourceType = inferResourceType(record);
    const resourceId =
      record.gameServerId ||
      record.deploymentId ||
      "";
    const resourceName =
      record.gameServerName ||
      record.deploymentName ||
      "";

    return {
      ...record,
      resourceType,
      resourceId,
      resourceName,
    };
  }

  const normalizedDNSRecords = computed(() => {
    return dnsRecords.value.map((record) => normalizeDNSRecord(record));
  });

  const resourceTypeFilterOptions = computed(() => {
    const resourceTypes = new Set<string>();
    normalizedDNSRecords.value.forEach((record) => {
      if (record.resourceType) resourceTypes.add(record.resourceType);
    });

    return Array.from(resourceTypes)
      .sort()
      .map((resourceType) => ({ label: resourceType, value: resourceType }));
  });

  const resourceFilterOptions = computed(() => {
    const resources = new Map<string, { label: string; value: string }>();

    normalizedDNSRecords.value.forEach((record) => {
      if (!record.resourceId) return;

      const label = record.resourceName
        ? `${record.resourceName} (${record.resourceId})`
        : record.resourceId;

      resources.set(record.resourceId, {
        label,
        value: record.resourceId,
      });
    });

    return Array.from(resources.values()).sort((a, b) =>
      a.label.localeCompare(b.label)
    );
  });

  const orgFilterOptions = computed(() => {
    const orgs = new Set<string>();
    dnsRecords.value.forEach((record) => {
      if (record.organizationId) orgs.add(record.organizationId);
    });
    return Array.from(orgs)
      .sort()
      .map((org) => ({ label: org, value: org }));
  });

  const delegatedOrgFilterOptions = computed(() => {
    const orgs = new Set<string>();
    delegatedDNSRecords.value.forEach((record) => {
      if (record.organizationId) orgs.add(record.organizationId);
    });
    return Array.from(orgs)
      .sort()
      .map((org) => ({ label: org, value: org }));
  });

  const delegatedAPIKeyFilterOptions = computed(() => {
    const keys = new Set<string>();
    delegatedDNSRecords.value.forEach((record) => {
      if (record.apiKeyId) keys.add(record.apiKeyId);
    });
    return Array.from(keys)
      .sort()
      .map((key) => ({ label: key.substring(0, 8) + "...", value: key }));
  });

  const filteredDelegatedRecords = computed(() => {
    const term = delegatedRecordsSearch.value.trim().toLowerCase();
    const recordTypeFilter = delegatedRecordsRecordTypeFilter.value;
    const orgFilter = delegatedRecordsOrgFilter.value;
    const apiKeyFilter = delegatedRecordsAPIKeyFilter.value;

    return delegatedDNSRecords.value.filter((record) => {
      if (recordTypeFilter && record.recordType !== recordTypeFilter)
        return false;
      if (orgFilter && record.organizationId !== orgFilter) return false;
      if (apiKeyFilter && record.apiKeyId !== apiKeyFilter) return false;

      if (!term) return true;

      const searchable = [
        record.domain,
        record.organizationId,
        record.apiKeyId,
        record.sourceApi,
        record.recordType,
        ...(record.records || []),
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();

      return searchable.includes(term);
    });
  });

  const delegatedTableColumns = computed(() => {
    // Simplified columns for delegated DNS users
    if (hasDelegatedDNS.value) {
      return [
        { key: "recordType", label: "Type", defaultWidth: 80, minWidth: 60 },
        { key: "domain", label: "Domain", defaultWidth: 300, minWidth: 200 },
        { key: "records", label: "Records", defaultWidth: 250, minWidth: 200 },
        { key: "ttl", label: "TTL", defaultWidth: 80, minWidth: 60 },
        { key: "expiresAt", label: "Expires At", defaultWidth: 180, minWidth: 150 },
        { key: "lastUpdated", label: "Last Updated", defaultWidth: 180, minWidth: 150 },
      ];
    }
    // Full columns for superadmins
    return [
      { key: "recordType", label: "Type", defaultWidth: 80, minWidth: 60 },
      { key: "domain", label: "Domain", defaultWidth: 300, minWidth: 200 },
      { key: "records", label: "Records", defaultWidth: 250, minWidth: 200 },
      { key: "organizationId", label: "Organization", defaultWidth: 180, minWidth: 150 },
      { key: "apiKeyId", label: "API Key ID", defaultWidth: 150, minWidth: 120 },
      { key: "sourceApi", label: "Source API", defaultWidth: 200, minWidth: 150 },
      { key: "ttl", label: "TTL", defaultWidth: 80, minWidth: 60 },
      { key: "expiresAt", label: "Expires At", defaultWidth: 180, minWidth: 150 },
      { key: "lastUpdated", label: "Last Updated", defaultWidth: 180, minWidth: 150 },
    ];
  });

  const delegatedTableRows = computed(() => {
    return filteredDelegatedRecords.value.map((record) => ({
      ...record,
      organizationId: record.organizationId || "—",
      apiKeyId: record.apiKeyId || "—",
      sourceApi: record.sourceApi || "—",
      recordType: record.recordType || "A",
    }));
  });

  const filteredRecords = computed(() => {
    const term = recordsSearch.value.trim().toLowerCase();
    const recordTypeFilter = recordsRecordTypeFilter.value;
    const resourceTypeFilter = recordsResourceTypeFilter.value;
    const resourceFilter = recordsResourceFilter.value;
    const orgFilter = recordsOrgFilter.value;

    return normalizedDNSRecords.value.filter((record) => {
      if (recordTypeFilter && record.recordType !== recordTypeFilter)
        return false;
      if (resourceTypeFilter && record.resourceType !== resourceTypeFilter)
        return false;
      if (resourceFilter && record.resourceId !== resourceFilter)
        return false;
      if (orgFilter && record.organizationId !== orgFilter) return false;

      if (!term) return true;

      const searchable = [
        record.domain,
        record.resourceId,
        record.resourceName,
        record.resourceType,
        record.deploymentId,
        record.deploymentName,
        record.gameServerId,
        record.gameServerName,
        record.organizationId,
        record.region,
        record.status,
        record.recordType,
        record.target,
        ...(record.ipAddresses || []),
      ]
        .filter(Boolean)
        .join(" ")
        .toLowerCase();

      return searchable.includes(term);
    });
  });

  const tableColumns = computed(() => {
    const baseColumns = [
      { key: "recordType", label: "Type", defaultWidth: 80, minWidth: 60 },
      { key: "resourceType", label: "Resource Type", defaultWidth: 130, minWidth: 110 },
      { key: "resource", label: "Resource", defaultWidth: 220, minWidth: 180 },
      { key: "domain", label: "Domain", defaultWidth: 300, minWidth: 250 },
    ];

    // Add type-specific columns
    const recordTypeFilter = recordsRecordTypeFilter.value;
    if (!recordTypeFilter || recordTypeFilter === "A") {
      baseColumns.push({
        key: "ips",
        label: "IP Addresses",
        defaultWidth: 250,
        minWidth: 200,
      });
    }
    if (!recordTypeFilter || recordTypeFilter === "SRV") {
      baseColumns.push({
        key: "srvTarget",
        label: "Target:Port",
        defaultWidth: 200,
        minWidth: 150,
      });
    }

    // Common columns
    baseColumns.push(
      { key: "region", label: "Region", defaultWidth: 120, minWidth: 100 },
      { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
      {
        key: "organizationId",
        label: "Organization",
        defaultWidth: 180,
        minWidth: 150,
      }
    );

    return baseColumns;
  });

  const tableRows = computed(() => {
    const rows = filteredRecords.value.map((record) => {
      // Ensure status is converted to string if it's a number or string number
      let status: string;
      if (typeof record.status === "number") {
        status = convertStatusNumberToString(record.status);
      } else if (typeof record.status === "string") {
        // Handle string numbers like "3" -> "RUNNING"
        const numStatus = parseInt(record.status, 10);
        if (!isNaN(numStatus)) {
          status = convertStatusNumberToString(numStatus);
        } else {
          // Already a string status name like "RUNNING"
          status = record.status.toUpperCase();
        }
      } else {
        status = "UNKNOWN";
      }
      return {
        ...record,
        status,
        organizationId: record.organizationId || "—",
        recordType: record.recordType || "A",
        resourceType: record.resourceType || "Custom",
        resourceId: record.resourceId || "",
        resourceName: record.resourceName || "",
        // Map ipAddresses to ips for table column
        ips: record.ipAddresses || [],
        // Map target and port to srvTarget for SRV records
        srvTarget: record.recordType === "SRV" ? (record.target || "") : undefined,
        // Ensure region is set
        region: record.region || "",
      };
    });
    return rows;
  });

  function convertStatusNumberToString(status: number): string {
    switch (status) {
      case 1:
        return "CREATED";
      case 2:
        return "BUILDING";
      case 3:
        return "RUNNING";
      case 4:
        return "STOPPED";
      case 5:
        return "FAILED";
      case 6:
        return "DEPLOYING";
      default:
        return "UNKNOWN";
    }
  }

  async function queryDNS() {
    if (!queryDomain.value) return;

    queryLoading.value = true;
    queryResult.value = null;

    try {
      const response = await client.queryDNS({
        domain: queryDomain.value,
        recordType: queryRecordType.value,
      });
      queryResult.value = response;
    } catch (err: unknown) {
      queryResult.value = {
        domain: queryDomain.value,
        recordType: queryRecordType.value,
        error: (err as Error).message || "Failed to query DNS",
        records: [],
        ttl: 0,
      };
    } finally {
      queryLoading.value = false;
    }
  }

  async function loadDNSRecords() {
    recordsLoading.value = true;
    try {
      const response = await client.listDNSRecords({
        recordType: recordsRecordTypeFilter.value || undefined,
      });
      dnsRecords.value = response.records || [];
    } catch (err) {
      console.error("Failed to load DNS records:", err);
      dnsRecords.value = [];
    } finally {
      recordsLoading.value = false;
    }
  }

  async function loadDNSConfig() {
    dnsConfigLoading.value = true;
    try {
      const response = await client.getDNSConfig({});
      dnsConfig.value = response.config;
    } catch (err) {
      console.error("Failed to load DNS config:", err);
      dnsConfig.value = null;
    } finally {
      dnsConfigLoading.value = false;
    }
  }

  async function checkHasDelegatedDNS() {
    try {
      const response = await client.hasDelegatedDNS({});
      hasDelegatedDNS.value = response.hasDelegatedDns || false;
      if (response.hasDelegatedDns) {
        delegatedDNSInfo.value = {
          organizationId: response.organizationId,
          apiKeyId: response.apiKeyId,
        };
        // For superadmins, don't auto-filter by organization - show all records by default
        // This allows them to see records pushed by any organization
        // They can manually filter if needed
        delegatedRecordsOrgFilter.value = null;
      } else {
        // For superadmins without delegated DNS, don't filter by default
        // They can manually filter if needed
        delegatedRecordsOrgFilter.value = null;
      }
    } catch (err) {
      console.error("Failed to check delegated DNS status:", err);
      hasDelegatedDNS.value = false;
      // Don't filter by default for superadmins
      delegatedRecordsOrgFilter.value = null;
    }
  }

  async function loadDelegatedDNSRecords() {
    delegatedRecordsLoading.value = true;
    try {
      const response = await client.listDelegatedDNSRecords({
        organizationId: delegatedRecordsOrgFilter.value || undefined,
        apiKeyId: delegatedRecordsAPIKeyFilter.value || undefined,
        recordType: delegatedRecordsRecordTypeFilter.value || undefined,
      });
      delegatedDNSRecords.value = response.records || [];
    } catch (err) {
      console.error("Failed to load delegated DNS records:", err);
      delegatedDNSRecords.value = [];
    } finally {
      delegatedRecordsLoading.value = false;
    }
  }

  async function refresh() {
    await Promise.all([loadDNSRecords(), loadDNSConfig(), loadDelegatedDNSRecords()]);
  }

  async function loadAPIKeys() {
    apiKeysLoading.value = true;
    try {
      const response = await client.listDNSDelegationAPIKeys({});
      apiKeys.value = response.apiKeys || [];
    } catch (err) {
      console.error("Failed to load API keys:", err);
      apiKeys.value = [];
    } finally {
      apiKeysLoading.value = false;
    }
  }

  async function revokeAPIKeyByID(keyID: string) {
    if (!confirm("Are you sure you want to revoke this API key? This will stop DNS delegation from working.")) {
      return;
    }
    
    revokingAPIKey.value = keyID;
    try {
      // We need to revoke by key hash, but we only have the ID
      // For now, we'll use the organization revoke endpoint if we have org ID
      const key = apiKeys.value.find((k: DNSDelegationAPIKeyInfo) => k.id === keyID);
      if (key?.organization_id) {
        await client.revokeDNSDelegationAPIKeyForOrganization({
          organizationId: key.organization_id,
        });
        toast.success("API key revoked successfully");
        await loadAPIKeys();
      } else {
        toast.error("Cannot revoke: organization ID not found");
      }
    } catch (err: unknown) {
      toast.error((err as Error).message || "Failed to revoke API key");
    } finally {
      revokingAPIKey.value = null;
    }
  }

  function formatDate(timestamp: Timestamp | null | undefined): string {
    if (!timestamp) return "—";
    const seconds = typeof timestamp.seconds === "bigint"
      ? Number(timestamp.seconds)
      : (timestamp.seconds || 0);
    const nanos = timestamp.nanos || 0;
    const millis = seconds * 1000 + Math.floor(nanos / 1_000_000);
    return new Date(millis).toLocaleString();
  }

  const apiKeyColumns = computed(() => [
    { key: "description", label: "Description", defaultWidth: 200 },
    { key: "organization_id", label: "Organization", defaultWidth: 180 },
    { key: "is_active", label: "Status", defaultWidth: 100 },
    { key: "created_at", label: "Created", defaultWidth: 180 },
    { key: "revoked_at", label: "Revoked", defaultWidth: 180 },
    { key: "actions", label: "Actions", defaultWidth: 100 },
  ]);

  const apiKeyRows = computed(() => {
    return apiKeys.value.map((key: DNSDelegationAPIKeyInfo) => ({
      id: key.id,
      description: key.description,
      organization_id: key.organizationId,
      is_active: key.isActive,
      created_at: key.createdAt,
      revoked_at: key.revokedAt,
      stripe_subscription_id: key.stripeSubscriptionId,
    }));
  });

  function getStatusBadgeVariant(
    status: string | number
  ): "primary" | "secondary" | "success" | "warning" | "danger" | "outline" {
    // Handle both string and number status values
    let statusStr: string;
    if (typeof status === "number") {
      statusStr = convertStatusNumberToString(status);
    } else {
      // Handle string numbers like "3" or status names like "RUNNING"
      const numStatus = parseInt(String(status), 10);
      if (!isNaN(numStatus)) {
        statusStr = convertStatusNumberToString(numStatus);
      } else {
        statusStr = String(status || "").toUpperCase();
      }
    }
    switch (statusStr) {
      case "RUNNING":
        return "success";
      case "STOPPED":
        return "danger";
      case "BUILDING":
      case "DEPLOYING":
        return "warning";
      case "FAILED":
        return "danger";
      case "CREATED":
        return "secondary";
      default:
        return "secondary";
    }
  }

  function getStatusDotClass(status: string | number): string {
    let statusStr: string;
    if (typeof status === "number") {
      statusStr = convertStatusNumberToString(status);
    } else {
      const numStatus = parseInt(String(status), 10);
      if (!isNaN(numStatus)) {
        statusStr = convertStatusNumberToString(numStatus);
      } else {
        statusStr = String(status || "").toUpperCase();
      }
    }
    switch (statusStr) {
      case "RUNNING":
        return "bg-success animate-pulse";
      case "STOPPED":
        return "bg-danger";
      case "BUILDING":
      case "DEPLOYING":
        return "bg-warning animate-pulse";
      case "FAILED":
        return "bg-danger";
      case "CREATED":
        return "bg-secondary";
      default:
        return "bg-secondary";
    }
  }

  function getStatusLabel(status: string | number): string {
    let statusStr: string;
    if (typeof status === "number") {
      statusStr = convertStatusNumberToString(status);
    } else {
      const numStatus = parseInt(String(status), 10);
      if (!isNaN(numStatus)) {
        statusStr = convertStatusNumberToString(numStatus);
      } else {
        statusStr = String(status || "").toUpperCase();
      }
    }
    switch (statusStr) {
      case "RUNNING":
        return "Running";
      case "STOPPED":
        return "Stopped";
      case "BUILDING":
        return "Building";
      case "DEPLOYING":
        return "Deploying";
      case "FAILED":
        return "Failed";
      case "CREATED":
        return "Created";
      default:
        return statusStr || "Unknown";
    }
  }

  async function createAPIKey() {
    if (!apiKeyDescription.value) {
      apiKeyError.value = "Description is required";
      return;
    }

    creatingAPIKey.value = true;
    apiKeyError.value = "";

    try {
      const response = await client.createDNSDelegationAPIKey({
        description: apiKeyDescription.value,
        sourceApi: apiKeySourceAPI.value || undefined,
      });

      if (response.apiKey) {
        createdAPIKey.value = response.apiKey;
        apiKeyDescription.value = "";
        apiKeySourceAPI.value = "";
        toast.success("API key created successfully!");

        // Reload API keys list
        await loadAPIKeys();

        // Close dialog after showing key
        setTimeout(() => {
          createAPIKeyDialogOpen.value = false;
          createdAPIKey.value = null;
        }, 5000);
      }
    } catch (err: unknown) {
      apiKeyError.value = (err as Error).message || "Failed to create API key";
      toast.error(apiKeyError.value);
    } finally {
      creatingAPIKey.value = false;
    }
  }

  function copyAPIKey() {
    if (!createdAPIKey.value) return;
    navigator.clipboard.writeText(createdAPIKey.value);
    toast.success("API key copied to clipboard");
  }

  onMounted(async () => {
    await checkHasDelegatedDNS();
    refresh();
    loadAPIKeys();
    if (hasDelegatedDNS.value) {
      loadDelegatedDNSRecords();
    }
  });

  // Watch for filter changes to reload records
  watch(recordsRecordTypeFilter, () => {
    loadDNSRecords();
  });

  // Watch for delegated DNS records filter changes
  watch(
    [
      delegatedRecordsRecordTypeFilter,
      delegatedRecordsOrgFilter,
      delegatedRecordsAPIKeyFilter,
    ],
    () => {
      loadDelegatedDNSRecords();
    }
  );
</script>
