<template>
  <OuiContainer size="full" py="xl">
    <OuiStack spacing="xl">
      <!-- Back Button -->
      <OuiButton
        variant="ghost"
        @click="router.push('/support')"
        class="self-start gap-2"
        size="sm"
      >
        <ArrowLeftIcon class="h-4 w-4" />
        Back to Tickets
      </OuiButton>

      <!-- Loading State -->
      <OuiCard v-if="pending">
        <OuiCardBody>
          <OuiStack align="center" gap="md" class="py-16">
            <OuiSpinner size="lg" />
            <OuiText color="secondary">Loading ticket...</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <!-- Error State -->
      <OuiCard v-else-if="error">
        <OuiCardBody>
          <div class="text-center py-16">
            <ExclamationCircleIcon class="h-12 w-12 text-danger mx-auto mb-4" />
            <OuiText color="danger" size="lg" weight="semibold" class="mb-2">
              Failed to load ticket
            </OuiText>
            <OuiText color="secondary" class="mb-4">{{ error }}</OuiText>
            <OuiButton @click="refreshTicket()" variant="outline" class="gap-2">
              <ArrowPathIcon class="h-4 w-4" />
              Try Again
            </OuiButton>
          </div>
        </OuiCardBody>
      </OuiCard>

      <!-- Ticket Content -->
      <template v-else-if="ticket">
        <OuiGrid cols="1" cols-lg="3" gap="xl">
          <!-- Main Content -->
          <OuiStack spacing="lg" class="lg:col-span-2">
            <!-- Ticket Header Card -->
            <OuiCard>
              <OuiCardHeader>
                <OuiStack spacing="md">
                  <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
                    <OuiStack spacing="xs" class="flex-1 min-w-0">
                      <OuiFlex gap="sm" align="center" wrap="wrap">
                        <OuiHeading size="2xl" class="line-clamp-2">
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
                      <OuiFlex gap="md" align="center" wrap="wrap">
                        <OuiFlex gap="sm" align="center" class="text-muted">
                          <ClockIcon class="h-4 w-4" />
                          <OuiText size="sm">
                            Created <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                          </OuiText>
                        </OuiFlex>
                        <OuiFlex v-if="ticket.resolvedAt" gap="sm" align="center" class="text-muted">
                          <CheckCircleIcon class="h-4 w-4" />
                          <OuiText size="sm">
                            Resolved <OuiRelativeTime :value="ticket.resolvedAt ? new Date(Number(ticket.resolvedAt.seconds) * 1000) : undefined" />
                          </OuiText>
                        </OuiFlex>
                      </OuiFlex>
                    </OuiStack>
                    <OuiFlex gap="xs" wrap="wrap" class="shrink-0">
                      <OuiBadge
                        :variant="getStatusColor(ticket.status) as any"
                        tone="soft"
                        size="md"
                      >
                        {{ getStatusLabel(ticket.status) }}
                      </OuiBadge>
                      <OuiBadge
                        :variant="getPriorityColor(ticket.priority) as any"
                        tone="soft"
                        size="md"
                      >
                        {{ getPriorityLabel(ticket.priority) }}
                      </OuiBadge>
                    </OuiFlex>
                  </OuiFlex>
                </OuiStack>
              </OuiCardHeader>
              <OuiCardBody>
                <OuiStack spacing="lg">
                  <div>
                    <OuiText size="sm" weight="semibold" color="secondary" class="mb-2">
                      Description
                    </OuiText>
                    <OuiText class="whitespace-pre-wrap">{{ ticket.description }}</OuiText>
                  </div>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Comments Section -->
            <OuiCard>
              <OuiCardHeader>
                <OuiFlex justify="between" align="center">
                  <OuiStack gap="xs">
                    <OuiHeading size="lg">Comments</OuiHeading>
                    <OuiText v-if="comments && comments.length > 0" size="sm" color="secondary">
                      {{ comments.length }} comment{{ comments.length !== 1 ? 's' : '' }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardHeader>
              <OuiCardBody>
                <OuiStack spacing="lg">
                  <!-- Comments List -->
                  <div v-if="commentsPending" class="text-center py-8">
                    <OuiSpinner />
                    <OuiText color="secondary" class="mt-2">Loading comments...</OuiText>
                  </div>
                  <div v-else-if="!comments?.length" class="text-center py-12">
                    <ChatBubbleLeftRightIcon class="h-12 w-12 text-muted mx-auto mb-4 opacity-50" />
                    <OuiText size="lg" weight="semibold" color="primary" class="mb-2">
                      No comments yet
                    </OuiText>
                    <OuiText color="secondary">
                      Be the first to add a comment
                    </OuiText>
                  </div>
                  <OuiStack v-else spacing="md">
                    <OuiCard
                      v-for="comment in comments"
                      :key="comment.id"
                      :variant="comment.internal ? 'outline' : 'default'"
                      :class="comment.internal ? 'border-warning/30 bg-warning/5' : ''"
                    >
                      <OuiCardBody>
                        <OuiStack spacing="sm">
                          <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
                            <OuiFlex gap="sm" align="center" wrap="wrap">
                              <OuiText size="xs" weight="medium" color="primary">
                                {{ comment.createdByName || comment.createdByEmail || comment.createdBy || 'Unknown User' }}
                              </OuiText>
                              <OuiBadge
                                v-if="comment.isSuperadmin"
                                variant="primary"
                                tone="soft"
                                size="xs"
                              >
                                Support Team
                              </OuiBadge>
                              <OuiBadge
                                v-if="comment.internal"
                                variant="warning"
                                tone="soft"
                                size="xs"
                              >
                                <LockClosedIcon class="h-3 w-3 mr-1" />
                                Internal
                              </OuiBadge>
                              <OuiText size="xs" weight="medium" color="secondary">
                                <OuiRelativeTime :value="comment.createdAt ? new Date(Number(comment.createdAt.seconds) * 1000) : undefined" />
                              </OuiText>
                            </OuiFlex>
                          </OuiFlex>
                          <OuiText class="whitespace-pre-wrap">{{ comment.content }}</OuiText>
                        </OuiStack>
                      </OuiCardBody>
                    </OuiCard>
                  </OuiStack>

                  <!-- Add Comment Form -->
                  <div class="border-t border-border-muted pt-6">
                    <OuiStack spacing="md">
                      <OuiText size="sm" weight="semibold" color="primary">
                        Add a comment
                      </OuiText>
                      <OuiTextarea
                        v-model="newComment"
                        placeholder="Type your comment here..."
                        :rows="5"
                        class="resize-none"
                      />
                      <OuiFlex v-if="isSuperAdmin" gap="sm" align="center">
                        <OuiCheckbox
                          v-model="commentInternal"
                          label="Internal comment (not visible to user)"
                        />
                      </OuiFlex>
                      <OuiFlex justify="end">
                        <OuiButton
                          @click="addComment"
                          color="primary"
                          :disabled="addingComment || !newComment.trim()"
                          :loading="addingComment"
                          class="gap-2"
                        >
                          <PaperAirplaneIcon class="h-4 w-4" />
                          {{ addingComment ? 'Adding...' : 'Add Comment' }}
                        </OuiButton>
                      </OuiFlex>
                    </OuiStack>
                  </div>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiStack>

          <!-- Sidebar -->
          <OuiStack spacing="lg" class="lg:col-span-1">
            <!-- Ticket Info Card -->
            <OuiCard>
              <OuiCardHeader>
                <OuiHeading size="md">Ticket Information</OuiHeading>
              </OuiCardHeader>
              <OuiCardBody>
                <OuiStack spacing="md">
                  <div>
                    <OuiText size="xs" weight="semibold" color="secondary" class="mb-1">
                      Status
                    </OuiText>
                    <div class="flex justify-center">
                      <OuiBadge
                        :variant="getStatusColor(ticket.status) as any"
                        tone="soft"
                        size="md"
                      >
                        {{ getStatusLabel(ticket.status) }}
                      </OuiBadge>
                    </div>
                  </div>
                  <div>
                    <OuiText size="xs" weight="semibold" color="secondary" class="mb-1">
                      Priority
                    </OuiText>
                    <div class="flex justify-center">
                      <OuiBadge
                        :variant="getPriorityColor(ticket.priority) as any"
                        tone="soft"
                        size="md"
                      >
                        {{ getPriorityLabel(ticket.priority) }}
                      </OuiBadge>
                    </div>
                  </div>
                  <div>
                    <OuiText size="xs" weight="semibold" color="secondary" class="mb-1">
                      Category
                    </OuiText>
                    <div class="flex justify-center">
                      <OuiBadge
                        :variant="getCategoryVariant(ticket.category) as any"
                        tone="soft"
                        size="md"
                      >
                        {{ getCategoryLabel(ticket.category) }}
                      </OuiBadge>
                    </div>
                  </div>
                  <div class="border-t border-border-muted pt-3">
                    <OuiText size="xs" weight="semibold" color="secondary" class="mb-2">
                      Details
                    </OuiText>
                    <OuiStack spacing="xs">
                      <OuiFlex justify="between" align="center">
                        <OuiText size="xs" color="secondary">Created</OuiText>
                        <OuiText size="xs" weight="medium">
                          <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                        </OuiText>
                      </OuiFlex>
                      <OuiFlex v-if="ticket.createdByName || ticket.createdByEmail" justify="between" align="center">
                        <OuiText size="xs" color="secondary">Created by</OuiText>
                        <OuiText size="xs" weight="medium">
                          {{ ticket.createdByName || ticket.createdByEmail || ticket.createdBy || 'Unknown User' }}
                        </OuiText>
                      </OuiFlex>
                      <OuiFlex v-if="ticket.assignedToName || ticket.assignedToEmail" justify="between" align="center">
                        <OuiText size="xs" color="secondary">Assigned to</OuiText>
                        <OuiText size="xs" weight="medium">
                          {{ ticket.assignedToName || ticket.assignedToEmail || ticket.assignedTo || 'Unassigned' }}
                        </OuiText>
                      </OuiFlex>
                      <OuiFlex v-if="ticket.resolvedAt" justify="between" align="center">
                        <OuiText size="xs" color="secondary">Resolved</OuiText>
                        <OuiText size="xs" weight="medium">
                          <OuiRelativeTime :value="ticket.resolvedAt ? new Date(Number(ticket.resolvedAt.seconds) * 1000) : undefined" />
                        </OuiText>
                      </OuiFlex>
                      <OuiFlex justify="between" align="center">
                        <OuiText size="xs" color="secondary">Comments</OuiText>
                        <OuiText size="xs" weight="medium">
                          {{ ticket.commentCount || 0 }}
                        </OuiText>
                      </OuiFlex>
                    </OuiStack>
                  </div>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Admin Controls -->
            <OuiCard v-if="isSuperAdmin">
              <OuiCardHeader>
                <OuiHeading size="md">Admin Controls</OuiHeading>
              </OuiCardHeader>
              <OuiCardBody>
                <OuiStack spacing="md">
                  <div>
                    <OuiText size="xs" weight="semibold" color="secondary" class="mb-2">
                      Update Status
                    </OuiText>
                    <OuiSelect
                      v-model="ticketUpdate.status"
                      :items="statusOptions"
                      placeholder="Select status"
                      @update:model-value="updateTicket"
                    />
                  </div>
                  <div>
                    <OuiText size="xs" weight="semibold" color="secondary" class="mb-2">
                      Update Priority
                    </OuiText>
                    <OuiSelect
                      v-model="ticketUpdate.priority"
                      :items="priorityOptions"
                      placeholder="Select priority"
                      @update:model-value="updateTicket"
                    />
                  </div>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
          </OuiStack>
        </OuiGrid>
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
import {
  ArrowLeftIcon,
  ArrowPathIcon,
  ClockIcon,
  CheckCircleIcon,
  ExclamationCircleIcon,
  ChatBubbleLeftRightIcon,
  LockClosedIcon,
  PaperAirplaneIcon,
} from "@heroicons/vue/24/outline";
import { ConnectError, Code } from "@connectrpc/connect";
import OuiRelativeTime from "~/components/oui/RelativeTime.vue";
import { useSuperAdmin } from "~/composables/useSuperAdmin";
import { useDocumentVisibility } from "@vueuse/core";

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
  const categoryOptions = [
    { label: "Technical", value: SupportTicketCategory.TECHNICAL },
    { label: "Billing", value: SupportTicketCategory.BILLING },
    { label: "Feature Request", value: SupportTicketCategory.FEATURE_REQUEST },
    { label: "Bug Report", value: SupportTicketCategory.BUG_REPORT },
    { label: "Account", value: SupportTicketCategory.ACCOUNT },
    { label: "Other", value: SupportTicketCategory.OTHER },
  ];
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

// Auto-refresh comments and ticket periodically
const visibility = useDocumentVisibility();
const isVisible = computed(() => visibility.value === "visible");

// Refresh interval: 15 seconds (reasonable for support tickets)
const REFRESH_INTERVAL_MS = 15000;

const refreshIntervalId = ref<ReturnType<typeof setInterval> | null>(null);

// Function to setup/restart the interval
const setupRefreshInterval = () => {
  // Clear existing interval if any
  if (refreshIntervalId.value) {
    clearInterval(refreshIntervalId.value);
    refreshIntervalId.value = null;
  }

  // Only setup if page is visible and we have a ticket loaded
  if (isVisible.value && ticket.value && !error.value) {
    refreshIntervalId.value = setInterval(async () => {
      if (isVisible.value && ticket.value && !error.value) {
        try {
          // Refresh both comments and ticket to get latest updates
          await refreshComments();
          await refreshTicket();
        } catch (err) {
          console.error("Failed to auto-refresh support ticket:", err);
        }
      }
    }, REFRESH_INTERVAL_MS);
  }
};

// Watch for visibility changes
watch([isVisible, ticket, error], () => {
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

