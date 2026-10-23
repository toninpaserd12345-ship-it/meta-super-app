export type SupportedLocale = 'lo' | 'th' | 'en'

export function useAdminLocale() {
  const locale = useState<SupportedLocale>('admin.locale', () => 'lo')

  if (import.meta.client) {
    const saved = localStorage.getItem('meta_admin_locale') as SupportedLocale | null
    if (saved && (saved === 'lo' || saved === 'th' || saved === 'en')) {
      locale.value = saved
    }
  }

  function setLocale(newLocale: SupportedLocale) {
    locale.value = newLocale
    if (import.meta.client) {
      localStorage.setItem('meta_admin_locale', newLocale)
      document.documentElement.lang = newLocale
    }
  }

  function l(messages: { lo: string; th?: string; en: string }): string {
    const current = locale.value
    if (current === 'lo') return messages.lo
    if (current === 'th') return messages.th || messages.lo
    return messages.en
  }

  return { locale, setLocale, l }
}
