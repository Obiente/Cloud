// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from "@tailwindcss/vite";
export default defineNuxtConfig({
  devtools: { enabled: true },
  vite: {
    plugins: [tailwindcss()],
    server: {
      hmr: {
        port: 24678, // Use a different port for HMR
      },
      watch: {
        usePolling: true,
      }
    },
  },
  // Modules
  modules: ["@pinia/nuxt", "@vueuse/nuxt"],

  // CSS Framework - using Nuxt UI (built on Tailwind CSS)
  css: ["~/assets/css/main.css"],
  // TypeScript configuration
  typescript: {
    typeCheck: true,
  },

  // Runtime config
  runtimeConfig: {
    // Private keys (only available on server-side)
    apiSecret: "",
    session: {
      password: "changeme_" + crypto.randomUUID(), // CHANGE THIS IN PRODUCTION, should be at least 32 characters
      cookie: {
        secure: false, // Set to true if using HTTPS
      },
    },
    requestHost: undefined,
    // Public keys (exposed to client-side)
    public: {
      apiBaseUrl: "http://localhost:3001",
      oidcIssuer: "https://obiente.cloud",
      oidcBase: "https://auth.obiente.cloud",
      oidcClientId: "339499954043158530",
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
    },
  },
});
