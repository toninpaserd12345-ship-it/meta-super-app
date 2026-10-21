<script setup lang="ts">
definePageMeta({ middleware: 'auth' })

type Plan = { id: string; name: string; price: number; maxUsers: number; maxPages: number; isActive: boolean }
type Subscription = { planId: string; status: string; currentPeriodEnd: string; plan?: Plan }

const { locale, l } = useLocale()
const plans = ref<Plan[]>([])
const subscription = ref<Subscription | null>(null)
const loading = ref(true)
const loadError = ref('')
const selectedPlan = ref<Plan | null>(null)
const enterpriseExpanded = ref(false)

const activePlans = computed(() => plans.value.filter(plan => plan.isActive))
const currentPlan = computed(() => subscription.value?.plan || activePlans.value.find(plan => plan.id === subscription.value?.planId))

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
 if (status === 'past_due') return l({ lo: 'ຕ້ອງກວດສອບການຊຳລະ', th: 'ต้องตรวจสอบการชำระเงิน', en: 'Payment attention' })
 if (status === 'canceled') return l({ lo: 'ຍົກເລີກແລ້ວ', th: 'ยกเลิกแล้ว', en: 'Canceled' })
 return status || l({ lo: 'ບໍ່ຮູ້ສະຖານະ', th: 'ไม่ทราบสถานะ', en: 'Unknown' })
}

const planActionLabel = (plan: Plan) => {
 if (subscription.value?.planId === plan.id) return l({ lo: 'ແພັກເກດປັດຈຸບັນ', th: 'แพ็กเกจปัจจุบัน', en: 'Current plan' })
 if ((currentPlan.value?.price || 0) > plan.price) return l({ lo: 'ກວດສອບການຫຼຸດແພັກເກດ', th: 'ตรวจสอบการลดแพ็กเกจ', en: 'Review downgrade' })
 return l({ lo: 'ກວດສອບການອັບເກຣດ', th: 'ตรวจสอบการอัปเกรด', en: 'Review upgrade' })
}

const reveal = async (selector: string) => {
 await nextTick()
 document.querySelector(selector)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
}

const reviewPlan = (plan: Plan) => {
 if (subscription.value?.planId === plan.id) return
  selectedPlan.value = plan
 enterpriseExpanded.value = false
 reveal('#activation-details')
}

const showEnterprise = () => {
 selectedPlan.value = null
 enterpriseExpanded.value = true
 reveal('#enterprise-details')
}

onMounted(loadBilling)
</script>

