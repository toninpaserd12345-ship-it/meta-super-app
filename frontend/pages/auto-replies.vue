<script setup lang="ts">
import type { AutomationFlow, AutomationTarget, ItemsResponse, MetaAdAccount, MetaCampaign, MetaPage, MetaPost, Product, ReplySet } from '~/types/automation'

definePageMeta({ middleware: 'auth' })
const { can } = useAuth()
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
  name.value = ''; pageId.value = pages.value[0]?.id || ''; productId.value = products.value[0]?.id || ''; replySetId.value = usableSets.value[0]?.id || ''
  targetType.value = 'post'; targets.value = []; firstMessageOnly.value = true; cooldownSeconds.value = 0; step.value = 1; notice.value = ''; view.value = 'wizard'
}
function toggleTarget(target: AutomationTarget) {
  const key = `${target.type}:${target.value}`
  targets.value = selectedIds.value.has(key) ? targets.value.filter(item => `${item.type}:${item.value}` !== key) : [...targets.value, target]
}
function isSelected(type: string, value: string) { return selectedIds.value.has(`${type}:${value}`) }
function isUsed(type: string, value: string) { return flows.value.some(flow => flow.pageId === pageId.value && flow.targets.some(target => target.type === type && target.value === value)) }
function next() { if (stepReady.value && step.value < 5) step.value++ }
function back() { if (step.value > 1) step.value--; else view.value = 'dashboard' }

async function loadPosts() {
  if (!pageId.value) return
  loading.value = true
  try { posts.value = (await $fetch<ItemsResponse<MetaPost>>(`/api/proxy/api/v1/meta/pages/${pageId.value}/posts`)).items }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load Facebook posts.') }
  finally { loading.value = false }
}
async function loadAdAccounts() {
  if (adAccounts.value.length) return
  loading.value = true
  try { adAccounts.value = (await $fetch<ItemsResponse<MetaAdAccount>>('/api/proxy/api/v1/meta/ad-accounts')).items; adAccountId.value = adAccounts.value[0]?.id || '' }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load Ad Accounts. Reconnect Facebook with ads_read permission.') }
  finally { loading.value = false }
}
async function loadCampaigns() {
  campaigns.value = []
  if (!adAccountId.value) return
  loading.value = true
  try { campaigns.value = (await $fetch<ItemsResponse<MetaCampaign>>(`/api/proxy/api/v1/meta/ad-accounts/${adAccountId.value}/campaigns`)).items }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not load campaigns.') }
  finally { loading.value = false }
}
async function saveFlow() {
  saving.value = true; notice.value = ''
  try {
    await $fetch('/api/proxy/api/v1/automations', { method: 'POST', body: { name: name.value, pageId: pageId.value, productId: productId.value, replySetId: replySetId.value, firstMessageOnly: firstMessageOnly.value, cooldownSeconds: cooldownSeconds.value, targets: targets.value } })
    await refreshFlows(); noticeType.value = 'success'; notice.value = 'Automation created with all selected targets.'; view.value = 'dashboard'
  } catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not create the automation.') }
  finally { saving.value = false }
}
async function setStatus(flow: AutomationFlow, active: boolean) {
  try { await $fetch(`/api/proxy/api/v1/automations/${flow.id}/status`, { method: 'PATCH', body: { isActive: active } }); await refreshFlows() }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not update this automation.') }
}
async function deleteFlow(flow: AutomationFlow) {
  if (!confirm(`Delete “${flow.name}” and all ${flow.targets.length} targets?`)) return
  deletingId.value = flow.id
  try { await $fetch(`/api/proxy/api/v1/automations/${flow.id}`, { method: 'DELETE' }); await refreshFlows(); noticeType.value = 'success'; notice.value = 'Automation deleted.' }
  catch (error) { noticeType.value = 'error'; notice.value = apiError(error, 'Could not delete this automation.') }
  finally { deletingId.value = '' }
}

watch(pageId, () => { targets.value = []; if (targetType.value === 'post') loadPosts() })
watch(targetType, value => { targets.value = []; if (value === 'post') loadPosts(); else loadAdAccounts() })
watch(adAccountId, loadCampaigns)
</script>

