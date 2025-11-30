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
        usePolling: true, // Use polling to avoid EMFILE errors
        interval: 1000, // Poll every 1 second
        ignored: [
          /node_modules/,
          /\.git/,
          /packages/,
          /dist/,
          /apps\/api/,
          /scripts/,
          /tools/,
          /\.nx/,
          /tsconfig\.tsbuildinfo/,
          /\.log$/,
          /\.turbo/,
        ],
      },
    },
    build: {
      // Optimize build for memory usage and speed
      chunkSizeWarningLimit: 1000,
      minify: "esbuild", // Use esbuild instead of terser for lower memory usage and faster builds
      // Enable build optimizations
      cssMinify: "esbuild", // Use esbuild for CSS minification (faster than default)
      // Optimize chunk splitting for better caching and parallel loading
      target: "esnext", // Target modern browsers for smaller bundles
      // Reduce source map generation in production for faster builds
      sourcemap: process.env.NODE_ENV === "development",
      rollupOptions: {
        output: {
          // Improved code splitting strategy for better performance
          manualChunks: (id) => {
            // Split large, independent dependencies into separate chunks
            if (id.includes("node_modules")) {
              // Monaco Editor - very large editor dependency (~2MB)
              if (id.includes("monaco-editor")) {
                return "monaco";
              }
              // ECharts - large charting library (~500KB+)
              if (id.includes("echarts") || id.includes("vue-echarts")) {
                return "echarts";
              }
              // XTerm - terminal emulator (~200KB+)
              if (id.includes("@xterm")) {
                return "xterm";
              }
              // Highlight.js - syntax highlighting (~300KB+)
              if (id.includes("highlight.js")) {
                return "highlight";
              }
              // JSZip - file compression library
              if (id.includes("jszip")) {
                return "jszip";
              }
              // Connect RPC - API client library
              if (id.includes("@connectrpc") || id.includes("@bufbuild")) {
                return "connect";
              }
              // Heroicons - icon library (used everywhere)
              if (id.includes("@heroicons")) {
                return "icons";
              }
              // VueUse - utility library
              if (id.includes("@vueuse")) {
                return "vueuse";
              }
              // Ark UI - component library
              if (id.includes("@ark-ui")) {
                return "ark-ui";
              }
              // Zod - validation library
              if (id.includes("zod")) {
                return "zod";
              }
              // Keep vendor chunks reasonable - group smaller deps
              if (id.includes("node_modules")) {
                return "vendor";
              }
            }
            // Let Nuxt handle page-level chunk splitting automatically
          },
        },
        watch: {
          // Exclude directories that shouldn't be watched
          // Note: Rollup watches all files in the dependency graph, so we can't completely avoid watching packages/node_modules
          // but we can exclude them from the watch list to reduce file handles
          exclude: [
            "**/node_modules/**",
            "**/.git/**",
            "**/packages/**",
            "**/dist/**",
            "**/apps/api/**",
            "**/scripts/**",
            "**/tools/**",
            "**/.nx/**",
            "**/tsconfig.tsbuildinfo",
            "**/*.log",
            "**/.turbo/**",
          ],
        },
      },
    },

    // Only configure if you have specific issues with certain dependencies
    optimizeDeps: {
      // Exclude large dependencies that don't benefit from pre-bundling
      exclude: ["monaco-editor"],
    },
  },
  // Nuxt-level watch configuration
  watch: [
    // Only watch files within the dashboard app directory
    "apps/dashboard/**",
  ],
  ignore: [
    // Ignore patterns for Nuxt's watcher
    "**/node_modules/**",
    "**/.git/**",
    "**/packages/**",
    "**/dist/**",
    "**/apps/api/**",
    "**/apps/*/dist/**",
    "**/scripts/**",
    "**/tools/**",
    "**/.nx/**",
    "**/tsconfig.tsbuildinfo",
    "**/*.log",
    "**/.turbo/**",
  ],
  // Modules
  modules: ["@pinia/nuxt", "@vueuse/nuxt", "@nuxt/icon"],

  // CSS Framework - using Nuxt UI (built on Tailwind CSS)
  css: ["~/assets/css/main.css"],
  // TypeScript configuration
  // Disable type checking during Docker builds for faster builds
  // Type checking can be done separately via `pnpm typecheck` or in CI
  typescript: {
    typeCheck: process.env.SKIP_TYPE_CHECK !== "true",
    tsConfig: {
      compilerOptions: {
        module: "ESNext",
        moduleResolution: "bundler",
        target: "ES2022",
        skipLibCheck: true,
      },
    },
  },

  // Runtime config
  runtimeConfig: {
    // Private keys (only available on server-side)
    apiSecret: "",
    // Server-side API host (for internal Docker service communication)
    // Use API Gateway for all requests (routes to microservices)
    // When running locally (not in Docker), use localhost with Traefik port
    // When running in Docker, use api-gateway service name
    apiHostInternal: process.env.NUXT_API_HOST_INTERNAL || process.env.NUXT_PUBLIC_API_HOST || "http://localhost:80",
    githubClientSecret: process.env.GITHUB_CLIENT_SECRET || "", // Server-side only - never expose to client
    session: {
      password: "changeme_" + crypto.randomUUID(), // CHANGE THIS IN PRODUCTION, should be at least 32 characters
      cookie: {
        secure: true, // Set to true if using HTTPS
      },
    },
    // Public keys (exposed to client-side)
    public: {
      requestHost: "http://localhost:3000",
      // API Gateway is the single entry point for all API requests
      // It routes to appropriate microservices automatically
      apiHost: "http://api.localhost",
      oidcIssuer: "https://obiente.cloud",
      oidcBase: "https://auth.obiente.cloud",
      oidcClientId: "339499954043158530",
      githubClientId: "",
      stripePublishableKey: "",
    },
  },

  ssr: true,

  devServer: {
    port: 3000,
    host: "0.0.0.0",
  },

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

  nitro: {
    // Disable experimental features that might cause build hangs
    experimental: {
      wasm: false, // Disable WASM to avoid potential build hangs
      websocket: false, // Disable websocket to avoid potential build hangs
    },
    minify: true,
    sourceMap: false,
    // Optimize build performance
    compressPublicAssets: true,
    // Enable parallel builds for faster compilation
    prerender: {
      crawlLinks: false, // Disable link crawling for faster builds (only prerender explicit routes)
    },
    // Optimize server build to prevent hangs
    esbuild: {
      options: {
        // Limit concurrency to prevent memory issues
        target: "node20",
      },
    },
    // Disable features that might cause hangs in Docker builds
    storage: {},
    // Reduce build complexity
    routeRules: {},
  },
});
