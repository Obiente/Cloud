<template>
  <OuiStack align="center" gap="md" class="text-center py-16">
    <OuiBox
      v-if="icon"
      class="inline-flex items-center justify-center w-10 h-10 rounded-lg bg-surface-muted"
    >
      <component :is="icon" class="h-5 w-5 text-tertiary" />
    </OuiBox>

    <OuiStack align="center" gap="xs">
        <OuiText as="h3" size="sm" weight="semibold">
        {{ title }}
      </OuiText>
      <OuiText v-if="description" size="sm" color="tertiary" class="max-w-sm">
        {{ description }}
      </OuiText>
    </OuiStack>

    <slot />
  </OuiStack>
</template>

<script setup lang="ts">
import type { Component } from "vue";

interface Props {
  title: string;
  description?: string;
  icon?: Component;
}

defineProps<Props>();
</script>
