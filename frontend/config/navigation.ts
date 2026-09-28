import type { Permission } from '~/types/auth'

export interface NavigationItem {
  label: string
  icon: string
  to: string
  claim?: Permission
}

export const appNavigation: NavigationItem[] = [
  { label: 'Overview', icon: 'mdi-view-dashboard-outline', to: '/', claim: 'dashboard:read' },
  { label: 'Meta Pages', icon: 'mdi-facebook', to: '/meta-pages', claim: 'pages:read' },
  { label: 'Product Automation', icon: 'mdi-tag-multiple-outline', to: '/product-automation', claim: 'pages:read' },
  { label: 'Quick Replies', icon: 'mdi-message-flash-outline', to: '/replies', claim: 'pages:read' },
  { label: 'Analytics', icon: 'mdi-chart-box-outline', to: '/analytics', claim: 'analytics:read' },
  { label: 'Customers', icon: 'mdi-account-group-outline', to: '/customers', claim: 'customers:read' },
  { label: 'Orders', icon: 'mdi-shopping-outline', to: '/orders', claim: 'orders:read' },
  { label: 'Settings', icon: 'mdi-cog-outline', to: '/settings', claim: 'settings:read' },
]
