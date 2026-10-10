<script setup lang="ts">
const { user, activeAccount, switchAccount } = useAuth()
const { locale, setLocale, t } = useLocale()
const membership = computed(() => user.value?.accounts.find(item => item.account.id === activeAccount.value?.id))
const greeting = computed(() => { const hour = new Date().getHours(); return t(hour < 12 ? 'header.morning' : hour < 18 ? 'header.afternoon' : 'header.evening') })
async function changeAccount(event: Event) { await switchAccount((event.target as HTMLSelectElement).value) }
</script>

<template>
  <header class="app-header">
    <div class="title"><p>{{ membership?.role?.toUpperCase() }} · {{ activeAccount?.name }}</p><h1><slot name="title">{{greeting}}, {{ user?.name?.split(' ')[0] }}</slot></h1></div>
    <div class="account-actions">
      <div class="locale-switch" :aria-label="t('common.language')"><button :class="{active:locale==='lo'}" @click="setLocale('lo')">ລາວ</button><button :class="{active:locale==='th'}" @click="setLocale('th')">ไทย</button><button :class="{active:locale==='en'}" @click="setLocale('en')">EN</button></div>
      <label><span>{{ t('common.workspace') }}</span><select :value="activeAccount?.id" @change="changeAccount"><option v-for="item in user?.accounts" :key="item.account.id" :value="item.account.id">{{ item.account.name }} · {{ item.role }}</option></select></label>
      <div class="avatar">{{ user?.name?.slice(0,2).toUpperCase() }}</div>
    </div>
  </header>
</template>

<style scoped>
.app-header{min-height:64px;display:flex;justify-content:space-between;align-items:center;gap:20px;margin-bottom:28px}.title{min-width:0}.title p{margin:0 0 6px;color:var(--color-text-muted);font-size:10px;font-weight:800;letter-spacing:.11em;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.title h1{margin:0;font-size:clamp(23px,2.5vw,31px);line-height:1.25;letter-spacing:-.025em}.account-actions{display:flex;align-items:center;gap:10px}.account-actions label{display:grid;gap:4px}.account-actions label>span{font-size:9px;color:var(--color-text-muted);font-weight:700}.account-actions select{height:44px;min-width:190px;padding:0 32px 0 12px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-surface);color:var(--color-text);font:inherit;font-size:12px;font-weight:650}.avatar{width:44px;height:44px;border-radius:12px;display:grid;place-items:center;color:var(--color-surface);background:var(--color-primary-deep);font-size:12px;font-weight:800}.locale-switch{display:flex;padding:3px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:11px}.locale-switch button{min-width:42px;height:36px;border:0;border-radius:8px;background:none;color:var(--color-text-secondary);font-size:11px;font-weight:750;cursor:pointer}.locale-switch button.active{color:var(--color-primary);background:var(--color-primary-soft)}
@media(max-width:760px){.app-header{align-items:flex-start;margin-bottom:22px}.account-actions label{display:none}.title p{max-width:42vw}.locale-switch{position:absolute;right:62px;top:22px}.locale-switch button{min-width:34px;padding:0 5px}.avatar{width:40px;height:40px}}
</style>
