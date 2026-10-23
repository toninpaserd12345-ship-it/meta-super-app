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
    lo: 'ບໍ່ສາມາດເຂົ້າສູ່ລະບົບດ້ວຍ Facebook ໄດ້ ກະລຸນາລອງໃໝ່ອີກຄັ້ງ',
    th: 'ไม่สามารถเข้าสู่ระบบด้วย Facebook ได้ กรุณาลองใหม่อีกครั้ง',
    en: 'Facebook sign-in could not be completed. Please try again.'
  })
}
if (facebookResult === 'expired') {
  errorMessage.value = l({
    lo: 'ການເຊື່ອມຕໍ່ໝົດເວລາ ກະລຸນາກົດປຸ່ມເຂົ້າສູ່ລະບົບໃໝ່',
    th: 'การเชื่อมต่อหมดอายุ กรุณากดปุ่มเข้าสู่ระบบใหม่อีกครั้ง',
    en: 'This session expired. Please click Continue with Facebook again.'
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
      lo: 'ບໍ່ສາມາດເຊື່ອມຕໍ່ Facebook ໄດ້ໃນເວລານີ້ ກະລຸນາລອງໃໝ່',
      th: 'ไม่สามารถเชื่อมต่อ Facebook ได้ในขณะนี้ กรุณาลองใหม่อีกครั้ง',
      en: 'Unable to connect to Facebook. Please try again.'
    })
    facebookLoading.value = false
  }
}

async function loginWithEmail() {
  if (!email.value.trim() || !password.value) {
    errorMessage.value = l({
      lo: 'ກະລຸນາປ້ອນອີເມວ ແລະ ລະຫັດຜ່ານໃຫ້ຄົບຖ້ວນ',
      th: 'กรุณากรอกอีเมลและรหัสผ่านให้ครบถ้วน',
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
      lo: 'ອີເມວ ຫຼື ລະຫັດຜ່ານບໍ່ຖືກຕ້ອງ ກະລຸນາກວດສອບຄືນ',
      th: 'อีเมลหรือรหัสผ่านไม่ถูกต้อง กรุณาตรวจสอบอีกครั้ง',
      en: 'Invalid email or password. Please try again.'
    })
    emailLoading.value = false
  }
}
</script>

