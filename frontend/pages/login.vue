<script setup lang="ts">
definePageMeta({ layout: 'auth' })

const { user, fetchUser } = useAuth()
const { locale, setLocale, l } = useLocale()

if (!user.value) await fetchUser()
if (user.value) await navigateTo('/')

const errorMessage = ref('')
const facebookLoading = ref(false)
const emailLoading = ref(false)
const email = ref('')
const password = ref('')
const showPassword = ref(false)

const route = useRoute()
const facebookResult = route.query.facebook
if (facebookResult === 'error') {
  errorMessage.value = l({
    lo: 'ບໍ່ສາມາດເຂົ້າດ້ວຍ Facebook ໄດ້ ກະລຸນາເລີ່ມໃໝ່',
    th: 'ไม่สามารถเข้าสู่ระบบด้วย Facebook ได้ โปรดเริ่มใหม่',
    en: 'Facebook sign-in could not be completed. Please try again.'
  })
}
if (facebookResult === 'expired') {
  errorMessage.value = l({
    lo: 'ຄຳຮ້ອງຂໍ Facebook ໝົດອາຍຸ ກະລຸນາກົດປຸ່ມເຂົ້າສູ່ລະບົບໃໝ່',
    th: 'คำขอ Facebook หมดอายุ โปรดกดปุ่มเข้าสู่ระบบใหม่อีกครั้ง',
    en: 'This Facebook session expired. Please click Continue with Facebook again.'
  })
}

async function loginWithFacebook() {
  facebookLoading.value = true
  errorMessage.value = ''
  try {
    const result = await $fetch<{ authorizationUrl: string }>('/api/auth/facebook/start', { method: 'POST' })
    window.location.assign(result.authorizationUrl)
  } catch {
    errorMessage.value = l({
      lo: 'ບໍ່ສາມາດເລີ່ມຕົ້ນ Facebook Login ໄດ້ ກະລຸນາລອງໃໝ່',
      th: 'ไม่สามารถเริ่ม Facebook Login ได้ กรุณาลองใหม่อีกครั้ง',
      en: 'Unable to initialize Facebook Login. Please try again.'
    })
    facebookLoading.value = false
  }
}

