<script setup lang="ts">
import { ref, computed, reactive, watchEffect, onMounted, onUnmounted } from 'vue'
import type { ItemsResponse, Product } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { can } = useAuth()
const { l, locale } = useLocale()
const router = useRouter()

if (!can('customers:read')) throw createError({ statusCode: 403 })

interface Lead {
  id: string
  accountId: string
  pageId: string
  senderId: string
  sourceType: string
  sourceId: string
  productId?: string
  name?: string
  status: string
  assignedUserId: string
  alertRead: boolean
  createdAt: string
}

interface Step {
  delayMinutes: number
  message: string
  isEnabled: boolean
}
interface CAPIHealth {
  state: 'disabled' | 'setup_required' | 'healthy' | 'warning' | 'failing' | 'no_events'
  enabled: boolean
  configured: boolean
  sent: number
  failed: number
  pending: number
  notConfigured: number
  successRate?: number
  lastEventAt?: string
  lastError?: string
}

interface StatusOption {
  key: string
  label: { lo: string; th: string; en: string }
  color: string
  bg: string
  border: string
  icon: string
}

const STATUS_OPTIONS: StatusOption[] = [
  {
    key: 'new',
    label: { lo: 'ໃໝ່', th: 'ลูกค้าใหม่', en: 'New' },
    color: '#2563eb',
    bg: '#eff6ff',
    border: '#bfdbfe',
    icon: 'mdi-account-plus',
  },
  {
    key: 'in_progress',
    label: { lo: 'ກຳລັງລົມ', th: 'กำลังคุย', en: 'Chatting' },
    color: '#7c3aed',
    bg: '#f5f3ff',
    border: '#ddd6fe',
    icon: 'mdi-chat-processing-outline',
  },
  {
    key: 'pending_payment',
    label: { lo: 'ລໍຖ້າໂອນ', th: 'รอโอนเงิน', en: 'Pending Payment' },
    color: '#ea580c',
    bg: '#fff7ed',
    border: '#fed7aa',
    icon: 'mdi-clock-outline',
  },
  {
    key: 'purchased',
    label: { lo: 'ປິດການຂາຍ', th: 'ซื้อแล้ว / ปิดการขาย', en: 'Purchased' },
    color: '#16a34a',
    bg: '#f0fdf4',
    border: '#bbf7d0',
    icon: 'mdi-check-circle',
  },
  {
    key: 'follow_up',
    label: { lo: 'ຕິດຕາມຕໍ່', th: 'ติดตามต่อ', en: 'Follow-up' },
    color: '#0891b2',
    bg: '#ecfeff',
    border: '#a5f3fc',
    icon: 'mdi-calendar-clock',
  },
  {
    key: 'cancelled',
    label: { lo: 'ຍົກເລີກ / ບໍ່ຊື້', th: 'ยกเลิก / ไม่ซื้อ', en: 'Cancelled' },
    color: '#64748b',
    bg: '#f8fafc',
    border: '#e2e8f0',
    icon: 'mdi-close-circle-outline',
  },
]

const getStatusInfo = (status?: string): StatusOption => {
  const norm = (status || 'new').toLowerCase()
  return STATUS_OPTIONS.find(s => s.key === norm) || {
    key: norm,
    label: { lo: norm, th: norm, en: norm },
    color: '#64748b',
    bg: '#f1f5f9',
    border: '#cbd5e1',
    icon: 'mdi-tag-outline',
  }
}

const { data, pending, error, refresh } = await useApi<{ items: Lead[] }>('/proxy/api/v1/leads')
const { data: productData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: config, refresh: refreshConfig } = await useApi<any>('/proxy/api/v1/leads/settings')
const { data: capiHealth, refresh: refreshCAPIHealth } = await useApi<CAPIHealth>('/proxy/api/v1/leads/capi/health')

const leads = computed(() => data.value?.items || [])
const productById = computed(() => new Map((productData.value?.items || []).map(product => [product.id, product])))

const settings = reactive({
  assignmentMode: 'round_robin',
  alertsEnabled: true,
  followUpEnabled: false,
  capiEnabled: false,
  capiDatasetId: '',
  capiToken: '',
})

const steps = ref<Step[]>([])
const saving = ref(false)
const notice = ref('')
const copiedId = ref<string | null>(null)
const settingsDialog = ref(false)
const updatingStatusId = ref<string | null>(null)

// Filtering & Search
const searchQuery = ref('')
const selectedStatus = ref('all')
const selectedSource = ref('all')

const filteredLeads = computed(() => {
  let list = leads.value
  if (selectedStatus.value !== 'all') {
    list = list.filter(l => (l.status || 'new').toLowerCase() === selectedStatus.value)
  }
  if (selectedSource.value !== 'all') {
    list = list.filter(l => l.sourceType === selectedSource.value)
  }
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(l => 
      (l.senderId && l.senderId.toLowerCase().includes(q)) ||
      (l.name && l.name.toLowerCase().includes(q)) ||
      (l.sourceId && l.sourceId.toLowerCase().includes(q)) ||
      (l.pageId && l.pageId.toLowerCase().includes(q))
    )
  }
  return list
})

const statusCounts = computed(() => {
  const counts: Record<string, number> = { all: leads.value.length }
  for (const opt of STATUS_OPTIONS) counts[opt.key] = 0
  for (const lead of leads.value) {
    const st = (lead.status || 'new').toLowerCase()
    counts[st] = (counts[st] ?? 0) + 1
  }
  return counts
})

