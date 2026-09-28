import type { H3Event } from 'h3'

export function getUpstreamUrl(event: H3Event, path: string) {
  const base = useRuntimeConfig(event).apiBaseUrl
  if (!base) throw createError({ statusCode: 503, statusMessage: 'API server is not configured.' })
  return new URL(path.replace(/^\/+/, ''), `${base.replace(/\/$/, '')}/`).toString()
}

export function getSessionToken(event: H3Event) {
  const config = useRuntimeConfig(event)
  const token = getCookie(event, config.sessionCookieName)
  if (!token) throw createError({ statusCode: 401, statusMessage: 'Authentication required.' })
  return token
}

export function upstreamError(error: any) {
  const statusCode = Number(error?.response?.status || error?.statusCode || 502)
  const safeStatus = statusCode >= 400 && statusCode < 600 ? statusCode : 502
  const upstreamMessage = error?.data?.error?.message
  return createError({
    statusCode: safeStatus,
    statusMessage: safeStatus === 401 ? 'Invalid credentials or expired session.' : (typeof upstreamMessage === 'string' ? upstreamMessage : 'The API request failed.'),
  })
}
