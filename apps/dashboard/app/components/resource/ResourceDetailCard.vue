<template>
  <OuiCard variant="default">
    <OuiCardBody>
      <OuiStack gap="xs">
        <OuiText size="xs" color="tertiary" weight="medium">
          {{ label }}
        </OuiText>
        <OuiText size="xl" weight="semibold">
          <slot />
        </OuiText>
        <OuiText v-if="description" size="xs" color="tertiary">
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


