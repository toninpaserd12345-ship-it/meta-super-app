<script setup lang="ts">
import type { AutomationFlow, ItemsResponse, MetaAdAccount, MetaAudienceAnalysis, MetaCampaign, MetaInsightItem, MetaInsightsReport, Product } from '~/types/automation'
import { useAdVat } from '~/composables/useAdVat'

definePageMeta({ middleware: 'auth' })
const { l } = useLocale()

type Level = 'campaign' | 'adset' | 'ad'
type DecisionCode = 'keep' | 'review' | 'pause' | 'no_data'
type StatusFilter = 'all' | 'active' | 'paused' | 'issues'

const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const report = ref<MetaInsightsReport | null>(null)
const adAccountId = useCookie('meta_ad_account_id', { default: () => '', maxAge: 2592000 })
const level = ref<Level>('campaign')
const loadingAccounts = ref(true)
const loadingReport = ref(false)
const updatingId = ref('')
const notice = ref('')
const success = ref('')
const search = ref('')
const statusFilter = ref<StatusFilter>('all')
const selectedId = ref('')
const selectedProductId = ref('all')
const products = ref<Product[]>([])
const flows = ref<AutomationFlow[]>([])
const loadingAutomations = ref(false)
const settingsOpen = ref(false)
const actionPlanOpen = ref(false)
const actionPlanAcknowledged = ref(false)
const selectedActionIds = ref<string[]>([])
const applyingActionPlan = ref(false)
const ready = ref(false)
const campaignsAccountId = ref('')
const audienceByAdSet = reactive<Record<string, MetaAudienceAnalysis>>({})
const audienceLoading = reactive<Record<string, boolean>>({})
const audienceErrors = reactive<Record<string, string>>({})

// Scale / Budget Control Modal
const budgetModalOpen = ref(false)
const targetBudgetItem = ref<MetaInsightItem | null>(null)
const baseDailyBudget = ref<number>(0)
const newDailyBudget = ref<number>(0)
const selectedPercent = ref<number | null>(20)
const updatingBudget = ref(false)

const zeroDecimalCurrencies = new Set(['LAK', 'VND', 'JPY', 'KRW', 'KHR', 'IDR', 'CLP', 'PYG'])
const currencyOffset = computed(() => zeroDecimalCurrencies.has((currency.value || '').toUpperCase()) ? 1 : 100)

const iso = (date: Date) => {
  const copy = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return copy.toISOString().slice(0, 10)
}
const today = new Date()
const since = ref(iso(new Date(today.getFullYear(), today.getMonth(), today.getDate() - 6)))
const until = ref(iso(today))
const datePreset = ref('7')

const targetCost = ref<number | null>(null)
const goodCTR = ref(3)
const weakCTR = ref(1)
const maxFrequency = ref(3)
const grossMargin = ref<number | null>(null)
const minimumPlanImpressions = ref(1000)
const minimumPlanSpend = ref<number | null>(null)
const sortBy = ref<'spend' | 'newest' | 'name'>('spend')

const selectedAccount = computed(() => adAccounts.value.find(item => item.id === adAccountId.value))
const { detectVatRate, computeVat } = useAdVat()
const vatDetails = computed(() => detectVatRate(selectedAccount.value))
const vatComputation = computed(() => computeVat(summary.value?.spend, vatDetails.value.rate))

async function loadAutomationsAndProducts() {
  loadingAutomations.value = true
  try {
    const [prodRes, flowRes] = await Promise.all([
      $fetch<ItemsResponse<Product>>('/api/proxy/api/v1/products').catch(() => ({ items: [] })),
      $fetch<ItemsResponse<AutomationFlow>>('/api/proxy/api/v1/automations').catch(() => ({ items: [] })),
    ])
    products.value = prodRes.items || []
    flows.value = flowRes.items || []
  } catch {}
  finally {
    loadingAutomations.value = false
  }
}

const triggerToProduct = computed(() => {
  const map: Record<string, { product: Product; flowName: string }> = {}
  const productMap = new Map<string, Product>(products.value.map(p => [p.id, p]))

  for (const flow of flows.value) {
    const prod = productMap.get(flow.productId)
    if (!prod) continue
    for (const target of flow.targets || []) {
      if (target.value) {
        map[target.value] = { product: prod, flowName: flow.name }
      }
    }
  }
  return map
})

function getLinkedProduct(item: MetaInsightItem) {
  if (triggerToProduct.value[item.entityId]) return triggerToProduct.value[item.entityId]
  if (item.adSetId && triggerToProduct.value[item.adSetId]) return triggerToProduct.value[item.adSetId]
  if (item.campaignId && triggerToProduct.value[item.campaignId]) return triggerToProduct.value[item.campaignId]
  return undefined
}

const productOptions = computed(() => {
  const allList = report.value?.items || []
  const countsByProduct: Record<string, { count: number; name: string; img?: string }> = {}
  let unlinkedCount = 0

  for (const it of allList) {
    const linked = getLinkedProduct(it)
    if (linked) {
      const existing = countsByProduct[linked.product.id]
      if (!existing) {
        countsByProduct[linked.product.id] = {
          count: 1,
          name: linked.product.name,
          img: linked.product.imageUrl,
        }
      } else {
        existing.count++
      }
    } else {
      unlinkedCount++
    }
  }

  const list = [
    { title: l({ lo: `ທຸກສິນຄ້າ (${allList.length})`, th: `สินค้าทั้งหมด (${allList.length})`, en: `All Products (${allList.length})` }), value: 'all' },
  ]

  for (const [pId, pData] of Object.entries(countsByProduct)) {
    list.push({
      title: `📦 ${pData.name} (${pData.count} ແອດ)`,
      value: pId,
    })
  }

  if (unlinkedCount > 0) {
    list.push({
      title: l({ lo: `⚠️ ແອດທີ່ຍັງບໍ່ຜູກສິນຄ້າ (${unlinkedCount})`, th: `⚠️ แอดที่ยังไม่ผูกสินค้า (${unlinkedCount})`, en: `⚠️ Unlinked Ads (${unlinkedCount})` }),
      value: 'unlinked',
    })
  }

  return list
})

const items = computed(() => {
  const query = search.value.trim().toLowerCase()
  const list = report.value?.items || []
  return list
    .filter(item => !query || [item.entityName, item.campaignName, item.adSetName, item.adName, item.entityId].some(value => value?.toLowerCase().includes(query)))
    .filter((item) => {
      const status = (item.effectiveStatus || item.status || '').toUpperCase()
      if (statusFilter.value === 'active') return status === 'ACTIVE' || status === 'IN_PROCESS' || status === 'PENDING_REVIEW'
      if (statusFilter.value === 'paused') return status === 'PAUSED' || status === 'CAMPAIGN_PAUSED' || status === 'ADSET_PAUSED'
      if (statusFilter.value === 'issues') return !!status && status !== 'ACTIVE' && status !== 'IN_PROCESS' && status !== 'PENDING_REVIEW' && status !== 'PAUSED' && status !== 'CAMPAIGN_PAUSED' && status !== 'ADSET_PAUSED'
      return true
    })
    .filter((item) => {
      if (selectedProductId.value === 'all') return true
      const linked = getLinkedProduct(item)
      if (selectedProductId.value === 'unlinked') return !linked
      return linked?.product.id === selectedProductId.value
    })
    .sort((a, b) => {
      if (sortBy.value === 'newest') {
        const timeA = a.createdTime || ''
        const timeB = b.createdTime || ''
        if (timeA || timeB) return timeB.localeCompare(timeA)
        return (b.spend ?? -1) - (a.spend ?? -1)
      }
      if (sortBy.value === 'name') {
        return (a.entityName || '').localeCompare(b.entityName || '')
      }
      // Default: Highest spend first; if spends are equal (like newly created items), sort newest first
      const spendA = a.spend ?? -1
      const spendB = b.spend ?? -1
      if (spendA !== spendB) return spendB - spendA
      const timeA = a.createdTime || ''
      const timeB = b.createdTime || ''
      if (timeA || timeB) return timeB.localeCompare(timeA)
      return (a.entityName || '').localeCompare(b.entityName || '')
    })
})
const summary = computed(() => report.value?.summary)
const currency = computed(() => report.value?.currency || 'USD')
const maxTrendSpend = computed(() => Math.max(1, ...(report.value?.trend || []).map(point => point.spend || 0)))
const statusCounts = computed(() => (report.value?.items || []).reduce((counts, item) => {
  const status = (item.effectiveStatus || item.status || '').toUpperCase()
  if (status === 'ACTIVE' || status === 'IN_PROCESS' || status === 'PENDING_REVIEW') counts.active += 1
  else if (status === 'PAUSED' || status === 'CAMPAIGN_PAUSED' || status === 'ADSET_PAUSED') counts.paused += 1
  else counts.issues += 1
  return counts
}, { active: 0, paused: 0, issues: 0 } as Record<'active' | 'paused' | 'issues', number>))

const fmt = (value?: number, digits = 2) => value == null ? '—' : new Intl.NumberFormat(undefined, { maximumFractionDigits: digits }).format(value)
const money = (value?: number) => value == null ? '—' : new Intl.NumberFormat(undefined, { style: 'currency', currency: currency.value, maximumFractionDigits: 2 }).format(value)
const percent = (value?: number) => value == null ? '—' : `${fmt(value, 2)}%`
const contribution = (item?: MetaInsightItem) => item?.conversionValue != null && item?.spend != null && grossMargin.value != null
  ? item.conversionValue * (grossMargin.value / 100) - item.spend
  : undefined
const breakEvenROAS = computed(() => grossMargin.value && grossMargin.value > 0 ? 100 / grossMargin.value : undefined)
const adSetIDFor = (item: MetaInsightItem) => item.entityType === 'adset' ? item.entityId : item.adSetId || ''
const audienceFor = (item: MetaInsightItem) => audienceByAdSet[adSetIDFor(item)]
const estimatedAudienceSize = (item: MetaInsightItem) => {
  const audience = audienceFor(item)
  if (!audience) return undefined
  if (audience.estimatedMonthlyUsers) return audience.estimatedMonthlyUsers
  if (since.value === until.value) return audience.estimatedDailyUsers
  return undefined
}
const audienceCoverage = (item: MetaInsightItem) => {
  const size = estimatedAudienceSize(item)
  return size && item.reach != null ? (item.reach / size) * 100 : undefined
}
const resultLabel = (type?: string) => {
  const labels: Record<string, { lo: string; th: string; en: string }> = {
    purchase: { lo: 'ການຊື້', th: 'การซื้อ', en: 'Purchases' },
    lead: { lo: 'ລີດ', th: 'ลีด', en: 'Leads' },
    'onsite_conversion.lead_grouped': { lo: 'ລີດ', th: 'ลีด', en: 'Leads' },
    messaging_conversation_started_7d: { lo: 'ການສົນທະນາ', th: 'การสนทนา', en: 'Conversations' },
    'onsite_conversion.messaging_first_reply': { lo: 'ຕອບຄັ້ງທຳອິດ', th: 'ตอบครั้งแรก', en: 'First replies' },
    landing_page_view: { lo: 'ເບິ່ງໜ້າເວັບ', th: 'ดูหน้าเว็บ', en: 'Landing-page views' },
    link_click: { lo: 'ຄລິກລິ້ງ', th: 'คลิกลิงก์', en: 'Link clicks' },
  }
  return type && labels[type] ? l(labels[type]) : l({ lo: 'ຜົນລັບ', th: 'ผลลัพธ์', en: 'Results' })
}

async function setPreset(value: string) {
  datePreset.value = value
  const days = Number(value)
  if (!Number.isFinite(days)) return
  const end = new Date()
  const start = new Date(end.getFullYear(), end.getMonth(), end.getDate() - days + 1)
  since.value = iso(start)
  until.value = iso(end)
  if (ready.value) await loadPerformance()
}

function statusFor(item: MetaInsightItem) {
  return item.effectiveStatus || item.status || campaigns.value.find(campaign => campaign.id === item.entityId)?.effectiveStatus || ''
}

const statusLabel = (status?: string) => {
  const value = (status || '').toUpperCase()
  if (value === 'ACTIVE') return l({ lo: 'ເປີດໃຊ້ງານ', th: 'กำลังใช้งาน', en: 'Active' })
  if (value === 'PAUSED') return l({ lo: 'ປິດຊົ່ວຄາວ', th: 'หยุดชั่วคราว', en: 'Paused' })
  if (value === 'CAMPAIGN_PAUSED') return l({ lo: 'Campaign ຖືກປິດ', th: 'แคมเปญถูกหยุด', en: 'Campaign paused' })
  if (value === 'ADSET_PAUSED') return l({ lo: 'Ad Set ຖືກປິດ', th: 'ชุดโฆษณาถูกหยุด', en: 'Ad set paused' })
  if (value === 'IN_PROCESS' || value === 'PENDING_REVIEW') return l({ lo: 'ກຳລັງກວດສອບ', th: 'กำลังตรวจสอบ', en: 'In review' })
  if (value === 'DISAPPROVED') return l({ lo: 'ບໍ່ຜ່ານການອະນຸມັດ', th: 'ไม่ผ่านการอนุมัติ', en: 'Disapproved' })
  return status || l({ lo: 'ບໍ່ຮູ້ສະຖານະ', th: 'ไม่ทราบสถานะ', en: 'Unknown' })
}

