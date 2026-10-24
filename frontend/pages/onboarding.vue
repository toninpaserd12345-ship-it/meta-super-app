<script setup lang="ts">
definePageMeta({ layout: 'auth', middleware: 'auth' })

const { l, locale } = useLocale()
const { user } = useAuth()

interface Plan {
  id: string
  name: string
  price: number
  maxUsers: number
  maxPages: number
  isActive: boolean
}

interface MetaPage {
  id: string
  name: string
  category: string
  pictureUrl?: string
  connected: boolean
  tokenReady: boolean
}

interface OnboardingStatus {
  hasPlan: boolean
  plan?: {
    id: string
    name: string
    price: number
    lockedPrice: number
    status: string
    currentPeriodEnd: string
  }
  connectedPagesCount: number
  connectedPages: { id: string; name: string; category: string; pictureUrl?: string }[]
  totalAvailablePages: number
  step: 'plan' | 'page' | 'completed'
  isCompleted: boolean
}

const currentStep = ref<1 | 2 | 3>(1)
const loading = ref(true)
const plans = ref<Plan[]>([])
const pages = ref<MetaPage[]>([])
const onboardingStatus = ref<OnboardingStatus | null>(null)
const actionLoading = ref(false)
const errorMessage = ref('')

// Pro payment state
const showProPayment = ref(false)
const slipFile = ref<File | null>(null)
const slipUploading = ref(false)
const slipReference = ref('')

const route = useRoute()

const formatLAK = (price: number) => {
  if (price === 0) return l({ lo: 'ຟຣີ', th: 'ฟรี', en: 'Free' })
  return new Intl.NumberFormat(locale.value === 'lo' ? 'lo-LA' : locale.value === 'th' ? 'th-TH' : 'en-US').format(price) + ' ₭'
}

const loadData = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const [statusData, plansData] = await Promise.all([
      $fetch<OnboardingStatus>('/api/proxy/api/v1/onboarding/status'),
      $fetch<Plan[]>('/api/proxy/api/v1/billing/plans').catch(() => []),
    ])
    onboardingStatus.value = statusData
    plans.value = plansData

    // Route step
    if (route.query.meta === 'error') {
      errorMessage.value = String(route.query.reason || 'Facebook authorization failed.')
      currentStep.value = 2
    } else if (currentStep.value === 2) {
      // Keep on step 2 so user can select/review pages according to their plan
    } else if (route.query.meta === 'connected') {
      currentStep.value = 2
    } else if (statusData.step === 'plan') {
      currentStep.value = 1
    } else if (statusData.step === 'page') {
      currentStep.value = 2
    } else if (statusData.isCompleted) {
      currentStep.value = 3
    }

    // If on step 2, load pages
    if (currentStep.value === 2 || statusData.hasPlan) {
      await loadPages()
    }
  } catch (err: any) {
    errorMessage.value = err?.data?.message || err?.message || 'Failed to initialize onboarding.'
  } finally {
    loading.value = false
  }
}

const loadPages = async () => {
  try {
    const res = await $fetch<{ items: MetaPage[] }>('/api/proxy/api/v1/meta/pages')
    pages.value = res.items || []
  } catch (err) {
    console.error('Failed to load pages:', err)
  }
}

// Action: Start Pro 7-Day Free Trial
const selectTrial = async () => {
  actionLoading.value = true
  errorMessage.value = ''
  try {
    await $fetch('/api/proxy/api/v1/billing/start-trial', { method: 'POST' })
    await loadData()
    currentStep.value = 2
  } catch (err: any) {
    errorMessage.value = err?.data?.message || err?.message || 'Could not activate trial.'
  } finally {
    actionLoading.value = false
  }
}

// Action: Select Free Plan
const selectFree = async () => {
  actionLoading.value = true
  errorMessage.value = ''
  try {
    await $fetch('/api/proxy/api/v1/billing/select-free', { method: 'POST' })
    await loadData()
    currentStep.value = 2
  } catch (err: any) {
    errorMessage.value = err?.data?.message || err?.message || 'Could not select free plan.'
  } finally {
    actionLoading.value = false
  }
}

// Action: Submit Pro Slip
const onSlipSelected = async (e: Event) => {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  slipFile.value = file
  slipUploading.value = true
  errorMessage.value = ''
  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await $fetch<{ fileUrl: string; filename: string }>('/api/proxy/api/v1/storage/upload', {
      method: 'POST',
      body: formData,
    })
    slipReference.value = res.fileUrl || res.filename || 'slip-uploaded'
  } catch (err: any) {
    errorMessage.value = err?.data?.message || err?.message || 'Slip upload failed.'
  } finally {
    slipUploading.value = false
  }
}

const submitProPayment = async () => {
  if (!slipReference.value.trim()) {
    errorMessage.value = l({
      lo: 'ກະລຸນາອັບໂຫຼດຮູບສະລິບ ຫຼື ປ້ອນເລກທີອ້າງອີງການໂອນເງິນ',
      th: 'กรุณาอัปโหลดสลิปหรือระบุเลขอ้างอิงการโอนเงิน',
      en: 'Please upload transfer slip or enter reference number.'
    })
    return
  }

  actionLoading.value = true
  errorMessage.value = ''
  try {
    await $fetch('/api/proxy/api/v1/billing/checkout', {
      method: 'POST',
      body: {
        planId: 'pro',
        paymentMethod: 'bcel_one',
        reference: slipReference.value.trim(),
      },
    })
    // In the meantime, start trial so user can immediately continue setup!
    await $fetch('/api/proxy/api/v1/billing/start-trial', { method: 'POST' }).catch(() => {})
    await loadData()
    currentStep.value = 2
  } catch (err: any) {
    errorMessage.value = err?.data?.message || err?.message || 'Payment submission failed.'
  } finally {
    actionLoading.value = false
  }
}

