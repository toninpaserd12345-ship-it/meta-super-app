<script setup lang="ts">
import type { ReplyItem, ReplyItemType, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

const route = useRoute()
const router = useRouter()
const { can } = useAuth()
const { locale, l } = useLocale()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Reply Sets.' })

const { data: replySet, pending, error, refresh } = await useApi<ReplySet>(`/proxy/api/v1/replies/${String(route.params.id)}`)
const name = ref('')
const items = ref<ReplyItem[]>([])
const saving = ref(false)
const uploadingIndex = ref(-1)
const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')
const previewOpen = ref(false)

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

function duplicateItem(index: number) {
  const item = items.value[index]
  if (!item) return
  items.value.splice(index + 1, 0, {
    ...item,
    content: item.content,
    orderIndex: index + 1,
    isEnabled: true,
  })
}

function insertVariable(index: number, variable: string) {
  if (!items.value[index]) return
  items.value[index].content = (items.value[index].content ? items.value[index].content + ' ' : '') + variable
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
  if (type === 'audio') return 'audio/*, .mp3, .m4a, .wav, .ogg, .opus, .aac, .weba'
  if (type === 'video') return 'video/*, .mp4, .mov, .webm, .3gp'
  if (type === 'image') return 'image/*, .jpg, .jpeg, .png, .webp, .gif'
  return `${type}/*`
}

function errorMessage(error: unknown, fallback: string) {
  const value = error as { data?: { error?: string | { message?: string }; message?: string }; statusMessage?: string }
  const upstream = value.data?.error
  return (typeof upstream === 'string' ? upstream : upstream?.message) || value.data?.message || value.statusMessage || fallback
}

function extractFiles(value: unknown): File[] {
  if (!value) return []
  if (value instanceof File) return [value]
  if (Array.isArray(value)) return value.filter((f): f is File => f instanceof File)
  if (typeof FileList !== 'undefined' && value instanceof FileList) {
    return Array.from(value)
  }
  if (typeof Event !== 'undefined' && value instanceof Event) {
    const target = (value as Event).target as HTMLInputElement
    if (target?.files) return Array.from(target.files)
  }
  if (typeof value === 'object' && value !== null && 'target' in value) {
    const target = (value as any).target
    if (target?.files) return Array.from(target.files)
  }
  return []
}

const recordingIndex = ref(-1)
const mediaRecorder = ref<MediaRecorder | null>(null)
const audioChunks = ref<Blob[]>([])
const recordingTime = ref(0)
let recordingTimer: any = null

async function startRecording(index: number) {
  try {
    const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
    const recorder = new MediaRecorder(stream)
    mediaRecorder.value = recorder
    audioChunks.value = []
    recordingIndex.value = index
    recordingTime.value = 0

    recorder.ondataavailable = (e) => {
      if (e.data.size > 0) audioChunks.value.push(e.data)
    }

    recorder.onstop = async () => {
      stream.getTracks().forEach(track => track.stop())
      if (audioChunks.value.length === 0) return
      const mimeType = recorder.mimeType || 'audio/webm'
      const audioBlob = new Blob(audioChunks.value, { type: mimeType })
      const ext = mimeType.includes('mp4') || mimeType.includes('m4a') ? '.m4a' : mimeType.includes('ogg') ? '.ogg' : '.webm'
      const file = new File([audioBlob], `voice_recording_${Date.now()}${ext}`, { type: mimeType })
      await uploadMedia(file, index)
    }

    recorder.start(200)
    recordingTimer = setInterval(() => {
      recordingTime.value++
    }, 1000)
  } catch (err: any) {
    noticeType.value = 'error'
    notice.value = locale.value === 'lo' ? 'ບໍ່ສາມາດເຂົ້າເຖິງໄມໂຄຣໂຟນໄດ້ ກະລຸນາອະນຸຍາດການໃຊ້ໄມ' : 'Could not access microphone.'
  }
}

function stopRecording() {
  if (mediaRecorder.value && mediaRecorder.value.state !== 'inactive') {
    mediaRecorder.value.stop()
  }
  if (recordingTimer) {
    clearInterval(recordingTimer)
    recordingTimer = null
  }
  recordingIndex.value = -1
}

function triggerNativeUpload(index: number) {
  const el = document.getElementById(`native-file-${index}`) as HTMLInputElement
  el?.click()
}

async function uploadMedia(value: unknown, index: number) {
  const files = extractFiles(value)
  const file = files[0]
  if (!file || !items.value[index]) return
  if (file.size > 25 * 1024 * 1024) {
    noticeType.value = 'error'
    notice.value = locale.value === 'lo' ? 'ຂະໜາດໄຟລ໌ເກີນກຳນົດ (ສູງສຸດ 25MB)' : 'File size exceeds 25MB limit.'
    return
  }
  uploadingIndex.value = index
  notice.value = ''
  try {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('folder', 'replies')
    const result = await $fetch<{ url: string }>('/api/proxy/api/v1/storage/upload', { method: 'POST', body: formData })
    items.value[index].content = result.url
    noticeType.value = 'success'
    notice.value = locale.value === 'lo' ? 'ອັບໂຫລດໄຟລ໌ຂຶ້ນ Cloudflare R2 ສຳເລັດແລ້ວ!' : 'File uploaded to Cloudflare R2 successfully!'
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
        <span>{{ locale==='lo'?'ຈັດລຳດັບຂໍ້ຄວາມຕາມລຳດັບທີ່ຕ້ອງການໃຫ້ລູກຄ້າໄດ້ຮັບ':'Arrange messages in the exact order customers should receive them.' }}</span>
      </div>
      <div class="d-flex align-center gap-2">
        <v-btn variant="outlined" prepend-icon="mdi-eye-outline" @click="previewOpen = true">
          {{ locale==='lo'?'ເບິ່ງຕົວຢ່າງແຊັດ':'Preview Chat' }}
        </v-btn>
        <v-btn color="primary" prepend-icon="mdi-content-save-outline" :loading="saving" :disabled="!can('pages:connect') || !canSave" @click="saveChanges">
          {{ locale==='lo'?'ບັນທຶກຊຸດຂໍ້ຄວາມ':'Save changes' }}
        </v-btn>
      </div>
    </section>

    <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>
    <v-alert v-if="error" type="error" variant="tonal">This Reply Set could not be loaded. <button class="inline-action" @click="refresh()">Try again</button></v-alert>
    <div v-else-if="pending" class="loading"><v-progress-circular indeterminate color="primary"/></div>

    <template v-else-if="replySet">
      <section class="name-panel">
        <v-text-field v-model="name" :label="locale==='lo'?'ຊື່ຊຸດຂໍ້ຄວາມ':'Reply Set name'" variant="outlined" maxlength="120" counter hide-details/>
        <v-alert type="info" variant="tonal" density="compact">
          {{ locale==='lo'?'ຊຸດຂໍ້ຄວາມນີ້ໃຊ້ໄດ້ກັບທຸກຊ່ອງທາງ Messenger & WhatsApp ທີ່ຮ້ານນີ້ເຊື່ອມຕໍ່. ຄລິກປຸ່ມຕົວປ່ຽນລຸ່ມນີ້ເພື່ອດຶງຊື່/ລາຄາສິນຄ້າອັດຕະໂນມັດ.':'This Reply Set belongs to the current store and triggers across all channels. Use the variable buttons below to insert product details dynamically.' }}
        </v-alert>
      </section>

      <div class="builder-layout">
        <main class="sequence" aria-label="Message sequence">
          <div class="sequence-head">
            <div><small>MESSAGE SEQUENCE</small><strong>{{ items.length }} step{{ items.length === 1 ? '' : 's' }}</strong></div>
            <span>{{ locale==='lo'?'ລະບົບຈະສົ່ງຈາກເທິງລົງລຸ່ມ ພ້ອມໜ່ວງເວລາການພິມ 1-2 ວິ':'Sends top to bottom with 1-2s humanized typing simulation' }}</span>
          </div>
          <section v-if="!items.length" class="sequence-empty">
            <v-icon icon="mdi-message-plus-outline" size="32"/>
            <h3>{{ locale==='lo'?'ເລີ່ມຕົ້ນເພີ່ມຂໍ້ຄວາມທຳອິດ':'Add the first message' }}</h3>
            <p>{{ locale==='lo'?'ເລືອກປະເພດຂໍ້ຄວາມຈາກແຖບດ້ານຂວາ':'Choose a message type from the panel on the right.' }}</p>
          </section>
          <article v-for="(item,index) in items" :key="item.id || `${item.type}-${index}`" class="message-card" :class="{ disabled: !item.isEnabled }">
            <div class="order-controls">
              <v-btn icon="mdi-chevron-up" variant="text" size="small" :disabled="index===0" :aria-label="`Move message ${index+1} up`" @click="moveItem(index,-1)"/>
              <strong>{{ index + 1 }}</strong>
              <v-btn icon="mdi-chevron-down" variant="text" size="small" :disabled="index===items.length-1" :aria-label="`Move message ${index+1} down`" @click="moveItem(index,1)"/>
            </div>
            <div class="message-editor">
              <div class="message-title">
                <span><v-icon :icon="messageTypes.find(option=>option.type===item.type)?.icon" size="18"/>{{ item.type }}</span>
                <div class="message-actions">
                  <v-switch v-model="item.isEnabled" color="success" density="compact" hide-details :label="item.isEnabled ? (locale==='lo'?'ສົ່ງ':'Send') : (locale==='lo'?'ຂ້າມ':'Skip')"/>
                  <v-btn icon="mdi-content-copy" variant="text" size="small" color="primary" :title="locale==='lo'?'ຄັດລອກຂັ້ນຕອນນີ້':'Duplicate'" :aria-label="`Duplicate message ${index+1}`" @click="duplicateItem(index)"/>
                  <v-btn icon="mdi-delete-outline" variant="text" size="small" color="error" :aria-label="`Delete message ${index+1}`" @click="removeItem(index)"/>
                </div>
              </div>

              <!-- Variable & Emoji Helper for Text Items -->
              <div v-if="item.type==='text'" class="d-flex align-center gap-1 flex-wrap mb-2">
                <span class="text-caption text-medium-emphasis mr-1">{{ locale==='lo'?'ຕົວປ່ຽນ:':'Variables:' }}</span>
                <v-chip size="x-small" variant="tonal" color="primary" @click="insertVariable(index, '{{product.name}}')">+ &#123;&#123;product.name&#125;&#125;</v-chip>
                <v-chip size="x-small" variant="tonal" color="primary" @click="insertVariable(index, '{{product.price}}')">+ &#123;&#123;product.price&#125;&#125;</v-chip>
                <v-chip size="x-small" variant="tonal" color="primary" @click="insertVariable(index, '{{product.description}}')">+ &#123;&#123;product.description&#125;&#125;</v-chip>
                <span class="text-disabled mx-1">|</span>
                <v-chip v-for="em in ['✨','🔥','📦','💰','🙏','⚡']" :key="em" size="x-small" variant="text" @click="insertVariable(index, em)">{{ em }}</v-chip>
              </div>

              <v-textarea v-if="item.type==='text'" v-model="item.content" :label="locale==='lo'?'ເນື້ອໃນຂໍ້ຄວາມ':'Message text'" variant="outlined" rows="4" auto-grow maxlength="2000" counter hide-details/>
              <template v-else>
                <div v-if="item.content" class="media-container mb-3">
                  <v-img v-if="item.type==='image'" :src="item.content" max-height="240" contain class="media-preview rounded-lg"/>
                  <div v-else-if="item.type==='video'" class="video-preview-wrapper">
                    <video
                      :key="item.content"
                      :src="item.content"
                      controls
                      playsinline
                      preload="metadata"
                      class="media-preview rounded-lg"
                      style="max-height: 240px; width: 100%; border: 1px solid var(--color-border); border-radius: 8px; background: #000;"
                    >
                      <source :src="item.content" type="video/mp4">
                      {{ locale==='lo'?'ບຣາວເຊີຂອງທ່ານບໍ່ຮອງຮັບການຫຼິ້ນວິດີໂອໂດຍກົງ':'Your browser does not support the video tag.' }}
                    </video>
                    <div class="d-flex align-center justify-space-between mt-1 px-1">
                      <a :href="item.content" target="_blank" rel="noopener noreferrer" class="text-caption text-primary d-inline-flex align-center gap-1 text-decoration-none">
                        <v-icon icon="mdi-open-in-new" size="14"/>
                        <span>{{ locale==='lo'?'ເປີດເບິ່ງວິດີໂອ / ດາວໂຫຼດ':'Open / Download video' }}</span>
                      </a>
                    </div>
                  </div>
                  <div v-else-if="item.type==='audio'" class="audio-preview-wrapper">
                    <audio
                      :key="item.content"
                      :src="item.content"
                      controls
                      preload="metadata"
                      class="audio-preview"
                      style="width: 100%; border-radius: 8px;"
                    >
                      <source :src="item.content">
                      {{ locale==='lo'?'ບຣາວເຊີຂອງທ່ານບໍ່ຮອງຮັບການຫຼິ້ນສຽງໂດຍກົງ':'Your browser does not support audio.' }}
                    </audio>
                    <div class="d-flex align-center justify-space-between mt-1 px-1">
                      <a :href="item.content" target="_blank" rel="noopener noreferrer" class="text-caption text-primary d-inline-flex align-center gap-1 text-decoration-none">
                        <v-icon icon="mdi-open-in-new" size="14"/>
                        <span>{{ locale==='lo'?'ເປີດຟັງສຽງ / ດາວໂຫຼດ':'Open / Download audio' }}</span>
                      </a>
                    </div>
                  </div>
                </div>

                <!-- Voice Recording Quick Bar if Audio -->
                <div v-if="item.type==='audio'" class="voice-recorder-bar mb-3">
                  <v-btn
                    v-if="recordingIndex !== index"
                    color="error"
                    variant="tonal"
                    prepend-icon="mdi-microphone"
                    size="small"
                    @click="startRecording(index)"
                  >
                    {{ locale==='lo'?'🎙️ ກົດອັດສຽງດຽວນີ້ (Record Voice)':'🎙️ Record Voice' }}
                  </v-btn>
                  <div v-else class="recording-active-bar d-flex align-center gap-2">
                    <span class="recording-dot"></span>
                    <strong class="text-error font-weight-bold">Recording: {{ recordingTime }}s</strong>
                    <v-btn color="error" size="small" prepend-icon="mdi-stop" @click="stopRecording">
                      {{ locale==='lo'?'ຢຸດ ແລະ ອັບໂຫລດ (Stop & Save)':'Stop & Upload' }}
                    </v-btn>
                  </div>
                </div>

                <v-text-field v-model="item.content" :label="`${item.type} public HTTPS URL`" variant="outlined" prepend-inner-icon="mdi-link" hide-details class="mb-2"/>

                <div class="upload-row">
                  <input
                    :id="`native-file-${index}`"
                    type="file"
                    :accept="acceptFor(item.type)"
                    style="display: none"
                    @change="uploadMedia($event, index)"
                  />
                  <v-btn
                    color="primary"
                    variant="outlined"
                    prepend-icon="mdi-cloud-upload-outline"
                    size="small"
                    :loading="uploadingIndex === index"
                    @click="triggerNativeUpload(index)"
                  >
                    {{ locale==='lo' ? `📁 ເລືອກໄຟລ໌ ${item.type === 'audio' ? 'ສຽງ (Voice)' : item.type === 'video' ? 'ວິດີໂອ (Video)' : 'ຮູບພາບ'} ຂຶ້ນ Cloudflare` : `📁 Choose ${item.type} file` }}
                  </v-btn>
                  <span class="text-caption text-medium-emphasis">({{ item.type === 'audio' ? 'MP3, M4A, WAV, OGG, AAC' : item.type === 'video' ? 'MP4, MOV, WebM, 3GP' : 'JPG, PNG, WebP' }} · Max 25MB)</span>
                </div>
              </template>
            </div>
          </article>
        </main>

        <aside class="toolbox">
          <small>ADD MESSAGE</small><h3>{{ locale==='lo'?'ເລືອກປະເພດຂໍ້ຄວາມ':'Choose a type' }}</h3>
          <button v-for="option in messageTypes" :key="option.type" @click="addItem(option.type)">
            <span><v-icon :icon="option.icon"/></span><div><strong>{{ option.label }}</strong><small>{{ option.description }}</small></div><v-icon icon="mdi-plus" size="18"/>
          </button>
          <v-btn block color="primary" variant="tonal" class="mt-4" prepend-icon="mdi-robot-happy-outline" to="/auto-replies">{{ locale==='lo'?'ນຳໄປໃຊ້ໃນ Automation':'Use in Auto Reply' }}</v-btn>
          <v-btn block variant="outlined" color="info" class="mt-2" prepend-icon="mdi-eye-outline" @click="previewOpen=true">{{ locale==='lo'?'ເບິ່ງຕົວຢ່າງລຳດັບຂໍ້ຄວາມ':'Preview sequence' }}</v-btn>
        </aside>
      </div>

      <!-- Live Preview Dialog -->
      <v-dialog v-model="previewOpen" max-width="480">
        <v-card rounded="xl" class="pa-4">
          <div class="d-flex align-center justify-between mb-3">
            <div class="d-flex align-center gap-2">
              <v-avatar color="primary" size="32"><v-icon icon="mdi-robot-outline" color="white" size="18"/></v-avatar>
              <div>
                <strong class="text-subtitle-2">{{ replySet?.name }}</strong>
                <div class="text-caption text-medium-emphasis">Customer Chat Preview</div>
              </div>
            </div>
            <v-spacer/>
            <v-btn icon="mdi-close" variant="text" size="small" @click="previewOpen=false"/>
          </div>
          <div class="pa-3 rounded-lg" style="background: #f0f2f5; min-height: 280px; display: flex; flex-direction: column; gap: 10px;">
            <div v-for="(item, idx) in items.filter(i => i.isEnabled)" :key="idx" class="d-flex flex-column align-start" style="max-width: 88%;">
              <div class="text-caption text-disabled mb-1 font-weight-bold">Step {{ idx + 1 }} · {{ item.type }} (1-2s delay)</div>
              <div v-if="item.type==='text'" class="pa-3 rounded-xl bg-white text-body-2" style="white-space: pre-wrap; word-break: break-word; color: #1c1e21; border-bottom-left-radius: 4px; box-shadow: 0 1px 2px rgba(0,0,0,0.06);">
                {{ item.content || '(empty message)' }}
              </div>
              <v-img v-else-if="item.type==='image'" :src="item.content" max-height="180" class="rounded-lg bg-white" contain/>
              <video v-else-if="item.type==='video'" :src="item.content" controls playsinline preload="metadata" class="rounded-lg bg-black" style="max-height: 180px; width: 100%; border-radius: 8px;"/>
              <audio v-else-if="item.type==='audio'" :src="item.content" controls class="audio-preview" style="width: 100%; border-radius: 8px;"/>
              <div v-else class="pa-2 rounded bg-white text-caption text-medium-emphasis">
                [{{ item.type }}: {{ item.content }}]
              </div>
            </div>
          </div>
          <div class="text-caption text-medium-emphasis mt-3 text-center">
            {{ locale==='lo'?'ລະບົບຈະສົ່ງຂໍ້ຄວາມຕາມລຳດັບເທື່ອລະອັນ ພ້ອມສະແດງສະຖານະກຳລັງພິມ...':'Messages send sequentially from top to bottom with natural typing indicators.' }}
          </div>
        </v-card>
      </v-dialog>
    </template>
  </div>
</template>

<style scoped>
.back-row{margin-bottom:10px}.page-head{display:flex;align-items:flex-start;justify-content:space-between;gap:20px;margin-bottom:16px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head p,.sequence-head small,.toolbox>small{margin:0 0 6px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 6px;font-size:24px}.page-head span,.sequence-head span{display:block;color:var(--color-text-secondary);font-size:12px}.name-panel{display:grid;grid-template-columns:minmax(260px,.7fr) minmax(320px,1.3fr);gap:14px;align-items:center;margin-bottom:16px;padding:16px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.builder-layout{display:grid;grid-template-columns:minmax(0,1fr) 320px;gap:16px;align-items:start}.sequence,.toolbox{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.sequence{overflow:hidden}.sequence-head{display:flex;justify-content:space-between;align-items:center;gap:18px;padding:18px 20px;border-bottom:1px solid var(--color-border)}.sequence-head>div{display:grid}.sequence-empty{min-height:280px;display:grid;place-items:center;align-content:center;text-align:center;color:var(--color-text-muted)}.sequence-empty h3{margin:13px 0 4px;color:var(--color-text)}.sequence-empty p{margin:0}.message-card{display:grid;grid-template-columns:46px 1fr;gap:10px;padding:18px;border-bottom:1px solid var(--color-border-subtle);transition:var(--transition-fast)}.message-card.disabled{background:var(--color-surface-soft);opacity:.62}.message-card:last-child{border-bottom:0}.order-controls{display:flex;flex-direction:column;align-items:center}.order-controls strong{width:28px;height:28px;display:grid;place-items:center;color:white;background:var(--color-primary);border-radius:50%;font-size:12px}.message-editor{display:grid;gap:12px;min-width:0}.message-title{display:flex;align-items:center;justify-content:space-between}.message-title>span{display:flex;align-items:center;gap:7px;text-transform:capitalize;font-weight:800}.message-actions{display:flex;align-items:center;gap:4px}.media-preview{width:100%;max-height:240px;border-radius:12px;background:var(--color-background)}video.media-preview{display:block;object-fit:contain}.audio-preview{width:100%}.upload-row{display:grid;grid-template-columns:auto minmax(240px,1fr);align-items:center;gap:12px}.upload-row>span{color:var(--color-text-muted);font-size:11px}.toolbox{position:sticky;top:20px;padding:20px}.toolbox h3{margin:0 0 14px}.toolbox button{width:100%;display:grid;grid-template-columns:auto 1fr auto;align-items:center;gap:11px;padding:12px;border:0;border-bottom:1px solid var(--color-border-subtle);color:var(--color-text);background:none;text-align:left;cursor:pointer}.toolbox button:hover{background:var(--color-primary-soft)}.toolbox button>span{width:37px;height:37px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:10px}.toolbox button>div{display:grid}.toolbox button small{color:var(--color-text-muted);font-size:10px}.loading{min-height:400px;display:grid;place-items:center}.inline-action{border:0;color:var(--color-primary);background:none;font-weight:800;cursor:pointer}
.voice-recorder-bar{display:flex;align-items:center;gap:10px}.recording-active-bar{background:#fee2e2;padding:6px 12px;border-radius:20px;border:1px solid #fca5a5}.recording-dot{width:10px;height:10px;background:#dc2626;border-radius:50%;animation:pulse-red 1s infinite}@keyframes pulse-red{0%{transform:scale(0.95);box-shadow:0 0 0 0 rgba(220,38,38,0.7)}70%{transform:scale(1.1);box-shadow:0 0 0 6px rgba(220,38,38,0)}100%{transform:scale(0.95);box-shadow:0 0 0 0 rgba(220,38,38,0)}}
@media(max-width:950px){.builder-layout{grid-template-columns:1fr}.toolbox{position:static;order:-1}.name-panel{grid-template-columns:1fr}}
@media(max-width:640px){.page-head{display:grid;padding:20px}.page-head :deep(.v-btn){width:100%}.message-card{grid-template-columns:35px 1fr;padding:14px 10px}.upload-row{grid-template-columns:1fr}.sequence-head{align-items:flex-start}.sequence-head>span{max-width:130px;text-align:right}}
</style>
