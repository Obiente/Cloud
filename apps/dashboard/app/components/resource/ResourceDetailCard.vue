<template>
  <OuiCard variant="default">
    <OuiCardBody>
      <OuiStack gap="sm">
        <OuiFlex align="center" gap="sm">
          <component
            v-if="icon"
            :is="icon"
            class="h-5 w-5 text-secondary"
          />
          <OuiText size="sm" color="secondary">
            {{ label }}
          </OuiText>
        </OuiFlex>
        <OuiText size="2xl" weight="bold" color="primary">
          <slot />
        </OuiText>
        <OuiText v-if="description" size="xs" color="secondary" class="mt-1">
          {{ description }}
        </OuiText>
      </OuiStack>
    </OuiCardBody>
  </OuiCard>
</template>

<script setup lang="ts">
import type { Component } from "vue";

defineProps<{
  label: string;
  icon?: Component;
  description?: string;
}>();
</script>


