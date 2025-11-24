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

// Use light text for dark and dark-purple themes (colored backgrounds), dark text for extra-dark (white background)
const textColorClass = computed(() => {
  return currentTheme.value === "extra-dark" ? "" : "text-primary";
});

const textColorStyle = computed(() => {
  // In extra-dark theme, use dark text (#0a0a0a) on white/grey background
  // In dark and dark-purple themes, use light text (text-primary) on colored background
  if (currentTheme.value === "extra-dark") {
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