const statusClass = (status?: string) => {
  const value = (status || '').toUpperCase()
  if (value === 'ACTIVE' || value === 'IN_PROCESS' || value === 'PENDING_REVIEW') return 'active'
  if (value === 'PAUSED' || value === 'CAMPAIGN_PAUSED' || value === 'ADSET_PAUSED') return 'paused'
  return 'issues'
}

function decision(item: MetaInsightItem): { code: DecisionCode; color: string; icon: string; title: string; detail: string; signals: string[] } {
  if (!item.impressions && !item.spend) {
    return { code: 'no_data', color: 'default', icon: 'mdi-minus-circle-outline', title: l({ lo: 'ສ້າງໃໝ່ / ຍັງບໍ່ມີຂໍ້ມູນ', th: 'เพิ่งสร้าง / ยังไม่มีข้อมูล', en: 'New / No delivery data' }), detail: l({ lo: 'ໂຄສະນາຫາກໍ່ສ້າງໃໝ່ ຫຼື ຍັງບໍ່ທັນມີຄ່າໃຊ້ຈ່າຍ ແລະການສະແດງຜົນໃນຊ່ວງເວລານີ້.', th: 'โฆษณาเพิ่งสร้างหรือยังไม่มีค่าใช้จ่ายและการแสดงผลในช่วงเวลานี้', en: 'Newly created ad or no delivery/spend in selected date range.' }), signals: [] }
  }
  const bad: string[] = []
  const good: string[] = []
  if (targetCost.value && item.results.cost != null) {
    if (item.results.cost > targetCost.value) bad.push(l({ lo: 'ຕົ້ນທຶນ/ຜົນລັບສູງກວ່າຈຸດຄຸ້ມທຶນ', th: 'ต้นทุน/ผลลัพธ์สูงกว่าจุดคุ้มทุน', en: 'Cost/result is above break-even' }))
    else good.push(l({ lo: 'ຕົ້ນທຶນ/ຜົນລັບຢູ່ໃນເປົ້າ', th: 'ต้นทุน/ผลลัพธ์อยู่ในเป้า', en: 'Cost/result is within target' }))
  }
  const estimatedContribution = contribution(item)
  if (estimatedContribution != null) {
    if (estimatedContribution < 0) bad.push(l({ lo: 'ມູນຄ່າຂາຍຫຼັງຫັກ Margin ບໍ່ພໍຄ່າ Ads', th: 'มูลค่าขายหลังหัก Margin ไม่พอค่าโฆษณา', en: 'Margin-adjusted conversion value does not cover ad spend' }))
    else good.push(l({ lo: 'ມູນຄ່າຂາຍຫຼັງຫັກ Margin ຍັງຄຸ້ມຄ່າ Ads', th: 'มูลค่าขายหลังหัก Margin ยังคุ้มค่าโฆษณา', en: 'Margin-adjusted conversion value covers ad spend' }))
  }
  if (item.linkCtr != null) {
    if (item.linkCtr < weakCTR.value) bad.push(l({ lo: 'CTR ຕ່ຳ—Creative ອາດບໍ່ດຶງດູດ', th: 'CTR ต่ำ—ครีเอทีฟอาจไม่น่าสนใจ', en: 'Low CTR—creative may not attract clicks' }))
    else if (item.linkCtr >= goodCTR.value) good.push(l({ lo: 'CTR ດີ', th: 'CTR ดี', en: 'Strong CTR' }))
  }
  if (item.frequency != null) {
    if (item.frequency >= maxFrequency.value) bad.push(l({ lo: 'Frequency ສູງ—Audience ເລີ່ມເຫັນຊ້ຳ', th: 'Frequency สูง—กลุ่มเป้าหมายเริ่มเห็นซ้ำ', en: 'High frequency—audience is seeing repeats' }))
    else if (item.frequency < maxFrequency.value - 0.5) good.push(l({ lo: 'Frequency ຍັງປົກກະຕິ', th: 'Frequency ยังปกติ', en: 'Frequency is still healthy' }))
  }
  const coverage = audienceCoverage(item)
  if (coverage != null && coverage >= 60 && item.frequency != null && item.frequency >= maxFrequency.value) {
    bad.push(l({ lo: 'ເຂົ້າເຖິງ Audience ສ່ວນໃຫຍ່ແລ້ວ ແລະ Frequency ສູງ', th: 'เข้าถึงกลุ่มเป้าหมายส่วนใหญ่แล้วและ Frequency สูง', en: 'Most of the estimated audience is reached and frequency is high' }))
  } else if (coverage != null && coverage < 30) {
    good.push(l({ lo: 'Audience ຍັງມີພື້ນທີ່ໃຫ້ຂະຫຍາຍ', th: 'กลุ่มเป้าหมายยังมีพื้นที่ให้ขยาย', en: 'The audience still has room to scale' }))
  }
  const rankings = [item.qualityRanking, item.engagementRateRanking, item.conversionRateRanking].filter(Boolean)
  if (rankings.some(value => value?.includes('BELOW_AVERAGE'))) bad.push(l({ lo: 'Ranking ຕ່ຳກວ່າຄ່າສະເລ່ຍ', th: 'อันดับต่ำกว่าค่าเฉลี่ย', en: 'Ranking is below average' }))
  if (bad.length >= 2) return { code: 'pause', color: 'error', icon: 'mdi-pause-circle-outline', title: l({ lo: 'ຄວນພິຈາລະນາປິດ', th: 'ควรพิจารณาปิด', en: 'Pause candidate' }), detail: l({ lo: 'ມີຫຼາຍສັນຍານອ່ອນ. ກວດ 3–7 ມື້ ແລະກຳໄລກ່ອນປິດ.', th: 'มีหลายสัญญาณอ่อน ตรวจแนวโน้ม 3–7 วันและกำไรก่อนปิด', en: 'Several weak signals. Check the 3–7 day trend and profit before pausing.' }), signals: bad }
  if (bad.length === 1) return { code: 'review', color: 'warning', icon: 'mdi-alert-circle-outline', title: l({ lo: 'ຄວນກວດສອບ', th: 'ควรตรวจสอบ', en: 'Review' }), detail: l({ lo: 'ຍັງບໍ່ຄວນປິດທັນທີ. ປັບ Creative, Audience ຫຼືລໍຖ້າເບິ່ງ trend.', th: 'ยังไม่ควรปิดทันที ลองปรับครีเอทีฟ กลุ่มเป้าหมาย หรือดูแนวโน้มต่อ', en: 'Do not pause yet. Review creative, audience, and the trend.' }), signals: bad }
  return { code: 'keep', color: 'success', icon: 'mdi-check-circle-outline', title: l({ lo: 'ຄວນເປີດຕໍ່', th: 'ควรเปิดต่อ', en: 'Keep running' }), detail: l({ lo: 'ສັນຍານຫຼັກຍັງດີ. ຕິດຕາມຕົ້ນທຶນ ແລະ trend ຕໍ່ໄປ.', th: 'สัญญาณหลักยังดี ติดตามต้นทุนและแนวโน้มต่อไป', en: 'Core signals look healthy. Continue monitoring cost and trend.' }), signals: good }
}

const decisionCounts = computed(() => items.value.reduce((result, item) => {
  result[decision(item).code] += 1
  return result
}, { keep: 0, review: 0, pause: 0, no_data: 0 } as Record<DecisionCode, number>))
const actionCandidates = computed(() => (report.value?.items || []).filter((item) => {
  if (statusFor(item).toUpperCase() !== 'ACTIVE' || decision(item).code !== 'pause') return false
  if ((item.impressions || 0) < minimumPlanImpressions.value) return false
  return minimumPlanSpend.value == null || (item.spend || 0) >= minimumPlanSpend.value
}))
const selectedActionItems = computed(() => actionCandidates.value.filter(item => selectedActionIds.value.includes(item.entityId)))

function loadThresholds() {
  if (!import.meta.client || !adAccountId.value) return
  const saved = localStorage.getItem(`ads-thresholds:${adAccountId.value}`)
  if (!saved) return
  try {
    const value = JSON.parse(saved)
    targetCost.value = value.targetCost ?? null
    goodCTR.value = value.goodCTR ?? 3
    weakCTR.value = value.weakCTR ?? 1
    maxFrequency.value = value.maxFrequency ?? 3
    grossMargin.value = value.grossMargin ?? null
    minimumPlanImpressions.value = value.minimumPlanImpressions ?? 1000
    minimumPlanSpend.value = value.minimumPlanSpend ?? null
  } catch {}
}
function saveThresholds() {
  if (!import.meta.client || !adAccountId.value) return
  localStorage.setItem(`ads-thresholds:${adAccountId.value}`, JSON.stringify({ targetCost: targetCost.value, goodCTR: goodCTR.value, weakCTR: weakCTR.value, maxFrequency: maxFrequency.value, grossMargin: grossMargin.value, minimumPlanImpressions: minimumPlanImpressions.value, minimumPlanSpend: minimumPlanSpend.value }))
  settingsOpen.value = false
}

async function loadAccounts() {
  loadingAccounts.value = true
  notice.value = ''
  try {
    const result = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts')
    adAccounts.value = result.items || []
    if (!adAccountId.value || !adAccounts.value.some(item => item.id === adAccountId.value)) adAccountId.value = adAccounts.value[0]?.id || ''
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || error?.statusMessage || l({ lo: 'ບໍ່ສາມາດໂຫຼດບັນຊີໂຄສະນາໄດ້.', th: 'โหลดบัญชีโฆษณาไม่สำเร็จ', en: 'Could not load ad accounts.' })
  } finally { loadingAccounts.value = false }
}

async function loadPerformance() {
  if (!adAccountId.value || !since.value || !until.value) return
  loadingReport.value = true
  notice.value = ''
  success.value = ''
  try {
    const query = new URLSearchParams({ level: level.value, since: since.value, until: until.value })
    const [campaignResult, insightResult] = await Promise.all([
      $fetch<{ items: MetaCampaign[] }>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/campaigns`),
      $fetch<MetaInsightsReport>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/insights?${query.toString()}`),
    ])
    campaigns.value = campaignResult.items || []
    campaignsAccountId.value = adAccountId.value
    report.value = insightResult
    selectedId.value = ''
  } catch (error: any) {
    report.value = null
    notice.value = error?.data?.error?.message || error?.data?.message || error?.statusMessage || l({ lo: 'ດຶງຂໍ້ມູນ Insights ບໍ່ສຳເລັດ. ກວດສິດ ads_read ແລ້ວລອງໃໝ່.', th: 'โหลด Insights ไม่สำเร็จ ตรวจสิทธิ์ ads_read แล้วลองใหม่', en: 'Could not load Insights. Check ads_read permission and retry.' })
  } finally { loadingReport.value = false }
}

