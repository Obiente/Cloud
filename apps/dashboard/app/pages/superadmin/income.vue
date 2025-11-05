<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Income Overview</OuiText>
        <OuiText color="muted">
          View billing analytics, revenue trends, and top customers.
        </OuiText>
      </OuiStack>
      <OuiFlex gap="sm" wrap="wrap">
        <div class="min-w-[140px]">
          <OuiInput
            v-model="startDate"
            type="date"
            size="sm"
            label="Start Date"
            @change="fetchIncome"
          />
        </div>
        <div class="min-w-[140px]">
          <OuiInput
            v-model="endDate"
            type="date"
            size="sm"
            label="End Date"
            @change="fetchIncome"
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
        v-for="metric in summaryMetrics"
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

    <!-- Payment Metrics -->
    <OuiCard class="border border-border-muted rounded-xl">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">Payment Metrics</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-6">
        <OuiGrid cols="1" colsMd="3" gap="md">
          <div>
            <OuiText size="sm" color="muted">Success Rate</OuiText>
            <OuiText size="2xl" weight="semibold">
              {{ paymentMetrics.successRate?.toFixed(1) || 0 }}%
            </OuiText>
          </div>
          <div>
            <OuiText size="sm" color="muted">Successful Payments</OuiText>
            <OuiText size="2xl" weight="semibold">
              {{ paymentMetrics.successfulPayments || 0 }}
            </OuiText>
          </div>
          <div>
            <OuiText size="sm" color="muted">Average Payment</OuiText>
            <OuiText size="2xl" weight="semibold">
              {{ formatCurrency(paymentMetrics.averagePaymentAmount || 0) }}
            </OuiText>
          </div>
        </OuiGrid>
      </OuiCardBody>
    </OuiCard>

    <!-- Monthly Income Chart -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">Monthly Income</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-6">
        <div v-if="monthlyIncome.length === 0" class="text-center py-8">
          <OuiText color="muted">No monthly data available</OuiText>
        </div>
        <OuiTable
          v-else
          :columns="monthlyColumns"
          :rows="monthlyIncome"
          :empty-text="'No monthly income data.'"
        />
      </OuiCardBody>
    </OuiCard>

    <!-- Top Customers -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiText tag="h2" size="xl" weight="bold">Top Customers</OuiText>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="customerColumns"
          :rows="topCustomers"
          :empty-text="'No customer data available.'"
        >
          <template #cell-organization="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ row.organizationName }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ value }}</div>
            </div>
          </template>
          <template #cell-totalRevenue="{ value }">
            <span class="font-semibold text-success">{{ formatCurrency(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <OuiButton
              variant="ghost"
              size="xs"
              @click="viewOrganization(row.organizationId)"
            >
              View
            </OuiButton>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>

    <!-- Billing Transactions -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiFlex align="center" justify="between">
          <OuiText tag="h2" size="xl" weight="bold">Billing Transactions</OuiText>
          <div class="w-72 max-w-full">
            <OuiInput
              v-model="transactionSearch"
              type="search"
              placeholder="Search transactions..."
              clearable
              size="sm"
            />
          </div>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <OuiTable
          :columns="transactionColumns"
          :rows="filteredTransactions"
          :empty-text="'No transactions found.'"
        >
          <template #cell-organization="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">{{ row.organizationName }}</div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">{{ value }}</div>
            </div>
          </template>
          <template #cell-amount="{ value }">
            <span
              class="font-semibold"
              :class="value >= 0 ? 'text-success' : 'text-danger'"
            >
              {{ formatCurrency(value) }}
            </span>
          </template>
          <template #cell-status="{ value }">
            <OuiBadge :variant="getStatusVariant(value)">
              {{ value }}
            </OuiBadge>
          </template>
        </OuiTable>
      </OuiCardBody>
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

const isLoading = ref(false);
const incomeData = ref<any>(null);

const startDate = ref("");
const endDate = ref("");
const transactionSearch = ref("");

// Set default dates (30 days ago to today)
const today = new Date();
const thirtyDaysAgo = new Date();
thirtyDaysAgo.setDate(today.getDate() - 30);
startDate.value = thirtyDaysAgo.toISOString().split("T")[0] || "";
endDate.value = today.toISOString().split("T")[0] || "";

async function fetchIncome() {
  isLoading.value = true;
  try {
    const response = await client.getIncomeOverview({
      startDate: startDate.value ? startDate.value : undefined,
      endDate: endDate.value ? endDate.value : undefined,
    });
    incomeData.value = response;
  } catch (err) {
    console.error("Failed to fetch income overview:", err);
  } finally {
    isLoading.value = false;
  }
}

await fetchIncome();

const summaryMetrics = computed(() => {
  const s = incomeData.value?.summary;
  if (!s) return [];
  return [
    {
      label: "Total Revenue",
      value: formatCurrency(s.totalRevenue || 0),
    },
    {
      label: "Net Revenue",
      value: formatCurrency(s.netRevenue || 0),
    },
    {
      label: "Monthly Recurring",
      value: formatCurrency(s.monthlyRecurringRevenue || 0),
    },
    {
      label: "Avg Monthly",
      value: formatCurrency(s.averageMonthlyRevenue || 0),
    },
    {
      label: "Estimated Monthly",
      value: formatCurrency(s.estimatedMonthlyIncome || 0),
    },
    {
      label: "Total Transactions",
      value: s.totalTransactions?.toString() || "0",
    },
    {
      label: "Total Refunds",
      value: formatCurrency(s.totalRefunds || 0),
    },
  ];
});

const paymentMetrics = computed(() => {
  return incomeData.value?.paymentMetrics || {};
});

const monthlyIncome = computed(() => {
  return (
    incomeData.value?.monthlyIncome?.map((mi: any) => ({
      month: mi.month || "—",
      revenue: mi.revenue || 0,
      transactionCount: mi.transactionCount || 0,
      refunds: mi.refunds || 0,
      net: (mi.revenue || 0) - (mi.refunds || 0),
    })) || []
  );
});

const topCustomers = computed(() => {
  return (
    incomeData.value?.topCustomers?.map((tc: any) => ({
      organizationId: tc.organizationId,
      organizationName: tc.organizationName || "Unknown",
      totalRevenue: tc.totalRevenue || 0,
      transactionCount: tc.transactionCount || 0,
      firstPayment: formatDate(tc.firstPayment),
      lastPayment: formatDate(tc.lastPayment),
    })) || []
  );
});

const transactions = computed(() => {
  return (
    incomeData.value?.transactions?.map((t: any) => ({
      id: t.id,
      organizationId: t.organizationId,
      organizationName: t.organizationName || "Unknown",
      type: t.type || "—",
      amountCents: t.amountCents || 0,
      amount: (t.amountCents || 0) / 100,
      currency: t.currency || "USD",
      status: t.status || "—",
      stripeInvoiceId: t.stripeInvoiceId,
      stripePaymentIntentId: t.stripePaymentIntentId,
      note: t.note || "—",
      createdAt: formatDate(t.createdAt),
    })) || []
  );
});

const filteredTransactions = computed(() => {
  const term = transactionSearch.value.trim().toLowerCase();
  if (!term) return transactions.value;
  return transactions.value.filter((t: any) => {
    const searchable = [
      t.organizationId,
      t.organizationName,
      t.type,
      t.status,
      t.note,
      t.id,
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    return searchable.includes(term);
  });
});

const monthlyColumns = computed(() => [
  { key: "month", label: "Month", defaultWidth: 120, minWidth: 100 },
  { key: "revenue", label: "Revenue", defaultWidth: 150, minWidth: 120 },
  { key: "refunds", label: "Refunds", defaultWidth: 150, minWidth: 120 },
  { key: "net", label: "Net", defaultWidth: 150, minWidth: 120 },
  { key: "transactionCount", label: "Transactions", defaultWidth: 120, minWidth: 100 },
]);

const customerColumns = computed(() => [
  { key: "organization", label: "Organization", defaultWidth: 200, minWidth: 150 },
  { key: "totalRevenue", label: "Total Revenue", defaultWidth: 150, minWidth: 120 },
  { key: "transactionCount", label: "Transactions", defaultWidth: 120, minWidth: 100 },
  { key: "firstPayment", label: "First Payment", defaultWidth: 150, minWidth: 120 },
  { key: "lastPayment", label: "Last Payment", defaultWidth: 150, minWidth: 120 },
  { key: "actions", label: "Actions", defaultWidth: 100, minWidth: 80, resizable: false },
]);

const transactionColumns = computed(() => [
  { key: "organization", label: "Organization", defaultWidth: 200, minWidth: 150 },
  { key: "type", label: "Type", defaultWidth: 120, minWidth: 100 },
  { key: "amount", label: "Amount", defaultWidth: 120, minWidth: 100 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "createdAt", label: "Date", defaultWidth: 150, minWidth: 120 },
  { key: "note", label: "Note", defaultWidth: 200, minWidth: 150 },
]);

function formatCurrency(cents: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(cents);
}

function getStatusVariant(status: string): "success" | "danger" | "warning" {
  const lower = status.toLowerCase();
  if (lower === "succeeded" || lower === "paid") return "success";
  if (lower === "failed" || lower === "refunded") return "danger";
  return "warning";
}

function viewOrganization(orgId: string) {
  router.push(`/superadmin/organizations?org=${orgId}`);
}

function refresh() {
  fetchIncome();
}

function formatDate(timestamp?: { seconds?: number | bigint; nanos?: number } | null) {
  if (!timestamp || timestamp.seconds === undefined) return "—";
  const seconds = typeof timestamp.seconds === "bigint" ? Number(timestamp.seconds) : timestamp.seconds;
  const millis = seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium", timeStyle: "short" }).format(date);
}
</script>