<template>
  <main class="login-page">
    <!-- Top Floating Language Switcher -->
    <header class="lang-bar">
      <div class="lang-pill">
        <button
          type="button"
          :class="{ active: locale === 'lo' }"
          @click="setLocale('lo')"
        >
          🇱🇦 ລາວ
        </button>
        <button
          type="button"
          :class="{ active: locale === 'th' }"
          @click="setLocale('th')"
        >
          🇹🇭 ไทย
        </button>
        <button
          type="button"
          :class="{ active: locale === 'en' }"
          @click="setLocale('en')"
        >
          🇬🇧 EN
        </button>
      </div>
    </header>

    <!-- Main Card Container -->
    <div class="login-wrapper">
      <div class="login-card">
        <!-- Left Side: Warm, Friendly Merchant Showcase -->
        <div class="showcase-side">
          <!-- Brand Header -->
          <div class="showcase-brand">
            <div class="logo-box">
              <v-icon icon="mdi-package-variant-closed" size="26" color="white" />
            </div>
            <div class="logo-title">
              <span class="app-name">Packdee</span>
              <span class="app-slogan">
                {{ l({ lo: 'ແພັກດີ · ລະບົບຊ່ວຍຂາຍອອນລາຍ', th: 'แพ็คดี · ระบบช่วยขายออนไลน์', en: 'Online Store Assistant' }) }}
              </span>
            </div>
          </div>

          <!-- Hero Headline -->
          <div class="showcase-headline">
            <h1>
              {{ l({
                lo: 'ຊ່ວຍໃຫ້ທຸກການຂາຍ, ແພັກເຄື່ອງ ແລະ ຕອບແຊັດ ເປັນເລື່ອງງ່າຍ',
                th: 'ช่วยให้ทุกการขาย, แพ็คสินค้า และตอบแชท เป็นเรื่องง่าย',
                en: 'Make selling, packing, and chatting effortless for your shop'
              }) }}
            </h1>
            <p>
              {{ l({
                lo: 'ລວມແຊັດ Facebook & WhatsApp, ຕອບລູກຄ້າອັດຕະໂນມັດ 24 ຊົ່ວໂມງ, ຕັດສະຕັອກ ແລະ ພິມໃບບິນຂົນສົ່ງ ຄົບຈົບໃນແອັບດຽວ.',
                th: 'รวมแชท Facebook & WhatsApp, ตอบลูกค้าอัตโนมัติ 24 ชม., ตัดสต็อก และพิมพ์ใบปะหน้าขนส่ง ครบจบในที่เดียว',
                en: 'Unified chats, 24/7 auto-replies, inventory tracking, and express waybills in one friendly place.'
              }) }}
            </p>
          </div>

          <!-- Visual Activity Mockups (Demonstrates Real Shop Usage) -->
          <div class="shop-demo-cards">
            <!-- Demo 1: Customer Chat -->
            <div class="demo-card chat-demo">
              <div class="demo-badge">
                <v-icon icon="mdi-chat" size="14" />
                <span>{{ l({ lo: 'ແຊັດສົດ & ຕອບອັດຕະໂນມັດ', th: 'แชทสด & ตอบอัตโนมัติ', en: 'Live Chat & Bot' }) }}</span>
              </div>
              <div class="chat-bubble-user">
                <span>{{ l({ lo: '👤 ລູກຄ້າ: ສົນໃຈຊຸດນີ້ເຈົ້າ ມີເຄື່ອງພ້ອມສົ່ງບໍ່?', th: '👤 ลูกค้า: สนใจชุดนี้ค่ะ มีของพร้อมส่งไหม?', en: '👤 Customer: Is this item in stock?' }) }}</span>
              </div>
              <div class="chat-bubble-bot">
                <span>{{ l({ lo: '🤖 ບອດແພັກດີ: ມີພ້ອມສົ່ງເຈົ້າ! ລະບົບລັອກສະຕັອກໃຫ້ແລ້ວ 📦', th: '🤖 บอทแพ็คดี: มีพร้อมส่งค่ะ! ระบบล็อคสต็อกให้แล้ว 📦', en: '🤖 Packdee: In stock! Reserved for you 📦' }) }}</span>
              </div>
            </div>

            <!-- Demo 2: Quick Packing & Waybill -->
            <div class="demo-card waybill-demo">
              <div class="waybill-left">
                <div class="waybill-icon">
                  <v-icon icon="mdi-printer-check" size="22" color="#059669" />
                </div>
                <div class="waybill-info">
                  <strong>{{ l({ lo: 'ພິມໃບບິນ Anousith Express', th: 'พิมพ์ใบปะหน้าขนส่ง', en: 'Express Waybill Printing' }) }}</strong>
                  <small>{{ l({ lo: 'ອໍເດີ #PK-8821 · ພ້ອມສົ່ງທັນທີ', th: 'ออเดอร์ #PK-8821 · พร้อมส่งทันที', en: 'Order #PK-8821 · Ready to ship' }) }}</small>
                </div>
              </div>
              <span class="status-pill-ready">
                <v-icon icon="mdi-check" size="14" />
                {{ l({ lo: 'ແພັກແລ້ວ', th: 'แพ็คแล้ว', en: 'Packed' }) }}
              </span>
            </div>

            <!-- Demo 3: Shop Sales Today -->
            <div class="demo-card stats-demo">
              <div class="stats-item">
                <small>{{ l({ lo: 'ຍອດຂາຍມື້ນີ້', th: 'ยอดขายวันนี้', en: 'Today’s Sales' }) }}</small>
                <span class="sales-number">₭ 4,850,000</span>
              </div>
              <div class="stats-growth">
                <v-icon icon="mdi-trending-up" size="16" />
                <span>+32 {{ l({ lo: 'ອໍເດີ', th: 'ออเดอร์', en: 'orders' }) }}</span>
              </div>
            </div>
          </div>

          <!-- Trust Footer -->
          <div class="showcase-trust">
            <v-icon icon="mdi-shield-check" size="18" color="#60a5fa" />
            <span>
              {{ l({
                lo: 'ເຊື່ອມຕໍ່ປອດໄພດ້ວຍ Facebook ຢ່າງເປັນທາງການ · ໃຊ້ງານງ່າຍ ບໍ່ຕ້ອງມີຄວາມຮູ້ໄອທີ',
                th: 'เชื่อมต่อปลอดภัยด้วย Facebook อย่างเป็นทางการ · ใช้งานง่าย ไม่ต้องมีความรู้ไอที',
                en: 'Official Facebook Partner Integration · Simple, secure and easy to use'
              }) }}
            </span>
          </div>
        </div>

        <!-- Right Side: Friendly Form Side -->
        <div class="form-side">
          <div class="form-inner">
            <!-- Mobile Brand Header -->
            <div class="mobile-logo-row">
              <div class="logo-box-mobile">
                <v-icon icon="mdi-package-variant-closed" size="24" color="white" />
              </div>
              <div>
                <strong class="mobile-name">Packdee</strong>
                <span class="mobile-tag">{{ l({ lo: 'ລະບົບຊ່ວຍຂາຍອອນລາຍ', th: 'ระบบช่วยขายออนไลน์', en: 'Shop Assistant' }) }}</span>
              </div>
            </div>

            <!-- Header Titles -->
            <div class="form-intro">
              <h2>
                {{ l({
                  lo: 'ເຂົ້າສູ່ລະບົບຮ້ານຄ້າ',
                  th: 'เข้าสู่ระบบร้านค้า',
                  en: 'Sign In to Your Shop'
                }) }}
              </h2>
              <p>
                {{ l({
                  lo: 'ຍິນດີຕ້ອນຮັບສູ່ Packdee! ເຂົ້າຈັດການອໍເດີ ແລະ ຕອບແຊັດລູກຄ້າ',
                  th: 'ยินดีต้อนรับสู่ Packdee! เข้าจัดการออเดอร์และตอบแชทลูกค้า',
                  en: 'Welcome to Packdee! Manage orders and customer messages'
                }) }}
              </p>
            </div>

            <!-- Error Notification -->
            <transition name="fade">
              <div v-if="errorMessage" class="friendly-error">
                <v-icon icon="mdi-alert-circle" size="20" color="#dc2626" />
                <span>{{ errorMessage }}</span>
                <button type="button" @click="errorMessage = ''" class="close-err">
                  <v-icon icon="mdi-close" size="16" />
                </button>
              </div>
            </transition>

            <!-- 1-Click Facebook Login for Shop Owners (Prominent, Big, Friendly) -->
            <div class="facebook-section">
              <button
                type="button"
                class="fb-main-btn"
                :disabled="facebookLoading"
                @click="loginWithFacebook"
              >
                <v-progress-circular v-if="facebookLoading" indeterminate size="22" width="2" color="white" />
                <template v-else>
                  <div class="fb-icon-circle">
                    <v-icon icon="mdi-facebook" size="24" color="#1877f2" />
                  </div>
                  <div class="fb-btn-text">
                    <span class="fb-title">
                      {{ l({
                        lo: 'ເຂົ້າສູ່ລະບົບດ້ວຍ Facebook',
                        th: 'เข้าสู่ระบบด้วย Facebook',
                        en: 'Continue with Facebook'
                      }) }}
                    </span>
                    <small class="fb-subtitle">
                      {{ l({
                        lo: 'ແນະນຳສຳລັບເຈົ້າຂອງຮ້ານ (ເຊື່ອມຕໍ່ເພຈອັດຕະໂນມັດ)',
                        th: 'แนะนำสำหรับเจ้าของร้าน (เชื่อมต่อเพจทันที)',
                        en: 'Recommended for Store Owners'
                      }) }}
                    </small>
                  </div>
                </template>
              </button>
            </div>

            <!-- Friendly Divider -->
            <div class="form-divider">
              <span>{{ l({ lo: 'ຫຼື ເຂົ້າດ້ວຍອີເມວຂອງທີມງານ', th: 'หรือ เข้าด้วยอีเมลของทีมงาน', en: 'Or sign in with email' }) }}</span>
            </div>

            <!-- Email & Password Form -->
            <form class="clean-form" @submit.prevent="loginWithEmail">
              <div class="input-block">
                <label for="shop-email">
                  {{ l({ lo: 'ອີເມວ (Email)', th: 'อีเมล (Email)', en: 'Email' }) }}
                </label>
                <div class="input-field-wrap">
                  <v-icon icon="mdi-email-outline" size="20" class="field-icon" />
                  <input
                    id="shop-email"
                    v-model="email"
                    type="email"
                    required
                    autocomplete="email"
                    :placeholder="l({ lo: 'ອີເມວຮ້ານຄ້າຂອງທ່ານ', th: 'อีเมลร้านค้าของคุณ', en: 'your-email@example.com' })"
                    class="shop-input"
                  />
                </div>
              </div>

              <div class="input-block">
                <label for="shop-password">
                  {{ l({ lo: 'ລະຫັດຜ່ານ (Password)', th: 'รหัสผ่าน (Password)', en: 'Password' }) }}
                </label>
                <div class="input-field-wrap">
                  <v-icon icon="mdi-lock-outline" size="20" class="field-icon" />
                  <input
                    id="shop-password"
                    v-model="password"
                    :type="showPassword ? 'text' : 'password'"
                    required
                    autocomplete="current-password"
                    :placeholder="l({ lo: 'ໃສ່ລະຫັດຜ່ານ', th: 'กรอกรหัสผ่าน', en: 'Enter password' })"
                    class="shop-input"
                  />
                  <button
                    type="button"
                    class="eye-btn"
                    :title="showPassword ? 'Hide password' : 'Show password'"
                    @click="showPassword = !showPassword"
                  >
                    <v-icon :icon="showPassword ? 'mdi-eye-off-outline' : 'mdi-eye-outline'" size="20" />
                  </button>
                </div>
              </div>

              <button
                type="submit"
                class="login-submit-btn"
                :disabled="emailLoading"
              >
                <v-progress-circular v-if="emailLoading" indeterminate size="20" width="2" color="white" />
                <span v-else>
                  {{ l({ lo: 'ເຂົ້າສູ່ລະບົບ', th: 'เข้าสู่ระบบ', en: 'Sign In' }) }}
                </span>
              </button>
            </form>

            <div class="card-footer-info">
              <span>Packdee (ແພັກດີ) · {{ l({ lo: 'ລະບົບຊ່ວຍຂາຍອອນລາຍຄົບວົງຈອນ', th: 'ระบบช่วยขายออนไลน์ครบวงจร', en: 'All-in-One E-Commerce' }) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </main>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 32px 16px;
  background: linear-gradient(135deg, #f0f4ff 0%, #f8faff 50%, #eef2ff 100%);
  position: relative;
  font-family: inherit;
}

/* Floating Language Bar */
.lang-bar {
  position: absolute;
  top: 20px;
  right: 24px;
  z-index: 10;
}

.lang-pill {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #ffffff;
  padding: 4px 6px;
  border-radius: 999px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 4px 12px rgba(15, 23, 42, 0.05);
}

.lang-pill button {
  border: none;
  background: transparent;
  padding: 6px 12px;
  font-size: 13px;
  font-weight: 600;
  color: #64748b;
  border-radius: 999px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.lang-pill button:hover {
  color: #1e40af;
  background: #f1f5f9;
}

.lang-pill button.active {
  background: #2563eb;
  color: #ffffff;
  box-shadow: 0 2px 6px rgba(37, 99, 235, 0.3);
}

/* Wrapper & Main Card */
.login-wrapper {
  width: 100%;
  max-width: 1080px;
}

.login-card {
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  background: #ffffff;
  border-radius: 28px;
  box-shadow: 0 20px 60px -15px rgba(30, 58, 138, 0.12), 0 0 0 1px rgba(226, 232, 240, 0.8);
  overflow: hidden;
  min-height: 640px;
}

/* Left: Warm, Vibrant Showcase Side */
.showcase-side {
  background: linear-gradient(150deg, #1e3a8a 0%, #2563eb 55%, #3b82f6 100%);
  color: #ffffff;
  padding: 44px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  position: relative;
  overflow: hidden;
}

.showcase-side::before {
  content: '';
  position: absolute;
  top: -80px;
  right: -80px;
  width: 260px;
  height: 260px;
  border-radius: 50%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.18) 0%, rgba(255, 255, 255, 0) 70%);
  pointer-events: none;
}

.showcase-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.logo-box {
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.2);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255, 255, 255, 0.3);
  display: grid;
  place-items: center;
  box-shadow: 0 8px 16px rgba(0, 0, 0, 0.1);
}

