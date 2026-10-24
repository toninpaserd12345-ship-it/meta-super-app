<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

type Plan = { id: string; name: string; price: number; maxUsers: number; maxPages: number; isActive: boolean }
type Subscription = { planId: string; status: string; lockedPrice?: number; currentPeriodEnd: string; plan?: Plan }

const { locale, l } = useLocale()
const plans = ref<Plan[]>([])
const subscription = ref<Subscription | null>(null)
const loading = ref(true)
const loadError = ref('')
const selectedPlan = ref<Plan | null>(null)
const enterpriseExpanded = ref(false)

const checkoutReference = ref('')
const slipFile = ref<File | null>(null)
const slipUploading = ref(false)
const checkoutSubmitting = ref(false)
const checkoutSuccess = ref(false)
const checkoutError = ref('')

const activePlans = computed(() => plans.value.filter(plan => plan.isActive))
const currentPlan = computed(() => subscription.value?.plan || activePlans.value.find(plan => plan.id === subscription.value?.planId))

const payAmount = computed(() => {
  if (!selectedPlan.value) return 0
  if (subscription.value?.planId === selectedPlan.value.id && subscription.value?.lockedPrice && subscription.value.lockedPrice > 0) {
    return subscription.value.lockedPrice
  }
  return selectedPlan.value.price
})

const loadBilling = async () => {
  loading.value = true
  loadError.value = ''
  try {
    const [plansData, subData] = await Promise.all([
      $fetch<Plan[]>('/api/proxy/api/v1/billing/plans'),
      $fetch<Subscription>('/api/proxy/api/v1/billing/subscription')
    ])
    plans.value = plansData
    subscription.value = subData
  } catch (error: any) {
    loadError.value = error?.data?.message || error?.statusMessage || error?.message || l({
      lo: 'ບໍ່ສາມາດໂຫຼດຂໍ້ມູນແພັກເກດໄດ້.',
      th: 'ไม่สามารถโหลดข้อมูลแพ็กเกจได้',
      en: 'Billing details could not be loaded.'
    })
  } finally {
    loading.value = false
  }
}

const formatPrice = (price: number) => price > 0
  ? new Intl.NumberFormat(locale.value === 'lo' ? 'lo-LA' : locale.value === 'th' ? 'th-TH' : 'en-US').format(price)
  : l({ lo: 'ຟຣີ', th: 'ฟรี', en: 'Free' })

const formatDate = (value: string) => {
  if (!value) return l({ lo: 'ບໍ່ມີກຳນົດ', th: 'ไม่มีกำหนด', en: 'No expiry' })
  const date = new Date(value)
  if (Number.isNaN(date.getTime()) || date.getFullYear() > 2100) return l({ lo: 'ບໍ່ມີກຳນົດ', th: 'ไม่มีกำหนด', en: 'No expiry' })
  return date.toLocaleDateString(locale.value === 'lo' ? 'lo-LA' : locale.value === 'th' ? 'th-TH' : 'en-US', {
    year: 'numeric', month: 'long', day: 'numeric'
  })
}

const statusLabel = (status?: string) => {
  if (status === 'active') return l({ lo: 'ໃຊ້ງານຢູ່', th: 'กำลังใช้งาน', en: 'Active' })
  if (status === 'trialing') return l({ lo: 'ທົດລອງໃຊ້ ຟຣີ 7 ມື້', th: 'ทดลองใช้ฟรี 7 วัน', en: '7-Day Free Trial' })
  if (status === 'expired') return l({ lo: 'ໝົດອາຍຸການໃຊ້ງານ', th: 'หมดอายุการใช้งาน', en: 'Expired' })
  if (status === 'past_due') return l({ lo: 'ຕ້ອງກວດສອບການຊຳລະ', th: 'ต้องตรวจสอบการชำระเงิน', en: 'Payment attention' })
  if (status === 'canceled') return l({ lo: 'ຍົກເລີກແລ້ວ', th: 'ยกเลิกแล้ว', en: 'Canceled' })
  return status || l({ lo: 'ບໍ່ຮູ້ສະຖານະ', th: 'ไม่ทราบสถานะ', en: 'Unknown' })
}

const planActionLabel = (plan: Plan) => {
  if (subscription.value?.planId === plan.id && subscription.value?.status === 'active') {
    return l({ lo: 'ແພັກເກດປັດຈຸບັນ', th: 'แพ็กเกจปัจจุบัน', en: 'Current plan' })
  }
  if (subscription.value?.status === 'trialing' && subscription.value?.planId === plan.id) {
    return l({ lo: 'ຕໍ່ອາຍຸ / ຊຳລະເງິນ Pro', th: 'ต่ออายุ / ชำระเงิน Pro', en: 'Activate / Pay Pro' })
  }
  if ((currentPlan.value?.price || 0) > plan.price) {
    return l({ lo: 'ກວດສອບການຫຼຸດແພັກເກດ', th: 'ตรวจสอบการลดแพ็กเกจ', en: 'Review downgrade' })
  }
  return l({ lo: 'ອັບເກຣດແພັກເກດນີ້', th: 'อัปเกรดแพ็กเกจนี้', en: 'Upgrade to this plan' })
}

