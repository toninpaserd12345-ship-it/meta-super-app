<script setup lang="ts">
definePageMeta({ layout: false })

const { login } = useAdminAuth()
const { l } = useAdminLocale()
const router = useRouter()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

const handleLogin = async () => {
  if (!email.value.trim() || !password.value) {
    error.value = l({ lo: 'ກະລຸນາປ້ອນອີເມວ ແລະ ລະຫັດຜ່ານ', th: 'กรุณากรอกอีเมลและรหัสผ่าน', en: 'Please enter email and password' })
    return
  }

  loading.value = true
  error.value = ''

  try {
    await login(email.value.trim(), password.value)
    router.push('/')
  } catch (err: any) {
    error.value = err?.data?.message || err?.message || l({
      lo: 'ອີເມວ ຫຼື ລະຫັດຜ່ານບໍ່ຖືກຕ້ອງ ຫຼື ບໍ່ມີສິດ Super Admin',
      th: 'อีเมลหรือรหัสผ่านไม่ถูกต้อง หรือไม่มีสิทธิ์ Super Admin',
      en: 'Invalid credentials or lacking Super Admin privileges'
    })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-brand">
        <div class="brand-badge">
          <v-icon icon="mdi-shield-crown" color="white" size="28" />
        </div>
        <h1>Packdee</h1>
        <p>Super Admin Platform</p>
      </div>

      <form class="login-form" @submit.prevent="handleLogin">
        <div v-if="error" class="error-alert">
          <v-icon icon="mdi-alert-circle" size="18" />
          <span>{{ error }}</span>
        </div>

        <div class="form-group">
          <label>{{ l({ lo: 'ອີເມວ Super Admin', th: 'อีเมล Super Admin', en: 'Admin Email' }) }}</label>
          <input
            v-model="email"
            type="email"
            placeholder="admin@meta.app"
            required
            autocomplete="email"
          />
        </div>

        <div class="form-group">
          <label>{{ l({ lo: 'ລະຫັດຜ່ານ', th: 'รหัสผ่าน', en: 'Password' }) }}</label>
          <input
            v-model="password"
            type="password"
            placeholder="••••••••"
            required
            autocomplete="current-password"
          />
        </div>

        <button type="submit" class="btn-primary login-btn" :disabled="loading">
          <v-icon v-if="loading" icon="mdi-loading mdi-spin" size="18" />
          <span v-else>{{ l({ lo: 'ເຂົ້າສູ່ລະບົບຜູ້ດູແລ', th: 'เข้าสู่ระบบผู้ดูแล', en: 'Sign in to Platform' }) }}</span>
        </button>
      </form>

      <footer class="login-footer">
        <span>Protected platform for authorized administrators only</span>
      </footer>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #0F172A;
  padding: var(--space-3);
}

.login-card {
  width: 100%;
  max-width: 26rem;
  background-color: #FFFFFF;
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.login-brand {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  margin-bottom: var(--space-4);
}

.brand-badge {
  width: 3.5rem;
  height: 3.5rem;
  background: linear-gradient(135deg, #1E40AF 0%, #3B82F6 100%);
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: var(--space-2);
}

.login-brand h1 {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: 800;
  color: #0F172A;
}

.login-brand p {
  margin: var(--space-0-5) 0 0;
  font-size: var(--text-sm);
  color: #64748B;
  font-weight: 600;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-0-5);
}

.form-group label {
  font-size: var(--text-xs);
  font-weight: 700;
  color: #334155;
}

.form-group input {
  height: 2.75rem;
  padding: 0 var(--space-2);
  border: 1px solid #CBD5E1;
  border-radius: var(--radius-sm);
  background-color: #F8FAFC;
  font-size: var(--text-sm);
  color: #0F172A;
}

.form-group input:focus {
  outline: 2px solid #1E40AF;
  background-color: #FFFFFF;
}

.login-btn {
  height: 3rem;
  margin-top: var(--space-1);
}

.error-alert {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-2);
  background-color: #FEF2F2;
  color: #B91C1C;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 600;
}

.login-footer {
  margin-top: var(--space-4);
  text-align: center;
  font-size: 11px;
  color: #94A3B8;
}
</style>
