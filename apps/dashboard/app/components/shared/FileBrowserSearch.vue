<template>
  <div>
    <!-- Centered Search Input - shown in header when modal is closed -->
    <div
      v-if="!isSearchModalOpen"
      class="absolute left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-10"
    >
      <OuiInput
        ref="searchInputRef"
        v-model="searchQueryModel"
        placeholder="Search files..."
        size="md"
        clearable
        class="w-80"
        @keyup.enter="searchState.handleSearch"
      >
        <template #prefix>
          <MagnifyingGlassIcon class="h-5 w-5 text-text-secondary" />
        </template>
      </OuiInput>
    </div>

    <!-- Search Results Modal/Overlay -->
    <Teleport to="body">
      <Transition
        name="fade"
        @after-enter="handleModalEnter"
      >
        <div
          v-if="isSearchModalOpen"
          role="dialog"
          aria-modal="true"
          aria-labelledby="search-results-title"
          aria-describedby="search-results-description"
          class="fixed inset-0 z-50 flex items-start justify-center pt-4 sm:pt-20 pb-4 sm:pb-8 px-4 bg-black/50 backdrop-blur-sm overflow-y-auto"
          @click.self="closeSearchModal"
          @keydown.esc="closeSearchModal"
          @keydown="handleSearchModalKeydown"
        >
        <div
          ref="searchModalRef"
          tabindex="-1"
          class="w-full max-w-4xl max-h-[calc(100vh-2rem)] sm:max-h-[calc(100vh-8rem)] bg-surface-base border border-border-default rounded-lg shadow-xl flex flex-col overflow-hidden focus:outline-none my-auto"
          @click.stop
        >
          <div class="flex items-center justify-between p-4 border-b border-border-default gap-4">
            <OuiText id="search-results-title" size="lg" weight="semibold" as="h2" class="shrink-0">
              Search Results
              <span v-if="searchResultsArray.length > 0" class="text-text-secondary font-normal">
                ({{ searchResultsArray.length }} found)
              </span>
            </OuiText>
            
            <!-- Search Input in Overlay Header - same input instance -->
            <div class="flex-1 max-w-md mx-auto">
              <OuiInput
                ref="searchInputRef"
                v-model="searchQueryModel"
                placeholder="Search files..."
                size="sm"
                clearable
                class="w-full"
                autofocus
                @update:model-value="searchState.handleSearch"
                @keyup.enter="searchState.handleSearch"
              >
                <template #prefix>
                  <MagnifyingGlassIcon class="h-4 w-4 text-text-secondary" />
                </template>
              </OuiInput>
            </div>

            <OuiButton
              variant="ghost"
              size="sm"
              aria-label="Close search results"
              class="shrink-0"
              @click="closeSearchModal"
            >
              <XMarkIcon class="h-5 w-5" />
            </OuiButton>
          </div>
          <!-- Screen reader announcement -->
          <div class="sr-only" role="status" aria-live="polite" aria-atomic="true">
            <span v-if="isSearchingState">Searching for files...</span>
            <span v-else-if="searchErrorState">Search error: {{ searchErrorState }}</span>
            <span v-else-if="searchResultsArray.length === 0">No files found matching your search</span>
            <span v-else>{{ searchResultsArray.length }} file{{ searchResultsArray.length === 1 ? '' : 's' }} found. Use arrow keys to navigate, Enter to open.</span>
          </div>

          <div
            ref="searchResultsRef"
            class="flex-1 overflow-y-auto p-4"
            role="listbox"
            aria-label="Search results"
            tabindex="0"
            @keydown="handleResultsKeydown"
          >
            <div v-if="isSearchingState" class="flex items-center justify-center py-8" role="status" aria-live="polite">
              <ArrowPathIcon class="h-6 w-6 animate-spin text-primary" aria-hidden="true" />
              <OuiText size="sm" color="secondary" class="ml-2">Searching...</OuiText>
            </div>

            <div v-else-if="searchErrorState" class="flex flex-col items-center justify-center py-8" role="alert">
              <ExclamationTriangleIcon class="h-8 w-8 text-danger mb-2" aria-hidden="true" />
              <OuiText size="sm" color="danger">{{ searchErrorState }}</OuiText>
            </div>

            <div v-else-if="searchResultsArray.length === 0" class="flex flex-col items-center justify-center py-8" role="status">
              <MagnifyingGlassIcon class="h-8 w-8 text-text-tertiary mb-2" aria-hidden="true" />
              <OuiText id="search-results-description" size="sm" color="secondary">
                No files found matching "{{ searchQueryModel }}"
              </OuiText>
            </div>

            <div v-else class="space-y-1" role="group">
              <button
                v-for="(result, index) in searchResultsArray"
                :key="result.path"
                :id="`search-result-${index}`"
                type="button"
                role="option"
                :aria-selected="focusedResultIndex === index"
                :tabindex="focusedResultIndex === index ? 0 : -1"
                :class="[
                  'w-full flex items-center gap-3 p-2 rounded-md cursor-pointer transition-colors text-left',
                  'hover:bg-surface-elevated focus:bg-surface-elevated focus:outline-none focus:ring-2 focus:ring-primary',
                  focusedResultIndex === index ? 'bg-surface-elevated' : ''
                ]"
                @click="handleSearchResultClick(result)"
                @keydown.enter="handleSearchResultClick(result)"
                @keydown.space.prevent="handleSearchResultClick(result)"
              >
                <DocumentIcon
                  v-if="result.type === 'file'"
                  class="h-5 w-5 text-text-secondary shrink-0"
                  aria-hidden="true"
                />
                <CubeIcon
                  v-else-if="result.type === 'directory'"
                  class="h-5 w-5 text-text-secondary shrink-0"
                  aria-hidden="true"
                />
                <LinkIcon
                  v-else-if="result.type === 'symlink'"
                  class="h-5 w-5 text-text-secondary shrink-0"
                  aria-hidden="true"
                />
                <div class="flex-1 min-w-0">
                  <OuiText size="sm" weight="medium" class="truncate">
                    {{ result.name }}
                  </OuiText>
                  <OuiText size="xs" color="secondary" class="truncate">
                    {{ result.path }}
                  </OuiText>
                </div>
              </button>
            </div>
          </div>
        </div>
      </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onUnmounted, ref, watch } from "vue";
  import { Teleport, Transition } from "vue";
  import {
    MagnifyingGlassIcon,
    XMarkIcon,
    ArrowPathIcon,
    ExclamationTriangleIcon,
    DocumentIcon,
    CubeIcon,
    LinkIcon,
  } from "@heroicons/vue/24/outline";
  import OuiInput from "~/components/oui/Input.vue";
  import OuiButton from "~/components/oui/Button.vue";
  import OuiText from "~/components/oui/Text.vue";
  import { useFileBrowserSearch } from "~/composables/useFileBrowserSearch";
  import type { FileBrowserClientAdapter } from "~/composables/useFileBrowserClient";
  import type { ExplorerNode } from "./fileExplorerTypes";

  interface Props {
    fileBrowserClient: FileBrowserClientAdapter;
    source: { type: "container" | "volume"; volumeName?: string };
  }

  interface Emits {
    (e: "result-click", result: ExplorerNode): void;
  }

  const props = defineProps<Props>();
  const emit = defineEmits<Emits>();

  const sourceRef = computed(() => ({
    type: props.source.type,
    volumeName: props.source.volumeName,
  }));

  const searchState = useFileBrowserSearch(props.fileBrowserClient, sourceRef);

  // Create computed properties for template use
  const searchQueryModel = computed({
    get: () => searchState.searchQuery.value,
    set: (val: string) => {
      searchState.searchQuery.value = val;
    },
  });

  const searchResultsArray = computed(() => searchState.searchResults.value);
  const isSearchingState = computed(() => searchState.isSearching.value);
  const searchErrorState = computed(() => searchState.searchError.value);
  const isSearchModalOpen = computed(() => searchState.isSearchModalOpen.value);

  const searchModalRef = ref<HTMLElement | null>(null);
  const searchResultsRef = ref<HTMLElement | null>(null);
  const searchInputRef = ref<HTMLElement | null>(null);
  const focusedResultIndex = ref<number>(-1);
  let previousActiveElement: HTMLElement | null = null;
  let focusTrapCleanup: (() => void) | null = null;

  function closeSearchModal() {
    searchState.closeSearchModal();
  }

  function handleSearchResultClick(result: ExplorerNode) {
    closeSearchModal();
    emit("result-click", result);
  }

  function handleSearchModalKeydown(event: KeyboardEvent) {
    // ESC key is handled by @keydown.esc on the overlay
    if (event.key === "Escape") {
      closeSearchModal();
    }
  }

  function handleResultsKeydown(event: KeyboardEvent) {
    if (searchState.searchResults.value.length === 0) return;

    switch (event.key) {
      case "ArrowDown":
        event.preventDefault();
        focusedResultIndex.value = Math.min(
          focusedResultIndex.value + 1,
          searchResultsArray.value.length - 1
        );
        // Scroll into view
        nextTick(() => {
          const focusedElement = document.getElementById(
            `search-result-${focusedResultIndex.value}`
          );
          focusedElement?.scrollIntoView({ block: "nearest", behavior: "smooth" });
          focusedElement?.focus();
        });
        break;
      case "ArrowUp":
        event.preventDefault();
        focusedResultIndex.value = Math.max(focusedResultIndex.value - 1, 0);
        // Scroll into view
        nextTick(() => {
          const focusedElement = document.getElementById(
            `search-result-${focusedResultIndex.value}`
          );
          focusedElement?.scrollIntoView({ block: "nearest", behavior: "smooth" });
          focusedElement?.focus();
        });
        break;
      case "Home":
        event.preventDefault();
        focusedResultIndex.value = 0;
        nextTick(() => {
          const focusedElement = document.getElementById(`search-result-0`);
          focusedElement?.scrollIntoView({ block: "nearest", behavior: "smooth" });
          focusedElement?.focus();
        });
        break;
      case "End":
        event.preventDefault();
        focusedResultIndex.value = searchResultsArray.value.length - 1;
        nextTick(() => {
          const focusedElement = document.getElementById(
            `search-result-${focusedResultIndex.value}`
          );
          focusedElement?.scrollIntoView({ block: "nearest", behavior: "smooth" });
          focusedElement?.focus();
        });
        break;
    }
  }

  // Focus trap and modal focus management
  function setupFocusTrap(): (() => void) | null {
    if (!import.meta.client) return null;

    const modal = searchModalRef.value;
    if (!modal) return null;

    // Store the element that had focus before opening modal
    previousActiveElement = document.activeElement as HTMLElement;

    // Get all focusable elements in the modal
    const getFocusableElements = (): HTMLElement[] => {
      const selector = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';
      return Array.from(modal.querySelectorAll<HTMLElement>(selector))
        .filter(el => {
          return !el.hasAttribute('disabled') && 
                 !el.hasAttribute('aria-hidden') &&
                 el.offsetParent !== null; // Visible elements only
        });
    };

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;

      const focusableElements = getFocusableElements();
      if (focusableElements.length === 0) {
        e.preventDefault();
        return;
      }

      const firstElement = focusableElements[0];
      const lastElement = focusableElements[focusableElements.length - 1];
      const currentElement = document.activeElement as HTMLElement;

      if (!firstElement || !lastElement) {
        e.preventDefault();
        return;
      }

      if (e.shiftKey) {
        // Shift + Tab: go to previous
        if (currentElement === firstElement || !modal.contains(currentElement)) {
          e.preventDefault();
          lastElement.focus();
        }
      } else {
        // Tab: go to next
        if (currentElement === lastElement || !modal.contains(currentElement)) {
          e.preventDefault();
          firstElement.focus();
        }
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    // Return cleanup function
    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      // Restore focus to previous element
      if (previousActiveElement && document.contains(previousActiveElement)) {
        previousActiveElement.focus();
      }
      previousActiveElement = null;
    };
  }

  // Helper function to focus the search input
  function focusSearchInput() {
    // Use multiple strategies to find and focus the input
    const tryFocus = (attempt = 0): boolean => {
      let actualInput: HTMLInputElement | null = null;
      
      // Strategy 1: Use the ref if available
      if (searchInputRef.value) {
        // OuiInput wraps Field.Input, which renders an actual input element
        actualInput = searchInputRef.value.querySelector('input') as HTMLInputElement;
      }
      
      // Strategy 2: Search in the modal
      if (!actualInput && searchModalRef.value) {
        actualInput = searchModalRef.value.querySelector('input[type="text"], input:not([type])') as HTMLInputElement;
      }
      
      // Strategy 3: Search in the document (fallback)
      if (!actualInput) {
        const modal = document.querySelector('[role="dialog"][aria-modal="true"]');
        if (modal) {
          actualInput = modal.querySelector('input[type="text"], input:not([type])') as HTMLInputElement;
        }
      }
      
      if (actualInput) {
        // Check if element is actually focusable
        if (actualInput.offsetParent !== null) {
          try {
            actualInput.focus({ preventScroll: false });
            // Select all text if there's a query
            if (searchQueryModel.value) {
              // Small delay to ensure focus happened first
              setTimeout(() => {
                actualInput?.select();
              }, 10);
            }
            return true;
          } catch (e) {
            console.warn('Failed to focus input:', e);
          }
        }
      }
      
      // Retry up to 5 times with increasing delays
      if (attempt < 5) {
        setTimeout(() => {
          tryFocus(attempt + 1);
        }, 50 * (attempt + 1));
      }
      
      return false;
    };
    
    // Start trying to focus
    tryFocus();
  }

  // Handle modal enter transition - focus input after transition completes
  function handleModalEnter() {
    focusSearchInput();
  }

  // Auto-focus search input when modal opens
  watch(isSearchModalOpen, (isOpen) => {
    if (isOpen) {
      // Setup focus trap
      if (focusTrapCleanup) {
        focusTrapCleanup();
      }
      focusTrapCleanup = setupFocusTrap();

      // Reset focused index
      focusedResultIndex.value = -1;
      
      // Try to focus immediately, and also after transition
      nextTick(() => {
        if (typeof requestAnimationFrame !== 'undefined') {
          requestAnimationFrame(() => {
            focusSearchInput();
          });
        } else {
          focusSearchInput();
        }
      });
    } else {
      // Cleanup focus trap when modal closes
      if (focusTrapCleanup) {
        focusTrapCleanup();
        focusTrapCleanup = null;
      }
    }
  });

  // Cleanup on unmount
  onUnmounted(() => {
    if (focusTrapCleanup) {
      focusTrapCleanup();
    }
  });
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
</style>