async function loginWithEmail() {
  if (!email.value.trim() || !password.value) {
    errorMessage.value = l({
      lo: 'ກະລຸນາໃສ່ອີເມວ ແລະ ລະຫັດຜ່ານ',
      th: 'กรุณากรอกอีเมลและรหัสผ่าน',
      en: 'Email and password are required.'
    })
    return
  }
  emailLoading.value = true
  errorMessage.value = ''
  try {
    await $fetch('/api/auth/login', {
      method: 'POST',
      body: { email: email.value.trim(), password: password.value }
    })
    navigateTo('/meta-pages')
  } catch (err: any) {
    errorMessage.value = err?.statusMessage || err?.data?.message || err?.message || l({
      lo: 'ອີເມວ ຫຼື ລະຫັດຜ່ານບໍ່ຖືກຕ້ອງ',
      th: 'อีเมลหรือรหัสผ่านไม่ถูกต้อง',
      en: 'Invalid email or password.'
    })
    emailLoading.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <!-- Top-Right Language Switcher -->
    <header class="login-header">
      <div class="lang-switch" :aria-label="l({ lo: 'ເລືອກພາສາ', th: 'เลือกภาษา', en: 'Select Language' })">
        <button
          type="button"
          :class="{ active: locale === 'lo' }"
          @click="setLocale('lo')"
        >
          ລາວ
        </button>
        <button
          type="button"
          :class="{ active: locale === 'th' }"
          @click="setLocale('th')"
        >
          ไทย
        </button>
        <button
          type="button"
          :class="{ active: locale === 'en' }"
          @click="setLocale('en')"
        >
          EN
        </button>
      </div>
    </header>

    <!-- Main Login Shell -->
    <section class="login-shell">
      <!-- Left Side: Modern Brand & Feature Showcase -->
      <aside class="brand-panel">
        <div class="brand-top">
          <div class="brand-badge-row">
            <div class="brand-logo">
              <v-icon icon="mdi-package-variant-closed" size="24" color="white" />
            </div>
            <div class="brand-naming">
              <span class="brand-name">Packdee</span>
              <span class="brand-tag">E-Commerce OS</span>
            </div>
          </div>
        </div>

        <div class="brand-hero">
          <h1>
            {{ l({
              lo: 'ຈັດການຮ້ານ, ແພັກອໍເດີ, ແຊັດ ແລະ ໂຄສະນາ ຄົບຈົບໃນບ່ອນດຽວ',
              th: 'จัดการร้าน, แพ็คออเดอร์, แชท และโฆษณา ครบจบในที่เดียว',
              en: 'Orders, Fulfillment, Live Chat & Ads Intelligence in One Platform'
            }) }}
          </h1>
          <p class="hero-desc">
            {{ l({
              lo: 'ຕິດຕາມເສັ້ນທາງຈາກ Campaign ໂຄສະນາ ຫາ Auto Reply, ອໍເດີ, ການແພັກ ແລະ ລາຍຮັບ ໂດຍບໍ່ຕ້ອງສະຫຼັບຫຼາຍແອັບ.',
              th: 'ติดตามเส้นทางจากแคมเปญโฆษณา สู่ระบบตอบกลับอัตโนมัติ ออเดอร์ การแพ็ค และรายได้ โดยไม่ต้องสลับหลายระบบ',
              en: 'Streamline customer conversations, multi-warehouse stock, express waybills, and real-time ad performance.'
            }) }}
          </p>
        </div>

        <!-- Value Highlights -->
        <div class="features-list">
          <div class="feature-card">
            <div class="feature-icon icon-emerald">
              <v-icon icon="mdi-package-variant-closed-check" size="20" />
            </div>
            <div class="feature-text">
              <strong>
                {{ l({
                  lo: 'Orders & Fulfillment (ແພັກດີ & ສົ່ງໄວ)',
                  th: 'Orders & Fulfillment (แพ็คไว & จัดส่งแม่นยำ)',
                  en: 'Orders & Fulfillment'
                }) }}
              </strong>
              <span>
                {{ l({
                  lo: 'ຈັດການອໍເດີ, ຕັດສະຕັອກສິນຄ້າ, ພິມໃບບິນ Anousith Express ທັນທີ',
                  th: 'จัดการออเดอร์, ตัดสต็อกอัตโนมัติ, พิมพ์ใบปะหน้าขนส่งทันที',
                  en: 'Order workflow, inventory sync, and instant express shipping waybills'
                }) }}
              </span>
            </div>
          </div>

          <div class="feature-card">
            <div class="feature-icon icon-blue">
              <v-icon icon="mdi-forum" size="20" />
            </div>
            <div class="feature-text">
              <strong>
                {{ l({
                  lo: 'Live Chat & WhatsApp (ຕອບໄວ 24/7)',
                  th: 'Live Chat & WhatsApp (ตอบไว 24/7)',
                  en: 'Omnichannel Chat & Automation'
                }) }}
              </strong>
              <span>
                {{ l({
                  lo: 'ລວມທຸກ Inbox ຂອງທຸກ Facebook Page, ຕອບລູກຄ້າອັດຕະໂນມັດ ບໍ່ພາດທຸກ Lead',
                  th: 'รวมทุก Inbox ของทุกเพจ, ตอบกลับอัตโนมัติ ไม่พลาดทุกลีดการขาย',
                  en: 'Unified Facebook & WhatsApp inboxes with 24/7 smart auto-replies'
                }) }}
              </span>
            </div>
          </div>

          <div class="feature-card">
            <div class="feature-icon icon-indigo">
              <v-icon icon="mdi-chart-line-variant" size="20" />
            </div>
            <div class="feature-text">
              <strong>
                {{ l({
                  lo: 'Meta Ads Intelligence (ວິເຄາະໂຄສະນາ)',
                  th: 'Meta Ads Intelligence (วิเคราะห์โฆษณา)',
                  en: 'Meta Ads Intelligence'
                }) }}
              </strong>
              <span>
                {{ l({
                  lo: 'ວັດຜົນ ROAS ແບບ Real-time, ປິດແອດຂາດທຶນ, ເພີ່ມກຳໄລຮ້ານຄ້າ',
                  th: 'วัดผล ROAS แบบ Real-time, ปิดแอดขาดทุน, ขยายกำไรธุรกิจ',
                  en: 'Real-time ROAS tracking, cost-per-lead analysis, and smart ad rules'
                }) }}
              </span>
            </div>
          </div>
        </div>

        <!-- Trust & Security Footer -->
        <div class="brand-security">
          <v-icon icon="mdi-shield-check" size="18" color="#38bdf8" />
          <span>
            {{ l({
              lo: 'ລະບົບມາດຕະຖານ Meta · ເຂົ້າສູ່ລະບົບດ້ວຍ Facebook ປອດໄພ 100% ບໍ່ເກັບລະຫັດຜ່ານ',
              th: 'ระบบมาตรฐาน Meta · เข้าสู่ระบบด้วย Facebook ปลอดภัย 100% ไม่จัดเก็บรหัสผ่าน',
              en: 'Meta Standard Platform · Secure Facebook OAuth authentication without storing passwords'
            }) }}
          </span>
        </div>
      </aside>

      <!-- Right Side: Login Form -->
      <section class="form-panel">
        <div class="form-container">
          <!-- Mobile Brand Indicator -->
          <div class="mobile-brand">
            <div class="mobile-brand-icon">
              <v-icon icon="mdi-package-variant-closed" size="22" color="white" />
            </div>
            <div class="mobile-brand-text">
              <strong>Packdee</strong>
              <span>E-Commerce OS</span>
            </div>
          </div>

          <!-- Titles -->
          <div class="form-head">
            <h2>
              {{ l({
                lo: 'ເຂົ້າສູ່ລະບົບ Packdee',
                th: 'เข้าสู่ระบบ Packdee',
                en: 'Sign in to Packdee'
              }) }}
            </h2>
            <p class="form-subtitle">
              {{ l({
                lo: 'ເຂົ້າສູ່ Workspace ຂອງທ່ານເພື່ອຈັດການຮ້ານຄ້າ, ອໍເດີ ແລະ ໂຄສະນາ',
                th: 'เข้าสู่เวิร์กสเปซของคุณเพื่อจัดการร้านค้า ออเดอร์ และโฆษณา',
                en: 'Access your workspace to manage stores, orders, and ad campaigns'
              }) }}
            </p>
          </div>

          <!-- Error Alert -->
          <transition name="fade">
            <div v-if="errorMessage" class="error-banner">
              <v-icon icon="mdi-alert-circle-outline" size="20" color="#dc2626" />
              <div class="error-text">{{ errorMessage }}</div>
              <button type="button" class="error-dismiss" @click="errorMessage = ''" aria-label="Close">
                <v-icon icon="mdi-close" size="16" />
              </button>
            </div>
          </transition>

          <!-- Email / Password Form -->
          <form class="login-form" @submit.prevent="loginWithEmail">
            <div class="form-group">
              <label for="login-email">
                {{ l({ lo: 'ອີເມວ (Email)', th: 'อีเมล (Email)', en: 'Email Address' }) }}
              </label>
              <div class="input-wrapper">
                <v-icon icon="mdi-email-outline" size="20" class="input-icon" />
                <input
                  id="login-email"
                  v-model="email"
                  type="email"
                  required
                  autocomplete="email"
                  placeholder="name@business.com"
                  class="field-input"
                  :class="{ 'has-error': !!errorMessage }"
                />
              </div>
            </div>

            <div class="form-group">
              <div class="label-row">
                <label for="login-password">
                  {{ l({ lo: 'ລະຫັດຜ່ານ (Password)', th: 'รหัสผ่าน (Password)', en: 'Password' }) }}
                </label>
              </div>
              <div class="input-wrapper">
                <v-icon icon="mdi-lock-outline" size="20" class="input-icon" />
                <input
                  id="login-password"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  required
                  autocomplete="current-password"
                  :placeholder="l({ lo: 'ໃສ່ລະຫັດຜ່ານ', th: 'กรอกรหัสผ่าน', en: 'Enter your password' })"
                  class="field-input"
                  :class="{ 'has-error': !!errorMessage }"
                />
                <button
                  type="button"
                  class="password-toggle"
                  :aria-label="showPassword ? 'Hide password' : 'Show password'"
                  @click="showPassword = !showPassword"
                >
                  <v-icon :icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'" size="20" />
                </button>
              </div>
            </div>

            <button
              type="submit"
              class="btn-submit"
              :disabled="emailLoading"
            >
              <v-progress-circular v-if="emailLoading" indeterminate size="20" width="2" color="white" />
              <span v-else>
                {{ l({ lo: 'ເຂົ້າສູ່ລະບົບ', th: 'เข้าสู่ระบบ', en: 'Sign In' }) }}
              </span>
            </button>
          </form>

          <!-- Divider -->
          <div class="divider">
            <span class="divider-line"></span>
            <span class="divider-text">
              {{ l({ lo: 'ຫຼື ເຂົ້າດ້ວຍບັນຊີເຈົ້າຂອງຮ້ານ', th: 'หรือ เข้าด้วยบัญชีเจ้าของร้าน', en: 'Or Continue as Store Owner' }) }}
            </span>
            <span class="divider-line"></span>
          </div>

          <!-- Facebook Login Button -->
          <button
            type="button"
            class="btn-facebook"
            :disabled="facebookLoading"
            @click="loginWithFacebook"
          >
            <v-progress-circular v-if="facebookLoading" indeterminate size="22" width="2" color="white" />
            <template v-else>
              <v-icon icon="mdi-facebook" size="22" color="white" />
              <span>
                {{ l({ lo: 'ສືບຕໍ່ດ້ວຍ Facebook', th: 'ดำเนินการต่อด้วย Facebook', en: 'Continue with Facebook' }) }}
              </span>
            </template>
          </button>

          <p class="facebook-hint">
            <v-icon icon="mdi-shield-account-outline" size="16" />
            <span>
              {{ l({
                lo: 'ສຳລັບເຈົ້າຂອງຮ້ານຄ້າ (Owner) ເພື່ອເຊື່ອມຕໍ່ Facebook Page, Ads ແລະ WhatsApp ໂດຍກົງ.',
                th: 'สำหรับเจ้าของร้านค้า เพื่อเชื่อมต่อ Facebook Page, Ads และ WhatsApp โดยตรง',
                en: 'For store owners to seamlessly link Facebook Pages, Ads Manager and WhatsApp.'
              }) }}
            </span>
          </p>

          <footer class="form-footer">
            <span>© {{ new Date().getFullYear() }} Packdee. All rights reserved.</span>
          </footer>
        </div>
      </section>
    </section>
  </main>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  place-items: center;
  padding: 32px 20px;
  background: #f1f5f9;
  position: relative;
  font-family: inherit;
}

/* Language selector floating top-right */
.login-header {
  position: absolute;
  top: 24px;
  right: 28px;
  z-index: 10;
}

.lang-switch {
  display: flex;
  background: #ffffff;
  padding: 4px;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.04);
  gap: 4px;
}

