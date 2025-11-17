<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Stripe Webhook Events</OuiText>
        <OuiText color="muted">
          View all Stripe webhook events and their associated organizations.
        </OuiText>
      </OuiStack>
      <OuiFlex gap="sm" wrap="wrap">
        <div class="min-w-[200px]">
          <OuiInput
            v-model="eventTypeFilter"
            placeholder="Event Type"
            size="sm"
            clearable
            @change="fetchEvents"
          />
        </div>
        <div class="min-w-[200px]">
          <OuiInput
            v-model="customerIdFilter"
            placeholder="Customer ID"
            size="sm"
            clearable
            @change="fetchEvents"
          />
        </div>
        <OuiButton variant="ghost" size="sm" @click="refresh" :disabled="isLoading">
          <span class="flex items-center gap-2">
            <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
            Refresh
          </span>
        </OuiButton>
      </OuiFlex>
    </OuiFlex>

    <!-- Summary Metrics -->
    <OuiGrid class="gap-4" cols="1" colsMd="2" colsXl="4">
      <OuiCard
        v-for="metric in metrics"
        :key="metric.label"
        class="p-6 bg-surface-raised border border-border-muted rounded-xl"
      >
        <OuiStack gap="xs">
          <OuiText
            size="sm"
            weight="medium"
            color="secondary"
            transform="uppercase"
            class="tracking-wide"
            >{{ metric.label }}</OuiText
          >
          <OuiText size="3xl" weight="semibold" color="primary">{{
            metric.value
          }}</OuiText>
        </OuiStack>
      </OuiCard>
    </OuiGrid>

    <!-- Events Table -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiFlex align="center" justify="between">
          <OuiText tag="h2" size="xl" weight="bold">Events</OuiText>
          <div class="w-72 max-w-full">
            <OuiInput
              v-model="search"
              type="search"
              placeholder="Search events..."
              clearable
              size="sm"
            />
          </div>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <div v-if="isLoading" class="text-center py-8">
          <OuiText color="muted">Loading events...</OuiText>
        </div>
        <OuiTable
          v-else
          :columns="columns"
          :rows="filteredEvents"
          :empty-text="'No events found.'"
        >
          <template #cell-eventType="{ value }">
            <OuiBadge variant="secondary" size="sm">
              {{ value }}
            </OuiBadge>
          </template>
          <template #cell-organization="{ value, row }">
            <div v-if="value">
              <div class="font-medium text-text-primary">{{ row.organizationName || "Unknown" }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ value }}</div>
            </div>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-customerId="{ value }">
            <span v-if="value" class="font-mono text-sm text-text-secondary">{{ value }}</span>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-subscriptionId="{ value }">
            <span v-if="value" class="font-mono text-sm text-text-secondary">{{ value }}</span>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-invoiceId="{ value }">
            <span v-if="value" class="font-mono text-sm text-text-secondary">{{ value }}</span>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-checkoutSessionId="{ value }">
            <span v-if="value" class="font-mono text-sm text-text-secondary">{{ value }}</span>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-processedAt="{ value }">
            <span v-if="value" class="text-text-secondary">{{ formatDate(value) }}</span>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-actions="{ row }">
            <OuiFlex gap="xs">
              <OuiButton
                variant="ghost"
                size="xs"
                @click="viewEvent(row)"
              >
                View Details
              </OuiButton>
              <OuiButton
                v-if="row.organizationId"
                variant="ghost"
                size="xs"
                @click="viewOrganization(row.organizationId)"
              >
                View Org
              </OuiButton>
            </OuiFlex>
          </template>
        </OuiTable>
      </OuiCardBody>
      <OuiCardFooter v-if="totalCount > limit" class="px-6 py-4 border-t border-border-muted">
        <OuiFlex align="center" justify="between">
          <OuiText size="sm" color="muted">
            Showing {{ events.length }} of {{ totalCount }} events
          </OuiText>
          <OuiFlex gap="sm">
            <OuiButton
              variant="ghost"
              size="sm"
              @click="loadMore"
              :disabled="isLoading || events.length >= totalCount"
            >
              Load More
            </OuiButton>
          </OuiFlex>
        </OuiFlex>
      </OuiCardFooter>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";
