import { createVuetify } from 'vuetify'
import { aliases, mdi } from 'vuetify/iconsets/mdi'
import { themeColors } from '~/config/theme'

export default defineNuxtPlugin((nuxtApp) => {
  const vuetify = createVuetify({
    icons: { defaultSet: 'mdi', aliases, sets: { mdi } },
    theme: {
      defaultTheme: 'appLight',
      themes: {
        appLight: {
          dark: false,
          colors: {
            primary: themeColors.primary,
            secondary: themeColors.secondary,
            surface: themeColors.surface,
            background: themeColors.background,
            error: themeColors.error,
          },
        },
      },
    },
    defaults: {
      VBtn: { rounded: 'lg', elevation: 0 },
      VCard: { rounded: 'xl', elevation: 0 },
      VTextField: { variant: 'outlined', density: 'comfortable' },
    },
  })

  nuxtApp.vueApp.use(vuetify)
})
