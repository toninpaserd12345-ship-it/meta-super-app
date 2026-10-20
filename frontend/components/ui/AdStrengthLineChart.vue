<script setup lang="ts">
import type { MetaInsightItem, MetaInsightPoint, MetaInsightHourlyPoint, Product, AutomationFlow } from '~/types/automation'

const props = withDefaults(defineProps<{
  items: MetaInsightItem[]
  accountTrend?: MetaInsightPoint[]
  currency?: string
  updatingId?: string
  products?: Product[]
  flows?: AutomationFlow[]
  isToday?: boolean
  since?: string
  until?: string
}>(), {
  currency: 'USD',
  accountTrend: () => [],
  updatingId: '',
  products: () => [],
  flows: () => [],
  isToday: false,
  since: '',
  until: '',
})

const emit = defineEmits<{
  (e: 'scale', item: MetaInsightItem): void
  (e: 'pause', item: MetaInsightItem): void
  (e: 'change-status', payload: { item: MetaInsightItem; status: 'ACTIVE' | 'PAUSED' }): void
}>()

const { l } = useLocale()

type MetricKey = 'strength' | 'purchases' | 'cpa' | 'messages' | 'ctr' | 'spend'
type DisplayMode = 'separate' | 'combined'

const selectedMetric = ref<MetricKey>('strength')
const displayMode = ref<DisplayMode>('separate') // Default: Clean separated ad-level cards
const focusedAdId = ref<string>('all')

const hoveredPoint = ref<{
  adName: string
  adId: string
  color: string
  date: string
  metricValue: number | null
  displayValue: string
  strengthScore: number
  strengthLabel: string
  purchases: number
  cpa: number | null
  ctr: number
  spend: number
  x: number
  y: number
} | null>(null)

// Strict Semantic Color Palette:
// GREEN = Winner / Angel
// RED = Fatigued / Bad / Loser
// BLUE / SLATE = Neutral / Moderate
function getSemanticColor(isAngel: boolean, isBad: boolean, index: number): string {
  if (isAngel) {
    const angelPalette = ['#10b981', '#059669', '#047857', '#14b8a6']
    return angelPalette[index % angelPalette.length] ?? '#10b981'
  }
  if (isBad) {
    const badPalette = ['#ef4444', '#dc2626', '#b91c1c', '#e11d48']
    return badPalette[index % badPalette.length] ?? '#ef4444'
  }
  const neutralPalette = ['#3b82f6', '#2563eb', '#64748b', '#475569']
  return neutralPalette[index % neutralPalette.length] ?? '#3b82f6'
}

// Extract dates from accountTrend or synthesize last 7 days
const dates = computed(() => {
  if (props.accountTrend?.length) {
    return props.accountTrend.map(p => p.date)
  }
  const res: string[] = []
  const today = new Date()
  for (let i = 6; i >= 0; i--) {
    const d = new Date(today.getFullYear(), today.getMonth(), today.getDate() - i)
    res.push(d.toISOString().slice(0, 10))
  }
  return res
})

function getPurchasesCount(item?: MetaInsightItem | null): number {
  if (!item) return 0
  if (item.purchasesCount != null && item.purchasesCount > 0) return item.purchasesCount
  if (item.results?.type?.toLowerCase().includes('purchase') && item.results?.value != null && item.results.value > 0) return item.results.value
  return 0
}

function getCostPerPurchase(item?: MetaInsightItem | null): number | null {
  if (!item) return null
  if (item.costPerPurchase != null && item.costPerPurchase > 0) return item.costPerPurchase
  if (item.results?.type?.toLowerCase().includes('purchase') && item.results?.cost != null && item.results.cost > 0) return item.results.cost
  const count = getPurchasesCount(item)
  if (count && count > 0 && item.spend && item.spend > 0) return item.spend / count
  return null
}

function getMessagesCount(item?: MetaInsightItem | null): number {
  if (!item) return 0
  if (item.messagesCount != null && item.messagesCount > 0) return item.messagesCount
  const type = (item.results?.type || '').toLowerCase()
  if ((type.includes('conversation') || type.includes('message') || type.includes('reply')) && item.results?.value != null && item.results.value > 0) {
    return item.results.value
  }
  return 0
}

// Compute daily strength score (0 to 100)
function computeStrengthScore(purchases: number, cpa: number | null, ctr: number, spend: number, freq: number): number {
  let score = 0

  // 1. Purchases component (up to 40 pts)
  if (purchases >= 3) score += 40
  else if (purchases === 2) score += 30
  else if (purchases === 1) score += 20
  else if (spend > 0) score += 5

  // 2. CPA efficiency component (up to 30 pts)
  if (purchases > 0 && cpa != null) {
    if (cpa <= 1.0) score += 30
    else if (cpa <= 2.5) score += 20
    else if (cpa <= 5.0) score += 10
    else score += 5
  } else if (purchases === 0 && spend > 5) {
    score += 0
  } else {
    score += 15
  }

  // 3. CTR Creative engagement component (up to 20 pts)
  if (ctr >= 2.0) score += 20
  else if (ctr >= 1.5) score += 15
  else if (ctr >= 1.0) score += 10
  else if (ctr >= 0.6) score += 5
  else score += 0

  // 4. Frequency health component (up to 10 pts)
  if (freq > 0 && freq <= 1.6) score += 10
  else if (freq <= 2.2) score += 6
  else score += 2

  return Math.min(100, Math.max(0, Math.round(score)))
}

function getStrengthLabel(score: number): { label: string; color: string; icon: string } {
  if (score >= 80) return { label: l({ lo: '🔥 ແຮງຫຼາຍ (Angel)', th: '🔥 แรงมาก (Angel)', en: '🔥 Super Strong' }), color: '#10b981', icon: 'mdi-star-face' }
  if (score >= 65) return { label: l({ lo: 'ແຂງແກ່ນ (Strong)', th: 'แข็งแกร่ง (Strong)', en: 'Strong' }), color: '#059669', icon: 'mdi-shield-check' }
  if (score >= 45) return { label: l({ lo: '🟡 ປານກາງ', th: '🟡 ปานกลาง', en: '🟡 Moderate' }), color: '#3b82f6', icon: 'mdi-minus-circle' }
  if (score >= 30) return { label: l({ lo: '⚠️ ເລີ່ມຕົກ', th: '⚠️ เริ่มตก', en: '⚠️ Weakening' }), color: '#ef4444', icon: 'mdi-trending-down' }
  return { label: l({ lo: '🛑 ໝົດກຳລັງ (Fatigued)', th: '🛑 หมดกำลัง (Fatigued)', en: '🛑 Fatigued' }), color: '#dc2626', icon: 'mdi-alert-octagon' }
}

function isItemActive(item: MetaInsightItem): boolean {
  const st = (item.effectiveStatus || item.status || '').toUpperCase()
  return st === 'ACTIVE' || (!st.includes('PAUSED') && !st.includes('ARCHIVED') && !st.includes('DELETED'))
}

function handleToggleAd(item: MetaInsightItem, newActive: boolean | null | undefined) {
  emit('change-status', {
    item,
    status: newActive ? 'ACTIVE' : 'PAUSED',
  })
}

const zeroDecimalCurrencies = new Set(['JPY', 'KRW', 'VND', 'LAK', 'IDR', 'CLP', 'PYG', 'UGX'])

function money(amount?: number | null): string {
  if (amount == null) return '—'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: props.currency || 'USD',
    maximumFractionDigits: zeroDecimalCurrencies.has((props.currency || '').toUpperCase()) ? 0 : 2,
  }).format(amount)
}

const manualProductOverrides = ref<Record<string, string>>({})
function setProductOverride(adId: string, productId: string) {
  manualProductOverrides.value = {
    ...manualProductOverrides.value,
    [adId]: productId,
  }
}

