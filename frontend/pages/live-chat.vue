<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import type { ItemsResponse, Order, Product, ReplySet, Warehouse } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

interface Lead {
  id: string
  accountId: string
  pageId: string
  senderId: string
  sourceType: string
  sourceId: string
  productId?: string
  name?: string
  phone?: string
  address?: string
  notes?: string
  tags?: string
  status: string
  assignedUserId?: string
  alertRead?: boolean
  createdAt?: string
  updatedAt?: string
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
const { user } = useAuth()
const liveConnected = ref(false)
let pollTimer: ReturnType<typeof setInterval> | null = null
const messagesContainer = ref<HTMLElement | null>(null)

// Reply Sets and Product Codes for Quick Auto-reply by Admin
const { data: replySetsData } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const { data: productsData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: warehouseData } = await useApi<ItemsResponse<Warehouse>>('/proxy/api/v1/warehouses').catch(() => ({ value: { items: [] } } as any))
const { data: teamData } = await useApi<any[]>('/proxy/api/v1/team').catch(() => ({ value: [] } as any))
const replySets = computed(() => replySetsData.value || [])
const products = computed(() => productsData.value?.items || [])
const warehouses = computed(() => warehouseData.value?.items || [])
const teamMembers = computed(() => (teamData?.value || []).map((m: any) => ({
  id: m.userId || m.id,
  name: m.user?.name || m.name || m.user?.email || m.email || 'Staff',
  email: m.user?.email || m.email || '',
  role: m.role || 'member'
})))

const replySetsDialog = ref(false)
const dialogSearch = ref('')
const sendSuccessNotice = ref('')

// Pancake CRM States
const pancakeDrawerOpen = ref(false)
const selectedStaffFilter = ref<'all' | 'mine' | 'unassigned'>('all')
const cannedPhrasesDialog = ref(false)
const quickOrderDialog = ref(false)
const quickOrderForm = reactive({
  warehouseId: '',
  productId: '',
  quantity: 1,
  shippingFee: 0,
  deliveryCourier: 'Anousith Express',
  deliveryTracking: '',
  notes: '',
})
const copiedField = ref('')

interface CannedPhrase {
  id: string
  title: { lo: string; th: string; en: string }
  text: { lo: string; th: string; en: string }
  category: 'greeting' | 'product' | 'payment' | 'shipping' | 'closing'
  icon: string
}

const CANNED_PHRASES: CannedPhrase[] = [
  {
    id: 'greet_1',
    title: { lo: 'ທັກທາຍສຸພາບ', th: 'ทักทายสุภาพ', en: 'Polite Greeting' },
    text: {
      lo: 'ສະບາຍດີເຈົ້າ 🙏 ຍິນດີຕ້ອນຮັບສູ່ຮ້ານເຮົາເຈົ້າ! ສົນໃຈສິນຄ້າຕົວໃດສາມາດສອບຖາມໄດ້ເລີຍເດີເຈົ້າ 😊',
      th: 'สวัสดีค่ะ/ครับ 🙏 ยินดีต้อนรับนะคะ สนใจสินค้าตัวไหนสอบถามได้เลยนะคะ 😊',
      en: 'Hello! Welcome to our store. How may I assist you today?'
    },
    category: 'greeting',
    icon: 'mdi-hand-wave',
  },
  {
    id: 'stock_1',
    title: { lo: 'ແຈ້ງສິນຄ້າພ້ອມສົ່ງ', th: 'แจ้งสินค้าพร้อมส่ง', en: 'In Stock Notice' },
    text: {
      lo: 'ສິນຄ້າມີພ້ອມສົ່ງເລີຍເຈົ້າ ລູກຄ້າຮັບຈັກຊຸດດີເຈົ້າ? ສັ່ງມື້ນີ້ຈັດສົ່ງໃຫ້ທັນທີເລີຍເຈົ້າ ✨',
      th: 'สินค้ามีพร้อมส่งเลยค่ะ/ครับ ลูกค้ารับกี่ชิ้นดีคะ สั่งวันนี้จัดส่งให้ทันทีเลยค่ะ ✨',
      en: 'The item is currently in stock and ready to ship! How many would you like?'
    },
    category: 'product',
    icon: 'mdi-package-variant-closed-check',
  },
  {
    id: 'pay_1',
    title: { lo: 'ແຈ້ງເລກບັນຊີ (BCEL)', th: 'แจ้งเลขบัญชีโอนเงิน', en: 'Payment Details' },
    text: {
      lo: 'ເລກບັນຊີຊຳລະເງິນເຈົ້າ:\n🏦 ທະນາຄານການຄ້າ (BCEL One)\nເລກບັນຊີ: 160-xx-xxxxxx\nຊື່ບັນຊີ: META STORE\nໂອນແລ້ວສົ່ງສະລິບຢືນຢັນໄດ້ເລີຍເດີເຈົ້າ 💳',
      th: 'ช่องทางการชำระเงินนะคะ:\n🏦 ธนาคารกสิกรไทย / พร้อมเพย์\nเลขบัญชี: xxx-x-xxxxx-x\nโอนแล้วแจ้งสลิปได้เลยนะคะ 💳',
      en: 'Payment Account:\nBank: BCEL One / Mobile Banking\nAccount No: 160-xx-xxxxxx\nPlease share your payment slip once transferred. Thank you!'
    },
    category: 'payment',
    icon: 'mdi-credit-card-outline',
  },
  {
    id: 'ship_1',
    title: { lo: 'ຂໍທີ່ຢູ່ຈັດສົ່ງ', th: 'ขอที่อยู่จัดส่ง', en: 'Request Address' },
    text: {
      lo: 'ລົບກວນລູກຄ້າແຈ້ງ: ຊື່, ເບີໂທລະສັບຕິດຕໍ່ ແລະ ທີ່ຢູ່ຈັດສົ່ງທີ່ສະດວກໃຫ້ທາງເຮົານຳເດີເຈົ້າ 📦🚚',
      th: 'รบกวนลูกค้าแจ้งชื่อ, เบอร์โทร และที่อยู่จัดส่งให้ทางเราด้วยนะคะ 📦🚚',
      en: 'Please provide your full name, phone number, and delivery address. 📦'
    },
    category: 'shipping',
    icon: 'mdi-map-marker-outline',
  },
  {
    id: 'close_1',
    title: { lo: 'ຂອບໃຈ & ປິດການຂາຍ', th: 'ขอบคุณปิดการขาย', en: 'Thank you & Close' },
    text: {
      lo: 'ທາງຮ້ານໄດ້ຮັບອໍເດີ ແລະ ດຳເນີນການຈັດກຽມເຄື່ອງຮຽບຮ້ອຍແລ້ວເຈົ້າ ຂອບໃຈຫຼາຍໆທີ່ໄວ້ວາງໃຈອຸດໜູນຮ້ານເຮົາເດີເຈົ້າ ❤️',
      th: 'ทางร้านได้รับยอดและเตรียมจัดส่งสินค้าเรียบร้อยแล้วนะคะ ขอบพระคุณมากค่ะ ❤️',
      en: 'Your order is confirmed and being prepared for shipment. Thank you so much for choosing us! ❤️'
    },
    category: 'closing',
    icon: 'mdi-heart-outline',
  },
]

const PRESET_TAGS = [
  { key: 'VIP', label: 'VIP', color: '#b45309', bg: '#fef3c7', border: '#fde68a' },
  { key: 'ລູກຄ້າເກົ່າ', label: 'ລູກຄ້າເກົ່າ', color: '#1d4ed8', bg: '#dbeafe', border: '#bfdbfe' },
  { key: 'ລໍຖ້າຈັດສົ່ງ', label: 'ລໍຖ້າຈັດສົ່ງ', color: '#c2410c', bg: '#ffedd5', border: '#fed7aa' },
  { key: 'ໂອນແລ້ວ', label: 'ໂອນແລ້ວ', color: '#15803d', bg: '#dcfce7', border: '#bbf7d0' },
  { key: 'ສອບຖາມລາຄາ', label: 'ສອບຖາມລາຄາ', color: '#6d28d9', bg: '#ede9fe', border: '#ddd6fe' },
  { key: 'ສົ່ງດ່ວນ', label: 'ສົ່ງດ່ວນ', color: '#b91c1c', bg: '#fee2e2', border: '#fecaca' },
]

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
        status: newStatus,
        updatedAt: new Date().toISOString()
      }
    } else {
      leads.value[senderId].status = newStatus
      leads.value[senderId].updatedAt = new Date().toISOString()
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

const updateLeadField = async (senderId: string, fields: Partial<Lead>) => {
  if (!senderId) return
  try {
    const res = await $fetch<{ status: string; lead: Lead }>(`/api/proxy/api/v1/leads/${senderId}/details`, {
      method: 'PUT',
      body: fields
    })
    if (res.lead) {
      leads.value[senderId] = { ...leads.value[senderId], ...res.lead, updatedAt: new Date().toISOString() }
    } else if (leads.value[senderId]) {
      leads.value[senderId] = { ...leads.value[senderId], ...fields, updatedAt: new Date().toISOString() }
    }
    statusNotice.value = l({ lo: 'ບັນທຶກຂໍ້ມູນສຳເລັດ', th: 'บันทึกข้อมูลเรียบร้อย', en: 'Saved successfully' })
    setTimeout(() => { statusNotice.value = '' }, 3000)
  } catch (err: any) {
    loadError.value = l({ lo: 'ບັນທຶກຂໍ້ມູນບໍ່ສຳເລັດ', th: 'บันทึกข้อมูลไม่สำเร็จ', en: 'Failed to save details' })
  }
}

const toggleCustomerTag = async (senderId: string, tagKey: string) => {
  const lead = leads.value[senderId]
  const currentTags = (lead?.tags || '').split(',').map(t => t.trim()).filter(Boolean)
  let nextTags: string[]
  if (currentTags.includes(tagKey)) {
    nextTags = currentTags.filter(t => t !== tagKey)
  } else {
    nextTags = [...currentTags, tagKey]
  }
  const tagStr = nextTags.join(',')
  if (leads.value[senderId]) {
    leads.value[senderId].tags = tagStr
  }
  await updateLeadField(senderId, { tags: tagStr })
}

const insertPhrase = (phrase: CannedPhrase, autoSend = false) => {
  const text = l(phrase.text)
  if (autoSend) {
    inputMessage.value = text
    sendReply()
  } else {
    inputMessage.value = inputMessage.value ? `${inputMessage.value}\n${text}` : text
  }
  cannedPhrasesDialog.value = false
}

const copyToClipboard = async (text: string, label: string) => {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    copiedField.value = label
    setTimeout(() => { copiedField.value = '' }, 2500)
  } catch {}
}

