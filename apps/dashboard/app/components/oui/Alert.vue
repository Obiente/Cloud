<template>
  <div
    :role="tone === 'danger' ? 'alert' : 'status'"
    class="rounded-xl border text-primary"
    :class="[size === 'sm' ? 'px-3 py-2 text-sm' : 'px-4 py-3', toneClasses]"
  >
    <p v-if="title" class="font-semibold" :class="{ 'mb-1': $slots.default }">{{ title }}</p>
    <slot />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";

type Tone = "info" | "success" | "warning" | "danger" | "error" | "muted";
const props = withDefaults(defineProps<{
  title?: string;
  variant?: Tone;
  color?: Tone;
  size?: "sm" | "md";
}>(), { size: "md" });

const tone = computed(() => {
  const value = props.variant || props.color || "info";
  return value === "error" ? "danger" : value;
});
const toneClasses = computed(() => ({
  info: "border-info/30 bg-info/10",
  success: "border-success/30 bg-success/10",
  warning: "border-warning/30 bg-warning/10",
  danger: "border-danger/30 bg-danger/10",
  muted: "border-border-default bg-surface-subtle",
})[tone.value]);
</script>