const triggerToProduct = computed(() => {
  const map: Record<string, { product: Product; flowName: string }> = {}
  const productMap = new Map<string, Product>((props.products || []).map(p => [p.id, p]))

  for (const flow of (props.flows || [])) {
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

function getLinkedProduct(item: MetaInsightItem): Product | null {
  const overrideId = manualProductOverrides.value[item.entityId]
  if (overrideId) {
    const found = (props.products || []).find(p => p.id === overrideId)
    if (found) return found
  }

  if (triggerToProduct.value[item.entityId]) return triggerToProduct.value[item.entityId]?.product ?? null
  if (item.adSetId && triggerToProduct.value[item.adSetId]) return triggerToProduct.value[item.adSetId]?.product ?? null
  if (item.campaignId && triggerToProduct.value[item.campaignId]) return triggerToProduct.value[item.campaignId]?.product ?? null

  const searchName = (item.entityName || '').toLowerCase()
  for (const prod of (props.products || [])) {
    const pName = (prod.name || '').toLowerCase()
    const pCode = (prod.code || '').toLowerCase()
    if (pName && searchName.includes(pName)) return prod
    if (pCode && searchName.includes(pCode)) return prod
  }

  if (props.products && props.products.length === 1) {
    return props.products[0] ?? null
  }

  return null
}

export interface ProductFinancials {
  product: Product | null
  productName: string
  imageUrl?: string
  hasProduct: boolean
  sellingPrice: number
  costPrice: number
  margin: number
  marginPercent: number
  orders: number
  spend: number
  cpa: number | null
  breakEvenCPA: number
  netAdProfit: number
  profitPerOrder: number | null
  isProfitable: boolean
  isAboveBreakEven: boolean
}

function getProductFinancials(item: MetaInsightItem): ProductFinancials {
  const prod = getLinkedProduct(item)
  const orders = getPurchasesCount(item)
  const spend = item.spend || 0
  const cpa = getCostPerPurchase(item)

  if (!prod) {
    return {
      product: null,
      productName: '',
      imageUrl: '',
      hasProduct: false,
      sellingPrice: 0,
      costPrice: 0,
      margin: 0,
      marginPercent: 0,
      orders,
      spend,
      cpa,
      breakEvenCPA: 0,
      netAdProfit: -spend,
      profitPerOrder: null,
      isProfitable: false,
      isAboveBreakEven: false,
    }
  }

  const rawPrice = prod.salePrice || (prod.price ? parseFloat(prod.price) : 0) || 0
  const rawCost = prod.cost || 0
  const margin = Math.max(0, rawPrice - rawCost)
  const marginPercent = rawPrice > 0 ? (margin / rawPrice) * 100 : 0

  const grossProfit = orders * margin
  const netAdProfit = grossProfit - spend
  const profitPerOrder = orders > 0 && cpa != null ? (margin - cpa) : null
  const isProfitable = netAdProfit > 0
  const isAboveBreakEven = cpa != null && cpa > margin

  return {
    product: prod,
    productName: prod.name,
    imageUrl: prod.imageUrl,
    hasProduct: true,
    sellingPrice: rawPrice,
    costPrice: rawCost,
    margin,
    marginPercent,
    orders,
    spend,
    cpa,
    breakEvenCPA: margin,
    netAdProfit,
    profitPerOrder,
    isProfitable,
    isAboveBreakEven,
  }
}

export interface DaypartSlot {
  key: 'morning' | 'afternoon' | 'evening' | 'night'
  name: string
  sub: string
  icon: string
  timeRange: string
  spend: number
  purchases: number
  messages: number
  cpa: number | null
  ctr: number
  profit: number
  isCurrentSlot: boolean
  isPeak: boolean
}

export interface AdDiagnostic {
  type: 'danger' | 'warning' | 'success' | 'info'
  icon: string
  title: string
  message: string
  actionLabel?: string
  actionType?: 'pause' | 'scale'
}

function parseHourInt(hourStr?: string): number {
  if (!hourStr) return 0
  const match = hourStr.match(/^(\d{1,2})/)
  return match && match[1] ? parseInt(match[1], 10) : 0
}

function getAdDayparts(item: MetaInsightItem): { slots: DaypartSlot[]; diagnostic: AdDiagnostic } {
  const now = new Date()
  const currentHour = now.getHours()
  const fin = getProductFinancials(item)
  const margin = fin.margin || 0

  const buckets: Record<'morning' | 'afternoon' | 'evening' | 'night', {
    spend: number
    purchases: number
    messages: number
    impressions: number
    clicks: number
  }> = {
    morning: { spend: 0, purchases: 0, messages: 0, impressions: 0, clicks: 0 },
    afternoon: { spend: 0, purchases: 0, messages: 0, impressions: 0, clicks: 0 },
    evening: { spend: 0, purchases: 0, messages: 0, impressions: 0, clicks: 0 },
    night: { spend: 0, purchases: 0, messages: 0, impressions: 0, clicks: 0 },
  }

  const hourlyData = item.hourly || []
  if (hourlyData.length > 0) {
    for (const h of hourlyData) {
      const hInt = parseHourInt(h.hour)
      let slotKey: 'morning' | 'afternoon' | 'evening' | 'night' = 'night'
      if (hInt >= 6 && hInt < 12) slotKey = 'morning'
      else if (hInt >= 12 && hInt < 18) slotKey = 'afternoon'
      else if (hInt >= 18 && hInt < 24) slotKey = 'evening'
      else slotKey = 'night'

      const b = buckets[slotKey]
      b.spend += h.spend || 0
      b.purchases += h.purchasesCount || h.results || 0
      b.messages += h.messagesCount || 0
      b.impressions += h.impressions || 0
      if (h.linkCtr && h.impressions) {
        b.clicks += Math.round((h.linkCtr / 100) * h.impressions)
      }
    }
  } else {
    const totalSpend = item.spend || 0
    const totalPurchases = getPurchasesCount(item)
    const totalMessages = getMessagesCount(item)
    const totalImp = item.impressions || 100
    const totalClicks = item.linkClicks || Math.round((item.linkCtr || 1) * totalImp / 100)

    const weights: Record<'morning' | 'afternoon' | 'evening' | 'night', number> = {
      night: currentHour >= 0 ? 0.10 : 0,
      morning: currentHour >= 6 ? 0.30 : 0,
      afternoon: currentHour >= 12 ? 0.35 : 0,
      evening: currentHour >= 18 ? 0.25 : 0,
    }
    const weightSum = (weights.night + weights.morning + weights.afternoon + weights.evening) || 1

    for (const k of ['night', 'morning', 'afternoon', 'evening'] as const) {
      const share = weights[k] / weightSum
      buckets[k].spend = Math.round(totalSpend * share * 100) / 100
      buckets[k].purchases = Math.round(totalPurchases * share)
      buckets[k].messages = Math.round(totalMessages * share)
      buckets[k].impressions = Math.round(totalImp * share)
      buckets[k].clicks = Math.round(totalClicks * share)
    }
  }

  const maxOrders = Math.max(...Object.values(buckets).map(b => b.purchases))

  const slots: DaypartSlot[] = [
    {
      key: 'morning',
      name: l({ lo: '🌅 ຕອນເຊົ້າ', th: '🌅 ตอนเช้า', en: '🌅 Morning' }),
      sub: '06:00 - 12:00',
      icon: 'mdi-weather-sunset-up',
      timeRange: '06:00 - 12:00',
      spend: buckets.morning.spend,
      purchases: buckets.morning.purchases,
      messages: buckets.morning.messages,
      cpa: buckets.morning.purchases > 0 ? buckets.morning.spend / buckets.morning.purchases : null,
      ctr: buckets.morning.impressions > 0 ? (buckets.morning.clicks / buckets.morning.impressions) * 100 : (item.linkCtr || 0),
      profit: (buckets.morning.purchases * margin) - buckets.morning.spend,
      isCurrentSlot: currentHour >= 6 && currentHour < 12,
      isPeak: maxOrders > 0 && buckets.morning.purchases === maxOrders,
    },
    {
      key: 'afternoon',
      name: l({ lo: '☀️ ຕອນສວາຍ', th: '☀️ ตอนบ่าย', en: '☀️ Afternoon' }),
      sub: '12:00 - 18:00',
      icon: 'mdi-weather-sunny',
      timeRange: '12:00 - 18:00',
      spend: buckets.afternoon.spend,
      purchases: buckets.afternoon.purchases,
      messages: buckets.afternoon.messages,
      cpa: buckets.afternoon.purchases > 0 ? buckets.afternoon.spend / buckets.afternoon.purchases : null,
      ctr: buckets.afternoon.impressions > 0 ? (buckets.afternoon.clicks / buckets.afternoon.impressions) * 100 : (item.linkCtr || 0),
      profit: (buckets.afternoon.purchases * margin) - buckets.afternoon.spend,
      isCurrentSlot: currentHour >= 12 && currentHour < 18,
      isPeak: maxOrders > 0 && buckets.afternoon.purchases === maxOrders,
    },
    {
      key: 'evening',
      name: l({ lo: '🌙 ຫົວຄ່ຳ (Prime)', th: '🌙 หัวค่ำ (Prime)', en: '🌙 Evening (Prime)' }),
      sub: '18:00 - 24:00',
      icon: 'mdi-weather-night',
      timeRange: '18:00 - 24:00',
      spend: buckets.evening.spend,
      purchases: buckets.evening.purchases,
      messages: buckets.evening.messages,
      cpa: buckets.evening.purchases > 0 ? buckets.evening.spend / buckets.evening.purchases : null,
      ctr: buckets.evening.impressions > 0 ? (buckets.evening.clicks / buckets.evening.impressions) * 100 : (item.linkCtr || 0),
      profit: (buckets.evening.purchases * margin) - buckets.evening.spend,
      isCurrentSlot: currentHour >= 18 && currentHour < 24,
      isPeak: maxOrders > 0 && buckets.evening.purchases === maxOrders,
    },
    {
      key: 'night',
      name: l({ lo: '🌌 ຕອນເດິກ', th: '🌌 ดึก', en: '🌌 Late Night' }),
      sub: '00:00 - 06:00',
      icon: 'mdi-moon-waning-crescent',
      timeRange: '00:00 - 06:00',
      spend: buckets.night.spend,
      purchases: buckets.night.purchases,
      messages: buckets.night.messages,
      cpa: buckets.night.purchases > 0 ? buckets.night.spend / buckets.night.purchases : null,
      ctr: buckets.night.impressions > 0 ? (buckets.night.clicks / buckets.night.impressions) * 100 : (item.linkCtr || 0),
      profit: (buckets.night.purchases * margin) - buckets.night.spend,
      isCurrentSlot: currentHour >= 0 && currentHour < 6,
      isPeak: maxOrders > 0 && buckets.night.purchases === maxOrders,
    },
  ]

  const daytimeSpend = buckets.morning.spend + buckets.afternoon.spend
  const daytimePurchases = buckets.morning.purchases + buckets.afternoon.purchases
  const totalPurchases = getPurchasesCount(item)
  const totalSpend = item.spend || 0
  const cpa = getCostPerPurchase(item)
  const ctr = item.linkCtr || 0

  let diagnostic: AdDiagnostic = {
    type: 'info',
    icon: 'mdi-information-outline',
    title: l({ lo: 'ແອດເຮັດວຽກປົກກະຕິ', th: 'แอดทำงานปกติ', en: 'Ad Operating Normally' }),
    message: l({ lo: 'ການໃຊ້ຈ່າຍ ແລະ ປະສິດທິພາບຢູ່ໃນເກນປົກກະຕິ, ສືບຕໍ່ຕິດຕາມໃນຊ່ວງ Prime Time.', th: 'การใช้จ่ายและประสิทธิภาพอยู่ในเกณฑ์ปกติ ติดตามดูต่อเนื่องช่วง Prime Time', en: 'Spend and performance are healthy. Keep monitoring for prime time.' }),
  }

  if (daytimeSpend >= 3 && daytimePurchases === 0) {
    diagnostic = {
      type: 'warning',
      icon: 'mdi-fire-alert',
      title: l({ lo: '⚠️ ງົບໄຫຼຕອນກາງເວັນ ແຕ່ບໍ່ມີຍອດຊື້ (Daytime Drain)', th: '⚠️ งบไหลตอนกลางวันแต่ไม่มีออเดอร์', en: '⚠️ Daytime Budget Drain' }),
      message: l({
        lo: `ໃຊ້ງົບໃນຊ່ວງເຊົ້າ-ສວາຍແລ້ວ $${daytimeSpend.toFixed(2)} ແຕ່ຍັງບໍ່ມີອໍເດີ. ຖ້າລູກຄ້າສິນຄ້ານີ້ຊື້ຕອນແລງ, ຄວນຕັ້ງເວລາ Dayparting ຫຼື ພັກແອດໄວ້ກ່ອນ.`,
        th: `ใช้งบตอนเช้า-บ่ายไปแล้ว $${daytimeSpend.toFixed(2)} แต่ยังไม่มีออเดอร์ หากลูกค้าซื้อตอนค่ำ แนะนำตั้ง Dayparting หรือพักแอด`,
        en: `Spent $${daytimeSpend.toFixed(2)} daytime with 0 purchases. Consider dayparting schedule or pausing.`
      }),
      actionLabel: l({ lo: '⏸️ ພັກແອດກ່ອນ', th: '⏸️ พักแอดก่อน', en: 'Pause Ad' }),
      actionType: 'pause',
    }
  } else if (totalPurchases >= 2 && (cpa == null || (margin > 0 ? cpa <= margin : cpa <= 3.0))) {
    diagnostic = {
      type: 'success',
      icon: 'mdi-rocket-launch',
      title: l({ lo: '🚀 ແອດແຮງມື້ນີ້ ໂອກາດ Scale ງົບຕອນແລງ (Scale Opportunity)', th: '🚀 แอดแรงวันนี้ โอกาส Scale งบตอนค่ำ', en: '🚀 Prime Scale Opportunity' }),
      message: l({
        lo: `ຍອດຊື້ເຂົ້າມາແລ້ວ ${totalPurchases} ອໍເດີ, CPA ຖືກພຽງ $${(cpa || 0).toFixed(2)}. ແນະນຳກົດ Scale ງົບ (+20%) ເພື່ອດັກຈັບຍອດຊ່ວງຫົວຄ່ຳ (18:00 - 24:00)!`,
        th: `ยอดซื้อเข้ามาแล้ว ${totalPurchases} ออเดอร์ CPA ถูกเพียง $${(cpa || 0).toFixed(2)} แนะนำกด Scale งบ (+20%) เพื่อดักยอดช่วงหัวค่ำ`,
        en: `${totalPurchases} orders today, cheap CPA $${(cpa || 0).toFixed(2)}. Recommended to Scale (+20%) for prime time.`
      }),
      actionLabel: l({ lo: '🚀 Scale ງົບ (+20%)', th: '🚀 Scale งบ (+20%)', en: 'Scale Budget' }),
      actionType: 'scale',
    }
  } else if (totalPurchases > 0 && margin > 0 && cpa != null && cpa > margin) {
    diagnostic = {
      type: 'danger',
      icon: 'mdi-alert-decagram',
      title: l({ lo: '🛑 ຂາຍໄດ້ແຕ່ຕິດລົບ (CPA > Margin)', th: '🛑 ขายได้แต่ติดลบ (CPA > Margin)', en: '🛑 Loss Per Sale (CPA > Margin)' }),
      message: l({
        lo: `ຕົ້ນທຶນແອດ ($${cpa.toFixed(2)}/ອໍເດີ) ສູງກວ່າກຳໄລຕໍ່ຊິ້ນ ($${margin.toFixed(2)}). ຂາຍທຸກອໍເດີຂາດທຶນ -$${(cpa - margin).toFixed(2)}. ຄວນປັບ Creative ຫຼື Up-sell ແພັກຄູ່.`,
        th: `ต้นทุนแอด ($${cpa.toFixed(2)}) สูงกว่ากำไรสินค้า ($${margin.toFixed(2)}) ขายขาดทุน -$${(cpa - margin).toFixed(2)} ต่องวด`,
        en: `Ad cost per order ($${cpa.toFixed(2)}) exceeds product margin ($${margin.toFixed(2)}). Net loss -$${(cpa - margin).toFixed(2)}/sale.`
      }),
      actionLabel: l({ lo: '⏸️ ປິດຢຸດຂາດທຶນ', th: '⏸️ ปิดหยุดขาดทุน', en: 'Pause Ad' }),
      actionType: 'pause',
    }
  } else if (totalSpend >= 3.0 && totalPurchases === 0 && ctr < 1.0) {
    diagnostic = {
      type: 'danger',
      icon: 'mdi-trending-down',
      title: l({ lo: '🛑 ຄົນບໍ່ຄລິກ CTR ຕ່ຳ (Creative Fatigue)', th: '🛑 คนไม่คลิก CTR ต่ำ', en: '🛑 Creative Fatigue Today' }),
      message: l({
        lo: `Link CTR ພຽງ ${ctr.toFixed(2)}% (ຕ່ຳກວ່າເກນ 1.2%) ແລະ ໃຊ້ງົບໄປແລ້ວ $${totalSpend.toFixed(2)} ໂດຍຍັງບໍ່ມີອໍເດີ. ແນະນຳໃຫ້ປິດພັກແອດເພື່ອບໍ່ໃຫ້ເສຍງົບຟຣີ.`,
        th: `Link CTR เพียง ${ctr.toFixed(2)}% และใช้งบไป $${totalSpend.toFixed(2)} ยังไม่มีออเดอร์ แนะนำพักแอด`,
        en: `Low Link CTR ${ctr.toFixed(2)}% and spent $${totalSpend.toFixed(2)} with 0 orders. Recommended to pause.`
      }),
      actionLabel: l({ lo: '⏸️ ປິດແອດ', th: '⏸️ ปิดแอด', en: 'Pause Ad' }),
      actionType: 'pause',
    }
  }

  return { slots, diagnostic }
}

export interface AdSeries {
  id: string
  name: string
  color: string
  visible: boolean
  isAngel: boolean
  isBad: boolean
  currentScore: number
  trendDiff: number
  item: MetaInsightItem
  dataPoints: Array<{
    date: string
    purchases: number
    cpa: number | null
    messages: number
    ctr: number
    spend: number
    strengthScore: number
  }>
}

const activeAdSeries = ref<AdSeries[]>([])
const expandedSlots = ref<Record<string, boolean>>({})

function isSlotExpanded(id: string): boolean {
  return !!expandedSlots.value[id]
}

function toggleSlotExpand(id: string) {
  expandedSlots.value = {
    ...expandedSlots.value,
    [id]: !expandedSlots.value[id],
  }
}

const allSlotsExpanded = ref(false)
function toggleAllSlots() {
  allSlotsExpanded.value = !allSlotsExpanded.value
  const next: Record<string, boolean> = {}
  for (const s of activeAdSeries.value) {
    next[s.id] = allSlotsExpanded.value
  }
  expandedSlots.value = next
}

const isZeroDecimal = computed(() => zeroDecimalCurrencies.has((props.currency || '').toUpperCase()))

function checkIsAngel(item: MetaInsightItem): boolean {
  if (!item.impressions && !item.spend) return false
  const purchases = getPurchasesCount(item)
  const cost = getCostPerPurchase(item)
  const messages = getMessagesCount(item)
  const ctr = item.linkCtr || 0
  const roas = item.roas || 0
  const targetCpa = isZeroDecimal.value ? 100000 : 5

  if (purchases >= 3 && (cost == null || cost <= targetCpa * 1.35)) return true
  if (purchases >= 1 && (roas >= 2.0 || (cost != null && cost <= targetCpa))) return true
  if (purchases === 0 && messages >= 5 && ctr >= 2.0) return true
  return false
}

function checkIsBad(item: MetaInsightItem): boolean {
  if (!item.impressions && !item.spend) return false
  const purchases = getPurchasesCount(item)
  const ctr = item.linkCtr || 0
  const spend = item.spend || 0
  const freq = item.frequency || 0
  const messages = getMessagesCount(item)
  const minSpendThreshold = isZeroDecimal.value ? 50000 : 3
  const reasons: boolean[] = [
    purchases === 0 && spend >= minSpendThreshold,
    ctr > 0 && ctr < 1.0,
    freq >= 2.2,
    purchases === 0 && messages > 0 && spend >= minSpendThreshold * 1.5,
  ]
  return purchases === 0 && reasons.some(Boolean)
}

// Build series data from items
watch(() => [props.items, props.accountTrend], () => {
  const allDates = dates.value
  const list = props.items || []

  activeAdSeries.value = list.map((item, index) => {
    const purchases = getPurchasesCount(item)
    const cpa = getCostPerPurchase(item)
    const messages = getMessagesCount(item)
    const ctr = item.linkCtr || 0
    const spend = item.spend || 0
    const freq = item.frequency || 1.2

    const overallScore = computeStrengthScore(purchases, cpa, ctr, spend, freq)
    const isAngel = checkIsAngel(item)
    const isBad = checkIsBad(item)
    const color = getSemanticColor(isAngel, isBad, index)

    // Build day-by-day points
    const dailyPoints = allDates.map((d, dIdx) => {
      const matched = item.trend?.find((t: MetaInsightPoint) => t.date === d)
      if (matched) {
        const dPurchases = matched.results || 0
        const dCpa = matched.costPerResult || (dPurchases > 0 && matched.spend ? matched.spend / dPurchases : null)
        const dMessages = matched.messagesCount || 0
        const dCtr = matched.linkCtr || 0
        const dSpend = matched.spend || 0
        const dFreq = matched.frequency || 1.1
        const dScore = computeStrengthScore(dPurchases, dCpa, dCtr, dSpend, dFreq)
        return {
          date: d,
          purchases: dPurchases,
          cpa: dCpa,
          messages: dMessages,
          ctr: dCtr,
          spend: dSpend,
          strengthScore: dScore,
        }
      }

      const accountPoint = props.accountTrend?.find(t => t.date === d)
      const dayFactor = accountPoint?.spend && props.accountTrend?.length
        ? Math.max(0.1, (accountPoint.spend / (props.accountTrend.reduce((sum, p) => sum + (p.spend || 0), 0) || 1)) * allDates.length)
        : 1.0

      const dSpend = Math.round(((spend / allDates.length) * dayFactor) * 100) / 100
      const dPurchases = purchases > 0 ? (dIdx % 2 === 0 ? Math.ceil(purchases / allDates.length) : Math.floor(purchases / allDates.length)) : 0
      const dCpa = dPurchases > 0 ? dSpend / dPurchases : null
      const dMessages = messages > 0 ? Math.round(messages / allDates.length) : 0
      const dCtr = Math.max(0.2, Math.round((ctr + (Math.sin(dIdx) * 0.3)) * 100) / 100)
      const dScore = computeStrengthScore(dPurchases, dCpa, dCtr, dSpend, freq)

      return {
        date: d,
        purchases: dPurchases,
        cpa: dCpa,
        messages: dMessages,
        ctr: dCtr,
        spend: dSpend,
        strengthScore: dScore,
      }
    })

    const firstScore = dailyPoints[0]?.strengthScore ?? overallScore
    const lastScore = dailyPoints[dailyPoints.length - 1]?.strengthScore ?? overallScore
    const trendDiff = lastScore - firstScore

    return {
      id: item.entityId,
      name: item.entityName,
      color,
      visible: true,
      isAngel,
      isBad,
      currentScore: overallScore,
      trendDiff,
      item,
      dataPoints: dailyPoints,
    }
  })
}, { immediate: true })

// Combined Chart SVG Dimensions
const svgWidth = 980
const svgHeight = 330
const padding = { top: 30, right: 160, bottom: 40, left: 55 }

const innerWidth = computed(() => svgWidth - padding.left - padding.right)
const innerHeight = computed(() => svgHeight - padding.top - padding.bottom)

function getMetricValue(pt: AdSeries['dataPoints'][0]): number {
  switch (selectedMetric.value) {
    case 'strength': return pt.strengthScore
    case 'purchases': return pt.purchases
    case 'cpa': return pt.cpa ?? 0
    case 'messages': return pt.messages
    case 'ctr': return pt.ctr
    case 'spend': return pt.spend
    default: return pt.strengthScore
  }
}

function formatMetricDisplay(val: number | null): string {
  if (val == null) return '—'
  switch (selectedMetric.value) {
    case 'strength': return `${val}/100`
    case 'purchases': return `${val} ອໍເດີ`
    case 'cpa': return `$${Number(val).toFixed(2)}`
    case 'messages': return `${val} ຄົນ`
    case 'ctr': return `${Number(val).toFixed(2)}%`
    case 'spend': return `$${Number(val).toFixed(2)}`
    default: return `${val}`
  }
}

const yRange = computed(() => {
  if (selectedMetric.value === 'strength') {
    return { min: 0, max: 100 }
  }

  let max = 1
  for (const series of activeAdSeries.value) {
    if (!series.visible) continue
    for (const pt of series.dataPoints) {
      const v = getMetricValue(pt)
      if (v > max) max = v
    }
  }
  return { min: 0, max: Math.ceil(max * 1.15) || 5 }
})

function getX(dIdx: number): number {
  const count = dates.value.length
  if (count <= 1) return padding.left + innerWidth.value / 2
  return padding.left + (dIdx / (count - 1)) * innerWidth.value
}

function getY(value: number): number {
  const { min, max } = yRange.value
  const norm = (value - min) / (max - min || 1)
  return padding.top + innerHeight.value - (norm * innerHeight.value)
}

function generatePath(series: AdSeries): string {
  if (series.dataPoints.length === 0) return ''
  const points = series.dataPoints.map((pt, idx) => ({
    x: getX(idx),
    y: getY(getMetricValue(pt)),
  }))

  const first = points[0]
  if (!first) return ''
  if (points.length === 1) {
    return `M ${first.x} ${first.y}`
  }

  let d = `M ${first.x} ${first.y}`
  for (let i = 0; i < points.length - 1; i++) {
    const p0 = points[i === 0 ? i : i - 1] ?? first
    const p1 = points[i] ?? first
    const p2 = points[i + 1] ?? first
    const p3 = points[i + 2] ?? p2

    const cp1x = p1.x + (p2.x - p0.x) / 6
    const cp1y = p1.y + (p2.y - p0.y) / 6
    const cp2x = p2.x - (p3.x - p1.x) / 6
    const cp2y = p2.y - (p3.y - p1.y) / 6

    d += ` C ${cp1x.toFixed(1)} ${cp1y.toFixed(1)}, ${cp2x.toFixed(1)} ${cp2y.toFixed(1)}, ${p2.x.toFixed(1)} ${p2.y.toFixed(1)}`
  }
  return d
}

// Dedicated Generator for Single Ad Cards
function generateSingleAdPath(series: AdSeries, width = 520, height = 130, paddingX = 40, paddingY = 25) {
  if (!series.dataPoints.length) return { linePath: '', areaPath: '', points: [] }
  const count = series.dataPoints.length

  let min = 0
  let max = 100
  if (selectedMetric.value !== 'strength') {
    let m = 1
    for (const p of series.dataPoints) {
      const v = getMetricValue(p)
      if (v > m) m = v
    }
    max = Math.ceil(m * 1.25) || 5
  }

  const innerW = width - 2 * paddingX
  const innerH = height - 2 * paddingY

  const pts = series.dataPoints.map((pt, idx) => {
    const val = getMetricValue(pt)
    const x = count <= 1 ? width / 2 : paddingX + (idx / (count - 1)) * innerW
    const norm = (val - min) / (max - min || 1)
    const y = paddingY + innerH - (norm * innerH)
    return {
      x,
      y,
      val,
      label: formatMetricDisplay(val),
      date: pt.date,
      pt,
    }
  })

  const first = pts[0]
  if (!first) return { linePath: '', areaPath: '', points: [] }
  if (pts.length === 1) {
    return { linePath: `M ${first.x} ${first.y}`, areaPath: '', points: pts }
  }

  let linePath = `M ${first.x} ${first.y}`
  for (let i = 0; i < pts.length - 1; i++) {
    const p0 = pts[i === 0 ? i : i - 1] ?? first
    const p1 = pts[i] ?? first
    const p2 = pts[i + 1] ?? first
    const p3 = pts[i + 2] ?? p2

    const cp1x = p1.x + (p2.x - p0.x) / 6
    const cp1y = p1.y + (p2.y - p0.y) / 6
    const cp2x = p2.x - (p3.x - p1.x) / 6
    const cp2y = p2.y - (p3.y - p1.y) / 6

    linePath += ` C ${cp1x.toFixed(1)} ${cp1y.toFixed(1)}, ${cp2x.toFixed(1)} ${cp2y.toFixed(1)}, ${p2.x.toFixed(1)} ${p2.y.toFixed(1)}`
  }

  const last = pts[pts.length - 1] ?? first
  const areaPath = `${linePath} L ${last.x} ${height - 12} L ${first.x} ${height - 12} Z`

  return { linePath, areaPath, points: pts }
}

const currentFilter = ref<'all' | 'angels' | 'bad' | 'top3'>('all')

function filterSeries(type: 'all' | 'angels' | 'bad' | 'top3') {
  currentFilter.value = type
  if (type === 'all') {
    activeAdSeries.value.forEach(s => { s.visible = true })
  } else if (type === 'angels') {
    activeAdSeries.value.forEach(s => { s.visible = s.isAngel })
  } else if (type === 'bad') {
    activeAdSeries.value.forEach(s => { s.visible = s.isBad })
  } else if (type === 'top3') {
    activeAdSeries.value.forEach((s, idx) => {
      s.visible = idx < 3 || s.isAngel || s.isBad
    })
  }
}

function handlePointHover(series: AdSeries, pt: AdSeries['dataPoints'][0], idx: number) {
  const x = getX(idx)
  const y = getY(getMetricValue(pt))
  const val = getMetricValue(pt)
  const score = pt.strengthScore
  const info = getStrengthLabel(score)

  hoveredPoint.value = {
    adName: series.name,
    adId: series.id,
    color: series.color,
    date: pt.date,
    metricValue: val,
    displayValue: formatMetricDisplay(val),
    strengthScore: score,
    strengthLabel: info.label,
    purchases: pt.purchases,
    cpa: pt.cpa,
    ctr: pt.ctr,
    spend: pt.spend,
    x,
    y,
  }
}

const visibleSeriesList = computed(() => {
  if (focusedAdId.value !== 'all') {
    return activeAdSeries.value.filter(s => s.id === focusedAdId.value)
  }
  return activeAdSeries.value.filter(s => s.visible)
})

const strongestAd = computed(() => {
  if (!activeAdSeries.value.length) return null
  const sorted = [...activeAdSeries.value].sort((a, b) => b.currentScore - a.currentScore)
  return sorted[0] ?? null
})

const weakestAd = computed(() => {
  if (!activeAdSeries.value.length) return null
  const sorted = [...activeAdSeries.value].sort((a, b) => a.currentScore - b.currentScore)
  return sorted[0] ?? null
})

const aggregatedMetrics = computed(() => {
  let totalSpend = 0
  let totalPurchases = 0
  let totalMessages = 0
  let angelSpend = 0
  let badSpend = 0
  let totalNetProfit = 0

  for (const s of activeAdSeries.value) {
    const it = s.item
    const sp = it.spend || 0
    const pur = getPurchasesCount(it)
    const msg = getMessagesCount(it)
    const fin = getProductFinancials(it)

    totalSpend += sp
    totalPurchases += pur
    totalMessages += msg
    totalNetProfit += fin.netAdProfit

    if (s.isAngel) {
      angelSpend += sp
    } else if (s.isBad) {
      badSpend += sp
    }
  }

  const neutralSpend = Math.max(0, totalSpend - angelSpend - badSpend)
  const avgCpa = totalPurchases > 0 ? totalSpend / totalPurchases : null

  return {
    totalSpend,
    totalPurchases,
    totalMessages,
    angelSpend,
    badSpend,
    neutralSpend,
    avgCpa,
    totalNetProfit,
    adCount: activeAdSeries.value.length,
  }
})
</script>

<template>
  <div class="ad-strength-chart-card">
    <!-- Header with Metric Selector & Display Mode -->
    <div class="chart-header">
      <div class="chart-title-area">
        <div class="d-flex align-center gap-2 mb-1 flex-wrap">
          <span class="chart-tag">
            <v-icon icon="mdi-chart-bell-curve-cumulative" size="14" />
            DAILY AD MOMENTUM
          </span>
          <span class="color-guide-hint">
            <span class="dot green-dot" /> 🟢 ສີຂຽວ = Ads ນາງຟ້າ
            <span class="dot red-dot ml-2" /> 🔴 ສີແດງ = ແອດໝົດກຳລັງ
          </span>
        </div>
        <h2>{{ l({ lo: '📈 ກຣາຟເສັ້ນຄວາມແຮງ ຂອງແຕ່ລະ Ads (ແຍກລະດັບ Ads ຊັດເຈນ)', th: '📈 กราฟเส้นความแรง ของแต่ละ Ads (แยกระดับ Ads ชัดเจน)', en: 'Daily Ad Momentum (Clean Ad-Level Separation)' }) }}</h2>
        <p class="chart-sub">
          {{ l({ lo: 'ກວດເບິ່ງແຮງຂອງແຕ່ລະແອດແບບລາຍມື້, ເຫັນຊື່ແອດຊັດເຈນ, ພ້ອມປຸ່ມ ເປີດ-ປິດ (Open / Close Ads) ໃນແຕ່ລະກາດທັນທີ', th: 'ตรวจดูความแรงของแต่ละแอดแบบรายวัน เห็นชื่อแอดชัดเจน พร้อมปุ่ม เปิด-ปิด (Open / Close Ads) ในแต่ละการ์ดทันที', en: 'Audit ad momentum daily with clear names and direct Open/Close toggles on each card.' }) }}
        </p>
      </div>

      <!-- Controls: Display Mode & Metric Selector -->
      <div class="header-control-cluster">
        <!-- Display Mode Switcher (Separate vs Combined) -->
        <div class="mode-switcher-wrap">
          <v-btn-toggle
            v-model="displayMode"
            mandatory
            color="primary"
            density="compact"
            variant="flat"
            class="mode-toggle-group"
          >
            <v-btn value="separate">
              <v-icon icon="mdi-view-grid-outline" size="15" class="mr-1 text-primary" />
              <span>{{ l({ lo: '🔲 ແຍກກາດແຕ່ລະ Ads (ແນະນຳ)', th: '🔲 แยกการ์ดแต่ละ Ads (แนะนำ)', en: '🔲 Separate Cards' }) }}</span>
            </v-btn>
            <v-btn value="combined">
              <v-icon icon="mdi-chart-multiline" size="15" class="mr-1 text-secondary" />
              <span>{{ l({ lo: '📊 ຮວມກັນໃນກຣາຟດຽວ', th: '📊 รวมกันในกราฟเดียว', en: '📊 Combined View' }) }}</span>
            </v-btn>
          </v-btn-toggle>
        </div>

        <!-- Metric Toggle Buttons -->
        <div class="metric-switchers">
          <v-btn-toggle
            v-model="selectedMetric"
            mandatory
            color="primary"
            density="compact"
            variant="outlined"
            class="metric-toggle-group"
          >
            <v-btn value="strength">
              <v-icon icon="mdi-lightning-bolt" size="14" class="mr-1 text-amber-darken-3" />
              <span>{{ l({ lo: '⚡ ຄວາມແຮງ (Score)', th: '⚡ ความแรง (Score)', en: '⚡ Strength' }) }}</span>
            </v-btn>
            <v-btn value="purchases">
              <v-icon icon="mdi-cart-check" size="14" class="mr-1 text-success" />
              <span>{{ l({ lo: '🛒 ອໍເດີ', th: '🛒 ออเดอร์', en: '🛒 Orders' }) }}</span>
            </v-btn>
            <v-btn value="cpa">
              <v-icon icon="mdi-target" size="14" class="mr-1 text-primary" />
              <span>{{ l({ lo: '🎯 CPA', th: '🎯 CPA', en: '🎯 CPA' }) }}</span>
            </v-btn>
            <v-btn value="ctr">
              <v-icon icon="mdi-cursor-default-click" size="14" class="mr-1 text-purple" />
              <span>{{ l({ lo: '📈 CTR %', th: '📈 CTR %', en: '📈 CTR' }) }}</span>
            </v-btn>
            <v-btn value="spend">
              <v-icon icon="mdi-cash" size="14" class="mr-1 text-error" />
              <span>{{ l({ lo: '💸 ງົບໃຊ້', th: '💸 ใช้งบ', en: '💸 Spend' }) }}</span>
            </v-btn>
          </v-btn-toggle>
        </div>
      </div>
    </div>


    <!-- Quick Preset Filters & Spotlight -->
    <div class="chart-ad-filters">
      <div class="d-flex align-center gap-2 flex-wrap">
        <span class="filter-label">
          <v-icon icon="mdi-filter-variant" size="14" />
          {{ l({ lo: 'ກັ່ນຕອງ:', th: 'กรอง:', en: 'Filter:' }) }}
        </span>
        <div class="filter-pills">
          <button
            type="button"
            class="pill-btn"
            :class="{ active: currentFilter === 'all' }"
            @click="filterSeries('all')"
          >
            {{ l({ lo: '📋 ທຸກ Ads', th: '📋 ทุก Ads', en: 'All' }) }}
          </button>
          <button
            type="button"
            class="pill-btn angel"
            :class="{ active: currentFilter === 'angels' }"
            @click="filterSeries('angels')"
          >
            🟢 {{ l({ lo: 'ສະເພາະ Ads ນາງຟ້າ', th: 'เฉพาะแอดนางฟ้า', en: 'Angel Ads' }) }}
          </button>
          <button
            type="button"
            class="pill-btn bad"
            :class="{ active: currentFilter === 'bad' }"
            @click="filterSeries('bad')"
          >
            🔴 {{ l({ lo: 'ສະເພາະ ແອດໝົດກຳລັງ', th: 'เฉพาะแอดหมดกำลัง', en: 'Fatigued Ads' }) }}
          </button>
          <button
            type="button"
            class="pill-btn top3"
            :class="{ active: currentFilter === 'top3' }"
            @click="filterSeries('top3')"
          >
            {{ l({ lo: 'ທຽບ ນາງຟ້າ VS ໂຕບໍ່ດີ', th: 'เทียบ นางฟ้า VS ตัวไม่ดี', en: 'Battle Top 3' }) }}
          </button>
          <button
            v-if="props.isToday"
            type="button"
            class="pill-btn"
            :class="{ active: allSlotsExpanded }"
            @click="toggleAllSlots"
          >
            {{ allSlotsExpanded ? '▲ ຫຍໍ້ 4 ຊ່ວງເວລາ' : '🕒 ເບິ່ງ 4 ຊ່ວງເວລາ' }}
          </button>
        </div>
      </div>

      <!-- Single Ad Isolator / Spotlight Dropdown -->
      <div class="ad-spotlight-select">
        <label>{{ l({ lo: '🔍 ເຈາະຈົງແອດດຽວ:', th: '🔍 เจาะจงแอดเดียว:', en: 'Focus Ad:' }) }}</label>
        <select v-model="focusedAdId" class="ad-select-dropdown">
          <option value="all">{{ l({ lo: '— ສະແດງທັງໝົດ —', th: '— แสดงทั้งหมด —', en: '— Show All —' }) }}</option>
          <option v-for="s in activeAdSeries" :key="s.id" :value="s.id">
            {{ s.isAngel ? '🟢 ນາງຟ້າ: ' : s.isBad ? '🔴 ໝົດກຳລັງ: ' : '🔘 ທົ່ວໄປ: ' }}{{ s.name }}
          </option>
        </select>
      </div>
    </div>

    <!-- ========================================================================= -->
    <!-- VIEW MODE 1: SEPARATE INDIVIDUAL AD CARDS (ໂໝດແຍກກາດແຕ່ລະ Ads ຊັດເຈນ) -->
    <!-- ========================================================================= -->
    <div v-if="displayMode === 'separate'" class="separate-view-container">
      <div class="separate-grid">
        <article
          v-for="series in visibleSeriesList"
          :key="series.id"
          class="ad-solo-card"
          :class="{
            'theme-angel': series.isAngel,
            'theme-bad': series.isBad,
            'theme-neutral': !series.isAngel && !series.isBad,
            'is-paused-card': !isItemActive(series.item)
          }"
        >
          <!-- Top Row: Semantic Badge, Trajectory, and OPEN/CLOSE ADS SWITCH -->
          <div class="solo-top-control-bar">
            <div class="d-flex align-center gap-2 flex-wrap">
              <!-- Clear Semantic Classification -->
              <span v-if="series.isAngel" class="semantic-pill angel">
                <v-icon icon="mdi-star-face" size="13" class="mr-1" />
                Ads ນາງຟ້າ
              </span>
              <span v-else-if="series.isBad" class="semantic-pill bad">
                <v-icon icon="mdi-alert-octagon" size="13" class="mr-1" />
                ແອດໝົດກຳລັງ
              </span>
              <span v-else class="semantic-pill neutral">
                <v-icon icon="mdi-shield-check" size="13" class="mr-1" />
                ແອດທົ່ວໄປ
              </span>

              <!-- Trajectory Pill -->
              <span
                class="trajectory-pill"
                :class="series.trendDiff > 3 ? 'trend-up' : series.trendDiff < -3 ? 'trend-down' : 'trend-flat'"
              >
                <v-icon
                  :icon="series.trendDiff > 3 ? 'mdi-trending-up' : series.trendDiff < -3 ? 'mdi-trending-down' : 'mdi-trending-neutral'"
                  size="13"
                />
                {{ series.trendDiff > 3 ? `ຂາຂຶ້ນ (+${series.trendDiff}%)` : series.trendDiff < -3 ? `ຂາລົງ (${series.trendDiff}%)` : 'ຄົງທີ່' }}
              </span>

              <!-- Score Pill -->
              <span class="score-badge-clear" :class="series.isAngel ? 'score-green' : series.isBad ? 'score-red' : 'score-blue'">
                Score: <b>{{ series.currentScore }}/100</b>
              </span>
            </div>

            <!-- FUNCTION OPEN / CLOSE AD DIRECT TOGGLE SWITCH -->
            <div
              class="ad-switch-container"
              :class="isItemActive(series.item) ? 'status-active-box' : 'status-paused-box'"
            >
              <span class="switch-text">
                {{ isItemActive(series.item) ? '🟢 ເປີດແອດ (Active)' : '⏸️ ປິດແອດ (Paused)' }}
              </span>
              <v-switch
                :model-value="isItemActive(series.item)"
                color="success"
                density="compact"
                hide-details
                :loading="props.updatingId === series.id"
                class="compact-ad-switch"
                @update:model-value="(val) => handleToggleAd(series.item, val)"
              />
            </div>
          </div>

          <!-- Middle Row: Prominent Ad Name & ID -->
          <div class="solo-identity-block">
            <h3 class="ad-clean-name" :title="series.name">
              <span class="semantic-dot" :class="series.isAngel ? 'bg-green' : series.isBad ? 'bg-red' : 'bg-blue'" />
              {{ series.name }}
            </h3>
            <div class="ad-sub-id">
              <code>ID: {{ series.id }}</code>
            </div>
          </div>

          <!-- Product Financials & Margin Box -->
          <div class="ad-product-financial-card">
            <div class="product-header-bar">
              <div class="d-flex align-center gap-2 flex-grow-1 min-w-0">
                <img
                  v-if="getProductFinancials(series.item).imageUrl"
                  
                  :src="getProductFinancials(series.item).imageUrl"
                  class="product-thumb-mini"
                  alt="Product"
                />
                <div v-else class="product-thumb-placeholder">
                  <v-icon icon="mdi-package-variant-closed" size="16" />
                </div>
                <div class="product-info-texts min-w-0">
                  <div class="d-flex align-center gap-1">
                    <span class="product-name-badge" :title="getProductFinancials(series.item).productName || 'ຍັງບໍ່ໄດ້ຜູກສິນຄ້າ'">
                      📦 {{ getProductFinancials(series.item).productName || 'ຍັງບໍ່ໄດ້ຜູກສິນຄ້າ' }}
                    </span>
                    <v-menu v-if="props.products && props.products.length > 0" location="bottom start">
                      <template #activator="{ props: menuProps }">
                        <button
                          type="button"
                          v-bind="menuProps"
                          class="product-change-btn"
                          title="ປ່ຽນສິນຄ້າທີ່ຜູກ"
                        >
                          <v-icon icon="mdi-pencil-outline" size="12" />
                        </button>
                      </template>
                      <v-list density="compact" class="product-select-menu">
                        <v-list-subheader>{{ l({ lo: 'ເລືອກສິນຄ້າເພື່ອຄິດໄລ່ກຳໄລ', th: 'เลือกสินค้าเพื่อคิดกำไร', en: 'Select Product for Margin' }) }}</v-list-subheader>
                        <v-list-item
                          v-for="p in props.products"
                          :key="p.id"
                          :value="p.id"
                          :active="getProductFinancials(series.item).product?.id === p.id"
                          @click="setProductOverride(series.id, p.id)"
                        >
                          <template #prepend>
                            <v-avatar size="24" rounded class="mr-2">
                              <v-img v-if="p.imageUrl" :src="p.imageUrl" />
                              <v-icon v-else icon="mdi-package-variant-closed" size="14" />
                            </v-avatar>
                          </template>
                          <v-list-item-title class="text-caption font-weight-bold">{{ p.name }}</v-list-item-title>
                          <v-list-item-subtitle class="text-caption">
                            ຂາຍ: {{ money(p.salePrice || (p.price ? parseFloat(p.price) : 0)) }} | ຕົ້ນທຶນ: {{ money(p.cost || 0) }}
                          </v-list-item-subtitle>
                        </v-list-item>
                      </v-list>
                    </v-menu>
                  </div>
                </div>
              </div>

              <!-- Net Ad Profit Badge -->
              <div
                class="net-profit-badge"
                :class="getProductFinancials(series.item).netAdProfit >= 0 ? 'profit-positive' : 'profit-negative'"
              >
                <small>{{ l({ lo: 'ກຳໄລແອດສຸດທິ', th: 'กำไรแอดสุทธิ', en: 'Net Ad Profit' }) }}</small>
                <span class="profit-number">
                  {{ getProductFinancials(series.item).netAdProfit >= 0 ? '+' : '' }}{{ money(getProductFinancials(series.item).netAdProfit) }}
                </span>
              </div>
            </div>

            <!-- Pricing & Unit Margin Grid -->
            <div class="product-pricing-grid">
              <div class="price-cell">
                <span class="p-label">{{ l({ lo: '🏷️ ລາຄາຂາຍ:', th: '🏷️ ราคาขาย:', en: 'Price:' }) }}</span>
                <span class="p-val">{{ money(getProductFinancials(series.item).sellingPrice) }}</span>
              </div>
              <div class="price-cell">
                <span class="p-label">{{ l({ lo: '📦 ຕົ້ນທຶນ:', th: '📦 ต้นทุน:', en: 'Cost:' }) }}</span>
                <span class="p-val">{{ money(getProductFinancials(series.item).costPrice) }}</span>
              </div>
              <div class="price-cell">
                <span class="p-label">{{ l({ lo: '💎 ກຳໄລ/ຊິ້ນ:', th: '💎 กำไร/ชิ้น:', en: 'Margin:' }) }}</span>
                <span class="p-val text-success font-weight-bold">
                  +{{ money(getProductFinancials(series.item).margin) }}
                  <small v-if="getProductFinancials(series.item).marginPercent > 0">({{ getProductFinancials(series.item).marginPercent.toFixed(0) }}%)</small>
                </span>
              </div>
              <div class="price-cell">
                <span class="p-label">{{ l({ lo: '📊 ກຳໄລ/ອໍເດີ:', th: '📊 กำไร/ออเดอร์:', en: 'Profit/Order:' }) }}</span>
                <span
                  class="p-val font-weight-bold"
                  :class="(getProductFinancials(series.item).profitPerOrder ?? 0) >= 0 ? 'text-success' : 'text-error'"
                >
                  <template v-if="getProductFinancials(series.item).profitPerOrder != null">
                    {{ (getProductFinancials(series.item).profitPerOrder ?? 0) >= 0 ? '+' : '' }}{{ money(getProductFinancials(series.item).profitPerOrder) }}
                  </template>
                  <template v-else>—</template>
                </span>
              </div>
            </div>

            <!-- Alert if CPA > Margin -->
            <div v-if="getProductFinancials(series.item).isAboveBreakEven" class="margin-danger-alert">
              <v-icon icon="mdi-alert-circle-outline" size="14" class="mr-1" />
              <span><strong>{{ l({ lo: 'ເຕືອນໄພຂາດທຶນ:', th: 'เตือนภัยขาดทุน:', en: 'Loss Alert:' }) }}</strong> CPA ({{ money(getProductFinancials(series.item).cpa) }}) ສູງກວ່າກຳໄລຕໍ່ຊິ້ນ ({{ money(getProductFinancials(series.item).margin) }}).</span>
            </div>
          </div>

          <!-- 4 Clean Metric Boxes -->
          <div class="metric-clean-grid">
            <div class="stat-cell">
              <small>{{ l({ lo: 'ໃຊ້ຈ່າຍ', th: 'ใช้จ่าย', en: 'Spend' }) }}</small>
              <strong>{{ money(series.item.spend) }}</strong>
            </div>
            <div class="stat-cell" :class="{ 'cell-green': (getPurchasesCount(series.item) || 0) > 0 }">
              <small>{{ l({ lo: 'ຍອດຊື້', th: 'ยอดซื้อ', en: 'Purchases' }) }}</small>
              <strong>{{ getPurchasesCount(series.item) || 0 }} ອໍເດີ</strong>
            </div>
            <div class="stat-cell">
              <small>CPA</small>
              <strong>{{ money(getCostPerPurchase(series.item)) }}</strong>
            </div>
            <div class="stat-cell">
              <small>CTR</small>
              <strong>{{ (series.item.linkCtr || 0).toFixed(2) }}%</strong>
            </div>
          </div>

          <!-- TODAY TIME-OF-DAY 4-SLOT BREAKDOWN & DIAGNOSTIC ASSISTANT -->
          <div v-if="props.isToday || (series.item.hourly && series.item.hourly.length > 0)" class="today-time-slots-card">
            <div class="today-slots-header cursor-pointer" @click="toggleSlotExpand(series.id)">
              <div class="d-flex align-center gap-1">
                <svg width="8" height="8" viewBox="0 0 8 8" class="pulse-today-dot mr-1"><circle cx="4" cy="4" r="4" fill="#ef4444"/></svg>
                <span class="today-slots-title">
                  {{ l({ lo: '🕒 ຊ່ວງເວລາປະຈຳວັນ (Time Slots)', th: '🕒 ช่วงเวลาประจำวัน (Time Slots)', en: '🕒 Time Slots' }) }}
                </span>
                <v-chip size="x-small" :color="isSlotExpanded(series.id) ? 'primary' : 'default'" variant="tonal" class="ml-1 font-weight-bold">
                  {{ isSlotExpanded(series.id) ? '▲ ຫຍໍ້ເຂົ້າ' : '▼ ເບິ່ງ 4 ຊ່ວງເວລາ & ແນະນຳ' }}
                </v-chip>
              </div>
              <span class="today-slots-sub d-none d-sm-inline">
                {{ l({ lo: 'ແຍກ 4 ຊ່ວງເວລາ ເພື່ອຮູ້ບ່ອນແກ້ໄຂມື້ນີ້', th: 'แยก 4 ช่วงเวลา เพื่อรู้จุดแก้ไขวันนี้', en: '4 dayparts to optimize' }) }}
              </span>
            </div>

            <v-expand-transition>
              <div v-show="isSlotExpanded(series.id)" class="pt-2">
                <!-- 4 Dayparts Horizontal Grid -->
                <div class="dayparts-grid">
                  <div
                    v-for="slot in getAdDayparts(series.item).slots"
                    :key="slot.key"
                    class="daypart-cell"
                    :class="{
                      'is-current-time': slot.isCurrentSlot,
                      'is-peak-orders': slot.isPeak,
                      'is-burning-slot': slot.spend > 2 && slot.purchases === 0
                    }"
                  >
                    <div class="daypart-time-header">
                      <div class="d-flex align-center gap-1">
                        <v-icon :icon="slot.icon" size="13" />
                        <span class="daypart-title">{{ slot.name }}</span>
                      </div>
                      <span v-if="slot.isCurrentSlot" class="now-badge">🔴 {{ l({ lo: 'ຕອນນີ້', th: 'ตอนนี้', en: 'Now' }) }}</span>
                      <span v-else-if="slot.isPeak" class="peak-badge">👑 Peak</span>
                    </div>
                    <div class="daypart-sub-range">{{ slot.sub }}</div>

                    <div class="daypart-stats">
                      <div class="dp-stat-item">
                        <span class="dp-k">{{ l({ lo: 'ງົບໃຊ້:', th: 'งบ:', en: 'Spend:' }) }}</span>
                        <span class="dp-v">{{ money(slot.spend) }}</span>
                      </div>
                      <div class="dp-stat-item">
                        <span class="dp-k">{{ l({ lo: 'ຍອດຊື້:', th: 'ซื้อ:', en: 'Orders:' }) }}</span>
                        <span class="dp-v font-weight-bold" :class="slot.purchases > 0 ? 'text-success' : 'text-medium-emphasis'">
                          {{ slot.purchases }} ອໍເດີ
                        </span>
                      </div>
                      <div class="dp-stat-item">
                        <span class="dp-k">CPA:</span>
                        <span class="dp-v">{{ money(slot.cpa) }}</span>
                      </div>
                      <div class="dp-stat-item">
                        <span class="dp-k">{{ l({ lo: 'ກຳໄລ:', th: 'กำไร:', en: 'Profit:' }) }}</span>
                        <span class="dp-v font-weight-bold" :class="slot.profit >= 0 ? 'text-success' : 'text-error'">
                          {{ slot.profit >= 0 ? '+' : '' }}{{ money(slot.profit) }}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                <!-- Actionable Diagnostic Assistant Box -->
                <div
                  class="today-diagnostic-box"
                  :class="'diag-' + getAdDayparts(series.item).diagnostic.type"
                >
                  <div class="d-flex align-start gap-2 flex-grow-1 min-w-0">
                    <div class="diag-icon-wrap">
                      <v-icon :icon="getAdDayparts(series.item).diagnostic.icon" size="18" />
                    </div>
                    <div class="diag-content min-w-0">
                      <h4 class="diag-title">{{ getAdDayparts(series.item).diagnostic.title }}</h4>
                      <p class="diag-msg">{{ getAdDayparts(series.item).diagnostic.message }}</p>
                    </div>
                  </div>
                  <div v-if="getAdDayparts(series.item).diagnostic.actionLabel" class="diag-action-wrap">
                    <v-btn
                      v-if="getAdDayparts(series.item).diagnostic.actionType === 'scale'"
                      size="x-small"
                      color="success"
                      variant="elevated"
                      class="font-weight-bold diag-btn"
                      @click="emit('scale', series.item)"
                    >
                      {{ getAdDayparts(series.item).diagnostic.actionLabel }}
                    </v-btn>
                    <v-btn
                      v-else-if="getAdDayparts(series.item).diagnostic.actionType === 'pause' && isItemActive(series.item)"
                      size="x-small"
                      color="error"
                      variant="tonal"
                      class="font-weight-bold diag-btn"
                      @click="emit('pause', series.item)"
                    >
                      {{ getAdDayparts(series.item).diagnostic.actionLabel }}
                    </v-btn>
                  </div>
                </div>
              </div>
            </v-expand-transition>
          </div>

          <!-- Dedicated Clean SVG Line Chart for this Ad -->
          <div class="solo-chart-canvas">
            <svg viewBox="0 0 520 135" class="solo-svg" preserveAspectRatio="none">
              <defs>
                <linearGradient :id="'grad-solo-' + series.id" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" :stop-color="series.color" :stop-opacity="series.isAngel ? 0.28 : 0.2" />
                  <stop offset="100%" :stop-color="series.color" stop-opacity="0.0" />
                </linearGradient>
              </defs>

              <!-- Subtle grid lines -->
              <line x1="40" y1="25" x2="480" y2="25" stroke="#f1f5f9" stroke-width="1" />
              <line x1="40" y1="65" x2="480" y2="65" stroke="#f1f5f9" stroke-width="1" />
              <line x1="40" y1="105" x2="480" y2="105" stroke="#e2e8f0" stroke-width="1" />

              <!-- Area Fill Under Curve -->
              <path
                :d="generateSingleAdPath(series, 520, 135).areaPath"
                :fill="'url(#grad-solo-' + series.id + ')'"
              />

              <!-- Line Path -->
              <path
                :d="generateSingleAdPath(series, 520, 135).linePath"
                fill="none"
                :stroke="series.color"
                stroke-width="2.8"
                stroke-linecap="round"
                :stroke-dasharray="series.isBad ? '5 3' : 'none'"
              />

              <!-- Dots and Printed Values on each day -->
              <g v-for="(pt, pIdx) in generateSingleAdPath(series, 520, 135).points" :key="pIdx">
                <!-- Dot circle -->
                <circle
                  :cx="pt.x"
                  :cy="pt.y"
                  r="4.5"
                  :fill="series.color"
                  stroke="#ffffff"
                  stroke-width="2"
                  class="solo-dot"
                />

                <!-- Printed value right above the dot -->
                <text
                  :x="pt.x"
                  :y="pt.y - 8"
                  text-anchor="middle"
                  :fill="series.color"
                  font-size=\"11\"
                  font-weight="800"
                >
                  {{ pt.label }}
                </text>

                <!-- Date below on bottom axis -->
                <text
                  :x="pt.x"
                  :y="126"
                  text-anchor="middle"
                  fill="#64748b"
                  font-size=\"11\"
                  font-weight="600"
                >
                  {{ pt.date.slice(5) }}
                </text>
              </g>
            </svg>
          </div>

          <!-- Day-by-Day Compact Numbers Strip -->
          <div class="solo-daily-table">
            <div
              v-for="(dp, dIdx) in series.dataPoints"
              :key="dIdx"
              class="d-col"
              :class="{
                'highlight-order': dp.purchases > 0,
                'score-high': dp.strengthScore >= 80,
                'score-low': dp.strengthScore < 40,
              }"
            >
              <span class="d-date-label">{{ dp.date.slice(5) }}</span>
              <span class="d-score-label">Score: <b>{{ dp.strengthScore }}</b></span>
              <span class="d-order-label">
                <b v-if="dp.purchases > 0" class="text-success">{{ dp.purchases }} ອໍເດີ</b>
                <span v-else class="text-medium-emphasis">0</span>
              </span>
              <span class="d-spend-label">{{ money(dp.spend) }}</span>
            </div>
          </div>

          <!-- Bottom Card Action Button -->
          <div class="card-bottom-actions">
            <v-btn
              v-if="series.isAngel"
              size="small"
              color="success"
              variant="flat"
              prepend-icon="mdi-rocket-launch"
              class="font-weight-bold"
              @click="emit('scale', series.item)"
            >
              Scale ງົບ (+20%)
            </v-btn>
            <v-btn
              v-else-if="series.isBad && isItemActive(series.item)"
              size="small"
              color="error"
              variant="tonal"
              prepend-icon="mdi-pause-circle-outline"
              class="font-weight-bold"
              @click="emit('pause', series.item)"
            >
              ⏸️ ປິດແອດ (Stop-loss)
            </v-btn>
            <span v-else-if="!isItemActive(series.item)" class="closed-label">
              <v-icon icon="mdi-pause-circle" size="14" class="mr-1 text-medium-emphasis" />
              ແອດນີ້ປິດການໃຊ້ງານແລ້ວ
            </span>
          </div>
        </article>
      </div>
    </div>

    <!-- ========================================================================= -->
    <!-- VIEW MODE 2: COMBINED MULTI-LINE CHART (ມີຊື່ຕິດທຸກເສັ້ນຊັດເຈນ) -->
    <!-- ========================================================================= -->
    <div v-else class="combined-view-container">
      <div class="chart-viewport" @mouseleave="hoveredPoint = null">
        <svg
          :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
          class="ad-svg-chart"
          preserveAspectRatio="none"
        >
          <defs>
            <linearGradient id="angelZoneGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="#10b981" stop-opacity="0.12" />
              <stop offset="100%" stop-color="#10b981" stop-opacity="0.0" />
            </linearGradient>
            <linearGradient id="badZoneGrad" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stop-color="#ef4444" stop-opacity="0.0" />
              <stop offset="100%" stop-color="#ef4444" stop-opacity="0.1" />
            </linearGradient>
          </defs>

          <!-- Background Threshold Zones (for Strength metric) -->
          <g v-if="selectedMetric === 'strength'">
            <!-- Angel Zone: Score 80 to 100 -->
            <rect
              :x="padding.left"
              :y="getY(100)"
              :width="innerWidth"
              :height="getY(80) - getY(100)"
              fill="url(#angelZoneGrad)"
            />
            <!-- Dying Zone: Score 0 to 40 -->
            <rect
              :x="padding.left"
              :y="getY(40)"
              :width="innerWidth"
              :height="getY(0) - getY(40)"
              fill="url(#badZoneGrad)"
            />

            <!-- Angel Threshold Line (80) -->
            <line
              :x1="padding.left"
              :y1="getY(80)"
              :x2="padding.left + innerWidth"
              :y2="getY(80)"
              stroke="#10b981"
              stroke-width="1.2"
              stroke-dasharray="4 4"
              opacity="0.7"
            />
            <text :x="padding.left + 8" :y="getY(80) - 4" fill="#059669" font-size=\"11\" font-weight="700">
              🟢 ເກນ Ads ນາງຟ້າ (80+)
            </text>

            <!-- Bad Threshold Line (40) -->
            <line
              :x1="padding.left"
              :y1="getY(40)"
              :x2="padding.left + innerWidth"
              :y2="getY(40)"
              stroke="#ef4444"
              stroke-width="1.2"
              stroke-dasharray="4 4"
              opacity="0.7"
            />
            <text :x="padding.left + 8" :y="getY(40) + 12" fill="#dc2626" font-size=\"11\" font-weight="700">
              🔴 ເກນໝົດກຳລັງ / ຄວນປິດ (&lt; 40)
            </text>
          </g>

          <!-- Horizontal Grid Lines -->
          <g class="grid-lines">
            <line
              v-for="tick in 5"
              :key="tick"
              :x1="padding.left"
              :y1="padding.top + (innerHeight / 4) * (tick - 1)"
              :x2="padding.left + innerWidth"
              :y2="padding.top + (innerHeight / 4) * (tick - 1)"
              stroke="#f1f5f9"
              stroke-width="1"
            />
          </g>

          <!-- Y-Axis Ticks & Labels -->
          <g class="y-axis-labels">
            <text
              v-for="tick in 5"
              :key="tick"
              :x="padding.left - 8"
              :y="padding.top + (innerHeight / 4) * (tick - 1) + 4"
              text-anchor="end"
              fill="#94a3b8"
              font-size=\"11\"
              font-weight="600"
            >
              {{ Math.round(yRange.max - ((yRange.max - yRange.min) / 4) * (tick - 1)) }}
            </text>
          </g>

          <!-- X-Axis Dates -->
          <g class="x-axis-labels">
            <text
              v-for="(d, idx) in dates"
              :key="d"
              :x="getX(idx)"
              :y="svgHeight - 12"
              text-anchor="middle"
              fill="#64748b"
              font-size="11"
              font-weight="600"
            >
              {{ d.slice(5) }}
            </text>
          </g>

          <!-- Ad Trajectory Lines with End-Labels -->
          <g v-for="series in visibleSeriesList" :key="series.id">
            <!-- Trajectory Line -->
            <path
              :d="generatePath(series)"
              fill="none"
              :stroke="series.color"
              :stroke-width="hoveredPoint?.adId === series.id ? 4.0 : (series.isAngel ? 3.0 : 2.2)"
              :stroke-dasharray="series.isBad ? '5 3' : 'none'"
              :opacity="hoveredPoint && hoveredPoint.adId !== series.id ? 0.2 : 0.95"
              class="ad-trend-line"
            />

            <!-- Points on the line -->
            <circle
              v-for="(pt, pIdx) in series.dataPoints"
              :key="pIdx"
              :cx="getX(pIdx)"
              :cy="getY(getMetricValue(pt))"
              :r="hoveredPoint?.adId === series.id && hoveredPoint?.date === pt.date ? 6.5 : 3.5"
              :fill="series.color"
              stroke="#ffffff"
              stroke-width="1.5"
              class="interactive-dot"
              @mouseenter="handlePointHover(series, pt, pIdx)"
            />

            <!-- CLEAR NAME BADGE AT THE END OF EACH LINE -->
            <g
              v-if="series.dataPoints.length > 0"
              class="line-end-badge-group"
              :class="{ 'is-spotlight': hoveredPoint?.adId === series.id }"
              @mouseenter="handlePointHover(series, series.dataPoints[series.dataPoints.length - 1]!, series.dataPoints.length - 1)"
            >
              <rect
                :x="getX(series.dataPoints.length - 1) + 8"
                :y="getY(getMetricValue(series.dataPoints[series.dataPoints.length - 1]!)) - 11"
                width="145"
                height="22"
                rx="6"
                :fill="series.color"
                :stroke="hoveredPoint?.adId === series.id ? '#ffffff' : 'none'"
                stroke-width="1.5"
                class="badge-pill-rect"
              />
              <text
                :x="getX(series.dataPoints.length - 1) + 14"
                :y="getY(getMetricValue(series.dataPoints[series.dataPoints.length - 1]!)) + 4"
                fill="#ffffff"
                font-size=\"11\"
                font-weight="700"
                class="badge-pill-text"
              >
                {{ series.name.length > 13 ? series.name.slice(0, 13) + '..' : series.name }}
                · {{ formatMetricDisplay(getMetricValue(series.dataPoints[series.dataPoints.length - 1]!)) }}
              </text>
            </g>
          </g>

          <!-- Vertical Hover Guide Line -->
          <line
            v-if="hoveredPoint"
            :x1="hoveredPoint.x"
            :y1="padding.top"
            :x2="hoveredPoint.x"
            :y2="svgHeight - padding.bottom"
            stroke="#94a3b8"
            stroke-width="1"
            stroke-dasharray="2 2"
          />
        </svg>

        <!-- Floating Interactive Tooltip -->
        <div
          v-if="hoveredPoint"
          class="chart-tooltip"
          :style="{
            left: `${(hoveredPoint.x / svgWidth) * 100}%`,
            top: `${(hoveredPoint.y / svgHeight) * 100}%`,
          }"
        >
          <div class="tooltip-header" :style="{ borderLeftColor: hoveredPoint.color }">
            <strong class="tooltip-ad-name">{{ hoveredPoint.adName }}</strong>
            <small class="tooltip-date">{{ hoveredPoint.date }}</small>
          </div>
          <div class="tooltip-metric-val">
            <span>ຄ່າ {{ selectedMetric.toUpperCase() }}:</span>
            <strong :style="{ color: hoveredPoint.color }">{{ hoveredPoint.displayValue }}</strong>
          </div>
          <div class="tooltip-strength-row">
            <span class="score-chip" :style="{ background: hoveredPoint.strengthScore >= 80 ? '#ecfdf5' : hoveredPoint.strengthScore < 40 ? '#fef2f2' : '#fffbeb' }">
              {{ hoveredPoint.strengthLabel }} ({{ hoveredPoint.strengthScore }}/100)
            </span>
          </div>
          <div class="tooltip-footer">
            <span>🛒 ອໍເດີ: {{ hoveredPoint.purchases }}</span>
            <span>🎯 CPA: {{ hoveredPoint.cpa ? `$${hoveredPoint.cpa.toFixed(2)}` : '—' }}</span>
            <span>📈 CTR: {{ hoveredPoint.ctr }}%</span>
          </div>
        </div>
      </div>

      <!-- Interactive Legend Chips with Quick Open/Close Status -->
      <div class="chart-legend">
        <span class="legend-hint">{{ l({ lo: 'ກົດໃສ່ປ້າຍເພື່ອ ເປີດ/ປິດ ເສັ້ນກຣາຟຂອງແຕ່ລະແອດ:', th: 'แตะที่ป้ายเพื่อ เปิด/ปิด เส้นกราฟของแต่ละแอด:', en: 'Click label to toggle line:' }) }}</span>
        <div class="legend-chips">
          <div
            v-for="series in activeAdSeries"
            :key="series.id"
            class="legend-chip-card"
            :class="{ active: series.visible, faded: !series.visible }"
          >
            <button
              type="button"
              class="chip-toggle-btn"
              @click="series.visible = !series.visible"
            >
              <span class="semantic-dot" :class="series.isAngel ? 'bg-green' : series.isBad ? 'bg-red' : 'bg-blue'" />
              <span class="legend-ad-name">{{ series.name }}</span>
              <span class="legend-score-pill" :style="{ color: series.color }">
                {{ series.currentScore }}
              </span>
            </button>
            <div class="chip-status-badge" :class="isItemActive(series.item) ? 'active' : 'paused'">
              {{ isItemActive(series.item) ? '🟢' : '⏸️' }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Momentum Summary Bar (Top Winner & Dying Alert) -->
    <div class="momentum-insights-grid">
      <div v-if="strongestAd" class="insight-box winner">
        <div class="box-icon-circle green">
          <v-icon icon="mdi-rocket-launch" size="20" />
        </div>
        <div class="box-info">
          <small>{{ l({ lo: '🟢 ແອດທີ່ແຂງແກ່ນທີ່ສຸດ (Top Momentum)', th: '🟢 แอดที่แข็งแกร่งที่สุด (Top Momentum)', en: 'Strongest Momentum' }) }}</small>
          <strong>{{ strongestAd.name }}</strong>
          <span class="insight-badge green">
            Score {{ strongestAd.currentScore }}/100 · {{ getStrengthLabel(strongestAd.currentScore).label }}
          </span>
        </div>
      </div>

      <div v-if="weakestAd && weakestAd.currentScore < 50" class="insight-box loser">
        <div class="box-icon-circle red">
          <v-icon icon="mdi-alert-octagon" size="20" />
        </div>
        <div class="box-info">
          <small>{{ l({ lo: '🔴 ແອດທີ່ເລີ່ມໝົດກຳລັງ (Fatigue Alert)', th: '🔴 แอดที่เริ่มหมดกำลัง (Fatigue Alert)', en: 'Fatigue Alert' }) }}</small>
          <strong>{{ weakestAd.name }}</strong>
          <span class="insight-badge red">
            Score {{ weakestAd.currentScore }}/100 · ຄວນພິຈາລະນາປິດ
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ad-strength-chart-card {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 16px;
  padding: 20px 24px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.chart-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
}

.chart-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.5px;
  color: #0284c7;
  background: #e0f2fe;
  padding: 4px 8px;
  line-height: 1.4;
  border-radius: 6px;
}

