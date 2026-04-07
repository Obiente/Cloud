<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold"
          >Invoice Management</OuiText
        >
        <OuiText color="tertiary">
          View and manage all invoices across organizations.
        </OuiText>
      </OuiStack>
      <OuiFlex gap="sm" wrap="wrap">
        <div class="min-w-[140px]">
          <OuiInput
            v-model="startDate"
            type="date"
            size="sm"
            label="Start Date"
          />
        </div>
        <div class="min-w-[140px]">
          <OuiInput v-model="endDate" type="date" size="sm" label="End Date" />
        </div>
        <div class="min-w-[160px]">
          <OuiSelect
            v-model="statusFilter"
            :items="statusOptions"
            placeholder="Status"
            size="sm"
          />
        </div>
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
    </OuiFlex>

    <!-- Summary Metrics -->
    <OuiGrid :cols="{ sm: 1, md: 2, xl: 4 }" gap="md">
      <OuiCard
        v-for="metric in metrics"
        :key="metric.label"
        class="p-6 bg-surface-raised border border-border-muted rounded-xl"
      >
        <OuiStack gap="xs">
          <OuiText
            size="sm"
            weight="medium"
            color="tertiary"
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

    <!-- Invoices Table -->
    <OuiCard class="border border-border-muted rounded-xl overflow-hidden">
      <OuiCardHeader class="px-6 py-4 border-b border-border-muted">
        <OuiFlex align="center" justify="between">
          <OuiText tag="h2" size="xl" weight="bold">Invoices</OuiText>
          <div class="w-72 max-w-full">
            <OuiInput
              v-model="search"
              type="search"
              placeholder="Search invoices..."
              clearable
              size="sm"
            />
          </div>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody class="p-0">
        <div v-if="isLoading" class="text-center py-8">
          <OuiText color="tertiary">Loading invoices...</OuiText>
        </div>
        <OuiTable
          v-else
          :columns="columns"
          :rows="filteredInvoices"
          :empty-text="'No invoices found.'"
        >
          <template #cell-organization="{ value, row }">
            <div>
              <div class="flex items-center gap-1.5 flex-wrap">
                <NuxtLink
                  v-if="row.organizationId"
                  :to="`/superadmin/organizations/${row.organizationId}`"
                  class="font-medium text-text-primary hover:text-primary transition-colors"
                >
                  {{ row.organizationName }}
                </NuxtLink>
                <span v-else class="font-medium text-text-primary">{{ row.organizationName }}</span>
                <span
                  v-if="orgOwnerMap.get(row.organizationId)?.plan === 'personal' || orgOwnerMap.get(row.organizationId)?.slug?.startsWith('personal-') || row.organizationName === 'Personal'"
                  class="inline-flex items-center rounded px-1 py-0.5 text-[10px] font-medium bg-primary/10 text-primary leading-none"
                >Personal</span>
              </div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">
                {{ value }}
              </div>
              <div v-if="orgOwnerMap.get(row.organizationId)" class="text-xs text-text-muted mt-0.5 flex items-center gap-1">
                <span class="text-text-tertiary">Owner:</span>
                <NuxtLink
                  :to="`/superadmin/users/${orgOwnerMap.get(row.organizationId)?.ownerId}`"
                  class="text-primary hover:underline truncate max-w-[160px]"
                >{{ orgOwnerMap.get(row.organizationId)?.ownerName }}</NuxtLink>
              </div>
              <div v-else-if="row.customerEmail" class="text-xs text-text-muted mt-0.5">
                {{ row.customerEmail }}
              </div>
            </div>
          </template>
          <template #cell-invoiceLabel="{ value, row }">
            <div>
              <div class="font-medium text-text-primary">
                {{ row.invoice?.number || value }}
              </div>
              <div class="text-xs font-mono text-text-tertiary mt-0.5">
                {{ value }}
              </div>
            </div>
          </template>
          <template #cell-amount="{ row }">
            <div class="space-y-1">
              <div class="font-semibold">
                {{ formatCurrency(row.total ?? row.amountDue) }}
              </div>
              <div
                class="text-xs text-text-tertiary"
                v-if="row.amountPaid || row.amountRemaining !== undefined"
              >
                Paid {{ formatCurrency(row.amountPaid || 0) }}
                <span v-if="row.amountRemaining !== undefined">
                  • Remaining {{ formatCurrency(row.amountRemaining || 0) }}
                </span>
              </div>
            </div>
          </template>
          <template #cell-status="{ value }">
            <OuiBadge :variant="getStatusVariant(value)">
              {{ value.toUpperCase() }}
            </OuiBadge>
          </template>
          <template #cell-dueDate="{ value }">
            <span v-if="value" class="text-text-secondary">{{
              formatDate(value)
            }}</span>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-date="{ value }">
            <span v-if="value" class="text-text-secondary">{{
              formatDate(value)
            }}</span>
            <span v-else class="text-text-tertiary">—</span>
          </template>
          <template #cell-actions="{ row }">
            <OuiFlex gap="xs">
              <OuiButton
                v-if="row.invoice?.hostedInvoiceUrl"
                variant="ghost"
                size="xs"
                @click="openInvoice(row.invoice.hostedInvoiceUrl)"
              >
                View
              </OuiButton>
              <OuiButton
                v-if="
                  row.invoice?.status === 'open' ||
                  row.invoice?.status === 'draft'
                "
                variant="ghost"
                size="xs"
                @click="sendReminder(row.invoice?.id)"
                :disabled="sendingReminder === row.invoice?.id"
              >
                {{
                  sendingReminder === row.invoice?.id
                    ? "Sending..."
                    : "Send Reminder"
                }}
              </OuiButton>
            </OuiFlex>
          </template>
        </OuiTable>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { computed, ref } from "vue";