const markAsPurchased = (senderId: string) => changeLeadStatus(senderId, 'purchased')

const leadForm = reactive({
  name: '',
  phone: '',
  address: '',
  notes: '',
  assignedUserId: '',
})

const saveLeadInfo = async () => {
  if (!activeSender.value) return
  await updateLeadField(activeSender.value, {
    name: leadForm.name,
    phone: leadForm.phone,
    address: leadForm.address,
    notes: leadForm.notes,
    assignedUserId: leadForm.assignedUserId,
  })
}

const onAssignStaff = async (userId: string) => {
  leadForm.assignedUserId = userId
  if (activeSender.value) {
    await updateLeadField(activeSender.value, { assignedUserId: userId })
  }
}

const currentCustomerTags = computed(() => {
  if (!activeSender.value) return []
  return (leads.value[activeSender.value]?.tags || '').split(',').map(t => t.trim()).filter(Boolean)
})

const hasTag = (tagKey: string) => currentCustomerTags.value.includes(tagKey)

const CANNED_CATEGORIES = [
  { key: 'all', label: { lo: 'ທັງໝົດ', th: 'ทั้งหมด', en: 'All' }, icon: 'mdi-apps' },
  { key: 'greeting', label: { lo: 'ທັກທາຍ', th: 'ทักทาย', en: 'Greetings' }, icon: 'mdi-hand-wave' },
  { key: 'product', label: { lo: 'ສິນຄ້າ', th: 'สินค้า', en: 'Products' }, icon: 'mdi-package-variant-closed' },
  { key: 'payment', label: { lo: 'ການຊຳລະ', th: 'การชำระเงิน', en: 'Payment' }, icon: 'mdi-credit-card-outline' },
  { key: 'shipping', label: { lo: 'ຈັດສົ່ງ', th: 'จัดส่ง', en: 'Shipping' }, icon: 'mdi-truck-fast-outline' },
  { key: 'closing', label: { lo: 'ປິດການຂາຍ', th: 'ปิดการขาย', en: 'Closing' }, icon: 'mdi-check-decagram-outline' },
]

const selectedCannedCategory = ref<string>('all')
const filteredCannedPhrases = computed(() => {
  if (selectedCannedCategory.value === 'all') return CANNED_PHRASES
  return CANNED_PHRASES.filter(p => p.category === selectedCannedCategory.value)
})

const selectedOrderProduct = computed(() => products.value.find(p => p.id === quickOrderForm.productId))
const selectedOrderTotal = computed(() => {
  const price = Number(selectedOrderProduct.value?.salePrice ?? parseFloat(selectedOrderProduct.value?.price || '0') ?? 0)
  const sub = price * (quickOrderForm.quantity || 1)
  const total = sub + Number(quickOrderForm.shippingFee || 0)
  return total.toLocaleString()
})

watch(() => activeSender.value, (id) => {
  if (id) {
    const l = leads.value[id]
    leadForm.name = l?.name || activeSenderSummary.value?.senderName || ''
    leadForm.phone = l?.phone || ''
    leadForm.address = l?.address || ''
    leadForm.notes = l?.notes || ''
    leadForm.assignedUserId = l?.assignedUserId || ''
  } else {
    leadForm.name = ''
    leadForm.phone = ''
    leadForm.address = ''
    leadForm.notes = ''
    leadForm.assignedUserId = ''
  }
}, { immediate: true })

