// Client-side plugin to configure highlight.js with OUI theming
export default defineNuxtPlugin({
  name: "highlight-js",
  async setup() {
    if (typeof window === "undefined") return;

    // Import highlight.js
    const hljs = await import("highlight.js");
    
    // Import and apply OUI theme utility
    const { applyOUIThemeToHighlightJS } = await import("~/utils/highlight-theme");
    
    // Apply OUI theme to highlight.js
    applyOUIThemeToHighlightJS();

    // Make highlight.js available globally
    const highlightInstance = hljs.default;
    (window as any).hljs = highlightInstance;

    // Helper function to highlight code blocks
    const highlightCodeBlocks = () => {
      if (typeof window === "undefined" || !highlightInstance) return;
      
      document.querySelectorAll("pre code").forEach((block) => {
        // Only highlight if not already highlighted
        if (!block.classList.contains("hljs")) {
          highlightInstance.highlightElement(block as HTMLElement);
        }
      });
    };

    // Highlight on initial load
    if (document.readyState === "loading") {
      document.addEventListener("DOMContentLoaded", highlightCodeBlocks);
    } else {
      // Use nextTick to ensure DOM is ready
      setTimeout(highlightCodeBlocks, 0);
    }

    // Re-apply theme when colors might change (e.g., theme switch)
    const observer = new MutationObserver(() => {
      applyOUIThemeToHighlightJS();
      
      // Re-highlight code blocks when DOM changes (for docs pages)
      setTimeout(highlightCodeBlocks, 100);
    });
    
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["style", "class"],
      childList: true,
      subtree: true,
    });

    // Also watch for route changes in Nuxt (for docs pages)
    // Use useRouter composable when available
    try {
      const { useRouter } = await import("#app");
      const router = useRouter();
      if (router) {
        router.afterEach(() => {
          setTimeout(highlightCodeBlocks, 100);
        });
      }
    } catch {
      // Fallback: if router isn't available, just rely on MutationObserver
    }
  },
});

