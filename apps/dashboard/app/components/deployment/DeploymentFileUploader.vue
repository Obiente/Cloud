<template>
  <BaseFileUploader
    :uploaderId="deploymentId"
    :destinationPath="destinationPath"
    :volumeName="volumeName"
    :additionalParams="{ containerId, serviceName }"
    @uploaded="emit('uploaded', $event)"
  />
</template>

<script setup lang="ts">
  import BaseFileUploader from "~/components/shared/BaseFileUploader.vue";

  interface Props {
    deploymentId: string;
    destinationPath?: string;
    volumeName?: string;
    containerId?: string;
    serviceName?: string;
  }

  interface Emits {
    (e: "uploaded", files: File[]): void;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();
</script>
