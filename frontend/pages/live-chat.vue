<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import type { ItemsResponse, Product, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

interface Lead {
  id: string
  accountId: string
  pageId: string
  senderId: string
  sourceType: string
  sourceId: string
  name: string
  status: string
}
interface ChatMessage {
  id: string
  sender_id: string
  sender_name?: string
  sender_pic?: string
  message: string
  type: string
  timestamp: string
  platform: 'facebook' | 'whatsapp' | 'system'
  page_id: string
}

const leads = ref<Record<string, Lead>>({})
const conversations = ref<Record<string, ChatMessage[]>>({})
const activeSender = ref<string | null>(null)
const inputMessage = ref('')
const evtSource = ref<EventSource | null>(null)
const sending = ref(false)
const loadError = ref('')
const { locale, l } = useLocale()
const liveConnected = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null
const messagesContainer = ref<HTMLElement | null>(null)

// Reply Sets and Product Codes for Quick Auto-reply by Admin
const { data: replySetsData } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const { data: productsData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const replySets = computed(() => replySetsData.value || [])
const products = computed(() => productsData.value?.items || [])

const replySetsDialog = ref(false)
const dialogSearch = ref('')
const sendSuccessNotice = ref('')

interface ReplyCodeItem {
  code: string
  name: string
  type: 'set' | 'product'
  details: string
  replySetId?: string
}

const availableCodes = computed<ReplyCodeItem[]>(() => {
  const list: ReplyCodeItem[] = []
  for (const s of replySets.value) {
    if (s.name) {
      list.push({
        code: s.code || s.name,
        name: s.name,
        type: 'set',
        details: `${s.items?.length || 0} ${l({lo:'ຂໍ້ຄວາມ',th:'ข้อความ',en:'messages'})}`,
        replySetId: s.id,
      })
    }
  }
  for (const p of products.value) {
    if (p.code || p.name) {
      list.push({
        code: p.code || p.name,
        name: p.name,
        type: 'product',
        details: `${p.price ? `${p.price} · ` : ''}${p.code ? `Code: ${p.code}` : ''}`,
      })
    }
  }
  return list
})

const filteredDialogCodes = computed(() => {
  const q = dialogSearch.value.trim().toLowerCase()
  if (!q) return availableCodes.value
  return availableCodes.value.filter(c => c.code.toLowerCase().includes(q) || c.name.toLowerCase().includes(q) || c.details.toLowerCase().includes(q))
})

const matchingCodes = computed(() => {
  const query = inputMessage.value.trim().toLowerCase()
  if (!query) return []
  const clean = query.replace(/^[/#!:]+/, '').trim()
  if (!clean && (query.startsWith('/') || query.startsWith('#'))) {
    return availableCodes.value.slice(0, 6)
  }
  if (!clean) return []
  return availableCodes.value.filter(c =>
    c.code.toLowerCase().includes(clean) || c.name.toLowerCase().includes(clean)
  ).slice(0, 6)
})

const route = useRoute()
const router = useRouter()
const isMounted = ref(false)

const selectConversation = (senderId: string) => {
  activeSender.value = senderId
  if (typeof window !== 'undefined' && window.innerWidth <= 640) {
    void router.replace({ query: { ...route.query, c: senderId } })
  }
}

const goBackToList = () => {
  activeSender.value = null
  if (route.query.c) {
    const q = { ...route.query }
    delete q.c
    void router.replace({ query: q })
  }
}

watch(() => route.query.c, (newC) => {
  if (newC && typeof newC === 'string') {
    if (activeSender.value !== newC) {
      activeSender.value = newC
    }
  } else if (!newC && typeof window !== 'undefined' && window.innerWidth <= 640) {
    activeSender.value = null
  }
}, { immediate: true })

const selectCode = (item: ReplyCodeItem, autoSend = false) => {
  if (autoSend) {
    sendReply(item.code, item.replySetId)
  } else {
    inputMessage.value = item.code.startsWith('#') ? item.code : `#${item.code}`
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

watch(activeSender, () => {
  scrollToBottom()
})

const activeChatMessages = computed(() => {
  if (!activeSender.value) return []
  return conversations.value[activeSender.value] || []
})

const getSenderName = (msgs?: ChatMessage[]) => {
  if (!msgs || msgs.length === 0) return 'Unknown'
  const msg = msgs.find(m => m.sender_name)
  if (msg?.sender_name) return msg.sender_name
  const id = msgs[0]?.sender_id || ''
  return `${l({lo:'ລູກຄ້າ',th:'ลูกค้า',en:'Customer'})} · ${id.slice(-4)}`
}

const getSenderPic = (msgs?: ChatMessage[]) => {
  if (!msgs || msgs.length === 0) return undefined
  const msg = msgs.find(m => m.sender_pic)
  return msg?.sender_pic
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

const statusNotice = ref('')
const updatingStatus = ref(false)

const changeLeadStatus = async (senderId: string, newStatus: string) => {
  if (!senderId || updatingStatus.value) return
  updatingStatus.value = true
  try {
    await $fetch(`/api/proxy/api/v1/leads/${senderId}/status`, {
      method: 'PUT',
      body: { status: newStatus }
    })
    if (!leads.value[senderId]) {
      const msgs = conversations.value[senderId]
      const pageId = msgs?.find(m => m.page_id)?.page_id || ''
      leads.value[senderId] = {
        id: senderId,
        accountId: '',
        pageId,
        senderId,
        sourceType: 'message',
        sourceId: '',
        name: getSenderName(msgs),
        status: newStatus
      }
    } else {
      leads.value[senderId].status = newStatus
    }
    const info = getStatusInfo(newStatus)
    statusNotice.value = l({
      lo: `ປ່ຽນສະຖານະເປັນ “${l(info.label)}” ສຳເລັດ`,
      th: `เปลี่ยนสถานะเป็น “${l(info.label)}” สำเร็จ`,
      en: `Status updated to “${l(info.label)}”`
    })
    setTimeout(() => { statusNotice.value = '' }, 4000)
  } catch (e: any) {
    loadError.value = l({ lo: 'ອັບເດດສະຖານະບໍ່ສຳເລັດ', th: 'อัปเดตสถานะไม่สำเร็จ', en: 'Failed to update lead status' })
  } finally {
    updatingStatus.value = false
  }
}

const markAsPurchased = (senderId: string) => changeLeadStatus(senderId, 'purchased')

const searchQuery = ref('')
const selectedStatusFilter = ref<string>('all')

const parseTimestamp = (ts: string | number | undefined): number => {
  if (!ts) return 0
  if (ts === 'now') return Date.now()
  const n = Number(ts)
  if (!isNaN(n) && n > 0) return n
  const d = new Date(ts).getTime()
  return isNaN(d) ? 0 : d
}

interface ConversationSummary {
  senderId: string
  messages: ChatMessage[]
  lastMessage: ChatMessage | null
  lastTime: number
  lead?: Lead
  status: string
  senderName: string
  senderPic?: string
  platform: 'facebook' | 'whatsapp' | 'system'
  pageId: string
}

const allConversationsList = computed<ConversationSummary[]>(() => {
  const list: ConversationSummary[] = []
  for (const [senderId, msgs] of Object.entries(conversations.value)) {
    if (!msgs || msgs.length === 0) continue
    const lastMsg = msgs[msgs.length - 1] || null
    const lastTime = parseTimestamp(lastMsg?.timestamp)
    const lead = leads.value[senderId]
    const status = (lead?.status || 'new').toLowerCase()
    const senderName = getSenderName(msgs)
    const senderPic = getSenderPic(msgs)
    const platform = msgs[0]?.platform || 'facebook'
    const pageId = msgs.find(m => m.page_id)?.page_id || lead?.pageId || ''

    list.push({
      senderId,
      messages: msgs,
      lastMessage: lastMsg,
      lastTime,
      lead,
      status,
      senderName,
      senderPic,
      platform,
      pageId,
    })
  }

  // Sort descending: Recent activity at the very top!
  list.sort((a, b) => b.lastTime - a.lastTime)
  return list
})

const filteredConversations = computed<ConversationSummary[]>(() => {
  let list = allConversationsList.value

  if (selectedStatusFilter.value !== 'all') {
    list = list.filter(c => c.status === selectedStatusFilter.value)
  }

  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(c => {
      const matchName = c.senderName.toLowerCase().includes(q)
      const matchId = c.senderId.toLowerCase().includes(q)
      const matchPage = c.pageId.toLowerCase().includes(q)
      const matchAd = (c.lead?.sourceId || '').toLowerCase().includes(q)
      const matchText = (c.lastMessage?.message || '').toLowerCase().includes(q)
      return matchName || matchId || matchPage || matchAd || matchText
    })
  }

  return list
})

const statusCounts = computed(() => {
  const counts: Record<string, number> = { all: allConversationsList.value.length }
  for (const opt of STATUS_OPTIONS) {
    counts[opt.key] = 0
  }
  for (const c of allConversationsList.value) {
    counts[c.status] = (counts[c.status] ?? 0) + 1
  }
  return counts
})

const activeSenderSummary = computed(() => {
  if (!activeSender.value) return null
  return allConversationsList.value.find(c => c.senderId === activeSender.value) || null
})

const activeLeadStatus = computed(() => {
  if (!activeSender.value) return 'new'
  return (leads.value[activeSender.value]?.status || 'new').toLowerCase()
})

const formatPancakeTime = (ts: string | number | undefined) => {
  if (!isMounted.value) return ''
  if (!ts) return ''
  if (ts === 'now') return l({ lo: 'ດຽວນີ້', th: 'ตอนนี้', en: 'Now' })
  const time = parseTimestamp(ts)
  if (!time) return ''
  const date = new Date(time)
  const now = new Date()

  const isToday = date.toDateString() === now.toDateString()
  if (isToday) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })
  }

  const yesterday = new Date(now)
  yesterday.setDate(yesterday.getDate() - 1)
  if (date.toDateString() === yesterday.toDateString()) {
    return l({ lo: 'ມື້ວານ', th: 'เมื่อวาน', en: 'Yesterday' })
  }

  const day = String(date.getDate()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')
  return `${day}/${month}`
}

const sendReply = async (explicitCode?: string, explicitReplySetId?: string) => {
  const msgToSend = (explicitCode !== undefined ? explicitCode : inputMessage.value).trim()
  if (!msgToSend || !activeSender.value || sending.value) return

  // Find the page ID associated with this sender
  const msgs = conversations.value[activeSender.value]
  const pageId = msgs?.find(m => m.page_id)?.page_id || leads.value[activeSender.value]?.pageId || ''

  sending.value = true
  loadError.value = ''
  sendSuccessNotice.value = ''
  try {
    const res = await $fetch<{ status: string; reply_set_sent?: boolean; reply_set_name?: string; items_count?: number }>('/api/proxy/api/v1/chat/send', {
      method: 'POST',
      body: {
        page_id: pageId,
        recipient_id: activeSender.value,
        message: msgToSend,
        reply_set_id: explicitReplySetId || ''
      },
    })
    inputMessage.value = ''
    if (res.reply_set_sent) {
      sendSuccessNotice.value = l({
        lo: `⚡ ສົ່ງຊຸດຂໍ້ຄວາມ “${res.reply_set_name}” (${res.items_count} ລາຍການ) ແລ້ວ!`,
        th: `⚡ ส่งชุดข้อความ “${res.reply_set_name}” (${res.items_count} รายการ) แล้ว!`,
        en: `⚡ Sent Reply Set “${res.reply_set_name}” (${res.items_count} messages)!`,
      })
      setTimeout(() => { sendSuccessNotice.value = '' }, 5000)
    }
    await fetchHistoryAndLeads(true)
    scrollToBottom()
  } catch (error: any) {
    const detail = error?.data?.detail || error?.data?.message || error?.statusMessage || ''
    if (detail.includes('not found') || detail.includes('not connected')) {
      loadError.value = l({
        lo: 'Page ຍັງບໍ່ໄດ້ເຊື່ອມຕໍ່ ຫຼື Token ໝົດອາຍຸ, ກະລຸນາກວດສອບທີ່ເມນູ Meta Pages',
        th: 'เพจยังไม่ได้เชื่อมต่อ หรือ Token หมดอายุ กรุณาตรวจสอบที่เมนู Meta Pages',
        en: 'Page not connected or token expired. Please check Meta Pages.'
      })
    } else {
      loadError.value = detail || l({lo:'ສົ່ງຂໍ້ຄວາມບໍ່ສຳເລັດ',th:'ส่งข้อความไม่สำเร็จ',en:'Message could not be sent.'})
    }
  } finally {
    sending.value = false
  }
}

let reconnectTimeout: any = null

const messageKey = (msg: ChatMessage) => `${msg.page_id}|${msg.sender_id}|${msg.platform}|${msg.timestamp}|${msg.type}|${msg.message}`
const mergeMessages = (items: ChatMessage[]) => {
  const byKey = new Map<string, ChatMessage>()
  for (const list of Object.values(conversations.value)) for (const item of list) byKey.set(messageKey(item), item)
  for (const item of items) byKey.set(messageKey(item), { ...item, id: messageKey(item) })
  const next: Record<string, ChatMessage[]> = {}
  for (const item of byKey.values()) (next[item.sender_id] ||= []).push(item)
  for (const list of Object.values(next)) {
    list.sort((a,b) => {
      const timeA = a.timestamp === 'now' ? Date.now() : (Number(a.timestamp) || new Date(a.timestamp).getTime() || 0)
      const timeB = b.timestamp === 'now' ? Date.now() : (Number(b.timestamp) || new Date(b.timestamp).getTime() || 0)
      return timeA - timeB
    })
  }
  conversations.value = next
  if (!activeSender.value && allConversationsList.value.length) {
    if (typeof window !== 'undefined' && window.innerWidth > 640) {
      activeSender.value = allConversationsList.value[0]?.senderId || null
    }
  }
  scrollToBottom()
}

const fetchHistoryAndLeads = async (silent = false) => {
  try {
    const [historyRes, leadsRes] = await Promise.all([
      $fetch<{items: ChatMessage[]}>('/api/proxy/api/v1/chat/history').catch(()=>({items:[]})),
      $fetch<{items: Lead[]}>('/api/proxy/api/v1/leads').catch(()=>({items:[]}))
    ])

    const leadsList = leadsRes.items || []
    leadsList.forEach(l => { leads.value[l.senderId] = l })

    const history = historyRes.items || []
    mergeMessages(history)
  } catch (err) {
    if (!silent) loadError.value = l({lo:'ໂຫຼດປະຫວັດສົນທະນາບໍ່ສຳເລັດ',th:'โหลดประวัติการสนทนาไม่สำเร็จ',en:'Conversation history could not be loaded.'})
  }
}

const connectLiveChat = () => {
  if (evtSource.value) {
    evtSource.value.close()
    evtSource.value = null
  }

  const url = '/api/proxy/api/v1/chat/stream'
  try {
    const es = new EventSource(url)
    evtSource.value = es

    es.onopen = async () => {
      liveConnected.value = true
      await fetchHistoryAndLeads(true)
    }

    es.onmessage = (event) => {
      let data: ChatMessage
      try {
        data = JSON.parse(event.data) as ChatMessage
      } catch {
        return
      }
      if (data.type !== 'lead_alert') {
        mergeMessages([data])
      }
      void fetchHistoryAndLeads(true)
    }

    es.onerror = () => {
      liveConnected.value = false
      es.close()
      evtSource.value = null
      clearTimeout(reconnectTimeout)
      reconnectTimeout = setTimeout(() => {
        connectLiveChat()
      }, 5000)
    }
  } catch {
    liveConnected.value = false
  }
}

onMounted(() => {
  isMounted.value = true
  if (import.meta.server) return
  if (route.query.c && typeof route.query.c === 'string') {
    activeSender.value = String(route.query.c)
  }
  connectLiveChat()
  void fetchHistoryAndLeads()
  pollTimer = setInterval(() => { void fetchHistoryAndLeads(true) }, 3000)
})

onUnmounted(() => {
  if (evtSource.value) {
    evtSource.value.close()
  }
  clearTimeout(reconnectTimeout)
  if (pollTimer) clearInterval(pollTimer)
})

const formatTime = (ts: string) => {
  if (!isMounted.value) return ''
  if (ts === 'now') return l({lo:'ດຽວນີ້',th:'เมื่อสักครู่',en:'Just now'})
  const d = new Date(Number(ts))
  if (isNaN(d.getTime())) return ts
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="chat-layout" :class="{ 'mobile-in-chat': activeSender }">
    <div class="chat-sidebar" :class="{ 'mobile-hidden': activeSender }">
      <div class="sidebar-header">
        <div class="sidebar-header-top">
          <h2>{{locale==='lo'?'ກ່ອງຂໍ້ຄວາມ':'Live Inbox'}}</h2>
          <span class="live-state" :class="{online:liveConnected}"><i/>{{ liveConnected ? l({lo:'Real-time',th:'Real-time',en:'Live'}) : l({lo:'ກຳລັງເຊື່ອມຕໍ່...',th:'กำลังเชื่อมต่อ...',en:'Connecting'}) }}</span>
        </div>
        <!-- Pancake Search Box -->
        <div class="sidebar-search">
          <v-icon icon="mdi-magnify" size="18" class="search-icon" />
          <input
            v-model="searchQuery"
            type="text"
            :placeholder="l({lo:'ຄົ້ນຫາຊື່, ເບີ, ຂໍ້ຄວາມ...',th:'ค้นหาชื่อ, เบอร์, ข้อความ...',en:'Search name, phone, msg...'})"
          />
          <button v-if="searchQuery" class="clear-search-btn" @click="searchQuery = ''" type="button">
            <v-icon icon="mdi-close" size="14"/>
          </button>
        </div>
        <!-- Pancake Status Filter Tabs -->
        <div class="status-filters">
          <button
            type="button"
            class="filter-pill"
            :class="{ active: selectedStatusFilter === 'all' }"
            @click="selectedStatusFilter = 'all'"
          >
            {{ l({lo:'ທັງໝົດ',th:'ทั้งหมด',en:'All'}) }}
            <span class="filter-count">{{ statusCounts.all || 0 }}</span>
          </button>
          <button
            v-for="opt in STATUS_OPTIONS"
            :key="opt.key"
            type="button"
            class="filter-pill"
            :class="{ active: selectedStatusFilter === opt.key }"
            :style="selectedStatusFilter === opt.key ? { backgroundColor: opt.bg, borderColor: opt.color, color: opt.color } : {}"
            @click="selectedStatusFilter = opt.key"
          >
            <span class="status-dot" :style="{ backgroundColor: opt.color }"></span>
            {{ l(opt.label) }}
            <span class="filter-count" v-if="statusCounts[opt.key]">{{ statusCounts[opt.key] }}</span>
          </button>
        </div>
      </div>

      <div class="conversation-list">
        <ClientOnly>
          <div v-if="filteredConversations.length === 0" class="empty-list">
            <v-icon icon="mdi-chat-question-outline" size="32" class="mb-2"/>
            <p>{{locale==='lo'?'ບໍ່ພົບການສົນທະນາ':'No conversations found.'}}</p>
          </div>
          <div
            v-for="item in filteredConversations"
            :key="item.senderId"
            class="conv-item"
            :class="{ active: activeSender === item.senderId }"
            @click="selectConversation(item.senderId)"
          >
            <div class="conv-avatar">
              <img v-if="item.senderPic" :src="item.senderPic" alt="Profile" class="avatar-img" />
              <v-icon v-else :icon="item.platform === 'whatsapp' ? 'mdi-whatsapp' : 'mdi-facebook-messenger'" />
              <span class="platform-mini-badge" :class="item.platform">
                <v-icon :icon="item.platform === 'whatsapp' ? 'mdi-whatsapp' : 'mdi-facebook-messenger'" size="9"/>
              </span>
            </div>
            <div class="conv-details">
              <div class="conv-top-row">
                <span class="conv-name" :title="item.senderName">{{ item.senderName }}</span>
                <span class="conv-time">{{ formatPancakeTime(item.lastTime) }}</span>
              </div>
              <div class="conv-mid-row">
                <span
                  class="status-pill-badge"
                  :style="{
                    color: getStatusInfo(item.status).color,
                    backgroundColor: getStatusInfo(item.status).bg,
                    borderColor: getStatusInfo(item.status).border
                  }"
                >
                  <v-icon :icon="getStatusInfo(item.status).icon" size="10" class="mr-1"/>
                  {{ l(getStatusInfo(item.status).label) }}
                </span>
                <span class="conv-preview" :title="item.lastMessage?.message || ''">
                  {{ item.lastMessage?.message || (item.lastMessage?.type === 'image' ? (locale === 'lo' ? '📷 ຮູບພາບ' : '📷 รูปภาพ') : '...') }}
                </span>
              </div>
              <div class="conv-meta" v-if="item.lead">
                <span class="meta-tag page-tag" v-if="item.pageId"><v-icon icon="mdi-flag" size="10"/> {{ item.pageId }}</span>
                <span class="meta-tag ad-tag" v-if="item.lead.sourceType === 'ad' && item.lead.sourceId"><v-icon icon="mdi-bullhorn" size="10"/> Ad: {{ item.lead.sourceId }}</span>
              </div>
            </div>
          </div>
        </ClientOnly>
      </div>
    </div>

    <div class="chat-main" v-if="activeSender">
      <div class="chat-header">
        <div class="chat-header-info">
          <button class="mobile-chat-back" @click="goBackToList()" :title="locale === 'lo' ? 'ກັບຄືນ' : 'Back'">
            <v-icon icon="mdi-arrow-left" size="22"/>
          </button>
          <div class="chat-header-user">
            <h3>{{ activeSenderSummary?.senderName || getSenderName(conversations[activeSender]) }}</h3>
            <div class="chat-header-sub">
              <span class="platform-badge" :class="activeSenderSummary?.platform">
                {{ activeSenderSummary?.platform }}
              </span>
              <span class="header-page-id" v-if="activeSenderSummary?.pageId">
                <v-icon icon="mdi-flag" size="12"/> {{ activeSenderSummary.pageId }}
              </span>
            </div>
          </div>
        </div>
        <div class="chat-header-actions" v-if="activeSender">
          <!-- Pancake Status Menu Dropdown -->
          <v-menu location="bottom end">
            <template #activator="{ props }">
              <button
                type="button"
                v-bind="props"
                class="current-status-btn"
                :style="{
                  color: getStatusInfo(activeLeadStatus).color,
                  backgroundColor: getStatusInfo(activeLeadStatus).bg,
                  borderColor: getStatusInfo(activeLeadStatus).border
                }"
              >
                <v-icon :icon="getStatusInfo(activeLeadStatus).icon" size="15" class="mr-1"/>
                <span>{{ l(getStatusInfo(activeLeadStatus).label) }}</span>
                <v-icon icon="mdi-chevron-down" size="15" class="ml-1"/>
              </button>
            </template>
            <v-list density="compact" class="pancake-status-menu">
              <v-list-subheader class="status-menu-header">
                {{ l({lo:'ປ່ຽນສະຖານະລູກຄ້າ',th:'เปลี่ยนสถานะลูกค้า',en:'Change Status'}) }}
              </v-list-subheader>
              <v-list-item
                v-for="opt in STATUS_OPTIONS"
                :key="opt.key"
                :value="opt.key"
                class="status-menu-item"
                :class="{ active: activeLeadStatus === opt.key }"
                @click="changeLeadStatus(activeSender, opt.key)"
              >
                <template #prepend>
                  <span class="status-dot-menu" :style="{ backgroundColor: opt.color }"></span>
                </template>
                <v-list-item-title :style="{ color: opt.color, fontWeight: activeLeadStatus === opt.key ? 750 : 500 }">
                  {{ l(opt.label) }}
                </v-list-item-title>
                <template #append v-if="activeLeadStatus === opt.key">
                  <v-icon icon="mdi-check" size="16" :color="opt.color"/>
                </template>
              </v-list-item>
            </v-list>
          </v-menu>

          <!-- 1-Click Purchase Shortcut -->
          <button
            v-if="activeLeadStatus !== 'purchased'"
            @click="changeLeadStatus(activeSender, 'purchased')"
            class="action-btn-purchased"
            :title="l({lo:'ປິດການຂາຍທັນທີ (ສົ່ງ Meta CAPI)',th:'ปิดการขายทันที (ส่ง Meta CAPI)',en:'Mark Purchased (Meta CAPI)'})"
          >
            <v-icon icon="mdi-lightning-bolt" size="14" class="mr-1"/>
            <span>{{ locale === 'lo' ? 'ປິດການຂາຍ' : (locale === 'th' ? 'ปิดการขาย' : 'Close Deal') }}</span>
          </button>
          <span v-else class="status-badge purchased-pill">
            <v-icon icon="mdi-check-circle" size="14" class="mr-1"/>
            {{ locale === 'lo' ? 'ປິດການຂາຍແລ້ວ' : (locale === 'th' ? 'ซื้อแล้ว' : 'Purchased') }}
          </span>
        </div>
      </div>
      <div v-if="statusNotice" class="status-notice-bar">
        <v-icon icon="mdi-check-circle" size="15"/>
        <span>{{ statusNotice }}</span>
      </div>

      <div class="chat-messages" ref="messagesContainer">
        <ClientOnly>
          <div
            v-for="msg in activeChatMessages"
            :key="msg.id"
            class="message-bubble"
            :class="msg.platform === 'system' ? 'outgoing' : 'incoming'"
          >
            <div class="bubble-content">
              <img v-if="msg.type === 'image' && msg.message" :src="msg.message" :alt="locale==='lo'?'ຮູບຈາກການສົນທະນາ':'Conversation image'" class="message-image">
              <video v-else-if="msg.type === 'video' && msg.message" :src="msg.message" controls class="message-video"/>
              <audio v-else-if="msg.type === 'audio' && msg.message" :src="msg.message" controls class="message-audio"/>
              <span v-else-if="msg.type === 'image'" class="media-placeholder">{{locale==='lo'?'ຮູບພາບ':'Image'}}</span>
              <span v-else>{{ msg.message }}</span>
            </div>
            <div class="bubble-time">{{ formatTime(msg.timestamp) }}</div>
          </div>
        </ClientOnly>
      </div>

      <div class="chat-input-wrapper">
        <!-- Quick 1-Tap Codes Bar: Specially optimized for iPhone 11 and Mobile -->
        <div class="quick-chips-bar" v-if="availableCodes.length">
          <div class="quick-chips-header">
            <span class="quick-chips-title">
              <v-icon icon="mdi-flash" size="14" color="primary"/>
              {{ l({lo:'ສົ່ງດ່ວນ (1-Tap):',th:'ส่งด่วน (1-Tap):',en:'Quick Codes:'}) }}
            </span>
            <button type="button" class="more-codes-link" @click="replySetsDialog = true">
              {{ l({lo:'ທັງໝົດ',th:'ทั้งหมด',en:'All'}) }}
              <v-icon icon="mdi-chevron-right" size="14"/>
            </button>
          </div>
          <div class="quick-chips-scroll">
            <button
              v-for="item in availableCodes.slice(0, 12)"
              :key="item.code"
              type="button"
              class="quick-tap-chip"
              @click="selectCode(item, true)"
              :title="item.details"
            >
              <span class="chip-badge">#{{ item.code }}</span>
              <span class="chip-text">{{ item.name }}</span>
            </button>
          </div>
        </div>

        <div v-if="matchingCodes.length" class="code-suggestions">
          <span class="suggestion-label"><v-icon icon="mdi-flash" size="14"/> {{ l({lo:'ໂຄ້ດທີ່ກົງກັນ:',th:'โค้ดที่ตรงกัน:',en:'Matching codes:'}) }}</span>
          <button
            v-for="codeItem in matchingCodes"
            :key="codeItem.code"
            type="button"
            class="suggestion-chip"
            @click="selectCode(codeItem, false)"
            :title="codeItem.details"
          >
            <strong>#{{ codeItem.code }}</strong>
            <small>{{ codeItem.name }}</small>
          </button>
        </div>

        <div v-if="sendSuccessNotice" class="chat-notice-success">
          <v-icon icon="mdi-check-circle" size="16"/>
          <span>{{ sendSuccessNotice }}</span>
        </div>

        <div class="chat-input">
          <div v-if="loadError" class="chat-error">{{loadError}}</div>
          <button
            type="button"
            class="quick-reply-btn"
            @click="replySetsDialog = true"
            :title="l({lo:'ເລືອກຊຸດຂໍ້ຄວາມ / ໂຄ້ດ',th:'เลือกชุดข้อความ / โค้ด',en:'Quick Reply Sets & Codes'})"
          >
            <v-icon icon="mdi-lightning-bolt" size="18"/>
            <span class="quick-btn-text">{{ l({lo:'ໂຄ້ດ',th:'โค้ด',en:'Code'}) }}</span>
          </button>
          <input
            v-model="inputMessage"
            @keyup.enter="sendReply()"
            type="text"
            :placeholder="l({
              lo: 'ພິມຂໍ້ຄວາມ ຫຼື ໂຄ້ດຊຸດຂໍ້ຄວາມ (#P01, ຊຸດ1)...',
              th: 'พิมพ์ข้อความ หรือ โค้ดชุดข้อความ (#P01, ชุด1)...',
              en: 'Write a message or code (#P01, Set1)...'
            })"
            :disabled="sending"
          />
          <button :disabled="sending||!inputMessage.trim()" @click="sendReply()">
            <v-progress-circular v-if="sending" indeterminate size="18" width="2"/>
            <v-icon v-else icon="mdi-send"/>
          </button>
        </div>
      </div>
    </div>

    <div class="chat-empty" v-else>
      <v-icon icon="mdi-message-text-outline" size="64" />
      <p>{{locale==='lo'?'ເລືອກການສົນທະນາເພື່ອເບິ່ງ ແລະຕອບຂໍ້ຄວາມ':'Select a conversation to view and reply.'}}</p>
    </div>

    <!-- Quick Reply Sets / Codes Dialog -->
    <v-dialog v-model="replySetsDialog" max-width="520">
      <v-card class="reply-sets-dialog-card">
        <v-card-title class="dialog-title">
          <v-icon icon="mdi-lightning-bolt" color="primary" class="mr-2"/>
          <span>{{ l({lo:'ຊຸດຂໍ້ຄວາມ ແລະ ໂຄ້ດສິນຄ້າ',th:'ชุดข้อความ และ โค้ดสินค้า',en:'Reply Sets & Product Codes'}) }}</span>
        </v-card-title>
        <v-card-text class="dialog-body">
          <p class="dialog-hint">
            {{ l({
              lo:'ພຽງແຕ່ພິມຊື່ ຫຼື ໂຄ້ດໃນຊ່ອງແຊັດ (ເຊັ່ນ #P01 ຫຼື ຊຸດ1) ລະບົບກໍຈະສົ່ງຊຸດຂໍ້ຄວາມໃຫ້ລູກຄ້າທັນທີ, ຫຼືກົດເລືອກຈາກລາຍການລຸ່ມນີ້:',
              th:'เพียงพิมพ์ชื่อหรือโค้ดในช่องแชท (เช่น #P01 หรือ ชุด1) ระบบจะส่งชุดข้อความให้ลูกค้าทันที หรือกดเลือกจากรายการด้านล่าง:',
              en:'Type the name or code in chat (e.g. #P01 or Set1) to send the sequence, or select from the list below:'
            }) }}
          </p>
          <v-text-field
            v-model="dialogSearch"
            density="compact"
            variant="outlined"
            prepend-inner-icon="mdi-magnify"
            :placeholder="l({lo:'ຄົ້ນຫາຊື່ ຫຼື ໂຄ້ດ...',th:'ค้นหาชื่อ หรือ โค้ด...',en:'Search name or code...'})"
            hide-details
            class="mb-3"
          />
          <div v-if="filteredDialogCodes.length" class="codes-list">
            <div v-for="item in filteredDialogCodes" :key="item.code" class="code-card">
              <div class="code-info">
                <div class="code-badge">#{{ item.code }}</div>
                <strong>{{ item.name }}</strong>
                <small>{{ item.details }}</small>
              </div>
              <div class="code-actions">
                <button type="button" class="btn-insert" @click="selectCode(item, false); replySetsDialog = false">
                  {{ l({lo:'ໃສ່ໂຄ້ດ',th:'ใส่โค้ด',en:'Insert'}) }}
                </button>
                <button type="button" class="btn-send-now" @click="selectCode(item, true); replySetsDialog = false">
                  {{ l({lo:'ສົ່ງເລີຍ',th:'ส่งเลย',en:'Send now'}) }}
                </button>
              </div>
            </div>
          </div>
          <div v-else class="empty-codes">
            <p>{{ l({lo:'ບໍ່ພົບຊຸດຂໍ້ຄວາມ ຫຼື ໂຄ້ດທີ່ກົງກັນ',th:'ไม่พบชุดข้อความหรือโค้ดที่ตรงกัน',en:'No matching Reply Sets or Codes found.'}) }}</p>
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer/>
          <v-btn variant="text" @click="replySetsDialog = false">{{ l({lo:'ປິດ',th:'ปิด',en:'Close'}) }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style scoped>
.chat-layout {
  display: flex;
  height: calc(100vh - 180px);
  max-height: calc(100dvh - 180px);
  min-height: 520px;
  background: var(--color-surface);
  border-radius: 12px;
  border: 1px solid var(--color-border);
  overflow: hidden;
}

.chat-sidebar {
  width: 340px;
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
}

.sidebar-header {
  padding: 16px 14px 12px;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 10px;
  background: var(--color-surface);
}

.sidebar-header-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.sidebar-header h2 {
  margin: 0;
  font-size: 17px;
  font-weight: 750;
}
.live-state{display:flex;align-items:center;gap:6px;color:var(--color-text-muted);font-size:11px;font-weight:600}.live-state i{width:7px;height:7px;border-radius:50%;background:#f59e0b}.live-state.online{color:#15803d}.live-state.online i{background:#22c55e;box-shadow:0 0 0 3px rgba(34,197,94,.16)}

.sidebar-search {
  position: relative;
  display: flex;
  align-items: center;
}
.sidebar-search input {
  width: 100%;
  height: 36px;
  border: 1px solid var(--color-border);
  border-radius: 18px;
  padding: 0 32px 0 34px;
  font-size: 13px;
  background: var(--color-background);
  color: var(--color-text);
  outline: none;
  transition: border-color 0.15s ease;
}
.sidebar-search input:focus {
  border-color: var(--color-primary);
}
.sidebar-search .search-icon {
  position: absolute;
  left: 10px;
  color: var(--color-text-muted);
  pointer-events: none;
}
.clear-search-btn {
  position: absolute;
  right: 8px;
  background: transparent;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  display: grid;
  place-items: center;
  padding: 2px;
  border-radius: 50%;
}

.status-filters {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  white-space: nowrap;
  padding-bottom: 2px;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}
.status-filters::-webkit-scrollbar {
  display: none;
}

.filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px;
  border-radius: 14px;
  font-size: 11px;
  font-weight: 700;
  border: 1px solid var(--color-border);
  background: var(--color-background);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}
.filter-pill:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.filter-pill.active {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: #fff !important;
}
.filter-count {
  padding: 1px 5px;
  border-radius: 10px;
  background: rgba(0,0,0,0.06);
  font-size: 10px;
  font-weight: 750;
}
.filter-pill.active .filter-count {
  background: rgba(255,255,255,0.25);
  color: #fff;
}
.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  display: inline-block;
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
}

.empty-list {
  padding: 30px 20px;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 13px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.conv-item {
  display: flex;
  padding: 12px 14px;
  gap: 12px;
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
  transition: all 0.15s ease;
  position: relative;
}

.conv-item:hover {
  background: var(--color-background);
}

.conv-item.active {
  background: var(--color-primary-soft);
  border-left: 3px solid var(--color-primary);
}

.conv-avatar {
  position: relative;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: var(--color-surface-soft);
  display: grid;
  place-items: center;
  overflow: visible;
  flex-shrink: 0;
}
.avatar-img {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
}
.platform-mini-badge {
  position: absolute;
  bottom: -2px;
  right: -2px;
  width: 17px;
  height: 17px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: #fff;
  border: 1px solid var(--color-border);
  box-shadow: 0 1px 3px rgba(0,0,0,0.12);
}
.platform-mini-badge.facebook { color: #1877F2; }
.platform-mini-badge.whatsapp { color: #25D366; }

.conv-details {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.conv-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.conv-name {
  font-weight: 700;
  font-size: 13.5px;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.conv-time {
  font-size: 11px;
  color: var(--color-text-muted);
  flex-shrink: 0;
  font-weight: 500;
}

.conv-mid-row {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.status-pill-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 7px;
  border-radius: 10px;
  font-size: 10px;
  font-weight: 750;
  border: 1px solid transparent;
  flex-shrink: 0;
  line-height: 1.2;
}

.conv-preview {
  font-size: 12.5px;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
}


.chat-header {
  padding: 12px 18px;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--color-surface);
  gap: 12px;
}

.chat-header-info {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.chat-header-user {
  min-width: 0;
}

.chat-header-user h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 750;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chat-header-sub {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 3px;
}

.header-page-id {
  font-size: 11px;
  color: var(--color-text-muted);
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.platform-badge {
  font-size: 10px;
  padding: 2px 7px;
  border-radius: 10px;
  font-weight: 700;
  text-transform: uppercase;
}
.platform-badge.facebook { background: #E7F3FF; color: #1877F2; }
.platform-badge.whatsapp { background: #E7FCE3; color: #25D366; }

.chat-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.current-status-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 750;
  border: 1px solid transparent;
  cursor: pointer;
  transition: all 0.15s ease;
}
.current-status-btn:hover {
  opacity: 0.88;
  box-shadow: 0 2px 6px rgba(0,0,0,0.08);
}

.action-btn-purchased {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: #16a34a;
  color: white;
  border: none;
  padding: 6px 14px;
  border-radius: 20px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 750;
  transition: background 0.15s ease;
}
.action-btn-purchased:hover {
  background: #15803d;
}

.purchased-pill {
  display: inline-flex;
  align-items: center;
  padding: 5px 12px;
  border-radius: 20px;
  background: #dcfce7;
  color: #166534;
  border: 1px solid #bbf7d0;
  font-size: 12px;
  font-weight: 750;
}

.status-notice-bar {
  padding: 7px 18px;
  background: #eff6ff;
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 650;
  display: flex;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid #dbeafe;
}

.status-dot-menu {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  display: inline-block;
  margin-right: 8px;
}

.pancake-status-menu {
  min-width: 190px;
  border-radius: 12px !important;
  padding: 6px !important;
}

.status-menu-header {
  font-size: 11px !important;
  font-weight: 750 !important;
  color: var(--color-text-muted) !important;
  padding: 4px 12px !important;
}

.status-menu-item {
  border-radius: 8px !important;
  margin-bottom: 2px !important;
  cursor: pointer !important;
  padding: 6px 12px !important;
}
.status-menu-item:hover {
  background: var(--color-background) !important;
}
.status-menu-item.active {
  background: var(--color-primary-soft) !important;
}

.chat-messages {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
  background: var(--color-background);
}

.message-bubble {
  max-width: 70%;
  display: flex;
  flex-direction: column;
}

.message-bubble.incoming {
  align-self: flex-start;
}

.message-bubble.outgoing {
  align-self: flex-end;
  align-items: flex-end;
}

.bubble-content {
  padding: 12px 16px;
  border-radius: 16px;
  font-size: 14px;
  line-height: 1.4;
}
.message-image,.message-video{display:block;max-width:min(360px,60vw);max-height:360px;border-radius:10px;object-fit:contain}.message-audio{display:block;width:min(360px,60vw);max-width:100%}.media-placeholder{font-weight:600}

.incoming .bubble-content {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-bottom-left-radius: 4px;
}

.outgoing .bubble-content {
  background: var(--color-primary);
  color: #fff;
  border-bottom-right-radius: 4px;
}

.bubble-time {
  font-size: 11px;
  color: var(--color-text-muted);
  margin-top: 4px;
}

.chat-input-wrapper {
  position: relative;
  border-top: 1px solid var(--color-border);
  background: var(--color-surface);
}

.quick-chips-bar {
  padding: 8px 14px 6px;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.quick-chips-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.quick-chips-title {
  font-size: 11px;
  font-weight: 750;
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}
.more-codes-link {
  background: none;
  border: none;
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 0;
}
.quick-chips-scroll {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  padding-bottom: 4px;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
}
.quick-chips-scroll::-webkit-scrollbar {
  display: none;
}
.quick-tap-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 18px;
  border: 1px solid var(--color-border);
  background: var(--color-background);
  color: var(--color-text);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
  flex-shrink: 0;
  transition: all 0.15s ease;
  touch-action: manipulation;
}
.quick-tap-chip:active, .quick-tap-chip:hover {
  border-color: var(--color-primary);
  background: var(--color-primary-soft);
}
.quick-tap-chip .chip-badge {
  color: var(--color-primary);
  font-weight: 800;
  font-size: 11.5px;
}
.quick-tap-chip .chip-text {
  font-weight: 600;
  color: var(--color-text);
}

.code-suggestions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: var(--color-surface-soft);
  border-bottom: 1px solid var(--color-border-subtle);
  overflow-x: auto;
  white-space: nowrap;
  -webkit-overflow-scrolling: touch;
}

.suggestion-label {
  font-size: 11px;
  font-weight: 750;
  color: var(--color-primary);
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.suggestion-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: 14px;
  font-size: 12px;
  color: var(--color-text);
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}
.suggestion-chip:hover {
  background: var(--color-primary-soft);
  border-color: var(--color-primary);
}
.suggestion-chip strong {
  color: var(--color-primary);
  font-weight: 750;
}
.suggestion-chip small {
  color: var(--color-text-secondary);
  font-size: 11px;
}

.chat-notice-success {
  padding: 8px 16px;
  background: #dcfce7;
  color: #166534;
  font-size: 12px;
  font-weight: 650;
  display: flex;
  align-items: center;
  gap: 8px;
  border-bottom: 1px solid #bbf7d0;
}

.quick-reply-btn {
  height: 44px;
  padding: 0 12px;
  border-radius: 22px;
  border: 1px solid var(--color-border);
  background: var(--color-surface-soft);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 750;
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}
.quick-reply-btn:hover {
  background: var(--color-primary-soft);
  border-color: var(--color-primary);
}

.reply-sets-dialog-card {
  border-radius: 18px !important;
}
.dialog-title {
  display: flex;
  align-items: center;
  font-size: 18px;
  font-weight: 750;
  padding: 20px 24px 12px;
}
.dialog-body {
  padding: 0 24px 16px;
}
.dialog-hint {
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.5;
  margin-bottom: 12px;
}
.codes-list {
  display: grid;
  gap: 10px;
  max-height: 380px;
  overflow-y: auto;
  padding-right: 4px;
}
.code-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-background);
}
.code-info {
  display: grid;
  gap: 2px;
  min-width: 0;
}
.code-badge {
  display: inline-block;
  padding: 2px 7px;
  background: var(--color-primary-soft);
  color: var(--color-primary);
  border-radius: 6px;
  font-size: 11px;
  font-weight: 800;
  width: fit-content;
}
.code-info strong {
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.code-info small {
  font-size: 11px;
  color: var(--color-text-secondary);
}
.code-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.btn-insert, .btn-send-now {
  height: 32px;
  padding: 0 10px;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 700;
  cursor: pointer;
}
.btn-insert {
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text);
}
.btn-insert:hover {
  background: var(--color-surface-hover);
}
.btn-send-now {
  border: none;
  background: var(--color-primary);
  color: white;
}
.btn-send-now:hover {
  background: var(--color-primary-dark);
}
.empty-codes {
  padding: 30px 10px;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 13px;
}

.chat-input {
  position: relative;
  padding: 14px 16px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.chat-error{position:absolute;left:16px;right:16px;bottom:calc(100% + 8px);padding:8px 12px;color:var(--color-error);background:var(--color-error-soft);border:1px solid color-mix(in srgb,var(--color-error) 30%,transparent);border-radius:9px;font-size:12px;z-index:10}

.chat-input input {
  flex: 1;
  height: 44px;
  border: 1px solid var(--color-border);
  border-radius: 22px;
  padding: 0 20px;
  font-size: 14px;
  outline: none;
}

.chat-input input:focus {
  border-color: var(--color-primary);
}

.chat-input button {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: none;
  background: var(--color-primary);
  color: #fff;
  cursor: pointer;
  display: grid;
  place-items: center;
}
.chat-input button:disabled{opacity:.45;cursor:not-allowed}

.chat-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  gap: 16px;
}
.mobile-chat-back {
  display: none;
}
@media(max-width:860px){
  .chat-layout{height:calc(100dvh - 120px)}
  .chat-sidebar{width:270px}
  .sidebar-header{padding:12px 10px}
  .sidebar-header h2{font-size:15px}
  .conv-item{padding:10px 8px}
  .conv-name{font-size:12.5px}
  .conv-preview{font-size:11.5px}
  .chat-header,.chat-messages{padding:12px}
  .message-bubble{max-width:88%}
}
@media(max-width:640px){
  .chat-layout{
    height: calc(100dvh - 125px);
    border-radius: 12px;
  }
  .chat-layout.mobile-in-chat {
    height: 100dvh;
    border-radius: 0;
    border: none;
    margin: -20px -14px calc(-102px - env(safe-area-inset-bottom)) -14px;
    position: relative;
    z-index: 90;
  }
  .chat-sidebar{width:100%!important;border-right:none}
  .chat-sidebar.mobile-hidden{display:none!important}
  .chat-main{width:100%!important;height:100%;display:flex;flex-direction:column}
  .chat-empty{display:none!important}
  .mobile-chat-back{
    display: inline-grid;
    place-items: center;
    width: 44px;
    height: 44px;
    border: none;
    border-radius: 12px;
    background: var(--color-background);
    color: var(--color-text);
    cursor: pointer;
    margin-right: 8px;
    flex-shrink: 0;
    touch-action: manipulation;
  }
  .mobile-chat-back:active {
    background: var(--color-primary-soft);
    color: var(--color-primary);
  }
  .chat-header{
    padding: max(10px, env(safe-area-inset-top)) 12px 10px;
    flex-wrap: wrap;
    gap: 8px;
    min-height: 54px;
  }
  .chat-header-actions{gap:6px}
  .current-status-btn{
    padding: 7px 11px;
    font-size: 11.5px;
    min-height: 40px;
    border-radius: 10px;
    touch-action: manipulation;
  }
  .action-btn-purchased{
    padding: 7px 11px;
    font-size: 11.5px;
    min-height: 40px;
    border-radius: 10px;
    touch-action: manipulation;
  }
  .purchased-pill{padding:5px 10px;font-size:11.5px}
  .chat-messages{
    padding: 12px 10px;
    -webkit-overflow-scrolling: touch;
  }
  .message-bubble {
    max-width: 90%;
  }
  .chat-input-wrapper {
    padding-bottom: max(12px, env(safe-area-inset-bottom));
    background: var(--color-surface);
  }
  .chat-input {
    padding: 8px 10px;
    gap: 8px;
  }
  .chat-input input {
    height: 44px;
    font-size: 15px;
  }
  .chat-input button {
    width: 44px;
    height: 44px;
    touch-action: manipulation;
  }
  .quick-reply-btn {
    height: 44px;
    padding: 0 10px;
    touch-action: manipulation;
  }
}

.status-badge {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 4px;
  margin-left: 6px;
  background: #f0f0f0;
  color: #555;
  text-transform: capitalize;
  display: inline-block;
  vertical-align: middle;
}
.status-badge.new { background: #E7F3FF; color: #1877F2; }
.status-badge.purchased { background: #E7FCE3; color: #25D366; }

.conv-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}
.meta-tag {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: 12px;
  background: var(--color-background);
  color: var(--color-text-muted);
  display: flex;
  align-items: center;
  gap: 2px;
}
.ad-tag { background: #FFF3E0; color: #E65100; }
.page-tag { background: #F3E5F5; color: #7B1FA2; }

</style>
