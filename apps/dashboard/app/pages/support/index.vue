<template>
  <OuiContainer size="full" p="none">
    <OuiStack gap="lg">
      <!-- Page Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="md">
        <OuiStack gap="xs">
          <OuiText as="h1" size="xl" weight="semibold">Support</OuiText>
          <OuiText color="tertiary" size="sm">Manage and track your support tickets.</OuiText>
        </OuiStack>
        <OuiButton @click="showCreateDialog = true" color="primary" size="sm" class="gap-1.5">
          <PlusIcon class="h-3.5 w-3.5" />
          New Ticket
        </OuiButton>
      </OuiFlex>

      <ErrorAlert v-if="error" :error="error" title="Failed to load tickets" hint="Please try refreshing the page. If the problem persists, contact support." />

      <!-- Stats Row -->
      <OuiGrid v-if="tickets" :cols="{ sm: 2, md: 4 }" gap="sm">
        <UiStatCard label="Open" :icon="InboxIcon" color="primary" :value="String(ticketStats.open)" />
        <UiStatCard label="In Progress" :icon="ArrowPathIcon" color="warning" :value="String(ticketStats.inProgress)" />
        <UiStatCard label="Resolved" :icon="CheckCircleIcon" color="success" :value="String(ticketStats.resolved)" />
        <UiStatCard label="Total" :icon="ChatBubbleLeftRightIcon" color="secondary" :value="String(tickets.length)" />
      </OuiGrid>

      <!-- Toolbar -->
      <OuiCard variant="outline">
        <OuiCardBody class="py-2! px-4!">
          <OuiFlex align="center" gap="md" wrap="wrap">
            <OuiInput
              v-model="searchQuery"
              placeholder="Search tickets…"
              size="sm"
              class="flex-1"
              :style="{ minWidth: '160px' }"
              @update:model-value="handleSearch"
            >
              <template #prefix>
                <MagnifyingGlassIcon class="h-3.5 w-3.5 text-tertiary" />
              </template>
              <template v-if="searchQuery" #suffix>
                <button class="text-tertiary hover:text-primary transition-colors" @click="searchQuery = ''">
                  <XMarkIcon class="h-3.5 w-3.5" />
                </button>
              </template>
            </OuiInput>
            <OuiSelect
              v-model="filters.status"
              :items="statusOptions"
              placeholder="All Status"
              size="sm"
              clearable
              :style="{ minWidth: '130px' }"
              @update:model-value="refreshTickets"
            />
            <OuiSelect
              v-model="filters.category"
              :items="categoryOptions"
              placeholder="All Categories"
              size="sm"
              clearable
              :style="{ minWidth: '150px' }"
              @update:model-value="refreshTickets"
            />
            <OuiSelect
              v-model="filters.priority"
              :items="priorityOptions"
              placeholder="All Priorities"
              size="sm"
              clearable
              :style="{ minWidth: '130px' }"
              @update:model-value="refreshTickets"
            />
            <OuiButton
              v-if="hasActiveFilters"
              variant="ghost"
              size="sm"
              class="gap-1 shrink-0 whitespace-nowrap"
              @click="clearFilters"
            >
              <XMarkIcon class="h-3.5 w-3.5" />
              Clear
            </OuiButton>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Loading Skeletons -->
      <OuiStack v-if="pending && !tickets" gap="sm">
        <OuiCard v-for="i in 5" :key="i" variant="outline">
          <OuiCardBody>
            <OuiFlex align="center" gap="md">
              <OuiSkeleton width="2.5rem" height="2.5rem" variant="rectangle" rounded class="shrink-0" />
              <OuiStack gap="xs" class="flex-1 min-w-0">
                <OuiSkeleton width="55%" height="0.875rem" variant="text" />
                <OuiSkeleton width="80%" height="0.75rem" variant="text" />
              </OuiStack>
              <OuiFlex gap="xs" class="shrink-0">
                <OuiSkeleton width="4rem" height="1.25rem" variant="rectangle" rounded />
                <OuiSkeleton width="4rem" height="1.25rem" variant="rectangle" rounded />
              </OuiFlex>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </OuiStack>

      <!-- Empty State -->
      <SharedEmptyState
        v-else-if="!filteredTickets?.length"
        :icon="InboxIcon"
        :title="hasActiveFilters ? 'No tickets match your filters' : 'No tickets yet'"
        :description="hasActiveFilters ? 'Try adjusting your filters or search query.' : 'Create your first support ticket to get started.'"
      >
        <OuiButton
          v-if="hasActiveFilters"
          @click="clearFilters"
          variant="ghost"
          size="sm"
          class="gap-1.5"
        >
          <XMarkIcon class="h-3.5 w-3.5" />
          Clear Filters
        </OuiButton>
        <OuiButton
          v-else
          @click="showCreateDialog = true"
          color="primary"
          size="sm"
          class="gap-1.5"
        >
          <PlusIcon class="h-3.5 w-3.5" />
          Create Ticket
        </OuiButton>
      </SharedEmptyState>

      <!-- Tickets List -->
      <OuiStack v-else gap="sm">
        <OuiCard
          v-for="ticket in filteredTickets"
          :key="ticket.id"
          variant="outline"
          class="cursor-pointer hover:border-border-strong transition-colors"
          @click="navigateToTicket(ticket.id)"
        >
          <OuiCardBody>
            <OuiFlex align="start" gap="md">
              <!-- Status dot -->
              <div class="shrink-0 mt-1">
                <span
                  class="block h-2 w-2 rounded-full"
                  :class="{
                    'bg-accent-primary': ticket.status === SupportTicketStatus.OPEN,
                    'bg-warning': ticket.status === SupportTicketStatus.IN_PROGRESS || ticket.status === SupportTicketStatus.WAITING_FOR_USER,
                    'bg-success': ticket.status === SupportTicketStatus.RESOLVED,
                    'bg-border-strong': ticket.status === SupportTicketStatus.CLOSED,
                  }"
                />
              </div>
              <!-- Content -->
              <OuiStack gap="xs" class="flex-1 min-w-0">
                <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
                  <OuiText size="sm" weight="semibold" class="line-clamp-1 flex-1 min-w-0">{{ ticket.subject }}</OuiText>
                  <OuiFlex gap="xs" class="shrink-0">
                    <OuiBadge :variant="getStatusColor(ticket.status) as any" size="xs">{{ getStatusLabel(ticket.status) }}</OuiBadge>
                    <OuiBadge :variant="getPriorityColor(ticket.priority) as any" size="xs">{{ getPriorityLabel(ticket.priority) }}</OuiBadge>
                    <OuiBadge :variant="getCategoryVariant(ticket.category) as any" size="xs" class="hidden sm:inline-flex">{{ getCategoryLabel(ticket.category) }}</OuiBadge>
                  </OuiFlex>
                </OuiFlex>
                <OuiText color="tertiary" size="xs" class="line-clamp-1">{{ ticket.description }}</OuiText>
                <OuiFlex gap="sm" align="center" wrap="wrap">
                  <OuiText size="xs" color="tertiary">
                    <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                  </OuiText>
                  <span class="text-border-strong text-xs">·</span>
                  <OuiText v-if="ticket.createdByName || ticket.createdByEmail" size="xs" color="tertiary">
                    {{ ticket.createdByName || ticket.createdByEmail || 'Unknown' }}
                  </OuiText>
                  <template v-if="ticket.commentCount > 0">
                    <span class="text-border-strong text-xs">·</span>
                    <OuiFlex align="center" gap="xs">
                      <ChatBubbleLeftRightIcon class="h-3 w-3 text-tertiary" />
                      <OuiText size="xs" color="tertiary">{{ ticket.commentCount }}</OuiText>
                    </OuiFlex>
                  </template>
                </OuiFlex>
              </OuiStack>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </OuiStack>
    </OuiStack>

    <!-- Create Ticket Dialog -->
    <OuiDialog
      v-model:open="showCreateDialog"
      title="Create Support Ticket"
      description="Describe your issue and we'll get back to you as soon as possible."
    >
      <OuiStack gap="md" class="py-1">
        <OuiInput
          v-model="newTicket.subject"
          label="Subject"
          placeholder="Brief description of your issue"
          required
        />
        <OuiTextarea
          v-model="newTicket.description"
          label="Description"
          placeholder="Please provide as much detail as possible…"
          :rows="4"
          required
        />
        <OuiGrid :cols="{ sm: 1, md: 2 }" gap="md">
          <OuiSelect
            v-model="newTicket.category"
            label="Category"
            :items="categoryOptions"
            required
          />
          <OuiSelect
            v-model="newTicket.priority"
            label="Priority"
            :items="priorityOptions"
            required
          />
        </OuiGrid>
      </OuiStack>
      <template #footer>
        <OuiFlex justify="end" gap="sm">
          <OuiButton @click="showCreateDialog = false" variant="ghost">Cancel</OuiButton>
          <OuiButton @click="createTicket" color="primary" :disabled="!canCreateTicket" :loading="creating">
            Create Ticket
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>
  </OuiContainer>
