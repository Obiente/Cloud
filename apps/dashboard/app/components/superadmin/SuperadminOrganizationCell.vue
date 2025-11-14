<template>
  <div>
    <div class="font-medium text-text-primary">{{ organizationName || organizationId || "—" }}</div>
    <div v-if="organizationId && organizationName" class="text-xs font-mono text-text-tertiary mt-0.5">
      {{ organizationId }}
    </div>
    <div v-else-if="organizationId" class="text-xs font-mono text-text-tertiary mt-0.5">
      {{ organizationId }}
    </div>
    <div v-if="ownerName" class="text-xs text-text-muted mt-0.5">
      Owner: {{ ownerName }}
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  organizationName?: string;
  organizationId?: string;
  ownerName?: string;
}>();
</script>

