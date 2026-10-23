<script setup lang="ts">
import { useNavDrawer } from '~/composables/useNavDrawer'
const { user, activeAccount, switchAccount } = useAuth()
const { locale, setLocale, t, l } = useLocale()
const { toggle: toggleDrawer } = useNavDrawer()
const membership = computed(() => user.value?.accounts.find(item => item.account.id === activeAccount.value?.id))

async function changeAccount(event: Event) {
  await switchAccount((event.target as HTMLSelectElement).value)
}
</script>

<template>
  <header class="app-header">
    <div class="header-left">
      <button class="mobile-drawer-toggle" :aria-label="t('common.allMenus')" @click="toggleDrawer">
        <v-icon icon="mdi-menu" size="22"/>
      </button>

      <div class="workspace-crumb">
        <div class="crumb-text">
          <span class="account-name">{{ activeAccount?.name || 'Packdee' }}</span>
          <span v-if="membership?.role" class="role-pill">{{ membership.role }}</span>
        </div>
      </div>
    </div>

    <div class="account-actions">
      <!-- Quick Action Buttons in Header -->
      <NuxtLink to="/live-chat" class="header-quick-action chat-btn" :title="l({lo:'ເປີດ Live Chat',th:'เปิด Live Chat',en:'Live Chat'})">
        <v-icon icon="mdi-chat-processing-outline" size="16"/>
        <span>Live Chat</span>
      </NuxtLink>
      <NuxtLink to="/replies" class="header-quick-action reply-btn" :title="l({lo:'ຊຸດຂໍ້ຄວາມດ່ວນ',th:'ชุดข้อความด่วน',en:'Quick Replies'})">
        <v-icon icon="mdi-lightning-bolt-outline" size="15"/>
        <span>{{ l({lo:'ໂຄ້ດ',th:'โค้ด',en:'Codes'}) }}</span>
      </NuxtLink>

      <div class="locale-switch" :aria-label="t('common.language')">
        <button :class="{active:locale==='lo'}" @click="setLocale('lo')">ລາວ</button>
        <button :class="{active:locale==='th'}" @click="setLocale('th')">ไทย</button>
        <button :class="{active:locale==='en'}" @click="setLocale('en')">EN</button>
      </div>

      <div class="account-select-wrapper">
        <select :value="activeAccount?.id" :aria-label="l({ lo: 'ເລືອກບັນຊີ', th: 'เลือกบัญชี', en: 'Select Account' })" @change="changeAccount">
          <option v-for="item in user?.accounts" :key="item.account.id" :value="item.account.id">
            {{ item.account.name }} ({{ item.role }})
          </option>
        </select>
        <v-icon icon="mdi-chevron-down" size="14" class="select-chevron" />
      </div>

      <div class="avatar" :title="user?.name">{{ user?.name?.slice(0,2).toUpperCase() }}</div>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  min-height: 58px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
  padding: 8px 0;
  border-bottom: 1px solid var(--color-border);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.mobile-drawer-toggle {
  display: none;
  width: 38px;
  height: 38px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  place-items: center;
  flex-shrink: 0;
}

.workspace-crumb {
  display: flex;
  align-items: center;
  gap: 8px;
}
.crumb-text {
  display: flex;
  align-items: center;
  gap: 6px;
}
.account-name {
  font-size: 14px;
  font-weight: 750;
  color: var(--color-text);
  letter-spacing: -0.01em;
}
.role-pill {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  background: var(--color-surface-soft);
  color: var(--color-text-muted);
  padding: 2px 8px;
  border-radius: 4px;
  border: 1px solid var(--color-border);
}

.account-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-quick-action {
  height: 34px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 11px;
  border-radius: 9px;
  font-size: 12px;
  font-weight: 700;
  text-decoration: none;
  border: 1px solid var(--color-border);
  transition: all var(--transition-fast);
}
.header-quick-action.chat-btn {
  background: var(--color-surface);
  color: var(--color-text);
  border-color: var(--color-border);
}
.header-quick-action.chat-btn:hover {
  border-color:var(--color-primary);
  color:var(--color-primary);
}

.header-quick-action.reply-btn {
  background: var(--color-surface);
  color: var(--color-text);
}
.header-quick-action.reply-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.account-select-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}
.account-select-wrapper select {
  height: 34px;
  min-width: 170px;
  padding: 0 26px 0 10px;
  border: 1px solid var(--color-border);
  border-radius: 9px;
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 12px;
  font-weight: 650;
  appearance: none;
  cursor: pointer;
}
.select-chevron {
  position: absolute;
  right: 8px;
  pointer-events: none;
  color: var(--color-text-muted);
}

.avatar {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  display: grid;
  place-items: center;
  color: #ffffff;
  background: var(--gradient-brand);
  font-size: 12px;
  font-weight: 800;
  box-shadow: 0 2px 8px rgba(0, 100, 224, 0.25);
}

.locale-switch {
  display: flex;
  padding: 2px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 9px;
}
.locale-switch button {
  min-width: 34px;
  height: 28px;
  border: 0;
  border-radius: 7px;
  background: none;
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.locale-switch button.active {
  color: var(--color-primary);
  background: var(--color-primary-soft);
}

@media(max-width: 860px) {
  .header-quick-action.reply-btn { display: none; }
}
@media(max-width: 760px) {
  .account-select-wrapper { display: none; }
}
@media(max-width: 640px) {
  .app-header {
    gap: 8px;
    margin-bottom: 12px;
    padding: 2px 0;
  }
  .mobile-drawer-toggle { display: grid; }
  /* Live Chat is already in bottom mobile bar */
  .header-quick-action.chat-btn { display: none !important; }
  
  .account-name {
    max-width: 120px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
  }
  .crumb-icon {
    width: 28px;
    height: 28px;
  }
  .locale-switch button {
    min-width: 28px;
    height: 26px;
    font-size: 12px;
  }
  .avatar {
    width: 30px;
    height: 30px;
    font-size: 12px;
  }
}
@media(max-width: 400px) {
  .role-pill { display: none; }
  .account-name { max-width: 95px; }
}
</style>