.lang-switch button {
  padding: 6px 14px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: #64748b;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s ease;
}

.lang-switch button:hover {
  color: #0f172a;
  background: #f8fafc;
}

.lang-switch button.active {
  background: #2563eb;
  color: #ffffff;
  box-shadow: 0 2px 6px rgba(37, 99, 235, 0.25);
}

/* Main card shell */
.login-shell {
  width: min(1080px, 100%);
  min-height: 680px;
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(0, 1fr);
  background: #ffffff;
  border-radius: 24px;
  overflow: hidden;
  box-shadow: 0 25px 60px -15px rgba(15, 23, 42, 0.15), 0 0 0 1px rgba(15, 23, 42, 0.05);
}

/* Left Hero Brand Panel (Dark Slate / Deep Navy) */
.brand-panel {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding: 48px;
  background: linear-gradient(150deg, #090e1a 0%, #0f172a 45%, #132247 100%);
  color: #ffffff;
  position: relative;
  overflow: hidden;
}

.brand-panel::before {
  content: '';
  position: absolute;
  top: -120px;
  right: -120px;
  width: 340px;
  height: 340px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(37, 99, 235, 0.25) 0%, rgba(37, 99, 235, 0) 70%);
  pointer-events: none;
}

.brand-panel::after {
  content: '';
  position: absolute;
  bottom: -100px;
  left: -80px;
  width: 280px;
  height: 280px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(6, 182, 212, 0.18) 0%, rgba(6, 182, 212, 0) 70%);
  pointer-events: none;
}

