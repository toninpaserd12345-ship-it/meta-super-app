import { getUpstreamUrl, upstreamError } from '~/server/utils/upstream'

export default defineEventHandler(async (event) => {
  try {
    return await $fetch<{ authorizationUrl: string }>(getUpstreamUrl(event, '/auth/facebook/start'), {
      method: 'POST', timeout: 10_000,
    })
  } catch (error) {
    throw upstreamError(error)
  }
})
