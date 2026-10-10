import type { UseFetchOptions } from 'nuxt/app'

type ApiOptions<T> = UseFetchOptions<T> & { clientCacheTtl?: number }
const inflightRequests = new Map<string, Promise<unknown>>()

export function clearApiCache(prefix = '') {
  void prefix
}

export function useApiClient() {
  const account = useState<{ id?: string } | null>('auth.account', () => null)
  async function cachedFetch<T>(url: string, ttl = 0, force = false): Promise<T> {
    void ttl
    const key = `client:${account.value?.id || 'none'}:${url}`
    const pending = !force ? inflightRequests.get(key) : undefined
    if (pending) return pending as Promise<T>

    // Dynamic workspace data must always be read from the API. We only collapse
    // identical requests that are in flight at the same time; nothing is stored.
    const request = $fetch<T>(url, {
      headers: { 'Cache-Control': 'no-cache' },
    }).finally(() => {
      if (inflightRequests.get(key) === request) inflightRequests.delete(key)
    })
    inflightRequests.set(key, request)
    return request
  }

  return { cachedFetch, clear: clearApiCache }
}

export function useApi<T>(url: string | (() => string), options: ApiOptions<T> = {}) {
  const account = useState<{ id?: string } | null>('auth.account', () => null)
  const { clientCacheTtl: _clientCacheTtl, ...fetchOptions } = options
  const resolvedUrl = computed(() => typeof url === 'function' ? url() : url)
  const cacheKey = computed(() => `api:${account.value?.id || 'none'}:${resolvedUrl.value}`)

  const result = useFetch(url, {
    ...fetchOptions,
    key: cacheKey,
    baseURL: '/api',
    credentials: 'same-origin',
    headers: { Accept: 'application/json', 'Cache-Control': 'no-cache', ...fetchOptions.headers },
    dedupe: 'defer',
    getCachedData: () => undefined,
  })

  return result
}