watch(() => leads.value[activeSender.value || ''], (l) => {
  if (l) {
    leadForm.phone = l.phone || ''
    leadForm.address = l.address || ''
    leadForm.notes = l.notes || ''
    leadForm.assignedUserId = l.assignedUserId || ''
    if (l.name) leadForm.name = l.name
  }
}, { deep: true })

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
  const map = new Map<string, ConversationSummary>()

  for (const [senderId, msgs] of Object.entries(conversations.value)) {
    const lastMsg = (msgs && msgs.length > 0) ? msgs[msgs.length - 1] : null
    const lead = leads.value[senderId]
    const lastMsgTime = parseTimestamp(lastMsg?.timestamp)
    const leadUpdatedTime = parseTimestamp(lead?.updatedAt)
    const leadCreatedTime = parseTimestamp(lead?.createdAt)
    const lastTime = Math.max(lastMsgTime, leadUpdatedTime, leadCreatedTime, 0)
    const status = (lead?.status || 'new').toLowerCase()
    const senderName = getSenderName(msgs) || lead?.name || `${l({lo:'ລູກຄ້າ',th:'ลูกค้า',en:'Customer'})} · ${senderId.slice(-4)}`
    const senderPic = getSenderPic(msgs)
    const platform = msgs?.[0]?.platform || (lead?.sourceType === 'whatsapp' ? 'whatsapp' : 'facebook')
    const pageId = msgs?.find(m => m.page_id)?.page_id || lead?.pageId || ''

    map.set(senderId, {
      senderId,
      messages: msgs || [],
      lastMessage: lastMsg || null,
      lastTime,
      lead,
      status,
      senderName,
      senderPic,
      platform,
      pageId,
    })
  }

  for (const [senderId, lead] of Object.entries(leads.value)) {
    if (!map.has(senderId)) {
      const lastTime = Math.max(parseTimestamp(lead.updatedAt), parseTimestamp(lead.createdAt), 0)
      const senderName = lead.name || `${l({lo:'ລູກຄ້າ',th:'ลูกค้า',en:'Customer'})} · ${senderId.slice(-4)}`
      const platform = lead.sourceType === 'whatsapp' ? 'whatsapp' : 'facebook'

      map.set(senderId, {
        senderId,
        messages: [],
        lastMessage: null,
        lastTime,
        lead,
        status: (lead.status || 'new').toLowerCase(),
        senderName,
        senderPic: undefined,
        platform,
        pageId: lead.pageId || '',
      })
    }
  }

  const list = Array.from(map.values())
  // Sort descending: Recent activity at the very top!
  list.sort((a, b) => b.lastTime - a.lastTime)
  return list
})