.brand-badge-row {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  position: relative;
  z-index: 1;
}

.brand-logo {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, #2563eb 0%, #06b6d4 100%);
  display: grid;
  place-items: center;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.4);
}

.brand-naming {
  display: flex;
  align-items: center;
  gap: 8px;
}

.brand-name {
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
}

.brand-tag {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 3px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.1);
  color: #38bdf8;
  border: 1px solid rgba(56, 189, 248, 0.25);
}

.brand-hero {
  margin: 36px 0 28px;
  position: relative;
  z-index: 1;
}

.brand-hero h1 {
  font-size: clamp(24px, 2.5vw, 32px);
  font-weight: 800;
  line-height: 1.28;
  letter-spacing: -0.025em;
  color: #ffffff;
  margin: 0 0 14px;
}

.hero-desc {
  font-size: 14.5px;
  line-height: 1.65;
  color: #94a3b8;
  margin: 0;
  max-width: 440px;
}

/* Feature list */
.features-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 28px;
  position: relative;
  z-index: 1;
}

.feature-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 13px 16px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(8px);
  transition: all 0.2s ease;
}

.feature-card:hover {
  background: rgba(255, 255, 255, 0.07);
  border-color: rgba(255, 255, 255, 0.16);
  transform: translateY(-1px);
}

