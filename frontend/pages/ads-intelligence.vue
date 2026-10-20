<script setup lang="ts">
import type {
  MetaAdAccount,
  MetaAudienceAnalysis,
  MetaCampaign,
  PerformanceAnalysisReport,
  AdPerformanceItem,
  AdStrategySettings,
} from '~/types/automation'
import { useAdVat } from '~/composables/useAdVat'

definePageMeta({ middleware: 'auth' })
const { l } = useLocale()

type Level = 'campaign' | 'adset' | 'ad'
type StatusFilter = 'all' | 'active' | 'paused' | 'scale' | 'hold' | 'improve' | 'pause'

const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const report = ref<PerformanceAnalysisReport | null>(null)
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
const selectedProductId = ref<string>('all')
const viewMode = ref<'table' | 'grouped'>('table')

// Strategy Settings & Versioning Modals
const strategyModalOpen = ref(false)
const historyModalOpen = ref(false)
const strategyHistory = ref<AdStrategySettings[]>([])
const savingStrategy = ref(false)
const loadingHistory = ref(false)

// Strategy Settings Form
const strategyForm = reactive({
  targetProfitPerOrder: 30000,
  avgPackagingCost: 5000,
  avgShippingCostStore: 15000,
  codReturnRatePercent: 8.0,
  returnFeePerOrder: 15000,
  dailyBudgetCap: 500000,
  maxTestLossBudget: 200000,
  effectiveFrom: '',
  note: '',
})

// Budget Scale Modal
const budgetModalOpen = ref(false)
const targetBudgetItem = ref<AdPerformanceItem | null>(null)
const newDailyBudget = ref<number>(0)
const updatingBudget = ref(false)

const ready = ref(false)
const audienceByAdSet = reactive<Record<string, MetaAudienceAnalysis>>({})
const audienceLoading = reactive<Record<string, boolean>>({})
const audienceErrors = reactive<Record<string, string>>({})

const iso = (date: Date) => {
  const copy = new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
  return copy.toISOString().slice(0, 10)
}
const today = new Date()
const since = ref(iso(new Date(today.getFullYear(), today.getMonth(), today.getDate() - 6)))
const until = ref(iso(today))
const datePreset = ref('7')
const sortBy = ref<'spend' | 'cpa' | 'profit' | 'orders' | 'name'>('spend')

const selectedAccount = computed(() => adAccounts.value.find(item => item.id === adAccountId.value))
const { detectVatRate, computeVat } = useAdVat()
const vatDetails = computed(() => detectVatRate(selectedAccount.value))
const vatComputation = computed(() => computeVat(summary.value?.totalSpend, vatDetails.value.rate))

const items = computed(() => {
  const query = search.value.trim().toLowerCase()
  const list = report.value?.items || []
  return list
    .filter(item => !query || [item.entityName, item.campaignName, item.adSetName, item.entityId, item.productName].some(val => val?.toLowerCase().includes(query)))
    .filter((item) => {
      const status = (item.effectiveStatus || item.status || '').toUpperCase()
      if (statusFilter.value === 'active') return status === 'ACTIVE' || status === 'IN_PROCESS' || status === 'PENDING_REVIEW'
      if (statusFilter.value === 'paused') return status === 'PAUSED' || status === 'CAMPAIGN_PAUSED' || status === 'ADSET_PAUSED'
      if (statusFilter.value === 'scale') return item.decisionCode === 'SCALE'
      if (statusFilter.value === 'hold') return item.decisionCode === 'HOLD'
      if (statusFilter.value === 'improve') return item.decisionCode === 'IMPROVE'
      if (statusFilter.value === 'pause') return item.decisionCode === 'PAUSE' || item.decisionCode === 'PAUSE_TEST'
      return true
    })
    .filter((item) => {
      if (selectedProductId.value === 'all') return true
      if (selectedProductId.value === 'unlinked') return !item.productId
      return item.productId === selectedProductId.value
    })
    .sort((a, b) => {
      if (sortBy.value === 'cpa') {
        const cpaA = a.realCpa ?? 999999999
        const cpaB = b.realCpa ?? 999999999
        return cpaA - cpaB
      }
      if (sortBy.value === 'profit') {
        return b.netContribution - a.netContribution
      }
      if (sortBy.value === 'orders') {
        return b.completedOrders - a.completedOrders
      }
      if (sortBy.value === 'name') {
        return (a.entityName || '').localeCompare(b.entityName || '')
      }
      return (b.spend ?? -1) - (a.spend ?? -1)
    })
})

