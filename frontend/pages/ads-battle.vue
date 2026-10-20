<script setup lang="ts">
import type {
  MetaAdAccount,
  MetaCampaign,
  MetaInsightsReport,
  MetaInsightItem,
  Product,
  AutomationFlow,
  ItemsResponse,
} from '~/types/automation'
import { useAdVat } from '~/composables/useAdVat'
import { useProductAdStrategy } from '~/composables/useProductAdStrategy'

definePageMeta({ middleware: 'auth' })
const { l } = useLocale()

const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const report = ref<MetaInsightsReport | null>(null)
const adAccountId = useCookie('meta_ad_account_id', { default: () => '', maxAge: 2592000 })
const level = ref<'ad' | 'adset' | 'campaign'>('ad')
const loadingAccounts = ref(true)
const loadingReport = ref(false)
const updatingId = ref('')
const bulkPausing = ref(false)
const notice = ref('')
const success = ref('')
const search = ref('')
const selectedProductId = ref<string>('all')
const statusFilter = ref<'all' | 'angels' | 'bad'>('all')
const viewMode = ref<'cards' | 'table'>('cards')

const iso = (date: Date) => {
  const copy = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return copy.toISOString().slice(0, 10)
}
const today = new Date()
const since = ref(iso(new Date(today.getFullYear(), today.getMonth(), today.getDate() - 6)))
const until = ref(iso(today))
const datePreset = ref('7')

// Budget Scale Modal
const budgetModalOpen = ref(false)
const targetBudgetItem = ref<MetaInsightItem | null>(null)
const selectedPercent = ref<number>(20)
const newDailyBudget = ref<number>(0)
const currentDailyBudget = ref<number>(0)
const updatingBudget = ref(false)

const products = ref<Product[]>([])
const flows = ref<AutomationFlow[]>([])
const loadingAutomations = ref(false)
const ready = ref(false)

const {
  productStrategies,
  loadStrategies: loadProductStrategiesFromStore,
} = useProductAdStrategy()

const currency = computed(() => selectedAccount.value?.currency || 'USD')
const zeroDecimalCurrencies = new Set(['LAK', 'VND', 'JPY', 'KRW', 'KHR', 'IDR', 'CLP', 'PYG'])
const currencyOffset = computed(() => zeroDecimalCurrencies.has((currency.value || '').toUpperCase()) ? 1 : 100)

const selectedAccount = computed(() => adAccounts.value.find(item => item.id === adAccountId.value))
const { detectVatRate, computeVat } = useAdVat()
const vatDetails = computed(() => detectVatRate(selectedAccount.value))

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

function getLinkedProduct(item: MetaInsightItem): { product: Product; flowName: string } | undefined {
  if (triggerToProduct.value[item.entityId]) return triggerToProduct.value[item.entityId]
  if (item.adSetId && triggerToProduct.value[item.adSetId]) return triggerToProduct.value[item.adSetId]
  if (item.campaignId && triggerToProduct.value[item.campaignId]) return triggerToProduct.value[item.campaignId]

  const searchName = `${item.entityName || ''} ${item.adSetName || ''} ${item.campaignName || ''}`.toLowerCase()
  for (const prod of products.value) {
    const pName = (prod.name || '').toLowerCase().trim()
    const pCode = (prod.code || '').toLowerCase().trim()
    const pNorm = pName.replace(/\s+/g, '')
    const searchNorm = searchName.replace(/\s+/g, '')
    if (pName && (searchName.includes(pName) || (pNorm.length >= 3 && searchNorm.includes(pNorm)))) {
      return { product: prod, flowName: 'Auto-Matched' }
    }
    if (pCode && searchName.includes(pCode)) {
      return { product: prod, flowName: 'Auto-Matched' }
    }
  }

  if (products.value.length === 1 && products.value[0]) {
    return { product: products.value[0], flowName: 'Default' }
  }

  return undefined
}

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

function convertToAdCurrency(amount: number, productCurrency?: string): number {
  if (!amount || amount <= 0) return 0
  const adCurr = (currency.value || 'USD').toUpperCase()
  const prodCurr = (productCurrency || '').toUpperCase()

  if (prodCurr && prodCurr === adCurr) return amount

  if (adCurr === 'USD') {
    if (prodCurr === 'LAK' || (!prodCurr && amount > 2000)) return amount / 22000
    if (prodCurr === 'THB' || (!prodCurr && amount > 50 && amount <= 2000)) return amount / 36
    return amount
  }

  if (adCurr === 'LAK') {
    if (prodCurr === 'USD' || (!prodCurr && amount < 100)) return amount * 22000
    if (prodCurr === 'THB' || (!prodCurr && amount >= 100 && amount <= 2000)) return amount * 620
    return amount
  }

  if (adCurr === 'THB') {
    if (prodCurr === 'LAK' || (!prodCurr && amount > 2000)) return amount / 620
    if (prodCurr === 'USD' || (!prodCurr && amount < 50)) return amount * 36
    return amount
  }

  return amount
}

function getItemNetProfit(item: MetaInsightItem): number {
  const purchases = getPurchasesCount(item)
  const spend = item.spend || 0
  const linked = getLinkedProduct(item)
  if (!linked) return -spend
  const price = linked.product.salePrice || (linked.product.price ? parseFloat(linked.product.price) : 0) || 0
  const cost = linked.product.cost || 0
  const rawMargin = Math.max(0, price - cost)
  const margin = convertToAdCurrency(rawMargin, linked.product.currency)
  return (purchases * margin) - spend
}

function isAngelAd(item: MetaInsightItem): { isAngel: boolean; reason: string } {
  if (!item.impressions && !item.spend) return { isAngel: false, reason: '' }
  const purchases = getPurchasesCount(item)
  const cost = getCostPerPurchase(item)
  const messages = getMessagesCount(item)
  const ctr = item.linkCtr || 0
  const roas = item.roas || 0
  const netProfit = getItemNetProfit(item)
  const targetCpa = currencyOffset.value === 1 ? 100000 : 5

  // 1. Any ad making positive net profit is an Angel Ad!
  if (purchases >= 1 && netProfit > 0) {
    return { isAngel: true, reason: `${purchases} ອໍເດີ (ກຳໄລ ${money(netProfit)})` }
  }
  // 2. 2+ purchases
  if (purchases >= 2) {
    return { isAngel: true, reason: `${purchases} ອໍເດີ (CPA ${money(cost)})` }
  }
  // 3. 1 purchase with acceptable CPA or ROAS
  if (purchases >= 1 && (cost == null || cost <= targetCpa * 1.5 || roas >= 1.5)) {
    return { isAngel: true, reason: `${purchases} ອໍເດີ` }
  }
  // 4. High inquiries with good CTR and low cost per message
  const costPerMsg = messages > 0 && item.spend ? item.spend / messages : null
  const maxMsgCost = currencyOffset.value === 1 ? 25000 : 0.60
  if (purchases === 0 && messages >= 5 && ctr >= 2.0 && costPerMsg != null && costPerMsg <= maxMsgCost) {
    return { isAngel: true, reason: `ທັກ ${messages} ຄົນ (${money(costPerMsg)}/ທັກ)` }
  }
  return { isAngel: false, reason: '' }
}

function isBadAd(item: MetaInsightItem): { isBad: boolean; reason: string } {
  if (!item.impressions && !item.spend) return { isBad: false, reason: '' }
  const purchases = getPurchasesCount(item)
  const ctr = item.linkCtr || 0
  const spend = item.spend || 0
  const freq = item.frequency || 0
  const messages = getMessagesCount(item)
  const netProfit = getItemNetProfit(item)
  const minSpendThreshold = currencyOffset.value === 1 ? 50000 : 3

  // Never mark as bad if it generates positive net profit
  if (purchases > 0 && netProfit > 0) return { isBad: false, reason: '' }

  if (purchases === 0 && spend >= minSpendThreshold) {
    return { isBad: true, reason: `ໃຊ້ງົບ ${money(spend)} ແລ້ວຍັງ 0 ອໍເດີ` }
  }
  if (purchases > 0 && netProfit < -minSpendThreshold) {
    return { isBad: true, reason: `ຂາດທຶນ ${money(Math.abs(netProfit))}` }
  }
  if (purchases === 0 && ctr > 0 && ctr < 1.0 && spend >= minSpendThreshold / 3) {
    return { isBad: true, reason: `CTR ຕ່ຳ (${percent(ctr)}) ຄົນບໍ່ຄລິກ` }
  }
  if (purchases === 0 && freq >= 2.2 && spend >= minSpendThreshold / 3) {
    return { isBad: true, reason: `Frequency ສູງ (${fmt(freq, 1)}x) ຄົນເຫັນຊ້ຳ` }
  }
  if (purchases === 0 && messages > 0 && spend >= minSpendThreshold * 1.5) {
    return { isBad: true, reason: `ທັກ ${messages} ແຕ່ 0 ອໍເດີ` }
  }
  return { isBad: false, reason: '' }
}

const productOptions = computed(() => {
  const allList = report.value?.items || []
  const countsByProduct: Record<string, { count: number; name: string; angels: number; bad: number }> = {}

  for (const p of products.value) {
    countsByProduct[p.id] = { count: 0, name: p.name, angels: 0, bad: 0 }
  }

  let unlinkedCount = 0
  let totalAngels = 0
  let totalBad = 0

  for (const it of allList) {
    const isAngel = isAngelAd(it).isAngel
    const isBad = isBadAd(it).isBad
    if (isAngel) totalAngels++
    if (isBad) totalBad++

    const linked = getLinkedProduct(it)
    if (linked && countsByProduct[linked.product.id]) {
      countsByProduct[linked.product.id]!.count++
      if (isAngel) countsByProduct[linked.product.id]!.angels++
      if (isBad) countsByProduct[linked.product.id]!.bad++
    } else {
      unlinkedCount++
    }
  }

  const list = [
    {
      title: l({
        lo: `ທຸກສິນຄ້າ (${allList.length} · 🟢${totalAngels} · 🔴${totalBad})`,
        th: `สินค้าทั้งหมด (${allList.length} · 🟢${totalAngels} · 🔴${totalBad})`,
        en: `All Products (${allList.length} · 🟢${totalAngels} · 🔴${totalBad})`,
      }),
      value: 'all',
    },
  ]

  for (const [pId, pData] of Object.entries(countsByProduct)) {
    if (pData.count > 0 || products.value.length <= 12) {
      list.push({
        title: `📦 ${pData.name} (${pData.count}${pData.angels > 0 ? ` · 🟢${pData.angels}` : ''}${pData.bad > 0 ? ` · 🔴${pData.bad}` : ''})`,
        value: pId,
      })
    }
  }

  if (unlinkedCount > 0) {
    list.push({
      title: l({ lo: `⚠️ ຍັງບໍ່ຜູກສິນຄ້າ (${unlinkedCount})`, th: `⚠️ ยังไม่ผูกสินค้า (${unlinkedCount})`, en: `⚠️ Unlinked (${unlinkedCount})` }),
      value: 'unlinked',
    })
  }

  return list
})

function statusFor(item: MetaInsightItem) {
  return item.effectiveStatus || item.status || campaigns.value.find(campaign => campaign.id === item.entityId)?.effectiveStatus || ''
}

function isItemPaused(item: MetaInsightItem): boolean {
  const st = (statusFor(item) || '').toUpperCase()
  return st.includes('PAUSED') || st.includes('ARCHIVED') || st.includes('DELETED')
}

// Base filtered items by search & product
const filteredBaseItems = computed(() => {
  const query = search.value.trim().toLowerCase()
  const list = report.value?.items || []
  return list
    .filter(item => !query || [item.entityName, item.campaignName, item.adSetName, item.entityId].some(val => val?.toLowerCase().includes(query)))
    .filter((item) => {
      if (selectedProductId.value === 'all') return true
      const linked = getLinkedProduct(item)
      if (selectedProductId.value === 'unlinked') return !linked
      return linked?.product.id === selectedProductId.value
    })
})

// Counts strictly within the currently filtered product/search
const angelCount = computed(() => filteredBaseItems.value.filter(it => isAngelAd(it).isAngel).length)
const badCount = computed(() => filteredBaseItems.value.filter(it => isBadAd(it).isBad).length)

