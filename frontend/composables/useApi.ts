import type { UseFetchOptions } from 'nuxt/app'

export function useApi<T>(url: string | (() => string), options: UseFetchOptions<T> = {}) {
  return useFetch(url, {
    ...options,
    baseURL: '/api',
    credentials: 'same-origin',
    headers: { Accept: 'application/json', ...options.headers },
  })
}
