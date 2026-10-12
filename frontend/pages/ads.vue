<script setup lang="ts">
import type { MetaAdAccount, MetaCampaign, MetaInsightItem, MetaInsightsReport } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { l } = useLocale()

type Level = 'campaign' | 'adset' | 'ad'
type DecisionCode = 'keep' | 'review' | 'pause' | 'no_data'

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
const selectedId = ref('')
const settingsOpen = ref(false)
const ready = ref(false)
const campaignsAccountId = ref('')

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

const selectedAccount = computed(() => adAccounts.value.find(item => item.id === adAccountId.value))
const items = computed(() => {
  const query = search.value.trim().toLowerCase()
  const list = report.value?.items || []
  return list
    .filter(item => !query || [item.entityName, item.campaignName, item.adSetName, item.adName, item.entityId].some(value => value?.toLowerCase().includes(query)))
    .sort((a, b) => (b.spend ?? -1) - (a.spend ?? -1))
})
const summary = computed(() => report.value?.summary)
const currency = computed(() => report.value?.currency || 'USD')
const maxTrendSpend = computed(() => Math.max(1, ...(report.value?.trend || []).map(point => point.spend || 0)))

const fmt = (value?: number, digits = 2) => value == null ? '—' : new Intl.NumberFormat(undefined, { maximumFractionDigits: digits }).format(value)
const money = (value?: number) => value == null ? '—' : new Intl.NumberFormat(undefined, { style: 'currency', currency: currency.value, maximumFractionDigits: 2 }).format(value)
const percent = (value?: number) => value == null ? '—' : `${fmt(value, 2)}%`
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

function setPreset(value: string) {
  datePreset.value = value
  const days = Number(value)
  if (!Number.isFinite(days)) return
  const end = new Date()
  const start = new Date(end.getFullYear(), end.getMonth(), end.getDate() - days + 1)
  since.value = iso(start)
  until.value = iso(end)
}

function statusFor(item: MetaInsightItem) {
  if (item.entityType !== 'campaign') return ''
  return campaigns.value.find(campaign => campaign.id === item.entityId)?.effectiveStatus || ''
}

