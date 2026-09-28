import type { User } from '~/types/auth'
import { getUpstreamUrl, upstreamError } from '~/server/utils/upstream'

interface UpstreamLogin { access_token?: string; token?: string; user?: User }

export default defineEventHandler(async (event) => {
  const body = await readBody<{ email?: unknown; password?: unknown }>(event)
  const email = typeof body.email === 'string' ? body.email.trim().toLowerCase() : ''
  const password = typeof body.password === 'string' ? body.password : ''
  if (!/^\S+@\S+\.\S+$/.test(email) || password.length < 8 || password.length > 128) {
    throw createError({ statusCode: 400, statusMessage: 'Invalid email or password format.' })
  }

  const config = useRuntimeConfig(event)
  try {
    const result = await $fetch<UpstreamLogin>(getUpstreamUrl(event, config.apiLoginPath), {
      method: 'POST', body: { email, password }, timeout: 10_000,
    })
    const token = result.access_token || result.token
    if (!token || !result.user) throw createError({ statusCode: 502 })
    setCookie(event, config.sessionCookieName, token, {
      httpOnly: true, secure: !import.meta.dev, sameSite: 'strict', path: '/',
      maxAge: Number(config.sessionMaxAge),
    })
    const membership = result.user.accounts?.[0]
    if (!membership) throw createError({ statusCode: 403, statusMessage: 'This user has no account access.' })
    setCookie(event, config.accountCookieName, membership.account.id, {
      httpOnly: true, secure: !import.meta.dev, sameSite: 'strict', path: '/', maxAge: Number(config.sessionMaxAge),
    })
    return { user: result.user, activeAccount: membership.account }
  } catch (error) {
    throw upstreamError(error)
  }
})
