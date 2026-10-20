<script setup lang="ts">
import type { AutomationFlow, ItemsResponse, MetaAdAccount, MetaAudienceAnalysis, MetaCampaign, MetaInsightItem, MetaInsightsReport, Product } from '~/types/automation'
import { useAdVat } from '~/composables/useAdVat'
import { useProductAdStrategy, type ProductAdStrategy } from '~/composables/useProductAdStrategy'

definePageMeta({ middleware: 'auth' })
const { l } = useLocale()

type Level = 'campaign' | 'adset' | 'ad'
type DecisionCode = 'keep' | 'review' | 'pause' | 'no_data'
type StatusFilter = 'all' | 'active' | 'paused' | 'issues'
type OutcomeFilter = 'all' | 'angel' | 'purchases' | 'messages' | 'scale' | 'pause'

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
const outcomeFilter = ref<OutcomeFilter>('all')
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

// Dual-System Auto-Pilot & Scale Tracking Engine
interface AutoPilotConfig {
  enabled: boolean
  scope: 'all' | 'filtered_product'
  targetType: 'all' | 'campaign' | 'adset'
  autoScaleEnabled: boolean
  scalePercent: number
  scaleMinPurchases: number
  scaleMaxCPA: number | null
  scaleMaxFrequency: number
  scaleCooldownHours: number
  maxDailyBudgetCap: number | null
  autoStopEnabled: boolean
  stopMinSpend: number | null
  stopMaxCTR: number
}

interface AutoPilotLog {
  id: string
  timestamp: string
  action: 'SCALE' | 'PAUSE'
  entityId: string
  entityType: string
  entityName: string
  detail: string
  beforeValue?: string
  afterValue?: string
  success: boolean
}

interface ScaleRecord {
  entityId: string
  entityName: string
  entityType: string
  scaledAt: number
  oldBudget: number
  newBudget: number
  scalePercent: number
  purchasesAtScale: number
  scaledBy: 'AUTO' | 'MANUAL'
}

const autoPilotModalOpen = ref(false)
const autoPilotTab = ref<'rules' | 'logs'>('rules')
const runningAutoPilot = ref(false)
const autoPilotLogs = ref<AutoPilotLog[]>([])
const scaleHistory = ref<Record<string, ScaleRecord>>({})
const guideModalOpen = ref(false)
const guideStep = ref(1)
const isLiveTourActive = ref(false)
const liveTourStep = ref(1)

function startLiveTour() {
  guideModalOpen.value = false
  isLiveTourActive.value = true
  liveTourStep.value = 1
  nextTick(() => {
    scrollToTourTarget(1)
  })
}

function nextTourStep() {
  if (liveTourStep.value < 5) {
    liveTourStep.value++
    nextTick(() => {
      scrollToTourTarget(liveTourStep.value)
    })
  } else {
    endLiveTour()
  }
}

function prevTourStep() {
  if (liveTourStep.value > 1) {
    liveTourStep.value--
    nextTick(() => {
      scrollToTourTarget(liveTourStep.value)
    })
  }
}

function endLiveTour() {
  isLiveTourActive.value = false
  liveTourStep.value = 1
}

function scrollToTourTarget(step: number) {
  if (!import.meta.client) return
  let targetId = ''
  if (step === 2) targetId = 'tour-filter-bar'
  else if (step === 3) targetId = 'tour-kpi-summary'
  else if (step === 4) targetId = 'tour-mode-banner'
  else if (step === 5) targetId = 'tour-performance-section'

  if (targetId) {
    const el = document.getElementById(targetId)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  } else {
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}
const autoPilotConfig = ref<AutoPilotConfig>({
  enabled: false,
  scope: 'all',
  targetType: 'all',
  autoScaleEnabled: true,
  scalePercent: 20,
  scaleMinPurchases: 3,
  scaleMaxCPA: null,
  scaleMaxFrequency: 2.5,
  scaleCooldownHours: 24,
  maxDailyBudgetCap: null,
  autoStopEnabled: true,
  stopMinSpend: null,
  stopMaxCTR: 1.0,
})

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

const {
  productStrategies,
  loadStrategies: loadProductStrategiesFromStore,
  saveStrategies: saveProductStrategiesToStore,
} = useProductAdStrategy()
const route = useRoute()
const strategyTab = ref<'products' | 'default'>('products')
const selectedStrategyProductId = ref<string>('')
const targetCost = ref<number | null>(null)
const goodCTR = ref(3)
const weakCTR = ref(1)
const maxFrequency = ref(3)
const grossMargin = ref<number | null>(null)
const minimumPlanImpressions = ref(1000)
const minimumPlanSpend = ref<number | null>(null)
const sortBy = ref<'spend' | 'purchases' | 'messages' | 'roas' | 'newest' | 'name'>('spend')

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

  // Also include remaining products from store catalog
  for (const prod of products.value) {
    if (!countsByProduct[prod.id]) {
      list.push({
        title: `📦 ${prod.name} (0 ແອດ)`,
        value: prod.id,
      })
    }
  }

  if (unlinkedCount > 0) {
    list.push({
      title: l({ lo: `⚠️ ແອດທີ່ຍັງບໍ່ຜູກສິນຄ້າ (${unlinkedCount})`, th: `⚠️ แอดที่ยังไม่ผูกสินค้า (${unlinkedCount})`, en: `⚠️ Unlinked Ads (${unlinkedCount})` }),
      value: 'unlinked',
    })
  }

  return list
})

const productFilteredItems = computed(() => {
  const list = report.value?.items || []
  if (selectedProductId.value === 'all') return list
  return list.filter((item) => {
    const linked = getLinkedProduct(item)
    if (selectedProductId.value === 'unlinked') return !linked
    return linked?.product.id === selectedProductId.value
  })
})

