<script setup lang="ts">
import { appNavigation } from '~/config/navigation'
const route = useRoute()
const { can, logout } = useAuth()
const { locale, setLocale, t } = useLocale()
const items = computed(() => appNavigation.filter(item => !item.claim || can(item.claim)))
</script>

<template>
  <aside class="app-sidebar">
    <NuxtLink to="/" class="brand"><span>M</span><strong>Meta</strong></NuxtLink>
    <nav aria-label="Main navigation">
      <NuxtLink v-for="item in items" :key="item.to" :to="item.to" :class="{ active: item.to === '/' ? route.path === '/' : route.path.startsWith(item.to) }" :title="t(item.labelKey)">
        <v-icon :icon="item.icon" size="21"/><span>{{ t(item.labelKey) }}</span>
      </NuxtLink>
    </nav>
    <div class="language" role="group" :aria-label="t('common.language')"><button :class="{active:locale==='lo'}" @click="setLocale('lo')">ລາວ</button><button :class="{active:locale==='en'}" @click="setLocale('en')">EN</button></div>
    <div class="version-badge">Version 1.0.6 (Campaign & Ads)</div>
    <button class="signout" @click="logout"><v-icon icon="mdi-logout" size="20"/><span>{{ t('common.signOut') }}</span></button>
  </aside>
</template>

<style scoped>
.app-sidebar{width:236px;height:100dvh;position:fixed;inset:0 auto 0 0;z-index:20;padding:24px 16px 18px;background:var(--color-surface);border-right:1px solid var(--color-border);display:flex;flex-direction:column}.brand{display:flex;align-items:center;gap:11px;padding:0 9px 24px;color:var(--color-text);text-decoration:none;font-size:20px}.brand span{width:40px;height:40px;border-radius:12px;display:grid;place-items:center;color:var(--color-surface);background:var(--color-primary);font-weight:800}.app-sidebar nav{display:grid;gap:4px;overflow-y:auto;overscroll-behavior:contain;padding-right:2px}.app-sidebar nav a{min-height:44px;border-radius:12px;display:flex;align-items:center;gap:13px;padding:9px 13px;color:var(--color-text-secondary);text-decoration:none;font-size:13px;font-weight:650;transition:.2s ease}.app-sidebar nav a:hover{color:var(--color-primary);background:var(--color-primary-soft)}.app-sidebar nav a.active{color:var(--color-primary);background:var(--color-primary-soft)}.signout{border:0;background:none;cursor:pointer;display:flex;align-items:center;gap:11px;margin:18px 10px 0;padding:10px 0;color:var(--color-text-secondary)}
.language{display:grid;grid-template-columns:1fr 1fr;gap:4px;margin-top:auto;padding:4px;background:var(--color-background);border-radius:12px}.language button{min-height:36px;border:0;border-radius:9px;background:transparent;color:var(--color-text-secondary);font-size:12px;font-weight:750;cursor:pointer}.language button.active{color:var(--color-primary);background:var(--color-surface);box-shadow:0 2px 8px rgba(31,35,75,.08)}.version-badge{margin-top:10px;text-align:center;font-size:10px;color:var(--color-text-muted);font-weight:600}
.signout{margin-top:10px}
@media(max-width:1023px) and (min-width:641px){.app-sidebar{width:78px;padding:24px 13px}.brand{padding:0 6px 24px}.brand strong,.app-sidebar nav span,.signout span,.version-badge{display:none}.app-sidebar nav a{justify-content:center}.language{grid-template-columns:1fr}.language button{font-size:10px}.signout{justify-content:center;margin-inline:0}}
@media(max-width:640px){.app-sidebar{display:none}}
</style>