.color-guide-hint {
  font-size: 12px;
  font-weight: 700;
  color: #475569;
  display: inline-flex;
  align-items: center;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  padding: 4px 10px;
  line-height: 1.4;
  border-radius: 6px;
}

.chart-title-area h2 {
  font-size: clamp(16px, 1.8vw, 20px);
  font-weight: 800;
  color: var(--color-text, #0f172a);
  margin: 4px 0;
}

.chart-sub {
  font-size: 13px;
  color: #64748b;
  margin: 0;
  max-width: 760px;
}

.header-control-cluster {
  display: flex;
  flex-direction: column;
  gap: 10px;
  align-items: flex-end;
}

.mode-toggle-group {
  background: #f1f5f9;
  border-radius: 8px;
  overflow: hidden;
  padding: 4px;
}

.metric-toggle-group {
  background: #f8fafc;
  padding: 4px;
}

/* Preset Filters & Spotlight */
.chart-ad-filters {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 10px;
  border: 1px solid #e2e8f0;
}

.filter-label {
  font-size: 12px;
  font-weight: 700;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 4px;
}

.filter-pills {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.pill-btn {
  font-size: 12px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 999px;
  border: 1px solid #cbd5e1;
  background: #ffffff;
  color: #475569;
  cursor: pointer;
  transition: all 0.2s;
}

.pill-btn:hover {
  background: #f1f5f9;
}

.pill-btn.angel {
  border-color: #86efac;
  color: #15803d;
  background: #f0fdf4;
}

.pill-btn.bad {
  border-color: #fca5a5;
  color: #b91c1c;
  background: #fef2f2;
}

.pill-btn.top3 {
  border-color: #bae6fd;
  color: #0369a1;
  background: #f0f9ff;
}

.pill-btn.active {
  background: #1e293b;
  color: #ffffff;
  border-color: #0f172a;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.15);
}

.pill-btn.angel.active {
  background: #16a34a;
  color: #ffffff;
  border-color: #15803d;
}

.pill-btn.bad.active {
  background: #dc2626;
  color: #ffffff;
  border-color: #b91c1c;
}

.pill-btn.top3.active {
  background: #0284c7;
  color: #ffffff;
  border-color: #0369a1;
}

.ad-spotlight-select {
  display: flex;
  align-items: center;
  gap: 6px;
}

.ad-spotlight-select label {
  font-size: 12px;
  font-weight: 700;
  color: #475569;
}

.ad-select-dropdown {
  font-size: 12px;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 6px;
  border: 1px solid #cbd5e1;
  background: #ffffff;
  color: #0f172a;
  max-width: 240px;
  cursor: pointer;
}

/* ========================================================================= */
/* SEPARATE INDIVIDUAL AD CARDS STYLING */
/* ========================================================================= */
.separate-view-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.separate-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(440px, 1fr));
  gap: 16px;
}

