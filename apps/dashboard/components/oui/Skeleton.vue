<template>
  <div 
    :class="[
      'oui-skeleton-base',
      `oui-skeleton-${variant}`,
      {
        'oui-skeleton-animated': animated,
        'oui-skeleton-rounded': rounded,
      }
    ]"
    :style="{
      width: width,
      height: height,
    }"
    v-bind="$attrs"
  >
    <div v-if="animated" class="oui-skeleton-shimmer"></div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  width?: string;
  height?: string;
  variant?: 'rectangle' | 'circle' | 'text';
  animated?: boolean;
  rounded?: boolean;
}

withDefaults(defineProps<Props>(), {
  variant: 'rectangle',
  animated: true,
  rounded: false,
});

defineOptions({
  inheritAttrs: false,
});
</script>