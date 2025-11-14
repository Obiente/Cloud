<template>
  <OuiStack gap="xs">
    <OuiText tag="h1" size="3xl" weight="extrabold">{{ title }}</OuiText>
    <OuiText v-if="description" color="muted">{{ description }}</OuiText>
  </OuiStack>
</template>

<script setup lang="ts">
defineProps<{
  title: string;
  description?: string;
}>();
</script>

