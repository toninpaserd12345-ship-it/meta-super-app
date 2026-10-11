<script setup lang="ts">
import { useNavDrawer } from '~/composables/useNavDrawer'
const { user, activeAccount, switchAccount } = useAuth()
const { locale, setLocale, t, l } = useLocale()
const { toggle: toggleDrawer } = useNavDrawer()
const membership = computed(() => user.value?.accounts.find(item => item.account.id === activeAccount.value?.id))
const greeting = computed(() => { const hour = new Date().getHours(); return t(hour < 12 ? 'header.morning' : hour < 18 ? 'header.afternoon' : 'header.evening') })
async function changeAccount(event: Event) { await switchAccount((event.target as HTMLSelectElement).value) }
</script>

<template>
  <header class="app-header">
    <div class="header-left">
      <button class="mobile-drawer-toggle" :aria-label="t('common.allMenus')" @click="toggleDrawer">
        <v-icon icon="mdi-menu" size="24"/>
      </button>
      <div class="title">
        <p>{{ membership?.role?.toUpperCase() }} · {{ activeAccount?.name }}</p>
        <h1><slot name="title">{{greeting}}, {{ user?.name?.split(' ')[0] }}</slot></h1>
      </div>
    </div>
    <div class="account-actions">
      <!-- Quick Action Buttons in Header -->
      <NuxtLink to="/live-chat" class="header-quick-action chat-btn" :title="l({lo:'ເປີດ Live Chat',th:'เปิด Live Chat',en:'Live Chat'})">
        <v-icon icon="mdi-chat-processing-outline" size="18"/>
        <span>Live Chat</span>
        <span class="live-dot"/>
      </NuxtLink>
      <NuxtLink to="/replies" class="header-quick-action reply-btn" :title="l({lo:'ຊຸດຂໍ້ຄວາມດ່ວນ',th:'ชุดข้อความด่วน',en:'Quick Replies'})">
        <v-icon icon="mdi-lightning-bolt" size="17"/>
        <span>{{ l({lo:'ໂຄ້ດ',th:'โค้ด',en:'Codes'}) }}</span>
      </NuxtLink>

      <div class="locale-switch" :aria-label="t('common.language')">
        <button :class="{active:locale==='lo'}" @click="setLocale('lo')">ລາວ</button>
        <button :class="{active:locale==='th'}" @click="setLocale('th')">ไทย</button>
        <button :class="{active:locale==='en'}" @click="setLocale('en')">EN</button>
      </div>
      <label>
        <span>{{ t('common.workspace') }}</span>
        <select :value="activeAccount?.id" @change="changeAccount">
          <option v-for="item in user?.accounts" :key="item.account.id" :value="item.account.id">
            {{ item.account.name }} · {{ item.role }}
          </option>
        </select>
      </label>
      <div class="avatar">{{ user?.name?.slice(0,2).toUpperCase() }}</div>
    </div>
  </header>
</template>

<style scoped>
.app-header{min-height:64px;display:flex;justify-content:space-between;align-items:center;gap:20px;margin-bottom:28px}
.header-left{display:flex;align-items:center;gap:12px;min-width:0}
.mobile-drawer-toggle{display:none;width:42px;height:42px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-surface);color:var(--color-text);cursor:pointer;place-items:center;flex-shrink:0}
.title{min-width:0}
.title p{margin:0 0 6px;color:var(--color-text-muted);font-size:10px;font-weight:800;letter-spacing:.11em;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.title h1{margin:0;font-size:clamp(23px,2.5vw,31px);line-height:1.25;letter-spacing:-.025em}
.account-actions{display:flex;align-items:center;gap:8px}
.header-quick-action{height:38px;display:inline-flex;align-items:center;gap:6px;padding:0 12px;border-radius:11px;font-size:12px;font-weight:750;text-decoration:none;border:1px solid var(--color-border);transition:all .15s ease}
.header-quick-action.chat-btn{background:var(--color-primary-soft);color:var(--color-primary);border-color:color-mix(in srgb,var(--color-primary) 28%,transparent)}
.header-quick-action.chat-btn:hover{background:var(--color-primary);color:#fff}
.live-dot{width:7px;height:7px;border-radius:50%;background:#22c55e;box-shadow:0 0 0 2px rgba(34,197,94,.28)}
.header-quick-action.reply-btn{background:var(--color-surface);color:var(--color-text)}
.header-quick-action.reply-btn:hover{border-color:var(--color-primary);color:var(--color-primary)}
.account-actions label{display:grid;gap:4px}
.account-actions label>span{font-size:9px;color:var(--color-text-muted);font-weight:700}
.account-actions select{height:38px;min-width:180px;padding:0 28px 0 10px;border:1px solid var(--color-border);border-radius:11px;background:var(--color-surface);color:var(--color-text);font:inherit;font-size:12px;font-weight:650}
.avatar{width:38px;height:38px;border-radius:11px;display:grid;place-items:center;color:var(--color-surface);background:var(--color-primary-deep);font-size:12px;font-weight:800}
.locale-switch{display:flex;padding:2px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:10px}
.locale-switch button{min-width:38px;height:32px;border:0;border-radius:8px;background:none;color:var(--color-text-secondary);font-size:11px;font-weight:750;cursor:pointer}
.locale-switch button.active{color:var(--color-primary);background:var(--color-primary-soft)}

@media(max-width:860px){
  .header-quick-action.reply-btn{display:none}
}
@media(max-width:760px){
  .app-header{align-items:center;margin-bottom:18px}
  .account-actions label{display:none}
  .header-quick-action.chat-btn span:not(.live-dot){display:none}
  .header-quick-action.chat-btn{padding:0 8px;width:38px;height:38px;justify-content:center}
  .locale-switch button{min-width:30px;height:28px;padding:0 3px;font-size:10px}
  .avatar{width:34px;height:34px;font-size:11px}
}
@media(max-width:640px){
  .mobile-drawer-toggle{display:grid}
}
</style>
