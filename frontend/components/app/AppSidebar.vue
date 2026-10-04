<script setup lang="ts">
import { appNavigation } from '~/config/navigation'
const route = useRoute()
const { can, logout } = useAuth()
const items = computed(() => appNavigation.filter(item => !item.claim || can(item.claim)))
</script>

<template>
  <aside class="app-sidebar">
    <NuxtLink to="/" class="brand"><span>M</span><strong>Meta</strong></NuxtLink>
    <nav aria-label="Main navigation">
      <NuxtLink v-for="item in items" :key="item.to" :to="item.to" :class="{ active: item.to === '/' ? route.path === '/' : route.path.startsWith(item.to) }" :title="item.label">
        <v-icon :icon="item.icon" size="21"/><span>{{ item.label }}</span>
      </NuxtLink>
    </nav>
    <div class="version-badge">Version 1.0.4 (Live Chat)</div>
    <button class="signout" @click="logout"><v-icon icon="mdi-logout" size="20"/><span>Sign out</span></button>
  </aside>
</template>

<style scoped>
.app-sidebar{width:236px;height:100dvh;position:fixed;inset:0 auto 0 0;z-index:20;padding:28px 20px 22px;background:var(--color-surface);border-right:1px solid var(--color-border);display:flex;flex-direction:column}.brand{display:flex;align-items:center;gap:11px;padding:0 9px 35px;color:var(--color-text);text-decoration:none;font-size:20px}.brand span{width:37px;height:37px;border-radius:11px;display:grid;place-items:center;color:var(--color-surface);background:var(--color-primary);font-weight:800}.app-sidebar nav{display:grid;gap:6px}.app-sidebar nav a{height:46px;border-radius:12px;display:flex;align-items:center;gap:13px;padding:0 13px;color:var(--color-text-secondary);text-decoration:none;font-size:13px;font-weight:650;transition:.2s ease}.app-sidebar nav a:hover{color:var(--color-primary);background:var(--color-primary-soft)}.app-sidebar nav a.active{color:var(--color-primary);background:var(--color-primary-soft)}.upgrade{margin-top:auto;padding:18px;background:var(--gradient-upgrade);border-radius:16px;color:var(--color-primary-ink)}.upgrade strong{display:block;margin-top:9px;font-size:13px}.upgrade p{color:var(--color-primary-muted);font-size:10px;line-height:1.5}.upgrade button,.signout{border:0;background:none;cursor:pointer}.upgrade button{padding:0;color:var(--color-primary);font-size:11px;font-weight:800}.signout{display:flex;align-items:center;gap:11px;margin:18px 10px 0;padding:10px 0;color:var(--color-text-secondary)}
.version-badge{margin-top:auto;text-align:center;font-size:11px;color:var(--color-text-muted);font-weight:600}
.signout{margin-top:10px}
@media(max-width:1023px) and (min-width:641px){.app-sidebar{width:78px;padding:28px 13px}.brand{padding:0 7px 35px}.brand strong,.app-sidebar nav span,.upgrade,.signout span{display:none}.app-sidebar nav a{justify-content:center}.signout{justify-content:center;margin-inline:0}}
@media(max-width:640px){.app-sidebar{display:none}}
</style>
