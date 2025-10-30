<template>
  <DeploymentLogs :id="id" />
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { useRoute } from "vue-router";
  import DeploymentLogs from "~/components/deployments/DeploymentLogs.vue";

  definePageMeta({ layout: "deployment", middleware: "auth" });

  const route = useRoute();
  const id = computed(() => route.params.id as string);
  // Display handled by component
</script>
