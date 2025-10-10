<template>
  <Avatar.Root
    :class="['oui-avatar-base', `oui-avatar-${size}`]"
    v-bind="$attrs"
  >
    <Avatar.Fallback
      :class="['oui-avatar-fallback', `oui-avatar-fallback-${size}`]"
    >
      {{ fallbackText }}
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
import { Avatar } from "@ark-ui/vue/avatar";

interface Props {
  src?: string;
  alt?: string;
  fallbackText?: string;
  size?: "sm" | "md" | "lg" | "xl";
}

const props = withDefaults(defineProps<Props>(), {
  fallbackText: "??",
  size: "md",
});

defineOptions({
  inheritAttrs: false,
});
</script>
