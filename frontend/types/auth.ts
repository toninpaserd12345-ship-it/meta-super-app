export interface User {
  id: string | number
  name: string
  email: string
  accounts: AccountMembership[]
}

export interface AuthResponse {
  user: User
  activeAccount: Account | null
}

export type AccountRole = 'owner' | 'admin' | 'manager' | 'member' | 'viewer'

export type Permission =
  | '*'
  | 'dashboard:read'
  | 'pages:read' | 'pages:connect'
  | 'analytics:read'
  | 'customers:read' | 'customers:create' | 'customers:update' | 'customers:delete'
  | 'orders:read' | 'orders:create' | 'orders:update' | 'orders:delete'
  | 'users:read' | 'users:invite' | 'users:update' | 'users:remove'
  | 'settings:read' | 'settings:update'
  | 'billing:read' | 'billing:update'

export interface Account {
  id: string
  name: string
  slug: string
  logoUrl?: string
}

export interface AccountMembership {
  account: Account
  role: AccountRole
  claims: Permission[]
}

export interface AuthContext {
  user: User
  activeAccount: Account
  membership: AccountMembership
}
