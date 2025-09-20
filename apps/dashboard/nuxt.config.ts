// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from '@tailwindcss/vite';
export default defineNuxtConfig({
  devtools: { enabled: true },
  vite: { plugins: [tailwindcss()] },
  // Modules
  modules: ['@pinia/nuxt'],

  // CSS Framework - using Nuxt UI (built on Tailwind CSS)
  css: ['~/assets/css/main.css'],
  watch: ['composables/**', 'stores/**', 'utils/**', 'components/**'],
  // TypeScript configuration
  typescript: {
    typeCheck: true,
  },

  // Runtime config
  runtimeConfig: {
    // Private keys (only available on server-side)
    apiSecret: process.env.API_SECRET || '',

    // Public keys (exposed to client-side)
    public: {
      apiBaseUrl: process.env.API_BASE_URL || 'http://localhost:3001',
      zitadelUrl: process.env.ZITADEL_URL || 'https://your-zitadel.domain.com',
      zitadelClientId: process.env.ZITADEL_CLIENT_ID || '',
    },
  },

  // SSR configuration
  ssr: true,

  // Auto-import configuration
  imports: {
    dirs: ['composables/**', 'stores/**', 'utils/**'],
  },

  // Development server
  devServer: {
    port: 3000,
    host: '0.0.0.0',
  },

  // App configuration
  app: {
    head: {
      title: 'Obiente Cloud',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Multi-tenant cloud dashboard platform' },
      ],
      link: [{ rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }],
    },
  },

  // Nitro configuration (for server-side)
  nitro: {
    experimental: {
      wasm: true,
    },
  },
});
