<template>
  <OuiContainer size="full" p="none">
    <OuiStack gap="lg">
      <!-- Back Button -->
      <OuiButton
        variant="ghost"
        size="sm"
        class="self-start gap-1.5"
        @click="router.push('/support')"
      >
        <ArrowLeftIcon class="h-3.5 w-3.5" />
        Back to Tickets
      </OuiButton>

      <!-- Loading State -->
      <OuiCard v-if="pending" variant="outline">
        <OuiCardBody>
          <OuiFlex align="center" justify="center" class="py-16" gap="md">
            <OuiSpinner size="lg" />
            <OuiText color="tertiary">Loading ticket…</OuiText>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <!-- Error State -->
      <ErrorAlert
        v-else-if="error"
        :error="error"
        title="Failed to load ticket"
        hint="The ticket may not exist or you may not have permission to view it."
      />

      <!-- Ticket Content -->
      <template v-else-if="ticket">
        <OuiGrid :cols="{ sm: 1, lg: 3 }" gap="lg">

          <!-- Main Column -->
          <OuiStack gap="md" class="lg:col-span-2">

            <!-- Ticket Header Card -->
            <OuiCard variant="outline">
              <OuiCardBody>
                <OuiStack gap="md">
                  <!-- Title + badges -->
                  <OuiFlex justify="between" align="start" gap="md" wrap="wrap">
                    <OuiStack gap="xs" class="flex-1 min-w-0">
                      <OuiText as="h1" size="lg" weight="semibold" class="leading-snug">{{ ticket.subject }}</OuiText>
                      <OuiFlex align="center" gap="sm" wrap="wrap">
                        <OuiText size="xs" color="tertiary">
                          Opened <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                        </OuiText>
                        <template v-if="ticket.createdByName || ticket.createdByEmail">
                          <span class="text-border-strong text-xs">·</span>
                          <OuiText size="xs" color="tertiary">{{ ticket.createdByName || ticket.createdByEmail }}</OuiText>
                        </template>
                        <template v-if="ticket.resolvedAt">
                          <span class="text-border-strong text-xs">·</span>
                          <OuiFlex align="center" gap="xs">
                            <CheckCircleIcon class="h-3.5 w-3.5 text-success" />
                            <OuiText size="xs" color="tertiary">
                              Resolved <OuiRelativeTime :value="new Date(Number(ticket.resolvedAt.seconds) * 1000)" />
                            </OuiText>
                          </OuiFlex>
                        </template>
                      </OuiFlex>
                    </OuiStack>
                    <OuiFlex gap="xs" wrap="wrap" class="shrink-0">
                      <OuiBadge :variant="getStatusColor(ticket.status) as any" size="sm">{{ getStatusLabel(ticket.status) }}</OuiBadge>
                      <OuiBadge :variant="getPriorityColor(ticket.priority) as any" size="sm">{{ getPriorityLabel(ticket.priority) }}</OuiBadge>
                      <OuiBadge :variant="getCategoryVariant(ticket.category) as any" size="sm">{{ getCategoryLabel(ticket.category) }}</OuiBadge>
                    </OuiFlex>
                  </OuiFlex>

                  <!-- Description -->
                  <div class="border-t border-border-muted pt-4">
                    <OuiText size="sm" color="tertiary" weight="medium" class="mb-2">Description</OuiText>
                    <OuiText size="sm" class="whitespace-pre-wrap leading-relaxed">{{ ticket.description }}</OuiText>
                  </div>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Comments Section -->
            <OuiCard variant="outline">
              <OuiCardBody>
                <OuiStack gap="md">
                  <UiSectionHeader :icon="ChatBubbleLeftRightIcon" color="primary" size="md">
                    Comments
                    <template v-if="comments && comments.length > 0">
                      <OuiBadge variant="secondary" size="xs" class="ml-1">{{ comments.length }}</OuiBadge>
                    </template>
                  </UiSectionHeader>

                  <!-- Loading -->
                  <OuiFlex v-if="commentsPending" align="center" gap="sm" class="py-6">
                    <OuiSpinner size="sm" />
                    <OuiText size="sm" color="tertiary">Loading comments…</OuiText>
                  </OuiFlex>

                  <!-- Empty comments -->
                  <div v-else-if="!comments?.length" class="text-center py-8">
                    <ChatBubbleLeftRightIcon class="h-10 w-10 text-tertiary mx-auto mb-3 opacity-40" />
                    <OuiText size="sm" color="tertiary">No comments yet — be the first to reply.</OuiText>
                  </div>

                  <!-- Comment thread -->
                  <OuiStack v-else gap="none" class="divide-y divide-border-muted">
                    <div
                      v-for="comment in comments"
                      :key="comment.id"
                      class="py-4"
                      :class="comment.internal ? 'bg-warning/5 -mx-4 px-4 rounded' : ''"
                    >
                      <OuiStack gap="xs">
                        <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
                          <OuiFlex align="center" gap="sm" wrap="wrap">
                            <OuiText size="xs" weight="semibold">
                              {{ comment.createdByName || comment.createdByEmail || comment.createdBy || 'Unknown' }}
                            </OuiText>
                            <OuiBadge v-if="comment.isSuperadmin" variant="primary" size="xs">Support Team</OuiBadge>
                            <OuiFlex v-if="comment.internal" align="center" gap="xs">
                              <LockClosedIcon class="h-3 w-3 text-warning" />
                              <OuiText size="xs" color="tertiary" class="text-warning">Internal</OuiText>
                            </OuiFlex>
                          </OuiFlex>
                          <OuiText size="xs" color="tertiary">
                            <OuiRelativeTime :value="comment.createdAt ? new Date(Number(comment.createdAt.seconds) * 1000) : undefined" />
                          </OuiText>
                        </OuiFlex>
                        <OuiText size="sm" class="whitespace-pre-wrap leading-relaxed">{{ comment.content }}</OuiText>
                      </OuiStack>
                    </div>
                  </OuiStack>

                  <!-- Add Comment -->
                  <div class="border-t border-border-muted pt-4">
                    <OuiStack gap="sm">
                      <OuiTextarea
                        v-model="newComment"
                        placeholder="Type your reply…"
                        :rows="4"
                        class="resize-none"
                      />
                      <OuiFlex justify="between" align="center">
                        <OuiCheckbox
                          v-if="isSuperAdmin"
                          v-model="commentInternal"
                          label="Internal note (not visible to user)"
                        />
                        <div v-else />
                        <OuiButton
                          color="primary"
                          size="sm"
                          class="gap-1.5"
                          :disabled="addingComment || !newComment.trim()"
                          :loading="addingComment"
                          @click="addComment"
                        >
                          <PaperAirplaneIcon class="h-3.5 w-3.5" />
                          {{ addingComment ? 'Sending…' : 'Send Reply' }}
                        </OuiButton>
                      </OuiFlex>
                    </OuiStack>
                  </div>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

          </OuiStack>

          <!-- Sidebar -->
          <OuiStack gap="md">

            <!-- Ticket Info -->
            <OuiCard variant="outline">
              <OuiCardBody>
                <OuiStack gap="md">
                  <UiSectionHeader :icon="InformationCircleIcon" color="secondary" size="md">Details</UiSectionHeader>
                  <OuiStack gap="none" class="divide-y divide-border-default">
                    <OuiFlex justify="between" align="center" class="py-2">
                      <OuiText size="xs" color="tertiary">Status</OuiText>
                      <OuiBadge :variant="getStatusColor(ticket.status) as any" size="xs">{{ getStatusLabel(ticket.status) }}</OuiBadge>
                    </OuiFlex>
                    <OuiFlex justify="between" align="center" class="py-2">
                      <OuiText size="xs" color="tertiary">Priority</OuiText>
                      <OuiBadge :variant="getPriorityColor(ticket.priority) as any" size="xs">{{ getPriorityLabel(ticket.priority) }}</OuiBadge>
                    </OuiFlex>
                    <OuiFlex justify="between" align="center" class="py-2">
                      <OuiText size="xs" color="tertiary">Category</OuiText>
                      <OuiBadge :variant="getCategoryVariant(ticket.category) as any" size="xs">{{ getCategoryLabel(ticket.category) }}</OuiBadge>
                    </OuiFlex>
                    <OuiFlex justify="between" align="start" class="py-2" gap="md">
                      <OuiText size="xs" color="tertiary" class="shrink-0">Opened</OuiText>
                      <OuiText size="xs" weight="medium" class="text-right">
                        <OuiRelativeTime :value="ticket.createdAt ? new Date(Number(ticket.createdAt.seconds) * 1000) : undefined" />
                      </OuiText>
                    </OuiFlex>
                    <OuiFlex v-if="ticket.createdByName || ticket.createdByEmail" justify="between" align="start" class="py-2" gap="md">
                      <OuiText size="xs" color="tertiary" class="shrink-0">Created by</OuiText>
                      <OuiText size="xs" weight="medium" class="text-right truncate">{{ ticket.createdByName || ticket.createdByEmail }}</OuiText>
                    </OuiFlex>
                    <OuiFlex v-if="ticket.assignedToName || ticket.assignedToEmail" justify="between" align="start" class="py-2" gap="md">
                      <OuiText size="xs" color="tertiary" class="shrink-0">Assigned to</OuiText>
                      <OuiText size="xs" weight="medium" class="text-right truncate">{{ ticket.assignedToName || ticket.assignedToEmail }}</OuiText>
                    </OuiFlex>
                    <OuiFlex v-if="ticket.resolvedAt" justify="between" align="start" class="py-2" gap="md">
                      <OuiText size="xs" color="tertiary" class="shrink-0">Resolved</OuiText>
                      <OuiText size="xs" weight="medium" class="text-right">
                        <OuiRelativeTime :value="new Date(Number(ticket.resolvedAt.seconds) * 1000)" />
                      </OuiText>
                    </OuiFlex>
                    <OuiFlex justify="between" align="center" class="py-2">
                      <OuiText size="xs" color="tertiary">Comments</OuiText>
                      <OuiText size="xs" weight="medium">{{ ticket.commentCount || 0 }}</OuiText>
                    </OuiFlex>
                  </OuiStack>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>

            <!-- Admin Controls -->
            <OuiCard v-if="isSuperAdmin" variant="outline">
              <OuiCardBody>
                <OuiStack gap="md">
                  <UiSectionHeader :icon="WrenchScrewdriverIcon" color="warning" size="md">Admin Controls</UiSectionHeader>
                  <OuiSelect
                    v-model="adminStatus"
                    label="Status"
                    :items="statusOptions"
                    placeholder="Select status"
                    @update:model-value="(v) => updateTicket(v, ticket!.priority)"
                  />
                  <OuiSelect
                    v-model="adminPriority"
                    label="Priority"
                    :items="priorityOptions"
                    placeholder="Select priority"
                    @update:model-value="(v) => updateTicket(ticket!.status, v)"
                  />
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
  CheckCircleIcon,
  ChatBubbleLeftRightIcon,
  LockClosedIcon,
  PaperAirplaneIcon,
  InformationCircleIcon,
  WrenchScrewdriverIcon,
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
} = await useClientFetch<SupportTicket>(
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
} = await useClientFetch<TicketComment[]>(
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

// Plain refs for admin controls (initialized from ticket on load)
const adminStatus = ref<SupportTicketStatus | undefined>(undefined);
const adminPriority = ref<SupportTicketPriority | undefined>(undefined);

watch(ticket, (t) => {
  if (t) {
    adminStatus.value = t.status;
    adminPriority.value = t.priority;
  }
}, { immediate: true });

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

