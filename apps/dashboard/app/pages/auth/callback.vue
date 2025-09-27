<template>
  <div></div>
</template>

<script lang="ts" setup>
definePageMeta({
  layout: 'default',
  middleware: 'auth',
});

window?.close();
</script>
