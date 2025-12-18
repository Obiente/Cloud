<template>
  <OuiCard>
    <OuiCardHeader>
      <OuiStack gap="xs">
        <OuiText as="h2" class="oui-card-title">Network Leases</OuiText>
        <OuiText color="secondary" size="sm">
          View DHCP leases and MAC addresses for this VPS instance
        </OuiText>
      </OuiStack>
    </OuiCardHeader>
    <OuiCardBody>
      <OuiStack gap="md">
        <!-- Loading State -->
        <div v-if="loading" class="space-y-3">
          <OuiBox
            v-for="i in 2"
            :key="i"
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="sm">
              <OuiFlex justify="between" align="start">
                <OuiStack gap="xs" class="flex-1">
                  <OuiSkeleton width="12rem" height="1rem" variant="text" />
                  <OuiSkeleton width="16rem" height="0.875rem" variant="text" />
                </OuiStack>
                <OuiSkeleton width="6rem" height="1.25rem" variant="rectangle" rounded />
              </OuiFlex>
            </OuiStack>
          </OuiBox>
        </div>

        <!-- Error State -->
        <div v-else-if="error" class="py-4">
          <OuiBox variant="danger" p="sm" rounded="md">
            <OuiText color="danger" size="sm" class="text-center">
              Failed to load leases: {{ error }}
            </OuiText>
          </OuiBox>
        </div>

        <!-- Empty State -->
        <div v-else-if="leases.length === 0" class="py-8">
          <OuiStack gap="md" align="center">
            <OuiText color="secondary" class="text-center">
              No active DHCP leases assigned to this VPS instance.
            </OuiText>
          </OuiStack>
        </div>

        <!-- Leases List -->
        <div v-else class="space-y-3">
          <OuiBox
            v-for="lease in leases"
            :key="`${lease.ipAddress}-${lease.macAddress ?? ''}`"
            p="md"
            rounded="lg"
            class="bg-surface-muted/40 ring-1 ring-border-muted"
          >
            <OuiStack gap="md">
              <!-- Main lease info -->
              <OuiFlex justify="between" align="start" gap="md">
                <OuiStack gap="sm" class="flex-1">
                  <!-- IP Address -->
                  <OuiStack gap="xs">
                    <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">
                      IP Address
                    </OuiText>
                    <OuiFlex align="center" gap="sm">
                      <OuiText size="sm" weight="semibold" class="font-mono">
                        {{ lease.ipAddress }}
                      </OuiText>
                      <OuiButton
                        variant="ghost"
                        size="xs"
                        icon-only
                        @click="copyToClipboard(lease.ipAddress)"
                        title="Copy IP address"
                      >
                        <ClipboardDocumentListIcon class="h-3 w-3" />
                      </OuiButton>
                    </OuiFlex>
                  </OuiStack>

                  <!-- MAC Address -->
                  <OuiStack gap="xs">
                    <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">
                      MAC Address
                    </OuiText>
                    <OuiFlex align="center" gap="sm">
                      <OuiText size="sm" class="font-mono">{{ lease.macAddress ?? '—' }}</OuiText>
                      <OuiButton
                        v-if="lease.macAddress"
                        variant="ghost"
                        size="xs"
                        icon-only
                        @click="copyToClipboard(lease.macAddress)"
                        title="Copy MAC address"
                      >
                        <ClipboardDocumentListIcon class="h-3 w-3" />
                      </OuiButton>
                    </OuiFlex>
                  </OuiStack>
                </OuiStack>

                <!-- Badge -->
                <OuiBadge
                  :variant="lease.isPublic ? 'warning' : 'primary'"
                  size="sm"
                >
                  {{ lease.isPublic ? 'Public IP' : 'DHCP Pool' }}
                </OuiBadge>
              </OuiFlex>

              <!-- Expiry Information -->
              <OuiStack gap="xs" v-if="lease.expiresAt">
                <OuiText size="xs" color="muted" weight="semibold" transform="uppercase">
                  Lease Expiration
                </OuiText>
                <OuiText size="sm" color="secondary">
                  <OuiRelativeTime
                    :value="date(lease.expiresAt)"
                    :style="'short'"
                  />
                  <span class="text-xs ml-1 text-muted">
                    (<OuiDate :value="lease.expiresAt" />)
                  </span>
                </OuiText>
              </OuiStack>
            </OuiStack>
          </OuiBox>
        </div>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { ClipboardDocumentListIcon } from '@heroicons/vue/24/outline';
import { useToast } from '~/composables/useToast';
import { date } from '@obiente/proto/utils';
import type { VPSLease } from '@obiente/proto';
import OuiSkeleton from '~/components/oui/Skeleton.vue';
import OuiRelativeTime from '~/components/oui/RelativeTime.vue';
import OuiDate from '~/components/oui/Date.vue';

interface Props {
  leases: VPSLease[];
  loading?: boolean;
  error?: string | null;
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  error: null,
});

const { toast } = useToast();

const copyToClipboard = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text);
    toast.success('Copied to clipboard');
  } catch (err) {
    toast.error('Failed to copy to clipboard');
  }
};

</script>
