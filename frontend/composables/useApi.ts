import type { UseFetchOptions } from 'nuxt/app'

type CachedValue = { value: unknown; expiresAt: number }
type ApiOptions<T> = UseFetchOptions<T> & { clientCacheTtl?: number }
const inflightRequests = new Map<string, Promise<unknown>>()

export function clearApiCache(prefix = '') {
  const cache = useState<Record<string, CachedValue>>('api.response-cache', () => ({}))
  for (const key of Object.keys(cache.value)) {
    if (!prefix || key.includes(prefix)) delete cache.value[key]
  }
}

export function useApiClient() {
  const account = useState<{ id?: string } | null>('auth.account', () => null)
  const cache = useState<Record<string, CachedValue>>('api.response-cache', () => ({}))

  async function cachedFetch<T>(url: string, ttl = 60_000, force = false): Promise<T> {
    const key = `client:${account.value?.id || 'none'}:${url}`
    const item = cache.value[key]
    if (!force && item && Date.now() < item.expiresAt) return item.value as T
    if (item) delete cache.value[key]

    const pending = !force ? inflightRequests.get(key) : undefined
    if (pending) return pending as Promise<T>

    const request = $fetch<T>(url).then((value) => {
      cache.value[key] = { value, expiresAt: Date.now() + ttl }
      return value
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
  const cache = useState<Record<string, CachedValue>>('api.response-cache', () => ({}))
  const { clientCacheTtl = 30_000, ...fetchOptions } = options
  const resolvedUrl = computed(() => typeof url === 'function' ? url() : url)
  const method = computed(() => String(toValue(fetchOptions.method as any) || 'GET').toUpperCase())
  const cacheKey = computed(() => `api:${account.value?.id || 'none'}:${resolvedUrl.value}`)
  const canCache = computed(() => method.value === 'GET' && clientCacheTtl > 0)

  const result = useFetch(url, {
    ...fetchOptions,
    key: cacheKey,
    baseURL: '/api',
    credentials: 'same-origin',
    headers: { Accept: 'application/json', ...fetchOptions.headers },
    dedupe: 'defer',
    getCachedData: (key) => {
      if (!canCache.value) return undefined
      const item = cache.value[key]
      if (!item || Date.now() >= item.expiresAt) {
        if (item) delete cache.value[key]
        return undefined
      }
      return item.value as T
    },
  })

  watch(result.data, (value) => {
    if (canCache.value && value !== undefined && value !== null) {
      cache.value[cacheKey.value] = { value, expiresAt: Date.now() + clientCacheTtl }
    }
  })

  return result
}
