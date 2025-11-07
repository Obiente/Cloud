<template>
  <ComingSoon
    title="Virtual Private Servers"
    description="Deploy and manage your own VPS instances with full root access and complete control."
    :icon="ServerIcon"
    additional-info="Launch virtual private servers with custom configurations, operating systems, and networking options. Full root access and complete control over your infrastructure."
  />
</template>

<script setup lang="ts">
import { ServerIcon } from "@heroicons/vue/24/outline";
import ComingSoon from "~/components/ComingSoon.vue";

definePageMeta({
  layout: "default",
});
</script>

