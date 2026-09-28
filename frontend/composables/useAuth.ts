import type { Account, AuthResponse, Permission, User } from '~/types/auth'

export function useAuth() {
  const user = useState<User | null>('auth.user', () => null)
  const activeAccount = useState<Account | null>('auth.account', () => null)
  const requestFetch = useRequestFetch()

  async function login(email: string, password: string) {
    const result = await $fetch<AuthResponse>('/api/auth/login', {
      method: 'POST',
      body: { email, password },
    })
    user.value = result.user
    activeAccount.value = result.activeAccount
    return result.user
  }

  async function fetchUser() {
    try {
      // Forward the incoming session cookie during SSR. Plain $fetch creates a
      // new server-side request without the browser cookies and caused the
      // auth middleware to redirect successful Facebook logins back to /login.
      const result = await requestFetch<AuthResponse>('/api/auth/me')
      user.value = result.user
      activeAccount.value = result.activeAccount
    } catch {
      user.value = null
      activeAccount.value = null
    }
    return user.value
  }

  async function switchAccount(accountId: string) {
    const result = await $fetch<AuthResponse>('/api/auth/account', { method: 'POST', body: { accountId } })
    user.value = result.user
    activeAccount.value = result.activeAccount
    await refreshNuxtData()
  }

  function can(claim: Permission) {
    const membership = user.value?.accounts?.find(item => item.account.id === activeAccount.value?.id)
    return Boolean(membership?.claims.includes('*') || membership?.claims.includes(claim))
  }

  async function logout() {
    await $fetch('/api/auth/logout', { method: 'POST' })
    user.value = null
    await navigateTo('/login')
  }

  return { user, activeAccount, login, fetchUser, switchAccount, can, logout }
}
