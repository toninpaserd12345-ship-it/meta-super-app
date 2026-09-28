<script setup lang="ts">
const { user, activeAccount, switchAccount } = useAuth()
const membership = computed(() => user.value?.accounts.find(item => item.account.id === activeAccount.value?.id))
const greeting = computed(() => { const hour = new Date().getHours(); return hour < 12 ? 'Good morning' : hour < 18 ? 'Good afternoon' : 'Good evening' })
async function changeAccount(event: Event) { await switchAccount((event.target as HTMLSelectElement).value) }
</script>

<template>
  <header class="app-header">
    <div class="title"><p>{{ membership?.role?.toUpperCase() }} · {{ activeAccount?.name }}</p><h1><slot name="title">{{greeting}}, {{ user?.name?.split(' ')[0] }}</slot></h1></div>
    <div class="account-actions">
      <label><span>Workspace</span><select :value="activeAccount?.id" @change="changeAccount"><option v-for="item in user?.accounts" :key="item.account.id" :value="item.account.id">{{ item.account.name }} · {{ item.role }}</option></select></label>
      <div class="avatar">{{ user?.name?.slice(0,2).toUpperCase() }}</div>
    </div>
  </header>
</template>

<style scoped>
.app-header{min-height:64px;display:flex;justify-content:space-between;align-items:center;gap:20px;margin-bottom:33px}.title p{margin:0 0 6px;color:var(--color-text-muted);font-size:10px;font-weight:800;letter-spacing:.11em}.title h1{margin:0;font-size:clamp(24px,2.5vw,31px);line-height:1.2;letter-spacing:-.035em}.account-actions{display:flex;align-items:center;gap:12px}.account-actions label{display:grid;gap:4px}.account-actions label>span{font-size:9px;color:var(--color-text-muted);font-weight:700}.account-actions select{height:42px;min-width:190px;padding:0 32px 0 12px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-surface);color:var(--color-text);font:inherit;font-size:12px;font-weight:650}.icon-button,.avatar{width:42px;height:42px;border-radius:12px;display:grid;place-items:center}.icon-button{border:1px solid var(--color-border);background:var(--color-surface);cursor:pointer}.avatar{color:var(--color-surface);background:var(--color-primary-deep);font-size:12px;font-weight:800}
@media(max-width:640px){.app-header{margin-bottom:24px}.account-actions label{display:none}.title p{max-width:190px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.icon-button{display:none}.avatar{width:38px;height:38px}}
</style>
