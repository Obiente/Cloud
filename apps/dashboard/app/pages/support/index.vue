<template>
  <OuiContainer size="full" py="xl">
    <OuiStack spacing="xl">
      <!-- Page Header -->
      <OuiFlex justify="between" align="center" wrap="wrap" gap="lg">
        <OuiStack gap="xs" class="flex-1">
          <OuiText as="h1" size="3xl" weight="bold" color="primary"
            >Support Desk</OuiText
          >
          <OuiText color="secondary"
            >Manage and track your support tickets</OuiText
          >
        </OuiStack>
        <OuiButton @click="showCreateDialog = true" color="primary" size="lg" class="gap-2 shadow-md">
          <PlusIcon class="h-5 w-5" />
          New Ticket
        </OuiButton>
      </OuiFlex>

      <!-- Stats Overview -->
      <OuiGrid cols="1" cols-sm="2" cols-md="4" gap="md" v-if="!pending && tickets">
        <OuiCard hoverable class="transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg">
          <OuiCardBody>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText size="sm" color="secondary">Total Tickets</OuiText>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ tickets.length }}
                </OuiText>
              </OuiStack>
              <OuiBox p="md" rounded="lg" class="bg-primary/10 flex items-center justify-center">
                <DocumentTextIcon class="h-6 w-6 text-primary" />
              </OuiBox>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
        <OuiCard hoverable class="transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg">
          <OuiCardBody>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText size="sm" color="secondary">Open</OuiText>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ ticketStats.open }}
                </OuiText>
              </OuiStack>
              <OuiBox p="md" rounded="lg" class="bg-info/10 flex items-center justify-center">
                <ClockIcon class="h-6 w-6 text-info" />
              </OuiBox>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
        <OuiCard hoverable class="transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg">
          <OuiCardBody>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText size="sm" color="secondary">In Progress</OuiText>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ ticketStats.inProgress }}
                </OuiText>
              </OuiStack>
              <OuiBox p="md" rounded="lg" class="bg-warning/10 flex items-center justify-center">
                <ArrowPathIcon class="h-6 w-6 text-warning" />
              </OuiBox>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
        <OuiCard hoverable class="transition-all duration-200 hover:-translate-y-0.5 hover:shadow-lg">
          <OuiCardBody>
            <OuiFlex align="center" justify="between">
              <OuiStack gap="xs">
                <OuiText size="sm" color="secondary">Resolved</OuiText>
                <OuiText size="2xl" weight="bold" color="primary">
                  {{ ticketStats.resolved }}
                </OuiText>
              </OuiStack>
              <OuiBox p="md" rounded="lg" class="bg-success/10 flex items-center justify-center">
                <CheckCircleIcon class="h-6 w-6 text-success" />
              </OuiBox>
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </OuiGrid>

      <!-- Main Content Card -->
      <OuiCard>
        <OuiCardHeader>
          <OuiStack spacing="md">
            <!-- Search and Filters -->
            <OuiFlex gap="md" wrap="wrap" align="end">
              <OuiInput
                v-model="searchQuery"
                placeholder="Search tickets..."
                class="flex-1 min-w-[200px]"
                @update:model-value="handleSearch"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-5 w-5 text-text-secondary" />
                </template>
              </OuiInput>
              <OuiSelect
                v-model="filters.status"
                label="Status"
                :items="statusOptions"
                clearable
                @update:model-value="refreshTickets"
                class="min-w-[140px]"
              />
              <OuiSelect
                v-model="filters.category"
                label="Category"
                :items="categoryOptions"
                clearable
                @update:model-value="refreshTickets"
                class="min-w-[140px]"
              />
              <OuiSelect
                v-model="filters.priority"
                label="Priority"
                :items="priorityOptions"
                clearable
                @update:model-value="refreshTickets"
                class="min-w-[140px]"
              />
              <OuiButton
                v-if="hasActiveFilters"
                variant="ghost"
                size="sm"
                @click="clearFilters"
                class="gap-2"
              >
                <XMarkIcon class="h-4 w-4" />
                Clear
              </OuiButton>
            </OuiFlex>
          </OuiStack>
        </OuiCardHeader>
        <OuiCardBody>
          <!-- Loading State -->
          <div v-if="pending" class="text-center py-16">
            <OuiSpinner size="lg" />
            <OuiText color="secondary" class="mt-4">Loading tickets...</OuiText>
          </div>

          <!-- Error State -->
          <div v-else-if="error" class="text-center py-16">
            <ExclamationCircleIcon class="h-12 w-12 text-danger mx-auto mb-4" />
            <OuiText color="danger" size="lg" weight="semibold" class="mb-2">
              Failed to load tickets
            </OuiText>
            <OuiText color="secondary" class="mb-4">{{ error }}</OuiText>
            <OuiButton @click="refreshTickets()" variant="outline" class="gap-2">
              <ArrowPathIcon class="h-4 w-4" />
              Try Again
            </OuiButton>
          </div>

          <!-- Empty State -->
          <div v-else-if="!filteredTickets?.length" class="text-center py-16">
            <InboxIcon class="h-16 w-16 text-muted mx-auto mb-4 opacity-50" />
            <OuiText size="lg" weight="semibold" color="primary" class="mb-2">
              {{ hasActiveFilters ? 'No tickets match your filters' : 'No tickets yet' }}
            </OuiText>
            <OuiText color="secondary" class="mb-6 max-w-md mx-auto">
              {{ hasActiveFilters ? 'Try adjusting your filters or search query' : 'Create your first support ticket to get started' }}
            </OuiText>
            <OuiButton
              v-if="hasActiveFilters"
              @click="clearFilters"
              variant="outline"
              class="gap-2 mr-2"
            >
              <XMarkIcon class="h-4 w-4" />
              Clear Filters
            </OuiButton>
            <OuiButton
              v-if="!hasActiveFilters"
              @click="showCreateDialog = true"
              color="primary"
              class="gap-2"
            >
              <PlusIcon class="h-5 w-5" />
              Create Ticket
            </OuiButton>
          </div>

          <!-- Tickets List -->
          <OuiStack v-else spacing="sm">
            <OuiCard
              v-for="ticket in filteredTickets"
              :key="ticket.id"
              interactive
              hoverable
              class="transition-all duration-200"
              @click="navigateToTicket(ticket.id)"
            >
              <OuiCardBody>
                <OuiStack spacing="md">
                  <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
                    <OuiStack spacing="xs" class="flex-1 min-w-0">
                      <OuiFlex gap="sm" align="center" wrap="wrap">
                        <OuiHeading size="lg" class="line-clamp-2">
                          {{ ticket.subject }}
                        </OuiHeading>
                        <OuiBadge
                          :variant="getCategoryVariant(ticket.category) as any"
                          tone="soft"
                          size="sm"
                          class="shrink-0"
                        >
                          {{ getCategoryLabel(ticket.category) }}
                        </OuiBadge>
                      </OuiFlex>
                      <OuiText color="muted" size="sm" class="line-clamp-2">
                        {{ ticket.description }}
                      </OuiText>
                    </OuiStack>
                    <OuiFlex gap="xs" wrap="wrap" class="shrink-0">
                      <OuiBadge
                        :variant="getStatusColor(ticket.status) as any"
                        tone="soft"
                        size="sm"
                      >
                        {{ getStatusLabel(ticket.status) }}
                      </OuiBadge>
                      <OuiBadge
                        :variant="getPriorityColor(ticket.priority) as any"
                        tone="soft"
                        size="sm"
                      >
                        {{ getPriorityLabel(ticket.priority) }}
                      </OuiBadge>
                    </OuiFlex>
                  </OuiFlex>

                  <OuiFlex gap="md" align="center" wrap="wrap" class="pt-2 border-t border-border-muted">
                    <OuiFlex gap="sm" align="center" class="text-muted">
                      <ClockIcon class="h-4 w-4" />
                      <OuiText size="xs">
                        <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                      </OuiText>
                    </OuiFlex>
                    <OuiFlex v-if="ticket.commentCount > 0" gap="sm" align="center" class="text-muted">
                      <ChatBubbleLeftRightIcon class="h-4 w-4" />
                      <OuiText size="xs">
                        {{ ticket.commentCount }} comment{{ ticket.commentCount !== 1 ? 's' : '' }}
                      </OuiText>
                    </OuiFlex>
                    <OuiFlex gap="sm" align="center" class="text-muted ml-auto">
                      <ArrowRightIcon class="h-4 w-4" />
                      <OuiText size="xs" weight="medium">View Details</OuiText>
                    </OuiFlex>
                  </OuiFlex>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>

    <!-- Create Ticket Dialog -->
    <OuiDialog v-model:open="showCreateDialog" title="Create Support Ticket">
      <OuiStack spacing="md">
        <OuiInput
          v-model="newTicket.subject"
          label="Subject"
          placeholder="Brief description of your issue"
          required
        />
        <OuiTextarea
          v-model="newTicket.description"
          label="Description"
          placeholder="Please provide details about your issue..."
          :rows="6"
          required
        />
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
        <OuiFlex justify="end" gap="md">
          <OuiButton @click="showCreateDialog = false" variant="ghost">
            Cancel
          </OuiButton>
          <OuiButton
            @click="createTicket"
            color="primary"
            :disabled="!canCreateTicket"
          >
            {{ creating ? 'Creating...' : 'Create Ticket' }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
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
  ClockIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  InboxIcon,
  ChatBubbleLeftRightIcon,
  ArrowRightIcon,
  DocumentTextIcon,
} from "@heroicons/vue/24/outline";
import { ConnectError, Code } from "@connectrpc/connect";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import { useSuperAdmin } from "~/composables/useSuperAdmin";

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
} = await useAsyncData<SupportTicket[]>(
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
</script>

