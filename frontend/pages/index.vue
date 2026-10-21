<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { AutomationFlow, ItemsResponse, MetaPage, Product, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const route = useRoute()
const router = useRouter()
const { user, activeAccount } = useAuth()
const { t, l, locale } = useLocale()

interface Lead {
  id: string
  accountId: string
  pageId: string
  senderId: string
  sourceType: string
  sourceId: string
  name?: string
  status: string
  assignedUserId: string
  alertRead: boolean
  createdAt: string
}

const { data: pageData, pending: pagePending, refresh: refreshPages } = await useApi<ItemsResponse<MetaPage>>('/proxy/api/v1/meta/pages')
const { data: productData, refresh: refreshProducts } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: replyData, refresh: refreshReplies } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const { data: automationData, refresh: refreshAutomations } = await useApi<ItemsResponse<AutomationFlow>>('/proxy/api/v1/automations')
const { data: leadsData, refresh: refreshLeads } = await useApi<{ items: Lead[] }>('/proxy/api/v1/leads')

const pages = computed(() => pageData.value?.items.filter(item => item.connected) || [])
const products = computed(() => productData.value?.items || [])
const replies = computed(() => replyData.value || [])
const automations = computed(() => automationData.value?.items || [])
const activeAutomations = computed(() => automations.value.filter(item => item.isActive))
const leads = computed(() => leadsData.value?.items || [])

const justConnected = computed(() => route.query.meta === 'connected' && pages.value.length > 0)
const readyCount = computed(() => [pages.value.length, products.value.length, replies.value.length, automations.value.length].filter(Boolean).length)

// KPI calculations
const purchasedLeads = computed(() => leads.value.filter(l => (l.status || '').toLowerCase() === 'purchased'))
const newLeads = computed(() => leads.value.filter(l => (l.status || '').toLowerCase() === 'new'))
const recentLeads = computed(() => leads.value.slice(0, 6))

const steps = computed(() => [
  { done: pages.value.length > 0, icon: 'mdi-facebook', title: l({ lo: 'ເຊື່ອມ Facebook Page', th: 'เชื่อมต่อ Facebook Page', en: 'Connect Facebook Page' }), detail: pages.value.length ? `${pages.value.length} ${l({ lo: 'Page ເຊື່ອມແລ້ວ', th: 'เพจที่เชื่อมต่อแล้ว', en: 'Page(s) connected' })}` : l({ lo: 'ຈຳເປັນສຳລັບຮັບຂໍ້ຄວາມ', th: 'จำเป็นสำหรับรับข้อความ', en: 'Required to receive messages' }), to: '/meta-pages' },
  { done: products.value.length > 0, icon: 'mdi-package-variant-closed', title: l({ lo: 'ສ້າງສິນຄ້າ', th: 'สร้างสินค้า', en: 'Create a Product' }), detail: `${products.value.length} ${t('nav.products')}`, to: '/products' },
  { done: replies.value.length > 0, icon: 'mdi-message-text-fast-outline', title: l({ lo: 'ສ້າງຊຸດຂໍ້ຄວາມ', th: 'สร้างชุดข้อความ', en: 'Build a Reply Set' }), detail: `${replies.value.length} ${t('nav.replies')}`, to: '/replies' },
  { done: automations.value.length > 0, icon: 'mdi-robot-happy-outline', title: l({ lo: 'ສ້າງ Automation', th: 'สร้าง Automation', en: 'Create an Automation' }), detail: `${activeAutomations.value.length} ${l({ lo: 'ເປີດໃຊ້ຢູ່', th: 'กำลังใช้งาน', en: 'active' })}`, to: '/auto-replies' },
])

const quickNotice = ref('')
const updatingStatusId = ref<string | null>(null)

async function refreshAll() {
  await Promise.all([
    refreshPages(),
    refreshProducts(),
    refreshReplies(),
    refreshAutomations(),
    refreshLeads(),
  ])
}