const reveal = async (selector: string) => {
  await nextTick()
  document.querySelector(selector)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}

const reviewPlan = (plan: Plan) => {
  if (subscription.value?.planId === plan.id && subscription.value?.status === 'active') return
  selectedPlan.value = plan
  enterpriseExpanded.value = false
  checkoutSuccess.value = false
  checkoutError.value = ''
  checkoutReference.value = ''
  reveal('#activation-details')
}

const showEnterprise = () => {
  selectedPlan.value = null
  enterpriseExpanded.value = true
  reveal('#enterprise-details')
}

const onSlipSelected = async (e: Event) => {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return
  const file = target.files[0]
  if (!file) return
  slipFile.value = file
  slipUploading.value = true
  checkoutError.value = ''

  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('folder', 'slips')
    const res = await $fetch<{ url?: string; path?: string }>('/api/proxy/api/v1/storage/upload', {
      method: 'POST',
      body: formData,
    })
    checkoutReference.value = res.url || res.path || file.name
  } catch (err: any) {
    checkoutError.value = err?.data?.message || err?.message || l({
      lo: 'ອັບໂຫຼດຮູບສະລິບບໍ່ສຳເລັດ ກະລຸນາໃສ່ເລກທີໂອນແທນ.',
      th: 'อัปโหลดสลิปไม่สำเร็จ กรุณากรอกเลขอ้างอิงแทน',
      en: 'Slip upload failed. Please type transfer reference instead.'
    })
  } finally {
    slipUploading.value = false
  }
}

const submitPayment = async () => {
  if (!selectedPlan.value) return
  if (!checkoutReference.value.trim()) {
    checkoutError.value = l({
      lo: 'ກະລຸນາອັບໂຫຼດຮູບສະລິບ ຫຼື ປ້ອນເລກທີອ້າງອີງການໂອນເງິນ',
      th: 'กรุณาอัปโหลดสลิปหรือระบุเลขอ้างอิงการโอนเงิน',
      en: 'Please upload transfer slip or enter reference number.'
    })
    return
  }

  checkoutSubmitting.value = true
  checkoutError.value = ''

  try {
    await $fetch('/api/proxy/api/v1/billing/checkout', {
      method: 'POST',
      body: {
        planId: selectedPlan.value.id,
        paymentMethod: 'bcel_one',
        reference: checkoutReference.value.trim(),
      },
    })
    checkoutSuccess.value = true
    await loadBilling()
  } catch (err: any) {
    checkoutError.value = err?.data?.message || err?.message || l({
      lo: 'ສົ່ງຂໍ້ມູນບໍ່ສຳເລັດ ກະລຸນາລອງໃໝ່',
      th: 'ส่งข้อมูลไม่สำเร็จ กรุณาลองใหม่',
      en: 'Submission failed, please try again.'
    })
  } finally {
    checkoutSubmitting.value = false
  }
}

onMounted(loadBilling)
</script>

