<script setup lang="ts">
import type { ReplyItem, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })

const router = useRouter()
const { can } = useAuth()
const { locale } = useLocale()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Reply Sets.' })

const { data: setData, pending, error, refresh } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const sets = computed(() => setData.value || [])

const createDialog = ref(false)
const deleteDialog = ref(false)
const newSetName = ref('')
const creating = ref(false)
const deleting = ref(false)
const setToDelete = ref<ReplySet | null>(null)
const notice = ref('')
const noticeType = ref<'success' | 'error'>('success')

function openCreateDialog() {
  newSetName.value = ''
  createDialog.value = true
}

function itemStats(items: ReplyItem[]) {
  return items.reduce<Record<string, number>>((result, item) => {
    result[item.type] = (result[item.type] || 0) + 1
    return result
  }, {})
}

function errorMessage(error: unknown, fallback: string) {
  const value = error as { data?: { error?: string | { message?: string }; message?: string }; statusMessage?: string }
  const upstream = value.data?.error
  return (typeof upstream === 'string' ? upstream : upstream?.message) || value.data?.message || value.statusMessage || fallback
}

async function createSet() {
  const name = newSetName.value.trim()
  if (!name) return
  creating.value = true
  notice.value = ''
  try {
    const created = await $fetch<ReplySet>('/api/proxy/api/v1/replies', {
      method: 'POST',
      body: { name },
    })
    createDialog.value = false
    newSetName.value = ''
    await router.push(`/replies/${created.id}`)
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, 'Unable to create the Reply Set.')
  } finally {
    creating.value = false
  }
}

function requestDelete(set: ReplySet) {
  setToDelete.value = set
  deleteDialog.value = true
}

async function deleteSet() {
  if (!setToDelete.value) return
  deleting.value = true
  notice.value = ''
  try {
    await $fetch(`/api/proxy/api/v1/replies/${setToDelete.value.id}`, { method: 'DELETE' })
    deleteDialog.value = false
    noticeType.value = 'success'
    notice.value = `Deleted “${setToDelete.value.name}”.`
    await refresh()
  } catch (error) {
    noticeType.value = 'error'
    notice.value = errorMessage(error, 'Unable to delete the Reply Set.')
  } finally {
    deleting.value = false
    setToDelete.value = null
  }
}
</script>

