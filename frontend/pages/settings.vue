<template>
  <div class="settings-page">
    <div class="page-header">
      <div>
        <h1>Settings</h1>
        <p>Manage your account, team members, and billing</p>
      </div>
    </div>
    
    <div class="settings-layout">
      <nav class="settings-nav">
        <NuxtLink to="/settings" class="nav-item" exact-active-class="active">
          <v-icon icon="mdi-account-outline" /> Profile
        </NuxtLink>
        <NuxtLink to="/settings/team" class="nav-item" exact-active-class="active">
          <v-icon icon="mdi-account-multiple-outline" /> Team Members
        </NuxtLink>
        <NuxtLink to="/settings/billing" class="nav-item" exact-active-class="active">
          <v-icon icon="mdi-credit-card-outline" /> Billing & Plans
        </NuxtLink>
      </nav>
      
      <div class="settings-content">
        <NuxtPage />
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header {
  margin-bottom: 32px;
}
.page-header h1 {
  font-size: 28px;
  font-weight: 800;
  color: var(--color-text);
  margin-bottom: 4px;
}
.page-header p {
  color: var(--color-text-secondary);
  font-size: 15px;
}

.settings-layout {
  display: flex;
  gap: 32px;
  align-items: flex-start;
}

.settings-nav {
  width: 240px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: var(--color-surface);
  padding: 16px;
  border-radius: 16px;
  border: 1px solid var(--color-border);
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  border-radius: 10px;
  color: var(--color-text-secondary);
  text-decoration: none;
  font-weight: 600;
  font-size: 14px;
  transition: all 0.2s ease;
}

.nav-item:hover {
  background: var(--color-surface-hover);
  color: var(--color-text);
}

.nav-item.active {
  background: var(--color-primary-soft);
  color: var(--color-primary);
}

.settings-content {
  flex: 1;
  min-width: 0;
  background: var(--color-surface);
  border-radius: 16px;
  border: 1px solid var(--color-border);
  padding: 32px;
  min-height: 400px;
}

@media (max-width: 768px) {
  .settings-layout {
    flex-direction: column;
  }
  .settings-nav {
    width: 100%;
    flex-direction: row;
    overflow-x: auto;
    padding: 8px;
  }
  .nav-item {
    white-space: nowrap;
  }
  .settings-content {
    padding: 20px;
  }
}
</style>