async function markPurchased(senderId: string) {
  updatingStatusId.value = senderId
  try {
    await $fetch(`/api/proxy/api/v1/leads/${senderId}/status`, {
      method: 'PUT',
      body: { status: 'purchased' },
    })
    quickNotice.value = l({
      lo: `ປິດການຂາຍສຳເລັດ! ສົ່ງ Meta CAPI ຮຽບຮ້ອຍແລ້ວ`,
      th: `ปิดการขายสำเร็จ! ส่ง Meta CAPI เรียบร้อยแล้ว`,
      en: `Lead marked as Purchased and reported to Meta CAPI.`,
    })
    setTimeout(() => { quickNotice.value = '' }, 4000)
    await refreshLeads()
  } finally {
    updatingStatusId.value = null
  }
}

function openChat(senderId: string) {
  router.push(`/live-chat?senderId=${senderId}`)
}
</script>

<template>
  <div class="dashboard-container">
    <v-alert v-if="justConnected" type="success" variant="tonal" closable class="mb-4">
      {{ l({ lo: 'ເຊື່ອມ Facebook Page ສຳເລັດ ແລະ ລະບົບຮັບຂໍ້ຄວາມເປີດໃຊ້ແລ້ວ', th: 'เชื่อมต่อ Facebook Page สำเร็จและระบบรับข้อความเปิดใช้งานแล้ว', en: 'Facebook Page connected and messages system is active.' }) }}
    </v-alert>

    <v-alert v-if="quickNotice" type="success" variant="tonal" closable class="mb-4" @click:close="quickNotice = ''">
      {{ quickNotice }}
    </v-alert>

    <!-- Top Dashboard Hero -->
    <section class="dashboard-hero">
      <div class="hero-left">
        <div class="live-pill">
          <span class="live-pulse"></span>
          <span>{{ l({ lo: 'ລະບົບ Real-time ເປີດໃຊ້ງານຢູ່', th: 'ระบบ Real-time ทำงานอยู่', en: 'Real-time System Online' }) }}</span>
        </div>
        <h2>{{ activeAccount?.name || 'Workspace' }}</h2>
        <p>{{ l({ lo: 'ສູນຄວບຄຸມການຂາຍ, ບອດຕອບອັດຕະໂນມັດ, ຈັດການໂຄສະນາ ແລະ ການສົນທະນາ', th: 'ศูนย์ควบคุมการขาย บอทตอบอัตโนมัติ จัดการโฆษณา และการสนทนา', en: 'Central command for sales automation, bot replies, Meta ads and customer messaging.' }) }}</p>
      </div>

      <div class="hero-right">
        <v-btn prepend-icon="mdi-refresh" variant="tonal" class="text-white" @click="refreshAll">
          {{ l({ lo: 'ອັບເດດຂໍ້ມູນ', th: 'อัปเดตข้อมูล', en: 'Refresh Data' }) }}
        </v-btn>
      </div>
    </section>

    <!-- Quick Actions Launchpad (6 Instant Action Tiles) -->
    <section class="launchpad-section">
      <div class="section-header">
        <small class="section-tag">QUICK LAUNCHPAD</small>
        <h3>{{ l({ lo: 'ປຸ່ມລັດໃຊ້ງານດ່ວນ', th: 'ปุ่มลัดใช้งานด่วน', en: 'Quick Launchpad' }) }}</h3>
      </div>
      <div class="launchpad-grid">
        <NuxtLink to="/live-chat" class="launch-tile primary-tile">
          <div class="tile-icon"><v-icon icon="mdi-chat-processing-outline" size="26"/></div>
          <div class="tile-body">
            <strong>{{ l({ lo: 'Live Chat (ແຊັດສົດ)', th: 'Live Chat (แชทสด)', en: 'Live Chat' }) }}</strong>
            <small>{{ leads.length }} {{ l({ lo: 'ການສົນທະນາ', th: 'การสนทนา', en: 'chats' }) }}</small>
          </div>
          <v-icon icon="mdi-arrow-top-right" size="18" class="tile-arrow"/>
        </NuxtLink>

        <NuxtLink to="/leads" class="launch-tile">
          <div class="tile-icon leads-icon"><v-icon icon="mdi-account-group-outline" size="26"/></div>
          <div class="tile-body">
            <strong>{{ l({ lo: 'CRM & Leads', th: 'CRM & Leads', en: 'Leads & CRM' }) }}</strong>
            <small>{{ newLeads.length }} {{ l({ lo: 'Lead ໃໝ່', th: 'Lead ใหม่', en: 'new leads' }) }}</small>
          </div>
          <v-icon icon="mdi-arrow-top-right" size="18" class="tile-arrow"/>
        </NuxtLink>

        <NuxtLink to="/replies" class="launch-tile">
          <div class="tile-icon replies-icon"><v-icon icon="mdi-lightning-bolt-outline" size="26"/></div>
          <div class="tile-body">
            <strong>{{ l({ lo: 'ຊຸດຂໍ້ຄວາມດ່ວນ', th: 'ชุดข้อความด่วน', en: 'Quick Replies' }) }}</strong>
            <small>{{ replies.length }} {{ l({ lo: 'ຊຸດຂໍ້ຄວາມ', th: 'ชุดข้อความ', en: 'sets' }) }}</small>
          </div>
          <v-icon icon="mdi-arrow-top-right" size="18" class="tile-arrow"/>
        </NuxtLink>

        <NuxtLink to="/products" class="launch-tile">
          <div class="tile-icon products-icon"><v-icon icon="mdi-package-variant-closed" size="26"/></div>
          <div class="tile-body">
            <strong>{{ l({ lo: 'ຈັດການສິນຄ້າ', th: 'จัดการสินค้า', en: 'Products' }) }}</strong>
            <small>{{ products.length }} {{ l({ lo: 'ລາຍການ', th: 'รายการ', en: 'items' }) }}</small>
          </div>
          <v-icon icon="mdi-arrow-top-right" size="18" class="tile-arrow"/>
        </NuxtLink>

        <NuxtLink to="/auto-replies" class="launch-tile">
          <div class="tile-icon bot-icon"><v-icon icon="mdi-robot-happy-outline" size="26"/></div>
          <div class="tile-body">
            <strong>{{ l({ lo: 'ບອດ Auto-Reply', th: 'บอท Auto-Reply', en: 'Bot Automations' }) }}</strong>
            <small>{{ activeAutomations.length }} {{ l({ lo: 'Flow ເປີດໃຊ້', th: 'Flow เปิดใช้', en: 'active' }) }}</small>
          </div>
          <v-icon icon="mdi-arrow-top-right" size="18" class="tile-arrow"/>
        </NuxtLink>

        <NuxtLink to="/ads" class="launch-tile">
          <div class="tile-icon ads-icon"><v-icon icon="mdi-bullhorn-outline" size="26"/></div>
          <div class="tile-body">
            <strong>{{ l({ lo: 'Meta Ads (ໂຄສະນາ)', th: 'Meta Ads (โฆษณา)', en: 'Meta Ads' }) }}</strong>
            <small>{{ l({ lo: 'ຕັ້ງຄ່າດັກທັກແຊັດ', th: 'ตั้งค่าดักทักแชท', en: 'Ad comment rules' }) }}</small>
          </div>
          <v-icon icon="mdi-arrow-top-right" size="18" class="tile-arrow"/>
        </NuxtLink>
      </div>
    </section>

    <!-- KPI Metric Cards Grid -->
    <section class="kpi-section">
      <div class="kpi-card">
        <div class="kpi-icon-box leads"><v-icon icon="mdi-account-plus" size="24"/></div>
        <div class="kpi-meta">
          <small>{{ l({ lo: 'Leads ທັງໝົດ', th: 'Leads ทั้งหมด', en: 'Total Leads' }) }}</small>
          <strong>{{ leads.length }}</strong>
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-icon-box purchased"><v-icon icon="mdi-check-decagram" size="24"/></div>
        <div class="kpi-meta">
          <small>{{ l({ lo: 'ປິດການຂາຍແລ້ວ (CAPI)', th: 'ปิดการขายแล้ว (CAPI)', en: 'Deals Closed (CAPI)' }) }}</small>
          <strong class="text-success">{{ purchasedLeads.length }}</strong>
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-icon-box channels"><v-icon icon="mdi-facebook" size="24"/></div>
        <div class="kpi-meta">
          <small>{{ l({ lo: 'Page ທີ່ເຊື່ອມຕໍ່', th: 'เพจที่เชื่อมต่อ', en: 'Connected Pages' }) }}</small>
          <strong>{{ pagePending ? '…' : pages.length }}</strong>
        </div>
      </div>

      <div class="kpi-card">
        <div class="kpi-icon-box flows"><v-icon icon="mdi-target" size="24"/></div>
        <div class="kpi-meta">
          <small>{{ l({ lo: 'Automation ທີ່ເປີດໃຊ້', th: 'Automation ที่เปิดใช้', en: 'Active Automations' }) }}</small>
          <strong>{{ activeAutomations.length }}</strong>
        </div>
      </div>
    </section>

    <!-- Main Split: Recent Activity & Setup Checklist -->
    <div class="dashboard-split">
      <!-- Left: Recent Leads Feed with 1-click actions -->
      <section class="dashboard-panel">
        <div class="panel-header">
          <div>
            <small class="section-tag">REAL-TIME INBOUND</small>
            <h3>{{ l({ lo: 'ລູກຄ້າ / Leads ລ່າສຸດ', th: 'ลูกค้า / Leads ล่าสุด', en: 'Recent Inbound Leads' }) }}</h3>
          </div>
          <v-btn variant="text" size="small" to="/leads" append-icon="mdi-arrow-right">
            {{ l({ lo: 'ເບິ່ງທັງໝົດ', th: 'ดูทั้งหมด', en: 'View all' }) }}
          </v-btn>
        </div>

        <div v-if="recentLeads.length" class="recent-leads-list">
          <div v-for="lead in recentLeads" :key="lead.id" class="lead-item-card">
            <div class="lead-item-left">
              <div class="lead-mini-avatar">
                <v-icon icon="mdi-account" size="18"/>
              </div>
              <div class="lead-item-info">
                <strong>{{ lead.name || `${l({lo:'ລູກຄ້າ',th:'ลูกค้า',en:'Customer'})} · ${lead.senderId.slice(-4)}` }}</strong>
                <div class="lead-item-tags">
                  <span class="source-tag"><v-icon icon="mdi-flag" size="11"/> {{ lead.pageId }}</span>
                  <span v-if="lead.sourceId" class="ad-tag"><v-icon icon="mdi-bullhorn" size="11"/> Ad: {{ lead.sourceId }}</span>
                  <span class="time-tag">{{ new Date(lead.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) }}</span>
                </div>
              </div>
            </div>

            <div class="lead-item-actions">
              <span v-if="lead.status === 'purchased'" class="status-pill purchased">
                <v-icon icon="mdi-check-circle" size="12" class="mr-1"/>
                {{ locale === 'lo' ? 'ປິດການຂາຍແລ້ວ' : 'Purchased' }}
              </span>
              <button 
                v-else 
                type="button" 
                class="btn-quick-buy" 
                :disabled="updatingStatusId === lead.senderId"
                @click="markPurchased(lead.senderId)"
                :title="l({lo:'ປິດການຂາຍທັນທີ (ສົ່ງ Meta CAPI)',th:'ปิดการขายทันที (ส่ง Meta CAPI)',en:'Mark Purchased (Meta CAPI)'})"
              >
                <v-icon icon="mdi-lightning-bolt" size="13"/>
                <span>{{ locale === 'lo' ? 'ປິດການຂາຍ' : 'Close' }}</span>
              </button>

              <button 
                type="button" 
                class="btn-quick-chat" 
                @click="openChat(lead.senderId)"
                :title="l({lo:'ເປີດແຊັດກັບລູກຄ້າ',th:'เปิดแชทกับลูกค้า',en:'Open Chat'})"
              >
                <v-icon icon="mdi-message-text-outline" size="14"/>
              </button>
            </div>
          </div>
        </div>

        <div v-else class="empty-leads-box">
          <v-icon icon="mdi-chat-question-outline" size="40" color="grey"/>
          <p>{{ l({ lo: 'ຍັງບໍ່ມີ Leads ເຂົ້າມາ; ເມື່ອມີລູກຄ້າທັກແຊັດ ຫຼື ຄອມເມັ້ນຈະຂຶ້ນຢູ່ນີ້ທັນທີ', th: 'ยังไม่มี Leads เข้ามา เมื่อมีลูกค้าทักแชทหรือคอมเมนต์จะขึ้นที่นี่ทันที', en: 'No leads yet. Messages and comments will appear here automatically.' }) }}</p>
          <v-btn color="primary" size="small" to="/meta-pages">
            {{ l({ lo: 'ເຊື່ອມຕໍ່ Facebook Page', th: 'เชื่อมต่อ Facebook Page', en: 'Connect Facebook Page' }) }}
          </v-btn>
        </div>
      </section>

      <!-- Right: Readiness Checklist -->
      <section class="dashboard-panel">
        <div class="panel-header">
          <div>
            <small class="section-tag">WORKSPACE READINESS</small>
            <h3>{{ readyCount === 4 ? l({ lo: 'ພ້ອມໃຊ້ງານສົມບູນ 100%', th: 'พร้อมใช้งานสมบูรณ์ 100%', en: 'Setup Complete 100%' }) : l({ lo: 'ຂັ້ນຕອນການຕັ້ງຄ່າ', th: 'ขั้นตอนการตั้งค่า', en: 'Setup Checklist' }) }}</h3>
          </div>
          <v-btn variant="text" size="small" to="/automation-guide" append-icon="mdi-help-circle-outline">
            {{ t('auto.guide') }}
          </v-btn>
        </div>

        <div class="readiness-progress-bar">
          <div class="bar-fill" :style="{ width: `${readyCount * 25}%` }"></div>
        </div>
        <small class="readiness-count-text">{{ readyCount }}/4 {{ l({ lo: 'ຂັ້ນຕອນສຳເລັດແລ້ວ', th: 'ขั้นตอนสำเร็จแล้ว', en: 'steps completed' }) }}</small>

        <div class="steps-compact-list">
          <NuxtLink v-for="(item, index) in steps" :key="item.to" :to="item.to" class="step-link">
            <span class="step-check" :class="{ done: item.done }">
              <v-icon :icon="item.done ? 'mdi-check' : item.icon" size="18"/>
            </span>
            <div class="step-info">
              <strong>{{ index + 1 }}. {{ item.title }}</strong>
              <small>{{ item.detail }}</small>
            </div>
            <v-icon icon="mdi-chevron-right" size="18" color="grey"/>
          </NuxtLink>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.dashboard-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 24px;
  padding: 28px;
  background: var(--color-rail);
  border-radius: var(--radius-lg);
  color: white;
  border: 1px solid var(--color-rail-hover);
}

