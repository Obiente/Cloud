// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from "@tailwindcss/vite";

export default defineNuxtConfig({
  // Disable devtools in production builds
  devtools: { enabled: process.env.NODE_ENV !== "production" },
  vite: {
    // @ts-ignore - Type conflict between Vite versions (@types/node@20 vs @types/node@24) in dependency tree
    plugins: [tailwindcss()],
    server: {
      hmr: {
        port: 24678, // Use a different port for HMR
      },
      watch: {
        ignored: [
          "**/node_modules/**",
          "**/.git/**",
          "**/packages/**",
          "**/dist/**",
          "**/.nuxt/**",
          "**/apps/api/**",
          "**/apps/*/dist/**",
          "**/scripts/**",
          "**/tools/**",
          "**/.nx/**",
        ],
      },
    },
    build: {
      // Enable minification and tree-shaking
      minify: "terser",
      terserOptions: {
        compress: {
          drop_console: process.env.NODE_ENV === "production", // Remove console.log in production
          drop_debugger: true,
          pure_funcs: process.env.NODE_ENV === "production" ? ["console.log", "console.info"] : [],
        },
      },
      // Optimize chunk splitting for better code splitting
      rollupOptions: {
        output: {
          // Separate vendor chunks for better caching
          manualChunks: (id) => {
            // Separate heavy libraries into their own chunks
            if (id.includes("monaco-editor")) {
              return "monaco";
            }
            if (id.includes("echarts") || id.includes("vue-echarts")) {
              return "echarts";
            }
            if (id.includes("@xterm")) {
              return "xterm";
            }
            if (id.includes("highlight.js")) {
              return "highlight";
            }
            if (id.includes("prettier")) {
              return "prettier";
            }
            // Separate Vue ecosystem
            if (id.includes("vue") || id.includes("pinia") || id.includes("@vueuse")) {
              return "vue-vendor";
            }
            // Separate connectrpc
            if (id.includes("@connectrpc") || id.includes("@bufbuild")) {
              return "grpc";
            }
            // Separate node_modules into vendor chunk
            if (id.includes("node_modules")) {
              return "vendor";
            }
          },
          // Optimize chunk file names for better caching
          chunkFileNames: "js/[name]-[hash].js",
          entryFileNames: "js/[name]-[hash].js",
          assetFileNames: (assetInfo) => {
            if (assetInfo.name?.endsWith(".css")) {
              return "css/[name]-[hash][extname]";
            }
            return "assets/[name]-[hash][extname]";
          },
        },
      },
      // Increase chunk size warning limit (some heavy libraries are legitimately large)
      chunkSizeWarningLimit: 1000,
    },
    // Optimize dependencies
    optimizeDeps: {
      include: [
        "vue",
        "pinia",
        "@vueuse/core",
        "@vueuse/nuxt",
        "@pinia/nuxt",
        "@connectrpc/connect-web",
        "highlight.js", // Include highlight.js so Vite can properly optimize it
      ],
      // Exclude heavy libraries from pre-bundling (they're lazy-loaded)
      exclude: ["monaco-editor", "@xterm/xterm", "echarts", "prettier"],
    },
  },
  // Modules
  modules: ["@pinia/nuxt", "@vueuse/nuxt", "@nuxt/icon"],

  // CSS Framework - using Nuxt UI (built on Tailwind CSS)
  css: ["~/assets/css/main.css"],
  // TypeScript configuration
  // Disable type checking during Docker builds for faster builds
  // Type checking can be done separately via `pnpm typecheck` or in CI
  typescript: {
    typeCheck: process.env.SKIP_TYPE_CHECK !== "true",
  },

  // Runtime config
  runtimeConfig: {
    // Private keys (only available on server-side)
    apiSecret: "",
    githubClientSecret: process.env.GITHUB_CLIENT_SECRET || "", // Server-side only - never expose to client
    session: {
      password:
         "changeme_" + crypto.randomUUID(), // CHANGE THIS IN PRODUCTION, should be at least 32 characters
      cookie: {
        secure: true, // Set to true if using HTTPS
      },
    },
    // Public keys (exposed to client-side)
    public: {
      requestHost: "http://localhost:3000",
      apiHost: "http://localhost:3001",
      oidcIssuer: "https://obiente.cloud",
      oidcBase: "https://auth.obiente.cloud",
      oidcClientId: "339499954043158530",
      githubClientId: "",
      disableAuth: false,
      stripePublishableKey: "",
    },
  },

  // SSR configuration
  ssr: true,

  // // Auto-import configuration
  // imports: {
  //   dirs: ['composables/**', 'stores/**', 'utils/**'],
  // },

  // Development server
  devServer: {
    port: 3000,
    host: "0.0.0.0",
  },

  // App configuration
  app: {
    head: {
      title: "Obiente Cloud",
      meta: [
        { charset: "utf-8" },
        { name: "viewport", content: "width=device-width, initial-scale=1" },
        {
          name: "description",
          content: "Multi-tenant cloud dashboard platform",
        },
      ],
      link: [{ rel: "icon", type: "image/x-icon", href: "/favicon.ico" }],
    },
  },

  // Nitro configuration (for server-side)
  nitro: {
    experimental: {
      wasm: true,
      websocket: true,
    },
    // Enable compression for production builds
    compressPublicAssets: process.env.NODE_ENV === "production",
    // Optimize prerendering
    prerender: {
      crawlLinks: false, // Disable link crawling for faster builds
      concurrency: 1,
    },
    // Minify server output
    minify: process.env.NODE_ENV === "production",
  },

  // Build optimization
  experimental: {
    payloadExtraction: true, // Extract payloads for better caching
  },
});
