export function useAdminApi() {
  const config = useRuntimeConfig()
  const { token, logout } = useAdminAuth()

  async function adminFetch<T>(path: string, options: any = {}): Promise<T> {
    const url = `${config.public.apiBaseUrl}${path.startsWith('/') ? path : '/' + path}`
    try {
      return await $fetch<T>(url, {
        ...options,
        headers: {
          Accept: 'application/json',
          ...(token.value ? { Authorization: `Bearer ${token.value}` } : {}),
          ...options.headers,
        },
      })
    } catch (err: any) {
      if (err?.statusCode === 401 || err?.statusCode === 403) {
        // Session invalid or token expired
        logout()
      }
      throw err
    }
  }

  return { adminFetch }
}
