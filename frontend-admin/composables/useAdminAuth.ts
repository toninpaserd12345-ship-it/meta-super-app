export interface AdminUser {
  id: string
  name: string
  email: string
  isSuperAdmin?: boolean
}

export function useAdminAuth() {
  const config = useRuntimeConfig()
  const token = useCookie<string | null>('admin_access_token', { maxAge: 60 * 60 * 24 * 7 })
  const user = useCookie<AdminUser | null>('admin_user_data', { maxAge: 60 * 60 * 24 * 7 })
  const router = useRouter()

  const isAuthenticated = computed(() => !!token.value)

  async function login(email: string, password: string) {
    const res = await $fetch<{ access_token: string; user: AdminUser }>(`${config.public.apiBaseUrl}/api/v1/admin/auth/login`, {
      method: 'POST',
      body: { email, password },
    })

    token.value = res.access_token
    user.value = res.user
    return res
  }

  function logout() {
    token.value = null
    user.value = null
    router.push('/login')
  }

  return {
    token,
    user,
    isAuthenticated,
    login,
    logout,
  }
}