// Action: Connect Facebook OAuth
const connectFacebook = async () => {
  actionLoading.value = true
  errorMessage.value = ''
  try {
    const res = await $fetch<{ authorizationUrl: string }>('/api/proxy/api/v1/meta/oauth/start', {
      method: 'POST',
      body: { permissions: [] },
    })
    if (res.authorizationUrl) {
      window.location.assign(res.authorizationUrl)
    }
  } catch (err: any) {
    errorMessage.value = err?.data?.message || err?.message || 'Could not start Facebook connect.'
    actionLoading.value = false
  }
}

// Action: Toggle Page Connection
const togglingPage = ref<string | null>(null)
const hasConnectedPages = computed(() => {
  return (onboardingStatus.value?.connectedPagesCount || 0) > 0 || pages.value.some(p => p.connected)
})

const togglePage = async (page: MetaPage) => {
  togglingPage.value = page.id
  errorMessage.value = ''
  const newConnected = !page.connected
  try {
    await $fetch(`/api/proxy/api/v1/meta/pages/${newConnected ? 'connect' : 'disconnect'}`, {
      method: 'POST',
      body: { pageId: page.id },
    })
    await loadPages()
    await loadData()
  } catch (err: any) {
    const code = err?.data?.error?.code
    const rawMsg = err?.data?.error?.message || err?.data?.message || err?.message || ''
    if (code === 'plan_limit_exceeded' || rawMsg.includes('plan_limit_exceeded')) {
      errorMessage.value = l({
        lo: 'ແພັກເກດປັດຈຸບັນຂອງທ່ານເຊື່ອມຕໍ່ໄດ້ຕາມໂຄຕ້າເທົ່ານັ້ນ (ກະລຸນາອັບເກຣດແພັກເກດເພື່ອເຊື່ອມຕໍ່ເພີ່ມ)',
        th: 'แพ็กเกจปัจจุบันของคุณเชื่อมต่อได้ตามโควตาแล้ว (กรุณาอัปเกรดเพื่อเชื่อมต่อเพิ่ม)',
        en: 'Your current plan limit has been reached. Please upgrade to connect more pages.',
      })
    } else {
      errorMessage.value = rawMsg || 'Could not update page.'
    }
    await loadPages()
  } finally {
    togglingPage.value = null
  }
}

const finishOnboarding = async () => {
  await navigateTo('/')
}

onMounted(loadData)
</script>

