<template>
  <div class="text-right">
    <OuiMenu v-if="actions.length > 1">
      <template #trigger>
        <OuiButton 
          variant="ghost" 
          size="sm"
          @click.stop
        >
          <EllipsisVerticalIcon class="h-4 w-4" />
        </OuiButton>
      </template>
      <OuiMenuItem
        v-for="(action, index) in actions"
        :key="index"
        :value="action.key || `action-${index}`"
        :color="action.color"
        @select="action.onClick"
      >
        {{ action.label }}
      </OuiMenuItem>
    </OuiMenu>
    <OuiButton
      v-else-if="actions.length === 1 && actions[0]"
      :variant="actions[0].variant || 'ghost'"
      :color="actions[0].color"
      :size="actions[0].size || 'sm'"
      @click.stop="actions[0].onClick"
    >
      {{ actions[0].label }}
    </OuiButton>
  </div>
</template>

<script setup lang="ts">
import { EllipsisVerticalIcon } from "@heroicons/vue/24/outline";

export interface Action {
  key?: string;
  label: string;
  onClick: () => void;
  variant?: "solid" | "outline" | "ghost";
  color?: "primary" | "danger" | "warning" | "success" | "neutral";
  size?: "xs" | "sm" | "md" | "lg";
}

defineProps<{
  actions: Action[];
}>();
</script>