import {
  SuperadminService,
  type InvoiceWithOrganization,
} from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";

definePageMeta({
  middleware: ["auth", "superadmin"],
});

const client = useConnectClient(SuperadminService);

// Build org → owner map from the cached superadmin overview
const superAdmin = useSuperAdmin();
const orgOwnerMap = computed(() => {
  const map = new Map<string, { ownerName: string; ownerId: string; plan?: string; slug?: string }>();
  for (const org of superAdmin.overview.value?.organizations ?? []) {
    if (org.ownerId) {
      map.set(org.id, { ownerName: org.ownerName || org.ownerId, ownerId: org.ownerId, plan: org.plan, slug: org.slug });
    }
  }
  return map;
});

const sendingReminder = ref<string | null>(null);

const startDate = ref("");
const endDate = ref("");
const statusFilter = ref<string>("all");
const search = ref("");

// Set default dates (30 days ago to today)
const today = new Date();
const thirtyDaysAgo = new Date();
thirtyDaysAgo.setDate(today.getDate() - 30);
startDate.value = thirtyDaysAgo.toISOString().split("T")[0] || "";
endDate.value = today.toISOString().split("T")[0] || "";

const statusOptions = [
  { label: "All Statuses", value: "all" },
  { label: "Open", value: "open" },
  { label: "Paid", value: "paid" },
  { label: "Draft", value: "draft" },
  { label: "Void", value: "void" },
  { label: "Uncollectible", value: "uncollectible" },
];

async function fetchInvoices() {
  try {
    const response = await client.listAllInvoices({
      startDate: startDate.value ? startDate.value : undefined,
      endDate: endDate.value ? endDate.value : undefined,
      status: statusFilter.value !== "all" ? statusFilter.value : undefined,
      limit: 200,
    });
    return response;
  } catch (err) {
    console.error("Failed to fetch invoices:", err);
    throw err;
  }
}

// Use client-side fetching for non-blocking navigation
const {
  data: invoicesData,
  pending: isLoading,
  refresh: refreshInvoices,
} = useClientFetch("superadmin-invoices", fetchInvoices, {
  watch: [startDate, endDate, statusFilter],
});

const metrics = computed(() => {
  const invoices = invoicesData.value?.invoices || [];
  const total = invoices.length;
  const open = invoices.filter(
    (inv: InvoiceWithOrganization) => inv.invoice?.status === "open"
  ).length;
  const paid = invoices.filter(
    (inv: InvoiceWithOrganization) => inv.invoice?.status === "paid"
  ).length;
  const overdue = invoices.filter((inv: InvoiceWithOrganization) => {
    const dueDate = inv.invoice?.dueDate;
    if (!dueDate || inv.invoice?.status === "paid") return false;
    const due = new Date(Number(dueDate.seconds) * 1000);
    return due < new Date() && inv.invoice?.status === "open";
  }).length;

  return [
    {
      label: "Total Invoices",
      value: total.toString(),
    },
    {
      label: "Open",
      value: open.toString(),
    },
    {
      label: "Paid",
      value: paid.toString(),
    },
    {
      label: "Overdue",
      value: overdue.toString(),
    },
  ];
});

