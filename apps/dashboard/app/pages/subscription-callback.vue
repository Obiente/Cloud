<template>
  <div class="flex min-h-screen items-center justify-center bg-surface-base">
    <div class="text-center p-8">
      <div class="mb-4">
        <svg class="animate-spin h-16 w-16 text-gray-400 mx-auto" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      </div>
      <h2 class="text-2xl font-bold mb-2">Processing...</h2>
      <p class="text-gray-500">Please wait while we complete your subscription.</p>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  layout: false, // No layout needed for callback page
  middleware: [], // No auth required for callback
});

import { onMounted } from "vue";

onMounted(() => {
  const urlParams = new URLSearchParams(window.location.search);
  const status = urlParams.get("status") || "success";
  
  // Notify parent window
  if (window.opener) {
    window.opener.postMessage(
      { type: "stripe-checkout-complete", status },
      window.location.origin
    );
  }
  
  // Close popup after a short delay
  setTimeout(() => {
    window.close();
  }, 500);
});
</script>

