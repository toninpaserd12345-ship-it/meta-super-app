<script setup lang="ts">
definePageMeta({ middleware: 'auth' })
const { user } = useAuth()
const { l } = useLocale()
</script>

<template>
  <div class="settings-section">
    <h2>{{l({lo:'ຂໍ້ມູນບັນຊີ',th:'การตั้งค่าโปรไฟล์',en:'Profile Settings'})}}</h2>
    <p class="subtitle">{{l({lo:'ກວດສອບຂໍ້ມູນຜູ້ໃຊ້ທີ່ເຂົ້າລະບົບ',th:'ตรวจสอบรายละเอียดบัญชีที่เข้าสู่ระบบ',en:'Review the signed-in account details.'})}}</p>
    
    <div class="profile-card" v-if="user">
      <div class="avatar">
        {{ user.name.charAt(0).toUpperCase() }}
      </div>
      <div class="info">
        <h3>{{ user.name }}</h3>
        <p>{{ user.email }}</p>
 <dl><div><dt>User ID</dt><dd>{{ user.id }}</dd></div></dl>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-section h2{margin:0 0 5px;font-size:20px}.subtitle{margin:0 0 24px;color:var(--color-text-secondary);font-size:13px}.profile-card{display:grid;grid-template-columns:52px minmax(0,1fr);gap:16px;align-items:start;padding:20px 0;border-top:1px solid var(--color-border);border-bottom:1px solid var(--color-border)}.avatar{display:grid;place-items:center;width:52px;height:52px;border-radius:10px;background:var(--color-primary-deep);color:var(--color-surface);font-size:18px;font-weight:780}.info h3{margin:0 0 3px;font-size:16px}.info p{margin:0 0 16px;color:var(--color-text-secondary);font-size:13px}.info dl{margin:0}.info dl div{display:grid;grid-template-columns:90px minmax(0,1fr);gap:12px;padding-top:12px;border-top:1px solid var(--color-border-subtle)}.info dt{color:var(--color-text-muted);font-size:12px}.info dd{margin:0;overflow-wrap:anywhere;color:var(--color-text);font-family:var(--font-mono);font-size:12px}
</style>
