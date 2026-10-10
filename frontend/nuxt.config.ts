import vuetify, { transformAssetUrls } from 'vite-plugin-vuetify'

export default defineNuxtConfig({
  compatibilityDate: '2026-09-11',
  devtools: { enabled: true },
  app: {
    head: {
      title: 'Meta Super App',
      meta: [
        { name: 'description', content: 'A secure workspace for your business.' },
        { name: 'theme-color', content: '#F6F7FB' },
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
    vue: { template: { transformAssetUrls } },
    ssr: { noExternal: ['vuetify'] },
  },
  runtimeConfig: {
    apiBaseUrl: 'https://api2.157.230.245.52.nip.io',
    apiLoginPath: '/auth/login',
    apiMePath: '/auth/me',
    sessionCookieName: 'app_session',
    accountCookieName: 'active_account',
    sessionMaxAge: 60 * 60 * 8,
    public: { appName: 'Meta Super App' },
  },
  routeRules: {
    '/**': {
      headers: {
        'Content-Security-Policy': "default-src 'self'; img-src 'self' data: https:; font-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-inline' https://connect.facebook.net; connect-src 'self' https://www.facebook.com https://web.facebook.com https://graph.facebook.com https://connect.facebook.net; frame-src https://www.facebook.com https://web.facebook.com; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
        'Referrer-Policy': 'strict-origin-when-cross-origin',
        'Permissions-Policy': 'camera=(), microphone=(), geolocation=()',
        'X-Content-Type-Options': 'nosniff',
        'X-Frame-Options': 'DENY',
      },
    },
  },
  typescript: { typeCheck: true, strict: true },
})
