<template>
  <OuiBox
    bg="surface-base"
    class="min-h-screen flex items-center justify-center"
    p="md"
  >
    <OuiContainer size="2xl" class="text-center">
      <OuiStack gap="2xl" align="center">
        <!-- 404 Illustration -->
        <OuiBox position="relative">
          <OuiBox
            position="absolute"
            class="inset-0 bg-primary/10 blur-3xl rounded-full"
          />
          <OuiStack gap="md" align="center" class="relative">
            <OuiText
              as="h1"
              size="7xl"
              weight="bold"
              color="primary"
              class="font-mono"
            >
              404
            </OuiText>
            <OuiBox
              bg="surface-raised"
              rounded="xl"
              p="xl"
              border="2"
              borderColor="muted"
              shadow="lg"
              class="relative"
            >
              <OuiFlex align="center" justify="center" gap="md">
                <MagnifyingGlassIcon class="h-12 w-12 text-muted" />
                <OuiText size="xl" weight="semibold" color="muted">
                  Page Not Found
                </OuiText>
              </OuiFlex>
            </OuiBox>
          </OuiStack>
        </OuiBox>

        <!-- Content -->
        <OuiStack gap="lg" align="center" class="max-w-xl">
          <OuiText as="h2" size="3xl" weight="bold" color="primary">
            Oops! This page doesn't exist
          </OuiText>
          <OuiText size="md" color="secondary" class="max-w-md">
            The page you're looking for might have been moved, deleted, or doesn't
            exist. Let's get you back on track.
          </OuiText>
        </OuiStack>

        <!-- Quick Links -->
        <OuiCard variant="outline" class="max-w-md w-full">
          <OuiCardHeader>
            <OuiText size="sm" weight="semibold" color="muted">
              Quick Links
            </OuiText>
          </OuiCardHeader>
          <OuiCardBody>
            <OuiStack gap="sm">
              <OuiBox
                as="component"
                :is="NuxtLink"
                to="/"
                class="block p-3 rounded-md border border-border-muted text-decoration-none transition-all duration-150 hover:bg-surface-muted hover:border-primary hover:translate-x-1"
              >
                <OuiFlex align="center" justify="between" gap="md">
                  <OuiFlex align="center" gap="sm">
                    <HomeIcon class="h-5 w-5 text-primary" />
                    <OuiText weight="medium">Home</OuiText>
                  </OuiFlex>
                  <ArrowRightIcon class="h-4 w-4 text-secondary" />
                </OuiFlex>
              </OuiBox>
              <OuiBox
                as="component"
                :is="NuxtLink"
                to="/dashboard"
                class="block p-3 rounded-md border border-border-muted text-decoration-none transition-all duration-150 hover:bg-surface-muted hover:border-primary hover:translate-x-1"
              >
                <OuiFlex align="center" justify="between" gap="md">
                  <OuiFlex align="center" gap="sm">
                    <RocketLaunchIcon class="h-5 w-5 text-success" />
                    <OuiText weight="medium">Dashboard</OuiText>
                  </OuiFlex>
                  <ArrowRightIcon class="h-4 w-4 text-secondary" />
                </OuiFlex>
              </OuiBox>
              <OuiBox
                as="component"
                :is="NuxtLink"
                to="/docs"
                class="block p-3 rounded-md border border-border-muted text-decoration-none transition-all duration-150 hover:bg-surface-muted hover:border-primary hover:translate-x-1"
              >
                <OuiFlex align="center" justify="between" gap="md">
                  <OuiFlex align="center" gap="sm">
                    <BookOpenIcon class="h-5 w-5 text-warning" />
                    <OuiText weight="medium">Documentation</OuiText>
                  </OuiFlex>
                  <ArrowRightIcon class="h-4 w-4 text-secondary" />
                </OuiFlex>
              </OuiBox>
              <OuiBox
                as="component"
                :is="NuxtLink"
                to="/support"
                class="block p-3 rounded-md border border-border-muted text-decoration-none transition-all duration-150 hover:bg-surface-muted hover:border-primary hover:translate-x-1"
              >
                <OuiFlex align="center" justify="between" gap="md">
                  <OuiFlex align="center" gap="sm">
                    <ChatBubbleLeftRightIcon class="h-5 w-5 text-primary" />
                    <OuiText weight="medium">Support</OuiText>
                  </OuiFlex>
                  <ArrowRightIcon class="h-4 w-4 text-secondary" />
                </OuiFlex>
              </OuiBox>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <!-- Actions -->
        <OuiFlex gap="md" wrap="wrap" justify="center">
          <OuiButton size="lg" @click="goHome">
            <HomeIcon class="h-5 w-5 mr-2" />
            Go Home
          </OuiButton>
          <OuiButton variant="outline" size="lg" @click="goBack">
            <ArrowLeftIcon class="h-5 w-5 mr-2" />
            Go Back
          </OuiButton>
        </OuiFlex>

        <!-- Logo -->
        <OuiFlex align="center" justify="center" gap="sm" class="mt-8">
          <ObienteLogo size="sm" />
          <OuiText size="sm" color="muted">Obiente Cloud</OuiText>
        </OuiFlex>
      </OuiStack>
    </OuiContainer>
  </OuiBox>
</template>

<script setup lang="ts">
import {
  MagnifyingGlassIcon,
  HomeIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  RocketLaunchIcon,
  BookOpenIcon,
  ChatBubbleLeftRightIcon,
} from "@heroicons/vue/24/outline";
import ObienteLogo from "~/components/app/ObienteLogo.vue";

const NuxtLink = resolveComponent("NuxtLink");

const goHome = () => {
  navigateTo("/");
};

const goBack = () => {
  if (import.meta.client && window.history.length > 1) {
    window.history.back();
  } else {
    navigateTo("/");
  }
};

// Set page meta
useHead({
  title: "404 - Page Not Found",
});
</script>