<template>
  <section class="page-head">
    <div><p>AUTOMATION CENTER</p><h2>{{ view === 'dashboard' ? 'Auto Reply Manager' : 'New Automation' }}</h2><span>One automation can cover many Posts or Campaigns without duplicating its Product and Reply Set.</span></div>
    <div class="head-actions"><v-btn variant="outlined" to="/products" prepend-icon="mdi-package-variant-closed">Products</v-btn><v-btn variant="outlined" to="/replies" prepend-icon="mdi-message-text-fast-outline">Reply Sets</v-btn><v-btn v-if="view==='dashboard'" color="primary" prepend-icon="mdi-plus" :disabled="!can('pages:connect')" @click="startWizard">New Automation</v-btn></div>
  </section>
  <v-alert v-if="notice" :type="noticeType" variant="tonal" closable class="mb-4" @click:close="notice=''">{{ notice }}</v-alert>

  <template v-if="view==='dashboard'">
    <section class="summary-grid"><article><v-icon icon="mdi-robot-happy-outline"/><div><strong>{{ flows.length }}</strong><span>Automations</span></div></article><article><v-icon icon="mdi-target"/><div><strong>{{ flows.reduce((sum,item)=>sum+item.targets.length,0) }}</strong><span>Connected targets</span></div></article><article><v-icon icon="mdi-check-circle-outline"/><div><strong>{{ flows.filter(item=>item.isActive).length }}</strong><span>Active</span></div></article></section>
    <v-alert v-if="!products.length || !usableSets.length || !pages.length" type="info" variant="tonal" class="mb-4"><strong>Setup needed:</strong> create a Product, a Reply Set with an enabled message, and connect a Facebook Page.</v-alert>
    <section class="rules-panel"><div class="panel-head"><div><small>LIVE CONFIGURATION</small><h3>Automations</h3></div><v-text-field v-model="search" density="compact" variant="outlined" prepend-inner-icon="mdi-magnify" placeholder="Search automation or target" hide-details/></div>
      <div v-if="filteredFlows.length" class="rule-list"><article v-for="flow in filteredFlows" :key="flow.id" class="flow-row"><div class="flow-icon"><v-icon icon="mdi-robot-happy-outline"/></div><div class="rule-main"><strong>{{ flow.name }}</strong><span>{{ productName(flow.productId) }} → {{ setName(flow.replySetId) }}</span><small>{{ pageName(flow.pageId) }} · {{ flow.targets.length }} target{{ flow.targets.length===1?'':'s' }} · {{ flow.firstMessageOnly ? 'first message only' : 'every match' }}</small><div class="chips"><v-chip v-for="target in flow.targets.slice(0,4)" :key="target.id || `${target.type}:${target.value}`" size="x-small" variant="tonal">{{ target.type }} · {{ target.name || target.value }}</v-chip><v-chip v-if="flow.targets.length>4" size="x-small">+{{ flow.targets.length-4 }}</v-chip></div></div><v-switch :model-value="flow.isActive" color="success" hide-details density="compact" :disabled="!can('pages:connect')" @update:model-value="setStatus(flow,Boolean($event))"/><v-btn icon="mdi-delete-outline" variant="text" color="error" :loading="deletingId===flow.id" :disabled="!can('pages:connect')" @click="deleteFlow(flow)"/></article></div>
      <div v-else class="empty"><v-icon icon="mdi-robot-happy-outline" size="38"/><h3>No automations yet</h3><p>Create one workflow, then attach as many Posts or Campaigns as you need.</p><v-btn color="primary" :disabled="!products.length || !usableSets.length || !pages.length" @click="startWizard">Create Automation</v-btn></div>
    </section>
  </template>

  <template v-else>
    <nav class="steps"><button v-for="item in [{n:1,label:'Setup'},{n:2,label:'Trigger'},{n:3,label:'Response'},{n:4,label:'Behavior'},{n:5,label:'Review'}]" :key="item.n" :class="{active:step===item.n,done:step>item.n}" :disabled="item.n>step" @click="step=item.n"><span>{{ step>item.n?'✓':item.n }}</span><b>{{ item.label }}</b></button></nav>
    <section class="wizard-card">
      <template v-if="step===1"><div class="section-title"><small>STEP 1 · SETUP</small><h3>Name it and choose a Page</h3><p>The Page is fixed for this automation so every target receives the correct Page token.</p></div><div class="form-grid"><v-text-field v-model="name" label="Automation name" placeholder="e.g. Durian campaign replies" variant="outlined"/><v-select v-model="pageId" :items="pages" item-title="name" item-value="id" label="Facebook Page" variant="outlined"/></div></template>
      <template v-else-if="step===2"><div class="section-title"><small>STEP 2 · TRIGGER</small><h3>Where should this automation run?</h3><p>Campaign binding is dynamic: new Ads inside that Campaign are covered without duplicate automations.</p></div><v-btn-toggle v-model="targetType" mandatory color="primary" class="mb-4"><v-btn value="post" prepend-icon="mdi-post-outline">Posts</v-btn><v-btn value="campaign" prepend-icon="mdi-bullhorn-outline">Campaigns</v-btn></v-btn-toggle><div v-if="loading" class="loading"><v-progress-circular indeterminate color="primary"/></div><div v-else-if="targetType==='post'" class="target-list"><button v-for="post in posts" :key="post.id" :disabled="isUsed('post',post.id)" :class="{selected:isSelected('post',post.id),disabled:isUsed('post',post.id)}" @click="toggleTarget({type:'post',value:post.id,name:post.message || 'Facebook Post'})"><v-checkbox-btn :model-value="isSelected('post',post.id)"/><div><strong>{{ post.message || 'Facebook Post' }}</strong><span>{{ post.id }}</span><small v-if="isUsed('post',post.id)">Already used by another automation</small></div></button></div><template v-else><v-select v-model="adAccountId" :items="adAccounts.map(item=>({...item,label:`${item.accountType==='business'?'Business':'Personal'} · ${item.name}`}))" item-title="label" item-value="id" label="Ad Account" variant="outlined"/><div class="target-list"><button v-for="campaign in campaigns" :key="campaign.id" :disabled="isUsed('campaign',campaign.id)" :class="{selected:isSelected('campaign',campaign.id),disabled:isUsed('campaign',campaign.id)}" @click="toggleTarget({type:'campaign',value:campaign.id,name:campaign.name})"><v-checkbox-btn :model-value="isSelected('campaign',campaign.id)"/><div><strong>{{ campaign.name }}</strong><span>{{ campaign.adSetCount }} ad sets · {{ campaign.adCount }} ads</span><small>{{ isUsed('campaign',campaign.id)?'Already used by another automation':campaign.effectiveStatus }}</small></div></button></div></template></template>
      <template v-else-if="step===3"><div class="section-title"><small>STEP 3 · RESPONSE</small><h3>Choose Product and Reply Set</h3><p>The Product fills message variables; enabled Reply Set items send in their saved order.</p></div><div class="form-grid"><v-select v-model="productId" :items="products" item-title="name" item-value="id" label="Product" variant="outlined"/><v-select v-model="replySetId" :items="usableSets" item-title="name" item-value="id" label="Reply Set" variant="outlined"/></div><div v-if="selectedSet" class="response-preview"><strong>{{ selectedSet.name }}</strong><span>{{ selectedSet.items.filter(item=>item.isEnabled).length }} enabled messages</span><small>Text, image, video and audio send from top to bottom.</small></div></template>
      <template v-else-if="step===4"><div class="section-title"><small>STEP 4 · BEHAVIOR</small><h3>Prevent unwanted repeat replies</h3><p>Safe defaults are enabled. You can change them per automation.</p></div><v-switch v-model="firstMessageOnly" color="primary" label="Reply only to the customer’s first matching message" inset/><v-text-field v-model.number="cooldownSeconds" type="number" min="0" max="86400" label="Cooldown before this automation can reply again (seconds)" hint="0 disables cooldown. First-message-only still prevents duplicates." persistent-hint variant="outlined"/></template>
      <template v-else><div class="section-title"><small>STEP 5 · REVIEW</small><h3>Review one complete automation</h3><p>Saving is transactional: either every selected target is added, or none are.</p></div><div class="review"><article><span>Automation</span><strong>{{ name }}</strong><small>{{ selectedPage?.name }}</small></article><article><span>Response</span><strong>{{ selectedProduct?.name }}</strong><small>{{ selectedSet?.name }}</small></article><article><span>Targets</span><strong>{{ targets.length }} {{ targetType }}</strong><small>{{ firstMessageOnly?'First message only':'Every matching message' }}</small></article></div><div class="chips"><v-chip v-for="target in targets" :key="`${target.type}:${target.value}`" color="primary" variant="tonal">{{ target.name || target.value }}</v-chip></div></template>
    </section>
    <footer class="wizard-actions"><v-btn variant="text" @click="back">{{ step===1?'Cancel':'Back' }}</v-btn><v-spacer/><v-btn v-if="step<5" color="primary" append-icon="mdi-arrow-right" :disabled="!stepReady" @click="next">Continue</v-btn><v-btn v-else color="primary" prepend-icon="mdi-check" :loading="saving" @click="saveFlow">Create Automation</v-btn></footer>
  </template>