<template>
 <main class="pricing-page">
 <header class="pricing-header">
      <div>
 <h1>{{ l({ lo: 'ເລືອກແພັກເກດທີ່ເຕີບໂຕໄປກັບຮ້ານຂອງທ່ານ', th: 'เลือกแพ็กเกจที่เติบโตไปพร้อมกับร้านของคุณ', en: 'Choose a plan that grows with your operation' }) }}</h1>
 <p>{{ l({ lo: 'ປຽບທຽບຈຳນວນຜູ້ໃຊ້ ແລະ Facebook Pages ຈາກແພັກເກດຈິງຂອງລະບົບ. ຄ່າບໍລິການແມ່ນລາຍເດືອນ.', th: 'เปรียบเทียบจำนวนผู้ใช้และ Facebook Pages จากแพ็กเกจจริงของระบบ ค่าบริการเรียกเก็บรายเดือน', en: 'Compare team and Facebook Page capacity from the plans available in this workspace. Paid plans are billed monthly.' }) }}</p>
      </div>
 <span class="cadence"><v-icon icon="mdi-calendar-month-outline" size="18" />{{ l({ lo: 'ຊຳລະລາຍເດືອນ', th: 'ชำระรายเดือน', en: 'Monthly billing' }) }}</span>
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
 <article v-for="plan in activePlans" :key="plan.id" class="plan-card" :class="{ current: subscription?.planId === plan.id }">
 <div class="plan-topline">
 <h3>{{ plan.name }}</h3>
 <span v-if="subscription?.planId === plan.id" class="current-chip"><v-icon icon="mdi-check-circle" size="16" />{{ l({ lo: 'ກຳລັງໃຊ້', th: 'กำลังใช้', en: 'In use' }) }}</span>
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
 <button type="button" :class="subscription?.planId === plan.id ? 'btn-secondary' : 'btn-primary'" :disabled="subscription?.planId === plan.id" @click="reviewPlan(plan)">
 {{ planActionLabel(plan) }}<v-icon v-if="subscription?.planId !== plan.id" icon="mdi-arrow-right" size="18" />
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

 <div v-else class="empty-plans">
 <v-icon icon="mdi-package-variant-closed" size="28" /><h3>{{ l({ lo: 'ຍັງບໍ່ມີແພັກເກດເປີດໃຊ້', th: 'ยังไม่มีแพ็กเกจที่เปิดใช้งาน', en: 'No active plans are available' }) }}</h3><p>{{ l({ lo: 'ລອງໂຫຼດໃໝ່ ຫຼືຕິດຕໍ່ຜູ້ດູແລລະບົບ.', th: 'ลองโหลดใหม่หรือติดต่อผู้ดูแลระบบ', en: 'Try again or contact the system administrator.' }) }}</p><button type="button" class="btn-secondary" @click="loadBilling">{{ l({ lo: 'ໂຫຼດໃໝ່', th: 'โหลดใหม่', en: 'Reload' }) }}</button>
        </div>
 </section>

 <section v-if="selectedPlan" id="activation-details" class="activation-panel" aria-live="polite">
 <div class="activation-copy"><span class="activation-icon"><v-icon icon="mdi-file-document-check-outline" size="23" /></span><div><h2>{{ l({ lo: 'ກວດສອບການປ່ຽນເປັນ', th: 'ตรวจสอบการเปลี่ยนเป็น', en: 'Review change to' }) }} {{ selectedPlan.name }}</h2><p>{{ l({ lo: 'ການເປີດແພັກເກດນີ້ຍັງດຳເນີນການແບບ manual. ໜ້ານີ້ບໍ່ຕັດເງິນ ແລະບໍ່ສ້າງການຊຳລະ.', th: 'แพ็กเกจนี้ยังเปิดใช้งานแบบ manual หน้านี้จะไม่ตัดเงินและไม่สร้างรายการชำระเงิน', en: 'This plan is activated manually. This page will not charge you or create a payment.' }) }}</p></div></div>
 <dl class="activation-summary">
 <div><dt>{{ l({ lo: 'ລາຄາ', th: 'ราคา', en: 'Price' }) }}</dt><dd>{{ formatPrice(selectedPlan.price) }} <span v-if="selectedPlan.price > 0">LAK / {{ l({ lo: 'ເດືອນ', th: 'เดือน', en: 'month' }) }}</span></dd></div>
 <div><dt>{{ l({ lo: 'ຂີດຈຳກັດ', th: 'ขีดจำกัด', en: 'Capacity' }) }}</dt><dd>{{ selectedPlan.maxUsers }} {{ l({ lo: 'ຜູ້ໃຊ້', th: 'ผู้ใช้', en: 'users' }) }} · {{ selectedPlan.maxPages }} Pages</dd></div>
 </dl>
 <button type="button" class="close-panel" @click="selectedPlan = null"><v-icon icon="mdi-close" size="18" />{{ l({ lo: 'ປິດລາຍລະອຽດ', th: 'ปิดรายละเอียด', en: 'Close details' }) }}</button>
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
.pricing-header p,.section-heading p,.enterprise-card p,.activation-copy p,.enterprise-details p,.empty-plans p,.load-error p{margin:0;color:var(--color-text-secondary);line-height:var(--leading-relaxed)}
.cadence,.plan-count,.current-chip{display:inline-flex;align-items:center;gap:var(--space-1);flex:0 0 auto;border-radius:999px;font-size:var(--text-sm);font-weight:650}
.cadence{min-height:2.25rem;padding:var(--space-0-5) var(--space-2);color:var(--color-primary-ink);background:var(--color-primary-soft)}
.subscription-bar{display:flex;align-items:center;justify-content:space-between;gap:var(--space-4);padding:var(--space-3);color:var(--color-text-on-primary);background:var(--color-primary-deep);border-radius:var(--radius-lg);box-shadow:var(--shadow-sm)}
.subscription-main{display:flex;align-items:center;gap:var(--space-2)}.subscription-mark{display:grid;place-items:center;width:3rem;height:3rem;flex:0 0 3rem;color:var(--color-primary-deep);background:var(--color-brand-caption);border-radius:var(--radius-md)}
.subscription-main p{margin:0 0 var(--space-0-5);color:var(--color-brand-caption);font-size:var(--text-sm);font-weight:600}.subscription-main h2{margin:0;color:inherit;font-size:var(--text-xl);line-height:var(--leading-tight)}
.subscription-facts{display:flex;align-items:center;gap:var(--space-5);margin:0}.subscription-facts div{display:grid;gap:var(--space-0-5)}.subscription-facts dt{color:var(--color-brand-caption);font-size:var(--text-xs)}.subscription-facts dd{display:flex;align-items:center;gap:var(--space-1);margin:0;font-size:var(--text-md);font-weight:650;font-variant-numeric:tabular-nums}
.status-dot{width:.625rem;height:.625rem;border-radius:50%;background:var(--color-text-muted)}.status-dot.active{background:var(--color-success)}.status-dot.past_due{background:var(--color-warning)}.status-dot.canceled{background:var(--color-error)}
.plans-section{display:grid;gap:var(--space-3)}.section-heading{display:flex;align-items:flex-end;justify-content:space-between;gap:var(--space-2)}.section-heading h2,.activation-panel h2,.enterprise-details h2,.load-error h2{margin:0 0 var(--space-0-5);font-size:var(--text-xl);line-height:var(--leading-tight)}.plan-count{min-height:2.25rem;padding:var(--space-0-5) var(--space-2);color:var(--color-text-secondary);background:var(--color-surface-soft)}
.plans-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:var(--space-2);align-items:stretch}.plan-card,.enterprise-card{min-width:0;padding:var(--space-3);border-radius:var(--radius-lg)}.plan-card{display:flex;flex-direction:column;gap:var(--space-3);background:var(--color-surface);border:1px solid var(--color-border-strong)}.plan-card.current{border-color:var(--color-primary);box-shadow:var(--shadow-card)}
.plan-topline{display:flex;align-items:center;justify-content:space-between;gap:var(--space-1);min-height:2.25rem}.plan-topline h3,.enterprise-card h3{margin:0;font-size:var(--text-xl);line-height:var(--leading-tight)}.current-chip{min-height:1.75rem;padding:.125rem var(--space-1);color:var(--color-primary-ink);background:var(--color-primary-soft);white-space:nowrap}
.plan-price{display:grid;gap:var(--space-0-5)}.plan-price strong{font-size:clamp(1.7rem,3vw,2.35rem);line-height:1;letter-spacing:-.025em;font-variant-numeric:tabular-nums}.plan-price span{color:var(--color-text-muted);font-size:var(--text-sm)}
.capacity-list{display:grid;gap:var(--space-1);margin-top:auto;padding-top:var(--space-1)}.capacity-row{display:flex;align-items:center;gap:var(--space-2);min-height:3.75rem;padding:var(--space-1) 0;border-top:1px solid var(--color-border)}.capacity-row>.v-icon{color:var(--color-primary)}.capacity-row div{display:grid}.capacity-row strong{font-size:var(--text-lg);font-variant-numeric:tabular-nums}.capacity-row span{color:var(--color-text-secondary);font-size:var(--text-sm)}.plan-card button,.enterprise-action{width:100%;min-height:3rem}
.enterprise-card{display:flex;flex-direction:column;gap:var(--space-3);color:var(--color-text-on-primary);background:var(--gradient-brand);box-shadow:var(--shadow-sm)}.enterprise-card p{margin-top:var(--space-1);color:var(--color-brand-caption)}.enterprise-capacity{display:grid;gap:var(--space-1);margin-top:auto;padding:var(--space-2) 0;border-block:1px solid color-mix(in srgb,var(--color-text-on-primary) 22%,transparent)}.enterprise-capacity span{display:flex;justify-content:space-between;gap:var(--space-2);color:var(--color-brand-caption);font-size:var(--text-sm)}.enterprise-capacity strong{color:var(--color-text-on-primary)}
.enterprise-action{display:inline-flex;align-items:center;justify-content:center;gap:var(--space-1);padding:.75rem var(--space-2);color:var(--color-primary-deep);background:var(--color-text-on-primary);border:0;border-radius:var(--radius-sm);font-weight:700;cursor:pointer;transition:transform var(--transition-fast),background var(--transition-fast)}.enterprise-action:hover{background:var(--color-primary-soft)}.enterprise-action:active{transform:scale(.985)}
.activation-panel,.enterprise-details{display:grid;gap:var(--space-3);padding:var(--space-3);border-radius:var(--radius-lg)}.activation-panel{grid-template-columns:minmax(0,1.4fr) minmax(15rem,.8fr) auto;align-items:center;background:var(--color-primary-softer);border:1px solid color-mix(in srgb,var(--color-primary) 24%,transparent)}.activation-copy{display:flex;gap:var(--space-2);align-items:flex-start}.activation-icon{display:grid;place-items:center;width:2.75rem;height:2.75rem;flex:0 0 2.75rem;color:var(--color-primary);background:var(--color-surface);border-radius:var(--radius-md)}
.activation-summary{display:grid;gap:var(--space-1);margin:0}.activation-summary div{display:flex;justify-content:space-between;gap:var(--space-2)}.activation-summary dt{color:var(--color-text-muted);font-size:var(--text-sm)}.activation-summary dd{margin:0;text-align:right;font-size:var(--text-sm);font-weight:650;font-variant-numeric:tabular-nums}.activation-summary dd span{font-weight:500}
.close-panel{display:inline-flex;align-items:center;justify-content:center;gap:var(--space-1);min-height:var(--tap-target);padding:var(--space-1) var(--space-2);color:var(--color-text-secondary);background:transparent;border:0;border-radius:var(--radius-sm);font-weight:650;cursor:pointer}.close-panel:hover{color:var(--color-text);background:var(--color-surface-soft)}
.enterprise-details{grid-template-columns:minmax(0,.85fr) minmax(18rem,1.15fr) auto;align-items:start;background:var(--color-surface);border:1px solid var(--color-border-strong)}.enterprise-details ol{display:grid;gap:var(--space-1);margin:0;padding-left:1.5rem;color:var(--color-text-secondary);line-height:var(--leading-relaxed)}
.load-error,.empty-plans{display:grid;justify-items:start;gap:var(--space-2);padding:var(--space-4);background:var(--color-error-soft);border:1px solid var(--color-error-border);border-radius:var(--radius-lg)}.load-error{grid-template-columns:auto minmax(0,1fr) auto;align-items:center}.load-error>.v-icon{color:var(--color-error)}.empty-plans{justify-items:center;text-align:center;background:var(--color-surface-soft);border-color:var(--color-border)}
.plans-loading{display:grid;gap:var(--space-3)}.subscription-skeleton,.plan-skeleton{background:linear-gradient(90deg,var(--color-surface-soft),var(--color-surface),var(--color-surface-soft));background-size:220% 100%;animation:pricing-shimmer 1.4s ease-in-out infinite;border-radius:var(--radius-lg)}.subscription-skeleton{height:7rem}.skeleton-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:var(--space-2)}.plan-skeleton{height:24rem}@keyframes pricing-shimmer{to{background-position:-220% 0}}
.sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}
@media(max-width:1100px){.plans-grid{grid-template-columns:repeat(2,minmax(0,1fr))}.enterprise-card{grid-column:1/-1}.enterprise-details,.activation-panel{grid-template-columns:minmax(0,1fr) minmax(15rem,.8fr)}.close-panel{grid-column:1/-1;justify-self:end}}
@media(max-width:760px){.pricing-page{gap:var(--space-3)}.pricing-header,.section-heading,.subscription-bar{align-items:flex-start;flex-direction:column}.cadence,.plan-count{align-self:flex-start}.subscription-facts{width:100%;display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:var(--space-2);padding-top:var(--space-2);border-top:1px solid color-mix(in srgb,var(--color-text-on-primary) 20%,transparent)}.plans-grid,.skeleton-grid,.enterprise-details,.activation-panel,.load-error{grid-template-columns:1fr}.enterprise-card{grid-column:auto}.load-error button{width:100%}.close-panel{grid-column:auto;justify-self:stretch}}
@media(max-width:430px){.pricing-header h1{font-size:1.75rem}.subscription-bar,.plan-card,.enterprise-card,.activation-panel,.enterprise-details{padding:var(--space-2)}.subscription-facts{grid-template-columns:1fr}.plan-topline{align-items:flex-start;flex-direction:column}.current-chip{align-self:flex-start}}
</style>