const filteredConversations = computed<ConversationSummary[]>(() => {
  let list = allConversationsList.value

  if (selectedStatusFilter.value !== 'all') {
    list = list.filter(c => c.status === selectedStatusFilter.value)
  }

  if (selectedStaffFilter.value === 'mine' && user.value?.id) {
    list = list.filter(c => c.lead?.assignedUserId === user.value?.id)
  } else if (selectedStaffFilter.value === 'unassigned') {
    list = list.filter(c => !c.lead?.assignedUserId)
  }

  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    list = list.filter(c => {
      const matchName = c.senderName.toLowerCase().includes(q)
      const matchId = c.senderId.toLowerCase().includes(q)
      const matchPage = c.pageId.toLowerCase().includes(q)
      const matchAd = (c.lead?.sourceId || '').toLowerCase().includes(q)
      const matchPhone = (c.lead?.phone || '').toLowerCase().includes(q)
      const matchText = (c.lastMessage?.message || '').toLowerCase().includes(q)
      const matchTag = (c.lead?.tags || '').toLowerCase().includes(q)
      return matchName || matchId || matchPage || matchAd || matchPhone || matchText || matchTag
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

const submittingOrder = ref(false)
const submitQuickOrder = async () => {
  if (!activeSender.value || !quickOrderForm.productId || submittingOrder.value) return
  submittingOrder.value = true
  try {
    const prod = products.value.find(p => p.id === quickOrderForm.productId)
    if (!prod) return
    const prodName = prod.name || 'Product'
    const unitPrice = Number(prod.salePrice ?? parseFloat(prod.price || '0') ?? 0)
    const qty = quickOrderForm.quantity || 1
    const subtotal = unitPrice * qty
    const shippingFee = Number(quickOrderForm.shippingFee || 0)
    const totalAmount = subtotal + shippingFee
    const currency = prod.currency || 'LAK'

    const senderId = activeSender.value
    const msgs = conversations.value[senderId]
    const pageId = msgs?.find(m => m.page_id)?.page_id || leads.value[senderId]?.pageId || ''
    const lead = leads.value[senderId]

    const defaultWh = warehouses.value.find((w: Warehouse) => w.isDefault) || warehouses.value[0]
    const whId = quickOrderForm.warehouseId || defaultWh?.id || ''

    const createdOrder = await $fetch<Order>('/api/proxy/api/v1/orders', {
      method: 'POST',
      body: {
        senderId,
        leadId: lead?.id || undefined,
        pageId: pageId || undefined,
        warehouseId: whId,
        customerName: leadForm.name || lead?.name || activeSenderSummary.value?.senderName || 'Customer',
        customerPhone: leadForm.phone || lead?.phone || '',
        customerAddress: leadForm.address || lead?.address || '',
        subtotal,
        shippingFee,
        totalAmount,
        currency,
        paymentStatus: 'paid',
        paymentMethod: 'BCEL One / Transfer',
        fulfillmentStatus: 'confirmed',
        deliveryCourier: quickOrderForm.deliveryCourier || 'Anousith Express',
        deliveryTracking: quickOrderForm.deliveryTracking || '',
        notes: quickOrderForm.notes,
        items: [
          {
            productId: prod.id,
            productCode: prod.code,
            productName: prodName,
            productImage: prod.imageUrl,
            quantity: qty,
            unitPrice,
          }
        ]
      }
    })

    const orderMemo = `[Order #${createdOrder.orderNumber}] ${prodName} x ${qty} (ລວມ: ${totalAmount.toLocaleString()} ${currency})`
    const updatedNotes = lead?.notes ? `${lead.notes}\n${orderMemo}` : orderMemo

    await updateLeadField(activeSender.value, {
      status: 'purchased',
      notes: updatedNotes,
      tags: lead?.tags ? (lead.tags.includes('ໂອນແລ້ວ') ? lead.tags : `${lead.tags},ໂອນແລ້ວ`) : 'ໂອນແລ້ວ',
    })

    // Also send confirmation message into chat
    await sendReply(l({
      lo: `ຂອບໃຈຫຼາຍໆເຈົ້າ! ຢືນຢັນອໍເດີ #${createdOrder.orderNumber}: ${prodName} x ${qty} ຍອດລວມ: ${totalAmount.toLocaleString()} ${currency} ຕັດສະຕ໋ອກຮຽບຮ້ອຍແລ້ວເຈົ້າ 🎉📦`,
      th: `ขอบพระคุณมากค่ะ! ยืนยันออเดอร์ #${createdOrder.orderNumber}: ${prodName} x ${qty} ยอดรวม: ${totalAmount.toLocaleString()} ${currency} ตัดสต็อกเรียบร้อยแล้วค่ะ 🎉📦`,
      en: `Thank you! Order #${createdOrder.orderNumber} confirmed: ${prodName} x ${qty} (Total: ${totalAmount.toLocaleString()} ${currency}), stock deducted! 🎉📦`
    }))

    quickOrderDialog.value = false
    statusNotice.value = l({
      lo: `ສ້າງອໍເດີ #${createdOrder.orderNumber} ສຳເລັດແລ້ວ! ຕັດສະຕ໋ອກ ແລະ ສົ່ງ Meta CAPI ຮຽບຮ້ອຍ`,
      th: `สร้างออเดอร์ #${createdOrder.orderNumber} สำเร็จแล้ว! ตัดสต็อกและส่ง Meta CAPI เรียบร้อย`,
      en: `Order #${createdOrder.orderNumber} created! Stock deducted and Meta CAPI sent.`
    })
    setTimeout(() => { statusNotice.value = '' }, 4000)
  } catch (err: any) {
    console.error('Quick order failed', err)
    loadError.value = err?.data?.error || err?.message || l({
      lo: 'ສ້າງອໍເດີບໍ່ສຳເລັດ ກະລຸນາກວດສອບສະຕ໋ອກສິນຄ້າ',
      th: 'สร้างออเดอร์ไม่สำเร็จ โปรดตรวจสอบสต็อกสินค้า',
      en: 'Failed to create order. Please check stock.'
    })
  } finally {
    submittingOrder.value = false
  }
}

const sendReply = async (explicitCode?: string, explicitReplySetId?: string) => {
  const msgToSend = (explicitCode !== undefined ? explicitCode : inputMessage.value).trim()
  if (!msgToSend || !activeSender.value || sending.value) return

  const senderId = activeSender.value
  const msgs = conversations.value[senderId]
  const pageId = msgs?.find(m => m.page_id)?.page_id || leads.value[senderId]?.pageId || ''

  // Optimistically bump timestamp for instant top ranking!
  if (conversations.value[senderId]?.length) {
    const lastMsg = conversations.value[senderId][conversations.value[senderId].length - 1]
    if (lastMsg) lastMsg.timestamp = String(Date.now())
  }
  if (leads.value[senderId]) {
    leads.value[senderId].updatedAt = new Date().toISOString()
  }

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

        <!-- Pancake Staff Filter -->
        <div class="staff-filter-row" v-if="teamMembers.length > 0">
          <button
            type="button"
            class="staff-filter-btn"
            :class="{ active: selectedStaffFilter === 'all' }"
            @click="selectedStaffFilter = 'all'"
          >
            {{ l({lo:'ທຸກແອດມິນ',th:'ทุกแอดมิน',en:'All Staff'}) }}
          </button>
          <button
            type="button"
            class="staff-filter-btn"
            :class="{ active: selectedStaffFilter === 'mine' }"
            @click="selectedStaffFilter = 'mine'"
          >
            <v-icon icon="mdi-account" size="13" class="mr-1"/>
            {{ l({lo:'ຂອງຂ້ອຍ',th:'ของฉัน',en:'My Chats'}) }}
          </button>
          <button
            type="button"
            class="staff-filter-btn"
            :class="{ active: selectedStaffFilter === 'unassigned' }"
            @click="selectedStaffFilter = 'unassigned'"
          >
            {{ l({lo:'ຍັງບໍ່ມອບໝາຍ',th:'ยังไม่มอบหมาย',en:'Unassigned'}) }}
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

          <!-- Quick Order Button -->
          <button
            type="button"
            class="action-btn-order"
            @click="quickOrderDialog = true"
            :title="l({lo:'ສ້າງອໍເດີດ່ວນ (Pancake Order)',th:'สร้างออเดอร์ด่วน (Pancake Order)',en:'Quick Order'})"
          >
            <v-icon icon="mdi-cart-plus" size="14" class="mr-1"/>
            <span>{{ l({lo:'ສ້າງອໍເດີ',th:'สร้างออเดอร์',en:'Order'}) }}</span>
          </button>

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

          <!-- Pancake CRM Details Toggle -->
          <button
            type="button"
            class="action-btn-crm"
            :class="{ active: pancakeDrawerOpen }"
            @click="pancakeDrawerOpen = !pancakeDrawerOpen"
            :title="l({lo:'ຂໍ້ມູນລູກຄ້າ & CRM Pancake',th:'ข้อมูลลูกค้า & CRM Pancake',en:'Customer CRM Details'})"
          >
            <v-icon icon="mdi-account-details-outline" size="16" class="mr-1"/>
            <span>CRM</span>
          </button>
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
          <button
            type="button"
            class="quick-reply-btn canned-btn"
            @click="cannedPhrasesDialog = true"
            :title="l({lo:'ຄຳຕອບມາດຕະຖານ / ຕອບດ່ວນ (Pancake)',th:'คำตอบมาตรฐาน / ตอบด่วน (Pancake)',en:'Canned Quick Replies'})"
          >
            <v-icon icon="mdi-chat-quote-outline" size="18"/>
            <span class="quick-btn-text">{{ l({lo:'ຕອບດ່ວນ',th:'ตอบด่วน',en:'Quick'}) }}</span>
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

    <!-- Pancake CRM Details Sidebar / Drawer -->
    <aside
      class="pancake-drawer"
      :class="{ open: pancakeDrawerOpen }"
      v-if="activeSender"
    >
      <div class="pancake-drawer-header">
        <div class="drawer-title-group">
          <v-icon icon="mdi-badge-account-horizontal-outline" color="primary" size="20"/>
          <h3>{{ l({lo:'ຂໍ້ມູນລູກຄ້າ CRM',th:'ข้อมูลลูกค้า CRM',en:'Customer CRM'}) }}</h3>
        </div>
        <button type="button" class="close-drawer-btn" @click="pancakeDrawerOpen = false" :title="l({lo:'ປິດ',th:'ปิด',en:'Close'})">
          <v-icon icon="mdi-close" size="18"/>
        </button>
      </div>

      <div class="pancake-drawer-content" v-if="activeSenderSummary">
        <!-- Customer Profile Card -->
        <div class="pancake-card profile-card">
          <div class="customer-avatar-lg">
            <img v-if="activeSenderSummary.senderPic" :src="activeSenderSummary.senderPic" alt="Avatar"/>
            <v-icon v-else :icon="activeSenderSummary.platform === 'whatsapp' ? 'mdi-whatsapp' : 'mdi-facebook-messenger'" size="28"/>
          </div>
          <div class="customer-profile-info">
            <div class="customer-name-edit">
              <input
                v-model="leadForm.name"
                class="name-input"
                :placeholder="l({lo:'ຊື່ລູກຄ້າ...',th:'ชื่อลูกค้า...',en:'Customer name...'})"
                @blur="saveLeadInfo"
              />
            </div>
            <div class="customer-id-row">
              <span class="platform-chip" :class="activeSenderSummary.platform">
                {{ activeSenderSummary.platform }}
              </span>
              <span class="customer-id-text" :title="activeSender">ID: {{ activeSender.slice(-8) }}</span>
              <button type="button" class="icon-btn-mini" @click="copyToClipboard(activeSender, 'ID')" :title="l({lo:'ກັອບປີ້ ID',th:'คัดลอก ID',en:'Copy ID'})">
                <v-icon :icon="copiedField === 'ID' ? 'mdi-check' : 'mdi-content-copy'" size="13"/>
              </button>
            </div>
          </div>
        </div>

        <!-- 1-Click Status Quick Changer -->
        <div class="pancake-card">
          <div class="pancake-section-label">
            <v-icon icon="mdi-flag-outline" size="15"/>
            <span>{{ l({lo:'ສະຖານະປັດຈຸບັນ',th:'สถานะปัจจุบัน',en:'Current Status'}) }}</span>
          </div>
          <div class="pancake-status-selector">
            <button
              v-for="opt in STATUS_OPTIONS"
              :key="opt.key"
              type="button"
              class="crm-status-btn"
              :class="{ active: activeLeadStatus === opt.key }"
              :style="activeLeadStatus === opt.key ? { backgroundColor: opt.bg, borderColor: opt.color, color: opt.color } : {}"
              @click="changeLeadStatus(activeSender, opt.key)"
            >
              <span class="status-dot-sm" :style="{ backgroundColor: opt.color }"></span>
              {{ l(opt.label) }}
            </button>
          </div>
        </div>

        <!-- Staff Assignment (Pancake Feature) -->
        <div class="pancake-card">
          <div class="pancake-section-label">
            <v-icon icon="mdi-account-tie" size="15"/>
            <span>{{ l({lo:'ແອດມິນຜູ້ຮັບຜິດຊອບ',th:'แอดมินผู้รับผิดชอบ',en:'Assigned Staff'}) }}</span>
          </div>
          <div class="staff-select-wrapper">
            <select
              :value="leadForm.assignedUserId"
              @change="onAssignStaff(($event.target as HTMLSelectElement).value)"
              class="staff-select"
            >
              <option value="">-- {{ l({lo:'ຍັງບໍ່ໄດ້ມອບໝາຍ',th:'ยังไม่ได้มอบหมาย',en:'Unassigned'}) }} --</option>
              <option v-for="staff in teamMembers" :key="staff.id" :value="staff.id">
                {{ staff.name }} ({{ staff.role }})
              </option>
            </select>
          </div>
        </div>

        <!-- Contact: Phone & Address -->
        <div class="pancake-card">
          <div class="pancake-section-label">
            <v-icon icon="mdi-phone-outline" size="15"/>
            <span>{{ l({lo:'ເບີໂທລະສັບ',th:'เบอร์โทรศัพท์',en:'Phone Number'}) }}</span>
            <div class="label-actions" v-if="leadForm.phone">
              <a :href="'tel:' + leadForm.phone" class="icon-link-mini" :title="l({lo:'ໂທອອກ',th:'โทรออก',en:'Call'})">
                <v-icon icon="mdi-phone" size="14"/>
              </a>
              <button type="button" class="icon-btn-mini" @click="copyToClipboard(leadForm.phone, 'phone')" :title="l({lo:'ກັອບປີ້ເບີໂທ',th:'คัดลอกเบอร์',en:'Copy Phone'})">
                <v-icon :icon="copiedField === 'phone' ? 'mdi-check' : 'mdi-content-copy'" size="13"/>
              </button>
            </div>
          </div>
          <div class="input-with-action">
            <input
              v-model="leadForm.phone"
              type="tel"
              class="crm-input"
              :placeholder="l({lo:'ປ້ອນເບີໂທ (ເຊັ່ນ 020...)',th:'เบอร์โทรศัพท์ (เช่น 08...)',en:'Phone number...'})"
              @blur="saveLeadInfo"
            />
          </div>

          <div class="pancake-section-label mt-3">
            <v-icon icon="mdi-map-marker-outline" size="15"/>
            <span>{{ l({lo:'ທີ່ຢູ່ຈັດສົ່ງ',th:'ที่อยู่จัดส่ง',en:'Delivery Address'}) }}</span>
            <div class="label-actions" v-if="leadForm.address">
              <button type="button" class="icon-btn-mini" @click="copyToClipboard(leadForm.address, 'address')" :title="l({lo:'ກັອບປີ້ທີ່ຢູ່',th:'คัดลอกที่อยู่',en:'Copy Address'})">
                <v-icon :icon="copiedField === 'address' ? 'mdi-check' : 'mdi-content-copy'" size="13"/>
              </button>
            </div>
          </div>
          <textarea
            v-model="leadForm.address"
            rows="2"
            class="crm-textarea"
            :placeholder="l({lo:'ບ້ານ, ເມືອງ, ແຂວງ, ຈຸດສັງເກດ...',th:'ที่อยู่จัดส่ง, ตำบล/อำเภอ/จังหวัด...',en:'Delivery address details...'})"
            @blur="saveLeadInfo"
          ></textarea>
        </div>

        <!-- Customer Tags (Pancake Feature) -->
        <div class="pancake-card">
          <div class="pancake-section-label">
            <v-icon icon="mdi-tag-multiple-outline" size="15"/>
            <span>{{ l({lo:'ປ້າຍກຳກັບລູກຄ້າ (Tags)',th:'แท็กลูกค้า (Tags)',en:'Customer Tags'}) }}</span>
          </div>
          <div class="tags-cloud">
            <button
              v-for="t in PRESET_TAGS"
              :key="t.key"
              type="button"
              class="crm-tag-chip"
              :class="{ selected: hasTag(t.key) }"
              :style="hasTag(t.key) ? { backgroundColor: t.bg, borderColor: t.border, color: t.color } : {}"
              @click="toggleCustomerTag(activeSender, t.key)"
            >
              <v-icon :icon="hasTag(t.key) ? 'mdi-check-circle' : 'mdi-plus-circle-outline'" size="13" class="mr-1"/>
              {{ t.label }}
            </button>
          </div>
        </div>

        <!-- Staff Internal Notes / Memo -->
        <div class="pancake-card">
          <div class="pancake-section-label">
            <v-icon icon="mdi-note-text-outline" size="15"/>
            <span>{{ l({lo:'ໝາຍເຫດພາຍໃນ (Staff Memo)',th:'บันทึกภายใน (Staff Memo)',en:'Internal Notes'}) }}</span>
          </div>
          <textarea
            v-model="leadForm.notes"
            rows="3"
            class="crm-textarea"
            :placeholder="l({lo:'ບັນທຶກຂໍ້ມູນພິເສດ, ປະຫວັດການສັ່ງ...',th:'บันทึกรายละเอียดเพิ่มเติม, ความต้องการ...',en:'Staff notes, order history memo...'})"
            @blur="saveLeadInfo"
          ></textarea>
          <div class="notes-footer">
            <button type="button" class="btn-save-note" @click="saveLeadInfo">
              <v-icon icon="mdi-content-save-outline" size="14" class="mr-1"/>
              {{ l({lo:'ບັນທຶກ',th:'บันทึก',en:'Save'}) }}
            </button>
          </div>
        </div>

        <!-- 1-Click Quick Order CTA -->
        <div class="pancake-card order-cta-card">
          <button type="button" class="btn-quick-order-full" @click="quickOrderDialog = true">
            <v-icon icon="mdi-cart-plus" size="18" class="mr-1"/>
            <span>{{ l({lo:'ສ້າງອໍເດີດ່ວນ (1-Click Order)',th:'สร้างออเดอร์ด่วน (1-Click Order)',en:'Quick Order (1-Click)'}) }}</span>
          </button>
        </div>
      </div>
    </aside>

    <!-- Mobile Pancake Backdrop -->
    <div
      v-if="activeSender && pancakeDrawerOpen"
      class="pancake-backdrop"
      @click="pancakeDrawerOpen = false"
    ></div>

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

    <!-- Canned Phrases Modal (Pancake Style) -->
    <v-dialog v-model="cannedPhrasesDialog" max-width="540">
      <v-card class="canned-dialog-card">
        <v-card-title class="dialog-title">
          <v-icon icon="mdi-chat-quote" color="primary" class="mr-2"/>
          <span>{{ l({lo:'ຂໍ້ຄວາມຕອບດ່ວນ / ຄຳຕອບມາດຕະຖານ',th:'ข้อความตอบด่วน / คำตอบมาตรฐาน',en:'Canned Quick Responses'}) }}</span>
        </v-card-title>
        <v-card-text class="dialog-body">
          <div class="canned-tabs">
            <button
              v-for="cat in CANNED_CATEGORIES"
              :key="cat.key"
              type="button"
              class="canned-cat-btn"
              :class="{ active: selectedCannedCategory === cat.key }"
              @click="selectedCannedCategory = cat.key"
            >
              <v-icon :icon="cat.icon" size="14" class="mr-1"/>
              {{ l(cat.label) }}
            </button>
          </div>

          <div class="canned-list">
            <div v-for="phrase in filteredCannedPhrases" :key="phrase.id" class="canned-item-card">
              <div class="canned-header">
                <div class="canned-title">
                  <v-icon :icon="phrase.icon" size="16" color="primary" class="mr-1"/>
                  <strong>{{ l(phrase.title) }}</strong>
                </div>
                <div class="canned-actions">
                  <button type="button" class="btn-insert" @click="insertPhrase(phrase, false)">
                    <v-icon icon="mdi-pencil-outline" size="13" class="mr-1"/>
                    {{ l({lo:'ໃສ່ໃນແຊັດ',th:'ใส่ในแชท',en:'Insert'}) }}
                  </button>
                  <button type="button" class="btn-send-now" @click="insertPhrase(phrase, true)">
                    <v-icon icon="mdi-send" size="13" class="mr-1"/>
                    {{ l({lo:'ສົ່ງເລີຍ',th:'ส่งทันที',en:'Send'}) }}
                  </button>
                </div>
              </div>
              <p class="canned-text-preview">{{ l(phrase.text) }}</p>
            </div>
          </div>
        </v-card-text>
        <v-card-actions>
          <v-spacer/>
          <v-btn variant="text" @click="cannedPhrasesDialog = false">{{ l({lo:'ປິດ',th:'ปิด',en:'Close'}) }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <!-- 1-Click Quick Order Modal (Pancake Style) -->
    <v-dialog v-model="quickOrderDialog" max-width="500">
      <v-card class="quick-order-card">
        <v-card-title class="dialog-title">
          <v-icon icon="mdi-cart-check" color="success" class="mr-2"/>
          <span>{{ l({lo:'ສ້າງອໍເດີດ່ວນ & ປິດການຂາຍ',th:'สร้างออเดอร์ด่วน & ปิดการขาย',en:'Quick Order & Close Deal'}) }}</span>
        </v-card-title>
        <v-card-text class="dialog-body">
          <p class="dialog-hint">
            {{ l({
              lo: 'ເລືອກສິນຄ້າ ແລະ ສາງທີ່ຕ້ອງການຕັດສະຕ໋ອກ, ລະບົບຈະສ້າງອໍເດີ, ຕັດສະຕ໋ອກ, ປ່ຽນສະຖານະເປັນ “ປິດການຂາຍ”, ແລະ ສົ່ງ Meta CAPI ອັດຕະໂນມັດ:',
              th: 'เลือกสินค้าและคลังที่ต้องการตัดสต็อก ระบบจะสร้างออเดอร์ ตัดสต็อก เปลี่ยนสถานะเป็น “ปิดการขาย” และส่ง Meta CAPI อัตโนมัติ:',
              en: 'Select product and warehouse to deduct stock. The system will create the order, deduct stock, mark Purchased, and trigger Meta CAPI:'
            }) }}
          </p>

          <div class="order-form-group mb-2">
            <label class="order-label">{{ l({lo:'ຕັດສະຕ໋ອກຈາກສາງ / ໜ້າຮ້ານ',th:'ตัดสต็อกสินค้าจากคลัง / หน้าร้าน',en:'Deduct from Location'}) }} *</label>
            <select v-model="quickOrderForm.warehouseId" class="order-select">
              <option value="">-- {{ l({lo:'ເລືອກສາງເລີ່ມຕົ້ນ...',th:'เลือกคลังเริ่มต้น...',en:'Default Location...'}) }} --</option>
              <option v-for="w in warehouses" :key="w.id" :value="w.id">
                {{ w.name }} ({{ w.code }}) {{ w.isDefault ? '★' : '' }}
              </option>
            </select>
          </div>

          <div class="order-form-group">
            <label class="order-label">{{ l({lo:'ສິນຄ້າ',th:'สินค้า',en:'Product'}) }} *</label>
            <select v-model="quickOrderForm.productId" class="order-select">
              <option value="">-- {{ l({lo:'ເລືອກສິນຄ້າ...',th:'เลือกสินค้า...',en:'Select product...'}) }} --</option>
              <option v-for="p in products" :key="p.id" :value="p.id">
                {{ p.code ? `[#${p.code}] ` : '' }}{{ p.name }} - {{ p.salePrice != null ? p.salePrice : p.price }} {{ p.currency || 'LAK' }}
              </option>
            </select>
          </div>

          <div class="order-form-row">
            <div class="order-form-group flex-1">
              <label class="order-label">{{ l({lo:'ຈຳນວນ',th:'จำนวน',en:'Quantity'}) }}</label>
              <div class="qty-stepper">
                <button type="button" @click="quickOrderForm.quantity = Math.max(1, quickOrderForm.quantity - 1)">-</button>
                <input v-model.number="quickOrderForm.quantity" type="number" min="1" class="qty-input"/>
                <button type="button" @click="quickOrderForm.quantity = quickOrderForm.quantity + 1">+</button>
              </div>
            </div>
            <div class="order-form-group flex-1" v-if="selectedOrderProduct">
              <label class="order-label">{{ l({lo:'ລາຄາຕໍ່ຊຸດ',th:'ราคาต่อชิ้น',en:'Unit Price'}) }}</label>
              <div class="readonly-price">{{ selectedOrderProduct.salePrice != null ? selectedOrderProduct.salePrice : selectedOrderProduct.price }} {{ selectedOrderProduct.currency || 'LAK' }}</div>
            </div>
          </div>

          <div class="order-form-row mt-2">
            <div class="order-form-group flex-1">
              <label class="order-label">{{ l({lo:'ບໍລິສັດຂົນສົ່ງ',th:'บริษัทขนส่ง',en:'Courier'}) }}</label>
              <select v-model="quickOrderForm.deliveryCourier" class="order-select">
                <option value="Anousith Express">Anousith Express (ອານຸສິດ)</option>
                <option value="HAL Logistics">HAL Logistics (ຮຸ່ງອາລຸນ)</option>
                <option value="Houngaloun">Houngaloun (ຮຸ່ງອາລຸນ)</option>
                <option value="Flash Express">Flash Express</option>
                <option value="Kerry Express">Kerry Express</option>
                <option value="Store Pickup">ຮັບເອງທີ່ຮ້ານ (Store Pickup)</option>
              </select>
            </div>
            <div class="order-form-group flex-1">
              <label class="order-label">{{ l({lo:'ຄ່າຈັດສົ່ງ',th:'ค่าจัดส่ง',en:'Shipping Fee'}) }}</label>
              <input v-model.number="quickOrderForm.shippingFee" type="number" min="0" class="order-input"/>
            </div>
          </div>

          <div class="order-form-group mt-2">
            <label class="order-label">{{ l({lo:'ເລກພັດສະດຸ / Tracking (ທາງເລືອກ)',th:'เลขพัสดุ / Tracking (ตัวเลือก)',en:'Tracking # (Optional)'}) }}</label>
            <input
              v-model="quickOrderForm.deliveryTracking"
              type="text"
              class="order-input"
              placeholder="AN123456..."
            />
          </div>

          <div class="order-form-group mt-2">
            <label class="order-label">{{ l({lo:'ໝາຍເຫດອໍເດີ (ທາງເລືອກ)',th:'หมายเหตุออเดอร์ (ตัวเลือก)',en:'Order Note (Optional)'}) }}</label>
            <input
              v-model="quickOrderForm.notes"
              type="text"
              class="order-input"
              :placeholder="l({lo:'ເຊັ່ນ: ໄຊສ໌ M, ສີຂາວ, ສົ່ງດ່ວນ...',th:'เช่น ไซส์ M, สีขาว, ส่งด่วน...',en:'e.g. Size M, Color White...'})"
            />
          </div>

          <!-- Total Calculation Box -->
          <div class="order-summary-box" v-if="selectedOrderProduct">
            <div class="summary-row">
              <span>{{ l({lo:'ຍອດລວມທັງໝົດ:',th:'ยอดรวมทั้งหมด:',en:'Total Amount:'}) }}</span>
              <strong class="total-price">{{ selectedOrderTotal }} {{ selectedOrderProduct.currency || 'LAK' }}</strong>
            </div>
          </div>
        </v-card-text>
        <v-card-actions class="dialog-actions-padding">
          <v-btn variant="text" @click="quickOrderDialog = false">{{ l({lo:'ຍົກເລີກ',th:'ยกเลิก',en:'Cancel'}) }}</v-btn>
          <v-btn
            color="success"
            variant="flat"
            :loading="submittingOrder"
            :disabled="!quickOrderForm.productId || submittingOrder"
            @click="submitQuickOrder"
          >
            <v-icon icon="mdi-check-circle" class="mr-1"/>
            {{ l({lo:'ຢືນຢັນອໍເດີ & ປິດການຂາຍ',th:'ยืนยันออเดอร์ & ปิดการขาย',en:'Confirm & Close Deal'}) }}
          </v-btn>
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

/* Header Action Buttons */
.action-btn-order {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--color-primary);
  color: white;
  border: none;
  padding: 6px 13px;
  border-radius: 20px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 750;
  transition: all 0.15s ease;
  touch-action: manipulation;
}
.action-btn-order:hover {
  background: var(--color-primary-dark);
}

.action-btn-crm {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--color-surface-soft);
  color: var(--color-text);
  border: 1px solid var(--color-border);
  padding: 6px 12px;
  border-radius: 20px;
  cursor: pointer;
  font-size: 12px;
  font-weight: 750;
  transition: all 0.15s ease;
  touch-action: manipulation;
}
.action-btn-crm:hover, .action-btn-crm.active {
  background: var(--color-primary-soft);
  color: var(--color-primary);
  border-color: var(--color-primary);
}

/* Sidebar Staff Filter */
.staff-filter-row {
  display: flex;
  align-items: center;
  gap: 5px;
  overflow-x: auto;
  white-space: nowrap;
  scrollbar-width: none;
  padding-top: 2px;
}
.staff-filter-row::-webkit-scrollbar {
  display: none;
}
.staff-filter-btn {
  padding: 3px 9px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 650;
  border: 1px solid var(--color-border);
  background: var(--color-background);
  color: var(--color-text-secondary);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  transition: all 0.15s ease;
  flex-shrink: 0;
}
.staff-filter-btn.active {
  background: var(--color-surface-soft);
  border-color: var(--color-primary);
  color: var(--color-primary);
  font-weight: 750;
}

/* Canned Button in Chat Input */
.canned-btn {
  color: #7c3aed;
  background: #f5f3ff;
  border-color: #ddd6fe;
}
.canned-btn:hover {
  background: #ede9fe;
  border-color: #7c3aed;
}

/* Pancake CRM Drawer (Desktop) */
.pancake-drawer {
  width: 320px;
  border-left: 1px solid var(--color-border);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
  transition: width 0.2s ease, opacity 0.2s ease;
}
.pancake-drawer-header {
  padding: 14px 16px;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: var(--color-surface);
}
.drawer-title-group {
  display: flex;
  align-items: center;
  gap: 8px;
}
.drawer-title-group h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 750;
  color: var(--color-text);
}
.close-drawer-btn {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  display: grid;
  place-items: center;
}
.close-drawer-btn:hover {
  background: var(--color-background);
  color: var(--color-text);
}

.pancake-drawer-content {
  flex: 1;
  overflow-y: auto;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.pancake-card {
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-background);
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.profile-card {
  flex-direction: row;
  align-items: center;
  gap: 12px;
}
.customer-avatar-lg {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--color-surface-soft);
  display: grid;
  place-items: center;
  flex-shrink: 0;
  overflow: hidden;
}
.customer-avatar-lg img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.customer-profile-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.name-input {
  width: 100%;
  font-size: 14px;
  font-weight: 750;
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text);
  padding: 2px 4px;
  border-radius: 6px;
  outline: none;
}
.name-input:focus {
  border-color: var(--color-primary);
  background: var(--color-surface);
}
.customer-id-row {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
}
.platform-chip {
  padding: 1px 6px;
  border-radius: 8px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
}
.platform-chip.facebook { background: #E7F3FF; color: #1877F2; }
.platform-chip.whatsapp { background: #E7FCE3; color: #25D366; }
.customer-id-text {
  color: var(--color-text-muted);
}
.icon-btn-mini {
  background: none;
  border: none;
  color: var(--color-text-muted);
  cursor: pointer;
  padding: 2px;
  display: grid;
  place-items: center;
  border-radius: 4px;
}
.icon-btn-mini:hover {
  color: var(--color-primary);
  background: var(--color-surface-soft);
}
.icon-link-mini {
  color: #16a34a;
  text-decoration: none;
  display: grid;
  place-items: center;
  padding: 2px;
  border-radius: 4px;
}
.icon-link-mini:hover {
  background: #dcfce7;
}

.pancake-section-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 11px;
  font-weight: 750;
  color: var(--color-text-secondary);
  gap: 4px;
}
.pancake-section-label span {
  flex: 1;
}
.label-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.pancake-status-selector {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 6px;
}
.crm-status-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 6px 8px;
  border-radius: 8px;
  font-size: 11px;
  font-weight: 700;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}
.status-dot-sm {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.staff-select-wrapper {
  position: relative;
}
.staff-select {
  width: 100%;
  height: 36px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 0 10px;
  font-size: 12px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
  cursor: pointer;
}
.staff-select:focus {
  border-color: var(--color-primary);
}

.crm-input {
  width: 100%;
  height: 36px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 0 10px;
  font-size: 12.5px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
}
.crm-input:focus {
  border-color: var(--color-primary);
}
.crm-textarea {
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: 8px;
  padding: 8px 10px;
  font-size: 12px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
  resize: vertical;
}
.crm-textarea:focus {
  border-color: var(--color-primary);
}

.tags-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.crm-tag-chip {
  display: inline-flex;
  align-items: center;
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 700;
  border: 1px solid var(--color-border);
  background: var(--color-surface);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}
.crm-tag-chip:hover {
  border-color: var(--color-primary);
}

.notes-footer {
  display: flex;
  justify-content: flex-end;
}
.btn-save-note {
  display: inline-flex;
  align-items: center;
  padding: 5px 12px;
  border-radius: 6px;
  background: var(--color-primary);
  color: white;
  border: none;
  font-size: 11.5px;
  font-weight: 700;
  cursor: pointer;
}
.btn-save-note:hover {
  background: var(--color-primary-dark);
}

.order-cta-card {
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  border-color: #bbf7d0;
}
.btn-quick-order-full {
  width: 100%;
  height: 42px;
  border: none;
  border-radius: 10px;
  background: #16a34a;
  color: white;
  font-size: 13px;
  font-weight: 800;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 8px rgba(22,163,74,0.25);
  transition: all 0.15s ease;
  touch-action: manipulation;
}
.btn-quick-order-full:hover {
  background: #15803d;
  box-shadow: 0 4px 12px rgba(22,163,74,0.35);
}

.pancake-backdrop {
  display: none;
}

/* Canned Phrases Dialog */
.canned-dialog-card {
  border-radius: 18px !important;
}
.canned-tabs {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  white-space: nowrap;
  padding-bottom: 8px;
  margin-bottom: 12px;
  scrollbar-width: none;
}
.canned-tabs::-webkit-scrollbar {
  display: none;
}
.canned-cat-btn {
  display: inline-flex;
  align-items: center;
  padding: 5px 11px;
  border-radius: 14px;
  font-size: 11.5px;
  font-weight: 700;
  border: 1px solid var(--color-border);
  background: var(--color-background);
  color: var(--color-text-secondary);
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.15s ease;
}
.canned-cat-btn.active {
  background: #7c3aed;
  border-color: #7c3aed;
  color: white;
}
.canned-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 380px;
  overflow-y: auto;
  padding-right: 4px;
}
.canned-item-card {
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: 12px;
  background: var(--color-background);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.canned-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.canned-title {
  display: flex;
  align-items: center;
  font-size: 13px;
  font-weight: 750;
  color: var(--color-text);
}
.canned-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.canned-text-preview {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-secondary);
  line-height: 1.45;
  white-space: pre-line;
}

/* Quick Order Dialog */
.quick-order-card {
  border-radius: 18px !important;
}
.order-form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.order-label {
  font-size: 11.5px;
  font-weight: 750;
  color: var(--color-text-secondary);
}
.order-select, .order-input {
  width: 100%;
  height: 40px;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  padding: 0 12px;
  font-size: 13px;
  background: var(--color-background);
  color: var(--color-text);
  outline: none;
}
.order-select:focus, .order-input:focus {
  border-color: var(--color-primary);
}
.order-form-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 10px;
}
.qty-stepper {
  display: flex;
  align-items: center;
  border: 1px solid var(--color-border);
  border-radius: 10px;
  overflow: hidden;
  height: 40px;
  background: var(--color-background);
}
.qty-stepper button {
  width: 38px;
  height: 100%;
  border: none;
  background: var(--color-surface-soft);
  color: var(--color-text);
  font-size: 16px;
  font-weight: 800;
  cursor: pointer;
  display: grid;
  place-items: center;
}
.qty-input {
  flex: 1;
  width: 44px;
  text-align: center;
  border: none;
  background: transparent;
  font-size: 14px;
  font-weight: 750;
  outline: none;
}
.readonly-price {
  height: 40px;
  display: flex;
  align-items: center;
  padding: 0 12px;
  background: var(--color-surface-soft);
  border: 1px solid var(--color-border);
  border-radius: 10px;
  font-size: 13px;
  font-weight: 750;
  color: var(--color-text);
}
.order-summary-box {
  margin-top: 14px;
  padding: 12px 14px;
  background: #f0fdf4;
  border: 1px solid #bbf7d0;
  border-radius: 12px;
}
.summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  color: #166534;
}
.total-price {
  font-size: 16px;
  font-weight: 850;
  color: #15803d;
}
.dialog-actions-padding {
  padding: 12px 24px 16px !important;
}

/* Mobile & Tablet Responsive Rules for Pancake CRM Drawer */
@media(max-width:1024px) {
  .pancake-drawer {
    position: fixed;
    top: 0;
    right: -360px;
    bottom: 0;
    width: min(340px, 86vw);
    z-index: 1000;
    box-shadow: -4px 0 24px rgba(0,0,0,0.2);
    transition: right 0.25s cubic-bezier(0.16, 1, 0.3, 1);
  }
  .pancake-drawer.open {
    right: 0;
  }
  .pancake-backdrop {
    display: block;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0,0,0,0.4);
    z-index: 999;
    backdrop-filter: blur(2px);
  }
}
</style>
