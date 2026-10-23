export default defineNuxtRouteMiddleware((to) => {
  const { isAuthenticated } = useAdminAuth()

  if (!isAuthenticated.value && to.path !== '/login') {
    return navigateTo('/login')
  }

  if (isAuthenticated.value && to.path === '/login') {
    return navigateTo('/')
  }
})