watchEffect(() => {
  if (config.value?.settings) {
    const s = config.value.settings
    settings.assignmentMode = s.assignmentMode || 'round_robin'
    settings.alertsEnabled = s.alertsEnabled
    settings.followUpEnabled = s.followUpEnabled
    settings.capiEnabled = s.capiEnabled
    settings.capiDatasetId = s.capiDatasetId || ''
    settings.capiToken = config.value.capiConfigured ? '********' : ''
    steps.value = (config.value.steps || []).map((x: any) => ({
      delayMinutes: x.delayMinutes,
      message: x.message,
      isEnabled: x.isEnabled,
    }))
  }
})

function addStep() {
  steps.value.push({ delayMinutes: 60, message: '', isEnabled: true })
}

async function saveSettings() {
  saving.value = true
  notice.value = ''
  try {
    await $fetch('/api/proxy/api/v1/leads/settings', {
      method: 'PUT',
      body: { settings, steps: steps.value },
    })
    notice.value = l({ lo: 'ບັນທຶກການຕັ້ງຄ່າຮຽບຮ້ອຍແລ້ວ', th: 'บันทึกการตั้งค่าเรียบร้อยแล้ว', en: 'Settings saved.' })
    settingsDialog.value = false
    await refreshConfig()
    await refreshCAPIHealth()
    setTimeout(() => { notice.value = '' }, 4000)
  } finally {
    saving.value = false
  }
}

async function markLeadRead(lead: Lead) {
  if (lead.alertRead) return
  await $fetch(`/api/proxy/api/v1/leads/${lead.id}/read`, { method: 'PATCH' })
  await refresh()
}

async function updateLeadStatus(senderId: string, newStatus: string) {
  updatingStatusId.value = senderId
  try {
    await $fetch(`/api/proxy/api/v1/leads/${senderId}/status`, {
      method: 'PUT',
      body: { status: newStatus },
    })
    const target = leads.value.find(l => l.senderId === senderId)
    if (target) target.status = newStatus
    const info = getStatusInfo(newStatus)
    notice.value = l({
      lo: `ປ່ຽນສະຖານະລູກຄ້າ ${senderId.slice(-4)} ເປັນ “${l(info.label)}” ສຳເລັດ`,
      th: `เปลี่ยนสถานะลูกค้า ${senderId.slice(-4)} เป็น “${l(info.label)}” สำเร็จ`,
      en: `Lead ${senderId.slice(-4)} updated to “${l(info.label)}”`,
    })
    setTimeout(() => { notice.value = '' }, 4000)
    await refresh()
  } catch {
    notice.value = l({ lo: 'ອັບເດດສະຖານະບໍ່ສຳເລັດ', th: 'อัปเดตสถานะไม่สำเร็จ', en: 'Failed to update status' })
  } finally {
    updatingStatusId.value = null
  }
}

function copySenderId(senderId: string) {
  navigator.clipboard.writeText(senderId)
  copiedId.value = senderId
  setTimeout(() => { copiedId.value = null }, 2000)
}

function openLiveChat(senderId: string) {
  router.push(`/live-chat?senderId=${senderId}`)
}

let evtSource: EventSource | null = null
onMounted(() => {
  evtSource = new EventSource('/api/proxy/api/v1/chat/stream')
  evtSource.onmessage = (e) => {
    try {
      const parsed = JSON.parse(e.data)
      if (parsed.type === 'lead_alert') {
        notice.value = l({ lo: '🎉 ມີ Lead ໃໝ່ເຂົ້າມາທັນທີ!', th: '🎉 มี Lead ใหม่เข้ามาทันที!', en: '🎉 New lead incoming!' })
        void refresh()
      }
    } catch {}
  }
})

onUnmounted(() => {
  if (evtSource) evtSource.close()
})
</script>

