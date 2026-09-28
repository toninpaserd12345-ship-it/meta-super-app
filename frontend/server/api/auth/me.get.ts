import { getAuthContext } from '~/server/utils/access'

export default defineEventHandler(async (event) => {
  const { user, activeAccount } = await getAuthContext(event)
  return { user, activeAccount }
})
