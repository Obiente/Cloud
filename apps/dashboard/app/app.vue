<template>
  <AccessRestriction v-if="showRestriction" :account="restriction?.scope === 'account'" />
  <NuxtLayout v-else>
    <NuxtPage />
    <AlertDialog />
  </NuxtLayout>
</template>

<script setup lang="ts">
import { defineAsyncComponent, onMounted } from "vue";
import { useTheme } from "~/composables/useTheme";

const AlertDialog = defineAsyncComponent(() => import("~/components/oui/AlertDialog.vue"));
const restriction = useAccessRestriction();
const route = useRoute();
const showRestriction = computed(() => restriction.value && (
  restriction.value.scope === "account" || route.path !== "/organizations"
));

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
