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

    // EventSource needs the upstream response body to remain a stream. Parsing
    // this endpoint with ofetch buffers the response and leaves Live Chat stuck.
    if (event.method === 'GET' && path === 'api/v1/chat/stream') {
      const response = await fetch(getUpstreamUrl(event, path), {
        headers: {
          Authorization: `Bearer ${getSessionToken(event)}`,
          'X-Account-ID': context.activeAccount.id,
          Accept: 'text/event-stream',
        },
        cache: 'no-store',
      })
      if (!response.ok || !response.body) {
        throw createError({ statusCode: response.status || 502, statusMessage: 'Unable to open the live message stream.' })
      }
      return new Response(response.body, {
        status: response.status,
        headers: {
          'Content-Type': 'text/event-stream; charset=utf-8',
          'Cache-Control': 'no-store, no-cache, must-revalidate',
          'X-Accel-Buffering': 'no',
          Connection: 'keep-alive',
        },
      })
    }

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
      timeout: 30_000,
      cache: 'no-store',
    })
  } catch (error) {
    throw upstreamError(error)
  }
})