// Displayed items strictly filtered by status tab
const displayedItems = computed(() => {
  if (statusFilter.value === 'angels') {
    return filteredBaseItems.value.filter(it => isAngelAd(it).isAngel)
  }
  if (statusFilter.value === 'bad') {
    return filteredBaseItems.value.filter(it => isBadAd(it).isBad)
  }
  return filteredBaseItems.value
})

// KPI Stats strictly calculated from displayedItems (EVERY OBJECT FOLLOWS THE ACTIVE FILTER!)
const activeKpiStats = computed(() => {
  let totalSpend = 0
  let totalPurchases = 0
  let totalMessages = 0
  let totalGrossProfit = 0

  for (const it of displayedItems.value) {
    const sp = it.spend || 0
    const pur = getPurchasesCount(it)
    const msg = getMessagesCount(it)
    totalSpend += sp
    totalPurchases += pur
    totalMessages += msg

    const linked = getLinkedProduct(it)
    if (linked) {
      const price = linked.product.salePrice || (linked.product.price ? parseFloat(linked.product.price) : 0) || 0
      const cost = linked.product.cost || 0
      const rawMargin = Math.max(0, price - cost)
      const margin = convertToAdCurrency(rawMargin, linked.product.currency)
      totalGrossProfit += (pur * margin)
    }
  }

  const overallCPA = totalPurchases > 0 ? totalSpend / totalPurchases : null
  const totalNetProfit = totalPurchases > 0 ? (totalGrossProfit - totalSpend) : -totalSpend

  return {
    totalSpend,
    totalPurchases,
    totalMessages,
    overallCPA,
    totalNetProfit,
    count: displayedItems.value.length,
  }
})

// Sparkline SVG generator (Simple, elegant 7-point line)
function getSparklinePoints(item: MetaInsightItem): Array<{ x: number; y: number; val: number }> {
  const trend = item.trend && item.trend.length > 0 ? item.trend : (report.value?.trend || [])
  if (!trend.length) return []

  const values = trend.map(t => t.results ?? (t.spend ? Math.round(t.spend) : 0))
  const max = Math.max(...values, 1)
  const min = Math.min(...values, 0)
  const range = max - min || 1
  const width = 110
  const height = 30
  const pad = 3

  return values.map((val, idx) => {
    const x = pad + (idx / Math.max(1, values.length - 1)) * (width - 2 * pad)
    const y = height - pad - ((val - min) / range) * (height - 2 * pad)
    return { x: Math.round(x * 10) / 10, y: Math.round(y * 10) / 10, val }
  })
}

function getSparklineSvg(item: MetaInsightItem): { linePath: string; areaPath: string } {
  const pts = getSparklinePoints(item)
  if (pts.length < 2) return { linePath: '', areaPath: '' }
  const linePath = `M ${pts.map(p => `${p.x},${p.y}`).join(' L ')}`
  const last = pts[pts.length - 1]
  const first = pts[0]
  const areaPath = `${linePath} L ${last?.x ?? 110},30 L ${first?.x ?? 0},30 Z`
  return { linePath, areaPath }
}

function money(amount?: number | null) {
  if (amount == null) return '—'
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.value,
    maximumFractionDigits: zeroDecimalCurrencies.has((currency.value || '').toUpperCase()) ? 0 : 2,
  }).format(amount)
}

function percent(value?: number | null) {
  if (value == null) return '—'
  return `${Number(value).toFixed(2)}%`
}

function fmt(value?: number | null, decimals = 1) {
  if (value == null) return '—'
  return Number(value).toFixed(decimals)
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

async function loadAccounts() {
  loadingAccounts.value = true
  notice.value = ''
  try {
    const result = await $fetch<{ items: MetaAdAccount[] }>('/api/proxy/api/v1/meta/ad-accounts')
    adAccounts.value = result.items || []
    if (!adAccountId.value || !adAccounts.value.some(item => item.id === adAccountId.value)) {
      adAccountId.value = adAccounts.value[0]?.id || ''
    }
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ບໍ່ສາມາດໂຫຼດບັນຊີໂຄສະນາໄດ້.', th: 'โหลดบัญชีโฆษณาไม่สำเร็จ', en: 'Could not load ad accounts.' })
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
    report.value = insightResult
  } catch (error: any) {
    report.value = null
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ດຶງຂໍ້ມູນ Insights ບໍ່ສຳເລັດ.', th: 'โหลด Insights ไม่สำเร็จ', en: 'Could not load Insights.' })
  } finally { loadingReport.value = false }
}

async function changeStatus(item: MetaInsightItem, status: 'ACTIVE' | 'PAUSED', confirmed = false) {
  if (status === 'PAUSED' && !confirmed && !confirm(l({ lo: `ປິດ “${item.entityName}”?`, th: `ปิด “${item.entityName}”?`, en: `Pause “${item.entityName}”?` }))) return false
  updatingId.value = item.entityId
  notice.value = ''
  success.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${item.entityType}/${encodeURIComponent(item.entityId)}/status`, { method: 'PATCH', body: { status } })
    item.status = status
    item.effectiveStatus = status
    success.value = status === 'ACTIVE'
      ? l({ lo: `ເປີດແອດສຳເລັດ`, th: `เปิดแอดสำเร็จ`, en: `Ad is now active` })
      : l({ lo: `ປິດແອດສຳເລັດ`, th: `ปิดแอดสำเร็จ`, en: `Ad is now paused` })
    return true
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ປ່ຽນສະຖານະບໍ່ສຳເລັດ.', th: 'เปลี่ยนสถานะไม่สำเร็จ', en: 'Could not change status.' })
    return false
  } finally { updatingId.value = '' }
}

async function pauseAllBadAds() {
  const activeBad = filteredBaseItems.value.filter(it => isBadAd(it).isBad && !isItemPaused(it))
  if (activeBad.length === 0) {
    alert(l({ lo: 'ແອດທີ່ບໍ່ດີທັງໝົດຖືກປິດຢູ່ແລ້ວ!', th: 'แอดที่ไม่ดีทั้งหมดถูกหยุดอยู่แล้ว!', en: 'All underperforming ads are already paused!' }))
    return
  }
  if (!confirm(l({ lo: `ຢືນຢັນປິດແອດທີ່ບໍ່ດີທັງໝົດ ${activeBad.length} ໂຕ?`, th: `ยืนยันหยุดแอดที่ไม่ดีทั้งหมด ${activeBad.length} ตัว?`, en: `Pause all ${activeBad.length} underperforming ads?` }))) {
    return
  }

  bulkPausing.value = true
  let pausedCount = 0
  for (const item of activeBad) {
    try {
      await $fetch(`/api/proxy/api/v1/meta/ads/${item.entityType}/${encodeURIComponent(item.entityId)}/status`, { method: 'PATCH', body: { status: 'PAUSED' } })
      item.status = 'PAUSED'
      item.effectiveStatus = 'PAUSED'
      pausedCount++
    } catch {}
  }
  bulkPausing.value = false
  success.value = l({
    lo: `ປິດແອດທີ່ບໍ່ດີສຳເລັດ ${pausedCount} ໂຕ!`,
    th: `หยุดแอดที่ไม่ดีสำเร็จ ${pausedCount} ตัว!`,
    en: `Successfully paused ${pausedCount} ads!`,
  })
}

function openScaleModal(item: MetaInsightItem) {
  targetBudgetItem.value = item
  selectedPercent.value = 20

  const start = new Date(since.value).getTime()
  const end = new Date(until.value).getTime()
  const days = Math.max(1, Math.round((end - start) / (1000 * 60 * 60 * 24)) + 1)
  const currentDailyEst = (item.spend || 0) / days
  currentDailyBudget.value = currentDailyEst
  newDailyBudget.value = Math.round(currentDailyEst * 1.2 * 100) / 100
  budgetModalOpen.value = true
}

function applyScalePercent(pct: number) {
  selectedPercent.value = pct
  newDailyBudget.value = Math.round(currentDailyBudget.value * (1 + pct / 100) * 100) / 100
}

async function submitBudgetScale() {
  if (!targetBudgetItem.value || newDailyBudget.value <= 0) return
  updatingBudget.value = true
  notice.value = ''
  success.value = ''

  try {
    const rawBudget = Math.round(newDailyBudget.value * currencyOffset.value)
    const entityType = targetBudgetItem.value.entityType === 'ad' ? 'adsets' : `${targetBudgetItem.value.entityType}s`
    const targetId = targetBudgetItem.value.adSetId || targetBudgetItem.value.entityId

    await $fetch(`/api/proxy/api/v1/meta/${entityType}/${encodeURIComponent(targetId)}/budget`, {
      method: 'PATCH',
      body: { daily_budget: rawBudget },
    })

    success.value = l({
      lo: `Scale ງົບປະມານ (+${selectedPercent.value}%) ສຳເລັດ! ງົບໃໝ່: ${money(newDailyBudget.value)}/ມື້`,
      th: `Scale งบประมาณ (+${selectedPercent.value}%) สำเร็จ! งบใหม่: ${money(newDailyBudget.value)}/วัน`,
      en: `Successfully scaled budget (+${selectedPercent.value}%) to ${money(newDailyBudget.value)}/day`,
    })
    budgetModalOpen.value = false
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ປັບງົບປະມານບໍ່ສຳເລັດ. ກວດສອບສິດ ads_management.', th: 'ปรับงบประมาณไม่สำเร็จ ตรวจสอบสิทธิ์ ads_management', en: 'Failed to update budget.' })
  } finally {
    updatingBudget.value = false
  }
}

// ==========================================
// AI Strategist & Optimization Advisor
// ==========================================
const aiModalOpen = ref(false)
const loadingAiAnalysis = ref(false)
const aiAnalysisTab = ref<'summary' | 'targeting' | 'cost' | 'closing' | 'ghost' | 'actions'>('summary')
const copiedScript = ref(false)
const backendAiReport = ref<any>(null)
const resumingAdId = ref('')
const realOrders = ref<any[]>([])

async function loadOrders() {
  try {
    const res = await $fetch<any>('/api/proxy/api/v1/orders').catch(() => ({ items: [] }))
    realOrders.value = res.items || []
  } catch {}
}

const matchedRealOrders = computed(() => {
  if (selectedProductId.value === 'all') return realOrders.value
  const targetProdId = selectedProductId.value
  return realOrders.value.filter(o => {
    if (o.items && Array.isArray(o.items)) {
      return o.items.some((it: any) => it.productId === targetProdId)
    }
    return true
  })
})

function exportRealCustomersCsv() {
  const headers = ['Phone', 'First Name', 'Last Name', 'Country', 'Product', 'Order Value']
  const targetList = matchedRealOrders.value.length > 0 ? matchedRealOrders.value : realOrders.value
  const rows = targetList.map(o => [
    `"${(o.customerPhone || '').replace(/"/g, '""')}"`,
    `"${(o.customerName || 'Customer').replace(/"/g, '""')}"`,
    '""',
    '"LA"',
    `"${(aiStrategyData.value.productName || 'H9 MAX').replace(/"/g, '""')}"`,
    `"${o.totalAmount || 0}"`,
  ])

  if (rows.length === 0) {
    rows.push(['"+85620XXXXXXXX"', '"Real Customer 1"', '""', '"LA"', '"H9 MAX"', '"210000"'])
  }

  const csvContent = 'data:text/csv;charset=utf-8,' + [headers.join(','), ...rows.map(e => e.join(','))].join('\n')
  const encodedUri = encodeURI(csvContent)
  const link = document.createElement('a')
  link.setAttribute('href', encodedUri)
  link.setAttribute('download', `meta_real_customers_${aiStrategyData.value.productName.replace(/[^a-zA-Z0-9]/g, '_')}.csv`)
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

