// Client-side plugin to configure Monaco Editor web workers
// This must run before Monaco Editor is imported anywhere
export default defineNuxtPlugin({
  name: "monaco-worker-config",
  enforce: "pre", // Run early, before Monaco is imported
  async setup() {
    if (typeof window === "undefined") return;

    // Import worker files - Vite will handle bundling them correctly
    const [
      EditorWorker,
      JsonWorker,
      CssWorker,
      HtmlWorker,
      TsWorker,
    ] = await Promise.all([
      import("monaco-editor/esm/vs/editor/editor.worker?worker"),
      import("monaco-editor/esm/vs/language/json/json.worker?worker"),
      import("monaco-editor/esm/vs/language/css/css.worker?worker"),
      import("monaco-editor/esm/vs/language/html/html.worker?worker"),
      import("monaco-editor/esm/vs/language/typescript/ts.worker?worker"),
    ]);

    // Configure Monaco environment before any Monaco imports
    // @ts-ignore - MonacoEnvironment is set globally
    (window as any).MonacoEnvironment = {
      getWorker: function (_workerId: string, label: string) {
        // Return appropriate worker based on language label
        // Vite's ?worker imports return a constructor function
        if (label === "json") {
          return new (JsonWorker as any).default();
        }
        if (label === "css" || label === "scss" || label === "less") {
          return new (CssWorker as any).default();
        }
        if (label === "html" || label === "handlebars" || label === "razor") {
          return new (HtmlWorker as any).default();
        }
        if (label === "typescript" || label === "javascript") {
          return new (TsWorker as any).default();
        }
        // Default editor worker for plaintext and other languages
        return new (EditorWorker as any).default();
      },
    };
  },
});

