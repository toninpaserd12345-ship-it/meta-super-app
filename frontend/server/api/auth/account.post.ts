import { getAuthContext } from '~/server/utils/access'

export default defineEventHandler(async (event) => {
  const body = await readBody<{ accountId?: unknown }>(event)
  const accountId = typeof body.accountId === 'string' ? body.accountId : ''
  if (!accountId || accountId.length > 100) throw createError({ statusCode: 400, statusMessage: 'Invalid account.' })

  const context = await getAuthContext(event)
  const membership = context.user.accounts.find(item => item.account.id === accountId)
  if (!membership) throw createError({ statusCode: 403, statusMessage: 'You do not have access to this account.' })

  const config = useRuntimeConfig(event)
  setCookie(event, config.accountCookieName, accountId, {
    httpOnly: true, secure: !import.meta.dev, sameSite: 'strict', path: '/', maxAge: Number(config.sessionMaxAge),
  })
  return { user: context.user, activeAccount: membership.account }
})