.live-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.14);
  font-size: 12px;
  font-weight: 750;
  margin-bottom: 8px;
}
.live-pulse {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-success);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-success) 30%, transparent);
}

.hero-left h2 {
  margin: 0 0 6px;
  font-size: clamp(24px, 2.6vw, 32px);
  font-weight: 850;
}
.hero-left p {
  margin: 0;
  color: rgba(255, 255, 255, 0.75);
  font-size: 13.5px;
  max-width: 580px;
  line-height: 1.5;
}

.section-tag {
  font-size: 12px;
  font-weight: 850;
  letter-spacing: 0.12em;
  color: var(--color-primary);
}

/* Launchpad Grid */
.launchpad-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.section-header h3 {
  margin: 4px 0 0;
  font-size: 18px;
  font-weight: 800;
}

.launchpad-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}

.launch-tile {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  color: var(--color-text);
  text-decoration: none;
  transition: all 0.18s ease;
  position: relative;
}
.launch-tile:hover {
  border-color: var(--color-primary);
  background: var(--color-surface-hover);
}
.launch-tile.primary-tile {
  background: var(--color-primary-softer);
  border-color: color-mix(in srgb, var(--color-primary) 30%, transparent);
}
.launch-tile.primary-tile:hover {
  border-color: var(--color-primary);
}