const invoices = computed(() => {
  return (
    invoicesData.value?.invoices?.map((inv: InvoiceWithOrganization) => {
      // Sanitize invoice object to convert BigInt values to numbers/strings
      const invoice = inv.invoice
        ? {
            ...inv.invoice,
            id: String(inv.invoice.id || ""),
            amountDue: Number(inv.invoice.amountDue || 0),
            amountPaid: Number(inv.invoice.amountPaid || 0),
            subtotal: inv.invoice.subtotal
              ? Number(inv.invoice.subtotal)
              : undefined,
            total: inv.invoice.total ? Number(inv.invoice.total) : undefined,
            amountRemaining: inv.invoice.amountRemaining
              ? Number(inv.invoice.amountRemaining)
              : undefined,
          }
        : null;

      return {
        id: String(inv.invoice?.id || ""),
        invoice: invoice,
        organization:
          inv.organizationName || inv.organizationId || "Unknown organization",
        invoiceLabel: invoice?.number || invoice?.id || "Unknown invoice",
        organizationId: inv.organizationId || "",
        organizationName: inv.organizationName || "Unknown",
        customerEmail: inv.customerEmail || "",
        amountDue: Number(inv.invoice?.amountDue || 0),
        amountPaid: Number(inv.invoice?.amountPaid || 0),
        subtotal:
          inv.invoice?.subtotal !== undefined
            ? Number(inv.invoice.subtotal)
            : undefined,
        total:
          inv.invoice?.total !== undefined
            ? Number(inv.invoice.total)
            : undefined,
        amountRemaining:
          inv.invoice?.amountRemaining !== undefined
            ? Number(inv.invoice.amountRemaining)
            : undefined,
        status: inv.invoice?.status || "unknown",
        dueDate: inv.invoice?.dueDate,
        date: inv.invoice?.date,
      };
    }) || []
  );
});

const filteredInvoices = computed(() => {
  const term = search.value.trim().toLowerCase();
  if (!term) return invoices.value;
  return invoices.value.filter((inv) => {
    const searchable = [
      inv.organizationId,
      inv.organizationName,
      inv.customerEmail,
      inv.invoice?.number,
      inv.invoice?.id,
      inv.invoice?.status,
    ]
      .filter(Boolean)
      .join(" ")
      .toLowerCase();
    return searchable.includes(term);
  });
});

const columns = computed(() => [
  {
    key: "organization",
    label: "Organization",
    defaultWidth: 200,
    minWidth: 150,
  },
  { key: "invoiceLabel", label: "Invoice", defaultWidth: 150, minWidth: 120 },
  {
    key: "customerEmail",
    label: "Customer Email",
    defaultWidth: 200,
    minWidth: 150,
  },
  { key: "amount", label: "Amounts", defaultWidth: 180, minWidth: 140 },
  { key: "status", label: "Status", defaultWidth: 120, minWidth: 100 },
  { key: "dueDate", label: "Due Date", defaultWidth: 150, minWidth: 120 },
  { key: "date", label: "Date", defaultWidth: 150, minWidth: 120 },
  {
    key: "actions",
    label: "Actions",
    defaultWidth: 150,
    minWidth: 120,
    resizable: false,
  },
]);

function formatCurrency(cents: number): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
  }).format(cents / 100);
}

function getStatusVariant(status: string): "success" | "danger" | "warning" {
  const lower = status.toLowerCase();
  if (lower === "paid") return "success";
  if (lower === "uncollectible" || lower === "void") return "danger";
  return "warning";
}

function formatDate(
  timestamp?: { seconds?: number | bigint; nanos?: number } | null
) {
  if (!timestamp || timestamp.seconds === undefined) return "—";
  const seconds =
    typeof timestamp.seconds === "bigint"
      ? Number(timestamp.seconds)
      : timestamp.seconds;
  const millis =
    seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(
    date
  );
}

function openInvoice(url?: string) {
  if (url) {
    window.open(url, "_blank");
  }
}

async function sendReminder(invoiceId?: string) {
  if (!invoiceId) return;

  sendingReminder.value = invoiceId;
  try {
    await client.sendInvoiceReminder({
      invoiceId: invoiceId,
    });
    // Refresh invoices after sending
    await refreshInvoices();
  } catch (err) {
    console.error("Failed to send invoice reminder:", err);
    alert("Failed to send invoice reminder. Please try again.");
  } finally {
    sendingReminder.value = null;
  }
}

function refresh() {
  void refreshInvoices();
}
</script>
