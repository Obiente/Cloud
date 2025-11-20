// Client-side plugin to configure highlight.js with OUI theming
export default defineNuxtPlugin({
  name: "highlight-js",
  async setup() {
    if (typeof window === "undefined") return;

    // Import highlight.js - handle both default and named exports for v11 compatibility
    let highlightInstance: any;
    try {
      const hljs = await import("highlight.js");
      // highlight.js v11 uses default export
      highlightInstance = hljs.default || hljs;
      
      // If it's still not available, try named imports
      if (!highlightInstance && (hljs as any).highlight) {
        highlightInstance = hljs;
      }
    } catch (err) {
      console.error("Failed to load highlight.js:", err);
      return;
    }
    
    if (!highlightInstance) {
      console.error("highlight.js instance not available");
      return;
    }
    
    // Import and apply OUI theme utility
    const { applyOUIThemeToHighlightJS } = await import("~/utils/highlight-theme");
    
    // Apply OUI theme to highlight.js
    applyOUIThemeToHighlightJS();

    // Make highlight.js available globally
    (window as any).hljs = highlightInstance;

    // Helper function to highlight code blocks
    const highlightCodeBlocks = () => {
      if (typeof window === "undefined" || !highlightInstance) return;
      
      document.querySelectorAll("pre code").forEach((block) => {
        // Only highlight if not already highlighted
        if (!block.classList.contains("hljs")) {
          // Use highlightElement if available, otherwise use highlight API
          if (highlightInstance.highlightElement) {
            highlightInstance.highlightElement(block as HTMLElement);
          } else if (highlightInstance.highlight) {
            const code = block.textContent || "";
            const language = block.className.match(/language-(\w+)/)?.[1] || "plaintext";
            try {
              const result = highlightInstance.highlight(code, { language });
              block.innerHTML = result.value;
              block.classList.add("hljs");
            } catch {
              // Fallback: just add the class
              block.classList.add("hljs");
            }
          }
        }
      });
    };

    // Highlight on initial load - defer until after hydration
    if (import.meta.client) {
      // Use double requestAnimationFrame to ensure hydration is complete
      requestAnimationFrame(() => {
        requestAnimationFrame(() => {
          if (document.readyState === "loading") {
            document.addEventListener("DOMContentLoaded", highlightCodeBlocks);
          } else {
            highlightCodeBlocks();
          }
        });
      });
    }

    // Watch for route changes in Nuxt (for docs pages)
    // MutationObserver removed as it was causing performance issues (freezing on Chrome)
    // Route-based highlighting is sufficient for most use cases
    try {
      const { useRouter } = await import("#app");
      const router = useRouter();
      if (router) {
        router.afterEach(() => {
          setTimeout(highlightCodeBlocks, 100);
        });
      }
    } catch {
      // Router not available, skip route watching
    }
  },
});

