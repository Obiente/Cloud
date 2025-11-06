// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from "@tailwindcss/vite";

export default defineNuxtConfig({
  devtools: { enabled: true },
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
  },
});
