<template>
  <NuxtLink
    :to="to"
    class="flex items-center space-x-3 px-3 py-2 rounded-md text-sm font-medium transition-all duration-200 cursor-pointer group"
    :class="[
      isActive
        ? 'bg-primary/10 text-primary border border-primary/20'
        : 'text-secondary hover:bg-hover hover:text-primary',
    ]"
    @click="handleClick"
  >
    <component
      :is="icon"
      class="w-5 h-5 transition-colors"
      :class="[
        isActive ? 'text-primary' : 'text-secondary group-hover:text-primary',
      ]"
    />
    <span>{{ label }}</span>
  </NuxtLink>
</template>

<script setup lang="ts">
import type { Component } from "vue";

interface Props {
  to: string;
  label: string;
  icon: Component;
  exactMatch?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  exactMatch: false,
});

const route = useRoute();

const emit = defineEmits<{
  navigate: [];
}>();

const isActive = computed(() => {
  if (props.exactMatch) {
    return route.path === props.to;
  }
  return route.path.startsWith(props.to);
});

const handleClick = () => {
  emit("navigate");
};
</script>
