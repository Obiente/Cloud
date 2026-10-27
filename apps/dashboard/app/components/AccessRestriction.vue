<template>
  <main class="min-h-screen bg-background text-text-primary flex items-center justify-center p-6">
    <section class="w-full max-w-lg rounded-xl border border-border bg-surface p-8 space-y-5" role="alert">
      <h1 class="text-2xl font-semibold">{{ account ? "Account restricted" : "Organization restricted" }}</h1>
      <p class="text-text-secondary">
        {{ account
          ? "Your account is suspended or banned. Contact support if you believe this is a mistake."
          : "This organization is suspended or banned. Its resources are unavailable to members." }}
      </p>
      <div class="flex flex-wrap gap-3">
        <OuiButton v-if="!account" @click="navigateTo('/organizations')">Choose another organization</OuiButton>
        <OuiButton variant="ghost" @click="auth.logout()">Sign out</OuiButton>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
defineProps<{ account: boolean }>();
const auth = useAuth();
</script>
