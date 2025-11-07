<template>
  <OuiStack gap="xl">
    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
      <OuiStack gap="xs">
        <OuiText tag="h1" size="3xl" weight="extrabold">Invitations</OuiText>
        <OuiText color="muted">Accept or decline invitations to join organizations.</OuiText>
      </OuiStack>
      <OuiButton variant="ghost" size="sm" @click="refresh" :disabled="isLoading">
        <span class="flex items-center gap-2">
          <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': isLoading }" />
          Refresh
        </span>
      </OuiButton>
    </OuiFlex>

    <OuiCard v-if="isLoading" class="border border-border-muted rounded-xl">
      <OuiCardBody class="p-8 text-center">
        <OuiText color="muted">Loading invitations...</OuiText>
      </OuiCardBody>
    </OuiCard>

    <OuiCard v-else-if="invites.length === 0" class="border border-border-muted rounded-xl">
      <OuiCardBody class="p-8 text-center">
        <OuiText color="muted">You have no pending invitations.</OuiText>
      </OuiCardBody>
    </OuiCard>

    <OuiStack v-else gap="md">
      <OuiCard
        v-for="invite in invites"
        :key="invite.id"
        class="border border-border-muted rounded-xl"
      >
        <OuiCardBody>
          <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
            <OuiStack gap="xs" class="flex-1 min-w-0">
              <OuiText size="lg" weight="semibold">{{ invite.organizationName }}</OuiText>
              <OuiText color="muted" size="sm">
                You've been invited to join as <span class="uppercase font-medium">{{ invite.role }}</span>
              </OuiText>
              <OuiText color="muted" size="xs">
                Invited {{ formatDate(invite.invitedAt) }}
              </OuiText>
            </OuiStack>
            <OuiFlex gap="sm" wrap="wrap">
              <OuiButton
                variant="ghost"
                color="danger"
                @click="declineInvite(invite)"
                :disabled="processingInvite === invite.id"
              >
                {{ processingInvite === invite.id && declining ? 'Declining...' : 'Decline' }}
              </OuiButton>
              <OuiButton
                @click="acceptInvite(invite)"
                :disabled="processingInvite === invite.id"
              >
                {{ processingInvite === invite.id && !declining ? 'Accepting...' : 'Accept' }}
              </OuiButton>
            </OuiFlex>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>
    </OuiStack>
  </OuiStack>
</template>

<script setup lang="ts">
import { ArrowPathIcon } from "@heroicons/vue/24/outline";
import { computed, ref, onMounted } from "vue";
import { OrganizationService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import { useAuth } from "~/composables/useAuth";
import { useRouter } from "vue-router";

definePageMeta({
  middleware: ["auth"],
});

const orgClient = useConnectClient(OrganizationService);
const { toast } = useToast();
const auth = useAuth();
const router = useRouter();

const invites = ref<any[]>([]);
const isLoading = ref(false);
const processingInvite = ref<string | null>(null);
const declining = ref(false);

const dateFormatter = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });
function formatDate(timestamp?: { seconds?: number | bigint; nanos?: number } | null) {
  if (!timestamp || timestamp.seconds === undefined) return "—";
  const seconds = typeof timestamp.seconds === "bigint" ? Number(timestamp.seconds) : timestamp.seconds;
  const millis = seconds * 1000 + Math.floor((timestamp.nanos ?? 0) / 1_000_000);
  const date = new Date(millis);
  return Number.isNaN(date.getTime()) ? "—" : dateFormatter.format(date);
}

async function refresh() {
  isLoading.value = true;
  try {
    const res = await orgClient.listMyInvites({});
    invites.value = res.invites || [];
  } catch (error: any) {
    toast.error(error?.message || "Failed to load invitations");
  } finally {
    isLoading.value = false;
  }
}

async function acceptInvite(invite: any) {
  if (processingInvite.value === invite.id) return;
  processingInvite.value = invite.id;
  declining.value = false;
  
  try {
    const res = await orgClient.acceptInvite({
      organizationId: invite.organizationId,
      memberId: invite.id,
    });
    
    toast.success(`You've joined ${res.organization?.name || 'the organization'}!`);
    
    // Refresh user's organizations list
    const orgRes = await orgClient.listOrganizations({ onlyMine: true });
    auth.setOrganizations(orgRes.organizations || []);
    auth.notifyOrganizationsUpdated();
    
    // Switch to the new organization
    if (res.organization?.id) {
      await auth.switchOrganization(res.organization.id);
    }
    
    // Remove from list
    invites.value = invites.value.filter(i => i.id !== invite.id);
    
    // Navigate to dashboard or organizations page
    router.push('/dashboard');
  } catch (error: any) {
    toast.error(error?.message || "Failed to accept invitation");
  } finally {
    processingInvite.value = null;
  }
}

async function declineInvite(invite: any) {
  if (processingInvite.value === invite.id) return;
  processingInvite.value = invite.id;
  declining.value = true;
  
  try {
    await orgClient.declineInvite({
      organizationId: invite.organizationId,
      memberId: invite.id,
    });
    
    toast.success("Invitation declined");
    
    // Remove from list
    invites.value = invites.value.filter(i => i.id !== invite.id);
  } catch (error: any) {
    toast.error(error?.message || "Failed to decline invitation");
  } finally {
    processingInvite.value = null;
    declining.value = false;
  }
}

onMounted(() => {
  refresh();
});
</script>

