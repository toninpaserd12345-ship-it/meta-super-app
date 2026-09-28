import type { H3Event } from 'h3'
import type { AccountMembership, AuthContext, Permission, User } from '~/types/auth'
import { getSessionToken, getUpstreamUrl, upstreamError } from './upstream'

export async function getAuthContext(event: H3Event): Promise<AuthContext> {
  const config = useRuntimeConfig(event)
  let user: User
  try {
    user = await $fetch<User>(getUpstreamUrl(event, config.apiMePath), {
      headers: { Authorization: `Bearer ${getSessionToken(event)}` }, timeout: 10_000,
    })
  } catch (error) {
    throw upstreamError(error)
  }
  if (!Array.isArray(user.accounts) || user.accounts.length === 0) {
    throw createError({ statusCode: 403, statusMessage: 'This user has no account access.' })
  }
  const requestedId = getCookie(event, config.accountCookieName)
  const membership = user.accounts.find(item => item.account.id === requestedId) || user.accounts[0]
  if (!membership) throw createError({ statusCode: 403, statusMessage: 'This user has no account access.' })
  validateMembership(membership)
  return { user, activeAccount: membership.account, membership }
}

function validateMembership(value: AccountMembership) {
  if (!value?.account?.id || !Array.isArray(value.claims)) {
    throw createError({ statusCode: 502, statusMessage: 'Invalid account claims returned by the API.' })
  }
}

export function requireClaim(context: AuthContext, claim: Permission) {
  if (!context.membership.claims.includes('*') && !context.membership.claims.includes(claim)) {
    throw createError({ statusCode: 403, statusMessage: `Missing permission: ${claim}` })
  }
}

export function claimForRequest(path: string, method: string): Permission | null {
  const resource = path.split('/')[0] || ''
  const supported = ['analytics', 'customers', 'orders', 'users', 'settings', 'billing']
  if (!supported.includes(resource)) return null
  if (resource === 'analytics') return 'analytics:read'
  const action = method === 'GET' ? 'read' : method === 'POST' ? 'create' : method === 'DELETE' ? 'delete' : 'update'
  const special: Record<string, Permission> = { 'users:create': 'users:invite', 'billing:create': 'billing:update' }
  return special[`${resource}:${action}`] || `${resource}:${action}` as Permission
}