.tile-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: var(--color-primary-soft);
  color: var(--color-primary);
  flex-shrink: 0;
}
.tile-icon.leads-icon,
.tile-icon.replies-icon,
.tile-icon.products-icon,
.tile-icon.bot-icon,
.tile-icon.ads-icon { background: var(--color-primary-soft); color: var(--color-primary); }

.tile-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}
.tile-body strong {
  font-size: 13.5px;
  font-weight: 750;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tile-body small {
  font-size: 12px;
  color: var(--color-text-muted);
}
.tile-arrow {
  color: var(--color-text-muted);
  transition: transform 0.15s ease;
}
.launch-tile:hover .tile-arrow {
  color: var(--color-primary);
  transform: translate(2px, -2px);
}

/* KPI Cards */
.kpi-section {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
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
}
.kpi-icon-box {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: grid;
  place-items: center;
}
.kpi-icon-box.leads { background: var(--color-primary-soft); color: var(--color-primary); }
.kpi-icon-box.purchased { background: var(--color-success-soft); color: var(--color-success); }
.kpi-icon-box.channels { background: var(--color-surface-soft); color: var(--color-text-secondary); }
.kpi-icon-box.flows { background: var(--color-purple-soft); color: #9333ea; }

.kpi-meta {
  display: flex;
  flex-direction: column;
}
.kpi-meta small {
  font-size: 12px;
  color: var(--color-text-secondary);
  font-weight: 600;
}
.kpi-meta strong {
  font-size: 24px;
  font-weight: 850;
}

/* Split View */
.dashboard-split {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(320px, 1fr);
  gap: 18px;
}
.dashboard-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  flex-direction: column;
}
.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.panel-header h3 {
  margin: 3px 0 0;
  font-size: 16px;
  font-weight: 800;
}

/* Recent Leads List */
.recent-leads-list {
  display: grid;
  gap: 10px;
}
.lead-item-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--color-border-subtle);
  background: var(--color-background);
  transition: all 0.15s ease;
}
.lead-item-card:hover {
  background: var(--color-surface);
  border-color: var(--color-border);
}
.lead-item-left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.lead-mini-avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--color-surface-soft);
  display: grid;
  place-items: center;
  color: var(--color-text-secondary);
  flex-shrink: 0;
}
.lead-item-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.lead-item-info strong {
  font-size: 13px;
  font-weight: 750;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.lead-item-tags {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 2px;
}
.source-tag, .ad-tag, .time-tag {
  font-size: 12px;
  color: var(--color-text-muted);
}
.ad-tag { color: var(--color-warning); }

.lead-item-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.status-pill.purchased {
  font-size: 12px;
  font-weight: 750;
  padding: 4px 9px;
  border-radius: 14px;
  background: var(--color-success-soft);
  color: var(--color-success);
}
.btn-quick-buy {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 5px 10px;
  border-radius: 14px;
  background: var(--color-primary);
  color: var(--color-surface);
  border: none;
  font-size: 12px;
  font-weight: 750;
  cursor: pointer;
  transition: background 0.15s ease;
}
.btn-quick-buy:hover { background: var(--color-primary-hover); }

.btn-quick-chat {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--color-primary-soft);
  color: var(--color-primary);
  border: none;
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: all 0.15s ease;
}
.btn-quick-chat:hover {
  background: var(--color-primary);
  color: var(--color-surface);
}

