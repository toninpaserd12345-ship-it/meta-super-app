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
const bulkScaling = ref(false)
const notice = ref('')
const success = ref('')
const search = ref('')
const selectedProductId = ref<string>('all')

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

const summary = computed(() => report.value?.summary)

function isMessagingAction(type?: string): boolean {
  if (!type) return false
  const t = type.toLowerCase()
  return t.includes('messaging') || t.includes('conversation') || t.includes('reply') || t.includes('contact')
}

function getMessagesCount(item?: MetaInsightItem | null): number {
  if (!item) return 0
  if (item.messagesCount != null && item.messagesCount > 0) return item.messagesCount
  if (isMessagingAction(item.results?.type) && item.results?.value != null && item.results.value > 0) return item.results.value
  return 0
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

function isAngelAd(item: MetaInsightItem): { isAngel: boolean; score: number; reason: string } {
  if (!item.impressions && !item.spend) return { isAngel: false, score: 0, reason: '' }
  const purchases = getPurchasesCount(item)
  const cost = getCostPerPurchase(item)
  const messages = getMessagesCount(item)
  const ctr = item.linkCtr || 0
  const roas = item.roas || 0
  const targetCpa = currencyOffset.value === 1 ? 100000 : 5

  if (purchases >= 3 && (cost == null || cost <= targetCpa * 1.35)) {
    return {
      isAngel: true,
      score: 100,
      reason: l({
        lo: `ປິດໄດ້ ${purchases} ອໍເດີ${cost != null ? ` (CPA ${money(cost)}/ອໍເດີ)` : ''}${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''}`,
        th: `ปิดได้ ${purchases} ออเดอร์${cost != null ? ` (CPA ${money(cost)}/ออเดอร์)` : ''}${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''}`,
        en: `${purchases} orders${cost != null ? ` (CPA ${money(cost)}/order)` : ''}${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''}`,
      }),
    }
  }

  if (purchases >= 1 && (roas >= 2.0 || (cost != null && cost <= targetCpa))) {
    return {
      isAngel: true,
      score: 85,
      reason: l({
        lo: `ປິດໄດ້ ${purchases} ອໍເດີ${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''} (ກຳໄລສູງ / ພ້ອມຂະຫຍາຍງົບ)`,
        th: `ปิดได้ ${purchases} ออเดอร์${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''} (กำไรสูง / พร้อมขยายงบ)`,
        en: `${purchases} orders${roas > 0 ? ` · ROAS ${fmt(roas, 2)}x` : ''} (High conversion)`,
      }),
    }
  }

  if (purchases === 0 && messages >= 5 && ctr >= 2.0) {
    return {
      isAngel: true,
      score: 75,
      reason: l({
        lo: `ຄົນທັກ ${messages} ຄົນ (CTR ${percent(ctr)})`,
        th: `คนทัก ${messages} คน (CTR ${percent(ctr)})`,
        en: `${messages} messages (CTR ${percent(ctr)})`,
      }),
    }
  }

  return { isAngel: false, score: 0, reason: '' }
}

function isBadAd(item: MetaInsightItem): { isBad: boolean; severity: 'high' | 'medium'; reasons: string[] } {
  if (!item.impressions && !item.spend) return { isBad: false, severity: 'medium', reasons: [] }
  const purchases = getPurchasesCount(item)
  const messages = getMessagesCount(item)
  const ctr = item.linkCtr || 0
  const spend = item.spend || 0
  const freq = item.frequency || 0
  const minSpendThreshold = currencyOffset.value === 1 ? 50000 : 3
  const reasons: string[] = []

  // 1. Wasted spend without conversions
  if (purchases === 0 && spend >= minSpendThreshold) {
    reasons.push(l({
      lo: `ໃຊ້ງົບ ${money(spend)} ແລ້ວຍັງບໍ່ມີການຊື້ເລີຍ (0 ອໍເດີ)`,
      th: `ใช้งบ ${money(spend)} แล้วยังไม่มีการซื้อเลย (0 ออเดอร์)`,
      en: `Spent ${money(spend)} with 0 purchases`,
    }))
  }

  // 2. Creative Fatigue: Low CTR
  if (ctr > 0 && ctr < 1.0) {
    reasons.push(l({
      lo: `CTR ຕ່ຳພຽງ ${percent(ctr)}—ຄົນເລີ່ມເບື່ອຮູບ/ວິດີໂອນີ້ (Creative Fatigue)`,
      th: `CTR ต่ำเพียง ${percent(ctr)}—คนเริ่มเบื่อภาพ/วิดีโอนี้ (Creative Fatigue)`,
      en: `Low CTR (${percent(ctr)})—creative fatigue detected`,
    }))
  }

  // 3. High Frequency saturation
  if (freq >= 2.2) {
    reasons.push(l({
      lo: `Frequency ສູງ (${fmt(freq, 2)}x)—Audience ເລີ່ມເຫັນຊ້ຳຫຼາຍເກີນໄປ`,
      th: `Frequency สูง (${fmt(freq, 2)}x)—กลุ่มเป้าหมายเริ่มเห็นซ้ำมากเกินไป`,
      en: `High Frequency (${fmt(freq, 2)}x)—audience saturation`,
    }))
  }

  // 4. Inquiries with zero sales after significant spend
  if (purchases === 0 && messages > 0 && spend >= minSpendThreshold * 1.5) {
    reasons.push(l({
      lo: `ທັກ ${messages} ຄົນແຕ່ປິດການຂາຍບໍ່ໄດ້ (ກິນງົບ ${money(spend)})`,
      th: `ทัก ${messages} คนแต่ปิดการขายไม่ได้ (ใช้งบ ${money(spend)})`,
      en: `${messages} inquiries but 0 closed orders (${money(spend)} spent)`,
    }))
  }

  const isHighSeverity = (purchases === 0 && spend >= minSpendThreshold * 2) || (reasons.length >= 2)
  return {
    isBad: reasons.length > 0 && purchases === 0,
    severity: isHighSeverity ? 'high' : 'medium',
    reasons,
  }
}

// Filtered base items by search & product
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

// Winning / Angel Ads List
const angelAds = computed(() => {
  return filteredBaseItems.value
    .filter(it => isAngelAd(it).isAngel)
    .sort((a, b) => {
      const pDiff = (getPurchasesCount(b) || 0) - (getPurchasesCount(a) || 0)
      if (pDiff !== 0) return pDiff
      const cpaA = getCostPerPurchase(a) ?? 999999
      const cpaB = getCostPerPurchase(b) ?? 999999
      return cpaA - cpaB
    })
})

// Bad / Underperforming Ads List
const badAds = computed(() => {
  return filteredBaseItems.value
    .filter(it => isBadAd(it).isBad)
    .sort((a, b) => (b.spend || 0) - (a.spend || 0))
})

// Head-to-Head Comparative Aggregates
const battleStats = computed(() => {
  // Angels
  const angelCount = angelAds.value.length
  let angelSpend = 0
  let angelPurchases = 0
  let angelMessages = 0
  let angelCtrSum = 0

  for (const it of angelAds.value) {
    angelSpend += it.spend || 0
    angelPurchases += getPurchasesCount(it) || 0
    angelMessages += getMessagesCount(it) || 0
    angelCtrSum += it.linkCtr || 0
  }

  const avgAngelCPA = angelPurchases > 0 ? angelSpend / angelPurchases : null
  const avgAngelCTR = angelCount > 0 ? angelCtrSum / angelCount : 0

  // Bad / Wasters
  const badCount = badAds.value.length
  let badSpend = 0
  let badPurchases = 0
  let badMessages = 0
  let badCtrSum = 0

  for (const it of badAds.value) {
    badSpend += it.spend || 0
    badPurchases += getPurchasesCount(it) || 0
    badMessages += getMessagesCount(it) || 0
    badCtrSum += it.linkCtr || 0
  }

  const avgBadCTR = badCount > 0 ? badCtrSum / badCount : 0
  const potentialNewOrders = avgAngelCPA && avgAngelCPA > 0 ? Math.floor(badSpend / avgAngelCPA) : 0

  return {
    angelCount,
    angelSpend,
    angelPurchases,
    angelMessages,
    avgAngelCPA,
    avgAngelCTR,
    badCount,
    badSpend,
    badPurchases,
    badMessages,
    avgBadCTR,
    potentialNewOrders,
  }
})

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

function statusFor(item: MetaInsightItem) {
  return item.effectiveStatus || item.status || campaigns.value.find(campaign => campaign.id === item.entityId)?.effectiveStatus || ''
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
  if (status === 'PAUSED' && !confirmed && !confirm(l({ lo: `ປິດ “${item.entityName}”? ການສົ່ງໂຄສະນາຈະຢຸດຈົນກວ່າຈະເປີດຄືນ.`, th: `ปิด “${item.entityName}”? การส่งโฆษณาจะหยุดจนกว่าจะเปิดอีกครั้ง`, en: `Pause “${item.entityName}”?` }))) return false
  updatingId.value = item.entityId
  notice.value = ''
  success.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${item.entityType}/${encodeURIComponent(item.entityId)}/status`, { method: 'PATCH', body: { status } })
    item.status = status
    item.effectiveStatus = status
    success.value = status === 'ACTIVE'
      ? l({ lo: `ເປີດ “${item.entityName}” ສຳເລັດ.`, th: `เปิด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now active.` })
      : l({ lo: `ປິດ “${item.entityName}” ສຳເລັດ.`, th: `ปิด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now paused.` })
    return true
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ປ່ຽນສະຖານະບໍ່ສຳເລັດ.', th: 'เปลี่ยนสถานะไม่สำเร็จ', en: 'Could not change status.' })
    return false
  } finally { updatingId.value = '' }
}

function handlePauseItem(item: MetaInsightItem) {
  changeStatus(item, 'PAUSED')
}

async function pauseAllBadAds() {
  const activeBad = badAds.value.filter(it => !statusFor(it).toUpperCase().includes('PAUSED'))
  if (activeBad.length === 0) {
    alert(l({ lo: 'ແອດທີ່ບໍ່ດີທັງໝົດຖືກປິດຢູ່ແລ້ວ!', th: 'แอดที่ไม่ดีทั้งหมดถูกหยุดอยู่แล้ว!', en: 'All underperforming ads are already paused!' }))
    return
  }
  if (!confirm(l({ lo: `ຢືນຢັນປິດແອດທີ່ບໍ່ດີທັງໝົດ ${activeBad.length} ໂຕ? ເພື່ອຢຸດການສູນເສຍງົບປະມານ.`, th: `ยืนยันหยุดแอดที่ไม่ดีทั้งหมด ${activeBad.length} ตัว? เพื่อหยุดการเสียงบประมาณ`, en: `Pause all ${activeBad.length} underperforming ads to stop budget waste?` }))) {
    return
  }

  bulkPausing.value = true
  notice.value = ''
  success.value = ''
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
    lo: `ປິດແອດທີ່ບໍ່ດີສຳເລັດ ${pausedCount} ໂຕ! ຢຸດການສູນເສຍງົບປະມານແລ້ວ.`,
    th: `หยุดแอดที่ไม่ดีสำเร็จ ${pausedCount} ตัว! หยุดการเสียงบประมาณแล้ว`,
    en: `Successfully paused ${pausedCount} underperforming ads!`,
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

watch(adAccountId, async (newVal, oldVal) => {
  if (newVal && newVal !== oldVal && ready.value) {
    await loadPerformance()
  }
})

watch(level, async () => {
  if (ready.value) await loadPerformance()
})

onMounted(async () => {
  await Promise.all([loadAccounts(), loadAutomationsAndProducts(), loadProductStrategiesFromStore()])
  await loadPerformance()
  ready.value = true
})
</script>

<template>
  <div class="ads-battle-page">
    <!-- Header Banner -->
    <header class="battle-header">
      <div class="battle-title-wrap">
        <div class="d-flex align-center gap-2 mb-1">
          <v-chip size="small" color="primary" variant="tonal" class="font-weight-bold">
            <v-icon icon="mdi-scale-balance" start size="14" />
            HEAD-TO-HEAD BATTLE
          </v-chip>
          <v-chip size="small" variant="flat" color="amber-darken-3" class="font-weight-bold">
            🧚‍♀️ ນາງຟ້າ VS 🛑 ໂຕທີ່ບໍ່ດີ
          </v-chip>
        </div>
        <h1>{{ l({ lo: 'ວິເຄາະປຽບທຽບ: Ads ນາງຟ້າ VS ແອດທີ່ບໍ່ດີ / ໝົດກຳລັງ', th: 'วิเคราะห์เปรียบเทียบ: แอดนางฟ้า VS แอดที่ไม่ดี / หมดกำลัง', en: 'Ad Battle: Angel Ads vs Underperforming / Fatigued Ads' }) }}</h1>
        <p class="battle-subtitle">
          {{ l({ lo: 'ປຽບທຽບແບບຄູ່ຂະໜານ 2 ຝັ່ງເພື່ອເບິ່ງວ່າແອດໂຕໃດສ້າງກຳໄລ ແລະ ແອດໂຕໃດກຳລັງກິນເງິນລ້າໆ ພ້ອມປຸ່ມຈັດການ 1-Click ໂດຍບໍ່ຕ້ອງມານັ່ງຊອກຫາ', th: 'เปรียบเทียบแบบคู่ขนาน 2 ฝั่งเพื่อดูว่าแอดตัวไหนทำกำไร และแอดตัวไหนกำลังกินเงินเปล่า พร้อมปุ่มจัดการ 1-Click ไม่ต้องเสียเวลาค้นหา', en: 'Side-by-side comparison of winning vs underperforming ads with 1-click actions to scale or pause.' }) }}
        </p>
      </div>

      <div class="battle-header-actions">
        <NuxtLink to="/ads">
          <v-btn variant="outlined" color="primary" prepend-icon="mdi-bullhorn-outline" class="font-weight-bold">
            {{ l({ lo: 'ຈັດການ Ads ຫຼັກ (All Ads)', th: 'จัดการแอดหลัก (All Ads)', en: 'Main Ads Manager' }) }}
          </v-btn>
        </NuxtLink>
      </div>
    </header>

    <!-- Alerts -->
    <v-alert v-if="notice" type="warning" variant="tonal" class="mb-4" closable @click:close="notice = ''">{{ notice }}</v-alert>
    <v-alert v-if="success" type="success" variant="tonal" class="mb-4" closable @click:close="success = ''">{{ success }}</v-alert>

    <!-- Filter Bar -->
    <section class="battle-filter-bar">
      <div class="filter-inputs">
        <v-select
          v-model="adAccountId"
          :items="adAccounts.map(a => ({ ...a, label: `${a.accountType === 'business' ? 'Business' : 'Personal'} · ${a.name}` }))"
          item-title="label"
          item-value="id"
          :label="l({ lo: 'ບັນຊີໂຄສະນາ', th: 'บัญชีโฆษณา', en: 'Ad account' })"
          variant="outlined"
          density="compact"
          hide-details
          class="filter-select"
        />

        <v-select
          v-model="selectedProductId"
          :items="productOptions"
          item-title="title"
          item-value="value"
          :label="l({ lo: 'ສິນຄ້າ (Product)', th: 'สินค้า (Product)', en: 'Product' })"
          variant="outlined"
          density="compact"
          hide-details
          class="filter-select"
        />

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
          class="filter-select"
          @update:model-value="setPreset"
        />

        <v-text-field v-if="datePreset === 'custom'" v-model="since" type="date" label="From" variant="outlined" density="compact" hide-details class="date-field" />
        <v-text-field v-if="datePreset === 'custom'" v-model="until" type="date" label="To" variant="outlined" density="compact" hide-details class="date-field" />

        <v-btn-toggle v-model="level" mandatory color="primary" variant="outlined" density="compact" class="level-toggle">
          <v-btn value="ad"><span>Ads</span></v-btn>
          <v-btn value="adset"><span>Ad Set</span></v-btn>
          <v-btn value="campaign"><span>Campaign</span></v-btn>
        </v-btn-toggle>
      </div>

      <div class="filter-actions">
        <v-chip size="small" variant="tonal" color="primary">
          VAT {{ vatDetails.rate }}% (Auto)
        </v-chip>
        <v-btn color="primary" variant="tonal" height="40" :loading="loadingReport" @click="loadPerformance">
          {{ l({ lo: 'ນຳໃຊ້', th: 'นำไปใช้', en: 'Apply' }) }}
        </v-btn>
      </div>
    </section>

    <div v-if="loadingReport" class="loading-state">
      <v-progress-circular indeterminate color="primary" size="48" />
      <p class="mt-3 text-medium-emphasis">{{ l({ lo: 'ກຳລັງໂຫຼດ ແລະ ວິເຄາະແອດ...', th: 'กำลังโหลดและวิเคราะห์แอด...', en: 'Analyzing ad performance...' }) }}</p>
    </div>

    <template v-else-if="report">
      <!-- Head-to-Head Scoreboard (3 Hero Cards) -->
      <section class="scoreboard-grid">
        <!-- Card 1: The Winners -->
        <div class="score-card winner-card">
          <div class="score-card-header">
            <span class="card-badge gold">
              <v-icon icon="mdi-star-face" size="16" />
              🧚‍♀️ ADS ນາງຟ້າ (THE WINNERS)
            </span>
            <span class="count-tag">{{ battleStats.angelCount }} ໂຕ</span>
          </div>

          <div class="score-main-stats">
            <div class="stat-unit">
              <small>{{ l({ lo: 'ຍອດປິດການຊື້', th: 'ยอดปิดการซื้อ', en: 'Purchases' }) }}</small>
              <strong class="text-success">{{ battleStats.angelPurchases }} ອໍເດີ</strong>
            </div>
            <div class="stat-unit">
              <small>{{ l({ lo: 'ສະເລ່ຍຕົ້ນທຶນ/ອໍເດີ', th: 'เฉลี่ยต้นทุน/ออเดอร์', en: 'Avg CPA' }) }}</small>
              <strong class="text-primary">{{ money(battleStats.avgAngelCPA) }}</strong>
            </div>
          </div>

          <div class="score-sub-stats">
            <span>ໃຊ້ຈ່າຍ: <strong>{{ money(battleStats.angelSpend) }}</strong></span>
            <span>ຄົນທັກ: <strong>{{ battleStats.angelMessages }} ຄົນ</strong></span>
            <span>Link CTR: <strong>{{ percent(battleStats.avgAngelCTR) }}</strong></span>
          </div>
        </div>

        <!-- Card 2: VS & Reallocation Opportunity -->
        <div class="score-card opportunity-card">
          <div class="score-card-header">
            <span class="card-badge cyan">
              <v-icon icon="mdi-swap-horizontal-bold" size="16" />
              VS ມູນຄ່າທີ່ປະຢັດໄດ້
            </span>
            <span class="count-tag highlight">{{ money(battleStats.badSpend) }}</span>
          </div>

          <div class="opportunity-message">
            <p>
              {{ l({
                lo: `ຖ້າຍົກຍ້າຍງົບ ${money(battleStats.badSpend)} ຈາກແອດທີ່ບໍ່ດີ ມາອັດໃສ່ Ads ນາງຟ້າ ຈະໄດ້ອໍເດີເພີ່ມປະມານ:`,
                th: `ถ้าย้ายงบ ${money(battleStats.badSpend)} จากแอดที่ไม่ดีมาใส่แอดนางฟ้า จะได้ออเดอร์เพิ่มประมาณ:`,
                en: `Reallocating ${money(battleStats.badSpend)} from bad ads to winners could generate:`,
              }) }}
            </p>
            <strong class="potential-orders">+{{ battleStats.potentialNewOrders }} ອໍເດີ! 🔥</strong>
          </div>

          <div class="score-actions">
            <v-btn
              color="error"
              variant="flat"
              prepend-icon="mdi-pause-circle-outline"
              size="small"
              :disabled="battleStats.badCount === 0"
              :loading="bulkPausing"
              class="font-weight-bold"
              @click="pauseAllBadAds"
            >
              {{ l({ lo: '⏸️ ປິດແອດທີ່ບໍ່ດີທັງໝົດ', th: '⏸️ หยุดแอดที่ไม่ดีทั้งหมด', en: 'Pause All Bad Ads' }) }}
            </v-btn>
          </div>
        </div>

        <!-- Card 3: The Losers / Fatigued -->
        <div class="score-card loser-card">
          <div class="score-card-header">
            <span class="card-badge crimson">
              <v-icon icon="mdi-alert-octagon-outline" size="16" />
              🛑 ແອດທີ່ບໍ່ດີ / ໝົດກຳລັງ
            </span>
            <span class="count-tag red">{{ battleStats.badCount }} ໂຕ</span>
          </div>

          <div class="score-main-stats">
            <div class="stat-unit">
              <small>{{ l({ lo: 'ເງິນທີ່ເສຍລ້າໆ', th: 'เงินที่เสียเปล่า', en: 'Wasted Spend' }) }}</small>
              <strong class="text-error">{{ money(battleStats.badSpend) }}</strong>
            </div>
            <div class="stat-unit">
              <small>{{ l({ lo: 'ຍອດການຊື້', th: 'ยอดการซื้อ', en: 'Purchases' }) }}</small>
              <strong class="text-medium-emphasis">0 ອໍເດີ</strong>
            </div>
          </div>

          <div class="score-sub-stats">
            <span>Link CTR: <strong>{{ percent(battleStats.avgBadCTR) }} (ຕ່ຳ)</strong></span>
            <span>ຄົນທັກ: <strong>{{ battleStats.badMessages }} ຄົນ</strong></span>
            <span>ສະຖານະ: <strong class="text-error">ຄວນປິດທັນທີ</strong></span>
          </div>
        </div>
      </section>

      <!-- Daily Ad Momentum & Strength Line Chart -->
      <UiAdStrengthLineChart
        :items="filteredBaseItems"
        :account-trend="report.trend"
        :currency="currency"
        :updating-id="updatingId"
        @scale="openScaleModal"
        @pause="handlePauseItem"
        @change-status="({ item, status }: { item: MetaInsightItem; status: 'ACTIVE' | 'PAUSED' }) => changeStatus(item, status)"
      />

      <!-- Side-by-Side Dual Column View -->
      <section class="battle-arena">
        <!-- Left Side: Winning Ads (Ads ນາງຟ້າ) -->
        <div class="arena-column winner-column">
          <div class="arena-column-header">
            <div class="d-flex align-center gap-2">
              <span class="column-icon-circle green"><v-icon icon="mdi-star" size="18" /></span>
              <div>
                <h2>{{ l({ lo: '🧚‍♀️ ລາຍຊື່ Ads ນາງຟ້າ (The Winners)', th: '🧚‍♀️ รายชื่อแอดนางฟ้า (The Winners)', en: 'Angel Ads (Winners)' }) }}</h2>
                <small>{{ l({ lo: 'ແອດທີ່ສ້າງກຳໄລ, CPA ຖືກ ແລະ ຄວນ Scale ງົບ', th: 'แอดที่ทำกำไร CPA ถูก และควร Scale งบ', en: 'High profit, low CPA, scale candidates' }) }}</small>
              </div>
            </div>
            <span class="badge-count green">{{ angelAds.length }}</span>
          </div>

          <div v-if="angelAds.length === 0" class="arena-empty">
            <v-icon icon="mdi-star-outline" size="36" class="text-medium-emphasis" />
            <p>{{ l({ lo: 'ຍັງບໍ່ພົບ Ads ນາງຟ້າ ໃນໄລຍະເວລານີ້', th: 'ยังไม่พบแอดนางฟ้าในช่วงเวลานี้', en: 'No angel ads found in this period' }) }}</p>
          </div>

          <div v-else class="ad-card-list">
            <article
              v-for="(item, idx) in angelAds"
              :key="item.entityId"
              class="battle-ad-card winner-style"
            >
              <div class="card-top">
                <div class="rank-pill">
                  <span v-if="idx === 0">🥇 ອັນດັບ 1</span>
                  <span v-else-if="idx === 1">🥈 ອັນດັບ 2</span>
                  <span v-else-if="idx === 2">🥉 ອັນດັບ 3</span>
                  <span v-else>⭐ Top {{ idx + 1 }}</span>
                </div>
                <div class="card-status-switch">
                  <v-switch
                    :model-value="!statusFor(item).toUpperCase().includes('PAUSED')"
                    color="success"
                    density="compact"
                    hide-details
                    :loading="updatingId === item.entityId"
                    @update:model-value="(val) => changeStatus(item, val ? 'ACTIVE' : 'PAUSED')"
                  />
                  <small>{{ statusFor(item).toUpperCase().includes('PAUSED') ? 'ປິດຢູ່' : 'ເປີດຢູ່' }}</small>
                </div>
              </div>

              <h3 class="ad-title">{{ item.entityName }}</h3>
              <div class="ad-meta">
                <code>ID: {{ item.entityId }}</code>
                <span v-if="getLinkedProduct(item)" class="product-badge">
                  📦 {{ getLinkedProduct(item)?.product.name }}
                </span>
              </div>

              <div class="ad-metric-grid">
                <div class="metric-box">
                  <small>{{ l({ lo: 'ໃຊ້ຈ່າຍ', th: 'ใช้จ่าย', en: 'Spend' }) }}</small>
                  <strong>{{ money(item.spend) }}</strong>
                </div>
                <div class="metric-box green-highlight">
                  <small>{{ l({ lo: 'ການຊື້', th: 'การซื้อ', en: 'Orders' }) }}</small>
                  <strong>{{ getPurchasesCount(item) }} ອໍເດີ</strong>
                </div>
                <div class="metric-box green-highlight">
                  <small>{{ l({ lo: 'ຕົ້ນທຶນ/ອໍເດີ (CPA)', th: 'ต้นทุน/ออเดอร์', en: 'CPA' }) }}</small>
                  <strong>{{ money(getCostPerPurchase(item)) }}</strong>
                </div>
                <div class="metric-box">
                  <small>CTR</small>
                  <strong>{{ percent(item.linkCtr) }}</strong>
                </div>
                <div class="metric-box">
                  <small>ຄົນທັກ</small>
                  <strong>{{ getMessagesCount(item) }}</strong>
                </div>
                <div class="metric-box">
                  <small>Frequency</small>
                  <strong>{{ fmt(item.frequency, 2) }}x</strong>
                </div>
              </div>

              <div class="ad-reason-box green">
                <v-icon icon="mdi-check-decagram" size="14" class="text-success" />
                <span>{{ isAngelAd(item).reason }}</span>
              </div>

              <div class="card-footer-actions">
                <v-btn
                  size="small"
                  color="success"
                  variant="flat"
                  prepend-icon="mdi-rocket-launch"
                  class="font-weight-bold"
                  @click="openScaleModal(item)"
                >
                  {{ l({ lo: '🚀 Scale ງົບ (+20%)', th: '🚀 Scale งบ (+20%)', en: '🚀 Scale Budget' }) }}
                </v-btn>
              </div>
            </article>
          </div>
        </div>

        <!-- Right Side: Bad / Fatigued Ads (ແອດທີ່ບໍ່ດີ / ໝົດກຳລັງ) -->
        <div class="arena-column loser-column">
          <div class="arena-column-header">
            <div class="d-flex align-center gap-2">
              <span class="column-icon-circle red"><v-icon icon="mdi-alert-octagon" size="18" /></span>
              <div>
                <h2>{{ l({ lo: '🛑 ລາຍຊື່ ແອດທີ່ບໍ່ດີ / ໝົດກຳລັງ', th: '🛑 รายชื่อแอดที่ไม่ดี / หมดกำลัง', en: 'Underperforming & Fatigued Ads' }) }}</h2>
                <small>{{ l({ lo: 'ແອດທີ່ກິນເງິນລ້າໆ, CTR ຕ່ຳ ຫຼື ຄວນກົດປິດ Stop-loss', th: 'แอดที่กินเงินเปล่า CTR ต่ำ หรือควรหยุด Stop-loss', en: 'Wasting budget, low CTR, pause candidates' }) }}</small>
              </div>
            </div>
            <span class="badge-count red">{{ badAds.length }}</span>
          </div>

          <div v-if="badAds.length === 0" class="arena-empty">
            <v-icon icon="mdi-check-circle-outline" size="36" class="text-success" />
            <p>{{ l({ lo: 'ຍິນດີນຳ! ບໍ່ມີແອດທີ່ກິນເງິນລ້າໆໃນໄລຍະເວລານີ້', th: 'ยินดีด้วย! ไม่มีแอดที่กินเงินเปล่าในช่วงเวลานี้', en: 'Great! No money wasters found in this period' }) }}</p>
          </div>

          <div v-else class="ad-card-list">
            <article
              v-for="item in badAds"
              :key="item.entityId"
              class="battle-ad-card loser-style"
            >
              <div class="card-top">
                <div class="bad-pill">
                  <v-icon icon="mdi-alert-circle" size="12" />
                  <span>{{ isBadAd(item).severity === 'high' ? '🔴 ກິນງົບສູງສຸດບໍ່ມີຍອດ' : '⚠️ ໝົດກຳລັງ / CTR ຕ່ຳ' }}</span>
                </div>
                <div class="card-status-switch">
                  <v-switch
                    :model-value="!statusFor(item).toUpperCase().includes('PAUSED')"
                    color="error"
                    density="compact"
                    hide-details
                    :loading="updatingId === item.entityId"
                    @update:model-value="(val) => changeStatus(item, val ? 'ACTIVE' : 'PAUSED')"
                  />
                  <small>{{ statusFor(item).toUpperCase().includes('PAUSED') ? 'ປິດຢູ່' : 'ເປີດຢູ່' }}</small>
                </div>
              </div>

              <h3 class="ad-title">{{ item.entityName }}</h3>
              <div class="ad-meta">
                <code>ID: {{ item.entityId }}</code>
                <span v-if="getLinkedProduct(item)" class="product-badge">
                  📦 {{ getLinkedProduct(item)?.product.name }}
                </span>
              </div>

              <div class="ad-metric-grid">
                <div class="metric-box red-highlight">
                  <small>{{ l({ lo: 'ເງິນທີ່ເສຍລ້າໆ', th: 'เงินที่เสียเปล่า', en: 'Wasted Spend' }) }}</small>
                  <strong class="text-error">{{ money(item.spend) }}</strong>
                </div>
                <div class="metric-box">
                  <small>{{ l({ lo: 'ການຊື້', th: 'การซื้อ', en: 'Orders' }) }}</small>
                  <strong>0 ອໍເດີ</strong>
                </div>
                <div class="metric-box">
                  <small>CTR</small>
                  <strong :class="{ 'text-error': (item.linkCtr || 0) < 1.0 }">{{ percent(item.linkCtr) }}</strong>
                </div>
                <div class="metric-box">
                  <small>ຄົນທັກ</small>
                  <strong>{{ getMessagesCount(item) }}</strong>
                </div>
                <div class="metric-box">
                  <small>Frequency</small>
                  <strong :class="{ 'text-warning': (item.frequency || 0) >= 2.0 }">{{ fmt(item.frequency, 2) }}x</strong>
                </div>
                <div class="metric-box">
                  <small>CPA</small>
                  <span class="text-medium-emphasis">—</span>
                </div>
              </div>

              <div class="ad-reason-box red">
                <div v-for="(reason, rIdx) in isBadAd(item).reasons" :key="rIdx" class="d-flex align-center gap-1">
                  <v-icon icon="mdi-close-circle" size="13" class="text-error" />
                  <span>{{ reason }}</span>
                </div>
              </div>

              <div class="card-footer-actions">
                <v-btn
                  v-if="!statusFor(item).toUpperCase().includes('PAUSED')"
                  size="small"
                  color="error"
                  variant="flat"
                  prepend-icon="mdi-pause-circle-outline"
                  class="font-weight-bold"
                  :loading="updatingId === item.entityId"
                  @click="changeStatus(item, 'PAUSED', true)"
                >
                  {{ l({ lo: '⏸️ ປິດແອດດຽວນີ້ (Pause)', th: '⏸️ ปิดแอดเดี๋ยวนี้ (Pause)', en: 'Pause Ad' }) }}
                </v-btn>
                <v-chip v-else size="small" variant="tonal" color="default">
                  {{ l({ lo: 'ແອດນີ້ຖືກປິດແລ້ວ', th: 'แอดนี้ถูกปิดแล้ว', en: 'Already Paused' }) }}
                </v-chip>
              </div>
            </article>
          </div>
        </div>
      </section>

      <!-- Metric-by-Metric Matrix Comparison Table -->
      <section class="battle-matrix-section">
        <div class="matrix-header">
          <h2>{{ l({ lo: '📊 ຕາຕະລາງປຽບທຽບຕົວເລກ Metric ຕໍ່ Metric', th: '📊 ตารางเปรียบเทียบตัวเลข Metric ต่อ Metric', en: 'Direct Metric-by-Metric Comparison' }) }}</h2>
          <small>{{ l({ lo: 'ສະຫຼຸບຕົວເລກສະເລ່ຍທຽບກັນລະຫວ່າງຝັ່ງນາງຟ້າ ແລະ ຝັ່ງທີ່ບໍ່ດີ', th: 'สรุปตัวเลขเฉลี่ยเทียบกันระหว่างฝั่งนางฟ้าและฝั่งที่ไม่ดี', en: 'Side-by-side metric averages and differences' }) }}</small>
        </div>

        <div class="table-responsive">
          <table class="battle-matrix-table">
            <thead>
              <tr>
                <th>{{ l({ lo: 'Metric ຫຼັກ', th: 'Metric หลัก', en: 'Key Metric' }) }}</th>
                <th class="text-success">🧚‍♀️ {{ l({ lo: 'Ads ນາງຟ້າ (Winners)', th: 'แอดนางฟ้า (Winners)', en: 'Angel Ads' }) }}</th>
                <th class="text-error">🛑 {{ l({ lo: 'ແອດທີ່ບໍ່ດີ (Losers)', th: 'แอดที่ไม่ดี (Losers)', en: 'Bad Ads' }) }}</th>
                <th>{{ l({ lo: 'ສ່ວນຕ່າງ & ຜົນກະທົບ', th: 'ส่วนต่าง & ผลกระทบ', en: 'Impact / Difference' }) }}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td><strong>{{ l({ lo: 'ຕົ້ນທຶນ/ອໍເດີ (Avg CPA)', th: 'ต้นทุน/ออเดอร์ (Avg CPA)', en: 'Cost Per Order (CPA)' }) }}</strong></td>
                <td class="text-success font-weight-bold">{{ money(battleStats.avgAngelCPA) }}</td>
                <td class="text-error font-weight-bold">— (0 ອໍເດີ)</td>
                <td><span class="impact-chip green">ປະຢັດຕົ້ນທຶນ 100%</span></td>
              </tr>
              <tr>
                <td><strong>{{ l({ lo: 'Link CTR (ຄວາມດຶງດູດ)', th: 'Link CTR (ความดึงดูด)', en: 'Link CTR' }) }}</strong></td>
                <td class="text-success font-weight-bold">{{ percent(battleStats.avgAngelCTR) }}</td>
                <td class="text-error font-weight-bold">{{ percent(battleStats.avgBadCTR) }}</td>
                <td>
                  <span v-if="battleStats.avgAngelCTR > battleStats.avgBadCTR" class="impact-chip green">
                    +{{ fmt(((battleStats.avgAngelCTR - battleStats.avgBadCTR) / Math.max(0.01, battleStats.avgBadCTR)) * 100, 0) }}% ດຶງດູດດີກວ່າ
                  </span>
                  <span v-else>—</span>
                </td>
              </tr>
              <tr>
                <td><strong>{{ l({ lo: 'ຍອດການຊື້ລວມ (Purchases)', th: 'ยอดการซื้อรวม', en: 'Total Purchases' }) }}</strong></td>
                <td class="text-success font-weight-bold">{{ battleStats.angelPurchases }} ອໍເດີ</td>
                <td class="text-error font-weight-bold">0 ອໍເດີ</td>
                <td><span class="impact-chip green">+{{ battleStats.angelPurchases }} ອໍເດີ</span></td>
              </tr>
              <tr>
                <td><strong>{{ l({ lo: 'ຍອດໃຊ້ຈ່າຍງົບ (Spend)', th: 'ยอดใช้งบประมาณ', en: 'Total Spend' }) }}</strong></td>
                <td>{{ money(battleStats.angelSpend) }}</td>
                <td class="text-error">{{ money(battleStats.badSpend) }}</td>
                <td>
                  <span class="impact-chip red">ສູນເສຍງົບ {{ money(battleStats.badSpend) }}</span>
                </td>
              </tr>
              <tr>
                <td><strong>{{ l({ lo: 'ຄົນທັກຂໍ້ຄວາມ (Messages)', th: 'คนทักข้อความ', en: 'Total Inquiries' }) }}</strong></td>
                <td>{{ battleStats.angelMessages }} ຄົນ</td>
                <td>{{ battleStats.badMessages }} ຄົນ</td>
                <td><span>{{ battleStats.angelMessages >= battleStats.badMessages ? 'ຝັ່ງນາງຟ້າປິດການຂາຍໄດ້ຈິງ' : 'ທັກແຕ່ປິດບໍ່ໄດ້' }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <!-- Scale Budget Modal -->
    <v-dialog v-model="budgetModalOpen" max-width="480">
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
            ເພີ່ມງົບໃຫ້ກັບແອດນາງຟ້າ: <strong>{{ targetBudgetItem?.entityName }}</strong>
          </p>

          <div class="scale-presets d-flex gap-2 mb-4">
            <v-btn
              v-for="pct in [10, 20, 30, 50]"
              :key="pct"
              size="small"
              :variant="selectedPercent === pct ? 'flat' : 'outlined'"
              :color="selectedPercent === pct ? 'success' : 'default'"
              class="flex-grow-1"
              @click="applyScalePercent(pct)"
            >
              +{{ pct }}%
            </v-btn>
          </div>

          <v-text-field
            v-model.number="newDailyBudget"
            type="number"
            min="0"
            step="0.01"
            :label="l({ lo: 'ງົບປະມານໃໝ່ຕໍ່ມື້ (Daily Budget)', th: 'งบประมาณใหม่ต่อวัน', en: 'New Daily Budget' })"
            :suffix="currency"
            variant="outlined"
            density="comfortable"
          />

          <div class="scale-info-box pa-3 rounded-lg bg-grey-lighten-4 mt-2">
            <div class="d-flex justify-space-between text-caption mb-1">
              <span>ງົບເດີມປະມານ:</span>
              <strong>{{ money(currentDailyBudget) }}/ມື້</strong>
            </div>
            <div class="d-flex justify-space-between text-caption">
              <span>ງົບໃໝ່ຫຼັງ Scale:</span>
              <strong class="text-success">{{ money(newDailyBudget) }}/ມື້ (+{{ selectedPercent }}%)</strong>
            </div>
          </div>
        </v-card-text>

        <v-card-actions class="pt-2">
          <v-spacer />
          <v-btn variant="text" @click="budgetModalOpen = false">{{ l({ lo: 'ຍົກເລີກ', th: 'ยกเลิก', en: 'Cancel' }) }}</v-btn>
          <v-btn
            color="success"
            variant="flat"
            :loading="updatingBudget"
            class="font-weight-bold px-4"
            @click="submitBudgetScale"
          >
            {{ l({ lo: 'ຢືນຢັນ Scale ງົບ', th: 'ยืนยัน Scale งบ', en: 'Confirm Scale' }) }}
          </v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.ads-battle-page {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.battle-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
  padding-bottom: 8px;
}

.battle-title-wrap h1 {
  font-size: clamp(20px, 2.2vw, 26px);
  font-weight: 800;
  color: var(--color-text, #0f172a);
  margin: 4px 0 6px;
  line-height: 1.25;
}

.battle-subtitle {
  font-size: 14px;
  color: #64748b;
  max-width: 820px;
  line-height: 1.5;
  margin: 0;
}

/* Filter Bar */
.battle-filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 12px;
  padding: 12px 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.filter-inputs {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1;
}

.filter-select {
  min-width: 170px;
  max-width: 220px;
}

.date-field {
  max-width: 140px;
}

.level-toggle {
  height: 40px;
}

.filter-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* Loading State */
.loading-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
}

/* Scoreboard Grid */
.scoreboard-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
}

.score-card {
  background: var(--color-surface, #ffffff);
  border-radius: 14px;
  padding: 18px 20px;
  border: 1px solid var(--color-border, #e2e8f0);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.winner-card {
  border-left: 5px solid #10b981;
  background: linear-gradient(180deg, #f0fdf4 0%, #ffffff 60%);
}

.opportunity-card {
  border-left: 5px solid #06b6d4;
  background: linear-gradient(180deg, #ecfeff 0%, #ffffff 60%);
}

.loser-card {
  border-left: 5px solid #ef4444;
  background: linear-gradient(180deg, #fef2f2 0%, #ffffff 60%);
}

.score-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0.5px;
  padding: 4px 10px;
  border-radius: 999px;
}

.card-badge.gold {
  background: #fef3c7;
  color: #b45309;
}

.card-badge.cyan {
  background: #cffafe;
  color: #0e7490;
}

.card-badge.crimson {
  background: #fee2e2;
  color: #b91c1c;
}

.count-tag {
  font-size: 13px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 6px;
  background: #e2e8f0;
  color: #334155;
}

.count-tag.highlight {
  background: #cffafe;
  color: #0891b2;
}

.count-tag.red {
  background: #fee2e2;
  color: #dc2626;
}

.score-main-stats {
  display: flex;
  gap: 20px;
}

.stat-unit small {
  display: block;
  font-size: 11px;
  color: #64748b;
  text-transform: uppercase;
  margin-bottom: 2px;
}

.stat-unit strong {
  font-size: 20px;
  font-weight: 800;
}

.score-sub-stats {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #64748b;
  padding-top: 8px;
  border-top: 1px dashed #e2e8f0;
}

.opportunity-message p {
  font-size: 13px;
  color: #475569;
  margin: 0 0 4px;
}

.potential-orders {
  font-size: 22px;
  font-weight: 900;
  color: #0891b2;
}

.score-actions {
  margin-top: auto;
  padding-top: 6px;
}

/* Battle Arena (Side-by-Side) */
.battle-arena {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

@media (max-width: 900px) {
  .battle-arena {
    grid-template-columns: 1fr;
  }
}

.arena-column {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 14px;
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.arena-column-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border, #e2e8f0);
}

.column-icon-circle {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.column-icon-circle.green {
  background: #d1fae5;
  color: #059669;
}

.column-icon-circle.red {
  background: #fee2e2;
  color: #dc2626;
}

.arena-column-header h2 {
  font-size: 16px;
  font-weight: 700;
  margin: 0;
  color: var(--color-text, #0f172a);
}

.arena-column-header small {
  color: #64748b;
  font-size: 12px;
}

.badge-count {
  font-size: 13px;
  font-weight: 800;
  padding: 3px 10px;
  border-radius: 999px;
}

.badge-count.green {
  background: #d1fae5;
  color: #047857;
}

.badge-count.red {
  background: #fee2e2;
  color: #b91c1c;
}

.arena-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 16px;
  text-align: center;
  color: #94a3b8;
}

.ad-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.battle-ad-card {
  border-radius: 12px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  transition: all 0.2s ease;
}

.battle-ad-card.winner-style {
  background: #fbfdfc;
  border: 1.5px solid #a7f3d0;
  box-shadow: 0 2px 6px rgba(16, 185, 129, 0.05);
}

.battle-ad-card.loser-style {
  background: #fcfbfb;
  border: 1.5px solid #fecaca;
  box-shadow: 0 2px 6px rgba(239, 68, 68, 0.05);
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.rank-pill {
  font-size: 12px;
  font-weight: 800;
  background: #fef3c7;
  color: #b45309;
  padding: 2px 8px;
  border-radius: 6px;
}

.bad-pill {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 700;
  background: #fee2e2;
  color: #b91c1c;
  padding: 2px 8px;
  border-radius: 6px;
}

.card-status-switch {
  display: flex;
  align-items: center;
  gap: 6px;
}

.ad-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text, #0f172a);
  margin: 0;
  line-height: 1.35;
}

.ad-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.ad-meta code {
  font-size: 11px;
  background: #f1f5f9;
  padding: 1px 5px;
  border-radius: 4px;
  color: #475569;
}

.product-badge {
  font-size: 11px;
  font-weight: 600;
  background: #e0f2fe;
  color: #0369a1;
  padding: 1px 7px;
  border-radius: 4px;
}

.ad-metric-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  background: #ffffff;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 8px;
  padding: 8px 10px;
}

.metric-box small {
  display: block;
  font-size: 10px;
  color: #64748b;
  text-transform: uppercase;
}

.metric-box strong {
  font-size: 13px;
  font-weight: 700;
  color: #1e293b;
}

.metric-box.green-highlight strong {
  color: #059669;
}

.metric-box.red-highlight strong {
  color: #dc2626;
}

.ad-reason-box {
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  line-height: 1.4;
}

.ad-reason-box.green {
  background: #ecfdf5;
  color: #065f46;
}

.ad-reason-box.red {
  background: #fef2f2;
  color: #991b1b;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.card-footer-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 4px;
}

/* Matrix Table Section */
.battle-matrix-section {
  background: var(--color-surface, #ffffff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 14px;
  padding: 20px;
}

.matrix-header {
  margin-bottom: 14px;
}

.matrix-header h2 {
  font-size: 16px;
  font-weight: 700;
  margin: 0;
  color: var(--color-text, #0f172a);
}

.matrix-header small {
  color: #64748b;
  font-size: 12px;
}

.table-responsive {
  overflow-x: auto;
}

.battle-matrix-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.battle-matrix-table th {
  text-align: left;
  padding: 10px 14px;
  background: #f8fafc;
  border-bottom: 2px solid #e2e8f0;
  font-weight: 700;
  color: #475569;
}

.battle-matrix-table td {
  padding: 12px 14px;
  border-bottom: 1px solid #f1f5f9;
  color: #1e293b;
}

.impact-chip {
  display: inline-block;
  font-size: 11px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 999px;
}

.impact-chip.green {
  background: #d1fae5;
  color: #047857;
}

.impact-chip.red {
  background: #fee2e2;
  color: #b91c1c;
}
</style>
