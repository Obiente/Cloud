<template>
  <NuxtLink
    :to="to"
    class="app-nav-link w-full px-3 py-1.5 rounded-lg text-[0.8125rem] font-medium transition-colors cursor-pointer group flex items-center gap-2.5"
    :class="[
      isActive ? 'app-nav-link-active text-primary' : 'text-secondary hover:text-primary',
    ]"
    @click="handleClick"
  >
    <OuiBox as="span" :shrink="false" class="app-nav-link-icon">
      <component
        :is="icon"
        class="h-4 w-4 shrink-0 transition-colors"
        :class="[
          isActive ? 'text-primary' : 'text-tertiary group-hover:text-secondary',
        ]"
      />
    </OuiBox>
    <OuiText as="span" size="sm" truncate class="flex-1">{{ label }}</OuiText>
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
  // Parse the 'to' prop to extract path and query params
  const [path = '', queryString] = props.to.split('?');
  const pathMatches = route.path.startsWith(path);
  
  // If query params are specified, check them too
  if (queryString && pathMatches) {
    const queryParams = new URLSearchParams(queryString);
    for (const [key, value] of queryParams.entries()) {
      if (route.query[key] !== value) {
        return false;
      }
    }
  }
  
  return pathMatches;
});

const handleClick = () => {
  emit("navigate");
};
</script>
