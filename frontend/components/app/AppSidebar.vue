<script setup lang="ts">
import { appNavigation, navigationSections } from '~/config/navigation'
import { useNavDrawer } from '~/composables/useNavDrawer'
const route = useRoute()
const { can, logout } = useAuth()
const { locale, setLocale, t } = useLocale()
const { isOpen, close } = useNavDrawer()
const items = computed(() => appNavigation.filter(item => !item.claim || can(item.claim)))

const groupedItems = computed(() => {
  return navigationSections.map(section => ({
    ...section,
    items: items.value.filter(it => it.sectionKey === section.key)
  })).filter(group => group.items.length > 0)
})

// Close drawer on route navigation
watch(() => route.path, () => {
  close()
})
</script>

<template>
  <div>
    <!-- Mobile Backdrop Overlay -->
    <div 
      v-if="isOpen" 
      class="sidebar-backdrop" 
      @click="close"
      aria-hidden="true"
    />

    <aside class="app-sidebar" :class="{ 'drawer-open': isOpen }">
      <div class="sidebar-top">
        <NuxtLink to="/" class="brand" @click="close">
          <span>M</span>
          <div class="brand-text">
            <strong>Meta Super App</strong>
            <small>{{ t('nav.dashboard') }}</small>
          </div>
        </NuxtLink>
        <button class="drawer-close-btn" :aria-label="t('common.close')" @click="close">
          <v-icon icon="mdi-close" size="20"/>
        </button>
      </div>

      <nav aria-label="Main navigation">
        <div v-for="group in groupedItems" :key="group.key" class="nav-group">
          <div class="nav-group-title">{{ t(group.titleKey) }}</div>
          <NuxtLink 
            v-for="item in group.items" 
            :key="item.to" 
            :to="item.to" 
            :class="{ active: item.to === '/' ? route.path === '/' : route.path.startsWith(item.to) }" 
            :title="t(item.labelKey)"
            @click="close"
          >
            <v-icon :icon="item.icon" size="19"/>
            <span>{{ t(item.labelKey) }}</span>
          </NuxtLink>
        </div>
      </nav>

      <div class="sidebar-bottom">
        <div class="language" role="group" :aria-label="t('common.language')">
          <button :class="{ active: locale === 'lo' }" @click="setLocale('lo')">ລາວ</button>
          <button :class="{ active: locale === 'th' }" @click="setLocale('th')">ไทย</button>
          <button :class="{ active: locale === 'en' }" @click="setLocale('en')">EN</button>
        </div>
        <button class="signout" @click="logout">
          <v-icon icon="mdi-logout" size="18"/>
          <span>{{ t('common.signOut') }}</span>
        </button>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.sidebar-backdrop {
  display: none;
}
@media (max-width: 640px) {
  .sidebar-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    z-index: 89;
    background: rgba(15, 23, 42, 0.45);
  }
}

.drawer-close-btn {
  display: none;
}
@media (max-width: 640px) {
  .drawer-close-btn {
    display: grid;
    place-items: center;
    width: 36px;
    height: 36px;
    margin-left: auto;
    border: none;
    border-radius: 10px;
    background: var(--color-background);
    color: var(--color-text-secondary);
    cursor: pointer;
  }
}

.sidebar-top {
  display: flex;
  align-items: center;
  padding-bottom: 12px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--color-border-subtle);
}

.app-sidebar {
  width: 242px;
  height: 100dvh;
  max-height: 100dvh;
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 90;
  padding: 18px 13px 14px;
  background: var(--color-rail);
  border-right: 0;
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 2px 4px;
  color: var(--color-surface);
  text-decoration: none;
}
.brand span {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  color: var(--color-primary);
  background: var(--color-primary-soft);
  font-weight: 850;
  font-size: 17px;
  flex-shrink: 0;
  box-shadow: none;
}
.brand-text {
  display: flex;
  flex-direction: column;
  line-height: 1.2;
}
.brand-text strong {
  font-size: 15px;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.brand-text small {
  font-size: 12px;
  color: var(--color-rail-muted);
  font-weight: 600;
}

/* Nav scroll area: crucial flex shrink and scrolling */
.app-sidebar nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 2px;
  margin: 12px 0;
}

.nav-group {
  margin-bottom: 8px;
}
.nav-group-title {
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--color-rail-muted);
  padding: 6px 10px 4px;
}

.app-sidebar nav a {
  min-height: 36px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  color: var(--color-rail-text);
  text-decoration: none;
  font-size: 12.5px;
  font-weight: 600;
  transition: all 0.14s ease;
  flex-shrink: 0;
}
.app-sidebar nav a:hover {
  color: var(--color-surface);
  background: rgba(255,255,255,.07);
}
.app-sidebar nav a.active {
  color: var(--color-surface);
  background: var(--color-primary);
  font-weight: 750;
}

.sidebar-bottom {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 10px;
  border-top: 1px solid rgba(255,255,255,.12);
  flex-shrink: 0;
}

.language {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 3px;
  padding: 3px;
  background: rgba(255,255,255,.06);
  border-radius: 9px;
}
.language button {
  min-height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--color-rail-text);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.12s ease;
}
.language button.active {
  color: var(--color-rail);
  background: var(--color-surface);
  box-shadow: none;
}

.version-badge {
  text-align: center;
  font-size: 12px;
  color: var(--color-text-faint);
  font-weight: 600;
}

.signout {
  border: 0;
  border-radius: 9px;
  background: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 10px;
  color: var(--color-rail-text);
  font-size: 12.5px;
  font-weight: 600;
  transition: all 0.14s ease;
}
.signout:hover {
  color: var(--color-surface);
  background: rgba(255,255,255,.07);
}

/* Tablet compact view (641px to 1023px) */
@media(max-width:1023px) and (min-width:641px){
  .app-sidebar {
    width: 74px;
    padding: 16px 8px 12px;
  }
  .sidebar-top {
    justify-content: center;
    padding-bottom: 8px;
  }
  .brand {
    padding: 0;
  }
  .brand-text,
  .nav-group-title,
  .app-sidebar nav span,
  .signout span,
  .version-badge {
    display: none;
  }
  .app-sidebar nav a {
    justify-content: center;
    padding: 8px 0;
  }
  .sidebar-bottom {
    padding-top: 8px;
    gap: 4px;
  }
  .language {
    grid-template-columns: 1fr;
    gap: 2px;
    padding: 2px;
  }
  .language button {
    min-height: 28px;
    font-size: 12px;
  }
  .signout {
    justify-content: center;
    padding: 8px 0;
  }
}

/* Mobile responsive drawer (<= 640px) */
@media(max-width:640px){
  .app-sidebar {
    width: 270px;
    max-width: 86vw;
    transform: translateX(-100%);
    transition: transform 0.28s cubic-bezier(0.32, 0.72, 0, 1);
    box-shadow: 16px 0 40px rgba(15,23,42,.24);
    padding: calc(16px + env(safe-area-inset-top)) 14px calc(16px + env(safe-area-inset-bottom));
  }
  .app-sidebar.drawer-open {
    transform: translateX(0);
  }
}
</style>