async function loadAudience(item: MetaInsightItem) {
  const adSetID = adSetIDFor(item)
  if (!adSetID || audienceByAdSet[adSetID] || audienceLoading[adSetID]) return
  audienceLoading[adSetID] = true
  audienceErrors[adSetID] = ''
  try {
    audienceByAdSet[adSetID] = await $fetch<MetaAudienceAnalysis>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/audience/${encodeURIComponent(adSetID)}`)
  } catch (error: any) {
    audienceErrors[adSetID] = error?.data?.error?.message || error?.data?.message || l({ lo: 'ບໍ່ສາມາດໂຫຼດ Audience ໄດ້.', th: 'โหลดข้อมูลกลุ่มเป้าหมายไม่สำเร็จ', en: 'Could not load audience details.' })
  } finally {
    audienceLoading[adSetID] = false
  }
}

async function toggleItem(item: MetaInsightItem) {
  selectedId.value = selectedId.value === item.entityId ? '' : item.entityId
  if (selectedId.value && item.entityType !== 'campaign') await loadAudience(item)
}

function audienceAdvice(item: MetaInsightItem) {
  const coverage = audienceCoverage(item)
  if (coverage == null) return { tone: 'info', title: l({ lo: 'ຂໍ້ມູນປະມານຍັງບໍ່ພໍ', th: 'ข้อมูลประมาณการยังไม่พอ', en: 'Estimate not available' }), detail: l({ lo: 'ໃຫ້ໃຊ້ Frequency, CTR ແລະ Cost/Result ເປັນຫຼັກ.', th: 'ให้ใช้ Frequency, CTR และ Cost/Result เป็นหลัก', en: 'Use frequency, CTR, and cost per result as the primary signals.' }) }
  if (coverage >= 60 && (item.frequency || 0) >= maxFrequency.value) return { tone: 'warning', title: l({ lo: 'Audience ອາດເລີ່ມອີ່ມຕົວ', th: 'กลุ่มเป้าหมายอาจเริ่มอิ่มตัว', en: 'Audience may be saturating' }), detail: l({ lo: 'ຖ້າ CTR ລົງ ແລະ Cost/Result ສູງຂຶ້ນ ໃຫ້ປ່ຽນ Creative, ຂະຫຍາຍ Audience ຫຼືປິດຕົວທີ່ອ່ອນ.', th: 'หาก CTR ลดและ Cost/Result สูงขึ้น ให้เปลี่ยนครีเอทีฟ ขยายกลุ่มเป้าหมาย หรือหยุดตัวที่อ่อน', en: 'If CTR is falling and cost/result is rising, refresh creative, broaden the audience, or pause the weak item.' }) }
  return { tone: 'success', title: l({ lo: 'Audience ຍັງໄປຕໍ່ໄດ້', th: 'กลุ่มเป้าหมายยังไปต่อได้', en: 'Audience still has capacity' }), detail: l({ lo: 'ຢ່າປິດຈາກ Ratio ຢ່າງດຽວ; ໃຫ້ເບິ່ງ Cost/Result ແລະຜົນລັບຮ່ວມກັນ.', th: 'อย่าหยุดจาก Ratio เพียงอย่างเดียว ให้ดู Cost/Result และผลลัพธ์ร่วมกัน', en: 'Do not pause from coverage alone; compare it with cost per result and actual outcomes.' }) }
}

function targetingTags(audience?: MetaAudienceAnalysis) {
  if (!audience) return []
  const targeting = audience.targeting || {}
  const tags: { icon: string; label: string }[] = []
  if (targeting.ageMin || targeting.ageMax) tags.push({ icon: 'mdi-account-clock-outline', label: `${targeting.ageMin || 18}–${targeting.ageMax || '65+'}` })
  if (targeting.genders?.length) tags.push({ icon: 'mdi-gender-male-female', label: targeting.genders.join(', ') })
  for (const value of [...(targeting.countries || []), ...(targeting.regions || []), ...(targeting.cities || [])].slice(0, 6)) tags.push({ icon: 'mdi-map-marker-outline', label: value })
  for (const value of (targeting.interests || []).slice(0, 6)) tags.push({ icon: 'mdi-heart-outline', label: value })
  for (const value of (targeting.customAudiences || []).slice(0, 5)) tags.push({ icon: 'mdi-account-group-outline', label: value })
  for (const value of (targeting.publisherPlatforms || []).slice(0, 4)) tags.push({ icon: 'mdi-cellphone-link', label: value })
  return tags
}

async function changeStatus(item: MetaInsightItem, status: 'ACTIVE' | 'PAUSED', confirmed = false) {
  if (status === 'PAUSED' && !confirmed && !confirm(l({ lo: `ປິດ “${item.entityName}”? ການສົ່ງໂຄສະນາຈະຢຸດຈົນກວ່າຈະເປີດຄືນ.`, th: `ปิด “${item.entityName}”? การส่งโฆษณาจะหยุดจนกว่าจะเปิดอีกครั้ง`, en: `Pause “${item.entityName}”? Delivery will stop until it is activated again.` }))) return false
  updatingId.value = item.entityId
  notice.value = ''
  success.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${item.entityType}/${encodeURIComponent(item.entityId)}/status`, { method: 'PATCH', body: { status } })
    const campaign = campaigns.value.find(value => value.id === item.entityId)
    if (campaign) campaign.effectiveStatus = status
    item.status = status
    item.effectiveStatus = status
    success.value = status === 'ACTIVE'
      ? l({ lo: `ເປີດ “${item.entityName}” ສຳເລັດ.`, th: `เปิด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now active.` })
      : l({ lo: `ປິດ “${item.entityName}” ສຳເລັດ.`, th: `ปิด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now paused.` })
    return true
  } catch (error: any) {
    notice.value = error?.data?.error?.code === 'internal_error'
      ? l({ lo: 'Meta ບໍ່ອະນຸຍາດໃຫ້ປ່ຽນສະຖານະ. Reconnect Facebook ແລະເລືອກ Manage ads (ads_management).', th: 'Meta ไม่อนุญาตให้เปลี่ยนสถานะ เชื่อม Facebook ใหม่และเลือก Manage ads (ads_management)', en: 'Meta denied the change. Reconnect Facebook and select Manage ads (ads_management).' })
      : error?.data?.error?.message || error?.data?.message || error?.statusMessage || l({ lo: 'ປ່ຽນສະຖານະບໍ່ສຳເລັດ.', th: 'เปลี่ยนสถานะไม่สำเร็จ', en: 'Could not change status.' })
    return false
  } finally { updatingId.value = '' }
}

function openBudgetModal(item: MetaInsightItem) {
  targetBudgetItem.value = item
  selectedPercent.value = 20

  const start = new Date(since.value).getTime()
  const end = new Date(until.value).getTime()
  const days = Math.max(1, Math.round((end - start) / (1000 * 60 * 60 * 24)) + 1)

  let base = 0
  if (item.dailyBudget && item.dailyBudget > 0) {
    base = item.dailyBudget / currencyOffset.value
  } else if (item.spend && item.spend > 0) {
    base = item.spend / days
  } else {
    base = currencyOffset.value === 1 ? 100000 : 10
  }

  if (currencyOffset.value === 1) {
    base = Math.max(20000, Math.round(base / 10000) * 10000)
  } else {
    base = Math.max(1, Math.round(base * 100) / 100)
  }

  baseDailyBudget.value = base
  setBudgetPercent(20)
  budgetModalOpen.value = true
}

function setBudgetPercent(pct: number) {
  selectedPercent.value = pct
  let calculated = baseDailyBudget.value * (1 + pct / 100)
  if (currencyOffset.value === 1) {
    calculated = Math.max(20000, Math.round(calculated / 10000) * 10000)
  } else {
    calculated = Math.max(1, Math.round(calculated * 100) / 100)
  }
  newDailyBudget.value = calculated
}

const budgetDiff = computed(() => {
  return (newDailyBudget.value || 0) - (baseDailyBudget.value || 0)
})

const budgetDiffPercent = computed(() => {
  if (!baseDailyBudget.value || baseDailyBudget.value <= 0) return 0
  return Math.round((((newDailyBudget.value || 0) - baseDailyBudget.value) / baseDailyBudget.value) * 100)
})

