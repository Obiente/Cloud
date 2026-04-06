<template>
  <!--
    A monospace value with an optional label and a hover-reveal copy button.
    The outer `group` is on this component's root, so hover anywhere on the
    field to reveal the copy button.

    Variants:
      bare    — just the content, no border/padding (default, compose inside other containers)
      field   — adds its own bordered rounded container with padding
  -->
  <div :class="rootClass">
    <OuiStack gap="xs" class="min-w-0 flex-1">
      <OuiText v-if="label" size="xs" color="tertiary">{{ label }}</OuiText>
      <OuiText
        :size="size"
        weight="medium"
        :class="[mono ? 'font-mono' : '', breakAll ? 'break-all' : 'truncate']"
      >
        {{ value }}
      </OuiText>
    </OuiStack>
    <button
      class="p-1 rounded text-tertiary hover:text-primary transition-all shrink-0 opacity-0 group-hover:opacity-100"
      :title="`Copy${label ? ' ' + label : ''}`"
      @click.stop="handleCopy"
    >
      <CheckIcon v-if="copied" class="h-3.5 w-3.5 text-success" />
      <ClipboardIcon v-else class="h-3.5 w-3.5" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from "vue";
import { ClipboardIcon, CheckIcon } from "@heroicons/vue/24/outline";
import { useToast } from "~/composables/useToast";

const props = withDefaults(defineProps<{
  value: string;
  label?: string;
  /** Show value in font-mono */
  mono?: boolean;
  /** Break long values (e.g. long connection strings) */
  breakAll?: boolean;
  /** Text size for the value */
  size?: "xs" | "sm" | "md";
  /** Whether to add a border/padding container or just render bare */
  variant?: "bare" | "field";
}>(), {
  mono: true,
  breakAll: false,
  size: "sm",
  variant: "bare",
});

const { toast } = useToast();
const copied = ref(false);

const rootClass = computed(() => [
  "group flex items-start justify-between gap-2 min-w-0",
  props.variant === "field"
    ? "rounded-lg border border-border-default px-3 py-2.5"
    : "",
]);

async function handleCopy() {
  try {
    await navigator.clipboard.writeText(props.value);
    copied.value = true;
    toast.success(`${props.label ?? "Value"} copied`);
    setTimeout(() => (copied.value = false), 2000);
  } catch {
    toast.error("Failed to copy");
  }
}
</script>
