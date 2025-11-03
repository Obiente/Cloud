<template>
  <Toaster :toaster="toaster" v-slot="toast" class="fixed z-1000 sm:p-0 p-4 w-full sm:w-auto">
    <OuiToast
      :title="String(toast.title || '')"
      :description="toast.description ? String(toast.description) : undefined"
      :type="(toast.type as 'success' | 'error' | 'warning' | 'info')"
      :icon="getIconForType(toast.type)"
    />
  </Toaster>
</template>

<script setup lang="ts">
import { Toaster } from "@ark-ui/vue/toast";
import OuiToast from "./Toast.vue";
import { useToast } from "~/composables/useToast";

interface Props {
  toaster: any;
}

defineProps<Props>();

const { iconMap } = useToast();

const getIconForType = (type: string | undefined) => {
  if (!type) return iconMap.info;
  return iconMap[type as keyof typeof iconMap] || iconMap.info;
};
</script>
