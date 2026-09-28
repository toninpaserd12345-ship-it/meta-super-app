const buckets = new Map<string, { count: number; resetAt: number }>()

export default defineEventHandler((event) => {
  if (!event.path.startsWith('/api/')) return

  const method = event.method.toUpperCase()
  if (!['GET', 'HEAD', 'OPTIONS'].includes(method)) {
    const origin = getHeader(event, 'origin')
    const host = getHeader(event, 'host')
    if (origin && host && new URL(origin).host !== host) {
      throw createError({ statusCode: 403, statusMessage: 'Cross-site request rejected.' })
    }
  }

  const key = `${getRequestIP(event, { xForwardedFor: true }) || 'unknown'}:${event.path}`
  const now = Date.now()
  const bucket = buckets.get(key)
  if (!bucket || bucket.resetAt <= now) {
    buckets.set(key, { count: 1, resetAt: now + 60_000 })
  } else if (++bucket.count > 60) {
    throw createError({ statusCode: 429, statusMessage: 'Too many requests.' })
  }
})
