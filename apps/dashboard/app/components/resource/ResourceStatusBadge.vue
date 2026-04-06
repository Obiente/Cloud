<template>
  <OuiBadge :variant="badge" size="xs">
    <span
      class="inline-flex h-1.5 w-1.5 rounded-full mr-1"
      :class="dotClass"
    />
    {{ label }}
  </OuiBadge>
</template>

<script setup lang="ts">
type BadgeVariant = "primary" | "secondary" | "success" | "warning" | "danger" | "outline";

defineProps<{
  label: string;
  badge: BadgeVariant;
  dotClass: string;
}>();
</script>

