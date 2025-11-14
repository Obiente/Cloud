<template>
  <div>
    <div class="font-medium text-text-primary hover:text-primary transition-colors">
      {{ name || fallback || "—" }}
    </div>
    <div v-if="description" class="text-xs text-text-muted mt-0.5">{{ description }}</div>
    <div v-if="subtitle" class="text-xs text-text-muted mt-0.5">{{ subtitle }}</div>
    <div v-if="id" class="text-xs font-mono text-text-tertiary mt-0.5">{{ id }}</div>
    <div v-if="domain" class="text-xs text-text-muted mt-0.5">{{ domain }}</div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  name?: string;
  description?: string;
  subtitle?: string;
  id?: string;
  domain?: string;
  fallback?: string;
}>();
</script>

