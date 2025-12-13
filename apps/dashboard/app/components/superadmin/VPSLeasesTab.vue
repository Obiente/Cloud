<template>
  <SuperadminPageLayout
    title="All DHCP Leases"
    description="View DHCP leases across all VPS instances."
    :columns="tableColumns"
    :rows="filteredTableRows"
    :filters="filterConfigs"
    :search="search"
    :empty-text="isLoading ? 'Loading leases…' : 'No DHCP leases found. This feature requires VPSGatewayService integration.'"
    :loading="isLoading"
    search-placeholder="Search by IP, MAC, VPS name, organization…"
    @update:search="search = $event"
    @filter-change="handleFilterChange"
    @refresh="() => fetchLeases()"
  >
    <template #cell-ip="{ row }">
      <OuiFlex align="center" gap="sm">
        <OuiText weight="medium" class="font-mono text-sm">
          {{ row.ip }}
        </OuiText>
        <OuiButton
          size="sm"
          variant="ghost"
          @click.stop="copyToClipboard(row.ip)"
          class="p-1 h-auto opacity-0 group-hover:opacity-100 transition-opacity"
        >
          <DocumentDuplicateIcon class="w-4 h-4" />
        </OuiButton>
      </OuiFlex>
    </template>

    <template #cell-mac="{ row }">
      <OuiText class="font-mono text-sm" color="secondary">
        {{ row.mac }}
      </OuiText>
    </template>

    <template #cell-vps="{ row }">
      <OuiText weight="medium">
        {{ row.vpsId }}
      </OuiText>
    </template>

    <template #cell-organization="{ row }">
      <OuiText color="secondary" size="sm">
        {{ row.organizationName || "N/A" }}
      </OuiText>
    </template>

    <template #cell-expiry="{ row }">
      <OuiFlex align="center" gap="sm">
        <OuiBox
          v-if="isExpiring(row.expiresAt)"
          class="w-2 h-2 rounded-full bg-warning"
        />
        <OuiText size="sm">
          {{ formatExpiry(row.expiresAt) }}
        </OuiText>
      </OuiFlex>
    </template>

    <template #cell-type="{ row }">
      <OuiBadge v-if="row.isPublic" color="success">
        Public IP
      </OuiBadge>
      <OuiBadge v-else color="info">
        Pool
      </OuiBadge>
    </template>
  </SuperadminPageLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from "vue";
import { DocumentDuplicateIcon } from "@heroicons/vue/24/outline";
import SuperadminPageLayout from "~/components/superadmin/SuperadminPageLayout.vue";
import OuiText from "~/components/oui/Text.vue";
import OuiBox from "~/components/oui/Box.vue";
import OuiFlex from "~/components/oui/Flex.vue";
import OuiButton from "~/components/oui/Button.vue";
import OuiBadge from "~/components/oui/Badge.vue";
import { useToast } from "~/composables/useToast";
import { useConnectClient } from "~/lib/connect-client";
import { SuperadminService, OrganizationService } from "@obiente/proto";

interface Lease {
  ip: string;
  mac: string;
  vpsId: string;
  organizationId: string;
  organizationName: string;
  expiresAt?: Date;
  isPublic: boolean;
}

const { toast } = useToast();
const superadminClient = useConnectClient(SuperadminService);
const orgClient = useConnectClient(OrganizationService);

const isLoading = ref(false);
const search = ref("");
const leases = ref<Lease[]>([]);
const appliedFilters = ref<Record<string, string>>({});

const tableColumns = computed(() => [
  { key: "ip", label: "IP Address", defaultWidth: 200, minWidth: 150 },
  { key: "mac", label: "MAC Address", defaultWidth: 200, minWidth: 150 },
  { key: "vps", label: "VPS Instance", defaultWidth: 180, minWidth: 150 },
  { key: "organization", label: "Organization", defaultWidth: 180, minWidth: 150 },
  { key: "expiry", label: "Expiry", defaultWidth: 150, minWidth: 120 },
  { key: "type", label: "Type", defaultWidth: 120, minWidth: 100 },
]);

