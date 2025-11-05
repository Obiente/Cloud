import { ref, watch, computed, type Ref, type ComputedRef } from "vue";
import { useRoute, useRouter } from "vue-router";
import type { TabItem } from "~/components/oui/Tabs.vue";

/**
 * Composable for managing tab state with URL query parameters.
 * Syncs the active tab with the URL query parameter for shareable links and browser navigation.
 *
 * @param tabs - A ref, computed ref, or function that returns an array of available tabs
 * @param defaultTab - The default tab ID to use when no query param is present (default: "overview")
 * @param queryParamName - The name of the query parameter (default: "tab")
 * @returns A ref for the active tab ID
 */
export function useTabQuery(
  tabs: Ref<TabItem[]> | ComputedRef<TabItem[]> | (() => TabItem[]),
  defaultTab: string = "overview",
  queryParamName: string = "tab"
) {
  const route = useRoute();
  const router = useRouter();

  // Convert tabs to a computed if it's a function, otherwise use as-is
  const tabsRef = typeof tabs === "function" ? computed(tabs) : tabs;

  /**
   * Get initial tab from query parameter or default to the provided default tab
   */
  const getInitialTab = (): string => {
    const tabParam = route.query[queryParamName];
    if (typeof tabParam === "string") {
      // Validate that the tab exists
      const tabIds = tabsRef.value.map((t: TabItem) => t.id);
      return tabIds.includes(tabParam) ? tabParam : defaultTab;
    }
    return defaultTab;
  };

  const activeTab = ref(getInitialTab());

  // Watch for tab changes and update query parameter
  watch(activeTab, (newTab) => {
    const availableTabs = tabsRef.value.map((t: TabItem) => t.id);
    // If tab is removed from available tabs (e.g., conditional tab hidden), switch to default
    if (!availableTabs.includes(newTab)) {
      activeTab.value = defaultTab;
      return;
    }
    if (route.query[queryParamName] !== newTab) {
      router.replace({
        query: {
          ...route.query,
          [queryParamName]: newTab === defaultTab ? undefined : newTab, // Remove query param for default tab
        },
      });
    }
  });

  // Watch for query parameter changes (e.g., back/forward navigation)
  watch(
    () => route.query[queryParamName],
    (tabParam) => {
      if (typeof tabParam === "string") {
        const tabIds = tabsRef.value.map((t: TabItem) => t.id);
        if (tabIds.includes(tabParam) && activeTab.value !== tabParam) {
          activeTab.value = tabParam;
        }
      } else if (!tabParam && activeTab.value !== defaultTab) {
        activeTab.value = defaultTab;
      }
    }
  );

  return activeTab;
}

