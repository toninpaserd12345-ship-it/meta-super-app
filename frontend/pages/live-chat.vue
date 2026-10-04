<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'

const { token, currentAccountID } = useAuth()
const api = useApi()

interface ChatMessage {
  id: string
  sender_id: string
  message: string
  type: string
  timestamp: string
  platform: 'facebook' | 'whatsapp' | 'system'
  page_id: string
}

const conversations = ref<Record<string, ChatMessage[]>>({})
const activeSender = ref<string | null>(null)
const inputMessage = ref('')
const evtSource = ref<EventSource | null>(null)

const activeChatMessages = computed(() => {
  if (!activeSender.value) return []
  return conversations.value[activeSender.value] || []
})

const sendReply = async () => {
  if (!inputMessage.value.trim() || !activeSender.value) return
  const msg = inputMessage.value
  inputMessage.value = ''
  
  // Find the page ID associated with this sender
  const msgs = conversations.value[activeSender.value]
  const pageId = msgs[0]?.page_id

  if (!pageId) return

  await api.post('/chat/send', {
    page_id: pageId,
    recipient_id: activeSender.value,
    message: msg
  })
  
  // Note: the backend will stream it back so it will appear automatically
}

onMounted(() => {
  if (import.meta.server) return
  
  const baseURL = useRuntimeConfig().public.apiBase
  const url = `${baseURL}/chat/stream?token=${token.value}&account_id=${currentAccountID.value}`
  
  evtSource.value = new EventSource(url)
  
  evtSource.value.onmessage = (event) => {
    const data = JSON.parse(event.data) as ChatMessage
    data.id = data.timestamp + Math.random() // Temp ID
    
    // Determine the key for the conversation. Group by SenderID.
    const key = data.sender_id
    if (!conversations.value[key]) {
      conversations.value[key] = []
    }
    conversations.value[key].push(data)
    
    // Sort by timestamp
    conversations.value[key].sort((a, b) => Number(a.timestamp) - Number(b.timestamp))
  }
  
  evtSource.value.onerror = (error) => {
    console.error('SSE Error:', error)
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
        <h2>Live Inbox</h2>
      </div>
      <div class="conversation-list">
        <div v-if="Object.keys(conversations).length === 0" class="empty-list">
          No active conversations. Waiting for messages...
        </div>
        <div 
          v-for="(msgs, senderId) in conversations" 
          :key="senderId"
          class="conv-item"
          :class="{ active: activeSender === senderId }"
          @click="activeSender = senderId"
        >
          <div class="conv-avatar">
            <v-icon :icon="msgs[0].platform === 'whatsapp' ? 'mdi-whatsapp' : 'mdi-facebook-messenger'" />
          </div>
          <div class="conv-details">
            <div class="conv-name">{{ senderId }}</div>
            <div class="conv-preview">{{ msgs[msgs.length - 1].message || 'Media message' }}</div>
          </div>
        </div>
      </div>
    </div>
    
    <div class="chat-main" v-if="activeSender">
      <div class="chat-header">
        <div class="chat-header-info">
          <h3>{{ activeSender }}</h3>
          <span class="platform-badge" :class="conversations[activeSender][0].platform">
            {{ conversations[activeSender][0].platform }}
          </span>
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
            <span v-if="msg.type === 'image'" class="media-placeholder">[Image]</span>
            <span v-else>{{ msg.message }}</span>
          </div>
          <div class="bubble-time">{{ formatTime(msg.timestamp) }}</div>
        </div>
      </div>
      
      <div class="chat-input">
        <input 
          v-model="inputMessage" 
          @keyup.enter="sendReply" 
          type="text" 
          placeholder="Type a reply..."
        />
        <button @click="sendReply"><v-icon icon="mdi-send"/></button>
      </div>
    </div>
    
    <div class="chat-empty" v-else>
      <v-icon icon="mdi-message-text-outline" size="64" />
      <p>Select a conversation to start chatting</p>
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
  border-bottom: 1px solid var(--color-border);
}

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
}

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
  padding: 16px;
  border-top: 1px solid var(--color-border);
  display: flex;
  gap: 12px;
}

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

.chat-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted);
  gap: 16px;
}
</style>