.logo-title {
  display: flex;
  flex-direction: column;
}

.app-name {
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: #ffffff;
}

.app-slogan {
  font-size: 12px;
  font-weight: 600;
  color: #dbeafe;
}

.showcase-headline {
  margin: 28px 0 20px;
}

.showcase-headline h1 {
  font-size: clamp(22px, 2.2vw, 28px);
  font-weight: 800;
  line-height: 1.35;
  color: #ffffff;
  margin: 0 0 12px;
}

.showcase-headline p {
  font-size: 14px;
  line-height: 1.6;
  color: #eff6ff;
  margin: 0;
  opacity: 0.95;
}

/* Shop Activity Demo Cards */
.shop-demo-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 16px 0 24px;
}

.demo-card {
  background: rgba(255, 255, 255, 0.15);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 16px;
  padding: 12px 16px;
  box-shadow: 0 4px 14px rgba(0, 0, 0, 0.06);
}

/* Chat Demo */
.chat-demo {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.demo-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 700;
  color: #fef08a;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.chat-bubble-user,
.chat-bubble-bot {
  font-size: 12.5px;
  line-height: 1.45;
  padding: 7px 11px;
  border-radius: 10px;
}

.chat-bubble-user {
  background: rgba(255, 255, 255, 0.2);
  color: #ffffff;
  align-self: flex-start;
}

.chat-bubble-bot {
  background: #ffffff;
  color: #1e3a8a;
  font-weight: 600;
  align-self: flex-start;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
}

/* Waybill Demo */
.waybill-demo {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.waybill-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.waybill-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: #ecfdf5;
  display: grid;
  place-items: center;
}

.waybill-info {
  display: flex;
  flex-direction: column;
}

.waybill-info strong {
  font-size: 13px;
  color: #ffffff;
}

.waybill-info small {
  font-size: 11.5px;
  color: #dbeafe;
}

.status-pill-ready {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #10b981;
  color: #ffffff;
  font-size: 11.5px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 999px;
  box-shadow: 0 2px 6px rgba(16, 185, 129, 0.3);
}

/* Stats Demo */
.stats-demo {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
}

.stats-item small {
  display: block;
  font-size: 11px;
  color: #dbeafe;
}

.sales-number {
  font-size: 16px;
  font-weight: 800;
  color: #ffffff;
}

.stats-growth {
  display: flex;
  align-items: center;
  gap: 4px;
  background: rgba(255, 255, 255, 0.22);
  color: #ffffff;
  font-size: 12px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 8px;
}

.showcase-trust {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #dbeafe;
  padding-top: 14px;
  border-top: 1px solid rgba(255, 255, 255, 0.16);
}

/* Right: Clean, Friendly Form Side */
.form-side {
  padding: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #ffffff;
}

.form-inner {
  width: 100%;
  max-width: 380px;
}

.mobile-logo-row {
  display: none;
  align-items: center;
  gap: 12px;
  margin-bottom: 24px;
}

.logo-box-mobile {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  background: linear-gradient(135deg, #1e40af 0%, #2563eb 100%);
  display: grid;
  place-items: center;
}

.mobile-name {
  display: block;
  font-size: 20px;
  font-weight: 800;
  color: #0f172a;
}

.mobile-tag {
  font-size: 12px;
  color: #2563eb;
  font-weight: 600;
}

.form-intro {
  margin-bottom: 24px;
}

.form-intro h2 {
  font-size: 26px;
  font-weight: 800;
  color: #0f172a;
  letter-spacing: -0.02em;
  margin: 0 0 6px;
}

.form-intro p {
  font-size: 14px;
  color: #64748b;
  line-height: 1.5;
  margin: 0;
}

/* Friendly Error Banner */
.friendly-error {
  display: flex;
  align-items: center;
  gap: 10px;
  background: #fef2f2;
  border: 1px solid #fee2e2;
  border-radius: 12px;
  padding: 10px 14px;
  color: #b91c1c;
  font-size: 13px;
  margin-bottom: 20px;
}

.friendly-error span {
  flex: 1;
  line-height: 1.4;
}

.close-err {
  border: none;
  background: transparent;
  color: #991b1b;
  cursor: pointer;
  padding: 2px;
}

/* Facebook Primary Option */
.facebook-section {
  margin-bottom: 20px;
}

.fb-main-btn {
  width: 100%;
  min-height: 54px;
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 8px 16px;
  background: #1877f2;
  color: #ffffff;
  border: none;
  border-radius: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 4px 16px rgba(24, 119, 242, 0.28);
  text-align: left;
}

.fb-main-btn:hover:not(:disabled) {
  background: #166fe5;
  box-shadow: 0 6px 20px rgba(24, 119, 242, 0.38);
  transform: translateY(-1px);
}

.fb-main-btn:disabled {
  opacity: 0.75;
  cursor: not-allowed;
}

.fb-icon-circle {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #ffffff;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.fb-btn-text {
  display: flex;
  flex-direction: column;
}

.fb-title {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.25;
}

.fb-subtitle {
  font-size: 11.5px;
  color: #dbeafe;
  line-height: 1.3;
  margin-top: 2px;
}

/* Divider */
.form-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 22px 0 20px;
  color: #94a3b8;
  font-size: 12.5px;
}

.form-divider::before,
.form-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: #e2e8f0;
}

