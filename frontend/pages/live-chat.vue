<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'





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
const { locale } = useLocale()

const activeChatMessages = computed(() => {
  if (!activeSender.value) return []
  return conversations.value[activeSender.value] || []
})

const getSenderName = (msgs?: ChatMessage[]) => {
  if (!msgs || msgs.length === 0) return 'Unknown'
  const msg = msgs.find(m => m.sender_name)
  return msg?.sender_name || msgs[0]?.sender_id
}

const getSenderPic = (msgs?: ChatMessage[]) => {
  if (!msgs || msgs.length === 0) return undefined
  const msg = msgs.find(m => m.sender_pic)
  return msg?.sender_pic
}


const markAsPurchased = async (senderId: string) => {
  const lead = leads.value[senderId]
  if (!lead) return
  try {
    await $fetch(`/api/proxy/api/v1/leads/${senderId}/status`, {
      method: 'PUT',
      body: { status: 'purchased' }
    })
    leads.value[senderId].status = 'purchased'
  } catch (e) {
    alert(locale.value === 'lo' ? 'ອັບເດດສະຖານະບໍ່ສຳເລັດ' : 'Failed to update lead status')
  }
}

const sendReply = async () => {
  if (!inputMessage.value.trim() || !activeSender.value || sending.value) return
  const msg = inputMessage.value
  
  // Find the page ID associated with this sender
  const msgs = conversations.value[activeSender.value]
  const pageId = msgs?.[0]?.page_id

  if (!pageId) return

  sending.value = true
  loadError.value = ''
  try {
    await $fetch('/api/proxy/api/v1/chat/send', {
      method: 'POST',
      body: { page_id: pageId, recipient_id: activeSender.value, message: msg },
    })
    inputMessage.value = ''
  } catch (error: any) {
    loadError.value = error?.data?.message || error?.statusMessage || (locale.value === 'lo' ? 'ສົ່ງຂໍ້ຄວາມບໍ່ສຳເລັດ' : 'Message could not be sent.')
  } finally {
    sending.value = false
  }
}

onMounted(async () => {
  if (import.meta.server) return
  
  // Load initial history and leads
  try {
    const [historyRes, leadsRes] = await Promise.all([
      $fetch<{items: ChatMessage[]}>('/api/proxy/api/v1/chat/history').catch(()=>({items:[]})),
      $fetch<{items: Lead[]}>('/api/proxy/api/v1/leads').catch(()=>({items:[]}))
    ])
    
    const leadsList = leadsRes.items || []
    leadsList.forEach(l => { leads.value[l.senderId] = l })
    
    const history = historyRes.items || []
    if (history.length > 0) {
      for (const msg of history) {
        msg.id = msg.timestamp + Math.random()
        const key = msg.sender_id
        if (!conversations.value[key]) {
          conversations.value[key] = []
        }
        conversations.value[key].push(msg)
      }
      for (const key in conversations.value) {
        conversations.value[key]?.sort((a, b) => Number(a.timestamp) - Number(b.timestamp))
      }
    }
  } catch (err) {
    loadError.value = locale.value === 'lo' ? 'ໂຫຼດປະຫວັດສົນທະນາບໍ່ສຳເລັດ' : 'Conversation history could not be loaded.'
  }

  const url = '/api/proxy/api/v1/chat/stream'
  
  evtSource.value = new EventSource(url)
  
  evtSource.value.onopen = () => {
    loadError.value = ''
  }

  evtSource.value.onmessage = (event) => {
    let data: ChatMessage
    try {
      data = JSON.parse(event.data) as ChatMessage
    } catch {
      return
    }
    data.id = data.timestamp + Math.random() // Temp ID
    
    // Determine the key for the conversation. Group by SenderID.
    const key = data.sender_id
    if (!conversations.value[key]) {
      conversations.value[key] = []
    }
    conversations.value[key].push(data)
    
    // Sort by timestamp
    conversations.value[key]?.sort((a, b) => Number(a.timestamp) - Number(b.timestamp))
  }
  
  evtSource.value.onerror = (error) => {
    loadError.value = locale.value === 'lo' ? 'ການເຊື່ອມຕໍ່ຂໍ້ຄວາມຂາດຊົ່ວຄາວ' : 'The live message connection was interrupted.'
  }
})

onUnmounted(() => {
  if (evtSource.value) {
    evtSource.value.close()
  }
})

