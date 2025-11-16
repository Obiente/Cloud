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
      rollupOptions: {
        output: {
          // Only split out very large, independent dependencies
          // Let Nuxt/Vite handle the rest automatically to avoid circular dependencies
          manualChunks: (id) => {
            // Only split monaco-editor as it's very large and independent
            if (id.includes("node_modules") && id.includes("monaco-editor")) {
              return "monaco";
            }
            // Let Nuxt handle all other chunk splitting automatically
            // This prevents circular dependency issues
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
    experimental: {
      wasm: true,
      websocket: true,
    },
    minify: true,
    sourceMap: false,
  },
});
