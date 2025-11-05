<template>
  <div class="bg-surface-base min-h-screen">
    <!-- Header -->
    <header class="sticky top-0 z-30 border-b border-border-muted bg-surface-base/95 backdrop-blur-sm">
      <OuiContainer size="full" py="md">
        <OuiFlex align="center" justify="between" gap="md" wrap="wrap">
          <OuiFlex align="center" gap="md">
            <NuxtLink to="/" class="flex items-center gap-2 no-underline">
              <OuiBox
                class="w-8 h-8 bg-primary rounded-xl flex items-center justify-center"
              >
                <OuiText size="lg" weight="bold" color="primary">O</OuiText>
              </OuiBox>
              <OuiStack gap="none" class="leading-tight">
                <OuiText size="xl" weight="bold" color="primary">Obiente</OuiText>
                <OuiText size="xs" color="secondary">Documentation</OuiText>
              </OuiStack>
            </NuxtLink>
          </OuiFlex>
          
          <OuiFlex align="center" gap="sm" wrap="wrap">
            <OuiButton
              variant="ghost"
              size="sm"
              @click="navigateTo('/dashboard')"
              class="gap-2"
            >
              <ArrowLeftIcon class="h-4 w-4" />
              Back to Dashboard
            </OuiButton>
            <OuiButton
              v-if="user.isAuthenticated"
              variant="ghost"
              size="sm"
              @click="navigateTo('/dashboard')"
            >
              Dashboard
            </OuiButton>
            <OuiButton
              v-else
              variant="ghost"
              size="sm"
              @click="user.popupLogin()"
            >
              Sign In
            </OuiButton>
          </OuiFlex>
        </OuiFlex>
      </OuiContainer>
    </header>

    <!-- Main Content -->
    <main class="min-h-[calc(100vh-4rem)] flex">
      <!-- Desktop Sidebar -->
      <div class="hidden lg:block lg:w-64 lg:shrink-0">
        <div class="sticky top-[4rem] h-[calc(100vh-4rem)] bg-surface-base border-r border-border-muted">
          <DocsSidebar />
        </div>
      </div>

      <!-- Mobile Sidebar Toggle -->
      <div class="lg:hidden border-b border-border-muted">
        <OuiContainer size="full" py="sm">
          <OuiButton
            variant="ghost"
            size="sm"
            @click="isMobileSidebarOpen = !isMobileSidebarOpen"
            class="gap-2"
          >
            <Bars3Icon class="h-4 w-4" />
            {{ isMobileSidebarOpen ? 'Hide' : 'Show' }} Navigation
          </OuiButton>
        </OuiContainer>
      </div>

      <!-- Mobile Sidebar Overlay -->
      <Transition name="fade">
        <div
          v-if="isMobileSidebarOpen"
          class="lg:hidden fixed inset-0 z-40 bg-background/80 backdrop-blur-sm"
          @click="isMobileSidebarOpen = false"
        />
      </Transition>

      <!-- Mobile Sidebar -->
      <Transition name="slide">
        <aside
          v-if="isMobileSidebarOpen"
          class="lg:hidden fixed inset-y-0 left-0 z-50 w-72 max-w-[80vw] border-r border-border-muted bg-surface-base shadow-2xl overflow-y-auto"
          style="top: 4rem;"
        >
          <DocsSidebar @navigate="isMobileSidebarOpen = false" />
        </aside>
      </Transition>

      <!-- Content Area -->
      <div class="flex-1 overflow-y-auto">
        <OuiContainer size="full" py="xl">
          <slot />
        </OuiContainer>
      </div>
    </main>

    <!-- Footer -->
    <footer class="border-t border-border-muted bg-surface-subtle mt-auto">
      <OuiContainer size="full" py="xl">
        <OuiStack gap="md" align="center">
          <OuiFlex align="center" gap="sm" wrap="wrap" justify="center">
            <OuiText size="sm" color="secondary">
              © {{ new Date().getFullYear() }} Obiente Cloud
            </OuiText>
            <OuiText size="sm" color="secondary">•</OuiText>
            <NuxtLink to="/support" class="text-sm text-secondary hover:text-primary transition-colors">
              Support
            </NuxtLink>
          </OuiFlex>
          <OuiText size="xs" color="secondary" class="text-center max-w-2xl">
            This documentation is publicly accessible. Sign in to access your dashboard and manage your resources.
          </OuiText>
        </OuiStack>
      </OuiContainer>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ArrowLeftIcon, Bars3Icon } from "@heroicons/vue/24/outline";

// Use auth composable but don't require authentication
const user = useAuth();

const isMobileSidebarOpen = ref(false);
</script>

<style scoped>
/* Fade transition for overlay */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Slide transition for sidebar */
.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease;
}

.slide-enter-from {
  transform: translateX(-100%);
}

.slide-leave-to {
  transform: translateX(-100%);
}
</style>

