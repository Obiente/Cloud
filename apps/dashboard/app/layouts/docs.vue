<template>
  <div class="bg-surface-base h-screen flex flex-col overflow-hidden">
    <header
      class="sticky top-0 z-30 border-b border-border-muted bg-surface-base/95 backdrop-blur-sm"
    >
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
                <OuiText size="xl" weight="bold" color="primary"
                  >Obiente</OuiText
                >
                <OuiText size="xs" color="tertiary">Documentation</OuiText>
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
            <template v-else>
              <OuiButton
                variant="outline"
                size="sm"
                @click="user.popupSignup()"
              >
                Sign Up
              </OuiButton>
              <OuiButton variant="ghost" size="sm" @click="user.popupLogin()">
                Sign In
              </OuiButton>
            </template>
          </OuiFlex>
        </OuiFlex>
      </OuiContainer>
    </header>

    <main class="flex-1 min-h-0 flex overflow-hidden">
      <div
        class="hidden lg:block lg:w-64 lg:shrink-0 lg:min-h-0 bg-surface-base border-r border-border-muted"
      >
        <div class="h-full">
          <DocsSidebar />
        </div>
      </div>

      <div class="lg:hidden border-b border-border-muted">
        <OuiContainer size="full" py="sm">
          <OuiFlex align="center" gap="sm" wrap="wrap">
            <OuiButton
              variant="ghost"
              size="sm"
              @click="isMobileSidebarOpen = !isMobileSidebarOpen"
              class="gap-2"
              :aria-expanded="isMobileSidebarOpen"
              aria-controls="docs-mobile-sidebar"
            >
              <Bars3Icon class="h-4 w-4" />
              {{ isMobileSidebarOpen ? "Hide" : "Show" }} Navigation
            </OuiButton>
          </OuiFlex>
        </OuiContainer>
      </div>

      <Transition name="fade">
        <div
          v-if="isMobileSidebarOpen"
          class="lg:hidden fixed inset-0 z-40 bg-background/80 backdrop-blur-sm"
          @click="isMobileSidebarOpen = false"
        />
      </Transition>

      <Transition name="slide">
        <aside
          v-if="isMobileSidebarOpen"
          id="docs-mobile-sidebar"
          class="lg:hidden fixed inset-y-0 left-0 z-50 w-72 max-w-[80vw] border-r border-border-muted bg-surface-base shadow-2xl overflow-y-auto"
          style="top: 4rem"
        >
          <DocsSidebar @navigate="isMobileSidebarOpen = false" />
        </aside>
      </Transition>

      <div class="flex-1 min-w-0 min-h-0 flex flex-col">
        <div class="flex-1 overflow-y-auto">
          <OuiContainer size="full" py="xl">
            <OuiStack gap="lg">
              <OuiFlex
                v-if="currentDoc"
                align="center"
                justify="between"
                wrap="wrap"
                gap="sm"
                class="border-b border-border-muted pb-4"
              >
                <OuiStack gap="xs">
                  <OuiText size="xs" transform="uppercase" color="tertiary">
                    Documentation
                  </OuiText>
                  <OuiText size="sm" color="tertiary">
                    {{ currentDoc.label }}
                  </OuiText>
                </OuiStack>
                <OuiFlex gap="sm" wrap="wrap">
                  <OuiButton
                    v-if="previousDoc"
                    variant="ghost"
                    size="sm"
                    @click="navigateTo(previousDoc.path)"
                  >
                    <ChevronLeftIcon class="h-4 w-4" />
                    {{ previousDoc.label }}
                  </OuiButton>
                  <OuiButton
                    v-if="nextDoc"
                    variant="ghost"
                    size="sm"
                    @click="navigateTo(nextDoc.path)"
                  >
                    {{ nextDoc.label }}
                    <ChevronRightIcon class="h-4 w-4" />
                  </OuiButton>
                </OuiFlex>
              </OuiFlex>

              <slot />
            </OuiStack>
          </OuiContainer>
        </div>

        <footer class="border-t border-border-muted bg-surface-subtle shrink-0">
          <OuiContainer size="full" py="md">
            <OuiStack gap="sm" align="center">
              <OuiFlex
                v-if="previousDoc || nextDoc"
                align="center"
                gap="sm"
                wrap="wrap"
                justify="center"
              >
                <NuxtLink
                  v-if="previousDoc"
                  :to="previousDoc.path"
                  class="text-sm text-secondary hover:text-primary transition-colors"
                >
                  Previous: {{ previousDoc.label }}
                </NuxtLink>
                <OuiText
                  v-if="previousDoc && nextDoc"
                  size="sm"
                  color="tertiary"
                >
                  •
                </OuiText>
                <NuxtLink
                  v-if="nextDoc"
                  :to="nextDoc.path"
                  class="text-sm text-secondary hover:text-primary transition-colors"
                >
                  Next: {{ nextDoc.label }}
                </NuxtLink>
              </OuiFlex>

              <OuiFlex align="center" gap="sm" wrap="wrap" justify="center">
                <OuiText size="xs" color="tertiary">
                  © {{ new Date().getFullYear() }} Obiente Cloud
                </OuiText>
                <OuiText size="xs" color="tertiary">•</OuiText>
                <NuxtLink
                  to="/support"
                  class="text-xs text-secondary hover:text-primary transition-colors"
                >
                  Support
                </NuxtLink>
              </OuiFlex>
            </OuiStack>
          </OuiContainer>
        </footer>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import type DocsSidebar from "~/components/docs/DocsSidebar.vue";
import {
  ArrowLeftIcon,
  Bars3Icon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "@heroicons/vue/24/outline";
import { getDocsCurrentItem, getDocsNeighbors } from "~/utils/docsNavigation";

const user = useAuth();
const route = useRoute();
const isMobileSidebarOpen = ref(false);

const currentDoc = computed(() => getDocsCurrentItem(route.path));
const previousDoc = computed(() => getDocsNeighbors(route.path).previous);
const nextDoc = computed(() => getDocsNeighbors(route.path).next);

watch(
  () => route.path,
  () => {
    isMobileSidebarOpen.value = false;
  }
);
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.slide-enter-active,
.slide-leave-active {
  transition: transform 0.3s ease;
}

.slide-enter-from,
.slide-leave-to {
  transform: translateX(-100%);
}
</style>
