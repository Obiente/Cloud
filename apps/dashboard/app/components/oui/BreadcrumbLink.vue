<template>
  <a
    class="oui-breadcrumb-link"
    v-bind="$attrs"
    :aria-current="ariaCurrent"
  >
    <slot />
  </a>
</template>

<script setup lang="ts">
const props = defineProps<{ ariaCurrent?: "page" | "step" | "location" | "date" | "time" | "true" | "false" }>();
const ariaCurrent = props.ariaCurrent ?? undefined;
</script>

<style scoped>
.oui-breadcrumb-link {
  color: inherit;
  text-decoration: none;
  border-radius: 4px;
  padding: 2px 4px;
  transition: background-color 0.12s ease, color 0.12s ease;
}

.oui-breadcrumb-link:hover {
  background: var(--oui-surface-hover);
  color: var(--oui-text-primary);
}

.oui-breadcrumb-link[aria-current="page"] {
  color: var(--oui-text-primary);
  pointer-events: none;
  font-weight: 600;
}
</style>