<template>
  <div class="leads-page">
    <!-- Header Hero Banner -->
    <section class="leads-hero">
      <div class="hero-text">
        <small class="hero-tag">PANCAKE CRM & SALES PIPELINE</small>
        <h2>{{ l({ lo: 'ຈັດການລູກຄ້າ & Leads', th: 'จัดการลูกค้า & Leads', en: 'Lead & Customer Management' }) }}</h2>
        <p>{{ l({ lo: 'ຕິດຕາມລູກຄ້າທຸກຊ່ອງທາງ, ປິດການຂາຍ, ແບ່ງງານໃນທີມ ແລະ ສົ່ງ Meta Conversions API (CAPI) ໃນບ່ອນດຽວ', th: 'ติดตามลูกค้าทุกช่องทาง ปิดการขาย มอบหมายทีม และส่ง Meta Conversions API (CAPI) ครบจบในที่เดียว', en: 'Track customers, close deals, assign team members and report Meta Conversions API in one place.' }) }}</p>
      </div>
      <div class="hero-actions">
        <v-btn prepend-icon="mdi-chat-processing-outline" color="primary" class="hero-btn" to="/live-chat">
          {{ l({ lo: 'ໄປທີ່ Live Chat', th: 'ไปที่ Live Chat', en: 'Open Live Chat' }) }}
        </v-btn>
        <v-btn prepend-icon="mdi-cog-outline" variant="outlined" class="hero-btn" @click="settingsDialog = true">
          {{ l({ lo: 'ຕັ້ງຄ່າ Automation / CAPI', th: 'ตั้งค่า Automation / CAPI', en: 'Settings & CAPI' }) }}
        </v-btn>
        <v-btn icon="mdi-refresh" variant="text" :loading="pending" @click="refresh()" :title="l({lo:'ໂຫຼດໃໝ່',th:'รีเฟรช',en:'Refresh'})"/>
      </div>
    </section>

    <v-alert v-if="notice" type="success" variant="tonal" closable class="mb-4" @click:close="notice = ''">
      {{ notice }}
    </v-alert>

    <section v-if="capiHealth" class="capi-health" :class="capiHealth.state">
      <div class="capi-health-icon"><v-icon :icon="capiHealth.state==='healthy'?'mdi-check-decagram-outline':capiHealth.state==='failing'?'mdi-alert-octagon-outline':'mdi-access-point-network'" /></div>
      <div class="capi-health-copy"><small>META CONVERSIONS API · 7 DAYS</small><strong>{{ l(capiHealth.state==='healthy'?{lo:'CAPI ເຮັດວຽກປົກກະຕິ',th:'CAPI ทำงานปกติ',en:'CAPI is healthy'}:capiHealth.state==='disabled'?{lo:'CAPI ຍັງບໍ່ເປີດ',th:'CAPI ยังไม่เปิด',en:'CAPI is disabled'}:capiHealth.state==='setup_required'?{lo:'CAPI ຕ້ອງຕັ້ງຄ່າ',th:'ต้องตั้งค่า CAPI',en:'CAPI setup required'}:capiHealth.state==='no_events'?{lo:'ຕັ້ງຄ່າແລ້ວ—ຍັງບໍ່ມີ Event',th:'ตั้งค่าแล้ว—ยังไม่มี Event',en:'Configured—no events yet'}:{lo:'CAPI ຕ້ອງກວດສອບ',th:'CAPI ต้องตรวจสอบ',en:'CAPI needs attention'}) }}</strong><span v-if="capiHealth.lastError">{{ capiHealth.lastError }}</span><span v-else>{{ l({lo:'ຕິດຕາມ Lead/Purchase event ທີ່ສົ່ງໃຫ້ Meta.',th:'ติดตาม Lead/Purchase event ที่ส่งให้ Meta',en:'Monitor Lead and Purchase events sent to Meta.'}) }}</span></div>
      <div class="capi-health-metrics"><div><small>{{ l({lo:'ສົ່ງສຳເລັດ',th:'ส่งสำเร็จ',en:'Sent'}) }}</small><strong>{{ capiHealth.sent }}</strong></div><div><small>{{ l({lo:'ບໍ່ສຳເລັດ',th:'ล้มเหลว',en:'Failed'}) }}</small><strong>{{ capiHealth.failed }}</strong></div><div><small>Success</small><strong>{{ capiHealth.successRate==null?'—':`${capiHealth.successRate.toFixed(1)}%` }}</strong></div></div>
      <v-btn size="small" variant="outlined" @click="settingsDialog=true">{{ l({lo:'ຕັ້ງຄ່າ',th:'ตั้งค่า',en:'Configure'}) }}</v-btn>
    </section>

    <!-- KPI Metric Cards Grid -->
    <section class="kpi-grid">
      <div class="kpi-card" @click="selectedStatus = 'all'">
        <div class="kpi-icon all"><v-icon icon="mdi-account-group" size="24"/></div>
        <div class="kpi-content">
          <small>{{ l({ lo: 'Leads ທັງໝົດ', th: 'Leads ทั้งหมด', en: 'Total Leads' }) }}</small>
          <strong>{{ statusCounts.all || 0 }}</strong>
        </div>
      </div>

      <div class="kpi-card" @click="selectedStatus = 'new'">
        <div class="kpi-icon new"><v-icon icon="mdi-account-plus" size="24"/></div>
        <div class="kpi-content">
          <small>{{ l({ lo: 'ລູກຄ້າໃໝ່', th: 'ลูกค้าใหม่', en: 'New Leads' }) }}</small>
          <strong style="color: #2563eb;">{{ statusCounts.new || 0 }}</strong>
        </div>
      </div>

      <div class="kpi-card" @click="selectedStatus = 'in_progress'">
        <div class="kpi-icon in_progress"><v-icon icon="mdi-chat-processing-outline" size="24"/></div>
        <div class="kpi-content">
          <small>{{ l({ lo: 'ກຳລັງລົມ', th: 'กำลังคุย', en: 'Chatting' }) }}</small>
          <strong style="color: #7c3aed;">{{ statusCounts.in_progress || 0 }}</strong>
        </div>
      </div>

      <div class="kpi-card" @click="selectedStatus = 'pending_payment'">
        <div class="kpi-icon pending_payment"><v-icon icon="mdi-clock-outline" size="24"/></div>
        <div class="kpi-content">
          <small>{{ l({ lo: 'ລໍຖ້າໂອນ', th: 'รอโอนเงิน', en: 'Pending Payment' }) }}</small>
          <strong style="color: #ea580c;">{{ statusCounts.pending_payment || 0 }}</strong>
        </div>
      </div>

      <div class="kpi-card highlight-purchased" @click="selectedStatus = 'purchased'">
        <div class="kpi-icon purchased"><v-icon icon="mdi-check-circle" size="24"/></div>
        <div class="kpi-content">
          <small>{{ l({ lo: 'ປິດການຂາຍແລ້ວ (CAPI)', th: 'ปิดการขายแล้ว (CAPI)', en: 'Purchased (CAPI)' }) }}</small>
          <strong style="color: #16a34a;">{{ statusCounts.purchased || 0 }}</strong>
        </div>
      </div>
    </section>

    <!-- Main Content Panel -->
    <section class="leads-main-panel">
      <!-- Search & Filter Controls -->
      <div class="filter-bar">
        <div class="search-box">
          <v-icon icon="mdi-magnify" size="18" class="search-icon"/>
          <input 
            v-model="searchQuery" 
            type="text" 
            :placeholder="l({ lo: 'ຄົ້ນຫາ Sender ID, ຊື່, ລະຫັດ Page, Ad ID...', th: 'ค้นหา Sender ID, ชื่อ, รหัส Page, Ad ID...', en: 'Search Sender ID, name, Page ID, Ad ID...' })"
          />
          <button v-if="searchQuery" class="clear-btn" @click="searchQuery = ''"><v-icon icon="mdi-close" size="14"/></button>
        </div>

        <!-- Horizontal Status Filter Tabs -->
        <div class="status-tabs">
          <button 
            type="button" 
            class="tab-btn" 
            :class="{ active: selectedStatus === 'all' }" 
            @click="selectedStatus = 'all'"
          >
            {{ l({ lo: 'ທັງໝົດ', th: 'ทั้งหมด', en: 'All' }) }}
            <span class="count-badge">{{ statusCounts.all || 0 }}</span>
          </button>

          <button 
            v-for="opt in STATUS_OPTIONS" 
            :key="opt.key"
            type="button" 
            class="tab-btn"
            :class="{ active: selectedStatus === opt.key }"
            :style="selectedStatus === opt.key ? { backgroundColor: opt.bg, borderColor: opt.color, color: opt.color } : {}"
            @click="selectedStatus = opt.key"
          >
            <span class="status-dot" :style="{ backgroundColor: opt.color }"></span>
            {{ l(opt.label) }}
            <span class="count-badge" v-if="statusCounts[opt.key]">{{ statusCounts[opt.key] }}</span>
          </button>
        </div>
      </div>

      <!-- Lead Cards / List -->
      <v-progress-linear v-if="pending" indeterminate class="my-4"/>
      <v-alert v-else-if="error" type="error" variant="tonal" class="my-4">
        {{ l({ lo: 'ໂຫຼດ Lead ບໍ່ສຳເລັດ', th: 'โหลด Lead ไม่สำเร็จ', en: 'Unable to load leads.' }) }}
      </v-alert>

      <div v-else-if="filteredLeads.length" class="lead-cards-list">
        <article 
          v-for="lead in filteredLeads" 
          :key="lead.id" 
          class="lead-card"
          :class="{ unread: !lead.alertRead }"
          @click="markLeadRead(lead)"
        >
          <div class="lead-main-col">
            <div class="lead-avatar">
              <v-icon icon="mdi-account" size="24"/>
            </div>

            <div class="lead-info">
              <div class="lead-name-row">
                <strong>{{ lead.name || `${l({lo:'ລູກຄ້າ',th:'ลูกค้า',en:'Customer'})} · ${lead.senderId.slice(-4)}` }}</strong>
                <span class="id-tag">
                  ID: {{ lead.senderId }}
                  <button 
                    type="button" 
                    class="copy-id-btn" 
                    :title="l({lo:'ຄັດລອກ ID',th:'คัดลอก ID',en:'Copy ID'})"
                    @click.stop="copySenderId(lead.senderId)"
                  >
                    <v-icon :icon="copiedId === lead.senderId ? 'mdi-check' : 'mdi-content-copy'" size="13"/>
                  </button>
                </span>
                <span v-if="!lead.alertRead" class="unread-dot" :title="l({lo:'ໃໝ່ຍັງບໍ່ໄດ້ອ່ານ',th:'ใหม่ยังไม่ได้อ่าน',en:'New Unread'})"/>
              </div>

              <div class="lead-tags-row">
                <span class="meta-pill page"><v-icon icon="mdi-flag" size="12"/> {{ lead.pageId }}</span>
                <span v-if="lead.sourceId" class="meta-pill ad"><v-icon icon="mdi-bullhorn" size="12"/> Ad: {{ lead.sourceId }}</span>
                <span v-if="lead.productId && productById.get(lead.productId)" class="meta-pill product"><v-icon icon="mdi-package-variant-closed" size="12"/> {{ productById.get(lead.productId)?.name }} · {{ productById.get(lead.productId)?.price }}</span>
                <span class="meta-pill time"><v-icon icon="mdi-calendar-clock" size="12"/> {{ new Date(lead.createdAt).toLocaleString([], { dateStyle: 'short', timeStyle: 'short' }) }}</span>
              </div>
            </div>
          </div>

          <!-- Quick Actions & Status Control -->
          <div class="lead-actions-col" @click.stop>
            <!-- Pancake Status Dropdown -->
            <v-menu location="bottom end">
              <template #activator="{ props }">
                <button 
                  type="button" 
                  v-bind="props" 
                  class="lead-status-btn"
                  :style="{
                    color: getStatusInfo(lead.status).color,
                    backgroundColor: getStatusInfo(lead.status).bg,
                    borderColor: getStatusInfo(lead.status).border
                  }"
                  :disabled="updatingStatusId === lead.senderId"
                >
                  <v-icon :icon="getStatusInfo(lead.status).icon" size="15" class="mr-1"/>
                  <span>{{ l(getStatusInfo(lead.status).label) }}</span>
                  <v-icon icon="mdi-chevron-down" size="15" class="ml-1"/>
                </button>
              </template>
              <v-list density="compact" class="status-menu-popup">
                <v-list-subheader>{{ l({lo:'ປ່ຽນສະຖານະລູກຄ້າ',th:'เปลี่ยนสถานะลูกค้า',en:'Change Status'}) }}</v-list-subheader>
                <v-list-item
                  v-for="opt in STATUS_OPTIONS"
                  :key="opt.key"
                  class="status-menu-item"
                  :class="{ active: (lead.status || 'new') === opt.key }"
                  @click="updateLeadStatus(lead.senderId, opt.key)"
                >
                  <template #prepend>
                    <span class="menu-dot" :style="{ backgroundColor: opt.color }"></span>
                  </template>
                  <v-list-item-title :style="{ color: opt.color, fontWeight: (lead.status || 'new') === opt.key ? 750 : 500 }">
                    {{ l(opt.label) }}
                  </v-list-item-title>
                  <template #append v-if="(lead.status || 'new') === opt.key">
                    <v-icon icon="mdi-check" size="16" :color="opt.color"/>
                  </template>
                </v-list-item>
              </v-list>
            </v-menu>

            <!-- 1-Click Purchase Shortcut (Meta CAPI) -->
            <button 
              v-if="(lead.status || 'new') !== 'purchased'" 
              type="button"
              class="quick-buy-btn"
              @click="updateLeadStatus(lead.senderId, 'purchased')"
              :title="l({lo:'ປິດການຂາຍທັນທີ (ສົ່ງ Meta CAPI)',th:'ปิดการขายทันที (ส่ง Meta CAPI)',en:'Mark Purchased (Meta CAPI)'})"
            >
              <v-icon icon="mdi-lightning-bolt" size="14" class="mr-1"/>
              <span>{{ locale === 'lo' ? 'ປິດການຂາຍ' : (locale === 'th' ? 'ปิดการขาย' : 'Close Deal') }}</span>
            </button>
            <span v-else class="purchased-badge">
              <v-icon icon="mdi-check-circle" size="14" class="mr-1"/>
              {{ locale === 'lo' ? 'ປິດການຂາຍແລ້ວ' : (locale === 'th' ? 'ซื้อแล้ว' : 'Purchased') }}
            </span>

            <!-- Direct Live Chat Jump -->
            <button 
              type="button" 
              class="open-chat-btn"
              @click="openLiveChat(lead.senderId)"
              :title="l({lo:'ເປີດແຊັດສົນທະນາກັບລູກຄ້າ',th:'เปิดแชทสนทนากับลูกค้า',en:'Open Chat with Lead'})"
            >
              <v-icon icon="mdi-message-text-outline" size="16"/>
              <span>{{ l({lo:'ແຊັດເລີຍ',th:'แชทเลย',en:'Chat'}) }}</span>
            </button>
          </div>
        </article>
      </div>

      <div v-else class="empty-leads">
        <v-icon icon="mdi-account-search-outline" size="52" color="grey"/>
        <h3>{{ l({lo:'ບໍ່ພົບ Lead ທີ່ກົງກັບເງື່ອນໄຂ',th:'ไม่พบ Lead ที่ตรงกับเงื่อนไข',en:'No leads matching filters.'}) }}</h3>
        <p>{{ l({lo:'ເມື່ອມີລູກຄ້າທັກແຊັດ ຫຼື ຄອມເມັ້ນຈາກໂຄສະນາ/Page ລະບົບຈະບັນທຶກ Lead ຢູ່ນີ້ອັດຕະໂນມັດ',th:'เมื่อมีลูกค้าทักแชทหรือคอมเมนต์จากโฆษณา/เพจ ระบบจะบันทึก Lead ที่นี่อัตโนมัติ',en:'New incoming messages or comments will automatically appear as leads here.'}) }}</p>
        <v-btn color="primary" prepend-icon="mdi-chat" to="/live-chat" class="mt-2">
          {{ l({lo:'ໄປທີ່ Live Chat',th:'ไปที่ Live Chat',en:'Open Live Chat'}) }}
        </v-btn>
      </div>
    </section>

    <!-- Automation & CAPI Settings Dialog -->
    <v-dialog v-model="settingsDialog" max-width="640">
      <v-card class="settings-dialog-card">
        <v-card-title class="dialog-header">
          <v-icon icon="mdi-robot-outline" color="primary" class="mr-2"/>
          <span>{{ l({lo:'ຕັ້ງຄ່າ Lead Automation & Meta CAPI',th:'ตั้งค่า Lead Automation & Meta CAPI',en:'Lead Automation & Meta CAPI Settings'}) }}</span>
        </v-card-title>
        <v-card-text class="dialog-content">
          <v-select 
            v-model="settings.assignmentMode" 
            :items="[
              { title: l({lo:'ແບ່ງໃຫ້ທີມງານຕາມລຳດັບ (Round-Robin)',th:'แบ่งให้ทีมตามลำดับ (Round-Robin)',en:'Round-robin team assignment'}), value: 'round_robin' },
              { title: l({lo:'ບໍ່ຈັດຄົນອັດຕະໂນມັດ',th:'ไม่มอบหมายอัตโนมัติ',en:'No automatic assignment'}), value: 'none' }
            ]" 
            item-title="title" 
            item-value="value" 
            :label="l({lo:'ຮູບແບບການແບ່ງ Lead',th:'รูปแบบการมอบหมาย Lead',en:'Lead Assignment Mode'})"
            variant="outlined"
            density="compact"
            class="mb-3"
          />

          <div class="toggle-row">
            <v-switch v-model="settings.alertsEnabled" color="primary" hide-details :label="l({lo:'ແຈ້ງເຕືອນ Real-Time ເມື່ອມີ Lead ໃໝ່',th:'แจ้งเตือน Real-Time เมื่อมี Lead ใหม่',en:'Instant new-lead real-time alert'})"/>
          </div>

          <div class="toggle-row">
            <v-switch v-model="settings.capiEnabled" color="primary" hide-details :label="l({lo:'ສົ່ງ Lead & Purchase ໄປ Meta Conversions API (CAPI)',th:'ส่ง Lead & Purchase ไป Meta Conversions API (CAPI)',en:'Send Leads & Purchases to Meta CAPI'})"/>
          </div>

          <!-- CAPI Configuration Section -->
          <div v-if="settings.capiEnabled" class="capi-config-box my-3">
            <v-alert type="info" variant="tonal" density="compact" class="mb-3 text-caption">
              <strong>{{ l({lo:'ວິທີເອົາ Token ຈາກ Facebook Events Manager:',th:'วิธีรับ Token จาก Facebook Events Manager:',en:'How to get your Token:'}) }}</strong>
              <ol class="pl-4 mt-1">
                <li>{{ l({lo:'ໄປທີ່',th:'ไปที่',en:'Go to'}) }} <a href="https://business.facebook.com/events_manager2" target="_blank" class="text-primary font-weight-bold">Meta Events Manager <v-icon icon="mdi-open-in-new" size="12"/></a></li>
                <li>{{ l({lo:'ເລືອກ Data Source (Pixel ID) ທີ່ຕ້ອງການ',th:'เลือก Data Source (Pixel ID) ที่ต้องการ',en:'Select your Data Source / Pixel'}) }}</li>
                <li>{{ l({lo:'ໄປແທັບ Settings > ເລື່ອນລົງຫາ Conversions API',th:'ไปแท็บ Settings > เลื่อนลงหา Conversions API',en:'Go to Settings > Conversions API'}) }}</li>
                <li>{{ l({lo:'ກົດປຸ່ມ "Generate access token"',th:'กดปุ่ม "Generate access token"',en:'Click "Generate access token"'}) }}</li>
              </ol>
            </v-alert>
            <v-text-field 
              v-model="settings.capiDatasetId" 
              label="Dataset ID (Pixel ID)" 
              placeholder="e.g. 100376625971155" 
              variant="outlined" 
              density="compact"
              class="mb-2"
            />
            <v-text-field 
              v-model="settings.capiToken" 
              label="Meta Access Token (EAAB...)" 
              placeholder="Paste CAPI Access Token here" 
              type="password" 
              variant="outlined"
              density="compact"
            />
          </div>

          <div class="toggle-row">
            <v-switch v-model="settings.followUpEnabled" color="primary" hide-details :label="l({lo:'ເປີດ Follow-up Sequence (ຕິດຕາມລູກຄ້າອັດຕະໂນມັດ)',th:'เปิด Follow-up Sequence (ติดตามลูกค้าอัตโนมัติ)',en:'Enable automatic follow-up sequence'})"/>
          </div>

          <!-- Follow-up Steps Builder -->
          <div v-if="settings.followUpEnabled" class="followup-builder my-3">
            <div class="d-flex justify-space-between align-center mb-2">
              <strong>{{ l({lo:'ຂັ້ນຕອນຕິດຕາມ (Steps)',th:'ขั้นตอนติดตาม (Steps)',en:'Follow-up Steps'}) }}</strong>
              <v-btn size="small" variant="tonal" prepend-icon="mdi-plus" @click="addStep">
                {{ l({lo:'ເພີ່ມຂັ້ນຕອນ',th:'เพิ่มขั้นตอน',en:'Add step'}) }}
              </v-btn>
            </div>
            <div v-for="(step, i) in steps" :key="i" class="step-card mb-2">
              <div class="d-flex align-center gap-2 mb-2">
                <span class="step-badge">Step {{ i + 1 }}</span>
                <v-text-field 
                  v-model.number="step.delayMinutes" 
                  type="number" 
                  min="1" 
                  :label="l({lo:'ລໍຖ້າ (ນາທີ)',th:'รอ (นาที)',en:'Wait (mins)'})" 
                  density="compact" 
                  variant="outlined" 
                  hide-details
                  style="max-width: 140px;"
                />
                <v-btn icon="mdi-delete-outline" color="error" variant="text" size="small" @click="steps.splice(i, 1)"/>
              </div>
              <v-textarea 
                v-model="step.message" 
                rows="2" 
                :label="l({lo:'ຂໍ້ຄວາມຕິດຕາມ',th:'ข้อความติดตาม',en:'Follow-up message'})" 
                density="compact" 
                variant="outlined" 
                hide-details
              />
            </div>
          </div>
        </v-card-text>
        <v-card-actions class="dialog-actions">
          <v-btn variant="text" @click="settingsDialog = false">{{ l({lo:'ຍົກເລີກ',th:'ยกเลิก',en:'Cancel'}) }}</v-btn>
          <v-btn color="primary" :loading="saving" @click="saveSettings">{{ l({lo:'ບັນທຶກການຕັ້ງຄ່າ',th:'บันทึกการตั้งค่า',en:'Save Settings'}) }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.leads-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.leads-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  padding: 26px 28px;
  background: linear-gradient(135deg, #1e293b 0%, #0f172a 100%);
  border-radius: var(--radius-lg);
  color: #fff;
  border: 1px solid rgba(255, 255, 255, 0.08);
}
.hero-tag {
  color: #38bdf8;
  font-weight: 850;
  font-size: 11px;
  letter-spacing: 0.12em;
}
.hero-text h2 {
  margin: 6px 0 8px;
  font-size: clamp(22px, 2.5vw, 30px);
  font-weight: 800;
}
.hero-text p {
  margin: 0;
  color: #94a3b8;
  font-size: 13.5px;
  max-width: 640px;
  line-height: 1.5;
}
.hero-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.hero-btn {
  border-radius: 12px;
  font-weight: 750;
}

/* KPI Cards Grid */
.capi-health{display:grid;grid-template-columns:auto 1fr auto auto;align-items:center;gap:14px;margin-bottom:16px;padding:14px 16px;border:1px solid var(--color-border);border-radius:16px;background:var(--color-surface)}
.capi-health-icon{width:42px;height:42px;display:grid;place-items:center;border-radius:12px;color:var(--color-primary);background:var(--color-primary-soft)}
.capi-health-copy{display:grid;min-width:0}.capi-health-copy small{color:var(--color-text-muted);font-size:9px;font-weight:800;letter-spacing:.09em}.capi-health-copy strong{font-size:14px}.capi-health-copy span{overflow:hidden;color:var(--color-text-secondary);font-size:10px;text-overflow:ellipsis;white-space:nowrap}
.capi-health-metrics{display:flex;gap:8px}.capi-health-metrics>div{display:grid;min-width:70px;padding:8px 10px;border-radius:10px;background:var(--color-background)}.capi-health-metrics small{color:var(--color-text-muted);font-size:9px}.capi-health-metrics strong{font-size:15px}
.capi-health.healthy{border-color:rgba(22,163,74,.3)}.capi-health.healthy .capi-health-icon{color:#167a4b;background:rgba(22,163,74,.1)}.capi-health.failing{border-color:rgba(239,68,68,.35)}.capi-health.failing .capi-health-icon{color:var(--color-error);background:rgba(239,68,68,.1)}
.kpi-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 14px;
}
.kpi-card {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 16px 18px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.15s ease;
}
.kpi-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
  border-color: var(--color-primary);
}
.kpi-card.highlight-purchased {
  border-color: #86efac;
  background: #f0fdf4;
}
.kpi-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: grid;
  place-items: center;
}
.kpi-icon.all { background: #f1f5f9; color: #475569; }
.kpi-icon.new { background: #eff6ff; color: #2563eb; }
.kpi-icon.in_progress { background: #f5f3ff; color: #7c3aed; }
.kpi-icon.pending_payment { background: #fff7ed; color: #ea580c; }
.kpi-icon.purchased { background: #dcfce7; color: #16a34a; }
.kpi-content {
  display: flex;
  flex-direction: column;
}
.kpi-content small {
  font-size: 11.5px;
  color: var(--color-text-secondary);
  font-weight: 600;
}
.kpi-content strong {
  font-size: 22px;
  font-weight: 850;
}

/* Main Leads Panel */
.leads-main-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 20px;
}

.filter-bar {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: 20px;
}

.search-box {
  position: relative;
  display: flex;
  align-items: center;
  width: 100%;
}
.search-box input {
  width: 100%;
  height: 42px;
  padding: 0 36px 0 40px;
  border: 1px solid var(--color-border);
  border-radius: 21px;
  background: var(--color-background);
  color: var(--color-text);
  font-size: 13.5px;
  outline: none;
  transition: border-color 0.15s ease;
}
.search-box input:focus {
  border-color: var(--color-primary);
}
.search-box .search-icon {
  position: absolute;
  left: 14px;
  color: var(--color-text-muted);
}
.search-box .clear-btn {
  position: absolute;
  right: 12px;
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
}

.status-tabs {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  white-space: nowrap;
  padding-bottom: 2px;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}
.status-tabs::-webkit-scrollbar { display: none; }

.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 750;
  border: 1px solid var(--color-border);
  background: var(--color-background);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}
.tab-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.tab-btn.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: #fff !important;
}
.count-badge {
  padding: 1px 6px;
  border-radius: 10px;
  background: rgba(0,0,0,0.06);
  font-size: 10.5px;
}
.tab-btn.active .count-badge {
  background: rgba(255,255,255,0.25);
  color: #fff;
}
.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
}

/* Lead Cards List */
.lead-cards-list {
  display: grid;
  gap: 12px;
}
.lead-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 18px;
  border: 1px solid var(--color-border);
  border-radius: 14px;
  background: var(--color-background);
  transition: all 0.15s ease;
}
.lead-card:hover {
  border-color: var(--color-primary);
  background: var(--color-surface);
  box-shadow: 0 4px 14px rgba(0,0,0,0.04);
}
.lead-card.unread {
  border-left: 4px solid var(--color-primary);
  background: var(--color-primary-soft);
}

