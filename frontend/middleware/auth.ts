export default defineNuxtRouteMiddleware(async (to) => {
  const { user, fetchUser } = useAuth()
  if (!user.value) await fetchUser()
  if (!user.value) return navigateTo('/login')

  if (to.path !== '/login') {
    try {
      const status = await $fetch<{ isCompleted: boolean; step: string }>('/api/proxy/api/v1/onboarding/status')
      if (!status.isCompleted) {
        if (to.path !== '/onboarding') {
          return navigateTo('/onboarding')
        }
      } else if (to.path === '/onboarding') {
        return navigateTo('/')
      }
    } catch {
      // Continue without blocking
    }
  }
})
