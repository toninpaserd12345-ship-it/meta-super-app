import type { User } from '~/types/auth'
import { getUpstreamUrl, upstreamError } from '~/server/utils/upstream'

interface LoginResult { access_token?: string; user?: User }

export default defineEventHandler(async (event) => {
  const query = getQuery(event)
  const ticket = query.ticket
  if (query.meta === 'error') return sendRedirect(event, '/login?facebook=expired')
  if (typeof ticket !== 'string' || !ticket) return sendRedirect(event, '/login?facebook=error')
  const config = useRuntimeConfig(event)
  try {
    const result = await $fetch<LoginResult>(getUpstreamUrl(event, '/auth/facebook/exchange'), {
      method: 'POST', body: { ticket }, timeout: 10_000,
    })
    const membership = result.user?.accounts?.[0]
    if (!result.access_token || !membership) throw createError({ statusCode: 502 })
    // OAuth returns through a cross-site top-level redirect (Facebook ->
    // callback gateway -> this app). Lax allows the newly issued session to be
    // sent on the immediate redirect to /meta-pages; Strict can cause a false
    // unauthenticated redirect to /login during that same navigation chain.
    const cookie = { httpOnly: true, secure: !import.meta.dev, sameSite: 'lax' as const, path: '/', maxAge: Number(config.sessionMaxAge) }
    setCookie(event, config.sessionCookieName, result.access_token, cookie)
    setCookie(event, config.accountCookieName, membership.account.id, cookie)
    return sendRedirect(event, '/meta-pages?meta=connected')
  } catch (error) {
    upstreamError(error)
    return sendRedirect(event, '/login?facebook=error')
  }
})