/* Clean Form */
.clean-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.input-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.input-block label {
  font-size: 13px;
  font-weight: 600;
  color: #334155;
}

.input-field-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.field-icon {
  position: absolute;
  left: 14px;
  color: #94a3b8;
  pointer-events: none;
}

.shop-input {
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

.shop-input:focus {
  outline: none;
  background: #ffffff;
  border-color: #2563eb;
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.12);
}

.eye-btn {
  position: absolute;
  right: 12px;
  border: none;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  display: grid;
  place-items: center;
  border-radius: 8px;
}

.eye-btn:hover {
  color: #334155;
}

/* Submit Button */
.login-submit-btn {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f172a;
  color: #ffffff;
  border: none;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 4px 14px rgba(15, 23, 42, 0.15);
  margin-top: 4px;
}

.login-submit-btn:hover:not(:disabled) {
  background: #1e293b;
  transform: translateY(-1px);
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.22);
}

.login-submit-btn:disabled {
  opacity: 0.75;
  cursor: not-allowed;
}

.card-footer-info {
  margin-top: 26px;
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

/* Responsive */
@media (max-width: 920px) {
  .login-card {
    grid-template-columns: 1fr;
    max-width: 480px;
    min-height: auto;
  }

  .showcase-side {
    display: none;
  }

  .mobile-logo-row {
    display: flex;
  }

  .form-side {
    padding: 36px 24px;
  }
}

@media (max-width: 480px) {
  .login-page {
    padding: 12px;
  }

  .lang-bar {
    top: 14px;
    right: 14px;
  }

  .login-card {
    border-radius: 20px;
  }

  .form-side {
    padding: 28px 18px;
  }

  .form-intro h2 {
    font-size: 22px;
  }
}
</style>
