import type { Permission } from '~/types/auth'

export interface NavigationItem {
  labelKey: string
  icon: string
  to: string
  claim?: Permission
}

export const appNavigation: NavigationItem[] = [
  { labelKey: 'nav.overview', icon: 'mdi-view-dashboard-outline', to: '/', claim: 'dashboard:read' },
  { labelKey: 'nav.chat', icon: 'mdi-chat-processing-outline', to: '/live-chat', claim: 'pages:read' },
  { labelKey: 'nav.channels', icon: 'mdi-facebook', to: '/meta-pages', claim: 'pages:read' },
  { labelKey: 'nav.ads', icon: 'mdi-bullhorn-outline', to: '/ads', claim: 'pages:read' },
  { labelKey: 'nav.products', icon: 'mdi-package-variant-closed', to: '/products', claim: 'pages:read' },
  { labelKey: 'nav.replies', icon: 'mdi-message-text-fast-outline', to: '/replies', claim: 'pages:read' },
  { labelKey: 'nav.automation', icon: 'mdi-robot-happy-outline', to: '/auto-replies', claim: 'pages:read' },
  { labelKey: 'nav.guide', icon: 'mdi-book-open-page-variant-outline', to: '/automation-guide', claim: 'pages:read' },
  { labelKey: 'nav.analytics', icon: 'mdi-chart-box-outline', to: '/analytics', claim: 'analytics:read' },
  { labelKey: 'nav.customers', icon: 'mdi-account-group-outline', to: '/customers', claim: 'customers:read' },
  { labelKey: 'nav.orders', icon: 'mdi-shopping-outline', to: '/orders', claim: 'orders:read' },
  { labelKey: 'nav.settings', icon: 'mdi-cog-outline', to: '/settings', claim: 'settings:read' },
]
