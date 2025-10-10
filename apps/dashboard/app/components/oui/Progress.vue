<template>
  <div :class="['oui-progress-base', `oui-progress-${size}`]" v-bind="$attrs">
    <div class="oui-progress-track">
      <div
        class="oui-progress-bar"
        :style="{ width: `${clampedValue}%` }"
        :class="[`oui-progress-bar-${variant}`]"
      >
        <div v-if="animated" class="oui-progress-animated" />
      </div>
    </div>
    <span v-if="showValue" class="oui-progress-label">
      {{ Math.round(clampedValue) }}%
    </span>
  </div>
</template>

<script setup lang="ts">
interface Props {
  value?: number;
  max?: number;
  size?: "sm" | "md" | "lg";
  variant?: "primary" | "success" | "warning" | "danger";
  showValue?: boolean;
  animated?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  value: 0,
  max: 100,
  size: "md",
  variant: "primary",
  showValue: false,
  animated: false,
});

const clampedValue = computed(() => {
  return Math.min(Math.max(props.value, 0), props.max) * (100 / props.max);
});

defineOptions({
  inheritAttrs: false,
});
</script>