const productOptions = computed(() => {
  const allList = report.value?.items || []
  const countsByProduct: Record<string, { count: number; name: string; img?: string }> = {}
  let unlinkedCount = 0

  for (const it of allList) {
    if (it.productId) {
      const existing = countsByProduct[it.productId]
      if (!existing) {
        countsByProduct[it.productId] = {
          count: 1,
          name: it.productName || 'Product',
          img: it.productImageUrl,
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

interface ProductGroup {
  productId: string
  productName: string
  productImageUrl?: string
  productPrice?: string
  autoReplyFlowName?: string
  autoReplyFlowId?: string
  isLinked: boolean
  items: AdPerformanceItem[]
  totalSpend: number
  totalCompletedOrders: number
  totalRevenue: number
  totalNetProfit: number
  averageRealCPA?: number
}

const groupedByProduct = computed<ProductGroup[]>(() => {
  const groups: Record<string, ProductGroup> = {}

  for (const item of items.value) {
    const key = item.productId || '__unlinked__'
    if (!groups[key]) {
      groups[key] = {
        productId: item.productId || '',
        productName: item.productName || (item.productId ? 'Product' : l({ lo: 'ແອດທີ່ຍັງບໍ່ໄດ້ຜູກກັບ Auto Reply', th: 'แอดที่ยังไม่ผูกกับ Auto Reply', en: 'Ads not linked to Auto Reply' })),
        productImageUrl: item.productImageUrl,
        productPrice: item.productPrice,
        autoReplyFlowName: item.autoReplyFlowName,
        autoReplyFlowId: item.autoReplyFlowId,
        isLinked: !!item.productId,
        items: [],
        totalSpend: 0,
        totalCompletedOrders: 0,
        totalRevenue: 0,
        totalNetProfit: 0,
      }
    }
    const g = groups[key]
    g.items.push(item)
    g.totalSpend += item.spend
    g.totalCompletedOrders += item.completedOrders
    g.totalRevenue += item.completedRevenue
    g.totalNetProfit += item.netContribution
  }

  const result = Object.values(groups).map((g) => {
    if (g.totalCompletedOrders > 0) {
      g.averageRealCPA = g.totalSpend / g.totalCompletedOrders
    }
    return g
  })

  return result.sort((a, b) => {
    if (a.isLinked !== b.isLinked) return a.isLinked ? -1 : 1
    return b.totalSpend - a.totalSpend
  })
})

const summary = computed(() => report.value?.summary)
const strategy = computed(() => report.value?.strategy)
const currency = computed(() => report.value?.currency || 'LAK')

const fmt = (value?: number | null, digits = 0) => value == null ? '—' : new Intl.NumberFormat(undefined, { maximumFractionDigits: digits }).format(value)
const money = (value?: number | null) => value == null ? '—' : `${new Intl.NumberFormat(undefined, { maximumFractionDigits: 0 }).format(Math.round(value))} ₭`
const percent = (value?: number | null) => value == null ? '—' : `${fmt(value, 1)}%`

const adSetIDFor = (item: AdPerformanceItem) => item.entityType === 'adset' ? item.entityId : item.adSetId || ''
const audienceFor = (item: AdPerformanceItem) => audienceByAdSet[adSetIDFor(item)]

const statusLabel = (status?: string) => {
  const value = (status || '').toUpperCase()
  if (value === 'ACTIVE') return l({ lo: 'ເປີດໃຊ້ງານ', th: 'กำลังใช้งาน', en: 'Active' })
  if (value === 'PAUSED') return l({ lo: 'ປິດຊົ່ວຄາວ', th: 'หยุดชั่วคราว', en: 'Paused' })
  if (value === 'CAMPAIGN_PAUSED') return l({ lo: 'Campaign ຖືກປິດ', th: 'แคมเปญถูกหยุด', en: 'Campaign paused' })
  if (value === 'ADSET_PAUSED') return l({ lo: 'Ad Set ຖືກປິດ', th: 'ชุดโฆษณาถูกหยุด', en: 'Ad set paused' })
  if (value === 'IN_PROCESS' || value === 'PENDING_REVIEW') return l({ lo: 'ກຳລັງກວດສອບ', th: 'กำลังตรวจสอบ', en: 'In review' })
  return status || l({ lo: 'ບໍ່ຮູ້ສະຖານະ', th: 'ไม่ทราบสถานะ', en: 'Unknown' })
}

const statusClass = (status?: string) => {
  const value = (status || '').toUpperCase()
  if (value === 'ACTIVE' || value === 'IN_PROCESS' || value === 'PENDING_REVIEW') return 'active'
  if (value === 'PAUSED' || value === 'CAMPAIGN_PAUSED' || value === 'ADSET_PAUSED') return 'paused'
  return 'issues'
}

function decisionBadgeInfo(code: string) {
  switch (code) {
    case 'SCALE':
      return { label: 'SCALE (+20%)', color: 'success', icon: 'mdi-trending-up', class: 'badge-scale' }
    case 'HOLD':
      return { label: 'HOLD (ລໍຖ້າ COD)', color: 'info', icon: 'mdi-timer-sand', class: 'badge-hold' }
    case 'IMPROVE':
      return { label: 'IMPROVE (ປັບແອດ)', color: 'warning', icon: 'mdi-wrench', class: 'badge-improve' }
    case 'PAUSE':
      return { label: 'PAUSE (ຫຼຸດທຶນ)', color: 'error', icon: 'mdi-pause-circle', class: 'badge-pause' }
    case 'PAUSE_TEST':
      return { label: 'PAUSE (ໝົດງົບລອງ)', color: 'error', icon: 'mdi-alert-octagon', class: 'badge-pause-test' }
    default:
      return { label: 'ລໍຖ້າຂໍ້ມູນ 24-48h', color: 'secondary', icon: 'mdi-clock-outline', class: 'badge-nodata' }
  }
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
  } finally {
    loadingAccounts.value = false
  }
}

async function loadPerformance() {
  if (!adAccountId.value || !since.value || !until.value) return
  loadingReport.value = true
  notice.value = ''
  success.value = ''
  try {
    const query = new URLSearchParams({ level: level.value, since: since.value, until: until.value })
    const [campaignResult, analysisResult] = await Promise.all([
      $fetch<{ items: MetaCampaign[] }>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/campaigns`),
      $fetch<PerformanceAnalysisReport>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/performance-analysis?${query.toString()}`),
    ])
    campaigns.value = campaignResult.items || []
    report.value = analysisResult
    selectedId.value = ''

    // Populate strategy form from active settings
    if (analysisResult?.strategy) {
      const s = analysisResult.strategy
      strategyForm.targetProfitPerOrder = s.targetProfitPerOrder || 30000
      strategyForm.avgPackagingCost = s.avgPackagingCost || 5000
      strategyForm.avgShippingCostStore = s.avgShippingCostStore || 15000
      strategyForm.codReturnRatePercent = s.codReturnRatePercent || 8.0
      strategyForm.returnFeePerOrder = s.returnFeePerOrder || 15000
      strategyForm.dailyBudgetCap = s.dailyBudgetCap || 500000
      strategyForm.maxTestLossBudget = s.maxTestLossBudget || 200000
      strategyForm.effectiveFrom = iso(new Date())
      strategyForm.note = s.note || ''

      // If user hasn't configured custom settings yet (ID is 'default'), auto-prompt the 1-minute setup modal
      if (s.id === 'default' && !strategyModalOpen.value) {
        strategyModalOpen.value = true
      }
    }
  } catch (error: any) {
    report.value = null
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ດຶງຂໍ້ມູນການວິເຄາະໂຄສະນາບໍ່ສຳເລັດ.', th: 'โหลดการวิเคราะห์โฆษณาไม่สำเร็จ', en: 'Could not load performance analysis.' })
  } finally {
    loadingReport.value = false
  }
}

async function saveStrategySettings() {
  if (!adAccountId.value) return
  savingStrategy.value = true
  notice.value = ''
  try {
    const payload = {
      adAccountId: adAccountId.value,
      targetProfitPerOrder: Number(strategyForm.targetProfitPerOrder),
      avgPackagingCost: Number(strategyForm.avgPackagingCost),
      avgShippingCostStore: Number(strategyForm.avgShippingCostStore),
      codReturnRatePercent: Number(strategyForm.codReturnRatePercent),
      returnFeePerOrder: Number(strategyForm.returnFeePerOrder),
      dailyBudgetCap: Number(strategyForm.dailyBudgetCap),
      maxTestLossBudget: Number(strategyForm.maxTestLossBudget),
      effectiveFrom: strategyForm.effectiveFrom ? new Date(strategyForm.effectiveFrom).toISOString() : new Date().toISOString(),
      note: strategyForm.note,
    }
    await $fetch(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/strategy-settings`, {
      method: 'POST',
      body: payload,
    })
    success.value = l({ lo: 'ບັນທຶກເກນຍຸດທະສາດສຳເລັດແລ້ວ! ລະບົບຈະຄິດໄລ່ຕາມເກນໃໝ່ນີ້.', th: 'บันทึกเกณฑ์ยุทธศาสตร์เรียบร้อยแล้ว', en: 'Strategy settings updated successfully.' })
    strategyModalOpen.value = false
    await loadPerformance()
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ບັນທຶກເກນບໍ່ສຳເລັດ.', th: 'บันทึกเกณฑ์ไม่สำเร็จ', en: 'Could not save strategy settings.' })
  } finally {
    savingStrategy.value = false
  }
}

async function openHistoryModal() {
  if (!adAccountId.value) return
  historyModalOpen.value = true
  loadingHistory.value = true
  try {
    const res = await $fetch<{ items: AdStrategySettings[] }>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/strategy-settings/history`)
    strategyHistory.value = res.items || []
  } catch (error: any) {
    strategyHistory.value = []
  } finally {
    loadingHistory.value = false
  }
}

async function changeStatus(item: AdPerformanceItem, status: 'ACTIVE' | 'PAUSED', confirmed = false) {
  if (status === 'PAUSED' && !confirmed && !confirm(l({ lo: `ຢືນຢັນພັກໂຄສະນາ “${item.entityName}”?`, th: `ยืนยันหยุดโฆษณา “${item.entityName}”?`, en: `Pause “${item.entityName}”?` }))) return false
  updatingId.value = item.entityId
  notice.value = ''
  success.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${item.entityType}/${encodeURIComponent(item.entityId)}/status`, {
      method: 'PATCH',
      body: { status },
    })
    item.status = status
    item.effectiveStatus = status
    success.value = status === 'ACTIVE'
      ? l({ lo: `ເປີດ “${item.entityName}” ສຳເລັດ.`, th: `เปิด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now active.` })
      : l({ lo: `ພັກ “${item.entityName}” ສຳເລັດ.`, th: `หยุด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now paused.` })
    return true
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ປ່ຽນສະຖານະບໍ່ສຳເລັດ.', th: 'เปลี่ยนสถานะไม่สำเร็จ', en: 'Could not change status.' })
    return false
  } finally {
    updatingId.value = ''
  }
}

function promptScaleBudget(item: AdPerformanceItem) {
  targetBudgetItem.value = item
  // Suggest +20% from current spend or a minimum base
  const base = Math.max(item.spend * 1.2, 100000)
  newDailyBudget.value = Math.round(base / 10000) * 10000
  budgetModalOpen.value = true
}

async function applyBudgetChange() {
  if (!targetBudgetItem.value || newDailyBudget.value <= 0) return
  updatingBudget.value = true
  notice.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${targetBudgetItem.value.entityType}/${encodeURIComponent(targetBudgetItem.value.entityId)}/budget`, {
      method: 'PATCH',
      body: { dailyBudget: Math.round(newDailyBudget.value) },
    })
    success.value = l({
      lo: `ເພີ່ມງົບໃຫ້ “${targetBudgetItem.value.entityName}” ເປັນ ${money(newDailyBudget.value)}/ມື້ ສຳເລັດແລ້ວ!`,
      th: `ปรับงบประมาณให้ “${targetBudgetItem.value.entityName}” เรียบร้อยแล้ว`,
      en: `Updated budget successfully.`,
    })
    budgetModalOpen.value = false
    await loadPerformance()
  } catch (error: any) {
    notice.value = error?.data?.error?.message || error?.data?.message || l({ lo: 'ປັບງົບບໍ່ສຳເລັດ. ກວດສອບສິດ ads_management.', th: 'ปรับงบไม่สำเร็จ', en: 'Could not update budget.' })
  } finally {
    updatingBudget.value = false
  }
}

async function loadAudience(item: AdPerformanceItem) {
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

async function toggleItem(item: AdPerformanceItem) {
  selectedId.value = selectedId.value === item.entityId ? '' : item.entityId
  if (selectedId.value && item.entityType !== 'campaign') await loadAudience(item)
}

watch(adAccountId, async () => { if (!ready.value) return; await loadPerformance() })
watch(level, async () => { if (ready.value) await loadPerformance() })
onMounted(async () => {
  await loadAccounts()
  ready.value = true
  if (adAccountId.value) await loadPerformance()
})
</script>

<template>
  <!-- Shared Ads Suite Navigation -->
  <nav class="app-sub-nav" aria-label="Ads suite navigation">
    <NuxtLink to="/ads" class="app-sub-nav-item">
      <v-icon icon="mdi-bullhorn-outline" size="18" />
      <span>{{ l({ lo: '1. Ads Manager (ນຳສົ່ງ & ຄ່າແອດ)', th: '1. Ads Manager (นำส่ง & ค่าแอด)', en: '1. Ads Manager' }) }}</span>
    </NuxtLink>
    <NuxtLink to="/ads-intelligence" class="app-sub-nav-item active">
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
        <v-icon icon="mdi-chart-timeline-variant-shimmer" size="14" />
        Meta Ads Suite · Chat Commerce Decision Engine
      </span>
      <h1>{{ l({ lo: 'ສູນວິເຄາະ & ຕັດສິນໃຈ Meta Ads (Ads Intelligence)', th: 'ศูนย์วิเคราะห์และตัดสินใจ Meta Ads', en: 'Meta Ads Decision & Profit Center' }) }}</h1>
      <p>{{ l({ lo: 'ວິເຄາະ Real CPA, ກຳໄລຕົວຈິງຫຼັງຫັກຕົ້ນທຶນ ແລະ ຕິດຕາມຍອດໂອນກ່ອນ vs COD ປ້ອງກັນການປິດແອດຜິດພາດ.', th: 'วิเคราะห์ Real CPA กำไรจริงหลังหักต้นทุน และติดตามยอดโอนก่อน vs COD', en: 'Track Real CPA, net contribution profit, and COD vs prepaid cash flows.' }) }}</p>
    </div>
    <div class="head-actions">
      <v-btn variant="outlined" prepend-icon="mdi-history" @click="openHistoryModal">
        {{ l({ lo: 'ປະຫວັດເກນ', th: 'ประวัติเกณฑ์', en: 'History' }) }}
      </v-btn>
      <v-btn color="secondary" variant="tonal" prepend-icon="mdi-tune-variant" @click="strategyModalOpen = true">
        {{ l({ lo: 'ຕັ້ງຄ່າຍຸດທະສາດ (Setup)', th: 'ตั้งค่ายุทธศาสตร์', en: 'Setup Strategy' }) }}
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
      <v-btn-toggle v-model="viewMode" mandatory color="primary" variant="outlined" density="compact">
        <v-btn value="table" size="small"><v-icon icon="mdi-table-large" start size="14" /> <span>{{ l({lo:'ຕາຕະລາງ',th:'ตาราง',en:'Table'}) }}</span></v-btn>
        <v-btn value="grouped" size="small"><v-icon icon="mdi-package-variant-closed" start size="14" /> <span>{{ l({lo:'ແຍກຕາມສິນຄ້າ',th:'แยกสินค้า',en:'Grouped'}) }}</span></v-btn>
      </v-btn-toggle>
      <v-btn color="primary" variant="tonal" height="40" :loading="loadingReport" @click="loadPerformance">
        {{ l({ lo: 'ນຳໃຊ້', th: 'ใช้ช่วงเวลา', en: 'Apply' }) }}
      </v-btn>
    </div>
  </section>

  <div v-if="loadingReport && !report" class="loading-grid"><v-skeleton-loader v-for="n in 6" :key="n" type="article" /></div>

  <template v-else-if="report">
    <!-- Active Strategy Banner -->
    <div class="strategy-banner mb-4">
      <div class="d-flex align-center justify-between flex-wrap gap-3">
        <div class="d-flex align-center gap-2">
          <v-icon icon="mdi-shield-check" color="success" size="18" />
          <span class="text-caption font-weight-bold">
            {{ l({ lo: 'ເກນຍຸດທະສາດ:', th: 'เกณฑ์ยุทธศาสตร์:', en: 'Strategy:' }) }}
          </span>
          <v-chip size="x-small" color="primary" variant="tonal">
            {{ strategy?.note || 'ເກນມາດຕະຖານ' }}
          </v-chip>
        </div>
        <div class="strategy-metrics-pills">
          <span class="pill">🎯 ເປົ້າກຳໄລ: <b>{{ money(strategy?.targetProfitPerOrder) }}</b>/ອໍເດີ</span>
          <span class="pill">📦 ຄ່າແພັກ+ສົ່ງ: <b>{{ money((strategy?.avgPackagingCost || 0) + (strategy?.avgShippingCostStore || 0)) }}</b></span>
          <span class="pill">🚚 ຕີກັບ COD: <b>{{ strategy?.codReturnRatePercent }}%</b></span>
          <span class="pill">🛑 ງົບລອງແອດ: <b>{{ money(strategy?.maxTestLossBudget) }}</b></span>
          <v-btn size="x-small" variant="text" prepend-icon="mdi-pencil" color="primary" @click="strategyModalOpen = true">ປັບເກນ</v-btn>
        </div>
      </div>
    </div>

    <!-- Action Plan Summary Banner -->
    <v-card v-if="report.actionPlan" class="action-plan-card mb-4" variant="outlined">
      <div class="card-heading">
        <div class="d-flex align-center gap-2">
          <v-icon icon="mdi-clipboard-check-outline" color="primary" size="20" />
          <div>
            <p>ACTION PLAN & RECOMMENDATIONS</p>
            <h2>{{ l({ lo: 'ແຜນປະຕິບັດການ & ຄຳແນະນຳດ່ວນ', th: 'แผนปฏิบัติการและคำแนะนำด่วน', en: 'Action Plan & Next Steps' }) }}</h2>
          </div>
        </div>
        <v-chip color="info" size="small" variant="tonal">
          📅 ວັນກວດຄືນຮອບຖັດໄປ: {{ report.actionPlan.nextReviewDate }}
        </v-chip>
      </div>
      <div class="action-plan-grid">
        <div class="plan-col scale">
          <div class="plan-col-title text-success"><v-icon icon="mdi-trending-up" size="16" /> ສະເໜີ Scale +15-20% ({{ report.actionPlan.toScale.length }})</div>
          <ul v-if="report.actionPlan.toScale.length"><li v-for="t in report.actionPlan.toScale" :key="t">{{ t }}</li></ul>
          <small v-else class="text-muted">ຍັງບໍ່ມີ Campaign ທີ່ Real CPA ຜ່ານເກນ Scale</small>
        </div>
        <div class="plan-col hold">
          <div class="plan-col-title text-info"><v-icon icon="mdi-timer-sand" size="16" /> ຄົງງົບ & ລໍຖ້າ COD ສົ່ງຮອດ ({{ report.actionPlan.toHold.length }})</div>
          <ul v-if="report.actionPlan.toHold.length"><li v-for="t in report.actionPlan.toHold" :key="t">{{ t }}</li></ul>
          <small v-else class="text-muted">ບໍ່ມີອໍເດີ COD ຄ້າງສົ່ງ</small>
        </div>
        <div class="plan-col pause">
          <div class="plan-col-title text-error"><v-icon icon="mdi-pause-circle" size="16" /> ສະເໜີພັກ/ກວດສອບ ({{ report.actionPlan.toPause.length }})</div>
          <ul v-if="report.actionPlan.toPause.length"><li v-for="t in report.actionPlan.toPause" :key="t">{{ t }}</li></ul>
          <small v-else class="text-muted">ບໍ່ມີໂຄສະນາທີ່ສູນເສຍງົບຮົ່ວໄຫຼ</small>
        </div>
      </div>
    </v-card>

    <!-- Zone 1: Financials & Cash Flow Metrics -->
    <div class="text-caption font-weight-bold text-uppercase text-medium-emphasis mb-2">
      💰 {{ l({ lo: 'ການເງິນ & ກະແສເງິນສົດ (Financials & Cash Flow)', th: 'การเงิน & กระแสเงินสด', en: 'Financials & Cash Flow' }) }}
    </div>
    <section class="app-kpi-grid mb-4" aria-label="Financials summary">
      <!-- Total Spend with Gross + VAT -->
      <article class="app-kpi-card accent-primary">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ຄ່າແອດຕົວຈິງ (Gross Spend)', th: 'ยอดใช้จ่ายจริงรวม VAT', en: 'Gross Spend (with VAT)' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-cash" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ money(vatComputation.grossSpend) }}</strong>
        <div class="kpi-sub">
          <span>Net: {{ money(vatComputation.netSpend) }}</span>
          <span class="text-primary font-weight-bold">+ VAT {{ vatDetails.rate }}% ({{ money(vatComputation.vatAmount) }})</span>
        </div>
      </article>

      <!-- Real CPA -->
      <article class="app-kpi-card" :class="((summary?.overallRealCpa || 0) <= (strategy?.targetProfitPerOrder || 30000)) ? 'accent-success' : 'accent-error'">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'Real CPA ສະເລ່ຍ', th: 'Real CPA เฉลี่ย', en: 'Overall Real CPA' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-target" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ money(summary?.overallRealCpa) }}</strong>
        <div class="kpi-sub">
          <span>ເປົ້າໝາຍ: <b>{{ money(strategy?.targetProfitPerOrder) }}</b> / ອໍເດີ</span>
        </div>
      </article>

      <!-- Net Profit -->
      <article class="app-kpi-card" :class="(summary?.totalNetProfit || 0) >= 0 ? 'accent-success' : 'accent-error'">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ກຳໄລສຸດທິຫຼັງຫັກແອດ', th: 'กำไรสุทธิหลังหักโฆษณา', en: 'Net Contribution Profit' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-finance" size="16" /></span>
        </div>
        <strong class="kpi-value" :class="(summary?.totalNetProfit || 0) >= 0 ? 'text-success' : 'text-error'">
          {{ money(summary?.totalNetProfit) }}
        </strong>
        <div class="kpi-sub">
          <span>ກຳໄລຂັ້ນຕົ້ນກ່ອນແອດ: {{ money(summary?.totalGrossProfit) }}</span>
        </div>
      </article>

      <!-- Prepaid Track -->
      <article class="app-kpi-card accent-info">
        <div class="kpi-header">
          <span class="kpi-label">💳 {{ l({ lo: 'ໂອນກ່ອນ (Prepaid)', th: 'โอนก่อน', en: 'Prepaid Cash' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-credit-card-check-outline" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ money(summary?.totalPrepaidRevenue) }}</strong>
        <div class="kpi-sub">
          <span>{{ fmt(summary?.totalPrepaidOrders) }} ອໍເດີ ({{ summary?.prepaidPercentage }}% ຄວາມສ່ຽງ 0%)</span>
        </div>
      </article>

      <!-- COD Delivered Track -->
      <article class="app-kpi-card accent-success">
        <div class="kpi-header">
          <span class="kpi-label">🚚 {{ l({ lo: 'COD ສົ່ງສຳເລັດ (Delivered)', th: 'COD ส่งสำเร็จ', en: 'COD Delivered' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-truck-check-outline" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ money(summary?.totalCODDeliveredRev) }}</strong>
        <div class="kpi-sub">
          <span>{{ fmt(summary?.totalCODDelivered) }} ອໍເດີ ຮັບເງິນຄົບແລ້ວ</span>
        </div>
      </article>
    </section>

    <!-- Zone 2: Operations, Orders & Risk Control -->
    <div class="text-caption font-weight-bold text-uppercase text-medium-emphasis mb-2">
      🎯 {{ l({ lo: 'ປະສິດທິພາບແອດ & ຄວບຄຸມຄວາມສ່ຽງ (Operations & Risk Control)', th: 'ประสิทธิภาพแอด & ควบคุมความเสี่ยง', en: 'Operations & Risk Control' }) }}
    </div>
    <section class="app-kpi-grid mb-6" aria-label="Operations summary">
      <!-- Completed Orders -->
      <article class="app-kpi-card accent-success">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ອໍເດີສຳເລັດ (ຮັບເງິນແລ້ວ)', th: 'ออเดอร์สำเร็จ', en: 'Completed Orders' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-check-decagram" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ fmt(summary?.totalCompletedOrders) }} ອໍເດີ</strong>
        <div class="kpi-sub">
          <span>ຍອດຂາຍຕົວຈິງ: {{ money(summary?.totalCompletedRevenue) }}</span>
        </div>
      </article>

      <!-- Chats & Closing Rate -->
      <article class="app-kpi-card accent-info">
        <div class="kpi-header">
          <span class="kpi-label">{{ l({ lo: 'ແຊັດ & ອັດຕາປິດການຂາຍ', th: 'แชท & อัตราปิดการขาย', en: 'Chats & Closing Rate' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-message-text-outline" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ fmt(summary?.totalChats) }} ແຊັດ</strong>
        <div class="kpi-sub">
          <span>ປິດຂາຍ: <b>{{ summary?.totalChats ? percent(((summary?.totalCompletedOrders || 0) / summary.totalChats) * 100) : '—' }}</b></span>
        </div>
      </article>

      <!-- COD Pending in Transit -->
      <article class="app-kpi-card accent-warning">
        <div class="kpi-header">
          <span class="kpi-label">⏳ {{ l({ lo: 'COD ກຳລັງສົ່ງ (Pending)', th: 'COD กำลังส่ง', en: 'COD In Transit' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-timer-sand" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ money(summary?.totalCODPendingRev) }}</strong>
        <div class="kpi-sub">
          <span>{{ fmt(summary?.totalCODPending) }} ອໍເດີ (ກຳລັງເດີນທາງ ຫ້າມປິດແອດ)</span>
        </div>
      </article>

      <!-- COD Returned -->
      <article class="app-kpi-card accent-error">
        <div class="kpi-header">
          <span class="kpi-label">↩️ {{ l({ lo: 'COD ຕີກັບ (Returned)', th: 'COD ตีกลับ', en: 'COD Returned' }) }}</span>
          <span class="kpi-icon"><v-icon icon="mdi-keyboard-return" size="16" /></span>
        </div>
        <strong class="kpi-value">{{ fmt(summary?.totalCODReturned) }} ອໍເດີ</strong>
        <div class="kpi-sub">
          <span>ເສຍຄ່າສົ່ງກັບປະມານ {{ money((summary?.totalCODReturned || 0) * (strategy?.returnFeePerOrder || 15000)) }}</span>
        </div>
      </article>
    </section>

    <!-- 8-Column Performance Table with 6 SOP Statuses -->
    <section class="performance-section">
      <div class="section-head">
        <div>
          <p>8-COLUMN PERFORMANCE & ACTION MATRIX</p>
          <h2>{{ l({ lo: 'ຕາຕະລາງວິເຄາະ 8 ຖັນ & ລະບົບຕັດສິນໃຈ', th: 'ตารางวิเคราะห์ 8 คอลัมน์', en: '8-Column Performance & SOP Matrix' }) }}</h2>
        </div>
        <div class="report-meta">
          <v-select
            v-model="sortBy"
            :items="[
              { title: '⚡ ຄ່າແອດສູງສຸດ (Spend)', value: 'spend' },
              { title: '🎯 Real CPA ຕ່ຳສຸດ (ດີສຸດ)', value: 'cpa' },
              { title: '💰 ກຳໄລສຸດທິສູງສຸດ', value: 'profit' },
              { title: '📦 ຈຳນວນອໍເດີສູງສຸດ', value: 'orders' },
              { title: '🔤 ຊື່ (A-Z)', value: 'name' }
            ]"
            item-title="title"
            item-value="value"
            density="compact"
            variant="outlined"
            hide-details
            class="sort-select"
            style="min-width: 190px;"
          />
          <v-text-field v-model="search" density="compact" variant="outlined" prepend-inner-icon="mdi-magnify" :label="l({lo:'ຄົ້ນຫາ...',th:'ค้นหา...',en:'Search...'})" hide-details clearable />
        </div>
      </div>

      <!-- Controls & Status Filter Chips -->
      <div class="detail-toolbar">
        <div class="toolbar-group">
          <small>{{ l({lo:'ຮູບແບບການເບິ່ງ',th:'รูปแบบการดู',en:'View mode'}) }}</small>
          <v-btn-toggle v-model="viewMode" mandatory color="primary" variant="outlined" class="level-tabs">
            <v-btn value="table"><v-icon icon="mdi-table-large" /> <span>{{ l({lo:'ຕາຕະລາງລວມ',th:'ตารางรวม',en:'Flat Table'}) }}</span></v-btn>
            <v-btn value="grouped"><v-icon icon="mdi-package-variant-closed" /> <span>{{ l({lo:'ແຍກຕາມສິນຄ້າ',th:'แยกตามสินค้า',en:'Group by Product'}) }}</span></v-btn>
          </v-btn-toggle>
        </div>
        <div class="toolbar-group">
          <small>{{ l({lo:'ລະດັບຂໍ້ມູນ',th:'ระดับข้อมูล',en:'View level'}) }}</small>
          <v-btn-toggle v-model="level" mandatory color="primary" variant="outlined" class="level-tabs">
            <v-btn value="campaign"><v-icon icon="mdi-bullhorn-outline" /> <span>Campaign</span></v-btn>
            <v-btn value="adset"><v-icon icon="mdi-account-group-outline" /> <span>Ad Set</span></v-btn>
            <v-btn value="ad"><v-icon icon="mdi-advertisements" /> <span>Ads</span></v-btn>
          </v-btn-toggle>
        </div>
        <div class="toolbar-group status-filter-group">
          <small>{{ l({lo:'ກອງຕາມການຕັດສິນໃຈ SOP',th:'กรองตามการตัดสินใจ',en:'SOP Decision Filter'}) }}</small>
          <v-btn-toggle v-model="statusFilter" mandatory color="primary" variant="outlined" class="status-tabs">
            <v-btn value="all">ທັງໝົດ ({{ report.items.length }})</v-btn>
            <v-btn value="scale" color="success"><v-icon icon="mdi-trending-up" size="14" /> SCALE</v-btn>
            <v-btn value="hold" color="info"><v-icon icon="mdi-timer-sand" size="14" /> HOLD</v-btn>
            <v-btn value="improve" color="warning"><v-icon icon="mdi-wrench" size="14" /> IMPROVE</v-btn>
            <v-btn value="pause" color="error"><v-icon icon="mdi-pause-circle" size="14" /> PAUSE</v-btn>
          </v-btn-toggle>
        </div>
      </div>

      <div v-if="!items.length" class="empty-state">
        <v-icon icon="mdi-filter-off-outline" size="44" />
        <h3>{{ l({lo:'ບໍ່ພົບລາຍການຕາມ filter',th:'ไม่พบรายการ',en:'No items match filters'}) }}</h3>
        <p>{{ l({lo:'ລອງປ່ຽນຕົວກອງ ຫຼື ເລືອກຊ່ວງເວລາອື່ນ.',th:'ลองเปลี่ยนตัวกรอง',en:'Try selecting another filter or date range.'}) }}</p>
      </div>

      <!-- Matrix Rows -->
      <div v-else-if="viewMode === 'table'" class="performance-list">
        <!-- Table Header (Desktop) -->
        <div class="matrix-header">
          <div class="col col-entity">1. Campaign / Ad</div>
          <div class="col col-status">2. ສະຖານະ & SOP</div>
          <div class="col col-chats">3. ແຊັດ & ປິດຂາຍ</div>
          <div class="col col-spend">4. ຄ່າແອດ (Spend)</div>
          <div class="col col-orders">5. ອໍເດີສຳເລັດ</div>
          <div class="col col-cpa">6. Real CPA</div>
          <div class="col col-profit">7. ກຳໄລສຸດທິ</div>
          <div class="col col-action">8. ຄຳແນະນຳ & ຈັດການ</div>
        </div>

        <article v-for="item in items" :key="item.entityId" class="performance-row" :class="{ expanded: selectedId === item.entityId }">
          <div class="row-main" @click="toggleItem(item)">
            <!-- Col 1: Entity Name & Meta Info -->
            <div class="col col-entity">
              <span class="entity-icon"><v-icon :icon="item.entityType==='campaign'?'mdi-bullhorn-outline':item.entityType==='adset'?'mdi-account-group-outline':'mdi-advertisements'" /></span>
              <div class="entity-text">
                <strong>{{ item.entityName || item.entityId }}</strong>
                <small v-if="item.campaignName && item.entityType!=='campaign'">{{ item.campaignName }}</small>
                <code>{{ item.entityId }}</code>
                <!-- Product & Auto Reply Badge -->
                <div v-if="item.isLinkedToProduct" class="d-flex align-center gap-1 mt-1 flex-wrap">
                  <v-chip size="x-small" color="primary" variant="flat" class="font-weight-bold">
                    <v-avatar start v-if="item.productImageUrl" size="14">
                      <v-img :src="item.productImageUrl" cover />
                    </v-avatar>
                    {{ item.productName }}
                  </v-chip>
                  <span class="text-caption text-medium-emphasis">({{ item.autoReplyFlowName }})</span>
                </div>
                <div v-else class="d-flex align-center gap-1 mt-1">
                  <v-chip size="x-small" color="grey" variant="text" class="text-caption">
                    {{ l({ lo: '⚠️ ຍັງບໍ່ຜູກ Auto Reply', th: '⚠️ ยังไม่ผูก Auto Reply', en: '⚠️ Unlinked' }) }}
                  </v-chip>
                  <NuxtLink to="/auto-replies" class="text-caption text-primary text-decoration-none font-weight-medium" @click.stop>
                    + ຜູກແອດນີ້
                  </NuxtLink>
                </div>
              </div>
            </div>

            <!-- Col 2: Status & SOP Badge -->
            <div class="col col-status">
              <div class="d-flex flex-column gap-1">
                <span class="delivery-chip" :class="statusClass(item.effectiveStatus || item.status)">
                  <span class="status-dot" :class="statusClass(item.effectiveStatus || item.status)" />
                  {{ statusLabel(item.effectiveStatus || item.status) }}
                </span>
                <v-chip size="x-small" :color="decisionBadgeInfo(item.decisionCode).color" variant="flat" class="font-weight-bold">
                  <v-icon :icon="decisionBadgeInfo(item.decisionCode).icon" start size="12" />
                  {{ decisionBadgeInfo(item.decisionCode).label }}
                </v-chip>
              </div>
            </div>

            <!-- Col 3: Chats & Closing Rate -->
            <div class="col col-chats">
              <span class="col-kpi-label">💬 ແຊັດ & ປິດຂາຍ</span>
              <div>
                <strong>{{ fmt(item.reportedResults) }} <small class="font-weight-normal text-caption">ແຊັດ</small></strong>
                <small class="text-medium-emphasis ml-2">ປິດຂາຍ: {{ percent(item.closingRate ? item.closingRate * 100 : null) }}</small>
              </div>
            </div>

            <!-- Col 4: Spend -->
            <div class="col col-spend">
              <span class="col-kpi-label">💳 ຄ່າແອດ (Spend)</span>
              <strong>{{ money(item.spend) }}</strong>
              <small class="text-caption text-medium-emphasis">CPM: {{ money(item.cpm) }}</small>
            </div>

            <!-- Col 5: Completed Orders & Payment Tracks -->
            <div class="col col-orders">
              <span class="col-kpi-label">📦 ອໍເດີສຳເລັດ</span>
              <strong>{{ fmt(item.completedOrders) }} <small class="font-weight-normal text-caption">ອໍເດີ</small></strong>
              <div class="order-sub-track">
                <span class="text-success">💳 ໂອນ: {{ item.prepaidOrders }}</span>
                <span class="text-info">🚚 COD: {{ item.codDeliveredOrders }}</span>
                <span v-if="item.codPendingOrders > 0" class="text-warning">⏳ ສົ່ງຢູ່: {{ item.codPendingOrders }}</span>
              </div>
            </div>

            <!-- Col 6: Real CPA vs Target -->
            <div class="col col-cpa">
              <span class="col-kpi-label">🎯 Real CPA</span>
              <strong :class="{ 'text-success': item.realCpa && item.realCpa <= item.targetCpa, 'text-error': item.realCpa && item.realCpa >= item.breakEvenCpa }">
                {{ money(item.realCpa) }}
              </strong>
              <small class="text-caption text-medium-emphasis">
                ເປົ້າ: {{ money(item.targetCpa) }}
              </small>
            </div>

            <!-- Col 7: Net Contribution Profit -->
            <div class="col col-profit">
              <span class="col-kpi-label">💰 ກຳໄລສຸດທິ</span>
              <strong :class="item.netContribution >= 0 ? 'text-success' : 'text-error'">
                {{ money(item.netContribution) }}
              </strong>
              <small class="text-caption text-medium-emphasis">
                ຍອດຂາຍ: {{ money(item.completedRevenue) }}
              </small>
            </div>

            <!-- Col 8: Action & Quick Control -->
            <div class="col col-action" @click.stop>
              <div class="d-flex align-center gap-1">
                <v-btn
                  v-if="item.decisionCode === 'SCALE'"
                  size="x-small"
                  color="success"
                  variant="flat"
                  prepend-icon="mdi-arrow-up-bold"
                  class="font-weight-bold"
                  @click="promptScaleBudget(item)"
                >
                  Scale +20%
                </v-btn>
                <v-btn
                  size="x-small"
                  :color="(item.effectiveStatus || item.status) === 'ACTIVE' ? 'error' : 'success'"
                  variant="tonal"
                  :loading="updatingId === item.entityId"
                  @click="changeStatus(item, (item.effectiveStatus || item.status) === 'ACTIVE' ? 'PAUSED' : 'ACTIVE')"
                >
                  {{ (item.effectiveStatus || item.status) === 'ACTIVE' ? 'ພັກ' : 'ເປີດ' }}
                </v-btn>
                <v-btn size="x-small" variant="text" icon="mdi-chevron-down" :class="{ 'rotate-180': selectedId === item.entityId }" @click="toggleItem(item)" />
              </div>
            </div>
          </div>

          <!-- Expanded Detail Section -->
          <div v-if="selectedId === item.entityId" class="row-detail">
            <!-- SOP Reason & Concrete Action Recommendation -->
            <div class="sop-reason-box" :class="item.statusBadgeColor">
              <div class="d-flex align-start gap-3">
                <v-icon :icon="decisionBadgeInfo(item.decisionCode).icon" size="24" />
                <div>
                  <h4 class="mb-1 font-weight-bold">{{ decisionBadgeInfo(item.decisionCode).label }}: {{ item.reason }}</h4>
                  <p class="mb-0 text-body-2">{{ item.recommendedAction }}</p>
                </div>
              </div>
            </div>

            <!-- Financial Breakdown Table for this Ad -->
            <div class="metric-detail-grid">
              <div><span>ລາຍຮັບຕົວຈິງ</span><strong>{{ money(item.completedRevenue) }}</strong><small>ໂອນກ່ອນ + COD ຮັບເງິນແລ້ວ</small></div>
              <div><span>ຄ່າແອດ (Spend)</span><strong>{{ money(item.spend) }}</strong><small>ຈ່າຍໃຫ້ Meta</small></div>
              <div><span>Real CPA</span><strong>{{ money(item.realCpa) }}</strong><small>ຕົ້ນທຶນແອດຕໍ່ອໍເດີຕົວຈິງ</small></div>
              <div><span>Break-even CPA</span><strong>{{ money(item.breakEvenCpa) }}</strong><small>ຈຸດຄຸ້ມທຶນສູງສຸດ</small></div>
              <div><span>Target CPA</span><strong>{{ money(item.targetCpa) }}</strong><small>ເປົ້າໝາຍກຳໄລທີ່ຕ້ອງການ</small></div>
              <div><span>ກຳໄລສຸດທິ (Net Contribution)</span><strong :class="item.netContribution>=0?'text-success':'text-error'">{{ money(item.netContribution) }}</strong><small>ລາຍຮັບ − ຕົ້ນທຶນ − ຄ່າແອດ</small></div>
            </div>

            <!-- Audience Details (AdSet level) -->
            <section v-if="item.entityType !== 'campaign'" class="audience-panel mt-3">
              <div class="audience-heading">
                <div><p>AUDIENCE & DELIVERY</p><h3>ກຸ່ມເປົ້າໝາຍ ແລະການເຂົ້າເຖິງ</h3></div>
                <small>Ad Set Level</small>
              </div>
              <v-skeleton-loader v-if="audienceLoading[adSetIDFor(item)]" type="list-item-three-line" />
              <div v-else-if="audienceFor(item)" class="audience-metrics">
                <div><span>Audience ຂະໜາດປະມານ</span><strong>{{ fmt(audienceFor(item)?.estimatedMonthlyUsers || audienceFor(item)?.estimatedDailyUsers) }}</strong></div>
                <div><span>Reach</span><strong>{{ fmt(item.reach) }}</strong></div>
                <div><span>Frequency</span><strong>{{ fmt(item.frequency, 2) }}</strong></div>
                <div><span>Link CTR</span><strong>{{ percent(item.linkCtr) }}</strong></div>
              </div>
            </section>
          </div>
        </article>
      </div>

      <!-- Grouped by Product View -->
      <div v-else class="grouped-product-container">
        <section
          v-for="group in groupedByProduct"
          :key="group.productId || '__unlinked__'"
          class="product-group-card mb-6"
        >
          <!-- Product Header -->
          <div class="product-group-header">
            <div class="d-flex align-center gap-3">
              <v-avatar size="46" rounded="lg" color="surface-variant" class="border">
                <v-img v-if="group.productImageUrl" :src="group.productImageUrl" cover />
                <v-icon v-else icon="mdi-package-variant-closed" size="26" color="primary" />
              </v-avatar>
              <div>
                <div class="d-flex align-center gap-2 flex-wrap">
                  <h3 class="text-subtitle-1 font-weight-bold mb-0">{{ group.productName }}</h3>
                  <v-chip v-if="group.isLinked" size="x-small" color="purple-darken-1" variant="flat" class="font-weight-medium">
                    Auto Reply: {{ group.autoReplyFlowName || 'Active' }}
                  </v-chip>
                  <v-chip v-else size="x-small" color="warning" variant="tonal">
                    ⚠️ ຍັງບໍ່ຜູກ Auto Reply
                  </v-chip>
                </div>
                <div class="text-caption text-medium-emphasis">
                  <span v-if="group.productPrice">ລາຄາ: {{ group.productPrice }} · </span>
                  <span>{{ group.items.length }} ແອດ / ແຄມເປນ</span>
                </div>
              </div>
            </div>

            <!-- Group Rollup Metrics -->
            <div class="group-metrics">
              <div class="metric-box">
                <small>ໃຊ້ຈ່າຍລວມ</small>
                <strong>{{ money(group.totalSpend) }}</strong>
              </div>
              <div class="metric-box">
                <small>ອໍເດີສຳເລັດ</small>
                <strong>{{ fmt(group.totalCompletedOrders) }} ບິນ</strong>
              </div>
              <div class="metric-box">
                <small>Real CPA ສະເລ່ຍ</small>
                <strong :class="{ 'text-success': group.averageRealCPA && group.averageRealCPA <= 45000, 'text-error': group.averageRealCPA && group.averageRealCPA > 85000 }">
                  {{ group.averageRealCPA ? money(group.averageRealCPA) : '—' }}
                </strong>
              </div>
              <div class="metric-box">
                <small>ກຳໄລສຸດທິ (Net Profit)</small>
                <strong :class="group.totalNetProfit >= 0 ? 'text-success' : 'text-error'">
                  {{ money(group.totalNetProfit) }}
                </strong>
              </div>
            </div>
          </div>

          <!-- Items in Group -->
          <div class="performance-list border-0">
            <div class="matrix-header">
              <div class="col col-entity">1. Campaign / Ad</div>
              <div class="col col-status">2. ສະຖານະ & SOP</div>
              <div class="col col-chats">3. ແຊັດ & ປິດຂາຍ</div>
              <div class="col col-spend">4. ຄ່າແອດ (Spend)</div>
              <div class="col col-orders">5. ອໍເດີສຳເລັດ</div>
              <div class="col col-cpa">6. Real CPA</div>
              <div class="col col-profit">7. ກຳໄລສຸດທິ</div>
              <div class="col col-action">8. ຄຳແນະນຳ & ຈັດການ</div>
            </div>

            <article v-for="item in group.items" :key="item.entityId" class="performance-row" :class="{ expanded: selectedId === item.entityId }">
              <div class="row-main" @click="toggleItem(item)">
                <div class="col col-entity">
                  <span class="entity-icon"><v-icon :icon="item.entityType==='campaign'?'mdi-bullhorn-outline':item.entityType==='adset'?'mdi-account-group-outline':'mdi-advertisements'" /></span>
                  <div class="entity-text">
                    <strong>{{ item.entityName || item.entityId }}</strong>
                    <small v-if="item.campaignName && item.entityType!=='campaign'">{{ item.campaignName }}</small>
                    <code>{{ item.entityId }}</code>
                  </div>
                </div>

                <div class="col col-status">
                  <div class="d-flex flex-column gap-1">
                    <span class="delivery-chip" :class="statusClass(item.effectiveStatus || item.status)">
                      <span class="status-dot" :class="statusClass(item.effectiveStatus || item.status)" />
                      {{ statusLabel(item.effectiveStatus || item.status) }}
                    </span>
                    <v-chip size="x-small" :color="decisionBadgeInfo(item.decisionCode).color" variant="flat" class="font-weight-bold">
                      <v-icon :icon="decisionBadgeInfo(item.decisionCode).icon" start size="12" />
                      {{ decisionBadgeInfo(item.decisionCode).label }}
                    </v-chip>
                  </div>
                </div>

                <div class="col col-chats">
                  <span class="col-kpi-label">💬 ແຊັດ & ປິດຂາຍ</span>
                  <div>
                    <strong>{{ fmt(item.reportedResults) }} <small class="font-weight-normal text-caption">ແຊັດ</small></strong>
                    <small class="text-medium-emphasis ml-2">ປິດຂາຍ: {{ percent(item.closingRate ? item.closingRate * 100 : null) }}</small>
                  </div>
                </div>

                <div class="col col-spend">
                  <span class="col-kpi-label">💳 ຄ່າແອດ (Spend)</span>
                  <strong>{{ money(item.spend) }}</strong>
                  <small class="text-caption text-medium-emphasis">CPM: {{ money(item.cpm) }}</small>
                </div>

                <div class="col col-orders">
                  <span class="col-kpi-label">📦 ອໍເດີສຳເລັດ</span>
                  <strong>{{ fmt(item.completedOrders) }} <small class="font-weight-normal text-caption">ອໍເດີ</small></strong>
                  <div class="order-sub-track">
                    <span class="text-success">💳 ໂອນ: {{ item.prepaidOrders }}</span>
                    <span class="text-info">🚚 COD: {{ item.codDeliveredOrders }}</span>
                    <span v-if="item.codPendingOrders > 0" class="text-warning">⏳ ສົ່ງຢູ່: {{ item.codPendingOrders }}</span>
                  </div>
                </div>

                <div class="col col-cpa">
                  <span class="col-kpi-label">🎯 Real CPA</span>
                  <strong :class="{ 'text-success': item.realCpa && item.realCpa <= item.targetCpa, 'text-error': item.realCpa && item.realCpa >= item.breakEvenCpa }">
                    {{ money(item.realCpa) }}
                  </strong>
                  <small class="text-caption text-medium-emphasis">ເປົ້າ: {{ money(item.targetCpa) }}</small>
                </div>

                <div class="col col-profit">
                  <span class="col-kpi-label">💰 ກຳໄລສຸດທິ</span>
                  <strong :class="item.netContribution >= 0 ? 'text-success' : 'text-error'">
                    {{ money(item.netContribution) }}
                  </strong>
                </div>

                <div class="col col-action" @click.stop>
                  <div class="d-flex align-center gap-1">
                    <v-btn
                      v-if="item.decisionCode === 'SCALE'"
                      size="x-small"
                      color="success"
                      variant="flat"
                      prepend-icon="mdi-arrow-up-bold"
                      class="font-weight-bold"
                      @click="promptScaleBudget(item)"
                    >
                      Scale +20%
                    </v-btn>
                    <v-btn
                      size="x-small"
                      :color="(item.effectiveStatus || item.status) === 'ACTIVE' ? 'error' : 'success'"
                      variant="tonal"
                      :loading="updatingId === item.entityId"
                      @click="changeStatus(item, (item.effectiveStatus || item.status) === 'ACTIVE' ? 'PAUSED' : 'ACTIVE')"
                    >
                      {{ (item.effectiveStatus || item.status) === 'ACTIVE' ? 'ພັກ' : 'ເປີດ' }}
                    </v-btn>
                    <v-btn size="x-small" variant="text" icon="mdi-chevron-down" :class="{ 'rotate-180': selectedId === item.entityId }" @click="toggleItem(item)" />
                  </div>
                </div>
              </div>

              <!-- Expanded Accordion Detail -->
              <div v-if="selectedId === item.entityId" class="row-detail">
                <div class="detail-alert" :class="item.statusBadgeColor">
                  <v-icon :icon="decisionBadgeInfo(item.decisionCode).icon" size="24" />
                  <div>
                    <h4 class="font-weight-bold mb-1">{{ decisionBadgeInfo(item.decisionCode).label }}: {{ item.reason }}</h4>
                    <p class="mb-0 text-caption">{{ item.recommendedAction }}</p>
                  </div>
                </div>

                <div class="metric-detail-grid mt-3">
                  <div><span>ລາຍຮັບຕົວຈິງ</span><strong>{{ money(item.completedRevenue) }}</strong><small>ໂອນກ່ອນ + COD ຮັບເງິນແລ້ວ</small></div>
                  <div><span>ຄ່າແອດ (Spend)</span><strong>{{ money(item.spend) }}</strong><small>ຈ່າຍໃຫ້ Meta</small></div>
                  <div><span>Real CPA</span><strong>{{ money(item.realCpa) }}</strong><small>ຕົ້ນທຶນແອດຕໍ່ອໍເດີຕົວຈິງ</small></div>
                  <div><span>Break-even CPA</span><strong>{{ money(item.breakEvenCpa) }}</strong><small>ຈຸດຄຸ້ມທຶນສູງສຸດ</small></div>
                  <div><span>Target CPA</span><strong>{{ money(item.targetCpa) }}</strong><small>ເປົ້າໝາຍກຳໄລທີ່ຕ້ອງການ</small></div>
                  <div><span>ກຳໄລສຸດທິ (Net Contribution)</span><strong :class="item.netContribution>=0?'text-success':'text-error'">{{ money(item.netContribution) }}</strong><small>ລາຍຮັບ − ຕົ້ນທຶນ − ຄ່າແອດ</small></div>
                </div>
              </div>
            </article>
          </div>
        </section>
      </div>
    </section>
  </template>

  <!-- Setup Strategy Settings Modal (1-minute prompt) -->
  <v-dialog v-model="strategyModalOpen" max-width="640" persistent>
    <v-card rounded="xl">
      <v-card-title class="dialog-title">
        <div>
          <small class="text-primary font-weight-bold">SETUP AD STRATEGY</small>
          <h2>ຕັ້ງຄ່າຍຸດທະສາດໂຄສະນາ (Setup Strategy)</h2>
        </div>
        <v-btn icon="mdi-close" variant="text" :disabled="savingStrategy" @click="strategyModalOpen = false" />
      </v-card-title>
      <v-card-text class="settings-body">
        <v-alert type="info" variant="tonal" density="compact" class="mb-4">
          ປ້ອນ 4 ຕົວເລກຫຼັກໆ ໃຊ້ເວລາພຽງ 1 ນາທີ. ລະບົບຈະບັນທຶກເປັນ <b>Version ຕາມວັນທີ (Point-in-Time)</b> ເພື່ອໃຫ້ລາຍງານຍ້ອນຫຼັງຖືກຕ້ອງຕະຫຼອດໄປ.
        </v-alert>

        <div class="settings-grid primary-settings mb-3">
          <v-text-field
            v-model.number="strategyForm.targetProfitPerOrder"
            type="number"
            label="1. ກຳໄລທີ່ຕ້ອງການຕໍ່ອໍເດີ (Target Profit)"
            suffix="₭"
            hint="ເຊັ່ນ: 30,000 ກີບ/ອໍເດີ"
            persistent-hint
            variant="outlined"
          />
          <v-text-field
            v-model.number="strategyForm.maxTestLossBudget"
            type="number"
            label="2. ງົບທົດລອງ / ງົບຂາດທຶນສູງສຸດຕໍ່ມື້"
            suffix="₭"
            hint="ເຊັ່ນ: 200,000 ກີບ/ມື້ (ຖ້າ Spend ຮອດແລ້ວບໍ່ມີອໍເດີ ຈະສະເໜີພັກ)"
            persistent-hint
            variant="outlined"
          />
        </div>

        <div class="settings-grid mb-3">
          <v-text-field
            v-model.number="strategyForm.avgPackagingCost"
            type="number"
            label="3.1 ຄ່າແພັກສະເລ່ຍ/ອໍເດີ"
            suffix="₭"
            variant="outlined"
          />
          <v-text-field
            v-model.number="strategyForm.avgShippingCostStore"
            type="number"
            label="3.2 ຄ່າສົ່ງທີ່ຮ້ານຊ່ວຍອອກ/ອໍເດີ"
            suffix="₭"
            variant="outlined"
          />
          <v-text-field
            v-model.number="strategyForm.codReturnRatePercent"
            type="number"
            label="4. ອັດຕາຕີກັບ COD ປະມານການ"
            suffix="%"
            hint="ເຊັ່ນ: 8%"
            persistent-hint
            variant="outlined"
          />
        </div>

        <div class="settings-grid mb-3">
          <v-text-field
            v-model.number="strategyForm.returnFeePerOrder"
            type="number"
            label="ຄ່າສົ່ງຕີກັບຕໍ່ອໍເດີ (Return Fee)"
            suffix="₭"
            variant="outlined"
          />
          <v-text-field
            v-model.number="strategyForm.dailyBudgetCap"
            type="number"
            label="ເພດານງົບຕໍ່ມື້ສູງສຸດ (Cap)"
            suffix="₭"
            variant="outlined"
          />
          <v-text-field
            v-model="strategyForm.effectiveFrom"
            type="date"
            label="ມີຜົນເລີ່ມຕັ້ງແຕ່ວັນທີ"
            variant="outlined"
          />
        </div>

        <v-text-field
          v-model="strategyForm.note"
          label="ໝາຍເຫດການປ່ຽນແປງ (Note)"
          placeholder="ເຊັ່ນ: ປັບເກນເດືອນ 10 ຍ້ອນຕົ້ນທຶນຄ່າສົ່ງເພີ່ມ"
          variant="outlined"
        />
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-btn variant="text" :disabled="savingStrategy" @click="strategyModalOpen = false">ຍົກເລີກ</v-btn>
        <v-spacer />
        <v-btn color="primary" :loading="savingStrategy" @click="saveStrategySettings">ບັນທຶກເກນຍຸດທະສາດ</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Strategy Settings Version History Modal -->
  <v-dialog v-model="historyModalOpen" max-width="700">
    <v-card rounded="xl">
      <v-card-title class="dialog-title">
        <div>
          <small class="text-primary font-weight-bold">STRATEGY TIMELINE</small>
          <h2>ປະຫວັດເກນຍຸດທະສາດ (Version History)</h2>
        </div>
        <v-btn icon="mdi-close" variant="text" @click="historyModalOpen = false" />
      </v-card-title>
      <v-card-text class="settings-body">
        <v-skeleton-loader v-if="loadingHistory" type="list-item-three-line, list-item-three-line" />
        <div v-else-if="strategyHistory.length" class="history-list">
          <div v-for="item in strategyHistory" :key="item.id" class="history-card">
            <div class="d-flex justify-between align-center mb-1">
              <strong>{{ item.note || 'ເກນຍຸດທະສາດ' }}</strong>
              <v-chip size="x-small" :color="!item.effectiveTo ? 'success' : 'default'">
                {{ !item.effectiveTo ? 'Active ໃນປະຈຸບັນ' : 'Archived' }}
              </v-chip>
            </div>
            <div class="text-caption text-medium-emphasis mb-2">
              ມີຜົນ: {{ item.effectiveFrom?.slice(0, 10) }} ເຖິງ {{ item.effectiveTo ? item.effectiveTo.slice(0, 10) : 'ປະຈຸບັນ' }}
            </div>
            <div class="history-grid-pills">
              <span>🎯 ເປົ້າກຳໄລ: {{ money(item.targetProfitPerOrder) }}</span>
              <span>📦 ຄ່າແພັກ: {{ money(item.avgPackagingCost) }}</span>
              <span>🚚 ຄ່າສົ່ງຮ້ານອອກ: {{ money(item.avgShippingCostStore) }}</span>
              <span>🛑 ງົບທົດລອງ: {{ money(item.maxTestLossBudget) }}</span>
              <span>↩️ COD Return: {{ item.codReturnRatePercent }}%</span>
            </div>
          </div>
        </div>
        <div v-else class="empty-inline">ຍັງບໍ່ມີປະຫວັດການປ່ຽນເກນ</div>
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-spacer />
        <v-btn color="primary" @click="historyModalOpen = false">ປິດ</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <!-- Scale Budget Modal -->
  <v-dialog v-model="budgetModalOpen" max-width="500">
    <v-card rounded="xl">
      <v-card-title class="dialog-title">
        <div>
          <small class="text-success font-weight-bold">SCALE BUDGET</small>
          <h2>ເພີ່ມງົບປະມານໂຄສະນາ (+20%)</h2>
        </div>
        <v-btn icon="mdi-close" variant="text" :disabled="updatingBudget" @click="budgetModalOpen = false" />
      </v-card-title>
      <v-card-text class="settings-body">
        <v-alert type="success" variant="tonal" density="compact" class="mb-4">
          ໂຄສະນາ “{{ targetBudgetItem?.entityName }}” ມີ Real CPA ດີກວ່າເປົ້າໝາຍ. ແນະນຳເພີ່ມງົບ 15-20% ເພື່ອຂະຫຍາຍຍອດຂາຍ.
        </v-alert>

        <div class="mb-3">
          <small class="text-medium-emphasis">ຄ່າໃຊ້ຈ່າຍສະເລ່ຍໃນຊ່ວງນີ້: {{ money(targetBudgetItem?.spend) }}</small>
        </div>

        <v-text-field
          v-model.number="newDailyBudget"
          type="number"
          label="ງົບປະມານຕໍ່ມື້ໃໝ່ (Daily Budget)"
          suffix="₭"
          variant="outlined"
          hint="ແນະນຳເພີ່ມ 15-20% ເພື່ອບໍ່ໃຫ້ Algorithm ຕົກໃຈ (Learning Reset)"
          persistent-hint
        />
      </v-card-text>
      <v-card-actions class="dialog-actions">
        <v-btn variant="text" :disabled="updatingBudget" @click="budgetModalOpen = false">ຍົກເລີກ</v-btn>
        <v-spacer />
        <v-btn color="success" :loading="updatingBudget" @click="applyBudgetChange">ຢືນຢັນເພີ່ມງົບ</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.page-head,.filter-bar,.performance-section{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
.page-head{padding:24px;display:flex;justify-content:space-between;align-items:flex-start;gap:24px;margin-bottom:16px}
.page-head h1{font-size:clamp(22px,2.5vw,30px);line-height:1.2;margin:0 0 6px}
.head-sub{color:var(--color-text-secondary);font-size:13px;line-height:1.5}
.head-actions{display:flex;gap:8px;flex-wrap:wrap}

.filter-bar{display:grid;grid-template-columns:minmax(240px,1.5fr) minmax(170px,.8fr) minmax(145px,.7fr) minmax(145px,.7fr) auto;align-items:center;gap:12px;padding:16px;margin-bottom:16px}

.strategy-banner{padding:14px 18px;background:color-mix(in srgb,var(--color-primary) 8%,var(--color-surface));border:1px solid color-mix(in srgb,var(--color-primary) 25%,var(--color-border));border-radius:var(--radius-lg)}
.strategy-metrics-pills{display:flex;align-items:center;gap:8px;flex-wrap:wrap}
.strategy-metrics-pills .pill{background:var(--color-surface);padding:4px 8px;border-radius:8px;border:1px solid var(--color-border);font-size: 12px}

.action-plan-card{padding:18px;border-radius:var(--radius-lg);background:var(--color-surface)}
.action-plan-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:14px;margin-top:14px}
.plan-col{padding:14px;border-radius:12px;border:1px solid var(--color-border);background:var(--color-background)}
.plan-col-title{font-size:12px;font-weight:750;margin-bottom:8px;display:flex;align-items:center;gap:6px}
.plan-col ul{margin:0;padding-left:18px;font-size:12px;line-height:1.6}

.summary-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-bottom:16px}
.summary-card{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);padding:16px;display:grid;gap:5px}
.summary-card>span{color:var(--color-text-secondary);font-size:12px}
.summary-card strong{font-size:clamp(20px,2.4vw,26px);line-height:1.1}
.summary-card small{color:var(--color-text-muted);font-size: 12px}
.summary-card.finance{border-color:rgba(32,164,100,.3);background:linear-gradient(145deg,var(--color-surface),rgba(32,164,100,.04))}
.summary-card.finance strong{color:#167a4b}
.summary-card.loss{border-color:rgba(239,68,68,.3);background:linear-gradient(145deg,var(--color-surface),rgba(239,68,68,.04))}
.summary-card.loss strong{color:var(--color-error)}
.summary-card.prepaid-track{border-color:rgba(14,165,233,.3);background:linear-gradient(145deg,var(--color-surface),rgba(14,165,233,.04))}
.summary-card.cod-delivered-track{border-color:rgba(16,185,129,.3)}
.summary-card.cod-pending-track{border-color:rgba(245,158,11,.3);background:linear-gradient(145deg,var(--color-surface),rgba(245,158,11,.04))}
.summary-card.cod-returned-track{border-color:rgba(239,68,68,.25)}

.performance-section{padding:20px}
.section-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;margin-bottom:14px}
.section-head p{margin:0 0 4px;color:var(--color-primary);font-size: 12px;font-weight:800;letter-spacing:.12em}
.section-head h2{margin:0;font-size:20px}
.report-meta{display:flex;align-items:center;gap:12px}

.detail-toolbar{display:flex;align-items:flex-end;justify-content:space-between;gap:16px;padding:12px 16px;margin-bottom:12px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-surface-soft)}
.toolbar-group{display:grid;gap:6px}
.toolbar-group>small{font-size: 12px;font-weight:750;color:var(--color-text-muted);letter-spacing:.04em}

.matrix-header{display:grid;grid-template-columns:minmax(220px,1.6fr) minmax(130px,.8fr) minmax(90px,.6fr) minmax(110px,.7fr) minmax(120px,.8fr) minmax(100px,.7fr) minmax(110px,.7fr) minmax(130px,.9fr);gap:10px;padding:10px 16px;background:var(--color-surface-soft);border:1px solid var(--color-border);border-radius:10px 10px 0 0;font-size: 12px;font-weight:750;color:var(--color-text-secondary)}

.performance-list{display:grid;border:1px solid var(--color-border);border-top:0;border-radius:0 0 14px 14px;overflow:hidden}
.performance-row+.performance-row{border-top:1px solid var(--color-border)}
.performance-row.expanded{background:var(--color-surface-soft)}

.row-main{width:100%;display:grid;grid-template-columns:minmax(220px,1.6fr) minmax(130px,.8fr) minmax(90px,.6fr) minmax(110px,.7fr) minmax(120px,.8fr) minmax(100px,.7fr) minmax(110px,.7fr) minmax(130px,.9fr);align-items:center;gap:10px;padding:12px 16px;text-align:left;color:inherit;cursor:pointer}
.row-main:hover{background:var(--color-background)}

.col-entity{display:flex;align-items:center;gap:10px;min-width:0}
.entity-icon{width:38px;height:38px;display:grid;place-items:center;border-radius:10px;background:var(--color-primary-soft);color:var(--color-primary);flex:0 0 38px}
.entity-text{display:grid;min-width:0}
.entity-text strong,.entity-text small,.entity-text code{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.entity-text small{color:var(--color-text-secondary);font-size: 12px}
.entity-text code{color:var(--color-text-muted);font-size: 12px}

.order-sub-track{display:flex;gap:6px;font-size: 12px}
.delivery-chip{display:inline-flex;align-items:center;gap:5px;padding:3px 7px;border-radius:999px;font-size: 12px;font-weight:700}
.delivery-chip.active{color:#167a4b;background:rgba(32,164,100,.1)}
.delivery-chip.paused{color:#697184;background:rgba(137,145,165,.13)}
.delivery-chip.issues{color:#a96000;background:rgba(229,138,24,.12)}
.status-dot{width:7px;height:7px;border-radius:50%;display:inline-block}
.status-dot.active{background:#20a464}
.status-dot.paused{background:#8991a5}
.status-dot.issues{background:#e58a18}

.row-detail{padding:16px 20px;border-top:1px solid var(--color-border);display:grid;gap:14px;background:var(--color-surface)}
.sop-reason-box{padding:14px 18px;border-radius:12px;border:1px solid var(--color-border)}
.sop-reason-box.success{background:rgba(32,164,100,.08);border-color:rgba(32,164,100,.3);color:#167a4b}
.sop-reason-box.info{background:rgba(2,132,199,.08);border-color:rgba(2,132,199,.3);color:#0369a1}
.sop-reason-box.warning{background:rgba(217,119,6,.08);border-color:rgba(217,119,6,.3);color:#b45309}
.sop-reason-box.error{background:rgba(220,38,38,.08);border-color:rgba(220,38,38,.3);color:#b91c1c}
.sop-reason-box.secondary{background:var(--color-surface-soft);border-color:var(--color-border);color:var(--color-text-secondary)}

.metric-detail-grid{display:grid;grid-template-columns:repeat(6,1fr);gap:10px}
.metric-detail-grid>div{padding:10px 12px;border:1px solid var(--color-border);border-radius:10px;display:grid;gap:3px;background:var(--color-background)}
.metric-detail-grid span,.metric-detail-grid small{font-size: 12px;color:var(--color-text-muted)}
.metric-detail-grid strong{font-size:14px}

.audience-panel{padding:14px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-surface)}
.audience-heading{display:flex;align-items:center;justify-content:space-between;margin-bottom:10px}
.audience-heading p{margin:0;font-size: 12px;font-weight:800;color:var(--color-primary)}
.audience-heading h3{margin:0;font-size:14px}
.audience-metrics{display:grid;grid-template-columns:repeat(4,1fr);gap:8px}
.audience-metrics>div{padding:8px 10px;border-radius:8px;background:var(--color-background);display:grid}
.audience-metrics span{font-size: 12px;color:var(--color-text-muted)}
.audience-metrics strong{font-size:15px}

.rotate-180{transform:rotate(180deg)}
.dialog-title{display:flex;justify-content:space-between;align-items:flex-start;padding:20px}
.dialog-title small{font-size: 12px;letter-spacing:.1em}
.dialog-title h2{margin:2px 0 0;font-size:20px}
.settings-body{padding:0 20px 12px}
.settings-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px}
.settings-grid.primary-settings{grid-template-columns:repeat(2,1fr)}
.dialog-actions{padding:12px 20px 20px}

.history-list{display:grid;gap:10px;max-height:420px;overflow-y:auto}
.history-card{padding:12px 14px;border:1px solid var(--color-border);border-radius:12px;background:var(--color-background)}
.history-grid-pills{display:flex;flex-wrap:wrap;gap:6px;font-size: 12px}
.history-grid-pills span{background:var(--color-surface);padding:3px 6px;border-radius:6px;border:1px solid var(--color-border)}

.product-group-card{border:1px solid var(--color-border);border-radius:var(--radius-lg);background:var(--color-surface);overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.03)}
.product-group-header{padding:16px 20px;background:linear-gradient(145deg,var(--color-surface),color-mix(in srgb,var(--color-primary) 6%,var(--color-surface)));border-bottom:1px solid var(--color-border);display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:16px}
.group-metrics{display:flex;align-items:center;gap:16px;flex-wrap:wrap}
.metric-box{display:grid;text-align:right;gap:2px}
.metric-box small{font-size: 12px;color:var(--color-text-secondary)}
.metric-box strong{font-size:15px;font-weight:700}

.col-kpi-label { display: none; }
.filter-account-select { min-width: 220px; }
.filter-product-select { min-width: 200px; }
.filter-date-select { min-width: 140px; }
.filter-custom-date { max-width: 130px; }

@media(max-width:1280px){
  .filter-bar{grid-template-columns:repeat(2,1fr)}
  .matrix-header{display:none}
  .row-main{grid-template-columns:1fr 1fr;gap:12px;padding:14px}
  .col-entity{grid-column:1/-1}
  .action-plan-grid{grid-template-columns:1fr}
  .metric-detail-grid{grid-template-columns:repeat(3,1fr)}
}

@media(max-width:720px){
  .col-kpi-label {
    display: block;
    font-size: 12px;
    font-weight: 750;
    color: var(--color-text-muted);
    margin-bottom: 2px;
  }
  .filter-account-select, .filter-product-select, .filter-date-select, .filter-custom-date {
    width: 100% !important;
    min-width: 0 !important;
    max-width: 100% !important;
  }
  .strategy-banner {
    padding: 12px;
  }
  .strategy-metrics-pills {
    gap: 6px;
  }
  .strategy-metrics-pills .pill {
    font-size: 12px;
    padding: 3px 6px;
  }
  .action-plan-card {
    padding: 14px;
  }
  .action-plan-grid {
    grid-template-columns: 1fr;
    gap: 8px;
  }
  .plan-col {
    padding: 10px 12px;
  }

  .performance-section {
    padding: 12px;
    border-radius: 14px;
  }
  .section-head, .report-meta {
    display: grid;
    width: 100%;
  }
  .report-meta .sort-select, .report-meta .v-text-field {
    width: 100% !important;
    max-width: none !important;
  }
  .detail-toolbar {
    position: static;
    margin-inline: 0;
    border-radius: 12px;
    padding: 10px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .level-tabs {
    width: 100%;
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(80px, 1fr));
  }
  .level-tabs .v-btn {
    min-width: 0;
    padding-inline: 6px;
    font-size: 12px;
  }
  .status-filter-group {
    width: 100%;
    overflow: hidden;
  }
  .status-tabs {
    display: flex;
    width: 100%;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    padding-bottom: 3px;
    scrollbar-width: none;
  }
  .status-tabs::-webkit-scrollbar {
    display: none;
  }
  .status-tabs .v-btn {
    flex: 0 0 auto;
    padding-inline: 10px;
    font-size: 12px;
  }

  /* Well-proportioned Mobile Matrix Card */
  .row-main {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    padding: 14px;
    position: relative;
    border-radius: 14px;
  }
  .col-entity {
    grid-column: 1 / -1;
    width: 100%;
    display: flex;
    align-items: flex-start;
    gap: 10px;
  }
  .col-status {
    grid-column: 1 / -1;
    display: flex;
    flex-direction: row !important;
    align-items: center;
    gap: 8px;
    margin-bottom: 2px;
  }
  .col-spend, .col-orders, .col-cpa, .col-profit {
    display: grid;
    padding: 8px 10px;
    border-radius: 9px;
    background: var(--color-background);
    min-width: 0;
  }
  .col-spend strong, .col-orders strong, .col-cpa strong, .col-profit strong {
    font-size: 14px;
    line-height: 1.2;
  }
  .col-spend { grid-column: 1; }
  .col-orders { grid-column: 2; }
  .col-cpa { grid-column: 1; }
  .col-profit { grid-column: 2; }

  .col-chats {
    grid-column: 1 / -1;
    padding: 8px 10px;
    border-radius: 9px;
    background: var(--color-surface-soft);
    display: flex;
    align-items: center;
    justify-content: space-between;
  }
  .col-action {
    grid-column: 1 / -1;
    margin-top: 6px;
    padding-top: 10px;
    border-top: 1px solid var(--color-border);
  }
  .col-action > div {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(110px, 1fr));
    gap: 8px;
    width: 100%;
  }
  .col-action .v-btn {
    width: 100%;
    min-height: 40px;
    font-size: 12.5px;
    font-weight: 700;
  }

  .product-group-header {
    padding: 12px 14px;
    flex-direction: column;
    align-items: flex-start;
    gap: 10px;
  }
  .group-metrics {
    width: 100%;
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }
  .metric-box {
    text-align: left;
    padding: 6px 8px;
    background: var(--color-surface);
    border-radius: 8px;
    border: 1px solid var(--color-border);
  }

  .metric-detail-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 6px;
  }
  .metric-detail-grid > div {
    padding: 8px 10px;
  }
  .metric-detail-grid strong {
    font-size: 13.5px;
  }

  .settings-grid, .settings-grid.primary-settings {
    grid-template-columns: 1fr;
  }
}
</style>
