<template>
  <OuiMenu>
    <template #trigger>
      <OuiButton variant="ghost" size="sm" class="!p-2" title="Switch theme">
        <SwatchIcon class="h-5 w-5" />
      </OuiButton>
    </template>
    <template #default>
      <OuiMenuItem
        value="dark-purple"
        @select="() => setTheme('dark-purple')"
      >
        <OuiFlex align="center" gap="sm" justify="between" class="w-full">
          <OuiFlex align="center" gap="sm">
            <div class="w-4 h-4 rounded border border-border-default" style="background: linear-gradient(135deg, #0b0b12 0%, #121022 50%, #a855f7 100%);" />
            <OuiText size="sm">Dark Purple</OuiText>
          </OuiFlex>
          <CheckIcon v-if="currentTheme === 'dark-purple'" class="h-4 w-4 text-primary" />
        </OuiFlex>
      </OuiMenuItem>
      <OuiMenuItem
        value="dark"
        @select="() => setTheme('dark')"
      >
        <OuiFlex align="center" gap="sm" justify="between" class="w-full">
          <OuiFlex align="center" gap="sm">
            <div class="w-4 h-4 rounded border border-border-default bg-surface-base" />
            <OuiText size="sm">Dark</OuiText>
          </OuiFlex>
          <CheckIcon v-if="currentTheme === 'dark'" class="h-4 w-4 text-primary" />
        </OuiFlex>
      </OuiMenuItem>
    </template>
  </OuiMenu>
</template>

<script setup lang="ts">
import { SwatchIcon, CheckIcon } from "@heroicons/vue/24/outline";
import { useTheme } from "~/composables/useTheme";
import OuiMenu from "./Menu.vue";
import OuiMenuItem from "./MenuItem.vue";
import OuiButton from "./Button.vue";
import OuiFlex from "./Flex.vue";
import OuiText from "./Text.vue";

const { currentTheme, setTheme } = useTheme();
</script>

