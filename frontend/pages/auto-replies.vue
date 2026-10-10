<script setup lang="ts">
import type { AutomationFlow, AutomationTarget, ItemsResponse, MetaAdAccount, MetaCampaign, MetaPage, MetaPost, Product, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { can } = useAuth()
const { locale, t } = useLocale()
const { cachedFetch } = useApiClient()
if (!can('pages:read')) throw createError({ statusCode: 403, statusMessage: 'You do not have permission to view Auto Replies.' })

const { data: pageData } = await useApi<ItemsResponse<MetaPage>>('/proxy/api/v1/meta/pages')
const { data: productData } = await useApi<ItemsResponse<Product>>('/proxy/api/v1/products')
const { data: setData } = await useApi<ReplySet[]>('/proxy/api/v1/replies')
const { data: flowData, refresh: refreshFlows } = await useApi<ItemsResponse<AutomationFlow>>('/proxy/api/v1/automations')
const pages = computed(() => pageData.value?.items.filter(item => item.connected) || [])
const products = computed(() => productData.value?.items || [])
const replySets = computed(() => setData.value || [])
const flows = computed(() => flowData.value?.items || [])
const usableSets = computed(() => replySets.value.filter(set => set.items?.some(item => item.isEnabled && item.content.trim())))

const view = ref<'dashboard' | 'wizard'>('dashboard')
const step = ref(1)
const name = ref('')
const pageId = ref('')
const productId = ref('')
const replySetId = ref('')
const targetType = ref<'post' | 'campaign'>('post')
const targets = ref<AutomationTarget[]>([])
const posts = ref<MetaPost[]>([])
const adAccounts = ref<MetaAdAccount[]>([])
const campaigns = ref<MetaCampaign[]>([])
const adAccountId = ref('')
const firstMessageOnly = ref(true)
const cooldownSeconds = ref(0)
const loading = ref(false)
const saving = ref(false)
const deletingId = ref('')
const editingId = ref('')
const expandedId = ref('')
const hydrating = ref(false)
const search = ref('')
const notice = ref('')
const noticeType = ref<'success' | 'error' | 'warning'>('success')

const selectedProduct = computed(() => products.value.find(item => item.id === productId.value))
const selectedSet = computed(() => replySets.value.find(item => item.id === replySetId.value))
const selectedPage = computed(() => pages.value.find(item => item.id === pageId.value))
const selectedIds = computed(() => new Set(targets.value.map(item => `${item.type}:${item.value}`)))
const filteredFlows = computed(() => {
  const value = search.value.trim().toLowerCase()
  if (!value) return flows.value
  return flows.value.filter(flow => [flow.name, productName(flow.productId), setName(flow.replySetId), pageName(flow.pageId), ...flow.targets.map(item => item.name || item.value)].join(' ').toLowerCase().includes(value))
})
const stepReady = computed(() => step.value === 1 ? Boolean(name.value.trim() && pageId.value) : step.value === 2 ? targets.value.length > 0 : step.value === 3 ? Boolean(productId.value && replySetId.value) : true)

function productName(id: string) { return products.value.find(item => item.id === id)?.name || 'No product' }
function setName(id: string) { return replySets.value.find(item => item.id === id)?.name || 'Missing Reply Set' }
function pageName(id: string) { return pages.value.find(item => item.id === id)?.name || id }
function apiError(error: unknown, fallback: string) {
  const value = error as { data?: { error?: string | { message?: string }; message?: string }; statusMessage?: string }
  const dataError = value.data?.error
  return (typeof dataError === 'string' ? dataError : dataError?.message) || value.data?.message || value.statusMessage || fallback
}
function startWizard() {
  editingId.value = ''
  name.value = ''; pageId.value = pages.value[0]?.id || ''; productId.value = products.value[0]?.id || ''; replySetId.value = usableSets.value[0]?.id || ''
  targetType.value = 'post'; targets.value = []; firstMessageOnly.value = true; cooldownSeconds.value = 0; step.value = 1; notice.value = ''; view.value = 'wizard'
}
async function editFlow(flow: AutomationFlow) {
  hydrating.value = true
  editingId.value = flow.id; name.value = flow.name; pageId.value = flow.pageId; productId.value = flow.productId; replySetId.value = flow.replySetId
  targetType.value = flow.targets[0]?.type === 'campaign' ? 'campaign' : 'post'; targets.value = flow.targets.map(({ type, value, name }) => ({ type, value, name }))
  firstMessageOnly.value = flow.firstMessageOnly; cooldownSeconds.value = flow.cooldownSeconds; step.value = 1; notice.value = ''; view.value = 'wizard'
  await nextTick()
  hydrating.value = false
  if (targetType.value === 'post') await loadPosts(); else await loadAdAccounts()
}
function toggleTarget(target: AutomationTarget) {
  const key = `${target.type}:${target.value}`
  targets.value = selectedIds.value.has(key) ? targets.value.filter(item => `${item.type}:${item.value}` !== key) : [...targets.value, target]
}
function isSelected(type: string, value: string) { return selectedIds.value.has(`${type}:${value}`) }
function isUsed(type: string, value: string) { return flows.value.some(flow => flow.id !== editingId.value && flow.pageId === pageId.value && flow.targets.some(target => target.type === type && target.value === value)) }
function next() { if (stepReady.value && step.value < 5) step.value++ }
function back() { if (step.value > 1) step.value--; else view.value = 'dashboard' }

async function loadPosts() {
  if (!pageId.value) return
  loading.value = true
  try { posts.value = (await cachedFetch<ItemsResponse<MetaPost>>(`/api/proxy/api/v1/meta/pages/${pageId.value}/posts`, 120_000)).items }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load Facebook posts.') }
  finally { loading.value = false }
}
async function loadAdAccounts() {
  if (adAccounts.value.length) return
  loading.value = true
  try { adAccounts.value = (await cachedFetch<ItemsResponse<MetaAdAccount>>('/api/proxy/api/v1/meta/ad-accounts', 300_000)).items; adAccountId.value = adAccounts.value[0]?.id || '' }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load Ad Accounts. Reconnect Facebook with ads_read permission.') }
  finally { loading.value = false }
}
async function loadCampaigns() {
  campaigns.value = []
  if (!adAccountId.value) return
  loading.value = true
  try { campaigns.value = (await cachedFetch<ItemsResponse<MetaCampaign>>(`/api/proxy/api/v1/meta/ad-accounts/${adAccountId.value}/campaigns`, 120_000)).items }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load campaigns.') }
  finally { loading.value = false }
}
async function saveFlow() {
  saving.value = true; notice.value = ''
  try {
    const url = editingId.value ? `/api/proxy/api/v1/automations/${editingId.value}` : '/api/proxy/api/v1/automations'
    await $fetch(url, { method: editingId.value ? 'PUT' : 'POST', body: { name: name.value, pageId: pageId.value, productId: productId.value, replySetId: replySetId.value, firstMessageOnly: firstMessageOnly.value, cooldownSeconds: cooldownSeconds.value, targets: targets.value } })
    const wasEditing = Boolean(editingId.value)
    await refreshFlows(); noticeType.value = 'success'; notice.value = wasEditing ? (locale.value === 'lo' ? 'ບັນທຶກ Automation ແລ້ວ' : 'Automation updated.') : (locale.value === 'lo' ? 'ສ້າງ Automation ແລ້ວ' : 'Automation created.'); view.value = 'dashboard'; editingId.value = ''
  } catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not create the automation.') }
  finally { saving.value = false }
}
async function setStatus(flow: AutomationFlow, active: boolean) {
  try { await $fetch(`/api/proxy/api/v1/automations/${flow.id}/status`, { method: 'PATCH', body: { isActive: active } }); await refreshFlows() }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not update this automation.') }
}
async function deleteFlow(flow: AutomationFlow) {
  if (!confirm(locale.value === 'lo' ? `ລຶບ “${flow.name}” ແລະ ${flow.targets.length} ເປົ້າໝາຍທັງໝົດບໍ?` : `Delete “${flow.name}” and all ${flow.targets.length} targets?`)) return
  deletingId.value = flow.id
  try { await $fetch(`/api/proxy/api/v1/automations/${flow.id}`, { method: 'DELETE' }); await refreshFlows(); noticeType.value = 'success'; notice.value = 'Automation deleted.' }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not delete this automation.') }
  finally { deletingId.value = '' }
}

watch(pageId, () => { if (hydrating.value) return; targets.value = []; if (targetType.value === 'post') loadPosts() })
watch(targetType, value => { if (hydrating.value) return; targets.value = []; if (value === 'post') loadPosts(); else loadAdAccounts() })
watch(adAccountId, loadCampaigns)
</script>

<template>
  <section class="page-head">
    <div><p>{{ t('auto.eyebrow') }}</p><h2>{{ view === 'dashboard' ? t('auto.title') : editingId ? t('auto.editTitle') : t('auto.newTitle') }}</h2><span>{{ t('auto.subtitle') }}</span></div>
    <div class="head-actions"><v-btn variant="text" to="/automation-guide" prepend-icon="mdi-book-open-page-variant-outline">{{ t('auto.guide') }}</v-btn><v-btn v-if="view==='dashboard'" color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="startWizard">{{ t('auto.new') }}</v-btn></div>
  </section>
  <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>

  <template v-if="view==='dashboard'">
    <section class="summary-grid"><article><v-icon icon="mdi-robot-happy-outline"/><div><strong>{{ flows.length }}</strong><span>{{ t('auto.list') }}</span></div></article><article><v-icon icon="mdi-target"/><div><strong>{{ flows.reduce((sum,item)=>sum+item.targets.length,0) }}</strong><span>{{ t('auto.connected') }}</span></div></article><article><v-icon icon="mdi-check-circle-outline"/><div><strong>{{ flows.filter(item=>item.isActive).length }}</strong><span>{{ t('common.active') }}</span></div></article></section>
    <v-alert v-if="!products.length || !usableSets.length || !pages.length" type="info" variant="tonal" class="mb-4">{{ t('auto.setupNeeded') }}</v-alert>
    <section class="rules-panel"><div class="panel-head"><div><small>LIVE CONFIGURATION</small><h3>{{ t('auto.list') }}</h3></div><v-text-field v-model="search" density="compact" variant="outlined" prepend-inner-icon="mdi-magnify" :placeholder="t('auto.search')" hide-details/></div>
      <div v-if="filteredFlows.length" class="rule-list"><div v-for="flow in filteredFlows" :key="flow.id" class="flow-block"><article class="flow-row"><div class="flow-icon"><v-icon icon="mdi-robot-happy-outline"/></div><div class="rule-main"><strong>{{ flow.name }}</strong><span>{{ productName(flow.productId) }} → {{ setName(flow.replySetId) }}</span><small>{{ pageName(flow.pageId) }} · {{ flow.targets.length }} {{ t('common.targets') }} · {{ flow.firstMessageOnly ? t('auto.firstOnly') : t('auto.everyMatch') }}</small><div class="chips"><v-chip v-for="target in flow.targets.slice(0,3)" :key="target.id || `${target.type}:${target.value}`" size="x-small" variant="tonal">{{ target.type }} · {{ target.name || target.value }}</v-chip><v-chip v-if="flow.targets.length>3" size="x-small">+{{ flow.targets.length-3 }}</v-chip></div></div><v-switch :model-value="flow.isActive" color="success" hide-details density="compact" :aria-label="t('common.active')" :disabled="!can('pages:connect')" @update:model-value="setStatus(flow,Boolean($event))"/><div class="flow-actions"><v-btn icon="mdi-pencil-outline" :aria-label="t('common.edit')" variant="text" :disabled="!can('pages:connect')" @click="editFlow(flow)"/><v-btn :icon="expandedId===flow.id?'mdi-chevron-up':'mdi-chevron-down'" :aria-label="t('common.details')" variant="text" @click="expandedId=expandedId===flow.id?'':flow.id"/><v-btn icon="mdi-delete-outline" :aria-label="t('common.delete')" variant="text" color="error" :loading="deletingId===flow.id" :disabled="!can('pages:connect')" @click="deleteFlow(flow)"/></div></article><section v-if="expandedId===flow.id" class="flow-details"><div><span>{{ t('auto.page') }}</span><strong>{{ pageName(flow.pageId) }}</strong></div><div><span>{{ t('auto.product') }}</span><strong>{{ productName(flow.productId) }}</strong></div><div><span>{{ t('auto.replySet') }}</span><strong>{{ setName(flow.replySetId) }}</strong></div><div><span>{{ t('auto.cooldown') }}</span><strong>{{ flow.cooldownSeconds }}s</strong></div><div class="detail-targets"><span>{{ t('common.targets') }}</span><div class="chips"><v-chip v-for="target in flow.targets" :key="target.id || `${target.type}:${target.value}`" size="small" color="primary" variant="tonal">{{ target.type }} · {{ target.name || target.value }}</v-chip></div></div><v-btn color="primary" variant="tonal" prepend-icon="mdi-pencil-outline" :disabled="!can('pages:connect')" @click="editFlow(flow)">{{ t('common.edit') }}</v-btn></section></div></div>
      <div v-else class="empty"><v-icon icon="mdi-robot-happy-outline" size="38"/><h3>{{ t('auto.empty') }}</h3><p>{{ t('auto.emptyHelp') }}</p><v-btn color="primary" prepend-icon="mdi-plus" :disabled="!products.length || !usableSets.length || !pages.length" @click="startWizard">{{ t('auto.new') }}</v-btn><v-btn variant="text" to="/automation-guide">{{ t('auto.guide') }}</v-btn></div>
    </section>
  </template>

  <template v-else>
    <nav class="steps"><button v-for="item in [{n:1,label:t('auto.step.setup')},{n:2,label:t('auto.step.trigger')},{n:3,label:t('auto.step.response')},{n:4,label:t('auto.step.behavior')},{n:5,label:t('auto.step.review')}]" :key="item.n" :class="{active:step===item.n,done:step>item.n}" :disabled="item.n>step" @click="step=item.n"><span>{{ step>item.n?'✓':item.n }}</span><b>{{ item.label }}</b></button></nav>
    <section class="wizard-card">
      <template v-if="step===1"><div class="section-title"><small>STEP 1</small><h3>{{ t('auto.setupTitle') }}</h3><p>{{ t('auto.setupHelp') }}</p></div><div class="form-grid"><v-text-field v-model="name" :label="t('auto.name')" :placeholder="t('auto.nameExample')" variant="outlined"/><v-select v-model="pageId" :items="pages" item-title="name" item-value="id" :label="t('auto.page')" variant="outlined"/></div></template>
      <template v-else-if="step===2"><div class="section-title"><small>STEP 2</small><h3>{{ t('auto.triggerTitle') }}</h3><p>{{ t('auto.triggerHelp') }}</p></div><v-btn-toggle v-model="targetType" mandatory color="primary" class="target-toggle"><v-btn value="post" prepend-icon="mdi-post-outline">{{ t('auto.posts') }}</v-btn><v-btn value="campaign" prepend-icon="mdi-bullhorn-outline">{{ t('auto.campaigns') }}</v-btn></v-btn-toggle><div v-if="loading" class="loading"><v-progress-circular indeterminate color="primary"/></div><div v-else-if="targetType==='post'" class="target-list"><button v-for="post in posts" :key="post.id" :disabled="isUsed('post',post.id)" :class="{selected:isSelected('post',post.id),disabled:isUsed('post',post.id)}" @click="toggleTarget({type:'post',value:post.id,name:post.message || 'Facebook Post'})"><v-checkbox-btn :model-value="isSelected('post',post.id)"/><div><strong>{{ post.message || 'Facebook Post' }}</strong><span>{{ post.id }}</span><small v-if="isUsed('post',post.id)">{{ t('auto.alreadyUsed') }}</small></div></button></div><template v-else><v-select v-model="adAccountId" :items="adAccounts.map(item=>({...item,label:`${item.accountType==='business'?'Business':'Personal'} · ${item.name}`}))" item-title="label" item-value="id" :label="t('auto.adAccount')" variant="outlined"/><div class="target-list"><button v-for="campaign in campaigns" :key="campaign.id" :disabled="isUsed('campaign',campaign.id)" :class="{selected:isSelected('campaign',campaign.id),disabled:isUsed('campaign',campaign.id)}" @click="toggleTarget({type:'campaign',value:campaign.id,name:campaign.name})"><v-checkbox-btn :model-value="isSelected('campaign',campaign.id)"/><div><strong>{{ campaign.name }}</strong><span>{{ campaign.adSetCount }} ad sets · {{ campaign.adCount }} ads</span><small>{{ isUsed('campaign',campaign.id)?t('auto.alreadyUsed'):campaign.effectiveStatus }}</small></div></button></div></template></template>
      <template v-else-if="step===3"><div class="section-title"><small>STEP 3</small><h3>{{ t('auto.responseTitle') }}</h3><p>{{ t('auto.responseHelp') }}</p></div><div class="form-grid"><v-select v-model="productId" :items="products" item-title="name" item-value="id" :label="t('auto.product')" variant="outlined"/><v-select v-model="replySetId" :items="usableSets" item-title="name" item-value="id" :label="t('auto.replySet')" variant="outlined"/></div><div v-if="selectedSet" class="response-preview"><strong>{{ selectedSet.name }}</strong><span>{{ selectedSet.items.filter(item=>item.isEnabled).length }} {{ t('auto.enabledMessages') }}</span><small>{{ t('auto.sendOrder') }}</small></div></template>
      <template v-else-if="step===4"><div class="section-title"><small>STEP 4</small><h3>{{ t('auto.behaviorTitle') }}</h3><p>{{ t('auto.behaviorHelp') }}</p></div><v-switch v-model="firstMessageOnly" color="primary" :label="t('auto.firstLabel')" inset/><v-text-field v-model.number="cooldownSeconds" type="number" min="0" max="86400" :label="t('auto.cooldown')" :hint="t('auto.cooldownHelp')" persistent-hint variant="outlined"/></template>
      <template v-else><div class="section-title"><small>STEP 5</small><h3>{{ t('auto.reviewTitle') }}</h3><p>{{ t('auto.reviewHelp') }}</p></div><div class="review"><article><span>{{ t('auto.automation') }}</span><strong>{{ name }}</strong><small>{{ selectedPage?.name }}</small></article><article><span>{{ t('auto.response') }}</span><strong>{{ selectedProduct?.name }}</strong><small>{{ selectedSet?.name }}</small></article><article><span>{{ t('common.targets') }}</span><strong>{{ targets.length }} {{ targetType }}</strong><small>{{ firstMessageOnly?t('auto.firstOnly'):t('auto.everyMatch') }}</small></article></div><div class="chips"><v-chip v-for="target in targets" :key="`${target.type}:${target.value}`" color="primary" variant="tonal">{{ target.name || target.value }}</v-chip></div></template>
    </section>
    <footer class="wizard-actions"><v-btn variant="text" @click="back">{{ step===1?t('common.cancel'):t('common.back') }}</v-btn><v-spacer/><v-btn v-if="step<5" color="primary" append-icon="mdi-arrow-right" :disabled="!stepReady" @click="next">{{ t('common.continue') }}</v-btn><v-btn v-else color="primary" prepend-icon="mdi-check" :loading="saving" @click="saveFlow">{{ editingId ? t('auto.update') : t('auto.create') }}</v-btn></footer>
  </template>
</template>

<style scoped>
.page-head{display:flex;justify-content:space-between;align-items:center;gap:22px;margin-bottom:16px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head>div:first-child{max-width:720px}.page-head p,.section-title small,.panel-head small{margin:0 0 6px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 6px;font-size:clamp(22px,3vw,28px)}.page-head span,.section-title p{color:var(--color-text-secondary);font-size:13px;line-height:1.6}.head-actions{display:flex;gap:8px;flex-wrap:wrap;align-items:center}.summary-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin-bottom:16px}.summary-grid article{display:flex;align-items:center;gap:14px;padding:18px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);color:var(--color-primary)}.summary-grid article>div{display:grid}.summary-grid strong{color:var(--color-text);font-size:22px}.summary-grid span{color:var(--color-text-secondary);font-size:11px}.rules-panel,.wizard-card{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.panel-head{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:18px 20px;border-bottom:1px solid var(--color-border)}.panel-head h3{margin:0}.panel-head :deep(.v-input){width:min(100%,340px)}.flow-row{display:grid;grid-template-columns:auto minmax(0,1fr) auto auto;align-items:center;gap:14px;padding:18px 20px;border-bottom:1px solid var(--color-border-subtle)}.flow-row:last-child{border:0}.flow-icon{width:44px;height:44px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:12px}.rule-main{display:grid;gap:4px;min-width:0}.rule-main strong,.rule-main span,.rule-main small{overflow:hidden;text-overflow:ellipsis}.rule-main span,.rule-main small{color:var(--color-text-secondary);font-size:11px}.chips{display:flex;gap:6px;flex-wrap:wrap;margin-top:6px}.empty{min-height:310px;display:grid;place-items:center;align-content:center;text-align:center;padding:30px;color:var(--color-text-muted)}.empty h3{margin:12px 0 3px;color:var(--color-text)}.empty p{max-width:560px;margin:0 0 15px;line-height:1.6}.steps{display:grid;grid-template-columns:repeat(5,1fr);margin-bottom:14px;padding:8px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.steps button{min-height:48px;display:flex;align-items:center;justify-content:center;gap:8px;padding:8px;border:0;border-radius:11px;background:none;color:var(--color-text-muted);cursor:pointer}.steps button span{flex:0 0 28px;width:28px;height:28px;display:grid;place-items:center;border-radius:50%;background:var(--color-surface-soft);font-size:11px}.steps button.active{color:var(--color-primary);background:var(--color-primary-soft)}.steps button.done{color:var(--color-primary)}.steps button.active span,.steps button.done span{color:white;background:var(--color-primary)}.wizard-card{min-height:430px;padding:clamp(18px,3vw,30px)}.section-title{max-width:780px}.section-title h3{margin:0 0 6px;font-size:21px}.section-title p{margin:0 0 20px}.form-grid,.review{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.target-toggle{margin-bottom:16px}.target-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.target-list>button{min-height:76px;display:grid;grid-template-columns:auto minmax(0,1fr);align-items:center;gap:10px;padding:13px;border:1px solid var(--color-border);border-radius:13px;background:white;text-align:left;color:var(--color-text);cursor:pointer}.target-list>button:hover:not(:disabled){border-color:var(--color-primary)}.target-list>button.selected{border-color:var(--color-primary);background:var(--color-primary-soft)}.target-list>button.disabled{opacity:.55;cursor:not-allowed}.target-list>button>div,.response-preview{display:grid;min-width:0}.target-list strong,.target-list span{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.target-list span,.target-list small,.response-preview span,.response-preview small{color:var(--color-text-secondary);font-size:11px}.response-preview{padding:17px;background:var(--color-surface-soft);border-radius:14px}.loading{min-height:220px;display:grid;place-items:center}.review{grid-template-columns:repeat(3,1fr);margin-bottom:14px}.review article{display:grid;gap:5px;padding:18px;background:var(--color-surface-soft);border-radius:14px}.review span,.review small{color:var(--color-text-secondary);font-size:11px}.wizard-actions{position:sticky;bottom:12px;z-index:5;display:flex;align-items:center;margin-top:14px;padding:12px 14px;background:rgba(255,255,255,.96);border:1px solid var(--color-border);border-radius:var(--radius-lg);box-shadow:0 10px 30px rgba(31,35,75,.1);backdrop-filter:blur(12px)}
.flow-block{border-bottom:1px solid var(--color-border-subtle)}.flow-block:last-child{border:0}.flow-block .flow-row{border-bottom:0}.flow-actions{display:flex;align-items:center;gap:2px}.flow-details{display:grid;grid-template-columns:repeat(4,minmax(0,1fr)) auto;align-items:end;gap:12px;padding:16px 20px 20px 78px;background:var(--color-surface-soft)}.flow-details>div{display:grid;gap:3px;min-width:0}.flow-details span{color:var(--color-text-secondary);font-size:11px}.flow-details strong{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}.flow-details .detail-targets{grid-column:1/-2}
@media(max-width:850px){.summary-grid,.review,.form-grid,.target-list{grid-template-columns:1fr}.flow-details{grid-template-columns:repeat(2,minmax(0,1fr));padding-left:20px}.flow-details .detail-targets{grid-column:1/-1}.steps b{display:none}.steps button{padding-inline:4px}.review article{padding:14px}}@media(max-width:640px){.page-head{display:grid;align-items:stretch;padding:19px}.head-actions{display:grid;grid-template-columns:1fr 1fr}.head-actions>*{width:100%}.summary-grid{grid-template-columns:1fr 1fr}.summary-grid article{padding:14px}.summary-grid article:first-child{grid-column:1/-1}.wizard-card{min-height:390px;padding:16px}.flow-row{grid-template-columns:auto minmax(0,1fr);padding:15px 12px;gap:10px}.flow-row>.v-switch,.flow-actions{grid-column:2}.flow-actions{justify-content:flex-end}.flow-details{grid-template-columns:1fr;padding:14px}.flow-details .detail-targets{grid-column:1}.flow-details>.v-btn{width:100%}.flow-icon{width:38px;height:38px}.panel-head{display:grid;padding:16px}.panel-head :deep(.v-input){width:100%;max-width:none}.steps{gap:2px;padding:5px;overflow-x:auto}.steps button{min-width:48px}.target-toggle{display:grid!important;grid-template-columns:1fr 1fr;width:100%}.target-toggle :deep(.v-btn){width:100%}.wizard-actions{bottom:88px}.wizard-actions :deep(.v-btn){min-width:112px}}
</style>
