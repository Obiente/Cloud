<template>
  <div
    :class="[
      // Base card composition (reusable)
      'oui-card-base',
      // Variant styles applied directly
      {
        'shadow-md bg-surface-raised border-muted': variant === 'raised',
        'shadow-lg bg-surface-overlay border-strong': variant === 'overlay',
        'bg-transparent border-default': variant === 'outline',
        'cursor-pointer transition-all duration-200 hover:shadow-md hover:bg-hover active:shadow active:bg-active oui-focus-ring':
          interactive,
        'hover:-translate-y-0.5 hover:shadow-lg transition-all duration-200 cursor-pointer':
          hoverable,
        [`oui-card-status-${status}`]: status,
      },
    ]"
    v-bind="$attrs"
  >
    <slot />
  </div>
</template>

<script setup lang="ts">
interface Props {
  variant?: "default" | "raised" | "overlay" | "outline";
  interactive?: boolean;
  hoverable?: boolean;
  status?: "success" | "warning" | "danger" | "info";
}

defineProps<Props>();
defineOptions({
  inheritAttrs: false,
});
</script>