.feature-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.icon-emerald {
  background: rgba(16, 185, 129, 0.16);
  color: #34d399;
}

.icon-blue {
  background: rgba(37, 99, 235, 0.2);
  color: #60a5fa;
}

.icon-indigo {
  background: rgba(147, 51, 234, 0.2);
  color: #c084fc;
}

.feature-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.feature-text strong {
  font-size: 13.5px;
  font-weight: 700;
  color: #f1f5f9;
}

.feature-text span {
  font-size: 12px;
  line-height: 1.5;
  color: #94a3b8;
}

.brand-security {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-top: 18px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  color: #94a3b8;
  font-size: 12px;
  line-height: 1.5;
  position: relative;
  z-index: 1;
}

/* Right Side: Form Panel */
.form-panel {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
  background: #ffffff;
}

.form-container {
  width: 100%;
  max-width: 400px;
}

.mobile-brand {
  display: none;
  align-items: center;
  gap: 12px;
  margin-bottom: 28px;
}

.mobile-brand-icon {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  background: linear-gradient(135deg, #2563eb 0%, #06b6d4 100%);
  display: grid;
  place-items: center;
}

.mobile-brand-text {
  display: flex;
  flex-direction: column;
}

.mobile-brand-text strong {
  font-size: 20px;
  font-weight: 800;
  color: #0f172a;
}

.mobile-brand-text span {
  font-size: 11px;
  color: #2563eb;
  font-weight: 700;
  text-transform: uppercase;
}

.form-head {
  margin-bottom: 24px;
}

.form-head h2 {
  font-size: 26px;
  font-weight: 800;
  color: #0f172a;
  letter-spacing: -0.02em;
  margin: 0 0 8px;
}

.form-subtitle {
  font-size: 14px;
  color: #64748b;
  line-height: 1.5;
  margin: 0;
}

/* Error Banner */
.error-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  border-radius: 12px;
  color: #b91c1c;
  font-size: 13px;
  margin-bottom: 20px;
}

