<script setup lang="ts">
const { locale, setLocale, l } = useAdminLocale()
const { user, logout } = useAdminAuth()
const route = useRoute()

const navItems = computed(() => [
  {
    title: l({ lo: 'ພາບລວມລະບົບ', th: 'ภาพรวมระบบ', en: 'Platform Overview' }),
    icon: 'mdi-view-dashboard-outline',
    to: '/',
  },
  {
    title: l({ lo: 'ຈັດການຮ້ານຄ້າ (Tenants)', th: 'จัดการร้านค้า (Tenants)', en: 'Tenant Management' }),
    icon: 'mdi-storefront-outline',
    to: '/tenants',
  },
  {
    title: l({ lo: 'ກວດສອບສະລິບໂອນເງິນ', th: 'ตรวจสอบสลิปโอนเงิน', en: 'Payment Approvals' }),
    icon: 'mdi-credit-card-check-outline',
    to: '/approvals',
  },
])
</script>

<template>
  <div class="admin-shell">
    <!-- Sidebar -->
    <aside class="admin-sidebar">
      <div class="sidebar-brand">
        <div class="brand-icon">
          <v-icon icon="mdi-shield-crown" color="white" size="22" />
        </div>
        <div class="brand-text">
          <h2>Meta Admin</h2>
          <span>Super Admin Portal</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <NuxtLink
          v-for="item in navItems"
          :key="item.to"
          :to="item.to"
          class="nav-link"
          :class="{ active: route.path === item.to }"
        >
          <v-icon :icon="item.icon" size="20" />
          <span>{{ item.title }}</span>
        </NuxtLink>
      </nav>

      <div class="sidebar-footer">
        <div class="locale-bar">
          <button type="button" :class="{ active: locale === 'lo' }" @click="setLocale('lo')">LAO</button>
          <button type="button" :class="{ active: locale === 'th' }" @click="setLocale('th')">THAI</button>
          <button type="button" :class="{ active: locale === 'en' }" @click="setLocale('en')">ENG</button>
        </div>

        <div class="user-chip">
          <div class="user-avatar">
            <v-icon icon="mdi-account-tie" size="18" color="white" />
          </div>
          <div class="user-info">
            <strong>{{ user?.name || 'Super Admin' }}</strong>
            <small>{{ user?.email || 'admin@meta.app' }}</small>
          </div>
          <button type="button" class="logout-btn" title="Logout" @click="logout">
            <v-icon icon="mdi-logout" size="18" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main Content Area -->
    <div class="admin-main">
      <header class="admin-topbar">
        <div class="topbar-left">
          <span class="environment-badge">PRODUCTION PLATFORM</span>
        </div>
        <div class="topbar-right">
          <span class="system-time">{{ new Date().toLocaleDateString() }}</span>
        </div>
      </header>

      <div class="admin-content">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.admin-shell {
  display: flex;
  min-height: 100vh;
  background-color: var(--color-background);
}

.admin-sidebar {
  width: 17rem;
  background-color: #0F172A;
  color: #F8FAFC;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  border-right: 1px solid #1E293B;
}

.sidebar-brand {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-2);
  border-bottom: 1px solid #1E293B;
}

.brand-icon {
  width: 2.5rem;
  height: 2.5rem;
  background: linear-gradient(135deg, #1E40AF 0%, #3B82F6 100%);
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
}

.brand-text h2 {
  margin: 0;
  font-size: var(--text-md);
  font-weight: 700;
  color: #FFFFFF;
}

.brand-text span {
  font-size: var(--text-xs);
  color: #94A3B8;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: var(--space-2);
  flex: 1;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0.75rem 1rem;
  border-radius: var(--radius-sm);
  color: #94A3B8;
  font-size: var(--text-sm);
  font-weight: 500;
  transition: all var(--transition-fast);
}

.nav-link:hover {
  background-color: #1E293B;
  color: #FFFFFF;
}

.nav-link.active {
  background-color: #1E40AF;
  color: #FFFFFF;
  font-weight: 600;
}

.sidebar-footer {
  padding: var(--space-2);
  border-top: 1px solid #1E293B;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.locale-bar {
  display: flex;
  background-color: #1E293B;
  border-radius: var(--radius-xs);
  padding: 2px;
}

.locale-bar button {
  flex: 1;
  padding: 4px;
  background: transparent;
  border: 0;
  border-radius: var(--radius-xs);
  color: #94A3B8;
  font-size: var(--text-xs);
  font-weight: 700;
  cursor: pointer;
}

.locale-bar button.active {
  background-color: #3B82F6;
  color: #FFFFFF;
}

.user-chip {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: 0.5rem;
  background-color: #1E293B;
  border-radius: var(--radius-sm);
}

.user-avatar {
  width: 2rem;
  height: 2rem;
  background-color: #334155;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.user-info {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.user-info strong {
  font-size: var(--text-xs);
  color: #F8FAFC;
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
}

.user-info small {
  font-size: 10px;
  color: #94A3B8;
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
}

.logout-btn {
  background: transparent;
  border: 0;
  color: #94A3B8;
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
}

.logout-btn:hover {
  color: #EF4444;
}

.admin-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.admin-topbar {
  height: 3.5rem;
  background-color: #FFFFFF;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-4);
}

.environment-badge {
  font-size: 10px;
  font-weight: 800;
  background-color: #EFF6FF;
  color: #1E40AF;
  padding: 2px 8px;
  border-radius: 999px;
  letter-spacing: 0.5px;
}

.system-time {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}

.admin-content {
  flex: 1;
  padding: var(--space-4);
}
</style>
