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

const search = ref('')
const filteredSets = computed(() => {
  const query = search.value.trim().toLowerCase()
  if (!query) return sets.value
  return sets.value.filter(s => s.name.toLowerCase().includes(query))
})

function copyTriggerCode(name: string) {
  navigator.clipboard.writeText(`#${name}`)
  noticeType.value = 'success'
  notice.value = locale.value === 'lo' ? `ຄັດລອກລະຫັດ #${name} ສຳເລັດ! ພ້ອມນຳໄປພິມໃນແຊັດ.` : `คัดลอกรหัส #${name} สำเร็จ! พร้อมนำไปพิมพ์ในแชท.`
}

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
      <span>{{locale==='lo'?'ຈັດລຳດັບຂໍ້ຄວາມ, ຮູບ, ວິດີໂອ ແລະສຽງ ແລ້ວນຳໄປໃຊ້ກັບ Automation ຫຼືພິມໂຄດ #ຊື່ ໃນແຊັດເພື່ອສົ່ງທັນທີ':'Build reusable message sequences, trigger via #code in Live Chat, or connect to Posts & Ads.'}}</span>
    </div>
    <div class="head-actions">
      <v-btn variant="outlined" prepend-icon="mdi-robot-happy-outline" to="/auto-replies">{{ locale==='lo'?'ຈັດການ Automation':'Manage Automations' }}</v-btn>
      <v-btn variant="tonal" color="info" prepend-icon="mdi-chat-outline" to="/live-chat">{{ locale==='lo'?'ໄປທີ່ Live Chat':'Open Live Chat' }}</v-btn>
      <v-btn color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreateDialog">{{ locale==='lo'?'ສ້າງຊຸດຂໍ້ຄວາມໃໝ່':'Create Reply Set' }}</v-btn>
    </div>
  </section>

  <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>

  <!-- Quick Filter & Search Bar -->
  <v-card class="mb-4 pa-3" variant="outlined" rounded="lg">
    <div class="d-flex align-center gap-3 flex-wrap">
      <v-text-field
        v-model="search"
        density="compact"
        variant="outlined"
        prepend-inner-icon="mdi-magnify"
        :placeholder="locale==='lo'?'ຄົ້ນຫາຊຸດຂໍ້ຄວາມ...':'Search reply sets by name...'"
        hide-details
        clearable
        style="max-width: 400px; min-width: 240px;"
      />
      <v-chip size="small" variant="tonal" color="primary">
        {{ filteredSets.length }} / {{ sets.length }} {{ locale==='lo'?'ລາຍການ':'sets' }}
      </v-chip>
      <v-spacer />
      <span class="text-caption text-medium-emphasis">
        <v-icon icon="mdi-lightbulb-outline" size="small" class="mr-1" />
        {{ locale==='lo'?'ຄລິກປຸ່ມໂຄດເພື່ອຄັດລອກ #ລະຫັດ ໄປພິມໃນແຊັດໄດ້ທັນທີ':'Click #code chip to copy quick-trigger code for Live Chat' }}
      </span>
    </div>
  </v-card>

  <v-alert v-if="error" type="error" variant="tonal" class="mb-4">
    Reply Sets could not be loaded. <button class="inline-action" @click="refresh()">Try again</button>
  </v-alert>

  <section v-if="pending" class="set-grid" aria-busy="true">
    <v-skeleton-loader v-for="index in 3" :key="index" type="card"/>
  </section>

  <section v-else-if="filteredSets.length" class="set-grid">
    <article v-for="set in filteredSets" :key="set.id" class="set-card" @click="router.push(`/replies/${set.id}`)">
      <div class="set-icon"><v-icon icon="mdi-message-text-fast-outline"/></div>
      <div class="set-copy">
        <div class="d-flex align-center gap-2 flex-wrap mb-1">
          <h3 class="mb-0">{{ set.name }}</h3>
          <!-- 1-Click Copy Trigger Code -->
          <v-tooltip :text="locale==='lo'?'ຄລິກເພື່ອຄັດລອກລະຫັດສົ່ງດ່ວນ':'Click to copy trigger code'" location="top">
            <template #activator="{ props: tipProps }">
              <v-chip
                v-bind="tipProps"
                size="x-small"
                color="primary"
                variant="flat"
                prepend-icon="mdi-pound"
                class="font-weight-bold"
                @click.stop="copyTriggerCode(set.name)"
              >
                #{{ set.name }}
              </v-chip>
            </template>
          </v-tooltip>
        </div>
        <p>{{ set.items?.length || 0 }} message{{ (set.items?.length || 0) === 1 ? '' : 's' }}</p>
        <div class="chips">
          <v-chip v-for="(count, type) in itemStats(set.items || [])" :key="type" size="x-small" color="secondary" variant="tonal">{{ count }} {{ type }}</v-chip>
          <span v-if="!set.items?.length" class="text-caption text-disabled">Empty set — open to add messages</span>
        </div>
      </div>
      <div class="set-actions d-flex flex-column gap-1">
        <div class="d-flex align-center gap-1">
          <v-btn icon="mdi-pencil-outline" variant="text" size="small" :aria-label="`Edit ${set.name}`" @click.stop="router.push(`/replies/${set.id}`)"/>
          <v-btn icon="mdi-delete-outline" variant="text" size="small" color="error" :disabled="!can('pages:connect')" :aria-label="`Delete ${set.name}`" @click.stop="requestDelete(set)"/>
        </div>
        <div class="d-flex align-center gap-1 mt-1">
          <v-btn
            size="x-small"
            variant="tonal"
            color="primary"
            prepend-icon="mdi-robot-happy-outline"
            to="/auto-replies"
            @click.stop
          >
            Auto
          </v-btn>
          <v-btn
            size="x-small"
            variant="tonal"
            color="info"
            prepend-icon="mdi-chat-outline"
            to="/live-chat"
            @click.stop
          >
            Chat
          </v-btn>
        </div>
      </div>
    </article>
  </section>

  <section v-else-if="!error" class="empty-state">
    <div><v-icon icon="mdi-message-plus-outline" size="30"/></div>
    <h2>{{ search ? (locale==='lo'?'ບໍ່ພົບຊຸດຂໍ້ຄວາມທີ່ຄົ້ນຫາ':'No matching Reply Sets') : (locale==='lo'?'ສ້າງຊຸດຂໍ້ຄວາມທຳອິດຂອງທ່ານ':'Create your first Reply Set') }}</h2>
    <p>{{ search ? (locale==='lo'?'ລອງປ່ຽນຄຳຄົ້ນຫາໃໝ່':'Try adjusting your search query') : (locale==='lo'?'ເພີ່ມຂໍ້ຄວາມ, ຮູບ, ວິດີໂອຕາມລຳດັບທີ່ຕ້ອງການສົ່ງໃຫ້ລູກຄ້າ':'Add text, images, video or audio in the exact order customers should receive them.') }}</p>
    <v-btn v-if="!search" color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="openCreateDialog">{{ locale==='lo'?'ສ້າງຊຸດຂໍ້ຄວາມ':'Create Reply Set' }}</v-btn>
    <v-btn v-else variant="outlined" @click="search=''">{{ locale==='lo'?'ລ້າງການຄົ້ນຫາ':'Clear search' }}</v-btn>
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