<template>
  <main class="pricing-page">
    <header class="pricing-header">
      <div>
        <h1>{{ l({ lo: 'ເລືອກແພັກເກດທີ່ເຕີບໂຕໄປກັບຮ້ານຂອງທ່ານ', th: 'เลือกแพ็กเกจที่เติบโตไปพร้อมกับร้านของคุณ', en: 'Choose a plan that grows with your operation' }) }}</h1>
        <p>{{ l({ lo: 'ປຽບທຽບຈຳນວນຜູ້ໃຊ້ ແລະ Facebook Pages ຈາກແພັກເກດຈິງຂອງລະບົບ. ຮອງຮັບການໂອນຜ່ານ BCEL One / OnePay.', th: 'เปรียบเทียบจำนวนผู้ใช้และ Facebook Pages จากแพ็กเกจจริงของระบบ รองรับการโอนผ่าน BCEL One / OnePay', en: 'Compare team and Facebook Page capacity from the plans available. Supports payment via BCEL One / OnePay.' }) }}</p>
      </div>
      <span class="cadence"><v-icon icon="mdi-calendar-month-outline" size="18" />{{ l({ lo: 'ຊຳລະລາຍເດືອນ (30 ມື້)', th: 'ชำระรายเดือน (30 วัน)', en: 'Monthly (30 days)' }) }}</span>
    </header>

    <div v-if="loading" class="plans-loading" aria-live="polite" aria-busy="true">
      <div class="subscription-skeleton" />
      <div class="skeleton-grid"><div v-for="item in 3" :key="item" class="plan-skeleton" /></div>
      <span class="sr-only">{{ l({ lo: 'ກຳລັງໂຫຼດແພັກເກດ', th: 'กำลังโหลดแพ็กเกจ', en: 'Loading plans' }) }}</span>
    </div>

    <section v-else-if="loadError" class="load-error" aria-live="assertive">
      <v-icon icon="mdi-alert-circle-outline" size="26" />
      <div><h2>{{ l({ lo: 'ໂຫຼດແພັກເກດບໍ່ສຳເລັດ', th: 'โหลดแพ็กเกจไม่สำเร็จ', en: 'Plans did not load' }) }}</h2><p>{{ loadError }}</p></div>
      <button type="button" class="btn-secondary" @click="loadBilling"><v-icon icon="mdi-refresh" size="18" />{{ l({ lo: 'ລອງໃໝ່', th: 'ลองใหม่', en: 'Try again' }) }}</button>
    </section>

    <template v-else>
      <!-- Trial alert if trialing -->
      <aside v-if="subscription?.status === 'trialing'" class="trial-banner">
        <v-icon icon="mdi-gift-outline" size="24" color="amber-darken-3" />
        <div class="trial-text">
          <strong>{{ l({ lo: '🎉 ທ່ານກຳລັງຢູ່ໃນຊ່ວງທົດລອງໃຊ້ Pro ຟຣີ 7 ມື້!', th: '🎉 คุณกำลังอยู่ในช่วงทดลองใช้ Pro ฟรี 7 วัน!', en: '🎉 You are enjoying a 7-day Pro Free Trial!' }) }}</strong>
          <span>{{ l({ lo: 'ສາມາດໃຊ້ງານໄດ້ທຸກຟັງຊັນເຕັມຮູບແບບ ຮອດວັນທີ: ', th: 'ใช้งานได้ทุกฟังก์ชันเต็มรูปแบบ ถึงวันที่: ', en: 'Full access until: ' }) }} {{ formatDate(subscription.currentPeriodEnd) }}</span>
        </div>
      </aside>

      <section v-if="subscription" class="subscription-bar" aria-labelledby="current-plan-heading">
        <div class="subscription-main">
          <span class="subscription-mark"><v-icon icon="mdi-shield-check-outline" size="24" /></span>
          <div><p id="current-plan-heading">{{ l({ lo: 'ແພັກເກດປັດຈຸບັນ', th: 'แพ็กเกจปัจจุบัน', en: 'Current plan' }) }}</p><h2>{{ currentPlan?.name || subscription.planId }}</h2></div>
        </div>
        <dl class="subscription-facts">
          <div><dt>{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</dt><dd><span class="status-dot" :class="subscription.status" />{{ statusLabel(subscription.status) }}</dd></div>
          <div><dt>{{ l({ lo: 'ຮອບບໍລິການ', th: 'รอบบริการ', en: 'Service period' }) }}</dt><dd>{{ formatDate(subscription.currentPeriodEnd) }}</dd></div>
        </dl>
      </section>

      <section class="plans-section" aria-labelledby="available-plans-heading">
        <div class="section-heading">
          <div><h2 id="available-plans-heading">{{ l({ lo: 'ແພັກເກດທີ່ມີໃຫ້ໃຊ້', th: 'แพ็กเกจที่พร้อมใช้งาน', en: 'Available plans' }) }}</h2><p>{{ l({ lo: 'ລາຄາ ແລະຂີດຈຳກັດດຶງຈາກລະບົບໂດຍກົງ.', th: 'ราคาและขีดจำกัดดึงจากระบบโดยตรง', en: 'Prices and limits are loaded directly from billing.' }) }}</p></div>
          <span class="plan-count">{{ activePlans.length }} {{ l({ lo: 'ແພັກເກດ', th: 'แพ็กเกจ', en: 'plans' }) }}</span>
        </div>

        <div v-if="activePlans.length" class="plans-grid">
          <article v-for="plan in activePlans" :key="plan.id" class="plan-card" :class="{ current: subscription?.planId === plan.id && subscription?.status === 'active' }">
            <div class="plan-topline">
              <h3>{{ plan.name }}</h3>
              <span v-if="subscription?.planId === plan.id && subscription?.status === 'active'" class="current-chip"><v-icon icon="mdi-check-circle" size="16" />{{ l({ lo: 'ກຳລັງໃຊ້', th: 'กำลังใช้', en: 'In use' }) }}</span>
              <span v-else-if="subscription?.planId === plan.id && subscription?.status === 'trialing'" class="trial-chip"><v-icon icon="mdi-clock-outline" size="16" />{{ l({ lo: 'ທົດລອງ 7 ມື້', th: 'ทดลอง 7 วัน', en: '7-Day Trial' }) }}</span>
            </div>
            <div class="plan-price">
              <strong>{{ formatPrice(plan.price) }}</strong>
              <span v-if="plan.price > 0">LAK / {{ l({ lo: 'ເດືອນ', th: 'เดือน', en: 'month' }) }}</span>
              <span v-else>{{ l({ lo: 'ບໍ່ມີຄ່າບໍລິການລາຍເດືອນ', th: 'ไม่มีค่าบริการรายเดือน', en: 'No monthly charge' }) }}</span>
            </div>
            <div class="capacity-list">
              <div class="capacity-row"><v-icon icon="mdi-account-multiple-outline" size="21" /><div><strong>{{ plan.maxUsers }}</strong><span>{{ l({ lo: 'ຜູ້ໃຊ້ໃນ workspace', th: 'ผู้ใช้ในเวิร์กสเปซ', en: 'workspace users' }) }}</span></div></div>
              <div class="capacity-row"><v-icon icon="mdi-facebook" size="21" /><div><strong>{{ plan.maxPages }}</strong><span>Facebook Pages</span></div></div>
            </div>
            <button
              type="button"
              :class="(subscription?.planId === plan.id && subscription?.status === 'active') ? 'btn-secondary' : 'btn-primary'"
              :disabled="subscription?.planId === plan.id && subscription?.status === 'active'"
              @click="reviewPlan(plan)"
            >
              {{ planActionLabel(plan) }}<v-icon v-if="subscription?.planId !== plan.id || subscription?.status !== 'active'" icon="mdi-arrow-right" size="18" />
            </button>
          </article>

          <article class="enterprise-card">
            <div><h3>Enterprise</h3><p>{{ l({ lo: 'ສຳລັບຫຼາຍສາຂາ, ທີມຂະໜາດໃຫຍ່ ຫຼືຕ້ອງການຂີດຈຳກັດຫຼາຍກວ່າ Pro.', th: 'สำหรับหลายสาขา ทีมขนาดใหญ่ หรือขีดจำกัดที่มากกว่า Pro', en: 'For multi-branch operations, larger teams, or capacity beyond Pro.' }) }}</p></div>
            <div class="enterprise-capacity">
              <span>{{ l({ lo: 'ຜູ້ໃຊ້', th: 'ผู้ใช้', en: 'Users' }) }} <strong>{{ l({ lo: 'ກຳນົດເອງ', th: 'กำหนดเอง', en: 'Custom' }) }}</strong></span>
              <span>Facebook Pages <strong>{{ l({ lo: 'ກຳນົດເອງ', th: 'กำหนดเอง', en: 'Custom' }) }}</strong></span>
            </div>
            <button type="button" class="enterprise-action" :aria-expanded="enterpriseExpanded" aria-controls="enterprise-details" @click="showEnterprise">
              {{ l({ lo: 'ເບິ່ງຂັ້ນຕອນ Enterprise', th: 'ดูขั้นตอน Enterprise', en: 'View Enterprise process' }) }}<v-icon icon="mdi-arrow-right" size="18" />
            </button>
          </article>
        </div>
      </section>

      <!-- BCEL One Payment Panel -->
      <section v-if="selectedPlan" id="activation-details" class="bcel-payment-panel" aria-live="polite">
        <div class="payment-card">
          <div class="payment-header">
            <div class="d-flex align-center gap-2">
              <v-icon icon="mdi-qrcode-scan" color="primary" size="26" />
              <h3>{{ l({ lo: 'ຊຳລະເງິນຜ່ານ BCEL One / OnePay', th: 'ชำระเงินผ่าน BCEL One / OnePay', en: 'Pay via BCEL One / OnePay' }) }}</h3>
            </div>
            <button type="button" class="close-panel" @click="selectedPlan = null"><v-icon icon="mdi-close" size="20" /></button>
          </div>

          <div v-if="checkoutSuccess" class="success-box">
            <v-icon icon="mdi-check-circle" size="48" color="success" />
            <h4>{{ l({ lo: 'ສົ່ງຫຼັກຖານການໂອນເງິນສຳເລັດແລ້ວ!', th: 'ส่งหลักฐานการโอนเงินสำเร็จแล้ว!', en: 'Payment slip submitted successfully!' }) }}</h4>
            <p>{{ l({ lo: 'ເຈົ້າຂອງແອັບກຳລັງກວດສອບສະລິບ ແລະຈະອະນຸມັດເປີດໃຊ້ Pro Plan ໃຫ້ທ່ານພາຍໃນ 1-15 ນາທີ.', th: 'ผู้ดูแลระบบกำลังตรวจสอบสลิปและจะอนุมัติเปิดใช้งาน Pro Plan ภายใน 1-15 นาที', en: 'Admin is reviewing your slip and will activate your Pro Plan within 1-15 minutes.' }) }}</p>
            <button type="button" class="btn-primary" @click="selectedPlan = null; loadBilling()">{{ l({ lo: 'ຮັບຊາບ', th: 'รับทราบ', en: 'Got it' }) }}</button>
          </div>

          <div v-else class="payment-grid">
            <!-- Left: QR Code & Account Details -->
            <div class="qr-column">
              <div class="qr-frame">
                <!-- SVG styled QR code placeholder for BCEL One -->
                <div class="qr-mockup">
                  <div class="qr-logo-tag">BCEL One</div>
                  <v-icon icon="mdi-qrcode" size="140" color="primary" />
                  <div class="qr-scan-hint">{{ l({ lo: 'ສະແກນ QR ດ້ວຍ BCEL One', th: 'สแกน QR ด้วย BCEL One', en: 'Scan QR with BCEL One' }) }}</div>
                </div>
              </div>
              <div class="account-info">
                <div><label>{{ l({ lo: 'ຊື່ບັນຊີ', th: 'ชื่อบัญชี', en: 'Account Name' }) }}:</label> <strong>META SUPER APP (TONXAYIN)</strong></div>
                <div><label>{{ l({ lo: 'ເລກບັນຊີ BCEL', th: 'เลขบัญชี BCEL', en: 'BCEL Account' }) }}:</label> <strong class="copyable">010-12-00-01234567-001</strong></div>
                <div>
                  <label>{{ l({ lo: 'ຈຳນວນເງິນ', th: 'ยอดเงิน', en: 'Amount' }) }}:</label>
                  <strong class="price-highlight">{{ formatPrice(payAmount) }} LAK</strong>
                  <div v-if="payAmount < selectedPlan.price" class="locked-price-note">
                    🔒 {{ l({ lo: 'ລາຄາພິເສດທີ່ລັອກໄວ້ສຳລັບທ່ານ (ປົກກະຕິ: ' + formatPrice(selectedPlan.price) + ' LAK)', th: 'ราคาพิเศษที่ล็อกไว้สำหรับคุณ (ปกติ: ' + formatPrice(selectedPlan.price) + ' LAK)', en: 'Locked member price (regular: ' + formatPrice(selectedPlan.price) + ' LAK)' }) }}
                  </div>
                </div>
              </div>
            </div>

            <!-- Right: Slip Upload & Submission -->
            <div class="upload-column">
              <h4>{{ l({ lo: 'ແນບຫຼັກຖານການໂອນເງິນ', th: 'แนบหลักฐานการโอนเงิน', en: 'Attach Transfer Slip' }) }}</h4>
              <p class="upload-desc">{{ l({ lo: 'ຫຼັງຈາກໂອນເງິນແລ້ວ ກະລຸນາອັບໂຫຼດຮູບສະລິບ ຫຼື ປ້ອນເລກທີອ້າງອີງການໂອນ:', th: 'หลังจากโอนเงินแล้ว กรุณาอัปโหลดรูปสลิปหรือระบุเลขอ้างอิง:', en: 'After transferring, please upload the receipt slip or enter the transfer reference:' }) }}</p>

              <div class="file-dropzone">
                <input id="slip-file" type="file" accept="image/*" @change="onSlipSelected" />
                <label for="slip-file" class="dropzone-label">
                  <v-icon :icon="slipUploading ? 'mdi-loading mdi-spin' : 'mdi-cloud-upload-outline'" size="32" color="primary" />
                  <span v-if="slipUploading">{{ l({ lo: 'ກຳລັງອັບໂຫຼດຮູບ...', th: 'กำลังอัปโหลดรูป...', en: 'Uploading slip...' }) }}</span>
                  <span v-else-if="slipFile">{{ slipFile.name }} ({{ l({ lo: 'ປ່ຽນຮູບ', th: 'เปลี่ยนรูป', en: 'Change' }) }})</span>
                  <span v-else>{{ l({ lo: 'ຄລິກເພື່ອເລືອກຮູບສະລິບ (JPG, PNG)', th: 'คลิกเพื่อเลือกรูปสลิป (JPG, PNG)', en: 'Click to upload slip photo (JPG, PNG)' }) }}</span>
                </label>
              </div>

              <div class="reference-input-box">
                <label>{{ l({ lo: 'ຫຼື ປ້ອນເລກທີອ້າງອີງ / ເບີໂທຜູ້ໂອນ', th: 'หรือระบุเลขอ้างอิง / เบอร์โทรผู้โอน', en: 'Or enter Reference ID / Payer Phone' }) }}</label>
                <input
                  v-model="checkoutReference"
                  type="text"
                  class="text-input"
                  :placeholder="l({ lo: 'ຕົວຢ່າງ: BCEL-894123 ຫຼື 020 5xxxxxxx', th: 'ตัวอย่าง: BCEL-894123 หรือ 020 5xxxxxxx', en: 'e.g. BCEL-894123 or phone number' })"
                />
              </div>

              <div v-if="checkoutError" class="error-msg">
                <v-icon icon="mdi-alert-circle" size="18" /> {{ checkoutError }}
              </div>

              <button
                type="button"
                class="btn-primary submit-btn"
                :disabled="checkoutSubmitting || slipUploading"
                @click="submitPayment"
              >
                <v-icon v-if="checkoutSubmitting" icon="mdi-loading mdi-spin" size="18" />
                <v-icon v-else icon="mdi-check-circle-outline" size="18" />
                {{ l({ lo: 'ຢືນຢັນການໂອນເງິນ / ສົ່ງຫຼັກຖານ', th: 'ยืนยันการโอนเงิน / ส่งหลักฐาน', en: 'Confirm Payment / Submit Slip' }) }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <section v-if="enterpriseExpanded" id="enterprise-details" class="enterprise-details" aria-live="polite">
        <div><h2>{{ l({ lo: 'ການເປີດໃຊ້ Enterprise', th: 'การเปิดใช้ Enterprise', en: 'Enterprise activation' }) }}</h2><p>{{ l({ lo: 'Enterprise ບໍ່ແມ່ນແພັກເກດທີ່ຊື້ໄດ້ຈາກໜ້ານີ້. ຜູ້ດູແລລະບົບຈະກວດຈຳນວນຮ້ານ, ຜູ້ໃຊ້ ແລະ Pages ກ່ອນກຳນົດລາຄາ.', th: 'Enterprise ยังซื้อจากหน้านี้ไม่ได้ ผู้ดูแลระบบจะตรวจจำนวนร้าน ผู้ใช้ และ Pages ก่อนกำหนดราคา', en: 'Enterprise is not purchasable from this page. A system administrator reviews branch, user, and Page capacity before pricing it.' }) }}</p></div>
        <ol>
          <li>{{ l({ lo: 'ກຳນົດຈຳນວນຮ້ານ ແລະສະມາຊິກທີ່ຈະໃຊ້ງານ.', th: 'กำหนดจำนวนร้านและสมาชิกที่จะใช้งาน', en: 'Confirm the number of branches and team members.' }) }}</li>
          <li>{{ l({ lo: 'ລະບຸຈຳນວນ Facebook Pages ທີ່ຕ້ອງເຊື່ອມຕໍ່.', th: 'ระบุจำนวน Facebook Pages ที่ต้องเชื่อมต่อ', en: 'Confirm how many Facebook Pages need to be connected.' }) }}</li>
          <li>{{ l({ lo: 'ໃຫ້ຜູ້ດູແລລະບົບກຳນົດແພັກເກດ ແລະລາຄາ.', th: 'ให้ผู้ดูแลระบบกำหนดแพ็กเกจและราคา', en: 'Have the system administrator configure the plan and price.' }) }}</li>
        </ol>
        <button type="button" class="close-panel" @click="enterpriseExpanded = false"><v-icon icon="mdi-close" size="18" />{{ l({ lo: 'ປິດຂັ້ນຕອນ', th: 'ปิดขั้นตอน', en: 'Close process' }) }}</button>
      </section>
    </template>
  </main>
</template>

<style scoped>
.pricing-page{display:grid;gap:var(--space-4);color:var(--color-text)}
.pricing-header{display:flex;align-items:flex-end;justify-content:space-between;gap:var(--space-3);padding-bottom:var(--space-3);border-bottom:1px solid var(--color-border)}
.pricing-header>div{max-width:72ch}.pricing-header h1{max-width:22ch;margin:0 0 var(--space-1);font-size:clamp(1.75rem,3.4vw,2.75rem);font-weight:780;line-height:1.16;letter-spacing:-.03em;text-wrap:balance}
.pricing-header p,.section-heading p,.enterprise-card p,.enterprise-details p,.empty-plans p,.load-error p{margin:0;color:var(--color-text-secondary);line-height:var(--leading-relaxed)}
.cadence,.plan-count,.current-chip,.trial-chip{display:inline-flex;align-items:center;gap:var(--space-1);flex:0 0 auto;border-radius:999px;font-size:var(--text-sm);font-weight:650}
.cadence{min-height:2.25rem;padding:var(--space-0-5) var(--space-2);color:var(--color-primary-ink);background:var(--color-primary-soft)}
.trial-banner{display:flex;align-items:center;gap:var(--space-2);padding:var(--space-2) var(--space-3);background:#FEF3C7;color:#92400E;border-radius:var(--radius-md);border:1px solid #FCD34D}
.trial-text{display:flex;flex-direction:column;font-size:var(--text-sm)}
.subscription-bar{display:flex;align-items:center;justify-content:space-between;gap:var(--space-4);padding:var(--space-3);color:var(--color-text-on-primary);background:var(--color-primary-deep);border-radius:var(--radius-lg);box-shadow:var(--shadow-sm)}
.subscription-main{display:flex;align-items:center;gap:var(--space-2)}.subscription-mark{display:grid;place-items:center;width:3rem;height:3rem;flex:0 0 3rem;color:var(--color-primary-deep);background:var(--color-brand-caption);border-radius:var(--radius-md)}
.subscription-main p{margin:0 0 var(--space-0-5);color:var(--color-brand-caption);font-size:var(--text-sm);font-weight:600}.subscription-main h2{margin:0;color:inherit;font-size:var(--text-xl);line-height:var(--leading-tight)}
.subscription-facts{display:flex;align-items:center;gap:var(--space-5);margin:0}.subscription-facts div{display:grid;gap:var(--space-0-5)}.subscription-facts dt{color:var(--color-brand-caption);font-size:var(--text-xs)}.subscription-facts dd{display:flex;align-items:center;gap:var(--space-1);margin:0;font-size:var(--text-md);font-weight:650;font-variant-numeric:tabular-nums}
.status-dot{width:.625rem;height:.625rem;border-radius:50%;background:var(--color-text-muted)}.status-dot.active{background:var(--color-success)}.status-dot.trialing{background:#F59E0B}.status-dot.expired{background:var(--color-error)}.status-dot.past_due{background:var(--color-warning)}.status-dot.canceled{background:var(--color-error)}
.plans-section{display:grid;gap:var(--space-3)}.section-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:var(--space-2)}.section-heading h2,.enterprise-details h2,.load-error h2{margin:0 0 var(--space-0-5);font-size:var(--text-xl);line-height:var(--leading-tight)}.plan-count{min-height:2.25rem;padding:var(--space-0-5) var(--space-2);color:var(--color-text-secondary);background:var(--color-surface-soft)}
.plans-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:var(--space-2);align-items:stretch}.plan-card,.enterprise-card{min-width:0;padding:var(--space-3);border-radius:var(--radius-lg)}.plan-card{display:flex;flex-direction:column;gap:var(--space-3);background:var(--color-surface);border:1px solid var(--color-border-strong)}.plan-card.current{border-color:var(--color-primary);box-shadow:var(--shadow-card)}
.plan-topline{display:flex;align-items:center;justify-content:space-between;gap:var(--space-1);min-height:2.25rem}.plan-topline h3,.enterprise-card h3{margin:0;font-size:var(--text-xl);line-height:var(--leading-tight)}
.current-chip{min-height:1.75rem;padding:.125rem var(--space-1);color:var(--color-primary-ink);background:var(--color-primary-soft);white-space:nowrap}
.trial-chip{min-height:1.75rem;padding:.125rem var(--space-1);color:#92400E;background:#FEF3C7;white-space:nowrap}
.plan-price{display:grid;gap:var(--space-0-5)}.plan-price strong{font-size:clamp(1.7rem,3vw,2.35rem);line-height:1;letter-spacing:-.025em;font-variant-numeric:tabular-nums}.plan-price span{color:var(--color-text-muted);font-size:var(--text-sm)}
.capacity-list{display:grid;gap:var(--space-1);margin-top:auto;padding-top:var(--space-1)}.capacity-row{display:flex;align-items:center;gap:var(--space-2);min-height:3.75rem;padding:var(--space-1) 0;border-top:1px solid var(--color-border)}.capacity-row>.v-icon{color:var(--color-primary)}.capacity-row div{display:grid}.capacity-row strong{font-size:var(--text-lg);font-variant-numeric:tabular-nums}.capacity-row span{color:var(--color-text-secondary);font-size:var(--text-sm)}.plan-card button,.enterprise-action{width:100%;min-height:3rem}
.enterprise-card{display:flex;flex-direction:column;gap:var(--space-3);color:var(--color-text-on-primary);background:var(--gradient-brand);box-shadow:var(--shadow-sm)}.enterprise-card p{margin-top:var(--space-1);color:var(--color-brand-caption)}.enterprise-capacity{display:grid;gap:var(--space-1);margin-top:auto;padding:var(--space-2) 0;border-block:1px solid color-mix(in srgb,var(--color-text-on-primary) 22%,transparent)}.enterprise-capacity span{display:flex;justify-content:space-between;gap:var(--space-2);color:var(--color-brand-caption);font-size:var(--text-sm)}.enterprise-capacity strong{color:var(--color-text-on-primary)}
.enterprise-action{display:inline-flex;align-items:center;justify-content:center;gap:var(--space-1);padding:.75rem var(--space-2);color:var(--color-primary-deep);background:var(--color-text-on-primary);border:0;border-radius:var(--radius-sm);font-weight:700;cursor:pointer;transition:transform var(--transition-fast),background var(--transition-fast)}.enterprise-action:hover{background:var(--color-primary-soft)}.enterprise-action:active{transform:scale(.985)}

/* BCEL Payment Panel */
.bcel-payment-panel{margin-top:var(--space-3)}
.payment-card{background:var(--color-surface);border:1px solid var(--color-border-strong);border-radius:var(--radius-lg);padding:var(--space-4);box-shadow:var(--shadow-card)}
.payment-header{display:flex;align-items:center;justify-content:space-between;margin-bottom:var(--space-3);padding-bottom:var(--space-2);border-bottom:1px solid var(--color-border)}
.payment-header h3{margin:0;font-size:var(--text-lg);font-weight:700}
.payment-grid{display:grid;grid-template-columns:1fr 1.2fr;gap:var(--space-5)}
.qr-column{display:flex;flex-direction:column;align-items:center;gap:var(--space-3);padding:var(--space-3);background:var(--color-surface-soft);border-radius:var(--radius-md)}
.qr-frame{background:#fff;padding:var(--space-3);border-radius:var(--radius-md);box-shadow:var(--shadow-sm)}
.qr-mockup{display:flex;flex-direction:column;align-items:center;text-align:center}
.qr-logo-tag{font-size:var(--text-xs);font-weight:800;color:#C2410C;letter-spacing:1px;margin-bottom:var(--space-1)}
.qr-scan-hint{font-size:var(--text-xs);color:var(--color-text-secondary);font-weight:600}
.account-info{display:grid;gap:var(--space-1);width:100%;font-size:var(--text-sm)}
.account-info div{display:flex;justify-content:space-between;padding:var(--space-1) 0;border-bottom:1px dashed var(--color-border)}
.account-info label{color:var(--color-text-secondary)}
.copyable{font-family:monospace;letter-spacing:.5px}
.price-highlight{color:var(--color-primary);font-size:var(--text-md)}
.locked-price-note{font-size:11px;font-weight:700;color:#B45309;background:#FEF3C7;padding:3px 6px;border-radius:4px;margin-top:4px}
.upload-column{display:flex;flex-direction:column;gap:var(--space-3)}
.upload-column h4{margin:0;font-size:var(--text-md);font-weight:700}
.upload-desc{margin:0;font-size:var(--text-sm);color:var(--color-text-secondary)}
.file-dropzone{position:relative}
.file-dropzone input[type="file"]{display:none}
.dropzone-label{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:var(--space-1);padding:var(--space-4);border:2px dashed var(--color-primary);border-radius:var(--radius-md);background:var(--color-primary-softer);cursor:pointer;text-align:center;font-size:var(--text-sm);font-weight:600;color:var(--color-primary)}
.dropzone-label:hover{background:var(--color-primary-soft)}
.reference-input-box{display:grid;gap:var(--space-1)}
.reference-input-box label{font-size:var(--text-xs);font-weight:600;color:var(--color-text-secondary)}
.text-input{width:100%;height:2.75rem;padding:0 var(--space-2);border:1px solid var(--color-border);border-radius:var(--radius-sm);background:var(--color-surface);color:inherit}
.text-input:focus{outline:2px solid var(--color-primary)}
.error-msg{display:flex;align-items:center;gap:var(--space-1);color:var(--color-error);font-size:var(--text-sm)}
.submit-btn{height:3rem;font-weight:700;margin-top:auto}
.success-box{display:flex;flex-direction:column;align-items:center;text-align:center;gap:var(--space-2);padding:var(--space-5)}
.success-box h4{margin:0;font-size:var(--text-xl);font-weight:700;color:var(--color-success)}
.success-box p{max-width:50ch;margin:0;color:var(--color-text-secondary)}
.enterprise-details{display:grid;gap:var(--space-3);padding:var(--space-3);border-radius:var(--radius-lg);grid-template-columns:minmax(0,.85fr) minmax(18rem,1.15fr) auto;align-items:start;background:var(--color-surface);border:1px solid var(--color-border-strong)}
.enterprise-details ol{display:grid;gap:var(--space-1);margin:0;padding-left:1.5rem;color:var(--color-text-secondary);line-height:var(--leading-relaxed)}
.close-panel{display:inline-flex;align-items:center;justify-content:center;min-height:var(--tap-target);padding:var(--space-1);color:var(--color-text-secondary);background:transparent;border:0;border-radius:var(--radius-sm);cursor:pointer}
.close-panel:hover{color:var(--color-text);background:var(--color-surface-soft)}
.load-error,.empty-plans{display:grid;justify-items:start;gap:var(--space-2);padding:var(--space-4);background:var(--color-error-soft);border:1px solid var(--color-error-border);border-radius:var(--radius-lg)}.load-error{grid-template-columns:auto minmax(0,1fr) auto;align-items:center}.load-error>.v-icon{color:var(--color-error)}.empty-plans{justify-items:center;text-align:center;background:var(--color-surface-soft);border-color:var(--color-border)}
.plans-loading{display:grid;gap:var(--space-3)}.subscription-skeleton,.plan-skeleton{background:linear-gradient(90deg,var(--color-surface-soft),var(--color-surface),var(--color-surface-soft));background-size:220% 100%;animation:pricing-shimmer 1.4s ease-in-out infinite;border-radius:var(--radius-lg)}.subscription-skeleton{height:7rem}.skeleton-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:var(--space-2)}.plan-skeleton{height:24rem}@keyframes pricing-shimmer{to{background-position:-220% 0}}
.sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
@media(max-width:1100px){.plans-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.enterprise-card{grid-column:1/-1}.payment-grid{grid-template-columns:1fr}}
@media(max-width:760px){.pricing-page{gap:var(--space-3)}.pricing-header,.section-heading,.subscription-bar{align-items:flex-start;flex-direction:column}.plans-grid{grid-template-columns:1fr}}
</style>