<template>
  <section class="page-head">
    <div>
      <p>AUTO REPLY LIBRARY</p>
      <h2>{{locale==='lo'?'ຊຸດຂໍ້ຄວາມຕອບກັບ':'Reply Sets'}}</h2>
      <span>{{locale==='lo'?'ຈັດລຳດັບຂໍ້ຄວາມ, ຮູບ, ວິດີໂອ ແລະສຽງ ແລ້ວນຳໄປໃຊ້ກັບ Automation':'Build reusable message sequences, then connect them to Posts or Ads.'}}</span>
    </div>
    <div class="head-actions">
      <v-btn variant="outlined" prepend-icon="mdi-robot-happy-outline" to="/auto-replies">Manage Auto Replies</v-btn>
      <v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreateDialog">Create Reply Set</v-btn>
    </div>
  </section>

  <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>
  <v-alert type="info" variant="tonal" class="mb-4" icon="mdi-keyboard-outline">
    {{ locale==='lo' ? 'ຊຸດຂໍ້ຄວາມນີ້ເປັນຂອງຮ້ານ/Workspace ປັດຈຸບັນ ແລະໃຊ້ໄດ້ກັບທຸກ Messenger ຫຼື WhatsApp ທີ່ຮ້ານນີ້ເຊື່ອມຕໍ່. ເມື່ອລູກຄ້າພິມຊື່ຊຸດຕົງກັນ ລະບົບຈະສົ່ງຊຸດນັ້ນທັນທີ.' : 'Reply Sets belong to the current store/workspace and work across every Messenger or WhatsApp channel connected to that store. Sending the exact set name triggers the sequence immediately.' }}
  </v-alert>
  <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
    Reply Sets could not be loaded. <button class="inline-action" @click="refresh()">Try again</button>
  </v-alert>

  <section v-if="pending" class="set-grid" aria-busy="true">
    <v-skeleton-loader v-for="index in 3" :key="index" type="card"/>
  </section>

  <section v-else-if="sets.length" class="set-grid">
    <article v-for="set in sets" :key="set.id" class="set-card" @click="router.push(`/replies/${set.id}`)">
      <div class="set-icon"><v-icon icon="mdi-message-text-fast-outline"/></div>
      <div class="set-copy">
        <h3>{{ set.name }}</h3>
        <p>{{ set.items?.length || 0 }} message{{ (set.items?.length || 0) === 1 ? '' : 's' }}</p>
        <div class="chips">
          <v-chip v-for="(count, type) in itemStats(set.items || [])" :key="type" size="x-small" color="primary" variant="tonal">{{ count }} {{ type }}</v-chip>
          <span v-if="!set.items?.length">Empty set — open to add the first message</span>
        </div>
      </div>
      <div class="set-actions">
        <v-btn icon="mdi-pencil-outline" variant="text" size="small" :aria-label="`Edit ${set.name}`" @click.stop="router.push(`/replies/${set.id}`)"/>
        <v-btn icon="mdi-delete-outline" variant="text" size="small" color="error" :disabled="!can('pages:connect')" :aria-label="`Delete ${set.name}`" @click.stop="requestDelete(set)"/>
      </div>
    </article>
  </section>

  <section v-else-if="!error" class="empty-state">
    <div><v-icon icon="mdi-message-plus-outline" size="30"/></div>
    <h2>Create your first Reply Set</h2>
    <p>Add text, images, video or audio in the exact order customers should receive them.</p>
    <v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreateDialog">Create Reply Set</v-btn>
  </section>

  <v-dialog v-model="createDialog" max-width="500">
    <v-card class="dialog-card">
      <v-card-title>Create Auto Reply Set</v-card-title>
      <v-card-text>
        <v-text-field v-model="newSetName" label="Set name" placeholder="Name this Reply Set" autofocus variant="outlined" maxlength="120" counter @keyup.enter="createSet"/>
        <p class="dialog-hint">Customers can send this exact name to receive the sequence automatically.</p>
      </v-card-text>
      <v-card-actions>
        <v-btn variant="text" @click="createDialog=false">Cancel</v-btn><v-spacer/>
        <v-btn color="primary" :loading="creating" :disabled="!newSetName.trim()" @click="createSet">Create and edit</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>

  <v-dialog v-model="deleteDialog" max-width="430">
    <v-card class="dialog-card">
      <v-card-title>Delete Reply Set?</v-card-title>
      <v-card-text>“{{ setToDelete?.name }}” will no longer be available for new Auto Replies. This action cannot be undone.</v-card-text>
      <v-card-actions>
        <v-btn variant="text" @click="deleteDialog=false">Cancel</v-btn><v-spacer/>
        <v-btn color="error" :loading="deleting" @click="deleteSet">Delete</v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<style scoped>
.page-head{display:flex;align-items:flex-start;justify-content:space-between;gap:24px;margin-bottom:18px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head p{margin:0 0 7px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 7px;font-size:24px}.page-head span{color:var(--color-text-secondary);font-size:13px}.head-actions{display:flex;gap:10px;flex-wrap:wrap;justify-content:flex-end}.set-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.set-card{display:grid;grid-template-columns:auto 1fr auto;gap:15px;align-items:start;min-width:0;padding:20px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);cursor:pointer;transition:var(--transition-fast)}.set-card:hover{border-color:color-mix(in srgb,var(--color-primary) 45%,var(--color-border));box-shadow:var(--shadow-card);transform:translateY(-2px)}.set-icon{width:48px;height:48px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:14px}.set-copy{min-width:0}.set-copy h3{margin:1px 0 4px;font-size:15px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.set-copy p{margin:0 0 12px;color:var(--color-text-secondary);font-size:12px}.chips{display:flex;align-items:center;gap:6px;flex-wrap:wrap}.chips>span{color:var(--color-text-muted);font-size:11px}.set-actions{display:flex}.empty-state{min-height:390px;display:grid;place-items:center;align-content:center;text-align:center;padding:36px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.empty-state>div{width:58px;height:58px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:17px}.empty-state h2{margin:18px 0 6px}.empty-state p{max-width:480px;margin:0 0 20px;color:var(--color-text-secondary);line-height:1.6}.inline-action{border:0;color:var(--color-primary);background:none;font-weight:800;cursor:pointer}.dialog-card{border-radius:var(--radius-lg)!important}.dialog-hint{margin:0;color:var(--color-text-secondary);font-size:12px;line-height:1.5}
@media(max-width:900px){.set-grid{grid-template-columns:1fr}}
@media(max-width:640px){.page-head{display:grid;padding:20px}.head-actions{display:grid;grid-template-columns:1fr 1fr;width:100%}.set-card{grid-template-columns:auto 1fr}.set-actions{grid-column:1/-1;justify-content:flex-end}}
</style>
