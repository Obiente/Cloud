<template>
  <NuxtLayout>
    <NuxtPage />
    <AlertDialog />
  </NuxtLayout>
</template>

<script setup lang="ts">
import { defineAsyncComponent } from "vue";
const AlertDialog = defineAsyncComponent(() => import("~/components/oui/AlertDialog.vue"));
</script>
