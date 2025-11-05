<template>
  <OuiContainer>
    <OuiStack spacing="lg">
      <OuiButton
        variant="ghost"
        @click="router.back()"
        class="self-start"
      >
        ← Back to Tickets
      </OuiButton>

      <OuiCard v-if="pending">
        <OuiCardBody>
          <div class="text-center py-8">
            <OuiSpinner size="lg" />
          </div>
        </OuiCardBody>
      </OuiCard>

      <OuiCard v-else-if="error">
        <OuiCardBody>
          <div class="text-center py-8">
            <OuiText color="danger">{{ error }}</OuiText>
          </div>
        </OuiCardBody>
      </OuiCard>

      <template v-else-if="ticket">
        <!-- Ticket Header -->
        <OuiCard>
          <OuiCardHeader>
            <OuiStack spacing="md">
              <OuiFlex justify="between" align="start">
                <OuiHeading size="xl">{{ ticket.subject }}</OuiHeading>
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
              <OuiFlex gap="md" align="center">
                <OuiText color="muted" size="sm">
                  Created <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                </OuiText>
                <OuiText v-if="ticket.resolvedAt" color="muted" size="sm">
                  Resolved <OuiRelativeTime :value="ticket.resolvedAt ? new Date(Number(ticket.resolvedAt.seconds) * 1000) : undefined" />
                </OuiText>
              </OuiFlex>
            </OuiStack>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiStack spacing="md">
              <div>
                <OuiText size="sm" color="muted" class="mb-2">Description</OuiText>
                <OuiText>{{ ticket.description }}</OuiText>
              </div>

              <!-- Superadmin Controls -->
              <div v-if="isSuperAdmin" class="border-t pt-4">
                <OuiStack spacing="md">
                  <OuiHeading size="sm">Admin Controls</OuiHeading>
                  <OuiFlex gap="md" wrap="wrap">
                    <OuiSelect
                      v-model="ticketUpdate.status"
                      label="Status"
                      :items="statusOptions"
                      @update:model-value="updateTicket"
                    />
                    <OuiSelect
                      v-model="ticketUpdate.priority"
                      label="Priority"
                      :items="priorityOptions"
                      @update:model-value="updateTicket"
                    />
                  </OuiFlex>
                </OuiStack>
              </div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Comments Section -->
        <OuiCard>
          <OuiCardHeader>
            <OuiHeading size="lg">Comments</OuiHeading>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiStack spacing="md">
              <!-- Comments List -->
              <div v-if="commentsPending" class="text-center py-4">
                <OuiSpinner />
              </div>
              <div v-else-if="!comments?.length" class="text-center py-4">
                <OuiText color="muted">No comments yet</OuiText>
              </div>
              <OuiStack v-else spacing="md">
                <OuiCard
                  v-for="comment in comments"
                  :key="comment.id"
                  :class="comment.internal ? 'bg-warning-50 border-warning-200' : ''"
                >
                  <OuiCardBody>
                    <OuiStack spacing="xs">
                      <OuiFlex justify="between" align="center">
                        <OuiText size="sm" color="muted">
                          <OuiRelativeTime :value="comment.createdAt ? new Date(Number(comment.createdAt.seconds) * 1000) : undefined" />
                          <span v-if="comment.internal" class="ml-2">
                            (Internal)
                          </span>
                        </OuiText>
                      </OuiFlex>
                      <OuiText>{{ comment.content }}</OuiText>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </OuiStack>

              <!-- Add Comment Form -->
              <div class="border-t pt-4">
                <OuiStack spacing="md">
                  <OuiTextarea
                    v-model="newComment"
                    label="Add a comment"
                    placeholder="Type your comment here..."
                    :rows="4"
                  />
                  <OuiFlex v-if="isSuperAdmin" gap="sm">
                    <OuiCheckbox
                      v-model="commentInternal"
                      label="Internal comment (not visible to user)"
                    />
                  </OuiFlex>
                  <OuiButton
                    @click="addComment"
                    color="primary"
                    :disabled="addingComment || !newComment.trim()"
                  >
                    {{ addingComment ? 'Adding...' : 'Add Comment' }}
                  </OuiButton>
                </OuiStack>
              </div>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </template>
    </OuiStack>
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
  type TicketComment,
} from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { ConnectError, Code } from "@connectrpc/connect";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import { useSuperAdmin } from "~/composables/useSuperAdmin";

const route = useRoute();
const router = useRouter();
const auth = useAuth();
const ticketId = computed(() => String(route.params.id));

const superAdmin = useSuperAdmin();
const isSuperAdmin = computed(() => superAdmin.allowed.value === true);

const client = useConnectClient(SupportService);

const {
  data: ticket,
  pending,
  error,
  refresh: refreshTicket,
} = await useAsyncData<SupportTicket>(
  () => `support-ticket-${ticketId.value}`,
  async () => {
    try {
      const response = await client.getTicket({
        ticketId: ticketId.value,
      });
      return response.ticket!;
    } catch (err) {
      if (err instanceof ConnectError) {
        if (err.code === Code.NotFound || err.code === Code.PermissionDenied) {
          throw err;
        }
      }
      throw new Error("Failed to load ticket");
    }
  },
  { watch: [ticketId] }
);

const {
  data: comments,
  pending: commentsPending,
  refresh: refreshComments,
} = await useAsyncData<TicketComment[]>(
  () => `support-ticket-comments-${ticketId.value}`,
  async () => {
    if (!ticket.value) return [];
    try {
      const response = await client.listComments({
        ticketId: ticketId.value,
      });
      return response.comments || [];
    } catch (err) {
      console.error("Failed to load comments:", err);
      return [];
    }
  },
  { watch: [ticketId, ticket] }
);

const newComment = ref("");
const commentInternal = ref(false);
const addingComment = ref(false);

const ticketUpdate = ref({
  status: computed({
    get: () => ticket.value?.status,
    set: async (value) => {
      if (value !== undefined && ticket.value) {
        await updateTicket(value, ticket.value.priority);
      }
    },
  }),
  priority: computed({
    get: () => ticket.value?.priority,
    set: async (value) => {
      if (value !== undefined && ticket.value) {
        await updateTicket(ticket.value.status, value);
      }
    },
  }),
});

const statusOptions = [
  { label: "Open", value: SupportTicketStatus.OPEN },
  { label: "In Progress", value: SupportTicketStatus.IN_PROGRESS },
  { label: "Waiting for User", value: SupportTicketStatus.WAITING_FOR_USER },
  { label: "Resolved", value: SupportTicketStatus.RESOLVED },
  { label: "Closed", value: SupportTicketStatus.CLOSED },
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

async function updateTicket(
  status?: SupportTicketStatus,
  priority?: SupportTicketPriority
) {
  if (!ticket.value || !isSuperAdmin.value) return;

  try {
    await client.updateTicket({
      ticketId: ticketId.value,
      status: status !== undefined ? status : undefined,
      priority: priority !== undefined ? priority : undefined,
    });
    await refreshTicket();
  } catch (err) {
    console.error("Failed to update ticket:", err);
  }
}

async function addComment() {
  if (!newComment.value.trim() || !ticket.value) return;

  addingComment.value = true;
  try {
    await client.addComment({
      ticketId: ticketId.value,
      content: newComment.value,
      internal: isSuperAdmin.value ? commentInternal.value : false,
    });

    newComment.value = "";
    commentInternal.value = false;
    await refreshComments();
    await refreshTicket(); // Refresh to update comment count
  } catch (err) {
    console.error("Failed to add comment:", err);
  } finally {
    addingComment.value = false;
  }
}
</script>

