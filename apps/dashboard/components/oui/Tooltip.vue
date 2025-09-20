<template>
  <ArkTooltip.Root>
    <ArkTooltip.Trigger v-bind="triggerProps">
      <slot name="trigger" />
    </ArkTooltip.Trigger>
    <ArkTooltip.Positioner>
      <ArkTooltip.Content
        :class="[
          'tooltip-content',
          `tooltip-${variant}`
        ]"
      >
        <ArkTooltip.Arrow class="tooltip-arrow" />
        <slot />
      </ArkTooltip.Content>
    </ArkTooltip.Positioner>
  </ArkTooltip.Root>
</template>

<script setup lang="ts">
import { Tooltip as ArkTooltip } from '@ark-ui/vue'

interface Props {
  variant?: 'default' | 'dark' | 'light'
  triggerProps?: Record<string, any>
}

withDefaults(defineProps<Props>(), {
  variant: 'default',
  triggerProps: () => ({})
})
</script>