<script setup lang="ts">
import { appNavigation, navigationSections } from '~/config/navigation'
import { useNavDrawer } from '~/composables/useNavDrawer'
import { useWorkspacePlan } from '~/composables/useWorkspacePlan'
const route = useRoute()
const { can, logout } = useAuth()
const { locale, setLocale, t, l } = useLocale()
const { isOpen, close } = useNavDrawer()
const { planStatus, isFree, isTrial, isPro, planName, maxPages, connectedPagesCount, fetchPlanStatus } = useWorkspacePlan()

onMounted(() => {
  fetchPlanStatus()
})

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
          <span>P</span>
          <div class="brand-text">
            <strong>Packdee</strong>
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

      <!-- Plan summary box in sidebar -->
      <div v-if="planStatus?.hasPlan" class="sidebar-plan-box" :class="{ 'plan-is-trial': isTrial, 'plan-is-pro': isPro && !isTrial, 'plan-is-free': isFree }">
        <div class="plan-box-header">
          <div class="plan-name-wrap">
            <v-icon :icon="isPro ? (isTrial ? 'mdi-star-face' : 'mdi-crown') : 'mdi-shield-check'" size="16" class="plan-icon" />
            <span class="plan-name-text">{{ planName }}</span>
          </div>
          <span v-if="isTrial" class="plan-badge-mini trial">{{ l({ lo: 'ທົດລອງ', th: 'ทดลอง', en: 'Trial' }) }}</span>
          <span v-else-if="isPro" class="plan-badge-mini pro">PRO</span>
          <span v-else class="plan-badge-mini free">FREE</span>
        </div>

        <div class="plan-usage-bar">
          <div class="usage-label">
            <span>{{ l({ lo: 'ເພຈເຊື່ອມຕໍ່', th: 'เพจที่เชื่อมต่อ', en: 'Pages' }) }}</span>
            <strong>{{ connectedPagesCount }} / {{ maxPages }}</strong>
          </div>
          <div class="usage-track">
            <div
              class="usage-fill"
              :style="{ width: `${Math.min(100, (connectedPagesCount / Math.max(1, maxPages)) * 100)}%` }"
            />
          </div>
        </div>

        <NuxtLink to="/settings/billing" class="plan-upgrade-link" @click="close">
          <span>{{ isPro && !isTrial ? l({ lo: 'ຈັດການແພັກເກດ', th: 'จัดการแพ็กเกจ', en: 'Manage Plan' }) : l({ lo: 'ອັບເກຣດ / ຈັດການແພັກເກດ', th: 'อัปเกรด / จัดการแพ็กเกจ', en: 'Upgrade Plan' }) }}</span>
          <v-icon icon="mdi-chevron-right" size="14" />
        </NuxtLink>
      </div>

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
  padding: 16px 14px 14px;
  background: var(--color-surface);
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  box-sizing: border-box;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 2px 4px;
  color: var(--color-text);
  text-decoration: none;
}
.brand span {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  color: #ffffff;
  background: var(--gradient-brand);
  font-weight: 850;
  font-size: 17px;
  flex-shrink: 0;
  box-shadow: 0 4px 12px rgba(0, 100, 224, 0.28);
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
  color: var(--color-text);
}
.brand-text small {
  font-size: 12px;
  color: var(--color-text-muted);
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
  color: var(--color-text-muted);
  padding: 6px 10px 4px;
}

.app-sidebar nav a {
  min-height: 38px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 12px;
  color: var(--color-text-secondary);
  text-decoration: none;
  font-size: 13px;
  font-weight: 600;
  transition: all 0.14s ease;
  flex-shrink: 0;
}
.app-sidebar nav a:hover {
  color: var(--color-primary);
  background: var(--color-surface-hover);
}
.app-sidebar nav a.active {
  color: var(--color-primary);
  background: var(--color-primary-soft);
  font-weight: 750;
}

.sidebar-bottom {
  margin-top: auto;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding-top: 10px;
  border-top: 1px solid var(--color-border-subtle);
  flex-shrink: 0;
}

.language {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 3px;
  padding: 3px;
  background: var(--color-background);
  border-radius: 9px;
}
.language button {
  min-height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.12s ease;
}
.language button.active {
  color: var(--color-primary);
  background: var(--color-surface);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
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
  color: var(--color-text-secondary);
  font-size: 12.5px;
  font-weight: 600;
  transition: all 0.14s ease;
}
.signout:hover {
  color: var(--color-error);
  background: var(--color-error-soft);
}

.sidebar-plan-box {
  margin: 6px 0 10px;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex-shrink: 0;
  transition: all 0.2s ease;
}
.sidebar-plan-box.plan-is-pro {
  background: linear-gradient(135deg, rgba(0, 100, 224, 0.05), rgba(0, 100, 224, 0.12));
  border-color: rgba(0, 100, 224, 0.25);
}
.sidebar-plan-box.plan-is-trial {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.06), rgba(245, 158, 11, 0.14));
  border-color: rgba(245, 158, 11, 0.3);
}

.plan-box-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
}
.plan-name-wrap {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.plan-is-pro .plan-icon { color: var(--color-primary); }
.plan-is-trial .plan-icon { color: #d97706; }
.plan-is-free .plan-icon { color: var(--color-text-muted); }

.plan-name-text {
  font-size: 12px;
  font-weight: 750;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.plan-badge-mini {
  font-size: 9px;
  font-weight: 800;
  text-transform: uppercase;
  padding: 1.5px 6px;
  border-radius: 4px;
  letter-spacing: 0.04em;
}
.plan-badge-mini.free {
  background: var(--color-surface);
  color: var(--color-text-muted);
  border: 1px solid var(--color-border);
}
.plan-badge-mini.trial {
  background: #fef3c7;
  color: #92400e;
}
.plan-badge-mini.pro {
  background: var(--color-primary);
  color: #ffffff;
}

.plan-usage-bar {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.usage-label {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--color-text-muted);
}
.usage-label strong {
  color: var(--color-text);
  font-weight: 700;
}
.usage-track {
  height: 5px;
  background: var(--color-surface);
  border-radius: 99px;
  overflow: hidden;
  border: 1px solid var(--color-border-subtle);
}
.usage-fill {
  height: 100%;
  border-radius: 99px;
  background: var(--color-primary);
  transition: width 0.3s ease;
}
.plan-is-trial .usage-fill {
  background: #f59e0b;
}

.plan-upgrade-link {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11.5px;
  font-weight: 700;
  color: var(--color-primary);
  text-decoration: none;
  padding-top: 2px;
  transition: color 0.15s ease;
}
.plan-upgrade-link:hover {
  text-decoration: underline;
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
  .version-badge,
  .sidebar-plan-box {
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
