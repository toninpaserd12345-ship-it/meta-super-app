import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'

// @ts-ignore
const apiBase = (typeof process !== 'undefined' && (process.env?.NUXT_PUBLIC_API_BASE_URL || process.env?.API_BASE_URL)) || 'https://api2.157.230.245.52.nip.io'

export default defineNuxtConfig({
  compatibilityDate: '2026-09-11',
  devtools: { enabled: true },
  app: {
    head: {
      title: 'Packdee Admin — Management Platform',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1, maximum-scale=1, viewport-fit=cover' },
        { name: 'description', content: 'Super Admin Management Platform for Packdee.' },
        { name: 'theme-color', content: '#0F172A' },
      ],
    },
  },
  css: [
    'vuetify/styles',
    '@mdi/font/css/materialdesignicons.css',
    '@fontsource/noto-sans-lao/400.css',
    '@fontsource/noto-sans-lao/500.css',
    '@fontsource/noto-sans-lao/600.css',
    '@fontsource/noto-sans-lao/700.css',
    '@fontsource/noto-sans-thai/400.css',
    '@fontsource/noto-sans-thai/500.css',
    '@fontsource/noto-sans-thai/600.css',
    '@fontsource/noto-sans-thai/700.css',
    '~/assets/tokens.css',
    '~/assets/main.css',
  ],
  build: { transpile: ['vuetify'] },
  modules: [
    (_options, nuxt) => {
      nuxt.hooks.hook('vite:extendConfig', (config) => {
        config.plugins?.push(vuetify({ autoImport: true }))
      })
    },
  ],
  vite: {
    build: { cssMinify: false },
    vue: { template: { transformAssetUrls } },
    ssr: { noExternal: ['vuetify'] },
  },
  runtimeConfig: {
    public: {
      apiBaseUrl: apiBase,
      appName: 'Packdee Admin',
    },
  },
  devServer: {
    port: 3002,
  },
  typescript: { typeCheck: true, strict: true },
})
