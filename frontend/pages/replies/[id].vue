<script setup lang="ts">
import type { ReplyItem, ReplyItemType, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

const route = useRoute()
const router = useRouter()
const { can } = useAuth()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Reply Sets.' })

const { data: replySet, pending, error, refresh } = await useApi<ReplySet>(`/proxy/api/v1/replies/${String(route.params.id)}`)
const name = ref('')
const items = ref<ReplyItem[]>([])
const saving = ref(false)
const uploadingIndex = ref(-1)
const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')

watch(replySet, (value) => {
  if (!value) return
  name.value = value.name
  items.value = [...(value.items || [])]
    .sort((first, second) => first.orderIndex - second.orderIndex)
    .map(item => ({ ...item, isEnabled: item.isEnabled !== false }))
}, { immediate: true })

const canSave = computed(() => Boolean(
  name.value.trim()
  && items.value.length
  && items.value.some(item => item.isEnabled)
  && items.value.every(item => !item.isEnabled || item.content.trim()),
))

const messageTypes: Array<{ type: ReplyItemType; label: string; description: string; icon: string }> = [
  { type: 'text', label: 'Text', description: 'Message with product variables', icon: 'mdi-text' },
  { type: 'image', label: 'Image', description: 'Upload or use an HTTPS URL', icon: 'mdi-image-outline' },
  { type: 'video', label: 'Video', description: 'Send a product video', icon: 'mdi-video-outline' },
  { type: 'audio', label: 'Audio', description: 'Send a voice or audio file', icon: 'mdi-microphone-outline' },
]

function addItem(type: ReplyItemType) {
  items.value.push({ type, content: '', orderIndex: items.value.length, isEnabled: true })
}

function removeItem(index: number) {
  items.value.splice(index, 1)
}

function moveItem(index: number, direction: number) {
  const target = index + direction
  if (target < 0 || target >= items.value.length) return
  const current = items.value[index]
  const other = items.value[target]
  if (!current || !other) return
  items.value[index] = other
  items.value[target] = current
}

function acceptFor(type: ReplyItemType) {
  return `${type}/*`
}

function errorMessage(error: unknown, fallback: string) {
  const value = error as { data?: { error?: string | { message?: string }; message?: string }; statusMessage?: string }
  const upstream = value.data?.error
  return (typeof upstream === 'string' ? upstream : upstream?.message) || value.data?.message || value.statusMessage || fallback
}

async function uploadMedia(value: unknown, index: number) {
  const files = Array.isArray(value) ? value : value ? [value] : []
  const file = files[0]
  if (!(file instanceof File) || !items.value[index]) return
  uploadingIndex.value = index
  notice.value = ''
  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('folder', 'replies')
    const result = await $fetch<{ url: string }>('/api/proxy/api/v1/storage/upload', { method: 'POST', body: formData })
    items.value[index].content = result.url
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, 'Media upload failed. You can paste a public HTTPS URL instead.')
  } finally {
    uploadingIndex.value = -1
  }
}

