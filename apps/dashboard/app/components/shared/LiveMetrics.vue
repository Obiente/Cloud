<template>
  <OuiGrid :cols="{ sm: 2, md: 4 }" gap="sm">
    <!-- CPU Usage -->
    <OuiCard variant="outline" class="group relative overflow-hidden">
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiFlex justify="between" align="center">
            <OuiFlex align="center" gap="xs">
              <CpuChipIcon class="h-3.5 w-3.5 text-accent-primary" />
              <OuiText size="xs" color="tertiary">CPU</OuiText>
            </OuiFlex>
            <span
              v-if="isStreaming"
              class="h-1.5 w-1.5 rounded-full bg-success animate-pulse"
            />
          </OuiFlex>
          <OuiText size="xl" weight="semibold">
            {{ latestMetric ? currentCpuUsage.toFixed(1) + '%' : '—' }}
          </OuiText>
          <div class="h-1.5 w-full bg-surface-muted rounded-full overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500 ease-out"
              :class="cpuBarColor"
              :style="{ width: `${Math.min(currentCpuUsage, 100)}%` }"
            />
          </div>
        </OuiStack>
      </OuiCardBody>
      <!-- Subtle glow at bottom based on usage level -->
      <div
        v-if="latestMetric && currentCpuUsage > 0"
        class="absolute bottom-0 left-0 right-0 h-px transition-colors duration-500"
        :class="cpuBarColor"
      />
    </OuiCard>

    <!-- Memory Usage -->
    <OuiCard variant="outline" class="group relative overflow-hidden">
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiFlex justify="between" align="center">
            <OuiFlex align="center" gap="xs">
              <CircleStackIcon class="h-3.5 w-3.5 text-accent-info" />
              <OuiText size="xs" color="tertiary">Memory</OuiText>
            </OuiFlex>
            <span
              v-if="isStreaming"
              class="h-1.5 w-1.5 rounded-full bg-success animate-pulse"
            />
          </OuiFlex>
          <OuiText size="xl" weight="semibold">
            <template v-if="latestMetric">
              <OuiByte :value="currentMemoryUsage" unit-display="short" />
            </template>
            <template v-else>—</template>
          </OuiText>
          <OuiText size="xs" color="tertiary">
            {{ latestMetric ? 'Active' : 'Waiting for data' }}
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Network Rx -->
    <OuiCard variant="outline" class="group relative overflow-hidden">
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiFlex justify="between" align="center">
            <OuiFlex align="center" gap="xs">
              <ArrowDownTrayIcon class="h-3.5 w-3.5 text-success" />
              <OuiText size="xs" color="tertiary">Inbound</OuiText>
            </OuiFlex>
            <span
              v-if="isStreaming"
              class="h-1.5 w-1.5 rounded-full bg-success animate-pulse"
            />
          </OuiFlex>
          <OuiText size="xl" weight="semibold">
            <template v-if="latestMetric">
              <OuiByte :value="currentNetworkRx" unit-display="short" base="decimal" />
            </template>
            <template v-else>—</template>
          </OuiText>
          <OuiText size="xs" color="tertiary">
            {{ latestMetric ? 'Total received' : 'Waiting for data' }}
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

    <!-- Network Tx -->
    <OuiCard variant="outline" class="group relative overflow-hidden">
      <OuiCardBody>
        <OuiStack gap="sm">
          <OuiFlex justify="between" align="center">
            <OuiFlex align="center" gap="xs">
              <ArrowUpTrayIcon class="h-3.5 w-3.5 text-accent-secondary" />
              <OuiText size="xs" color="tertiary">Outbound</OuiText>
            </OuiFlex>
            <span
              v-if="isStreaming"
              class="h-1.5 w-1.5 rounded-full bg-success animate-pulse"
            />
          </OuiFlex>
          <OuiText size="xl" weight="semibold">
            <template v-if="latestMetric">
              <OuiByte :value="currentNetworkTx" unit-display="short" base="decimal" />
            </template>
            <template v-else>—</template>
          </OuiText>
          <OuiText size="xs" color="tertiary">
            {{ latestMetric ? 'Total sent' : 'Waiting for data' }}
          </OuiText>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>
  </OuiGrid>
</template>

<script setup lang="ts">
import { computed } from "vue";
import {
  CpuChipIcon,
  CircleStackIcon,
  ArrowDownTrayIcon,
  ArrowUpTrayIcon,
} from "@heroicons/vue/24/outline";
import OuiByte from "~/components/oui/Byte.vue";

interface Props {
  isStreaming: boolean;
  latestMetric: any;
  currentCpuUsage: number;
  currentMemoryUsage: number;
  currentNetworkRx: number;
  currentNetworkTx: number;
}

const props = defineProps<Props>();

const cpuBarColor = computed(() => {
  if (props.currentCpuUsage >= 90) return 'bg-danger';
  if (props.currentCpuUsage >= 70) return 'bg-warning';
  return 'bg-accent-primary';
});
</script>

