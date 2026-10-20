import type { Permission } from '~/types/auth'

export interface NavigationItem {
  labelKey: string
  icon: string
  to: string
  claim?: Permission
  sectionKey?: 'core' | 'crm' | 'marketing' | 'commerce' | 'automation'
}

export interface NavigationSection {
  key: 'core' | 'crm' | 'marketing' | 'commerce' | 'automation'
  titleKey: string
}

export const navigationSections: NavigationSection[] = [
  { key: 'core', titleKey: 'nav.groups.core' },
  { key: 'crm', titleKey: 'nav.groups.crm' },
  { key: 'marketing', titleKey: 'nav.groups.marketing' },
  { key: 'commerce', titleKey: 'nav.groups.commerce' },
  { key: 'automation', titleKey: 'nav.groups.automation' },
]

export const appNavigation: NavigationItem[] = [
  // CORE
  { labelKey: 'nav.overview', icon: 'mdi-view-dashboard-outline', to: '/', claim: 'dashboard:read', sectionKey: 'core' },

  // CRM & CHAT
  { labelKey: 'nav.chat', icon: 'mdi-chat-processing-outline', to: '/live-chat', claim: 'pages:read', sectionKey: 'crm' },
  { labelKey: 'nav.replies', icon: 'mdi-lightning-bolt-outline', to: '/replies', claim: 'pages:read', sectionKey: 'crm' },
  { labelKey: 'nav.leads', icon: 'mdi-account-multiple-plus-outline', to: '/leads', claim: 'customers:read', sectionKey: 'crm' },

  // MARKETING & ADS
  { labelKey: 'nav.channels', icon: 'mdi-facebook', to: '/meta-pages', claim: 'pages:read', sectionKey: 'marketing' },
  { labelKey: 'nav.ads', icon: 'mdi-bullhorn-outline', to: '/ads', claim: 'pages:read', sectionKey: 'marketing' },
  { labelKey: 'nav.adsBattle', icon: 'mdi-scale-balance', to: '/ads-battle', claim: 'pages:read', sectionKey: 'marketing' },
  { labelKey: 'nav.adsStrategy', icon: 'mdi-chart-timeline-variant-shimmer', to: '/ads-intelligence', claim: 'pages:read', sectionKey: 'marketing' },
  { labelKey: 'nav.adsPlaybook', icon: 'mdi-book-open-page-variant', to: '/ads-playbook', claim: 'pages:read', sectionKey: 'marketing' },

  // COMMERCE & ORDERS
  { labelKey: 'nav.products', icon: 'mdi-package-variant-closed', to: '/products', claim: 'pages:read', sectionKey: 'commerce' },
  { labelKey: 'nav.orders', icon: 'mdi-clipboard-list-outline', to: '/orders', claim: 'pages:read', sectionKey: 'commerce' },
  { labelKey: 'nav.warehouses', icon: 'mdi-warehouse', to: '/warehouses', claim: 'pages:read', sectionKey: 'commerce' },

  // AUTOMATION & SYSTEM
  { labelKey: 'nav.automation', icon: 'mdi-robot-happy-outline', to: '/auto-replies', claim: 'pages:read', sectionKey: 'automation' },
  { labelKey: 'nav.guide', icon: 'mdi-book-open-page-variant-outline', to: '/automation-guide', claim: 'pages:read', sectionKey: 'automation' },
  { labelKey: 'nav.settings', icon: 'mdi-cog-outline', to: '/settings', claim: 'settings:read', sectionKey: 'automation' },
]