const formatTime = (ts: string) => {
  if (ts === 'now') return 'Just now'
  const d = new Date(Number(ts))
  if (isNaN(d.getTime())) return ts
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div class="chat-layout">
    <div class="chat-sidebar">
      <div class="sidebar-header">
        <h2>{{locale==='lo'?'ກ່ອງຂໍ້ຄວາມ':'Live Inbox'}}</h2>
      </div>
      <div class="conversation-list">
        <div v-if="Object.keys(conversations).length === 0" class="empty-list">
          {{locale==='lo'?'ຍັງບໍ່ມີການສົນທະນາ':'No conversations yet.'}}
        </div>
        <div 
          v-for="(msgs, senderId) in conversations" 
          :key="senderId"
          class="conv-item"
          :class="{ active: activeSender === senderId }"
          @click="activeSender = senderId"
        >
          <div class="conv-avatar">
            <img v-if="getSenderPic(msgs)" :src="getSenderPic(msgs)" alt="Profile" class="avatar-img" />
            <v-icon v-else :icon="msgs?.[0]?.platform === 'whatsapp' ? 'mdi-whatsapp' : 'mdi-facebook-messenger'" />
          </div>
          <div class="conv-details">
            <div class="conv-name">
              {{ getSenderName(msgs) }}
              <span v-if="leads[senderId]" class="status-badge" :class="leads[senderId].status">{{ leads[senderId].status }}</span>
            </div>
            <div class="conv-preview">{{ msgs?.[msgs.length - 1]?.message || 'Media message' }}</div>
            <div class="conv-meta" v-if="leads[senderId]">
              <span class="meta-tag page-tag"><v-icon icon="mdi-flag" size="12"/> {{ leads[senderId].pageId }}</span>
              <span class="meta-tag ad-tag" v-if="leads[senderId].sourceType === 'ad'"><v-icon icon="mdi-bullhorn" size="12"/> Ad: {{ leads[senderId].sourceId }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
    
    <div class="chat-main" v-if="activeSender">
      <div class="chat-header">
        <div class="chat-header-info">
          <h3>{{ getSenderName(conversations[activeSender]) }}</h3>
          <span class="platform-badge" :class="conversations[activeSender]?.[0]?.platform">
            {{ conversations[activeSender]?.[0]?.platform }}
          </span>
        </div>
        <div class="chat-header-actions" v-if="leads[activeSender]">
          <button v-if="leads[activeSender].status !== 'purchased'" @click="markAsPurchased(activeSender)" class="action-btn">
            {{ locale === 'lo' ? 'ປິດການຂາຍ (ຊື້ແລ້ວ)' : 'Mark as Purchased' }}
          </button>
          <span v-else class="status-badge purchased">✔ {{ locale === 'lo' ? 'ປິດການຂາຍແລ້ວ' : 'Purchased' }}</span>
        </div>
      </div>
      
      <div class="chat-messages">
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
      </div>
      
      <div class="chat-input">
        <div v-if="loadError" class="chat-error">{{loadError}}</div>
        <input 
          v-model="inputMessage" 
          @keyup.enter="sendReply" 
          type="text" 
          :placeholder="locale==='lo'?'ພິມຂໍ້ຄວາມຕອບກັບ':'Write a reply'"
          :disabled="sending"
        />
        <button :disabled="sending||!inputMessage.trim()" @click="sendReply"><v-progress-circular v-if="sending" indeterminate size="18" width="2"/><v-icon v-else icon="mdi-send"/></button>
      </div>
    </div>
    
    <div class="chat-empty" v-else>
      <v-icon icon="mdi-message-text-outline" size="64" />
      <p>{{locale==='lo'?'ເລືອກການສົນທະນາເພື່ອເບິ່ງ ແລະຕອບຂໍ້ຄວາມ':'Select a conversation to view and reply.'}}</p>
    </div>
  </div>
</template>

<style scoped>
.chat-layout {
  display: flex;
  height: calc(100vh - 120px);
  background: var(--color-surface);
  border-radius: 12px;
  border: 1px solid var(--color-border);
  overflow: hidden;
}

.chat-sidebar {
  width: 320px;
  border-right: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  padding: 20px;
*/


.sidebar-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
}

.empty-list {
  padding: 20px;
  text-align: center;
  color: var(--color-text-muted);
  font-size: 14px;
}

.conv-item {
  display: flex;
  padding: 16px;
  gap: 12px;
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
  transition: background 0.2s;
}

.conv-item:hover {
  background: var(--color-background);
}

.conv-item.active {
  background: var(--color-primary-soft);
}

.conv-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-border);
  display: grid;
  place-items: center;
  overflow: hidden;
}
.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.conv-details {
  flex: 1;
  overflow: hidden;
}

.conv-name {
  font-weight: 600;
  font-size: 14px;
  margin-bottom: 4px;
}

.conv-preview {
  font-size: 13px;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
}


.chat-header {
  padding: 20px;
  border-bottom: 1px solid var(--color-border);
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.action-btn {
  background: var(--color-primary);
  color: white;
  border: none;
  padding: 6px 12px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
}
.action-btn:hover {
  background: var(--color-primary-dark);
}

/*
  padding: 20px;
*/


.chat-header-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chat-header-info h3 {
  margin: 0;
  font-size: 16px;
}

.platform-badge {
  font-size: 11px;
  padding: 4px 8px;
  border-radius: 12px;
  font-weight: 600;
  text-transform: uppercase;
}
.platform-badge.facebook { background: #E7F3FF; color: #1877F2; }
.platform-badge.whatsapp { background: #E7FCE3; color: #25D366; }

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

.chat-input {
  position: relative;
  padding: 16px;
  border-top: 1px solid var(--color-border);
  display: flex;
  gap: 12px;
}
.chat-error{position:absolute;left:16px;right:16px;bottom:68px;padding:8px 12px;color:var(--color-error);background:var(--color-error-soft);border:1px solid color-mix(in srgb,var(--color-error) 30%,transparent);border-radius:9px;font-size:12px}

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
@media(max-width:760px){.chat-layout{height:calc(100dvh - 96px)}.chat-sidebar{width:112px}.sidebar-header{padding:15px 10px}.sidebar-header h2{font-size:14px}.conv-item{display:grid;justify-items:center;padding:12px 8px}.conv-details{width:100%;text-align:center}.conv-name{font-size:10px;overflow:hidden;text-overflow:ellipsis}.conv-preview{display:none}.chat-header,.chat-messages{padding:14px}.message-bubble{max-width:88%}.chat-empty{padding:20px;text-align:center}}

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