.empty-leads-box {
  padding: 40px 16px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--color-text-secondary);
}
.empty-leads-box p {
  margin: 0;
  font-size: 12.5px;
  max-width: 380px;
}

/* Readiness Checklist */
.readiness-progress-bar {
  height: 6px;
  background: var(--color-background);
  border-radius: 99px;
  overflow: hidden;
  margin-bottom: 6px;
}
.bar-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: inherit;
  transition: width 0.3s ease;
}
.readiness-count-text {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 14px;
  display: block;
}

.steps-compact-list {
  display: grid;
  gap: 8px;
}
.text-success{color:var(--color-success)}
.step-link {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--color-border-subtle);
  background: var(--color-background);
  color: var(--color-text);
  text-decoration: none;
  transition: all 0.15s ease;
}
.step-link:hover {
  background: var(--color-surface-soft);
  border-color: var(--color-border);
}
.step-check {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: var(--color-primary-soft);
  color: var(--color-primary);
  flex-shrink: 0;
}
.step-check.done {
  background: var(--color-success-soft);
  color: var(--color-success);
}
.step-info {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}
.step-info strong {
  font-size: 13px;
  font-weight: 750;
}
.step-info small {
  font-size: 12px;
  color: var(--color-text-muted);
}

@media (max-width: 900px) {
  .dashboard-split {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .dashboard-hero {
    flex-direction: column;
    align-items: flex-start;
    padding: 20px;
    gap: 16px;
  }
  .launchpad-grid {
    grid-template-columns: repeat(2, 1fr);
  }
  .kpi-section {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
