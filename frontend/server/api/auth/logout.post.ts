export default defineEventHandler((event) => {
  const config = useRuntimeConfig(event)
  deleteCookie(event, config.sessionCookieName, { httpOnly: true, sameSite: 'strict', path: '/' })
  deleteCookie(event, config.accountCookieName, { httpOnly: true, sameSite: 'strict', path: '/' })
  return { ok: true }
})