</template>

<style scoped>
.page-head{display:flex;justify-content:space-between;gap:22px;margin-bottom:18px;padding:24px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.page-head p,.section-title small,.panel-head small{margin:0 0 6px;color:var(--color-primary);font-size:10px;font-weight:800;letter-spacing:.13em}.page-head h2{margin:0 0 6px;font-size:24px}.page-head span,.section-title p{color:var(--color-text-secondary);font-size:13px}.head-actions{display:flex;gap:8px;flex-wrap:wrap;align-items:flex-start}.summary-grid{display:grid;grid-template-columns:repeat(3,1fr);gap:12px;margin-bottom:16px}.summary-grid article{display:flex;align-items:center;gap:14px;padding:18px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg);color:var(--color-primary)}.summary-grid article>div{display:grid}.summary-grid strong{color:var(--color-text);font-size:22px}.summary-grid span{color:var(--color-text-secondary);font-size:11px}.rules-panel,.wizard-card{background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.panel-head{display:flex;align-items:center;justify-content:space-between;gap:20px;padding:18px 20px;border-bottom:1px solid var(--color-border)}.panel-head h3{margin:0}.panel-head :deep(.v-input){max-width:340px}.flow-row{display:grid;grid-template-columns:auto 1fr auto auto;align-items:center;gap:14px;padding:18px 20px;border-bottom:1px solid var(--color-border-subtle)}.flow-row:last-child{border:0}.flow-icon{width:42px;height:42px;display:grid;place-items:center;color:var(--color-primary);background:var(--color-primary-soft);border-radius:12px}.rule-main{display:grid;gap:3px;min-width:0}.rule-main span,.rule-main small{color:var(--color-text-secondary);font-size:11px}.chips{display:flex;gap:6px;flex-wrap:wrap;margin-top:6px}.empty{min-height:300px;display:grid;place-items:center;align-content:center;text-align:center;padding:30px;color:var(--color-text-muted)}.empty h3{margin:12px 0 3px;color:var(--color-text)}.empty p{margin:0 0 15px}.steps{display:grid;grid-template-columns:repeat(5,1fr);margin-bottom:14px;padding:9px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}.steps button{display:flex;align-items:center;justify-content:center;gap:8px;padding:10px;border:0;background:none;color:var(--color-text-muted)}.steps button span{width:27px;height:27px;display:grid;place-items:center;border-radius:50%;background:var(--color-surface-soft);font-size:11px}.steps button.active,.steps button.done{color:var(--color-primary)}.steps button.active span,.steps button.done span{color:white;background:var(--color-primary)}.wizard-card{min-height:430px;padding:24px}.section-title h3{margin:0 0 6px;font-size:21px}.section-title p{margin:0 0 20px}.form-grid,.review{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.target-list{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:10px}.target-list>button{display:grid;grid-template-columns:auto 1fr;align-items:center;gap:10px;padding:13px;border:1px solid var(--color-border);border-radius:13px;background:white;text-align:left;color:var(--color-text)}.target-list>button.selected{border-color:var(--color-primary);background:var(--color-primary-soft)}.target-list>button.disabled{opacity:.55}.target-list>button>div,.response-preview{display:grid}.target-list span,.target-list small,.response-preview span,.response-preview small{color:var(--color-text-secondary);font-size:11px}.response-preview{padding:17px;background:var(--color-surface-soft);border-radius:14px}.loading{min-height:220px;display:grid;place-items:center}.review{grid-template-columns:repeat(3,1fr);margin-bottom:14px}.review article{display:grid;gap:5px;padding:18px;background:var(--color-surface-soft);border-radius:14px}.review span,.review small{color:var(--color-text-secondary);font-size:11px}.wizard-actions{display:flex;align-items:center;margin-top:14px;padding:14px;background:var(--color-surface);border:1px solid var(--color-border);border-radius:var(--radius-lg)}
@media(max-width:850px){.summary-grid,.review,.form-grid,.target-list{grid-template-columns:1fr}.steps b{display:none}}@media(max-width:640px){.page-head{display:grid;padding:20px}.head-actions{display:grid;grid-template-columns:1fr 1fr}.head-actions>:last-child{grid-column:1/-1}.wizard-card{padding:16px}.flow-row{grid-template-columns:auto 1fr auto}.flow-row>:last-child{grid-column:3}.panel-head{display:grid}.panel-head :deep(.v-input){max-width:none}}
</style>