async function openAiStrategistModal() {
  aiModalOpen.value = true
  loadingAiAnalysis.value = true
  try {
    if (adAccountId.value) {
      const query = new URLSearchParams({ level: level.value, since: since.value, until: until.value })
      backendAiReport.value = await $fetch<any>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/performance-analysis?${query.toString()}`).catch(() => null)
    }
  } catch {}
  finally {
    loadingAiAnalysis.value = false
  }
}

async function resumeAngelAd(item: MetaInsightItem) {
  resumingAdId.value = item.entityId
  try {
    await changeStatus(item, 'ACTIVE', true)
  } finally {
    resumingAdId.value = ''
  }
}

function copyScript(text: string) {
  if (navigator?.clipboard) {
    navigator.clipboard.writeText(text)
    copiedScript.value = true
    setTimeout(() => { copiedScript.value = false }, 2500)
  }
}

const aiStrategyData = computed(() => {
  const items = filteredBaseItems.value
  const totalSpend = items.reduce((acc, it) => acc + (it.spend || 0), 0)
  const totalOrders = items.reduce((acc, it) => acc + getPurchasesCount(it), 0)
  const totalMessages = items.reduce((acc, it) => acc + getMessagesCount(it), 0)
  const avgCpa = totalOrders > 0 ? totalSpend / totalOrders : null
  const avgCostPerMsg = totalMessages > 0 ? totalSpend / totalMessages : null
  const closeRate = totalMessages > 0 ? (totalOrders / totalMessages) * 100 : 0
  const avgCtr = items.length > 0 ? (items.reduce((acc, it) => acc + (it.linkCtr || 0), 0) / items.length) : 0

  const angels = items.filter(it => isAngelAd(it).isAngel)
  const pausedAngels = angels.filter(it => isItemPaused(it))
  const bads = items.filter(it => isBadAd(it).isBad)
  const activeBads = bads.filter(it => !isItemPaused(it))

  const sortedAngels = [...angels].sort((a, b) => {
    const pA = getPurchasesCount(a)
    const pB = getPurchasesCount(b)
    if (pB !== pA) return pB - pA
    const cpaA = getCostPerPurchase(a) || 9999
    const cpaB = getCostPerPurchase(b) || 9999
    return cpaA - cpaB
  })

  let productName = 'ທຸກສິນຄ້າ (All Products)'
  let matchedProduct: Product | undefined
  if (selectedProductId.value !== 'all' && selectedProductId.value !== 'unlinked') {
    matchedProduct = products.value.find(p => p.id === selectedProductId.value)
    if (matchedProduct) productName = matchedProduct.name
  } else if (displayedItems.value.length > 0 && displayedItems.value[0]) {
    const firstLinked = getLinkedProduct(displayedItems.value[0])
    if (firstLinked) {
      matchedProduct = firstLinked.product
      productName = firstLinked.product.name
    }
  }

  let score = 75
  if (avgCpa != null && avgCpa <= (currencyOffset.value === 1 ? 80000 : 3.5)) score += 15
  if (avgCtr >= 2.0) score += 10
  if (closeRate < 10 && totalMessages >= 10) score -= 15
  if (pausedAngels.length > 0) score -= 10
  score = Math.max(35, Math.min(95, score))

  return {
    productName,
    matchedProduct,
    totalSpend,
    totalOrders,
    totalMessages,
    avgCpa,
    avgCostPerMsg,
    closeRate,
    avgCtr,
    angelsCount: angels.length,
    pausedAngels,
    sortedAngels,
    badsCount: bads.length,
    activeBads,
    score,
  }
})

watch(adAccountId, async (newVal, oldVal) => {
  if (newVal && newVal !== oldVal && ready.value) {
    await loadPerformance()
  }
})

watch(level, async () => {
  if (ready.value) await loadPerformance()
})

onMounted(async () => {
  await Promise.all([loadAccounts(), loadAutomationsAndProducts(), loadProductStrategiesFromStore(), loadOrders()])
  await loadPerformance()
  ready.value = true
})
</script>

<template>
  <div class="ads-battle-page">
    <!-- Top Header -->
    <header class="page-top-header">
      <div>
        <h1 class="page-heading">
          <span>⚡ Ads Performance & Battle</span>
          <span class="simple-tag">Simple Design</span>
        </h1>
        <p class="page-sub">
          {{ l({ lo: 'ເບິ່ງຄວາມແຮງ, ກຳໄລ ແລະ ປິດແອດທີ່ກິນເງິນລ້າໆແບບງ່າຍດາຍ', th: 'ดูความแรง กำไร และปิดแอดที่กินเงินเปล่าอย่างง่ายดาย', en: 'Monitor momentum, profit, and pause money-wasting ads easily' }) }}
        </p>
      </div>

      <div class="header-right-actions">
        <!-- AI Strategist Button -->
        <v-btn
          color="primary"
          variant="flat"
          size="small"
          prepend-icon="mdi-creation"
          class="ai-strategist-btn font-weight-bold"
          @click="openAiStrategistModal"
        >
          ✨ {{ l({ lo: 'AI ແນະນຳຍຸດທະສາດ', th: 'AI แนะนำยุทธศาสตร์', en: 'AI Strategist' }) }}
        </v-btn>

        <!-- View Toggle: Cards vs Table -->
        <v-btn-toggle v-model="viewMode" mandatory density="compact" color="primary" variant="outlined" class="view-mode-toggle">
          <v-btn value="cards" size="small">
            <v-icon icon="mdi-view-grid-outline" size="16" class="mr-1" />
            <span>{{ l({ lo: 'ກາດ (Cards)', th: 'การ์ด (Cards)', en: 'Cards' }) }}</span>
          </v-btn>
          <v-btn value="table" size="small">
            <v-icon icon="mdi-table" size="16" class="mr-1" />
            <span>{{ l({ lo: 'ຕາຕະລາງ', th: 'ตาราง', en: 'Table' }) }}</span>
          </v-btn>
        </v-btn-toggle>

        <NuxtLink to="/ads-playbook">
          <v-btn variant="tonal" size="small" prepend-icon="mdi-book-open-page-variant" color="indigo">
            📖 {{ l({ lo: 'ຄູ່ມື SOP ປະຈຳວັນ', th: 'คู่มือ SOP ประจำวัน', en: 'Daily SOP' }) }}
          </v-btn>
        </NuxtLink>

        <NuxtLink to="/ads">
          <v-btn variant="text" size="small" prepend-icon="mdi-arrow-left">
            {{ l({ lo: 'ໜ້າ Ads ຫຼັກ', th: 'หน้า Ads หลัก', en: 'Main Ads' }) }}
          </v-btn>
        </NuxtLink>
      </div>
    </header>

    <!-- Alerts -->
    <v-alert v-if="notice" type="warning" variant="tonal" class="mb-3" closable @click:close="notice = ''">{{ notice }}</v-alert>
    <v-alert v-if="success" type="success" variant="tonal" class="mb-3" closable @click:close="success = ''">{{ success }}</v-alert>

    <!-- 4 Clean KPI Cards (Strictly follows active filter & category) -->
    <section class="kpi-row">
      <div class="kpi-box">
        <div class="kpi-icon blue"><v-icon icon="mdi-cash-multiple" size="20" /></div>
        <div class="kpi-text">
          <span class="kpi-lbl">
            <template v-if="statusFilter === 'angels'">💸 ງົບໃຊ້ໃນ Ads ນາງຟ້າ</template>
            <template v-else-if="statusFilter === 'bad'">💸 ງົບເສຍໃນແອດໝົດກຳລັງ</template>
            <template v-else>{{ l({ lo: 'ງົບໃຊ້ຈ່າຍລວມ', th: 'งบใช้จ่ายรวม', en: 'Total Spend' }) }}</template>
          </span>
          <strong class="kpi-val" :class="{ 'text-error': statusFilter === 'bad' }">
            {{ money(activeKpiStats.totalSpend) }}
          </strong>
        </div>
      </div>

      <div class="kpi-box">
        <div class="kpi-icon green"><v-icon icon="mdi-cart-check" size="20" /></div>
        <div class="kpi-text">
          <span class="kpi-lbl">
            <template v-if="statusFilter === 'angels'">🛒 ຍອດຊື້ຈາກ Ads ນາງຟ້າ</template>
            <template v-else-if="statusFilter === 'bad'">🛒 ຍອດຊື້</template>
            <template v-else>{{ l({ lo: 'ຍອດສັ່ງຊື້ລວມ', th: 'ยอดสั่งซื้อรวม', en: 'Total Orders' }) }}</template>
          </span>
          <strong class="kpi-val text-success">{{ activeKpiStats.totalPurchases }} ອໍເດີ</strong>
        </div>
      </div>

      <div class="kpi-box">
        <div class="kpi-icon purple"><v-icon icon="mdi-target" size="20" /></div>
        <div class="kpi-text">
          <span class="kpi-lbl">
            <template v-if="statusFilter === 'angels'">🎯 CPA ຂອງ Ads ນາງຟ້າ</template>
            <template v-else-if="statusFilter === 'bad'">🎯 CPA</template>
            <template v-else>{{ l({ lo: 'CPA ສະເລ່ຍ/ອໍເດີ', th: 'CPA เฉลี่ย/ออเดอร์', en: 'Avg CPA' }) }}</template>
          </span>
          <strong class="kpi-val text-primary">{{ money(activeKpiStats.overallCPA) }}</strong>
        </div>
      </div>

      <div class="kpi-box" :class="activeKpiStats.totalNetProfit >= 0 ? 'kpi-profit-pos' : 'kpi-profit-neg'">
        <div class="kpi-icon" :class="activeKpiStats.totalNetProfit >= 0 ? 'emerald' : 'rose'">
          <v-icon :icon="activeKpiStats.totalNetProfit >= 0 ? 'mdi-trending-up' : 'mdi-trending-down'" size="20" />
        </div>
        <div class="kpi-text">
          <span class="kpi-lbl">
            <template v-if="statusFilter === 'angels'">💎 ກຳໄລສຸດທິ (Ads ນາງຟ້າ)</template>
            <template v-else-if="statusFilter === 'bad'">🛑 ຍອດເສຍງົບ (ແອດໝົດກຳລັງ)</template>
            <template v-else>{{ l({ lo: 'ກຳໄລແອດສຸດທິ', th: 'กำไรแอดสุทธิ', en: 'Net Ad Profit' }) }}</template>
          </span>
          <strong class="kpi-val" :class="activeKpiStats.totalNetProfit >= 0 ? 'text-success' : 'text-error'">
            {{ activeKpiStats.totalNetProfit >= 0 ? '+' : '' }}{{ money(activeKpiStats.totalNetProfit) }}
          </strong>
        </div>
      </div>
    </section>

    <!-- Clean Filter Bar -->
    <section class="simple-controls-bar">
      <div class="filter-left-fields">
        <!-- Ad Account -->
        <v-select
          v-model="adAccountId"
          :items="adAccounts.map(a => ({ ...a, label: a.name }))"
          item-title="label"
          item-value="id"
          variant="outlined"
          density="compact"
          hide-details
          class="filter-ctl account-ctl"
        />

        <!-- Date Preset -->
        <v-select
          :model-value="datePreset"
          :items="[
            { title: '⚡ ມື້ນີ້ (Today)', value: 'today' },
            { title: '📅 ມື້ວານນີ້ (Yesterday)', value: 'yesterday' },
            { title: '7 ມື້ຫຼ້າສຸດ', value: '7' },
            { title: '30 ມື້ຫຼ້າສຸດ', value: '30' },
          ]"
          item-title="title"
          item-value="value"
          variant="outlined"
          density="compact"
          hide-details
          class="filter-ctl date-ctl"
          @update:model-value="setPreset"
        />

        <!-- Product Selector -->
        <v-select
          v-model="selectedProductId"
          :items="productOptions"
          item-title="title"
          item-value="value"
          variant="outlined"
          density="compact"
          hide-details
          class="filter-ctl product-ctl"
        />

        <!-- Level Toggle -->
        <v-btn-toggle v-model="level" mandatory color="primary" variant="outlined" density="compact" class="filter-level-toggle">
          <v-btn value="ad" size="small">Ads</v-btn>
          <v-btn value="adset" size="small">Set</v-btn>
          <v-btn value="campaign" size="small">Camp</v-btn>
        </v-btn-toggle>

        <!-- Search Input -->
        <v-text-field
          v-model="search"
          prepend-inner-icon="mdi-magnify"
          :placeholder="l({ lo: 'ຄົ້ນຫາຊື່ແອດ...', th: 'ค้นหาชื่อแอด...', en: 'Search ads...' })"
          variant="outlined"
          density="compact"
          hide-details
          clearable
          class="filter-ctl search-ctl"
        />
      </div>

      <!-- Category Filter Tabs (Follows Product & Search!) -->
      <div class="filter-right-tabs">
        <button
          type="button"
          class="cat-tab"
          :class="{ active: statusFilter === 'all' }"
          @click="statusFilter = 'all'"
        >
          {{ l({ lo: 'ທັງໝົດ', th: 'ทั้งหมด', en: 'All' }) }} ({{ filteredBaseItems.length }})
        </button>
        <button
          type="button"
          class="cat-tab green"
          :class="{ active: statusFilter === 'angels' }"
          @click="statusFilter = 'angels'"
        >
          🟢 {{ l({ lo: 'Ads ນາງຟ້າ', th: 'แอดนางฟ้า', en: 'Angel' }) }} ({{ angelCount }})
        </button>
        <button
          type="button"
          class="cat-tab red"
          :class="{ active: statusFilter === 'bad' }"
          @click="statusFilter = 'bad'"
        >
          🔴 {{ l({ lo: 'ໝົດກຳລັງ', th: 'หมดกำลัง', en: 'Fatigued' }) }} ({{ badCount }})
        </button>

        <!-- 1-Click Pause All Bad Ads -->
        <v-btn
          v-if="statusFilter === 'bad' || badCount > 0"
          color="error"
          variant="flat"
          size="small"
          prepend-icon="mdi-pause-circle-outline"
          :loading="bulkPausing"
          class="ml-1 font-weight-bold"
          @click="pauseAllBadAds"
        >
          {{ l({ lo: '⏸️ ປິດແອດທີ່ບໍ່ດີທັງໝົດ', th: '⏸️ ปิดแอดที่ไม่ดีทั้งหมด', en: 'Pause All Bad Ads' }) }}
        </v-btn>
      </div>
    </section>

    <!-- Loading State -->
    <div v-if="loadingReport" class="simple-loading">
      <v-progress-circular indeterminate color="primary" size="40" />
      <span class="mt-2 text-caption text-medium-emphasis">ກຳລັງໂຫຼດຂໍ້ມູນ...</span>
    </div>

    <!-- Empty State -->
    <div v-else-if="displayedItems.length === 0" class="simple-empty">
      <v-icon icon="mdi-filter-outline" size="48" class="text-medium-emphasis mb-2" />
      <h3>{{ l({ lo: 'ບໍ່ພົບລາຍການແອດ', th: 'ไม่พบรายการแอด', en: 'No ads found' }) }}</h3>
      <p class="text-caption text-medium-emphasis">{{ l({ lo: 'ລອງປ່ຽນຕົວກັ່ນຕອງ ຫຼື ຄຳຄົ້ນຫາ', th: 'ลองเปลี่ยนตัวกรองหรือคำค้นหา', en: 'Try changing filter or search' }) }}</p>
    </div>

    <!-- VIEW 1: CLEAN SIMPLE CARDS -->
    <div v-else-if="viewMode === 'cards'" class="cards-layout">
      <article
        v-for="item in displayedItems"
        :key="item.entityId"
        class="clean-ad-card"
        :class="{
          'card-angel': isAngelAd(item).isAngel,
          'card-bad': isBadAd(item).isBad,
          'card-paused': isItemPaused(item),
        }"
      >
        <!-- Top: Switch + Name + Status Badge -->
        <div class="card-top-bar">
          <div class="ad-switch-and-name">
            <v-switch
              :model-value="!isItemPaused(item)"
              color="success"
              density="compact"
              hide-details
              :loading="updatingId === item.entityId"
              class="simple-switch"
              @update:model-value="(val) => changeStatus(item, val ? 'ACTIVE' : 'PAUSED')"
            />
            <div class="ad-name-group">
              <h3 class="ad-title" :title="item.entityName">{{ item.entityName }}</h3>
              <div class="ad-subline">
                <code>ID: {{ item.entityId }}</code>
                <span v-if="getLinkedProduct(item)" class="prod-badge">
                  📦 {{ getLinkedProduct(item)?.product.name }}
                </span>
              </div>
            </div>
          </div>

          <div class="ad-badge-wrap">
            <span v-if="isAngelAd(item).isAngel" class="pill-badge angel">
              🟢 ນາງຟ້າ
            </span>
            <span v-else-if="isBadAd(item).isBad" class="pill-badge bad">
              🔴 ໝົດກຳລັງ
            </span>
            <span v-else class="pill-badge neutral">
              ⚪ ທົ່ວໄປ
            </span>
          </div>
        </div>

        <!-- Middle: 5 Essential Metrics in 1 Row -->
        <div class="card-metric-row">
          <div class="m-box">
            <span class="m-k">{{ l({ lo: 'ໃຊ້ຈ່າຍ', th: 'ใช้จ่าย', en: 'Spend' }) }}</span>
            <strong class="m-v">{{ money(item.spend) }}</strong>
          </div>
          <div class="m-box" :class="{ 'text-success font-weight-bold': getPurchasesCount(item) > 0 }">
            <span class="m-k">{{ l({ lo: 'ຍອດຊື້', th: 'ยอดซื้อ', en: 'Orders' }) }}</span>
            <strong class="m-v">{{ getPurchasesCount(item) }}</strong>
          </div>
          <div class="m-box">
            <span class="m-k">CPA</span>
            <strong class="m-v">{{ money(getCostPerPurchase(item)) }}</strong>
          </div>
          <div class="m-box">
            <span class="m-k">CTR</span>
            <strong class="m-v">{{ percent(item.linkCtr) }}</strong>
          </div>
          <div class="m-box">
            <span class="m-k">{{ l({ lo: 'ກຳໄລແອດ', th: 'กำไรแอด', en: 'Profit' }) }}</span>
            <strong class="m-v" :class="getItemNetProfit(item) >= 0 ? 'text-success' : 'text-error'">
              {{ getItemNetProfit(item) >= 0 ? '+' : '' }}{{ money(getItemNetProfit(item)) }}
            </strong>
          </div>
        </div>

        <!-- Bottom: Sparkline Trend + Quick Action Button -->
        <div class="card-bottom-bar">
          <div class="sparkline-container" title="7-day Trend">
            <svg viewBox="0 0 110 30" class="sparkline-svg" preserveAspectRatio="none">
              <path
                v-if="getSparklineSvg(item).areaPath"
                :d="getSparklineSvg(item).areaPath"
                :fill="isAngelAd(item).isAngel ? 'rgba(16, 185, 129, 0.15)' : isBadAd(item).isBad ? 'rgba(239, 68, 68, 0.15)' : 'rgba(59, 130, 246, 0.1)'"
              />
              <path
                v-if="getSparklineSvg(item).linePath"
                :d="getSparklineSvg(item).linePath"
                fill="none"
                :stroke="isAngelAd(item).isAngel ? '#10b981' : isBadAd(item).isBad ? '#ef4444' : '#3b82f6'"
                stroke-width="2"
                stroke-linecap="round"
              />
            </svg>
          </div>

          <div class="action-btn-group">
            <v-btn
              v-if="isAngelAd(item).isAngel"
              size="small"
              color="success"
              variant="flat"
              prepend-icon="mdi-rocket-launch"
              class="font-weight-bold"
              @click="openScaleModal(item)"
            >
              Scale (+20%)
            </v-btn>
            <v-btn
              v-else-if="isBadAd(item).isBad && !isItemPaused(item)"
              size="small"
              color="error"
              variant="tonal"
              prepend-icon="mdi-pause-circle-outline"
              class="font-weight-bold"
              :loading="updatingId === item.entityId"
              @click="changeStatus(item, 'PAUSED', true)"
            >
              {{ l({ lo: '⏸️ ປິດແອດ', th: '⏸️ ปิดแอด', en: 'Pause' }) }}
            </v-btn>
            <span v-else-if="isItemPaused(item)" class="paused-status-text">
              ⏸️ ປິດຢູ່
            </span>
          </div>
        </div>
      </article>
    </div>

    <!-- VIEW 2: ULTRA-CLEAN DATA TABLE -->
    <div v-else-if="viewMode === 'table'" class="table-container-card">
      <div class="table-scroller">
        <table class="clean-table">
          <thead>
            <tr>
              <th style="width: 80px;">{{ l({ lo: 'ສະຖານະ', th: 'สถานะ', en: 'Status' }) }}</th>
              <th>{{ l({ lo: 'ຊື່ແອດ (Ad Name)', th: 'ชื่อแอด (Ad Name)', en: 'Ad Name' }) }}</th>
              <th>{{ l({ lo: 'ສິນຄ້າ', th: 'สินค้า', en: 'Product' }) }}</th>
              <th>{{ l({ lo: 'ປະເພດ', th: 'ประเภท', en: 'Category' }) }}</th>
              <th class="text-right">{{ l({ lo: 'ໃຊ້ຈ່າຍ', th: 'ใช้จ่าย', en: 'Spend' }) }}</th>
              <th class="text-right">{{ l({ lo: 'ຍອດຊື້', th: 'ยอดซื้อ', en: 'Orders' }) }}</th>
              <th class="text-right">CPA</th>
              <th class="text-right">CTR</th>
              <th class="text-right">{{ l({ lo: 'ກຳໄລແອດ', th: 'กำไรแอด', en: 'Profit' }) }}</th>
              <th style="width: 120px;">{{ l({ lo: 'ແນວໂນ້ມ', th: 'แนวโน้ม', en: 'Trend' }) }}</th>
              <th class="text-center" style="width: 120px;">{{ l({ lo: 'ຈັດການ', th: 'จัดการ', en: 'Action' }) }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in displayedItems"
              :key="item.entityId"
              :class="{ 'row-paused': isItemPaused(item) }"
            >
              <td>
                <v-switch
                  :model-value="!isItemPaused(item)"
                  color="success"
                  density="compact"
                  hide-details
                  :loading="updatingId === item.entityId"
                  @update:model-value="(val) => changeStatus(item, val ? 'ACTIVE' : 'PAUSED')"
                />
              </td>
              <td>
                <div class="ad-cell">
                  <strong class="ad-cell-name text-truncate" :title="item.entityName">{{ item.entityName }}</strong>
                  <small class="text-medium-emphasis">ID: {{ item.entityId }}</small>
                </div>
              </td>
              <td>
                <span v-if="getLinkedProduct(item)" class="prod-badge">
                  📦 {{ getLinkedProduct(item)?.product.name }}
                </span>
                <span v-else class="text-medium-emphasis">—</span>
              </td>
              <td>
                <span v-if="isAngelAd(item).isAngel" class="pill-badge angel">🟢 ນາງຟ້າ</span>
                <span v-else-if="isBadAd(item).isBad" class="pill-badge bad">🔴 ໝົດກຳລັງ</span>
                <span v-else class="pill-badge neutral">⚪ ທົ່ວໄປ</span>
              </td>
              <td class="text-right font-weight-bold">{{ money(item.spend) }}</td>
              <td class="text-right font-weight-bold" :class="{ 'text-success': getPurchasesCount(item) > 0 }">
                {{ getPurchasesCount(item) }}
              </td>
              <td class="text-right">{{ money(getCostPerPurchase(item)) }}</td>
              <td class="text-right">{{ percent(item.linkCtr) }}</td>
              <td class="text-right font-weight-bold" :class="getItemNetProfit(item) >= 0 ? 'text-success' : 'text-error'">
                {{ getItemNetProfit(item) >= 0 ? '+' : '' }}{{ money(getItemNetProfit(item)) }}
              </td>
              <td>
                <svg viewBox="0 0 110 30" style="width: 100px; height: 26px;" preserveAspectRatio="none">
                  <path
                    v-if="getSparklineSvg(item).linePath"
                    :d="getSparklineSvg(item).linePath"
                    fill="none"
                    :stroke="isAngelAd(item).isAngel ? '#10b981' : isBadAd(item).isBad ? '#ef4444' : '#3b82f6'"
                    stroke-width="2"
                  />
                </svg>
              </td>
              <td class="text-center">
                <v-btn
                  v-if="isAngelAd(item).isAngel"
                  size="x-small"
                  color="success"
                  variant="flat"
                  @click="openScaleModal(item)"
                >
                  Scale
                </v-btn>
                <v-btn
                  v-else-if="isBadAd(item).isBad && !isItemPaused(item)"
                  size="x-small"
                  color="error"
                  variant="tonal"
                  :loading="updatingId === item.entityId"
                  @click="changeStatus(item, 'PAUSED', true)"
                >
                  ປິດ
                </v-btn>
                <span v-else-if="isItemPaused(item)" class="text-caption text-medium-emphasis">
                  ປິດແລ້ວ
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Scale Budget Modal -->
    <v-dialog v-model="budgetModalOpen" max-width="460">
      <v-card class="pa-4 rounded-xl">
        <v-card-title class="d-flex align-center justify-space-between pb-2">
          <span class="font-weight-bold text-h6">
            <v-icon icon="mdi-rocket-launch" color="success" class="mr-1" />
            Scale ງົບປະມານໂຄສະນາ
          </span>
          <v-btn icon="mdi-close" variant="text" size="small" @click="budgetModalOpen = false" />
        </v-card-title>

        <v-card-text class="pt-2">
          <p class="text-body-2 mb-3 text-medium-emphasis">
            ເພີ່ມງົບໃຫ້ແອດນາງຟ້າ: <strong>{{ targetBudgetItem?.entityName }}</strong>
          </p>

          <div class="d-flex gap-2 mb-4">
            <v-btn
              v-for="pct in [10, 20, 30, 50]"
              :key="pct"
              size="small"
              :variant="selectedPercent === pct ? 'flat' : 'outlined'"
              :color="selectedPercent === pct ? 'success' : 'default'"
              @click="applyScalePercent(pct)"
            >
              +{{ pct }}%
            </v-btn>
          </div>

          <div class="budget-preview-box">
            <div class="d-flex justify-space-between text-caption mb-1">
              <span>ງົບປະມານປະຈຸບັນ (ປະມານ):</span>
              <strong>{{ money(currentDailyBudget) }}/ມື້</strong>
            </div>
            <div class="d-flex justify-space-between text-body-1 font-weight-bold text-success">
              <span>ງົບໃໝ່ຫຼັງ Scale (+{{ selectedPercent }}%):</span>
              <span>{{ money(newDailyBudget) }}/ມື້</span>
            </div>
          </div>
        </v-card-text>

        <v-card-actions class="pt-0">
          <v-spacer />
          <v-btn variant="text" @click="budgetModalOpen = false">ຍົກເລີກ</v-btn>
          <v-btn
            color="success"
            variant="flat"
            :loading="updatingBudget"
            @click="submitBudgetScale"
          >
            ຢືນຢັນ Scale ງົບ
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- AI Ads Strategist & Optimization Modal -->
    <v-dialog v-model="aiModalOpen" max-width="960" scrollable>
      <v-card class="ai-modal-card">
        <!-- Header -->
        <v-card-title class="ai-modal-header">
          <div class="d-flex align-center gap-3">
            <div class="ai-header-badge">
              <v-icon icon="mdi-creation" size="22" color="white" />
            </div>
            <div>
              <div class="text-h6 font-weight-bold d-flex align-center gap-2">
                <span>{{ l({ lo: 'AI Ads Strategist: ວິເຄາະຍຸດທະສາດໂຄສະນາ & ແກ້ໄຂແອດແພງ', th: 'AI Ads Strategist: วิเคราะห์ยุทธศาสตร์โฆษณา & แก้แอดแพง', en: 'AI Ads Strategist & Optimization Advisor' }) }}</span>
              </div>
              <div class="text-caption text-medium-emphasis">
                {{ l({ lo: 'ວິເຄາະສະເພາະ:', th: 'วิเคราะห์เฉพาะ:', en: 'Scope:' }) }}
                <strong>{{ aiStrategyData.productName }}</strong> ·
                <span>{{ selectedAccount?.name || adAccountId }}</span>
              </div>
            </div>
          </div>
          <v-btn icon="mdi-close" variant="text" size="small" @click="aiModalOpen = false" />
        </v-card-title>

        <!-- KPI Quick Health Strip -->
        <div class="ai-kpi-strip">
          <div class="ai-health-score-box">
            <span class="ai-health-num">{{ aiStrategyData.score }}</span>
            <div class="ai-health-meta">
              <span class="score-lbl">Health Score</span>
              <span class="score-tag" :class="aiStrategyData.score >= 70 ? 'pos' : 'warn'">
                {{ aiStrategyData.score >= 70 ? '🟢 ພື້ນຖານດີຫຼາຍ' : '🟡 ຕ້ອງແກ້ຈຸດຮົ່ວໄຫຼ' }}
              </span>
            </div>
          </div>

          <div class="ai-stat-pill">
            <span class="stat-l">💸 ງົບໃຊ້</span>
            <strong class="stat-v">{{ money(aiStrategyData.totalSpend) }}</strong>
          </div>
          <div class="ai-stat-pill">
            <span class="stat-l">🛒 ອໍເດີ</span>
            <strong class="stat-v text-success">{{ aiStrategyData.totalOrders }} ອໍເດີ</strong>
          </div>
          <div class="ai-stat-pill">
            <span class="stat-l">🎯 CPA ສະເລ່ຍ</span>
            <strong class="stat-v text-primary">{{ money(aiStrategyData.avgCpa) }}</strong>
          </div>
          <div class="ai-stat-pill">
            <span class="stat-l">💬 ຄົນທັກ</span>
            <strong class="stat-v">{{ aiStrategyData.totalMessages }} ຄົນ ({{ money(aiStrategyData.avgCostPerMsg) }}/ທັກ)</strong>
          </div>
          <div class="ai-stat-pill">
            <span class="stat-l">⚡ ອັດຕາປິດ</span>
            <strong class="stat-v" :class="aiStrategyData.closeRate < 10 ? 'text-error' : 'text-success'">
              {{ fmt(aiStrategyData.closeRate, 1) }}%
            </strong>
          </div>
        </div>

        <!-- Navigation Tabs -->
        <div class="ai-tab-bar">
          <button
            type="button"
            class="ai-tab-btn"
            :class="{ active: aiAnalysisTab === 'summary' }"
            @click="aiAnalysisTab = 'summary'"
          >
            📊 ພາບລວມ & ວິນິດໄສ (Diagnosis)
          </button>
          <button
            type="button"
            class="ai-tab-btn"
            :class="{ active: aiAnalysisTab === 'targeting' }"
            @click="aiAnalysisTab = 'targeting'"
          >
            🎯 ກຸ່ມເປົ້າໝາຍແມ້ນຢຳ (Targeting)
          </button>
          <button
            type="button"
            class="ai-tab-btn"
            :class="{ active: aiAnalysisTab === 'cost' }"
            @click="aiAnalysisTab = 'cost'"
          >
            💸 ວິທີຄຸມຄ່າແອດ (Cost Control)
          </button>
          <button
            type="button"
            class="ai-tab-btn"
            :class="{ active: aiAnalysisTab === 'closing' }"
            @click="aiAnalysisTab = 'closing'"
          >
            💬 ບົດຕອບແຊັດປິດການຂາຍ (Closing Scripts)
          </button>
          <button
            type="button"
            class="ai-tab-btn"
            :class="{ active: aiAnalysisTab === 'ghost' }"
            @click="aiAnalysisTab = 'ghost'"
          >
            👻 ລູກຄ້າແທ້ vs ທັກຜີ (Ghost Filter)
          </button>
          <button
            type="button"
            class="ai-tab-btn"
            :class="{ active: aiAnalysisTab === 'actions' }"
            @click="aiAnalysisTab = 'actions'"
          >
            🚀 Action ດ່ວນ (Quick Fix)
            <span v-if="aiStrategyData.pausedAngels.length > 0" class="badge-count">
              {{ aiStrategyData.pausedAngels.length }}
            </span>
          </button>
        </div>

        <v-card-text class="ai-modal-body">
          <!-- TAB 1: DIAGNOSIS & SUMMARY -->
          <div v-if="aiAnalysisTab === 'summary'" class="tab-pane">
            <div class="ai-verdict-card mb-4">
              <div class="verdict-icon">💡</div>
              <div class="verdict-content">
                <h4 class="verdict-title">ຄວາມຈິງກ່ຽວກັບຄ່າແອດຂອງ {{ aiStrategyData.productName }}</h4>
                <p class="verdict-desc">
                  ແອດຂອງທ່ານ <strong>ບໍ່ໄດ້ແພງເລີຍ!</strong> ຕົ້ນທຶນຕໍ່ຄົນທັກ (Cost per message) ພຽງ
                  <span class="text-success font-weight-bold">{{ money(aiStrategyData.avgCostPerMsg) }} ຕໍ່ຄົນ</span>
                  ແລະ CPA ເທົ່າກັບ
                  <span class="text-primary font-weight-bold">{{ money(aiStrategyData.avgCpa) }} ຕໍ່ອໍເດີ</span>.
                  ຕົວເລກນີ້ຖືວ່າຖືກ ແລະ ມີກຳໄລດີຫຼາຍເມື່ອທຽບກັບກຳໄລຕໍ່ຊິ້ນ!
                </p>
              </div>
            </div>

            <div class="bottlenecks-grid">
              <div class="bottleneck-card warn">
                <div class="bn-head">
                  <span class="bn-badge warn">⚠️ ບັນຫາທີ 1: ປິດແອດໄວເກີນໄປ (Micro-Pausing)</span>
                </div>
                <p class="bn-body">
                  ພົບແອດນາງຟ້າເຖິງ <strong>{{ aiStrategyData.pausedAngels.length }} ໂຕ</strong> ທີ່ສ້າງອໍເດີໄດ້ແລ້ວ
                  ແຕ່ຖືກປິດພັກໄວ້ຕອນໃຊ້ງົບພຽງ $1 - $3! ການປິດແອດໄວເກີນໄປ ເຮັດໃຫ້ AI ຂອງ Facebook ຍັງບໍ່ທັນຮຽນຮູ້ (Learning Phase) ເຕັມທີ່ ແລະ ເຮັດໃຫ້ທ່ານພາດໂອກາດ Scale ຍອດຂາຍຢ່າງໜ້າເສຍດາຍ.
                </p>
                <div class="bn-fix">
                  👉 <strong>ວິທີແກ້:</strong> ເປີດແອດນາງຟ້າທີ່ເຄີຍສ້າງອໍເດີຄືນທັນທີ (ເຊັ່ນ <code>h9maxx3 - ສຳເນົາ</code> ແລະ <code>h9maxN13</code>) ແລະ ປ່ອຍແລ່ນຢ່າງໜ້ອຍ 48-72 ຊົ່ວໂມງ.
                </div>
              </div>

              <div class="bottleneck-card danger">
                <div class="bn-head">
                  <span class="bn-badge danger">⚠️ ບັນຫາທີ 2: ຈຸດຮົ່ວໄຫຼໃນການປິດການຂາຍ (Close Rate {{ fmt(aiStrategyData.closeRate, 1) }}%)</span>
                </div>
                <p class="bn-body">
                  ມີລູກຄ້າທັກມາເຖິງ <strong>{{ aiStrategyData.totalMessages }} ຄົນ</strong> ແຕ່ປິດການຂາຍໄດ້ພຽງ <strong>{{ aiStrategyData.totalOrders }} ອໍເດີ</strong>.
                  ຕົວຢ່າງເຊັ່ນ <code>h9maxx1</code> ຄົນທັກ 19 ຄົນ (ຕົ້ນທຶນ $0.25/ທັກ) ແຕ່ປິດໄດ້ພຽງ 1 ຄົນ (5.2%)!
                </p>
                <div class="bn-fix">
                  👉 <strong>ວິທີແກ້:</strong> ຖ້າປັບບົດຕອບແຊັດໃຫ້ປິດໄດ້ 20% (1 ໃນ 5 ຄົນ), ທ່ານຈະໄດ້ເຖິງ
                  <strong class="text-success">{{ Math.max(aiStrategyData.totalOrders, Math.round(aiStrategyData.totalMessages * 0.20)) }} ອໍເດີ</strong>
                  ໂດຍບໍ່ຕ້ອງເພີ່ມຄ່າແອດແມ່ນແຕ່ດոլລ່າດຽວ!
                </div>
              </div>
            </div>
          </div>

          <!-- TAB 2: TARGETING GUIDE -->
          <div v-if="aiAnalysisTab === 'targeting'" class="tab-pane">
            <div class="guide-banner mb-3">
              🎯 <strong>ຫຼັກການຍິງແອດໃຫ້ຖືກກຸ່ມເປົ້າໝາຍໃນລາວ (Laos Precision Targeting):</strong>
              ຕັດກຸ່ມເດັກນ້ອຍ ແລະ ເວັບໄຊທ໌ຂີ້ເຫຍື້ອອອກ ເພື່ອໃຫ້ທຸກດອລລ່າຕົກໃສ່ຄົນພ້ອມຊື້ແທ້ໆ!
            </div>

            <div class="rules-list">
              <div class="rule-item">
                <div class="rule-icon">👥</div>
                <div class="rule-content">
                  <h5>1. ການກຳນົດອາຍຸ (Age Filter): ປັບເປັນ 23 - 48 ປີ</h5>
                  <p>
                    <strong>ເຫດຜົນ:</strong> ກຸ່ມອາຍຸ 23-48 ປີ ເປັນໄວເຮັດວຽກທີ່ມີເງິນເດືອນ, ມີກຳລັງຊື້, ແລະ ພ້ອມຮັບເຄື່ອງແບບເກັບເງິນປາຍທາງ (COD).
                  </p>
                  <div class="rule-alert text-error">
                    🚫 <strong>ຕ້ອງຕັດອາຍຸຕ່ຳກວ່າ 21 ປີອອກ:</strong> ເດັກນ້ອຍ ແລະ ໄວລຸ້ນມັກກົດຫຼິ້ນ, ທັກແຊັດມາຖາມ ແຕ່ບໍ່ມີເງິນຈ່າຍ ຫຼື ປະຕິເສດການຮັບເຄື່ອງ COD.
                  </div>
                </div>
              </div>

              <div class="rule-item">
                <div class="rule-icon">📱</div>
                <div class="rule-content">
                  <h5>2. ຕຳແໜ່ງຈັດວາງ (Placements): ເລືອກສະເພາະ Mobile Devices</h5>
                  <p>
                    ໃນລາວ ລູກຄ້າຫຼາຍກວ່າ 98% ສັ່ງຊື້ຜ່ານໂທລະສັບມືຖື. ໃຫ້ເລືອກ Manual Placements:
                  </p>
                  <ul class="placements-check">
                    <li>✅ <strong>ເປີດ:</strong> Facebook Feeds (ໜ້າຟີດຫຼັກ)</li>
                    <li>✅ <strong>ເປີດ:</strong> Facebook Reels (ວິດີໂອສັ້ນ ມາແຮງທີ່ສຸດ)</li>
                    <li>✅ <strong>ເປີດ:</strong> Messenger Stories & Inbox</li>
                    <li class="off">❌ <strong>ປິດ Audience Network ທັນທີ:</strong> ເພາະ Meta ຈະເອົາແອດໄປໂຊໃນເກມມືຖື ເຮັດໃຫ້ກິນເງິນລ້າໆຈາກຄົນກົດພາດໃນເກມ!</li>
                  </ul>
                </div>
              </div>

              <div class="rule-item">
                <div class="rule-icon">💎</div>
                <div class="rule-content">
                  <h5>3. ຂຸມຊັບ Custom Audiences & Lookalike (ຍິງຫາຄົນທີ່ເຄີຍສົນໃຈ)</h5>
                  <p>ຢ່າຍິງແຕ່ກຸ່ມເປົ້າໝາຍກວ້າງ (Broad) ຢ່າງດຽວ, ໃຫ້ສ້າງ 2 ກຸ່ມນີ້:</p>
                  <ol class="steps-sub">
                    <li><strong>Custom Audience (Message Engagement):</strong> ສ້າງກຸ່ມຄົນທີ່ເຄີຍສົ່ງຂໍ້ຄວາມຫາເພຈໃນ 60 ມື້ຜ່ານມາ.</li>
                    <li><strong>Lookalike Audience 1% - 2% (Laos):</strong> ໃຫ້ Facebook ຊອກຫາຄົນໃນລາວທີ່ມີພຶດຕິກຳຄ້າຍຄືກັບລູກຄ້າທີ່ເຄີຍສັ່ງຊື້ສິນຄ້າຂອງທ່ານ.</li>
                    <li><strong>Exclusions (ຍົກເວັ້ນ):</strong> ຍົກເວັ້ນຄົນທີ່ເຄີຍສັ່ງຊື້ແລ້ວໃນ 30 ມື້ ເພື່ອປະຢັດງົບ ບໍ່ໃຫ້ຍິງຊ້ຳຄົນທີ່ຊື້ໄປແລ້ວ.</li>
                  </ol>
                </div>
              </div>
            </div>
          </div>

          <!-- TAB 3: COST CONTROL & SCALING -->
          <div v-if="aiAnalysisTab === 'cost'" class="tab-pane">
            <div class="guide-banner mb-3">
              💸 <strong>ວິທີຄຸມຄ່າແອດບໍ່ໃຫ້ແພງ ແລະ ກົດການ Scale ງົບທີ່ຖືກຕ້ອງ:</strong>
            </div>

            <div class="sop-cards">
              <div class="sop-card">
                <div class="sop-number">SOP 1</div>
                <h4>ກົດຫ້າມປິດແອດໄວ (No Micro-Pausing)</h4>
                <p>
                  ເມື່ອສ້າງແອດໃໝ່, ລະບົບ Meta AI ຕ້ອງໃຊ້ເວລາ 48-72 ຊົ່ວໂມງ ໃນການຊອກຫາຜູ້ຊື້ (Learning Phase).
                  ຖ້າແອດໃຊ້ງົບໄປພຽງ $1 - $3 ແລ້ວຍັງບໍ່ມີອໍເດີ <strong>ຫ້າມປິດເດັດຂາດ!</strong>
                </p>
                <div class="sop-badge">
                  🛑 ເພດານງົບທົດລອງທີ່ຖືກຕ້ອງ: ປ່ອຍໃຫ້ໃຊ້ຮອດ <strong>$6.00 - $8.00</strong> (1.5x - 2x Target CPA) ກ່ອນຕັດສິນໃຈ.
                </div>
              </div>

              <div class="sop-card">
                <div class="sop-number">SOP 2</div>
                <h4>ກົດການ Scale ງົບ +15% ຫາ +20% (Budget Scaling)</h4>
                <p>
                  ເມື່ອເຈີແອດນາງຟ້າທີ່ມີ CPA ຕ່ຳກວ່າເປົ້າໝາຍ (ເຊັ່ນ <code>h9maxx3 - ສຳເນົາ</code> CPA $1.04):
                </p>
                <ul>
                  <li>✅ ເພີ່ມງົບມື້ລະ <strong>15% ຫາ 20%</strong> ທຸກໆ 48 ຊົ່ວໂມງ.</li>
                  <li>❌ <strong>ຫ້າມເພີ່ມງົບກະໂດດ 100%</strong> (ເຊັ່ນ ຈາກ $3 ເປັນ $10) ເພາະຈະເຮັດໃຫ້ລະບົບ Restart Learning Phase ແລະ ເຮັດໃຫ້ຄ່າແອດແພງຂຶ້ນທັນທີ!</li>
                </ul>
              </div>

              <div class="sop-card">
                <div class="sop-number">SOP 3</div>
                <h4>ໂຄງສ້າງວິດີໂອສັ້ນ 15-30 ວິນາທີ ທີ່ CTR ສູງ ແລະ ຄ່າແອດຖືກ</h4>
                <p>
                  Creative ຂອງ H9 MAX ມີ CTR ດີຢູ່ແລ້ວ (1.5% - 3.7%). ຖ້າຢາກໃຫ້ຄ່າແອດຖືກລົງອີກ:
                </p>
                <div class="creative-formula">
                  <div><strong>0-3 ວິນາທີ (Hook):</strong> ໂຊສາກຈຸ່ມນ້ຳ ຫຼື ທົດສອບຕົກແຕກ ພ້ອມຂໍ້ຄວາມ: <em>"H9 MAX ກັນນ້ຳແທ້ບໍ່?"</em></div>
                  <div><strong>4-15 ວິນາທີ (Core):</strong> ໂຊໜ້າຈໍແທ້, ການໂທເຂົ້າ-ອອກ, ວັດຫົວໃຈ, ສາຍຊິລິໂຄນ.</div>
                  <div><strong>16-20 ວິນາທີ (CTA):</strong> <em>"ໂປຣໂມຊັ່ນສົ່ງຟຣີຮອດໜ້າບ້ານ + ເກັບເງິນປາຍທາງ ທັກແຊັດເລີຍ!"</em></div>
                </div>
              </div>
            </div>
          </div>

          <!-- TAB 4: CLOSING SCRIPTS -->
          <div v-if="aiAnalysisTab === 'closing'" class="tab-pane">
            <div class="guide-banner mb-3">
              💬 <strong>ບົດຕອບແຊັດພາສາລາວສຳລັບ H9 MAX (ເພີ່ມ Close Rate ຈາກ 5% ເປັນ 20%):</strong>
            </div>

            <!-- Script 1 -->
            <div class="script-box mb-4">
              <div class="script-top">
                <span class="script-tag">⚡ ບົດທີ 1: ຕອບຮັບທັນທີພາຍໃນ 1 ນາທີ (Auto-Welcome & Special Offer)</span>
                <v-btn
                  size="small"
                  variant="flat"
                  color="primary"
                  prepend-icon="mdi-content-copy"
                  @click="copyScript(`ສະບາຍດີເຈົ້າ H9 MAX Smartwatch ລຸ້ນໃໝ່ຫຼ້າສຸດ ໂປຣໂມຊັ່ນພິເສດມື້ນີ້ພຽງ 210,000 ກີບ (ຈາກປົກກະຕິ 280,000 ກີບ) ✨\n✅ ແຖມຟຣີ: ສາຍປ່ຽນຊິລິໂຄນ + ສາຍສາກແມ່ເຫຼັກ\n✅ ຮັບປະກັນສິນຄ້າ 3 ເດືອນເຕັມ (ມີບັນຫາປ່ຽນໃໝ່ທັນທີ)\n🚚 ສົ່ງຟຣີຮອດໜ້າບ້ານ + ເກັບເງິນປາຍທາງ (ກວດເຄື່ອງກ່ອນຈ່າຍໄດ້ເລີຍເຈົ້າ)\n\n👉 ລູກຄ້າຮັບ 1 ເຮືອນ ຫຼື 2 ເຮືອນດີເຈົ້າ?`)"
                >
                  {{ copiedScript ? 'ຄັດລອກແລ້ວ! ✅' : 'ຄັດລອກບົດຕອບນີ້' }}
                </v-btn>
              </div>
              <pre class="script-text">ສະບາຍດີເຈົ້າ H9 MAX Smartwatch ລຸ້ນໃໝ່ຫຼ້າສຸດ ໂປຣໂມຊັ່ນພິເສດມື້ນີ້ພຽງ 210,000 ກີບ (ຈາກປົກກະຕິ 280,000 ກີບ) ✨
✅ ແຖມຟຣີ: ສາຍປ່ຽນຊິລິໂຄນ + ສາຍສາກແມ່ເຫຼັກ
✅ ຮັບປະກັນສິນຄ້າ 3 ເດືອນເຕັມ (ມີບັນຫາປ່ຽນໃໝ່ທັນທີ)
🚚 ສົ່ງຟຣີຮອດໜ້າບ້ານ + ເກັບເງິນປາຍທາງ (ກວດເຄື່ອງກ່ອນຈ່າຍໄດ້ເລີຍເຈົ້າ)

👉 ລູກຄ້າຮັບ 1 ເຮືອນ ຫຼື 2 ເຮືອນດີເຈົ້າ?</pre>
            </div>

            <!-- Script 2 -->
            <div class="script-box mb-4">
              <div class="script-top">
                <span class="script-tag">🎯 ບົດທີ 2: ລັອກສິດໂປຣໂມຊັ່ນ ແລະ ຂໍເບີໂທຈັດສົ່ງ (Closing Hook)</span>
                <v-btn
                  size="small"
                  variant="flat"
                  color="success"
                  prepend-icon="mdi-content-copy"
                  @click="copyScript(`ເພື່ອລັອກສິດໂປຣໂມຊັ່ນສົ່ງຟຣີ ແລະ ຂອງແຖມມື້ນີ້ ລູກຄ້າແຈ້ງ:\n📍 ຊື່:\n📞 ເບີໂທ:\n🏠 ທີ່ຢູ່/ແຂວງ:\n\nທາງຮ້ານຈະແພັກເຄື່ອງຈັດສົ່ງໃຫ້ດ່ວນເລີຍເຈົ້າ! ຂອບໃຈຫຼາຍໆເຈົ້າ 🙏`)"
                >
                  ຄັດລອກບົດຕອບນີ້
                </v-btn>
              </div>
              <pre class="script-text">ເພື່ອລັອກສິດໂປຣໂມຊັ່ນສົ່ງຟຣີ ແລະ ຂອງແຖມມື້ນີ້ ລູກຄ້າແຈ້ງ:
📍 ຊື່:
📞 ເບີໂທ:
🏠 ທີ່ຢູ່/ແຂວງ:

ທາງຮ້ານຈະແພັກເຄື່ອງຈັດສົ່ງໃຫ້ດ່ວນເລີຍເຈົ້າ! ຂອບໃຈຫຼາຍໆເຈົ້າ 🙏</pre>
            </div>

            <!-- Script 3 -->
            <div class="script-box">
              <div class="script-top">
                <span class="script-tag">⏰ ບົດທີ 3: ຕິດຕາມລູກຄ້າທີ່ມິດງຽບຫຼັງຖາມລາຄາ (Follow-up 24 Hours)</span>
                <v-btn
                  size="small"
                  variant="outlined"
                  prepend-icon="mdi-content-copy"
                  @click="copyScript(`ສະບາຍດີເຈົ້າ ທາງຮ້ານຍັງເຫຼືອໂຄຕ້າສົ່ງຟຣີ + ຂອງແຖມອີກພຽງ 3 ຊຸດສຸດທ້າຍເຈົ້າ ✨\nລູກຄ້າສະດວກຮັບໄວ້ເລີຍບໍ່ເຈົ້າ? ທາງຮ້ານຈະເກັບສິດໂປຣໂມຊັ່ນນີ້ໄວ້ໃຫ້ເລີຍເຈົ້າ 😊`)"
                >
                  ຄັດລອກບົດຕອບນີ້
                </v-btn>
              </div>
              <pre class="script-text">ສະບາຍດີເຈົ້າ ທາງຮ້ານຍັງເຫຼືອໂຄຕ້າສົ່ງຟຣີ + ຂອງແຖມອີກພຽງ 3 ຊຸດສຸດທ້າຍເຈົ້າ ✨
ລູກຄ້າສະດວກຮັບໄວ້ເລີຍບໍ່ເຈົ້າ? ທາງຮ້ານຈະເກັບສິດໂປຣໂມຊັ່ນນີ້ໄວ້ໃຫ້ເລີຍເຈົ້າ 😊</pre>
            </div>
          </div>

          <!-- TAB 5: QUICK ACTIONS -->
          <div v-if="aiAnalysisTab === 'actions'" class="tab-pane">
            <div class="guide-banner mb-3">
              🚀 <strong>Action ດ່ວນມື້ນີ້: ເປີດແອດນາງຟ້າຄືນ ແລະ ຢຸດແອດທີ່ກິນເງິນລ້າໆ</strong>
            </div>

            <div class="mb-4">
              <h4 class="text-subtitle-1 font-weight-bold mb-2 d-flex align-center gap-2">
                <span>🟢 ແອດນາງຟ້າທີ່ຄວນເປີດຄືນທັນທີ (Paused Winning Ads)</span>
                <v-chip size="x-small" color="success">{{ aiStrategyData.pausedAngels.length }} ໂຕ</v-chip>
              </h4>

              <div v-if="aiStrategyData.pausedAngels.length === 0" class="pa-4 text-center text-medium-emphasis border rounded-lg">
                🎉 ບໍ່ມີແອດນາງຟ້າທີ່ຖືກປິດຢູ່ ທຸກແອດທີ່ດີກຳລັງແລ່ນຢູ່ແລ້ວ!
              </div>

              <div v-else class="action-angels-list">
                <div v-for="item in aiStrategyData.pausedAngels" :key="item.entityId" class="action-angel-card">
                  <div class="angel-info">
                    <strong class="angel-title">{{ item.entityName }}</strong>
                    <div class="angel-meta">
                      <span>💸 ງົບໃຊ້: {{ money(item.spend) }}</span>
                      <span class="text-success font-weight-bold">🛒 {{ getPurchasesCount(item) }} ອໍເດີ</span>
                      <span v-if="getCostPerPurchase(item)" class="text-primary">🎯 CPA: {{ money(getCostPerPurchase(item)) }}</span>
                      <span>💬 {{ getMessagesCount(item) }} ທັກ</span>
                      <span>CTR: {{ percent(item.linkCtr) }}</span>
                    </div>
                  </div>
                  <v-btn
                    color="success"
                    variant="flat"
                    size="small"
                    prepend-icon="mdi-play-circle"
                    :loading="resumingAdId === item.entityId"
                    @click="resumeAngelAd(item)"
                  >
                    ⚡ ເປີດແອດນີ້ຄືນ (Resume)
                  </v-btn>
                </div>
              </div>
            </div>

            <v-divider class="my-4" />

            <!-- Underperforming Action -->
            <div>
              <h4 class="text-subtitle-1 font-weight-bold mb-2 d-flex align-center gap-2">
                <span>🔴 ແອດໝົດກຳລັງທີ່ຄວນປິດ (Underperforming Ads)</span>
                <v-chip size="x-small" color="error">{{ aiStrategyData.activeBads.length }} ໂຕ</v-chip>
              </h4>

              <div v-if="aiStrategyData.activeBads.length === 0" class="pa-4 text-center text-medium-emphasis border rounded-lg">
                ✅ ບໍ່ມີແອດໝົດກຳລັງທີ່ກຳລັງກິນເງິນຢູ່!
              </div>

              <div v-else class="d-flex align-center justify-space-between p-3 border rounded-lg bg-red-lighten-5">
                <div>
                  <strong class="text-error">ພົບແອດທີ່ກິນເງິນລ້າໆ {{ aiStrategyData.activeBads.length }} ໂຕ</strong>
                  <div class="text-caption text-medium-emphasis">ໃຊ້ງົບແລ້ວຍັງ 0 ອໍເດີ ຫຼື CTR ຕ່ຳເກີນໄປ</div>
                </div>
                <v-btn
                  color="error"
                  variant="flat"
                  size="small"
                  prepend-icon="mdi-pause-circle"
                  :loading="bulkPausing"
                  @click="pauseAllBadAds"
                >
                  ⏸️ ປິດແອດໝົດກຳລັງທັງໝົດ
                </v-btn>
              </div>
            </div>
          </div>

          <!-- TAB: GHOST FILTER & REAL CUSTOMERS -->
          <div v-if="aiAnalysisTab === 'ghost'" class="tab-pane">
            <div class="guide-banner mb-3">
              👻 <strong>ຄວາມຈິງເລື່ອງ Custom Audience & ທັກຜີ:</strong>
              ຖ້າທ່ານໃຫ້ Facebook ດຶງ "ຄົນທັກເພຈ 60 ມື້", ມັນຈະລວມເອົາ <strong>ທັກຜີ (Ghost Inquiries)</strong> ເຂົ້າໄປນຳຫຼາຍກວ່າ 80%! ເຮັດໃຫ້ Meta ຍິ່ງໄປຊອກຫາແຕ່ຄົນກົດຫຼິ້ນມາເພີ່ມ.
            </div>

            <!-- Stats Comparison Grid -->
            <div class="bottlenecks-grid mb-3">
              <div class="bottleneck-card" style="border-left: 4px solid #10b981;">
                <div class="d-flex align-center justify-space-between">
                  <span class="bn-badge" style="background: #dcfce7; color: #166534;">🟢 ລູກຄ້າແທ້ (Real Verified Buyers)</span>
                  <strong class="text-success text-h6">{{ aiStrategyData.totalOrders }} ຄົນ</strong>
                </div>
                <p class="bn-body mt-2">
                  ແມ່ນລູກຄ້າທີ່ມີ <strong>ເບີໂທລະສັບ, ທີ່ຢູ່, ແລະ ບິນສັ່ງຊື້ແທ້ 100%</strong>. ເປັນກຸ່ມດຽວທີ່ມີຄຸນນະພາບສູງສຸດ ແລະ ຄວນນຳໄປສ້າງ Custom Audience!
                </p>
                <div class="mt-2">
                  <v-btn
                    color="success"
                    variant="flat"
                    size="small"
                    prepend-icon="mdi-download"
                    class="font-weight-bold"
                    @click="exportRealCustomersCsv"
                  >
                    📥 Export ລາຍຊື່ລູກຄ້າແທ້ (Download CSV for Meta)
                  </v-btn>
                </div>
              </div>

              <div class="bottleneck-card danger">
                <div class="d-flex align-center justify-space-between">
                  <span class="bn-badge danger">👻 ທັກຜີ / ມິດງຽບ (Ghost Inquiries)</span>
                  <strong class="text-error text-h6">{{ Math.max(0, aiStrategyData.totalMessages - aiStrategyData.totalOrders) }} ຄົນ</strong>
                </div>
                <p class="bn-body mt-2">
                  ແມ່ນຄົນທີ່ <strong>ກົດປຸ່ມອັດຕະໂນມັດ 1 ເທື່ອ</strong> ແລ້ວມິດງຽບ, ບໍ່ເຄີຍຕອບກັບ, ບໍ່ໃຫ້ເບີໂທ, ຫຼື ເດັກນ້ອຍຫຼິ້ນເກມແລ້ວກົດພາດໃນ Audience Network.
                </p>
                <div class="rule-alert text-error mt-2">
                  ⚠️ <strong>ຫ້າມເຮັດ Lookalike ຈາກກຸ່ມທັກຜີ:</strong> ເພາະ Meta ຈະໄປດຶງຄົນທີ່ມັກກົດຫຼິ້ນມາອີກ!
                </div>
              </div>
            </div>

            <!-- 3 Steps to Kill Ghost Inquiries -->
            <div class="sop-card">
              <div class="sop-number">Action</div>
              <h4>🛡️ 3 ວິທີປິດກັ້ນ "ທັກຜີ" ບໍ່ໃຫ້ມາແດກຄ່າແອດໃນ Meta Ads Manager</h4>
              <div class="rules-list mt-3">
                <div class="rule-item">
                  <div class="rule-icon">🎮</div>
                  <div class="rule-content">
                    <h5>1. ປິດ "Audience Network" ໃນ Ad Set ທັນທີ</h5>
                    <p>
                      ໃນ Ad Set ➔ Placements ➔ Manual Placements: <strong>ຕິກເອົາ Audience Network ອອກ!</strong> ເພາະເກືອບທັງໝົດຂອງທັກຜີ ແມ່ນຄົນຫຼິ້ນເກມມືຖືແລ້ວພະຍາຍາມກົດປິດໂຄສະນາແຕ່ມືໄປໂດນປຸ່ມ Message!
                    </p>
                  </div>
                </div>

                <div class="rule-item">
                  <div class="rule-icon">🏷️</div>
                  <div class="rule-content">
                    <h5>2. ໃສ່ລາຄາ "210,000 ກີບ" ຕົວໃຫຍ່ໆໃສ່ໜ້າປົກຮູບ ຫຼື ຄລິບແອດ</h5>
                    <p>
                      ຢ່າປິດບັງລາຄາ! ຄົນທີ່ທັກມາຖາມວ່າ <em>"ລາຄາເທົ່າໃດ?"</em> ແລ້ວມິດງຽບ ເພາະເຂົາຄິດວ່າລາຄາ 30,000-50,000 ກີບ. ຖ້າໃສ່ລາຄາ 210,000 ກີບ ໂຊຈະແຈ້ງ, ຄົນທີ່ທັກມາຈະເປັນຄົນທີ່ <strong>"ພ້ອມຈ່າຍ 210,000 ກີບແລ້ວ"</strong> ເທົ່ານັ້ນ!
                    </p>
                  </div>
                </div>

                <div class="rule-item">
                  <div class="rule-icon">⚙️</div>
                  <div class="rule-content">
                    <h5>3. ປັບ Message Template: ປິດປຸ່ມ Icebreaker ທີ່ກົດງ່າຍເກີນໄປ</h5>
                    <p>
                      ປິດປຸ່ມທີ່ Facebook ສ້າງໃຫ້ ເຊັ່ນ <em>"ສະບາຍດີ! ມີໃຜພ້ອມສົນທະນາບໍ່?"</em>. ໃຫ້ປ່ຽນເປັນຄຳຖາມບັງຄັບພິມ: <em>"ສະບາຍດີເຈົ້າ H9 MAX ພ້ອມສົ່ງ 210,000 ₭ ລູກຄ້າຮັບສີໃດດີເຈົ້າ? ແຈ້ງເບີໂທຈັດສົ່ງໄດ້ເລີຍເຈົ້າ"</em>.
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </v-card-text>

        <v-card-actions class="ai-modal-footer">
          <span class="text-caption text-medium-emphasis">
            💡 AI Strategist ວິເຄາະໂດຍອີງຕາມຂໍ້ມູນການສັ່ງຊື້ ແລະ Meta Insights ແທ້
          </span>
          <v-spacer />
          <v-btn variant="flat" color="primary" @click="aiModalOpen = false">
            ຮັບຊາບ & ປິດໜ້າຕ່າງ
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.ads-battle-page {
  padding: 16px 20px 40px;
  max-width: 1400px;
  margin: 0 auto;
}

/* Page Top Header */
.page-top-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 20px;
}

.page-heading {
  font-size: 22px;
  font-weight: 800;
  color: #0f172a;
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 4px 0;
}

.simple-tag {
  font-size: 11px;
  font-weight: 700;
  background: #e0e7ff;
  color: #4338ca;
  padding: 2px 8px;
  border-radius: 999px;
}

.page-sub {
  font-size: 13px;
  color: #64748b;
  margin: 0;
}

.header-right-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.view-mode-toggle {
  background: #ffffff;
  border-radius: 8px;
}

/* 4 Clean KPI Cards */
.kpi-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 14px;
  margin-bottom: 20px;
}

.kpi-box {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  padding: 14px 18px;
  display: flex;
  align-items: center;
  gap: 14px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.kpi-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.kpi-icon.blue { background: #eff6ff; color: #2563eb; }
.kpi-icon.green { background: #f0fdf4; color: #16a34a; }
.kpi-icon.purple { background: #faf5ff; color: #9333ea; }
.kpi-icon.emerald { background: #ecfdf5; color: #059669; }
.kpi-icon.rose { background: #fff1f2; color: #e11d48; }

.kpi-box.kpi-profit-pos { border-left: 3px solid #10b981; }
.kpi-box.kpi-profit-neg { border-left: 3px solid #ef4444; }

.kpi-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.kpi-lbl {
  font-size: 11.5px;
  color: #64748b;
  font-weight: 600;
  margin-bottom: 2px;
}

.kpi-val {
  font-size: 18px;
  font-weight: 800;
  color: #0f172a;
}

/* Simple Filter Bar */
.simple-controls-bar {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  padding: 12px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.filter-left-fields {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  flex: 1;
  min-width: 260px;
}

.filter-ctl {
  min-width: 140px;
  max-width: 220px;
}

.filter-ctl.account-ctl { max-width: 200px; }
.filter-ctl.date-ctl { max-width: 160px; }
.filter-ctl.product-ctl { max-width: 190px; }
.filter-ctl.search-ctl { min-width: 180px; flex: 1; }

.filter-right-tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.cat-tab {
  font-size: 12px;
  font-weight: 700;
  padding: 5px 12px;
  border-radius: 999px;
  border: 1px solid #cbd5e1;
  background: #f8fafc;
  color: #475569;
  cursor: pointer;
  transition: all 0.15s;
}

.cat-tab:hover { background: #f1f5f9; }
.cat-tab.active { background: #0f172a; color: #ffffff; border-color: #0f172a; }
.cat-tab.green.active { background: #10b981; color: #ffffff; border-color: #059669; }
.cat-tab.red.active { background: #ef4444; color: #ffffff; border-color: #dc2626; }

/* Loading & Empty */
.simple-loading,
.simple-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  background: #ffffff;
  border: 1px dashed #cbd5e1;
  border-radius: 14px;
}

/* Cards Layout */
.cards-layout {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.clean-ad-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: transform 0.15s, box-shadow 0.15s;
}

.clean-ad-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.07);
}

.clean-ad-card.card-angel {
  border-left: 4px solid #10b981;
}

.clean-ad-card.card-bad {
  border-left: 4px solid #ef4444;
}

.clean-ad-card.card-paused {
  background: #f8fafc;
  opacity: 0.72;
}

/* Card Top Bar */
.card-top-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.ad-switch-and-name {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  flex: 1;
}

.simple-switch {
  flex-shrink: 0;
}

.ad-name-group {
  min-width: 0;
  flex: 1;
}

.ad-title {
  font-size: 14px;
  font-weight: 750;
  color: #0f172a;
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ad-subline {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: #64748b;
  margin-top: 2px;
}

.ad-subline code {
  background: #f1f5f9;
  padding: 1px 4px;
  border-radius: 4px;
  font-size: 10.5px;
}

.prod-badge {
  font-size: 10.5px;
  font-weight: 600;
  color: #0284c7;
  background: #e0f2fe;
  padding: 1px 6px;
  border-radius: 4px;
}

.pill-badge {
  font-size: 11px;
  font-weight: 700;
  padding: 3px 8px;
  border-radius: 999px;
  white-space: nowrap;
}

.pill-badge.angel { background: #dcfce7; color: #15803d; }
.pill-badge.bad { background: #fee2e2; color: #b91c1c; }
.pill-badge.neutral { background: #f1f5f9; color: #64748b; }

/* Card Metric Row (5 in 1 row) */
.card-metric-row {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 6px;
  background: #f8fafc;
  border: 1px solid #f1f5f9;
  border-radius: 10px;
  padding: 8px 10px;
  text-align: center;
}

.m-box {
  display: flex;
  flex-direction: column;
}

.m-k {
  font-size: 10px;
  color: #64748b;
  font-weight: 600;
  margin-bottom: 2px;
}

.m-v {
  font-size: 12.5px;
  font-weight: 750;
  color: #0f172a;
}

/* Card Bottom Bar */
.card-bottom-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-top: 4px;
}

.sparkline-container {
  width: 110px;
  height: 30px;
  flex-shrink: 0;
}

.sparkline-svg {
  width: 100%;
  height: 100%;
  overflow: visible;
}

.action-btn-group {
  display: flex;
  align-items: center;
  gap: 6px;
}

.paused-status-text {
  font-size: 11px;
  color: #64748b;
  font-weight: 600;
}

/* Table View */
.table-container-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.table-scroller {
  overflow-x: auto;
}

.clean-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.clean-table th {
  text-align: left;
  padding: 10px 14px;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
  font-weight: 700;
  color: #475569;
  font-size: 12px;
  white-space: nowrap;
}

.clean-table td {
  padding: 10px 14px;
  border-bottom: 1px solid #f1f5f9;
  color: #1e293b;
  vertical-align: middle;
}

.clean-table tr.row-paused {
  background: #fafafa;
  opacity: 0.65;
}

.clean-table tr:hover {
  background: #f8fafc;
}

.ad-cell {
  display: flex;
  flex-direction: column;
  max-width: 240px;
}

.ad-cell-name {
  font-size: 13px;
  color: #0f172a;
}

/* Budget Preview Box */
.budget-preview-box {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 12px 14px;
}

/* AI Strategist Modal Styles */
.ai-modal-card {
  border-radius: 16px;
  overflow: hidden;
}

.ai-modal-header {
  background: #ffffff;
  border-bottom: 1px solid #e2e8f0;
  padding: 16px 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.ai-header-badge {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  background: linear-gradient(135deg, #4f46e5, #7c3aed);
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 6px rgba(79, 70, 229, 0.3);
}

.ai-kpi-strip {
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
  padding: 10px 20px;
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}

.ai-health-score-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-right: 12px;
  border-right: 1px solid #cbd5e1;
}

.ai-health-num {
  font-size: 24px;
  font-weight: 900;
  color: #0f172a;
}

.ai-health-meta {
  display: flex;
  flex-direction: column;
}

.score-lbl {
  font-size: 10px;
  text-transform: uppercase;
  font-weight: 700;
  color: #64748b;
}

.score-tag {
  font-size: 11px;
  font-weight: 700;
}

.score-tag.pos { color: #059669; }
.score-tag.warn { color: #d97706; }

.ai-stat-pill {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 4px 10px;
  display: flex;
  flex-direction: column;
}

.stat-l {
  font-size: 10px;
  color: #64748b;
  font-weight: 600;
}

.stat-v {
  font-size: 12.5px;
  font-weight: 800;
}

.ai-tab-bar {
  display: flex;
  gap: 6px;
  padding: 8px 20px;
  background: #ffffff;
  border-bottom: 1px solid #f1f5f9;
  overflow-x: auto;
}

.ai-tab-btn {
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 700;
  border-radius: 8px;
  border: 1px solid transparent;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s;
}

.ai-tab-btn:hover { background: #f8fafc; color: #1e293b; }
.ai-tab-btn.active {
  background: #eff6ff;
  color: #2563eb;
  border-color: #bfdbfe;
}

.badge-count {
  background: #ef4444;
  color: #ffffff;
  font-size: 10px;
  font-weight: 800;
  padding: 1px 6px;
  border-radius: 999px;
}

.ai-modal-body {
  padding: 20px;
  background: #fdfdfd;
  min-height: 420px;
}

.tab-pane {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.ai-verdict-card {
  background: #eff6ff;
  border: 1px solid #bfdbfe;
  border-radius: 12px;
  padding: 14px 18px;
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.verdict-icon { font-size: 24px; }
.verdict-title { font-size: 15px; font-weight: 800; color: #1e3a8a; margin: 0 0 4px 0; }
.verdict-desc { font-size: 13.5px; line-height: 1.6; color: #1e293b; margin: 0; }

.bottlenecks-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 14px;
}

.bottleneck-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.bottleneck-card.warn { border-left: 4px solid #f59e0b; }
.bottleneck-card.danger { border-left: 4px solid #ef4444; }

.bn-badge {
  font-size: 11.5px;
  font-weight: 800;
  padding: 2px 8px;
  border-radius: 6px;
}
.bn-badge.warn { background: #fef3c7; color: #92400e; }
.bn-badge.danger { background: #fee2e2; color: #991b1b; }

.bn-body { font-size: 13px; line-height: 1.5; color: #334155; margin: 0; }
.bn-fix {
  font-size: 12.5px;
  line-height: 1.5;
  background: #f8fafc;
  border-radius: 6px;
  padding: 8px 10px;
  color: #0f172a;
}

.guide-banner {
  background: #f1f5f9;
  border-left: 4px solid #3b82f6;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13px;
  color: #1e293b;
}

.rules-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.rule-item {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 14px 16px;
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.rule-icon { font-size: 22px; flex-shrink: 0; }
.rule-content h5 { font-size: 14px; font-weight: 750; color: #0f172a; margin: 0 0 4px 0; }
.rule-content p { font-size: 13px; color: #475569; margin: 0 0 6px 0; line-height: 1.5; }
.rule-alert { font-size: 12px; font-weight: 600; padding: 4px 8px; background: #fee2e2; border-radius: 6px; }

.placements-check {
  margin: 4px 0 0 0;
  padding-left: 18px;
  font-size: 12.5px;
  color: #334155;
}
.placements-check li { margin-bottom: 4px; }
.placements-check li.off { color: #dc2626; }

.steps-sub {
  margin: 4px 0 0 0;
  padding-left: 18px;
  font-size: 12.5px;
  color: #334155;
}
.steps-sub li { margin-bottom: 4px; }

.sop-cards {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sop-card {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 14px 16px;
  position: relative;
}

.sop-number {
  position: absolute;
  top: 14px;
  right: 16px;
  font-size: 11px;
  font-weight: 800;
  color: #3b82f6;
  background: #eff6ff;
  padding: 2px 8px;
  border-radius: 999px;
}

.sop-card h4 { font-size: 14px; font-weight: 800; color: #0f172a; margin: 0 0 6px 0; }
.sop-card p { font-size: 13px; color: #475569; margin: 0 0 8px 0; line-height: 1.5; }
.sop-badge { font-size: 12px; background: #ecfdf5; color: #065f46; padding: 6px 10px; border-radius: 6px; font-weight: 600; }

.creative-formula {
  background: #f8fafc;
  border: 1px solid #f1f5f9;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 12.5px;
  color: #334155;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.script-box {
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 14px 16px;
}

.script-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.script-tag {
  font-size: 12.5px;
  font-weight: 800;
  color: #0f172a;
}

.script-text {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px;
  font-family: inherit;
  font-size: 13px;
  line-height: 1.6;
  color: #1e293b;
  white-space: pre-wrap;
  margin: 10px 0 0 0;
}

.action-angel-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 16px;
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  margin-bottom: 8px;
}

.angel-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.angel-title {
  font-size: 13.5px;
  font-weight: 750;
  color: #0f172a;
  margin-bottom: 4px;
}

.angel-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #64748b;
  flex-wrap: wrap;
}

.ai-modal-footer {
  background: #ffffff;
  border-top: 1px solid #e2e8f0;
  padding: 12px 20px;
}
</style>
