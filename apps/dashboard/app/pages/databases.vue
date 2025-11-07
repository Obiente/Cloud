<template>
  <ComingSoon
    title="Managed Databases"
    description="Deploy and manage your databases with automated backups, scaling, and monitoring."
    :icon="CircleStackIcon"
    additional-info="Launch managed database instances including PostgreSQL, MySQL, MongoDB, and more. Automated backups, scaling, and monitoring included."
  />
</template>

<script setup lang="ts">
import { CircleStackIcon } from "@heroicons/vue/24/outline";
import ComingSoon from "~/components/ComingSoon.vue";

definePageMeta({
  layout: "default",
});
</script>

