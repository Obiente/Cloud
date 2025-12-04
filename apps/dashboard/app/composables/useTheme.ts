import { computed, watch } from "vue";
import { usePreferencesStore } from "~/stores/preferences";

export type Theme = "dark" | "dark-purple" | "extra-dark";

const THEME_COOKIE_KEY = "obiente_theme";
const THEME_STORAGE_KEY = "obiente_theme";
const DEFAULT_THEME: Theme = "dark";

/**
 * Composable for managing OUI theme switching
 */
export function useTheme() {
  const preferencesStore = usePreferencesStore();

  // Use cookie for SSR compatibility - available on both server and client
  const themeCookie = useCookie<Theme>(THEME_COOKIE_KEY, {
    default: () => DEFAULT_THEME,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    httpOnly: false, // Needs to be accessible from client-side JS
    maxAge: 60 * 60 * 24 * 365, // 1 year
  });

  // Get theme from cookie (SSR-compatible), preferences store, or localStorage
  const currentTheme = computed<Theme>(() => {
    // Cookie is source of truth for SSR
    if (themeCookie.value && (themeCookie.value === "dark" || themeCookie.value === "dark-purple" || themeCookie.value === "extra-dark")) {
      return themeCookie.value;
    }

    if (!import.meta.client) {
      return DEFAULT_THEME;
    }

    // Try to get from preferences store first
    const themeFromStore = preferencesStore.preferences.theme as Theme | undefined;
    if (themeFromStore) {
      return themeFromStore;
    }

    // Fallback to localStorage
    try {
      const stored = localStorage.getItem(THEME_STORAGE_KEY);
      if (stored && (stored === "dark" || stored === "dark-purple" || stored === "extra-dark")) {
        return stored as Theme;
      }
    } catch (error) {
      console.warn("[useTheme] Failed to read theme from localStorage:", error);
    }

    return DEFAULT_THEME;
  });

  /**
   * Set the theme and persist it
   */
  function setTheme(theme: Theme) {
    // Always update cookie (works on both server and client for SSR)
    themeCookie.value = theme;

    if (!import.meta.client) {
      return;
    }

    // Update preferences store if hydrated
    if (preferencesStore.hydrated) {
      preferencesStore.setTheme(theme);
    } else {
      // Fallback to localStorage if store not hydrated yet
      try {
        localStorage.setItem(THEME_STORAGE_KEY, theme);
      } catch (error) {
        console.warn("[useTheme] Failed to save theme to localStorage:", error);
      }
    }

    // Apply theme to document
    applyTheme(theme);
  }

  /**
   * Apply theme to the document root element
   */
  function applyTheme(theme: Theme) {
    if (!import.meta.client) {
      return;
    }

    const root = document.documentElement;
    root.setAttribute("data-theme", theme);
  }

  /**
   * Initialize theme on mount
   */
  function initializeTheme() {
    const theme = currentTheme.value;
    
    // Apply theme to document (works on both server and client)
    if (import.meta.client) {
      applyTheme(theme);
    } else {
      // On server, set the data-theme attribute via useHead
      useHead({
        htmlAttrs: {
          "data-theme": theme,
        },
      });
    }
  }

  // Watch for theme changes and apply them
  watch(
    currentTheme,
    (newTheme) => {
      applyTheme(newTheme);
    },
    { immediate: true }
  );

  // Watch for preferences store hydration to sync theme
  watch(
    () => {
      // Access the computed value - hydrated is a ComputedRef<boolean>
      const h = preferencesStore.hydrated;
      return h;
    },
    (hydrated: boolean) => {
      if (hydrated && import.meta.client) {
        // Sync cookie to store if needed
        const cookieTheme = themeCookie.value;
        const themeFromStore = preferencesStore.preferences.theme as Theme | undefined;
        if (cookieTheme && (!themeFromStore || themeFromStore !== cookieTheme)) {
          preferencesStore.setTheme(cookieTheme);
        }
        // Also sync localStorage to cookie if needed
        try {
          const stored = localStorage.getItem(THEME_STORAGE_KEY);
          if (stored && (stored === "dark" || stored === "dark-purple" || stored === "extra-dark")) {
            if (!cookieTheme || cookieTheme !== stored) {
              themeCookie.value = stored as Theme;
            }
          }
        } catch (error) {
          console.warn("[useTheme] Failed to sync theme from localStorage:", error);
        }
      }
    }
  );

  return {
    currentTheme,
    setTheme,
    initializeTheme,
    themes: ["dark", "dark-purple", "extra-dark"] as const,
  };
}