</template>

<script setup lang="ts">
definePageMeta({ layout: "default", middleware: "auth" });

import {
  SupportService,
  SupportTicketStatus,
  SupportTicketPriority,
  SupportTicketCategory,
  type SupportTicket,
} from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import {
  PlusIcon,
  MagnifyingGlassIcon,
  XMarkIcon,
  ArrowPathIcon,
  CheckCircleIcon,
  InboxIcon,
  ChatBubbleLeftRightIcon,
} from "@heroicons/vue/24/outline";
import { ConnectError, Code } from "@connectrpc/connect";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import { useSuperAdmin } from "~/composables/useSuperAdmin";
import { useDocumentVisibility } from "@vueuse/core";

const client = useConnectClient(SupportService);
const router = useRouter();
const auth = useAuth();
const superAdmin = useSuperAdmin();
const isSuperAdmin = computed(() => superAdmin.allowed.value === true);

const showCreateDialog = ref(false);
const creating = ref(false);
const searchQuery = ref("");
const filters = ref({
  status: undefined as SupportTicketStatus | undefined,
  category: undefined as SupportTicketCategory | undefined,
  priority: undefined as SupportTicketPriority | undefined,
});

const newTicket = ref({
  subject: "",
  description: "",
  category: SupportTicketCategory.TECHNICAL,
  priority: SupportTicketPriority.MEDIUM,
});

