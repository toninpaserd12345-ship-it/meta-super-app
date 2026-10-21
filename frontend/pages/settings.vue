<script setup lang="ts">
const { t, l } = useLocale()
definePageMeta({ middleware: 'auth' })
</script>

<template>
  <div class="settings-page">
    <div class="page-header">
      <div>
        <h1>{{ t('nav.settings') }}</h1>
 <p>{{ l({lo:'ຈັດການບັນຊີ, ການແຈ້ງເຕືອນ, ສະມາຊິກທີມ ແລະການຊຳລະເງິນ',th:'จัดการบัญชี การแจ้งเตือน สมาชิกทีม และการเรียกเก็บเงิน',en:'Manage your account, notifications, team members, and billing'}) }}</p>
      </div>
    </div>
    
    <div class="settings-layout">
      <nav class="settings-nav">
        <NuxtLink to="/settings" class="nav-item" exact-active-class="active">
          <v-icon icon="mdi-account-outline" /> {{ l({lo:'ໂປຣໄຟລ໌',th:'โปรไฟล์',en:'Profile'}) }}
        </NuxtLink>
        <NuxtLink to="/settings/team" class="nav-item" exact-active-class="active">
          <v-icon icon="mdi-account-multiple-outline" /> {{ l({lo:'ສະມາຊິກທີມ',th:'สมาชิกทีม',en:'Team Members'}) }}
        </NuxtLink>
 <NuxtLink to="/settings/notifications" class="nav-item" exact-active-class="active">
 <v-icon icon="mdi-bell-outline" /> {{ l({lo:'ການແຈ້ງເຕືອນ',th:'การแจ้งเตือน',en:'Notifications'}) }}
 </NuxtLink>
        <NuxtLink to="/settings/billing" class="nav-item" exact-active-class="active">
          <v-icon icon="mdi-credit-card-outline" /> {{ l({lo:'ການຊຳລະ ແລະແພັກເກດ',th:'การเรียกเก็บเงินและแพ็กเกจ',en:'Billing & Plans'}) }}
        </NuxtLink>
      </nav>
      
      <div class="settings-content">
        <NuxtPage />
      </div>
    </div>
  </div>
</template>

<style scoped>
.page-header{margin-bottom:20px}.page-header h1{margin:0 0 5px;font-size:var(--text-display);font-weight:780}.page-header p{margin:0;color:var(--color-text-secondary);font-size:13px}.settings-layout{display:grid;gap:16px}.settings-nav{display:flex;gap:2px;overflow-x:auto;padding:3px;border:1px solid var(--color-border);border-radius:11px;background:var(--color-surface);scrollbar-width:none}.settings-nav::-webkit-scrollbar{display:none}.nav-item{display:inline-flex;align-items:center;gap:8px;min-height:40px;padding:0 13px;border-radius:8px;color:var(--color-text-secondary);text-decoration:none;white-space:nowrap;font-size:13px;font-weight:650}.nav-item:hover{color:var(--color-text);background:var(--color-surface-soft)}.nav-item.active{color:var(--color-primary);background:var(--color-primary-soft)}.settings-content{min-width:0;min-height:420px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:12px}@media(max-width:640px){.settings-content{padding:18px 15px}.nav-item{flex:1 0 auto;justify-content:center;padding:0 10px}}
</style>
