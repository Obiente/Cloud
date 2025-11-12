<template>
  <OuiBadge :variant="badge" size="xs" class="md:size-sm">
    <span
      class="inline-flex h-1.5 w-1.5 rounded-full mr-1.5"
      :class="dotClass"
    />
    <OuiText
      as="span"
      size="xs"
      weight="semibold"
      transform="uppercase"
    >
      {{ label }}
    </OuiText>
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