.error-text {
  flex: 1;
  font-weight: 500;
  line-height: 1.4;
}

.error-dismiss {
  border: none;
  background: transparent;
  color: #991b1b;
  cursor: pointer;
  padding: 2px;
  display: grid;
  place-items: center;
}

/* Form inputs */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.form-group label {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.label-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 14px;
  color: #94a3b8;
  pointer-events: none;
}

.field-input {
  width: 100%;
  height: 48px;
  padding: 0 16px 0 44px;
  background: #f8fafc;
  border: 1.5px solid #e2e8f0;
  border-radius: 12px;
  font-size: 14px;
  color: #0f172a;
  transition: all 0.15s ease;
  font-family: inherit;
}

.field-input:focus {
  outline: none;
  background: #ffffff;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
}

.field-input.has-error {
  border-color: #ef4444;
  background: #fffafa;
}

.password-toggle {
  position: absolute;
  right: 12px;
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  display: grid;
  place-items: center;
  border-radius: 8px;
}

.password-toggle:hover {
  color: #334155;
}

/* Submit Button */
.btn-submit {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  color: #ffffff;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 4px 14px rgba(37, 99, 235, 0.3);
  margin-top: 4px;
}

.btn-submit:hover:not(:disabled) {
  background: linear-gradient(135deg, #1d4ed8 0%, #1e40af 100%);
  box-shadow: 0 6px 18px rgba(37, 99, 235, 0.4);
  transform: translateY(-1px);
}

.btn-submit:disabled {
  opacity: 0.75;
  cursor: not-allowed;
}

/* Divider */
.divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 22px 0 18px;
}

.divider-line {
  flex: 1;
  height: 1px;
  background: #e2e8f0;
}

.divider-text {
  font-size: 12px;
  font-weight: 600;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
}

/* Facebook Button */
.btn-facebook {
  width: 100%;
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: #1877f2;
  color: #ffffff;
  border: none;
  border-radius: 12px;
  font-size: 14.5px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 4px 14px rgba(24, 119, 242, 0.25);
}

.btn-facebook:hover:not(:disabled) {
  background: #166fe5;
  box-shadow: 0 6px 18px rgba(24, 119, 242, 0.35);
  transform: translateY(-1px);
}

.btn-facebook:disabled {
  opacity: 0.75;
  cursor: not-allowed;
}

.facebook-hint {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin: 14px 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: #64748b;
}

.facebook-hint .v-icon {
  color: #1877f2;
  flex-shrink: 0;
  margin-top: 2px;
}

.form-footer {
  margin-top: 28px;
  text-align: center;
  font-size: 12px;
  color: #94a3b8;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Responsive adjustments */
@media (max-width: 960px) {
  .login-shell {
    grid-template-columns: 1fr;
    max-width: 520px;
    min-height: auto;
  }

  .brand-panel {
    display: none;
  }

  .form-panel {
    padding: 40px 28px;
  }

  .mobile-brand {
    display: flex;
  }
}

@media (max-width: 480px) {
  .login-page {
    padding: 12px;
  }

  .login-header {
    top: 16px;
    right: 16px;
  }

  .login-shell {
    border-radius: 18px;
  }

  .form-panel {
    padding: 32px 20px;
  }

  .form-head h2 {
    font-size: 22px;
  }
}
</style>