async function saveChanges() {
  if (!replySet.value || !canSave.value) return
  saving.value = true
  notice.value = ''
  const formattedItems = items.value.map((item, index) => ({ ...item, orderIndex: index }))
  try {
    if (name.value.trim() !== replySet.value.name) {
      await $fetch(`/api/proxy/api/v1/replies/${replySet.value.id}`, {
        method: 'PUT',
        body: { name: name.value.trim() },
      })
    }
    await $fetch(`/api/proxy/api/v1/replies/${replySet.value.id}/items`, {
      method: 'PUT',
      body: { items: formattedItems },
    })
    await refresh()
    noticeType.value = 'success'
    notice.value = 'Reply Set saved and ready to use.'
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, 'Unable to save this Reply Set.')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <div class="back-row"><v-btn variant="text" prepend-icon="mdi-arrow-left" @click="router.push('/replies')">All Reply Sets</v-btn></div>

    <section v-if="replySet" class="page-head">
      <div>
        <p>AUTO REPLY SET</p>
        <h2>{{ replySet.name }}</h2>
        <span>Arrange messages in the exact order customers should receive them.</span>
      </div>
      <v-btn color="primary" prepend-icon="mdi-content-save-outline" :loading="saving" :disabled="!can('pages:connect') || !canSave" @click="saveChanges">Save changes</v-btn>
    </section>

    <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>
    <v-alert v-if="error" type="error" variant="tonal">This Reply Set could not be loaded. <button class="inline-action" @click="refresh()">Try again</button></v-alert>
    <div v-else-if="pending" class="loading"><v-progress-circular indeterminate color="primary"/></div>

    <template v-else-if="replySet">
      <section class="name-panel">
        <v-text-field v-model="name" label="Reply Set name" variant="outlined" maxlength="120" counter hide-details/>
        <v-alert type="info" variant="tonal" density="compact">
          This Reply Set belongs to the current store/workspace and works across its connected Messenger and WhatsApp channels. Customers send the exact set name to receive it. Product variables: <code v-text="'{{product.name}}'"/>, <code v-text="'{{product.price}}'"/> and <code v-text="'{{product.description}}'"/>.
        </v-alert>
      </section>

      <div class="builder-layout">
        <main class="sequence" aria-label="Message sequence">
          <div class="sequence-head"><div><small>MESSAGE SEQUENCE</small><strong>{{ items.length }} step{{ items.length === 1 ? '' : 's' }}</strong></div><span>Messages send from top to bottom</span></div>
          <section v-if="!items.length" class="sequence-empty">
            <v-icon icon="mdi-message-plus-outline" size="32"/><h3>Add the first message</h3><p>Choose a message type from the panel on the right.</p>
          </section>
          <article v-for="(item,index) in items" :key="item.id || `${item.type}-${index}`" class="message-card" :class="{ disabled: !item.isEnabled }">
            <div class="order-controls">
              <v-btn icon="mdi-chevron-up" variant="text" size="small" :disabled="index===0" :aria-label="`Move message ${index+1} up`" @click="moveItem(index,-1)"/>
              <strong>{{ index + 1 }}</strong>
              <v-btn icon="mdi-chevron-down" variant="text" size="small" :disabled="index===items.length-1" :aria-label="`Move message ${index+1} down`" @click="moveItem(index,1)"/>
            </div>
            <div class="message-editor">
              <div class="message-title"><span><v-icon :icon="messageTypes.find(option=>option.type===item.type)?.icon" size="18"/>{{ item.type }}</span><div class="message-actions"><v-switch v-model="item.isEnabled" color="success" density="compact" hide-details :label="item.isEnabled ? 'Send' : 'Skip'"/><v-btn icon="mdi-delete-outline" variant="text" size="small" color="error" :aria-label="`Delete message ${index+1}`" @click="removeItem(index)"/></div></div>
              <v-textarea v-if="item.type==='text'" v-model="item.content" label="Message text" variant="outlined" rows="4" auto-grow maxlength="2000" counter hide-details/>
              <template v-else>
                <v-img v-if="item.type==='image' && item.content" :src="item.content" max-height="240" contain class="media-preview"/>
                <video v-else-if="item.type==='video' && item.content" :src="item.content" controls class="media-preview"/>
                <audio v-else-if="item.type==='audio' && item.content" :src="item.content" controls class="audio-preview"/>
                <v-text-field v-model="item.content" :label="`${item.type} public HTTPS URL`" variant="outlined" prepend-inner-icon="mdi-link" hide-details/>
                <div class="upload-row"><span>or upload a file</span><v-file-input :accept="acceptFor(item.type)" label="Choose file" variant="outlined" density="compact" prepend-icon="mdi-cloud-upload-outline" hide-details :loading="uploadingIndex===index" @update:model-value="uploadMedia($event,index)"/></div>
              </template>
            </div>
          </article>
        </main>

        <aside class="toolbox">
          <small>ADD MESSAGE</small><h3>Choose a type</h3>
          <button v-for="option in messageTypes" :key="option.type" @click="addItem(option.type)">
            <span><v-icon :icon="option.icon"/></span><div><strong>{{ option.label }}</strong><small>{{ option.description }}</small></div><v-icon icon="mdi-plus" size="18"/>
          </button>
          <v-btn block color="primary" variant="tonal" class="mt-4" prepend-icon="mdi-robot-happy-outline" to="/auto-replies">Use in Auto Reply</v-btn>
        </aside>
      </div>
    </template>
  </div>
</template>

<style scoped>
.back-row{margin-bottom:10px}.page-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:16px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head p,.sequence-head small,.toolbox>small{margin:0 0 6px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 6px;font-size:24px}.page-head span,.sequence-head span{display:block;color:var(--color-text-secondary);font-size:12px}.name-panel{display:grid;grid-template-columns:minmax(260px,.7fr) minmax(320px,1.3fr);gap:14px;align-items:center;margin-bottom:16px;padding:16px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.builder-layout{display:grid;grid-template-columns:minmax(0,1fr) 320px;gap:16px;align-items:start}.sequence,.toolbox{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.sequence{overflow:hidden}.sequence-head{display:flex;justify-content:space-between;align-items:center;gap:18px;padding:18px 20px;border-bottom:1px solid var(--color-border)}.sequence-head>div{display:grid}.sequence-empty{min-height:280px;display:grid;place-items:center;align-content:center;text-align:center;color:var(--color-text-muted)}.sequence-empty h3{margin:13px 0 4px;color:var(--color-text)}.sequence-empty p{margin:0}.message-card{display:grid;grid-template-columns:46px 1fr;gap:10px;padding:18px;border-bottom:1px solid var(--color-border-subtle);transition:var(--transition-fast)}.message-card.disabled{background:var(--color-surface-soft);opacity:.62}.message-card:last-child{border-bottom:0}.order-controls{display:flex;flex-direction:column;align-items:center}.order-controls strong{width:28px;height:28px;display:grid;place-items:center;color:white;background:var(--color-primary);border-radius:50%;font-size:12px}.message-editor{display:grid;gap:12px;min-width:0}.message-title{display:flex;align-items:center;justify-content:space-between}.message-title>span{display:flex;align-items:center;gap:7px;text-transform:capitalize;font-weight:800}.message-actions{display:flex;align-items:center;gap:4px}.media-preview{width:100%;max-height:240px;border-radius:12px;background:var(--color-background)}video.media-preview{display:block;object-fit:contain}.audio-preview{width:100%}.upload-row{display:grid;grid-template-columns:auto minmax(240px,1fr);align-items:center;gap:12px}.upload-row>span{color:var(--color-text-muted);font-size:11px}.toolbox{position:sticky;top:20px;padding:20px}.toolbox h3{margin:0 0 14px}.toolbox button{width:100%;display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:11px;padding:12px;border:0;border-bottom:1px solid var(--color-border-subtle);color:var(--color-text);background:none;text-align:left;cursor:pointer}.toolbox button:hover{background:var(--color-primary-soft)}.toolbox button>span{width:37px;height:37px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:10px}.toolbox button>div{display:grid}.toolbox button small{color:var(--color-text-muted);font-size:10px}.loading{min-height:400px;display:grid;place-items:center}.inline-action{border:0;color:var(--color-primary);background:none;font-weight:800;cursor:pointer}
@media(max-width:950px){.builder-layout{grid-template-columns:1fr}.toolbox{position:static;order:-1}.name-panel{grid-template-columns:1fr}}
@media(max-width:640px){.page-head{display:grid;padding:20px}.page-head :deep(.v-btn){width:100%}.message-card{grid-template-columns:35px 1fr;padding:14px 10px}.upload-row{grid-template-columns:1fr}.sequence-head{align-items:flex-start}.sequence-head>span{max-width:130px;text-align:right}}
</style>