.lead-main-col {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}
.lead-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--color-surface-soft);
  display: grid;
  place-items: center;
  color: var(--color-text-secondary);
  flex-shrink: 0;
  border: 1px solid var(--color-border);
}
.lead-info {
  display: flex;
  flex-direction: column;
  gap: 5px;
  min-width: 0;
}
.lead-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.lead-name-row strong {
  font-size: 14.5px;
  font-weight: 750;
  color: var(--color-text);
}
.id-tag {
  font-size: 11px;
  color: var(--color-text-muted);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  padding: 2px 7px;
  border-radius: 8px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.copy-id-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text-muted);
  display: grid;
  place-items: center;
}
.copy-id-btn:hover { color: var(--color-primary); }
.unread-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
}

.lead-tags-row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.meta-pill {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 10px;
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--color-text-secondary);
  background: var(--color-surface);
  border: 1px solid var(--color-border-subtle);
}
.meta-pill.ad { background: #fff7ed; color: #c2410c; }
.meta-pill.page { background: #eff6ff; color: #1d4ed8; }
.meta-pill.time { color: var(--color-text-muted); }

/* Lead Actions Column */
.lead-actions-col {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.lead-status-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border-radius: 18px;
  font-size: 12px;
  font-weight: 750;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.15s ease;
}
.lead-status-btn:hover { opacity: 0.88; }

.quick-buy-btn {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: #16a34a;
  color: #fff;
  border: none;
  padding: 6px 12px;
  border-radius: 18px;
  font-size: 12px;
  font-weight: 750;
  cursor: pointer;
  transition: background 0.15s ease;
}
.quick-buy-btn:hover { background: #15803d; }

.purchased-badge {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
  padding: 5px 10px;
  border-radius: 18px;
  font-size: 11.5px;
  font-weight: 750;
}

.open-chat-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--color-primary);
  color: #fff;
  border: none;
  padding: 6px 12px;
  border-radius: 18px;
  font-size: 12px;
  font-weight: 750;
  cursor: pointer;
  transition: background 0.15s ease;
}
.open-chat-btn:hover { background: var(--color-primary-dark); }

.status-menu-popup {
  min-width: 190px;
  border-radius: 12px !important;
  padding: 6px !important;
}
.status-menu-item {
  border-radius: 8px !important;
  cursor: pointer !important;
  padding: 6px 12px !important;
}
.menu-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  display: inline-block;
  margin-right: 8px;
}