<template>
  <div class="onboarding-container">
    <!-- Header Brand -->
    <header class="onboarding-header">
      <div class="brand">
        <div class="brand-logo">
          <v-icon icon="mdi-store" size="24" color="white" />
        </div>
        <div class="brand-info">
          <h2>Packdee <span class="sub-brand">· ແພັກດີ</span></h2>
          <small>{{ l({ lo: 'ຕັ້ງຄ່າເລີ່ມຕົ້ນຮ້ານຄ້າຂອງທ່ານ', th: 'ตั้งค่าเริ่มต้นร้านค้าของคุณ', en: 'Initial Store Setup' }) }}</small>
        </div>
      </div>
      <div class="user-pill">
        <v-icon icon="mdi-account-circle" size="18" color="#64748B" />
        <span>{{ user?.name || user?.email }}</span>
      </div>
    </header>

    <!-- Stepper Navigation Bar -->
    <div class="stepper-bar">
      <div class="step-item" :class="{ active: currentStep === 1, done: currentStep > 1 }">
        <div class="step-num">
          <v-icon v-if="currentStep > 1" icon="mdi-check" size="16" />
          <span v-else>1</span>
        </div>
        <div class="step-text">
          <strong>{{ l({ lo: 'ເລືອກແພັກເກດ', th: 'เลือกแพ็กเกจ', en: 'Choose Plan' }) }}</strong>
          <small>{{ l({ lo: 'ເລືອກແຜນທີ່ເໝາະສົມ', th: 'เลือกแผนที่เหมาะสม', en: 'Select your tier' }) }}</small>
        </div>
      </div>

      <div class="step-divider" :class="{ done: currentStep > 1 }" />

      <div class="step-item" :class="{ active: currentStep === 2, done: currentStep > 2 }">
        <div class="step-num">
          <v-icon v-if="currentStep > 2" icon="mdi-check" size="16" />
          <span v-else>2</span>
        </div>
        <div class="step-text">
          <strong>{{ l({ lo: 'ເລືອກ Facebook Page', th: 'เลือก Facebook Page', en: 'Connect Page' }) }}</strong>
          <small>{{ l({ lo: 'ເລືອກເພຈທີ່ຕ້ອງການຂາຍ', th: 'เลือกเพจที่ต้องการขาย', en: 'Connect your store page' }) }}</small>
        </div>
      </div>

      <div class="step-divider" :class="{ done: currentStep > 2 }" />

      <div class="step-item" :class="{ active: currentStep === 3 }">
        <div class="step-num">3</div>
        <div class="step-text">
          <strong>{{ l({ lo: 'ພ້ອມເລີ່ມຂາຍ!', th: 'พร้อมเริ่มขาย!', en: 'Ready to Sell' }) }}</strong>
          <small>{{ l({ lo: 'ເຂົ້າສູ່ລະບົບຮ້ານຄ້າ', th: 'เข้าสู่ระบบร้านค้า', en: 'Go to workspace' }) }}</small>
        </div>
      </div>
    </div>

    <!-- Error Banner -->
    <div v-if="errorMessage" class="error-alert">
      <v-icon icon="mdi-alert-circle" size="20" />
      <span>{{ errorMessage }}</span>
    </div>

    <!-- Main Card Body -->
    <main class="wizard-card">
      <!-- Loading State -->
      <div v-if="loading" class="loading-state">
        <v-progress-circular indeterminate color="#0284C7" size="44" />
        <p>{{ l({ lo: 'ກຳລັງກວດສອບສະຖານະຮ້ານຄ້າ...', th: 'กำลังตรวจสอบสถานะร้านค้า...', en: 'Loading store status...' }) }}</p>
      </div>

      <!-- ================= STEP 1: CHOOSE PLAN ================= -->
      <div v-else-if="currentStep === 1" class="step-content">
        <div class="content-heading">
          <h2>{{ l({ lo: 'ຂັ້ນຕອນທີ 1: ເລືອກແພັກເກດສຳລັບຮ້ານຂອງທ່ານ', th: 'ขั้นตอนที่ 1: เลือกแพ็กเกจสำหรับร้านของคุณ', en: 'Step 1: Choose Your Store Plan' }) }}</h2>
          <p>{{ l({ lo: 'ທົດລອງໃຊ້ Pro ຟຣີ 7 ວັນໄດ້ທັນທີ ຫຼື ເລືອກແຜນ Free ຕະຫຼອດຊີບ', th: 'ทดลองใช้ Pro ฟรี 7 วันได้ทันที หรือเลือกแผน Free ตลอดชีพ', en: 'Start with our 7-Day Pro Free Trial or pick the Free Forever plan.' }) }}</p>
        </div>

        <div class="pricing-cards-grid">
          <!-- CARD 1: PRO TRIAL (RECOMMENDED) -->
          <div class="plan-box popular-card">
            <div class="popular-ribbon">
              {{ l({ lo: '⭐ ແນະນຳ - ຍອດນິຍົມທີ່ສຸດ', th: '⭐ แนะนำ - ยอดนิยมที่สุด', en: '⭐ MOST POPULAR' }) }}
            </div>
            <div class="plan-box-head">
              <span class="plan-badge-pro">PRO TRIAL</span>
              <h3>{{ l({ lo: 'Pro 7 ວັນ (ທົດລອງຟຣີ)', th: 'Pro 7 วัน (ทดลองฟรี)', en: '7-Day Pro Free Trial' }) }}</h3>
              <div class="price-display">
                <span class="price-val">0 ₭</span>
                <span class="price-sub">/ 7 {{ l({ lo: 'ວັນທຳອິດ', th: 'วันแรก', en: 'days free' }) }}</span>
              </div>
              <p class="plan-desc">
                {{ l({ lo: 'ທົດລອງໃຊ້ທຸກຟີເຈີແບບບໍ່ມີຂໍ້ຜູກມັດ ບໍ່ຕ້ອງຈ່າຍເງິນກ່ອນ ລັອກລາຄາໂປຣໂມຊັ່ນໄວ້ທັນທີ.', th: 'ทดลองใช้ทุกฟีเจอร์แบบไม่มีข้อผูกมัด ไม่ต้องจ่ายเงินก่อน ล็อกราคาโปรไว้ทันที', en: 'Try all Pro features with zero payment upfront. Lock in the member price.' }) }}
              </p>
            </div>

            <ul class="plan-features">
              <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span><strong>10</strong> {{ l({ lo: 'Facebook Pages ເຊື່ອມຕໍ່', th: 'Facebook Pages เชื่อมต่อ', en: 'Facebook Pages' }) }}</span></li>
              <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span><strong>10</strong> {{ l({ lo: 'ບັນຊີທີມງານຊ່ວຍຕອບ', th: 'บัญชีทีมงานช่วยตอบ', en: 'Team Users' }) }}</span></li>
              <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span>{{ l({ lo: 'AI & Auto-Reply ຕອບແຊັດອັດຕະໂນມັດ', th: 'AI & Auto-Reply ตอบแช็ตอัตโนมัติ', en: 'Auto-Reply & AI Chat' }) }}</span></li>
              <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span>{{ l({ lo: 'ລະບົບອອກບິນ & QR BCEL One', th: 'ระบบออกบิล & QR BCEL One', en: 'BCEL One QR & Waybills' }) }}</span></li>
              <li><v-icon icon="mdi-lock-check" color="#0284C7" size="18" /> <span><strong>{{ l({ lo: 'ລັອກລາຄາເດີມ', th: 'ล็อกราคาเดิม', en: 'Locked Member Price' }) }}</strong> {{ l({ lo: 'ຕໍ່ອາຍຸໃນລາຄາໂປຣ', th: 'ต่ออายุในราคาโปร', en: 'for renewals' }) }}</span></li>
            </ul>

            <button
              type="button"
              class="btn-select-plan btn-glow"
              :disabled="actionLoading"
              @click="selectTrial"
            >
              <v-progress-circular v-if="actionLoading" indeterminate size="20" color="white" />
              <span v-else>{{ l({ lo: 'ເລີ່ມທົດລອງໃຊ້ຟຣີ 7 ວັນ ➔', th: 'เริ่มทดลองใช้ฟรี 7 วัน ➔', en: 'Start 7-Day Free Trial ➔' }) }}</span>
            </button>
          </div>

          <!-- CARD 2: PRO PAID (BCEL ONE) -->
          <div class="plan-box" :class="{ expanded: showProPayment }">
            <div class="plan-box-head">
              <span class="plan-badge-paid">PRO MONTHLY</span>
              <h3>{{ l({ lo: 'Pro ລາຍເດືອນ (ຊຳລະເລີຍ)', th: 'Pro รายเดือน (ชำระเลย)', en: 'Pro Monthly (Pay Now)' }) }}</h3>
              <div class="price-display">
                <span class="price-val">500,000 ₭</span>
                <span class="price-sub">/ {{ l({ lo: 'ເດືອນ', th: 'เดือน', en: 'month' }) }}</span>
              </div>
              <p class="plan-desc">
                {{ l({ lo: 'ສຳລັບຮ້ານທີ່ຕ້ອງການເປີດໃຊ້ງານໄລຍະຍາວ 30 ວັນ ພ້ອມໂອນຜ່ານ BCEL One / OnePay.', th: 'สำหรับร้านที่ต้องการเปิดใช้งานระยะยาว 30 วัน โอนผ่าน BCEL One', en: 'Full 30-day active Pro plan via BCEL One / OnePay QR transfer.' }) }}
              </p>
            </div>

            <div v-if="!showProPayment">
              <ul class="plan-features">
                <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span><strong>10</strong> {{ l({ lo: 'Facebook Pages', th: 'Facebook Pages', en: 'Facebook Pages' }) }}</span></li>
                <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span><strong>10</strong> {{ l({ lo: 'ບັນຊີທີມງານ', th: 'บัญชีทีมงาน', en: 'Team Users' }) }}</span></li>
                <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span>{{ l({ lo: 'ຕໍ່ອາຍຸ +30 ວັນທັນທີ', th: 'ต่ออายุ +30 วันทันที', en: '+30 Days Active' }) }}</span></li>
              </ul>

              <button
                type="button"
                class="btn-select-secondary"
                @click="showProPayment = true"
              >
                {{ l({ lo: 'ສະແກນ QR ຈ່າຍເງິນ ➔', th: 'สแกน QR จ่ายเงิน ➔', en: 'Pay via BCEL One QR ➔' }) }}
              </button>
            </div>

            <!-- Inline BCEL One Slip Payment -->
            <div v-else class="payment-accordion">
              <div class="qr-box">
                <v-icon icon="mdi-qrcode-scan" size="70" color="#0284C7" />
                <strong>500,000 LAK</strong>
                <small>BCEL: 010-12-00-01234567-001</small>
              </div>

              <div class="slip-upload-area">
                <input id="onboard-slip" type="file" accept="image/*" class="sr-only" @change="onSlipSelected" />
                <label for="onboard-slip" class="slip-btn-label">
                  <v-icon :icon="slipUploading ? 'mdi-loading mdi-spin' : 'mdi-cloud-upload'" size="18" />
                  <span v-if="slipUploading">{{ l({ lo: 'ກຳລັງອັບໂຫຼດ...', th: 'กำลังอัปโหลด...', en: 'Uploading...' }) }}</span>
                  <span v-else-if="slipFile">{{ slipFile.name }} ({{ l({ lo: 'ປ່ຽນ', th: 'เปลี่ยน', en: 'Change' }) }})</span>
                  <span v-else>{{ l({ lo: 'ອັບໂຫຼດຮູບສະລິບ (Slip)', th: 'อัปโหลดสลิป (Slip)', en: 'Upload Payment Slip' }) }}</span>
                </label>
              </div>

              <button
                type="button"
                class="btn-select-plan"
                :disabled="actionLoading || !slipReference"
                @click="submitProPayment"
              >
                <v-progress-circular v-if="actionLoading" indeterminate size="20" color="white" />
                <span v-else>{{ l({ lo: 'ຢືນຢັນການຊຳລະເງິນ ➔', th: 'ยืนยันการชำระเงิน ➔', en: 'Submit Payment & Continue ➔' }) }}</span>
              </button>
            </div>
          </div>

          <!-- CARD 3: FREE PLAN -->
          <div class="plan-box free-card">
            <div class="plan-box-head">
              <span class="plan-badge-free">FREE FOREVER</span>
              <h3>{{ l({ lo: 'Free Plan (ຟຣີຕະຫຼອດຊີບ)', th: 'Free Plan (ฟรีตลอดชีพ)', en: 'Free Forever Plan' }) }}</h3>
              <div class="price-display">
                <span class="price-val">0 ₭</span>
                <span class="price-sub">/ {{ l({ lo: 'ຕະຫຼອດໄປ', th: 'ตลอดไป', en: 'forever' }) }}</span>
              </div>
              <p class="plan-desc">
                {{ l({ lo: 'ເໝາະສຳລັບຮ້ານຄ້າເລີ່ມຕົ້ນ 1 ເພຈ ໃຊ້ງານລະບົບແຊັດ ແລະ ຈັດການອໍເດີພື້ນຖານ.', th: 'เหมาะสำหรับร้านค้าเริ่มต้น 1 เพจ ใช้งานระบบแช็ตพื้นฐาน', en: 'For starter shops with 1 Facebook Page.' }) }}
              </p>
            </div>

            <ul class="plan-features">
              <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span><strong>1</strong> {{ l({ lo: 'Facebook Page', th: 'Facebook Page', en: 'Facebook Page' }) }}</span></li>
              <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span><strong>3</strong> {{ l({ lo: 'ບັນຊີທີມງານ', th: 'บัญชีทีมงาน', en: 'Team Users' }) }}</span></li>
              <li><v-icon icon="mdi-check-circle" color="#059669" size="18" /> <span>{{ l({ lo: 'ລະບົບ Live Chat ພື້ນຖານ', th: 'ระบบ Live Chat พื้นฐาน', en: 'Basic Live Chat' }) }}</span></li>
            </ul>

            <button
              type="button"
              class="btn-select-secondary"
              :disabled="actionLoading"
              @click="selectFree"
            >
              {{ l({ lo: 'ເລືອກ Free Plan ➔', th: 'เลือก Free Plan ➔', en: 'Pick Free Plan ➔' }) }}
            </button>
          </div>
        </div>
      </div>

      <!-- ================= STEP 2: SELECT FACEBOOK PAGE ================= -->
      <div v-else-if="currentStep === 2" class="step-content">
        <div class="content-heading">
          <h2>{{ l({ lo: 'ຂັ້ນຕອນທີ 2: ເຊື່ອມຕໍ່ ແລະ ເລືອກ Facebook Page', th: 'ขั้นตอนที่ 2: เชื่อมต่อและเลือก Facebook Page', en: 'Step 2: Connect & Select Facebook Pages' }) }}</h2>
          <p>{{ l({ lo: 'ເລືອກເພຈທີ່ທ່ານຕ້ອງການໃຫ້ Packdee ຊ່ວຍຕອບແຊັດ ແລະ ຈັດການອໍເດີ', th: 'เลือกเพจที่คุณต้องการให้ Packdee ช่วยตอบแช็ตและจัดการออเดอร์', en: 'Select the Facebook Pages you want to manage with Packdee.' }) }}</p>
        </div>

        <!-- Plan Info Pill -->
        <div class="plan-summary-pill">
          <v-icon icon="mdi-shield-check" color="#059669" size="18" />
          <span>{{ l({ lo: 'ແພັກເກດຂອງທ່ານ:', th: 'แพ็กเกจของคุณ:', en: 'Your Plan:' }) }} <strong>{{ onboardingStatus?.plan?.name || 'Pro Plan' }}</strong></span>
          <span class="quota-tag">
            {{ l({ lo: 'ເຊື່ອມຕໍ່ແລ້ວ', th: 'เชื่อมต่อแล้ว', en: 'Connected' }) }}:
            <strong>{{ pages.filter(p => p.connected).length || onboardingStatus?.connectedPagesCount || 0 }}</strong> {{ l({ lo: 'ເພຈ', th: 'เพจ', en: 'pages' }) }}
          </span>
        </div>

        <!-- If No Pages Available: Connect Facebook Button -->
        <div v-if="pages.length === 0" class="connect-fb-box">
          <div class="fb-icon-circle">
            <v-icon icon="mdi-facebook" size="48" color="#1877F2" />
          </div>
          <h3>{{ l({ lo: 'ຍັງບໍ່ທັນໄດ້ເຊື່ອມຕໍ່ບັນຊີ Facebook', th: 'ยังไม่ได้เชื่อมต่อบัญชี Facebook', en: 'Connect Your Facebook Account' }) }}</h3>
          <p>{{ l({ lo: 'ກົດປຸ່ມດ້ານລຸ່ມເພື່ອເຂົ້າສູ່ລະບົບ Facebook ແລະ ອະນຸຍາດໃຫ້ Packdee ດຶງລາຍຊື່ເພຈຂອງທ່ານມາສະແດງ.', th: 'กดปุ่มด้านล่างเพื่อเชื่อมต่อ Facebook และเลือกเพจร้านค้าของคุณ', en: 'Click below to authorize Facebook so we can import your Facebook Pages.' }) }}</p>
          <div class="connect-fb-actions">
            <button
              type="button"
              class="btn-facebook"
              :disabled="actionLoading"
              @click="connectFacebook"
            >
              <v-icon icon="mdi-facebook" size="22" />
              <span>{{ l({ lo: 'ເຊື່ອມຕໍ່ກັບ Facebook (Connect Facebook)', th: 'เชื่อมต่อกับ Facebook (Connect Facebook)', en: 'Connect with Facebook' }) }}</span>
            </button>
            <button
              type="button"
              class="btn-skip-subtle"
              @click="navigateTo('/')"
            >
              <span>{{ l({ lo: 'ຕັ້ງຄ່າພາຍຫຼັງ / ຂ້າມໄປໜ້າ Dashboard ➔', th: 'ตั้งค่าภายหลัง / ข้ามไปหน้า Dashboard ➔', en: 'Setup later / Go to Dashboard ➔' }) }}</span>
            </button>
          </div>
        </div>

        <!-- If Pages Available: List of Pages to Choose -->
        <div v-else class="pages-list-container">
          <div class="pages-list-header">
            <h4>{{ l({ lo: 'ເລືອກເພຈທີ່ຕ້ອງການຈັດການ:', th: 'เลือกเพจที่ต้องการจัดการ:', en: 'Select Pages to Manage:' }) }}</h4>
            <button type="button" class="btn-text-reconnect" @click="connectFacebook">
              <v-icon icon="mdi-refresh" size="16" />
              <span>{{ l({ lo: 'ເຊື່ອມຕໍ່ເພຈເພີ່ມຕື່ມ / Reconnect Facebook', th: 'เชื่อมต่อเพจเพิ่ม / Reconnect Facebook', en: 'Import More Pages' }) }}</span>
            </button>
          </div>

          <div class="pages-grid">
            <div
              v-for="p in pages"
              :key="p.id"
              class="page-select-card"
              :class="{ connected: p.connected }"
            >
              <div class="page-card-avatar">
                <img v-if="p.pictureUrl" :src="p.pictureUrl" class="page-img" alt="Page avatar" />
                <v-icon v-else icon="mdi-facebook" size="32" color="#1877F2" />
              </div>
              <div class="page-card-details">
                <strong>{{ p.name }}</strong>
                <small>{{ p.category || 'Facebook Page' }}</small>
              </div>
              <button
                type="button"
                class="btn-toggle-page"
                :class="p.connected ? 'btn-connected' : 'btn-connect'"
                :disabled="togglingPage === p.id"
                @click="togglePage(p)"
              >
                <v-progress-circular v-if="togglingPage === p.id" indeterminate size="16" color="currentColor" />
                <span v-else-if="p.connected">
                  <v-icon icon="mdi-check" size="16" />
                  {{ l({ lo: 'ເລືອກແລ້ວ', th: 'เลือกแล้ว', en: 'Selected' }) }}
                </span>
                <span v-else>
                  <v-icon icon="mdi-plus" size="16" />
                  {{ l({ lo: 'ເລືອກເພຈນີ້', th: 'เลือกเพจนี้', en: 'Select' }) }}
                </span>
              </button>
            </div>
          </div>

          <!-- Bottom Next Button & Skip Action -->
          <div class="step-actions">
            <button
              type="button"
              class="btn-skip-setup"
              @click="navigateTo('/')"
            >
              <v-icon icon="mdi-arrow-right" size="16" />
              <span>{{ l({ lo: 'ຂ້າມໄປໜ້າຫຼັກກ່ອນ (Skip to Dashboard)', th: 'ข้ามไปหน้าหลักก่อน (Skip to Dashboard)', en: 'Skip to Dashboard' }) }}</span>
            </button>

            <button
              type="button"
              class="btn-next"
              :disabled="!hasConnectedPages"
              @click="currentStep = 3"
            >
              <span>{{ l({ lo: 'ຕໍ່ໄປ: ສຳເລັດການຕັ້ງຄ່າ ➔', th: 'ถัดไป: สำเร็จการตั้งค่า ➔', en: 'Next: Finish Setup ➔' }) }}</span>
            </button>
          </div>
        </div>
      </div>

      <!-- ================= STEP 3: READY & ALL SET ================= -->
      <div v-else-if="currentStep === 3" class="step-content">
        <div class="success-celebration">
          <div class="celebrate-icon">
            <v-icon icon="mdi-check-decagram" size="64" color="#059669" />
          </div>
          <h2>{{ l({ lo: 'ຍິນດີຕ້ອນຮັບສູ່ Packdee! ຮ້ານຄ້າຂອງທ່ານພ້ອມຂາຍແລ້ວ 🎉', th: 'ยินดีต้อนรับสู่ Packdee! ร้านค้าของคุณพร้อมขายแล้ว 🎉', en: 'Welcome to Packdee! Your store is ready to sell 🎉' }) }}</h2>
          <p>{{ l({ lo: 'ການຕັ້ງຄ່າທັງໝົດສຳເລັດສົມບູນ. ຂໍ້ຄວາມໃໝ່ຈາກລູກຄ້າຈະຖືກດຶງເຂົ້າສູ່ລະບົບ Live Chat ໂດຍອັດຕະໂນມັດ.', th: 'การตั้งค่าเสร็จสมบูรณ์ ข้อความจากลูกค้าจะถูกซิงค์เข้าระบบ Live Chat อัตโนมัติ', en: 'All setup steps are complete. New incoming messages will sync to Live Chat automatically.' }) }}</p>
        </div>

        <div class="summary-card">
          <div class="summary-row">
            <div class="summary-item">
              <span class="sum-lbl">{{ l({ lo: 'ແພັກເກດປັດຈຸບັນ', th: 'แพ็กเกจปัจจุบัน', en: 'Active Plan' }) }}</span>
              <strong class="text-blue">{{ onboardingStatus?.plan?.name || 'Pro Plan' }}</strong>
              <small v-if="onboardingStatus?.plan?.status === 'trialing'" class="trial-tag">7-Day Free Trial Active</small>
            </div>

            <div class="summary-item">
              <span class="sum-lbl">{{ l({ lo: 'Facebook Pages ທີ່ເລືອກ', th: 'Facebook Pages ที่เลือก', en: 'Connected Pages' }) }}</span>
              <strong>{{ onboardingStatus?.connectedPagesCount || 1 }} {{ l({ lo: 'ເພຈ', th: 'เพจ', en: 'pages' }) }}</strong>
              <div class="mini-pages-list">
                <span v-for="cp in (onboardingStatus?.connectedPages || [])" :key="cp.id" class="mini-page-chip">
                  📄 {{ cp.name }}
                </span>
              </div>
            </div>

            <div class="summary-item">
              <span class="sum-lbl">{{ l({ lo: 'ສະຖານະລະບົບ', th: 'สถานะระบบ', en: 'System Status' }) }}</span>
              <strong class="text-emerald">🟢 {{ l({ lo: 'ພ້ອມຮັບແຊັດ & ອອກບິນ', th: 'พร้อมรับแช็ต & ออกบิล', en: 'Live & Ready' }) }}</strong>
            </div>
          </div>
        </div>

        <div class="finish-action-wrap">
          <button type="button" class="btn-start-selling" @click="finishOnboarding">
            <v-icon icon="mdi-rocket-launch" size="22" />
            <span>{{ l({ lo: 'ເຂົ້າສູ່ລະບົບຮ້ານຄ້າ / ເລີ່ມຕົ້ນຂາຍສິນຄ້າ (Go to Dashboard)', th: 'เข้าสู่ระบบร้านค้า / เริ่มต้นขายสินค้า', en: 'Start Selling / Go to Workspace' }) }}</span>
          </button>
        </div>
      </div>
    </main>
  </div>
