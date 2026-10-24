export default defineNuxtRouteMiddleware(async (to) => {
  const { user, fetchUser } = useAuth()
  if (!user.value) await fetchUser()
  if (!user.value) return navigateTo('/login')

  // Don't intercept auth-related or exempt routes
  const exemptPaths = ['/onboarding', '/login', '/meta-pages', '/billing']
  if (exemptPaths.includes(to.path)) return

  try {
    const status = await $fetch<{ isCompleted: boolean; hasPlan: boolean; step: string }>('/api/proxy/api/v1/onboarding/status')
    // Only redirect to onboarding if workspace has not selected any plan yet
    if (!status.hasPlan && to.path !== '/onboarding') {
      return navigateTo('/onboarding')
    }
  } catch {
    // Continue without blocking if status endpoint fails
  }
})
