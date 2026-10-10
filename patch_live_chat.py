import re

with open('frontend/pages/live-chat.vue', 'r') as f:
    content = f.read()

# Add Lead Interface
lead_interface = """
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
"""
content = content.replace("interface ChatMessage", lead_interface + "interface ChatMessage")

# Add leads ref
content = content.replace("const conversations = ref", "const leads = ref<Record<string, Lead>>({})\nconst conversations = ref")

# Update onMounted
old_onmounted = """  // Load initial history
  try {
    const history = await $fetch<{items: ChatMessage[]}>('/api/proxy/api/v1/chat/history').then(r => r.items)
    if (history && history.length > 0) {
      for (const msg of history) {
        msg.id = msg.timestamp + Math.random()
        const key = msg.sender_id
        if (!conversations.value[key]) {
          conversations.value[key] = []
        }
        conversations.value[key].push(msg)
      }
      // Sort each conversation
      for (const key in conversations.value) {
        conversations.value[key]?.sort((a, b) => Number(a.timestamp) - Number(b.timestamp))
      }
    }
  } catch (err) {
    loadError.value = locale.value === 'lo' ? 'ໂຫຼດປະຫວັດສົນທະນາບໍ່ສຳເລັດ' : 'Conversation history could not be loaded.'
  }"""

new_onmounted = """  // Load initial history and leads
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
  }"""

content = content.replace(old_onmounted, new_onmounted)

# Update UI Template in Sidebar
old_sidebar_details = """          <div class="conv-details">
            <div class="conv-name">{{ getSenderName(msgs) }}</div>
            <div class="conv-preview">{{ msgs?.[msgs.length - 1]?.message || 'Media message' }}</div>
          </div>"""

new_sidebar_details = """          <div class="conv-details">
            <div class="conv-name">
              {{ getSenderName(msgs) }}
              <span v-if="leads[senderId]" class="status-badge" :class="leads[senderId].status">{{ leads[senderId].status }}</span>
            </div>
            <div class="conv-preview">{{ msgs?.[msgs.length - 1]?.message || 'Media message' }}</div>
            <div class="conv-meta" v-if="leads[senderId]">
              <span class="meta-tag page-tag"><v-icon icon="mdi-flag" size="12"/> {{ leads[senderId].pageId }}</span>
              <span class="meta-tag ad-tag" v-if="leads[senderId].sourceType === 'ad'"><v-icon icon="mdi-bullhorn" size="12"/> Ad: {{ leads[senderId].sourceId }}</span>
            </div>
          </div>"""

content = content.replace(old_sidebar_details, new_sidebar_details)

# Add CSS for new badges
css_addition = """
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
"""

content = content.replace("</style>", css_addition + "\n</style>")

with open('frontend/pages/live-chat.vue', 'w') as f:
    f.write(content)
