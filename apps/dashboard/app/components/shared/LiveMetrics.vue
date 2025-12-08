<template>
  <OuiCard>
    <OuiCardHeader>
      <OuiFlex justify="between" align="center">
        <OuiStack gap="xs">
          <OuiText size="lg" weight="bold">Live Metrics</OuiText>
          <OuiText size="xs" color="muted">
            Real-time resource usage
          </OuiText>
        </OuiStack>
        <OuiFlex align="center" gap="sm">
          <span
            class="h-2 w-2 rounded-full"
            :class="isStreaming ? 'bg-success animate-pulse' : 'bg-secondary'"
          />
          <OuiText size="xs" color="muted">
            {{ isStreaming ? "Live" : "Stopped" }}
          </OuiText>
        </OuiFlex>
      </OuiFlex>
    </OuiCardHeader>
    <OuiCardBody>
      <OuiGrid :cols="{ sm: 1, md: 2, lg: 4 }" gap="md">
        <!-- CPU Usage -->
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
        >
          <OuiStack gap="xs">
            <OuiFlex align="center" justify="between">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                CPU Usage
              </OuiText>
              <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                Live
              </OuiBadge>
            </OuiFlex>
            <OuiText size="2xl" weight="bold">
              {{ currentCpuUsage.toFixed(1) }}%
            </OuiText>
            <OuiBox
              w="full"
              class="h-1 bg-surface-muted overflow-hidden rounded-full mt-1"
            >
              <OuiBox
                class="h-full bg-accent-primary transition-all duration-300"
                :style="{ width: `${Math.min(currentCpuUsage, 100)}%` }"
              />
            </OuiBox>
            <OuiText size="xs" color="muted">
              {{ latestMetric ? "Current" : "No data" }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <!-- Memory Usage -->
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
        >
          <OuiStack gap="xs">
            <OuiFlex align="center" justify="between">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Memory Usage
              </OuiText>
              <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                Live
              </OuiBadge>
            </OuiFlex>
            <OuiText size="2xl" weight="bold">
              <OuiByte :value="currentMemoryUsage" unit-display="short" />
            </OuiText>
            <OuiText size="xs" color="muted">
              {{ latestMetric ? "Current" : "No data" }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <!-- Network Rx -->
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
        >
          <OuiStack gap="xs">
            <OuiFlex align="center" justify="between">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Network Rx
              </OuiText>
              <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                Live
              </OuiBadge>
            </OuiFlex>
            <OuiText size="2xl" weight="bold">
              <OuiByte :value="currentNetworkRx" unit-display="short" base="decimal" />
            </OuiText>
            <OuiText size="xs" color="muted">
              {{ latestMetric ? "Total received" : "No data" }}
            </OuiText>
          </OuiStack>
        </OuiBox>

        <!-- Network Tx -->
        <OuiBox
          p="md"
          rounded="xl"
          class="ring-1 ring-border-muted bg-surface-muted/30 hover:bg-surface-muted/50 transition-colors"
        >
          <OuiStack gap="xs">
            <OuiFlex align="center" justify="between">
              <OuiText size="xs" color="muted" transform="uppercase" weight="semibold">
                Network Tx
              </OuiText>
              <OuiBadge v-if="latestMetric" variant="secondary" size="sm">
                Live
              </OuiBadge>
            </OuiFlex>
            <OuiText size="2xl" weight="bold">
              <OuiByte :value="currentNetworkTx" unit-display="short" base="decimal" />
            </OuiText>
            <OuiText size="xs" color="muted">
              {{ latestMetric ? "Total sent" : "No data" }}
            </OuiText>
          </OuiStack>
        </OuiBox>
      </OuiGrid>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import { computed } from "vue";
import OuiByte from "~/components/oui/Byte.vue";

interface Props {
  isStreaming: boolean;
  latestMetric: any;
  currentCpuUsage: number;
  currentMemoryUsage: number;
  currentNetworkRx: number;
  currentNetworkTx: number;
}

defineProps<Props>();
</script>