async function applyBudgetChange() {
  if (!targetBudgetItem.value || newDailyBudget.value <= 0) return
  updatingBudget.value = true
  notice.value = ''

  const item = targetBudgetItem.value
  let targetType: 'campaign' | 'adset' = 'campaign'
  let targetId = item.entityId
  let targetName = item.entityName

  if (item.entityType === 'ad') {
    if (item.adSetId) {
      targetType = 'adset'
      targetId = item.adSetId
      targetName = item.adSetName || item.entityName
    } else {
      targetType = 'campaign'
      targetId = item.campaignId || item.entityId
      targetName = item.campaignName || item.entityName
    }
  } else if (item.entityType === 'adset') {
    targetType = 'adset'
    targetId = item.entityId
  } else {
    targetType = 'campaign'
    targetId = item.entityId
  }

  const apiBudget = Math.round(newDailyBudget.value * currencyOffset.value)

  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${targetType}/${encodeURIComponent(targetId)}/budget`, {
      method: 'PATCH',
      body: { dailyBudget: apiBudget },
    })

    success.value = l({
      lo: `ປັບງົບປະມານໃຫ້ “${targetName}” ເປັນ ${money(newDailyBudget.value)}/ມື້ ສຳເລັດແລ້ວ!`,
      th: `ปรับงบประมาณให้ “${targetName}” เป็น ${money(newDailyBudget.value)}/วัน เรียบร้อยแล้ว`,
      en: `Updated budget for “${targetName}” to ${money(newDailyBudget.value)}/day successfully.`,
    })
    budgetModalOpen.value = false
    await loadPerformance()
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || error?.statusMessage || l({
      lo: 'ປັບງົບບໍ່ສຳເລັດ. ກວດສອບສິດ ads_management ຫຼື ປະເພດງົບປະມານຂອງແຄມເປນ.',
      th: 'ปรับงบประมาณไม่สำเร็จ ตรวจสอบสิทธิ์ ads_management หรือประเภทงบประมาณ',
      en: 'Could not update budget. Confirm ads_management permission and budget type.',
    })
  } finally {
    updatingBudget.value = false
  }
}

function openActionPlan() {
  selectedActionIds.value = []
  actionPlanAcknowledged.value = false
  actionPlanOpen.value = true
}

async function applyActionPlan() {
  if (!actionPlanAcknowledged.value || !selectedActionItems.value.length) return
  applyingActionPlan.value = true
  let completed = 0
  for (const item of [...selectedActionItems.value]) {
    if (await changeStatus(item, 'PAUSED', true)) completed += 1
  }
  applyingActionPlan.value = false
  actionPlanOpen.value = false
  selectedActionIds.value = []
  actionPlanAcknowledged.value = false
  if (completed) success.value = l({ lo: `ປິດໂຄສະນາ ${completed} ລາຍການຕາມແຜນທີ່ອະນຸມັດແລ້ວ.`, th: `หยุดโฆษณา ${completed} รายการตามแผนที่อนุมัติแล้ว`, en: `Paused ${completed} approved item${completed === 1 ? '' : 's'}.` })
}

watch(adAccountId, async () => { if (!ready.value) return; loadThresholds(); await loadPerformance() })
watch(level, async () => { if (ready.value) await loadPerformance() })
onMounted(async () => {
  await Promise.all([loadAccounts(), loadAutomationsAndProducts()])
  loadThresholds()
  ready.value = true
  if (adAccountId.value) await loadPerformance()
})
</script>

<template>
  <!-- Shared Ads Suite Navigation -->
  <nav class="app-sub-nav" aria-label="Ads suite navigation">
    <NuxtLink to="/ads" class="app-sub-nav-item active">
      <v-icon icon="mdi-bullhorn-outline" size="18" />
      <span>{{ l({ lo: '1. Ads Manager (ນຳສົ່ງ & ຄ່າແອດ)', th: '1. Ads Manager (นำส่ง & ค่าแอด)', en: '1. Ads Manager (Delivery & Creatives)' }) }}</span>
    </NuxtLink>
    <NuxtLink to="/ads-intelligence" class="app-sub-nav-item">
      <v-icon icon="mdi-chart-timeline-variant-shimmer" size="18" />
      <span>{{ l({ lo: '2. Ads Intelligence (ກຳໄລ, Real CPA & SOP)', th: '2. Ads Intelligence (กำไร, Real CPA & SOP)', en: '2. Ads Intelligence (Real CPA & SOP)' }) }}</span>
    </NuxtLink>
  </nav>

  <!-- Modern Page Head -->
  <section class="app-page-head">
    <div class="head-content">
      <span class="head-badge">
        <v-icon icon="mdi-bullhorn-outline" size="14" />
        Meta Ads Suite · Delivery & Performance
      </span>
      <h1>{{ l({ lo: 'ຈັດການໂຄສະນາ (Ads Manager)', th: 'จัดการโฆษณา (Ads Manager)', en: 'Meta Ads Manager' }) }}</h1>
      <p>{{ l({ lo: 'ຕິດຕາມຜົນງານການນຳສົ່ງ, CTR, Impressions, Frequency, ປະສິດທິພາບແອດ ແລະ ຄວບຄຸມສະຖານະ.', th: 'ติดตามผลการนำส่ง CTR, Impressions, Frequency, ประสิทธิภาพแอด และควบคุมสถานะ', en: 'Track ad delivery, CTR, impressions, frequency, creative metrics, and toggle campaign status.' }) }}</p>
    </div>
    <div class="head-actions">
      <v-btn color="warning" variant="tonal" prepend-icon="mdi-shield-check-outline" :disabled="!report" @click="openActionPlan">
        {{ l({ lo: `ກວດແຜນປັບ Ads (${actionCandidates.length})`, th: `ตรวจแผนปรับ Ads (${actionCandidates.length})`, en: `Review action plan (${actionCandidates.length})` }) }}
      </v-btn>
      <v-btn variant="outlined" prepend-icon="mdi-tune-variant" @click="settingsOpen = true">
        {{ l({ lo: 'ຕັ້ງຄ່າເກນ', th: 'ตั้งค่าเกณฑ์', en: 'Thresholds' }) }}
      </v-btn>
      <v-btn color="primary" variant="flat" prepend-icon="mdi-refresh" :loading="loadingReport" :disabled="!adAccountId" @click="loadPerformance">
        {{ l({ lo: 'ອັບເດດຂໍ້ມູນ', th: 'อัปเดตข้อมูล', en: 'Refresh' }) }}
      </v-btn>
    </div>
  </section>

  <v-alert v-if="notice" type="error" variant="tonal" class="mb-4" closable @click:close="notice = ''">
    {{ notice }}
    <template #append><v-btn size="small" variant="outlined" to="/meta-pages">{{ l({ lo: 'ກວດສິດ Meta', th: 'ตรวจสิทธิ์ Meta', en: 'Check Meta access' }) }}</v-btn></template>
  </v-alert>
  <v-alert v-if="success" type="success" variant="tonal" class="mb-4" closable @click:close="success = ''">{{ success }}</v-alert>

  <!-- Modern Standardized Command & Filter Bar -->
  <section class="app-filter-bar" aria-label="Ads report filters">
    <div class="app-filter-group">
      <v-select
        v-model="adAccountId"
        :items="adAccounts.map(account => ({ ...account, label: `${account.accountType === 'business' ? 'Business' : 'Personal'} · ${account.name}` }))"
        item-title="label"
        item-value="id"
        :label="l({ lo: 'ບັນຊີໂຄສະນາ', th: 'บัญชีโฆษณา', en: 'Ad account' })"
        variant="outlined"
        density="compact"
        hide-details
        :loading="loadingAccounts"
        class="filter-account-select"
      />
      <v-select
        v-model="selectedProductId"
        :items="productOptions"
        item-title="title"
        item-value="value"
        :label="l({ lo: 'ສິນຄ້າ (Auto Reply)', th: 'สินค้า (Auto Reply)', en: 'Product' })"
        variant="outlined"
        density="compact"
        hide-details
        class="filter-product-select"
      />
      <div class="app-filter-divider" />
      <v-select
        :model-value="datePreset"
        :items="[
          { title: l({ lo: '7 ມື້ຫຼ້າສຸດ', th: '7 วันล่าสุด', en: 'Last 7 days' }), value: '7' },
          { title: l({ lo: '30 ມື້ຫຼ້າສຸດ', th: '30 วันล่าสุด', en: 'Last 30 days' }), value: '30' },
          { title: l({ lo: 'ກຳນົດເອງ', th: 'กำหนดเอง', en: 'Custom' }), value: 'custom' }
        ]"
        item-title="title"
        item-value="value"
        :label="l({ lo: 'ໄລຍະເວລາ', th: 'ช่วงเวลา', en: 'Date range' })"
        variant="outlined"
        density="compact"
        hide-details
        class="filter-date-select"
        @update:model-value="setPreset"
      />
      <v-text-field v-if="datePreset === 'custom'" v-model="since" type="date" label="From" variant="outlined" density="compact" hide-details class="filter-custom-date" />
      <v-text-field v-if="datePreset === 'custom'" v-model="until" type="date" label="To" variant="outlined" density="compact" hide-details class="filter-custom-date" />
    </div>

    <div class="app-filter-group">
      <!-- Auto VAT indicator pill -->
      <v-chip size="small" variant="tonal" color="primary" class="font-weight-medium">
        <v-icon icon="mdi-receipt-text-outline" start size="14" />
        VAT {{ vatDetails.flag }} {{ vatDetails.rate }}% (Auto)
      </v-chip>
      <v-btn color="primary" variant="tonal" height="40" :loading="loadingReport" @click="loadPerformance">
        {{ l({ lo: 'ນຳໃຊ້', th: 'ใช้ช่วงเวลา', en: 'Apply' }) }}
      </v-btn>
    </div>
  </section>

  <div v-if="loadingReport && !report" class="loading-grid" aria-live="polite"><v-skeleton-loader v-for="n in 6" :key="n" type="article" /></div>

  <template v-else-if="report">
    <!-- Modern 5-KPI Cards Grid with VAT Breakdown -->
    <section class="app-kpi-grid" aria-label="Ads summary">
      <article class="app-kpi-card accent-primary">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ຍອດຈ່າຍຄ່າແອດຕົວຈິງ (Gross)', th: 'ยอดใช้จ่ายจริงรวม VAT', en: 'Total Gross Spend' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-cash" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ money(vatComputation.grossSpend) }}</strong>
        <div class="kpi-sub">
          <span>Net: {{ money(vatComputation.netSpend) }}</span>
          <span class="text-primary font-weight-bold">+ VAT {{ vatDetails.rate }}% ({{ money(vatComputation.vatAmount) }})</span>
        </div>
      </article>

      <article class="app-kpi-card accent-info">
        <div class="kpi-header">
          <span class="kpi-label">{{ resultLabel(summary?.results.type) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-bullseye-arrow" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ fmt(summary?.results.value, 0) }}</strong>
        <div class="kpi-sub">
          <span>ຕົ້ນທຶນ/ຜົນລັບ: <b>{{ money(summary?.results.cost) }}</b></span>
        </div>
      </article>

      <article class="app-kpi-card accent-purple">
        <div class="kpi-header">
          <span class="kpi-label">Link CTR</span>
          <span class="kpi-icon"><v-icon icon="mdi-cursor-default-click" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ percent(summary?.linkCtr) }}</strong>
        <div class="kpi-sub">
          <span>{{ (summary?.linkCtr || 0) >= goodCTR ? '🟢 CTR ດີ' : (summary?.linkCtr || 0) < weakCTR ? '🔴 CTR ຕ່ຳ' : '🟡 CTR ປານກາງ' }}</span>
        </div>
      </article>

      <article class="app-kpi-card accent-warning">
        <div class="kpi-header">
          <span class="kpi-label">Frequency / Reach</span>
          <span class="kpi-icon"><v-icon icon="mdi-account-group-outline" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ fmt(summary?.frequency, 2) }}x</strong>
        <div class="kpi-sub">
          <span>Reach: {{ fmt(summary?.reach, 0) }} ຄົນ</span>
        </div>
      </article>

      <article class="app-kpi-card accent-success">
        <div class="kpi-header">
          <span class="kpi-label">ROAS / Conversion</span>
          <span class="kpi-icon"><v-icon icon="mdi-chart-line" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ summary?.roas == null ? '—' : `${fmt(summary.roas, 2)}×` }}</strong>
        <div class="kpi-sub">
          <span>ມູນຄ່າ: {{ money(summary?.conversionValue) }}</span>
        </div>
      </article>
    </section>

    <section class="analysis-grid">
      <v-card class="trend-card" variant="outlined">
        <div class="card-heading"><div><p>TREND</p><h2>{{ l({lo:'ຄ່າໃຊ້ຈ່າຍລາຍມື້',th:'ค่าใช้จ่ายรายวัน',en:'Daily spend trend'}) }}</h2></div><small>{{ l({lo:'ກົດໃສ່ແຕ່ລະແຖບເພື່ອເບິ່ງຄ່າ',th:'แตะแต่ละแท่งเพื่อดูค่า',en:'Hover or focus a bar for its value'}) }}</small></div>
        <div v-if="report.trend.length" class="trend-chart">
          <div v-for="point in report.trend" :key="point.date" class="trend-column" tabindex="0" :aria-label="`${point.date}: ${money(point.spend)}`">
            <span class="trend-tooltip">{{ point.date }}<br><strong>{{ money(point.spend) }}</strong><br>CTR {{ percent(point.linkCtr) }}</span>
            <div class="trend-bar" :style="{ height: `${Math.max(4, ((point.spend || 0) / maxTrendSpend) * 100)}%` }" />
            <small>{{ point.date.slice(5) }}</small>
          </div>
        </div>
        <div v-else class="empty-inline">{{ l({lo:'ຍັງບໍ່ມີ trend ສຳລັບໄລຍະນີ້',th:'ยังไม่มีแนวโน้มสำหรับช่วงนี้',en:'No trend data for this period.'}) }}</div>
      </v-card>

      <v-card class="decision-card" variant="outlined">
        <div class="card-heading"><div><p>DECISION GUIDE</p><h2>{{ l({lo:'ອ່ານສັນຍານໃຫ້ງ່າຍ',th:'อ่านสัญญาณให้ง่าย',en:'Decision signals'}) }}</h2></div><v-btn icon="mdi-tune-variant" variant="text" :aria-label="l({lo:'ຕັ້ງຄ່າເກນ',th:'ตั้งค่าเกณฑ์',en:'Configure thresholds'})" @click="settingsOpen=true" /></div>
        <div class="decision-summary"><div class="decision-count keep"><strong>{{ decisionCounts.keep }}</strong><span>{{ l({lo:'ເປີດຕໍ່',th:'เปิดต่อ',en:'Keep'}) }}</span></div><div class="decision-count review"><strong>{{ decisionCounts.review }}</strong><span>{{ l({lo:'ກວດສອບ',th:'ตรวจสอบ',en:'Review'}) }}</span></div><div class="decision-count pause"><strong>{{ decisionCounts.pause }}</strong><span>{{ l({lo:'ພິຈາລະນາປິດ',th:'พิจารณาปิด',en:'Pause candidates'}) }}</span></div></div>
        <p class="guide-note"><v-icon icon="mdi-information-outline" /> {{ l({lo:'ຄຳແນະນຳໃຊ້ຫຼາຍ metric ຮ່ວມກັນ. ຢ່າປິດເພາະມື້ດຽວ ຫຼື metric ດຽວ.',th:'คำแนะนำใช้หลาย metric ร่วมกัน อย่าปิดเพราะวันเดียวหรือค่าเดียว',en:'Recommendations combine several metrics. Never pause from one day or one metric alone.'}) }}</p>
        <ol class="decision-steps">
          <li><strong>{{ l({lo:'1. ເບິ່ງຜົນທາງທຸລະກິດ',th:'1. ดูผลทางธุรกิจ',en:'1. Check business outcomes'}) }}</strong><span>{{ l({lo:'Cost/Result ຕ້ອງບໍ່ເກີນຈຸດຄຸ້ມທຶນ.',th:'Cost/Result ต้องไม่เกินจุดคุ้มทุน',en:'Cost/result should stay below your break-even point.'}) }}</span></li>
          <li><strong>{{ l({lo:'2. ເບິ່ງຄຸນນະພາບ Creative',th:'2. ดูคุณภาพครีเอทีฟ',en:'2. Check creative quality'}) }}</strong><span>{{ l({lo:'CTR ລົງພ້ອມ Cost ສູງຂຶ້ນ ໝາຍເຖິງຄວນປ່ຽນ Creative.',th:'CTR ลดพร้อมต้นทุนสูงขึ้น หมายถึงควรเปลี่ยนครีเอทีฟ',en:'Falling CTR with rising cost suggests a creative refresh.'}) }}</span></li>
          <li><strong>{{ l({lo:'3. ເບິ່ງ Audience saturation',th:'3. ดูความอิ่มตัวของกลุ่มเป้าหมาย',en:'3. Check audience saturation'}) }}</strong><span>{{ l({lo:'Coverage ແລະ Frequency ສູງພ້ອມຜົນງານຕົກ ຈຶ່ງຄ່ອຍພິຈາລະນາປິດ.',th:'Coverage และ Frequency สูงพร้อมผลงานตก จึงค่อยพิจารณาหยุด',en:'Consider pausing only when high coverage and frequency coincide with declining results.'}) }}</span></li>
        </ol>
      </v-card>
    </section>

    <section class="performance-section">
      <div class="section-head">
        <div>
          <p>PERFORMANCE DETAIL</p>
          <h2>{{ l({lo:'ເຈາະເລິກເຖິງລະດັບ Ad',th:'เจาะลึกถึงระดับโฆษณา',en:'Drill down to individual ads'}) }}</h2>
        </div>
        <div class="report-meta">
          <span>{{ l({lo:'ອັບເດດ',th:'อัปเดต',en:'Updated'}) }} {{ new Date(report.generatedAt).toLocaleString() }}</span>
          <v-select
            v-model="sortBy"
            :items="[
              { title: l({ lo: '⚡ ໃຊ້ຈ່າຍສູງສຸດ', th: '⚡ ใช้จ่ายสูงสุด', en: 'Highest spend' }), value: 'spend' },
              { title: l({ lo: '✨ ສ້າງໃໝ່ສຸດ', th: '✨ สร้างใหม่ล่าสุด', en: 'Newest created' }), value: 'newest' },
              { title: l({ lo: '🔤 ຊື່ (A-Z)', th: '🔤 ชื่อ (A-Z)', en: 'Name (A-Z)' }), value: 'name' }
            ]"
            item-title="title"
            item-value="value"
            density="compact"
            variant="outlined"
            hide-details
            class="sort-select"
            style="min-width: 160px; max-width: 195px;"
          />
          <v-text-field v-model="search" density="compact" variant="outlined" prepend-inner-icon="mdi-magnify" :placeholder="l({lo:'ຄົ້ນຫາ...',th:'ค้นหา...',en:'Search...'})" hide-details clearable />
        </div>
      </div>
      <div class="detail-toolbar">
        <div class="toolbar-group">
          <small>{{ l({lo:'ລະດັບຂໍ້ມູນ',th:'ระดับข้อมูล',en:'View level'}) }}</small>
          <v-btn-toggle v-model="level" mandatory color="primary" variant="outlined" class="level-tabs">
            <v-btn value="campaign"><v-icon icon="mdi-bullhorn-outline" /> <span>Campaign</span></v-btn>
            <v-btn value="adset"><v-icon icon="mdi-account-group-outline" /> <span>Ad Set</span></v-btn>
            <v-btn value="ad"><v-icon icon="mdi-advertisements" /> <span>Ads</span></v-btn>
          </v-btn-toggle>
        </div>
        <div class="toolbar-group status-filter-group">
          <small>{{ l({lo:'ກອງຕາມສະຖານະ',th:'กรองตามสถานะ',en:'Filter by status'}) }}</small>
          <v-btn-toggle v-model="statusFilter" mandatory color="primary" variant="outlined" class="status-tabs">
            <v-btn value="all">{{ l({lo:'ທັງໝົດ',th:'ทั้งหมด',en:'All'}) }}</v-btn>
            <v-btn value="active"><span class="status-dot active" />{{ l({lo:'ເປີດຢູ່',th:'ใช้งาน',en:'Active'}) }} <b>{{ statusCounts.active }}</b></v-btn>
            <v-btn value="paused"><span class="status-dot paused" />{{ l({lo:'ປິດຢູ່',th:'หยุด',en:'Paused'}) }} <b>{{ statusCounts.paused }}</b></v-btn>
            <v-btn value="issues"><span class="status-dot issues" />{{ l({lo:'ມີບັນຫາ',th:'มีปัญหา',en:'Issues'}) }} <b>{{ statusCounts.issues }}</b></v-btn>
          </v-btn-toggle>
        </div>
      </div>
      <p class="filter-help"><v-icon icon="mdi-information-outline" /> {{ l({lo:'ສະຖານະນີ້ແມ່ນຄ່າຈິງຈາກ Meta. Ad ອາດຖືກຢຸດເພາະ Campaign ຫຼື Ad Set ຊັ້ນເທິງຖືກປິດ.',th:'สถานะนี้มาจาก Meta โดยตรง โฆษณาอาจหยุดเพราะแคมเปญหรือชุดโฆษณาด้านบนถูกปิด',en:'Statuses come directly from Meta. An ad may stop because its parent campaign or ad set is paused.'}) }}</p>

      <div v-if="loadingReport" class="table-loading"><v-progress-linear indeterminate color="primary" /></div>
      <div v-else-if="!items.length" class="empty-state"><v-icon icon="mdi-filter-off-outline" size="44" /><h3>{{ l({lo:'ບໍ່ພົບລາຍການຕາມ filter',th:'ไม่พบรายการตามตัวกรอง',en:'No items match these filters'}) }}</h3><p>{{ l({lo:'ລອງເລືອກສະຖານະທັງໝົດ ຫຼືລຶບຄຳຄົ້ນຫາ.',th:'ลองเลือกสถานะทั้งหมดหรือล้างคำค้นหา',en:'Choose All statuses or clear the search.'}) }}</p><v-btn variant="tonal" color="primary" @click="statusFilter='all';search=''">{{ l({lo:'ລ້າງ filter',th:'ล้างตัวกรอง',en:'Clear filters'}) }}</v-btn></div>

      <div v-else class="performance-list">
        <article v-for="item in items" :key="item.entityId" class="performance-row" :class="{ expanded: selectedId === item.entityId }">
          <button class="row-main" type="button" :aria-expanded="selectedId === item.entityId" @click="toggleItem(item)">
            <span class="entity-cell">
              <span class="entity-icon"><v-icon :icon="item.entityType==='campaign'?'mdi-bullhorn-outline':item.entityType==='adset'?'mdi-account-group-outline':'mdi-advertisements'" /></span>
              <span>
                <strong>{{ item.entityName || item.entityId }}</strong>
                <small v-if="item.entityType!=='campaign'">{{ item.campaignName }}<template v-if="item.entityType==='ad'"> · {{ item.adSetName }}</template></small>
                <code>{{ item.entityId }}</code>
                <!-- Product & Auto Reply Badge -->
                <div v-if="getLinkedProduct(item)" class="d-flex align-center gap-1 mt-1 flex-wrap">
                  <v-chip size="x-small" color="primary" variant="flat" class="font-weight-bold">
                    <v-avatar start v-if="getLinkedProduct(item)!.product.imageUrl" size="14">
                      <v-img :src="getLinkedProduct(item)!.product.imageUrl" cover />
                    </v-avatar>
                    {{ getLinkedProduct(item)!.product.name }}
                  </v-chip>
                  <span class="text-caption text-medium-emphasis">({{ getLinkedProduct(item)!.flowName }})</span>
                </div>
                <div v-else class="d-flex align-center gap-1 mt-1">
                  <v-chip size="x-small" color="grey" variant="text" class="text-caption">
                    {{ l({ lo: '⚠️ ຍັງບໍ່ຜູກ Auto Reply', th: '⚠️ ยังไม่ผูก Auto Reply', en: '⚠️ Unlinked' }) }}
                  </v-chip>
                  <NuxtLink to="/auto-replies" class="text-caption text-primary text-decoration-none font-weight-medium" @click.stop>
                    + ຜູກແອດນີ້
                  </NuxtLink>
                </div>
              </span>
            </span>
            <div class="delivery-wrap">
              <span class="delivery-chip" :class="statusClass(statusFor(item))"><span class="status-dot" :class="statusClass(statusFor(item))" />{{ statusLabel(statusFor(item)) }}</span>
              <v-btn
                size="x-small"
                variant="tonal"
                color="primary"
                prepend-icon="mdi-cash-fast"
                class="quick-scale-btn font-weight-bold ml-1"
                @click.stop="openBudgetModal(item)"
              >
                Scale
              </v-btn>
            </div>
            <span class="metric metric-spend mobile-key"><small>{{ l({lo:'ໃຊ້ຈ່າຍ',th:'ใช้จ่าย',en:'Spend'}) }}</small><strong>{{ money(item.spend) }}</strong></span>
            <span class="metric metric-results mobile-key"><small>{{ resultLabel(item.results.type) }}</small><strong>{{ fmt(item.results.value, 0) }}</strong></span>
            <span class="metric metric-cost mobile-key"><small>{{ l({lo:'ຕົ້ນທຶນ/ຜົນ',th:'ต้นทุน/ผล',en:'Cost/result'}) }}</small><strong>{{ money(item.results.cost) }}</strong></span>
            <span class="metric metric-ctr mobile-key"><small>CTR</small><strong>{{ percent(item.linkCtr) }}</strong></span>
            <span class="metric metric-frequency"><small>Frequency</small><strong>{{ fmt(item.frequency, 2) }}</strong></span>
            <span class="decision-chip" :class="decision(item).code"><v-icon :icon="decision(item).icon" /> <span><strong>{{ decision(item).title }}</strong><small>{{ decision(item).signals[0] || decision(item).detail }}</small></span></span>
            <v-icon :icon="selectedId === item.entityId ? 'mdi-chevron-up' : 'mdi-chevron-down'" class="expand-chevron" />
          </button>
          <div v-if="selectedId === item.entityId" class="row-detail">
            <div class="explanation"><v-icon :icon="decision(item).icon" :color="decision(item).color" /><div><h3>{{ decision(item).title }}</h3><p>{{ decision(item).detail }}</p><ul v-if="decision(item).signals.length"><li v-for="signal in decision(item).signals" :key="signal">{{ signal }}</li></ul></div></div>
            <section v-if="item.entityType==='campaign'" class="audience-callout"><v-icon icon="mdi-account-group-outline" /><div><strong>{{ l({lo:'Audience ກຳນົດຢູ່ລະດັບ Ad Set',th:'กลุ่มเป้าหมายกำหนดที่ระดับ Ad Set',en:'Audience is configured at Ad Set level'}) }}</strong><span>{{ l({lo:'ເລືອກແຖບ Ad Set ເພື່ອເບິ່ງ Targeting ແລະ Audience coverage ຂອງແຕ່ລະກຸ່ມ.',th:'เลือกแท็บ Ad Set เพื่อดู Targeting และ Audience coverage ของแต่ละกลุ่ม',en:'Open the Ad Set tab to inspect targeting and audience coverage for each audience.'}) }}</span></div></section>
            <section v-else class="audience-panel">
              <div class="audience-heading"><div><p>AUDIENCE SATURATION</p><h3>{{ l({lo:'ກຸ່ມເປົ້າໝາຍ ແລະການເຂົ້າເຖິງ',th:'กลุ่มเป้าหมายและการเข้าถึง',en:'Audience and delivery coverage'}) }}</h3></div><small>{{ l({lo:'ລະດັບ Ad Set',th:'ระดับ Ad Set',en:'Ad Set level'}) }}</small></div>
              <v-skeleton-loader v-if="audienceLoading[adSetIDFor(item)]" type="list-item-three-line, actions" />
              <v-alert v-else-if="audienceErrors[adSetIDFor(item)]" type="warning" variant="tonal" density="compact">{{ audienceErrors[adSetIDFor(item)] }} <v-btn size="small" variant="text" @click="loadAudience(item)">{{ l({lo:'ລອງໃໝ່',th:'ลองใหม่',en:'Retry'}) }}</v-btn></v-alert>
              <template v-else-if="audienceFor(item)">
                <div class="audience-metrics">
                  <div><span>{{ l({lo:'ຂະໜາດ Audience ປະມານ',th:'ขนาดกลุ่มเป้าหมายโดยประมาณ',en:'Estimated audience'}) }}</span><strong>{{ fmt(estimatedAudienceSize(item), 0) }}</strong><small>{{ audienceFor(item)?.estimatedMonthlyUsers ? l({lo:'Monthly estimate ຈາກ Meta',th:'ค่าประมาณรายเดือนจาก Meta',en:'Meta monthly estimate'}) : l({lo:'ບໍ່ມີຄ່າທຽບສຳລັບໄລຍະນີ້',th:'ไม่มีค่าประมาณที่เทียบกับช่วงนี้',en:'No comparable estimate for this range'}) }}</small></div>
                  <div><span>{{ l({lo:'Reach ໃນໄລຍະທີ່ເລືອກ',th:'Reach ในช่วงที่เลือก',en:'Reach in selected range'}) }}</span><strong>{{ fmt(item.reach, 0) }}</strong><small>{{ report.dateStart }} – {{ report.dateStop }}</small></div>
                  <div class="coverage-metric"><span>{{ l({lo:'Estimated coverage',th:'Estimated coverage',en:'Estimated coverage'}) }}</span><strong>{{ percent(audienceCoverage(item)) }}</strong><small>{{ l({lo:'Reach ÷ Estimated audience; ບໍ່ແມ່ນ lifetime ratio',th:'Reach ÷ Estimated audience; ไม่ใช่ lifetime ratio',en:'Reach ÷ estimated audience; not lifetime ratio'}) }}</small></div>
                  <div><span>Frequency</span><strong>{{ fmt(item.frequency, 2) }}</strong><small>{{ l({lo:'ຈຳນວນເທື່ອສະເລ່ຍທີ່ຄົນໜຶ່ງເຫັນ',th:'จำนวนครั้งเฉลี่ยที่หนึ่งคนเห็น',en:'Average views per reached person'}) }}</small></div>
                </div>
                <v-progress-linear v-if="audienceCoverage(item)!=null" :model-value="Math.min(100,audienceCoverage(item) || 0)" height="9" rounded color="primary" class="coverage-bar" />
                <div class="audience-advice" :class="audienceAdvice(item).tone"><v-icon :icon="audienceAdvice(item).tone==='warning'?'mdi-alert-outline':audienceAdvice(item).tone==='success'?'mdi-check-circle-outline':'mdi-information-outline'" /><div><strong>{{ audienceAdvice(item).title }}</strong><span>{{ audienceAdvice(item).detail }}</span></div></div>
                <div class="targeting-block"><strong>{{ l({lo:'Targeting ທີ່ຕັ້ງໄວ້',th:'Targeting ที่ตั้งไว้',en:'Configured targeting'}) }}</strong><div v-if="targetingTags(audienceFor(item)).length" class="targeting-tags"><span v-for="tag in targetingTags(audienceFor(item))" :key="`${tag.icon}:${tag.label}`"><v-icon :icon="tag.icon" size="15" />{{ tag.label }}</span></div><small v-else>{{ l({lo:'Meta ບໍ່ສົ່ງລາຍລະອຽດ Targeting ສຳລັບ Ad Set ນີ້.',th:'Meta ไม่ส่งรายละเอียด Targeting สำหรับ Ad Set นี้',en:'Meta did not return targeting details for this Ad Set.'}) }}</small></div>
                <div class="native-metric-note"><v-icon icon="mdi-shield-check-outline" /><span>{{ l({lo:'First-time impression ratio, Auction overlap ແລະ Meta lifetime Audience Reached Ratio ຈະສະແດງສະເພາະເມື່ອ Meta ສົ່ງຄ່າຜ່ານ API. ລະບົບຈະບໍ່ສ້າງຄ່າປອມ.',th:'First-time impression ratio, Auction overlap และ Meta lifetime Audience Reached Ratio จะแสดงเมื่อ Meta ส่งค่าผ่าน API เท่านั้น ระบบจะไม่สร้างค่าจำลอง',en:'First-time impression ratio, auction overlap, and Meta lifetime audience reached ratio appear only when Meta exposes them through the API; this system does not fabricate them.'}) }}</span></div>
              </template>
            </section>
            <div class="metric-detail-grid">
              <div><span>{{ l({lo:'ການສະແດງ',th:'การแสดงผล',en:'Impressions'}) }}</span><strong>{{ fmt(item.impressions, 0) }}</strong><small>{{ l({lo:'ຈຳນວນຄັ້ງທີ່ໂຄສະນາສະແດງ',th:'จำนวนครั้งที่โฆษณาแสดง',en:'Total ad displays'}) }}</small></div>
              <div><span>{{ l({lo:'ການເຂົ້າເຖິງ',th:'การเข้าถึง',en:'Reach'}) }}</span><strong>{{ fmt(item.reach, 0) }}</strong><small>{{ l({lo:'ຈຳນວນຄົນທີ່ເຫັນ',th:'จำนวนคนที่เห็น',en:'People who saw it'}) }}</small></div>
              <div><span>Link clicks</span><strong>{{ fmt(item.linkClicks, 0) }}</strong><small>{{ l({lo:'ຄລິກໄປປາຍທາງ',th:'คลิกไปยังปลายทาง',en:'Clicks to destination'}) }}</small></div>
              <div><span>CPC</span><strong>{{ money(item.linkCpc) }}</strong><small>{{ l({lo:'ຕົ້ນທຶນຕໍ່ຄລິກ',th:'ต้นทุนต่อคลิก',en:'Cost per link click'}) }}</small></div>
              <div><span>CPM</span><strong>{{ money(item.cpm) }}</strong><small>{{ l({lo:'ຕົ້ນທຶນຕໍ່ 1,000 ການສະແດງ',th:'ต้นทุนต่อ 1,000 การแสดงผล',en:'Cost per 1,000 impressions'}) }}</small></div>
              <div><span>{{ l({lo:'ມູນຄ່າ Conversion',th:'มูลค่า Conversion',en:'Conversion value'}) }}</span><strong>{{ money(item.conversionValue) }}</strong><small>{{ l({lo:'ລາຍຮັບທີ່ Meta attribution',th:'รายได้ที่ Meta attribution',en:'Meta-attributed value'}) }}</small></div>
              <div><span>ROAS</span><strong>{{ item.roas==null?'—':`${fmt(item.roas,2)}×` }}</strong><small>{{ breakEvenROAS?`Break-even ${fmt(breakEvenROAS,2)}×`:'Set gross margin' }}</small></div>
              <div><span>{{ l({lo:'ຫຼັງ Margin − Ads',th:'หลัง Margin − Ads',en:'Margin value − ads'}) }}</span><strong>{{ money(contribution(item)) }}</strong><small>{{ l({lo:'ບໍ່ລວມຄ່າດຳເນີນງານ',th:'ไม่รวมค่าใช้จ่ายดำเนินงาน',en:'Excludes operating costs'}) }}</small></div>
              <div v-if="item.entityType==='ad'"><span>Quality ranking</span><strong>{{ item.qualityRanking || '—' }}</strong><small>{{ item.engagementRateRanking || '—' }} · {{ item.conversionRateRanking || '—' }}</small></div>
            </div>
            <div class="status-actions">
              <div>
                <strong>{{ l({lo:'ສະຖານະການສົ່ງ',th:'สถานะการส่ง',en:'Delivery status'}) }}</strong>
                <span>{{ statusLabel(statusFor(item)) }}</span>
              </div>
              <v-btn
                color="primary"
                variant="flat"
                prepend-icon="mdi-cash-fast"
                class="font-weight-bold btn-scale-budget"
                @click="openBudgetModal(item)"
              >
                {{ l({ lo: 'ປັບງົບ / Scale ງົບ', th: 'ปรับงบ / Scale งบ', en: 'Scale Budget' }) }}
              </v-btn>
              <v-btn color="success" variant="tonal" prepend-icon="mdi-play" :loading="updatingId===item.entityId" :disabled="statusFor(item)==='ACTIVE'" @click="changeStatus(item,'ACTIVE')">{{ l({lo:'ເປີດໃຊ້ງານ',th:'เปิดใช้งาน',en:'Activate'}) }}</v-btn>
              <v-btn color="error" variant="outlined" prepend-icon="mdi-pause" :loading="updatingId===item.entityId" :disabled="statusFor(item)==='PAUSED'" @click="changeStatus(item,'PAUSED')">{{ l({lo:'ປິດຊົ່ວຄາວ',th:'หยุดชั่วคราว',en:'Pause'}) }}</v-btn>
            </div>
          </div>
        </article>
      </div>
    </section>
  </template>

  <v-card v-else-if="!loadingAccounts" class="empty-state" variant="outlined"><v-icon icon="mdi-facebook" size="48" /><h2>{{ l({lo:'ຍັງບໍ່ມີບັນຊີໂຄສະນາ',th:'ยังไม่มีบัญชีโฆษณา',en:'No ad account connected'}) }}</h2><p>{{ l({lo:'ເຊື່ອມ Facebook ແລະອະນຸຍາດ ads_read ເພື່ອເລີ່ມຕົ້ນ.',th:'เชื่อม Facebook และอนุญาต ads_read เพื่อเริ่มต้น',en:'Connect Facebook and allow ads_read to get started.'}) }}</p><v-btn color="primary" to="/meta-pages">{{ l({lo:'ເຊື່ອມຕໍ່ Meta',th:'เชื่อมต่อ Meta',en:'Connect Meta'}) }}</v-btn></v-card>

  <v-dialog v-model="actionPlanOpen" max-width="760" persistent>
    <v-card rounded="xl">
      <v-card-title class="dialog-title"><div><small>SAFE AUTOMATION</small><h2>{{ l({lo:'ກວດ ແລະອະນຸມັດແຜນປິດ Ads',th:'ตรวจและอนุมัติแผนหยุด Ads',en:'Review and approve pause plan'}) }}</h2></div><v-btn icon="mdi-close" variant="text" :disabled="applyingActionPlan" @click="actionPlanOpen=false" /></v-card-title>
      <v-card-text class="settings-body">
        <v-alert type="warning" variant="tonal" density="compact" class="mb-4"><strong>{{ l({lo:'Dry run:',th:'โหมดตรวจสอบ:',en:'Dry run:'}) }}</strong> {{ l({lo:'ຍັງບໍ່ມີ Ads ໃດຖືກປິດ. ເລືອກລາຍການ ແລະຢືນຢັນກ່ອນທຸກຄັ້ງ.',th:'ยังไม่มี Ads ใดถูกหยุด เลือกรายการและยืนยันก่อนทุกครั้ง',en:'Nothing has changed. Select items and explicitly approve before any ad is paused.'}) }}</v-alert>
        <div v-if="actionCandidates.length" class="action-plan-list">
          <label v-for="item in actionCandidates" :key="item.entityId" class="action-plan-item">
            <v-checkbox v-model="selectedActionIds" :value="item.entityId" hide-details color="error" />
            <span><strong>{{ item.entityName }}</strong><small>{{ statusLabel(statusFor(item)) }} · {{ money(item.spend) }} · {{ fmt(item.impressions,0) }} impressions</small><em>{{ decision(item).signals.join(' · ') }}</em></span>
          </label>
        </div>
        <div v-else class="empty-inline compact"><v-icon icon="mdi-check-decagram-outline" size="38" color="success" /><h3>{{ l({lo:'ບໍ່ມີ Ads ທີ່ເຂົ້າເກນປິດ',th:'ไม่มี Ads ที่เข้าเกณฑ์หยุด',en:'No ads meet the pause criteria'}) }}</h3><p>{{ l({lo:'ລະບົບຈະພິຈາລະນາສະເພາະ Ads ທີ່ເປີດຢູ່, ມີຂໍ້ມູນພໍ ແລະມີຢ່າງໜ້ອຍ 2 ສັນຍານອ່ອນ.',th:'ระบบพิจารณาเฉพาะ Ads ที่ทำงานอยู่ มีข้อมูลพอ และมีสัญญาณอ่อนอย่างน้อย 2 ข้อ',en:'Only active ads with enough data and at least two weak signals are considered.'}) }}</p></div>
        <v-checkbox v-if="actionCandidates.length" v-model="actionPlanAcknowledged" color="primary" :disabled="!selectedActionItems.length" :label="l({lo:`ຂ້ອຍໄດ້ກວດ ${selectedActionItems.length} ລາຍການ ແລະອະນຸມັດໃຫ້ປິດ`,th:`ฉันตรวจ ${selectedActionItems.length} รายการและอนุมัติให้หยุด`,en:`I reviewed ${selectedActionItems.length} item(s) and approve pausing them`})" />
      </v-card-text>
      <v-card-actions class="dialog-actions"><v-btn variant="text" :disabled="applyingActionPlan" @click="actionPlanOpen=false">{{ l({lo:'ຍົກເລີກ',th:'ยกเลิก',en:'Cancel'}) }}</v-btn><v-spacer /><v-btn color="error" prepend-icon="mdi-pause" :loading="applyingActionPlan" :disabled="!actionPlanAcknowledged || !selectedActionItems.length" @click="applyActionPlan">{{ l({lo:`ຢືນຢັນປິດ ${selectedActionItems.length} ລາຍການ`,th:`ยืนยันหยุด ${selectedActionItems.length} รายการ`,en:`Approve ${selectedActionItems.length} pause(s)`}) }}</v-btn></v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="settingsOpen" max-width="620">
    <v-card rounded="xl"><v-card-title class="dialog-title"><div><small>DECISION SETTINGS</small><h2>{{ l({lo:'ກຳນົດເກນຕາມຮ້ານ',th:'กำหนดเกณฑ์ตามร้าน',en:'Store-specific thresholds'}) }}</h2></div><v-btn icon="mdi-close" variant="text" @click="settingsOpen=false" /></v-card-title><v-card-text class="settings-body"><v-alert type="info" variant="tonal" density="compact" class="mb-4">{{ l({lo:'ແຕ່ລະຮ້ານມີກຳໄລ ແລະເປົ້າໝາຍບໍ່ຄືກັນ. ລະບົບຈະບໍ່ hard-code ຄຳຕັດສິນ.',th:'แต่ละร้านมีกำไรและเป้าหมายต่างกัน ระบบจะไม่ตัดสินด้วยค่าตายตัว',en:'Every store has different margins and goals; recommendations are not hard-coded.'}) }}</v-alert><div class="settings-grid primary-settings"><v-text-field v-model.number="targetCost" type="number" min="0" step="0.01" :label="l({lo:'ຕົ້ນທຶນ/ຜົນລັບສູງສຸດ',th:'ต้นทุน/ผลลัพธ์สูงสุด',en:'Maximum cost per result'})" :suffix="currency" variant="outlined" /><v-text-field v-model.number="grossMargin" type="number" min="0" max="100" step="1" :label="l({lo:'Gross margin ຂອງສິນຄ້າ',th:'Gross margin ของสินค้า',en:'Product gross margin'})" suffix="%" variant="outlined" :hint="breakEvenROAS?`Break-even ROAS ${fmt(breakEvenROAS,2)}×`:''" persistent-hint /></div><div class="settings-grid"><v-text-field v-model.number="weakCTR" type="number" min="0" step="0.1" label="Weak CTR below" suffix="%" variant="outlined" /><v-text-field v-model.number="goodCTR" type="number" min="0" step="0.1" label="Good CTR from" suffix="%" variant="outlined" /><v-text-field v-model.number="maxFrequency" type="number" min="1" step="0.1" label="Frequency warning from" variant="outlined" /><v-text-field v-model.number="minimumPlanImpressions" type="number" min="0" step="100" :label="l({lo:'Impressions ຕ່ຳສຸດສຳລັບແຜນ',th:'Impressions ขั้นต่ำสำหรับแผน',en:'Minimum plan impressions'})" variant="outlined" /><v-text-field v-model.number="minimumPlanSpend" type="number" min="0" step="0.01" :label="l({lo:'Spend ຕ່ຳສຸດສຳລັບແຜນ',th:'Spend ขั้นต่ำสำหรับแผน',en:'Minimum plan spend'})" :suffix="currency" variant="outlined" /></div></v-card-text><v-card-actions class="dialog-actions"><v-btn variant="text" @click="targetCost=null;grossMargin=null;goodCTR=3;weakCTR=1;maxFrequency=3;minimumPlanImpressions=1000;minimumPlanSpend=null">{{ l({lo:'ຄ່າເລີ່ມຕົ້ນ',th:'ค่าเริ่มต้น',en:'Reset'}) }}</v-btn><v-spacer /><v-btn color="primary" @click="saveThresholds">{{ l({lo:'ບັນທຶກ',th:'บันทึก',en:'Save settings'}) }}</v-btn></v-card-actions></v-card>
  </v-dialog>

  <!-- Scale Budget Modal -->
  <v-dialog v-model="budgetModalOpen" max-width="520" persistent>
    <v-card rounded="xl" class="budget-dialog-card">
      <v-card-title class="dialog-title">
        <div>
          <small class="text-primary font-weight-bold letter-spacing-1">BUDGET SCALING & CONTROL</small>
          <h2>{{ l({ lo: 'ປັບງົບປະມານໂຄສະນາ', th: 'ปรับงบประมาณโฆษณา', en: 'Scale / Adjust Ad Budget' }) }}</h2>
        </div>
        <v-btn icon="mdi-close" variant="text" :disabled="updatingBudget" @click="budgetModalOpen = false" />
      </v-card-title>

      <v-card-text class="settings-body">
        <!-- Target Info Card -->
        <div class="target-entity-card mb-4">
          <div class="d-flex align-center justify-space-between gap-2 mb-1">
            <span class="entity-type-badge text-uppercase">{{ targetBudgetItem?.entityType }}</span>
            <span v-if="targetBudgetItem" class="delivery-chip compact" :class="statusClass(statusFor(targetBudgetItem))">
              <span class="status-dot" :class="statusClass(statusFor(targetBudgetItem))" />
              {{ statusLabel(statusFor(targetBudgetItem)) }}
            </span>
          </div>
          <h3 class="entity-name-display">{{ targetBudgetItem?.entityName }}</h3>
          <p v-if="targetBudgetItem?.entityType === 'ad'" class="ad-parent-note">
            <v-icon icon="mdi-information-outline" size="14" />
            {{ l({
              lo: `Meta ກຳນົດງົບຢູ່ລະດັບ Ad Set: ຈະປັບງົບໃຫ້ Ad Set “${targetBudgetItem?.adSetName || targetBudgetItem?.campaignName}”`,
              th: `Meta กำหนดงบที่ระดับ Ad Set: จะปรับงบให้ Ad Set “${targetBudgetItem?.adSetName || targetBudgetItem?.campaignName}”`,
              en: `Meta controls budget at Ad Set level: will update Ad Set “${targetBudgetItem?.adSetName || targetBudgetItem?.campaignName}”`,
            }) }}
          </p>
          <div class="d-flex align-center gap-3 text-caption text-medium-emphasis mt-2">
            <span>{{ l({ lo: 'ໃຊ້ຈ່າຍສະເລ່ຍຊ່ວງນີ້:', th: 'ใช้จ่ายเฉลี่ยช่วงนี้:', en: 'Recent spend:' }) }} <b>{{ money(targetBudgetItem?.spend) }}</b></span>
          </div>
        </div>

        <!-- Quick Percentage Buttons (Pills) -->
        <div class="mb-4">
          <div class="d-flex align-center justify-space-between mb-2">
            <label class="text-caption font-weight-bold text-uppercase text-medium-emphasis">
              {{ l({ lo: 'ເລືອກເປີເຊັນເພີ່ມ/ຫຼຸດງົບ (Quick Scale)', th: 'เลือกเปอร์เซ็นต์เพิ่ม/ลดงบ', en: 'Quick Scale Percentage' }) }}
            </label>
            <span class="text-caption text-primary font-weight-medium">
              {{ l({ lo: 'ປອດໄພສຸດ: +10% ຫາ +20%', th: 'ปลอดภัยสุด: +10% ถึง +20%', en: 'Safest: +10% to +20%' }) }}
            </span>
          </div>

          <!-- Scale Up Pills -->
          <div class="scale-pills-row mb-2">
            <span class="pill-group-label text-success">
              <v-icon icon="mdi-arrow-up-bold" size="12" /> {{ l({ lo: 'ເພີ່ມ', th: 'เพิ่ม', en: 'Up' }) }}
            </span>
            <v-btn
              v-for="pct in [10, 15, 20, 30, 50]"
              :key="`up-${pct}`"
              size="small"
              :variant="selectedPercent === pct ? 'flat' : 'outlined'"
              :color="selectedPercent === pct ? 'success' : undefined"
              class="scale-pill-btn"
              :class="{ 'active-pill': selectedPercent === pct }"
              @click="setBudgetPercent(pct)"
            >
              +{{ pct }}%
            </v-btn>
          </div>

          <!-- Scale Down Pills -->
          <div class="scale-pills-row">
            <span class="pill-group-label text-warning">
              <v-icon icon="mdi-arrow-down-bold" size="12" /> {{ l({ lo: 'ຫຼຸດ', th: 'ลด', en: 'Down' }) }}
            </span>
            <v-btn
              v-for="pct in [-10, -20, -30, -50]"
              :key="`down-${pct}`"
              size="small"
              :variant="selectedPercent === pct ? 'flat' : 'outlined'"
              :color="selectedPercent === pct ? 'warning' : undefined"
              class="scale-pill-btn"
              :class="{ 'active-pill': selectedPercent === pct }"
              @click="setBudgetPercent(pct)"
            >
              {{ pct }}%
            </v-btn>
            <v-btn
              size="small"
              variant="text"
              class="scale-pill-btn"
              @click="setBudgetPercent(0)"
            >
              {{ l({ lo: 'ຄືນຄ່າ', th: 'รีเซ็ต', en: 'Reset' }) }}
            </v-btn>
          </div>
        </div>

        <!-- Before & After Comparison Card -->
        <div class="budget-comparison-box mb-4">
          <div class="comp-col">
            <span class="comp-label">{{ l({ lo: 'ງົບປັດຈຸບັນ (Baseline)', th: 'งบปัจจุบัน', en: 'Current Baseline' }) }}</span>
            <strong class="comp-value">{{ money(baseDailyBudget) }}<small>/ມື້</small></strong>
          </div>
          <div class="comp-arrow">
            <v-icon icon="mdi-arrow-right-bold" color="primary" />
          </div>
          <div class="comp-col highlight">
            <span class="comp-label">{{ l({ lo: 'ງົບໃໝ່ທີ່ຕ້ອງການຕັ້ງ', th: 'งบประมาณใหม่', en: 'New Budget' }) }}</span>
            <strong class="comp-value text-primary">{{ money(newDailyBudget) }}<small>/ມື້</small></strong>
          </div>
        </div>

        <!-- Difference Badge -->
        <div class="d-flex align-center justify-space-between px-2 mb-4">
          <span class="text-caption text-medium-emphasis">{{ l({ lo: 'ສ່ວນຕ່າງງົບປະມານ:', th: 'ส่วนต่างงบประมาณ:', en: 'Budget Difference:' }) }}</span>
          <v-chip
            size="small"
            :color="budgetDiff > 0 ? 'success' : budgetDiff < 0 ? 'warning' : 'default'"
            variant="flat"
            class="font-weight-bold"
          >
            {{ budgetDiff > 0 ? '+' : '' }}{{ money(budgetDiff) }}/ມື້ ({{ budgetDiff > 0 ? '+' : '' }}{{ budgetDiffPercent }}%)
          </v-chip>
        </div>

        <!-- Custom Input Field -->
        <v-text-field
          v-model.number="newDailyBudget"
          type="number"
          :label="l({ lo: 'ກຳນົດງົບປະມານຕໍ່ມື້ (Daily Budget)', th: 'ระบุงบประมาณต่อวัน', en: 'Daily Budget' })"
          :suffix="currency"
          variant="outlined"
          density="comfortable"
          :hint="l({ lo: 'ສາມາດພິມຕົວເລກທີ່ຕ້ອງການປັບເອງໄດ້ໂດຍກົງ', th: 'สามารถพิมพ์ตัวเลขที่ต้องการระบุได้โดยตรง', en: 'You can type any exact custom amount' })"
          persistent-hint
          @input="selectedPercent = null"
        />

        <!-- Best Practice Advisory Alert -->
        <v-alert
          type="info"
          variant="tonal"
          density="compact"
          class="mt-4"
          icon="mdi-lightbulb-on-outline"
        >
          <div class="text-caption">
            <strong>{{ l({ lo: 'ຄຳແນະນຳຈາກ Algorithm ຂອງ Meta:', th: 'คำแนะนำจาก Algorithm ของ Meta:', en: 'Meta Algorithm Tip:' }) }}</strong>
            {{ l({
              lo: 'ການປັບເພີ່ມງົບ 10% ຫາ 20% ຕໍ່ຄັ້ງ ແມ່ນວິທີທີ່ປອດໄພທີ່ສຸດ ເພື່ອປ້ອງກັນບໍ່ໃຫ້ Meta Reset Learning Phase.',
              th: 'การปรับเพิ่มงบ 10% ถึง 20% ต่อครั้ง เป็นวิธีที่ปลอดภัยที่สุด เพื่อไม่ให้ Meta รีเซ็ต Learning Phase',
              en: 'Adjusting budget within 10% to 20% is the safest way to avoid triggering a Learning Phase reset.',
            }) }}
          </div>
        </v-alert>
      </v-card-text>

      <v-card-actions class="dialog-actions">
        <v-btn variant="text" :disabled="updatingBudget" @click="budgetModalOpen = false">
          {{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}
        </v-btn>
        <v-spacer />
        <v-btn
          color="primary"
          variant="flat"
          :loading="updatingBudget"
          :disabled="!newDailyBudget || newDailyBudget <= 0"
          prepend-icon="mdi-check"
          class="font-weight-bold px-4"
          @click="applyBudgetChange"
        >
          {{ l({ lo: 'ຢືນຢັນປັບງົບ', th: 'ยืนยันปรับงบ', en: 'Apply Budget' }) }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.page-head,.filter-bar,.performance-section{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
.page-head{padding:24px;display:flex;justify-content:space-between;align-items:flex-start;gap:24px;margin-bottom:16px}.page-head p,.card-heading p,.section-head p{margin:0 0 6px;color:var(--color-primary);font-size:11px;font-weight:800;letter-spacing:.13em}.page-head h1{font-size:clamp(24px,3vw,34px);line-height:1.15;margin:0 0 8px}.page-head span{color:var(--color-text-secondary)}.head-actions{display:flex;gap:8px;flex-wrap:wrap}
.filter-bar{display:grid;grid-template-columns:minmax(240px,1.5fr) minmax(170px,.8fr) minmax(145px,.7fr) minmax(145px,.7fr) auto;align-items:center;gap:12px;padding:16px;margin-bottom:16px}
.loading-grid,.summary-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-bottom:16px}.summary-card{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);padding:18px;display:grid;gap:5px}.summary-card>span{color:var(--color-text-secondary);font-size:13px}.summary-card strong{font-size:clamp(22px,3vw,30px);line-height:1.1}.summary-card small{color:var(--color-text-muted);font-size:11px}.summary-card.finance{border-color:rgba(32,164,100,.28);background:linear-gradient(145deg,var(--color-surface),rgba(32,164,100,.045))}.summary-card.finance strong{color:#167a4b}.summary-card.loss{border-color:rgba(239,68,68,.3);background:linear-gradient(145deg,var(--color-surface),rgba(239,68,68,.04))}.summary-card.loss strong{color:var(--color-error)}
.analysis-grid{display:grid;grid-template-columns:minmax(0,1.6fr) minmax(320px,.8fr);gap:16px;margin-bottom:16px}.trend-card,.decision-card{padding:20px;border-radius:var(--radius-lg)}.card-heading,.section-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px}.card-heading h2,.section-head h2{margin:0;font-size:20px}.card-heading>small{color:var(--color-text-muted)}
.trend-chart{height:220px;display:flex;align-items:flex-end;gap:clamp(4px,1vw,12px);padding:28px 4px 24px;margin-top:12px;border-bottom:1px solid var(--color-border);position:relative}.trend-column{height:100%;flex:1;min-width:8px;display:flex;flex-direction:column;justify-content:flex-end;align-items:center;gap:7px;position:relative;outline:none}.trend-bar{width:min(34px,80%);min-height:4px;background:linear-gradient(180deg,var(--color-primary),#7c6cf1);border-radius:7px 7px 3px 3px;transition:transform .15s}.trend-column:is(:hover,:focus) .trend-bar{transform:scaleX(1.08)}.trend-column small{font-size:9px;color:var(--color-text-muted);white-space:nowrap}.trend-tooltip{display:none;position:absolute;bottom:calc(100% + 6px);z-index:2;background:#211d3f;color:#fff;padding:8px 10px;border-radius:8px;font-size:11px;white-space:nowrap;text-align:center}.trend-column:is(:hover,:focus) .trend-tooltip{display:block}
.decision-summary{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:18px 0}.decision-count{padding:14px 8px;border-radius:12px;display:grid;text-align:center}.decision-count strong{font-size:25px}.decision-count span{font-size:11px}.decision-count.keep{background:rgba(39,174,96,.1);color:var(--color-success)}.decision-count.review{background:rgba(245,158,11,.12);color:#a86700}.decision-count.pause{background:rgba(239,68,68,.1);color:var(--color-error)}.guide-note{display:flex;gap:8px;align-items:flex-start;color:var(--color-text-secondary);font-size:13px;line-height:1.55;margin:0}.decision-steps{display:grid;gap:9px;margin:15px 0 0;padding:0;list-style:none}.decision-steps li{display:grid;padding:10px 12px;border:1px solid var(--color-border);border-radius:10px}.decision-steps strong{font-size:11px}.decision-steps span{color:var(--color-text-secondary);font-size:10px;line-height:1.45}
.performance-section{padding:20px}.section-head{margin-bottom:14px}.report-meta{display:flex;align-items:center;gap:12px}.report-meta>span{font-size:11px;color:var(--color-text-muted)}.report-meta .v-text-field{width:250px}.detail-toolbar{display:flex;align-items:flex-end;justify-content:space-between;gap:16px;padding:14px;margin-bottom:8px;border:1px solid var(--color-border);border-radius:14px;background:var(--color-surface-soft)}.toolbar-group{display:grid;gap:6px;min-width:0}.toolbar-group>small{font-size:10px;font-weight:750;color:var(--color-text-muted);text-transform:uppercase;letter-spacing:.08em}.level-tabs,.status-tabs{max-width:100%}.level-tabs .v-btn,.status-tabs .v-btn{min-height:40px}.level-tabs .v-icon{margin-right:5px}.status-tabs .v-btn{gap:6px}.status-tabs b{min-width:20px;padding:1px 6px;border-radius:999px;background:color-mix(in srgb,currentColor 12%,transparent);font-size:10px}.status-dot{width:8px;height:8px;border-radius:50%;display:inline-block;flex:0 0 8px}.status-dot.active{background:#20a464}.status-dot.paused{background:#8991a5}.status-dot.issues{background:#e58a18}.filter-help{display:flex;align-items:flex-start;gap:7px;margin:0 2px 16px;color:var(--color-text-muted);font-size:11px;line-height:1.45}.table-loading{margin:-1px 0 16px}.performance-list{display:grid;border:1px solid var(--color-border);border-radius:14px;overflow:hidden}.performance-row+.performance-row{border-top:1px solid var(--color-border)}.performance-row.expanded{background:var(--color-surface-soft)}.row-main{width:100%;display:grid;grid-template-columns:minmax(230px,1.5fr) minmax(110px,.6fr) repeat(5,minmax(78px,.5fr)) minmax(175px,1fr) 28px;align-items:center;gap:10px;padding:14px 16px;text-align:left;color:inherit;background:transparent;border:0;cursor:pointer}.row-main:hover{background:var(--color-background)}.entity-cell{display:flex;align-items:center;gap:11px;min-width:0}.entity-cell>span:last-child{display:grid;min-width:0}.entity-cell strong,.entity-cell small,.entity-cell code{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.entity-cell small{color:var(--color-text-secondary);font-size:11px}.entity-cell code{color:var(--color-text-muted);font-size:9px}.entity-icon{width:42px;height:42px;display:grid;place-items:center;border-radius:11px;background:var(--color-primary-soft);color:var(--color-primary);flex:0 0 42px}.delivery-chip{display:inline-flex;align-items:center;gap:6px;width:max-content;max-width:100%;padding:6px 8px;border-radius:999px;font-size:10px;font-weight:750;white-space:nowrap}.delivery-chip.active{color:#167a4b;background:rgba(32,164,100,.1)}.delivery-chip.paused{color:#697184;background:rgba(137,145,165,.13)}.delivery-chip.issues{color:#a96000;background:rgba(229,138,24,.12)}.metric{display:grid;gap:3px}.metric small{font-size:10px;color:var(--color-text-muted)}.metric strong{font-size:14px}.decision-chip{display:flex;align-items:center;gap:8px;border-radius:11px;padding:9px 10px;min-width:0}.decision-chip>span{display:grid;min-width:0}.decision-chip strong{font-size:12px}.decision-chip small{font-size:9px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.decision-chip.keep{background:rgba(39,174,96,.1);color:var(--color-success)}.decision-chip.review{background:rgba(245,158,11,.12);color:#a86700}.decision-chip.pause{background:rgba(239,68,68,.1);color:var(--color-error)}.decision-chip.no_data{background:var(--color-surface-soft);color:var(--color-text-secondary)}
.row-detail{padding:20px;border-top:1px solid var(--color-border);display:grid;gap:18px}.explanation{display:flex;gap:12px;align-items:flex-start;max-width:850px}.explanation h3{margin:0 0 4px;font-size:17px}.explanation p{margin:0;color:var(--color-text-secondary)}.explanation ul{margin:8px 0 0;padding-left:20px;color:var(--color-text-secondary)}.audience-callout{display:flex;align-items:flex-start;gap:11px;padding:14px;color:var(--color-primary);background:var(--color-primary-soft);border-radius:12px}.audience-callout div{display:grid}.audience-callout strong{font-size:13px}.audience-callout span{color:var(--color-text-secondary);font-size:11px}.audience-panel{display:grid;gap:14px;padding:18px;border:1px solid color-mix(in srgb,var(--color-primary) 22%,var(--color-border));border-radius:14px;background:var(--color-surface)}.audience-heading{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}.audience-heading p{margin:0 0 3px;color:var(--color-primary);font-size:9px;font-weight:800;letter-spacing:.12em}.audience-heading h3{margin:0;font-size:17px}.audience-heading>small{padding:5px 8px;color:var(--color-primary);background:var(--color-primary-soft);border-radius:999px;font-size:9px}.audience-metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:9px}.audience-metrics>div{display:grid;gap:3px;padding:12px;background:var(--color-background);border-radius:11px}.audience-metrics span,.audience-metrics small{color:var(--color-text-muted);font-size:9px}.audience-metrics strong{font-size:19px}.coverage-metric strong{color:var(--color-primary)}.coverage-bar{margin-top:-5px}.audience-advice{display:flex;align-items:flex-start;gap:10px;padding:12px;border-radius:11px}.audience-advice>div{display:grid}.audience-advice strong{font-size:12px}.audience-advice span{font-size:10px;line-height:1.5}.audience-advice.success{color:#167a4b;background:rgba(32,164,100,.1)}.audience-advice.warning{color:#995c00;background:rgba(229,138,24,.12)}.audience-advice.info{color:var(--color-primary);background:var(--color-primary-soft)}.targeting-block{display:grid;gap:8px}.targeting-block>strong{font-size:12px}.targeting-block>small{color:var(--color-text-muted)}.targeting-tags{display:flex;flex-wrap:wrap;gap:6px}.targeting-tags span{display:inline-flex;align-items:center;gap:5px;padding:6px 8px;border:1px solid var(--color-border);border-radius:999px;color:var(--color-text-secondary);font-size:10px}.native-metric-note{display:flex;align-items:flex-start;gap:8px;padding-top:11px;border-top:1px solid var(--color-border);color:var(--color-text-muted);font-size:9px;line-height:1.5}.metric-detail-grid{display:grid;grid-template-columns:repeat(6,1fr);gap:10px}.metric-detail-grid>div{padding:12px;border:1px solid var(--color-border);border-radius:11px;display:grid;gap:3px}.metric-detail-grid span,.metric-detail-grid small{font-size:10px;color:var(--color-text-muted)}.metric-detail-grid strong{font-size:15px;overflow-wrap:anywhere}.status-actions{display:flex;align-items:center;justify-content:flex-end;gap:8px;padding-top:14px;border-top:1px solid var(--color-border)}.status-actions>div{display:grid;margin-right:auto}.status-actions>div span{font-size:12px;color:var(--color-text-secondary)}
.empty-state,.empty-inline{text-align:center;padding:48px 20px;color:var(--color-text-secondary)}.empty-inline.compact{padding:24px 16px}.empty-state h2,.empty-state h3,.empty-inline h3{color:var(--color-text);margin:12px 0 6px}.empty-state p,.empty-inline p{margin:0 0 16px}.dialog-title{display:flex;justify-content:space-between;align-items:flex-start;padding:22px}.dialog-title small{color:var(--color-primary);font-size:10px;letter-spacing:.12em}.dialog-title h2{margin:3px 0 0;font-size:22px}.settings-body{padding:0 22px 12px}.settings-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-top:16px}.settings-grid.primary-settings{grid-template-columns:repeat(2,minmax(0,1fr));margin-top:0}.dialog-actions{padding:12px 22px 22px}.action-plan-list{display:grid;gap:8px;max-height:420px;overflow:auto}.action-plan-item{display:grid;grid-template-columns:auto 1fr;align-items:flex-start;padding:10px 12px;border:1px solid var(--color-border);border-radius:12px;cursor:pointer}.action-plan-item:hover{background:var(--color-surface-soft)}.action-plan-item>span{display:grid;gap:3px;padding-top:8px;min-width:0}.action-plan-item small{color:var(--color-text-muted)}.action-plan-item em{color:var(--color-error);font-size:11px;font-style:normal;line-height:1.4}
.filter-account-select { min-width: 220px; }
.filter-product-select { min-width: 200px; }
.filter-date-select { min-width: 140px; }
.filter-custom-date { max-width: 130px; }

.delivery-wrap{display:inline-flex;align-items:center;gap:6px}
.quick-scale-btn{height:26px !important;padding-inline:8px !important;font-size:11px !important;letter-spacing:0}
.btn-scale-budget{font-size:13px}

.target-entity-card{background:var(--color-surface-soft);border:1px solid var(--color-border);border-radius:12px;padding:12px 14px}
.entity-type-badge{font-size:10px;font-weight:800;letter-spacing:.08em;color:var(--color-primary);background:var(--color-primary-soft);padding:3px 7px;border-radius:6px}
.entity-name-display{font-size:16px;font-weight:700;margin:6px 0 2px;line-height:1.3}
.ad-parent-note{font-size:11.5px;color:var(--color-primary);margin:4px 0 0;display:flex;align-items:center;gap:4px}
.scale-pills-row{display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.pill-group-label{font-size:11px;font-weight:800;min-width:44px;display:inline-flex;align-items:center;gap:2px}
.scale-pill-btn{min-width:52px;height:32px;font-weight:700;border-radius:8px;font-size:12px}
.budget-comparison-box{display:flex;align-items:center;justify-content:space-between;padding:12px 16px;border-radius:12px;background:color-mix(in srgb,var(--color-primary) 6%,var(--color-surface));border:1px dashed color-mix(in srgb,var(--color-primary) 30%,var(--color-border))}
.comp-col{display:grid;gap:2px}
.comp-label{font-size:10.5px;color:var(--color-text-secondary);font-weight:600}
.comp-value{font-size:17px;font-weight:800}
.comp-value small{font-size:11px;font-weight:500;color:var(--color-text-muted);margin-left:2px}
.comp-arrow{display:flex;align-items:center;justify-content:center}

@media(max-width:1280px){
  .filter-bar{grid-template-columns:repeat(2,minmax(0,1fr))}
  .analysis-grid{grid-template-columns:1fr}
  .detail-toolbar{align-items:stretch;flex-direction:column}
  .row-main{grid-template-columns:minmax(220px,1.4fr) minmax(105px,.65fr) repeat(3,minmax(78px,.55fr)) minmax(170px,1fr) 28px}
  .metric-cost,.metric-frequency{display:none}
  .audience-metrics{grid-template-columns:repeat(2,minmax(0,1fr))}
  .metric-detail-grid{grid-template-columns:repeat(3,1fr)}
}

@media(max-width:720px){
  .page-head{padding:16px;display:grid}
  .head-actions,.head-actions .v-btn{width:100%}
  .filter-account-select, .filter-product-select, .filter-date-select, .filter-custom-date {
    width: 100% !important;
    min-width: 0 !important;
    max-width: 100% !important;
  }
  .performance-section{padding:12px;border-radius:14px}
  .section-head,.report-meta{display:grid;width:100%}
  .report-meta .v-text-field, .report-meta .sort-select {width:100% !important; max-width:none !important;}
  .detail-toolbar{
    position: static;
    margin-inline: 0;
    border-radius: 12px;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .level-tabs{width:100%;display:grid;grid-template-columns:repeat(3,1fr)}
  .level-tabs .v-btn{min-width:0;padding-inline:6px;font-size:12px}
  .status-filter-group{width:100%;overflow:hidden}
  .status-tabs{display:flex;width:100%;overflow-x:auto;-webkit-overflow-scrolling:touch;padding-bottom:3px;scrollbar-width:none}
  .status-tabs::-webkit-scrollbar{display:none}
  .status-tabs .v-btn{flex:0 0 auto;padding-inline:10px;font-size:12px}
  .filter-help{margin-top:8px}

  .performance-list{border:0;gap:10px;overflow:visible;background:transparent}
  .performance-row,.performance-row+.performance-row{
    border:1px solid var(--color-border);
    border-radius:14px;
    overflow:hidden;
    background:var(--color-surface);
    margin-bottom:10px;
    box-shadow: 0 1px 3px rgba(0,0,0,0.04);
  }
  .row-main{
    display:grid;
    grid-template-columns:repeat(2,minmax(0,1fr));
    gap:8px;
    padding:14px;
    position:relative;
  }
  .entity-cell{
    grid-column:1 / -1;
    padding-right:32px;
    display:flex;
    align-items:flex-start;
    gap:10px;
    width:100%;
  }
  .expand-chevron, .row-main > .v-icon:last-child {
    position:absolute;
    top:16px;
    right:14px;
  }
  .delivery-wrap{
    grid-column:1;
    grid-row:2;
    justify-self:start;
    display:inline-flex;
    align-items:center;
    gap:6px;
  }
  .delivery-chip{
    font-size:10px;
  }
  .decision-chip{
    grid-column:2;
    grid-row:2;
    justify-self:end;
    padding:6px 8px;
    max-width:100%;
  }
  .decision-chip strong{font-size:11px}
  .decision-chip small{font-size:8.5px}

  .row-main>.metric{display:none}
  .row-main>.mobile-key{
    display:grid;
    min-width:0;
    padding:8px 10px;
    border-radius:9px;
    background:var(--color-background);
  }
  .row-main>.mobile-key small{font-size:9.5px}
  .row-main>.mobile-key strong{font-size:13.5px}
  .metric-spend{grid-column:1;grid-row:3}
  .metric-results{grid-column:2;grid-row:3}
  .metric-cost{grid-column:1;grid-row:4}
  .metric-ctr{grid-column:2;grid-row:4}

  .row-detail{padding:14px}
  .metric-detail-grid{grid-template-columns:repeat(2,minmax(0,1fr));gap:6px}
  .metric-detail-grid>div{padding:8px 10px}
  .metric-detail-grid strong{font-size:13.5px}

  /* Safe, non-sticky status actions on mobile that won't be covered by bottom nav */
  .status-actions{
    display:grid;
    grid-template-columns:1fr 1fr;
    position:relative;
    bottom:auto;
    padding:12px;
    border-radius:11px;
    background:var(--color-surface-soft);
    margin-top:12px;
    z-index:1;
    gap:8px;
  }
  .status-actions>div{grid-column:1/-1;margin-bottom:4px}
  .status-actions .btn-scale-budget{grid-column:1/-1}
  .status-actions .v-btn{width:100%;min-height:42px;font-size:13px;font-weight:700}
  .scale-pill-btn{min-width:44px;height:30px;font-size:11px;padding-inline:6px}
  .budget-comparison-box{padding:10px 12px}
  .comp-value{font-size:15px}

  .settings-grid,.settings-grid.primary-settings{grid-template-columns:1fr}
  .trend-card,.decision-card{padding:14px}
  .trend-chart{overflow-x:auto;justify-content:flex-start}
  .trend-column{flex:0 0 28px}
  .card-heading{display:grid}
}

@media(max-width:420px){
  .level-tabs .v-btn{font-size:11px}
  .level-tabs .v-icon{display:none}
  .audience-panel{padding:11px}
  .audience-heading{display:grid}
  .audience-metrics{grid-template-columns:1fr}
  .metric-detail-grid{grid-template-columns:1fr}
  .status-actions{grid-template-columns:1fr}
  .row-main>.mobile-key{padding:6px 8px}
  .dialog-actions{display:grid;grid-template-columns:1fr}
  .dialog-actions .v-spacer{display:none}
  .dialog-actions .v-btn{width:100%;margin:0}
}
</style>