const filterConfigs = computed(() => [
  {
    key: "type",
    placeholder: "Lease Type",
    items: [
      { key: "public", label: "Public IP", value: "public" },
      { key: "pool", label: "Pool", value: "pool" },
    ],
  },
  {
    key: "expiry",
    placeholder: "Expiry Status",
    items: [
      { key: "expiring", label: "Expiring Soon (< 24h)", value: "expiring" },
      { key: "active", label: "Active", value: "active" },
    ],
  },
]);

const filteredTableRows = computed(() => {
  return leases.value.filter((lease) => {
    const searchLower = search.value.toLowerCase();
    const matchesSearch =
      lease.ip.toLowerCase().includes(searchLower) ||
      lease.mac.toLowerCase().includes(searchLower) ||
      lease.vpsId.toLowerCase().includes(searchLower) ||
      lease.organizationName.toLowerCase().includes(searchLower);

    if (!matchesSearch) return false;

    const typeFilter = appliedFilters.value.type;
    if (typeFilter === "public" && !lease.isPublic) return false;
    if (typeFilter === "pool" && lease.isPublic) return false;

    const expiryFilter = appliedFilters.value.expiry;
    if (expiryFilter === "expiring" && !isExpiring(lease.expiresAt)) return false;
    if (expiryFilter === "active" && isExpiring(lease.expiresAt)) return false;

    return true;
  });
});

const fetchLeases = async () => {
  isLoading.value = true;
  try {
    // Get all organizations first
    const orgsResponse = await orgClient.listOrganizations({
      onlyMine: false,
    });
    const organizations = orgsResponse.organizations || [];

    // Create a map of org ID to org name for quick lookup
    const orgMap = new Map(
      organizations.map((org: any) => [org.id, org.name])
    );

    const allLeases: Lease[] = [];

    // Fetch leases for each organization
    for (const org of organizations) {
      try {
        const leaseResponse = await superadminClient.getOrgLeases({
          organizationId: (org as any).id,
          vpsId: "", // Empty to get all VPS instances for the org
        });

        const orgLeases = leaseResponse.leases || [];
        allLeases.push(
          ...orgLeases.map((lease: any) => ({
            ip: lease.ipAddress,
            mac: lease.macAddress,
            vpsId: lease.vpsId,
            organizationId: lease.organizationId,
            organizationName: orgMap.get(lease.organizationId) || lease.organizationId,
            expiresAt: lease.expiresAt?.toDate?.(),
            isPublic: lease.isPublic,
          }))
        );
      } catch (error) {
        console.error(`Failed to fetch leases for organization ${(org as any).id}:`, error);
      }
    }

    leases.value = allLeases;
  } catch (error: any) {
    toast.error(`Failed to load leases: ${error?.message || "Unknown error"}`);
  } finally {
    isLoading.value = false;
  }
};

const handleFilterChange = (key: string, value: string) => {
  if (appliedFilters.value[key] === value) {
    delete appliedFilters.value[key];
  } else {
    appliedFilters.value[key] = value;
  }
};

const isExpiring = (expiresAt?: Date): boolean => {
  if (!expiresAt) return false;
  const now = new Date();
  const hoursUntilExpiry = (expiresAt.getTime() - now.getTime()) / (1000 * 60 * 60);
  return hoursUntilExpiry < 24;
};

const formatExpiry = (expiresAt?: Date): string => {
  if (!expiresAt) return "Unknown";

  const now = new Date().getTime();
  const expireTime = expiresAt.getTime();
  const diffMs = expireTime - now;

  if (diffMs < 0) return "Expired";

  const hours = Math.floor(diffMs / (60 * 60 * 1000));
  if (hours < 24) return `${hours}h remaining`;

  const days = Math.floor(hours / 24);
  return `${days}d remaining`;
};

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success("IP address copied to clipboard");
  } catch (error) {
    toast.error("Failed to copy to clipboard");
  }
};

onMounted(() => {
  fetchLeases();
});
</script>
