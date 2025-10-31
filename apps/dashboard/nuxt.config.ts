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
    },
  },
  // Modules
  modules: ["@pinia/nuxt", "@vueuse/nuxt", "@nuxt/icon"],

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
    githubClientSecret: process.env.GITHUB_CLIENT_SECRET || "", // Server-side only - never expose to client
    session: {
      password: process.env.NUXT_SESSION_PASSWORD || "changeme_" + crypto.randomUUID(), // CHANGE THIS IN PRODUCTION, should be at least 32 characters
      cookie: {
        secure: true, // Set to true if using HTTPS
      },
    },
    // Public keys (exposed to client-side)
    public: {
      requestHost: process.env.NUXT_REQUEST_HOST || "http://localhost:3000",
      apiHost: process.env.NUXT_PUBLIC_API_HOST || "http://localhost:3001",
      oidcIssuer: process.env.NUXT_PUBLIC_OIDC_ISSUER || "https://obiente.cloud",
      oidcBase: process.env.NUXT_PUBLIC_OIDC_BASE || "https://auth.obiente.cloud",
      oidcClientId: process.env.NUXT_PUBLIC_OIDC_CLIENT_ID || "339499954043158530",
      githubClientId: process.env.NUXT_PUBLIC_GITHUB_CLIENT_ID || "",
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