const items = computed(() => {
  const query = search.value.trim().toLowerCase()
  return productFilteredItems.value
    .filter(item => !query || [item.entityName, item.campaignName, item.adSetName, item.adName, item.entityId].some(value => value?.toLowerCase().includes(query)))
    .filter((item) => {
      const status = (item.effectiveStatus || item.status || '').toUpperCase()
      if (statusFilter.value === 'active') return status === 'ACTIVE' || status === 'IN_PROCESS' || status === 'PENDING_REVIEW'
      if (statusFilter.value === 'paused') return status === 'PAUSED' || status === 'CAMPAIGN_PAUSED' || status === 'ADSET_PAUSED'
      if (statusFilter.value === 'issues') return !!status && status !== 'ACTIVE' && status !== 'IN_PROCESS' && status !== 'PENDING_REVIEW' && status !== 'PAUSED' && status !== 'CAMPAIGN_PAUSED' && status !== 'ADSET_PAUSED'
      return true
    })
    .filter((item) => {
      if (outcomeFilter.value === 'angel') return isAngelAd(item).isAngel
      if (outcomeFilter.value === 'purchases') return (getPurchasesCount(item) || 0) > 0
      if (outcomeFilter.value === 'messages') {
        return (getMessagesCount(item) || 0) > 0 || (isMessagingAction(item.results?.type) && (item.results?.value || 0) > 0)
      }
      if (outcomeFilter.value === 'scale') return isScaleCandidate(item).eligible
      if (outcomeFilter.value === 'pause') return decision(item).code === 'pause'
      return true
    })
    .sort((a, b) => {
      if (sortBy.value === 'purchases') {
        const pA = getPurchasesCount(a) || 0
        const pB = getPurchasesCount(b) || 0
        if (pA !== pB) return pB - pA
        return (b.spend ?? -1) - (a.spend ?? -1)
      }
      if (sortBy.value === 'messages') {
        const mA = getMessagesCount(a) || (isMessagingAction(a.results?.type) ? (a.results?.value || 0) : 0)
        const mB = getMessagesCount(b) || (isMessagingAction(b.results?.type) ? (b.results?.value || 0) : 0)
        if (mA !== mB) return mB - mA
        return (b.spend ?? -1) - (a.spend ?? -1)
      }
      if (sortBy.value === 'roas') {
        const rA = a.roas || 0
        const rB = b.roas || 0
        if (rA !== rB) return rB - rA
        return (b.spend ?? -1) - (a.spend ?? -1)
      }
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

const isFiltered = computed(() => {
  return selectedProductId.value !== 'all' || statusFilter.value !== 'all' || outcomeFilter.value !== 'all' || !!search.value.trim()
})

function isMessagingAction(type?: string): boolean {
  if (!type) return false
  const t = type.toLowerCase()
  return t.includes('messaging') || t.includes('conversation') || t.includes('reply') || t.includes('contact')
}

function getMessagesCount(item?: MetaInsightItem): number | undefined {
  if (!item) return undefined
  if (item.messagesCount != null && item.messagesCount > 0) return item.messagesCount
  if (isMessagingAction(item.results?.type) && item.results?.value != null && item.results.value > 0) return item.results.value
  return undefined
}

function getCostPerMessage(item?: MetaInsightItem): number | undefined {
  if (!item) return undefined
  if (item.costPerMessage != null && item.costPerMessage > 0) return item.costPerMessage
  if (isMessagingAction(item.results?.type) && item.results?.cost != null && item.results.cost > 0) return item.results.cost
  const count = getMessagesCount(item)
  if (count && count > 0 && item.spend && item.spend > 0) return item.spend / count
  return undefined
}

function getPurchasesCount(item?: MetaInsightItem): number | undefined {
  if (!item) return undefined
  if (item.purchasesCount != null && item.purchasesCount > 0) return item.purchasesCount
  if (item.results?.type?.toLowerCase().includes('purchase') && item.results?.value != null && item.results.value > 0) return item.results.value
  return undefined
}

function getCostPerPurchase(item?: MetaInsightItem): number | undefined {
  if (!item) return undefined
  if (item.costPerPurchase != null && item.costPerPurchase > 0) return item.costPerPurchase
  if (item.results?.type?.toLowerCase().includes('purchase') && item.results?.cost != null && item.results.cost > 0) return item.results.cost
  const count = getPurchasesCount(item)
  if (count && count > 0 && item.spend && item.spend > 0) return item.spend / count
  return undefined
}

const summary = computed<MetaInsightItem | undefined>(() => {
  let base: MetaInsightItem | undefined
  if (!isFiltered.value && report.value?.summary) {
    base = { ...report.value.summary }
  } else {
    const list = items.value
    if (!list.length) {
      return {
        entityType: 'account',
        entityId: 'filtered',
        entityName: 'Filtered',
        spend: 0,
        impressions: 0,
        reach: 0,
        frequency: 0,
        linkClicks: 0,
        linkCtr: 0,
        conversionValue: 0,
        roas: undefined,
        results: { type: report.value?.summary?.results?.type || 'purchase', value: 0, cost: 0 },
        messagesCount: 0,
        costPerMessage: 0,
        purchasesCount: 0,
        costPerPurchase: 0,
      }
    }

    let totalSpend = 0
    let totalImpressions = 0
    let totalReach = 0
    let totalLinkClicks = 0
    let totalResults = 0
    let totalConversionValue = 0
    let resultsType = report.value?.summary?.results?.type || 'purchase'

    for (const item of list) {
      if (item.spend) totalSpend += item.spend
      if (item.impressions) totalImpressions += item.impressions
      if (item.reach) totalReach += item.reach
      if (item.linkClicks) totalLinkClicks += item.linkClicks
      if (item.results?.value) {
        totalResults += item.results.value
        if (item.results.type) resultsType = item.results.type
      }
      if (item.conversionValue) totalConversionValue += item.conversionValue
    }

    const costPerResult = totalResults > 0 ? totalSpend / totalResults : (totalSpend > 0 ? 0 : undefined)
    const linkCtr = totalImpressions > 0 ? (totalLinkClicks / totalImpressions) * 100 : undefined
    const frequency = totalReach > 0 ? totalImpressions / totalReach : undefined
    const roas = totalSpend > 0 && totalConversionValue > 0 ? totalConversionValue / totalSpend : undefined

    base = {
      entityType: 'account',
      entityId: 'filtered',
      entityName: 'Filtered Summary',
      spend: totalSpend,
      impressions: totalImpressions,
      reach: totalReach,
      frequency,
      linkClicks: totalLinkClicks,
      linkCtr,
      conversionValue: totalConversionValue > 0 ? totalConversionValue : undefined,
      roas,
      results: {
        type: resultsType,
        value: totalResults,
        cost: costPerResult,
      },
    }
  }

  // Calculate sum of messages and purchases across items
  const targetList = isFiltered.value ? items.value : (report.value?.items || [])
  let sumMessages = 0
  let sumPurchases = 0
  for (const item of targetList) {
    const m = getMessagesCount(item)
    if (m) sumMessages += m
    const p = getPurchasesCount(item)
    if (p) sumPurchases += p
  }

  if (base.messagesCount == null || base.messagesCount === 0) {
    base.messagesCount = sumMessages > 0 ? sumMessages : undefined
  }
  if (base.costPerMessage == null && base.messagesCount && base.spend) {
    base.costPerMessage = base.messagesCount > 0 ? base.spend / base.messagesCount : undefined
  }
  if (base.purchasesCount == null || base.purchasesCount === 0) {
    base.purchasesCount = sumPurchases > 0 ? sumPurchases : undefined
  }
  if (base.costPerPurchase == null && base.purchasesCount && base.spend) {
    base.costPerPurchase = base.purchasesCount > 0 ? base.spend / base.purchasesCount : undefined
  }

  return base
})

const currency = computed(() => report.value?.currency || 'USD')
const maxTrendSpend = computed(() => Math.max(1, ...(report.value?.trend || []).map(point => point.spend || 0)))
const statusCounts = computed(() => productFilteredItems.value.reduce((counts, item) => {
  const status = (item.effectiveStatus || item.status || '').toUpperCase()
  if (status === 'ACTIVE' || status === 'IN_PROCESS' || status === 'PENDING_REVIEW') counts.active += 1
  else if (status === 'PAUSED' || status === 'CAMPAIGN_PAUSED' || status === 'ADSET_PAUSED') counts.paused += 1
  else counts.issues += 1
  return counts
}, { active: 0, paused: 0, issues: 0 } as Record<'active' | 'paused' | 'issues', number>))

const outcomeCounts = computed(() => {
  const list = productFilteredItems.value
  let angel = 0
  let purchases = 0
  let messages = 0
  let scale = 0
  let pause = 0

  for (const item of list) {
    if (isAngelAd(item).isAngel) angel++
    if ((getPurchasesCount(item) || 0) > 0) purchases++
    if ((getMessagesCount(item) || 0) > 0 || (isMessagingAction(item.results?.type) && (item.results?.value || 0) > 0)) messages++
    if (isScaleCandidate(item).eligible) scale++
    if (decision(item).code === 'pause') pause++
  }

  return { all: list.length, angel, purchases, messages, scale, pause }
})

const fmt = (value?: number | null, digits = 2) => value == null ? '—' : new Intl.NumberFormat(undefined, { maximumFractionDigits: digits }).format(value)
const money = (value?: number | null) => value == null ? '—' : new Intl.NumberFormat(undefined, { style: 'currency', currency: currency.value, maximumFractionDigits: 2 }).format(value)
const percent = (value?: number | null) => value == null ? '—' : `${fmt(value, 2)}%`
const contribution = (item?: MetaInsightItem) => {
  if (item?.conversionValue == null || item?.spend == null) return undefined
  const strat = getEffectiveStrategy(item)
  const margin = strat.grossMargin ?? grossMargin.value
  if (margin == null) return undefined
  return item.conversionValue * (margin / 100) - item.spend
}
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
    purchase: { lo: 'ການຊື້ (Orders)', th: 'การซื้อ (Orders)', en: 'Purchases' },
    lead: { lo: 'ລີດ (Leads)', th: 'ลีด (Leads)', en: 'Leads' },
    'onsite_conversion.lead_grouped': { lo: 'ລີດ (Leads)', th: 'ลีด (Leads)', en: 'Leads' },
    messaging_conversation_started_7d: { lo: 'ຄົນສົ່ງຂໍ້ຄວາມ (Conversations)', th: 'คนทักข้อความ (Conversations)', en: 'Conversations' },
    'onsite_conversion.messaging_first_reply': { lo: 'ຄົນສົ່ງຂໍ້ຄວາມ (Conversations)', th: 'คนทักข้อความ (Conversations)', en: 'Conversations' },
    'onsite_conversion.messaging_conversation_started_7d': { lo: 'ຄົນສົ່ງຂໍ້ຄວາມ (Conversations)', th: 'คนทักข้อความ (Conversations)', en: 'Conversations' },
    landing_page_view: { lo: 'ເບິ່ງໜ້າເວັບ', th: 'ดูหน้าเว็บ', en: 'Landing-page views' },
    link_click: { lo: 'ຄລິກລິ້ງ', th: 'คลิกลิงก์', en: 'Link clicks' },
  }
  return type && labels[type] ? l(labels[type]) : l({ lo: 'ຜົນລັບ (Results)', th: 'ผลลัพธ์ (Results)', en: 'Results' })
}

async function setPreset(value: string) {
  datePreset.value = value
  if (value === 'today') {
    const end = new Date()
    since.value = iso(end)
    until.value = iso(end)
  } else if (value === 'yesterday') {
    const y = new Date()
    y.setDate(y.getDate() - 1)
    since.value = iso(y)
    until.value = iso(y)
  } else {
    const days = Number(value)
    if (!Number.isFinite(days)) return
    const end = new Date()
    const start = new Date(end.getFullYear(), end.getMonth(), end.getDate() - days + 1)
    since.value = iso(start)
    until.value = iso(end)
  }
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

function parseProductPrice(product?: Product): number | null {
  if (!product) return null
  if (product.salePrice != null && product.salePrice > 0) return product.salePrice
  if (product.price) {
    const cleaned = String(product.price).replace(/[^0-9.]/g, '')
    const num = parseFloat(cleaned)
    if (!isNaN(num) && num > 0) return num
  }
  return null
}

function calculateCostRatio(adCost: number, productPrice: number, adCurrency: string, prodCurrency?: string): number | null {
  if (!adCost || !productPrice || productPrice <= 0) return null
  let normalizedCost = adCost
  if (adCurrency === 'USD' && (prodCurrency === 'LAK' || productPrice > 2000)) {
    normalizedCost = adCost * 22000
  } else if (adCurrency === 'THB' && (prodCurrency === 'LAK' || productPrice > 2000)) {
    normalizedCost = adCost * 620
  } else if (adCurrency === 'LAK' && productPrice < 200) {
    normalizedCost = adCost / 22000
  }
  const ratio = Math.round((normalizedCost / productPrice) * 100)
  return isFinite(ratio) ? ratio : null
}

function buyingOpportunity(item: MetaInsightItem) {
  const purchases = getPurchasesCount(item) || 0
  const cost = getCostPerPurchase(item)
  const messages = getMessagesCount(item) || 0
  const messageCost = getCostPerMessage(item)
  const clicks = item.linkClicks || 0
  const spend = item.spend || 0
  const ctr = item.linkCtr || 0
  const linked = getLinkedProduct(item)
  const productPrice = parseProductPrice(linked?.product)
  const costRatio = cost != null && productPrice != null
    ? calculateCostRatio(cost, productPrice, currency.value, linked?.product.currency)
    : null

  let conversionRate: number | null = null
  if (clicks > 0 && purchases > 0) {
    conversionRate = Math.min(100, Math.round((purchases / clicks) * 1000) / 10)
  }

  const costRatioText = costRatio != null ? percent(costRatio) : '—'
  const costRatioIsGood = costRatio != null && costRatio <= 20
  const costRatioIsHigh = costRatio != null && costRatio > 50
  const conversionRateText = conversionRate != null ? `${fmt(conversionRate, 1)}%` : '—'

  // 1. High Buying Opportunity (🔥) - Strict: requires at least 1 verified purchase with good metrics, or >= 3 purchases!
  const strat = getEffectiveStrategy(item)
  if (purchases >= 3 || (purchases >= 1 && (costRatio != null ? costRatio <= 25 : (cost == null || cost <= (strat.targetCost || targetCost.value || 1.8))))) {
    return {
      level: 'high',
      label: l({ lo: '🔥 ໂອກາດຊື້ສູງ', th: '🔥 โอกาสซื้อสูง', en: '🔥 High Opportunity' }),
      shortLabel: l({ lo: '🔥 ໂອກາດຊື້ສູງ', th: '🔥 โอกาสซื้อสูง', en: '🔥 High' }),
      badgeText: conversionRate != null
        ? l({ lo: `🔥 ໂອກາດຊື້ສູງ (${purchases} ອໍເດີ · CR ${conversionRate}%)`, th: `🔥 โอกาสซื้อสูง (${purchases} ออเดอร์ · CR ${conversionRate}%)`, en: `🔥 High Opportunity (${purchases} orders · CR ${conversionRate}%)` })
        : l({ lo: `🔥 ໂອກາດຊື້ສູງ (${purchases} ອໍເດີ)`, th: `🔥 โอกาสซื้อสูง (${purchases} ออเดอร์)`, en: `🔥 High Opportunity (${purchases} orders)` }),
      color: '#167a4b',
      chipColor: 'success',
      bgColor: 'rgba(32,164,100,.12)',
      conversionRate,
      conversionRateText,
      purchases,
      costRatio,
      costRatioText,
      costRatioIsGood,
      costRatioIsHigh,
    }
  }

  // 2. Moderate Buying Opportunity (⚡) - Has at least 1 verified purchase
  if (purchases >= 1) {
    return {
      level: 'medium',
      label: l({ lo: '⚡ ໂອກາດຊື້ປານກາງ', th: '⚡ โอกาสซื้อปานกลาง', en: '⚡ Moderate Opportunity' }),
      shortLabel: l({ lo: '⚡ ໂອກາດປານກາງ', th: '⚡ โอกาสปานกลาง', en: '⚡ Moderate' }),
      badgeText: l({ lo: `⚡ ໂອກາດຊື້ດີ (${purchases} ອໍເດີ${conversionRate != null ? ` · CR ${conversionRate}%` : ''})`, th: `⚡ โอกาสซื้อดี (${purchases} ออเดอร์)`, en: `⚡ Moderate (${purchases} orders)` }),
      color: '#0284c7',
      chipColor: 'info',
      bgColor: 'rgba(2,132,199,.12)',
      conversionRate,
      conversionRateText,
      purchases,
      costRatio,
      costRatioText,
      costRatioIsGood,
      costRatioIsHigh,
    }
  }

  // 3. Inquiries / Messaging (💬) - Received messages, but 0 purchases yet
  if (messages > 0) {
    const costText = messageCost != null ? ` · ${money(messageCost)}/ຂໍ້ຄວາມ` : ''
    return {
      level: 'inquiries',
      label: l({ lo: '💬 ມີຄົນທັກແຊັດ', th: '💬 มีคนทักแชต', en: '💬 Inquiries' }),
      shortLabel: l({ lo: '💬 ຄົນທັກ', th: '💬 คนทัก', en: '💬 Inquiries' }),
      badgeText: l({
        lo: `💬 ມີຄົນທັກ ${messages} ຂໍ້ຄວາມ (ຍັງບໍ່ມີອໍເດີ${costText})`,
        th: `💬 มีคนทัก ${messages} ข้อความ (ยังไม่มีออเดอร์${costText})`,
        en: `💬 ${messages} inquiries (0 orders${costText})`
      }),
      color: '#0284c7',
      chipColor: 'info',
      bgColor: 'rgba(2,132,199,.12)',
      conversionRate: null,
      conversionRateText: '—',
      purchases: 0,
      costRatio,
      costRatioText,
      costRatioIsGood,
      costRatioIsHigh,
    }
  }

  // 4. Good Traffic (⚡) - CTR >= 1.5% and budget spent
  if (ctr >= 1.5 && spend > 0) {
    return {
      level: 'medium',
      label: l({ lo: '⚡ ມີຄົນສົນໃຈຄລິກ', th: '⚡ มีคนสนใจคลิก', en: '⚡ Good traffic' }),
      shortLabel: l({ lo: '⚡ ສົນໃຈຄລິກ', th: '⚡ สนใจคลิก', en: '⚡ Good CTR' }),
      badgeText: l({ lo: `⚡ ມີຄົນສົນໃຈຄລິກ (CTR ${percent(ctr)})`, th: `⚡ มีคนสนใจคลิก (CTR ${percent(ctr)})`, en: `⚡ Good traffic (CTR ${percent(ctr)})` }),
      color: '#0284c7',
      chipColor: 'info',
      bgColor: 'rgba(2,132,199,.12)',
      conversionRate: null,
      conversionRateText: '—',
      purchases: 0,
      costRatio,
      costRatioText,
      costRatioIsGood,
      costRatioIsHigh,
    }
  }

  // 5. Low Buying Opportunity (💤)
  const thresholdSpend = currencyOffset.value === 1 ? 50000 : 5
  if (spend > thresholdSpend && purchases === 0) {
    return {
      level: 'low',
      label: l({ lo: '💤 ໂອກາດຊື້ຕ່ຳ', th: '💤 โอกาสซื้อต่ำ', en: '💤 Low Opportunity' }),
      shortLabel: l({ lo: '💤 ໂອກາດຕ່ຳ', th: '💤 โอกาสต่ำ', en: '💤 Low' }),
      badgeText: l({ lo: `💤 ໂອກາດຊື້ຕ່ຳ (ຍັງບໍ່ມີອໍເດີ)`, th: `💤 โอกาสซื้อต่ำ (ยังไม่มีออเดอร์)`, en: `💤 Low Opportunity (0 Orders)` }),
      color: '#dc2626',
      chipColor: 'error',
      bgColor: 'rgba(239,68,68,.12)',
      conversionRate: 0,
      conversionRateText: '0%',
      purchases: 0,
      costRatio,
      costRatioText,
      costRatioIsGood,
      costRatioIsHigh,
    }
  }

  // 6. Neutral / Testing (🌱)
  return {
    level: 'neutral',
    label: l({ lo: '🌱 ກຳລັງທົດສອບ', th: '🌱 กำลังทดสอบ', en: '🌱 Testing' }),
    shortLabel: l({ lo: '🌱 ທົດສອບ', th: '🌱 ทดสอบ', en: '🌱 Testing' }),
    badgeText: l({ lo: '🌱 ກຳລັງທົດສອບ (ປ່ອຍແລ່ນຕາມງົບ)', th: '🌱 กำลังทดสอบ (ปล่อยวิ่งตามงบ)', en: '🌱 Testing phase (Run on budget)' }),
    color: '#6b7280',
    chipColor: 'default',
    bgColor: 'rgba(107,114,128,.12)',
    conversionRate: null,
    conversionRateText: '—',
    purchases: 0,
    costRatio,
    costRatioText,
    costRatioIsGood,
    costRatioIsHigh,
  }
}

function isAngelAd(item: MetaInsightItem): { isAngel: boolean; score: number; reason: string } {
  if (!item) return { isAngel: false, score: 0, reason: '' }
  const status = statusFor(item).toUpperCase()
  const isPaused = status.includes('PAUSED')
  if (isPaused) {
    return { isAngel: false, score: 0, reason: '' }
  }
  const purchases = getPurchasesCount(item) || 0
  const cost = getCostPerPurchase(item)
  const messages = getMessagesCount(item) || 0
  const messageCost = getCostPerMessage(item)
  const ctr = item.linkCtr || 0
  const roas = item.roas || 0
  const opp = buyingOpportunity(item)
  const strat = getEffectiveStrategy(item)
  const targetCpa = strat.targetCost || targetCost.value || (currencyOffset.value === 1 ? 100000 : 5)

  // 1. Top Tier Angel Ad (🧚‍♀️): Verified purchases >= 3 with profitable CPA or high ROAS
  if (purchases >= 3 && (cost == null || cost <= targetCpa * 1.35)) {
    return {
      isAngel: true,
      score: 100,
      reason: l({
        lo: `🧚‍♀️ Ads ນາງຟ້າ (Super Winner): ປິດໄດ້ ${purchases} ອໍເດີ${cost != null ? ` (CPA ${money(cost)}/ອໍເດີ)` : ''}${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''}`,
        th: `🧚‍♀️ แอดนางฟ้า (Super Winner): ปิดได้ ${purchases} ออเดอร์${cost != null ? ` (CPA ${money(cost)}/ออเดอร์)` : ''}${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''}`,
        en: `🧚‍♀️ Angel Ad (Super Winner): ${purchases} orders closed${cost != null ? ` (CPA ${money(cost)}/order)` : ''}${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''}`,
      }),
    }
  }

  // 2. Strong Performer: High purchase intent (at least 1 order and good ROAS or high buying opportunity)
  if (purchases >= 1 && (roas >= 2.0 || opp.level === 'high' || (cost != null && cost <= targetCpa))) {
    return {
      isAngel: true,
      score: 85,
      reason: l({
        lo: `🌟 Ads ດາວເດັ່ນ: ປິດໄດ້ ${purchases} ອໍເດີ${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''} (ໂອກາດຊື້ສູງ / ພ້ອມຂະຫຍາຍງົບ)`,
        th: `🌟 แอดดาวเด่น: ปิดได้ ${purchases} ออเดอร์${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''} (โอกาสซื้อสูง / พร้อมขยายงบ)`,
        en: `🌟 Winning Ad: ${purchases} orders closed${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''} (High conversion / Scale candidate)`,
      }),
    }
  }

  // 3. High Inquiries Champion (💬): Low cost per message & high volume & good CTR
  if (purchases === 0 && messages >= 5 && ctr >= 2.0 && messageCost != null && messageCost <= (currencyOffset.value === 1 ? 25000 : 1.5)) {
    return {
      isAngel: true,
      score: 75,
      reason: l({
        lo: `💬 Ads ທັກແຊັດປັງ: ຄົນທັກ ${messages} ຄົນ (${money(messageCost)}/ຂໍ້ຄວາມ, CTR ${percent(ctr)})`,
        th: `💬 แอดทักแชตปัง: คนทัก ${messages} คน (${money(messageCost)}/ข้อความ, CTR ${percent(ctr)})`,
        en: `💬 Inquiries Champion: ${messages} messages (${money(messageCost)}/msg, CTR ${percent(ctr)})`,
      }),
    }
  }

  return { isAngel: false, score: 0, reason: '' }
}

function decision(item: MetaInsightItem): { code: DecisionCode; color: string; icon: string; title: string; detail: string; signals: string[] } {
  if (!item.impressions && !item.spend) {
    return { code: 'no_data', color: 'default', icon: 'mdi-minus-circle-outline', title: l({ lo: 'ສ້າງໃໝ່ / ຍັງບໍ່ມີຂໍ້ມູນ', th: 'เพิ่งสร้าง / ยังไม่มีข้อมูล', en: 'New / No delivery data' }), detail: l({ lo: 'ໂຄສະນາຫາກໍ່ສ້າງໃໝ່ ຫຼື ຍັງບໍ່ທັນມີຄ່າໃຊ້ຈ່າຍ ແລະການສະແດງຜົນໃນຊ່ວງເວລານີ້.', th: 'โฆษณาเพิ่งสร้างหรือยังไม่มีค่าใช้จ่ายและการแสดงผลในช่วงเวลานี้', en: 'Newly created ad or no delivery/spend in selected date range.' }), signals: [] }
  }
  const isPaused = statusFor(item).toUpperCase().includes('PAUSED')
  const bad: string[] = []
  const good: string[] = []
  const opp = buyingOpportunity(item)
  const purchases = getPurchasesCount(item) || 0
  const cost = getCostPerPurchase(item)
  const messages = getMessagesCount(item) || 0
  const messageCost = getCostPerMessage(item)

  // 1. Core Conversion & Purchase Outcome
  if (purchases >= 3) {
    good.push(l({
      lo: `ປິດການຊື້ດີ: ${purchases} ອໍເດີ${cost != null ? ` (ຕົ້ນທຶນ ${money(cost)}/ອໍເດີ)` : ''}`,
      th: `ปิดการซื้อดี: ${purchases} ออเดอร์${cost != null ? ` (ต้นทุน ${money(cost)}/ออเดอร์)` : ''}`,
      en: `Strong purchases: ${purchases} orders${cost != null ? ` (Cost ${money(cost)}/order)` : ''}`
    }))
  } else if (purchases >= 1) {
    good.push(l({
      lo: `ມີການຊື້ເກີດຂຶ້ນ: ${purchases} ອໍເດີ${cost != null ? ` (${money(cost)}/ອໍເດີ)` : ''}`,
      th: `มีคำสั่งซื้อเกิดขึ้น: ${purchases} ออเดอร์${cost != null ? ` (${money(cost)}/ออเดอร์)` : ''}`,
      en: `Has purchases: ${purchases} orders${cost != null ? ` (${money(cost)}/order)` : ''}`
    }))
  } else if (messages > 0) {
    good.push(l({
      lo: `ມີຄົນທັກແຊັດ: ${messages} ຂໍ້ຄວາມ${messageCost != null ? ` (ສະເລ່ຍ ${money(messageCost)}/ຂໍ້ຄວາມ)` : ''} (ຍັງບໍ່ມີອໍເດີ)`,
      th: `มีคนทักแชต: ${messages} ข้อความ${messageCost != null ? ` (เฉลี่ย ${money(messageCost)}/ข้อความ)` : ''} (ยังไม่มีออเดอร์)`,
      en: `Inquiries: ${messages} messages${messageCost != null ? ` (Avg ${money(messageCost)}/msg)` : ''} (0 purchases)`
    }))
  } else if (item.spend != null && item.spend > (currencyOffset.value === 1 ? 50000 : 5)) {
    bad.push(l({
      lo: `ໃຊ້ງົບແລ້ວແຕ່ຍັງບໍ່ມີການຊື້ (${money(item.spend)})`,
      th: `ใช้งบแล้วแต่ยังไม่มีการซื้อ (${money(item.spend)})`,
      en: `Spent budget with 0 purchases (${money(item.spend)})`
    }))
  }

  // 2. Linked Product Cost-to-Price Evaluation
  if (opp.costRatio != null) {
    if (opp.costRatio <= 20) {
      good.push(l({
        lo: `ຕົ້ນທຶນແອດພຽງ ${opp.costRatio}% ຂອງລາຄາສິນຄ້າ`,
        th: `ต้นทุนแอดเพียง ${opp.costRatio}% ของราคาสินค้า`,
        en: `Ad cost is only ${opp.costRatio}% of product price`
      }))
    } else if (opp.costRatio > 50) {
      bad.push(l({
        lo: `ຕົ້ນທຶນແອດສູງເຖິງ ${opp.costRatio}% ຂອງລາຄາສິນຄ້າ`,
        th: `ต้นทุนแอดสูงถึง ${opp.costRatio}% ของราคาสินค้า`,
        en: `Ad cost consumes ${opp.costRatio}% of product price`
      }))
    }
  }

  // 3. Target Cost (Product Strategy or Account Default)
  const strat = getEffectiveStrategy(item)
  const targetCpa = strat.targetCost
  if (targetCpa && cost != null) {
    if (cost > targetCpa) {
      bad.push(l({
        lo: `ຕົ້ນທຶນ/ຜົນລັບ ${money(cost)} ສູງກວ່າເກນ ${strat.isProductCustom ? `(${strat.product?.name}: ${money(targetCpa)})` : money(targetCpa)}`,
        th: `ต้นทุน/ผลลัพธ์ ${money(cost)} สูงกว่าเกณฑ์ ${strat.isProductCustom ? `(${strat.product?.name}: ${money(targetCpa)})` : money(targetCpa)}`,
        en: `Cost/result ${money(cost)} exceeds target ${strat.isProductCustom ? `(${strat.product?.name}: ${money(targetCpa)})` : money(targetCpa)}`
      }))
    } else {
      good.push(l({
        lo: `ຕົ້ນທຶນ/ຜົນລັບຢູ່ໃນເກນ ${strat.isProductCustom ? `(${strat.product?.name}: ${money(targetCpa)})` : money(targetCpa)}`,
        th: `ต้นทุน/ผลลัพธ์อยู่ในเกณฑ์ ${strat.isProductCustom ? `(${strat.product?.name}: ${money(targetCpa)})` : money(targetCpa)}`,
        en: `Cost/result is within target ${strat.isProductCustom ? `(${strat.product?.name}: ${money(targetCpa)})` : money(targetCpa)}`
      }))
    }
  }

  // 4. Contribution Margin (if configured)
  const estimatedContribution = contribution(item)
  if (estimatedContribution != null) {
    if (estimatedContribution < 0) bad.push(l({ lo: 'ມູນຄ່າຂາຍຫຼັງຫັກ Margin ບໍ່ພໍຄ່າ Ads', th: 'มูลค่าขายหลังหัก Margin ไม่พอค่าโฆษณา', en: 'Margin-adjusted conversion value does not cover ad spend' }))
    else good.push(l({ lo: 'ມູນຄ່າຂາຍຫຼັງຫັກ Margin ຍັງຄຸ້ມຄ່າ Ads', th: 'มูลค่าขายหลังหัก Margin ยังคุ้มค่าโฆษณา', en: 'Margin-adjusted conversion value covers ad spend' }))
  }

  // 5. CTR Analysis (Always evaluated and visible!)
  if (item.linkCtr != null) {
    if (item.linkCtr >= goodCTR.value) {
      good.push(l({ lo: `CTR ດີ (${percent(item.linkCtr)})—Creative ໜ້າສົນໃຈ`, th: `CTR ดี (${percent(item.linkCtr)})—ครีเอทีฟน่าสนใจ`, en: `Strong CTR (${percent(item.linkCtr)})—creative is engaging` }))
    } else if (item.linkCtr < weakCTR.value) {
      if (purchases >= 1) {
        good.push(l({
          lo: `CTR ${percent(item.linkCtr)} ແຕ່ປິດການຊື້ດີ (ລູກຄ້າຊື້ກົງກຸ່ມ)`,
          th: `CTR ${percent(item.linkCtr)} แต่ปิดการซื้อดี (ตรงกลุ่มผู้ซื้อ)`,
          en: `CTR is ${percent(item.linkCtr)} with positive purchase intent`
        }))
      } else {
        bad.push(l({
          lo: `CTR ຕ່ຳ (${percent(item.linkCtr)})—Creative ອາດບໍ່ດຶງດູດ`,
          th: `CTR ต่ำ (${percent(item.linkCtr)})—ครีเอทีฟอาจไม่น่าสนใจ`,
          en: `Low CTR (${percent(item.linkCtr)})—creative may not attract clicks`
        }))
      }
    } else {
      good.push(l({ lo: `CTR ປານກາງ (${percent(item.linkCtr)})`, th: `CTR ปานกลาง (${percent(item.linkCtr)})`, en: `Moderate CTR (${percent(item.linkCtr)})` }))
    }
  }

  // 6. Frequency & Audience Saturation
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

  // 7. Rankings
  const rankings = [item.qualityRanking, item.engagementRateRanking, item.conversionRateRanking].filter(Boolean)
  if (rankings.some(value => value?.includes('BELOW_AVERAGE'))) bad.push(l({ lo: 'Ranking ຕ່ຳກວ່າຄ່າສະເລ່ຍ', th: 'อันดับต่ำกว่าค่าเฉลี่ย', en: 'Ranking is below average' }))

  // Paused Ad Handling (NEVER recommend Scale for paused ads with 0 purchases!)
  if (isPaused) {
    if (purchases >= 3) {
      return {
        code: 'review',
        color: 'warning',
        icon: 'mdi-pause-circle-outline',
        title: l({ lo: '⏸️ ປິດຊົ່ວຄາວ (ເຄີຍໄດ້ອໍເດີດີ)', th: '⏸️ หยุดชั่วคราว (เคยได้ออเดอร์ดี)', en: 'Paused (Good historic orders)' }),
        detail: l({
          lo: `ແອດນີ້ຖືກປິດຢູ່ ແຕ່ເຄີຍປິດໄດ້ ${purchases} ອໍເດີ. ສາມາດກວດສອບແລ້ວກົດເປີດໃໝ່ໄດ້.`,
          th: `แอดนี้ถูกปิดอยู่ แต่เคยได้ ${purchases} ออเดอร์ สามารถตรวจสอบแล้วกดเปิดใหม่ได้`,
          en: `This ad is paused but achieved ${purchases} orders. You can review and activate it.`
        }),
        signals: [...good, ...bad]
      }
    }
    return {
      code: 'pause',
      color: 'default',
      icon: 'mdi-pause-circle-outline',
      title: l({ lo: '⏸️ ປິດຊົ່ວຄາວ (Paused)', th: '⏸️ หยุดชั่วคราว (Paused)', en: 'Paused' }),
      detail: messages > 0
        ? l({
            lo: `ແອດຖືກປິດຢູ່. ມີຄົນທັກ ${messages} ຂໍ້ຄວາມ (ໃຊ້ງົບ ${money(item.spend)}), ແຕ່ຍັງບໍ່ມີອໍເດີ. ບໍ່ຕ້ອງສະເກວ.`,
            th: `แอดถูกหยุดอยู่ มีคนทัก ${messages} ข้อความ (ใช้งบ ${money(item.spend)}) แต่ยังไม่มีออเดอร์ ไม่ต้อง Scale`,
            en: `Ad is paused. Received ${messages} messages (${money(item.spend)} spend) with 0 purchases. Do not scale.`
          })
        : l({
            lo: `ແອດຖືກປິດຕັ້ງແຕ່ກ່ອນໜ້ານີ້ (ໃຊ້ງົບ ${money(item.spend)}).`,
            th: `แอดถูกหยุดไว้ก่อนหน้านี้ (ใช้งบ ${money(item.spend)})`,
            en: `Ad was previously paused (spent ${money(item.spend)}).`
          }),
      signals: [...good, ...bad]
    }
  }

  // Active Ads Evaluation:
  const angel = isAngelAd(item)
  if (angel.isAngel && purchases >= 3) {
    return {
      code: 'keep',
      color: 'success',
      icon: 'mdi-star-face',
      title: l({ lo: '🧚‍♀️ Ads ນາງຟ້າ / Scale', th: '🧚‍♀️ แอดนางฟ้า / Scale', en: '🧚‍♀️ Angel Ad / Scale' }),
      detail: angel.reason,
      signals: [...good, ...bad]
    }
  }

  // ONLY scale if verified purchases >= 3
  if (purchases >= 3 && bad.length <= 1) {
    return {
      code: 'keep',
      color: 'success',
      icon: 'mdi-rocket-launch-outline',
      title: l({ lo: '🚀 ຄວນເປີດຕໍ່ / Scale', th: '🚀 ควรเปิดต่อ / Scale', en: 'Keep Running / Scale' }),
      detail: l({
        lo: `ປິດໄດ້ ${purchases} ອໍເດີ, ໂອກາດຊື້ສູງ. ສາມາດເພີ່ມງົບ +10% ຫາ +20% ໄດ້.`,
        th: `ปิดได้ ${purchases} ออเดอร์ โอกาสซื้อสูง สามารถเพิ่มงบ +10% ถึง +20% ได้`,
        en: `${purchases} orders closed, high buying opportunity. Ready for +10% to +20% scaling.`
      }),
      signals: [...good, ...bad]
    }
  }

  // Active Ads with messages but 0 purchases: RUN ON BUDGET! Do NOT scale!
  if (purchases === 0 && messages > 0) {
    return {
      code: 'keep',
      color: 'info',
      icon: 'mdi-chat-processing-outline',
      title: l({ lo: '🌱 ປ່ອຍແລ່ນຕາມງົບ (ເບິ່ງການປິດການຂາຍ)', th: '🌱 ปล่อยวิ่งตามงบ (ดูการปิดการขาย)', en: '🌱 Run on Budget (Monitor Sales)' }),
      detail: l({
        lo: `ມີຄົນທັກ ${messages} ຂໍ້ຄວາມ${messageCost != null ? ` (${money(messageCost)}/ຂໍ້ຄວາມ)` : ''}, ແຕ່ຍັງບໍ່ທັນມີອໍເດີ. ໃຫ້ປ່ອຍແລ່ນຕາມງົບປົກກະຕິ (ບໍ່ຕ້ອງສະເກວ) ເພື່ອກວດສອບການຕອບແຊັດ ແລະ ປິດການຂາຍ.`,
        th: `มีคนทัก ${messages} ข้อความ${messageCost != null ? ` (${money(messageCost)}/ข้อความ)` : ''} แต่ยังไม่มีออเดอร์ ให้ปล่อยวิ่งตามงบปกติ (ไม่ต้อง Scale) เพื่อตรวจสอบการตอบแชตและปิดการขาย`,
        en: `${messages} messages received${messageCost != null ? ` (${money(messageCost)}/msg)` : ''} with 0 purchases. Keep running on current budget (do not scale) while monitoring sales conversion.`
      }),
      signals: [...good, ...bad]
    }
  }

  if (bad.length >= 2) {
    return {
      code: 'pause',
      color: 'error',
      icon: 'mdi-pause-circle-outline',
      title: l({ lo: 'ຄວນພິຈາລະນາປິດ', th: 'ควรพิจารณาปิด', en: 'Pause candidate' }),
      detail: l({ lo: 'ມີຫຼາຍສັນຍານອ່ອນ. ກວດ 3–7 ມື້ ແລະກຳໄລກ່ອນປິດ.', th: 'มีหลายสัญญาณอ่อน ตรวจแนวโน้ม 3–7 วันและกำไรก่อนปิด', en: 'Several weak signals. Check the 3–7 day trend and profit before pausing.' }),
      signals: bad
    }
  }

  if (bad.length === 1 && purchases === 0) {
    return {
      code: 'review',
      color: 'warning',
      icon: 'mdi-alert-circle-outline',
      title: l({ lo: 'ຄວນກວດສອບ', th: 'ควรตรวจสอบ', en: 'Review' }),
      detail: l({ lo: 'ຍັງບໍ່ຄວນປິດທັນທີ. ປັບ Creative, Audience ຫຼືລໍຖ້າເບິ່ງ trend.', th: 'ยังไม่ควรปิดทันที ลองปรับครีเอทีฟ กลุ่มเป้าหมาย หรือดูแนวโน้มต่อ', en: 'Do not pause yet. Review creative, audience, and the trend.' }),
      signals: bad
    }
  }

  return {
    code: 'keep',
    color: 'success',
    icon: 'mdi-check-circle-outline',
    title: l({ lo: '🌱 ປ່ອຍແລ່ນຕາມງົບ', th: '🌱 ปล่อยวิ่งตามงบ', en: 'Run on Budget' }),
    detail: l({ lo: 'ສັນຍານຫຼັກຍັງປົກກະຕິ. ປ່ອຍແລ່ນຕາມງົບເດີມ ແລະຕິດຕາມຜົນຕໍ່ໄປ.', th: 'สัญญาณหลักยังปกติ ปล่อยวิ่งตามงบเดิมและติดตามผลต่อไป', en: 'Core signals are stable. Keep running on current budget and monitor.' }),
    signals: [...good, ...bad]
  }
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

function convertPriceToAdCurrency(productPrice: number, prodCurrency?: string): number {
  if (!productPrice) return 0
  if (currency.value === 'USD' && (prodCurrency === 'LAK' || productPrice > 2000)) {
    return Math.round((productPrice / 22000) * 100) / 100
  }
  if (currency.value === 'THB' && (prodCurrency === 'LAK' || productPrice > 2000)) {
    return Math.round((productPrice / 620) * 100) / 100
  }
  if (currency.value === 'LAK' && productPrice < 200) {
    return Math.round(productPrice * 22000)
  }
  return productPrice
}

function loadProductStrategies() {
  loadProductStrategiesFromStore(adAccountId.value)
}

function saveProductStrategies() {
  saveProductStrategiesToStore(adAccountId.value)
}

function getEffectiveStrategy(item?: MetaInsightItem) {
  const linked = item ? getLinkedProduct(item) : undefined
  const prodId = linked?.product?.id
  const prodStrat = prodId && productStrategies.value[prodId]?.enabled ? productStrategies.value[prodId] : undefined

  const effectiveTargetCost = prodStrat?.targetCost ?? targetCost.value
  const effectiveGrossMargin = prodStrat?.grossMargin ?? grossMargin.value
  const effectiveScaleMinPurchases = prodStrat?.scaleMinPurchases ?? autoPilotConfig.value.scaleMinPurchases ?? 3
  const effectiveScaleMaxCPA = prodStrat?.scaleMaxCPA ?? autoPilotConfig.value.scaleMaxCPA ?? effectiveTargetCost ?? (currencyOffset.value === 1 ? 50000 : 2.5)
  const effectiveMaxDailyBudgetCap = prodStrat?.scaleMaxDailyBudgetCap ?? autoPilotConfig.value.maxDailyBudgetCap ?? null
  const effectiveStopMinSpend = prodStrat?.stopMinSpend ?? autoPilotConfig.value.stopMinSpend ?? (currencyOffset.value === 1 ? 50000 : 5)
  const effectiveGoodCTR = goodCTR.value
  const effectiveWeakCTR = weakCTR.value
  const effectiveMaxFrequency = maxFrequency.value

  return {
    isProductCustom: !!prodStrat,
    product: linked?.product,
    flowName: linked?.flowName,
    targetCost: effectiveTargetCost,
    grossMargin: effectiveGrossMargin,
    scaleMinPurchases: effectiveScaleMinPurchases,
    scaleMaxCPA: effectiveScaleMaxCPA,
    maxDailyBudgetCap: effectiveMaxDailyBudgetCap,
    stopMinSpend: effectiveStopMinSpend,
    goodCTR: effectiveGoodCTR,
    weakCTR: effectiveWeakCTR,
    maxFrequency: effectiveMaxFrequency,
  }
}

function openProductStrategy(productId: string) {
  selectedStrategyProductId.value = productId
  strategyTab.value = 'products'
  settingsOpen.value = true
}

function openStrategyModal() {
  if (selectedProductId.value !== 'all' && selectedProductId.value !== 'unlinked') {
    selectedStrategyProductId.value = selectedProductId.value
    strategyTab.value = 'products'
  } else if (!selectedStrategyProductId.value && products.value.length && products.value[0]) {
    selectedStrategyProductId.value = products.value[0].id
  }
  settingsOpen.value = true
}

function autoCalculateProductStrategy(productId: string) {
  const prod = products.value.find(p => p.id === productId)
  if (!prod) return
  const rawPrice = parseProductPrice(prod) || (currencyOffset.value === 1 ? 250000 : 10)
  const price = convertPriceToAdCurrency(rawPrice, prod.currency)

  const margin = 50
  const isLak = currencyOffset.value === 1
  const targetCpa = isLak ? Math.round(price * 0.20 / 1000) * 1000 : Math.round(price * 0.20 * 100) / 100
  const scaleCpa = isLak ? Math.round(price * 0.25 / 1000) * 1000 : Math.round(price * 0.25 * 100) / 100
  const stopSpend = isLak ? Math.round(price * 0.80 / 1000) * 1000 : Math.round(price * 0.80 * 100) / 100
  const budgetCap = isLak ? Math.round(price * 3 / 1000) * 1000 : Math.round(price * 3 * 100) / 100

  if (!productStrategies.value[productId]) {
    productStrategies.value[productId] = {
      productId,
      enabled: true,
      targetCost: targetCpa,
      grossMargin: margin,
      scaleMinPurchases: 3,
      scaleMaxCPA: scaleCpa,
      scaleMaxDailyBudgetCap: budgetCap,
      stopMinSpend: stopSpend,
    }
  } else {
    const s = productStrategies.value[productId]
    s.enabled = true
    s.targetCost = targetCpa
    s.grossMargin = margin
    s.scaleMinPurchases = 3
    s.scaleMaxCPA = scaleCpa
    s.scaleMaxDailyBudgetCap = budgetCap
    s.stopMinSpend = stopSpend
  }

  saveProductStrategies()
  success.value = l({
    lo: `ຄິດໄລ່ຍຸດທະສາດສຳເລັດ: ລາຄາ ${money(price)}, ເປົ້າ CPA ${money(targetCpa)}, Scale CPA <= ${money(scaleCpa)}, Stop Loss >= ${money(stopSpend)}`,
    th: `คำนวณยุทธศาสตร์สำเร็จ: ราคา ${money(price)}, เป้า CPA ${money(targetCpa)}, Scale CPA <= ${money(scaleCpa)}, Stop Loss >= ${money(stopSpend)}`,
    en: `Auto-calculated strategy: Price ${money(price)}, Target CPA ${money(targetCpa)}, Scale CPA <= ${money(scaleCpa)}, Stop Loss >= ${money(stopSpend)}`
  })
}

const strategyProductSelectItems = computed(() => {
  return products.value.map(p => {
    const strat = productStrategies.value[p.id]
    const isCustom = strat && strat.enabled
    let adCount = 0
    for (const it of (report.value?.items || [])) {
      if (getLinkedProduct(it)?.product?.id === p.id) adCount++
    }
    const statusTag = isCustom ? '🟢 ກຳນົດເອງ' : '⚪ ໃຊ້ຄ່າເລີ່ມຕົ້ນ'
    const rawPrice = parseProductPrice(p) || 0
    const priceText = rawPrice > 0 ? money(convertPriceToAdCurrency(rawPrice, p.currency)) : '—'
    return {
      title: `${p.name} · ${priceText} [${statusTag}] (${adCount} ແອດ)`,
      value: p.id,
      product: p,
      adCount,
      isCustom,
    }
  })
})

const currentStrategyProduct = computed(() => {
  if (!selectedStrategyProductId.value && products.value.length) {
    return products.value[0]
  }
  return products.value.find(p => p.id === selectedStrategyProductId.value) || products.value[0]
})

const currentProductStrategy = computed<ProductAdStrategy>(() => {
  const prod = currentStrategyProduct.value
  if (!prod) return { productId: '' }
  if (!productStrategies.value[prod.id]) {
    productStrategies.value[prod.id] = {
      productId: prod.id,
      enabled: false,
    }
  }
  return productStrategies.value[prod.id]!
})

const currentStrategyProductPriceInAdCurrency = computed(() => {
  const prod = currentStrategyProduct.value
  if (!prod) return 0
  const rawPrice = parseProductPrice(prod) || (currencyOffset.value === 1 ? 250000 : 10)
  return convertPriceToAdCurrency(rawPrice, prod.currency)
})

const currentProductBreakEvenROAS = computed(() => {
  const margin = currentProductStrategy.value.grossMargin
  return margin && margin > 0 ? 100 / margin : undefined
})

const currentProductLinkedFlowName = computed(() => {
  const prod = currentStrategyProduct.value
  if (!prod) return undefined
  const flow = flows.value.find(f => f.productId === prod.id)
  return flow?.name
})

const currentProductAdCount = computed(() => {
  const prod = currentStrategyProduct.value
  if (!prod) return 0
  let count = 0
  for (const it of (report.value?.items || [])) {
    if (getLinkedProduct(it)?.product?.id === prod.id) count++
  }
  return count
})

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
  loadProductStrategies()
}
function saveThresholds() {
  if (!import.meta.client || !adAccountId.value) return
  localStorage.setItem(`ads-thresholds:${adAccountId.value}`, JSON.stringify({ targetCost: targetCost.value, goodCTR: goodCTR.value, weakCTR: weakCTR.value, maxFrequency: maxFrequency.value, grossMargin: grossMargin.value, minimumPlanImpressions: minimumPlanImpressions.value, minimumPlanSpend: minimumPlanSpend.value }))
  saveProductStrategies()
  settingsOpen.value = false
  success.value = l({ lo: 'ບັນທຶກການຕັ້ງຄ່າຍຸດທະສາດສຳເລັດແລ້ວ', th: 'บันทึกการตั้งค่ายุทธศาสตร์สำเร็จแล้ว', en: 'Ad strategy settings saved successfully.' })
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
    if (autoPilotConfig.value.enabled) {
      runAutoPilot(false)
    }
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

    recordScaleEvent(item, baseDailyBudget.value, newDailyBudget.value, budgetDiffPercent.value, 'MANUAL')
    addAutoPilotLog({
      id: 'log-' + Date.now() + '-' + Math.random().toString(36).slice(2, 7),
      timestamp: new Date().toISOString(),
      action: 'SCALE',
      entityId: item.entityId,
      entityType: item.entityType,
      entityName: targetName,
      detail: l({
        lo: `[Manual Scale] ປັບງົບໂດຍຜູ້ໃຊ້ (+${budgetDiffPercent.value}%): ຈາກ ${money(baseDailyBudget.value)} ➔ ${money(newDailyBudget.value)}/ມື້ (Cooldown ${autoPilotConfig.value.scaleCooldownHours}h ເລີ່ມຕົ້ນ)`,
        th: `[Manual Scale] ปรับงบโดยผู้ใช้ (+${budgetDiffPercent.value}%): จาก ${money(baseDailyBudget.value)} ➔ ${money(newDailyBudget.value)}/วัน (Cooldown ${autoPilotConfig.value.scaleCooldownHours}h เริ่มต้น)`,
        en: `[Manual Scale] Adjusted by user (+${budgetDiffPercent.value}%): from ${money(baseDailyBudget.value)} ➔ ${money(newDailyBudget.value)}/day (${autoPilotConfig.value.scaleCooldownHours}h cooldown started)`,
      }),
      beforeValue: money(baseDailyBudget.value) + '/ມື້',
      afterValue: money(newDailyBudget.value) + '/ມື້',
      success: true,
    })

    success.value = l({
      lo: `ປັບງົບປະມານໃຫ້ “${targetName}” ເປັນ ${money(newDailyBudget.value)}/ມື້ ສຳເລັດແລ້ວ! (ເຂົ້າສູ່ໄລຍະ Cooldown ${autoPilotConfig.value.scaleCooldownHours}h)`,
      th: `ปรับงบประมาณให้ “${targetName}” เป็น ${money(newDailyBudget.value)}/วัน เรียบร้อยแล้ว (เข้าสู่ระยะ Cooldown ${autoPilotConfig.value.scaleCooldownHours}h)`,
      en: `Updated budget for “${targetName}” to ${money(newDailyBudget.value)}/day successfully (${autoPilotConfig.value.scaleCooldownHours}h cooldown active).`,
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

// --- Auto-Pilot & Cooldown Scale Tracking Functions ---

function loadScaleHistory() {
  if (!import.meta.client || !adAccountId.value) return
  const saved = localStorage.getItem(`ads-scale-history:${adAccountId.value}`)
  if (saved) {
    try {
      scaleHistory.value = JSON.parse(saved)
    } catch {}
  } else {
    scaleHistory.value = {}
  }
}

function saveScaleHistory() {
  if (!import.meta.client || !adAccountId.value) return
  localStorage.setItem(`ads-scale-history:${adAccountId.value}`, JSON.stringify(scaleHistory.value))
}

function recordScaleEvent(item: MetaInsightItem, oldBudget: number, newBudget: number, pct: number, source: 'AUTO' | 'MANUAL') {
  const record: ScaleRecord = {
    entityId: item.entityId,
    entityName: item.entityName,
    entityType: item.entityType,
    scaledAt: Date.now(),
    oldBudget,
    newBudget,
    scalePercent: pct,
    purchasesAtScale: getPurchasesCount(item) || 0,
    scaledBy: source,
  }
  scaleHistory.value[item.entityId] = record
  if (item.adSetId) scaleHistory.value[item.adSetId] = record
  if (item.campaignId) scaleHistory.value[item.campaignId] = record
  saveScaleHistory()
}

function getCooldownInfo(item: MetaInsightItem): { inCooldown: boolean; hoursLeft: number; minutesLeft: number; record: ScaleRecord | null } {
  if (!import.meta.client) return { inCooldown: false, hoursLeft: 0, minutesLeft: 0, record: null }
  const ids = [item.entityId, item.adSetId, item.campaignId].filter(Boolean) as string[]
  let latestRecord: ScaleRecord | null = null
  for (const id of ids) {
    const rec = scaleHistory.value[id]
    if (rec && (!latestRecord || rec.scaledAt > latestRecord.scaledAt)) {
      latestRecord = rec
    }
  }
  if (!latestRecord) return { inCooldown: false, hoursLeft: 0, minutesLeft: 0, record: null }

  const cooldownHours = autoPilotConfig.value.scaleCooldownHours || 24
  const cooldownMs = cooldownHours * 3600 * 1000
  const elapsed = Date.now() - latestRecord.scaledAt
  const remaining = cooldownMs - elapsed

  if (remaining <= 0) {
    return { inCooldown: false, hoursLeft: 0, minutesLeft: 0, record: latestRecord }
  }

  const hoursLeft = Math.floor(remaining / (3600 * 1000))
  const minutesLeft = Math.floor((remaining % (3600 * 1000)) / (60 * 1000))
  return { inCooldown: true, hoursLeft, minutesLeft, record: latestRecord }
}

function loadAutoPilotConfig() {
  if (!import.meta.client || !adAccountId.value) return
  const saved = localStorage.getItem(`ads-autopilot-config:${adAccountId.value}`)
  if (saved) {
    try {
      autoPilotConfig.value = { ...autoPilotConfig.value, ...JSON.parse(saved) }
    } catch {}
  }
  const savedLogs = localStorage.getItem(`ads-autopilot-logs:${adAccountId.value}`)
  if (savedLogs) {
    try {
      autoPilotLogs.value = JSON.parse(savedLogs)
    } catch {}
  }
}

function saveAutoPilotConfig() {
  if (!import.meta.client || !adAccountId.value) return
  localStorage.setItem(`ads-autopilot-config:${adAccountId.value}`, JSON.stringify(autoPilotConfig.value))
}

function saveAutoPilotLogs() {
  if (!import.meta.client || !adAccountId.value) return
  localStorage.setItem(`ads-autopilot-logs:${adAccountId.value}`, JSON.stringify(autoPilotLogs.value))
}

function clearAutoPilotLogs() {
  autoPilotLogs.value = []
  saveAutoPilotLogs()
}

function toggleAutoPilotMaster(val: boolean | null) {
  const isEnabled = Boolean(val)
  autoPilotConfig.value.enabled = isEnabled
  saveAutoPilotConfig()
  if (isEnabled) {
    success.value = l({
      lo: '🤖 ເປີດລະບົບ Auto-Pilot ແລ້ວ! ບັອດຈະເລີ່ມເຝົ້າແອດ, ສະເກວງົບ +20% ໃຫ້ແອດທີ່ຜ່ານເກນ, ແລະ ຢຸດແອດຂາດທຶນ.',
      th: '🤖 เปิดระบบ Auto-Pilot แล้ว! บอทจะเริ่มเฝ้าแอด, Scale งบ +20% ให้แอดที่ผ่านเกณฑ์, และหยุดแอดขาดทุน',
      en: '🤖 Auto-Pilot activated! The bot will now monitor ads, auto-scale winners (+20%), and stop loss on losers.'
    })
    runAutoPilot(false)
  } else {
    notice.value = l({
      lo: '⏸️ ປິດລະບົບ Auto-Pilot ແລ້ວ. ລະບົບກັບສູ່ໂໝດ Manual (ທ່ານສາມາດກວດສອບ ແລະ ກົດເອງໄດ້ທຸກເວລາ).',
      th: '⏸️ ปิดระบบ Auto-Pilot แล้ว ระบบกลับสู่โหมด Manual (คุณสามารถตรวจสอบและกดเองได้ตลอดเวลา)',
      en: '⏸️ Auto-Pilot disabled. Returned to Manual Approval mode.'
    })
  }
}

function isStopLossCandidate(item: MetaInsightItem): { eligible: boolean; reason: string } {
  if (statusFor(item).toUpperCase() !== 'ACTIVE') return { eligible: false, reason: '' }
  if (!autoPilotConfig.value.autoStopEnabled) return { eligible: false, reason: '' }

  const strat = getEffectiveStrategy(item)
  const purchases = getPurchasesCount(item) || 0
  const spend = item.spend || 0
  const ctr = item.linkCtr ?? 0
  const minSpend = strat.stopMinSpend
  const maxCTR = autoPilotConfig.value.stopMaxCTR ?? 1.0

  if (purchases === 0 && spend >= minSpend && ctr < maxCTR) {
    return {
      eligible: true,
      reason: `Spend ${money(spend)} >= ${money(minSpend)}${strat.isProductCustom ? ` (ເກນ ${strat.product?.name})` : ''}, 0 ອໍເດີ, CTR ${percent(ctr)} < ${maxCTR}%`
    }
  }
  return { eligible: false, reason: '' }
}

function isScaleCandidate(item: MetaInsightItem): {
  eligible: boolean
  inCooldown: boolean
  cooldownInfo: { hoursLeft: number; minutesLeft: number; record: ScaleRecord | null }
  reason: string
} {
  const cooldown = getCooldownInfo(item)
  if (statusFor(item).toUpperCase() !== 'ACTIVE') {
    return { eligible: false, inCooldown: cooldown.inCooldown, cooldownInfo: cooldown, reason: '' }
  }
  if (!autoPilotConfig.value.autoScaleEnabled) {
    return { eligible: false, inCooldown: cooldown.inCooldown, cooldownInfo: cooldown, reason: '' }
  }

  const strat = getEffectiveStrategy(item)
  const purchases = getPurchasesCount(item) || 0
  const cost = getCostPerPurchase(item)
  const frequency = item.frequency || 1.0
  const minPurchases = strat.scaleMinPurchases
  const maxCPA = strat.scaleMaxCPA
  const maxFreq = strat.maxFrequency

  // 1. Budget Cap Safety check: if already at or above cap, stop scaling
  const currentBudget = item.dailyBudget ? (item.dailyBudget / currencyOffset.value) : 0
  const budgetCap = strat.maxDailyBudgetCap
  if (budgetCap && currentBudget >= budgetCap) {
    return {
      eligible: false,
      inCooldown: false,
      cooldownInfo: cooldown,
      reason: l({
        lo: `ງົບຮອດເພດານສູງສຸດແລ້ວ (${money(currentBudget)} >= ${money(budgetCap)}/ມື້)${strat.isProductCustom ? ` [ເກນ ${strat.product?.name}]` : ''}`,
        th: `งบถึงเพดานสูงสุดแล้ว (${money(currentBudget)} >= ${money(budgetCap)}/วัน)${strat.isProductCustom ? ` [เกณฑ์ ${strat.product?.name}]` : ''}`,
        en: `Current budget has reached max cap (${money(currentBudget)} >= ${money(budgetCap)}/day)${strat.isProductCustom ? ` [${strat.product?.name}]` : ''}`,
      })
    }
  }

  // 2. Incremental Purchase Requirement: if previously scaled and cooldown ended, require NEW purchases under the new budget!
  if (cooldown.record && !cooldown.inCooldown) {
    const lastPurchases = cooldown.record.purchasesAtScale || 0
    if (purchases <= lastPurchases) {
      return {
        eligible: false,
        inCooldown: false,
        cooldownInfo: cooldown,
        reason: l({
          lo: `ຍັງບໍ່ມີອໍເດີໃໝ່ເພີ່ມຂຶ້ນຫຼັງຈາກສະເກວຄັ້ງກ່ອນ (${purchases}/${lastPurchases} ອໍເດີ). ຕ້ອງມີອໍເດີໃໝ່ເຂົ້າມາຈຶ່ງຈະສະເກວຕໍ່ໄດ້`,
          th: `ยังไม่มีออเดอร์ใหม่เพิ่มขึ้นหลังจาก Scale ครั้งก่อน (${purchases}/${lastPurchases} ออเดอร์) ต้องมีออเดอร์ใหม่เข้ามาจึงจะ Scale ต่อได้`,
          en: `No new purchases since last scale (${purchases}/${lastPurchases} orders). Requires new purchases to scale again`,
        })
      }
    }
  }

  const meetsPurchases = purchases >= minPurchases
  const meetsCPA = cost == null || cost <= maxCPA
  const meetsFreq = frequency <= maxFreq

  if (meetsPurchases && meetsCPA && meetsFreq) {
    return {
      eligible: true,
      inCooldown: cooldown.inCooldown,
      cooldownInfo: cooldown,
      reason: `${purchases} ອໍເດີ >= ${minPurchases}, CPA ${cost != null ? money(cost) : '—'} <= ${money(maxCPA)}${strat.isProductCustom ? ` (${strat.product?.name})` : ''}, Freq ${fmt(frequency, 2)} <= ${maxFreq}`
    }
  }
  return { eligible: false, inCooldown: cooldown.inCooldown, cooldownInfo: cooldown, reason: '' }
}

function adScaleBadge(item: MetaInsightItem) {
  const cooldown = getCooldownInfo(item)
  if (cooldown.inCooldown && cooldown.record) {
    return {
      type: 'cooldown',
      label: l({
        lo: `⏳ ສະເກວແລ້ວ (Cooldown ${cooldown.hoursLeft}h ${cooldown.minutesLeft}m)`,
        th: `⏳ Scale แล้ว (Cooldown ${cooldown.hoursLeft}h ${cooldown.minutesLeft}m)`,
        en: `⏳ Scaled (Cooldown ${cooldown.hoursLeft}h ${cooldown.minutesLeft}m)`,
      }),
      detail: l({
        lo: `ປັບເມື່ອ ${new Date(cooldown.record.scaledAt).toLocaleDateString()} ${new Date(cooldown.record.scaledAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}: ${money(cooldown.record.oldBudget)} ➔ ${money(cooldown.record.newBudget)}/ມື້ (+${cooldown.record.scalePercent}%)`,
        th: `ปรับเมื่อ ${new Date(cooldown.record.scaledAt).toLocaleDateString()} ${new Date(cooldown.record.scaledAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}: ${money(cooldown.record.oldBudget)} ➔ ${money(cooldown.record.newBudget)}/วัน (+${cooldown.record.scalePercent}%)`,
        en: `Scaled at ${new Date(cooldown.record.scaledAt).toLocaleDateString()} ${new Date(cooldown.record.scaledAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}: ${money(cooldown.record.oldBudget)} ➔ ${money(cooldown.record.newBudget)}/day (+${cooldown.record.scalePercent}%)`,
      }),
      color: '#d97706',
      bgColor: 'rgba(245, 158, 11, 0.12)',
    }
  }

  const stop = isStopLossCandidate(item)
  if (stop.eligible) {
    return {
      type: 'stop',
      label: l({ lo: '🛑 ເຂົ້າເກນ Stop-Loss', th: '🛑 เข้าเกณฑ์ Stop-Loss', en: '🛑 Stop-Loss' }),
      detail: stop.reason,
      color: '#dc2626',
      bgColor: 'rgba(239, 68, 68, 0.12)',
    }
  }

  const scale = isScaleCandidate(item)
  if (scale.eligible && !scale.inCooldown) {
    return {
      type: 'scale_ready',
      label: l({ lo: `🚀 ພ້ອມ Scale (+${autoPilotConfig.value.scalePercent || 20}%)`, th: `🚀 พร้อม Scale (+${autoPilotConfig.value.scalePercent || 20}%)`, en: `🚀 Ready to Scale (+${autoPilotConfig.value.scalePercent || 20}%)` }),
      detail: scale.reason,
      color: '#16a34a',
      bgColor: 'rgba(22, 163, 74, 0.12)',
    }
  }

  return null
}

async function executeScaleItem(item: MetaInsightItem, scalePct: number): Promise<{ success: boolean; oldBudget: number; newBudget: number; error?: string }> {
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

  let calculated = base * (1 + scalePct / 100)
  if (currencyOffset.value === 1) {
    calculated = Math.max(20000, Math.round(calculated / 10000) * 10000)
  } else {
    calculated = Math.max(1, Math.round(calculated * 100) / 100)
  }

  // Check safety cap
  if (autoPilotConfig.value.maxDailyBudgetCap && autoPilotConfig.value.maxDailyBudgetCap > 0) {
    if (base >= autoPilotConfig.value.maxDailyBudgetCap) {
      return { success: false, oldBudget: base, newBudget: base, error: 'Reached max daily budget cap' }
    }
    if (calculated > autoPilotConfig.value.maxDailyBudgetCap) {
      calculated = autoPilotConfig.value.maxDailyBudgetCap
    }
  }

  const apiBudget = Math.round(calculated * currencyOffset.value)

  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${targetType}/${encodeURIComponent(targetId)}/budget`, {
      method: 'PATCH',
      body: { dailyBudget: apiBudget },
    })
    item.dailyBudget = apiBudget
    recordScaleEvent(item, base, calculated, scalePct, 'AUTO')
    return { success: true, oldBudget: base, newBudget: calculated }
  } catch (err: any) {
    return { success: false, oldBudget: base, newBudget: calculated, error: err?.data?.error?.message || err?.message }
  }
}

async function executePauseItem(item: MetaInsightItem): Promise<{ success: boolean; error?: string }> {
  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${item.entityType}/${encodeURIComponent(item.entityId)}/status`, {
      method: 'PATCH',
      body: { status: 'PAUSED' }
    })
    const campaign = campaigns.value.find(value => value.id === item.entityId)
    if (campaign) campaign.effectiveStatus = 'PAUSED'
    item.status = 'PAUSED'
    item.effectiveStatus = 'PAUSED'
    return { success: true }
  } catch (err: any) {
    return { success: false, error: err?.data?.error?.message || err?.message }
  }
}

async function runAutoPilot(isManual = false) {
  if (!report.value?.items?.length) return
  if (!autoPilotConfig.value.enabled && !isManual) return
  if (runningAutoPilot.value) return

  runningAutoPilot.value = true
  let scaledCount = 0
  let pausedCount = 0
  let cooldownCount = 0

  const itemsToCheck = (report.value?.items || []).filter(item => {
    // 1. Target Structure check (All vs Campaign CBO vs AdSet ABO)
    if (autoPilotConfig.value.targetType !== 'all') {
      if (item.entityType !== autoPilotConfig.value.targetType) return false
    }
    // 2. Product Scope check (All vs Only currently filtered product e.g. kito AG29)
    if (autoPilotConfig.value.scope === 'filtered_product' && selectedProductId.value !== 'all') {
      const linked = getLinkedProduct(item)
      if (linked?.product.id !== selectedProductId.value) return false
    }
    return true
  })

  for (const item of itemsToCheck) {
    // 1. Stop-Loss Check
    const stopCheck = isStopLossCandidate(item)
    if (stopCheck.eligible) {
      const res = await executePauseItem(item)
      addAutoPilotLog({
        id: 'log-' + Date.now() + '-' + Math.random().toString(36).slice(2, 7),
        timestamp: new Date().toISOString(),
        action: 'PAUSE',
        entityId: item.entityId,
        entityType: item.entityType,
        entityName: item.entityName,
        detail: l({
          lo: `[Stop-Loss] ຢຸດແອດຂາດທຶນ: ${stopCheck.reason}`,
          th: `[Stop-Loss] หยุดแอดขาดทุน: ${stopCheck.reason}`,
          en: `[Stop-Loss] Paused losing ad: ${stopCheck.reason}`,
        }),
        success: res.success,
      })
      if (res.success) pausedCount++
      continue
    }

    // 2. Auto-Scale Check
    const scaleCheck = isScaleCandidate(item)
    if (scaleCheck.eligible) {
      if (scaleCheck.inCooldown) {
        cooldownCount++
        continue
      }
      const scalePct = autoPilotConfig.value.scalePercent || 20
      const res = await executeScaleItem(item, scalePct)
      addAutoPilotLog({
        id: 'log-' + Date.now() + '-' + Math.random().toString(36).slice(2, 7),
        timestamp: new Date().toISOString(),
        action: 'SCALE',
        entityId: item.entityId,
        entityType: item.entityType,
        entityName: item.entityName,
        detail: l({
          lo: `[Auto-Scale] ເພີ່ມງົບ +${scalePct}%: ${scaleCheck.reason} (ເລີ່ມ Cooldown ${autoPilotConfig.value.scaleCooldownHours}h)`,
          th: `[Auto-Scale] เพิ่มงบ +${scalePct}%: ${scaleCheck.reason} (เริ่ม Cooldown ${autoPilotConfig.value.scaleCooldownHours}h)`,
          en: `[Auto-Scale] Scaled budget +${scalePct}%: ${scaleCheck.reason} (${autoPilotConfig.value.scaleCooldownHours}h cooldown started)`,
        }),
        beforeValue: money(res.oldBudget) + '/ມື້',
        afterValue: money(res.newBudget) + '/ມື້',
        success: res.success,
      })
      if (res.success) {
        scaledCount++
      }
    }
  }

  runningAutoPilot.value = false

  if (isManual || scaledCount > 0 || pausedCount > 0) {
    const parts: string[] = []
    if (scaledCount > 0) parts.push(l({ lo: `ສະເກວ ${scaledCount} ແອດ`, th: `Scale ${scaledCount} แอด`, en: `Scaled ${scaledCount} ad(s)` }))
    if (pausedCount > 0) parts.push(l({ lo: `ຢຸດ ${pausedCount} ແອດ`, th: `หยุด ${pausedCount} แอด`, en: `Paused ${pausedCount} ad(s)` }))
    if (cooldownCount > 0) parts.push(l({ lo: `ຂ້າມ ${cooldownCount} ແອດທີ່ຕິດ Cooldown 24h`, th: `ข้าม ${cooldownCount} แอดที่ติด Cooldown 24h`, en: `Skipped ${cooldownCount} ad(s) in 24h cooldown` }))

    if (parts.length > 0) {
      success.value = l({
        lo: `🤖 Auto-Pilot ປະມວນຜົນສຳເລັດ: ${parts.join(', ')}`,
        th: `🤖 Auto-Pilot ประมวลผลเสร็จสิ้น: ${parts.join(', ')}`,
        en: `🤖 Auto-Pilot execution finished: ${parts.join(', ')}`,
      })
    } else if (isManual) {
      notice.value = l({
        lo: '🤖 Auto-Pilot: ກວດສອບແລ້ວ ທຸກແອດຢູ່ໃນເກນປົກກະຕິ (ບໍ່ມີແອດທີ່ຕ້ອງສະເກວ ຫຼື ຢຸດຂາດທຶນໃນຕອນນີ້).',
        th: '🤖 Auto-Pilot: ตรวจสอบแล้ว ทุกแอดอยู่ในเกณฑ์ปกติ (ไม่มีแอดที่ต้อง Scale หรือหยุดขาดทุนในขณะนี้)',
        en: '🤖 Auto-Pilot: Checked. All ads are currently healthy (no ads meet scale or stop-loss criteria right now).',
      })
    }
  }
}

function addAutoPilotLog(log: AutoPilotLog) {
  autoPilotLogs.value.unshift(log)
  if (autoPilotLogs.value.length > 60) {
    autoPilotLogs.value = autoPilotLogs.value.slice(0, 60)
  }
  saveAutoPilotLogs()
}

watch(adAccountId, async () => {
  if (!ready.value) return
  loadThresholds()
  loadAutoPilotConfig()
  loadScaleHistory()
  await loadPerformance()
})
watch(level, async () => { if (ready.value) await loadPerformance() })
watch(() => route.query.product, (newProd) => {
  if (newProd) {
    selectedProductId.value = String(newProd)
  }
})
onMounted(async () => {
  if (route.query.product) {
    selectedProductId.value = String(route.query.product)
  }
  await Promise.all([loadAccounts(), loadAutomationsAndProducts()])
  loadThresholds()
  loadAutoPilotConfig()
  loadScaleHistory()
  ready.value = true
  if (adAccountId.value) await loadPerformance()
})
</script>

<template>
  <!-- Shared Ads Suite Navigation -->
  <nav class="app-sub-nav" aria-label="Ads suite navigation">
    <NuxtLink to="/ads" class="app-sub-nav-item active">
      <v-icon icon="mdi-bullhorn-outline" size="18" />
      <span>{{ l({ lo: '1. Ads Manager (ນຳສົ່ງ & ຄ່າແອດ)', th: '1. Ads Manager (นำส่ง & ค่าแอด)', en: '1. Ads Manager' }) }}</span>
    </NuxtLink>
    <NuxtLink to="/ads-intelligence" class="app-sub-nav-item">
      <v-icon icon="mdi-chart-timeline-variant-shimmer" size="18" />
      <span>{{ l({ lo: '2. Ads Intelligence (ກຳໄລ, Real CPA & SOP)', th: '2. Ads Intelligence (กำไร, Real CPA & SOP)', en: '2. Ads Intelligence' }) }}</span>
    </NuxtLink>
    <NuxtLink to="/ads-battle" class="app-sub-nav-item">
      <v-icon icon="mdi-scale-balance" size="18" />
      <span>{{ l({ lo: '3. Ads Battle (ປຽບທຽບ A/B & ຜູ້ຊື້)', th: '3. Ads Battle (เปรียบเทียบ A/B & ผู้ซื้อ)', en: '3. Ads Battle' }) }}</span>
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
      <!-- Live Tour & Guide -->
      <v-btn
        color="primary"
        variant="tonal"
        prepend-icon="mdi-crosshairs-gps"
        class="font-weight-bold"
        @click="startLiveTour"
      >
        {{ l({ lo: '🎯 ພາທົວ (Tour)', th: '🎯 พาทัวร์ (Tour)', en: '🎯 Live Tour' }) }}
      </v-btn>

      <v-btn
        variant="outlined"
        prepend-icon="mdi-book-open-page-variant-outline"
        @click="guideModalOpen = true; guideStep = 1"
      >
        {{ l({ lo: '📖 ຄູ່ມື (Guide)', th: '📖 คู่มือ (Guide)', en: '📖 Guide' }) }}
      </v-btn>

      <!-- Auto-Pilot Toggle & Modal Trigger Button -->
      <v-btn
        :color="autoPilotConfig.enabled ? 'success' : undefined"
        :variant="autoPilotConfig.enabled ? 'flat' : 'outlined'"
        :prepend-icon="autoPilotConfig.enabled ? 'mdi-robot' : 'mdi-robot-off-outline'"
        class="autopilot-head-btn font-weight-bold"
        @click="autoPilotModalOpen = true"
      >
        <span>{{ autoPilotConfig.enabled ? l({ lo: 'Auto-Pilot: ON', th: 'Auto-Pilot: เปิด', en: 'Auto-Pilot: ON' }) : l({ lo: 'Auto-Pilot: OFF', th: 'Auto-Pilot: ปิด', en: 'Auto-Pilot: OFF' }) }}</span>
        <span v-if="autoPilotConfig.enabled" class="autopilot-dot-pulse" />
      </v-btn>

      <v-btn v-if="actionCandidates.length > 0" color="warning" variant="tonal" prepend-icon="mdi-shield-check-outline" @click="openActionPlan">
        {{ l({ lo: `ກວດແຜນປັບ Ads (${actionCandidates.length})`, th: `ตรวจแผนปรับ Ads (${actionCandidates.length})`, en: `Review plan (${actionCandidates.length})` }) }}
      </v-btn>
      <v-btn variant="outlined" prepend-icon="mdi-tune-variant" @click="openStrategyModal">
        {{ l({ lo: 'ຍຸດທະສາດ (Strategy)', th: 'ยุทธศาสตร์ (Strategy)', en: 'Strategy' }) }}
      </v-btn>
      <v-btn color="primary" variant="flat" prepend-icon="mdi-refresh" :loading="loadingReport" :disabled="!adAccountId" @click="loadPerformance">
        {{ l({ lo: 'ອັບເດດ', th: 'อัปเดต', en: 'Refresh' }) }}
      </v-btn>
    </div>
  </section>

  <v-alert v-if="notice" type="error" variant="tonal" class="mb-4" closable @click:close="notice = ''">
    {{ notice }}
    <template #append><v-btn size="small" variant="outlined" to="/meta-pages">{{ l({ lo: 'ກວດສິດ Meta', th: 'ตรวจสิทธิ์ Meta', en: 'Check Meta access' }) }}</v-btn></template>
  </v-alert>
  <v-alert v-if="success" type="success" variant="tonal" class="mb-4" closable @click:close="success = ''">{{ success }}</v-alert>

  <!-- Dual-System Mode Banner -->
  <section class="system-mode-banner" id="tour-mode-banner" :class="{ 'autopilot-active': autoPilotConfig.enabled, 'tour-spotlight-active': isLiveTourActive && liveTourStep === 4 }">
    <div class="mode-badge-wrap">
      <span class="mode-pill" :class="autoPilotConfig.enabled ? 'autopilot' : 'manual'">
        <v-icon :icon="autoPilotConfig.enabled ? 'mdi-robot' : 'mdi-hand-back-right-outline'" size="18" />
        {{ autoPilotConfig.enabled
          ? l({ lo: 'ໂໝດ 2: Auto-Pilot (ບັອດອັດຕະໂນມັດ)', th: 'โหมด 2: Auto-Pilot (บอทอัตโนมัติ)', en: 'Mode 2: Auto-Pilot (Active)' })
          : l({ lo: 'ໂໝດ 1: Manual Approval (ກວດສອບ & ກົດເອງ)', th: 'โหมด 1: Manual Approval (ควบคุมเอง)', en: 'Mode 1: Manual Approval (Human)' })
        }}
      </span>
      <div class="mode-desc">
        <strong v-if="autoPilotConfig.enabled" class="mode-title-text text-success font-weight-bold">
          {{ l({ lo: '🤖 ບັອດກຳລັງເຝົ້າແອດ: Auto-Scale ແອດຂາຍດີ (+20%), Lock Cooldown 24h ບໍ່ໃຫ້ຫຼຸດ Learning Phase, ແລະ Auto-Stop ແອດຂາດທຶນ', th: '🤖 บอทกำลังเฝ้าแอด: Auto-Scale แอดขายดี (+20%), ล็อก Cooldown 24h ไม่ให้หลุด Learning Phase, และ Auto-Stop แอดขาดทุน', en: '🤖 Bot is watching ads: Auto-scaling winners (+20%), locking 24h cooldown to protect learning phase, and auto-pausing losers' }) }}
        </strong>
        <strong v-else class="mode-title-text font-weight-bold">
          {{ l({ lo: '👤 ໂໝດຄວບຄຸມດ້ວຍຕົນເອງ: ທ່ານສາມາດກົດປຸ່ມ Scale ຫຼື ອະນຸມັດແຜນປິດ Ads ຕາມຄວາມພໍໃຈ (ລະບົບຈະບໍ່ປັບປ່ຽນຫຍັງເອງ)', th: '👤 โหมดควบคุมด้วยตนเอง: คุณสามารถกดปุ่ม Scale หรืออนุมัติแผนหยุด Ads ตามความต้องการ (ระบบจะไม่ปรับอะไรเอง)', en: '👤 Manual mode: You review and approve budget scaling or ad pausing manually (no automatic changes)' }) }}
        </strong>
        <div class="mode-rules-summary">
          <template v-if="autoPilotConfig.enabled">
            <span class="rule-chip"><v-icon icon="mdi-rocket-launch-outline" size="12" /> Scale: ຊື້ >= {{ autoPilotConfig.scaleMinPurchases }}, CPA <= {{ autoPilotConfig.scaleMaxCPA ? money(autoPilotConfig.scaleMaxCPA) : (targetCost ? money(targetCost) : 'ເປົ້າໝາຍ') }}, Freq <= {{ autoPilotConfig.scaleMaxFrequency }} (+{{ autoPilotConfig.scalePercent }}%)</span>
            <span class="rule-chip"><v-icon icon="mdi-pause-circle-outline" size="12" /> Stop-Loss: Spend >= {{ autoPilotConfig.stopMinSpend ? money(autoPilotConfig.stopMinSpend) : money(currencyOffset === 1 ? 50000 : 5) }}, 0 ອໍເດີ, CTR < {{ autoPilotConfig.stopMaxCTR }}%</span>
            <span class="rule-chip"><v-icon icon="mdi-shield-lock-outline" size="12" /> Cooldown: 24h Lock</span>
          </template>
          <template v-else>
            <span class="rule-chip-muted">{{ l({ lo: 'ຄຳແນະນຳທັງໝົດຍັງສະແດງໃນຕາຕະລາງ, ແຕ່ຈະບໍ່ມີການປ່ຽນງົບ ຫຼື ສະຖານະໂດຍບໍ່ໄດ້ຮັບອະນຸມັດຈາກທ່ານ.', th: 'คำแนะนำทั้งหมดยังแสดงในตาราง แต่จะไม่เปลี่ยนงบหรือสถานะโดยไม่ได้รับอนุมัติจากคุณ', en: 'Recommendations remain visible, but no budgets or statuses will change without your explicit click.' }) }}</span>
          </template>
        </div>
      </div>
    </div>

    <div class="mode-controls">
      <div class="mode-switch-box">
        <span class="mode-switch-label">{{ l({ lo: 'ສະວິດ Auto-Pilot:', th: 'สวิตช์ Auto-Pilot:', en: 'Auto-Pilot Switch:' }) }}</span>
        <v-switch
          v-model="autoPilotConfig.enabled"
          color="success"
          hide-details
          density="compact"
          class="mode-v-switch"
          @update:model-value="toggleAutoPilotMaster"
        />
      </div>
      <v-btn
        v-if="autoPilotConfig.enabled"
        size="small"
        color="success"
        variant="flat"
        prepend-icon="mdi-flash"
        :loading="runningAutoPilot"
        class="font-weight-bold"
        @click="runAutoPilot(true)"
      >
        {{ l({ lo: '⚡ ປະມວນຜົນດຽວນີ້', th: '⚡ ประมวลผลตอนนี้', en: '⚡ Run Now' }) }}
      </v-btn>
      <v-btn
        size="small"
        variant="tonal"
        color="primary"
        prepend-icon="mdi-tune-vertical"
        class="font-weight-bold"
        @click="autoPilotModalOpen = true"
      >
        {{ l({ lo: 'ຕັ້ງຄ່າກົດ & Log', th: 'ตั้งค่ากฎ & Log', en: 'Rules & Logs' }) }}
      </v-btn>
    </div>
  </section>

  <!-- Modern Standardized Command & Filter Bar -->
  <section class="app-filter-bar" id="tour-filter-bar" :class="{ 'tour-spotlight-active': isLiveTourActive && liveTourStep === 2 }" aria-label="Ads report filters">
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
      <v-btn
        v-if="selectedProductId !== 'all' && selectedProductId !== 'unlinked'"
        size="small"
        variant="tonal"
        color="primary"
        prepend-icon="mdi-tune"
        class="font-weight-bold"
        @click="openProductStrategy(selectedProductId)"
      >
        {{ l({ lo: 'ຍຸດທະສາດສິນຄ້ານີ້', th: 'ยุทธศาสตร์สินค้านี้', en: 'Product Strategy' }) }}
      </v-btn>
      <div class="app-filter-divider" />
      <v-select
        :model-value="datePreset"
        :items="[
          { title: l({ lo: '⚡ ມື້ນີ້ (Today)', th: '⚡ วันนี้ (Today)', en: '⚡ Today' }), value: 'today' },
          { title: l({ lo: '📅 ມື້ວານນີ້ (Yesterday)', th: '📅 เมื่อวาน (Yesterday)', en: '📅 Yesterday' }), value: 'yesterday' },
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
    <!-- Active Filter Feedback Pill -->
    <div v-if="isFiltered" class="d-flex align-center justify-space-between mb-3 px-1 flex-wrap gap-2">
      <div class="d-flex align-center gap-2 flex-wrap">
        <v-chip size="small" color="primary" variant="tonal" prepend-icon="mdi-filter-check" class="font-weight-bold">
          {{ l({ lo: `ສະຫຼຸບຕົວເລກສະເພາະ Filter (${items.length} ລາຍການ)`, th: `สรุปตัวเลขเฉพาะ Filter (${items.length} รายการ)`, en: `Filtered metrics (${items.length} items)` }) }}
        </v-chip>
        <v-chip v-if="outcomeFilter !== 'all'" size="small" color="amber-darken-3" variant="flat" class="font-weight-bold elevation-1">
          <v-icon icon="mdi-filter-variant" start size="13" />
          {{ outcomeFilter === 'angel' ? '🧚‍♀️ Ads ນາງຟ້າ' : outcomeFilter === 'purchases' ? '🛒 ມີການຊື້' : outcomeFilter === 'messages' ? '💬 ມີຄົນທັກ' : outcomeFilter === 'scale' ? '🚀 ພ້ອມ Scale' : '⏸️ ຄວນປິດ' }}
        </v-chip>
      </div>
      <v-btn size="x-small" variant="text" color="primary" prepend-icon="mdi-refresh" @click="selectedProductId='all';statusFilter='all';outcomeFilter='all';search=''">
        {{ l({ lo: 'ເບິ່ງລວມທັງໝົດບັນຊີ', th: 'ดูรวมทั้งบัญชี', en: 'View full account' }) }}
      </v-btn>
    </div>

    <!-- Modern 6-KPI Cards Grid with VAT Breakdown and Messaging Conversations -->
    <section class="app-kpi-grid" id="tour-kpi-summary" :class="{ 'tour-spotlight-active': isLiveTourActive && liveTourStep === 3 }" aria-label="Ads summary">
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

      <!-- KPI Card 2: Messaging Conversations (ຄົນສົ່ງຂໍ້ຄວາມ) -->
      <article class="app-kpi-card accent-info">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ຄົນສົ່ງຂໍ້ຄວາມ (Conversations)', th: 'คนทักข้อความ (Conversations)', en: 'Conversations Started' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-message-text-outline" size="16" /></span>
        </div>
        <strong class="kpi-value">
          {{ getMessagesCount(summary) != null ? fmt(getMessagesCount(summary), 0) : (isMessagingAction(summary?.results?.type) && summary?.results?.value != null ? fmt(summary.results.value, 0) : '0') }}
          <small class="text-body-2 font-weight-medium">ຄົນ</small>
        </strong>
        <div class="kpi-sub">
          <span>{{ l({ lo: 'ຕົ້ນທຶນ/ຂໍ້ຄວາມ:', th: 'ต้นทุน/ข้อความ:', en: 'Cost/Msg:' }) }} <b>{{ getCostPerMessage(summary) != null ? money(getCostPerMessage(summary)) : (isMessagingAction(summary?.results?.type) && summary?.results?.cost != null ? money(summary.results.cost) : '—') }}</b></span>
        </div>
      </article>

      <!-- KPI Card 3: Purchases / Results (ການຊື້ / ຜົນລັບ) -->
      <article class="app-kpi-card accent-success">
        <div class="kpi-header">
          <span class="kpi-label">{{ getPurchasesCount(summary) != null ? l({ lo: 'ການຊື້ (Purchases / Orders)', th: 'การซื้อ (Purchases / Orders)', en: 'Purchases / Orders' }) : resultLabel(summary?.results?.type) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-cart-check" size="16" /></span>
        </div>
        <strong class="kpi-value">
          {{ getPurchasesCount(summary) != null ? fmt(getPurchasesCount(summary), 0) + ' ອໍເດີ' : fmt(summary?.results?.value, 0) }}
        </strong>
        <div class="kpi-sub">
          <span>{{ getPurchasesCount(summary) != null ? 'CPA: ' : l({ lo: 'ຕົ້ນທຶນ/ຜົນ:', th: 'ต้นทุน/ผล:', en: 'Cost/res:' }) }} <b>{{ getCostPerPurchase(summary) != null ? money(getCostPerPurchase(summary)) : money(summary?.results?.cost) }}</b></span>
        </div>
      </article>

      <!-- KPI Card 4: CTR -->
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

      <!-- KPI Card 5: Frequency & Reach -->
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

      <!-- KPI Card 6: ROAS & Conversion Value -->
      <article class="app-kpi-card accent-info">
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
        <div class="card-heading"><div><p>DECISION GUIDE</p><h2>{{ l({lo:'ອ່ານສັນຍານໃຫ້ງ່າຍ',th:'อ่านสัญญาณให้ง่าย',en:'Decision signals'}) }}</h2></div><v-btn icon="mdi-tune-variant" variant="text" :aria-label="l({lo:'ຕັ້ງຄ່າຍຸດທະສາດ',th:'ตั้งค่ายุทธศาสตร์',en:'Setup strategy'})" @click="openStrategyModal" /></div>
        <div class="decision-summary"><div class="decision-count keep"><strong>{{ decisionCounts.keep }}</strong><span>{{ l({lo:'ເປີດຕໍ່',th:'เปิดต่อ',en:'Keep'}) }}</span></div><div class="decision-count review"><strong>{{ decisionCounts.review }}</strong><span>{{ l({lo:'ກວດສອບ',th:'ตรวจสอบ',en:'Review'}) }}</span></div><div class="decision-count pause"><strong>{{ decisionCounts.pause }}</strong><span>{{ l({lo:'ພິຈາລະນາປິດ',th:'พิจารณาปิด',en:'Pause candidates'}) }}</span></div></div>
        <p class="guide-note"><v-icon icon="mdi-information-outline" /> {{ l({lo:'ຄຳແນະນຳໃຊ້ຫຼາຍ metric ຮ່ວມກັນ. ຢ່າປິດເພາະມື້ດຽວ ຫຼື metric ດຽວ.',th:'คำแนะนำใช้หลาย metric ร่วมกัน อย่าปิดเพราะวันเดียวหรือค่าเดียว',en:'Recommendations combine several metrics. Never pause from one day or one metric alone.'}) }}</p>
        <ol class="decision-steps">
          <li><strong>{{ l({lo:'1. ເບິ່ງຜົນທາງທຸລະກິດ',th:'1. ดูผลทางธุรกิจ',en:'1. Check business outcomes'}) }}</strong><span>{{ l({lo:'Cost/Result ຕ້ອງບໍ່ເກີນຈຸດຄຸ້ມທຶນ.',th:'Cost/Result ต้องไม่เกินจุดคุ้มทุน',en:'Cost/result should stay below your break-even point.'}) }}</span></li>
          <li><strong>{{ l({lo:'2. ເບິ່ງຄຸນນະພາບ Creative',th:'2. ดูคุณภาพครีเอทีฟ',en:'2. Check creative quality'}) }}</strong><span>{{ l({lo:'CTR ລົງພ້ອມ Cost ສູງຂຶ້ນ ໝາຍເຖິງຄວນປ່ຽນ Creative.',th:'CTR ลดพร้อมต้นทุนสูงขึ้น หมายถึงควรเปลี่ยนครีเอทีฟ',en:'Falling CTR with rising cost suggests a creative refresh.'}) }}</span></li>
          <li><strong>{{ l({lo:'3. ເບິ່ງ Audience saturation',th:'3. ดูความอิ่มตัวของกลุ่มเป้าหมาย',en:'3. Check audience saturation'}) }}</strong><span>{{ l({lo:'Coverage ແລະ Frequency ສູງພ້ອມຜົນງານຕົກ ຈຶ່ງຄ່ອຍພິຈາລະນາປິດ.',th:'Coverage และ Frequency สูงพร้อมผลงานตก จึงค่อยพิจารณาหยุด',en:'Consider pausing only when high coverage and frequency coincide with declining results.'}) }}</span></li>
        </ol>
      </v-card>
    </section>

    <section class="performance-section" id="tour-performance-section" :class="{ 'tour-spotlight-active': isLiveTourActive && liveTourStep === 5 }">
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
              { title: l({ lo: '🛒 ອໍເດີສູງສຸດ', th: '🛒 ออเดอร์สูงสุด', en: 'Highest purchases' }), value: 'purchases' },
              { title: l({ lo: '💬 ຄົນທັກຫຼາຍສຸດ', th: '💬 ทักแชตสูงสุด', en: 'Most messages' }), value: 'messages' },
              { title: l({ lo: '⚡ ໃຊ້ຈ່າຍສູງສຸດ', th: '⚡ ใช้จ่ายสูงสุด', en: 'Highest spend' }), value: 'spend' },
              { title: l({ lo: '📈 ROAS ສູງສຸດ', th: '📈 ROAS สูงสุด', en: 'Highest ROAS' }), value: 'roas' },
              { title: l({ lo: 'ສ້າງໃໝ່ສຸດ', th: 'สร้างใหม่ล่าสุด', en: 'Newest created' }), value: 'newest' },
              { title: l({ lo: '🔤 ຊື່ (A-Z)', th: '🔤 ชื่อ (A-Z)', en: 'Name (A-Z)' }), value: 'name' }
            ]"
            item-title="title"
            item-value="value"
            density="compact"
            variant="outlined"
            hide-details
            class="sort-select"
            style="min-width: 175px; max-width: 215px;"
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

      <!-- Quick Outcome / Conversion / Angel Ads Filter Bar (ການຊື້, ຄົນທັກ, Ads ນາງຟ້າ) -->
      <div class="outcome-filter-strip">
        <div class="outcome-strip-header">
          <span class="outcome-strip-label">
            <v-icon icon="mdi-filter-cog-outline" size="16" class="text-primary" />
            <strong>{{ l({ lo: 'ກອງຕາມຜົນລັບ & Ads ດາວເດັ່ນ:', th: 'กรองตามผลลัพธ์ & แอดดาวเด่น:', en: 'Filter by Outcomes & Winners:' }) }}</strong>
          </span>
          <span v-if="outcomeCounts.angel > 0" class="angel-found-pill">
            <v-icon icon="mdi-star" size="12" />
            {{ l({ lo: `ພົບ Ads ນາງຟ້າ ${outcomeCounts.angel} ໂຕ`, th: `พบแอดนางฟ้า ${outcomeCounts.angel} ตัว`, en: `Found ${outcomeCounts.angel} Angel Ads` }) }}
          </span>
        </div>
        <div class="outcome-tabs">
          <button
            type="button"
            class="outcome-tab-btn"
            :class="{ active: outcomeFilter === 'all' }"
            @click="outcomeFilter = 'all'"
          >
            <span>{{ l({ lo: '📋 ທັງໝົດ', th: '📋 ทั้งหมด', en: '📋 All' }) }}</span>
            <b class="count-pill">{{ outcomeCounts.all }}</b>
          </button>

          <button
            type="button"
            class="outcome-tab-btn angel-tab"
            :class="{ active: outcomeFilter === 'angel' }"
            @click="outcomeFilter = 'angel'"
          >
            <v-icon icon="mdi-star-face" size="15" class="angel-icon" />
            <span>{{ l({ lo: '🧚‍♀️ Ads ນາງຟ້າ (Winning)', th: '🧚‍♀️ แอดนางฟ้า (Winning)', en: '🧚‍♀️ Angel Ads' }) }}</span>
            <b class="count-pill gold">{{ outcomeCounts.angel }}</b>
          </button>

          <button
            type="button"
            class="outcome-tab-btn purchase-tab"
            :class="{ active: outcomeFilter === 'purchases' }"
            @click="outcomeFilter = 'purchases'"
          >
            <v-icon icon="mdi-cart-check" size="15" />
            <span>{{ l({ lo: '🛒 ມີການຊື້ (Purchases)', th: '🛒 มีการซื้อ (Purchases)', en: '🛒 Has Purchases' }) }}</span>
            <b class="count-pill green">{{ outcomeCounts.purchases }}</b>
          </button>

          <button
            type="button"
            class="outcome-tab-btn message-tab"
            :class="{ active: outcomeFilter === 'messages' }"
            @click="outcomeFilter = 'messages'"
          >
            <v-icon icon="mdi-message-text-outline" size="15" />
            <span>{{ l({ lo: '💬 ມີຄົນທັກ (Messages)', th: '💬 มีคนทัก (Messages)', en: '💬 Has Messages' }) }}</span>
            <b class="count-pill blue">{{ outcomeCounts.messages }}</b>
          </button>

          <button
            type="button"
            class="outcome-tab-btn scale-tab"
            :class="{ active: outcomeFilter === 'scale' }"
            @click="outcomeFilter = 'scale'"
          >
            <v-icon icon="mdi-rocket-launch" size="14" />
            <span>{{ l({ lo: '🚀 ພ້ອມ Scale', th: '🚀 พร้อม Scale', en: '🚀 Scale Ready' }) }}</span>
            <b class="count-pill purple">{{ outcomeCounts.scale }}</b>
          </button>

          <button
            type="button"
            class="outcome-tab-btn pause-tab"
            :class="{ active: outcomeFilter === 'pause' }"
            @click="outcomeFilter = 'pause'"
          >
            <v-icon icon="mdi-pause-circle-outline" size="15" />
            <span>{{ l({ lo: '⏸️ ຄວນປິດ', th: '⏸️ ควรปิด', en: '⏸️ Pause' }) }}</span>
            <b class="count-pill red">{{ outcomeCounts.pause }}</b>
          </button>
        </div>
      </div>

      <p class="filter-help"><v-icon icon="mdi-information-outline" /> {{ l({lo:'ສະຖານະນີ້ແມ່ນຄ່າຈິງຈາກ Meta. Ad ອາດຖືກຢຸດເພາະ Campaign ຫຼື Ad Set ຊັ້ນເທິງຖືກປິດ.',th:'สถานะนี้มาจาก Meta โดยตรง โฆษณาอาจหยุดเพราะแคมเปญหรือชุดโฆษณาด้านบนถูกปิด',en:'Statuses come directly from Meta. An ad may stop because its parent campaign or ad set is paused.'}) }}</p>

      <div v-if="loadingReport" class="table-loading"><v-progress-linear indeterminate color="primary" /></div>
      <div v-else-if="!items.length" class="empty-state">
        <v-icon icon="mdi-filter-off-outline" size="44" />
        <h3>{{ l({lo:'ບໍ່ພົບລາຍການຕາມ filter',th:'ไม่พบรายการตามตัวกรอง',en:'No items match these filters'}) }}</h3>
        <p>{{ l({lo:'ລອງເລືອກສະຖານະທັງໝົດ, ລ້າງການກອງຜົນລັບ ຫຼືລຶບຄຳຄົ້ນຫາ.',th:'ลองเลือกสถานะทั้งหมด ล้างตัวกรองผลลัพธ์ หรือล้างคำค้นหา',en:'Choose All statuses, reset outcome filter, or clear the search.'}) }}</p>
        <v-btn variant="tonal" color="primary" @click="statusFilter='all';outcomeFilter='all';search=''">
          {{ l({lo:'ລ້າງ filter ທັງໝົດ',th:'ล้างตัวกรองทั้งหมด',en:'Clear all filters'}) }}
        </v-btn>
      </div>

      <div v-else class="performance-list">
        <article
          v-for="item in items"
          :key="item.entityId"
          class="performance-row"
          :class="{
            expanded: selectedId === item.entityId,
            'is-angel-ad': isAngelAd(item).isAngel
          }"
        >
          <button class="row-main" type="button" :aria-expanded="selectedId === item.entityId" @click="toggleItem(item)">
            <span class="entity-cell">
              <span class="entity-icon" :class="{ 'angel-avatar': isAngelAd(item).isAngel }">
                <v-icon :icon="isAngelAd(item).isAngel ? 'mdi-star-face' : (item.entityType==='campaign'?'mdi-bullhorn-outline':item.entityType==='adset'?'mdi-account-group-outline':'mdi-advertisements')" />
              </span>
              <span>
                <div class="d-flex align-center gap-1 flex-wrap">
                  <strong>{{ item.entityName || item.entityId }}</strong>
                  <!-- Angel Ad Badge -->
                  <v-chip
                    v-if="isAngelAd(item).isAngel"
                    size="x-small"
                    color="amber-darken-3"
                    variant="flat"
                    class="font-weight-bold angel-badge elevation-1"
                    :title="isAngelAd(item).reason"
                  >
                    <v-icon icon="mdi-star" start size="11" />
                    🧚‍♀️ Ads ນາງຟ້າ
                  </v-chip>
                </div>
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
                  <v-btn
                    size="x-small"
                    variant="text"
                    color="primary"
                    density="compact"
                    icon="mdi-tune"
                    :title="l({ lo: 'ຕັ້ງຄ່າຍຸດທະສາດສິນຄ້ານີ້', th: 'ตั้งค่ายุทธศาสตร์สินค้านี้', en: 'Configure product strategy' })"
                    class="ml-1"
                    @click.stop="openProductStrategy(getLinkedProduct(item)!.product.id)"
                  />
                  <v-chip
                    v-if="buyingOpportunity(item).level !== 'neutral'"
                    size="x-small"
                    :color="buyingOpportunity(item).chipColor"
                    variant="tonal"
                    class="font-weight-bold"
                  >
                    {{ buyingOpportunity(item).badgeText }}
                  </v-chip>
                </div>
                <div v-else class="d-flex align-center gap-1 mt-1 flex-wrap">
                  <v-chip size="x-small" color="grey" variant="text" class="text-caption">
                    {{ l({ lo: '⚠️ ຍັງບໍ່ຜູກ Auto Reply', th: '⚠️ ยังไม่ผูก Auto Reply', en: '⚠️ Unlinked' }) }}
                  </v-chip>
                  <NuxtLink to="/auto-replies" class="text-caption text-primary text-decoration-none font-weight-medium" @click.stop>
                    + ຜູກແອດນີ້
                  </NuxtLink>
                  <v-chip
                    v-if="buyingOpportunity(item).level !== 'neutral'"
                    size="x-small"
                    :color="buyingOpportunity(item).chipColor"
                    variant="tonal"
                    class="font-weight-bold"
                  >
                    {{ buyingOpportunity(item).badgeText }}
                  </v-chip>
                </div>
                <!-- Prominent Conversion Badges (Messages & Orders & ROAS) -->
                <div v-if="getMessagesCount(item) != null || getPurchasesCount(item) != null || (item.roas != null && item.roas > 0)" class="d-flex align-center gap-1 mt-1 flex-wrap">
                  <v-chip v-if="getPurchasesCount(item) != null" size="x-small" color="success" variant="flat" class="font-weight-bold elevation-1">
                    <v-icon icon="mdi-cart-check" start size="12" />
                    {{ fmt(getPurchasesCount(item), 0) }} ອໍເດີ
                    <template v-if="getCostPerPurchase(item)"> (CPA: {{ money(getCostPerPurchase(item)) }})</template>
                  </v-chip>
                  <v-chip v-if="getMessagesCount(item) != null" size="x-small" color="info" variant="tonal" class="font-weight-bold">
                    <v-icon icon="mdi-message-text-outline" start size="12" />
                    {{ fmt(getMessagesCount(item), 0) }} ຂໍ້ຄວາມ
                    <template v-if="getCostPerMessage(item)"> ({{ money(getCostPerMessage(item)) }}/ຂໍ້ຄວາມ)</template>
                  </v-chip>
                  <v-chip v-if="item.roas != null && item.roas > 0" size="x-small" color="purple" variant="tonal" class="font-weight-bold">
                    <v-icon icon="mdi-chart-line" start size="12" />
                    ROAS {{ fmt(item.roas, 2) }}x
                  </v-chip>
                </div>
              </span>
            </span>
            <span class="delivery-chip" :class="statusClass(statusFor(item))"><span class="status-dot" :class="statusClass(statusFor(item))" />{{ statusLabel(statusFor(item)) }}</span>
            <span class="metric metric-spend mobile-key"><small>{{ l({lo:'ໃຊ້ຈ່າຍ',th:'ใช้จ่าย',en:'Spend'}) }}</small><strong>{{ money(item.spend) }}</strong></span>
            <span class="metric metric-results mobile-key">
              <small>{{ resultLabel(item.results.type) }}</small>
              <strong>{{ fmt(item.results.value, 0) }}</strong>
              <small v-if="getMessagesCount(item) != null && !isMessagingAction(item.results?.type)" class="text-primary font-weight-bold">
                💬 {{ fmt(getMessagesCount(item), 0) }} ຂໍ້ຄວາມ
              </small>
            </span>
            <span class="metric metric-cost mobile-key">
              <small>{{ l({lo:'ຕົ້ນທຶນ/ຜົນ',th:'ต้นทุน/ผล',en:'Cost/result'}) }}</small>
              <strong>{{ money(item.results.cost) }}</strong>
              <small v-if="getCostPerMessage(item) != null && !isMessagingAction(item.results?.type)" class="text-medium-emphasis">
                {{ money(getCostPerMessage(item)) }}/ຂໍ້ຄວາມ
              </small>
            </span>
            <span class="metric metric-ctr mobile-key"><small>CTR</small><strong>{{ percent(item.linkCtr) }}</strong></span>
            <span class="metric metric-frequency"><small>Frequency</small><strong>{{ fmt(item.frequency, 2) }}</strong></span>
            <span class="decision-chip" :class="decision(item).code">
              <v-icon :icon="decision(item).icon" />
              <span>
                <strong>{{ decision(item).title }}</strong>
                <small>{{ decision(item).signals[0] || decision(item).detail }}</small>
                <span
                  v-if="adScaleBadge(item)"
                  class="ad-scale-status-pill mt-1"
                  :class="adScaleBadge(item)!.type"
                  :title="adScaleBadge(item)!.detail"
                >
                  {{ adScaleBadge(item)!.label }}
                </span>
              </span>
            </span>
            <div class="row-actions" @click.stop>
              <v-btn
                size="x-small"
                :variant="getCooldownInfo(item).inCooldown ? 'tonal' : (isScaleCandidate(item).eligible ? 'flat' : 'outlined')"
                :color="getCooldownInfo(item).inCooldown ? 'warning' : (isScaleCandidate(item).eligible ? 'primary' : 'default')"
                :prepend-icon="getCooldownInfo(item).inCooldown ? 'mdi-clock-outline' : (isScaleCandidate(item).eligible ? 'mdi-rocket-launch' : 'mdi-tune-vertical')"
                class="quick-scale-btn font-weight-bold"
                @click="openBudgetModal(item)"
              >
                {{
                  getCooldownInfo(item).inCooldown
                    ? `Cooldown (${getCooldownInfo(item).hoursLeft}h)`
                    : (isScaleCandidate(item).eligible ? '🚀 Scale (+20%)' : l({ lo: '💰 ປັບງົບ', th: '💰 ปรับงบ', en: '💰 Budget' }))
                }}
              </v-btn>
              <v-icon :icon="selectedId === item.entityId ? 'mdi-chevron-up' : 'mdi-chevron-down'" class="expand-chevron" />
            </div>
            <div class="mobile-scale-bar" @click.stop>
              <v-btn
                size="small"
                :variant="getCooldownInfo(item).inCooldown ? 'tonal' : (isScaleCandidate(item).eligible ? 'flat' : 'outlined')"
                :color="getCooldownInfo(item).inCooldown ? 'warning' : (isScaleCandidate(item).eligible ? 'primary' : 'default')"
                :prepend-icon="getCooldownInfo(item).inCooldown ? 'mdi-clock-outline' : (isScaleCandidate(item).eligible ? 'mdi-rocket-launch' : 'mdi-tune-vertical')"
                class="font-weight-bold"
                @click="openBudgetModal(item)"
              >
                {{
                  getCooldownInfo(item).inCooldown
                    ? l({ lo: `⏳ ສະເກວແລ້ວ (Cooldown ອີກ ${getCooldownInfo(item).hoursLeft}h ${getCooldownInfo(item).minutesLeft}m)`, th: `⏳ Scale แล้ว (Cooldown อีก ${getCooldownInfo(item).hoursLeft}h ${getCooldownInfo(item).minutesLeft}m)`, en: `⏳ Scaled (Cooldown ${getCooldownInfo(item).hoursLeft}h ${getCooldownInfo(item).minutesLeft}m left)` })
                    : (isScaleCandidate(item).eligible
                        ? l({ lo: '🚀 Scale ງົບ (+10% / +20%)', th: '🚀 Scale งบ (+10% / +20%)', en: '🚀 Scale Budget (+10% / +20%)' })
                        : l({ lo: '💰 ປັບງົບປະມານ (Budget)', th: '💰 ปรับงบประมาณ (Budget)', en: '💰 Adjust Budget' })
                      )
                }}
              </v-btn>
            </div>
          </button>
          <div v-if="selectedId === item.entityId" class="row-detail">
            <!-- Angel Ad Spotlight Banner -->
            <div v-if="isAngelAd(item).isAngel" class="angel-callout">
              <div class="d-flex align-center gap-3">
                <div class="angel-callout-icon">
                  <v-icon icon="mdi-star-face" size="26" color="amber-darken-3" />
                </div>
                <div>
                  <h4 class="text-subtitle-2 font-weight-bold text-amber-darken-4 d-flex align-center gap-1">
                    🧚‍♀️ ແອດນີ້ແມ່ນ "Ads ນາງຟ້າ" (Winning / Top Performer)
                  </h4>
                  <p class="text-caption mb-0 text-amber-darken-3">
                    {{ isAngelAd(item).reason }}
                  </p>
                </div>
              </div>
            </div>
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
              <div class="border-primary" style="background: rgba(2, 132, 199, 0.04);">
                <span class="text-primary font-weight-bold"><v-icon icon="mdi-message-text-outline" size="13" class="mr-1" />{{ l({lo:'ຄົນສົ່ງຂໍ້ຄວາມ (Conversations)',th:'คนทักข้อความ (Conversations)',en:'Messages (Conversations)'}) }}</span>
                <strong class="text-primary">{{ getMessagesCount(item) != null ? fmt(getMessagesCount(item), 0) + ' ຄົນ' : '0' }}</strong>
                <small>{{ getCostPerMessage(item) != null ? money(getCostPerMessage(item)) + '/ຂໍ້ຄວາມ' : l({lo:'ຕົ້ນທຶນ/ຂໍ້ຄວາມ',th:'ต้นทุน/ข้อความ',en:'Cost/msg'}) }}</small>
              </div>
              <div class="border-success" style="background: rgba(22, 163, 74, 0.04);">
                <span class="text-success font-weight-bold"><v-icon icon="mdi-cart-check" size="13" class="mr-1" />{{ l({lo:'ການຊື້ (Purchases / Orders)',th:'การซื้อ (Purchases / Orders)',en:'Purchases / Orders'}) }}</span>
                <strong class="text-success">{{ getPurchasesCount(item) != null ? fmt(getPurchasesCount(item), 0) + ' ອໍເດີ' : '0' }}</strong>
                <small>{{ getCostPerPurchase(item) != null ? 'CPA: ' + money(getCostPerPurchase(item)) : l({lo:'ຕົ້ນທຶນ/ອໍເດີ',th:'ต้นทุน/ออเดอร์',en:'CPA'}) }}</small>
              </div>
              <div v-if="getMessagesCount(item) && getPurchasesCount(item)" class="border-info" style="background: rgba(6, 182, 212, 0.04);">
                <span class="text-info font-weight-bold"><v-icon icon="mdi-percent" size="13" class="mr-1" />{{ l({lo:'ອັດຕາປິດການຂາຍ (Close Rate)',th:'อัตราปิดการขาย (Close Rate)',en:'Close Rate (Msg to Order)'}) }}</span>
                <strong class="text-info">{{ percent((getPurchasesCount(item)! / getMessagesCount(item)!) * 100) }}</strong>
                <small>{{ fmt(getPurchasesCount(item), 0) }} ຊື້ ÷ {{ fmt(getMessagesCount(item), 0) }} ຂໍ້ຄວາມ</small>
              </div>
              <div><span>{{ l({lo:'ການສະແດງ',th:'การแสดงผล',en:'Impressions'}) }}</span><strong>{{ fmt(item.impressions, 0) }}</strong><small>{{ l({lo:'ຈຳນວນຄັ້ງທີ່ໂຄສະນາສະແດງ',th:'จำนวนครั้งที่โฆษณาแสดง',en:'Total ad displays'}) }}</small></div>
              <div><span>{{ l({lo:'ການເຂົ້າເຖິງ',th:'การเข้าถึง',en:'Reach'}) }}</span><strong>{{ fmt(item.reach, 0) }}</strong><small>{{ l({lo:'ຈຳນວນຄົນທີ່ເຫັນ',th:'จำนวนคนที่เห็น',en:'People who saw it'}) }}</small></div>
              <div><span>Link clicks</span><strong>{{ fmt(item.linkClicks, 0) }}</strong><small>{{ l({lo:'ຄລິກໄປປາຍທາງ',th:'คลิกไปยังปลายทาง',en:'Clicks to destination'}) }}</small></div>
              <div><span>CPC</span><strong>{{ money(item.linkCpc) }}</strong><small>{{ l({lo:'ຕົ້ນທຶນຕໍ່ຄລິກ',th:'ต้นทุนต่อคลิก',en:'Cost per link click'}) }}</small></div>
              <div><span>CPM</span><strong>{{ money(item.cpm) }}</strong><small>{{ l({lo:'ຕົ້ນທຶນຕໍ່ 1,000 ການສະແດງ',th:'ต้นทุนต่อ 1,000 การแสดงผล',en:'Cost per 1,000 impressions'}) }}</small></div>
              <div><span>{{ l({lo:'ໂອກາດຊື້ (CR)',th:'โอกาสซื้อ (CR)',en:'Conversion rate'}) }}</span><strong :style="{ color: buyingOpportunity(item).color }">{{ buyingOpportunity(item).conversionRateText }}</strong><small>{{ l({lo:'ການຊື້ ÷ ຄລິກ',th:'การซื้อ ÷ คลิก',en:'Purchases ÷ clicks'}) }}</small></div>
              <div v-if="buyingOpportunity(item).costRatio != null"><span>{{ l({lo:'ສັດສ່ວນຄ່າແອດ/ສິນຄ້າ',th:'สัดส่วนค่าแอด/สินค้า',en:'Ad cost/product'}) }}</span><strong :style="{ color: buyingOpportunity(item).costRatioIsGood ? '#167a4b' : buyingOpportunity(item).costRatioIsHigh ? '#dc2626' : 'inherit' }">{{ buyingOpportunity(item).costRatioText }}</strong><small>{{ buyingOpportunity(item).costRatioIsGood ? l({lo:'ກຳໄລສູງຫຼາຍ',th:'กำไรสูงมาก',en:'High margin'}) : l({lo:'ທຽບລາຄາສິນຄ້າ',th:'เทียบราคาสินค้า',en:'vs product price'}) }}</small></div>
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

  <v-dialog v-model="settingsOpen" max-width="820" scrollable>
    <v-card rounded="xl" class="strategy-dialog-card">
      <v-card-title class="dialog-title d-flex align-center justify-space-between pb-2">
        <div class="d-flex align-center gap-2">
          <v-avatar color="primary" variant="tonal" size="36">
            <v-icon icon="mdi-target-variant" size="22" />
          </v-avatar>
          <div>
            <small class="text-primary font-weight-bold letter-spacing-1">MARKETING STRATEGY & THRESHOLDS</small>
            <h2 class="text-h6 font-weight-bold mb-0">
              {{ l({ lo: 'ຕັ້ງຄ່າຍຸດທະສາດໂຄສະນາ (Setup Strategy)', th: 'ตั้งค่ายุทธศาสตร์โฆษณา (Setup Strategy)', en: 'Ad Strategy Setup' }) }}
            </h2>
          </div>
        </div>
        <v-btn icon="mdi-close" variant="text" @click="settingsOpen = false" />
      </v-card-title>

      <div class="px-4 pt-1 pb-1">
        <v-alert type="info" variant="tonal" density="compact" class="mb-2 rounded-lg">
          <div class="d-flex align-center gap-1 font-weight-bold">
            <v-icon icon="mdi-information-outline" size="18" />
            <span>{{ l({ lo: 'ຍຸດທະສາດແຍກຕາມສິນຄ້າ (Product Strategy)', th: 'ยุทธศาสตร์แยกตามสินค้า (Product Strategy)', en: 'Product-Level Strategy' }) }}</span>
          </div>
          <div class="text-caption mt-1">
            {{ l({
              lo: 'ແຕ່ລະສິນຄ້າ (ທີ່ຜູກກັບ Auto-Reply) ມີລາຄາ ແລະ ກຳໄລຕ່າງກັນ. ທ່ານສາມາດກຳນົດເປົ້າໝາຍ CPA, ເພດານງົບ ແລະ ເກນ Stop-Loss ແຍກສະເພາະແຕ່ລະສິນຄ້າໄດ້ເລີຍ.',
              th: 'แต่ละสินค้า (ที่ผูกกับ Auto-Reply) มีราคาและกำไรต่างกัน คุณสามารถกำหนดเป้าหมาย CPA, เพดานงบ และเกณฑ์ Stop-Loss แยกเฉพาะแต่ละสินค้าได้',
              en: 'Each product (linked via Auto-Reply) has different prices and margins. Configure custom target CPA, budget caps, and stop-loss per product.'
            }) }}
          </div>
        </v-alert>
      </div>

      <v-tabs v-model="strategyTab" color="primary" class="px-4 mb-2 border-b">
        <v-tab value="products" class="font-weight-bold">
          <v-icon start icon="mdi-package-variant-closed" />
          {{ l({ lo: '📦 ຍຸດທະສາດແຍກຕາມສິນຄ້າ', th: '📦 ยุทธศาสตร์แยกตามสินค้า', en: '📦 Per-Product Strategy' }) }}
          <v-chip size="x-small" color="primary" variant="flat" class="ml-2 font-weight-bold">
            {{ products.length }}
          </v-chip>
        </v-tab>
        <v-tab value="default" class="font-weight-bold">
          <v-icon start icon="mdi-earth" />
          {{ l({ lo: '🌐 ຄ່າເລີ່ມຕົ້ນຂອງບັນຊີ (Account Default)', th: '🌐 ค่าเริ่มต้นของบัญชี (Account Default)', en: '🌐 Account Default' }) }}
        </v-tab>
      </v-tabs>

      <v-card-text class="pa-4 strategy-body">
        <v-window v-model="strategyTab">
          <!-- Tab 1: Per-Product Strategy -->
          <v-window-item value="products">
            <div class="product-strategy-layout">
              <div class="mb-3">
                <v-select
                  v-model="selectedStrategyProductId"
                  :items="strategyProductSelectItems"
                  item-title="title"
                  item-value="value"
                  :label="l({ lo: 'ເລືອກສິນຄ້າທີ່ຕ້ອງການຕັ້ງຄ່າຍຸດທະສາດ', th: 'เลือกสินค้าที่ต้องการตั้งค่ายุทธศาสตร์', en: 'Select Product to Configure' })"
                  variant="outlined"
                  density="comfortable"
                  prepend-inner-icon="mdi-package-variant"
                  hide-details
                />
              </div>

              <div v-if="currentStrategyProduct" class="product-strategy-card">
                <div class="prod-header-box mb-3 pa-3 rounded-lg border d-flex align-center justify-space-between flex-wrap gap-2 bg-surface-soft">
                  <div class="d-flex align-center gap-3">
                    <v-avatar size="44" rounded="lg" color="grey-lighten-4">
                      <v-img v-if="currentStrategyProduct.imageUrl" :src="currentStrategyProduct.imageUrl" cover />
                      <v-icon v-else icon="mdi-package-variant-closed" size="24" color="grey" />
                    </v-avatar>
                    <div>
                      <h4 class="font-weight-bold text-subtitle-1 mb-0">{{ currentStrategyProduct.name }}</h4>
                      <div class="d-flex align-center gap-2 text-caption">
                        <span class="text-primary font-weight-bold">
                          {{ l({ lo: 'ລາຄາ:', th: 'ราคา:', en: 'Price:' }) }} {{ money(currentStrategyProductPriceInAdCurrency) }}
                        </span>
                        <span v-if="currentProductLinkedFlowName" class="text-medium-emphasis">
                          · Auto-Reply: {{ currentProductLinkedFlowName }}
                        </span>
                        <v-chip v-if="currentProductAdCount > 0" size="x-small" color="info" variant="tonal" class="font-weight-bold">
                          {{ currentProductAdCount }} ແອດຜູກຢູ່
                        </v-chip>
                      </div>
                    </div>
                  </div>

                  <div class="d-flex align-center gap-2">
                    <v-switch
                      v-model="currentProductStrategy.enabled"
                      color="success"
                      hide-details
                      density="compact"
                      :label="currentProductStrategy.enabled ? l({ lo: 'ເປີດໃຊ້ເກນສະເພາະ', th: 'เปิดใช้เกณฑ์เฉพาะ', en: 'Custom Strategy ON' }) : l({ lo: 'ໃຊ້ຄ່າເລີ່ມຕົ້ນ', th: 'ใช้ค่าเริ่มต้น', en: 'Use Default' })"
                    />
                  </div>
                </div>

                <!-- If Custom Strategy is Enabled -->
                <div v-if="currentProductStrategy.enabled" class="strategy-inputs-wrap">
                  <div class="auto-calc-bar mb-3 pa-3 rounded-lg d-flex align-center justify-space-between flex-wrap gap-2 bg-primary-soft">
                    <div class="d-flex align-center gap-2">
                      <v-icon icon="mdi-calculator-variant-outline" color="primary" />
                      <div>
                        <strong class="text-caption font-weight-bold">{{ l({ lo: 'ຄິດໄລ່ເກນອັດຕະໂນມັດຈາກລາຄາສິນຄ້າ', th: 'คำนวณเกณฑ์อัตโนมัติจากราคาสินค้า', en: 'Auto-Calculate from Product Price' }) }}</strong>
                        <div class="text-caption text-medium-emphasis">
                          {{ l({
                            lo: `ລາຄາ ${money(currentStrategyProductPriceInAdCurrency)} ➔ ແນະນຳ Target CPA 20% (${money(currentStrategyProductPriceInAdCurrency * 0.2)}), Stop-Loss 80% (${money(currentStrategyProductPriceInAdCurrency * 0.8)})`,
                            th: `ราคา ${money(currentStrategyProductPriceInAdCurrency)} ➔ แนะนำ Target CPA 20% (${money(currentStrategyProductPriceInAdCurrency * 0.2)}), Stop-Loss 80% (${money(currentStrategyProductPriceInAdCurrency * 0.8)})`,
                            en: `Price ${money(currentStrategyProductPriceInAdCurrency)} ➔ Suggested Target CPA 20% (${money(currentStrategyProductPriceInAdCurrency * 0.2)}), Stop-Loss 80% (${money(currentStrategyProductPriceInAdCurrency * 0.8)})`
                          }) }}
                        </div>
                      </div>
                    </div>
                    <v-btn
                      size="small"
                      color="primary"
                      variant="flat"
                      prepend-icon="mdi-auto-fix"
                      class="font-weight-bold"
                      @click="autoCalculateProductStrategy(currentStrategyProduct.id)"
                    >
                      {{ l({ lo: 'ຄິດໄລ່ໃຫ້ທັນທີ', th: 'คำนวณให้ทันที', en: 'Auto-Calculate' }) }}
                    </v-btn>
                  </div>

                  <div class="settings-grid primary-settings mb-3">
                    <v-text-field
                      v-model.number="currentProductStrategy.targetCost"
                      type="number"
                      min="0"
                      step="0.01"
                      :label="l({ lo: 'CPA ສູງສຸດ / ຈຸດຄຸ້ມທຶນ (Target CPA)', th: 'CPA สูงสุด / จุดคุ้มทุน (Target CPA)', en: 'Max Target CPA' })"
                      :suffix="currency"
                      variant="outlined"
                      density="comfortable"
                      :hint="l({ lo: 'ຕົ້ນທຶນ/ຜົນລັບ ບໍ່ຄວນເກີນຄ່ານີ້ສຳລັບສິນຄ້ານີ້', th: 'ต้นทุน/ผลลัพธ์ ไม่ควรเกินค่านี้สำหรับสินค้านี้', en: 'Max allowed cost/result for this product' })"
                      persistent-hint
                    />
                    <v-text-field
                      v-model.number="currentProductStrategy.grossMargin"
                      type="number"
                      min="0"
                      max="100"
                      step="1"
                      :label="l({ lo: 'Gross Margin ຂອງສິນຄ້ານີ້ (%)', th: 'Gross Margin ของสินค้านี้ (%)', en: 'Product Gross Margin (%)' })"
                      suffix="%"
                      variant="outlined"
                      density="comfortable"
                      :hint="currentProductBreakEvenROAS ? `Break-even ROAS: ${fmt(currentProductBreakEvenROAS, 2)}×` : ''"
                      persistent-hint
                    />
                  </div>

                  <div class="settings-grid mb-3">
                    <v-text-field
                      v-model.number="currentProductStrategy.scaleMaxCPA"
                      type="number"
                      min="0"
                      step="0.01"
                      :label="l({ lo: 'CPA ສູງສຸດທີ່ອະນຸຍາດໃຫ້ Scale', th: 'CPA สูงสุดที่อนุญาตให้ Scale', en: 'Scale Max CPA' })"
                      :suffix="currency"
                      variant="outlined"
                      density="comfortable"
                    />
                    <v-text-field
                      v-model.number="currentProductStrategy.scaleMinPurchases"
                      type="number"
                      min="1"
                      step="1"
                      :label="l({ lo: 'ອໍເດີຂັ້ນຕ່ຳເພື່ອ Scale', th: 'ออเดอร์ขั้นต่ำเพื่อ Scale', en: 'Scale Min Purchases' })"
                      variant="outlined"
                      density="comfortable"
                    />
                    <v-text-field
                      v-model.number="currentProductStrategy.scaleMaxDailyBudgetCap"
                      type="number"
                      min="0"
                      step="1"
                      :label="l({ lo: 'ເພດານງົບສູງສຸດຕໍ່ມື້ (Max Cap)', th: 'เพดานงบสูงสุดต่อวัน (Max Cap)', en: 'Max Daily Budget Cap' })"
                      :suffix="currency"
                      variant="outlined"
                      density="comfortable"
                      :hint="l({ lo: 'ຫາກຮອດເພດານນີ້ ບັອດຈະບໍ່ເພີ່ມງົບອີກ', th: 'หากถึงเพดานนี้ บอทจะไม่เพิ่มงบอีก', en: 'Stop scaling if budget reaches cap' })"
                      persistent-hint
                    />
                    <v-text-field
                      v-model.number="currentProductStrategy.stopMinSpend"
                      type="number"
                      min="0"
                      step="0.01"
                      :label="l({ lo: 'ເກນ Stop-Loss (Spend >=)', th: 'เกณฑ์ Stop-Loss (Spend >=)', en: 'Stop-Loss Spend (>=' })"
                      :suffix="currency"
                      variant="outlined"
                      density="comfortable"
                      :hint="l({ lo: 'ຖ້າໃຊ້ງົບຮອດນີ້ແລ້ວ 0 ອໍເດີ ຈະຖືກແນະນຳໃຫ້ປິດ', th: 'ถ้าใช้งบถึงนี้แล้ว 0 ออเดอร์ จะแนะนำให้หยุด', en: 'Pause ad if spent this much with 0 purchases' })"
                      persistent-hint
                    />
                  </div>
                </div>

                <!-- If Custom Strategy is OFF -->
                <div v-else class="pa-6 text-center text-medium-emphasis border rounded-lg">
                  <v-icon icon="mdi-shield-outline" size="36" class="mb-2" />
                  <p class="mb-2">
                    {{ l({
                      lo: `ສິນຄ້າ “${currentStrategyProduct.name}” ກຳລັງໃຊ້ຄ່າເລີ່ມຕົ້ນຂອງບັນຊີໂຄສະນາ (Account Default).`,
                      th: `สินค้า “${currentStrategyProduct.name}” กำลังใช้ค่าเริ่มต้นของบัญชีโฆษณา (Account Default)`,
                      en: `“${currentStrategyProduct.name}” is using Account Default strategy.`
                    }) }}
                  </p>
                  <v-btn
                    size="small"
                    variant="tonal"
                    color="primary"
                    prepend-icon="mdi-plus"
                    @click="currentProductStrategy.enabled = true; autoCalculateProductStrategy(currentStrategyProduct.id)"
                  >
                    {{ l({ lo: 'ກຳນົດຍຸດທະສາດສະເພາະໃຫ້ສິນຄ້ານີ້', th: 'กำหนดยุทธศาสตร์เฉพาะให้สินค้านี้', en: 'Configure custom strategy for this product' }) }}
                  </v-btn>
                </div>
              </div>
            </div>
          </v-window-item>

          <!-- Tab 2: Account Default Strategy -->
          <v-window-item value="default">
            <div>
              <div class="mb-3">
                <p class="text-caption text-medium-emphasis mb-2">
                  {{ l({
                    lo: 'ຄ່າເລີ່ມຕົ້ນນີ້ຈະຖືກນຳໃຊ້ກັບແອດທັງໝົດທີ່ຍັງບໍ່ໄດ້ກຳນົດຍຸດທະສາດສະເພາະສິນຄ້າ, ຫຼື ແອດທີ່ຍັງບໍ່ທັນຜູກກັບສິນຄ້າໃດໜຶ່ງ.',
                    th: 'ค่าเริ่มต้นนี้จะใช้กับแอดทั้งหมดที่ยังไม่ได้กำหนดยุทธศาสตร์เฉพาะ หรือแอดที่ยังไม่ได้ผูกกับสินค้า',
                    en: 'These defaults apply to all ads without a custom product strategy or unlinked ads.'
                  }) }}
                </p>
              </div>
              <div class="settings-grid primary-settings mb-3">
                <v-text-field v-model.number="targetCost" type="number" min="0" step="0.01" :label="l({lo:'ຕົ້ນທຶນ/ຜົນລັບສູງສຸດ (Default CPA)',th:'ต้นทุน/ผลลัพธ์สูงสุด (Default CPA)',en:'Default maximum cost per result'})" :suffix="currency" variant="outlined" density="comfortable" />
                <v-text-field v-model.number="grossMargin" type="number" min="0" max="100" step="1" :label="l({lo:'Gross margin ຂອງຮ້ານ (Default %)',th:'Gross margin ของร้าน (Default %)',en:'Default product gross margin'})" suffix="%" variant="outlined" density="comfortable" :hint="breakEvenROAS?`Break-even ROAS ${fmt(breakEvenROAS,2)}×`:''" persistent-hint />
              </div>
              <div class="settings-grid">
                <v-text-field v-model.number="weakCTR" type="number" min="0" step="0.1" label="Weak CTR below" suffix="%" variant="outlined" density="comfortable" />
                <v-text-field v-model.number="goodCTR" type="number" min="0" step="0.1" label="Good CTR from" suffix="%" variant="outlined" density="comfortable" />
                <v-text-field v-model.number="maxFrequency" type="number" min="1" step="0.1" label="Frequency warning from" variant="outlined" density="comfortable" />
                <v-text-field v-model.number="minimumPlanImpressions" type="number" min="0" step="100" :label="l({lo:'Impressions ຕ່ຳສຸດສຳລັບແຜນ',th:'Impressions ขั้นต่ำสำหรับแผน',en:'Minimum plan impressions'})" variant="outlined" density="comfortable" />
                <v-text-field v-model.number="minimumPlanSpend" type="number" min="0" step="0.01" :label="l({lo:'Spend ຕ່ຳສຸດສຳລັບແຜນ',th:'Spend ขั้นต่ำสำหรับแผน',en:'Minimum plan spend'})" :suffix="currency" variant="outlined" density="comfortable" />
              </div>
            </div>
          </v-window-item>
        </v-window>
      </v-card-text>

      <v-card-actions class="dialog-actions border-t">
        <v-btn variant="text" @click="targetCost=null;grossMargin=null;goodCTR=3;weakCTR=1;maxFrequency=3;minimumPlanImpressions=1000;minimumPlanSpend=null">
          {{ l({lo:'ຄ່າເລີ່ມຕົ້ນ',th:'ค่าเริ่มต้น',en:'Reset Defaults'}) }}
        </v-btn>
        <v-spacer />
        <v-btn variant="text" @click="settingsOpen=false">
          {{ l({lo:'ປິດ',th:'ปิด',en:'Close'}) }}
        </v-btn>
        <v-btn color="primary" variant="flat" prepend-icon="mdi-content-save" class="font-weight-bold px-4" @click="saveThresholds">
          {{ l({lo:'ບັນທຶກການຕັ້ງຄ່າ',th:'บันทึก',en:'Save Strategy'}) }}
        </v-btn>
      </v-card-actions>
    </v-card>
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

          <!-- Cooldown Warning Alert -->
          <v-alert
            v-if="targetBudgetItem && getCooldownInfo(targetBudgetItem).inCooldown"
            type="warning"
            variant="tonal"
            density="compact"
            class="mt-3 mb-1 rounded-lg"
          >
            <div class="d-flex align-center gap-1 font-weight-bold">
              <v-icon icon="mdi-clock-alert-outline" size="18" />
              <span>
                {{ l({
                  lo: `ແອດນີ້ຫາກໍ່ຖືກສະເກວໄປ (ຍັງເຫຼືອ Cooldown ${getCooldownInfo(targetBudgetItem).hoursLeft}h ${getCooldownInfo(targetBudgetItem).minutesLeft}m)`,
                  th: `แอดนี้เพิ่งถูก Scale ไป (เหลือ Cooldown ${getCooldownInfo(targetBudgetItem).hoursLeft}h ${getCooldownInfo(targetBudgetItem).minutesLeft}m)`,
                  en: `This ad was scaled recently (${getCooldownInfo(targetBudgetItem).hoursLeft}h ${getCooldownInfo(targetBudgetItem).minutesLeft}m cooldown remaining)`
                }) }}
              </span>
            </div>
            <div class="text-caption mt-1">
              {{ l({
                lo: 'Meta ຕ້ອງການເວລາ 24 ຊົ່ວໂມງ ເພື່ອປັບຕົວໃນ Learning Phase ໃໝ່. ຖ້າບໍ່ຈຳເປັນ ຄວນລໍຖ້າໃຫ້ຄົບ Cooldown ກ່ອນສະເກວອີກຄັ້ງ.',
                th: 'Meta ต้องการเวลา 24 ชั่วโมง เพื่อปรับตัวใน Learning Phase ใหม่ หากไม่จำเป็นควรรอให้ครบ Cooldown ก่อน Scale อีกครั้ง',
                en: 'Meta requires 24 hours to adapt to budget changes. It is advised to wait until cooldown completes.'
              }) }}
            </div>
          </v-alert>
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

  <!-- Auto-Pilot Modal Dialog -->
  <v-dialog v-model="autoPilotModalOpen" max-width="840" scrollable>
    <v-card rounded="xl" class="autopilot-dialog-card">
      <v-card-title class="dialog-title d-flex align-center justify-space-between">
        <div class="d-flex align-center gap-2">
          <v-avatar color="primary" variant="tonal" size="36">
            <v-icon icon="mdi-robot" size="22" />
          </v-avatar>
          <div>
            <small class="text-primary font-weight-bold letter-spacing-1">DUAL-SYSTEM AI ADS CONTROLLER</small>
            <h2 class="text-h6 font-weight-bold mb-0">
              {{ l({ lo: 'ຈັດການລະບົບ Auto-Pilot & Manual', th: 'จัดการระบบ Auto-Pilot & Manual', en: 'Auto-Pilot & Manual Controller' }) }}
            </h2>
          </div>
        </div>
        <div class="d-flex align-center gap-1">
          <v-btn
            size="small"
            variant="tonal"
            color="info"
            prepend-icon="mdi-help-circle-outline"
            class="font-weight-bold"
            @click="guideModalOpen = true; guideStep = 4"
          >
            {{ l({ lo: '📖 ຄູ່ມືຕັ້ງຄ່າ', th: '📖 คู่มือตั้งค่า', en: '📖 Setup Guide' }) }}
          </v-btn>
          <v-btn icon="mdi-close" variant="text" @click="autoPilotModalOpen = false" />
        </div>
      </v-card-title>

      <v-card-text class="pa-4 autopilot-body">
        <!-- Master Switch Card -->
        <v-card
          variant="tonal"
          :color="autoPilotConfig.enabled ? 'success' : undefined"
          class="pa-4 mb-4 rounded-xl master-switch-card"
          :class="{ 'is-active': autoPilotConfig.enabled }"
        >
          <div class="d-flex align-center justify-space-between flex-wrap gap-3">
            <div class="d-flex align-center gap-3">
              <v-icon
                :icon="autoPilotConfig.enabled ? 'mdi-robot-excited-outline' : 'mdi-robot-off-outline'"
                size="36"
                :color="autoPilotConfig.enabled ? 'success' : 'grey'"
              />
              <div>
                <h3 class="text-subtitle-1 font-weight-bold mb-0">
                  {{ autoPilotConfig.enabled
                    ? l({ lo: 'ສະຖານະ Auto-Pilot: ກຳລັງເປີດໃຊ້ງານ (ON)', th: 'สถานะ Auto-Pilot: กำลังเปิดใช้งาน (ON)', en: 'Auto-Pilot Status: ACTIVE (ON)' })
                    : l({ lo: 'ສະຖານະ Auto-Pilot: ປິດຢູ່ (OFF - ໂໝດ Manual)', th: 'สถานะ Auto-Pilot: ปิดอยู่ (OFF - โหมด Manual)', en: 'Auto-Pilot Status: DISABLED (OFF - Manual Mode)' })
                  }}
                </h3>
                <p class="text-caption mb-0 text-medium-emphasis">
                  {{ autoPilotConfig.enabled
                    ? l({ lo: 'ລະບົບຈະກວດສອບ ແລະ ປະມວນຜົນອັດຕະໂນມັດ: ສະເກວແອດຂາຍດີ +20% (ພ້ອມ Cooldown 24h) ແລະ ຢຸດແອດຂາດທຶນ.', th: 'ระบบจะตรวจสอบและประมวลผลอัตโนมัติ: Scale แอดขายดี +20% (พร้อม Cooldown 24h) และหยุดแอดขาดทุน', en: 'System will auto-scale winning ads (+20% with 24h cooldown) and auto-pause losing ads.' })
                    : l({ lo: 'ປິດການເຮັດວຽກອັດຕະໂນມັດທັງໝົດ. ທ່ານຈະເປັນຜູ້ກວດ ແລະ ກົດປັບງົບ/ປິດແອດເອງ 100% ໂດຍບໍ່ມີການປ່ຽນແປງໃດໆ.', th: 'ปิดการทำงานอัตโนมัติทั้งหมด คุณเป็นผู้ตรวจสอบและกดปรับงบ/หยุดแอดเอง 100% โดยไม่มีการเปลี่ยนแปลงใดๆ', en: 'All automatic actions disabled. You manually review and trigger actions.' })
                  }}
                </p>
              </div>
            </div>
            <v-switch
              v-model="autoPilotConfig.enabled"
              color="success"
              hide-details
              inset
              @update:model-value="toggleAutoPilotMaster"
            />
          </div>
        </v-card>

        <!-- Tabs: Rules vs Activity Logs -->
        <v-tabs v-model="autoPilotTab" color="primary" class="mb-4 autopilot-tabs">
          <v-tab value="rules">
            <v-icon start icon="mdi-tune" />
            {{ l({ lo: 'ກົດລະບຽບ & ຄວາມປອດໄພ', th: 'กฎระเบียบ & ความปลอดภัย', en: 'Rules & Safety' }) }}
          </v-tab>
          <v-tab value="logs">
            <v-icon start icon="mdi-history" />
            {{ l({ lo: 'ປະຫວັດການເຮັດວຽກ (Audit Log)', th: 'ประวัติการทำงาน (Audit Log)', en: 'Activity Logs' }) }}
            <v-chip size="x-small" class="ml-2 font-weight-bold" color="primary" variant="flat">{{ autoPilotLogs.length }}</v-chip>
          </v-tab>
        </v-tabs>

        <v-window v-model="autoPilotTab">
          <!-- Tab 1: Rules & Safety Settings -->
          <v-window-item value="rules">
            <div class="autopilot-rules-container">
              <!-- Scope & Target Selection Card -->
              <v-card variant="outlined" class="rule-card pa-4 mb-4 rounded-lg">
                <div class="d-flex align-center gap-2 mb-3">
                  <v-avatar color="primary" size="28" variant="tonal">
                    <v-icon icon="mdi-target" size="16" />
                  </v-avatar>
                  <div>
                    <h4 class="font-weight-bold mb-0">{{ l({ lo: 'ຂອບເຂດ ແລະ ປະເພດໂຄສະນາທີ່ໃຫ້ Auto-Pilot ຄຸມ', th: 'ขอบเขตและประเภทโฆษณาที่ให้ Auto-Pilot คุม', en: 'Scope & Target Structure' }) }}</h4>
                    <small class="text-medium-emphasis">{{ l({ lo: 'ເລືອກວ່າຈະໃຫ້ບັອດເຮັດວຽກກັບທຸກແອດ ຫຼື ເລືອກສະເພາະ Campaign / Ad Set / ສິນຄ້າ', th: 'เลือกว่าจะให้บอททำงานกับทุกแอด หรือเลือกเฉพาะ Campaign / Ad Set / สินค้า', en: 'Choose whether Auto-Pilot evaluates all ads or targets specific campaigns, ad sets, or products' }) }}</small>
                  </div>
                </div>

                <div class="rule-inputs-grid mb-1">
                  <v-select
                    v-model="autoPilotConfig.scope"
                    :items="[
                      { title: l({ lo: '🌐 ທຸກແອດທີ່ເປີດຢູ່ (All Active Ads)', th: '🌐 ทุกแอดที่เปิดอยู่ (All Active Ads)', en: '🌐 All Active Ads' }), value: 'all' },
                      { title: l({ lo: '📦 ສະເພາະສິນຄ້າທີ່ເລືອກໃນ Filter (e.g. kito AG29)', th: '📦 เฉพาะสินค้าที่เลือกใน Filter (e.g. kito AG29)', en: '📦 Current Filter Product Only' }), value: 'filtered_product' }
                    ]"
                    item-title="title"
                    item-value="value"
                    :label="l({ lo: 'ຂອບເຂດສິນຄ້າ', th: 'ขอบเขตสินค้า', en: 'Product Scope' })"
                    variant="outlined"
                    density="compact"
                    hide-details
                  />
                  <v-select
                    v-model="autoPilotConfig.targetType"
                    :items="[
                      { title: l({ lo: '⚡ ທຸກລະດັບ (CBO Campaign & ABO Ad Set)', th: '⚡ ทุกระดับ (CBO Campaign & ABO Ad Set)', en: '⚡ All (CBO & ABO)' }), value: 'all' },
                      { title: l({ lo: '📢 ສະເພາະ Campaign (CBO - Advantage Budget)', th: '📢 เฉพาะ Campaign (CBO - Advantage Budget)', en: '📢 Campaigns Only (CBO)' }), value: 'campaign' },
                      { title: l({ lo: '🎯 ສະເພາະ Ad Set (ABO - Ad Set Budget)', th: '🎯 เฉพาะ Ad Set (ABO - Ad Set Budget)', en: '🎯 Ad Sets Only (ABO)' }), value: 'adset' }
                    ]"
                    item-title="title"
                    item-value="value"
                    :label="l({ lo: 'ລະດັບໂຄງສ້າງງົບປະມານ', th: 'ระดับโครงสร้างงบประมาณ', en: 'Budget Structure Level' })"
                    variant="outlined"
                    density="compact"
                    hide-details
                  />
                </div>
              </v-card>

              <!-- Rule 1: Auto-Scale Winners -->
              <v-card variant="outlined" class="rule-card pa-4 mb-4 rounded-lg">
                <div class="d-flex align-center justify-space-between mb-3">
                  <div class="d-flex align-center gap-2">
                    <span class="rule-badge scale">🚀 ກົດທີ 1</span>
                    <div>
                      <h4 class="font-weight-bold mb-0">{{ l({ lo: 'ສະເກວງົບແອດຂາຍດີ (Auto-Scale Winners)', th: 'Scale งบแอดขายดี (Auto-Scale Winners)', en: 'Auto-Scale Winning Ads' }) }}</h4>
                      <small class="text-medium-emphasis">{{ l({ lo: 'ເພີ່ມງົບອັດຕະໂນມັດໃຫ້ແອດທີ່ມີການຊື້ດີ ແລະ ຕົ້ນທຶນຢູ່ໃນເກນ', th: 'เพิ่มงบอัตโนมัติให้แอดที่มีการซื้อดีและต้นทุนอยู่ในเกณฑ์', en: 'Automatically increase budget for ads with strong purchases and low CPA' }) }}</small>
                    </div>
                  </div>
                  <v-switch v-model="autoPilotConfig.autoScaleEnabled" color="success" hide-details density="compact" />
                </div>

                <div class="rule-inputs-grid" :class="{ 'opacity-50 pointer-events-none': !autoPilotConfig.autoScaleEnabled }">
                  <v-text-field
                    v-model.number="autoPilotConfig.scaleMinPurchases"
                    type="number"
                    min="1"
                    step="1"
                    :label="l({ lo: 'ຈຳນວນການຊື້ຢ່າງໜ້ອຍ (Purchases >=)', th: 'จำนวนการซื้อขั้นต่ำ (Purchases >=)', en: 'Min Purchases (>=' })"
                    variant="outlined"
                    density="compact"
                    suffix="ອໍເດີ"
                  />
                  <v-text-field
                    v-model.number="autoPilotConfig.scaleMaxCPA"
                    type="number"
                    min="0"
                    step="0.01"
                    :label="l({ lo: 'ຕົ້ນທຶນ/ຜົນລັບສູງສຸດ (CPA <=)', th: 'ต้นทุน/ผลลัพธ์สูงสุด (CPA <=)', en: 'Max CPA (<=' })"
                    :placeholder="targetCost ? String(targetCost) : (currencyOffset === 1 ? '50,000' : '2.5')"
                    :suffix="currency"
                    variant="outlined"
                    density="compact"
                    :hint="l({ lo: 'ຖ້າປະວ່າງ ຈະໃຊ້ຄ່າເປົ້າໝາຍຮ້ານ', th: 'หากเว้นว่าง จะใช้เป้าหมายของร้าน', en: 'Leave empty to use store target CPA' })"
                    persistent-hint
                  />
                  <v-text-field
                    v-model.number="autoPilotConfig.scaleMaxFrequency"
                    type="number"
                    min="1"
                    step="0.1"
                    :label="l({ lo: 'Frequency ສູງສຸດ (<=)', th: 'Frequency สูงสุด (<=)', en: 'Max Frequency (<=' })"
                    variant="outlined"
                    density="compact"
                  />
                  <v-text-field
                    v-model.number="autoPilotConfig.scalePercent"
                    type="number"
                    min="5"
                    max="50"
                    step="5"
                    :label="l({ lo: 'ເປີເຊັນເພີ່ມງົບຕໍ່ຄັ້ງ', th: 'เปอร์เซ็นต์เพิ่มงบต่อครั้ง', en: 'Scale Percentage' })"
                    suffix="%"
                    variant="outlined"
                    density="compact"
                  />
                  <v-text-field
                    v-model.number="autoPilotConfig.scaleCooldownHours"
                    type="number"
                    min="1"
                    max="72"
                    step="1"
                    :label="l({ lo: 'ໄລຍະ Cooldown ຕໍ່ແອດ', th: 'ระยะ Cooldown ต่อแอด', en: 'Cooldown Per Ad' })"
                    suffix="ຊົ່ວໂມງ"
                    variant="outlined"
                    density="compact"
                    :hint="l({ lo: 'ແນະນຳ 24h ເພື່ອບໍ່ໃຫ້ຫຼຸດ Meta Learning Phase', th: 'แนะนำ 24h เพื่อไม่ให้หลุด Learning Phase', en: 'Recommended 24h to protect learning phase' })"
                    persistent-hint
                  />
                  <v-text-field
                    v-model.number="autoPilotConfig.maxDailyBudgetCap"
                    type="number"
                    min="0"
                    step="1"
                    :label="l({ lo: 'ເພດານງົບສູງສຸດຕໍ່ມື້ (Budget Cap)', th: 'เพดานงบสูงสุดต่อวัน (Budget Cap)', en: 'Max Daily Budget Cap' })"
                    :suffix="currency"
                    variant="outlined"
                    density="compact"
                    :placeholder="currencyOffset === 1 ? '1,000,000' : '50'"
                    :hint="l({ lo: 'ປ້ອງກັນງົບເກີນຄວບຄຸມ', th: 'ป้องกันงบเกินควบคุม', en: 'Safety limit against runaway budget' })"
                    persistent-hint
                  />
                </div>
              </v-card>

              <!-- Rule 2: Auto-Stop Losers (Stop-Loss) -->
              <v-card variant="outlined" class="rule-card pa-4 mb-4 rounded-lg">
                <div class="d-flex align-center justify-space-between mb-3">
                  <div class="d-flex align-center gap-2">
                    <span class="rule-badge stop">🛑 ກົດທີ 2</span>
                    <div>
                      <h4 class="font-weight-bold mb-0">{{ l({ lo: 'ຢຸດແອດຂາດທຶນ (Auto-Stop Losers / Stop-Loss)', th: 'หยุดแอดขาดทุน (Auto-Stop Losers / Stop-Loss)', en: 'Auto-Stop Losing Ads (Stop-Loss)' }) }}</h4>
                      <small class="text-medium-emphasis">{{ l({ lo: 'ປິດແອດອັດຕະໂນມັດເມື່ອໃຊ້ງົບເຖິງເກນ ແຕ່ບໍ່ມີການຊື້ ແລະ CTR ຕ່ຳ', th: 'หยุดแอดอัตโนมัติเมื่อใช้งบถึงเกณฑ์แต่ไม่มีการซื้อและ CTR ต่ำ', en: 'Automatically pause ads when spend hits threshold with 0 purchases and low CTR' }) }}</small>
                    </div>
                  </div>
                  <v-switch v-model="autoPilotConfig.autoStopEnabled" color="error" hide-details density="compact" />
                </div>

                <div class="rule-inputs-grid" :class="{ 'opacity-50 pointer-events-none': !autoPilotConfig.autoStopEnabled }">
                  <v-text-field
                    v-model.number="autoPilotConfig.stopMinSpend"
                    type="number"
                    min="0"
                    step="1"
                    :label="l({ lo: 'ໃຊ້ງົບໄປແລ້ວຢ່າງໜ້ອຍ (Spend >=)', th: 'ใช้งบไปแล้วอย่างน้อย (Spend >=)', en: 'Spend Threshold (>=' })"
                    :suffix="currency"
                    :placeholder="currencyOffset === 1 ? '50,000' : '5.00'"
                    variant="outlined"
                    density="compact"
                  />
                  <v-text-field
                    :model-value="0"
                    readonly
                    disabled
                    :label="l({ lo: 'ຈຳນວນການຊື້ (Purchases == 0)', th: 'จำนวนการซื้อ (Purchases == 0)', en: 'Purchases (== 0)' })"
                    suffix="ອໍເດີ"
                    variant="outlined"
                    density="compact"
                    :hint="l({ lo: 'ຕ້ອງບໍ່ມີການຊື້ເລີຍ ຈຶ່ງເຂົ້າເກນຢຸດ', th: 'ต้องไม่มีการซื้อเลย จึงเข้าเกณฑ์หยุด', en: 'Must have 0 purchases to trigger' })"
                    persistent-hint
                  />
                  <v-text-field
                    v-model.number="autoPilotConfig.stopMaxCTR"
                    type="number"
                    min="0.1"
                    max="5"
                    step="0.1"
                    :label="l({ lo: 'CTR ຕ່ຳກວ່າ (<)', th: 'CTR ต่ำกว่า (<)', en: 'CTR Below (<)' })"
                    suffix="%"
                    variant="outlined"
                    density="compact"
                    :hint="l({ lo: 'CTR ຕ່ຳສະແດງວ່າ Creative ບໍ່ດຶງດູດ', th: 'CTR ต่ำแสดงว่าครีเอทีฟไม่ดึงดูด', en: 'Low CTR indicates poor engagement' })"
                    persistent-hint
                  />
                </div>
              </v-card>

              <!-- Safety Guarantee Callout -->
              <v-alert type="info" variant="tonal" density="compact" class="rounded-lg">
                <div class="d-flex align-center gap-2">
                  <v-icon icon="mdi-shield-check" size="20" />
                  <div>
                    <strong>{{ l({ lo: 'ລະບົບຄວາມປອດໄພ 3 ຊັ້ນ (3-Layer Safety):', th: 'ระบบความปลอดภัย 3 ชั้น (3-Layer Safety):', en: '3-Layer Safety Protection:' }) }}</strong>
                    <ul class="mb-0 mt-1 pl-4 text-caption">
                      <li>{{ l({ lo: '1. Cooldown 24h: ປ້ອງກັນບໍ່ໃຫ້ແອດຖືກສະເກວຊ້ຳຊ້ອນໃນມື້ດຽວກັນ ຈົນເສຍ Learning Phase', th: '1. Cooldown 24h: ป้องกันการ Scale ซ้ำซ้อนในวันเดียวกัน จนหลุด Learning Phase', en: '1. 24h Cooldown: Prevents repeated scaling in a single day, protecting learning phase' }) }}</li>
                      <li>{{ l({ lo: '2. Budget Cap: ຕັ້ງເພດານສູງສຸດຕໍ່ມື້ ປ້ອງກັນງົບເກີນຄວບຄຸມ', th: '2. Budget Cap: ตั้งเพดานสูงสุดต่อวัน ป้องกันงบไหลเกินควบคุม', en: '2. Daily Budget Cap: Limits maximum daily spend' }) }}</li>
                      <li>{{ l({ lo: '3. Audit Log: ບັນທຶກທຸກການສະເກວ ແລະ ການປິດແອດ ໃຫ້ທ່ານກວດສອບຍ້ອນຫຼັງໄດ້ຕະຫຼອດເວລາ', th: '3. Audit Log: บันทึกทุกการ Scale และการหยุดแอด ให้คุณตรวจสอบย้อนหลังได้ตลอดเวลา', en: '3. Audit Log: Records every single action with timestamp and before/after values' }) }}</li>
                    </ul>
                  </div>
                </div>
              </v-alert>
            </div>
          </v-window-item>

          <!-- Tab 2: Activity Audit Log -->
          <v-window-item value="logs">
            <div class="autopilot-logs-container">
              <div class="d-flex align-center justify-space-between mb-3">
                <span class="text-caption text-medium-emphasis">
                  {{ l({ lo: `ທັງໝົດ ${autoPilotLogs.length} ລາຍການຫຼ້າສຸດ`, th: `ทั้งหมด ${autoPilotLogs.length} รายการล่าสุด`, en: `Total ${autoPilotLogs.length} recent event(s)` }) }}
                </span>
                <v-btn
                  v-if="autoPilotLogs.length"
                  size="small"
                  variant="text"
                  color="error"
                  prepend-icon="mdi-trash-can-outline"
                  @click="clearAutoPilotLogs"
                >
                  {{ l({ lo: 'ລຶບປະຫວັດທັງໝົດ', th: 'ล้างประวัติทั้งหมด', en: 'Clear logs' }) }}
                </v-btn>
              </div>

              <div v-if="autoPilotLogs.length" class="autopilot-logs-list">
                <div v-for="log in autoPilotLogs" :key="log.id" class="autopilot-log-entry" :class="log.action.toLowerCase()">
                  <div class="log-icon-col">
                    <v-avatar size="28" :color="log.action === 'SCALE' ? 'success' : 'error'" variant="tonal">
                      <v-icon :icon="log.action === 'SCALE' ? 'mdi-rocket-launch' : 'mdi-pause'" size="16" />
                    </v-avatar>
                  </div>
                  <div class="log-content-col">
                    <div class="d-flex align-center justify-space-between flex-wrap gap-1">
                      <strong class="log-entity-name">{{ log.entityName }}</strong>
                      <span class="log-time">{{ new Date(log.timestamp).toLocaleString() }}</span>
                    </div>
                    <div class="log-detail">{{ log.detail }}</div>
                    <div v-if="log.beforeValue && log.afterValue" class="log-values">
                      <span class="text-medium-emphasis">{{ log.beforeValue }}</span>
                      <v-icon icon="mdi-arrow-right" size="14" class="mx-1" />
                      <strong class="text-success">{{ log.afterValue }}</strong>
                    </div>
                  </div>
                  <div class="log-status-col">
                    <v-icon :icon="log.success ? 'mdi-check-circle' : 'mdi-alert-circle'" :color="log.success ? 'success' : 'error'" size="18" />
                  </div>
                </div>
              </div>

              <div v-else class="empty-inline compact pa-8 text-center">
                <v-icon icon="mdi-clipboard-text-clock-outline" size="48" color="grey" class="mb-2" />
                <h4 class="font-weight-bold">{{ l({ lo: 'ຍັງບໍ່ມີປະຫວັດການເຮັດວຽກ', th: 'ยังไม่มีประวัติการทำงาน', en: 'No activity logs yet' }) }}</h4>
                <p class="text-caption text-medium-emphasis mb-0">
                  {{ l({ lo: 'ເມື່ອ Auto-Pilot ສະເກວງົບ ຫຼື ຢຸດແອດຂາດທຶນ ລະບົບຈະບັນທຶກລາຍລະອຽດໄວ້ທີ່ນີ້', th: 'เมื่อ Auto-Pilot ทำการ Scale งบ หรือหยุดแอดขาดทุน ระบบจะบันทึกรายละเอียดไว้ที่นี่', en: 'When Auto-Pilot scales budgets or pauses losing ads, entries will be recorded here.' }) }}
                </p>
              </div>
            </div>
          </v-window-item>
        </v-window>
      </v-card-text>

      <v-card-actions class="dialog-actions pa-4 border-t">
        <v-btn
          color="success"
          variant="flat"
          prepend-icon="mdi-flash"
          :loading="runningAutoPilot"
          class="font-weight-bold px-4"
          @click="runAutoPilot(true)"
        >
          {{ l({ lo: '⚡ ປະມວນຜົນດຽວນີ້ (Run Now)', th: '⚡ ประมวลผลตอนนี้ (Run Now)', en: '⚡ Run Now' }) }}
        </v-btn>
        <v-spacer />
        <v-btn variant="text" @click="autoPilotModalOpen = false">
          {{ l({ lo: 'ປິດ', th: 'ปิด', en: 'Close' }) }}
        </v-btn>
        <v-btn color="primary" variant="flat" prepend-icon="mdi-content-save" class="font-weight-bold px-4" @click="saveAutoPilotConfig(); autoPilotModalOpen = false; success = l({ lo: 'ບັນທຶກການຕັ້ງຄ່າ Auto-Pilot ສຳເລັດແລ້ວ', th: 'บันทึกการตั้งค่า Auto-Pilot สำเร็จแล้ว', en: 'Auto-Pilot settings saved successfully.' })">
          {{ l({ lo: 'ບັນທຶກການຕັ້ງຄ່າ', th: 'บันทึกการตั้งค่า', en: 'Save Settings' }) }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Interactive Visual Guide Dialog (Walkthrough with Pointer Callouts) -->
  <v-dialog v-model="guideModalOpen" max-width="960" scrollable>
    <v-card rounded="xl" class="guide-dialog-card">
      <v-card-title class="dialog-title d-flex align-center justify-space-between pb-3 border-b">
        <div class="d-flex align-center gap-2">
          <v-avatar color="info" variant="tonal" size="36">
            <v-icon icon="mdi-compass-outline" size="22" />
          </v-avatar>
          <div>
            <div class="text-caption font-weight-bold text-info letter-spacing-1">ONBOARDING & VISUAL WALKTHROUGH</div>
            <h2 class="text-h6 font-weight-bold mb-0">
              {{ l({ lo: 'ຄູ່ມືເລີ່ມຕົ້ນ ແລະ ແນະນຳການໃຊ້ງານລະບົບໂຄສະນາ', th: 'คู่มือเริ่มต้นและแนะนำการใช้งานระบบโฆษณา', en: 'Getting Started & Visual Walkthrough' }) }}
            </h2>
          </div>
        </div>
        <v-btn icon="mdi-close" variant="text" @click="guideModalOpen = false" />
      </v-card-title>

      <!-- Quick Switch to Live Tour Banner -->
      <div class="px-4 py-2 bg-primary-subtle d-flex align-center justify-space-between flex-wrap gap-2 border-b">
        <div class="d-flex align-center gap-2">
          <v-icon icon="mdi-lightbulb-on-outline" color="primary" size="18" />
          <span class="text-caption font-weight-medium">
            {{ l({ lo: 'ຕ້ອງການໃຫ້ລະບົບພາທົວ ແລະ ຊີ້ບອກເທິງໜ້າເວັບແທ້ເລີຍບໍ່?', th: 'ต้องการให้ระบบพาทัวร์และชี้บอกบนหน้าเว็บจริงเลยหรือไม่?', en: 'Want an interactive tour pointing directly at real elements on this page?' }) }}
          </span>
        </div>
        <v-btn
          size="small"
          color="primary"
          variant="flat"
          prepend-icon="mdi-crosshairs-gps"
          class="font-weight-bold"
          @click="startLiveTour"
        >
          {{ l({ lo: '🎯 ເລີ່ມພາທົວໜ້າເວັບແທ້ດຽວນີ້', th: '🎯 เริ่มพาทัวร์หน้าเว็บจริงตอนนี้', en: '🎯 Start Live Tour Now' }) }}
        </v-btn>
      </div>

      <!-- Stepper Header Navigation -->
      <div class="guide-stepper-bar px-4 py-3 border-b d-flex align-center justify-space-between flex-wrap gap-2">
        <div class="d-flex align-center flex-wrap gap-2">
          <v-btn
            v-for="s in [1, 2, 3, 4, 5]"
            :key="s"
            size="small"
            :variant="guideStep === s ? 'flat' : 'tonal'"
            :color="guideStep === s ? 'primary' : undefined"
            class="guide-step-chip font-weight-bold"
            @click="guideStep = s"
          >
            <span class="step-num">{{ s }}</span>
            <span class="ml-1">
              {{ s === 1 ? l({ lo: '1. ສ້າງສິນຄ້າ & ຜູກບັອດ', th: '1. สร้างสินค้า & ผูกบอท', en: '1. Product & Bot' })
               : s === 2 ? l({ lo: '2. ເລືອກບັນຊີ & 7 ມື້', th: '2. เลือกบัญชี & 7 วัน', en: '2. Account & Range' })
               : s === 3 ? l({ lo: '3. ອ່ານຄ່າຕົວເລກແທ້', th: '3. อ่านค่าตัวเลขแท้', en: '3. Real Metrics' })
               : s === 4 ? l({ lo: '4. ຕັ້ງກົດ & Budget Cap', th: '4. ตั้งกฎ & Budget Cap', en: '4. Rules & Cap' })
               : l({ lo: '5. ສະເກວ & Cooldown', th: '5. สเกล & Cooldown', en: '5. Scaling & Cooldown' }) }}
            </span>
          </v-btn>
        </div>
        <v-chip size="small" color="primary" variant="outlined" class="font-weight-bold">
          {{ l({ lo: `ຂັ້ນຕອນ ${guideStep} / 5`, th: `ขั้นตอน ${guideStep} / 5`, en: `Step ${guideStep} of 5` }) }}
        </v-chip>
      </div>

      <v-card-text class="pa-4 pa-md-6 guide-body-content">
        <!-- Step 1: Product Creation & Linking Auto-Reply Flow to Ads -->
        <div v-if="guideStep === 1" class="guide-step-container">
          <div class="guide-step-header mb-4">
            <div class="step-badge-pill mb-1 text-primary font-weight-bold">
              <v-icon icon="mdi-numeric-1-circle" size="18" class="mr-1" />
              {{ l({ lo: 'ຂັ້ນຕອນພື້ນຖານສຳຄັນທີ່ສຸດ (Foundation Setup)', th: 'ขั้นตอนพื้นฐานสำคัญที่สุด (Foundation Setup)', en: 'Foundation Setup' }) }}
            </div>
            <h3 class="text-h6 font-weight-bold mb-1">
              {{ l({ lo: 'ສ້າງສິນຄ້າ ແລະ ຜູກບັອດຕອບແຊັດ Auto-Reply ເຂົ້າກັບໂຄສະນາ (Ads)', th: 'สร้างสินค้า และผูกบอทตอบแชท Auto-Reply เข้ากับโฆษณา (Ads)', en: 'Create Product & Link Auto-Reply to Ads' }) }}
            </h3>
            <p class="text-body-2 text-medium-emphasis mb-0">
              {{ l({ lo: 'ຈຸດເລີ່ມຕົ້ນຂອງທຸກຢ່າງ: ຖ້າບໍ່ຜູກສິນຄ້າ ແລະ ບັອດຕອບແຊັດເຂົ້າກັບແອດ ລະບົບຈະບໍ່ສາມາດຕິດຕາມຍອດຊື້ (Purchases) ທີ່ແທ້ຈິງມາສະເກວໄດ້.', th: 'จุดเริ่มต้นของทุกอย่าง: หากไม่ผูกสินค้าและบอทเข้ากับแอด ระบบจะไม่สามารถติดตามยอดซื้อ (Purchases) ที่แท้จริงมา Scale ได้', en: 'The vital first step: linking product and auto-reply to ads allows true order tracking and scaling.' }) }}
            </p>
          </div>

          <div class="guide-mockup-wrapper">
            <div class="guide-mockup-card">
              <div class="mockup-header-bar">
                <span class="mockup-dot red"></span>
                <span class="mockup-dot yellow"></span>
                <span class="mockup-dot green"></span>
                <span class="mockup-title">{{ l({ lo: 'ຂະບວນການຜູກລະບົບ (Product & Auto-Reply Linking Flow)', th: 'กระบวนการผูกระบบ (Product & Auto-Reply Linking Flow)', en: 'Product & Auto-Reply Linking Flow' }) }}</span>
              </div>

              <div class="mockup-content pa-4">
                <!-- Item 1: Create Product -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: '1. ສິນຄ້າໃນລະບົບ (Product)', th: '1. สินค้าในระบบ (Product)', en: '1. System Product' }) }}</div>
                    <div class="mock-input-field">
                      <v-icon icon="mdi-package-variant-closed" color="primary" size="18" class="mr-2" />
                      <strong>{{ l({ lo: 'ເກີບ kito AG29 (ລະຫັດ: AG29-BLK)', th: 'รองเท้า kito AG29 (รหัส: AG29-BLK)', en: 'kito AG29 (SKU: AG29-BLK)' }) }}</strong>
                      <span class="text-caption text-medium-emphasis ml-2">129,000 ₭</span>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-plus-box-outline" size="14" class="mr-1" />
                      {{ l({ lo: '1. ສ້າງສິນຄ້າໃນເມນູ Products', th: '1. สร้างสินค้าในเมนู Products', en: '1. Create Product in Products menu' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ໃສ່ຊື່ສິນຄ້າ, ລະຫັດ SKU, ລາຄາຂາຍ, ແລະ ສະຕັອກ ໃຫ້ຄົບຖ້ວນເພື່ອໃຫ້ລະບົບຮູ້ຕົ້ນທຶນ ແລະ ຕັດສະຕັອກໄດ້.', th: 'ใส่ชื่อสินค้า, รหัส SKU, ราคาขาย และสต็อกให้ครบถ้วน เพื่อให้ระบบรู้ต้นทุนและตัดสต็อกได้', en: 'Set up product name, SKU, price, and stock so the system can calculate margins and stock.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Item 2: Auto-Reply Flow & Reply Set -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: '2. ບັອດຕອບແຊັດອັດຕະໂນມັດ (Auto-Reply Flow)', th: '2. บอทตอบแชทอัตโนมัติ (Auto-Reply Flow)', en: '2. Auto-Reply Flow' }) }}</div>
                    <div class="mock-input-field active-highlight">
                      <v-icon icon="mdi-robot-outline" color="success" size="18" class="mr-2" />
                      <strong class="text-success">{{ l({ lo: 'Flow: ຕອບປິດການຂາຍ kito AG29', th: 'Flow: ตอบปิดการขาย kito AG29', en: 'Flow: Sales Closing kito AG29' }) }}</strong>
                      <v-chip size="x-small" color="success" class="ml-2 font-weight-bold">{{ l({ lo: 'ຜູກກັບ kito AG29', th: 'ผูกกับ kito AG29', en: 'Linked to kito AG29' }) }}</v-chip>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot green"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box highlight-tip">
                    <h5 class="callout-title font-weight-bold text-success">
                      <v-icon icon="mdi-link-variant" size="14" class="mr-1" />
                      {{ l({ lo: '2. ຜູກ Flow ເຂົ້າກັບສິນຄ້ານັ້ນ (ເມນູ Auto Replies)', th: '2. ผูก Flow เข้ากับสินค้านั้น (เมนู Auto Replies)', en: '2. Link Flow to that product' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ໃນໜ້າ Auto Replies ຕອນສ້າງ Flow ໃຫ້ເລືອກ Product ເປັນ kito AG29 ເພື່ອໃຫ້ເວລາລູກຄ້າທັກມາ ບັອດຈະສົ່ງລາຄາ ແລະ ເອົາອໍເດີສິນຄ້ານັ້ນ.', th: 'ในหน้า Auto Replies ตอนสร้าง Flow ให้เลือก Product เป็น kito AG29 เพื่อให้บอทส่งราคาและรับออเดอร์สินค้านั้นได้ทันที', en: 'When creating the Flow, link it to the product so the bot automatically serves details and takes orders.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Item 3: Link Target to Meta Ads / Campaign -->
                <div class="mockup-element-row">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: '3. ເປົ້າໝາຍໂຄສະນາ (Target: Meta Ad / Campaign)', th: '3. เป้าหมายโฆษณา (Target: Meta Ad / Campaign)', en: '3. Ad Target' }) }}</div>
                    <div class="mock-input-field">
                      <v-icon icon="mdi-bullseye-arrow" color="warning" size="18" class="mr-2" />
                      <span>Target: Ad "kito AG29 - Video Review 01"</span>
                      <v-chip size="x-small" color="primary" class="ml-2 font-weight-bold">TRACKING ACTIVE</v-chip>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot orange"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-warning">
                      <v-icon icon="mdi-target" size="14" class="mr-1" />
                      {{ l({ lo: '3. ຜູກ Target ກັບແອດ (ສຳຄັນທີ່ສຸດ!)', th: '3. ผูก Target กับแอด (สำคัญที่สุด!)', en: '3. Connect Target to Ad (Vital!)' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ເລືອກ Target ເປັນ Ad ຫຼື Campaign ທີ່ເຈົ້າຍິງ. ເວລາລູກຄ້າສັ່ງຊື້ ລະບົບຈະຮູ້ທັນທີວ່າເປັນຍອດຂາຍຂອງແອດນີ້ ແລະ ນັບເປັນ Purchases ໃນໜ້າ Ads Manager ນີ້ທັນທີ!', th: 'เลือก Target เป็น Ad หรือ Campaign ที่คุณยิง เมื่อลูกค้าสั่งซื้อ ระบบจะรู้ทันทีว่ายอดขายมาจากแอดนี้ และส่งข้อมูลมาสะเกวที่หน้านี้', en: 'Target the ad running on Meta. When a lead orders, the system attributes it directly to this ad for scaling!' }) }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 2: Account Selection, 7-Day Window & Product Filter -->
        <div v-else-if="guideStep === 2" class="guide-step-container">
          <div class="guide-step-header mb-4">
            <div class="step-badge-pill mb-1 text-primary font-weight-bold">
              <v-icon icon="mdi-numeric-2-circle" size="18" class="mr-1" />
              {{ l({ lo: 'ສິ່ງທີ່ຕ້ອງເຮັດກ່ອນໝູ່ໝົດ (First Things First)', th: 'สิ่งที่ต้องทำก่อนเป็นอันดับแรก (First Things First)', en: 'First Things First' }) }}
            </div>
            <h3 class="text-h6 font-weight-bold mb-1">
              {{ l({ lo: 'ເລືອກບັນຊີ ແລະ ຕັ້ງຄ່າຊ່ວງເວລາ (Select Account & Date Range)', th: 'เลือกบัญชี และตั้งค่าช่วงเวลา (Select Account & Date Range)', en: 'Select Account & Date Range' }) }}
            </h3>
            <p class="text-body-2 text-medium-emphasis mb-0">
              {{ l({ lo: 'ກ່ອນຈະຕັດສິນໃຈສະເກວ ຫຼື ປິດແອດ ຕ້ອງໃຫ້ແນ່ໃຈວ່າທ່ານກຳລັງເບິ່ງຂໍ້ມູນທີ່ສະຖຽນ ແລະ ຖືກຕ້ອງຕາມສິນຄ້າຕົວຈິງ.', th: 'ก่อนตัดสินใจ Scale หรือหยุดแอด ต้องมั่นใจว่ากำลังดูข้อมูลที่นิ่งและถูกต้องตามสินค้าจริง', en: 'Before scaling or pausing ads, ensure you are looking at stable and accurate data for the actual product.' }) }}
            </p>
          </div>

          <div class="guide-mockup-wrapper">
            <div class="guide-mockup-card">
              <div class="mockup-header-bar">
                <span class="mockup-dot red"></span>
                <span class="mockup-dot yellow"></span>
                <span class="mockup-dot green"></span>
                <span class="mockup-title">{{ l({ lo: 'ແຖບເລືອກຂໍ້ມູນ (Filter Bar)', th: 'แถบเลือกข้อมูล (Filter Bar)', en: 'Filter Bar Preview' }) }}</span>
              </div>

              <div class="mockup-content pa-4">
                <!-- Mockup Item 1: Ad Account -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: 'ບັນຊີໂຄສະນາ (Ad Account)', th: 'บัญชีโฆษณา (Ad Account)', en: 'Ad Account' }) }}</div>
                    <div class="mock-input-field">
                      <v-icon icon="mdi-facebook" color="primary" size="18" class="mr-2" />
                      <span>Facebook: ຮ້ານຂາຍເກີບ kito Official</span>
                      <v-icon icon="mdi-chevron-down" size="18" class="ml-auto" />
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-cursor-default-click" size="14" class="mr-1" />
                      {{ l({ lo: '1. ເລືອກບັນຊີທີ່ຕ້ອງການກວດ', th: '1. เลือกบัญชีที่ต้องการตรวจ', en: '1. Select target account' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ເລືອກບັນຊີ Facebook Ad Account ທີ່ຕ້ອງການວິເຄາະ ແລະ ຄວບຄຸມງົບປະມານ.', th: 'เลือกบัญชี Facebook Ad Account ที่ต้องการวิเคราะห์และจัดการงบประมาณ', en: 'Select the Facebook Ad Account to evaluate and manage.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Mockup Item 2: Date Preset -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: 'ໄລຍະເວລາ (Date Preset)', th: 'ช่วงเวลา (Date Preset)', en: 'Date Preset' }) }}</div>
                    <div class="mock-input-field active-highlight">
                      <v-icon icon="mdi-calendar-range" color="primary" size="18" class="mr-2" />
                      <strong class="text-primary font-weight-bold">{{ l({ lo: '7 ມື້ຫຼ້າສຸດ (Last 7 Days)', th: '7 วันล่าสุด (Last 7 Days)', en: 'Last 7 Days' }) }}</strong>
                      <v-chip size="x-small" color="success" class="ml-2 font-weight-bold">{{ l({ lo: 'ແນະນຳ', th: 'แนะนำ', en: 'Recommended' }) }}</v-chip>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot green"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box highlight-tip">
                    <h5 class="callout-title font-weight-bold text-success">
                      <v-icon icon="mdi-star" size="14" class="mr-1 text-warning" />
                      {{ l({ lo: '2. ເລືອກ 7 ມື້ຫຼ້າສຸດ (ສຳຄັນທີ່ສຸດ!)', th: '2. เลือก 7 วันล่าสุด (สำคัญที่สุด!)', en: '2. Select Last 7 Days (Crucial!)' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ຢ່າເບິ່ງສະເພາະມື້ນີ້ ເພາະຂໍ້ມູນຍັງບໍ່ສະຖຽນ. ໄລຍະ 7 ມື້ ຈະຊ່ວຍໃຫ້ເຫັນຕົ້ນທຶນຕໍ່ອໍເດີ (CPA) ແລະ ຈຳນວນການຊື້ທີ່ແທ້ຈິງ.', th: 'อย่าดูแค่วันนี้เพราะข้อมูลยังไม่นิ่ง ช่วง 7 วันจะช่วยให้เห็น CPA และยอดซื้อที่แท้จริง', en: 'Avoid single-day view as data fluctuates. 7-day window gives stable CPA and true order conversion.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Mockup Item 3: Product Filter -->
                <div class="mockup-element-row">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: 'ຟິວເຕີສິນຄ້າ (Product Scope Filter)', th: 'ฟิลเตอร์สินค้า (Product Scope Filter)', en: 'Product Scope Filter' }) }}</div>
                    <div class="mock-input-field">
                      <v-icon icon="mdi-package-variant-closed" color="warning" size="18" class="mr-2" />
                      <span>{{ l({ lo: 'ເກີບ kito AG29 (ຮຸ່ນຂາຍດີ)', th: 'รองเท้า kito AG29 (รุ่นขายดี)', en: 'kito AG29 (Bestseller)' }) }}</span>
                      <v-chip size="x-small" color="info" class="ml-2">{{ l({ lo: '12 ແອດ', th: '12 แอด', en: '12 ads' }) }}</v-chip>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-filter" size="14" class="mr-1" />
                      {{ l({ lo: '3. ເລືອກສິນຄ້າເພື່ອຄຸມສະເພາະກຸ່ມ', th: '3. เลือกสินค้าเพื่อคุมเฉพาะกลุ่ม', en: '3. Filter by product' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ຖ້າທ່ານເລືອກສິນຄ້າໃດໜຶ່ງ, ທັງຕາຕະລາງ ແລະ Auto-Pilot ຈະສະແດງ ແລະ ຈັດການສະເພາະແອດທີ່ຜູກກັບສິນຄ້ານັ້ນ!', th: 'หากคุณเลือกสินค้าใด ทั้งตารางและ Auto-Pilot จะแสดงและจัดการเฉพาะแอดของสินค้านั้น', en: 'If selected, dashboard metrics and Auto-Pilot will isolate actions specifically to this product.' }) }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 3: Understanding Real Metrics & Decision Signals -->
        <div v-else-if="guideStep === 3" class="guide-step-container">
          <div class="guide-step-header mb-4">
            <div class="step-badge-pill mb-1 text-primary font-weight-bold">
              <v-icon icon="mdi-numeric-3-circle" size="18" class="mr-1" />
              {{ l({ lo: 'ວິທີອ່ານຄ່າຕົວເລກທີ່ແທ້ຈິງ', th: 'วิธีอ่านค่าตัวเลขที่แท้จริง', en: 'Understanding Real Metrics' }) }}
            </div>
            <h3 class="text-h6 font-weight-bold mb-1">
              {{ l({ lo: 'ແຍກແອດທີ່ສ້າງກຳໄລ (Winners) ອອກຈາກແອດຂາດທຶນ (Losers)', th: 'แยกแอดที่สร้างกำไร (Winners) ออกจากแอดขาดทุน (Losers)', en: 'Distinguish Winners from Losing Ads' }) }}
            </h3>
            <p class="text-body-2 text-medium-emphasis mb-0">
              {{ l({ lo: 'ລະບົບຈະຄິດໄລ່ຈາກຍອດຊື້ແທ້ຈິງທີ່ຜູກກັບອໍເດີ/Lead ໃນລະບົບ ບໍ່ແມ່ນຕົວເລກປະເມີນລອຍໆ.', th: 'ระบบจะคำนวณจากยอดซื้อแท้จริงที่ผูกกับออเดอร์/Lead ไม่ใช่ตัวเลขลอยๆ', en: 'The system computes real purchases matched with verified orders, not inflated estimates.' }) }}
            </p>
          </div>

          <div class="guide-mockup-wrapper">
            <div class="guide-mockup-card">
              <div class="mockup-header-bar">
                <span class="mockup-dot red"></span>
                <span class="mockup-dot yellow"></span>
                <span class="mockup-dot green"></span>
                <span class="mockup-title">{{ l({ lo: 'ສັນຍານຕົວເລກ (Metric Signals & KPI Cards)', th: 'สัญญาณตัวเลข (Metric Signals & KPI Cards)', en: 'Metric Signals & KPI Cards' }) }}</span>
              </div>

              <div class="mockup-content pa-4">
                <!-- Mock Metric 0: Messaging Conversations -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative kpi-card-mockup info">
                    <div class="d-flex justify-space-between align-center">
                      <span class="text-caption font-weight-bold">{{ l({ lo: 'ຄົນສົ່ງຂໍ້ຄວາມ (Conversations)', th: 'คนทักข้อความ (Conversations)', en: 'Conversations' }) }}</span>
                      <v-chip size="x-small" color="info" class="font-weight-bold">CONVERSIONS</v-chip>
                    </div>
                    <div class="kpi-value text-info font-weight-bold text-h6 mt-1">28 {{ l({ lo: 'ຄົນ', th: 'คน', en: 'people' }) }}</div>
                    <div class="kpi-sub text-caption">{{ l({ lo: 'ຕົ້ນທຶນ/ຂໍ້ຄວາມ: 12,000 ₭', th: 'ต้นทุน/ข้อความ: 12,000 ₭', en: 'Cost/msg: 12,000 ₭' }) }}</div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot cyan"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-message-text-outline" size="14" class="mr-1" />
                      {{ l({ lo: 'Conversations: ຈຳນວນຄົນທັກແຊັດຕົວຈິງ', th: 'Conversations: จำนวนคนทักแชทจริง', en: 'Conversations: Real Inquiries' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ສະແດງຈຳນວນຄົນທີ່ກົດແອດແລ້ວສົ່ງຂໍ້ຄວາມເຂົ້າມາສອບຖາມ ພ້ອມຕົ້ນທຶນສະເລ່ຍຕໍ່ 1 ຄົນທັກແຊັດ.', th: 'แสดงจำนวนคนที่คลิกแอดแล้วส่งข้อความเข้ามาสอบถาม พร้อมต้นทุนเฉลี่ยต่อ 1 คนทักแชท', en: 'Tracks how many people messaged your page from ads, with average cost per conversation.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Mock Metric 1: Purchases & CPA Winner -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative kpi-card-mockup winner">
                    <div class="d-flex justify-space-between align-center">
                      <span class="text-caption font-weight-bold">{{ l({ lo: 'ການຊື້ (Purchases)', th: 'การซื้อ (Purchases)', en: 'Purchases' }) }}</span>
                      <v-chip size="x-small" color="success" class="font-weight-bold">WINNER</v-chip>
                    </div>
                    <div class="kpi-value text-success font-weight-bold text-h6 mt-1">5 {{ l({ lo: 'ອໍເດີ', th: 'ออเดอร์', en: 'orders' }) }}</div>
                    <div class="kpi-sub text-caption">CPA: 35,000 ₭ / ອໍເດີ ({{ l({ lo: 'ເປົ້າໝາຍ', th: 'เป้าหมาย', en: 'Target' }) }} 50,000 ₭)</div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot green"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-success">
                      <v-icon icon="mdi-trophy-outline" size="14" class="mr-1" />
                      {{ l({ lo: 'ແອດ Winner: ພ້ອມສະເກວງົບ (+20%)', th: 'แอด Winner: พร้อม Scale งบ (+20%)', en: 'Winner Ad: Ready to Scale (+20%)' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ຖ້າການຊື້ >= 3 ອໍເດີ ແລະ CPA ຕ່ຳກວ່າເກນ ແປວ່າແອດນີ້ມີກຳໄລດີ ລະບົບຈະແນະນຳໃຫ້ສະເກວເພີ່ມງົບ.', th: 'ถ้าการซื้อ >= 3 ออเดอร์และ CPA ต่ำกว่าเกณฑ์ แอดนี้มีกำไรดี ระบบจะแนะนำให้ Scale เพิ่มงบ', en: 'If purchases >= 3 orders and CPA is below target, the ad is profitable and ready to scale.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Mock Metric 2: CTR Stop-Loss -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative kpi-card-mockup loser">
                    <div class="d-flex justify-space-between align-center">
                      <span class="text-caption font-weight-bold">{{ l({ lo: 'CTR (Click-Through Rate)', th: 'CTR (Click-Through Rate)', en: 'CTR' }) }}</span>
                      <v-chip size="x-small" color="error" class="font-weight-bold">LOSER SIGNAL</v-chip>
                    </div>
                    <div class="kpi-value text-error font-weight-bold text-h6 mt-1">0.45%</div>
                    <div class="kpi-sub text-caption">{{ l({ lo: 'ໃຊ້ງົບ 85,000 ₭ • 0 ອໍເດີ', th: 'ใช้งบ 85,000 ₭ • 0 ออเดอร์', en: 'Spent 85,000 ₭ • 0 orders' }) }}</div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot red"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-error">
                      <v-icon icon="mdi-alert-octagon-outline" size="14" class="mr-1" />
                      {{ l({ lo: 'ສັນຍານແອດຂາດທຶນ (Stop-Loss)', th: 'สัญญาณแอดขาดทุน (Stop-Loss)', en: 'Losing Ad Signal (Stop-Loss)' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ຖ້າໃຊ້ງົບໄປແລ້ວ ແຕ່ຍັງ 0 ອໍເດີ ແລະ CTR ຕ່ຳກວ່າ 1.0% ແປວ່າຄົນບໍ່ສົນໃຈ ບັອດຈະສັ່ງປິດແອດເພື່ອຕັດຂາດທຶນ.', th: 'ถ้าใช้งบไปแล้วแต่ยัง 0 ออเดอร์ และ CTR ต่ำกว่า 1.0% บอทจะหยุดแอดเพื่อตัดขาดทุนทันที', en: 'If spend is spent with 0 purchases and CTR < 1.0%, bot triggers Stop-Loss to prevent money burn.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Mock Metric 3: Frequency -->
                <div class="mockup-element-row">
                  <div class="mockup-element-box flex-grow-1 position-relative kpi-card-mockup warning">
                    <div class="d-flex justify-space-between align-center">
                      <span class="text-caption font-weight-bold">{{ l({ lo: 'Frequency (ຄວາມຖີ່)', th: 'Frequency (ความถี่)', en: 'Frequency' }) }}</span>
                      <v-chip size="x-small" color="warning" class="font-weight-bold">AUDIENCE FATIGUE</v-chip>
                    </div>
                    <div class="kpi-value text-warning font-weight-bold text-h6 mt-1">2.75 {{ l({ lo: 'ຄັ້ງ', th: 'ครั้ง', en: 'times' }) }}</div>
                    <div class="kpi-sub text-caption">{{ l({ lo: 'ກຸ່ມເປົ້າໝາຍເລີ່ມອີ່ມໂຕ (ເກີນ 2.5)', th: 'กลุ่มเป้าหมายเริ่มอิ่มตัว (เกิน 2.5)', en: 'Audience fatigue (exceeds 2.5)' }) }}</div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot orange"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-warning">
                      <v-icon icon="mdi-account-multiple-check" size="14" class="mr-1" />
                      {{ l({ lo: 'ຄວາມຖີ່ບໍ່ຄວນເກີນ 2.5', th: 'ความถี่ไม่ควรเกิน 2.5', en: 'Frequency must not exceed 2.5' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ຖ້າຄົນເຫັນຊ້ຳເກີນ 2.5 ຄັ້ງ, ລະບົບຈະລັອກບໍ່ໃຫ້ສະເກວເພີ່ມງົບ ເພາະກຸ່ມເປົ້າໝາຍອີ່ມໂຕ ອາດເຮັດໃຫ້ຕົ້ນທຶນແພງຂຶ້ນ.', th: 'ถ้าคนเห็นซ้ำเกิน 2.5 ครั้ง ระบบจะล็อคไม่ให้ Scale เพิ่มงบเพื่อไม่ให้ต้นทุนพุ่งสูง', en: 'When frequency > 2.5, scaling is blocked to prevent ad fatigue and sharp CPA spikes.' }) }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 4: Auto-Pilot Rules, Global 20% Standard & Daily Budget Cap -->
        <div v-else-if="guideStep === 4" class="guide-step-container">
          <div class="guide-step-header mb-4">
            <div class="step-badge-pill mb-1 text-primary font-weight-bold">
              <v-icon icon="mdi-numeric-4-circle" size="18" class="mr-1" />
              {{ l({ lo: 'ຕັ້ງຄ່າກົດ Auto-Pilot & ຄວາມປອດໄພ', th: 'ตั้งค่ากฎ Auto-Pilot & ความปลอดภัย', en: 'Rules & Safety Settings' }) }}
            </div>
            <h3 class="text-h6 font-weight-bold mb-1">
              {{ l({ lo: 'ສູດສະເກວ +20% ມາດຕະຖານໂລກ ແລະ ເພດານງົບສູງສຸດ', th: 'สูตรสเกล +20% มาตรฐานโลก และเพดานงบสูงสุด', en: 'Global 20% Formula & Daily Budget Cap' }) }}
            </h3>
            <p class="text-body-2 text-medium-emphasis mb-0">
              {{ l({ lo: 'ປັບແຕ່ງທຸກເງື່ອນໄຂໄດ້ຕາມໃຈ ແລະ ລັອກງົບສູງສຸດບໍ່ໃຫ້ໄຫຼເກີນຄວບຄຸມ.', th: 'ปรับแต่งทุกเงื่อนไขได้ตามใจ และล็อคงบสูงสุดไม่ให้ไหลเกินควบคุม', en: 'Customize rules freely and set hard budget ceiling to avoid runaway spend.' }) }}
            </p>
          </div>

          <div class="guide-mockup-wrapper">
            <div class="guide-mockup-card">
              <div class="mockup-header-bar">
                <span class="mockup-dot red"></span>
                <span class="mockup-dot yellow"></span>
                <span class="mockup-dot green"></span>
                <span class="mockup-title">{{ l({ lo: 'ການຕັ້ງຄ່າກົດອັດຕະໂນມັດ (Auto-Pilot Config)', th: 'การตั้งค่ากฎอัตโนมัติ (Auto-Pilot Config)', en: 'Auto-Pilot Config' }) }}</span>
              </div>

              <div class="mockup-content pa-4">
                <!-- Scale 20% Mockup -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: 'ເປີເຊັນເພີ່ມງົບຕໍ່ຄັ້ງ (Scale Percent)', th: 'เปอร์เซ็นต์เพิ่มงบต่อครั้ง (Scale Percent)', en: 'Scale Percentage' }) }}</div>
                    <div class="mock-input-field active-highlight">
                      <v-icon icon="mdi-rocket-launch" color="success" size="18" class="mr-2" />
                      <strong class="text-success text-h6 font-weight-bold">+20%</strong>
                      <span class="text-caption text-medium-emphasis ml-2">{{ l({ lo: '(ປ່ຽນເປັນ 10%, 15%, 25% ໄດ້)', th: '(เปลี่ยนเป็น 10%, 15%, 25% ได้)', en: '(Customizable: 10%, 15%, 25%)' }) }}</span>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot green"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-shield-check" size="14" class="mr-1" />
                      {{ l({ lo: 'ເປັນຫຍັງຕ້ອງເພີ່ມເທື່ອລະ 20%?', th: 'ทำไมต้องเพิ่มทีละ 20%?', en: 'Why 20% scale increments?' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ເພາະ Meta Algorithm ຈະຣີເຊັດການຮຽນຮູ້ (Reset Learning Phase) ທັນທີຖ້າປັບງົບເກີນ 20%. ການເພີ່ມ 20% ແມ່ນມາດຕະຖານທີ່ປອດໄພທີ່ສຸດ!', th: 'เพราะ Meta AI จะรีเซ็ต Learning Phase ทันทีหากปรับงบเกิน 20% การเพิ่ม 20% ปลอดภัยที่สุด', en: 'Meta AI resets the learning phase if budget jumps >20%. 20% is the safest global standard.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Budget Cap Mockup -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: 'ເພດານງົບສູງສຸດຕໍ່ມື້ (Max Daily Budget Cap)', th: 'เพดานงบสูงสุดต่อวัน (Max Daily Budget Cap)', en: 'Max Daily Budget Cap' }) }}</div>
                    <div class="mock-input-field">
                      <v-icon icon="mdi-cash-lock" color="warning" size="18" class="mr-2" />
                      <strong class="font-weight-bold">500,000 ₭ / {{ l({ lo: 'ມື້', th: 'วัน', en: 'day' }) }}</strong>
                      <v-chip size="x-small" color="warning" class="ml-2 font-weight-bold">HARD CAP</v-chip>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot orange"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box highlight-tip">
                    <h5 class="callout-title font-weight-bold text-warning">
                      <v-icon icon="mdi-lock" size="14" class="mr-1" />
                      {{ l({ lo: 'ເບຣກລັອກງົບ (Circuit Breaker 1)', th: 'เบรคล็อคงบ (Circuit Breaker 1)', en: 'Safety Cap (Circuit Breaker 1)' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ເມື່ອງົບແອດສະເກວຂຶ້ນມາຮອດ 500,000 ກີບແລ້ວ ບັອດຈະຢຸດສະເກວທັນທີ ບໍ່ມີທາງໄຫຼເກີນເພດານນີ້ເດັດຂາດ.', th: 'เมื่องบแอดสเกลขึ้นมาถึง 500,000 กีบ บอทจะหยุดสเกลทันที ไม่มีทางไหลเกินเพดานนี้เด็ดขาด', en: 'Once daily budget reaches 500,000 ₭, scaling stops permanently to guarantee budget safety.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Scope & Structure Mockup -->
                <div class="mockup-element-row">
                  <div class="mockup-element-box flex-grow-1 position-relative">
                    <div class="mock-input-label">{{ l({ lo: 'ຂອບເຂດ & ປະເພດແອດ (Scope & Target Structure)', th: 'ขอบเขต & ประเภทแอด (Scope & Target Structure)', en: 'Scope & Target Structure' }) }}</div>
                    <div class="mock-input-field">
                      <v-icon icon="mdi-bullseye-arrow" color="primary" size="18" class="mr-2" />
                      <span>{{ l({ lo: 'ສະເພາະສິນຄ້າທີ່ Filter • ສະເພາະ Campaign (CBO)', th: 'เฉพาะสินค้าที่ Filter • เฉพาะ Campaign (CBO)', en: 'Filter Product Only • Campaigns (CBO)' }) }}</span>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-cog" size="14" class="mr-1" />
                      {{ l({ lo: 'ເລືອກ CBO ຫຼື ABO ໄດ້ຕາມຕ້ອງການ', th: 'เลือก CBO หรือ ABO ได้ตามต้องการ', en: 'Target CBO or ABO' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ທ່ານສາມາດເລືອກໃຫ້ບັອດຄຸມສະເພາະ Campaign CBO, ຫຼື ສະເພາະ Ad Set ABO, ຫຼື ທຸກລະດັບໄດ້ຢ່າງອິດສະຫຼະ.', th: 'คุณสามารถเลือกให้บอทคุมเฉพาะ Campaign CBO หรือเฉพาะ Ad Set ABO ได้ตามโครงสร้างจริง', en: 'Isolate actions to Campaign CBO or Ad Set ABO according to your exact budget structure.' }) }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 5: Scaling & 24h Cooldown Lock & Incremental Purchases -->
        <div v-else-if="guideStep === 5" class="guide-step-container">
          <div class="guide-step-header mb-4">
            <div class="step-badge-pill mb-1 text-primary font-weight-bold">
              <v-icon icon="mdi-numeric-5-circle" size="18" class="mr-1" />
              {{ l({ lo: 'ວິທີສະເກວ & ລະບົບລັອກ Cooldown', th: 'วิธีสเกล & ระบบล็อค Cooldown', en: 'Scaling & 24h Cooldown Lock' }) }}
            </div>
            <h3 class="text-h6 font-weight-bold mb-1">
              {{ l({ lo: 'ກົນໄກປ້ອງກັນບໍ່ໃຫ້ສະເກວໄປເລື້ອຍໆ ໂດຍບໍ່ມີອໍເດີໃໝ່', th: 'กลไกป้องกันไม่ให้สเกลไปเรื่อยๆ โดยไม่มีออเดอร์ใหม่', en: 'Safety Mechanism Preventing Runaway Scaling' }) }}
            </h3>
            <p class="text-body-2 text-medium-emphasis mb-0">
              {{ l({ lo: 'ເລືອກໄດ້ທັງແບບ Auto-Pilot 24/7 ຫຼື ກົດອະນຸມັດດ້ວຍມືເອງ (Manual Mode).', th: 'เลือกได้ทั้งแบบ Auto-Pilot 24/7 หรือกดอนุมัติด้วยมือเอง (Manual Mode)', en: 'Choose between automated 24/7 Auto-Pilot or manual 1-click approvals.' }) }}
            </p>
          </div>

          <div class="guide-mockup-wrapper">
            <div class="guide-mockup-card">
              <div class="mockup-header-bar">
                <span class="mockup-dot red"></span>
                <span class="mockup-dot yellow"></span>
                <span class="mockup-dot green"></span>
                <span class="mockup-title">{{ l({ lo: 'ແຖວລາຍການແອດ (Ad Row Controls & Safety Status)', th: 'แถวรายการแอด (Ad Row Controls & Safety Status)', en: 'Ad Row Controls & Safety Status' }) }}</span>
              </div>

              <div class="mockup-content pa-4">
                <!-- Mock Row 1: Ready to Scale -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative ad-row-mockup">
                    <div class="d-flex align-center justify-space-between flex-wrap gap-2">
                      <div>
                        <strong class="d-block font-weight-bold">kito AG29 - Scaling Ad Set 01</strong>
                        <small class="text-medium-emphasis">{{ l({ lo: 'ງົບປະຈຸບັນ: 100,000 ₭/ມື້ • 4 ອໍເດີ', th: 'งบปัจจุบัน: 100,000 ₭/วัน • 4 ออเดอร์', en: 'Current: 100,000 ₭/day • 4 orders' }) }}</small>
                      </div>
                      <v-btn size="small" color="primary" variant="flat" prepend-icon="mdi-rocket-launch" class="font-weight-bold">
                        🚀 {{ l({ lo: 'ສະເກວ (+20%)', th: 'สเกล (+20%)', en: 'Scale (+20%)' }) }}
                      </v-btn>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-gesture-tap" size="14" class="mr-1" />
                      {{ l({ lo: 'ໂໝດ Manual: ກົດສະເກວດ້ວຍຕົນເອງ', th: 'โหมด Manual: กดสเกลด้วยตนเอง', en: 'Manual Mode: 1-Click Scale' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ຖ້າທ່ານປິດ Auto-Pilot, ທ່ານສາມາດກົດປຸ່ມ “🚀 ສະເກວ (+20%)” ຢູ່ແຕ່ລະແຖວເພື່ອອະນຸມັດປັບງົບດ້ວຍມືເອງໄດ້.', th: 'ถ้าปิด Auto-Pilot คุณสามารถกดปุ่ม “🚀 สเกล (+20%)” แต่ละแถวเพื่อปรับงบด้วยมือได้ตลอดเวลา', en: 'In manual mode, click the Scale button on any row to approve a +20% bump on your terms.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Mock Row 2: In Cooldown -->
                <div class="mockup-element-row mb-4">
                  <div class="mockup-element-box flex-grow-1 position-relative ad-row-mockup cooldown">
                    <div class="d-flex align-center justify-space-between flex-wrap gap-2">
                      <div>
                        <strong class="d-block font-weight-bold">kito AG29 - Winner Campaign</strong>
                        <small class="text-medium-emphasis">{{ l({ lo: 'ປັບແລ້ວ: 100,000 ₭ ➔ 120,000 ₭/ມື້', th: 'ปรับแล้ว: 100,000 ₭ ➔ 120,000 ₭/วัน', en: 'Scaled: 100,000 ₭ ➔ 120,000 ₭/day' }) }}</small>
                      </div>
                      <v-chip size="small" color="info" variant="tonal" class="font-weight-bold">
                        ⏳ {{ l({ lo: 'ສະເກວແລ້ວ (Cooldown ອີກ 21h 30m)', th: 'สเกลแล้ว (Cooldown อีก 21h 30m)', en: 'Scaled (Cooldown 21h 30m)' }) }}
                      </v-chip>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot cyan"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box highlight-tip">
                    <h5 class="callout-title font-weight-bold text-info">
                      <v-icon icon="mdi-timer-sand" size="14" class="mr-1" />
                      {{ l({ lo: 'ລັອກ Cooldown 24 ຊົ່ວໂມງ', th: 'ล็อค Cooldown 24 ชั่วโมง', en: '24h Cooldown Lock' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ລະບົບຈະລັອກບໍ່ໃຫ້ສະເກວຊ້ຳໃນມື້ດຽວກັນ ເພື່ອໃຫ້ເວລາ Meta AI ກະຈາຍງົບ ແລະ ປົກປ້ອງ Learning Phase.', th: 'ระบบจะล็อคไม่ให้สเกลซ้ำในวันเดียวกัน เพื่อให้เวลา Meta กระจายงบและปกป้อง Learning Phase', en: 'Ad is locked for 24h to prevent multi-scaling within the same day and safeguard learning stability.' }) }}
                    </p>
                  </div>
                </div>

                <!-- Mock Row 3: Incremental sales requirement -->
                <div class="mockup-element-row">
                  <div class="mockup-element-box flex-grow-1 position-relative ad-row-mockup info-state">
                    <div class="d-flex align-center justify-space-between flex-wrap gap-2">
                      <div>
                        <strong class="d-block font-weight-bold">{{ l({ lo: 'ເງື່ອນໄຂອໍເດີໃໝ່ (Incremental Sales)', th: 'เงื่อนไขออเดอร์ใหม่ (Incremental Sales)', en: 'Incremental Sales Requirement' }) }}</strong>
                        <small class="text-medium-emphasis">{{ l({ lo: 'ອໍເດີປັດຈຸບັນ: 4/4 ອໍເດີ (ຍັງບໍ່ມີອໍເດີໃໝ່ເພີ່ມ)', th: 'ออเดอร์ปัจจุบัน: 4/4 ออเดอร์ (ยังไม่มีออเดอร์ใหม่)', en: 'Orders: 4/4 (no new orders yet)' }) }}</small>
                      </div>
                      <v-chip size="small" color="secondary" variant="outlined">
                        🛑 {{ l({ lo: 'ລໍຖ້າອໍເດີໃໝ່', th: 'รอออเดอร์ใหม่', en: 'Awaiting new sales' }) }}
                      </v-chip>
                    </div>
                    <div class="guide-pointer-anchor right-center">
                      <span class="pointer-dot orange"></span>
                      <span class="pointer-line horizontal"></span>
                    </div>
                  </div>
                  <div class="guide-callout-box">
                    <h5 class="callout-title font-weight-bold text-warning">
                      <v-icon icon="mdi-shield-alert" size="14" class="mr-1" />
                      {{ l({ lo: 'ຈະບໍ່ສະເກວຕໍ່ ຖ້າບໍ່ມີອໍເດີໃໝ່!', th: 'จะไม่สเกลต่อถ้าไม่มีออเดอร์ใหม่!', en: 'No further scale without new sales!' }) }}
                    </h5>
                    <p class="callout-desc mb-0">
                      {{ l({ lo: 'ຫຼັງຈາກໝົດ 24h, ຖ້າແອດຍັງມີ 4 ອໍເດີເທົ່າເກົ່າ ບັອດຈະປະຕິເສດການສະເກວທັນທີ! ຕ້ອງມີອໍເດີໃໝ່ເພີ່ມຂຶ້ນແທ້ຈຶ່ງຈະສະເກວຕໍ່ໄດ້.', th: 'หลังหมด 24h ถ้าแอดยังมี 4 ออเดอร์เท่าเดิม บอทจะไม่สเกลต่อ ต้องมีออเดอร์ใหม่เข้ามาเพิ่มจึงจะสเกลได้', en: 'After 24h, if sales remain unchanged at 4 orders, Auto-Pilot refuses further scaling until new sales arrive.' }) }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </v-card-text>

      <v-card-actions class="dialog-actions pa-4 border-t d-flex align-center justify-space-between">
        <v-btn
          variant="outlined"
          :disabled="guideStep === 1"
          prepend-icon="mdi-arrow-left"
          @click="guideStep--"
        >
          {{ l({ lo: 'ຍ້ອນກັບ', th: 'ย้อนกลับ', en: 'Back' }) }}
        </v-btn>

        <div class="d-flex align-center gap-2">
          <v-btn
            variant="text"
            @click="guideModalOpen = false"
          >
            {{ l({ lo: 'ປິດ', th: 'ปิด', en: 'Close' }) }}
          </v-btn>

          <v-btn
            v-if="guideStep < 5"
            color="primary"
            variant="flat"
            append-icon="mdi-arrow-right"
            class="font-weight-bold px-4"
            @click="guideStep++"
          >
            {{ l({ lo: 'ຂັ້ນຕອນຕໍ່ໄປ', th: 'ขั้นตอนถัดไป', en: 'Next Step' }) }}
          </v-btn>

          <v-btn
            v-else
            color="success"
            variant="flat"
            prepend-icon="mdi-robot"
            class="font-weight-bold px-4"
            @click="guideModalOpen = false; autoPilotModalOpen = true"
          >
            {{ l({ lo: '🚀 ເຂົ້າສູ່ການຕັ້ງຄ່າ Auto-Pilot', th: '🚀 เข้าสู่การตั้งค่า Auto-Pilot', en: '🚀 Open Auto-Pilot' }) }}
          </v-btn>
        </div>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Live On-Page Tour Overlay & Floating Widget -->
  <Teleport to="body">
    <div v-if="isLiveTourActive" class="live-tour-root">
      <!-- Non-blocking backdrop tint -->
      <div class="live-tour-backdrop" @click="endLiveTour" />

      <!-- Floating Tour Popover Box -->
      <div class="live-tour-popover-card">
        <div class="tour-card-header d-flex align-center justify-space-between pa-3 border-b">
          <div class="d-flex align-center gap-2">
            <v-avatar color="primary" size="28" variant="tonal">
              <v-icon icon="mdi-crosshairs-gps" size="16" />
            </v-avatar>
            <div>
              <div class="tour-badge-text">LIVE ON-PAGE TOUR</div>
              <strong class="tour-title-text">
                {{ liveTourStep === 1 ? l({ lo: '1. ສ້າງສິນຄ້າ & ຜູກບັອດ Auto-Reply', th: '1. สร้างสินค้า & ผูกบอท Auto-Reply', en: '1. Product & Auto-Reply' })
                 : liveTourStep === 2 ? l({ lo: '2. ແຖບເລືອກຂໍ້ມູນຕົວຈິງ (Filter Bar)', th: '2. แถบเลือกข้อมูลจริง (Filter Bar)', en: '2. Real Filter Bar' })
                 : liveTourStep === 3 ? l({ lo: '3. ອ່ານຄ່າຕົວເລກທີ່ແທ້ຈິງ (KPI Cards)', th: '3. อ่านค่าตัวเลขจริง (KPI Cards)', en: '3. Real KPI Cards' })
                 : liveTourStep === 4 ? l({ lo: '4. ຄວບຄຸມ Auto-Pilot & ຕັ້ງ Budget Cap', th: '4. ควบคุม Auto-Pilot & ตั้ง Budget Cap', en: '4. Auto-Pilot & Safety Cap' })
                 : l({ lo: '5. ຕາຕະລາງແອດ, ປຸ່ມສະເກວ & Cooldown 24h', th: '5. ตารางแอด ปุ่มสเกล & Cooldown 24h', en: '5. Scaling Table & 24h Cooldown' }) }}
              </strong>
            </div>
          </div>
          <v-btn icon="mdi-close" size="small" variant="text" @click="endLiveTour" />
        </div>

        <v-progress-linear :model-value="liveTourStep * 20" color="primary" height="3" />

        <div class="tour-card-body pa-4">
          <!-- Step 1 Content -->
          <div v-if="liveTourStep === 1">
            <p class="text-body-2 mb-3">
              {{ l({ lo: 'ກ່ອນຈະສະເກວແອດ: ຕ້ອງສ້າງສິນຄ້າໃນລະບົບ ແລະ ຜູກບັອດຕອບແຊັດເຂົ້າກັບແອດກ່ອນ ເພື່ອໃຫ້ລະບົບຮູ້ຍອດຊື້ (Purchases) ທີ່ແທ້ຈິງ!', th: 'ก่อนจะสเกลแอด: ต้องสร้างสินค้าในระบบ และผูกบอทตอบแชทเข้ากับแอดก่อน เพื่อให้ระบบรู้ยอดซื้อที่แท้จริง', en: 'Before scaling ads: create products and link auto-reply to ads so the system captures real verified purchases!' }) }}
            </p>
            <div class="d-flex flex-wrap gap-2 mb-2">
              <v-btn to="/products" target="_blank" prepend-icon="mdi-package-variant-closed" size="small" color="primary" variant="tonal" class="font-weight-bold">
                📦 {{ l({ lo: 'ເປີດໜ້າສິນຄ້າ (Products)', th: 'เปิดหน้าสินค้า (Products)', en: 'Open Products' }) }}
              </v-btn>
              <v-btn to="/auto-replies" target="_blank" prepend-icon="mdi-robot" size="small" color="success" variant="tonal" class="font-weight-bold">
                🤖 {{ l({ lo: 'ເປີດໜ້າ Auto-Replies', th: 'เปิดหน้า Auto-Replies', en: 'Open Auto-Replies' }) }}
              </v-btn>
            </div>
            <small class="text-caption text-medium-emphasis">
              {{ l({ lo: '💡 ເມື່ອຜູກຮຽບຮ້ອຍແລ້ວ ກົດ "ຕໍ່ໄປ" ເພື່ອກວດໜ້າ Ads ຕົວຈິງ!', th: '💡 เมื่อผูกเรียบร้อยแล้ว กด "ถัดไป" เพื่อตรวจหน้า Ads นี้ได้เลย', en: '💡 Once linked, click "Next" to inspect this live Ads Manager!' }) }}
            </small>
          </div>

          <!-- Step 2 Content -->
          <div v-else-if="liveTourStep === 2">
            <div class="tour-spotlight-pointer-hint mb-2 text-primary font-weight-bold d-flex align-center gap-1">
              <v-icon icon="mdi-arrow-up-bold" size="16" class="bounce-up" />
              {{ l({ lo: 'ສັງເກດແຖບ Filter Bar ດ້ານເທິງທີ່ກຳລັງເຫຼື້ອມ:', th: 'สังเกตแถบ Filter Bar ด้านบนที่กำลังเรืองแสง:', en: 'Notice the spotlighted Filter Bar above:' }) }}
            </div>
            <ul class="text-body-2 pl-4 mb-2">
              <li><strong>{{ l({ lo: '1. ເລືອກບັນຊີ:', th: '1. เลือกบัญชี:', en: '1. Account:' }) }}</strong> {{ l({ lo: 'ເລືອກ Ad Account ທີ່ຕ້ອງການວິເຄາະ', th: 'เลือกบัญชีที่ต้องการ', en: 'Select target ad account' }) }}</li>
              <li><strong>{{ l({ lo: '2. ໄລຍະ 7 ມື້ (ສຳຄັນ!):', th: '2. ระยะ 7 วัน (สำคัญ!):', en: '2. Last 7 Days:' }) }}</strong> {{ l({ lo: 'ຢ່າເບິ່ງສະເພາະມື້ນີ້ ເພື່ອໃຫ້ຂໍ້ມູນສະຖຽນ', th: 'อย่าดูแค่วันนี้ เพื่อให้ข้อมูลนิ่ง', en: 'Avoid single-day data swings' }) }}</li>
              <li><strong>{{ l({ lo: '3. ຟິວເຕີສິນຄ້າ:', th: '3. ฟิลเตอร์สินค้า:', en: '3. Product Scope:' }) }}</strong> {{ l({ lo: 'ເລືອກສິນຄ້າ (ເຊັ່ນ kito AG29) ເພື່ອກວດສະເພາະແອດຂອງສິນຄ້ານີ້', th: 'เลือกสินค้าเพื่อดูเฉพาะแอดของสินค้านั้น', en: 'Filter by specific product e.g. kito AG29' }) }}</li>
            </ul>
          </div>

          <!-- Step 3 Content -->
          <div v-else-if="liveTourStep === 3">
            <div class="tour-spotlight-pointer-hint mb-2 text-primary font-weight-bold d-flex align-center gap-1">
              <v-icon icon="mdi-arrow-up-bold" size="16" class="bounce-up" />
              {{ l({ lo: 'ສັງເກດ 6 ກ່ອງຕົວເລກ KPI ດ້ານເທິງທີ່ກຳລັງເຫຼື້ອມ:', th: 'สังเกต 6 กล่องตัวเลข KPI ด้านบนที่กำลังเรืองแสง:', en: 'Notice the 6 KPI Cards above:' }) }}
            </div>
            <ul class="text-body-2 pl-4 mb-2">
              <li><strong>{{ l({ lo: '1. ຄົນສົ່ງຂໍ້ຄວາມ (Conversations):', th: '1. คนทักข้อความ (Conversations):', en: '1. Conversations:' }) }}</strong> {{ l({ lo: 'ຈຳນວນຄົນທີ່ສົ່ງຂໍ້ຄວາມມາຫາເພຈແທ້ ແລະ ຕົ້ນທຶນສະເລ່ຍ/ຂໍ້ຄວາມ', th: 'จำนวนคนที่ทักข้อความเข้ามา และต้นทุนเฉลี่ย/ข้อความ', en: 'People who messaged your page and cost per message' }) }}</li>
              <li><strong>{{ l({ lo: '2. Purchases & CPA:', th: '2. Purchases & CPA:', en: '2. Purchases & CPA:' }) }}</strong> {{ l({ lo: 'ຈຳນວນອໍເດີຊື້ແທ້ຈິງ. ຖ້າຊື້ >= 3 ແລະ CPA ຕ່ຳ = Winner ພ້ອມສະເກວ', th: 'จำนวนออเดอร์ซื้อจริง ถ้าซื้อ >= 3 และ CPA ต่ำ = Winner พร้อมสเกล', en: 'Verified orders. If >= 3 and low CPA = Winner ready to scale' }) }}</li>
              <li><strong>{{ l({ lo: '3. Spend (ລວມ VAT):', th: '3. Spend (รวม VAT):', en: '3. Spend (+VAT):' }) }}</strong> {{ l({ lo: 'ຍອດເງິນທີ່ຈ່າຍໃຫ້ Meta ຕົວຈິງບວກອາກອນ VAT ອັດຕະໂນມັດ', th: 'ยอดเงินจ่าย Meta จริงรวม VAT อัตโนมัติ', en: 'Actual gross ad spend with auto VAT breakdown' }) }}</li>
              <li><strong>{{ l({ lo: '4. CTR & Frequency:', th: '4. CTR & Frequency:', en: '4. CTR & Frequency:' }) }}</strong> {{ l({ lo: 'CTR < 1% ແລະ 0 ອໍເດີ = ເຂົ້າເກນ Stop-Loss; Frequency ບໍ່ຄວນເກີນ 2.5', th: 'CTR < 1% และ 0 ออเดอร์ = เกณฑ์หยุดแอด; Frequency ไม่ควรเกิน 2.5', en: 'CTR < 1% with 0 orders = Stop-loss; Frequency <= 2.5' }) }}</li>
            </ul>
          </div>

          <!-- Step 4 Content -->
          <div v-else-if="liveTourStep === 4">
            <div class="tour-spotlight-pointer-hint mb-2 text-success font-weight-bold d-flex align-center gap-1">
              <v-icon icon="mdi-arrow-up-bold" size="16" class="bounce-up" />
              {{ l({ lo: 'ສັງເກດແຖບຄວບຄຸມ Dual-System Banner ດ້ານເທິງ:', th: 'สังเกตแถบควบคุม Dual-System ด้านบน:', en: 'Notice the Dual-System Banner above:' }) }}
            </div>
            <p class="text-body-2 mb-2">
              {{ l({ lo: 'ທ່ານສາມາດສະຫຼັບລະຫວ່າງ Manual (ກົດສະເກວເອງ) ຫຼື Auto-Pilot (ບັອດເຝົ້າແອດ 24/7). ກົດປຸ່ມ "ຕັ້ງຄ່າກົດ & Log" ເພື່ອກຳນົດເພດານງົບ (Daily Budget Cap) ແລະ ສູດສະເກວ 20%!', th: 'คุณสามารถสลับระหว่าง Manual (กดสเกลเอง) หรือ Auto-Pilot (บอทเฝ้าแอด 24/7) กดปุ่ม "ตั้งค่ากฎ & Log" เพื่อตั้งเพดานงบสูงสุด!', en: 'Toggle between Manual and Auto-Pilot. Click Rules & Logs to configure Daily Budget Cap and 20% scaling!' }) }}
            </p>
          </div>

          <!-- Step 5 Content -->
          <div v-else-if="liveTourStep === 5">
            <div class="tour-spotlight-pointer-hint mb-2 text-primary font-weight-bold d-flex align-center gap-1">
              <v-icon icon="mdi-arrow-up-bold" size="16" class="bounce-up" />
              {{ l({ lo: 'ສັງເກດຕາຕະລາງແອດດ້ານເທິງ/ລຸ່ມ:', th: 'สังเกตตารางแอด:', en: 'Notice the ad table:' }) }}
            </div>
            <ul class="text-body-2 pl-4 mb-2">
              <li><strong>{{ l({ lo: 'ປຸ່ມ 🚀 ສະເກວ (+20%):', th: 'ปุ่ม 🚀 สเกล (+20%):', en: 'Scale Button (+20%):' }) }}</strong> {{ l({ lo: 'ກົດເພີ່ມງົບດ້ວຍມືໄດ້ທັນທີ', th: 'กดเพิ่มงบได้ทันที', en: 'Approve budget bump manually' }) }}</li>
              <li><strong>{{ l({ lo: 'ປ້າຍ ⏳ Cooldown 24h:', th: 'ป้าย ⏳ Cooldown 24h:', en: '24h Cooldown Badge:' }) }}</strong> {{ l({ lo: 'ລັອກ 24h ປ້ອງກັນບໍ່ໃຫ້ຫຼຸດ Meta Learning Phase', th: 'ล็อก 24h เพื่อปกป้อง Learning Phase', en: 'Locked for 24h to protect learning phase' }) }}</li>
              <li><strong>{{ l({ lo: 'ກົດອໍເດີໃໝ່ (Incremental Sales):', th: 'กฎออเดอร์ใหม่ (Incremental Sales):', en: 'Incremental Sales Guardrail:' }) }}</strong> {{ l({ lo: 'ຖ້າໝົດ 24h ແຕ່ບໍ່ມີອໍເດີໃໝ່ເພີ່ມຂຶ້ນ ລະບົບຈະບໍ່ສະເກວຕໍ່ເດັດຂາດ!', th: 'ถ้าหมด 24h แต่ไม่มีออเดอร์ใหม่เพิ่มขึ้น ระบบจะไม่สเกลต่อเด็ดขาด', en: 'No scaling without new orders!' }) }}</li>
            </ul>
          </div>
        </div>

        <div class="tour-card-actions pa-3 border-t d-flex align-center justify-space-between bg-surface-variant">
          <span class="text-caption font-weight-bold text-medium-emphasis">
            {{ l({ lo: `ຂັ້ນຕອນ ${liveTourStep} / 5`, th: `ขั้นตอน ${liveTourStep} / 5`, en: `Step ${liveTourStep} of 5` }) }}
          </span>
          <div class="d-flex align-center gap-2">
            <v-btn
              size="small"
              variant="outlined"
              :disabled="liveTourStep === 1"
              @click="prevTourStep"
            >
              {{ l({ lo: 'ຍ້ອນກັບ', th: 'ย้อนกลับ', en: 'Back' }) }}
            </v-btn>

            <v-btn
              v-if="liveTourStep < 5"
              size="small"
              color="primary"
              variant="flat"
              append-icon="mdi-arrow-right"
              class="font-weight-bold px-3"
              @click="nextTourStep"
            >
              {{ l({ lo: 'ຕໍ່ໄປ', th: 'ถัดไป', en: 'Next' }) }}
            </v-btn>

            <v-btn
              v-else
              size="small"
              color="success"
              variant="flat"
              prepend-icon="mdi-check"
              class="font-weight-bold px-3"
              @click="endLiveTour"
            >
              {{ l({ lo: 'ສຳເລັດ', th: 'เสร็จสิ้น', en: 'Finish' }) }}
            </v-btn>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.page-head,.filter-bar,.performance-section{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
.page-head{padding:24px;display:flex;justify-content:space-between;align-items:flex-start;gap:24px;margin-bottom:16px}.page-head p,.card-heading p,.section-head p{margin:0 0 6px;color:var(--color-primary);font-size: 12px;font-weight:800;letter-spacing:.13em}.page-head h1{font-size:clamp(24px,3vw,34px);line-height:1.15;margin:0 0 8px}.page-head span{color:var(--color-text-secondary)}.head-actions{display:flex;gap:8px;flex-wrap:wrap}
.filter-bar{display:grid;grid-template-columns:minmax(240px,1.5fr) minmax(170px,.8fr) minmax(145px,.7fr) minmax(145px,.7fr) auto;align-items:center;gap:12px;padding:16px;margin-bottom:16px}
.loading-grid,.summary-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-bottom:16px}.summary-card{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);padding:18px;display:grid;gap:5px}.summary-card>span{color:var(--color-text-secondary);font-size:13px}.summary-card strong{font-size:clamp(22px,3vw,30px);line-height:1.1}.summary-card small{color:var(--color-text-muted);font-size: 12px}.summary-card.finance{border-color:rgba(32,164,100,.28);background:linear-gradient(145deg,var(--color-surface),rgba(32,164,100,.045))}.summary-card.finance strong{color:#167a4b}.summary-card.loss{border-color:rgba(239,68,68,.3);background:linear-gradient(145deg,var(--color-surface),rgba(239,68,68,.04))}.summary-card.loss strong{color:var(--color-error)}
.analysis-grid{display:grid;grid-template-columns:minmax(0,1.6fr) minmax(320px,.8fr);gap:16px;margin-bottom:16px}.trend-card,.decision-card{padding:20px;border-radius:var(--radius-lg)}.card-heading,.section-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px}.card-heading h2,.section-head h2{margin:0;font-size:20px}.card-heading>small{color:var(--color-text-muted)}
.trend-chart{height:220px;display:flex;align-items:flex-end;gap:clamp(4px,1vw,12px);padding:28px 4px 24px;margin-top:12px;border-bottom:1px solid var(--color-border);position:relative}.trend-column{height:100%;flex:1;min-width:8px;display:flex;flex-direction:column;justify-content:flex-end;align-items:center;gap:7px;position:relative;outline:none}.trend-bar{width:min(34px,80%);min-height:4px;background:linear-gradient(180deg,var(--color-primary),#7c6cf1);border-radius:7px 7px 3px 3px;transition:transform .15s}.trend-column:is(:hover,:focus) .trend-bar{transform:scaleX(1.08)}.trend-column small{font-size: 12px;color:var(--color-text-muted);white-space:nowrap}.trend-tooltip{display:none;position:absolute;bottom:calc(100% + 6px);z-index:2;background:#211d3f;color:#fff;padding:8px 10px;border-radius:8px;font-size: 12px;white-space:nowrap;text-align:center}.trend-column:is(:hover,:focus) .trend-tooltip{display:block}
.decision-summary{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:18px 0}.decision-count{padding:14px 8px;border-radius:12px;display:grid;text-align:center}.decision-count strong{font-size:25px}.decision-count span{font-size: 12px}.decision-count.keep{background:rgba(39,174,96,.1);color:var(--color-success)}.decision-count.review{background:rgba(245,158,11,.12);color:#a86700}.decision-count.pause{background:rgba(239,68,68,.1);color:var(--color-error)}.guide-note{display:flex;gap:8px;align-items:flex-start;color:var(--color-text-secondary);font-size:13px;line-height:1.55;margin:0}.decision-steps{display:grid;gap:9px;margin:15px 0 0;padding:0;list-style:none}.decision-steps li{display:grid;padding:10px 12px;border:1px solid var(--color-border);border-radius:10px}.decision-steps strong{font-size: 12px}.decision-steps span{color:var(--color-text-secondary);font-size: 12px;line-height:1.45}
.performance-section{padding:20px}.section-head{margin-bottom:14px}.report-meta{display:flex;align-items:center;gap:12px}.report-meta>span{font-size: 12px;color:var(--color-text-muted)}.report-meta .v-text-field{width:250px}.detail-toolbar{display:flex;align-items:flex-end;justify-content:space-between;gap:16px;padding:14px;margin-bottom:8px;border:1px solid var(--color-border);border-radius:14px;background:var(--color-surface-soft)}.toolbar-group{display:grid;gap:6px;min-width:0}.toolbar-group>small{font-size: 12px;font-weight:750;color:var(--color-text-muted);letter-spacing:.04em}.level-tabs,.status-tabs{max-width:100%}.level-tabs .v-btn,.status-tabs .v-btn{min-height:40px}.level-tabs .v-icon{margin-right:5px}.status-tabs .v-btn{gap:6px}.status-tabs b{min-width:20px;padding:1px 6px;border-radius:999px;background:color-mix(in srgb,currentColor 12%,transparent);font-size: 12px}.status-dot{width:8px;height:8px;border-radius:50%;display:inline-block;flex:0 0 8px}.status-dot.active{background:#20a464}.status-dot.paused{background:#8991a5}.status-dot.issues{background:#e58a18}.filter-help{display:flex;align-items:flex-start;gap:7px;margin:0 2px 16px;color:var(--color-text-muted);font-size: 12px;line-height:1.45}.table-loading{margin:-1px 0 16px}.performance-list{display:grid;border:1px solid var(--color-border);border-radius:14px;overflow:hidden}.performance-row+.performance-row{border-top:1px solid var(--color-border)}.performance-row.expanded{background:var(--color-surface-soft)}.row-main{width:100%;display:grid;grid-template-columns:minmax(210px,1.4fr) minmax(95px,auto) repeat(5,minmax(72px,.5fr)) minmax(145px,auto) minmax(95px,auto);align-items:center;gap:10px;padding:14px 18px;text-align:left;color:inherit;background:transparent;border:0;cursor:pointer}.row-main:hover{background:var(--color-background)}.entity-cell{display:flex;align-items:center;gap:11px;min-width:0}.entity-cell>span:last-child{display:grid;min-width:0}.entity-cell strong,.entity-cell small,.entity-cell code{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.entity-cell small{color:var(--color-text-secondary);font-size: 12px}.entity-cell code{color:var(--color-text-muted);font-size: 12px}.entity-icon{width:42px;height:42px;display:grid;place-items:center;border-radius:11px;background:var(--color-primary-soft);color:var(--color-primary);flex:0 0 42px}.delivery-chip{display:inline-flex;align-items:center;gap:6px;width:max-content;max-width:100%;padding:6px 8px;border-radius:999px;font-size: 12px;font-weight:750;white-space:nowrap}.delivery-chip.active{color:#167a4b;background:rgba(32,164,100,.1)}.delivery-chip.paused{color:#697184;background:rgba(137,145,165,.13)}.delivery-chip.issues{color:#a96000;background:rgba(229,138,24,.12)}.metric{display:grid;gap:3px}.metric small{font-size: 12px;color:var(--color-text-muted)}.metric strong{font-size:14px}.decision-chip{display:flex;align-items:center;gap:8px;border-radius:11px;padding:9px 10px;min-width:0}.decision-chip>span{display:grid;min-width:0}.decision-chip strong{font-size:12px}.decision-chip small{font-size: 12px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.decision-chip.keep{background:rgba(39,174,96,.1);color:var(--color-success)}.decision-chip.review{background:rgba(245,158,11,.12);color:#a86700}.decision-chip.pause{background:rgba(239,68,68,.1);color:var(--color-error)}.decision-chip.no_data{background:var(--color-surface-soft);color:var(--color-text-secondary)}
.row-actions{display:inline-flex;align-items:center;justify-content:flex-end;gap:8px;white-space:nowrap}
.mobile-scale-bar{display:none}
.row-detail{padding:20px;border-top:1px solid var(--color-border);display:grid;gap:18px}.explanation{display:flex;gap:12px;align-items:flex-start;max-width:850px}.explanation h3{margin:0 0 4px;font-size:17px}.explanation p{margin:0;color:var(--color-text-secondary)}.explanation ul{margin:8px 0 0;padding-left:20px;color:var(--color-text-secondary)}.audience-callout{display:flex;align-items:flex-start;gap:11px;padding:14px;color:var(--color-primary);background:var(--color-primary-soft);border-radius:12px}.audience-callout div{display:grid}.audience-callout strong{font-size:13px}.audience-callout span{color:var(--color-text-secondary);font-size: 12px}.audience-panel{display:grid;gap:14px;padding:18px;border:1px solid color-mix(in srgb,var(--color-primary) 22%,var(--color-border));border-radius:14px;background:var(--color-surface)}.audience-heading{display:flex;align-items:flex-start;justify-content:space-between;gap:12px}.audience-heading p{margin:0 0 3px;color:var(--color-primary);font-size: 12px;font-weight:800;letter-spacing:.12em}.audience-heading h3{margin:0;font-size:17px}.audience-heading>small{padding:5px 8px;color:var(--color-primary);background:var(--color-primary-soft);border-radius:999px;font-size: 12px}.audience-metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:9px}.audience-metrics>div{display:grid;gap:3px;padding:12px;background:var(--color-background);border-radius:11px}.audience-metrics span,.audience-metrics small{color:var(--color-text-muted);font-size: 12px}.audience-metrics strong{font-size:19px}.coverage-metric strong{color:var(--color-primary)}.coverage-bar{margin-top:-5px}.audience-advice{display:flex;align-items:flex-start;gap:10px;padding:12px;border-radius:11px}.audience-advice>div{display:grid}.audience-advice strong{font-size:12px}.audience-advice span{font-size: 12px;line-height:1.5}.audience-advice.success{color:#167a4b;background:rgba(32,164,100,.1)}.audience-advice.warning{color:#995c00;background:rgba(229,138,24,.12)}.audience-advice.info{color:var(--color-primary);background:var(--color-primary-soft)}.targeting-block{display:grid;gap:8px}.targeting-block>strong{font-size:12px}.targeting-block>small{color:var(--color-text-muted)}.targeting-tags{display:flex;flex-wrap:wrap;gap:6px}.targeting-tags span{display:inline-flex;align-items:center;gap:5px;padding:6px 8px;border:1px solid var(--color-border);border-radius:999px;color:var(--color-text-secondary);font-size: 12px}.native-metric-note{display:flex;align-items:flex-start;gap:8px;padding-top:11px;border-top:1px solid var(--color-border);color:var(--color-text-muted);font-size: 12px;line-height:1.5}.metric-detail-grid{display:grid;grid-template-columns:repeat(6,1fr);gap:10px}.metric-detail-grid>div{padding:12px;border:1px solid var(--color-border);border-radius:11px;display:grid;gap:3px}.metric-detail-grid span,.metric-detail-grid small{font-size: 12px;color:var(--color-text-muted)}.metric-detail-grid strong{font-size:15px;overflow-wrap:anywhere}.status-actions{display:flex;align-items:center;justify-content:flex-end;gap:8px;padding-top:14px;border-top:1px solid var(--color-border)}.status-actions>div{display:grid;margin-right:auto}.status-actions>div span{font-size:12px;color:var(--color-text-secondary)}
.empty-state,.empty-inline{text-align:center;padding:48px 20px;color:var(--color-text-secondary)}.empty-inline.compact{padding:24px 16px}.empty-state h2,.empty-state h3,.empty-inline h3{color:var(--color-text);margin:12px 0 6px}.empty-state p,.empty-inline p{margin:0 0 16px}.dialog-title{display:flex;justify-content:space-between;align-items:flex-start;padding:22px}.dialog-title small{color:var(--color-primary);font-size: 12px;letter-spacing:.12em}.dialog-title h2{margin:3px 0 0;font-size:22px}.settings-body{padding:0 22px 12px}.settings-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-top:16px}.settings-grid.primary-settings{grid-template-columns:repeat(2,minmax(0,1fr));margin-top:0}.dialog-actions{padding:12px 22px 22px}.action-plan-list{display:grid;gap:8px;max-height:420px;overflow:auto}.action-plan-item{display:grid;grid-template-columns:auto 1fr;align-items:flex-start;padding:10px 12px;border:1px solid var(--color-border);border-radius:12px;cursor:pointer}.action-plan-item:hover{background:var(--color-surface-soft)}.action-plan-item>span{display:grid;gap:3px;padding-top:8px;min-width:0}.action-plan-item small{color:var(--color-text-muted)}.action-plan-item em{color:var(--color-error);font-size: 12px;font-style:normal;line-height:1.4}
.filter-account-select { min-width: 220px; }
.filter-product-select { min-width: 200px; }
.filter-date-select { min-width: 140px; }
.filter-custom-date { max-width: 130px; }

.quick-scale-btn{height:26px !important;padding-inline:8px !important;font-size: 12px !important;letter-spacing:0}
.btn-scale-budget{font-size:13px}

.target-entity-card{background:var(--color-surface-soft);border:1px solid var(--color-border);border-radius:12px;padding:12px 14px}
.entity-type-badge{font-size: 12px;font-weight:800;letter-spacing:.04em;color:var(--color-primary);background:var(--color-primary-soft);padding:3px 7px;border-radius:6px}
.entity-name-display{font-size:16px;font-weight:700;margin:6px 0 2px;line-height:1.3}
.ad-parent-note{font-size: 12px;color:var(--color-primary);margin:4px 0 0;display:flex;align-items:center;gap:4px}
.scale-pills-row{display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.pill-group-label{font-size: 12px;font-weight:800;min-width:44px;display:inline-flex;align-items:center;gap:2px}
.scale-pill-btn{min-width:52px;height:32px;font-weight:700;border-radius:8px;font-size:12px}
.budget-comparison-box{display:flex;align-items:center;justify-content:space-between;padding:12px 16px;border-radius:12px;background:color-mix(in srgb,var(--color-primary) 6%,var(--color-surface));border:1px dashed color-mix(in srgb,var(--color-primary) 30%,var(--color-border))}
.comp-col{display:grid;gap:2px}
.comp-label{font-size: 12px;color:var(--color-text-secondary);font-weight:600}
.comp-value{font-size:17px;font-weight:800}
.comp-value small{font-size: 12px;font-weight:500;color:var(--color-text-muted);margin-left:2px}
.comp-arrow{display:flex;align-items:center;justify-content:center}

@media(max-width:1280px){
  .filter-bar{grid-template-columns:repeat(2,minmax(0,1fr))}
  .analysis-grid{grid-template-columns:1fr}
  .detail-toolbar{align-items:stretch;flex-direction:column}
  .row-main{grid-template-columns:minmax(180px,1.3fr) minmax(95px,auto) repeat(3,minmax(70px,.55fr)) minmax(135px,auto) minmax(90px,auto);gap:8px;padding:12px 14px}
  .metric-cost,.metric-frequency{display:none}
  .audience-metrics{grid-template-columns:repeat(2,minmax(0,1fr))}
  .metric-detail-grid{grid-template-columns:repeat(3,1fr)}
  .mobile-scale-bar{display:none}
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
    padding-right:38px;
    display:flex;
    align-items:flex-start;
    gap:10px;
    width:100%;
  }
  .row-actions{
    position:absolute;
    top:14px;
    right:12px;
    display:flex;
    align-items:center;
  }
  .row-actions .quick-scale-btn{
    display:none;
  }
  .row-actions .expand-chevron{
    font-size:22px;
    color:var(--color-text-secondary);
  }
  .delivery-chip{
    grid-column:1;
    grid-row:2;
    justify-self:start;
    align-self:center;
    font-size: 12px;
  }
  .decision-chip{
    grid-column:2;
    grid-row:2;
    justify-self:end;
    align-self:center;
    padding:6px 8px;
    max-width:100%;
  }
  .decision-chip strong{font-size: 12px}
  .decision-chip small{font-size: 12px}

  .row-main>.metric{display:none}
  .row-main>.mobile-key{
    display:grid;
    min-width:0;
    padding:8px 10px;
    border-radius:9px;
    background:var(--color-background);
  }
  .row-main>.mobile-key small{font-size: 12px}
  .row-main>.mobile-key strong{font-size:13.5px}
  .metric-spend{grid-column:1;grid-row:3}
  .metric-results{grid-column:2;grid-row:3}
  .metric-cost{grid-column:1;grid-row:4}
  .metric-ctr{grid-column:2;grid-row:4}

  .mobile-scale-bar{
    grid-column:1 / -1;
    grid-row:5;
    display:block;
    width:100%;
    margin-top:4px;
  }
  .mobile-scale-bar .v-btn{
    width:100%;
    min-height:36px;
    font-size:12px;
  }

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
  .scale-pill-btn{min-width:44px;height:30px;font-size: 12px;padding-inline:6px}
  .budget-comparison-box{padding:10px 12px}
  .comp-value{font-size:15px}

  .settings-grid,.settings-grid.primary-settings{grid-template-columns:1fr}
  .trend-card,.decision-card{padding:14px}
  .trend-chart{overflow-x:auto;justify-content:flex-start}
  .trend-column{flex:0 0 28px}
  .card-heading{display:grid}
}

@media(max-width:420px){
  .level-tabs .v-btn{font-size: 12px}
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

/* --- Auto-Pilot & Dual-System Styles --- */
.autopilot-head-btn {
  position: relative;
  letter-spacing: 0;
}
.autopilot-dot-pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #22c55e;
  box-shadow: 0 0 0 rgba(34, 197, 94, 0.4);
  animation: pulse-dot 1.8s infinite;
  margin-left: 6px;
  display: inline-block;
}
@keyframes pulse-dot {
  0% { box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.7); }
  70% { box-shadow: 0 0 0 6px rgba(34, 197, 94, 0); }
  100% { box-shadow: 0 0 0 0 rgba(34, 197, 94, 0); }
}

.system-mode-banner {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 14px 18px;
  margin-bottom: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  transition: border-color 0.2s, background 0.2s;
}
.system-mode-banner.autopilot-active {
  border-color: rgba(34, 197, 94, 0.38);
  background: linear-gradient(135deg, var(--color-surface), rgba(34, 197, 94, 0.05));
}
.mode-badge-wrap {
  display: flex;
  align-items: center;
  gap: 14px;
  flex-wrap: wrap;
  flex: 1;
  min-width: 280px;
}
.mode-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.02em;
  white-space: nowrap;
}
.mode-pill.autopilot {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.35);
}
.mode-pill.manual {
  background: rgba(99, 102, 241, 0.12);
  color: #6366f1;
  border: 1px solid rgba(99, 102, 241, 0.25);
}
.mode-desc {
  display: grid;
  gap: 3px;
  min-width: 0;
}
.mode-title-text {
  font-size: 13px;
  line-height: 1.4;
}
.mode-rules-summary {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 3px;
}
.rule-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
}
.rule-chip-muted {
  font-size: 12px;
  color: var(--color-text-muted);
}
.mode-controls {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.mode-switch-box {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 3px 10px;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 999px;
}
.mode-switch-label {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-secondary);
}

.ad-scale-status-pill {
  display: inline-flex;
  align-items: center;
  padding: 2px 7px;
  border-radius: 5px;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.02em;
  width: max-content;
}
.ad-scale-status-pill.scale_ready {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
  border: 1px solid rgba(34, 197, 94, 0.3);
}
.ad-scale-status-pill.stop {
  background: rgba(239, 68, 68, 0.15);
  color: #dc2626;
  border: 1px solid rgba(239, 68, 68, 0.3);
}
.ad-scale-status-pill.cooldown {
  background: rgba(245, 158, 11, 0.15);
  color: #d97706;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.master-switch-card.is-active {
  border-color: rgba(34, 197, 94, 0.35);
  background: rgba(34, 197, 94, 0.05);
}
.rule-card {
  border-color: var(--color-border);
  background: var(--color-surface);
}
.rule-badge {
  display: inline-block;
  padding: 3px 8px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 800;
}
.rule-badge.scale {
  background: rgba(34, 197, 94, 0.15);
  color: #16a34a;
}
.rule-badge.stop {
  background: rgba(239, 68, 68, 0.15);
  color: #dc2626;
}
.rule-inputs-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 12px;
}
.autopilot-logs-list {
  display: grid;
  gap: 8px;
}
.autopilot-log-entry {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 11px 14px;
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 10px;
}
.autopilot-log-entry.scale {
  border-color: rgba(34, 197, 94, 0.35);
  background: rgba(34, 197, 94, 0.03);
}
.autopilot-log-entry.pause {
  border-color: rgba(239, 68, 68, 0.35);
  background: rgba(239, 68, 68, 0.03);
}
.log-content-col {
  flex: 1;
  min-width: 0;
  display: grid;
  gap: 2px;
}
.log-entity-name {
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.log-time {
  font-size: 12px;
  color: var(--color-text-muted);
}
.log-detail {
  font-size: 12px;
  color: var(--color-text-secondary);
}
.log-values {
  font-size: 12px;
  display: flex;
  align-items: center;
  margin-top: 2px;
}

@media (max-width: 768px) {
  .system-mode-banner {
    flex-direction: column;
    align-items: stretch;
  }
  .mode-controls {
    justify-content: space-between;
  }
  .mode-switch-box {
    flex: 1;
    justify-content: space-between;
  }
}

/* Interactive Guide Walkthrough Styles */
.guide-dialog-card {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
}
.guide-stepper-bar {
  background: rgba(0, 0, 0, 0.02);
}
.guide-step-chip {
  border-radius: 20px !important;
  text-transform: none !important;
  font-size: 12px !important;
}
.guide-step-chip .step-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.12);
  font-weight: 800;
  font-size: 12px;
}
.guide-step-header .step-badge-pill {
  display: inline-flex;
  align-items: center;
  font-size: 12px;
  letter-spacing: 0.03em;
}
.guide-mockup-wrapper {
  background: rgba(0, 0, 0, 0.02);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 14px;
  padding: 16px;
}
.guide-mockup-card {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #cbd5e1);
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.05);
  overflow: hidden;
}
.mockup-header-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  background: rgba(0, 0, 0, 0.03);
  border-bottom: 1px solid var(--color-border, #e2e8f0);
}
.mockup-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.mockup-dot.red { background: #ef4444; }
.mockup-dot.yellow { background: #f59e0b; }
.mockup-dot.green { background: #10b981; }
.mockup-title {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-secondary, #64748b);
  letter-spacing: 0.04em;
}
.mockup-element-row {
  display: grid;
  grid-template-columns: minmax(280px, 1.15fr) minmax(260px, 1fr);
  gap: 24px;
  align-items: center;
  position: relative;
}
@media (max-width: 860px) {
  .mockup-element-row {
    grid-template-columns: 1fr;
    gap: 12px;
  }
}
.mockup-element-box {
  background: var(--color-background, #f8fafc);
  border: 1.5px solid var(--color-border, #cbd5e1);
  border-radius: 10px;
  padding: 12px 14px;
  transition: all 0.2s ease;
}
.mockup-element-box.kpi-card-mockup {
  border-color: var(--color-border);
}
.mockup-element-box.kpi-card-mockup.info {
  border-color: rgba(2, 132, 199, 0.4);
  background: rgba(2, 132, 199, 0.05);
}
.mockup-element-box.kpi-card-mockup.winner {
  border-color: rgba(16, 185, 129, 0.4);
  background: rgba(16, 185, 129, 0.05);
}
.mockup-element-box.kpi-card-mockup.loser {
  border-color: rgba(239, 68, 68, 0.4);
  background: rgba(239, 68, 68, 0.05);
}
.mockup-element-box.kpi-card-mockup.warning {
  border-color: rgba(245, 158, 11, 0.4);
  background: rgba(245, 158, 11, 0.05);
}
.mock-input-label {
  font-size: 12px;
  color: var(--color-text-secondary, #64748b);
  margin-bottom: 5px;
  font-weight: 700;
}
.mock-input-field {
  display: flex;
  align-items: center;
  font-size: 13.5px;
  color: var(--color-text, #1e293b);
}
.mock-input-field.active-highlight {
  color: var(--color-primary, #0284c7);
}
.guide-pointer-anchor {
  position: absolute;
  right: -20px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  pointer-events: none;
  z-index: 2;
}
@media (max-width: 860px) {
  .guide-pointer-anchor {
    display: none;
  }
}
.pointer-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #0284c7;
  border: 2.5px solid #ffffff;
  box-shadow: 0 0 0 2px #0284c7;
  display: inline-block;
  flex-shrink: 0;
  animation: guide-dot-pulse 2s infinite ease-in-out;
}
.pointer-dot.green {
  background: #10b981;
  box-shadow: 0 0 0 2px #10b981;
}
.pointer-dot.red {
  background: #ef4444;
  box-shadow: 0 0 0 2px #ef4444;
}
.pointer-dot.orange {
  background: #f59e0b;
  box-shadow: 0 0 0 2px #f59e0b;
}
.pointer-dot.cyan {
  background: #06b6d4;
  box-shadow: 0 0 0 2px #06b6d4;
}
.pointer-line {
  height: 2px;
  width: 28px;
  background: #0284c7;
  display: inline-block;
}
@keyframes guide-dot-pulse {
  0% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.25); opacity: 0.85; }
  100% { transform: scale(1); opacity: 1; }
}
.guide-callout-box {
  background: rgba(2, 132, 199, 0.05);
  border: 1px dashed rgba(2, 132, 199, 0.4);
  border-radius: 10px;
  padding: 12px 14px;
}
.guide-callout-box.highlight-tip {
  background: rgba(245, 158, 11, 0.07);
  border: 1px dashed rgba(245, 158, 11, 0.5);
}
.callout-title {
  font-size: 13px;
  margin-bottom: 4px;
  display: flex;
  align-items: center;
}
.callout-desc {
  font-size: 12px;
  line-height: 1.45;
  color: var(--color-text-secondary, #475569);
}
.ad-row-mockup {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
}
.ad-row-mockup.cooldown {
  background: rgba(2, 132, 199, 0.04);
  border-color: rgba(2, 132, 199, 0.3);
}
.ad-row-mockup.info-state {
  background: rgba(100, 116, 139, 0.03);
}

/* Live On-Page Tour Styles */
.live-tour-root {
  position: relative;
  z-index: 99990;
}
.live-tour-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.45);
  backdrop-filter: blur(2px);
  z-index: 99991;
}
.tour-spotlight-active {
  position: relative !important;
  z-index: 99995 !important;
  box-shadow: 0 0 0 4px #0284c7, 0 0 35px rgba(2, 132, 199, 0.5) !important;
  border-radius: 14px !important;
  background: var(--color-surface, #ffffff) !important;
  transition: all 0.3s ease !important;
}
.live-tour-popover-card {
  position: fixed;
  bottom: 24px;
  right: 24px;
  width: 440px;
  max-width: calc(100vw - 32px);
  background: var(--color-surface, #ffffff);
  border: 1.5px solid var(--color-primary, #0284c7);
  border-radius: 16px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
  z-index: 99999;
  overflow: hidden;
  animation: slide-up-fade 0.25s ease-out;
}
.tour-badge-text {
  font-size: 12px;
  font-weight: 800;
  color: var(--color-primary, #0284c7);
  letter-spacing: 0.04em;
}
.tour-title-text {
  font-size: 13.5px;
  line-height: 1.3;
}
.bounce-up {
  animation: subtle-float 2.4s infinite cubic-bezier(0.16, 1, 0.3, 1);
}
@keyframes subtle-float {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-3px);
  }
}
@keyframes slide-up-fade {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

/* Outcome Filter Strip */
.outcome-filter-strip {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  margin-bottom: 12px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
}
.outcome-strip-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-wrap: wrap;
}
.outcome-strip-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-secondary);
}
.angel-found-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  line-height: 1.4;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
  color: #b45309;
  background: #fef3c7;
  border: 1px solid #fde68a;
  box-shadow: 0 1px 3px rgba(245, 158, 11, 0.15);
}
.outcome-tabs {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  padding-bottom: 2px;
}
.outcome-tabs::-webkit-scrollbar {
  display: none;
}
.outcome-tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 10px;
  font-size: 12px;
  font-weight: 600;
  background: var(--color-surface-soft);
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s ease;
}
.outcome-tab-btn:hover {
  background: color-mix(in srgb, var(--color-primary) 8%, var(--color-surface));
  color: var(--color-text);
}
.outcome-tab-btn.active {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.12);
}
.outcome-tab-btn.angel-tab.active {
  background: linear-gradient(135deg, #f59e0b, #d97706) !important;
  border-color: #d97706 !important;
  color: #fff !important;
}
.count-pill {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 800;
  background: rgba(0, 0, 0, 0.08);
}
.outcome-tab-btn.active .count-pill {
  background: rgba(255, 255, 255, 0.25);
  color: #fff;
}
.count-pill.gold {
  background: #fef3c7;
  color: #92400e;
}
.count-pill.green {
  background: #d1fae5;
  color: #065f46;
}
.count-pill.blue {
  background: #e0f2fe;
  color: #075985;
}
.count-pill.purple {
  background: #f3e8ff;
  color: #6b21a8;
}
.count-pill.red {
  background: #fee2e2;
  color: #991b1b;
}

/* Angel Ad Row & Callout Styling */
.performance-row.is-angel-ad {
  background: linear-gradient(90deg, rgba(245, 158, 11, 0.08) 0%, transparent 35%);
  box-shadow: inset 2px 0 0 rgba(245, 158, 11, 0.8);
}
.entity-icon.angel-avatar {
  background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%) !important;
  color: #b45309 !important;
  box-shadow: 0 0 10px rgba(245, 158, 11, 0.35);
}
.angel-badge {
  letter-spacing: 0;
  font-size: 12px !important;
  background: linear-gradient(135deg, #f59e0b, #d97706) !important;
  box-shadow: 0 1px 4px rgba(217, 119, 6, 0.35);
}
.angel-callout {
  padding: 12px 16px;
  border-radius: 12px;
  background: linear-gradient(135deg, #fffbeb 0%, #fef3c7 100%);
  border: 1.5px solid #fde68a;
  margin-bottom: 12px;
}
.angel-callout-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: #fff;
  box-shadow: 0 2px 6px rgba(245, 158, 11, 0.2);
  flex-shrink: 0;
}
</style>
