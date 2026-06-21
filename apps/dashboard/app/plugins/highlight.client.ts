// Client-side plugin to configure highlight.js with OUI theming
export default defineNuxtPlugin({
  name: "highlight-js",
  async setup() {
    if (typeof window === "undefined") return;

    let highlightInstance: any = null;
    let highlightLoadPromise: Promise<any> | null = null;

    const ensureHighlight = async () => {
      if (highlightInstance) return highlightInstance;
      if (!highlightLoadPromise) {
        highlightLoadPromise = Promise.all([
          import("highlight.js"),
          import("~/utils/highlight-theme"),
        ])
          .then(([hljs, theme]) => {
            highlightInstance = hljs.default || hljs;
            if (!highlightInstance && (hljs as any).highlight) {
              highlightInstance = hljs;
            }
            if (!highlightInstance) {
              throw new Error("highlight.js instance not available");
            }
            theme.applyOUIThemeToHighlightJS();
            (window as any).hljs = highlightInstance;
            return highlightInstance;
          })
          .catch((err) => {
            highlightLoadPromise = null;
            console.error("Failed to load highlight.js:", err);
            return null;
          });
      }
      return highlightLoadPromise;
    };

    // Helper function to highlight code blocks
    const highlightCodeBlocks = async () => {
      if (typeof window === "undefined") return;
      const blocks = Array.from(document.querySelectorAll("pre code:not(.hljs)"));
      if (blocks.length === 0) return;

      const highlighter = await ensureHighlight();
      if (!highlighter) return;

      blocks.forEach((block) => {
        // Only highlight if not already highlighted
        if (!block.classList.contains("hljs")) {
          // Use highlightElement if available, otherwise use highlight API
          if (highlighter.highlightElement) {
            highlighter.highlightElement(block as HTMLElement);
          } else if (highlighter.highlight) {
            const code = block.textContent || "";
            const language = block.className.match(/language-(\w+)/)?.[1] || "plaintext";
            try {
              const result = highlighter.highlight(code, { language });
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
            document.addEventListener("DOMContentLoaded", () => void highlightCodeBlocks(), { once: true });
          } else {
            void highlightCodeBlocks();
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
          setTimeout(() => void highlightCodeBlocks(), 100);
        });
      }
    } catch {
      // Router not available, skip route watching
    }
  },
});