</template>

<style scoped>
.onboarding-container {
  min-height: 100vh;
  background-color: #F8FAFC;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-4) var(--space-3);
  font-family: inherit;
}

.onboarding-header {
  width: 100%;
  max-width: 960px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}

.brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.brand-logo {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: linear-gradient(135deg, #0284C7, #2563EB);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 4px 6px -1px rgba(2, 132, 199, 0.25);
}

.brand-info h2 {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 800;
  color: #0F172A;
}

.sub-brand {
  color: #0284C7;
}

.brand-info small {
  font-size: var(--text-xs);
  color: #64748B;
}

.user-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background-color: #FFFFFF;
  border: 1px solid #E2E8F0;
  padding: 6px 12px;
  border-radius: 9999px;
  font-size: var(--text-xs);
  color: #475569;
}

/* Stepper Bar */
.stepper-bar {
  width: 100%;
  max-width: 960px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background-color: #FFFFFF;
  border: 1px solid #E2E8F0;
  border-radius: var(--radius-lg);
  padding: var(--space-3) var(--space-5);
  margin-bottom: var(--space-4);
  box-shadow: var(--shadow-sm);
}

.step-item {
  display: flex;
  align-items: center;
  gap: 12px;
  opacity: 0.45;
  transition: opacity 0.2s;
}