function decision(item: MetaInsightItem): { code: DecisionCode; color: string; icon: string; title: string; detail: string; signals: string[] } {
  if (!item.impressions && !item.spend) {
    return { code: 'no_data', color: 'default', icon: 'mdi-minus-circle-outline', title: l({ lo: 'ຂໍ້ມູນຍັງບໍ່ພໍ', th: 'ข้อมูลยังไม่พอ', en: 'Not enough data' }), detail: l({ lo: 'ລໍຖ້າໃຫ້ໂຄສະນາມີການສະແດງ ແລະຄ່າໃຊ້ຈ່າຍກ່ອນ.', th: 'รอให้โฆษณามีการแสดงผลและค่าใช้จ่ายก่อน', en: 'Wait for delivery and spend before deciding.' }), signals: [] }
  }
  const bad: string[] = []
  const good: string[] = []
  if (targetCost.value && item.results.cost != null) {
    if (item.results.cost > targetCost.value) bad.push(l({ lo: 'ຕົ້ນທຶນ/ຜົນລັບສູງກວ່າຈຸດຄຸ້ມທຶນ', th: 'ต้นทุน/ผลลัพธ์สูงกว่าจุดคุ้มทุน', en: 'Cost/result is above break-even' }))
    else good.push(l({ lo: 'ຕົ້ນທຶນ/ຜົນລັບຢູ່ໃນເປົ້າ', th: 'ต้นทุน/ผลลัพธ์อยู่ในเป้า', en: 'Cost/result is within target' }))
  }
  if (item.linkCtr != null) {
    if (item.linkCtr < weakCTR.value) bad.push(l({ lo: 'CTR ຕ່ຳ—Creative ອາດບໍ່ດຶງດູດ', th: 'CTR ต่ำ—ครีเอทีฟอาจไม่น่าสนใจ', en: 'Low CTR—creative may not attract clicks' }))
    else if (item.linkCtr >= goodCTR.value) good.push(l({ lo: 'CTR ດີ', th: 'CTR ดี', en: 'Strong CTR' }))
  }
  if (item.frequency != null) {
    if (item.frequency >= maxFrequency.value) bad.push(l({ lo: 'Frequency ສູງ—Audience ເລີ່ມເຫັນຊ້ຳ', th: 'Frequency สูง—กลุ่มเป้าหมายเริ่มเห็นซ้ำ', en: 'High frequency—audience is seeing repeats' }))
    else if (item.frequency < maxFrequency.value - 0.5) good.push(l({ lo: 'Frequency ຍັງປົກກະຕິ', th: 'Frequency ยังปกติ', en: 'Frequency is still healthy' }))
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
  } catch {}
}
function saveThresholds() {
  if (!import.meta.client || !adAccountId.value) return
  localStorage.setItem(`ads-thresholds:${adAccountId.value}`, JSON.stringify({ targetCost: targetCost.value, goodCTR: goodCTR.value, weakCTR: weakCTR.value, maxFrequency: maxFrequency.value }))
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
    const campaignRequest = campaignsAccountId.value === adAccountId.value
      ? Promise.resolve({ items: campaigns.value })
      : $fetch<{ items: MetaCampaign[] }>(`/api/proxy/api/v1/meta/ad-accounts/${encodeURIComponent(adAccountId.value)}/campaigns`)
    const [campaignResult, insightResult] = await Promise.all([
      campaignRequest,
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

async function changeStatus(item: MetaInsightItem, status: 'ACTIVE' | 'PAUSED') {
  if (status === 'PAUSED' && !confirm(l({ lo: `ປິດ “${item.entityName}”? ການສົ່ງໂຄສະນາຈະຢຸດຈົນກວ່າຈະເປີດຄືນ.`, th: `ปิด “${item.entityName}”? การส่งโฆษณาจะหยุดจนกว่าจะเปิดอีกครั้ง`, en: `Pause “${item.entityName}”? Delivery will stop until it is activated again.` }))) return
  updatingId.value = item.entityId
  notice.value = ''
  success.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/meta/ads/${item.entityType}/${encodeURIComponent(item.entityId)}/status`, { method: 'PATCH', body: { status } })
    const campaign = campaigns.value.find(value => value.id === item.entityId)
    if (campaign) campaign.effectiveStatus = status
    success.value = status === 'ACTIVE'
      ? l({ lo: `ເປີດ “${item.entityName}” ສຳເລັດ.`, th: `เปิด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now active.` })
      : l({ lo: `ປິດ “${item.entityName}” ສຳເລັດ.`, th: `ปิด “${item.entityName}” แล้ว`, en: `“${item.entityName}” is now paused.` })
  } catch (error: any) {
    notice.value = error?.data?.error?.code === 'internal_error'
      ? l({ lo: 'Meta ບໍ່ອະນຸຍາດໃຫ້ປ່ຽນສະຖານະ. Reconnect Facebook ແລະເລືອກ Manage ads (ads_management).', th: 'Meta ไม่อนุญาตให้เปลี่ยนสถานะ เชื่อม Facebook ใหม่และเลือก Manage ads (ads_management)', en: 'Meta denied the change. Reconnect Facebook and select Manage ads (ads_management).' })
      : error?.data?.error?.message || error?.data?.message || error?.statusMessage || l({ lo: 'ປ່ຽນສະຖານະບໍ່ສຳເລັດ.', th: 'เปลี่ยนสถานะไม่สำเร็จ', en: 'Could not change status.' })
  } finally { updatingId.value = '' }
}

watch(adAccountId, async () => { if (!ready.value) return; loadThresholds(); await loadPerformance() })
watch(level, async () => { if (ready.value) await loadPerformance() })
onMounted(async () => { await loadAccounts(); loadThresholds(); ready.value = true; if (adAccountId.value) await loadPerformance() })
</script>

<template>
  <section class="page-head">
    <div>
      <p>ADS PERFORMANCE</p>
      <h1>{{ l({ lo: 'ສູນວິເຄາະໂຄສະນາ', th: 'ศูนย์วิเคราะห์โฆษณา', en: 'Ads performance center' }) }}</h1>
      <span>{{ l({ lo: 'ເບິ່ງຜົນງານ, trend ແລະສັນຍານທີ່ຊ່ວຍຕັດສິນໃຈເປີດ ຫຼືປິດໂຄສະນາ.', th: 'ดูผลงาน แนวโน้ม และสัญญาณที่ช่วยตัดสินใจเปิดหรือปิดโฆษณา', en: 'Understand performance, trends, and signals before keeping or pausing ads.' }) }}</span>
    </div>
    <div class="head-actions">
      <v-btn variant="outlined" prepend-icon="mdi-tune-variant" @click="settingsOpen = true">{{ l({ lo: 'ຕັ້ງຄ່າເກນ', th: 'ตั้งค่าเกณฑ์', en: 'Decision settings' }) }}</v-btn>
      <v-btn color="primary" prepend-icon="mdi-refresh" :loading="loadingReport" :disabled="!adAccountId" @click="loadPerformance">{{ l({ lo: 'ອັບເດດຂໍ້ມູນ', th: 'อัปเดตข้อมูล', en: 'Refresh data' }) }}</v-btn>
    </div>
  </section>

  <v-alert v-if="notice" type="error" variant="tonal" class="mb-4" closable @click:close="notice = ''">
    {{ notice }}
    <template #append><v-btn size="small" variant="outlined" to="/meta-pages">{{ l({ lo: 'ກວດສິດ Meta', th: 'ตรวจสิทธิ์ Meta', en: 'Check Meta access' }) }}</v-btn></template>
  </v-alert>
  <v-alert v-if="success" type="success" variant="tonal" class="mb-4" closable @click:close="success = ''">{{ success }}</v-alert>

  <section class="filter-bar" aria-label="Ads report filters">
    <v-select v-model="adAccountId" :items="adAccounts.map(account => ({ ...account, label: `${account.accountType === 'business' ? 'Business' : 'Personal'} · ${account.name}` }))" item-title="label" item-value="id" :label="l({ lo: 'ບັນຊີໂຄສະນາ', th: 'บัญชีโฆษณา', en: 'Ad account' })" variant="outlined" density="comfortable" hide-details :loading="loadingAccounts" />
    <v-select :model-value="datePreset" :items="[{title:l({lo:'7 ມື້ຫຼ້າສຸດ',th:'7 วันล่าสุด',en:'Last 7 days'}),value:'7'},{title:l({lo:'30 ມື້ຫຼ້າສຸດ',th:'30 วันล่าสุด',en:'Last 30 days'}),value:'30'},{title:l({lo:'ກຳນົດເອງ',th:'กำหนดเอง',en:'Custom'}),value:'custom'}]" item-title="title" item-value="value" :label="l({lo:'ໄລຍະເວລາ',th:'ช่วงเวลา',en:'Date range'})" variant="outlined" density="comfortable" hide-details @update:model-value="setPreset" />
    <v-text-field v-model="since" type="date" label="From" variant="outlined" density="comfortable" hide-details @update:model-value="datePreset='custom'" />
    <v-text-field v-model="until" type="date" label="To" variant="outlined" density="comfortable" hide-details @update:model-value="datePreset='custom'" />
    <v-btn color="primary" variant="tonal" height="48" :loading="loadingReport" @click="loadPerformance">{{ l({lo:'ນຳໃຊ້',th:'ใช้ช่วงเวลา',en:'Apply'}) }}</v-btn>
  </section>

  <div v-if="loadingReport && !report" class="loading-grid" aria-live="polite"><v-skeleton-loader v-for="n in 6" :key="n" type="article" /></div>

  <template v-else-if="report">
    <section class="summary-grid" aria-label="Ads summary">
      <article class="summary-card"><span>{{ l({lo:'ໃຊ້ຈ່າຍ',th:'ยอดใช้จ่าย',en:'Amount spent'}) }}</span><strong>{{ money(summary?.spend) }}</strong><small>{{ report.dateStart }} – {{ report.dateStop }}</small></article>
      <article class="summary-card"><span>{{ resultLabel(summary?.results.type) }}</span><strong>{{ fmt(summary?.results.value, 0) }}</strong><small>{{ summary?.results.type || l({lo:'Meta ບໍ່ມີຜົນລັບ',th:'Meta ไม่มีผลลัพธ์',en:'No result type returned'}) }}</small></article>
      <article class="summary-card"><span>{{ l({lo:'ຕົ້ນທຶນ/ຜົນລັບ',th:'ต้นทุน/ผลลัพธ์',en:'Cost per result'}) }}</span><strong>{{ money(summary?.results.cost) }}</strong><small>{{ targetCost ? `${l({lo:'ເປົ້າ',th:'เป้า',en:'Target'})}: ${money(targetCost)}` : l({lo:'ກຳນົດຈຸດຄຸ້ມທຶນເພື່ອຮັບຄຳແນະນຳ',th:'กำหนดจุดคุ้มทุนเพื่อรับคำแนะนำ',en:'Set break-even cost for guidance'}) }}</small></article>
      <article class="summary-card"><span>Link CTR</span><strong>{{ percent(summary?.linkCtr) }}</strong><small>{{ l({lo:'ອັດຕາຄລິກຕໍ່ການສະແດງ',th:'อัตราคลิกต่อการแสดงผล',en:'Clicks divided by impressions'}) }}</small></article>
      <article class="summary-card"><span>Frequency</span><strong>{{ fmt(summary?.frequency, 2) }}</strong><small>{{ l({lo:'ຈຳນວນຄັ້ງສະເລ່ຍທີ່ 1 ຄົນເຫັນ',th:'จำนวนครั้งเฉลี่ยที่ 1 คนเห็น',en:'Average times each person saw an ad'}) }}</small></article>
      <article class="summary-card"><span>{{ l({lo:'ເຂົ້າເຖິງ / ສະແດງ',th:'เข้าถึง / แสดงผล',en:'Reach / impressions'}) }}</span><strong>{{ fmt(summary?.reach, 0) }} / {{ fmt(summary?.impressions, 0) }}</strong><small>{{ report.timezone || 'Meta account timezone' }}</small></article>
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
      </v-card>
    </section>

    <section class="performance-section">
      <div class="section-head"><div><p>PERFORMANCE DETAIL</p><h2>{{ l({lo:'ເຈາະເລິກເຖິງລະດັບ Ad',th:'เจาะลึกถึงระดับโฆษณา',en:'Drill down to individual ads'}) }}</h2></div><div class="report-meta"><span>{{ l({lo:'ອັບເດດ',th:'อัปเดต',en:'Updated'}) }} {{ new Date(report.generatedAt).toLocaleString() }}</span><v-text-field v-model="search" density="compact" variant="outlined" prepend-inner-icon="mdi-magnify" :placeholder="l({lo:'ຄົ້ນຫາ...',th:'ค้นหา...',en:'Search...'})" hide-details clearable /></div></div>
      <v-btn-toggle v-model="level" mandatory color="primary" variant="outlined" class="level-tabs">
        <v-btn value="campaign">Campaign</v-btn><v-btn value="adset">Ad Set</v-btn><v-btn value="ad">Ad</v-btn>
      </v-btn-toggle>

      <div v-if="loadingReport" class="table-loading"><v-progress-linear indeterminate color="primary" /></div>
      <div v-else-if="!items.length" class="empty-state"><v-icon icon="mdi-chart-box-outline" size="44" /><h3>{{ l({lo:'ບໍ່ມີຂໍ້ມູນໃນໄລຍະນີ້',th:'ไม่มีข้อมูลในช่วงนี้',en:'No performance data for this period'}) }}</h3><p>{{ l({lo:'ປ່ຽນໄລຍະເວລາ ຫຼືກວດວ່າໂຄສະນາເຄີຍສົ່ງແລ້ວ.',th:'เปลี่ยนช่วงเวลาหรือตรวจว่าโฆษณาเคยส่งแล้ว',en:'Change the date range or confirm that ads delivered.'}) }}</p></div>

      <div v-else class="performance-list">
        <article v-for="item in items" :key="item.entityId" class="performance-row" :class="{ expanded: selectedId === item.entityId }">
          <button class="row-main" type="button" :aria-expanded="selectedId === item.entityId" @click="selectedId = selectedId === item.entityId ? '' : item.entityId">
            <span class="entity-cell"><span class="entity-icon"><v-icon :icon="item.entityType==='campaign'?'mdi-bullhorn-outline':item.entityType==='adset'?'mdi-account-group-outline':'mdi-advertisements'" /></span><span><strong>{{ item.entityName || item.entityId }}</strong><small v-if="item.entityType!=='campaign'">{{ item.campaignName }}<template v-if="item.entityType==='ad'"> · {{ item.adSetName }}</template></small><code>{{ item.entityId }}</code></span></span>
            <span class="metric"><small>{{ l({lo:'ໃຊ້ຈ່າຍ',th:'ใช้จ่าย',en:'Spend'}) }}</small><strong>{{ money(item.spend) }}</strong></span>
            <span class="metric"><small>{{ resultLabel(item.results.type) }}</small><strong>{{ fmt(item.results.value, 0) }}</strong></span>
            <span class="metric"><small>{{ l({lo:'ຕົ້ນທຶນ/ຜົນ',th:'ต้นทุน/ผล',en:'Cost/result'}) }}</small><strong>{{ money(item.results.cost) }}</strong></span>
            <span class="metric"><small>CTR</small><strong>{{ percent(item.linkCtr) }}</strong></span>
            <span class="metric"><small>Frequency</small><strong>{{ fmt(item.frequency, 2) }}</strong></span>
            <span class="decision-chip" :class="decision(item).code"><v-icon :icon="decision(item).icon" /> <span><strong>{{ decision(item).title }}</strong><small>{{ decision(item).signals[0] || decision(item).detail }}</small></span></span>
            <v-icon :icon="selectedId === item.entityId ? 'mdi-chevron-up' : 'mdi-chevron-down'" />
          </button>
          <div v-if="selectedId === item.entityId" class="row-detail">
            <div class="explanation"><v-icon :icon="decision(item).icon" :color="decision(item).color" /><div><h3>{{ decision(item).title }}</h3><p>{{ decision(item).detail }}</p><ul v-if="decision(item).signals.length"><li v-for="signal in decision(item).signals" :key="signal">{{ signal }}</li></ul></div></div>
            <div class="metric-detail-grid">
              <div><span>{{ l({lo:'ການສະແດງ',th:'การแสดงผล',en:'Impressions'}) }}</span><strong>{{ fmt(item.impressions, 0) }}</strong><small>{{ l({lo:'ຈຳນວນຄັ້ງທີ່ໂຄສະນາສະແດງ',th:'จำนวนครั้งที่โฆษณาแสดง',en:'Total ad displays'}) }}</small></div>
              <div><span>{{ l({lo:'ການເຂົ້າເຖິງ',th:'การเข้าถึง',en:'Reach'}) }}</span><strong>{{ fmt(item.reach, 0) }}</strong><small>{{ l({lo:'ຈຳນວນຄົນທີ່ເຫັນ',th:'จำนวนคนที่เห็น',en:'People who saw it'}) }}</small></div>
              <div><span>Link clicks</span><strong>{{ fmt(item.linkClicks, 0) }}</strong><small>{{ l({lo:'ຄລິກໄປປາຍທາງ',th:'คลิกไปยังปลายทาง',en:'Clicks to destination'}) }}</small></div>
              <div><span>CPC</span><strong>{{ money(item.linkCpc) }}</strong><small>{{ l({lo:'ຕົ້ນທຶນຕໍ່ຄລິກ',th:'ต้นทุนต่อคลิก',en:'Cost per link click'}) }}</small></div>
              <div><span>CPM</span><strong>{{ money(item.cpm) }}</strong><small>{{ l({lo:'ຕົ້ນທຶນຕໍ່ 1,000 ການສະແດງ',th:'ต้นทุนต่อ 1,000 การแสดงผล',en:'Cost per 1,000 impressions'}) }}</small></div>
              <div v-if="item.entityType==='ad'"><span>Quality ranking</span><strong>{{ item.qualityRanking || '—' }}</strong><small>{{ item.engagementRateRanking || '—' }} · {{ item.conversionRateRanking || '—' }}</small></div>
            </div>
            <div class="status-actions"><div><strong>{{ l({lo:'ສະຖານະການສົ່ງ',th:'สถานะการส่ง',en:'Delivery status'}) }}</strong><span>{{ statusFor(item) || l({lo:'ເລືອກຄຳສັ່ງທີ່ຕ້ອງການ',th:'เลือกคำสั่งที่ต้องการ',en:'Choose the desired status'}) }}</span></div><v-btn color="success" variant="tonal" prepend-icon="mdi-play" :loading="updatingId===item.entityId" :disabled="statusFor(item)==='ACTIVE'" @click="changeStatus(item,'ACTIVE')">{{ l({lo:'ເປີດໃຊ້ງານ',th:'เปิดใช้งาน',en:'Activate'}) }}</v-btn><v-btn color="error" variant="outlined" prepend-icon="mdi-pause" :loading="updatingId===item.entityId" :disabled="statusFor(item)==='PAUSED'" @click="changeStatus(item,'PAUSED')">{{ l({lo:'ປິດຊົ່ວຄາວ',th:'หยุดชั่วคราว',en:'Pause'}) }}</v-btn></div>
          </div>
        </article>
      </div>
    </section>
  </template>

  <v-card v-else-if="!loadingAccounts" class="empty-state" variant="outlined"><v-icon icon="mdi-facebook" size="48" /><h2>{{ l({lo:'ຍັງບໍ່ມີບັນຊີໂຄສະນາ',th:'ยังไม่มีบัญชีโฆษณา',en:'No ad account connected'}) }}</h2><p>{{ l({lo:'ເຊື່ອມ Facebook ແລະອະນຸຍາດ ads_read ເພື່ອເລີ່ມຕົ້ນ.',th:'เชื่อม Facebook และอนุญาต ads_read เพื่อเริ่มต้น',en:'Connect Facebook and allow ads_read to get started.'}) }}</p><v-btn color="primary" to="/meta-pages">{{ l({lo:'ເຊື່ອມຕໍ່ Meta',th:'เชื่อมต่อ Meta',en:'Connect Meta'}) }}</v-btn></v-card>

  <v-dialog v-model="settingsOpen" max-width="620">
    <v-card rounded="xl"><v-card-title class="dialog-title"><div><small>DECISION SETTINGS</small><h2>{{ l({lo:'ກຳນົດເກນຕາມຮ້ານ',th:'กำหนดเกณฑ์ตามร้าน',en:'Store-specific thresholds'}) }}</h2></div><v-btn icon="mdi-close" variant="text" @click="settingsOpen=false" /></v-card-title><v-card-text class="settings-body"><v-alert type="info" variant="tonal" density="compact" class="mb-4">{{ l({lo:'ແຕ່ລະຮ້ານມີກຳໄລ ແລະເປົ້າໝາຍບໍ່ຄືກັນ. ລະບົບຈະບໍ່ hard-code ຄຳຕັດສິນ.',th:'แต่ละร้านมีกำไรและเป้าหมายต่างกัน ระบบจะไม่ตัดสินด้วยค่าตายตัว',en:'Every store has different margins and goals; recommendations are not hard-coded.'}) }}</v-alert><v-text-field v-model.number="targetCost" type="number" min="0" step="0.01" :label="l({lo:'ຕົ້ນທຶນ/ຜົນລັບສູງສຸດ',th:'ต้นทุน/ผลลัพธ์สูงสุด',en:'Maximum cost per result'})" :suffix="currency" variant="outlined" hint="Use your real break-even cost" persistent-hint /><div class="settings-grid"><v-text-field v-model.number="weakCTR" type="number" min="0" step="0.1" label="Weak CTR below" suffix="%" variant="outlined" /><v-text-field v-model.number="goodCTR" type="number" min="0" step="0.1" label="Good CTR from" suffix="%" variant="outlined" /><v-text-field v-model.number="maxFrequency" type="number" min="1" step="0.1" label="Frequency warning from" variant="outlined" /></div></v-card-text><v-card-actions class="dialog-actions"><v-btn variant="text" @click="targetCost=null;goodCTR=3;weakCTR=1;maxFrequency=3">{{ l({lo:'ຄ່າເລີ່ມຕົ້ນ',th:'ค่าเริ่มต้น',en:'Reset'}) }}</v-btn><v-spacer /><v-btn color="primary" @click="saveThresholds">{{ l({lo:'ບັນທຶກ',th:'บันทึก',en:'Save settings'}) }}</v-btn></v-card-actions></v-card>
  </v-dialog>
</template>

<style scoped>
.page-head,.filter-bar,.performance-section{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
.page-head{padding:24px;display:flex;justify-content:space-between;align-items:flex-start;gap:24px;margin-bottom:16px}.page-head p,.card-heading p,.section-head p{margin:0 0 6px;color:var(--color-primary);font-size:11px;font-weight:800;letter-spacing:.13em}.page-head h1{font-size:clamp(24px,3vw,34px);line-height:1.15;margin:0 0 8px}.page-head span{color:var(--color-text-secondary)}.head-actions{display:flex;gap:8px;flex-wrap:wrap}
.filter-bar{display:grid;grid-template-columns:minmax(240px,1.5fr) minmax(170px,.8fr) minmax(145px,.7fr) minmax(145px,.7fr) auto;align-items:center;gap:12px;padding:16px;margin-bottom:16px}
.loading-grid,.summary-grid{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px;margin-bottom:16px}.summary-card{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);padding:18px;display:grid;gap:5px}.summary-card>span{color:var(--color-text-secondary);font-size:13px}.summary-card strong{font-size:clamp(22px,3vw,30px);line-height:1.1}.summary-card small{color:var(--color-text-muted);font-size:11px}
.analysis-grid{display:grid;grid-template-columns:minmax(0,1.6fr) minmax(320px,.8fr);gap:16px;margin-bottom:16px}.trend-card,.decision-card{padding:20px;border-radius:var(--radius-lg)}.card-heading,.section-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px}.card-heading h2,.section-head h2{margin:0;font-size:20px}.card-heading>small{color:var(--color-text-muted)}
.trend-chart{height:220px;display:flex;align-items:flex-end;gap:clamp(4px,1vw,12px);padding:28px 4px 24px;margin-top:12px;border-bottom:1px solid var(--color-border);position:relative}.trend-column{height:100%;flex:1;min-width:8px;display:flex;flex-direction:column;justify-content:flex-end;align-items:center;gap:7px;position:relative;outline:none}.trend-bar{width:min(34px,80%);min-height:4px;background:linear-gradient(180deg,var(--color-primary),#7c6cf1);border-radius:7px 7px 3px 3px;transition:transform .15s}.trend-column:is(:hover,:focus) .trend-bar{transform:scaleX(1.08)}.trend-column small{font-size:9px;color:var(--color-text-muted);white-space:nowrap}.trend-tooltip{display:none;position:absolute;bottom:calc(100% + 6px);z-index:2;background:#211d3f;color:#fff;padding:8px 10px;border-radius:8px;font-size:11px;white-space:nowrap;text-align:center}.trend-column:is(:hover,:focus) .trend-tooltip{display:block}
.decision-summary{display:grid;grid-template-columns:repeat(3,1fr);gap:8px;margin:18px 0}.decision-count{padding:14px 8px;border-radius:12px;display:grid;text-align:center}.decision-count strong{font-size:25px}.decision-count span{font-size:11px}.decision-count.keep{background:rgba(39,174,96,.1);color:var(--color-success)}.decision-count.review{background:rgba(245,158,11,.12);color:#a86700}.decision-count.pause{background:rgba(239,68,68,.1);color:var(--color-error)}.guide-note{display:flex;gap:8px;align-items:flex-start;color:var(--color-text-secondary);font-size:13px;line-height:1.55;margin:0}
.performance-section{padding:20px}.section-head{margin-bottom:14px}.report-meta{display:flex;align-items:center;gap:12px}.report-meta>span{font-size:11px;color:var(--color-text-muted)}.report-meta .v-text-field{width:250px}.level-tabs{margin-bottom:16px}.table-loading{margin:-1px 0 16px}.performance-list{display:grid;border:1px solid var(--color-border);border-radius:14px;overflow:hidden}.performance-row+.performance-row{border-top:1px solid var(--color-border)}.performance-row.expanded{background:var(--color-surface-soft)}.row-main{width:100%;display:grid;grid-template-columns:minmax(250px,1.5fr) repeat(5,minmax(88px,.55fr)) minmax(190px,1fr) 28px;align-items:center;gap:12px;padding:14px 16px;text-align:left;color:inherit;background:transparent;border:0;cursor:pointer}.row-main:hover{background:var(--color-background)}.entity-cell{display:flex;align-items:center;gap:11px;min-width:0}.entity-cell>span:last-child{display:grid;min-width:0}.entity-cell strong,.entity-cell small,.entity-cell code{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.entity-cell small{color:var(--color-text-secondary);font-size:11px}.entity-cell code{color:var(--color-text-muted);font-size:9px}.entity-icon{width:42px;height:42px;display:grid;place-items:center;border-radius:11px;background:var(--color-primary-soft);color:var(--color-primary);flex:0 0 42px}.metric{display:grid;gap:3px}.metric small{font-size:10px;color:var(--color-text-muted)}.metric strong{font-size:14px}.decision-chip{display:flex;align-items:center;gap:8px;border-radius:11px;padding:9px 10px;min-width:0}.decision-chip>span{display:grid;min-width:0}.decision-chip strong{font-size:12px}.decision-chip small{font-size:9px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.decision-chip.keep{background:rgba(39,174,96,.1);color:var(--color-success)}.decision-chip.review{background:rgba(245,158,11,.12);color:#a86700}.decision-chip.pause{background:rgba(239,68,68,.1);color:var(--color-error)}.decision-chip.no_data{background:var(--color-surface-soft);color:var(--color-text-secondary)}
.row-detail{padding:20px;border-top:1px solid var(--color-border);display:grid;gap:18px}.explanation{display:flex;gap:12px;align-items:flex-start;max-width:850px}.explanation h3{margin:0 0 4px;font-size:17px}.explanation p{margin:0;color:var(--color-text-secondary)}.explanation ul{margin:8px 0 0;padding-left:20px;color:var(--color-text-secondary)}.metric-detail-grid{display:grid;grid-template-columns:repeat(6,1fr);gap:10px}.metric-detail-grid>div{padding:12px;border:1px solid var(--color-border);border-radius:11px;display:grid;gap:3px}.metric-detail-grid span,.metric-detail-grid small{font-size:10px;color:var(--color-text-muted)}.metric-detail-grid strong{font-size:15px;overflow-wrap:anywhere}.status-actions{display:flex;align-items:center;justify-content:flex-end;gap:8px;padding-top:14px;border-top:1px solid var(--color-border)}.status-actions>div{display:grid;margin-right:auto}.status-actions>div span{font-size:12px;color:var(--color-text-secondary)}
.empty-state,.empty-inline{text-align:center;padding:48px 20px;color:var(--color-text-secondary)}.empty-state h2,.empty-state h3{color:var(--color-text);margin:12px 0 6px}.empty-state p{margin:0 0 16px}.dialog-title{display:flex;justify-content:space-between;align-items:flex-start;padding:22px}.dialog-title small{color:var(--color-primary);font-size:10px;letter-spacing:.12em}.dialog-title h2{margin:3px 0 0;font-size:22px}.settings-body{padding:0 22px 12px}.settings-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:10px;margin-top:16px}.dialog-actions{padding:12px 22px 22px}
@media(max-width:1200px){.filter-bar{grid-template-columns:repeat(2,minmax(0,1fr))}.analysis-grid{grid-template-columns:1fr}.row-main{grid-template-columns:minmax(240px,1.4fr) repeat(3,minmax(85px,.55fr)) minmax(170px,1fr) 28px}.row-main>.metric:nth-of-type(4),.row-main>.metric:nth-of-type(5){display:none}.metric-detail-grid{grid-template-columns:repeat(3,1fr)}}
@media(max-width:720px){.page-head{padding:18px;display:grid}.head-actions,.head-actions .v-btn{width:100%}.filter-bar{grid-template-columns:1fr}.loading-grid,.summary-grid{grid-template-columns:repeat(2,1fr)}.performance-section{padding:14px}.section-head,.report-meta{display:grid}.report-meta .v-text-field{width:100%}.level-tabs{width:100%;display:grid;grid-template-columns:repeat(3,1fr)}.level-tabs .v-btn{min-width:0}.row-main{grid-template-columns:1fr auto;padding:14px}.row-main>.metric{display:none}.decision-chip{grid-column:1/2}.row-main>i{grid-column:2;grid-row:1/3}.metric-detail-grid{grid-template-columns:repeat(2,1fr)}.status-actions{display:grid;grid-template-columns:1fr 1fr}.status-actions>div{grid-column:1/-1}.settings-grid{grid-template-columns:1fr}.trend-card,.decision-card{padding:16px}}
@media(max-width:420px){.summary-grid{grid-template-columns:1fr}.metric-detail-grid{grid-template-columns:1fr}.status-actions{grid-template-columns:1fr}.status-actions>div{grid-column:auto}}
</style>