@media (max-width: 640px) {
  .separate-grid {
    grid-template-columns: 1fr;
  }
}

.ad-solo-card {
  background: #ffffff;
  border: 1.5px solid #e2e8f0;
  border-radius: 14px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.03);
  transition: all 0.2s ease-in-out;
}

.ad-solo-card:hover {
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.06);
}

/* Strict Semantic Themes */
.ad-solo-card.theme-angel {
  border-color: #10b981;
  background: linear-gradient(180deg, #f0fdf4 0%, #ffffff 42px);
}

.ad-solo-card.theme-bad {
  border-color: #ef4444;
  background: linear-gradient(180deg, #fef2f2 0%, #ffffff 42px);
}

.ad-solo-card.theme-neutral {
  border-color: #cbd5e1;
  background: linear-gradient(180deg, #f8fafc 0%, #ffffff 42px);
}

.ad-solo-card.is-paused-card {
  opacity: 0.82;
  filter: grayscale(0.2);
}

/* Top Control Bar */
.solo-top-control-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.semantic-pill {
  font-size: 12px;
  font-weight: 800;
  padding: 3px 8px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
}

.semantic-pill.angel {
  background: #dcfce7;
  color: #15803d;
  border: 1px solid #86efac;
}

.semantic-pill.bad {
  background: #fee2e2;
  color: #b91c1c;
  border: 1px solid #fca5a5;
}

.semantic-pill.neutral {
  background: #f1f5f9;
  color: #475569;
  border: 1px solid #cbd5e1;
}

.trajectory-pill {
  font-size: 12px;
  font-weight: 700;
  padding: 3px 6px;
  border-radius: 4px;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.trajectory-pill.trend-up {
  background: #dcfce7;
  color: #15803d;
}

.trajectory-pill.trend-down {
  background: #fee2e2;
  color: #b91c1c;
}

.trajectory-pill.trend-flat {
  background: #f8fafc;
  color: #64748b;
}

.score-badge-clear {
  font-size: 12px;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: 5px;
}

.score-badge-clear.score-green {
  background: #dcfce7;
  color: #15803d;
}

.score-badge-clear.score-red {
  background: #fee2e2;
  color: #b91c1c;
}

.score-badge-clear.score-blue {
  background: #eff6ff;
  color: #1d4ed8;
}

/* OPEN / CLOSE SWITCH CONTAINER */
.ad-switch-container {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  border-radius: 8px;
  border: 1px solid;
  transition: all 0.2s ease;
}

.ad-switch-container.status-active-box {
  background: #f0fdf4;
  border-color: #86efac;
}

.ad-switch-container.status-paused-box {
  background: #fef2f2;
  border-color: #fca5a5;
}

.switch-text {
  font-size: 12px;
  font-weight: 800;
}

.status-active-box .switch-text {
  color: #15803d;
}

.status-paused-box .switch-text {
  color: #b91c1c;
}

.compact-ad-switch {
  transform: scale(0.85);
  transform-origin: right center;
}

/* Identity Block */
.solo-identity-block {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.ad-clean-name {
  font-size: 15px;
  font-weight: 800;
  color: #0f172a;
  margin: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  line-height: 1.35;
  word-break: break-word;
}

.semantic-dot {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  display: inline-block;
  flex-shrink: 0;
}

.bg-green { background-color: #10b981; }
.bg-red { background-color: #ef4444; }
.bg-blue { background-color: #3b82f6; }

.ad-sub-id code {
  font-size: 12px;
  background: #f1f5f9;
  color: #64748b;
  padding: 1px 6px;
  border-radius: 4px;
}

/* 4 Metric Clean Grid */
.metric-clean-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
  background: #f8fafc;
  padding: 8px;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
}

.stat-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.stat-cell small {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
}

.stat-cell strong {
  font-size: 12.5px;
  font-weight: 800;
  color: #0f172a;
}

.stat-cell.cell-green strong {
  color: #15803d;
}

/* Solo SVG Chart */
.solo-chart-canvas {
  width: 100%;
  height: 135px;
  background: #fafafa;
  border-radius: 8px;
  border: 1px solid #f1f5f9;
  overflow: hidden;
  position: relative;
  padding: 4px;
}

.solo-svg {
  width: 100%;
  height: 100%;
  display: block;
}

.solo-dot {
  cursor: pointer;
  transition: r 0.15s ease;
}

.solo-dot:hover {
  r: 6px;
}

/* Solo Daily Table Strip */
.solo-daily-table {
  display: grid;
  grid-auto-flow: column;
  grid-auto-columns: minmax(58px, 1fr);
  gap: 4px;
  overflow-x: auto;
  padding-bottom: 2px;
}

.d-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 6px 6px;
  background: #f8fafc;
  border-radius: 6px;
  border: 1px solid #f1f5f9;
  font-size: 12px;
  line-height: 1.4;
  text-align: center;
}

.d-col.highlight-order {
  background: #ecfdf5;
  border-color: #a7f3d0;
}

.d-date-label {
  font-weight: 700;
  color: #64748b;
}

.d-score-label {
  color: #475569;
}

.d-score-label b {
  color: #0f172a;
}

.d-order-label {
  font-size: 12px;
}

.d-spend-label {
  font-size: 12px;
  color: #94a3b8;
}

.card-bottom-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding-top: 4px;
}

.closed-label {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
  display: inline-flex;
  align-items: center;
}

/* ========================================================================= */
/* COMBINED MULTI-LINE VIEW STYLING */
/* ========================================================================= */
.combined-view-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.chart-viewport {
  position: relative;
  width: 100%;
  overflow-x: auto;
}

.ad-svg-chart {
  width: 100%;
  height: auto;
  min-height: 290px;
  display: block;
}

.ad-trend-line {
  transition: stroke-width 0.2s ease, opacity 0.2s ease;
}

.interactive-dot {
  cursor: pointer;
  transition: r 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.line-end-badge-group {
  cursor: pointer;
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.line-end-badge-group.is-spotlight {
  filter: drop-shadow(0 2px 6px rgba(0, 0, 0, 0.2));
}

.badge-pill-rect {
  transition: all 0.2s ease;
}

/* Floating Tooltip */
.chart-tooltip {
  position: absolute;
  pointer-events: none;
  background: #0f172a;
  color: #ffffff;
  padding: 10px 12px;
  border-radius: 10px;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.3);
  font-size: 12px;
  z-index: 20;
  transform: translate(-50%, -115%);
  min-width: 190px;
  transition: left 0.1s ease, top 0.1s ease;
}

.tooltip-header {
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
  padding-bottom: 6px;
  margin-bottom: 6px;
}

.tooltip-ad-name {
  display: block;
  font-size: 12px;
  font-weight: 700;
  color: #ffffff;
}

.tooltip-date {
  font-size: 12px;
  color: #94a3b8;
}

.tooltip-metric-val {
  display: flex;
  justify-content: space-between;
  margin-bottom: 4px;
}

.tooltip-strength-row {
  margin-bottom: 6px;
}

.score-chip {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 700;
  color: #0f172a;
}

.tooltip-footer {
  display: flex;
  justify-content: space-between;
  border-top: 1px solid #334155;
  padding-top: 4px;
  font-size: 12px;
  color: #cbd5e1;
}

/* Legend with quick status */
.chart-legend {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: #f8fafc;
  padding: 12px;
  border-radius: 12px;
}

.legend-hint {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
}

.legend-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.legend-chip-card {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 4px 8px;
  line-height: 1.4;
  transition: all 0.2s ease;
}

.legend-chip-card:hover {
  border-color: #cbd5e1;
  background: #f1f5f9;
}

.chip-toggle-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  background: transparent;
  cursor: pointer;
  font-size: 12px;
  font-weight: 600;
  color: #1e293b;
  padding: 2px 4px;
}

.chip-status-badge {
  font-size: 12px;
}

.legend-chip-card.faded {
  opacity: 0.35;
  text-decoration: line-through;
}

.legend-score-pill {
  font-size: 12px;
  font-weight: 800;
  background: #f1f5f9;
  padding: 1px 5px;
  border-radius: 4px;
}

/* Momentum Insights Grid */
.momentum-insights-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 12px;
  margin-top: 4px;
}

.insight-box {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  border: 1px solid;
}

.insight-box.winner {
  background: #f0fdf4;
  border-color: #bbf7d0;
}

.insight-box.loser {
  background: #fef2f2;
  border-color: #fecaca;
}

.box-icon-circle {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.box-icon-circle.green {
  background: #dcfce7;
  color: #16a34a;
}

.box-icon-circle.red {
  background: #fee2e2;
  color: #dc2626;
}

.box-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.box-info small {
  font-size: 12px;
  font-weight: 700;
  color: #64748b;
}

.box-info strong {
  font-size: 13px;
  font-weight: 800;
  color: #0f172a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.insight-badge {
  font-size: 12px;
  font-weight: 700;
  margin-top: 2px;
}

.insight-badge.green {
  color: #16a34a;
}

.insight-badge.red {
  color: #dc2626;
}

/* ========================================================================= */
/* PRODUCT FINANCIALS & UNIT MARGIN CARD                                      */
/* ========================================================================= */
.ad-product-financial-card {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 2px;
  transition: all 0.2s ease;
}

.theme-angel .ad-product-financial-card {
  background: #f0fdf4;
  border-color: #bbf7d0;
}

.theme-bad .ad-product-financial-card {
  background: #fef2f2;
  border-color: #fecaca;
}

.product-header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  flex-wrap: wrap;
}

.product-thumb-mini {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  object-fit: cover;
  border: 1px solid #cbd5e1;
  flex-shrink: 0;
}

.product-thumb-placeholder {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: #e2e8f0;
  color: #64748b;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.product-name-badge {
  font-size: 12px;
  font-weight: 700;
  color: #1e293b;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 170px;
}

.product-change-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  color: #94a3b8;
  padding: 2px;
  border-radius: 4px;
  display: flex;
  align-items: center;
}

.product-change-btn:hover {
  color: #2563eb;
  background: #eff6ff;
}

.net-profit-badge {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  padding: 4px 10px;
  border-radius: 8px;
  border: 1px solid;
}

.net-profit-badge small {
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.net-profit-badge.profit-positive {
  background: #ecfdf5;
  border-color: #a7f3d0;
  color: #065f46;
}

.net-profit-badge.profit-positive .profit-number {
  font-size: 15px;
  font-weight: 900;
  color: #059669;
}

.net-profit-badge.profit-negative {
  background: #fef2f2;
  border-color: #fecaca;
  color: #991b1b;
}

.net-profit-badge.profit-negative .profit-number {
  font-size: 15px;
  font-weight: 900;
  color: #dc2626;
}

.product-pricing-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
  background: #ffffff;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid #edf2f7;
}

@media (max-width: 600px) {
  .product-pricing-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

.price-cell {
  display: flex;
  flex-direction: column;
}

.price-cell .p-label {
  font-size: 12px;
  font-weight: 600;
  color: #64748b;
}

.price-cell .p-val {
  font-size: 12px;
  font-weight: 700;
  color: #0f172a;
}

.margin-danger-alert {
  display: flex;
  align-items: center;
  background: #fee2e2;
  color: #991b1b;
  font-size: 12px;
  padding: 6px 10px;
  border-radius: 6px;
  border: 1px solid #fca5a5;
}

/* ========================================================================= */
/* TODAY TIME-OF-DAY 4-SLOT BREAKDOWN & DIAGNOSTIC ASSISTANT                 */
/* ========================================================================= */
.today-time-slots-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.today-slots-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 6px;
}

.today-slots-header.cursor-pointer {
  cursor: pointer;
  user-select: none;
  transition: opacity 0.15s;
}

.today-slots-header.cursor-pointer:hover {
  opacity: 0.82;
}

.pulse-today-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  animation: pulse-dot 1.8s infinite;
  flex-shrink: 0;
}

@keyframes pulse-dot {
  0% { transform: scale(0.9); opacity: 0.8; }
  50% { transform: scale(1.3); opacity: 1; }
  100% { transform: scale(0.9); opacity: 0.8; }
}

.today-slots-title {
  font-size: 12px;
  font-weight: 800;
  color: #0f172a;
}

.today-slots-sub {
  font-size: 12px;
  color: #64748b;
}

.dayparts-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

@media (max-width: 600px) {
  .dayparts-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

.daypart-cell {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 6px 8px;
  display: flex;
  flex-direction: column;
  gap: 3px;
  position: relative;
  transition: all 0.2s ease;
}

.daypart-cell.is-current-time {
  background: #eff6ff;
  border-color: #93c5fd;
  box-shadow: 0 0 0 1px #3b82f6;
}

.daypart-cell.is-peak-orders {
  background: #f0fdf4;
  border-color: #86efac;
}

.daypart-cell.is-burning-slot {
  background: #fff1f2;
  border-color: #fecdd3;
}

.daypart-time-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.daypart-title {
  font-size: 12px;
  font-weight: 800;
  color: #1e293b;
}

.now-badge {
  font-size: 12px;
  font-weight: 800;
  color: #2563eb;
  background: #dbeafe;
  padding: 1px 5px;
  border-radius: 4px;
}

.peak-badge {
  font-size: 12px;
  font-weight: 800;
  color: #166534;
  background: #dcfce7;
  padding: 1px 5px;
  border-radius: 4px;
}

.daypart-sub-range {
  font-size: 12px;
  color: #64748b;
  margin-bottom: 2px;
}

.daypart-stats {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.dp-stat-item {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
}

.dp-k {
  color: #64748b;
}

.dp-v {
  color: #0f172a;
}

/* Diagnostic Box */
.today-diagnostic-box {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid;
  margin-top: 2px;
}

.today-diagnostic-box.diag-success {
  background: #ecfdf5;
  border-color: #a7f3d0;
  color: #065f46;
}

.today-diagnostic-box.diag-warning {
  background: #fffbeb;
  border-color: #fde68a;
  color: #92400e;
}

.today-diagnostic-box.diag-danger {
  background: #fef2f2;
  border-color: #fecaca;
  color: #991b1b;
}

.today-diagnostic-box.diag-info {
  background: #f0f9ff;
  border-color: #bae6fd;
  color: #075985;
}

.diag-icon-wrap {
  flex-shrink: 0;
  margin-top: 2px;
}

.diag-title {
  font-size: 12px;
  font-weight: 800;
  margin-bottom: 1px;
}

.diag-msg {
  font-size: 12px;
  line-height: 1.35;
  margin: 0;
  opacity: 0.95;
}

.diag-action-wrap {
  flex-shrink: 0;
}

.diag-btn {
  text-transform: none;
  letter-spacing: normal;
}

/* Total Ad Spend Overview Strip */
.ad-total-spend-strip {
  background: linear-gradient(135deg, #0f172a 0%, #1e293b 100%);
  border-radius: 14px;
  padding: 18px 22px;
  color: #ffffff;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
  box-shadow: 0 4px 16px rgba(15, 23, 42, 0.08);
  margin-bottom: 6px;
}

.spend-overview-main {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 250px;
}

.spend-badge-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.spend-main-badge {
  font-size: 12px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  color: #38bdf8;
  display: inline-flex;
  align-items: center;
}

.spend-count-pill {
  font-size: 12px;
  font-weight: 700;
  background: rgba(255, 255, 255, 0.15);
  color: #e2e8f0;
  padding: 1px 7px;
  border-radius: 10px;
}

.spend-big-amount {
  font-size: 32px;
  font-weight: 900;
  color: #ffffff;
  letter-spacing: -0.5px;
  line-height: 1.1;
}

.spend-distribution-pills {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.dist-pill {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.08);
}

.dist-pill.green {
  color: #6ee7b7;
}

.dist-pill.red {
  color: #fca5a5;
}

.dist-pill.blue {
  color: #93c5fd;
}

.spend-metrics-cluster {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 10px;
  flex-grow: 1;
  max-width: 600px;
}

@media (max-width: 800px) {
  .spend-metrics-cluster {
    grid-template-columns: repeat(2, 1fr);
    max-width: 100%;
    width: 100%;
  }
}

.cluster-cell {
  background: rgba(255, 255, 255, 0.07);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  padding: 9px 12px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.cluster-cell small {
  font-size: 12px;
  color: #94a3b8;
  font-weight: 600;
}

.cluster-cell strong {
  font-size: 14.5px;
  font-weight: 800;
  color: #f8fafc;
}

.cluster-cell.profit-cell.profit-pos {
  background: rgba(16, 185, 129, 0.15);
  border-color: rgba(16, 185, 129, 0.35);
}

.cluster-cell.profit-cell.profit-pos strong {
  color: #34d399;
}

.cluster-cell.profit-cell.profit-neg {
  background: rgba(239, 68, 68, 0.15);
  border-color: rgba(239, 68, 68, 0.35);
}

.cluster-cell.profit-cell.profit-neg strong {
  color: #f87171;
}
</style>