.empty-leads {
  text-align: center;
  padding: 50px 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  color: var(--color-text-secondary);
}
.empty-leads h3 { margin: 0; font-size: 18px; color: var(--color-text); }
.empty-leads p { margin: 0; max-width: 480px; font-size: 13px; line-height: 1.5; }

/* Dialog Styles */
.settings-dialog-card {
  border-radius: 20px !important;
}
.dialog-header {
  padding: 20px 24px 14px;
  font-size: 18px;
  font-weight: 750;
  display: flex;
  align-items: center;
}
.dialog-content {
  padding: 0 24px 16px;
}
.dialog-actions {
  padding: 12px 24px 20px;
}
.toggle-row {
  margin-bottom: 12px;
}
.capi-config-box {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  padding: 14px;
}
.step-card {
  background: var(--color-background);
  border: 1px solid var(--color-border);
  border-radius: 12px;
  padding: 12px;
}
.step-badge {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  font-weight: 800;
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 6px;
}

@media (max-width: 760px) {
  .capi-health { grid-template-columns: auto 1fr; align-items: start; }
  .capi-health-metrics { grid-column: 1/-1; display: grid; grid-template-columns: repeat(3, 1fr); }
  .capi-health > .v-btn { grid-column: 1/-1; width: 100%; }
  .leads-hero {
    flex-direction: column;
    align-items: flex-start;
    padding: 18px 16px;
  }
  .hero-actions {
    width: 100%;
    flex-wrap: wrap;
  }
  .lead-card {
    flex-direction: column;
    align-items: stretch;
    padding: 14px 12px;
  }
  .lead-actions-col {
    width: 100%;
    justify-content: space-between;
    flex-wrap: wrap;
    border-top: 1px solid var(--color-border-subtle);
    padding-top: 10px;
    margin-top: 6px;
  }
}
@media (max-width: 640px) {
  .leads-hero {
    padding: 16px 14px;
    border-radius: 16px;
  }
  .tab-btn {
    padding: 8px 12px;
    font-size: 11.5px;
    min-height: 38px;
    touch-action: manipulation;
  }
  .lead-actions-col {
    gap: 8px;
  }
  .lead-status-btn, .quick-buy-btn {
    flex: 1;
    justify-content: center;
    min-height: 42px;
    padding: 8px 12px;
    font-size: 12px;
    touch-action: manipulation;
  }
  .chat-now-btn {
    width: 42px;
    height: 42px;
    min-height: 42px;
    border-radius: 12px;
    display: inline-grid;
    place-items: center;
    touch-action: manipulation;
  }
  .lead-main {
    gap: 10px;
  }
  .lead-avatar {
    width: 40px;
    height: 40px;
  }
}
</style>
