<template>
  <OuiBox
    :class="[
      sizeClasses[size],
      borderRadiusClasses[size],
      'bg-primary',
      $attrs.class
    ]"
  >
    <OuiFlex align="center" justify="center" class="h-full">
      <OuiText 
        :size="textSize[size]" 
        weight="bold" 
        :class="textColorClass"
        :style="textColorStyle"
      >O</OuiText>
    </OuiFlex>
  </OuiBox>
</template>

<script setup lang="ts">
import { computed } from "vue";
import OuiBox from "../oui/Box.vue";
import OuiFlex from "../oui/Flex.vue";
import OuiText from "../oui/Text.vue";
import type { OUISize } from "../oui/types";
import { useTheme } from "~/composables/useTheme";

type LogoSize = "sm" | "md" | "lg" | "xl";

interface Props {
  size?: LogoSize;
}

const props = withDefaults(defineProps<Props>(), {
  size: "md",
});

const { currentTheme } = useTheme();

// Use dark text for dark theme (white background), light text for dark-purple theme
const textColorClass = computed(() => {
  return currentTheme.value === "dark" ? "" : "text-primary";
});

const textColorStyle = computed(() => {
  // In dark theme, use dark text (#0a0a0a) on white background
  // In dark-purple theme, use light text (text-primary)
  if (currentTheme.value === "dark") {
    return { color: "#0a0a0a" };
  }
  return {};
});

const sizeClasses: Record<LogoSize, string> = {
  sm: "w-6 h-6",
  md: "w-8 h-8",
  lg: "w-16 h-16",
  xl: "w-20 h-20",
};

const borderRadiusClasses: Record<LogoSize, string> = {
  sm: "rounded-lg",
  md: "rounded-xl",
  lg: "rounded-2xl",
  xl: "rounded-2xl",
};

const textSize: Record<LogoSize, OUISize> = {
  sm: "sm",
  md: "lg",
  lg: "2xl",
  xl: "3xl",
};
</script>