.step-item.active {
  opacity: 1;
}

.step-item.done {
  opacity: 0.9;
}

.step-num {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background-color: #E2E8F0;
  color: #475569;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 14px;
}

.step-item.active .step-num {
  background-color: #0284C7;
  color: #FFFFFF;
}

.step-item.done .step-num {
  background-color: #059669;
  color: #FFFFFF;
}

.step-text {
  display: flex;
  flex-direction: column;
}

.step-text strong {
  font-size: var(--text-sm);
  color: #0F172A;
}

.step-text small {
  font-size: 11px;
  color: #64748B;
}

.step-divider {
  flex: 1;
  height: 2px;
  background-color: #E2E8F0;
  margin: 0 var(--space-3);
}

.step-divider.done {
  background-color: #059669;
}

/* Error Alert */
.error-alert {
  width: 100%;
  max-width: 960px;
  background-color: #FEE2E2;
  color: #B91C1C;
  padding: 10px 16px;
  border-radius: var(--radius-md);
  margin-bottom: var(--space-3);
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: var(--text-sm);
}

/* Wizard Card */
.wizard-card {
  width: 100%;
  max-width: 960px;
  background-color: #FFFFFF;
  border: 1px solid #E2E8F0;
  border-radius: var(--radius-lg);
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.05);
  padding: var(--space-5);
}

