<script setup lang="ts">
import { appNavigation } from '~/config/navigation'
import { useNavDrawer } from '~/composables/useNavDrawer'
const route = useRoute()
const { can } = useAuth()
const { t } = useLocale()
const { toggle: toggleDrawer, isOpen } = useNavDrawer()

const primaryRoutes = ['/', '/live-chat', '/orders', '/auto-replies']
const allItems = computed(() => appNavigation.filter(item => !item.claim || can(item.claim)))
const items = computed(() => {
  return primaryRoutes
    .map(path => allItems.value.find(item => item.to === path))
    .filter((item): item is NonNullable<typeof item> => Boolean(item))
})

const isChatActiveOnMobile = computed(() => {
  return route.path === '/live-chat' && Boolean(route.query.c)
})
</script>

<template>
  <nav class="mobile-nav" :class="{ 'chat-active-hidden': isChatActiveOnMobile }" :aria-label="t('common.allMenus')">
    <NuxtLink 
      v-for="item in items" 
      :key="item.to" 
      :to="item.to" 
      :class="{ active: item.to === '/' ? route.path === '/' : route.path.startsWith(item.to) }"
    >
      <v-icon :icon="item.icon" size="21"/>
      <span>{{ t(item.labelKey) }}</span>
    </NuxtLink>
    <button :class="{ active: isOpen }" @click="toggleDrawer">
      <v-icon icon="mdi-menu" size="22"/>
      <span>{{ t('common.more') }}</span>
    </button>
  </nav>
</template>

<style scoped>
.mobile-nav {
  display: none;
}
@media (max-width: 640px) {
  .mobile-nav.chat-active-hidden {
    display: none !important;
  }
  .mobile-nav {
    height: 68px;
    display: flex;
    position: fixed;
    z-index: 80;
    inset: auto 12px max(10px, env(safe-area-inset-bottom)) 12px;
    padding: 6px;
    background: rgba(255, 255, 255, 0.96);
    border: 1px solid var(--color-border);
    border-radius: 19px;
    box-shadow: 0 12px 34px rgba(31, 35, 75, 0.16);
    backdrop-filter: blur(16px);
    -webkit-backdrop-filter: blur(16px);
  }
  .mobile-nav a,
  .mobile-nav > button {
    flex: 1;
    min-width: 0;
    border: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 3px;
    border-radius: 14px;
    color: var(--color-text-secondary);
    background: none;
    text-decoration: none;
    font-size: 10px;
    font-weight: 650;
    cursor: pointer;
    transition: 0.18s ease;
  }
  .mobile-nav a.active,
  .mobile-nav > button.active {
    color: var(--color-primary);
    background: var(--color-primary-soft);
  }
}
</style>