const canCreateTicket = computed(() => {
  return !creating.value && 
         newTicket.value.subject.trim() !== "" && 
         newTicket.value.description.trim() !== "";
});

const {
  data: tickets,
  pending,
  error,
  refresh: refreshTickets,
} = await useClientFetch<SupportTicket[]>(
  "support-tickets",
  async () => {
    try {
      const response = await client.listTickets({
        status: filters.value.status,
        category: filters.value.category,
        priority: filters.value.priority,
      });
      return response.tickets || [];
    } catch (err) {
      if (err instanceof ConnectError) {
        throw err;
      }
      throw new Error("Failed to load tickets");
    }
  },
  { watch: [filters] }
);

const statusOptions = [
  { label: "Open", value: SupportTicketStatus.OPEN },
  { label: "In Progress", value: SupportTicketStatus.IN_PROGRESS },
  { label: "Waiting for User", value: SupportTicketStatus.WAITING_FOR_USER },
  { label: "Resolved", value: SupportTicketStatus.RESOLVED },
  { label: "Closed", value: SupportTicketStatus.CLOSED },
];

const categoryOptions = [
  { label: "Technical", value: SupportTicketCategory.TECHNICAL },
  { label: "Billing", value: SupportTicketCategory.BILLING },
  { label: "Feature Request", value: SupportTicketCategory.FEATURE_REQUEST },
  { label: "Bug Report", value: SupportTicketCategory.BUG_REPORT },
  { label: "Account", value: SupportTicketCategory.ACCOUNT },
  { label: "Other", value: SupportTicketCategory.OTHER },
];

const priorityOptions = [
  { label: "Low", value: SupportTicketPriority.LOW },
  { label: "Medium", value: SupportTicketPriority.MEDIUM },
  { label: "High", value: SupportTicketPriority.HIGH },
  { label: "Urgent", value: SupportTicketPriority.URGENT },
];

function getStatusLabel(status: SupportTicketStatus): string {
  const option = statusOptions.find((opt) => opt.value === status);
  return option?.label || "Unknown";
}

function getStatusColor(status: SupportTicketStatus): string {
  switch (status) {
    case SupportTicketStatus.OPEN:
      return "primary";
    case SupportTicketStatus.IN_PROGRESS:
      return "warning";
    case SupportTicketStatus.WAITING_FOR_USER:
      return "warning";
    case SupportTicketStatus.RESOLVED:
      return "success";
    case SupportTicketStatus.CLOSED:
      return "secondary";
    default:
      return "secondary";
  }
}

function getPriorityLabel(priority: SupportTicketPriority): string {
  const option = priorityOptions.find((opt) => opt.value === priority);
  return option?.label || "Unknown";
}

function getPriorityColor(priority: SupportTicketPriority): string {
  switch (priority) {
    case SupportTicketPriority.LOW:
      return "primary";
    case SupportTicketPriority.MEDIUM:
      return "warning";
    case SupportTicketPriority.HIGH:
      return "danger";
    case SupportTicketPriority.URGENT:
      return "danger";
    default:
      return "secondary";
  }
}

