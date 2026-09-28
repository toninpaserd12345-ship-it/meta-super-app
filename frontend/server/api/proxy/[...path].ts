import { getSessionToken, getUpstreamUrl, upstreamError } from '~/server/utils/upstream'
import { claimForRequest, getAuthContext, requireClaim } from '~/server/utils/access'
import { ofetch } from 'ofetch'

export default defineEventHandler(async (event) => {
  const path = getRouterParam(event, 'path') || ''
  if (!path || path.split('/').some((part) => part === '..')) {
    throw createError({ statusCode: 400, statusMessage: 'Invalid API path.' })
  }
  try {
    const context = await getAuthContext(event)
    const requiredClaim = claimForRequest(path, event.method)
    if (requiredClaim) requireClaim(context, requiredClaim)
    // A standalone client avoids Nitro trying to resolve the dynamic upstream
    // URL against every locally typed server route in development mode.
    return await ofetch(getUpstreamUrl(event, path), {
      method: event.method as any,
      query: getQuery(event),
      body: ['GET', 'HEAD'].includes(event.method) ? undefined : await readBody(event),
      headers: {
        Authorization: `Bearer ${getSessionToken(event)}`,
        'X-Account-ID': context.activeAccount.id,
        Accept: 'application/json',
      },
      timeout: 15_000,
    })
  } catch (error) {
    throw upstreamError(error)
  }
})