.content-heading {
  text-align: center;
  margin-bottom: var(--space-4);
}

.content-heading h2 {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: 800;
  color: #0F172A;
}

.content-heading p {
  margin: 6px 0 0 0;
  font-size: var(--text-sm);
  color: #64748B;
}

/* Step 1: Pricing Grid */
.pricing-cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: var(--space-3);
  margin-top: var(--space-3);
}

.plan-box {
  border: 1px solid #E2E8F0;
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  position: relative;
  background-color: #FFFFFF;
}

.plan-box.popular-card {
  border: 2px solid #0284C7;
  background: linear-gradient(180deg, #F0F9FF 0%, #FFFFFF 30%);
}

.popular-ribbon {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  background: linear-gradient(90deg, #0284C7, #2563EB);
  color: #FFFFFF;
  text-align: center;
  font-size: 11px;
  font-weight: 800;
  padding: 4px;
  border-top-left-radius: var(--radius-lg);
  border-top-right-radius: var(--radius-lg);
}

.popular-card .plan-box-head {
  margin-top: 14px;
}

.plan-badge-pro {
  background-color: #0284C7;
  color: #FFFFFF;
  font-size: 10px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 9999px;
  display: inline-block;
  margin-bottom: 8px;
}

.plan-badge-paid {
  background-color: #475569;
  color: #FFFFFF;
  font-size: 10px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 9999px;
  display: inline-block;
  margin-bottom: 8px;
}

.plan-badge-free {
  background-color: #64748B;
  color: #FFFFFF;
  font-size: 10px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 9999px;
  display: inline-block;
  margin-bottom: 8px;
}

.plan-box-head h3 {
  margin: 0;
  font-size: var(--text-md);
  font-weight: 800;
}

.price-display {
  display: flex;
  align-items: baseline;
  gap: 4px;
  margin: 8px 0;
}

.price-val {
  font-size: 26px;
  font-weight: 900;
  color: #0F172A;
}

.price-sub {
  font-size: var(--text-xs);
  color: #64748B;
}

.plan-desc {
  font-size: var(--text-xs);
  color: #64748B;
  line-height: 1.5;
  margin-bottom: var(--space-3);
}

.plan-features {
  list-style: none;
  padding: 0;
  margin: 0 0 var(--space-4) 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  font-size: var(--text-xs);
  color: #334155;
  flex: 1;
}

.plan-features li {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-select-plan {
  width: 100%;
  padding: 12px;
  background-color: #0284C7;
  color: #FFFFFF;
  border: 0;
  border-radius: var(--radius-md);
  font-weight: 700;
  font-size: var(--text-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
}

.btn-glow {
  box-shadow: 0 4px 14px rgba(2, 132, 199, 0.35);
}

.btn-glow:hover {
  background-color: #0369A1;
}

.btn-select-secondary {
  width: 100%;
  padding: 10px;
  background-color: #FFFFFF;
  color: #334155;
  border: 1px solid #CBD5E1;
  border-radius: var(--radius-md);
  font-weight: 700;
  font-size: var(--text-sm);
  cursor: pointer;
}

.btn-select-secondary:hover {
  background-color: #F8FAFC;
}

/* Payment Accordion */
.payment-accordion {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.qr-box {
  background-color: #F8FAFC;
  border: 1px dashed #CBD5E1;
  border-radius: var(--radius-md);
  padding: var(--space-2);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.slip-btn-label {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background-color: #EDE9FE;
  color: #6D28D9;
  border: 1px solid #DDD6FE;
  padding: 8px;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 700;
  cursor: pointer;
}

/* Step 2: Page Selection */
.plan-summary-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background-color: #F0FDF4;
  border: 1px solid #BBF7D0;
  padding: 8px 16px;
  border-radius: 9999px;
  font-size: var(--text-xs);
  color: #166534;
  margin-bottom: var(--space-3);
}

.quota-tag {
  background-color: #DCFCE7;
  padding: 2px 8px;
  border-radius: 9999px;
  margin-left: 6px;
}

.connect-fb-box {
  padding: var(--space-6) var(--space-4);
  background-color: #F8FAFC;
  border: 2px dashed #CBD5E1;
  border-radius: var(--radius-lg);
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  gap: 12px;
}

.fb-icon-circle {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background-color: #DBEAFE;
  display: flex;
  align-items: center;
  justify-content: center;
}

.connect-fb-box h3 {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: 800;
}

.connect-fb-box p {
  max-width: 450px;
  font-size: var(--text-xs);
  color: #64748B;
  margin: 0;
}

.btn-facebook {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background-color: #1877F2;
  color: #FFFFFF;
  border: 0;
  padding: 12px 24px;
  border-radius: var(--radius-md);
  font-weight: 700;
  font-size: var(--text-sm);
  cursor: pointer;
  box-shadow: 0 4px 12px rgba(24, 119, 242, 0.3);
}

.pages-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}

.pages-list-header h4 {
  margin: 0;
  font-size: var(--text-md);
  font-weight: 700;
}

.btn-text-reconnect {
  background: transparent;
  border: 0;
  color: #0284C7;
  font-size: var(--text-xs);
  font-weight: 700;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.pages-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.page-select-card {
  border: 1px solid #E2E8F0;
  border-radius: var(--radius-md);
  padding: var(--space-3);
  display: flex;
  align-items: center;
  gap: 12px;
  background-color: #FFFFFF;
}

.page-select-card.connected {
  border: 2px solid #059669;
  background-color: #F0FDF4;
}

.page-card-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  overflow: hidden;
  background-color: #E2E8F0;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.page-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.page-card-details {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.page-card-details strong {
  font-size: var(--text-sm);
  color: #0F172A;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.page-card-details small {
  font-size: 11px;
  color: #64748B;
}

.btn-toggle-page {
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.btn-connected {
  background-color: #D1FAE5;
  color: #065F46;
  border: 1px solid #A7F3D0;
}

.btn-connect {
  background-color: #FFFFFF;
  color: #0284C7;
  border: 1px solid #0284C7;
}

.connect-fb-actions {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
}

.btn-skip-subtle {
  background: transparent;
  color: #64748B;
  border: 0;
  font-size: var(--text-xs);
  cursor: pointer;
  text-decoration: underline;
  padding: 6px 12px;
  transition: color 0.2s ease;
}

.btn-skip-subtle:hover {
  color: #0F172A;
}

.step-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: var(--space-3);
  gap: var(--space-2);
}

.btn-skip-setup {
  background: transparent;
  color: #64748B;
  border: 1px solid #CBD5E1;
  padding: 10px 18px;
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: var(--text-sm);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: all 0.2s ease;
}

.btn-skip-setup:hover {
  background-color: #F8FAFC;
  color: #0F172A;
  border-color: #94A3B8;
}

.btn-next {
  background-color: #0284C7;
  color: #FFFFFF;
  border: 0;
  padding: 12px 24px;
  border-radius: var(--radius-md);
  font-weight: 700;
  font-size: var(--text-sm);
  cursor: pointer;
}

.btn-next:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Step 3: Success */
.success-celebration {
  text-align: center;
  margin-bottom: var(--space-4);
}

.celebrate-icon {
  margin-bottom: var(--space-2);
}

.success-celebration h2 {
  margin: 0;
  font-size: var(--text-xl);
  font-weight: 900;
  color: #0F172A;
}

.success-celebration p {
  max-width: 550px;
  margin: 8px auto 0 auto;
  font-size: var(--text-sm);
  color: #64748B;
  line-height: 1.5;
}

.summary-card {
  background-color: #F8FAFC;
  border: 1px solid #E2E8F0;
  border-radius: var(--radius-md);
  padding: var(--space-4);
  margin-bottom: var(--space-4);
}

.summary-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-3);
}

.summary-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sum-lbl {
  font-size: 11px;
  color: #64748B;
  font-weight: 600;
}

.text-blue { color: #0284C7; }
.text-emerald { color: #059669; }

.trial-tag {
  font-size: 10px;
  font-weight: 700;
  background-color: #DBEAFE;
  color: #1E40AF;
  padding: 1px 6px;
  border-radius: 4px;
  display: inline-block;
  width: fit-content;
}

.mini-pages-list {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}

.mini-page-chip {
  font-size: 11px;
  background-color: #FFFFFF;
  border: 1px solid #CBD5E1;
  padding: 2px 6px;
  border-radius: 4px;
}

.finish-action-wrap {
  display: flex;
  justify-content: center;
}

.btn-start-selling {
  background: linear-gradient(135deg, #0284C7, #2563EB);
  color: #FFFFFF;
  border: 0;
  padding: 14px 32px;
  border-radius: var(--radius-lg);
  font-size: var(--text-md);
  font-weight: 800;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  box-shadow: 0 10px 20px -5px rgba(2, 132, 199, 0.4);
}

.loading-state {
  padding: var(--space-6);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  color: #64748B;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
</style>
