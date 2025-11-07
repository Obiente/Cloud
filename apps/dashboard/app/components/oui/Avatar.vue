<template>
  <Avatar.Root
    :class="['oui-avatar-base', `oui-avatar-${size}`]"
    v-bind="$attrs"
  >
    <Avatar.Fallback
      :class="['oui-avatar-fallback', `oui-avatar-fallback-${size}`]"
    >
      {{ computedFallbackText }}
    </Avatar.Fallback>
    <Avatar.Image
      v-if="src"
      :src="src"
      :alt="alt || 'Avatar'"
      :class="['oui-avatar-image', `oui-avatar-image-${size}`]"
    />
  </Avatar.Root>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Avatar } from "@ark-ui/vue/avatar";
import { getInitials } from "~/utils/common";

interface Props {
  src?: string;
  alt?: string;
  fallbackText?: string;
  name?: string; // Optional name for LL (Lastname Lastname) initials generation
  size?: "sm" | "md" | "lg" | "xl";
}

const props = withDefaults(defineProps<Props>(), {
  fallbackText: undefined,
  size: "md",
});

const computedFallbackText = computed(() => {
  if (props.fallbackText !== undefined) {
    return props.fallbackText;
  }
  if (props.name) {
    return getInitials(props.name);
  }
  return "??";
});

defineOptions({
  inheritAttrs: false,
});
</script>
