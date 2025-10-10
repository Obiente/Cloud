<template>
  <span :class="['oui-badge', `oui-badge-${variant}`]" v-bind="$attrs">
    <slot />
  </span>
</template>

<script setup lang="ts">
interface Props {
  variant?:
    | "primary"
    | "secondary"
    | "success"
    | "warning"
    | "danger"
    | "outline";
}

withDefaults(defineProps<Props>(), {
  variant: "secondary",
});

defineOptions({
  inheritAttrs: false,
});
</script>
