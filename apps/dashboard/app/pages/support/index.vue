<template>
  <OuiContainer>
    <OuiStack spacing="lg">
      <OuiCard>
        <OuiCardHeader>
          <OuiFlex justify="between" align="center">
            <OuiHeading size="xl">Support Desk</OuiHeading>
            <OuiButton @click="showCreateDialog = true" color="primary">
              <PlusIcon class="w-5 h-5 mr-2" />
              New Ticket
            </OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack spacing="md">
            <!-- Filters -->
            <OuiFlex gap="md" wrap="wrap">
              <OuiSelect
                v-model="filters.status"
                label="Status"
                :items="statusOptions"
                clearable
                @update:model-value="refreshTickets"
              />
              <OuiSelect
                v-model="filters.category"
                label="Category"
                :items="categoryOptions"
                clearable
                @update:model-value="refreshTickets"
              />
              <OuiSelect
                v-model="filters.priority"
                label="Priority"
                :items="priorityOptions"
                clearable
                @update:model-value="refreshTickets"
              />
            </OuiFlex>

            <!-- Tickets List -->
            <div v-if="pending" class="text-center py-8">
              <OuiSpinner size="lg" />
            </div>
            <div v-else-if="error" class="text-center py-8">
              <OuiText color="danger">{{ error }}</OuiText>
            </div>
            <div v-else-if="!tickets?.length" class="text-center py-8">
              <OuiText color="muted">No tickets found</OuiText>
            </div>
            <OuiStack v-else spacing="sm">
              <OuiCard
                v-for="ticket in tickets"
                :key="ticket.id"
                class="cursor-pointer hover:shadow-md transition-shadow"
                @click="navigateToTicket(ticket.id)"
              >
                <OuiCardBody>
                  <OuiStack spacing="xs">
                    <OuiFlex justify="between" align="start">
                      <OuiHeading size="md">{{ ticket.subject }}</OuiHeading>
                      <OuiFlex gap="xs">
                        <OuiBadge
                          :color="getStatusColor(ticket.status)"
                          size="sm"
                        >
                          {{ getStatusLabel(ticket.status) }}
                        </OuiBadge>
                        <OuiBadge
                          :color="getPriorityColor(ticket.priority)"
                          size="sm"
                        >
                          {{ getPriorityLabel(ticket.priority) }}
                        </OuiBadge>
                      </OuiFlex>
                    </OuiFlex>
                    <OuiText color="muted" size="sm">
                      {{ ticket.description }}
                    </OuiText>
                    <OuiFlex gap="md" align="center">
                      <OuiText color="muted" size="xs">
                        <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                      </OuiText>
                      <OuiText v-if="ticket.commentCount > 0" color="muted" size="xs">
                        {{ ticket.commentCount }} comment{{ ticket.commentCount !== 1 ? 's' : '' }}
                      </OuiText>
                    </OuiFlex>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>
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
import { PlusIcon } from "@heroicons/vue/24/outline";
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
      return "info";
    case SupportTicketStatus.IN_PROGRESS:
      return "warning";
    case SupportTicketStatus.WAITING_FOR_USER:
      return "warning";
    case SupportTicketStatus.RESOLVED:
      return "success";
    case SupportTicketStatus.CLOSED:
      return "muted";
    default:
      return "muted";
  }
}

function getPriorityLabel(priority: SupportTicketPriority): string {
  const option = priorityOptions.find((opt) => opt.value === priority);
  return option?.label || "Unknown";
}

function getPriorityColor(priority: SupportTicketPriority): string {
  switch (priority) {
    case SupportTicketPriority.LOW:
      return "info";
    case SupportTicketPriority.MEDIUM:
      return "warning";
    case SupportTicketPriority.HIGH:
      return "danger";
    case SupportTicketPriority.URGENT:
      return "danger";
    default:
      return "muted";
  }
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