import { SuperadminService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useRouter } from "vue-router";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const router = useRouter();
const client = useConnectClient(SuperadminService);

const events = ref<any[]>([]);
const totalCount = ref(0);
const limit = ref(50);
const offset = ref(0);

const eventTypeFilter = ref("");
const customerIdFilter = ref("");
const search = ref("");

const columns = [
  { key: "id", label: "Event ID", sortable: true },
  { key: "eventType", label: "Event Type", sortable: true },
  { key: "organization", label: "Organization", sortable: true },
  { key: "customerId", label: "Customer ID", sortable: false },
  { key: "subscriptionId", label: "Subscription ID", sortable: false },
  { key: "invoiceId", label: "Invoice ID", sortable: false },
  { key: "checkoutSessionId", label: "Checkout Session ID", sortable: false },
  { key: "processedAt", label: "Processed At", sortable: true },
  { key: "actions", label: "Actions", sortable: false },
];

async function fetchEvents() {
  try {
    const response = await client.listStripeWebhookEvents({
      eventType: eventTypeFilter.value || undefined,
      customerId: customerIdFilter.value || undefined,
      limit: limit.value,
      offset: offset.value,
    });
    if (offset.value === 0) {
      events.value = response.events || [];
    } else {
      events.value = [...events.value, ...(response.events || [])];
    }
    totalCount.value = Number(response.totalCount || 0);
  } catch (err) {
    console.error("Failed to fetch webhook events:", err);
  }
}

async function loadMore() {
  offset.value += limit.value;
  await fetchEvents();
}

async function refresh() {
  offset.value = 0;
  await fetchEvents();
}

// Use client-side fetching for non-blocking navigation
const { pending: isLoading } = useClientFetch(
  () => `superadmin-webhook-events-${eventTypeFilter.value}-${customerIdFilter.value}-${offset.value}`,
  fetchEvents
);

const metrics = computed(() => {
  const total = totalCount.value;
  const withOrg = events.value.filter((e) => e.organizationId).length;
  const invoiceEvents = events.value.filter((e) => e.eventType?.startsWith("invoice.")).length;
  const subscriptionEvents = events.value.filter((e) => e.eventType?.startsWith("customer.subscription.")).length;

  return [
    {
      label: "Total Events",
      value: total.toString(),
    },
    {
      label: "With Organization",
      value: withOrg.toString(),
    },
    {
      label: "Invoice Events",
      value: invoiceEvents.toString(),
    },
    {
      label: "Subscription Events",
      value: subscriptionEvents.toString(),
    },
  ];
});

const filteredEvents = computed(() => {
  let filtered = events.value;
  
  if (search.value) {
    const searchLower = search.value.toLowerCase();
    filtered = filtered.filter((e) => {
      return (
        e.id?.toLowerCase().includes(searchLower) ||
        e.eventType?.toLowerCase().includes(searchLower) ||
        e.organizationName?.toLowerCase().includes(searchLower) ||
        e.organizationId?.toLowerCase().includes(searchLower) ||
        e.customerId?.toLowerCase().includes(searchLower) ||
        e.subscriptionId?.toLowerCase().includes(searchLower) ||
        e.invoiceId?.toLowerCase().includes(searchLower)
      );
    });
  }
  
  return filtered;
});

function formatDate(timestamp: any): string {
  if (!timestamp) return "—";
  const date = new Date(Number(timestamp.seconds) * 1000);
  return date.toLocaleString();
}

function viewEvent(event: any) {
  // Could open a dialog with full event details
  console.log("View event:", event);
}

function viewOrganization(orgId: string) {
  router.push(`/superadmin/organizations?org=${orgId}`);
}
</script>

