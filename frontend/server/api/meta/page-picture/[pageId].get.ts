import { getAuthContext, requireClaim } from '~/server/utils/access'
import { getSessionToken, getUpstreamUrl, upstreamError } from '~/server/utils/upstream'

export default defineEventHandler(async (event) => {
  try {
    const pageId = getRouterParam(event, 'pageId')?.trim()
    if (!pageId || !/^[A-Za-z0-9_-]+$/.test(pageId)) {
      throw createError({ statusCode: 400, statusMessage: 'Invalid Page ID.' })
    }

    const context = await getAuthContext(event)
    requireClaim(context, 'pages:read')
    const response = await fetch(getUpstreamUrl(event, `api/v1/meta/pages/${encodeURIComponent(pageId)}/picture`), {
      headers: {
        Authorization: `Bearer ${getSessionToken(event)}`,
        'X-Account-ID': context.activeAccount.id,
        Accept: 'image/*',
      },
    })
    if (!response.ok) {
      throw createError({ statusCode: response.status, statusMessage: 'Unable to load the Meta profile picture.' })
    }

    const contentType = response.headers.get('content-type') || 'image/jpeg'
    return new Response(await response.arrayBuffer(), {
      headers: {
        'Content-Type': contentType,
        'Cache-Control': 'private, max-age=600',
        'X-Content-Type-Options': 'nosniff',
      },
    })
  } catch (error) {
    throw upstreamError(error)
  }
})