function getCategoryLabel(category: SupportTicketCategory): string {
  const option = categoryOptions.find((opt) => opt.value === category);
  return option?.label || "Unknown";
}

function getCategoryVariant(category: SupportTicketCategory): string {
  switch (category) {
    case SupportTicketCategory.TECHNICAL:
      return "primary";
    case SupportTicketCategory.BILLING:
      return "warning";
    case SupportTicketCategory.FEATURE_REQUEST:
      return "primary";
    case SupportTicketCategory.BUG_REPORT:
      return "danger";
    case SupportTicketCategory.ACCOUNT:
      return "secondary";
    case SupportTicketCategory.OTHER:
      return "outline";
    default:
      return "secondary";
  }
}

const ticketStats = computed(() => {
  if (!tickets.value) return { open: 0, inProgress: 0, resolved: 0 };
  
  return {
    open: tickets.value.filter((t) => t.status === SupportTicketStatus.OPEN).length,
    inProgress: tickets.value.filter(
      (t) => t.status === SupportTicketStatus.IN_PROGRESS || t.status === SupportTicketStatus.WAITING_FOR_USER
    ).length,
    resolved: tickets.value.filter(
      (t) => t.status === SupportTicketStatus.RESOLVED || t.status === SupportTicketStatus.CLOSED
    ).length,
  };
});

const hasActiveFilters = computed(() => {
  return (
    filters.value.status !== undefined ||
    filters.value.category !== undefined ||
    filters.value.priority !== undefined ||
    searchQuery.value.trim() !== ""
  );
});

const filteredTickets = computed(() => {
  if (!tickets.value) return [];
  
  let filtered = [...tickets.value];
  
  // Apply search query
  if (searchQuery.value.trim()) {
    const query = searchQuery.value.toLowerCase().trim();
    filtered = filtered.filter(
      (ticket) =>
        ticket.subject.toLowerCase().includes(query) ||
        ticket.description.toLowerCase().includes(query)
    );
  }
  
  return filtered;
});

function handleSearch() {
  // Search is handled by computed property
  // This is here for potential debouncing if needed
}

function clearFilters() {
  filters.value = {
    status: undefined,
    category: undefined,
    priority: undefined,
  };
  searchQuery.value = "";
}

function navigateToTicket(ticketId: string) {
  router.push(`/support/${ticketId}`);
}

async function createTicket() {
  if (!newTicket.value.subject || !newTicket.value.description) {
    return;
  }

  creating.value = true;
  try {
    const response = await client.createTicket({
      subject: newTicket.value.subject,
      description: newTicket.value.description,
      category: newTicket.value.category,
      priority: newTicket.value.priority,
    });

    showCreateDialog.value = false;
    newTicket.value = {
      subject: "",
      description: "",
      category: SupportTicketCategory.TECHNICAL,
      priority: SupportTicketPriority.MEDIUM,
    };

    // Navigate to the new ticket
    if (response.ticket) {
      router.push(`/support/${response.ticket.id}`);
    } else {
      await refreshTickets();
    }
  } catch (err) {
    console.error("Failed to create ticket:", err);
    // Error handling can be improved with toast notifications
  } finally {
    creating.value = false;
  }
}

// Auto-refresh tickets list periodically
const visibility = useDocumentVisibility();
const isVisible = computed(() => visibility.value === "visible");

// Refresh interval: 30 seconds (less frequent than individual ticket page)
const REFRESH_INTERVAL_MS = 30000;

const refreshIntervalId = ref<ReturnType<typeof setInterval> | null>(null);

// Function to setup/restart the interval
const setupRefreshInterval = () => {
  // Clear existing interval if any
  if (refreshIntervalId.value) {
    clearInterval(refreshIntervalId.value);
    refreshIntervalId.value = null;
  }

  // Only setup if page is visible and no errors
  if (isVisible.value && !error.value) {
    refreshIntervalId.value = setInterval(async () => {
      if (isVisible.value && !error.value) {
        try {
          await refreshTickets();
        } catch (err) {
          console.error("Failed to auto-refresh tickets list:", err);
        }
      }
    }, REFRESH_INTERVAL_MS);
  }
};

// Watch for visibility changes
watch([isVisible, error], () => {
  setupRefreshInterval();
});

// Start refreshing when component is mounted
onMounted(() => {
  setupRefreshInterval();
});

// Cleanup on unmount
onUnmounted(() => {
  if (refreshIntervalId.value) {
    clearInterval(refreshIntervalId.value);
    refreshIntervalId.value = null;
  }
});
</script>

