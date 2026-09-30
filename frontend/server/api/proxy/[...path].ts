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

    // Disable caching for proxy requests
    setHeader(event, 'Cache-Control', 'no-store, no-cache, must-revalidate, proxy-revalidate')
    setHeader(event, 'Pragma', 'no-cache')
    setHeader(event, 'Expires', '0')

    const contentType = getHeader(event, 'content-type') || ''
    const requestBody = ['GET', 'HEAD'].includes(event.method)
      ? undefined
      : contentType.startsWith('multipart/form-data')
        ? await readRawBody(event, false)
        : await readBody(event)
    // A standalone client avoids Nitro trying to resolve the dynamic upstream
    // URL against every locally typed server route in development mode.
    return await ofetch(getUpstreamUrl(event, path), {
      method: event.method as any,
      query: getQuery(event),
      body: requestBody,
      headers: {
        Authorization: `Bearer ${getSessionToken(event)}`,
        'X-Account-ID': context.activeAccount.id,
        Accept: 'application/json',
        ...(contentType ? { 'Content-Type': contentType } : {}),
      },
      timeout: 15_000,
    })
  } catch (error) {
    throw upstreamError(error)
  }
})
