import { ref, computed, watch, type Ref } from "vue";
import type { ExplorerNode } from "~/components/shared/fileExplorerTypes";
import type { FileBrowserClientAdapter } from "./useFileBrowserClient";

export function useFileBrowserSearch(
  fileBrowserClient: FileBrowserClientAdapter,
  source: Ref<{ type: "container" | "volume"; volumeName?: string }>
) {
  // Initialize all state explicitly to ensure clean initial state
  const searchQuery = ref<string>("");
  const searchResults = ref<ExplorerNode[]>([]);
  const isSearching = ref<boolean>(false);
  const searchError = ref<string | null>(null);
  const hasCompletedSearch = ref<boolean>(false); // Track if a search has completed
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  const isSearchModalOpen = computed<boolean>(() => {
    // CRITICAL: Modal should NEVER be open without a query
    // This is the PRIMARY and ONLY gate - if no query, modal is closed
    const queryValue = searchQuery.value;
    
    // Early return if query is falsy, not a string, or empty
    if (!queryValue || typeof queryValue !== "string") {
      return false;
    }
    
    const trimmedQuery = queryValue.trim();
    if (trimmedQuery.length === 0) {
      return false;
    }
    
    // If we have a query, show the modal if:
    // - We're currently searching, OR
    // - We have results, OR  
    // - We have an error, OR
    // - We've finished searching (even if no results - user should see "no results" message)
    const results = searchResults.value;
    const hasResults = Array.isArray(results) && results.length > 0;
    const currentlySearching = isSearching.value === true;
    const errorValue = searchError.value;
    const hasError = Boolean(
      errorValue && 
      typeof errorValue === "string" && 
      errorValue.trim().length > 0
    );
    
    // Show modal if we have a query AND (searching, has results, has error, or search completed)
    // Once a search has been initiated, keep modal open to show results (even if empty)
    return currentlySearching || hasResults || hasError || hasCompletedSearch.value;
  });

  async function handleSearch() {
    const query = searchQuery.value.trim();

    if (!query) {
      searchResults.value = [];
      searchError.value = null;
      return;
    }

    if (!fileBrowserClient.searchFiles) {
      searchError.value = "Search is not available";
      return;
    }

    isSearching.value = true;
    searchError.value = null;

    try {
      const response = await fileBrowserClient.searchFiles({
        query,
        rootPath: "/",
        volumeName: source.value.type === "volume" ? source.value.volumeName : undefined,
        maxResults: 100,
      });

      searchResults.value = response.results;
      hasCompletedSearch.value = true;
    } catch (err: any) {
      console.error("Search failed:", err);
      searchError.value = err?.message || "Failed to search files";
      searchResults.value = [];
      hasCompletedSearch.value = true;
    } finally {
      isSearching.value = false;
    }
  }

  function closeSearchModal() {
    // Clear any pending search
    if (searchTimeout) {
      clearTimeout(searchTimeout);
      searchTimeout = null;
    }
    
    // Clear all state - query first (this closes the modal)
    searchQuery.value = "";
    isSearching.value = false;
    searchResults.value = [];
    searchError.value = null;
    hasCompletedSearch.value = false;
  }

  // Watch for search query changes with debounce
  // Don't trigger on initial mount - only watch for actual changes
  watch(
    searchQuery,
    (newQuery, oldQuery) => {
      // Clear any pending search timeout
      if (searchTimeout) {
        clearTimeout(searchTimeout);
        searchTimeout = null;
      }

      const trimmedQuery = (newQuery || "").trim();
    if (!trimmedQuery) {
      // Clear everything when query is empty
      searchResults.value = [];
      searchError.value = null;
      isSearching.value = false;
      hasCompletedSearch.value = false;
      return;
    }

      // Debounce search by 500ms
      searchTimeout = setTimeout(() => {
        handleSearch();
        searchTimeout = null;
      }, 500);
    },
    { immediate: false }
  );

  return {
    searchQuery,
    searchResults,
    isSearching,
    searchError,
    isSearchModalOpen,
    handleSearch,
    closeSearchModal,
  };
}

