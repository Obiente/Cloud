<template>
  <OuiCard variant="outline" class="border-border-default/50">
    <OuiCardBody>
      <OuiFlex justify="between" align="start" wrap="wrap" gap="md">
        <OuiStack gap="xs" class="flex-1 min-w-0">
          <OuiFlex align="center" gap="sm" wrap="wrap">
            <OuiText as="h1" size="lg" weight="semibold" truncate>
              {{ title }}
            </OuiText>
            <slot name="badges" />
          </OuiFlex>
          <OuiText size="xs" color="tertiary">
            <slot name="subtitle" />
          </OuiText>
        </OuiStack>

        <OuiFlex gap="xs" wrap="wrap" class="shrink-0">
          <slot name="actions" />
        </OuiFlex>
      </OuiFlex>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import type { Component } from "vue";

defineProps<{
  title: string;
  icon?: Component;
  iconClass?: string;
}>();
</script>


