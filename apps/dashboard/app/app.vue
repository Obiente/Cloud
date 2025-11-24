<template>
  <NuxtLayout>
    <NuxtPage />
    <AlertDialog />
  </NuxtLayout>
</template>

<script setup lang="ts">
import { defineAsyncComponent, onMounted } from "vue";
import { useTheme } from "~/composables/useTheme";

const AlertDialog = defineAsyncComponent(() => import("~/components/oui/AlertDialog.vue"));

// Initialize theme (works on both SSR and client)
const { currentTheme, initializeTheme } = useTheme();

// Set theme attribute for SSR (from cookie)
useHead({
  htmlAttrs: {
    "data-theme": currentTheme.value,
  },
});

// Initialize theme on client mount (applies to document)
onMounted(() => {
  initializeTheme();
});
</script>
