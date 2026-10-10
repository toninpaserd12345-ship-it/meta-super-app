import re

with open('frontend/pages/live-chat.vue', 'r') as f:
    content = f.read()

# Add button
old_header = """      <div class="chat-header">
        <div class="chat-header-info">
          <h3>{{ getSenderName(conversations[activeSender]) }}</h3>
          <span class="platform-badge" :class="conversations[activeSender]?.[0]?.platform">
            {{ conversations[activeSender]?.[0]?.platform }}
          </span>
        </div>
      </div>"""
new_header = """      <div class="chat-header">
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
      </div>"""
content = content.replace(old_header, new_header)

# Add markAsPurchased script
script_addition = """
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
"""
content = content.replace("const sendReply = async () => {", script_addition + "\nconst sendReply = async () => {")

# Add CSS for header
css_addition = """
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
"""
content = content.replace(".chat-header {", css_addition + "\n/*")
content = content.replace("  border-bottom: 1px solid var(--color-border);\n}", "*/\n")

with open('frontend/pages/live-chat.vue', 'w') as f:
    f.write(content)
