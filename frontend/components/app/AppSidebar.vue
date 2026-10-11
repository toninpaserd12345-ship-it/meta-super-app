<script setup lang="ts">
import { appNavigation } from '~/config/navigation'
import { useNavDrawer } from '~/composables/useNavDrawer'
const route = useRoute()
const { can, logout } = useAuth()
const { locale, setLocale, t } = useLocale()
const { isOpen, close } = useNavDrawer()
const items = computed(() => appNavigation.filter(item => !item.claim || can(item.claim)))

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
          <span>M</span><strong>Meta</strong>
        </NuxtLink>
        <button class="drawer-close-btn" :aria-label="t('common.close')" @click="close">
          <v-icon icon="mdi-close" size="20"/>
        </button>
      </div>

      <nav aria-label="Main navigation">
        <NuxtLink 
          v-for="item in items" 
          :key="item.to" 
          :to="item.to" 
          :class="{ active: item.to === '/' ? route.path === '/' : route.path.startsWith(item.to) }" 
          :title="t(item.labelKey)"
          @click="close"
        >
          <v-icon :icon="item.icon" size="21"/>
          <span>{{ t(item.labelKey) }}</span>
        </NuxtLink>
      </nav>

      <div class="sidebar-bottom">
        <div class="language" role="group" :aria-label="t('common.language')">
          <button :class="{ active: locale === 'lo' }" @click="setLocale('lo')">ລາວ</button>
          <button :class="{ active: locale === 'th' }" @click="setLocale('th')">ไทย</button>
          <button :class="{ active: locale === 'en' }" @click="setLocale('en')">EN</button>
        </div>
        <div class="version-badge">Version 1.0.6 (Campaign & Ads)</div>
        <button class="signout" @click="logout">
          <v-icon icon="mdi-logout" size="20"/>
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
    backdrop-filter: blur(4px);
    -webkit-backdrop-filter: blur(4px);
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
  padding-bottom: 8px;
  flex-shrink: 0;
}

.app-sidebar {
  width: 236px;
  height: 100dvh;
  max-height: 100dvh;
  position: fixed;
  inset: 0 auto 0 0;
  z-index: 90;
  padding: 20px 16px 16px;
  background: var(--color-surface);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

.brand {
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 0 4px;
  color: var(--color-text);
  text-decoration: none;
  font-size: 20px;
}
.brand span {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  color: var(--color-surface);
  background: var(--color-primary);
  font-weight: 800;
  flex-shrink: 0;
}

/* Nav scroll area: crucial flex shrink and scrolling */
.app-sidebar nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding-right: 2px;
  margin: 10px 0;
}

.app-sidebar nav a {
  min-height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  gap: 13px;
  padding: 8px 13px;
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: 13px;
  font-weight: 650;
  transition: .2s ease;
  flex-shrink: 0;
}
.app-sidebar nav a:hover {
  color: var(--color-primary);
  background: var(--color-primary-soft);
}
.app-sidebar nav a.active {
  color: var(--color-primary);
  background: var(--color-primary-soft);
}

.sidebar-bottom {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 10px;
  border-top: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}

.language {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 4px;
  padding: 4px;
  background: var(--color-background);
  border-radius: 12px;
}
.language button {
  min-height: 34px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 750;
  cursor: pointer;
}
.language button.active {
  color: var(--color-primary);
  background: var(--color-surface);
  box-shadow: 0 2px 8px rgba(31,35,75,.08);
}

.version-badge {
  text-align: center;
  font-size: 10px;
  color: var(--color-text-muted);
  font-weight: 600;
}

.signout {
  border: 0;
  border-radius: 10px;
  background: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 11px;
  padding: 8px 12px;
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 650;
  transition: .2s ease;
}
.signout:hover {
  color: var(--color-error);
  background: color-mix(in srgb, var(--color-error) 8%, transparent);
}

/* Tablet compact view (641px to 1023px) */
@media(max-width:1023px) and (min-width:641px){
  .app-sidebar {
    width: 78px;
    padding: 20px 10px 14px;
  }
  .sidebar-top {
    justify-content: center;
    padding-bottom: 4px;
  }
  .brand {
    padding: 0;
  }
  .brand strong,
  .app-sidebar nav span,
  .signout span,
  .version-badge {
    display: none;
  }
  .app-sidebar nav a {
    justify-content: center;
    padding: 9px 0;
  }
  .sidebar-bottom {
    padding-top: 8px;
    gap: 6px;
  }
  .language {
    grid-template-columns: 1fr;
    gap: 2px;
    padding: 2px;
  }
  .language button {
    min-height: 26px;
    font-size: 9px;
  }
  .signout {
    justify-content: center;
    padding: 8px 0;
  }
}

/* Mobile responsive drawer (<= 640px) */
@media(max-width:640px){
  .app-sidebar {
    width: 280px;
    max-width: 86vw;
    transform: translateX(-100%);
    transition: transform 0.28s cubic-bezier(0.32, 0.72, 0, 1);
    box-shadow: 0 20px 60px rgba(15, 23, 42, 0.25);
    padding: calc(16px + env(safe-area-inset-top)) 16px calc(16px + env(safe-area-inset-bottom));
  }
  .app-sidebar.drawer-open {
    transform: translateX(0);
  }
}
</style>
